package store_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connorpodea/splitit/internal/models"
)

// --- FulfillPaymentRequest ---

func TestFulfillPaymentRequest_TransfersFunds(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 5000)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 2000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	if err := s.FulfillPaymentRequest(req.ID, "bob"); err != nil {
		t.Fatalf("FulfillPaymentRequest: %v", err)
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

func TestFulfillPaymentRequest_MarksRequestCompleted(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 5000)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 1000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	if err := s.FulfillPaymentRequest(req.ID, "bob"); err != nil {
		t.Fatalf("FulfillPaymentRequest: %v", err)
	}

	// After fulfillment the request is no longer pending
	incoming, err := s.ListIncomingPaymentRequests("bob")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests: %v", err)
	}
	for _, r := range incoming {
		if r.ID == req.ID {
			t.Errorf("fulfilled request still appears as pending: %+v", r)
		}
	}
}

func TestFulfillPaymentRequest_RejectsInsufficientFunds(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 500)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 1000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	err := s.FulfillPaymentRequest(req.ID, "bob")
	if err == nil {
		t.Fatal("expected error for insufficient funds, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient") {
		t.Errorf("expected 'insufficient' in error, got: %v", err)
	}

	// Balances must be unchanged
	alice, _ := s.GetUser("alice")
	bob, _ := s.GetUser("bob")
	if alice.BalanceCents != 0 {
		t.Errorf("alice balance should be unchanged: got %d", alice.BalanceCents)
	}
	if bob.BalanceCents != 500 {
		t.Errorf("bob balance should be unchanged: got %d", bob.BalanceCents)
	}
}

func TestFulfillPaymentRequest_RejectsWrongPayer(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 5000)
	seedUser(t, s, "carol", 5000)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 1000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	// carol tries to fulfill bob's request
	if err := s.FulfillPaymentRequest(req.ID, "carol"); err == nil {
		t.Fatal("expected error when wrong user fulfills request, got nil")
	}
}

func TestFulfillPaymentRequest_RejectsDoubleFulfill(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 5000)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 1000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	if err := s.FulfillPaymentRequest(req.ID, "bob"); err != nil {
		t.Fatalf("first FulfillPaymentRequest: %v", err)
	}
	if err := s.FulfillPaymentRequest(req.ID, "bob"); err == nil {
		t.Fatal("expected error on second fulfillment, got nil")
	}
}

func TestFulfillPaymentRequest_RecordsWalletTransactions(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 5000)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 2000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}
	if err := s.FulfillPaymentRequest(req.ID, "bob"); err != nil {
		t.Fatalf("FulfillPaymentRequest: %v", err)
	}

	bobTxns, err := s.ListWalletTransactions("bob")
	if err != nil {
		t.Fatalf("ListWalletTransactions(bob): %v", err)
	}
	aliceTxns, err := s.ListWalletTransactions("alice")
	if err != nil {
		t.Fatalf("ListWalletTransactions(alice): %v", err)
	}

	if len(bobTxns) == 0 {
		t.Fatal("expected wallet transaction for bob (withdrawal)")
	}
	if bobTxns[0].TransactionType != "withdrawal" {
		t.Errorf("bob txn type: want 'withdrawal', got %q", bobTxns[0].TransactionType)
	}
	if len(aliceTxns) == 0 {
		t.Fatal("expected wallet transaction for alice (deposit)")
	}
	if aliceTxns[0].TransactionType != "deposit" {
		t.Errorf("alice txn type: want 'deposit', got %q", aliceTxns[0].TransactionType)
	}
}

// --- GetUnifiedActivity ---

func TestGetUnifiedActivity_ContainsAllKinds(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 10000)
	seedUser(t, s, "bob", 10000)

	// alice sends bob money → payment_sent for alice, payment_received for bob
	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 1000, TotalAmountCents: 1000,
		PaymentType: "direct", TotalInstallments: 1, Status: "completed",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	// bob requests money from alice → request_sent for bob, request_received for alice
	req := &models.PaymentRequest{RequesterID: "bob", PayerID: "alice", AmountCents: 500}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	aliceActivity, err := s.GetUnifiedActivity("alice", 50)
	if err != nil {
		t.Fatalf("GetUnifiedActivity(alice): %v", err)
	}

	kinds := map[string]bool{}
	for _, item := range aliceActivity {
		kinds[item.Kind] = true
	}

	if !kinds["payment_sent"] {
		t.Error("missing 'payment_sent' in alice's activity feed")
	}
	if !kinds["request_received"] {
		t.Error("missing 'request_received' in alice's activity feed")
	}
}

func TestGetUnifiedActivity_OrderedNewestFirst(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 10000)
	seedUser(t, s, "bob", 0)

	for i := 0; i < 3; i++ {
		if err := s.Pay(&models.Payment{
			SenderID: "alice", ReceiverID: "bob",
			AmountCents: 100, TotalAmountCents: 100,
			PaymentType: "direct", TotalInstallments: 1, Status: "completed",
		}); err != nil {
			t.Fatalf("Pay %d: %v", i, err)
		}
		// Small delay to guarantee distinct created_at timestamps
		time.Sleep(2 * time.Millisecond)
	}

	items, err := s.GetUnifiedActivity("alice", 50)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	if len(items) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(items))
	}
	for i := 1; i < len(items); i++ {
		if items[i].CreatedAt > items[i-1].CreatedAt {
			t.Errorf("activity feed not ordered newest-first at position %d: %s > %s",
				i, items[i].CreatedAt, items[i-1].CreatedAt)
		}
	}
}

func TestGetUnifiedActivity_RespectsLimit(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 100000)
	seedUser(t, s, "bob", 0)

	for i := 0; i < 10; i++ {
		s.Pay(&models.Payment{
			SenderID: "alice", ReceiverID: "bob",
			AmountCents: 100, TotalAmountCents: 100,
			PaymentType: "direct", TotalInstallments: 1, Status: "completed",
		})
	}

	items, err := s.GetUnifiedActivity("alice", 5)
	if err != nil {
		t.Fatalf("GetUnifiedActivity: %v", err)
	}
	if len(items) > 5 {
		t.Errorf("expected at most 5 items with limit=5, got %d", len(items))
	}
}

// --- BNPL loan creation ---

func TestCreateBNPLLoan_FundsReceiverAndGeneratesInstallments(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 50000) // buyer with enough balance for the down payment
	seedUser(t, s, "merchant", 0)

	loan := &models.Payment{
		SenderID:          "alice",
		ReceiverID:        "merchant",
		AmountCents:       10000, // $100
		TotalAmountCents:  10000,
		Note:              "test bnpl",
		TotalInstallments: 4,
		Status:            "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan: %v", err)
	}

	// Merchant must have received the item price immediately
	merchant, _ := s.GetUser("merchant")
	if merchant.BalanceCents != 10000 {
		t.Errorf("merchant balance: want 10000, got %d", merchant.BalanceCents)
	}

	// Installments must be generated
	installments, err := s.ListInstallments("alice")
	if err != nil {
		t.Fatalf("ListInstallments: %v", err)
	}
	if len(installments) != 4 {
		t.Errorf("expected 4 installments, got %d", len(installments))
	}

	// First installment must be pre-paid
	var paidCount int
	for _, inst := range installments {
		if inst.IsPaid {
			paidCount++
		}
	}
	if paidCount != 1 {
		t.Errorf("expected exactly 1 paid installment (down payment), got %d", paidCount)
	}
}

func TestCreateBNPLLoan_RejectsExceedingCreditLimit(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "merchant", 0)

	// alice has default CreditLimitCents=100000; request $1001 to exceed it
	loan := &models.Payment{
		SenderID: "alice", ReceiverID: "merchant",
		AmountCents: 100001, TotalAmountCents: 100001,
		TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err == nil {
		t.Fatal("expected error when loan exceeds credit limit, got nil")
	}
}

// --- UpdateProfileColor ---

func TestUpdateProfileColor_PersistsColor(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)

	if err := s.UpdateProfileColor("alice", "av-rose"); err != nil {
		t.Fatalf("UpdateProfileColor: %v", err)
	}

	user, err := s.GetUser("alice")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.ProfileColor != "av-rose" {
		t.Errorf("profile_color: want 'av-rose', got %q", user.ProfileColor)
	}
}

func TestUpdateProfileColor_RejectsUnknownUser(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpdateProfileColor("ghost", "av-rose"); err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
}

// --- Wallet transaction sign correctness ---

func TestWalletTransactionType_DepositOnReceive(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 5000)
	seedUser(t, s, "bob", 0)

	if err := s.Pay(&models.Payment{
		SenderID: "alice", ReceiverID: "bob",
		AmountCents: 1000, TotalAmountCents: 1000,
		PaymentType: "direct", TotalInstallments: 1, Status: "completed",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	bobTxns, err := s.ListWalletTransactions("bob")
	if err != nil {
		t.Fatalf("ListWalletTransactions: %v", err)
	}
	if len(bobTxns) == 0 {
		t.Fatal("expected wallet transaction for bob")
	}
	if bobTxns[0].TransactionType != "deposit" {
		t.Errorf("bob (receiver) txn type: want 'deposit', got %q", bobTxns[0].TransactionType)
	}

	aliceTxns, err := s.ListWalletTransactions("alice")
	if err != nil {
		t.Fatalf("ListWalletTransactions: %v", err)
	}
	if len(aliceTxns) == 0 {
		t.Fatal("expected wallet transaction for alice")
	}
	if aliceTxns[0].TransactionType != "withdrawal" {
		t.Errorf("alice (sender) txn type: want 'withdrawal', got %q", aliceTxns[0].TransactionType)
	}
}
