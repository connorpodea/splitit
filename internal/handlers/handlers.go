package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
	"golang.org/x/crypto/bcrypt"
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

// RegisterRoutes attaches all application endpoints to Go's standard HTTP router.
// This keeps main.go perfectly clean and centralized.
func (h *Handler) RegisterRoutes() {
	// Users & Profiles
	http.HandleFunc("/users/create", h.CreateUser)
	http.HandleFunc("/users/login", h.LoginUser)
	http.HandleFunc("/users/get", h.GetUser)
	http.HandleFunc("/users/list", h.ListUsers)
	http.HandleFunc("/profiles/list", h.ListProfiles)
	http.HandleFunc("/profiles/get", h.GetProfile)

	// Payments
	http.HandleFunc("/payments/pay", h.Pay)
	http.HandleFunc("/payments/get", h.GetPayment)
	http.HandleFunc("/payments/request/create", h.CreatePaymentRequest)
	http.HandleFunc("/payments/request/incoming", h.ListIncomingPaymentRequests)
	http.HandleFunc("/payments/request/outgoing", h.ListOutgoingPaymentRequests)
	http.HandleFunc("/payments/request/update", h.UpdatePaymentRequestStatus)

	// BNPL Loans
	http.HandleFunc("/bnpl/loan/create", h.CreateBNPLLoan)
	http.HandleFunc("/bnpl/installment/pay", h.PayInstallment)
	http.HandleFunc("/bnpl/installments/list", h.ListInstallments)
	http.HandleFunc("/bnpl/installments/overdue", h.ListOverdueInstallments)

	// Friends System
	http.HandleFunc("/friends/request/send", h.SendFriendRequest)
	http.HandleFunc("/friends/request/incoming", h.ListIncomingFriendRequests)
	http.HandleFunc("/friends/request/outgoing", h.ListOutgoingFriendRequests)
	http.HandleFunc("/friends/request/accept", h.AcceptFriendRequest)
	http.HandleFunc("/friends/request/decline", h.DeclineFriendRequest)
	http.HandleFunc("/friends/remove", h.RemoveFriendMutual)
	http.HandleFunc("/friends/list", h.ListFriends)
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
	// Ensure that this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type RegistrationInput struct {
		ID          string `json:"id"`
		Password    string `json:"password"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phone_number"`
	}
	var input RegistrationInput

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.Password == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Turn plain text password into a non-reversible cryptographic hash (bcrypt.DefaultCost tells it to scramble the password 14 times)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to protect password"})
		return
	}

	// Map the form input data over to the structural Database model
	newUser := models.User{
		ID:           input.ID,
		PasswordHash: string(hashedPassword),
		Name:         input.Name,
		Email:        input.Email,
		PhoneNumber:  input.PhoneNumber,
		Balance:      0.00,
		CreditScore:  50,
		CreditLimit:  1000.00,
	}

	// Pass the populated struct down to the database engine
	err = h.store.CreateUser(&newUser)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, input)
}

func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type LoginInput struct {
		ID       string `json:"id"`
		Password string `json:"password"`
	}
	var input LoginInput

	// Read the JSON text out of the web request body and decode it into the Go struct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Look up the user by ID
	user, err := h.store.GetUser(input.ID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Invalid User ID or password"})
		return
	}

	// Compare the password to the currently stored password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid User ID or password"})
		return
	}

	// Issue Cookie Wristband on absolute match success
    cookie := &http.Cookie{
        Name:     "session_user_id",
        Value:    user.ID,
        Path:     "/",
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    }
    http.SetCookie(w, cookie)

    // This special header tells HTMX: "The login was completely valid. Reload the main domain URL path 
    // so our server can read the new session wristband cookie and display the dashboard view."
    w.Header().Set("HX-Redirect", "/")
    w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
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
	// Ensure that this endpoint only accepts GET requests
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
	// Ensure that this endpoint only accepts GET requests
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, input)
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, input)
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
	requests, err := h.store.ListOutgoingFriendRequests(userID)
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
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

func (h *Handler) CreatePaymentRequest(w http.ResponseWriter, r *http.Request) {
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
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, input)
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
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		PaymentID string `json:"payment_id"`
		NewStatus string `json:"new_status"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the variable down to the database engine
	err = h.store.UpdatePaymentRequestStatus(input.PaymentID, input.NewStatus)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}
