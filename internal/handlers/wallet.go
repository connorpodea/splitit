package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// GetWalletDashboard returns a composite wallet snapshot: spending totals, BNPL utilization,
// installment summary, top recipients, and recent wallet transactions — all in one response.
func (h *Handler) GetWalletDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	since := time.Now().AddDate(0, -1, 0)

	totals, err := h.store.GetSpendingTotals(userID, since)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	utilization, err := h.store.GetBNPLUtilization(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	summary, err := h.store.GetInstallmentSummary(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	topRecipients, err := h.store.GetTopRecipients(userID, 5, since)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	transactions, err := h.store.ListWalletTransactions(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"spending_totals":     totals,
		"bnpl_utilization":    utilization,
		"installment_summary": summary,
		"top_recipients":      topRecipients,
		"transactions":        transactions,
	})
}

// Deposit credits a specified amount to the authenticated user's cash balance and records
// the event in the wallet_transactions ledger.
func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		AmountCents int `json:"amount_cents"`
	}
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.AmountCents <= 0 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing or invalid amount_cents — must be a positive integer"})
		return
	}

	if err := h.store.Deposit(userID, input.AmountCents); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Withdraw debits a specified amount from the authenticated user's cash balance after
// liquid funds verification, and records the event in the wallet_transactions ledger.
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only POST requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	type Input struct {
		AmountCents int `json:"amount_cents"`
	}
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.AmountCents <= 0 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing or invalid amount_cents — must be a positive integer"})
		return
	}

	if err := h.store.Withdraw(userID, input.AmountCents); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ListWalletTransactions returns the absolute balance mutation history (deposits and withdrawals)
// for the authenticated user, separate from the peer-to-peer payments ledger.
func (h *Handler) ListWalletTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	txns, err := h.store.ListWalletTransactions(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, txns)
}

// GetSpendingTotals returns total sent, received, and net cash flow for the authenticated user
// over a caller-specified window in days (defaults to 30 via the ?days= query parameter).
func (h *Handler) GetSpendingTotals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	since := time.Now().AddDate(0, 0, -days)
	totals, err := h.store.GetSpendingTotals(userID, since)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, totals)
}

// GetMonthlySpendingSummary returns absolute spending, income, and active BNPL charge totals
// for the current calendar month for the authenticated user.
func (h *Handler) GetMonthlySpendingSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	now := time.Now()
	summary, err := h.store.GetMonthlySpendingSummary(userID, now.Year(), now.Month())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, summary)
}

// GetTopRecipients returns the top 5 peer-to-peer payment targets ranked by total funds
// directed toward them over the last 90 days for the authenticated user.
func (h *Handler) GetTopRecipients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	since := time.Now().AddDate(0, -3, 0)
	recipients, err := h.store.GetTopRecipients(userID, 5, since)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, recipients)
}

// GetBNPLUtilization returns the ratio of outstanding installment debt to the authenticated
// user's master credit limit as a decimal utilization percentage.
func (h *Handler) GetBNPLUtilization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	utilization, err := h.store.GetBNPLUtilization(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]float64{"utilization": utilization})
}

// GetCreditScoreHistory returns the chronological audit log of credit score adjustment
// events for the authenticated user from the credit_score_log table.
func (h *Handler) GetCreditScoreHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Only GET requests are permitted on this route"})
		return
	}
	userID, authorized := h.authenticateSession(w, r)
	if !authorized {
		return
	}

	history, err := h.store.GetCreditScoreHistory(userID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}

	WriteJSON(w, http.StatusOK, history)
}
