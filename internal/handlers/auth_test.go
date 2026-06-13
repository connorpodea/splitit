package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// authScenario is one credential failure mode to exercise against every protected endpoint.
type authScenario struct {
	name   string
	cookie *http.Cookie // nil means no cookie is attached to the request at all
}

// authRejectionScenarios returns the three distinct ways a caller can be unauthenticated:
//
//	no_cookie    – request carries no session_token cookie whatsoever
//	empty_cookie – session_token cookie is present but its Value is ""
//	forged_token – a 64-hex-char token that was never issued by this server's session store
func authRejectionScenarios() []authScenario {
	return []authScenario{
		{name: "no_cookie", cookie: nil},
		{name: "empty_cookie", cookie: &http.Cookie{Name: "session_token", Value: ""}},
		{name: "forged_token", cookie: &http.Cookie{Name: "session_token", Value: strings.Repeat("ab", 32)}},
	}
}

// assert401 is the shared assertion for every auth-rejection subtest. It checks:
//  1. Status code is exactly 401 (not 403, 500, or a redirect).
//  2. Response body is valid JSON containing a non-empty "error" field.
//  3. The error message does not leak internal implementation details.
func assert401(t *testing.T, rr *httptest.ResponseRecorder, scenario, path string) {
	t.Helper()
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("%-13s %-44s  got %d, want 401", scenario, path, rr.Code)
		return
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Errorf("%-13s %-44s  response body is not valid JSON: %v", scenario, path, err)
		return
	}
	if body["error"] == "" {
		t.Errorf("%-13s %-44s  JSON body missing non-empty 'error' field", scenario, path)
	}
	// Ensure no raw Go or SQL internals appear in the error string returned to the client.
	for _, leak := range []string{"sql:", "database/sql", "panic:", "goroutine ", ".go:", "runtime error"} {
		if strings.Contains(body["error"], leak) {
			t.Errorf("%-13s %-44s  error message leaks internals: %q", scenario, path, body["error"])
		}
	}
}

// TestAuth_AllProtectedEndpoints_Require401 is an exhaustive contract test asserting that
// every endpoint guarded by authenticateSession rejects unauthenticated callers with
// HTTP 401 and a safe, non-leaking JSON error body.
//
// Three credential failure modes are exercised per endpoint (186 subtests total):
//
//	no_cookie    – no session_token cookie on the request
//	empty_cookie – session_token cookie value is the empty string
//	forged_token – a random-looking 64-hex token unknown to the server's session store
//
// Auth rejection happens before any store access, so all subtests share one handler
// instance and run in parallel safely.
func TestAuth_AllProtectedEndpoints_Require401(t *testing.T) {
	type endpoint struct {
		method  string
		path    string
		handler http.HandlerFunc
	}

	h, _ := newTestHandler(t)

	endpoints := []endpoint{
		// ---- Payments -------------------------------------------------------
		{http.MethodPost, "/payments/pay", h.Pay},
		{http.MethodGet, "/payments/get", h.GetPayment},
		{http.MethodGet, "/payments/received", h.ListPaymentsReceived},
		{http.MethodGet, "/payments/sent", h.ListPaymentsSent},
		{http.MethodGet, "/payments/history", h.ListPaymentsByUser},
		{http.MethodGet, "/payments/between", h.ListPaymentsBetweenUsers},
		{http.MethodPost, "/payments/request/create", h.CreatePaymentRequest},
		{http.MethodGet, "/payments/request/incoming", h.ListIncomingPaymentRequests},
		{http.MethodGet, "/payments/request/outgoing", h.ListOutgoingPaymentRequests},
		{http.MethodPost, "/payments/request/update", h.UpdatePaymentRequestStatus},
		{http.MethodPost, "/payments/request/fulfill", h.FulfillPaymentRequest},
		{http.MethodGet, "/payments/activity", h.GetUnifiedActivity},

		// ---- BNPL & Installments -------------------------------------------
		{http.MethodPost, "/bnpl/loan/create", h.CreateBNPLLoan},
		{http.MethodGet, "/bnpl/loan/detail", h.GetPaymentWithInstallments},
		{http.MethodPost, "/bnpl/installment/pay", h.PayInstallment},
		{http.MethodGet, "/bnpl/installments/list", h.ListInstallments},
		{http.MethodGet, "/bnpl/installments/overdue", h.ListOverdueInstallments},
		{http.MethodGet, "/bnpl/installments/summary", h.GetInstallmentSummary},

		// ---- Friends --------------------------------------------------------
		{http.MethodPost, "/friends/request/send", h.SendFriendRequest},
		{http.MethodGet, "/friends/request/incoming", h.ListIncomingFriendRequests},
		{http.MethodGet, "/friends/request/outgoing", h.ListOutgoingFriendRequests},
		{http.MethodPost, "/friends/request/accept", h.AcceptFriendRequest},
		{http.MethodPost, "/friends/request/decline", h.DeclineFriendRequest},
		{http.MethodPost, "/friends/remove", h.RemoveFriendMutual},
		{http.MethodGet, "/friends/list", h.ListFriends},

		// ---- Groups ---------------------------------------------------------
		{http.MethodPost, "/groups/create", h.CreateGroup},
		{http.MethodGet, "/groups/list", h.ListGroups},
		{http.MethodGet, "/groups/members", h.ListGroupMembers},
		{http.MethodPost, "/groups/invite/send", h.SendGroupInvitation},
		{http.MethodPost, "/groups/invite/accept", h.AcceptGroupInvitation},
		{http.MethodPost, "/groups/invite/decline", h.DeclineGroupInvitation},
		{http.MethodGet, "/groups/invite/incoming", h.ListIncomingGroupInvitations},
		{http.MethodGet, "/groups/invite/outgoing", h.ListOutgoingGroupInvitations},
		{http.MethodPost, "/groups/member/remove", h.RemoveGroupMember},
		{http.MethodPost, "/groups/leave", h.LeaveGroup},
		{http.MethodGet, "/groups/activity", h.ListGroupActivity},

		// ---- Notifications --------------------------------------------------
		{http.MethodGet, "/notifications/list", h.ListNotifications},
		{http.MethodGet, "/notifications/count", h.CountUnseenNotifications},
		{http.MethodPost, "/notifications/seen", h.MarkNotificationSeen},
		{http.MethodPost, "/notifications/clear", h.ClearAllNotifications},

		// ---- Wallet ---------------------------------------------------------
		{http.MethodGet, "/wallet/dashboard", h.GetWalletDashboard},
		{http.MethodPost, "/wallet/deposit", h.Deposit},
		{http.MethodPost, "/wallet/withdraw", h.Withdraw},
		{http.MethodGet, "/wallet/transactions", h.ListWalletTransactions},
		{http.MethodGet, "/wallet/spending/totals", h.GetSpendingTotals},
		{http.MethodGet, "/wallet/spending/monthly", h.GetMonthlySpendingSummary},
		{http.MethodGet, "/wallet/recipients/top", h.GetTopRecipients},
		{http.MethodGet, "/wallet/bnpl/utilization", h.GetBNPLUtilization},
		{http.MethodGet, "/wallet/credit/history", h.GetCreditScoreHistory},

		// ---- Users & Profiles (protected routes only) ----------------------
		{http.MethodGet, "/users/get", h.GetUser},
		{http.MethodGet, "/users/list", h.ListUsers},
		{http.MethodPost, "/users/update/password", h.UpdatePassword},
		{http.MethodPost, "/users/update/email", h.UpdateEmail},
		{http.MethodPost, "/users/update/phone", h.UpdatePhoneNumber},
		{http.MethodPost, "/users/update/name", h.UpdateDisplayName},
		{http.MethodPost, "/users/update/color", h.UpdateProfileColor},
		{http.MethodPost, "/users/deactivate", h.DeactivateAccount},
		{http.MethodGet, "/profiles/list", h.ListProfiles},
		{http.MethodGet, "/profiles/get", h.GetProfile},
		{http.MethodGet, "/profiles/search", h.SearchProfiles},

		// ---- Settings -------------------------------------------------------
		{http.MethodGet, "/settings/get", h.GetUserSettings},
		{http.MethodPost, "/settings/update", h.UpdateUserSettings},
	}

	for _, ep := range endpoints {
		ep := ep
		for _, sc := range authRejectionScenarios() {
			sc := sc
			t.Run(ep.path+"/"+sc.name, func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(ep.method, ep.path, strings.NewReader("{}"))
				req.Header.Set("Content-Type", "application/json")
				if sc.cookie != nil {
					req.AddCookie(sc.cookie)
				}
				rr := httptest.NewRecorder()
				ep.handler(rr, req)
				assert401(t, rr, sc.name, ep.path)
			})
		}
	}
}
