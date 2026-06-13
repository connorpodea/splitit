package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/connorpodea/splitit/internal/models"
)

// walletDashboard holds pre-fetched data for the server-rendered wallet tab.
type walletDashboard struct {
	SentCents        int
	RecvCents        int
	NetCents         int
	Utilization      float64
	OutstandingCents int
	OverdueCents     int
	SettledCents     int
	Transactions     []models.WalletTransaction
}

// analyticsDashboard holds pre-fetched data for the server-rendered analytics tab.
type analyticsDashboard struct {
	Recipients    []models.TopRecipient
	Monthly       *models.MonthlySummary
	CreditHistory []models.CreditScoreLog
}

func (h *Handler) GetInitialView(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		writeHTML(w, loginHTML())
		return
	}
	userID, ok := h.sessions.lookup(cookie.Value)
	if !ok {
		writeHTML(w, loginHTML())
		return
	}
	h.serveDashboard(w, userID)
}

func (h *Handler) GetDashboardView(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		writeHTML(w, loginHTML())
		return
	}
	userID, ok := h.sessions.lookup(cookie.Value)
	if !ok {
		writeHTML(w, loginHTML())
		return
	}
	h.serveDashboard(w, userID)
}

// serveDashboard loads all tab data for userID and renders the full dashboard.
// Core data failures are logged; the page degrades gracefully rather than hard-erroring.
func (h *Handler) serveDashboard(w http.ResponseWriter, userID string) {
	user, err := h.store.GetUser(userID)
	if err != nil || !user.IsActive {
		writeHTML(w, loginHTML())
		return
	}

	warn := func(label string, err error) {
		if err != nil {
			log.Printf("[WARN] dashboard %s user=%s: %v", label, userID, err)
		}
	}

	friends, err := h.store.ListFriends(user.ID)
	warn("ListFriends", err)
	installments, err := h.store.ListInstallmentDetailsForUser(user.ID)
	warn("ListInstallments", err)
	overdueInstallments, err := h.store.ListOverdueInstallmentDetailsForUser(user.ID)
	warn("ListOverdueInstallments", err)
	incomingRequests, err := h.store.ListIncomingPaymentRequests(user.ID)
	warn("ListIncomingRequests", err)
	friendRequests, err := h.store.ListIncomingFriendRequests(user.ID)
	warn("ListFriendRequests", err)
	notifications, err := h.store.ListNotifications(user.ID)
	warn("ListNotifications", err)
	groups, err := h.store.ListGroups(user.ID)
	warn("ListGroups", err)
	groupInvitations, err := h.store.ListIncomingGroupInvitations(user.ID)
	warn("ListGroupInvitations", err)
	activityFeed, err := h.store.GetUnifiedActivity(user.ID, 50)
	warn("GetUnifiedActivity", err)
	memberCounts, err := h.store.GetGroupMemberCounts(user.ID)
	warn("GetGroupMemberCounts", err)

	since30d := time.Now().AddDate(0, -1, 0)
	walletTotals, _ := h.store.GetSpendingTotals(user.ID, since30d)
	walletUtil, _ := h.store.GetBNPLUtilization(user.ID)
	walletSummary, _ := h.store.GetInstallmentSummary(user.ID)
	walletTxns, _ := h.store.ListWalletTransactions(user.ID)
	since90d := time.Now().AddDate(0, -3, 0)
	now := time.Now()
	analyticsRecipients, _ := h.store.GetTopRecipients(user.ID, 5, since90d)
	analyticsMonthly, _ := h.store.GetMonthlySpendingSummary(user.ID, now.Year(), now.Month())
	analyticsCreditHistory, _ := h.store.GetCreditScoreHistory(user.ID)
	userSettings, _ := h.store.GetUserSettings(user.ID)

	if friends == nil {
		friends = []models.Profile{}
	}
	if installments == nil {
		installments = []models.InstallmentDetail{}
	}
	if overdueInstallments == nil {
		overdueInstallments = []models.InstallmentDetail{}
	}
	if incomingRequests == nil {
		incomingRequests = []models.PaymentRequest{}
	}
	if friendRequests == nil {
		friendRequests = []models.FriendRequest{}
	}
	if notifications == nil {
		notifications = []models.Notification{}
	}
	if groups == nil {
		groups = []models.Group{}
	}
	if groupInvitations == nil {
		groupInvitations = []models.GroupInvitation{}
	}
	if activityFeed == nil {
		activityFeed = []models.ActivityItem{}
	}
	if memberCounts == nil {
		memberCounts = map[string]int{}
	}
	wDash := &walletDashboard{}
	if walletTotals != nil {
		wDash.SentCents = walletTotals.TotalSentCents
		wDash.RecvCents = walletTotals.TotalReceivedCents
		wDash.NetCents = walletTotals.NetCents
	}
	wDash.Utilization = walletUtil
	if walletSummary != nil {
		wDash.OutstandingCents = walletSummary.TotalOutstandingCents
		wDash.OverdueCents = walletSummary.TotalOverdueCents
		wDash.SettledCents = walletSummary.TotalSettledCents
	}
	if walletTxns == nil {
		walletTxns = []models.WalletTransaction{}
	}
	wDash.Transactions = walletTxns
	aDash := &analyticsDashboard{
		Recipients:    analyticsRecipients,
		Monthly:       analyticsMonthly,
		CreditHistory: analyticsCreditHistory,
	}
	if aDash.Recipients == nil {
		aDash.Recipients = []models.TopRecipient{}
	}
	if aDash.CreditHistory == nil {
		aDash.CreditHistory = []models.CreditScoreLog{}
	}

	writeHTML(w, dashboardHTML(user, friends, installments, overdueInstallments, incomingRequests, friendRequests, notifications, groups, groupInvitations, activityFeed, memberCounts, wDash, aDash, userSettings))
}
