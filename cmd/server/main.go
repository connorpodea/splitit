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


// split features

// GetSpendingTotals(userID, since time.Time) — total sent, total received, net over a time window
// ListPaymentsByUser(userID) — full payment history (both sent and received), for a feed
// ListPaymentsBetweenUsers(userID, otherID) — all transactions between two specific people
// GetMonthlySpendingSummary(userID, year, month) — total out, total in, BNPL charges for a given month
// GetTopRecipients(userID, limit, since) — who you've paid the most, ranked
// GetBNPLUtilization(userID) — outstanding balance vs credit limit as a ratio
// GetCreditScoreHistory(userID) — would require a new credit_score_log table to track changes over time
// GetInstallmentSummary(userID) — total paid, total remaining, overdue amount, all in one query

// Deposit(userID, amount) — add funds directly to balance
// Withdraw(userID, amount) — cash out, with balance check
// CreateLinkedCard(card *models.LinkedCard) — store a card (tokenized, never raw numbers)
// DeleteLinkedCard(cardID, userID) — remove a card
// ListLinkedCards(userID) — get all cards on file
// GetLinkedCard(cardID, userID)
// SetDefaultCard(cardID, userID) — mark one card as the default funding source
// ListWalletTransactions(userID) — deposits and withdrawals only, separate from p2p payments
// UpdateUsername(userID, newUsername) — with uniqueness check
// UpdatePassword(userID, newHashedPassword) — caller hashes before passing
// UpdateEmail(userID, newEmail)
// UpdatePhoneNumber(userID, newPhone)
// UpdateDisplayName(userID, newName)
// DeactivateAccount(userID) — soft delete, sets an is_active flag rather than hard deleting
// GetUserSettings(userID) — pull a settings row (separate table from users)
// UpsertUserSettings(settings *models.UserSettings) — save UI/preference settings

// CountUnseenNotifications(userID) — fast integer count for badge without loading the full list