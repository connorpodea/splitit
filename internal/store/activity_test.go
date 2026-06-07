package store_test

import (
	"testing"
	"time"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

// newTestStore returns an isolated in-memory store for each test.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewFromPath(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return s
}

// seedUser inserts a user with the given ID and starting balance.
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

// --- Bug 1: payment request status must default to "pending" ---

func TestCreatePaymentRequest_ForcesStatusPending(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	req := &models.PaymentRequest{
		RequesterID: "alice",
		PayerID:     "bob",
		AmountCents: 1000,
		Note:        "lunch",
		// Status intentionally omitted — client may send empty string
	}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	if req.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", req.Status)
	}
}

func TestListIncomingPaymentRequests_ShowsRequestAfterCreate(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	req := &models.PaymentRequest{
		RequesterID: "alice",
		PayerID:     "bob",
		AmountCents: 2500,
		Note:        "pizza",
	}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	list, err := s.ListIncomingPaymentRequests("bob")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 incoming request for bob, got %d", len(list))
	}
	if list[0].AmountCents != 2500 {
		t.Errorf("expected AmountCents 2500, got %d", list[0].AmountCents)
	}
	if list[0].Status != "pending" {
		t.Errorf("expected status 'pending', got %q", list[0].Status)
	}
}

func TestListIncomingPaymentRequests_EmptyForNewUser(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "carol", 0)

	list, err := s.ListIncomingPaymentRequests("carol")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 requests for new user, got %d", len(list))
	}
}

func TestListIncomingPaymentRequests_DoesNotReturnOutgoing(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	// alice requests FROM bob — so bob is payer, alice is requester
	if err := s.CreatePaymentRequest(&models.PaymentRequest{
		RequesterID: "alice",
		PayerID:     "bob",
		AmountCents: 500,
	}); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	// alice's OWN incoming list should be empty (she is the requester, not payer)
	list, err := s.ListIncomingPaymentRequests("alice")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("alice should have 0 incoming requests, got %d", len(list))
	}
}

func TestListIncomingPaymentRequests_ExcludesAcceptedRequests(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	req := &models.PaymentRequest{
		RequesterID: "alice",
		PayerID:     "bob",
		AmountCents: 750,
	}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}
	if err := s.UpdatePaymentRequestStatus(req.ID, "accepted"); err != nil {
		t.Fatalf("UpdatePaymentRequestStatus: %v", err)
	}

	list, err := s.ListIncomingPaymentRequests("bob")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("accepted request should not appear in incoming list, got %d", len(list))
	}
}

func TestCreatePaymentRequest_CreatesNotificationForPayer(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	if err := s.CreatePaymentRequest(&models.PaymentRequest{
		RequesterID: "alice",
		PayerID:     "bob",
		AmountCents: 1000,
	}); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	// Give the goroutine a moment to complete
	waitForNotification(t, s, "bob", 1)

	notifs, err := s.ListNotifications("bob")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if len(notifs) == 0 {
		t.Fatal("expected at least 1 notification for bob, got 0")
	}
	if notifs[0].Type != "payment_request" {
		t.Errorf("expected notification type 'payment_request', got %q", notifs[0].Type)
	}
	if notifs[0].LinkView != "activity" {
		t.Errorf("expected link_view 'activity', got %q", notifs[0].LinkView)
	}
}

// waitForNotification gives goroutine-dispatched notifications a moment to land.
// With SetMaxOpenConns(1) the goroutine queues behind the caller; 20ms is more than enough.
func waitForNotification(_ *testing.T, _ *store.Store, _ string, _ int) {
	time.Sleep(20 * time.Millisecond)
}
