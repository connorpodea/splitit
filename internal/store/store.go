package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	// This driver connects the Go code to the SQLite database file system

	_ "modernc.org/sqlite"
)

func create_new_ID() string {
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

	s := &Store{db: db}

	// Automate table generation schema rules execution
	err = s.createTables()
	if err != nil {
		db.Close() // Clean up the connection if table creation fails
		return nil, fmt.Errorf("database initialization failed during custom path schema setup: %w", err)
	}

	return s, nil
}

func (s *Store) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		password_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		phone_number TEXT NOT NULL,
		balance REAL NOT NULL,
		credit_score INTEGER NOT NULL DEFAULT 50,
		credit_limit REAL NOT NULL DEFAULT 1000.00,
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
		amount REAL NOT NULL,
		total_amount REAL NOT NULL,
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
		amount REAL NOT NULL,
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
		amount REAL NOT NULL,
		due_date TEXT NOT NULL,
		is_paid INTEGER NOT NULL DEFAULT 0,
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
		FOREIGN KEY (user_id) REFERENCES users (id),
		FOREIGN KEY (friend_id) REFERENCES users (id)
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
		FOREIGN KEY (sender_id) REFERENCES users (id),
		FOREIGN KEY (receiver_id) REFERENCES users (id)
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
		FOREIGN KEY (user_id) REFERENCES users (id)
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
		FOREIGN KEY (group_id)  REFERENCES groups (id),
		FOREIGN KEY (member_id) REFERENCES users (id)
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
		FOREIGN KEY (group_id) REFERENCES groups (id),
		FOREIGN KEY (sender_id) REFERENCES users (id),
		FOREIGN KEY (receiver_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS user_settings (
    	user_id TEXT PRIMARY KEY,
    	theme TEXT NOT NULL DEFAULT 'light',
    	email_notifications INTEGER NOT NULL DEFAULT 1, -- 1 = true
    	is_discoverable INTEGER NOT NULL DEFAULT 1, -- 1 = true
    	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	return nil
}
