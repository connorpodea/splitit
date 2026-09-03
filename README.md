# SplitIt

A peer-to-peer payment platform with a Buy Now Pay Later (BNPL) system, built in Go.

Users can send money, request payments, split purchases into installments, manage a friend network, and track spending — all through a single-page web interface.

---

## Features

- **P2P Payments** — send money directly to another user with an optional note
- **Payment Requests** — request money from a user; they can accept or decline
- **BNPL Loans** — split a payment into N installments with automatic due dates; missed installments trigger credit score penalties
- **Credit Score** — each user has a score (0–100) that changes based on payment behavior; full audit history is logged
- **Wallet** — deposit and withdraw funds; view monthly spending summaries, top recipients, and BNPL utilization
- **Friends** — send/accept/decline friend requests; remove connections
- **Groups** — create named groups, invite members, split bills within a group
- **Notifications** — in-app alerts for payments, requests, and friend activity
- **Activity Feed** — unified timeline of payments sent/received, requests, and upcoming installments

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go (standard library only — no framework) |
| Database | SQLite via `modernc.org/sqlite` |
| Frontend | HTMX + Tailwind CSS (CDN, no build step) |
| Auth | Session tokens (stored in DB) + CSRF (Double Submit Cookie) |
| Rate limiting | Custom token-bucket limiter (IP-based) |

---

## Project Structure

```
p2p-bnpl/
├── cmd/
│   └── server/
│       └── main.go          # Entry point — DB init, routing, HTTP server
├── internal/
│   ├── models/
│   │   └── models.go        # All data types (User, Payment, Installment, etc.)
│   ├── store/
│   │   ├── store.go         # DB connection, schema creation, indexes
│   │   ├── users.go         # User and session queries
│   │   ├── payments.go      # Payment and payment request queries
│   │   ├── bnpl.go          # Installment and loan queries
│   │   ├── friends.go       # Friend and friend request queries
│   │   ├── groups.go        # Group and invitation queries
│   │   ├── notifications.go # Notification queries
│   │   └── wallet.go        # Wallet and credit score queries
│   └── handlers/
│       ├── handler.go       # Handler struct, session auth, shared helpers
│       ├── middleware.go    # CSRF middleware, token-bucket rate limiter
│       ├── routes.go        # All route registrations
│       └── *.go             # One file per feature domain
└── app.db                   # SQLite database file (auto-created on first run)
```

---

## Getting Started

**Requirements:** Go 1.21+

```bash
# Clone the repo
git clone https://github.com/connorpodea/splitit
cd splitit

# Run the server
go run ./cmd/server

# The app is now running at http://localhost:8080
```

The database file (`app.db`) is created automatically on first startup. No migrations to run manually.

---

## Database Schema

13 tables covering the full feature set:

| Table | Purpose |
|---|---|
| `users` | Accounts — balance, credit score, profile color |
| `sessions` | Active login tokens |
| `payments` | P2P transfers and BNPL master loan records |
| `payment_requests` | Pending money requests between users |
| `installments` | Individual repayment slices within a BNPL loan |
| `friends` | Bidirectional friend connections |
| `friend_requests` | Pending friend requests |
| `groups` | Named groups for shared billing |
| `group_members` | Group membership records |
| `group_invitations` | Pending group invites |
| `notifications` | In-app alerts |
| `wallet_transactions` | Deposit and withdrawal events |
| `credit_score_log` | Full audit trail of credit score changes |

All foreign keys are enforced at the DB level. Every foreign key column has a B-tree index to prevent full table scans.

---

## Security

- **Session auth** — login issues a random token stored in the DB; every protected route looks it up to identify the user
- **CSRF protection** — Double Submit Cookie pattern; the frontend reads `csrf_token` from a cookie and echoes it as `X-CSRF-Token` on every mutating request; cross-site requests cannot do this
- **Rate limiting** — login and registration are limited to 5 requests/minute per IP using a custom token-bucket implementation
- **Password storage** — passwords are hashed before storage; plain text is never written to the DB
- **Cryptographic IDs** — all primary keys are generated with `crypto/rand`, not sequential integers

---

## Background Jobs

A goroutine runs on a 24-hour ticker and calls `ApplyMonthlyOverduePenalties()`, which scans for unpaid installments past their due date and applies credit score deductions. It runs concurrently so it never blocks the HTTP server.

---

## Author

Connor Podea — CS student project, built to learn Go backend development, relational database design, and web security fundamentals.
