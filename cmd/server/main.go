package main

import (
	"log"
	"net/http"

	"github.com/connorpodea/splitit/internal/handlers"
	"github.com/connorpodea/splitit/internal/store"
)

func main() {
	// 1. DATABASE SETUP
	s, err := store.New()
	if err != nil {
		log.Fatalf("[CRITICAL] Database failed to initialize: %v", err)
	}
	log.Println("[INFO] SQLite storage engine initialized successfully")

	// 2. CONTROLLER LAYER
	h := handlers.New(s)

	// 3. ROUTING REGISTRATION
	h.RegisterRoutes()
	log.Println("[INFO] Application route handlers mapped successfully")

	// 4. STATIC ASSET HOSTING
	fs := http.FileServer(http.Dir("./ui"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 5. MASTER FRAMEWORK CONTAINER ROUTER
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		masterCanvas := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SplitIt - Ledger Terminal</title>
    <script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/json-enc.js"></script>
</head>
<body class="bg-[#0b0f19] text-slate-100 min-h-screen flex items-center justify-center antialiased">

    <div id="main-application-viewport" 
         hx-get="/ui/initial-view" 
         hx-trigger="load" 
         class="w-full">
         <div class="text-center font-mono text-xs text-slate-600 animate-pulse">
             Authenticating background ledger session...
         </div>
    </div>

</body>
</html>`

		w.Write([]byte(masterCanvas))
	})

	// 6. START NETWORK SERVER
	port := ":8080"
	log.Printf("[INFO] SplitIt engine online. Streaming on http://localhost%s\n", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("[CRITICAL] HTTP network server collapsed: %v", err)
	}
}
