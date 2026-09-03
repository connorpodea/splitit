package handlers

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// CSRF checks the csrf_token cookie — this proves the request came from our frontend,
// not a malicious third-party site. It does NOT identify who the user is.
// For user identity, see authenticateSession() in handler.go which checks session_token.
// Implements the Double Submit Cookie pattern: the client reads the cookie (HttpOnly=false)
// and reflects its value as X-CSRF-Token header. Cross-site requests can't do this
// because they cannot read cookies from another origin.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("csrf_token")
		if err != nil || cookie.Value == "" {
			WriteJSON(w, http.StatusForbidden, map[string]string{"error": "CSRF token missing"})
			return
		}
		if r.Header.Get("X-CSRF-Token") != cookie.Value {
			WriteJSON(w, http.StatusForbidden, map[string]string{"error": "CSRF token invalid"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---- Token-bucket rate limiter -----------------------------------------------

type rlBucket struct {
	tokens     float64
	lastRefill time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*rlBucket
	rate     float64 // tokens added per second
	capacity float64 // maximum token count (= burst size)
}

func newRateLimiter(rate, capacity float64) *rateLimiter {
	rl := &rateLimiter{
		buckets:  make(map[string]*rlBucket),
		rate:     rate,
		capacity: capacity,
	}
	// Background goroutine evicts buckets idle for more than 5 minutes so the
	// map does not grow indefinitely under a large number of distinct IPs.
	go func() {
		for range time.NewTicker(5 * time.Minute).C {
			cutoff := time.Now().Add(-5 * time.Minute)
			rl.mu.Lock()
			for k, b := range rl.buckets {
				if b.lastRefill.Before(cutoff) {
					delete(rl.buckets, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

// allow returns true if the key has a token available, consuming one if so.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &rlBucket{tokens: rl.capacity, lastRefill: now}
		rl.buckets[key] = b
	}
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(rl.capacity, b.tokens+elapsed*rl.rate)
	b.lastRefill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// newAuthRateLimiter returns a middleware that permits 5 requests per minute
// per remote IP, with a burst of 5. Used on login and registration endpoints.
func newAuthRateLimiter() func(http.Handler) http.Handler {
	rl := newRateLimiter(5.0/60.0, 5)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if !rl.allow(ip) {
				WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many requests — please wait before trying again"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
