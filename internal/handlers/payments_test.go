package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/connorpodea/splitit/internal/handlers"
	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

// newTestHandler returns a Handler backed by a fresh in-memory store.
func newTestHandler(t *testing.T) (*handlers.Handler, *store.Store) {
	t.Helper()
	s, err := store.NewFromPath(":memory:")
	if err != nil {
		t.Fatalf("newTestHandler: %v", err)
	}
	return handlers.New(s), s
}

// seedUser inserts a user with the given balance.
func seedUser(t *testing.T, s *store.Store, id string, balanceCents int) {
	t.Helper()
	err := s.CreateUser(&models.User{
		ID:               id,
		PasswordHash:     "hash",
		Name:             id,
		Email:            id + "@test.com",
		PhoneNumber:      "0000000000",
		BalanceCents:     balanceCents,
		CreditScore:      50,
		CreditLimitCents: 100000,
	})
	if err != nil {
		t.Fatalf("seedUser %s: %v", id, err)
	}
}

// sessionReq builds an HTTP request with a real server-side session token for userID.
func sessionReq(t *testing.T, h *handlers.Handler, method, path, body, userID string) *http.Request {
	t.Helper()
	token, err := h.CreateSession(userID)
	if err != nil {
		t.Fatalf("sessionReq: CreateSession: %v", err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	return req
}

// --- Pay handler ---

func TestPay_ToUser_TransfersBalance(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 10000)
	seedUser(t, s, "bob", 0)

	rr := httptest.NewRecorder()
	h.Pay(rr, sessionReq(t, h, http.MethodPost, "/payments/pay",
		`{"receiver_id":"bob","amount_cents":3000,"payment_type":"direct"}`, "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	alice, _ := s.GetUser("alice")
	bob, _ := s.GetUser("bob")
	if alice.BalanceCents != 7000 {
		t.Errorf("alice balance: want 7000, got %d", alice.BalanceCents)
	}
	if bob.BalanceCents != 3000 {
		t.Errorf("bob balance: want 3000, got %d", bob.BalanceCents)
	}
}

func TestPay_InsufficientBalance_ReturnsFriendlyError(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 500)
	seedUser(t, s, "bob", 0)

	rr := httptest.NewRecorder()
	h.Pay(rr, sessionReq(t, h, http.MethodPost, "/payments/pay",
		`{"receiver_id":"bob","amount_cents":1000,"payment_type":"direct"}`, "alice"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Fatal("expected an error message")
	}
	// Must not leak raw internal error text
	if strings.Contains(resp["error"], "balance_cents") || strings.Contains(resp["error"], "sql:") {
		t.Errorf("error message leaks internals: %q", resp["error"])
	}
	// Must be the friendly insufficient-funds message
	if !strings.Contains(strings.ToLower(resp["error"]), "insufficient") {
		t.Errorf("expected 'insufficient' in error, got: %q", resp["error"])
	}
}

// --- CreatePaymentRequest handler ---

func TestCreatePaymentRequest_AppearInIncomingFeed(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	rr := httptest.NewRecorder()
	h.CreatePaymentRequest(rr, sessionReq(t, h, http.MethodPost, "/payments/request/create",
		`{"payer_id":"bob","amount_cents":2000}`, "alice"))

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	incoming, err := s.ListIncomingPaymentRequests("bob")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests: %v", err)
	}
	if len(incoming) != 1 {
		t.Fatalf("expected 1 incoming request for bob, got %d", len(incoming))
	}
	if incoming[0].Status != "pending" {
		t.Errorf("expected status 'pending', got %q", incoming[0].Status)
	}
}

func TestCreatePaymentRequest_ToGroup_CreatesRequestPerMember(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)
	seedUser(t, s, "carol", 0)

	group := &models.Group{Name: "Rent Group", CreatorID: "alice"}
	if err := s.CreateGroup(group); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	for _, id := range []string{"bob", "carol"} {
		if err := s.SendGroupInvitation(group.ID, "alice", id); err != nil {
			t.Fatalf("invite %s: %v", id, err)
		}
		invites, _ := s.ListIncomingGroupInvitations(id)
		if err := s.AcceptGroupInvitation(invites[0].ID, id); err != nil {
			t.Fatalf("accept %s: %v", id, err)
		}
	}

	rr := httptest.NewRecorder()
	h.CreatePaymentRequest(rr, sessionReq(t, h, http.MethodPost, "/payments/request/create",
		`{"payer_id":"`+group.ID+`","amount_cents":9000}`, "alice"))

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// bob and carol should each have one incoming pending request
	for _, id := range []string{"bob", "carol"} {
		reqs, err := s.ListIncomingPaymentRequests(id)
		if err != nil {
			t.Fatalf("ListIncomingPaymentRequests(%s): %v", id, err)
		}
		if len(reqs) != 1 {
			t.Errorf("expected 1 request for %s, got %d", id, len(reqs))
		}
	}
}

// --- Notification routing ---

func TestPay_ToUser_CreatesNotificationWithActivityLinkView(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 5000)
	seedUser(t, s, "bob", 0)

	rr := httptest.NewRecorder()
	h.Pay(rr, sessionReq(t, h, http.MethodPost, "/payments/pay",
		`{"receiver_id":"bob","amount_cents":1000,"payment_type":"direct"}`, "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	time.Sleep(20 * time.Millisecond) // allow goroutine notification to land

	notifs, err := s.ListNotifications("bob")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if len(notifs) == 0 {
		t.Fatal("expected at least 1 notification for bob")
	}
	if notifs[0].LinkView != "activity" {
		t.Errorf("expected link_view 'activity', got %q", notifs[0].LinkView)
	}
}

// --- FulfillPaymentRequest handler ---

func TestFulfillPaymentRequest_TransfersFunds(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 5000)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 2000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.FulfillPaymentRequest(rr, sessionReq(t, h, http.MethodPost, "/payments/request/fulfill",
		`{"request_id":"`+req.ID+`"}`, "bob"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	alice, _ := s.GetUser("alice")
	bob, _ := s.GetUser("bob")
	if alice.BalanceCents != 2000 {
		t.Errorf("alice balance: want 2000, got %d", alice.BalanceCents)
	}
	if bob.BalanceCents != 3000 {
		t.Errorf("bob balance: want 3000, got %d", bob.BalanceCents)
	}
}

func TestFulfillPaymentRequest_RejectsInsufficientBalance(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 100)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 500}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.FulfillPaymentRequest(rr, sessionReq(t, h, http.MethodPost, "/payments/request/fulfill",
		`{"request_id":"`+req.ID+`"}`, "bob"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if !strings.Contains(strings.ToLower(resp["error"]), "insufficient") {
		t.Errorf("expected 'insufficient' in error, got %q", resp["error"])
	}
}

// --- GetUnifiedActivity handler ---

func TestGetUnifiedActivity_ReturnsFeedItems(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 5000)
	seedUser(t, s, "bob", 0)

	// alice pays bob — should appear as payment_sent for alice
	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 1000, TotalAmountCents: 1000,
		PaymentType: "direct", TotalInstallments: 1, Status: "completed",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	rr := httptest.NewRecorder()
	h.GetUnifiedActivity(rr, sessionReq(t, h, http.MethodGet, "/payments/activity", "", "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var items []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one activity item")
	}
	found := false
	for _, item := range items {
		if item["kind"] == "payment_sent" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'payment_sent' item in alice's activity feed")
	}
}

// --- UpdateProfileColor handler ---

func TestUpdateProfileColor_PersistsColor(t *testing.T) {
	h, s := newTestHandler(t)
	seedUser(t, s, "alice", 0)

	rr := httptest.NewRecorder()
	h.UpdateProfileColor(rr, sessionReq(t, h, http.MethodPost, "/users/update/color",
		`{"color":"#fb7185"}`, "alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	user, err := s.GetUser("alice")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.ProfileColor != "#fb7185" {
		t.Errorf("profile_color: want '#fb7185', got %q", user.ProfileColor)
	}
}
