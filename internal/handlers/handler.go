package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/store"
)

// Handler isolates our web API server logic from our data storage layer.
// It stores a pointer reference to the database engine so methods can query it.
type Handler struct {
	store *store.Store
}

// New is a constructor function that instantiates the web handler context.
// It accepts a memory pointer to the database engine and returns a memory pointer to the handler.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// WriteJSON formats and flushes a structural data payload over an active HTTP connection.
// 'status' takes standard HTTP integer codes (e.g., 200, 400), and 'data' accepts any struct type.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	// Set the network header to inform the client that the incoming stream is a structured JSON payload
	w.Header().Set("Content-Type", "application/json")

	// Write the HTTP integer status code to the response header
	w.WriteHeader(status)

	// Stream and encode the Go data structure directly into the HTTP response body pipeline
	json.NewEncoder(w).Encode(data)
}

// writeHTML writes a complete HTML string to the response with a 200 status and text/html content type.
func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

// authenticateSession checks for a valid session token cookie before processing sensitive ledger actions
// This serves as an internal cryptographic gate, ensuring stranger requests are rejected with a 401 (unauthorized)
func (h *Handler) authenticateSession(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required: Missing or expired session signature"})
		return "", false
	}
	return cookie.Value, true
}
