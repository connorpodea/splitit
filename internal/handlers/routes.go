package handlers

import "net/http"

// RegisterRoutes attaches all application endpoints to Go's standard HTTP router.
func (h *Handler) RegisterRoutes() {
	// Users & Profiles
	http.HandleFunc("/users/create", h.CreateUser)
	http.HandleFunc("/users/login", h.LoginUser)
	http.HandleFunc("/users/logout", h.Logout)
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

	// Notifications
	http.HandleFunc("/notifications/list", h.ListNotifications)
	http.HandleFunc("/notifications/seen", h.MarkNotificationSeen)
	http.HandleFunc("/notifications/clear", h.ClearAllNotifications)

	// UI Content Screens
	http.HandleFunc("/ui/initial-view", h.GetInitialView)
	http.HandleFunc("/ui/register-view", h.GetRegistrationView)
	http.HandleFunc("/ui/dashboard-view", h.GetDashboardView)
}
