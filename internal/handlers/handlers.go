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
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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

	// Query the database engine for this user
	user, err := h.store.GetUser(userID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
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

	// Query the database engine for all users
	users, err := h.store.ListUsers()
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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

	// Query the database engine for all profiles
	profiles, err := h.store.ListProfiles()
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'user_id'"})
		return
	}

	// Query the database engine for this profile
	profile, err := h.store.GetProfile(userID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	// Ensure that this enpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted to this route"})
		return
	}

	// Initialize an empty Payment model struct to hold the incoming data
	var input models.Payment

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the populated struct down to the database engine
	err = h.store.Pay(&input)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Extract the payment ID from the URL query parameters
	paymentID := r.URL.Query().Get("payment_id")
	if paymentID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'payment_id'"})
		return
	}

	// Query the database engine for this payment
	payment, err := h.store.GetPayment(paymentID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, payment)
}

func (h *Handler) CreateBNPLLoan(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted to this route"})
		return
	}

	// Initialize an empty Payment model struct to hold the incoming data
	var input models.Payment

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the populated struct down to the database engine
	err = h.store.CreateBNPLLoan(&input)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) PayInstallment(w http.ResponseWriter, r *http.Request) {
	// Ensure that this enpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted to this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		InstallmentID string  `json:"installment_id"`
		PaymentID     string  `json:"payment_id"`
		UserID        string  `json:"user_id"`
		Amount        float64 `json:"amount"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the populated struct down to the database engine
	err = h.store.PayInstallment(input.InstallmentID, input.PaymentID, input.UserID, input.Amount)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) ListInstallments(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Extract the user ID from the URL query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'user_id'"})
		return
	}

	// Query the database engine for this users installments
	installments, err := h.store.ListInstallments(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, installments)
}

func (h *Handler) ListOverdueInstallments(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Extract the user ID from the URL query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'user_id'"})
		return
	}

	// Query the database engine for this users overdue installments
	installments, err := h.store.ListOverdueInstallments(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data stuct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, installments)
}

func (h *Handler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted to this route"})
		return
	}

	// Initialize an empty FriendRequest model struct to hold the incoming data
	var input models.FriendRequest

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the populated struct down to the database engine
	err = h.store.SendFriendRequest(&input)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) ListIncomingFriendRequests(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Extract the user ID from the URL query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL parameter: 'user_id'"})
		return
	}

	// Query the database engine for this users incoming friend requests
	requests, err := h.store.ListIncomingFriendRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, requests)
}

func (h *Handler) ListOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	// Extract the user ID from the URL query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL parameter: 'user_id'"})
		return
	}

	// Query the database engine for this users outgoing friend requests
	requests, err := h.store.ListIncomingFriendRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, requests)
}

func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		RequestID  string `json:"request_id"`
		SenderID   string `json:"sender_id"`
		ReceiverID string `json:"receiver_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the populated struct down to the database engine
	err = h.store.AcceptFriendRequest(input.RequestID, input.SenderID, input.ReceiverID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint on accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		RequestID string `json:"request_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the variable down to the database engine
	err = h.store.DeclineFriendRequest(input.RequestID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) RemoveFriendMutual(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint on accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		UserID   string `json:"user_id"`
		FriendID string `json:"friend_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the variable down to the database engine
	err = h.store.RemoveFriendMutual(input.UserID, input.FriendID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET methods are permitted to this route"})
		return
	}

	// Extract the user ID from the URL query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL parameter : 'user_id'"})
		return
	}

	// Query the database engine for this users friends
	friends, err := h.store.ListFriends(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, friends)
}

func (h *Handler) CreatePaymentRequst(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Initialize an empty Payment model struct to hold the incoming data
	var input models.PaymentRequest

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the variable down to the database engine
	err = h.store.CreatePaymentRequest(&input)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

func (h *Handler) ListIncomingPaymentRequests(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET methods are permitted to this route"})
		return
	}

	// Extract the userID from the URL query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL parameter: 'user_id'"})
		return
	}

	// Query the database engine for this users incoming payment requests
	requests, err := h.store.ListIncomingPaymentRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, requests)
}

func (h *Handler) ListOutgoingPaymentRequests(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET methods are permitted to this route"})
		return
	}

	// Extract the userID from the URL query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL parameter: 'user_id'"})
		return
	}

	// Query the database engine for this users outgoing payment requests
	requests, err := h.store.ListOutgoingPaymentRequests(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, requests)
}

func (h *Handler) UpdatePaymentRequestStatus(w http.ResponseWriter, r *http.Request) {

}
