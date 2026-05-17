package store

import (
	"database/sql"

	// This driver connects the Go code to the SQLite database file system
	"github.com/connorpodea/splitit/internal/models"
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

	// Create our tables (if not yet created)
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
		phone_number TEXT,
		balance REAL,
		credit_score INTEGER DEFAULT 600,
		credit_limit REAL DEFAULT 1000.00,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
	total_amount REAL,
	note TEXT,
	payment_type TEXT,
	total_installments INTEGER,
	status TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (sender_id) REFERENCES users (id),
	FOREIGN KEY (receiver_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// installments table
	// FOREIGN KEY's ensure the payment exists in the payments table and the user exists in the user table
	query = `
	CREATE TABLE IF NOT EXISTS installments (
	id TEXT PRIMARY KEY,
	payment_id TEXT,
	user_id TEXT,
	amount REAL,
	due_date TEXT,
	is_paid INTEGER DEFAULT 0,
	FOREIGN KEY (payment_id) REFERENCES payments (id),
	FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateUser(u *models.User) error {
	query := `
	INSERT INTO users (id, name, email, phone_number, balance) 
	values(?,?,?,?,?);`

	_, err := s.db.Exec(query, u.ID, u.Name, u.Email, u.PhoneNumber, u.Balance)
	return err
}

func (s *Store) GetUser(id string) (*models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users 
	WHERE id = ?;`

	row := s.db.QueryRow(query, id)

	var u models.User

	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PhoneNumber, &u.Balance, &u.CreditScore, &u.CreditLimit, &u.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (s *Store) ListUsers() ([]*models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []*models.User

	for rows.Next() {
		var u models.User

		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PhoneNumber, &u.Balance, &u.CreditScore, &u.CreditLimit, &u.CreatedAt)
		if err != nil {
			return nil, err
		}

		users = append(users, &u)
	}
	return users, nil
}

func (s *Store) Pay(p *models.Payment) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return err
	}

	// If the function exits early due to an error, discard all changes
	defer transaction.Rollback()

	// Update the senders balance to deduct the upfront payment
	query := `UPDATE users SET balance = balance - ? WHERE id = ?;`

	_, err = transaction.Exec(query, p.Amount, p.SenderID)
	if err != nil {
		return err
	}

	// Update receivers balance to receive the total payment
	query = `UPDATE users SET balance = balance + ? WHERE id = ?;`

	_, err = transaction.Exec(query, p.TotalAmount, p.ReceiverID)
	if err != nil {
		return err
	}

	// Create a new row in the senders payment table
	query = `INSERT INTO payments (id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status) 
             VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = transaction.Exec(query, p.ID, p.SenderID, p.ReceiverID, p.Amount, p.TotalAmount, p.Note, p.PaymentType, p.TotalInstallments, p.Status)

	if err != nil {
		return err
	}

	return transaction.Commit()
}