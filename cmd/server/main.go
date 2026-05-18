package main

import (
	"fmt"
	"log"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

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
	// STEP 1: INFRASTRUCTURE & ACCOUNT INITIALIZATION
	// =========================================================================
	fmt.Println("[STAGE 1] INITIALIZING ACCOUNT REGISTER STACK...")

	// Create and seed the centralized system treasury pool
	treasury := &models.User{
		ID:          "app_treasury",
		Name:        "SplitIt Treasury Pool",
		Email:       "treasury@splitit.internal",
		PhoneNumber: "000-000-0000",
		Balance:     10000.00, // Provides baseline capital to finance BNPL merchant payouts
		CreditScore: 100,
		CreditLimit: 0,
	}
	if err := s.CreateUser(treasury); err != nil {
		fmt.Printf(" -> Note: Treasury pool already initialized: %v\n", err)
	} else {
		fmt.Println(" -> Success: Initialized System Treasury Pool with $10,000.00")
	}

	// Create User 1 (Connor)
	connor := &models.User{
		ID:          "cpodea",
		Name:        "Connor",
		Email:       "cpodea@gmail.com",
		PhoneNumber: "123-456-7890",
		Balance:     200.00,
		CreditScore: 50, // Triggers a 3% risk fee if financing over time
		CreditLimit: 1000.00,
	}
	if err := s.CreateUser(connor); err != nil {
		fmt.Printf(" -> Note: Connor profile already exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Registered account profile: Connor ($200.00)")
	}

	// Create User 2 (Jason)
	jason := &models.User{
		ID:          "jpodea",
		Name:        "Jason",
		Email:       "jpodea@asu.edu",
		PhoneNumber: "987-654-3210",
		Balance:     100.00,
		CreditScore: 80, // Triggers a 2% risk fee if financing over time
		CreditLimit: 1000.00,
	}
	if err := s.CreateUser(jason); err != nil {
		fmt.Printf(" -> Note: Jason profile already exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Registered account profile: Jason ($100.00)")
	}

	// Create User 3 (Alice)
	alice := &models.User{
		ID:          "alice_w",
		Name:        "Alice",
		Email:       "alice@gmail.com",
		PhoneNumber: "555-555-5555",
		Balance:     150.00,
		CreditScore: 95, // Triggers a 1% risk fee if financing over time
		CreditLimit: 1500.00,
	}
	if err := s.CreateUser(alice); err != nil {
		fmt.Printf(" -> Note: Alice profile already exists: %v\n", err)
	} else {
		fmt.Println(" -> Success: Registered account profile: Alice ($150.00)")
	}
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STEP 2: BANKING & ACCOUNTING CORE TRANSFERS
	// =========================================================================
	fmt.Println("[STAGE 2] EXECUTING CORE LEDGER PAYMENTS...")

	// A standard, immediate peer-to-peer transaction transfer
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
	fmt.Println(" -> Processing standard Peer-to-Peer payment...")
	if err := s.Pay(p2pTx); err != nil {
		log.Fatalf("    CRITICAL: P2P payment settlement engine collapsed: %v", err)
	}
	fmt.Println(" -> Success: Balance adjustments executed and logged successfully.")
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STEP 3: BUY NOW, PAY LATER LOAN GENERATOR (BNPL)
	// =========================================================================
	fmt.Println("[STAGE 3] COMPILING CREDIT ENGINE LOAN SCHEDULER...")

	// Connor purchases a luxury $300.00 item from Jason via a split Pay-in-4 schedule
	bnplLoan := &models.Payment{
		ID:                "bnpl_couch_001",
		SenderID:          "cpodea",
		ReceiverID:        "jpodea",
		TotalAmount:       300.00,
		Note:              "Developer Desk Setup",
		TotalInstallments: 4,
		Status:            "pending",
	}
	fmt.Println(" -> Building structured amortization calendar schedule...")
	if err := s.CreateBNPLLoan(bnplLoan); err != nil {
		log.Fatalf("    CRITICAL: BNPL Compilation Error: %v", err)
	}
	fmt.Println(" -> Success: BNPL core distributions compiled. Full installment arrays generated.")
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STEP 4: SOCIAL DIRECTORY & NETWORKING MECHANICS
	// =========================================================================
	fmt.Println("[STAGE 4] EXECUTING SOCIAL NETWORK RELATIONSHIP GRAPHS...")

	// Connor dispatches a request to connect with Jason
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

	// Alice dispatches a request to connect with Jason
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

	// Read Jason's incoming requests before processing them
	fmt.Println("\n -> Checking Jason's incoming processing queue:")
	incomingRequests, err := s.ListIncomingFriendRequests("jpodea")
	if err != nil {
		log.Fatalf("    Failed to fetch friend requests: %v", err)
	}
	for i, req := range incomingRequests {
		fmt.Printf("    [%d] Request ID: %-22s | Sent By: %s\n", i+1, req.ID, req.SenderID)
	}

	// UNCOMMENT WHEN READY: Test the list outgoing method once you add it to your codebase
	/*
	fmt.Println("\n -> Checking Connor's outbound tracking directory:")
	outgoingRequests, _ := s.ListOutgoingFriendRequests("cpodea")
	for _, req := range outgoingRequests {
		fmt.Printf("    Pending Outgoing ID: %s | Sent To: %s\n", req.ID, req.ReceiverID)
	}
	*/

	// Executing actions: Jason accepts Connor, but declines Alice
	fmt.Println("\n -> Processing incoming friend request queue actions...")
	if err := s.AcceptFriendRequest("freq_connor_jason_01", "cpodea", "jpodea"); err != nil {
		fmt.Printf("    Note: Accept step skipped (likely already handled): %v\n", err)
	} else {
		fmt.Println("    -> Jason ACCEPTED Connor's friend request.")
	}

	if err := s.DeclineFriendRequest("freq_alice_jason_01"); err != nil {
		fmt.Printf("    Note: Decline step skipped: %v\n", err)
	} else {
		fmt.Println("    -> Jason DECLINED Alice's friend request.")
	}

	// Read back Jason's confirmed mutual friends group directory
	fmt.Println("\n -> Checking Jason's active friends directory:")
	jasonsFriends, err := s.ListFriends("jpodea")
	if err != nil {
		log.Fatalf("    Failed to fetch Jason's friend list: %v", err)
	}
	for i, f := range jasonsFriends {
		fmt.Printf("    [%d] User ID: %-10s | Name: %-6s | Connected: %s\n", i+1, f.ID, f.Name, f.CreatedAt)
	}
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STEP 5: BILL SPLITTING & PAYMENT REQUISITIONS
	// =========================================================================
	fmt.Println("[STAGE 5] TESTING BILL SPLITTING PAYMENT REQUISITIONS...")

	// Jason submits a bill splitting demand to Connor for a rideshare
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

	// Look up Connor's incoming dashboard notifications
	fmt.Println("\n -> Checking Connor's incoming pending collections panel:")
	connorsBills, err := s.ListIncomingPaymentRequests("cpodea")
	if err != nil {
		log.Fatalf("    Failed to read payment requests dashboard: %v", err)
	}
	for i, b := range connorsBills {
		fmt.Printf("    [%d] Request ID: %-15s | Owed To: %-8s | Amount: $%5.2f | Reason: %s\n", 
			i+1, b.ID, b.RequesterID, b.Amount, b.Note)
	}

	// UNCOMMENT WHEN READY: Test the outgoing payment tracker method once built
	/*
	fmt.Println("\n -> Checking Jason's outbound receivables collection board:")
	jasonsOutboundBills, _ := s.ListOutgoingPaymentRequests("jpodea")
	for _, b := range jasonsOutboundBills {
		fmt.Printf("    Bill ID: %s | Owed By: %s | Amount: $%5.2f | Status: %s\n", b.ID, b.PayerID, b.Amount, b.Status)
	}
	*/

	// Connor processes an instant settlement to clear the transaction bill balance entry
	fmt.Println("\n -> Settling open bill request balance pipelines...")
	if len(connorsBills) > 0 {
		activeBill := connorsBills[0]

		// Execute core ledger adjustment to clear liquid asset balances
		settlementTx := &models.Payment{
			ID:                "pay_clear_" + activeBill.ID,
			SenderID:          activeBill.PayerID,
			ReceiverID:        activeBill.RequesterID,
			Amount:            activeBill.Amount,
			TotalAmount:       activeBill.Amount,
			Note:              "Settling open invoice request: " + activeBill.Note,
			PaymentType:       "peer_to_peer",
			TotalInstallments: 1,
			Status:            "completed",
		}
		if err := s.Pay(settlementTx); err != nil {
			log.Fatalf("    CRITICAL: Asset balance reconciliation failed: %v", err)
		}
		fmt.Println("    -> Success: Liquid asset balances adjusted via the core bank ledger.")

		// Update the requisition state records from 'pending' to 'completed'
		if err := s.UpdatePaymentRequestStatus(activeBill.ID, "completed"); err != nil {
			log.Fatalf("    CRITICAL: State machine progression failure: %v", err)
		}
		fmt.Println("    -> Success: Bill Request State updated to 'completed' successfully.")
	}
	fmt.Println("---------------------------------------------------------------------")

	// =========================================================================
	// STEP 6: READ-BACK VERIFICATION SNAPSHOT
	// =========================================================================
	fmt.Println("[STAGE 6] COMPILING POST-TEST LEDGER BALANCE SHEETS...");
	allUsers, err := s.ListUsers()
	if err != nil {
		log.Fatalf("CRITICAL: Failed to poll snapshot records: %v", err)
	}
	for i, u := range allUsers {
		fmt.Printf(" [%d] User ID: %-13s | Name: %-6s | Liquid Balance: $%8.2f | Credit Score: %d\n", 
			i+1, u.ID, u.Name, u.Balance, u.CreditScore)
	}

	fmt.Println("=====================================================================")
	fmt.Println("             ALL INTERNAL MODULE EXECUTIONS SUCCESSFUL               ")
	fmt.Println("=====================================================================\n")
}