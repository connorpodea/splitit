package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
)

// CreateBNPLLoan handles a request to open a new buy-now-pay-later loan, binding the
// borrower identity from the session cookie before delegating to the store layer.
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

	input.SenderID = borrowerID

	if len(input.Note) > 500 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Note must be 500 characters or fewer"})
		return
	}

	err = h.store.CreateBNPLLoan(&input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusCreated, input)
}

// PayInstallment handles a request to settle a single BNPL installment, overriding the
// userID from the session cookie to prevent users from paying other accounts' debts.
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
		InstallmentID string `json:"installment_id"`
		PaymentID     string `json:"payment_id"`
		UserID        string `json:"user_id"`
		AmountCents   int    `json:"amount_cents"`
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
	err = h.store.PayInstallment(input.InstallmentID, input.PaymentID, input.UserID, input.AmountCents)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

// ListInstallments returns the full installment debt schedule for the authenticated user.
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
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, installments)
}

// ListOverdueInstallments returns all unpaid installments past their due date for the authenticated user.
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
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data struct into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, installments)
}

// GetPaymentWithInstallments returns a master payment record alongside its full installment
// schedule. The caller must be either the borrower (sender) or merchant (receiver).
func (h *Handler) GetPaymentWithInstallments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted to this route"})
		return
	}

	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	paymentID := r.URL.Query().Get("payment_id")
	if paymentID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'payment_id'"})
		return
	}

	result, err := h.store.GetPaymentWithInstallments(paymentID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": friendlyError(err)})
		return
	}

	if result.Payment.SenderID != userID && result.Payment.ReceiverID != userID {
		WriteJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have access to this loan record"})
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetInstallmentSummary returns a consolidated snapshot of the authenticated user's installment
// portfolio: total settled, outstanding, and overdue balances in a single response.
func (h *Handler) GetInstallmentSummary(w http.ResponseWriter, r *http.Request) {
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

	// Query the database engine for this user's installment portfolio aggregate metrics
	summary, err := h.store.GetInstallmentSummary(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, summary)
}
