package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
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

// CreateUser handles the HTTP POST request to register a new user profile
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Enforce that this endpoint only accepts POST requests (writing data)
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requestsare permitted on this route"})
		return
	}

	// Initialize an empty User model struct to hold the incoming data
	var input models.User

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the populated struct down to the database engine
	err = h.store.CreateUser(&input)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to write the user record to database"})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, input)
}

