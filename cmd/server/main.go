package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/connorpodea/splitit/internal/handlers"
	"github.com/connorpodea/splitit/internal/store"
)

// main initializes the database, registers all HTTP routes, and starts the HTTP server.
func main() {
	// 1. DATABASE SETUP
	s, err := store.New()
	if err != nil {
		log.Fatalf("[CRITICAL] Database failed to initialize: %v", err)
	}
	log.Println("[INFO] SQLite storage engine initialized successfully")

	// 2. OVERDUE PENALTY SCHEDULER
	// Goroutine runs concurrently so the ticker loop doesn't block startup or the HTTP server.
	go func() {
		for range time.NewTicker(24 * time.Hour).C {
			if err := s.ApplyMonthlyOverduePenalties(); err != nil {
				log.Printf("[WARN] Overdue penalty cycle failed: %v", err)
			}
		}
	}()

	// 3. CONTROLLER LAYER
	h := handlers.New(s)

	// 3. ROUTING REGISTRATION
	h.RegisterRoutes()
	log.Println("[INFO] Application route handlers mapped successfully")

	// 4. ROOT ROUTE — serves the single-page app shell
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// csrf_token — proves the request came from our frontend, not a malicious site.
		// Does NOT identify who the user is. Never stored in the database.
		// Distinct from session_token, which identifies the logged-in user via a DB lookup.
		b := make([]byte, 16)
		rand.Read(b)
		csrfToken := hex.EncodeToString(b)
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		masterCanvas := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SplitIt - Ledger Terminal</title>
    <script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/json-enc.js"></script>
    <script>
      // Echo the csrf_token cookie as X-CSRF-Token on every HTMX request.
      function _csrf() {
        var m = document.cookie.match('(?:^|;)\\s*csrf_token=([^;]+)');
        return m ? m[1] : '';
      }
      document.addEventListener('htmx:configRequest', function(e) {
        e.detail.headers['X-CSRF-Token'] = _csrf();
      });
    </script>
</head>
<body class="bg-[#0b0f19] text-slate-100 min-h-screen flex items-center justify-center antialiased">

    <div id="main-application-viewport"
         hx-get="/ui/initial-view"
         hx-trigger="load"
         class="w-full">
         <div class="text-center font-mono text-xs text-slate-600 animate-pulse">
             Authenticating background ledger session...
         </div>
    </div>

</body>
</html>`

		w.Write([]byte(masterCanvas))
	})

	// 5. START NETWORK SERVER — CSRF middleware wraps the entire mux so every
	// state-mutating route is protected without per-handler boilerplate.
	port := ":8080"
	log.Printf("[INFO] SplitIt engine online. Streaming on http://localhost%s\n", port)

	if err := http.ListenAndServe(port, handlers.CSRF(http.DefaultServeMux)); err != nil {
		log.Fatalf("[CRITICAL] HTTP network server collapsed: %v", err)
	}
}
