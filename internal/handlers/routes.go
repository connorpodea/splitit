package handlers

import "net/http"

// RegisterRoutes attaches all application endpoints to Go's standard HTTP router.
func (h *Handler) RegisterRoutes() {
	// Rate limiter shared across all auth endpoints: 5 req/min per IP, burst 5.
	authLimit := newAuthRateLimiter()

	// UI Content Screens
	http.HandleFunc("/ui/initial-view", h.GetInitialView)
	http.HandleFunc("/ui/register-view", h.GetRegistrationView)
	http.HandleFunc("/ui/dashboard-view", h.GetDashboardView)

	// Users & Profiles — login and register are rate-limited at the edge.
	// http.Handle is used here instead of http.HandleFunc because authLimit returns an
	// http.Handler interface (a middleware wrapper), not a plain function. http.HandlerFunc
	// converts the plain handler method into that interface so the middleware can wrap it.
	http.Handle("/users/create", authLimit(http.HandlerFunc(h.CreateUser)))
	http.Handle("/users/login", authLimit(http.HandlerFunc(h.LoginUser)))
	http.HandleFunc("/users/logout", h.Logout)
	http.HandleFunc("/users/get", h.GetUser)
	http.HandleFunc("/users/list", h.ListUsers)
	http.HandleFunc("/users/update/password", h.UpdatePassword)
	http.HandleFunc("/users/update/email", h.UpdateEmail)
	http.HandleFunc("/users/update/phone", h.UpdatePhoneNumber)
	http.HandleFunc("/users/update/name", h.UpdateDisplayName)
	http.HandleFunc("/users/update/color", h.UpdateProfileColor)
	http.HandleFunc("/users/deactivate", h.DeactivateAccount)
	http.HandleFunc("/profiles/list", h.ListProfiles)
	http.HandleFunc("/profiles/get", h.GetProfile)
	http.HandleFunc("/profiles/search", h.SearchProfiles)

	// Payments
	http.HandleFunc("/payments/pay", h.Pay)
	http.HandleFunc("/payments/get", h.GetPayment)
	http.HandleFunc("/payments/received", h.ListPaymentsReceived)
	http.HandleFunc("/payments/sent", h.ListPaymentsSent)
	http.HandleFunc("/payments/history", h.ListPaymentsByUser)
	http.HandleFunc("/payments/between", h.ListPaymentsBetweenUsers)
	http.HandleFunc("/payments/request/create", h.CreatePaymentRequest)
	http.HandleFunc("/payments/request/fulfill", h.FulfillPaymentRequest)
	http.HandleFunc("/payments/request/incoming", h.ListIncomingPaymentRequests)
	http.HandleFunc("/payments/request/outgoing", h.ListOutgoingPaymentRequests)
	http.HandleFunc("/payments/request/update", h.UpdatePaymentRequestStatus)
	http.HandleFunc("/payments/activity", h.GetUnifiedActivity)

	// BNPL Loans & Installments
	http.HandleFunc("/bnpl/loan/create", h.CreateBNPLLoan)
	http.HandleFunc("/bnpl/loan/detail", h.GetPaymentWithInstallments)
	http.HandleFunc("/bnpl/installment/pay", h.PayInstallment)
	http.HandleFunc("/bnpl/installments/list", h.ListInstallments)
	http.HandleFunc("/bnpl/installments/overdue", h.ListOverdueInstallments)
	http.HandleFunc("/bnpl/installments/summary", h.GetInstallmentSummary)

	// Friends System
	http.HandleFunc("/friends/request/send", h.SendFriendRequest)
	http.HandleFunc("/friends/request/incoming", h.ListIncomingFriendRequests)
	http.HandleFunc("/friends/request/outgoing", h.ListOutgoingFriendRequests)
	http.HandleFunc("/friends/request/accept", h.AcceptFriendRequest)
	http.HandleFunc("/friends/request/decline", h.DeclineFriendRequest)
	http.HandleFunc("/friends/remove", h.RemoveFriendMutual)
	http.HandleFunc("/friends/list", h.ListFriends)

	// Groups
	http.HandleFunc("/groups/create", h.CreateGroup)
	http.HandleFunc("/groups/list", h.ListGroups)
	http.HandleFunc("/groups/members", h.ListGroupMembers)
	http.HandleFunc("/groups/invite/send", h.SendGroupInvitation)
	http.HandleFunc("/groups/invite/accept", h.AcceptGroupInvitation)
	http.HandleFunc("/groups/invite/decline", h.DeclineGroupInvitation)
	http.HandleFunc("/groups/invite/incoming", h.ListIncomingGroupInvitations)
	http.HandleFunc("/groups/invite/outgoing", h.ListOutgoingGroupInvitations)
	http.HandleFunc("/groups/member/remove", h.RemoveGroupMember)
	http.HandleFunc("/groups/leave", h.LeaveGroup)
	http.HandleFunc("/groups/activity", h.ListGroupActivity)

	// Notifications
	http.HandleFunc("/notifications/list", h.ListNotifications)
	http.HandleFunc("/notifications/count", h.CountUnseenNotifications)
	http.HandleFunc("/notifications/seen", h.MarkNotificationSeen)
	http.HandleFunc("/notifications/clear", h.ClearAllNotifications)

	// Wallet
	http.HandleFunc("/wallet/dashboard", h.GetWalletDashboard)
	http.HandleFunc("/wallet/deposit", h.Deposit)
	http.HandleFunc("/wallet/withdraw", h.Withdraw)
	http.HandleFunc("/wallet/transactions", h.ListWalletTransactions)
	http.HandleFunc("/wallet/spending/totals", h.GetSpendingTotals)
	http.HandleFunc("/wallet/spending/monthly", h.GetMonthlySpendingSummary)
	http.HandleFunc("/wallet/recipients/top", h.GetTopRecipients)
	http.HandleFunc("/wallet/bnpl/utilization", h.GetBNPLUtilization)
	http.HandleFunc("/wallet/credit/history", h.GetCreditScoreHistory)

	// Settings
	http.HandleFunc("/settings/get", h.GetUserSettings)
	http.HandleFunc("/settings/update", h.UpdateUserSettings)
}
