package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/connorpodea/splitit/internal/handlers"
	"github.com/connorpodea/splitit/internal/store"
)

func main() {
	fmt.Println("\n=====================================================================")
	fmt.Println("                        SPLITIT API SERVER                           ")
	fmt.Println("=====================================================================")

	// Initialize the database engine and run table migrations
	s, err := store.New()
	if err != nil {
		log.Fatalf("CRITICAL: Database engine failed to start: %v", err)
	}
	fmt.Println("[-] SQLite Core Connected Engine Status: READY")

	// Wire the store into the HTTP handler layer
	h := handlers.New(s)

	// Register all API routes
	http.HandleFunc("/users/create", h.CreateUser)
	http.HandleFunc("/users/get", h.GetUser)

	// Start the HTTP server
	port := ":8080"
	fmt.Printf("[-] API Server listening on http://localhost%s\n", port)
	fmt.Println("=====================================================================")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("CRITICAL: HTTP server crashed: %v", err)
	}
}
