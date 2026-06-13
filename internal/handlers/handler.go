package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/connorpodea/splitit/internal/store"
)

// sessionStore is a concurrency-safe in-memory map of random token → userID.
// Tokens are issued on login and revoked on logout or account deactivation.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]string)}
}

// create generates a cryptographically random 64-hex-char token, stores the
// token→userID mapping, and returns the token to be placed in the session cookie.
func (ss *sessionStore) create(userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	ss.mu.Lock()
	ss.sessions[token] = userID
	ss.mu.Unlock()
	return token, nil
}

// lookup returns the userID associated with a token, or false if the token is
// unknown or has been revoked.
func (ss *sessionStore) lookup(token string) (string, bool) {
	ss.mu.RLock()
	userID, ok := ss.sessions[token]
	ss.mu.RUnlock()
	return userID, ok
}

// delete removes a token, effectively signing the user out on the server side.
func (ss *sessionStore) delete(token string) {
	ss.mu.Lock()
	delete(ss.sessions, token)
	ss.mu.Unlock()
}

// Handler isolates our web API server logic from our data storage layer.
type Handler struct {
	store    *store.Store
	sessions *sessionStore
}

// New instantiates the web handler with a live database reference and a fresh session store.
func New(s *store.Store) *Handler {
	return &Handler{
		store:    s,
		sessions: newSessionStore(),
	}
}

// WriteJSON serialises data to JSON and writes it to the response with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeHTML writes a pre-rendered HTML string to the response with a 200 status.
func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

// CreateSession creates a session token for the given userID and returns it.
// Exported for use in handler integration tests only.
func (h *Handler) CreateSession(userID string) (string, error) {
	return h.sessions.create(userID)
}

// authenticateSession validates the session cookie and returns the associated userID.
// The cookie value is an opaque random token looked up in the server-side session store —
// it carries no identity information on its own, so forging it is computationally infeasible.
func (h *Handler) authenticateSession(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required: missing or expired session"})
		return "", false
	}
	userID, ok := h.sessions.lookup(cookie.Value)
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required: session not recognised"})
		return "", false
	}
	return userID, true
}
