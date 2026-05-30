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

	// 1. DATABASE SETUP
	// What: Starts the database engine and builds your SQL tables in 'app.db'.
	// Why: The app cannot store users or process money if the database isn't running first.
	s, err := store.New()
	if err != nil {
		log.Fatalf("CRITICAL: Database engine failed to start: %v", err)
	}
	fmt.Println("[-] SQLite Core Connected Engine Status: READY")

	// 2. CONTROLLER / HANDLER LAYER
	// What: Creates your application controller and passes it the database access pointer.
	// Why: This layer acts as the bridge that takes incoming web data and sends it to the database.
	h := handlers.New(s)

	// 3. REGISTER API ENDPOINTS
	// What: Triggers the big routing function we just created inside your handlers package.
	// Why: It cleanly hooks up all 20+ of your backend features (like /users/create) to the network in one line.
	h.RegisterRoutes()

	// 4. FRONTEND ASSET HOSTING
	// What: Sets up a folder reader targeting your "./ui" directory.
	// Why: If you load images or custom style scripts later via "/static/filename.css", Go will find them here.
	fs := http.FileServer(http.Dir("./ui"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 5. HOMEPAGE ROUTER
	// What: Watches for someone opening your main web address (http://localhost:8080/).
	// Why: It reads your "index.html" canvas file from disk and streams it straight to the user's browser.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Strict Check: Go's standard "/" path is a wildcard matching any unregistered URL.
		// If a user types a broken URL, this returns a clean 404 Error instead of breaking your homepage.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "./ui/index.html")
	})

	// 6. START NETWORK LISTEN LOOP
	// What: Binds your application software to network port 8080.
	// Why: Puts the program into a perpetual listening state so it can catch incoming website visitors.
	port := ":8080"
	fmt.Printf("[-] API Server listening on http://localhost%s\n", port)
	fmt.Println("=====================================================================")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("CRITICAL: HTTP server crashed: %v", err)
	}
}
