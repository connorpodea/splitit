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

// current tables:
// users
// payments
// payment_requests
// installments
// friends
// friend_requests

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

	// payment_requests table
	query = `CREATE TABLE IF NOT EXISTS payment_requests (
	id TEXT PRIMARY KEY,
	requester_id TEXT,
	payer_id TEXT,
	amount REAL,
	note TEXT,
	status TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (requester_id) REFERENCES users (id),
	FOREIGN KEY (payer_id) REFERENCES users (id)
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
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (payment_id) REFERENCES payments (id),
	FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// friends table
	// FOREIGN KEY's ensure that the users exist within the database
	// PRIMARY KEY ensures that friendships cannot duplicate
	query = `
	CREATE TABLE IF NOT EXISTS friends (
	user_id TEXT,
	friend_id TEXT,
	PRIMARY KEY (user_id, friend_id),
	FOREIGN KEY (user_id) references users (id),
	FOREIGN KEY (friend_id) references users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// friend requests table
	// FOREIGN KEY's ensure that the users exist within the database
	query = `
	CREATE TABLE IF NOT EXISTS friend_requests (
	id TEXT PRIMARY KEY,
	sender_id TEXT,
	receiver_id TEXT,
	accepted INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (sender_id) REFERENCES users (id),
	FOREIGN KEY (receiver_id) REFERENCES users (id)
	)`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateUser(user *models.User) error {
	query := `
	INSERT INTO users 
	(id, name, email, phone_number, balance, credit_score, credit_limit) 
	VALUES (?, ?, ?, ?, ?, ?, ?);`

	_, err := s.db.Exec(query, user.ID, user.Name, user.Email, user.PhoneNumber, user.Balance, user.CreditScore, user.CreditLimit)
	return err
}

func (s *Store) GetUser(userID string) (*models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users 
	WHERE id = ?;`

	row := s.db.QueryRow(query, userID)

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
		return nil, fmt.Errorf("Failed to list all users: %v", err)
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
	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Store) Pay(payment *models.Payment) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return err
	}

	// If the function exits early due to an error, discard all changes
	defer transaction.Rollback()

	// Update the senders balance to deduct the upfront payment
	query := `
	UPDATE users 
	SET balance = balance - ? 
	WHERE id = ?;`

	_, err = transaction.Exec(query, payment.Amount, payment.SenderID)
	if err != nil {
		return err
	}

	// Update receivers balance to receive the total payment
	query = `
	UPDATE users 
	SET balance = balance + ? 
	WHERE id = ?;`

	_, err = transaction.Exec(query, payment.TotalAmount, payment.ReceiverID)
	if err != nil {
		return err
	}

	// Create a new row in the senders payment table
	query = `
	INSERT INTO payments 
	(id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = transaction.Exec(query, payment.ID, payment.SenderID, payment.ReceiverID, payment.Amount, payment.TotalAmount, payment.Note, payment.PaymentType, payment.TotalInstallments, payment.Status)

	if err != nil {
		return err
	}

	return transaction.Commit()
}

func (s *Store) CreateBNPLLoan(payment *models.Payment) error {
	if payment.TotalInstallments == 0 {
		return fmt.Errorf("Total installments cannot be zero")
	}

	// Variable for the raw price before fees
	itemPrice := payment.TotalAmount

	var feeRate float64 = 0.00
	// Calculate the risk fee based on their credit health score iff they are paying over time
	if payment.TotalInstallments > 1 {
		sender, err := s.GetUser(payment.SenderID)
		if err != nil {
			return fmt.Errorf("Failed to fetch sender metrics: %v", err)
		}
		feeRate = s.CalculateFeeRate(sender.CreditScore)
	}

	// Update the senders purchase amount by the fee rate
	totalDebt := itemPrice + (feeRate * itemPrice)

	baseAmount := math.Floor((totalDebt/float64(payment.TotalInstallments))*100) / 100

	// Calculate the leftover pennies to add to initial installment
	remainder := totalDebt - (baseAmount * float64(payment.TotalInstallments))

	// The app treasury pays the seller the full item price immediately
	fundingPayment := &models.Payment{
		ID:                fmt.Sprintf("fund_%s", payment.ID),
		SenderID:          "app_treasury",
		ReceiverID:        payment.ReceiverID,
		Amount:            itemPrice,
		TotalAmount:       itemPrice,
		PaymentType:       "treasury_funding",
		TotalInstallments: 1,
		Status:            "completed",
		Note:              fmt.Sprintf("Treasury funded purchase for payment %s", payment.ID),
	}

	err := s.Pay(fundingPayment)
	if err != nil {
		return fmt.Errorf("Treasure merchant funding failed: %v", err)
	}

	// Charge the buyer their upfront down payment back to the app treasury
	payment.Amount = baseAmount + remainder
	payment.TotalAmount = totalDebt
	payment.ReceiverID = "app_treasury"
	payment.PaymentType = "installment"

	// Trigger the ledger execution
	err = s.Pay(payment)
	if err != nil {
		return fmt.Errorf("Upfront payment failed: %v", err)
	}

	// Restore back to the total debt for the database installment records
	payment.TotalAmount = itemPrice + (feeRate * itemPrice)

	// Build the remaining debt calendar schedule into the installments table
	currentTime := time.Now()

	for i := uint8(1); i <= payment.TotalInstallments; i++ {
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
		installmentID := fmt.Sprintf("inst_%s_%d", payment.ID, i)

		isPaidInt := 0
		if isPaid {
			isPaidInt = 1
		}

		query := `
		INSERT INTO installments 
		(id, payment_id, user_id, amount, due_date, is_paid)
		VALUES (?,?,?,?,?,?);`
		_, err = s.db.Exec(query, installmentID, payment.ID, payment.SenderID, installmentAmount, dueDate.Format("2006-01-02"), isPaidInt)
		if err != nil {
			return fmt.Errorf("Failed to save installment %d: %v", i, err)
		}
	}
	return nil
}

func (s *Store) CalculateFeeRate(creditScore uint8) float64 {
	switch {
	case creditScore >= 90:
		return 0.01
	case creditScore >= 75:
		return 0.02
	case creditScore >= 50:
		return 0.03
	default:
		return 0.07
	}
}

// ************************************************
// *************** SOCIAL FEATURES ****************
// ************************************************

func (s *Store) SendFriendRequest(request *models.FriendRequest) error {
	query := `
	INSERT INTO friend_requests 
	(id, sender_id, receiver_id)
	VALUES (?,?,?);`

	_, err := s.db.Exec(query, request.ID, request.SenderID, request.ReceiverID)
	if err != nil {
		return fmt.Errorf("Unable to process the friend request %v", err)
	}
	return nil
}

func (s *Store) ListIncomingFriendRequests(userID string) ([]*models.FriendRequest, error) {
	query := `
	SELECT id, sender_id, receiver_id, accepted, created_at
	FROM friend_requests
	WHERE receiver_id = ? AND accepted = 0;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user's incoming friend requests: %v", err)
	}
	defer rows.Close()

	// collect a list of all the requests
	var requests []*models.FriendRequest
	for rows.Next() {
		var r models.FriendRequest
		err := rows.Scan(&r.ID, &r.SenderID, &r.ReceiverID, &r.Accepted, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		requests = append(requests, &r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

func (s *Store) ListOutgoingFriendRequests(userID string) ([]*models.FriendRequest, error) {
	// fill this out
	return nil, nil
}

func (s *Store) AcceptFriendRequest(requestID, senderID, receiverID string) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	// update the friend requests table
	query := `
	UPDATE friend_requests 
	SET accepted = 1 
	WHERE id = ?;`

	_, err = transaction.Exec(query, requestID)
	if err != nil {
		return err
	}

	// update the friends table to show the new relationship
	// (for both users, ie user A is friends with user B, and user B is friends with user A)
	query = `
	INSERT OR IGNORE INTO friends 
	VALUES (?1,?2), (?2,?1);`

	_, err = transaction.Exec(query, senderID, receiverID)
	if err != nil {
		return err
	}
	return transaction.Commit()
}

func (s *Store) DeclineFriendRequest(requestID string) error {
	query := `
	DELETE FROM friend_requests
	WHERE id = ?;`

	_, err := s.db.Exec(query, requestID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) RemoveFriendMutual(userID, friendID string) error {
	query := `
	DELETE FROM friends
	WHERE (user_id = ?1 AND friend_id = ?2) OR (user_id = ?2 AND friend_id = ?1);`

	_, err := s.db.Exec(query, userID, friendID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) ListFriends(userID string) ([]*models.Profile, error) {
	// Look up all friends of the current user and use the friend ids to return their profile
	query := `
	SELECT u.id, u.name, u.email, u.phone_number
	FROM friends AS fr
	JOIN users AS u ON fr.friend_id = u.id
	WHERE fr.user_id = ?;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []*models.Profile

	for rows.Next() {
		var f models.Profile

		err := rows.Scan(&f.ID, &f.Name, &f.Email, &f.PhoneNumber)
		if err != nil {
			return nil, err
		}
		friends = append(friends, &f)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return friends, nil
}

func (s *Store) CreatePaymentRequest(request *models.PaymentRequest) error {
	query := `
	INSERT INTO payment_requests
	(id, requester_id, payer_id, amount, note, status, created_at)
	VALUES (?,?,?,?,?,?,?);`

	_, err := s.db.Exec(query, request.ID, request.RequesterID, request.PayerID, request.Amount, request.Note, request.Status, request.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) ListIncomingPaymentRequests(userID string) ([]*models.PaymentRequest, error) {
	query := `
	SELECT id, requester_id, payer_id, amount, note, status,created_at
	FROM payment_requests
	WHERE payer_id = ? AND status = 'pending'`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.PaymentRequest

	for rows.Next() {
		var r models.PaymentRequest

		err = rows.Scan(&r.ID, &r.RequesterID, &r.PayerID, &r.Amount, &r.Note, &r.Status, &r.CreatedAt)

		if err != nil {
			return nil, err
		}
		requests = append(requests, &r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

func (s *Store) ListOutgoingPaymentRequests(userID string) ([]*models.PaymentRequest, error) {
	// fill this out
	return nil, nil
}

func (s *Store) UpdatePaymentRequestStatus(paymentID, new_status string) error {
	query := `
	UPDATE payment_requests
	SET status = ?
	WHERE id = ?;`

	_, err := s.db.Exec(query, paymentID, new_status)
	if err != nil {
		return err
	}
	return nil
}
