package store_test

import (
	"testing"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

// seedUserWithColor inserts a user with the given starting balance then immediately
// sets their profile color. Separate from seedUser because CreateUser does not accept
// a profile_color argument — the column defaults to '#4ade80' at the DB level.
func seedUserWithColor(t *testing.T, s *store.Store, id, color string, balanceCents int) {
	t.Helper()
	seedUser(t, s, id, balanceCents)
	if err := s.UpdateProfileColor(id, color); err != nil {
		t.Fatalf("seedUserWithColor: UpdateProfileColor(%s, %s): %v", id, color, err)
	}
}

// --- PeerColor resolution in GetUnifiedActivity ---

// TestGetUnifiedActivity_PeerColorFromPaymentSent verifies that the peer_color field
// on a payment_sent row reflects the receiver's current profile_color in the users table.
func TestGetUnifiedActivity_PeerColorFromPaymentSent(t *testing.T) {
	s := newTestStore(t)
	seedUserWithColor(t, s, "alice", "#fb7185", 10000)
	seedUserWithColor(t, s, "bob", "#22d3ee", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 1000, TotalAmountCents: 1000, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	items, err := s.GetUnifiedActivity("alice", 10)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	var sent *models.ActivityItem
	for i := range items {
		if items[i].Kind == "payment_sent" {
			sent = &items[i]
			break
		}
	}
	if sent == nil {
		t.Fatal("no payment_sent item found in alice's feed")
	}
	if sent.PeerColor != "#22d3ee" {
		t.Errorf("peer_color: want '#22d3ee' (bob's color), got %q", sent.PeerColor)
	}
}

// TestGetUnifiedActivity_PeerColorFromPaymentReceived verifies the sender's color appears
// on payment_received rows in the receiver's feed.
func TestGetUnifiedActivity_PeerColorFromPaymentReceived(t *testing.T) {
	s := newTestStore(t)
	seedUserWithColor(t, s, "alice", "#fb7185", 10000)
	seedUserWithColor(t, s, "bob", "#22d3ee", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 1500, TotalAmountCents: 1500, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	items, err := s.GetUnifiedActivity("bob", 10)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	var received *models.ActivityItem
	for i := range items {
		if items[i].Kind == "payment_received" {
			received = &items[i]
			break
		}
	}
	if received == nil {
		t.Fatal("no payment_received item found in bob's feed")
	}
	if received.PeerColor != "#fb7185" {
		t.Errorf("peer_color: want '#fb7185' (alice's color), got %q", received.PeerColor)
	}
}

// TestGetUnifiedActivity_PeerColorUpdatesRetroactively verifies that after a user changes
// their profile color, every historical entry in the feed reflects the new color on the
// next query — because peer_color is a live JOIN, not a snapshot stored at write time.
func TestGetUnifiedActivity_PeerColorUpdatesRetroactively(t *testing.T) {
	s := newTestStore(t)
	seedUserWithColor(t, s, "alice", "#fb7185", 10000)
	seedUserWithColor(t, s, "bob", "#22d3ee", 0)

	// Record a payment while alice has color #fb7185
	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 500, TotalAmountCents: 500, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	// Alice changes her profile color
	if err := s.UpdateProfileColor("alice", "#c084fc"); err != nil {
		t.Fatalf("UpdateProfileColor: %v", err)
	}

	// Bob's feed must now show alice's NEW color on the historical payment_received row
	items, err := s.GetUnifiedActivity("bob", 10)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	var received *models.ActivityItem
	for i := range items {
		if items[i].Kind == "payment_received" {
			received = &items[i]
			break
		}
	}
	if received == nil {
		t.Fatal("no payment_received item found in bob's feed")
	}
	if received.PeerColor != "#c084fc" {
		t.Errorf("retroactive peer_color: want '#c084fc' (alice's updated color), got %q", received.PeerColor)
	}
}

// TestGetUnifiedActivity_PeerColorFromRequestReceived verifies the requester's color
// appears on request_received rows in the payer's feed.
func TestGetUnifiedActivity_PeerColorFromRequestReceived(t *testing.T) {
	s := newTestStore(t)
	seedUserWithColor(t, s, "alice", "#facc15", 0)
	seedUserWithColor(t, s, "bob", "#4ade80", 5000)

	if err := s.CreatePaymentRequest(&models.PaymentRequest{
		RequesterID: "alice", PayerID: "bob", AmountCents: 2000, Note: "dinner",
	}); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	waitForNotification(t, s, "bob", 1)

	items, err := s.GetUnifiedActivity("bob", 10)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	var row *models.ActivityItem
	for i := range items {
		if items[i].Kind == "request_received" {
			row = &items[i]
			break
		}
	}
	if row == nil {
		t.Fatal("no request_received item found in bob's feed")
	}
	if row.PeerColor != "#facc15" {
		t.Errorf("peer_color: want '#facc15' (alice's color), got %q", row.PeerColor)
	}
}

// TestGetUnifiedActivity_PeerColorFromInstallmentDue verifies that the BNPL lender's
// profile color appears on installment_due rows in the debtor's feed.
func TestGetUnifiedActivity_PeerColorFromInstallmentDue(t *testing.T) {
	s := newTestStore(t)
	// alice is the merchant/lender; bob is the buyer with enough credit
	seedUserWithColor(t, s, "alice", "#c084fc", 0)
	seedUserWithColor(t, s, "bob", "#fb7185", 0)

	// Give bob sufficient credit and balance for the down payment
	if err := s.Pay(&models.Payment{
		SenderID: "app_treasury", ReceiverID: "bob",
		AmountCents: 100000, TotalAmountCents: 100000, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("fund bob via treasury: %v", err)
	}

	loan := &models.Payment{
		SenderID: "bob", ReceiverID: "alice",
		AmountCents: 2500, TotalAmountCents: 10000,
		Note: "laptop", PaymentType: "bnpl", TotalInstallments: 4,
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan: %v", err)
	}

	items, err := s.GetUnifiedActivity("bob", 20)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	var row *models.ActivityItem
	for i := range items {
		if items[i].Kind == "installment_due" {
			row = &items[i]
			break
		}
	}
	if row == nil {
		t.Fatal("no installment_due item found in bob's feed")
	}
	if row.PeerColor != "#c084fc" {
		t.Errorf("peer_color: want '#c084fc' (alice's color), got %q", row.PeerColor)
	}
	if row.Note != "laptop" {
		t.Errorf("note: want 'laptop', got %q", row.Note)
	}
	if row.PeerName == "" || row.PeerName == "alice" {
		// PeerName is resolved from users.name; seedUser sets name = id, so "alice" is correct
		if row.PeerName != "alice" {
			t.Errorf("peer_name: want 'alice', got %q", row.PeerName)
		}
	}
}

// TestGetUnifiedActivity_PeerColorDefaultsWhenMissing verifies that the fallback color
// '#4ade80' is returned when a user has no explicit profile_color set (DB schema default).
func TestGetUnifiedActivity_PeerColorDefaultsWhenMissing(t *testing.T) {
	s := newTestStore(t)
	// seedUser does not set profile_color; it takes the schema default '#4ade80'
	seedUser(t, s, "alice", 10000)
	seedUser(t, s, "bob", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 300, TotalAmountCents: 300, PaymentType: "direct",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	items, err := s.GetUnifiedActivity("alice", 10)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least 1 activity item")
	}
	if items[0].PeerColor == "" {
		t.Error("peer_color must never be empty; expected '#4ade80' default")
	}
}
