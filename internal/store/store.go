package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	// This driver connects the Go code to the SQLite database file system

	_ "modernc.org/sqlite"
)

const treasuryID = "app_treasury"

// newID generates a cryptographically random 24-character hex string for use as a primary key.
func newID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type Store struct {
	db *sql.DB
}

// New initializes our database file using the standard default local file.
func New() (*Store, error) {
	return NewFromPath("app.db")
}

// NewFromPath initializes the database engine from a custom path string.
// This allows testing suites to pass ":memory:" for isolated, in-memory testing.
func NewFromPath(path string) (*Store, error) {
	// Dynamically append the foreign key pragma check configuration onto the incoming path
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open database path '%s': %w", path, err)
	}

	// SQLite is single-writer; one connection prevents "database is locked" errors
	// and ensures in-memory databases are shared across all callers.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}

	// Automate table generation schema rules execution
	err = s.createTables()
	if err != nil {
		db.Close() // Clean up the connection if table creation fails
		return nil, fmt.Errorf("database initialization failed during custom path schema setup: %w", err)
	}

	// Idempotent migration: adds profile_color to databases created before the column existed.
	// SQLite returns "duplicate column name" on re-run; we intentionally ignore that error.
	s.db.Exec(`ALTER TABLE users ADD COLUMN profile_color TEXT NOT NULL DEFAULT 'av-indigo';`)

	// Idempotent migration: convert legacy CSS class names to the new hex palette.
	// Each UPDATE is a no-op on any row that was already migrated or defaulted to a hex value.
	for _, pair := range [][2]string{
		{"av-indigo", "#c084fc"},
		{"av-violet", "#c084fc"},
		{"av-amber", "#facc15"},
		{"av-emerald", "#4ade80"},
		{"av-cyan", "#22d3ee"},
		{"av-rose", "#fb7185"},
		{"av-pink", "#db2777"},
	} {
		// pair[0] = old CSS class name, pair[1] = replacement hex value
		s.db.Exec(`UPDATE users SET profile_color = ? WHERE profile_color = ?;`, pair[1], pair[0])
	}
	// Any remaining non-hex value (e.g. 'av-indigo' default on the column) becomes Mint Green.
	s.db.Exec(`UPDATE users SET profile_color = '#4ade80' WHERE profile_color NOT LIKE '#%';`)

	// Seed the virtual treasury account used as counterparty in BNPL fund flows.
	// INSERT OR IGNORE is idempotent — safe on every restart and on existing databases.
	// is_active=0 keeps it invisible in all user-facing list queries.
	s.db.Exec(`
INSERT OR IGNORE INTO users
  (id, password_hash, name, email, phone_number,
   balance_cents, credit_score, credit_limit_cents,
   is_active, profile_color)
VALUES
  ('app_treasury','NO_LOGIN','App Treasury','','',
   9999999999, 0, 0, 0, '#4ade80');
`)

	return s, nil
}

// createTables runs all CREATE TABLE IF NOT EXISTS statements on startup,
// ensuring the full database schema exists before any queries are executed.
func (s *Store) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		password_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		phone_number TEXT NOT NULL,
		balance_cents INTEGER NOT NULL,
		credit_score INTEGER NOT NULL DEFAULT 50,
		credit_limit_cents INTEGER NOT NULL DEFAULT 100000,
		is_active INTEGER NOT NULL DEFAULT 1,
		profile_color TEXT NOT NULL DEFAULT '#4ade80',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS payments (
		id TEXT PRIMARY KEY,
		sender_id TEXT NOT NULL,
		receiver_id TEXT NOT NULL,
		amount_cents INTEGER NOT NULL,
		total_amount_cents INTEGER NOT NULL,
		note TEXT,
		payment_type TEXT NOT NULL,
		total_installments INTEGER NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id) REFERENCES users (id),
		FOREIGN KEY (receiver_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS payment_requests (
		id TEXT PRIMARY KEY,
		requester_id TEXT NOT NULL,
		payer_id TEXT NOT NULL,
		amount_cents INTEGER NOT NULL,
		note TEXT,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (requester_id) REFERENCES users (id),
		FOREIGN KEY (payer_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS installments (
		id TEXT PRIMARY KEY,
		payment_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		amount_cents INTEGER NOT NULL,
		due_date TEXT NOT NULL,
		is_paid INTEGER NOT NULL DEFAULT 0,
		penalty_applied INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (payment_id) REFERENCES payments (id),
		FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS friends (
		user_id TEXT NOT NULL,
		friend_id TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, friend_id),
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (friend_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS friend_requests (
		id TEXT PRIMARY KEY,
		sender_id TEXT NOT NULL,
		receiver_id TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (receiver_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		link_view TEXT NOT NULL,
		is_seen INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		creator_id TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (creator_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS group_members (
		group_id TEXT NOT NULL,
		member_id TEXT NOT NULL,
		joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (group_id, member_id),
		FOREIGN KEY (group_id)  REFERENCES groups (id) ON DELETE CASCADE,
		FOREIGN KEY (member_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS group_invitations (
		id TEXT PRIMARY KEY,
		group_id TEXT NOT NULL,
		sender_id TEXT NOT NULL,
		receiver_id TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE,
		FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (receiver_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS user_settings (
    	user_id TEXT PRIMARY KEY,
    	email_notifications INTEGER NOT NULL DEFAULT 1,
    	is_discoverable INTEGER NOT NULL DEFAULT 1,
    	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
		CREATE TABLE IF NOT EXISTS wallet_transactions (
    	id TEXT PRIMARY KEY,
    	user_id TEXT NOT NULL,
    	amount_cents INTEGER NOT NULL,
    	transaction_type TEXT CHECK(transaction_type IN ('deposit', 'withdrawal')) NOT NULL,
    	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
		CREATE TABLE IF NOT EXISTS credit_history (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		credit_score INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS credit_score_log (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		score INTEGER NOT NULL,
		delta INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
