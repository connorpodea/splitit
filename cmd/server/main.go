package main

import (
	"fmt"
	"log"

	"github.com/connorpodea/splitit/internal/models"
	"github.com/connorpodea/splitit/internal/store"
)

func main() {
	fmt.Println("\nStarting the app...")
	s, err := store.New()
	if err != nil {
		log.Fatalf("Database failed to start: %v", err)
	}
	fmt.Println("Database is connected and ready")
	fmt.Println("---------------------------------------------------------------------")

	// 1. CRITICAL: Seed the App Treasury User Pool!
	// Without this, foreign key constraints will fail when routing BNPL money.
	treasury := &models.User{
		ID:          "app_treasury",
		Name:        "SplitIt Treasury Pool",
		Email:       "treasury@splitit.internal",
		PhoneNumber: "000-000-0000",
		Balance:     10000.00, // Giving your app seed money to fund loans
		CreditScore: 100,
		CreditLimit: 0,
	}
	err = s.CreateUser(treasury)
	if err != nil {
		// Using a print instead of Fatalf here so it doesn't crash if the user already exists in app.db
		fmt.Printf("Note: Treasury pool initialization skipped (likely already exists): %v\n", err)
	} else {
		fmt.Println("Successfully initialized App Treasury Pool with $10,000.00")
	}
	fmt.Println("---------------------------------------------------------------------")

	// 2. Create User 1 (Connor)
	connor := &models.User{
		ID:          "cpodea",
		Name:        "Connor",
		Email:       "cpodea@gmail.com",
		PhoneNumber: "123-456-7890",
		Balance:     100.00,
		CreditScore: 50, // This score triggers a 2% fee
		CreditLimit: 1000.00,
	}
	if err := s.CreateUser(connor); err != nil {
		fmt.Printf("Note: Connor account creation skipped: %v\n", err)
	} else {
		fmt.Println("Successfully saved Connor to the database")
	}

	// 3. Create User 2 (Jason)
	jason := &models.User{
		ID:          "jpodea",
		Name:        "Jason",
		Email:       "jpodea@asu.edu",
		PhoneNumber: "987-654-3210",
		Balance:     100.00,
		CreditScore: 80, // This score triggers a 1% fee
		CreditLimit: 1000.00,
	}
	if err := s.CreateUser(jason); err != nil {
		fmt.Printf("Note: Jason account creation skipped: %v\n", err)
	} else {
		fmt.Println("Successfully saved Jason to the database")
	}
	fmt.Println("---------------------------------------------------------------------")

	// 4. Test a standard Peer-to-Peer payment
	p2pPayment := &models.Payment{
		ID:                "p2p_001",
		SenderID:          "cpodea",
		ReceiverID:        "jpodea",
		Amount:            10.00,
		TotalAmount:       10.00, // P2P payments move identical values
		Note:              "Pizza Delivery",
		PaymentType:       "peer_to_peer",
		TotalInstallments: 1,
		Status:            "completed",
	}
	if err := s.Pay(p2pPayment); err != nil {
		log.Fatalf("Failed to process standard payment: %v", err)
	}
	fmt.Println("Successfully processed standard P2P payment!")
	fmt.Println("---------------------------------------------------------------------")

	// 5. TEST THE HOLY GRAIL: Process a Buy Now, Pay Later Loan!
	// Connor buys a $100 couch from Jason split over 4 installments
	bnplLoan := &models.Payment{
		ID:                "loan_001",
		SenderID:          "cpodea",
		ReceiverID:        "jpodea",
		TotalAmount:       100.00, // Sticker price of the item
		Note:              "Couch Purchase",
		TotalInstallments: 4,      // Pay-in-4 plan
		Status:            "pending",
	}

	fmt.Println("Executing Buy Now, Pay Later Loan Engine...")
	if err := s.CreateBNPLLoan(bnplLoan); err != nil {
		log.Fatalf("BNPL Engine compilation failed: %v", err)
	}
	fmt.Println("BNPL Loan created successfully! Schedules generated.")
	fmt.Println("---------------------------------------------------------------------")

	// 6. Print final account balances to verify the accounting entries match
	fmt.Println("Verifying post-transaction ledger state:")
	users, err := s.ListUsers()
	if err != nil {
		log.Fatalf("Failed to retrieve metrics: %v", err)
	}
	for index, u := range users {
		fmt.Printf("%d. ID: %-12s | Name: %-6s | Balance: $%7.2f | Credit Score: %d\n", 
			index+1, u.ID, u.Name, u.Balance, u.CreditScore)
	}
	fmt.Println("---------------------------------------------------------------------")
}