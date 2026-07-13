package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

	input.SenderID = sessionID

	if len(input.Note) > 500 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Note must be 500 characters or fewer"})
		return
	}

	// For direct payments the client sends only amount_cents; TotalAmountCents must equal AmountCents.
	if input.TotalAmountCents == 0 {
		input.TotalAmountCents = input.AmountCents
	}

	if err = h.store.Pay(&input); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	// Send a successful response back out the window along with the created data
	WriteJSON(w, http.StatusOK, input)
}

// GetPayment returns a single payment record by the payment_id URL query parameter.
// The caller must be either the sender or receiver of the payment.
func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
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

	payment, err := h.store.GetPayment(paymentID)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": friendlyError(err)})
		return
	}

	if payment.SenderID != userID && payment.ReceiverID != userID {
		WriteJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have access to this payment record"})
		return
	}

	WriteJSON(w, http.StatusOK, payment)
}

// ListPaymentsReceived returns all inbound payment records for the authenticated user, ordered newest first.
func (h *Handler) ListPaymentsReceived(w http.ResponseWriter, r *http.Request) {
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

	// Query the database engine for all payments received by this user
	payments, err := h.store.ListPaymentsReceived(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, payments)
}

// ListPaymentsSent returns all outbound payment records for the authenticated user, ordered newest first.
func (h *Handler) ListPaymentsSent(w http.ResponseWriter, r *http.Request) {
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

	// Query the database engine for all payments sent by this user
	payments, err := h.store.ListPaymentsSent(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, payments)
}

// ListPaymentsByUser returns a unified, chronologically sorted transaction feed containing
// both inbound and outbound payments for the authenticated user.
func (h *Handler) ListPaymentsByUser(w http.ResponseWriter, r *http.Request) {
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

	// Query the database engine for this user's full transaction history
	payments, err := h.store.ListPaymentsByUser(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, payments)
}

// ListPaymentsBetweenUsers returns the isolated payment stream shared between the authenticated
// user and a specified peer, identified by the other_id URL query parameter.
func (h *Handler) ListPaymentsBetweenUsers(w http.ResponseWriter, r *http.Request) {
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

	// Extract the peer identity from the URL query parameters
	otherID := r.URL.Query().Get("other_id")
	if otherID == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required URL query parameter: 'other_id'"})
		return
	}

	// Query the database engine for the bilateral transaction stream between both user identities
	payments, err := h.store.ListPaymentsBetweenUsers(userID, otherID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, payments)
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

	input.RequesterID = requesterID

	if len(input.Note) > 500 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Note must be 500 characters or fewer"})
		return
	}

	// Route to RequestFromGroup if the payer ID belongs to a group rather than an individual user.
	members, _ := h.store.ListGroupMembers(input.PayerID)
	if len(members) > 0 {
		err = h.store.RequestFromGroup(&input)
	} else {
		err = h.store.CreatePaymentRequest(&input)
	}
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
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
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
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
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	// Package the returned data into JSON and ship it over the wire
	WriteJSON(w, http.StatusOK, requests)
}

// UpdatePaymentRequestStatus transitions a payment request to "declined" or "cancelled".
// The payer may decline; the requester may cancel. Completed status is intentionally
// excluded — use FulfillPaymentRequest to atomically pay and close a request.
func (h *Handler) UpdatePaymentRequestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST methods are permitted to this route"})
		return
	}

	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		PaymentID string `json:"payment_id"`
		NewStatus string `json:"new_status"`
	}
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	allowed := map[string]bool{"declined": true, "cancelled": true}
	if !allowed[input.NewStatus] {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid status: must be 'declined' or 'cancelled'"})
		return
	}

	if err := h.store.UpdatePaymentRequestStatus(input.PaymentID, input.NewStatus, userID); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, input)
}

// FulfillPaymentRequest atomically pays a pending payment request on behalf of the session user.
func (h *Handler) FulfillPaymentRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	payerID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	type Input struct {
		RequestID string `json:"request_id"`
	}
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.RequestID) == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required field: request_id"})
		return
	}
	if err := h.store.FulfillPaymentRequest(input.RequestID, payerID); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetUnifiedActivity returns the chronological activity feed for the session user,
// combining payments sent/received and requests sent/received.
func (h *Handler) GetUnifiedActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}
	items, err := h.store.GetUnifiedActivity(userID, 50)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

// friendPickerHTML renders a horizontally scrollable avatar grid for friend selection,
// replacing plain <select> elements in pay, request, and BNPL sheets.
// User-supplied fields (IDs, names) are rendered through html/template which applies
// context-sensitive escaping: HTML entities in text/attribute context, JS string
// escaping inside onclick='...' arguments.
func friendPickerHTML(pickerID string, friends []models.Profile) string {
	if len(friends) == 0 {
		return `<div class="fp-empty">Add friends to get started.</div>`
	}
	fallbackColors := []string{"#4ade80", "#22d3ee", "#fb7185", "#facc15", "#c084fc", "#db2777"}
	var buf strings.Builder
	for i, f := range friends {
		dname := profileDisplayName(&f)
		color := f.ProfileColor
		if color == "" {
			color = fallbackColors[i%len(fallbackColors)]
		}
		buf.WriteString(renderTmpl(friendPickerItemTmpl, fpItemData{
			PickerID: pickerID,
			ID:       f.ID,
			Name:     dname,
			Initials: initials(dname),
			Color:    color,
		}))
	}
	return buf.String()
}

// groupFriendPickerHTML renders groups as fp-item tiles in a horizontally scrollable row,
// using the exact same component structure as friendPickerHTML — only the data source differs.
func groupFriendPickerHTML(pickerID string, groups []models.Group) string {
	if len(groups) == 0 {
		return `<div class="fp-empty">No groups yet.</div>`
	}
	hexColors := []string{"#c084fc", "#22d3ee", "#4ade80", "#facc15", "#fb7185", "#db2777"}
	var buf strings.Builder
	for i, g := range groups {
		runes := []rune(g.Name)
		ini := ""
		if len(runes) >= 2 {
			ini = strings.ToUpper(string(runes[0])) + strings.ToUpper(string(runes[1]))
		} else if len(runes) == 1 {
			ini = strings.ToUpper(string(runes[0]))
		}
		buf.WriteString(renderTmpl(groupFriendPickerItemTmpl, gpItemData{
			PickerID: pickerID,
			ID:       g.ID,
			Name:     g.Name,
			Initials: ini,
			Color:    hexColors[i%len(hexColors)],
		}))
	}
	return buf.String()
}

// groupPickerSectionHTML renders a collapsible groups section below the friends picker.
// Clicking a group row fetches its members via JS and injects them as selectable fp-items.
func groupPickerSectionHTML(pickerID string, groups []models.Group) string {
	if len(groups) == 0 {
		return ""
	}
	hexColors := []string{"#c084fc", "#22d3ee", "#4ade80", "#facc15", "#fb7185", "#db2777"}
	var rowsBuf strings.Builder
	for i, g := range groups {
		runes := []rune(g.Name)
		ini := ""
		if len(runes) >= 2 {
			ini = strings.ToUpper(string(runes[0])) + strings.ToUpper(string(runes[1]))
		} else if len(runes) == 1 {
			ini = strings.ToUpper(string(runes[0]))
		}
		rowsBuf.WriteString(renderTmpl(groupPickerRowTmpl, gpRowData{
			PickerID: pickerID,
			ID:       g.ID,
			Name:     g.Name,
			Initials: ini,
			Color:    hexColors[i%len(hexColors)],
		}))
	}
	return `<div class="gp-section">` +
		`<div class="gp-label">Groups</div>` +
		rowsBuf.String() +
		`</div>`
}

// paySheetHTML renders the bottom-sheet HTML for the payment action with an ATM-style
// amount display and a two-section recipient picker (Friends + Groups) in both panes.
func paySheetHTML(friends []models.Profile, groups []models.Group, availableCreditCents int) string {
	payPicker       := friendPickerHTML("pay-recv", friends)
	bnplPicker      := friendPickerHTML("bnpl-recv", friends)
	payGroupPicker  := groupFriendPickerHTML("pay-recv", groups)
	bnplGroupPicker := groupFriendPickerHTML("bnpl-recv", groups)

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
      <div class="amount-big mono atm-wrap" data-atm="pay-send" id="pay-send-atm" tabindex="0" onclick="this.focus()">
        <span class="dollar">$</span><span id="pay-send-display">0.00</span><span class="atm-cursor"></span>
      </div>
      <div class="chip-row" style="justify-content:center;">
        <button type="button" class="chip" onclick="setAtmCents('pay-send',1000)">$10</button>
        <button type="button" class="chip" onclick="setAtmCents('pay-send',2500)">$25</button>
        <button type="button" class="chip" onclick="setAtmCents('pay-send',5000)">$50</button>
        <button type="button" class="chip" onclick="setAtmCents('pay-send',10000)">$100</button>
      </div>
      <div>
        <div style="font-size:17px;font-weight:700;color:var(--text);letter-spacing:-0.01em;margin-bottom:14px;">Choose a Recipient</div>
        <label class="modal-lbl">Friends</label>
        <div class="friend-picker" id="pay-recv-picker">` + payPicker + `</div>
        <input type="hidden" id="pay-recv-val" value="" />
        <label class="modal-lbl" style="margin-top:14px;display:block;">Groups</label>
        <div class="friend-picker" id="pay-recv-group-picker">` + payGroupPicker + `</div>
      </div>
      <div><label class="modal-lbl">Note</label><input class="modal-inp" id="pay-note-inp" type="text" placeholder="Dinner, rent, etc." /></div>
      <button class="submit-btn" onclick="submitSendPayment()">Send payment</button>
    </div>

    <div class="sheet-pane" data-pay-pane="bnpl">
      <div class="info-card">
        <span class="label">Available credit</span>
        <span class="val mono">$` + fmt.Sprintf("%d.%02d", availableCreditCents/100, availableCreditCents%100) + `</span>
      </div>
      <div class="amount-big mono atm-wrap" data-atm="bnpl" id="bnpl-atm" tabindex="0" onclick="this.focus()">
        <span class="dollar">$</span><span id="bnpl-display">0.00</span><span class="atm-cursor"></span>
      </div>
      <div class="chip-row" style="justify-content:center;">
        <button type="button" class="chip" onclick="setAtmCents('bnpl',1000)">$10</button>
        <button type="button" class="chip" onclick="setAtmCents('bnpl',2000)">$20</button>
        <button type="button" class="chip" onclick="setAtmCents('bnpl',5000)">$50</button>
        <button type="button" class="chip" onclick="setAtmCents('bnpl',10000)">$100</button>
      </div>
      <div>
        <div style="font-size:17px;font-weight:700;color:var(--text);letter-spacing:-0.01em;margin-bottom:14px;">Choose a Recipient</div>
        <label class="modal-lbl">Friends</label>
        <div class="friend-picker" id="bnpl-recv-picker">` + bnplPicker + `</div>
        <input type="hidden" id="bnpl-recv-val" value="" />
        <label class="modal-lbl" style="margin-top:14px;display:block;">Groups</label>
        <div class="friend-picker" id="bnpl-recv-group-picker">` + bnplGroupPicker + `</div>
      </div>
      <div id="bnpl-breakdown" style="display:none;background:var(--surface-3);border:1px solid var(--border);border-radius:12px;padding:14px 16px;margin-top:4px;">
        <div style="font-size:11px;font-weight:700;color:var(--text-mute);text-transform:uppercase;letter-spacing:0.06em;margin-bottom:10px;">Pay-in-4 breakdown</div>
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
          <span style="font-size:13px;color:var(--text-dim);">Amount due now</span>
          <span id="bnpl-due-now" class="mono" style="font-size:14px;font-weight:700;color:var(--emerald-hi);">$0.00</span>
        </div>
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
          <span style="font-size:13px;color:var(--text-dim);">3 × every 2 weeks</span>
          <span id="bnpl-recurring" class="mono" style="font-size:14px;font-weight:700;color:var(--text);">$0.00</span>
        </div>
        <div style="display:flex;justify-content:space-between;align-items:center;padding-top:8px;border-top:1px solid var(--border-soft);">
          <span style="font-size:13px;color:var(--text-dim);">Interest rate</span>
          <span id="bnpl-rate-display" class="mono" style="font-size:13px;font-weight:700;color:var(--amber-hi);">0%</span>
        </div>
      </div>
      <input type="hidden" id="bnpl-total-cents" value="0" />
      <div><label class="modal-lbl">Note</label><input class="modal-inp" id="bnpl-note-inp" type="text" placeholder="Item or purchase description" /></div>
      <button class="submit-btn emerald" onclick="submitBNPL()">Approve plan</button>
    </div>
  </div>
</div>`
}

// requestSheetHTML renders the bottom-sheet HTML for creating a money request
// with an ATM-style amount display and a two-section recipient picker (Friends + Groups).
func requestSheetHTML(friends []models.Profile, groups []models.Group) string {
	reqPicker := friendPickerHTML("req-from", friends)
	reqGroupPicker := groupFriendPickerHTML("req-from", groups)

	return `
<div class="sheet-backdrop" id="request-sheet" onclick="if(event.target===this) closeSheet('request')">
  <div class="sheet">
    <div class="sheet-handle"></div>
    <div class="sheet-title">Request money</div>
    <div class="sheet-pane active" style="display:flex;">
      <div class="amount-big mono atm-wrap" data-atm="req" id="req-atm" tabindex="0" onclick="this.focus()">
        <span class="dollar">$</span><span id="req-display">0.00</span><span class="atm-cursor"></span>
      </div>
      <div class="chip-row" style="justify-content:center;">
        <button type="button" class="chip" onclick="setAtmCents('req',1000)">$10</button>
        <button type="button" class="chip" onclick="setAtmCents('req',2500)">$25</button>
        <button type="button" class="chip" onclick="setAtmCents('req',5000)">$50</button>
        <button type="button" class="chip" onclick="setAtmCents('req',10000)">$100</button>
      </div>
      <div>
        <div style="font-size:17px;font-weight:700;color:var(--text);letter-spacing:-0.01em;margin-bottom:14px;">Choose a Recipient</div>
        <label class="modal-lbl">Friends</label>
        <div class="friend-picker" id="req-from-picker">` + reqPicker + `</div>
        <input type="hidden" id="req-from-val" value="" />
        <label class="modal-lbl" style="margin-top:14px;display:block;">Groups</label>
        <div class="friend-picker" id="req-from-group-picker">` + reqGroupPicker + `</div>
      </div>
      <div><label class="modal-lbl">Note</label><input class="modal-inp" id="req-note-inp" type="text" placeholder="What's it for?" /></div>
      <button class="submit-btn amber" onclick="submitRequest()">Send request</button>
    </div>
  </div>
</div>`
}
