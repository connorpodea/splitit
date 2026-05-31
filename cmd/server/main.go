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

	// 5. HOMEPAGE ROUTER (UPDATED FOR HTMX CANVAS LAZY-LOADING)
	// What: Watches for someone opening your main web address (http://localhost:8080/).
	// Why: Streams an empty HTML framework containing Tailwind CSS and HTMX. The container instantly
	//      calls our UI router fork endpoint to fetch either the Login Form or the main Dashboard view.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Strict Check: Go's standard "/" path is a wildcard matching any unregistered URL.
		// If a user types a broken URL, this returns a clean 404 Error instead of breaking your homepage.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Inform the network layer that the upcoming data transmission is standard web markup text
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Create the global layout outer shell.
		// It establishes our dependencies and utilizes HTMX triggers to load views seamlessly.
		master_canvas := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SplitIt - Ledger Terminal</title>
    <script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/json-enc.js"></script>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen flex items-center justify-center p-4 antialiased">

    <div id="main-application-viewport" 
         hx-get="/ui/initial-view" 
         hx-trigger="load" 
         class="w-full max-w-6xl">
         <div class="text-center font-mono text-xs text-gray-500 animate-pulse">
             Initializing secure ledger session connection...
         </div>
    </div>

</body>
</html>`

		w.Write([]byte(master_canvas))
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
