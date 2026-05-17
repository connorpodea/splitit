package main

import (
	"fmt"
	"log"

	"github.com/you/p2p-bnpl/internal/models"
	"github.com/you/p2p-bnpl/internal/store"
)

func main() {
	fmt.Println("Starting our app...")

	// create an instance of the database
	s, err := store.New()
	if err != nil {
		log.Fatalf("Database failed to start: %v", err)
	}
	fmt.Println("Database is connected and ready")

	// create user 1
	connor := &models.User{
		ID:      "cpodea",
		Name:    "connor",
		Email:   "cpodea@gmail.com",
		Balance: 100.00,
	}

	user1 := s.CreateUser(connor)
	if user1 != nil {
		log.Fatalf("Failed to save user: %v", user1)
	}
	fmt.Println("Successfully saved connor to the database")

	// create user 2
	jason := &models.User{
		ID:      "jpodea",
		Name:    "jason",
		Email:   "jpodea@asu.edu",
		Balance: 100.00,
	}

	user2 := s.CreateUser(jason)
	if user2 != nil {
		log.Fatalf("Failed to save user: %v", user2)
	}
	fmt.Println("Successfully saved jason to the database")

	// process a payment between user 1 and user 2
	payment := &models.Payment{
		ID:         "1",
		SenderID:   "cpodea",
		ReceiverID: "jpodea",
		Amount:     10.00,
		Note:       "Testing",
	}

	err = s.Pay(payment)
	if err != nil {
		log.Fatalf("Failed to process payment: %v", err)
	}

	fmt.Printf("Successfully processed payment with id: %s (Sender: %s, Receiver: %s, Amount: %.2f, Note: %s)",
		payment.ID, payment.SenderID, payment.ReceiverID, payment.Amount, payment.Note)

	// search for user 1
	foundUser1, err := s.GetUser("cpodea")
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("Successfully retrieved the user with ID %s (Name: %s, Email: %s, Balance: %.2f)\n",
		foundUser1.ID, foundUser1.Name, foundUser1.Email, foundUser1.Balance)

	// search for user 2
	foundUser2, err := s.GetUser("jpodea")
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("Successfully retrieved the user with ID %s (Name: %s, Email: %s, Balance: %.2f)\n",
		foundUser2.ID, foundUser2.Name, foundUser2.Email, foundUser2.Balance)

	users, err := s.ListUsers()
	if err != nil {
		log.Fatalf("Failed to retrieve all users: %v", err)
	}
	for index, user := range users {
		fmt.Printf("%d. Name: %s | Balance: %.2f\n", index + 1, user.Name, user.Balance)
	}
}
