package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/connorpodea/splitit/internal/models"
)

// Pay handles a direct peer-to-peer payment request, binding the sender identity
// from the session cookie before passing the payload to the store.
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

// GetPayment returns a single payment record by the payment_id URL query parameter.
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

// CreatePaymentRequest creates a new pending payment request, binding the requester identity
// from the session cookie to prevent spoofing.
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

// ListIncomingPaymentRequests returns all pending payment requests where the authenticated user is the payer.
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

// ListOutgoingPaymentRequests returns all pending payment requests that the authenticated user has sent.
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

// UpdatePaymentRequestStatus transitions a payment request to a new status (e.g. accepted, declined).
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

// paySheetHTML renders the bottom-sheet HTML for the payment action, containing both the
// direct send tab and the BNPL loan tab with friend selector and amount inputs.
func paySheetHTML(friends []models.Profile, availableCredit float64) string {
	friendOpts := `<option value="">Select a friend…</option>`
	for _, f := range friends {
		dname := profileDisplayName(&f)
		friendOpts += `<option value="` + f.ID + `">` + dname + ` · @` + f.ID + `</option>`
	}

	return `
<div class="sheet-backdrop" id="pay-sheet" onclick="if(event.target===this) closeSheet('pay')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">New payment</div>
    <div class="sheet-tabs" role="tablist">
      <button class="sheet-tab active" data-paytab="send" onclick="payTab(this)">Send to friend</button>
      <button class="sheet-tab" data-paytab="bnpl" onclick="payTab(this)">Buy now, pay later</button>
    </div>

    <div class="sheet-pane active" data-pay-pane="send">
      <div class="amount-big mono"><span class="dollar">$</span><span id="send-amt">0.00</span></div>
      <div class="chip-row" style="justify-content:center;">
        <button type="button" class="chip" onclick="setSendAmt(10)">$10</button>
        <button type="button" class="chip" onclick="setSendAmt(25)">$25</button>
        <button type="button" class="chip" onclick="setSendAmt(50)">$50</button>
        <button type="button" class="chip" onclick="setSendAmt(100)">$100</button>
      </div>
      <div>
        <label class="modal-lbl">To</label>
        <select class="modal-sel" id="pay-receiver-sel">` + friendOpts + `</select>
      </div>
      <div>
        <label class="modal-lbl">Amount</label>
        <input class="modal-inp" id="pay-amount-inp" type="number" inputmode="decimal" placeholder="0.00"
               oninput="document.getElementById('send-amt').textContent=parseFloat(this.value||0).toFixed(2)" />
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" id="pay-note-inp" type="text" placeholder="Dinner, rent, etc." />
      </div>
      <button class="submit-btn" onclick="submitSendPayment()">Send payment</button>
    </div>

    <div class="sheet-pane" data-pay-pane="bnpl">
      <div class="info-card">
        <span class="label">Available credit</span>
        <span class="val mono">$` + fmt.Sprintf("%.2f", availableCredit) + `</span>
      </div>
      <div>
        <label class="modal-lbl">Recipient</label>
        <select class="modal-sel" id="bnpl-receiver-sel">` + friendOpts + `</select>
      </div>
      <div>
        <label class="modal-lbl">Purchase amount</label>
        <input class="modal-inp" id="bnpl-amount-inp" type="number" inputmode="decimal" placeholder="0.00" />
      </div>
      <div>
        <label class="modal-lbl">Plan</label>
        <select class="modal-sel" id="bnpl-plan-sel">
          <option value="4">Pay-in-4 (0% APR · 4 payments over 6 weeks)</option>
          <option value="6">Pay-in-6 (12.99% APR · 6 monthly payments)</option>
          <option value="12">Pay-in-12 (15.99% APR · 12 monthly payments)</option>
        </select>
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" id="bnpl-note-inp" type="text" placeholder="Item or purchase description" />
      </div>
      <button class="submit-btn emerald" onclick="submitBNPL()">Approve plan</button>
    </div>
  </div>
</div>`
}

// requestSheetHTML renders the bottom-sheet HTML for creating a money request,
// with a friend selector, amount input, and note field.
func requestSheetHTML(friends []models.Profile) string {
	friendOpts := `<option value="">Select a friend…</option>`
	for _, f := range friends {
		dname := profileDisplayName(&f)
		friendOpts += `<option value="` + f.ID + `">` + dname + ` · @` + f.ID + `</option>`
	}

	return `
<div class="sheet-backdrop" id="request-sheet" onclick="if(event.target===this) closeSheet('request')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">Request money</div>
    <div class="sheet-pane active" style="display:flex;">
      <div class="amount-big mono"><span class="dollar">$</span><span id="req-amt">0.00</span></div>
      <div class="chip-row" style="justify-content:center;">
        <button type="button" class="chip" onclick="setReqAmt(10)">$10</button>
        <button type="button" class="chip" onclick="setReqAmt(25)">$25</button>
        <button type="button" class="chip" onclick="setReqAmt(50)">$50</button>
        <button type="button" class="chip" onclick="setReqAmt(100)">$100</button>
      </div>
      <div>
        <label class="modal-lbl">Request from</label>
        <select class="modal-sel" id="req-payer-sel">` + friendOpts + `</select>
      </div>
      <div>
        <label class="modal-lbl">Amount</label>
        <input class="modal-inp" id="req-amount-inp" type="number" inputmode="decimal" placeholder="0.00"
               oninput="document.getElementById('req-amt').textContent=parseFloat(this.value||0).toFixed(2)" />
      </div>
      <div>
        <label class="modal-lbl">Note</label>
        <input class="modal-inp" id="req-note-inp" type="text" placeholder="What's it for?" />
      </div>
      <button class="submit-btn amber" onclick="submitRequest()">Send request</button>
    </div>
  </div>
</div>`
}
