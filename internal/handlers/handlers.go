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
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
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

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Enforce that this endpoint only accepts GET requests (reading data)
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	// Extract the user ID from the URL query parameters
	userID := r.URL.Query().Get("id")

	// Validate that the client actually provided an ID parameter
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'id'"})
		return
	}

	// Query the database engine using the extracted user ID string
	user, err := h.store.GetUser(userID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "No user profile found matching the provided ID"})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Enforce that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Query the database engine to retrieve a slice of all users
	users, err := h.store.ListUsers()
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "There was an error in retrieving users"})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, users)
}

func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	// Enforce that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Query the database engine to retrieve a slice of all profiles
	profiles, err := h.store.ListProfiles()
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "There was an error in retrieving profiles"})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, profiles)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Extract the user ID from the URL query parameters
	userID := r.URL.Query().Get("id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'id'"})
		return
	}

	// Query the database engine using the extracted user ID string
	profile, err := h.store.GetProfile(userID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "No user profile found matching the provided ID"})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) PostPay(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Extract the payment ID from the URL query parameters
	paymentID := r.URL.Query().Get("id")
	if paymentID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'id'"})
		return
	}

	// Query the database engine using the extracted payment ID string
	payment, err := h.store.GetPayment(paymentID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "No payment found matching the provided ID"})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, payment)
}

func (h *Handler) CreateBNPLLoan(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) PayInstallment(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) ListInstallments(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Parse the user ID
	userID := r.URL.Query().Get("id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'id'"})
		return
	}

	// Query the database engine to retrieve a slice of all installments
	installments, err := h.store.ListInstallments(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "There was an error in retrieving this users installments"})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, installments)
}

func (h *Handler) ListOverdueInstallments(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) ListIncomingFriendRequests(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) ListOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) RemoveFriendMutual(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) CreatePaymentRequst(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) ListIncomingPaymentRequests(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) ListOutgoingPaymentRequests(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) UpdatePaymentRequestStatus(w http.ResponseWriter, r *http.Request) {

}
