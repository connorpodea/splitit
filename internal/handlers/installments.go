package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/connorpodea/splitit/internal/models"
)

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

func viewHome(name string, user *models.User, overdueInstallments []models.Installment, installments []models.Installment, incomingRequests []models.PaymentRequest) string {
	firstName := strings.Fields(name)
	greet := "there"
	if len(firstName) > 0 {
		greet = firstName[0]
	}

	balanceWhole := int(user.Balance)
	balanceCents := int((user.Balance-float64(balanceWhole))*100 + 0.5)

	var outstandingBNPL float64
	splitIDs := make(map[string]struct{})
	for _, inst := range installments {
		if !inst.IsPaid {
			outstandingBNPL += inst.Amount
			splitIDs[inst.PaymentID] = struct{}{}
		}
	}
	activeSplits := len(splitIDs)

	overdueCount := len(overdueInstallments)
	overdueStatClass := "qstat"
	overdueStatVal := "0 bills"
	if overdueCount > 0 {
		overdueStatClass = "qstat warn"
		overdueStatVal = fmt.Sprintf("%d bills", overdueCount)
	}

	activeCount := 0
	for _, inst := range installments {
		if !inst.IsPaid {
			isOverdue := false
			for _, ov := range overdueInstallments {
				if ov.ID == inst.ID {
					isOverdue = true
					break
				}
			}
			if !isOverdue {
				activeCount++
			}
		}
	}

	// Build recent activity rows — incoming requests first, then installments, cap at 5.
	recentRows := ""
	count := 0
	for i, req := range incomingRequests {
		if count >= 5 {
			break
		}
		cls := avatarClass(i)
		ini := strings.ToUpper(req.RequesterID)
		if len([]rune(ini)) > 2 {
			ini = string([]rune(ini)[:2])
		}
		date := req.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		recentRows += `
    <div class="row">
      <div class="row-avatar ` + cls + `">` + ini + `</div>
      <div class="row-body">
        <div class="row-title"><b>@` + req.RequesterID + `</b> requested from <b>you</b></div>
        <div class="row-sub">` + req.Note + `</div>
      </div>
      <div class="row-right"><div class="row-amt req mono">$` + fmt.Sprintf("%.2f", req.Amount) + `</div><div class="row-time">` + date + `</div></div>
    </div>`
		count++
	}
	for i, inst := range installments {
		if count >= 5 {
			break
		}
		cls := avatarClass(i)
		statusLabel := "Due " + inst.DueDate
		amtClass := "bnpl"
		if inst.IsPaid {
			statusLabel = "Paid"
			amtClass = "pos"
		}
		recentRows += `
    <div class="row">
      <div class="row-avatar ` + cls + `">BN</div>
      <div class="row-body">
        <div class="row-title"><b>BNPL</b> installment <span class="pill">Pay-in-4</span></div>
        <div class="row-sub">` + statusLabel + `</div>
      </div>
      <div class="row-right"><div class="row-amt ` + amtClass + ` mono">$` + fmt.Sprintf("%.2f", inst.Amount) + `</div><div class="row-time">` + inst.DueDate + `</div></div>
    </div>`
		count++
	}
	if recentRows == "" {
		recentRows = `
    <div style="text-align:center; padding:28px 16px; color:var(--text-mute); font-size:13px;">No recent activity yet.</div>`
	}

	return `
<section class="view active" data-view="home">
  <div class="greeting" style="font-size:14px; color:var(--text-mute); margin-bottom:14px;">Welcome back, <b style="color:var(--text);">` + greet + `</b></div>

  <div class="hero">
    <div class="hero-label">Available balance</div>
    <div class="hero-amount mono">$` + fmt.Sprintf("%d", balanceWhole) + `<span class="cents">.` + fmt.Sprintf("%02d", balanceCents) + `</span></div>
    <div class="hero-meta">
      <span><b>$` + fmt.Sprintf("%.2f", outstandingBNPL) + `</b> outstanding</span>
      <span style="opacity:.4;">·</span>
      <span><b>` + fmt.Sprintf("%d", activeSplits) + `</b> active splits</span>
    </div>
  </div>

  <div class="actions">
    <button class="action req" onclick="openSheet('request')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 5v14"/><path d="M5 12h14"/></svg></span>
      <span class="lbl-stack"><b>Request</b><small>Ask a friend</small></span>
    </button>
    <button class="action pay" onclick="openSheet('pay')">
      <span class="ico"><svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg></span>
      <span class="lbl-stack"><b>Pay</b><small>Send · BNPL</small></span>
    </button>
  </div>

  <div class="quick-strip">
    <button class="` + overdueStatClass + `" onclick="goView('activity', 'overdue')">
      <div>
        <div class="qstat-label">Overdue</div>
        <div class="qstat-val mono">` + overdueStatVal + `</div>
      </div>
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
    <button class="qstat" onclick="goView('activity', 'active')">
      <div>
        <div class="qstat-label">Active plans</div>
        <div class="qstat-val mono">` + fmt.Sprintf("%d", activeCount) + `</div>
      </div>
      <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 18l6-6-6-6"/></svg>
    </button>
  </div>

  <div class="section-row">
    <h2>Recent activity</h2>
    <button class="linklike" onclick="goView('activity')">See all</button>
  </div>
  <div class="card">` + recentRows + `</div>
</section>
`
}

func viewActivity(installments []models.Installment, overdueInstallments []models.Installment, incomingRequests []models.PaymentRequest) string {
	// Build overdue ID set for fast lookup.
	overdueIDs := make(map[string]bool)
	for _, inst := range overdueInstallments {
		overdueIDs[inst.ID] = true
	}

	// --- Payments pane: incoming payment requests ---
	paymentsRows := ""
	for i, req := range incomingRequests {
		cls := avatarClass(i)
		ini := strings.ToUpper(req.RequesterID)
		if len([]rune(ini)) > 2 {
			ini = string([]rune(ini)[:2])
		}
		date := req.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		statusLabel := req.Note
		if statusLabel == "" {
			statusLabel = req.Status
		} else {
			statusLabel += " · " + req.Status
		}
		paymentsRows += `
      <div class="row">
        <div class="row-avatar ` + cls + `">` + ini + `</div>
        <div class="row-body"><div class="row-title"><b>@` + req.RequesterID + `</b> requested</div><div class="row-sub">` + statusLabel + `</div></div>
        <div class="row-right"><div class="row-amt req mono">$` + fmt.Sprintf("%.2f", req.Amount) + `</div><div class="row-time">` + date + `</div></div>
      </div>`
	}
	if paymentsRows == "" {
		paymentsRows = `<div style="text-align:center; padding:28px 16px; color:var(--text-mute); font-size:13px;">No incoming payment requests.</div>`
	}

	// --- Active pane: non-overdue, unpaid installments ---
	activeRows := ""
	activeIdx := 0
	for _, inst := range installments {
		if inst.IsPaid || overdueIDs[inst.ID] {
			continue
		}
		cls := avatarClass(activeIdx)
		activeRows += `
      <div class="row">
        <div class="row-avatar ` + cls + `">BN</div>
        <div class="row-body">
          <div class="row-title"><b>BNPL installment</b> <span class="pill">Pay-in-4</span></div>
          <div class="row-sub">Due ` + inst.DueDate + `</div>
          <div class="progress"><span style="width:50%;"></span></div>
        </div>
        <div class="row-right"><div class="row-amt bnpl mono">$` + fmt.Sprintf("%.2f", inst.Amount) + `</div><div class="row-time">/installment</div></div>
      </div>`
		activeIdx++
	}
	if activeRows == "" {
		activeRows = `
      <div style="text-align:center; padding:28px 16px; color:var(--text-mute); font-size:13px;">No active BNPL plans.</div>`
	}

	// --- Overdue pane ---
	overdueSubtabBadge := ""
	if len(overdueInstallments) > 0 {
		overdueSubtabBadge = ` <span class="badge mono">` + fmt.Sprintf("%d", len(overdueInstallments)) + `</span>`
	}

	overdueContent := ""
	if len(overdueInstallments) == 0 {
		overdueContent = `
    <div class="empty">
      <div class="empty-icon success">
        <svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>
      </div>
      <div class="empty-title">You're all caught up</div>
      <div class="empty-sub">No overdue bills. Your Splitit Score is safe.</div>
    </div>`
	} else {
		var overdueTotal float64
		for _, inst := range overdueInstallments {
			overdueTotal += inst.Amount
		}
		overdueRows := ""
		for i, inst := range overdueInstallments {
			_ = i
			overdueRows += `
        <div class="row overdue-row" data-installment-id="` + inst.ID + `" data-payment-id="` + inst.PaymentID + `" data-amount="` + fmt.Sprintf("%.2f", inst.Amount) + `">
          <div class="row-avatar av-rose">OD</div>
          <div class="row-body">
            <div class="row-title"><b>BNPL installment</b> <span class="pill warn">Overdue</span></div>
            <div class="row-sub">Due ` + inst.DueDate + `</div>
            <div class="progress warn"><span style="width:25%;"></span></div>
          </div>
          <div class="row-right">
            <button onclick="payInstallment('` + inst.ID + `','` + inst.PaymentID + `',` + fmt.Sprintf("%.2f", inst.Amount) + `)"
              style="background:var(--rose);color:#fff;border:none;border-radius:8px;padding:6px 12px;font-size:12px;font-weight:700;cursor:pointer;font-family:inherit;">
              Pay $` + fmt.Sprintf("%.2f", inst.Amount) + `
            </button>
          </div>
        </div>`
		}
		overdueContent = `
    <div class="overdue-banner">
      <svg width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      <div class="ob-text">
        <strong>$` + fmt.Sprintf("%.2f", overdueTotal) + ` overdue across ` + fmt.Sprintf("%d", len(overdueInstallments)) + ` bills</strong>
        <div>Late fees may apply. Pay now to keep your Splitit Score intact.</div>
      </div>
      <button onclick="payAllOverdue()">Pay all</button>
    </div>
    <div class="card">` + overdueRows + `</div>`
	}

	return `
<section class="view" data-view="activity">
  <div class="section-row" style="margin-top:0;">
    <h2 style="font-size:22px; font-weight:800; letter-spacing:-0.02em;">Activity</h2>
  </div>

  <div class="subtabs" role="tablist">
    <button class="subtab active" data-pane="all" onclick="goPane(this)">Payments</button>
    <button class="subtab" data-pane="active" onclick="goPane(this)">Active</button>
    <button class="subtab" data-pane="overdue" onclick="goPane(this)">Overdue` + overdueSubtabBadge + `</button>
  </div>

  <div data-pane-content="all" class="pane-content">
    <div class="card">` + paymentsRows + `</div>
  </div>

  <div data-pane-content="active" class="pane-content" style="display:none;">
    <div class="card">` + activeRows + `</div>
  </div>

  <div data-pane-content="overdue" class="pane-content" style="display:none;">
    ` + overdueContent + `
  </div>
</section>
`
}
