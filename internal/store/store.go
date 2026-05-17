package store

import (
	"database/sql"
	"fmt"
	"math"
	"time"

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
		credit_score INTEGER DEFAULT 50,
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
INSERT INTO users (id, name, email, phone_number, balance, credit_score, credit_limit) 
VALUES (?, ?, ?, ?, ?, ?, ?);`

	_, err := s.db.Exec(query, u.ID, u.Name, u.Email, u.PhoneNumber, u.Balance, u.CreditScore, u.CreditLimit)
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

func (s *Store) CreateBNPLLoan(p *models.Payment) error {
	if p.TotalInstallments == 0 {
		return fmt.Errorf("Total installments cannot be zero")
	}

	// Variable for the raw price before fees
	itemPrice := p.TotalAmount

	// Check the senders credit metrics
	sender, err := s.GetUser(p.SenderID)
	if err != nil {
		return fmt.Errorf("Failed to fetch sender metrics: %v", err)
	}

	// Calculate the risk fee based on their credit health score
	feeRate := s.CalculateFeeRate(sender.CreditScore)

	// Update the senders purchase amount by the fee rate
	totalDebt := itemPrice + (feeRate * itemPrice)

	baseAmount := math.Floor((totalDebt/float64(p.TotalInstallments))*100) / 100

	// Calculate the leftover pennies to add to initial installment
	remainder := totalDebt - (baseAmount * float64(p.TotalInstallments))

	// The app treasury pays the seller the full item price immediately
	fundingPayment := &models.Payment{
		ID:                fmt.Sprintf("fund_%s", p.ID),
		SenderID:          "app_treasury", // Your app pools the capital
		ReceiverID:        p.ReceiverID,   // The seller receives it
		Amount:            itemPrice,
		TotalAmount:       itemPrice,
		PaymentType:       "treasury_funding",
		TotalInstallments: 1,
		Status:            "completed",
		Note:              fmt.Sprintf("Treasury funded purchase for payment %s", p.ID),
	}
	err = s.Pay(fundingPayment)
	if err != nil {
		return fmt.Errorf("Treasure merchant funding failed: %v", err)
	}

	// Charge the buyer their upfront down payment back to the app treasury
	p.Amount = baseAmount + remainder
	p.TotalAmount = totalDebt
	p.ReceiverID = "app_treasury"
	p.PaymentType = "installment"

	// Trigger the ledger execution
	err = s.Pay(p)
	if err != nil {
		return fmt.Errorf("Upfront payment failed: %v", err)
	}

	// Restore back to the total debt for the database installment records
	p.TotalAmount = itemPrice + (feeRate * itemPrice)

	// Build the remaining debt calendar schedule into the installments table
	currentTime := time.Now()

	for i := uint8(1); i <= p.TotalInstallments; i++ {
		var installmentAmount float64
		var isPaid bool
		var dueDate time.Time

		if i == 1 {
			// Installment 1 is paid upfront during s.Pay(p)
			installmentAmount = baseAmount + remainder
			isPaid = true
			dueDate = currentTime
		} else {
			installmentAmount = baseAmount
			isPaid = false
			// Stagger deadlines by 14 days multiplies by the installment index
			dueDate = currentTime.AddDate(0, 0, int(i-1)*14)
		}

		// Generate a structured identifier for each installment row
		installmentID := fmt.Sprintf("inst_%s_%d", p.ID, i)

		isPaidInt := 0
		if isPaid {
			isPaidInt = 1
		}

		query := `INSERT INTO installments (id, payment_id, user_id, amount, due_date, is_paid)
		VALUES (?,?,?,?,?,?)`

		_, err = s.db.Exec(query, installmentID, p.ID, p.SenderID, installmentAmount, dueDate.Format("2006-01-02"), isPaidInt)
		if err != nil {
			return fmt.Errorf("Failed to save installment %d: %v", i, err)
		}
	}
	return nil
}

func (s *Store) CalculateFeeRate(creditScore uint8) float64 {
	switch {
	case creditScore >= 90:
		return 0.00
	case creditScore >= 75:
		return 0.01
	case creditScore >= 50:
		return 0.02
	default:
		return 0.07
	}
}
