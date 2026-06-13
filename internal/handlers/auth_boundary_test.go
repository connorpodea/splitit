package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

// assertRejected verifies that a handler returned a non-2xx status code and a JSON
// body with a non-empty "error" field. We accept any 4xx rather than pinning to a
// specific code (400 vs 403 vs 404) because the store error bubbles up as 400
// in most handlers while a 404 would also be acceptable.
func assertRejected(t *testing.T, rr *httptest.ResponseRecorder, label string) {
	t.Helper()
	if rr.Code < 400 || rr.Code >= 500 {
		t.Errorf("%s: expected 4xx rejection, got %d — body: %s", label, rr.Code, rr.Body.String())
		return
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Errorf("%s: response is not valid JSON: %v", label, err)
		return
	}
	if body["error"] == "" {
		t.Errorf("%s: expected non-empty 'error' field in rejection body", label)
	}
}

// assertAccepted verifies the handler returned a 2xx and is used as a positive control
// alongside every rejection test to confirm legitimate callers are not accidentally blocked.
func assertAccepted(t *testing.T, rr *httptest.ResponseRecorder, label string) {
	t.Helper()
	if rr.Code < 200 || rr.Code >= 300 {
		t.Errorf("%s: expected 2xx success, got %d — body: %s", label, rr.Code, rr.Body.String())
	}
}

// seedScenario provisions three users that appear in almost every boundary test:
//
//	alice  — resource owner / requester (starts with $100)
//	bob    — legitimate counterparty / payer (starts with $100)
//	carol  — the attacker: authenticated but has no ownership of alice/bob's resources
func seedScenario(t *testing.T, s *store.Store) {
	t.Helper()
	for _, u := range []struct {
		id  string
		bal int
	}{
		{"alice", 10000},
		{"bob", 10000},
		{"carol", 10000},
	} {
		seedUser(t, s, u.id, u.bal)
	}
}

// ============================================================
// Section 1 — Payment Request Boundary
// ============================================================

// TestBoundary_FulfillRequest_ThirdPartyCannotFulfill verifies that Carol cannot
// fulfill a payment request where Bob is the designated payer.
// The store enforces `WHERE id = ? AND payer_id = ?` so Carol's session produces a rejection.
func TestBoundary_FulfillRequest_ThirdPartyCannotFulfill(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	// Alice requests $20 from Bob; Bob is payer, Alice is requester.
	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 2000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	// Carol tries to fulfill Bob's obligation.
	rr := httptest.NewRecorder()
	h.FulfillPaymentRequest(rr, sessionReq(t, h, http.MethodPost, "/payments/request/fulfill",
		fmt.Sprintf(`{"request_id":%q}`, req.ID), "carol"))

	assertRejected(t, rr, "carol fulfilling bob's payment request")
}

// TestBoundary_FulfillRequest_LegitimatePayerSucceeds is the positive control for the
// test above: the actual designated payer (Bob) must be able to fulfill the request.
func TestBoundary_FulfillRequest_LegitimatePayerSucceeds(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 2000}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.FulfillPaymentRequest(rr, sessionReq(t, h, http.MethodPost, "/payments/request/fulfill",
		fmt.Sprintf(`{"request_id":%q}`, req.ID), "bob"))

	assertAccepted(t, rr, "bob fulfilling his own payment request")
}

// TestBoundary_UpdateRequest_ThirdPartyCannotChange verifies that Carol cannot decline
// or cancel a request she has no party to. The store's WHERE clause
// `(payer_id = ? OR requester_id = ?)` ensures only participants can transition status.
func TestBoundary_UpdateRequest_ThirdPartyCannotChange(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 1500}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	for _, status := range []string{"declined", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.UpdatePaymentRequestStatus(rr, sessionReq(t, h, http.MethodPost, "/payments/request/update",
				fmt.Sprintf(`{"payment_id":%q,"new_status":%q}`, req.ID, status), "carol"))

			assertRejected(t, rr, fmt.Sprintf("carol attempting to set status=%s", status))
		})
	}
}

// TestBoundary_UpdateRequest_PayerCanDecline verifies the payer (Bob) can decline
// the request directed at him — positive control for the test above.
func TestBoundary_UpdateRequest_PayerCanDecline(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 1500}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.UpdatePaymentRequestStatus(rr, sessionReq(t, h, http.MethodPost, "/payments/request/update",
		fmt.Sprintf(`{"payment_id":%q,"new_status":"declined"}`, req.ID), "bob"))

	assertAccepted(t, rr, "bob declining a request aimed at him")
}

// TestBoundary_UpdateRequest_RequesterCanCancel verifies the requester (Alice) can
// cancel her own outgoing request — positive control for the test above.
func TestBoundary_UpdateRequest_RequesterCanCancel(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	req := &models.PaymentRequest{RequesterID: "alice", PayerID: "bob", AmountCents: 1500}
	if err := s.CreatePaymentRequest(req); err != nil {
		t.Fatalf("CreatePaymentRequest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.UpdatePaymentRequestStatus(rr, sessionReq(t, h, http.MethodPost, "/payments/request/update",
		fmt.Sprintf(`{"payment_id":%q,"new_status":"cancelled"}`, req.ID), "alice"))

	assertAccepted(t, rr, "alice cancelling her own request")
}

// ============================================================
// Section 2 — BNPL Installment Boundary
// ============================================================

// setupBNPLLoan creates a BNPL loan for borrower→merchant and returns the first
// unpaid installment. The treasury is auto-seeded by the in-memory store; the
// borrower balance covers the down-payment from the first installment.
func setupBNPLLoan(t *testing.T, s *store.Store, borrowerID, merchantID string) models.InstallmentDetail {
	t.Helper()
	loan := &models.Payment{
		SenderID:          borrowerID,
		ReceiverID:        merchantID,
		TotalAmountCents:  2000,
		TotalInstallments: 4,
		PaymentType:       "bnpl",
	}
	if err := s.CreateBNPLLoan(loan); err != nil {
		t.Fatalf("CreateBNPLLoan: %v", err)
	}

	installments, err := s.ListInstallmentDetailsForUser(borrowerID)
	if err != nil {
		t.Fatalf("ListInstallmentDetailsForUser: %v", err)
	}
	for _, inst := range installments {
		if !inst.IsPaid {
			return inst
		}
	}
	t.Fatal("expected at least one unpaid installment after BNPL loan creation")
	return models.InstallmentDetail{}
}

// TestBoundary_PayInstallment_ThirdPartyCannotPayBorrowersDebt verifies that Carol
// cannot settle Alice's installment debt by submitting Alice's installment ID.
// After the store fix (AND user_id = ?), the UPDATE matches 0 rows → rejection.
func TestBoundary_PayInstallment_ThirdPartyCannotPayBorrowersDebt(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	inst := setupBNPLLoan(t, s, "alice", "bob")

	// Carol (authenticated, has balance) submits Alice's installment ID.
	rr := httptest.NewRecorder()
	h.PayInstallment(rr, sessionReq(t, h, http.MethodPost, "/bnpl/installment/pay",
		fmt.Sprintf(`{"installment_id":%q,"payment_id":%q,"amount_cents":%d}`,
			inst.ID, inst.PaymentID, inst.AmountCents),
		"carol"))

	assertRejected(t, rr, "carol paying alice's installment")

	// Confirm Alice's installment is still unpaid — no state mutation occurred.
	remaining, err := s.ListInstallmentDetailsForUser("alice")
	if err != nil {
		t.Fatalf("ListInstallmentDetailsForUser post-attempt: %v", err)
	}
	for _, r := range remaining {
		if r.ID == inst.ID && r.IsPaid {
			t.Error("alice's installment was marked paid despite carol's rejected attempt — ownership check failed")
		}
	}
}

// TestBoundary_PayInstallment_BorrowerCanPayOwnDebt is the positive control: Alice
// must be able to pay her own installment.
func TestBoundary_PayInstallment_BorrowerCanPayOwnDebt(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	inst := setupBNPLLoan(t, s, "alice", "bob")

	rr := httptest.NewRecorder()
	h.PayInstallment(rr, sessionReq(t, h, http.MethodPost, "/bnpl/installment/pay",
		fmt.Sprintf(`{"installment_id":%q,"payment_id":%q,"amount_cents":%d}`,
			inst.ID, inst.PaymentID, inst.AmountCents),
		"alice"))

	assertAccepted(t, rr, "alice paying her own installment")
}

// ============================================================
// Section 3 — Group Invitation Boundary
// ============================================================

// setupGroupWithInvite creates a group owned by creator, invites target, and returns
// the invitation. Used by both accept and decline boundary tests.
func setupGroupWithInvite(t *testing.T, s *store.Store, creatorID, targetID string) models.GroupInvitation {
	t.Helper()
	g := &models.Group{Name: "Test Group", CreatorID: creatorID}
	if err := s.CreateGroup(g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.SendGroupInvitation(g.ID, creatorID, targetID); err != nil {
		t.Fatalf("SendGroupInvitation: %v", err)
	}
	invitations, err := s.ListIncomingGroupInvitations(targetID)
	if err != nil || len(invitations) == 0 {
		t.Fatalf("ListIncomingGroupInvitations: %v (count=%d)", err, len(invitations))
	}
	return invitations[0]
}

// TestBoundary_AcceptInvitation_ThirdPartyCannotAccept verifies that Carol cannot
// accept an invitation that was sent to Alice.
// The store enforces `WHERE id = ? AND receiver_id = ?`.
func TestBoundary_AcceptInvitation_ThirdPartyCannotAccept(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	inv := setupGroupWithInvite(t, s, "bob", "alice")

	rr := httptest.NewRecorder()
	h.AcceptGroupInvitation(rr, sessionReq(t, h, http.MethodPost, "/groups/invite/accept",
		fmt.Sprintf(`{"invitation_id":%q}`, inv.ID), "carol"))

	assertRejected(t, rr, "carol accepting alice's group invitation")
}

// TestBoundary_AcceptInvitation_ReceiverSucceeds is the positive control: Alice
// must be able to accept her own invitation.
func TestBoundary_AcceptInvitation_ReceiverSucceeds(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	inv := setupGroupWithInvite(t, s, "bob", "alice")

	rr := httptest.NewRecorder()
	h.AcceptGroupInvitation(rr, sessionReq(t, h, http.MethodPost, "/groups/invite/accept",
		fmt.Sprintf(`{"invitation_id":%q}`, inv.ID), "alice"))

	assertAccepted(t, rr, "alice accepting her own invitation")
}

// TestBoundary_DeclineInvitation_ThirdPartyCannotDecline verifies that Carol cannot
// decline an invitation that was sent to Alice.
// The store enforces `WHERE id = ? AND receiver_id = ?` with a rowsAffected guard.
func TestBoundary_DeclineInvitation_ThirdPartyCannotDecline(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	inv := setupGroupWithInvite(t, s, "bob", "alice")

	rr := httptest.NewRecorder()
	h.DeclineGroupInvitation(rr, sessionReq(t, h, http.MethodPost, "/groups/invite/decline",
		fmt.Sprintf(`{"invitation_id":%q}`, inv.ID), "carol"))

	assertRejected(t, rr, "carol declining alice's group invitation")

	// Confirm the invitation still exists — no state change occurred.
	remaining, err := s.ListIncomingGroupInvitations("alice")
	if err != nil {
		t.Fatalf("ListIncomingGroupInvitations post-attempt: %v", err)
	}
	found := false
	for _, i := range remaining {
		if i.ID == inv.ID {
			found = true
		}
	}
	if !found {
		t.Error("alice's invitation was deleted despite carol's rejected decline — ownership check failed")
	}
}

// ============================================================
// Section 4 — Group Member Removal Boundary
// ============================================================

// TestBoundary_RemoveGroupMember_NonCreatorRejected verifies that Carol (a non-creator
// who is authenticated) cannot remove Bob from Alice's group.
// This tests the fix to groups.go + store/groups.go where the handler previously
// discarded the callerID, allowing any authenticated user to evict any member.
func TestBoundary_RemoveGroupMember_NonCreatorRejected(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	// Alice creates the group and invites Bob as a member.
	g := &models.Group{Name: "Alice's Group", CreatorID: "alice"}
	if err := s.CreateGroup(g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.SendGroupInvitation(g.ID, "alice", "bob"); err != nil {
		t.Fatalf("SendGroupInvitation: %v", err)
	}
	invites, _ := s.ListIncomingGroupInvitations("bob")
	if err := s.AcceptGroupInvitation(invites[0].ID, "bob"); err != nil {
		t.Fatalf("AcceptGroupInvitation: %v", err)
	}

	// Carol (authenticated, not the creator) attempts to remove Bob.
	rr := httptest.NewRecorder()
	h.RemoveGroupMember(rr, sessionReq(t, h, http.MethodPost, "/groups/member/remove",
		fmt.Sprintf(`{"group_id":%q,"target_user_id":"bob"}`, g.ID), "carol"))

	assertRejected(t, rr, "carol removing bob from alice's group")

	// Confirm Bob is still a member — no state change occurred.
	members, err := s.ListGroupMembers(g.ID)
	if err != nil {
		t.Fatalf("ListGroupMembers: %v", err)
	}
	bobStillPresent := false
	for _, m := range members {
		if m.ID == "bob" {
			bobStillPresent = true
		}
	}
	if !bobStillPresent {
		t.Error("bob was removed from the group despite carol's rejected attempt — creator guard failed")
	}
}

// TestBoundary_RemoveGroupMember_CreatorSucceeds is the positive control: Alice
// (the group creator) must be able to remove Bob.
func TestBoundary_RemoveGroupMember_CreatorSucceeds(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	g := &models.Group{Name: "Alice's Group", CreatorID: "alice"}
	if err := s.CreateGroup(g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.SendGroupInvitation(g.ID, "alice", "bob"); err != nil {
		t.Fatalf("SendGroupInvitation: %v", err)
	}
	invites, _ := s.ListIncomingGroupInvitations("bob")
	if err := s.AcceptGroupInvitation(invites[0].ID, "bob"); err != nil {
		t.Fatalf("AcceptGroupInvitation: %v", err)
	}

	rr := httptest.NewRecorder()
	h.RemoveGroupMember(rr, sessionReq(t, h, http.MethodPost, "/groups/member/remove",
		fmt.Sprintf(`{"group_id":%q,"target_user_id":"bob"}`, g.ID), "alice"))

	assertAccepted(t, rr, "alice (creator) removing bob")
}

// TestBoundary_LeaveGroup_NonMemberRejected verifies that Carol cannot "leave" a group
// she never joined. The store's rowsAffected guard rejects the no-op delete.
func TestBoundary_LeaveGroup_NonMemberRejected(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	g := &models.Group{Name: "Bob's Group", CreatorID: "bob"}
	if err := s.CreateGroup(g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	// Carol was never invited or joined.
	rr := httptest.NewRecorder()
	h.LeaveGroup(rr, sessionReq(t, h, http.MethodPost, "/groups/leave",
		fmt.Sprintf(`{"group_id":%q}`, g.ID), "carol"))

	assertRejected(t, rr, "carol leaving a group she never joined")
}

// ============================================================
// Section 5 — Notification Boundary
// ============================================================

// TestBoundary_MarkNotificationSeen_ThirdPartyCannotMarkOthers verifies that Carol
// cannot mark Alice's notification as seen by submitting Alice's notification ID.
// The store enforces `WHERE id = ? AND user_id = ?` with a rowsAffected guard.
func TestBoundary_MarkNotificationSeen_ThirdPartyCannotMarkOthers(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	// Directly create a notification for Alice.
	notif := &models.Notification{
		UserID:   "alice",
		Type:     "payment_received",
		Title:    "Test notification",
		Body:     "Alice received a payment",
		LinkView: "activity",
	}
	if err := s.CreateNotification(notif); err != nil {
		t.Fatalf("CreateNotification: %v", err)
	}

	// Carol submits Alice's notification ID with Carol's session.
	rr := httptest.NewRecorder()
	h.MarkNotificationSeen(rr, sessionReq(t, h, http.MethodPost, "/notifications/seen",
		fmt.Sprintf(`{"notif_id":%q}`, notif.ID), "carol"))

	assertRejected(t, rr, "carol marking alice's notification as seen")

	// Confirm Alice's notification is still unseen.
	notifs, err := s.ListNotifications("alice")
	if err != nil {
		t.Fatalf("ListNotifications post-attempt: %v", err)
	}
	found := false
	for _, n := range notifs {
		if n.ID == notif.ID {
			found = true
		}
	}
	if !found {
		t.Error("alice's notification was marked seen despite carol's rejected attempt — user_id guard failed")
	}
}

// TestBoundary_MarkNotificationSeen_OwnerSucceeds is the positive control: Alice
// must be able to mark her own notification as seen.
func TestBoundary_MarkNotificationSeen_OwnerSucceeds(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	notif := &models.Notification{
		UserID:   "alice",
		Type:     "payment_received",
		Title:    "Test",
		Body:     "body",
		LinkView: "activity",
	}
	if err := s.CreateNotification(notif); err != nil {
		t.Fatalf("CreateNotification: %v", err)
	}

	rr := httptest.NewRecorder()
	h.MarkNotificationSeen(rr, sessionReq(t, h, http.MethodPost, "/notifications/seen",
		fmt.Sprintf(`{"notif_id":%q}`, notif.ID), "alice"))

	assertAccepted(t, rr, "alice marking her own notification as seen")

	// Confirm it is now gone from the unseen list.
	notifs, _ := s.ListNotifications("alice")
	for _, n := range notifs {
		if n.ID == notif.ID {
			t.Error("alice's notification should no longer appear in the unseen list")
		}
	}
}

// ============================================================
// Section 6 — Pay: Sender Identity Cannot Be Spoofed
// ============================================================

// TestBoundary_Pay_SenderIDAlwaysFromSession verifies that even if the request body
// contains a spoofed sender_id, the handler always overwrites it with the session identity.
// This means Carol cannot send a payment "from Alice" by supplying alice's ID in the body.
func TestBoundary_Pay_SenderIDAlwaysFromSession(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	// Carol tries to pay Bob from Alice's wallet by spoofing sender_id in the JSON body.
	rr := httptest.NewRecorder()
	h.Pay(rr, sessionReq(t, h, http.MethodPost, "/payments/pay",
		`{"sender_id":"alice","receiver_id":"bob","amount_cents":5000,"payment_type":"direct"}`,
		"carol"))

	// The payment is processed with Carol as sender (not Alice).
	// It should succeed if carol has balance, but Alice's balance must be unchanged.
	aliceAfter, err := s.GetUser("alice")
	if err != nil {
		t.Fatalf("GetUser alice: %v", err)
	}
	if aliceAfter.BalanceCents != 10000 {
		t.Errorf("alice's balance changed despite carol's session: got %d, want 10000 — sender spoofing succeeded", aliceAfter.BalanceCents)
	}

	// Carol's balance should have decreased (she was the actual sender).
	carolAfter, err := s.GetUser("carol")
	if err != nil {
		t.Fatalf("GetUser carol: %v", err)
	}
	if rr.Code == http.StatusOK && carolAfter.BalanceCents >= 10000 {
		t.Error("carol's balance did not decrease after a successful payment — deduction bypassed")
	}
}

// TestBoundary_CreateBNPLLoan_BorrowerIDAlwaysFromSession verifies that the BNPL
// borrower identity is always the session user, not a client-supplied sender_id.
func TestBoundary_CreateBNPLLoan_BorrowerIDAlwaysFromSession(t *testing.T) {
	h, s := newTestHandler(t)
	seedScenario(t, s)

	aliceBefore, _ := s.GetUser("alice")

	// Carol tries to open a BNPL loan "as Alice" by spoofing sender_id.
	rr := httptest.NewRecorder()
	h.CreateBNPLLoan(rr, sessionReq(t, h, http.MethodPost, "/bnpl/loan/create",
		`{"sender_id":"alice","receiver_id":"bob","total_amount_cents":1000,"total_installments":4}`,
		"carol"))

	aliceAfter, _ := s.GetUser("alice")

	if aliceAfter.CreditLimitCents != aliceBefore.CreditLimitCents {
		t.Errorf("alice's credit limit changed despite carol's session — borrower spoofing succeeded")
	}
}
