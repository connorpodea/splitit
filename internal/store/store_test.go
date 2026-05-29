package store_test

import (
	"math"
	"os"
	"testing"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

// =========================================================================
// TEST HELPERS
// =========================================================================

// assertBalance fetches a user and checks their balance matches the expected value
func assertBalance(t *testing.T, s *store.Store, userID string, expected float64) {
	t.Helper()
	u, err := s.GetUser(userID)
	if err != nil {
		t.Errorf("Balance check for '%s': could not fetch user: %v", userID, err)
		return
	}
	// Round to 2 decimal places to avoid floating point drift
	actual := math.Round(u.Balance*100) / 100
	expected = math.Round(expected*100) / 100
	if actual != expected {
		t.Errorf("Balance for %-13s expected $%8.2f | got $%8.2f", userID, expected, actual)
	}
}

// assertCreditScore fetches a user and checks their credit score matches the expected value
func assertCreditScore(t *testing.T, s *store.Store, userID string, expected uint8) {
	t.Helper()
	u, err := s.GetUser(userID)
	if err != nil {
		t.Errorf("Credit score check for '%s': could not fetch user: %v", userID, err)
		return
	}
	if u.CreditScore != expected {
		t.Errorf("Credit score for %-13s expected %d | got %d", userID, expected, u.CreditScore)
	}
}

// assertCreditLimit fetches a user and checks their credit limit matches the expected value
func assertCreditLimit(t *testing.T, s *store.Store, userID string, expected float64) {
	t.Helper()
	u, err := s.GetUser(userID)
	if err != nil {
		t.Errorf("Credit limit check for '%s': could not fetch user: %v", userID, err)
		return
	}
	actual := math.Round(u.CreditLimit*100) / 100
	expected = math.Round(expected*100) / 100
	if actual != expected {
		t.Errorf("Credit limit for %-13s expected $%8.2f | got $%8.2f", userID, expected, actual)
	}
}

// newTestStore creates a fresh isolated in-memory database for each test
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize test store: %v", err)
	}
	return s
}

// seedUsers inserts the standard treasury, connor, jason, alice accounts into a store
func seedUsers(t *testing.T, s *store.Store) {
	t.Helper()
	users := []*models.User{
		{
			ID: "app_treasury", Name: "SplitIt Treasury Pool",
			Email: "treasury@splitit.internal", PhoneNumber: "000-000-0000",
			Balance: 10000.00, CreditScore: 100, CreditLimit: 0,
		},
		{
			ID: "cpodea", Name: "Connor",
			Email: "cpodea@gmail.com", PhoneNumber: "123-456-7890",
			Balance: 500.00, CreditScore: 50, CreditLimit: 1000.00,
		},
		{
			ID: "jpodea", Name: "Jason",
			Email: "jpodea@asu.edu", PhoneNumber: "987-654-3210",
			Balance: 100.00, CreditScore: 80, CreditLimit: 1000.00,
		},
		{
			ID: "alice_w", Name: "Alice",
			Email: "alice@gmail.com", PhoneNumber: "555-555-5555",
			Balance: 150.00, CreditScore: 95, CreditLimit: 1500.00,
		},
	}
	for _, u := range users {
		if err := s.CreateUser(u); err != nil {
			t.Fatalf("Failed to seed user '%s': %v", u.ID, err)
		}
	}
}

// =========================================================================
// STAGE 1 — USER CRUD
// =========================================================================

func TestCreateAndGetUser(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	u, err := s.GetUser("cpodea")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if u.Name != "Connor" {
		t.Errorf("Expected name 'Connor', got '%s'", u.Name)
	}
	if u.Balance != 500.00 {
		t.Errorf("Expected balance 500.00, got %.2f", u.Balance)
	}
	if u.CreditScore != 50 {
		t.Errorf("Expected credit score 50, got %d", u.CreditScore)
	}
}

func TestGetProfile(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	p, err := s.GetProfile("jpodea")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if p.Name != "Jason" {
		t.Errorf("Expected name 'Jason', got '%s'", p.Name)
	}
	if p.Email != "jpodea@asu.edu" {
		t.Errorf("Expected email 'jpodea@asu.edu', got '%s'", p.Email)
	}
}

func TestListUsers(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	users, err := s.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) < 4 {
		t.Errorf("Expected at least 4 users, got %d", len(users))
	}
}

func TestListProfiles(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	profiles, err := s.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) < 4 {
		t.Errorf("Expected at least 4 profiles, got %d", len(profiles))
	}
}

// =========================================================================
// STAGE 2 — CORE LEDGER PAYMENTS
// =========================================================================

func TestPay_HappyPath(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	tx := &models.Payment{
		ID: "p2p_tx_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 15.50, TotalAmount: 15.50, Note: "Dinner split",
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	if err := s.Pay(tx); err != nil {
		t.Fatalf("Pay failed: %v", err)
	}

	// Connor: 500.00 - 15.50 = 484.50
	// Jason:  100.00 + 15.50 = 115.50
	assertBalance(t, s, "cpodea", 484.50)
	assertBalance(t, s, "jpodea", 115.50)
}

func TestGetPayment(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	tx := &models.Payment{
		ID: "p2p_tx_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 15.50, TotalAmount: 15.50, Note: "Dinner split",
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	if err := s.Pay(tx); err != nil {
		t.Fatalf("Pay failed: %v", err)
	}

	fetched, err := s.GetPayment("p2p_tx_001")
	if err != nil {
		t.Fatalf("GetPayment failed: %v", err)
	}
	if fetched.SenderID != "cpodea" {
		t.Errorf("Expected sender 'cpodea', got '%s'", fetched.SenderID)
	}
	if fetched.Amount != 15.50 {
		t.Errorf("Expected amount 15.50, got %.2f", fetched.Amount)
	}
	if fetched.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", fetched.Status)
	}
}

func TestPay_InsufficientFunds(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	tx := &models.Payment{
		ID: "overdraft_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 9999.00, TotalAmount: 9999.00, Note: "Overdraft attempt",
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	err := s.Pay(tx)
	if err == nil {
		t.Fatal("Expected overdraft to be rejected, but it succeeded")
	}

	// Balances must be unchanged after rejection
	assertBalance(t, s, "cpodea", 500.00)
	assertBalance(t, s, "jpodea", 100.00)
}

// =========================================================================
// STAGE 3 — BNPL LOAN ENGINE
// =========================================================================

func TestCreateBNPLLoan_PayIn4(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	// Connor credit score = 50 → 3% fee → totalDebt = $206.00
	// baseAmount = $51.50, remainder = $0.00, down payment = $51.50
	// Credit limit: 1000.00 - 200.00 = 800.00
	loan := &models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, Note: "Developer Desk Setup",
		TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan failed: %v", err)
	}

	// Treasury: 10000.00 - 200.00 + 51.50 = 9851.50
	// Connor:   500.00 - 51.50 = 448.50
	// Jason:    100.00 + 200.00 = 300.00
	assertBalance(t, s, "app_treasury", 9851.50)
	assertBalance(t, s, "cpodea", 448.50)
	assertBalance(t, s, "jpodea", 300.00)
	assertCreditLimit(t, s, "cpodea", 800.00)

	// Verify master loan record
	record, err := s.GetPayment("bnpl_desk_001")
	if err != nil {
		t.Fatalf("GetPayment for master loan failed: %v", err)
	}
	if record.PaymentType != "bnpl_loan_master" {
		t.Errorf("Expected payment type 'bnpl_loan_master', got '%s'", record.PaymentType)
	}
	if record.TotalInstallments != 4 {
		t.Errorf("Expected 4 total installments, got %d", record.TotalInstallments)
	}
}

func TestCreateBNPLLoan_PayIn1_NoFee(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	// Pay-in-1: no fee applied, net zero on treasury
	// Alice credit limit: 1500.00 - 50.00 = 1450.00
	loan := &models.Payment{
		ID: "bnpl_book_001", SenderID: "alice_w", ReceiverID: "jpodea",
		TotalAmount: 50.00, Note: "Programming book",
		TotalInstallments: 1, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan Pay-in-1 failed: %v", err)
	}

	assertBalance(t, s, "app_treasury", 10000.00)
	assertBalance(t, s, "alice_w", 100.00)
	assertBalance(t, s, "jpodea", 150.00)
	assertCreditLimit(t, s, "alice_w", 1450.00)
}

func TestCreateBNPLLoan_ZeroInstallmentsRejected(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	loan := &models.Payment{
		ID: "bnpl_bad_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 100.00, TotalInstallments: 0, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err == nil {
		t.Fatal("Expected zero installments to be rejected, but it succeeded")
	}
}

func TestCreateBNPLLoan_OverCreditLimitRejected(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	// Connor's limit is $1000.00 — attempt $1100.00
	loan := &models.Payment{
		ID: "bnpl_over_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 1100.00, TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err == nil {
		t.Fatal("Expected over-limit loan to be rejected, but it succeeded")
	}

	// Credit limit must be unchanged
	assertCreditLimit(t, s, "cpodea", 1000.00)
}

// =========================================================================
// STAGE 4 — INSTALLMENT SCHEDULE & PAYMENT
// =========================================================================

func TestListInstallments(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	loan := &models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan failed: %v", err)
	}

	installments, err := s.ListInstallments("cpodea")
	if err != nil {
		t.Fatalf("ListInstallments failed: %v", err)
	}
	if len(installments) != 4 {
		t.Errorf("Expected 4 installments, got %d", len(installments))
	}
	// First installment is the down payment — should be paid
	if !installments[0].IsPaid {
		t.Error("Expected first installment (down payment) to be marked as paid")
	}
	// Remaining installments should be unpaid
	for i := 1; i < 4; i++ {
		if installments[i].IsPaid {
			t.Errorf("Expected installment %d to be unpaid", i+1)
		}
	}
}

func TestPayInstallment_OnTime(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	loan := &models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan failed: %v", err)
	}

	// Connor after down payment: 500.00 - 51.50 = 448.50
	// Pay installment 2 on time: 448.50 - 51.50 = 397.00
	if err := s.PayInstallment("inst_bnpl_desk_001_2", "bnpl_desk_001", "cpodea", 51.50); err != nil {
		t.Fatalf("PayInstallment 2 failed: %v", err)
	}

	assertBalance(t, s, "cpodea", 397.00)
	// On-time payment: credit score 50 + 1 = 51
	assertCreditScore(t, s, "cpodea", 51)
}

func TestPayInstallment_FullLoanPayoff(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	loan := &models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan failed: %v", err)
	}

	// Pay installments 2, 3, 4 to complete the loan
	for i, id := range []string{"inst_bnpl_desk_001_2", "inst_bnpl_desk_001_3", "inst_bnpl_desk_001_4"} {
		if err := s.PayInstallment(id, "bnpl_desk_001", "cpodea", 51.50); err != nil {
			t.Fatalf("PayInstallment %d failed: %v", i+2, err)
		}
	}

	// Credit limit should be fully restored after payoff: 800.00 + 200.00 = 1000.00
	assertCreditLimit(t, s, "cpodea", 1000.00)

	// Loan status should be updated to completed
	record, err := s.GetPayment("bnpl_desk_001")
	if err != nil {
		t.Fatalf("GetPayment after payoff failed: %v", err)
	}
	if record.Status != "completed" {
		t.Errorf("Expected loan status 'completed', got '%s'", record.Status)
	}
}

func TestPayInstallment_InsufficientFunds(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	loan := &models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan failed: %v", err)
	}

	// Drain Connor's balance so he can't afford the next installment
	drain := &models.Payment{
		ID: "drain_001", SenderID: "cpodea", ReceiverID: "jpodea",
		Amount: 448.00, TotalAmount: 448.00,
		PaymentType: "peer_to_peer", TotalInstallments: 1, Status: "completed",
	}
	if err := s.Pay(drain); err != nil {
		t.Fatalf("Drain payment failed: %v", err)
	}

	err := s.PayInstallment("inst_bnpl_desk_001_2", "bnpl_desk_001", "cpodea", 51.50)
	if err == nil {
		t.Fatal("Expected installment payment to be rejected due to insufficient funds")
	}
}

// =========================================================================
// STAGE 5 — CREDIT SCORE ENGINE
// =========================================================================

func TestUpdateCreditScore_PositiveDelta(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	if err := s.UpdateCreditScore("jpodea", 10); err != nil {
		t.Fatalf("UpdateCreditScore failed: %v", err)
	}
	assertCreditScore(t, s, "jpodea", 90) // 80 + 10 = 90
}

func TestUpdateCreditScore_NegativeDelta(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	if err := s.UpdateCreditScore("jpodea", -5); err != nil {
		t.Fatalf("UpdateCreditScore failed: %v", err)
	}
	assertCreditScore(t, s, "jpodea", 75) // 80 - 5 = 75
}

func TestUpdateCreditScore_CeilingClamp(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	if err := s.UpdateCreditScore("jpodea", 100); err != nil {
		t.Fatalf("UpdateCreditScore failed: %v", err)
	}
	// Should not exceed 100
	assertCreditScore(t, s, "jpodea", 100)
}

func TestUpdateCreditScore_FloorClamp(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	lowUser := &models.User{
		ID: "low_score_user", Name: "LowScore", Email: "low@test.com",
		PhoneNumber: "000-111-2222", Balance: 100.00, CreditScore: 5, CreditLimit: 500.00,
	}
	if err := s.CreateUser(lowUser); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if err := s.UpdateCreditScore("low_score_user", -100); err != nil {
		t.Fatalf("UpdateCreditScore failed: %v", err)
	}
	// Should not go below 0
	assertCreditScore(t, s, "low_score_user", 0)
}

// =========================================================================
// STAGE 6 — OVERDUE INSTALLMENT DETECTION
// =========================================================================

func TestListOverdueInstallments_NoneOverdue(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	loan := &models.Payment{
		ID: "bnpl_desk_001", SenderID: "cpodea", ReceiverID: "jpodea",
		TotalAmount: 200.00, TotalInstallments: 4, Status: "pending",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan failed: %v", err)
	}

	// All installments are future-dated — none should be overdue
	overdue, err := s.ListOverdueInstallments("cpodea")
	if err != nil {
		t.Fatalf("ListOverdueInstallments failed: %v", err)
	}
	if len(overdue) != 0 {
		t.Errorf("Expected 0 overdue installments, got %d", len(overdue))
	}
}

// =========================================================================
// STAGE 7 — SOCIAL NETWORK
// =========================================================================

func TestSendAndAcceptFriendRequest(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	req := &models.FriendRequest{
		ID: "freq_001", SenderID: "cpodea", ReceiverID: "jpodea",
	}
	if err := s.SendFriendRequest(req); err != nil {
		t.Fatalf("SendFriendRequest failed: %v", err)
	}

	incoming, err := s.ListIncomingFriendRequests("jpodea")
	if err != nil {
		t.Fatalf("ListIncomingFriendRequests failed: %v", err)
	}
	if len(incoming) != 1 {
		t.Errorf("Expected 1 incoming request, got %d", len(incoming))
	}

	if err := s.AcceptFriendRequest("freq_001", "cpodea", "jpodea"); err != nil {
		t.Fatalf("AcceptFriendRequest failed: %v", err)
	}

	// Verify bidirectional friendship
	jFriends, err := s.ListFriends("jpodea")
	if err != nil {
		t.Fatalf("ListFriends for Jason failed: %v", err)
	}
	if len(jFriends) != 1 || jFriends[0].ID != "cpodea" {
		t.Errorf("Jason's friends list incorrect after accept")
	}

	cFriends, err := s.ListFriends("cpodea")
	if err != nil {
		t.Fatalf("ListFriends for Connor failed: %v", err)
	}
	if len(cFriends) != 1 || cFriends[0].ID != "jpodea" {
		t.Errorf("Connor's friends list incorrect after accept (bidirectional)")
	}
}

func TestDeclineFriendRequest(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	req := &models.FriendRequest{
		ID: "freq_002", SenderID: "alice_w", ReceiverID: "jpodea",
	}
	if err := s.SendFriendRequest(req); err != nil {
		t.Fatalf("SendFriendRequest failed: %v", err)
	}

	if err := s.DeclineFriendRequest("freq_002"); err != nil {
		t.Fatalf("DeclineFriendRequest failed: %v", err)
	}

	// Request should be gone — incoming queue for Jason should be empty
	incoming, err := s.ListIncomingFriendRequests("jpodea")
	if err != nil {
		t.Fatalf("ListIncomingFriendRequests failed: %v", err)
	}
	if len(incoming) != 0 {
		t.Errorf("Expected 0 incoming requests after decline, got %d", len(incoming))
	}
}

func TestRemoveFriendMutual(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	req := &models.FriendRequest{ID: "freq_003", SenderID: "cpodea", ReceiverID: "alice_w"}
	if err := s.SendFriendRequest(req); err != nil {
		t.Fatalf("SendFriendRequest failed: %v", err)
	}
	if err := s.AcceptFriendRequest("freq_003", "cpodea", "alice_w"); err != nil {
		t.Fatalf("AcceptFriendRequest failed: %v", err)
	}
	if err := s.RemoveFriendMutual("cpodea", "alice_w"); err != nil {
		t.Fatalf("RemoveFriendMutual failed: %v", err)
	}

	// Verify both sides no longer see each other
	cFriends, _ := s.ListFriends("cpodea")
	for _, f := range cFriends {
		if f.ID == "alice_w" {
			t.Error("Alice still appears in Connor's friends list after mutual removal")
		}
	}
	aFriends, _ := s.ListFriends("alice_w")
	for _, f := range aFriends {
		if f.ID == "cpodea" {
			t.Error("Connor still appears in Alice's friends list after mutual removal")
		}
	}
}

func TestListOutgoingFriendRequests(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	reqs := []*models.FriendRequest{
		{ID: "freq_004", SenderID: "alice_w", ReceiverID: "jpodea"},
		{ID: "freq_005", SenderID: "alice_w", ReceiverID: "cpodea"},
	}
	for _, r := range reqs {
		if err := s.SendFriendRequest(r); err != nil {
			t.Fatalf("SendFriendRequest failed: %v", err)
		}
	}

	outgoing, err := s.ListOutgoingFriendRequests("alice_w")
	if err != nil {
		t.Fatalf("ListOutgoingFriendRequests failed: %v", err)
	}
	if len(outgoing) != 2 {
		t.Errorf("Expected 2 outgoing requests, got %d", len(outgoing))
	}
}

// =========================================================================
// STAGE 8 — PAYMENT REQUESTS
// =========================================================================

func TestCreateAndListPaymentRequests(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	req := &models.PaymentRequest{
		ID: "bill_001", RequesterID: "jpodea", PayerID: "cpodea",
		Amount: 22.75, Note: "Uber split", Status: "pending",
	}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest failed: %v", err)
	}

	// Connor should see 1 incoming request
	incoming, err := s.ListIncomingPaymentRequests("cpodea")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests failed: %v", err)
	}
	if len(incoming) != 1 {
		t.Errorf("Expected 1 incoming payment request, got %d", len(incoming))
	}

	// Jason should see 1 outgoing request
	outgoing, err := s.ListOutgoingPaymentRequests("jpodea")
	if err != nil {
		t.Fatalf("ListOutgoingPaymentRequests failed: %v", err)
	}
	if len(outgoing) != 1 {
		t.Errorf("Expected 1 outgoing payment request, got %d", len(outgoing))
	}
}

func TestUpdatePaymentRequestStatus(t *testing.T) {
	s := newTestStore(t)
	seedUsers(t, s)

	req := &models.PaymentRequest{
		ID: "bill_002", RequesterID: "jpodea", PayerID: "cpodea",
		Amount: 30.00, Note: "Groceries", Status: "pending",
	}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest failed: %v", err)
	}

	if err := s.UpdatePaymentRequestStatus("bill_002", "completed"); err != nil {
		t.Fatalf("UpdatePaymentRequestStatus failed: %v", err)
	}

	// After status update to completed, it should no longer appear in pending list
	incoming, err := s.ListIncomingPaymentRequests("cpodea")
	if err != nil {
		t.Fatalf("ListIncomingPaymentRequests failed: %v", err)
	}
	if len(incoming) != 0 {
		t.Errorf("Expected 0 pending requests after status update, got %d", len(incoming))
	}
}

// =========================================================================
// CLEANUP
// =========================================================================

func TestMain(m *testing.M) {
	code := m.Run()
	// Remove any leftover test database files
	os.Remove("test_app.db")
	os.Exit(code)
}