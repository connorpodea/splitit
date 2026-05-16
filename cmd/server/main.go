package main

import (
	"fmt"
	"log"

	"github.com/you/p2p-bnpl/internal/models"
	"github.com/you/p2p-bnpl/internal/store"
)

func main() {
	fmt.Println("Starting our app...")

	// Initialize our database
	s, err := store.New()
	if err != nil {
		log.Fatalf("Database failed to start: %v", err)
	}

	fmt.Println("Database is connected and ready!")

	connor := &models.User{
		ID:      "cpodea",
		Name:    "connor",
		Email:   "cpodea@gmail.com",
		Balance: 0.00,
	}

	err = s.CreateUser(connor)
	if err != nil {
		log.Fatalf("Failed to save user: %v", err)
	}

	fmt.Println("Successfully saved Connor to the database!")

	foundUser, err := s.GetUser("cpodea")
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("Successfylly retrieved the user with ID %s (Name: %s, Email: %s, Balance: %.2f)\n",
		foundUser.ID, foundUser.Name, foundUser.Email, foundUser.Balance)

}
