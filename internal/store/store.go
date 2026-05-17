package store

import (
	"database/sql"

	// This driver connects the Go code to the SQLite database file system
	"github.com/you/p2p-bnpl/internal/models"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

// New initializes our database file
func New() (*Store, error) {

	// This creates a file called "app.db" if it doesn't exist
	// Adding _pragma=foreign_keys=1 forces SQLite to enforce foreign key rules
	db, err := sql.Open("sqlite", "app.db?_pragma=foreign_keys=1")
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}

	// Tell the database to create our tables if not yet created
	err = s.createTables()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) createTables() error {
	// users table
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT,
		email TEXT,
		balance REAL
	);`

	_, err := s.db.Exec(query)

	if err != nil {
		return err
	}

	// payments table
	// FOREIGN KEY's ensure both user objects are found in users table
	query = `
	CREATE TABLE IF NOT EXISTS payments (
	id TEXT PRIMARY KEY,
	sender_id TEXT,
	receiver_id TEXT,
	amount REAL,
	note TEXT,
	FOREIGN KEY (sender_id) REFERENCES users (id),
	FOREIGN KEY (receiver_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)

	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateUser(u *models.User) error {
	query := `
	INSERT INTO users (id, name, email, balance) 
	values(?,?,?,?);`

	_, err := s.db.Exec(query, u.ID, u.Name, u.Email, u.Balance)
	return err
}

func (s *Store) GetUser(id string) (*models.User, error) {
	query := `
	SELECT id, name, email, balance FROM users WHERE id = ?;`

	row := s.db.QueryRow(query, id)

	var u models.User

	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Balance)

	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (s *Store) Pay(p *models.Payment) error {
	transaction, err := s.db.Begin()

	if err != nil {
		return err
	}

	// If the function exits early due to an error, all changes are discarded
	defer transaction.Rollback()

	// Update senders balance
	query := `UPDATE users SET balance = balance - ? WHERE id = ?;`

	_, err = transaction.Exec(query, p.Amount, p.SenderID)

	if err != nil {
		return err
	}

	// Update receivers balance
	query = `UPDATE users SET balance = balance + ? WHERE id = ?;`

	_, err = transaction.Exec(query, p.Amount, p.ReceiverID)

	if err != nil {
		return err
	}

	// Create a new row in the senders payment table
	query = `INSERT into payments (id, sender_id, receiver_id, amount, note) values (?,?,?,?,?);`

	_, err = transaction.Exec(query, p.ID, p.SenderID, p.ReceiverID, p.Amount, p.Note)

	if err != nil {
		return err
	}

	return transaction.Commit()
}
