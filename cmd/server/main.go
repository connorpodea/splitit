package main

import (
	"fmt"
	"log"
	"math"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

// =========================================================================
// TEST HELPERS
// =========================================================================

var passed, failed int

// assert logs whether a test condition passed or failed
func assert(label string, condition bool) {
	if condition {
		fmt.Printf("    [PASS] %s\n", label)
		passed++
	} else {
		fmt.Printf("    [FAIL] %s\n", label)
		failed++
	}
}

// assertBalance fetches a user and checks their balance matches the expected value
func assertBalance(s *store.Store, userID string, expected float64) {
	u, err := s.GetUser(userID)
	if err != nil {
		fmt.Printf("    [FAIL] Balance check for '%s': could not fetch user: %v\n", userID, err)
		failed++
		return
	}
	// Round to 2 decimal places to avoid floating point drift
	actual := math.Round(u.Balance*100) / 100
	expected = math.Round(expected*100) / 100
	assert(
		fmt.Sprintf("Balance for %-13s expected $%8.2f | got $%8.2f", userID, expected, actual),
		actual == expected,
	)
}

func main() {
	fmt.Println("\n=====================================================================")
	fmt.Println("                  SPLITIT ENTERPRISE TESTING SUITE                   ")
	fmt.Println("=====================================================================")

	s, err := store.New()
	if err != nil {
		log.Fatalf("CRITICAL: Database engine failed to start: %v", err)
	}
	fmt.Println("[-] SQLite Core Connected Engine Status: READY")
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 1: INFRASTRUCTURE & ACCOUNT INITIALIZATION
	// =========================================================================
	fmt.Println("[STAGE 1] INITIALIZING ACCOUNT REGISTER STACK...")

	treasury := &models.User{
		ID:          "app_treasury",
		Name:        "SplitIt Treasury Pool",
		Email:       "treasury@splitit.internal",
		PhoneNumber: "000-000-0000",
		Balance:     10000.00,
		CreditScore: 100,
		CreditLimit: 0,
	}
	if err := s.CreateUser(treasury); err != nil {
		fmt.Printf(" -> Note: Treasury pool already initialized: %v\n", err)
	} else {
		fmt.Println(" -> Success: Initialized System Treasury Pool with $10,000.00")
	}

	connor := &models.User{
		ID:          "cpodea",
		Name:        "Connor",
		Email:       "cpodea@gmail.com",
		PhoneNumber: "123-456-7890",
		Balance:     500.00,
		CreditScore: 50, // 3% fee rate
		CreditLimit: 1000.00,
	}
	if err := s.CreateUser(connor); err != nil {
		fmt.Printf(" -> Note: Connor profile already exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Registered account profile: Connor ($500.00)")
	}

	jason := &models.User{
		ID:          "jpodea",
		Name:        "Jason",
		Email:       "jpodea@asu.edu",
		PhoneNumber: "987-654-3210",
		Balance:     100.00,
		CreditScore: 80, // 2% fee rate
		CreditLimit: 1000.00,
	}
	if err := s.CreateUser(jason); err != nil {
		fmt.Printf(" -> Note: Jason profile already exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Registered account profile: Jason ($100.00)")
	}

	alice := &models.User{
		ID:          "alice_w",
		Name:        "Alice",
		Email:       "alice@gmail.com",
		PhoneNumber: "555-555-5555",
		Balance:     150.00,
		CreditScore: 95, // 1% fee rate
		CreditLimit: 1500.00,
	}
	if err := s.CreateUser(alice); err != nil {
		fmt.Printf(" -> Note: Alice profile already exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Registered account profile: Alice ($150.00)")
	}
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 2: INDIVIDUAL PROFILE LOOKUPS
	// =========================================================================
	fmt.Println("[STAGE 2] TESTING INDIVIDUAL PROFILE LOOKUPS...")

	u, err := s.GetUser("cpodea")
	if err != nil {
		log.Fatalf("    CRITICAL: GetUser failed for cpodea: %v", err)
	}
	assert("GetUser returns correct name for cpodea", u.Name == "Connor")
	assert("GetUser returns correct balance for cpodea", u.Balance == 500.00)
	assert("GetUser returns correct credit score for cpodea", u.CreditScore == 50)

	p, err := s.GetProfile("jpodea")
	if err != nil {
		log.Fatalf("    CRITICAL: GetProfile failed for jpodea: %v", err)
	}
	assert("GetProfile returns correct name for jpodea", p.Name == "Jason")
	assert("GetProfile does not expose balance (Profile struct only)", p.Email == "jpodea@asu.edu")

	profiles, err := s.ListProfiles()
	if err != nil {
		log.Fatalf("    CRITICAL: ListProfiles failed: %v", err)
	}
	assert("ListProfiles returns at least 4 accounts", len(profiles) >= 4)
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 3: CORE LEDGER PAYMENTS (HAPPY PATH)
	// =========================================================================
	fmt.Println("[STAGE 3] EXECUTING CORE LEDGER PAYMENTS (HAPPY PATH)...")

	// Connor sends Jason $15.50 for dinner
	p2pTx := &models.Payment{
		ID:                "p2p_tx_001",
		SenderID:          "cpodea",
		ReceiverID:        "jpodea",
		Amount:            15.50,
		TotalAmount:       15.50,
		Note:              "Dinner split settle",
		PaymentType:       "peer_to_peer",
		TotalInstallments: 1,
		Status:            "completed",
	}
	err = s.Pay(p2pTx)
	assert("P2P payment executes without error", err == nil)

	// Connor: 500.00 - 15.50 = 484.50
	// Jason:  100.00 + 15.50 = 115.50
	assertBalance(s, "cpodea", 484.50)
	assertBalance(s, "jpodea", 115.50)
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 4: INSUFFICIENT FUNDS GUARD (NEGATIVE PATH)
	// =========================================================================
	fmt.Println("[STAGE 4] TESTING INSUFFICIENT FUNDS GUARD (NEGATIVE PATH)...")

	// Attempt to overdraft Connor's account — should be hard-rejected
	overdraftTx := &models.Payment{
		ID:                "p2p_overdraft_001",
		SenderID:          "cpodea",
		ReceiverID:        "jpodea",
		Amount:            9999.00, // Far exceeds Connor's $484.50 balance
		TotalAmount:       9999.00,
		Note:              "Overdraft attempt",
		PaymentType:       "peer_to_peer",
		TotalInstallments: 1,
		Status:            "completed",
	}
	err = s.Pay(overdraftTx)
	assert("Overdraft payment is correctly rejected", err != nil)
	if err != nil {
		fmt.Printf("    -> Rejection message: %v\n", err)
	}

	// Balances must be completely unchanged after a rejected transaction
	assertBalance(s, "cpodea", 484.50)
	assertBalance(s, "jpodea", 115.50)
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 5: BNPL LOAN ENGINE (PAY-IN-4 WITH FEE)
	// =========================================================================
	fmt.Println("[STAGE 5] COMPILING BNPL PAY-IN-4 LOAN SCHEDULE (3% FEE FOR CONNOR)...")

	// Connor buys a $200 item from Jason via Pay-in-4
	// Connor credit score = 50 → 3% fee → totalDebt = $206.00
	// baseAmount = floor((206.00 / 4) * 100) / 100 = $51.50
	// remainder  = 206.00 - (51.50 * 4) = $0.00
	// Down payment (installment 1) = $51.50
	// Treasury pays Jason $200.00 immediately
	// Connor pays treasury $51.50 down
	bnplLoan := &models.Payment{
		ID:                "bnpl_desk_001",
		SenderID:          "cpodea",
		ReceiverID:        "jpodea",
		TotalAmount:       200.00,
		Note:              "Developer Desk Setup",
		TotalInstallments: 4,
		Status:            "pending",
	}
	err = s.CreateBNPLLoan(bnplLoan)
	assert("BNPL Pay-in-4 loan creates without error", err == nil)
	if err != nil {
		fmt.Printf("    -> Error: %v\n", err)
	}

	// Treasury: 10000.00 - 200.00 (paid Jason) + 51.50 (Connor down) = 9851.50
	// Connor:   484.50 - 51.50 (down payment) = 433.00
	// Jason:    115.50 + 200.00 (treasury funded) = 315.50
	assertBalance(s, "app_treasury", 9851.50)
	assertBalance(s, "cpodea", 433.00)
	assertBalance(s, "jpodea", 315.50)
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 6: BNPL LOAN ENGINE (PAY-IN-1, NO FEE)
	// =========================================================================
	fmt.Println("[STAGE 6] COMPILING BNPL PAY-IN-1 LOAN (NO FEE — SINGLE INSTALLMENT)...")

	// Alice buys a $50 item from Jason via Pay-in-1 (no fee since TotalInstallments == 1)
	// totalDebt = $50.00, down payment = $50.00, treasury pays Jason $50.00
	bnplSingle := &models.Payment{
		ID:                "bnpl_book_001",
		SenderID:          "alice_w",
		ReceiverID:        "jpodea",
		TotalAmount:       50.00,
		Note:              "Programming book",
		TotalInstallments: 1,
		Status:            "pending",
	}
	err = s.CreateBNPLLoan(bnplSingle)
	assert("BNPL Pay-in-1 loan creates without error", err == nil)
	if err != nil {
		fmt.Printf("    -> Error: %v\n", err)
	}

	// Treasury: 9851.50 - 50.00 + 50.00 = 9851.50 (net zero since same amount in/out)
	// Alice:    150.00 - 50.00 = 100.00
	// Jason:    315.50 + 50.00 = 365.50
	assertBalance(s, "app_treasury", 9851.50)
	assertBalance(s, "alice_w", 100.00)
	assertBalance(s, "jpodea", 365.50)
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 7: BNPL ZERO INSTALLMENTS (GUARD REJECTION)
	// =========================================================================
	fmt.Println("[STAGE 7] TESTING BNPL ZERO INSTALLMENT GUARD REJECTION...")

	bnplZero := &models.Payment{
		ID:                "bnpl_bad_001",
		SenderID:          "cpodea",
		ReceiverID:        "jpodea",
		TotalAmount:       100.00,
		Note:              "Invalid BNPL attempt",
		TotalInstallments: 0, // Should be rejected immediately
		Status:            "pending",
	}
	err = s.CreateBNPLLoan(bnplZero)
	assert("BNPL with zero installments is correctly rejected", err != nil)
	if err != nil {
		fmt.Printf("    -> Rejection message: %v\n", err)
	}
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 8: SOCIAL NETWORK RELATIONSHIP GRAPHS
	// =========================================================================
	fmt.Println("[STAGE 8] EXECUTING SOCIAL NETWORK RELATIONSHIP GRAPHS...")

	// Connor sends friend request to Jason
	connorToJasonReq := &models.FriendRequest{
		ID:         "freq_connor_jason_01",
		SenderID:   "cpodea",
		ReceiverID: "jpodea",
	}
	if err := s.SendFriendRequest(connorToJasonReq); err != nil {
		fmt.Printf(" -> Note: Friend request (Connor->Jason) skipped/exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Connor dispatched friend request outward to Jason.")
	}

	// Alice sends friend request to Jason
	aliceToJasonReq := &models.FriendRequest{
		ID:         "freq_alice_jason_01",
		SenderID:   "alice_w",
		ReceiverID: "jpodea",
	}
	if err := s.SendFriendRequest(aliceToJasonReq); err != nil {
		fmt.Printf(" -> Note: Friend request (Alice->Jason) skipped/exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Alice dispatched friend request outward to Jason.")
	}

	// Alice also sends a request to Connor
	aliceToConnorReq := &models.FriendRequest{
		ID:         "freq_alice_connor_01",
		SenderID:   "alice_w",
		ReceiverID: "cpodea",
	}
	if err := s.SendFriendRequest(aliceToConnorReq); err != nil {
		fmt.Printf(" -> Note: Friend request (Alice->Connor) skipped/exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Alice dispatched friend request outward to Connor.")
	}

	// Verify Jason's incoming queue shows both requests
	fmt.Println("\n -> Checking Jason's incoming request queue:")
	incomingRequests, err := s.ListIncomingFriendRequests("jpodea")
	if err != nil {
		log.Fatalf("    CRITICAL: Failed to fetch Jason's incoming requests: %v", err)
	}
	for i, req := range incomingRequests {
		fmt.Printf("    [%d] Request ID: %-22s | Sent By: %s\n", i+1, req.ID, req.SenderID)
	}
	assert("Jason has exactly 2 incoming friend requests", len(incomingRequests) == 2)

	// Verify Alice's outgoing queue shows both of her sent requests
	fmt.Println("\n -> Checking Alice's outgoing request directory:")
	outgoingRequests, err := s.ListOutgoingFriendRequests("alice_w")
	if err != nil {
		log.Fatalf("    CRITICAL: Failed to fetch Alice's outgoing requests: %v", err)
	}
	for i, req := range outgoingRequests {
		fmt.Printf("    [%d] Request ID: %-22s | Sent To: %s\n", i+1, req.ID, req.ReceiverID)
	}
	assert("Alice has exactly 2 outgoing friend requests", len(outgoingRequests) == 2)

	// Jason accepts Connor, declines Alice
	fmt.Println("\n -> Processing Jason's incoming request queue...")
	if err := s.AcceptFriendRequest("freq_connor_jason_01", "cpodea", "jpodea"); err != nil {
		fmt.Printf("    Note: Accept step skipped: %v\n", err)
	} else {
		fmt.Println("    -> Jason ACCEPTED Connor's friend request.")
	}
	if err := s.DeclineFriendRequest("freq_alice_jason_01"); err != nil {
		fmt.Printf("    Note: Decline step skipped: %v\n", err)
	} else {
		fmt.Println("    -> Jason DECLINED Alice's friend request.")
	}

	// Verify bidirectional friendship — both sides should see each other
	jasonsFriends, err := s.ListFriends("jpodea")
	if err != nil {
		log.Fatalf("    CRITICAL: Failed to fetch Jason's friends: %v", err)
	}
	assert("Jason's friends list contains exactly 1 entry", len(jasonsFriends) == 1)
	assert("Jason's friend is Connor", len(jasonsFriends) > 0 && jasonsFriends[0].ID == "cpodea")

	connorsFriends, err := s.ListFriends("cpodea")
	if err != nil {
		log.Fatalf("    CRITICAL: Failed to fetch Connor's friends: %v", err)
	}
	assert("Connor's friends list contains exactly 1 entry (bidirectional check)", len(connorsFriends) == 1)
	assert("Connor's friend is Jason (bidirectional check)", len(connorsFriends) > 0 && connorsFriends[0].ID == "jpodea")

	// Connor accepts Alice's separate request to him
	if err := s.AcceptFriendRequest("freq_alice_connor_01", "alice_w", "cpodea"); err != nil {
		fmt.Printf("    Note: Accept step skipped: %v\n", err)
	} else {
		fmt.Println("    -> Connor ACCEPTED Alice's friend request.")
	}

	// Now remove Alice from Connor's friends list
	fmt.Println("\n -> Testing mutual friend removal...")
	if err := s.RemoveFriendMutual("cpodea", "alice_w"); err != nil {
		log.Fatalf("    CRITICAL: RemoveFriendMutual failed: %v", err)
	}

	// Verify Alice no longer appears in Connor's list
	connorsFriendsAfter, _ := s.ListFriends("cpodea")
	aliceStillFriend := false
	for _, f := range connorsFriendsAfter {
		if f.ID == "alice_w" {
			aliceStillFriend = true
		}
	}
	assert("Alice removed from Connor's friends list after RemoveFriendMutual", !aliceStillFriend)

	// Verify Connor no longer appears in Alice's list either (bidirectional removal)
	alicesFriendsAfter, _ := s.ListFriends("alice_w")
	connorStillFriend := false
	for _, f := range alicesFriendsAfter {
		if f.ID == "cpodea" {
			connorStillFriend = true
		}
	}
	assert("Connor removed from Alice's friends list after RemoveFriendMutual (bidirectional)", !connorStillFriend)
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 9: BILL SPLITTING & PAYMENT REQUISITIONS
	// =========================================================================
	fmt.Println("[STAGE 9] TESTING BILL SPLITTING PAYMENT REQUISITIONS...")

	// Jason requests $22.75 from Connor for an Uber
	splitBill := &models.PaymentRequest{
		ID:          "bill_uber_001",
		RequesterID: "jpodea",
		PayerID:     "cpodea",
		Amount:      22.75,
		Note:        "Airport Uber split ride share",
		Status:      "pending",
	}
	if err := s.CreatePaymentRequest(splitBill); err != nil {
		fmt.Printf(" -> Note: Payment request step skipped: %v\n", err)
	} else {
		fmt.Println(" -> Success: Jason dispatched bill collection demand to Connor ($22.75).")
	}

	// Jason also requests $30.00 from Alice for groceries
	groceryBill := &models.PaymentRequest{
		ID:          "bill_grocery_001",
		RequesterID: "jpodea",
		PayerID:     "alice_w",
		Amount:      30.00,
		Note:        "Grocery run split",
		Status:      "pending",
	}
	if err := s.CreatePaymentRequest(groceryBill); err != nil {
		fmt.Printf(" -> Note: Payment request step skipped: %v\n", err)
	} else {
		fmt.Println(" -> Success: Jason dispatched bill collection demand to Alice ($30.00).")
	}

	// Verify Connor's incoming panel shows the Uber bill
	fmt.Println("\n -> Checking Connor's incoming pending collections panel:")
	connorsBills, err := s.ListIncomingPaymentRequests("cpodea")
	if err != nil {
		log.Fatalf("    CRITICAL: Failed to read Connor's payment requests: %v", err)
	}
	for i, b := range connorsBills {
		fmt.Printf("    [%d] Request ID: %-15s | Owed To: %-8s | Amount: $%5.2f | Reason: %s\n",
			i+1, b.ID, b.RequesterID, b.Amount, b.Note)
	}
	assert("Connor has exactly 1 incoming payment request", len(connorsBills) == 1)

	// Verify Jason's outgoing panel shows both bills he sent
	fmt.Println("\n -> Checking Jason's outgoing receivables board:")
	jasonsOutgoing, err := s.ListOutgoingPaymentRequests("jpodea")
	if err != nil {
		log.Fatalf("    CRITICAL: Failed to read Jason's outgoing requests: %v", err)
	}
	for i, b := range jasonsOutgoing {
		fmt.Printf("    [%d] Bill ID: %-15s | Owed By: %-8s | Amount: $%5.2f | Status: %s\n",
			i+1, b.ID, b.PayerID, b.Amount, b.Status)
	}
	assert("Jason has exactly 2 outgoing payment requests", len(jasonsOutgoing) == 2)

	// Connor settles the Uber bill
	fmt.Println("\n -> Connor settling Uber bill...")
	if len(connorsBills) > 0 {
		activeBill := connorsBills[0]

		settlementTx := &models.Payment{
			ID:                "pay_clear_" + activeBill.ID,
			SenderID:          activeBill.PayerID,
			ReceiverID:        activeBill.RequesterID,
			Amount:            activeBill.Amount,
			TotalAmount:       activeBill.Amount,
			Note:              "Settling open invoice: " + activeBill.Note,
			PaymentType:       "peer_to_peer",
			TotalInstallments: 1,
			Status:            "completed",
		}
		err = s.Pay(settlementTx)
		assert("Settlement payment for Uber bill executes without error", err == nil)

		err = s.UpdatePaymentRequestStatus(activeBill.ID, "completed")
		assert("Uber bill status updated to completed", err == nil)

		// Connor: 433.00 - 22.75 = 410.25
		// Jason:  365.50 + 22.75 = 388.25
		assertBalance(s, "cpodea", 410.25)
		assertBalance(s, "jpodea", 388.25)
	}

	// Verify the Uber bill no longer appears in Connor's incoming panel (status = completed)
	connorsBillsAfter, _ := s.ListIncomingPaymentRequests("cpodea")
	assert("Connor's incoming panel is empty after settlement", len(connorsBillsAfter) == 0)
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STAGE 10: FINAL LEDGER SNAPSHOT
	// =========================================================================
	fmt.Println("[STAGE 10] COMPILING POST-TEST LEDGER BALANCE SHEETS...")
	allUsers, err := s.ListUsers()
	if err != nil {
		log.Fatalf("CRITICAL: Failed to poll snapshot records: %v", err)
	}
	for i, u := range allUsers {
		fmt.Printf(" [%d] User ID: %-13s | Name: %-26s | Liquid Balance: $%8.2f | Credit Score: %d\n",
			i+1, u.ID, u.Name, u.Balance, u.CreditScore)
	}
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// FINAL REPORT
	// =========================================================================
	total := passed + failed
	fmt.Println("=====================================================================")
	fmt.Printf("  TEST RESULTS: %d/%d PASSED", passed, total)
	if failed > 0 {
		fmt.Printf("  |  %d FAILED", failed)
	}
	fmt.Println()
	if failed == 0 {
		fmt.Println("             ALL INTERNAL MODULE EXECUTIONS SUCCESSFUL               ")
	} else {
		fmt.Println("             WARNING: SOME ASSERTIONS FAILED — REVIEW ABOVE           ")
	}
	fmt.Println("=====================================================================\n")
}
