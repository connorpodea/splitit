package store_test

import (
	"testing"

	"github.com/connorpodea/splitit/internal/models"
)

// --- Helpers shared across payment tests ---

func seedGroup(t *testing.T, s interface {
	CreateGroup(*models.Group) error
}, creatorID, name string) string {
	t.Helper()
	g := &models.Group{Name: name, CreatorID: creatorID}

	// Use type assertion to call store-specific methods
	type groupCreator interface {
		CreateGroup(*models.Group) error
	}
	if gc, ok := s.(groupCreator); ok {
		if err := gc.CreateGroup(g); err != nil {
			t.Fatalf("seedGroup: %v", err)
		}
	}
	return g.ID
}

// --- Bug 3a: P2P Pay() must record wallet_transactions ---

func TestPay_TransfersBalancesCorrectly(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "sender", 10000)
	seedUser(t, s, "receiver", 0)

	payment := &models.Payment{
		SenderID:          "sender",
		ReceiverID:        "receiver",
		AmountCents:       3000,
		TotalAmountCents:  3000,
		Note:              "test",
		PaymentType:       "p2p",
		TotalInstallments: 1,
		Status:            "completed",
	}
	if err := s.Pay(payment); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	sender, _ := s.GetUser("sender")
	if sender.BalanceCents != 7000 {
		t.Errorf("sender balance: want 7000, got %d", sender.BalanceCents)
	}
	receiver, _ := s.GetUser("receiver")
	if receiver.BalanceCents != 3000 {
		t.Errorf("receiver balance: want 3000, got %d", receiver.BalanceCents)
	}
}

func TestPay_RejectsInsufficientFunds(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "broke", 500)
	seedUser(t, s, "receiver", 0)

	err := s.Pay(&models.Payment{
		SenderID:         "broke",
		ReceiverID:       "receiver",
		AmountCents:      1000,
		TotalAmountCents: 1000,
		PaymentType:      "p2p",
		Status:           "completed",
	})
	if err == nil {
		t.Fatal("expected error for insufficient funds, got nil")
	}

	// Balances must be unchanged after rejected payment
	user, _ := s.GetUser("broke")
	if user.BalanceCents != 500 {
		t.Errorf("broke balance should be unchanged at 500, got %d", user.BalanceCents)
	}
}

func TestPay_RecordsWalletTransactions(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 5000)
	seedUser(t, s, "bob", 0)

	if err := s.Pay(&models.Payment{
		SenderID:          "alice",
		ReceiverID:        "bob",
		AmountCents:       2000,
		TotalAmountCents:  2000,
		PaymentType:       "p2p",
		TotalInstallments: 1,
		Status:            "completed",
	}); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	aliceTxns, err := s.ListWalletTransactions("alice")
	if err != nil {
		t.Fatalf("ListWalletTransactions(alice): %v", err)
	}
	if len(aliceTxns) != 1 {
		t.Fatalf("alice: expected 1 wallet_transaction, got %d", len(aliceTxns))
	}
	if aliceTxns[0].TransactionType != "withdrawal" {
		t.Errorf("alice txn type: want 'withdrawal', got %q", aliceTxns[0].TransactionType)
	}
	if aliceTxns[0].AmountCents != 2000 {
		t.Errorf("alice withdrawal amount: want 2000, got %d", aliceTxns[0].AmountCents)
	}

	bobTxns, err := s.ListWalletTransactions("bob")
	if err != nil {
		t.Fatalf("ListWalletTransactions(bob): %v", err)
	}
	if len(bobTxns) != 1 {
		t.Fatalf("bob: expected 1 wallet_transaction, got %d", len(bobTxns))
	}
	if bobTxns[0].TransactionType != "deposit" {
		t.Errorf("bob txn type: want 'deposit', got %q", bobTxns[0].TransactionType)
	}
}

// --- Bug 3b: Deposit / Withdraw must record wallet_transactions ---

func TestDeposit_UpdatesBalanceAndRecordsTransaction(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "user1", 0)

	if err := s.Deposit("user1", 5000); err != nil {
		t.Fatalf("Deposit: %v", err)
	}

	u, _ := s.GetUser("user1")
	if u.BalanceCents != 5000 {
		t.Errorf("balance after deposit: want 5000, got %d", u.BalanceCents)
	}

	txns, err := s.ListWalletTransactions("user1")
	if err != nil {
		t.Fatalf("ListWalletTransactions: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 wallet_transaction after deposit, got %d", len(txns))
	}
	if txns[0].TransactionType != "deposit" {
		t.Errorf("txn type: want 'deposit', got %q", txns[0].TransactionType)
	}
	if txns[0].AmountCents != 5000 {
		t.Errorf("txn amount: want 5000, got %d", txns[0].AmountCents)
	}
}

func TestWithdraw_UpdatesBalanceAndRecordsTransaction(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "user2", 10000)

	if err := s.Withdraw("user2", 4000); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}

	u, _ := s.GetUser("user2")
	if u.BalanceCents != 6000 {
		t.Errorf("balance after withdrawal: want 6000, got %d", u.BalanceCents)
	}

	txns, err := s.ListWalletTransactions("user2")
	if err != nil {
		t.Fatalf("ListWalletTransactions: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 wallet_transaction after withdraw, got %d", len(txns))
	}
	if txns[0].TransactionType != "withdrawal" {
		t.Errorf("txn type: want 'withdrawal', got %q", txns[0].TransactionType)
	}
}

func TestWithdraw_RejectsInsufficientFunds(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "user3", 1000)

	err := s.Withdraw("user3", 2000)
	if err == nil {
		t.Fatal("expected error for insufficient funds, got nil")
	}

	// Balance and transaction log must be unchanged
	u, _ := s.GetUser("user3")
	if u.BalanceCents != 1000 {
		t.Errorf("balance should be unchanged at 1000, got %d", u.BalanceCents)
	}
	txns, _ := s.ListWalletTransactions("user3")
	if len(txns) != 0 {
		t.Errorf("expected 0 wallet_transactions after rejected withdrawal, got %d", len(txns))
	}
}

// --- Bug 3c: PayToGroup must split correctly and atomically ---

func TestPayToGroup_SplitsAmountEqually(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "leader", 9000)
	seedUser(t, s, "member1", 0)
	seedUser(t, s, "member2", 0)

	// Create a group; leader is auto-added as first member
	group := &models.Group{Name: "trip", CreatorID: "leader"}
	if err := s.CreateGroup(group); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	// Add the other members
	if err := s.AcceptGroupInvitation(mustSendGroupInvite(t, s, group.ID, "leader", "member1"), "member1"); err != nil {
		t.Fatalf("AcceptGroupInvitation member1: %v", err)
	}
	if err := s.AcceptGroupInvitation(mustSendGroupInvite(t, s, group.ID, "leader", "member2"), "member2"); err != nil {
		t.Fatalf("AcceptGroupInvitation member2: %v", err)
	}

	// leader pays 6000 cents to the group — should be 3000 per non-leader member
	if err := s.PayToGroup(&models.Payment{
		SenderID:    "leader",
		ReceiverID:  group.ID,
		AmountCents: 6000,
		Note:        "dinner",
	}); err != nil {
		t.Fatalf("PayToGroup: %v", err)
	}

	leader, _ := s.GetUser("leader")
	if leader.BalanceCents != 3000 {
		t.Errorf("leader balance: want 3000, got %d", leader.BalanceCents)
	}
	m1, _ := s.GetUser("member1")
	if m1.BalanceCents != 3000 {
		t.Errorf("member1 balance: want 3000, got %d", m1.BalanceCents)
	}
	m2, _ := s.GetUser("member2")
	if m2.BalanceCents != 3000 {
		t.Errorf("member2 balance: want 3000, got %d", m2.BalanceCents)
	}
}

func TestPayToGroup_RecordsWalletTransactionsForAll(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "payer", 6000)
	seedUser(t, s, "recv1", 0)
	seedUser(t, s, "recv2", 0)

	group := &models.Group{Name: "flat", CreatorID: "payer"}
	if err := s.CreateGroup(group); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.AcceptGroupInvitation(mustSendGroupInvite(t, s, group.ID, "payer", "recv1"), "recv1"); err != nil {
		t.Fatalf("AcceptGroupInvitation recv1: %v", err)
	}
	if err := s.AcceptGroupInvitation(mustSendGroupInvite(t, s, group.ID, "payer", "recv2"), "recv2"); err != nil {
		t.Fatalf("AcceptGroupInvitation recv2: %v", err)
	}

	if err := s.PayToGroup(&models.Payment{
		SenderID:    "payer",
		ReceiverID:  group.ID,
		AmountCents: 4000,
	}); err != nil {
		t.Fatalf("PayToGroup: %v", err)
	}

	payerTxns, _ := s.ListWalletTransactions("payer")
	if len(payerTxns) != 1 || payerTxns[0].TransactionType != "withdrawal" {
		t.Errorf("payer: want 1 withdrawal, got %+v", payerTxns)
	}
	recv1Txns, _ := s.ListWalletTransactions("recv1")
	if len(recv1Txns) != 1 || recv1Txns[0].TransactionType != "deposit" {
		t.Errorf("recv1: want 1 deposit, got %+v", recv1Txns)
	}
	recv2Txns, _ := s.ListWalletTransactions("recv2")
	if len(recv2Txns) != 1 || recv2Txns[0].TransactionType != "deposit" {
		t.Errorf("recv2: want 1 deposit, got %+v", recv2Txns)
	}
}

func TestPayToGroup_RejectsInsufficientFunds(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "poor", 100)
	seedUser(t, s, "rich", 0)

	group := &models.Group{Name: "g", CreatorID: "poor"}
	if err := s.CreateGroup(group); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.AcceptGroupInvitation(mustSendGroupInvite(t, s, group.ID, "poor", "rich"), "rich"); err != nil {
		t.Fatalf("AcceptGroupInvitation: %v", err)
	}

	err := s.PayToGroup(&models.Payment{
		SenderID:    "poor",
		ReceiverID:  group.ID,
		AmountCents: 5000,
	})
	if err == nil {
		t.Fatal("expected error for insufficient funds, got nil")
	}

	// All balances unchanged
	poor, _ := s.GetUser("poor")
	if poor.BalanceCents != 100 {
		t.Errorf("poor balance unchanged: want 100, got %d", poor.BalanceCents)
	}
}

// --- Bug 2: SendGroupInvitation must create a notification ---

func TestSendGroupInvitation_CreatesNotificationWithGroupsRoute(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "owner", 0)
	seedUser(t, s, "guest", 0)

	group := &models.Group{Name: "Book Club", CreatorID: "owner"}
	if err := s.CreateGroup(group); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	if err := s.SendGroupInvitation(group.ID, "owner", "guest"); err != nil {
		t.Fatalf("SendGroupInvitation: %v", err)
	}

	waitForNotification(t, s, "guest", 1)

	notifs, err := s.ListNotifications("guest")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if len(notifs) == 0 {
		t.Fatal("expected a notification for guest, got 0")
	}
	n := notifs[0]
	if n.Type != "group_invitation" {
		t.Errorf("notification type: want 'group_invitation', got %q", n.Type)
	}
	if n.LinkView != "groups" {
		t.Errorf("link_view: want 'groups', got %q", n.LinkView)
	}
}

// --- RequestFromGroup ---

func TestRequestFromGroup_CreatesOneRequestPerMember(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "requester", 0)
	seedUser(t, s, "payer1", 0)
	seedUser(t, s, "payer2", 0)

	group := &models.Group{Name: "roommates", CreatorID: "requester"}
	if err := s.CreateGroup(group); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.AcceptGroupInvitation(mustSendGroupInvite(t, s, group.ID, "requester", "payer1"), "payer1"); err != nil {
		t.Fatalf("AcceptGroupInvitation payer1: %v", err)
	}
	if err := s.AcceptGroupInvitation(mustSendGroupInvite(t, s, group.ID, "requester", "payer2"), "payer2"); err != nil {
		t.Fatalf("AcceptGroupInvitation payer2: %v", err)
	}

	req := &models.PaymentRequest{
		RequesterID: "requester",
		PayerID:     group.ID,
		AmountCents: 6000,
		Note:        "rent",
	}
	if err := s.RequestFromGroup(req); err != nil {
		t.Fatalf("RequestFromGroup: %v", err)
	}

	p1List, _ := s.ListIncomingPaymentRequests("payer1")
	if len(p1List) != 1 {
		t.Errorf("payer1: expected 1 incoming request, got %d", len(p1List))
	}
	p2List, _ := s.ListIncomingPaymentRequests("payer2")
	if len(p2List) != 1 {
		t.Errorf("payer2: expected 1 incoming request, got %d", len(p2List))
	}
	// Each payer owes 3000 (6000 / 2)
	if len(p1List) > 0 && p1List[0].AmountCents != 3000 {
		t.Errorf("payer1 split amount: want 3000, got %d", p1List[0].AmountCents)
	}
}

// mustSendGroupInvite is a test helper that sends an invitation and returns the invitation ID.
func mustSendGroupInvite(t *testing.T, s interface {
	SendGroupInvitation(groupID, senderID, receiverID string) error
	ListIncomingGroupInvitations(receiverID string) ([]models.GroupInvitation, error)
}, groupID, senderID, receiverID string) string {
	t.Helper()
	if err := s.SendGroupInvitation(groupID, senderID, receiverID); err != nil {
		t.Fatalf("SendGroupInvitation: %v", err)
	}
	invites, err := s.ListIncomingGroupInvitations(receiverID)
	if err != nil || len(invites) == 0 {
		t.Fatalf("mustSendGroupInvite: invitation not found for %s", receiverID)
	}
	return invites[0].ID
}
