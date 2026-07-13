package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/store"
)

// Handler isolates our web API server logic from our data storage layer.
type Handler struct {
	store *store.Store
}

// New instantiates the web handler with a live database reference.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
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
	return h.store.CreateSession(userID)
}

// authenticateSession validates the session cookie and returns the associated userID.
// The cookie value is an opaque random token looked up in the sessions table —
// it carries no identity information on its own, so forging it is computationally infeasible.
func (h *Handler) authenticateSession(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required: missing or expired session"})
		return "", false
	}
	userID, ok := h.store.LookupSession(cookie.Value)
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required: session not recognised"})
		return "", false
	}
	return userID, true
}
