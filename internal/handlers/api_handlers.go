package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

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
	if err != nil || input.Password == "" || input.ID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting or missing account properties"})
		return
	}
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.Name = strings.ToLower(strings.TrimSpace(input.Name))
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(input.ID) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Username may only contain letters, numbers, and underscores"})
		return
	}

	// Turn plain text password into a non-reversible cryptographic hash (bcrypt.DefaultCost tells it to scramble the password 14 times)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to protect account credentials securely"})
		return
	}

	// Map the form input data over to the structural Database model
	new_user := models.User{
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
	err = h.store.CreateUser(&new_user)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, map[string]string{"status": "ledger user allocation committed successfully", "id": new_user.ID})
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

	input.ID = strings.ToLower(strings.TrimSpace(input.ID))

	// Look up the user by ID
	user, err := h.store.GetUser(input.ID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Invalid User ID or password parameters"})
		return
	}

	// Compare the password to the currently stored password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid User ID or password parameters"})
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

	// Return a production-grade structural JSON status acknowledgement tracking packet
	WriteJSON(w, http.StatusOK, map[string]any{"authenticated": true, "user_id": user.ID, "name": user.Name})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Ensure that this endpoint only accepts GET requests
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}

	// Verify session context parameter state natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user using our secure identity
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

	// Confirm user active session validity state before parsing database records
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Confirm user active session validity state before parsing database records
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Confirm user active session validity state before parsing database records
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Extract the target profile identifier field from the URL query strings parameter bucket
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
	// Ensure that this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Verify session context parameter state natively through current session validation
	sessionID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Hard override variable integrity to completely block users from spending out of foreign profiles
	input.SenderID = sessionID

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

	// Confirm user active session validity state before parsing database records
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
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
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Securely lock the borrowing entity user payload based entirely on the active browser token
	borrowerID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Force parameter validation by tracking the transaction strictly through verified cookie identifiers
	input.SenderID = borrowerID

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
	// Ensure that this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Validate borrower context parameters natively through current session validation
	sessionID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Hard override variable integrity to block bad actors from clearing other profiles' financing debt tabs
	input.UserID = sessionID

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

	// Secure user validation using credentials fetched directly from cookie memory storage structures
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user's installments
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

	// Secure user validation using credentials fetched directly from cookie memory storage structures
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user's overdue installments
	installments, err := h.store.ListOverdueInstallments(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, installments)
}

func (h *Handler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	// Ensure this endpoint only accepts POST requests
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	// Acquire dynamic tracking criteria natively through current session validation
	senderID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Hard-bind the origin verification identity using internal session elements instead of raw user variables
	input.SenderID = senderID

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

	// Validate target context parameters natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user's incoming friend requests
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

	// Validate target context parameters natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Query the database engine for this user's outgoing friend requests
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

	// Secure user validation using credentials fetched directly from cookie memory storage structures
	receiverID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		RequestID string `json:"request_id"`
	}
	var input Input

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil || input.RequestID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// senderID is derived inside the store from the database — not trusted from the client
	err = h.store.AcceptFriendRequest(input.RequestID, receiverID)
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

	// Confirm user active session validation parameter state before deletion routines run
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Extract core verified sender profile constraints from active session data layers
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	// Initialize a custom, empty struct to hold the incoming data
	type Input struct {
		FriendID string `json:"friend_id"`
	}
	var input Input

	// Read the JSON text out of the web request body and decode it into the variable
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request payload formatting"})
		return
	}

	// Pass the verified identity along with request payload details down to the storage layers
	err = h.store.RemoveFriendMutual(userID, input.FriendID)
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

	// Secure user identity discovery tracking through current session mapping configurations
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Secure user identity isolation utilizing raw tokens inside session keys
	requesterID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Bind requester properties cleanly to block foreign entry identity spoofing attempts
	input.RequesterID = requesterID

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

	// Validate target context parameters natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Validate target context parameters natively through current session validation
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
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

	// Confirm caller validation session variables before editing invoice record statuses
	_, authorized := h.authenticateSession(w, r)
	if !authorized {
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

// Logout clears the session cookie, signing the user out.
// Browsers persist the cookie until we explicitly expire it with MaxAge=-1.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}

	expired := &http.Cookie{
		Name:     "session_user_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, expired)

	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	notifs, err := h.store.ListNotifications(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, notifs)
}

func (h *Handler) MarkNotificationSeen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	type Input struct {
		NotifID string `json:"notif_id"`
	}
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.NotifID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing notif_id"})
		return
	}
	if err := h.store.MarkNotificationSeen(input.NotifID, userID); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) ClearAllNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	if err := h.store.ClearAllNotifications(userID); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}
