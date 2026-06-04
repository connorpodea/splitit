package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	// This driver connects the Go code to the SQLite database file system
	"github.com/connorpodea/splitit/internal/models"
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
	// users table
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id            TEXT    PRIMARY KEY,
		password_hash TEXT    NOT NULL,
		name          TEXT    NOT NULL,
		email         TEXT    NOT NULL,
		phone_number  TEXT    NOT NULL,
		balance       REAL    NOT NULL,
		credit_score  INTEGER NOT NULL DEFAULT 50,
		credit_limit  REAL    NOT NULL DEFAULT 1000.00,
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	// payments table
	// FOREIGN KEY's ensure both user objects are found in the users table
	// note is intentionally nullable — it is an optional field
	query = `
	CREATE TABLE IF NOT EXISTS payments (
		id                 TEXT    PRIMARY KEY,
		sender_id          TEXT    NOT NULL,
		receiver_id        TEXT    NOT NULL,
		amount             REAL    NOT NULL,
		total_amount       REAL    NOT NULL,
		note               TEXT,
		payment_type       TEXT    NOT NULL,
		total_installments INTEGER NOT NULL,
		status             TEXT    NOT NULL,
		created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id)   REFERENCES users (id),
		FOREIGN KEY (receiver_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// payment_requests table
	// note is intentionally nullable — it is an optional field
	query = `
	CREATE TABLE IF NOT EXISTS payment_requests (
		id           TEXT    PRIMARY KEY,
		requester_id TEXT    NOT NULL,
		payer_id     TEXT    NOT NULL,
		amount       REAL    NOT NULL,
		note         TEXT,
		status       TEXT    NOT NULL,
		created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (requester_id) REFERENCES users (id),
		FOREIGN KEY (payer_id)     REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// installments table
	// FOREIGN KEY's ensure the payment exists in the payments table and the user exists in the users table
	query = `
	CREATE TABLE IF NOT EXISTS installments (
		id         TEXT    PRIMARY KEY,
		payment_id TEXT    NOT NULL,
		user_id    TEXT    NOT NULL,
		amount     REAL    NOT NULL,
		due_date   TEXT    NOT NULL,
		is_paid    INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (payment_id) REFERENCES payments (id),
		FOREIGN KEY (user_id)    REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// friends table
	// FOREIGN KEY's ensure that both users exist within the database
	// PRIMARY KEY ensures that friendships cannot duplicate
	query = `
	CREATE TABLE IF NOT EXISTS friends (
		user_id    TEXT     NOT NULL,
		friend_id  TEXT     NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, friend_id),
		FOREIGN KEY (user_id)   REFERENCES users (id),
		FOREIGN KEY (friend_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// friend requests table
	// FOREIGN KEY's ensure that both users exist within the database
	query = `
	CREATE TABLE IF NOT EXISTS friend_requests (
		id          TEXT     PRIMARY KEY,
		sender_id   TEXT     NOT NULL,
		receiver_id TEXT     NOT NULL,
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id)   REFERENCES users (id),
		FOREIGN KEY (receiver_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// notifications table
	query = `
	CREATE TABLE IF NOT EXISTS notifications (
		id         TEXT     PRIMARY KEY,
		user_id    TEXT     NOT NULL,
		type       TEXT     NOT NULL,
		title      TEXT     NOT NULL,
		body       TEXT     NOT NULL,
		link_view  TEXT     NOT NULL,
		is_seen    INTEGER  NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// groups table
	query = `
	CREATE TABLE IF NOT EXISTS groups (
		id         TEXT     PRIMARY KEY,
		name       TEXT     NOT NULL,
		creator_id TEXT     NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (creator_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// group members table
	// PRIMARY KEY ensures a user cannot appear in the same group twice
	query = `
	CREATE TABLE IF NOT EXISTS group_members (
		group_id   TEXT     NOT NULL,
		member_id  TEXT     NOT NULL,
		joined_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (group_id, member_id),
		FOREIGN KEY (group_id)  REFERENCES groups (id),
		FOREIGN KEY (member_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	// group invitations table — no status column; accepted invitations create a membership row,
	// declined invitations simply delete the row (mirrors the friend_requests pattern)
	query = `
	CREATE TABLE IF NOT EXISTS group_invitations (
		id          TEXT     PRIMARY KEY,
		group_id    TEXT     NOT NULL,
		sender_id   TEXT     NOT NULL,
		receiver_id TEXT     NOT NULL,
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id)    REFERENCES groups (id),
		FOREIGN KEY (sender_id)   REFERENCES users (id),
		FOREIGN KEY (receiver_id) REFERENCES users (id)
	);`

	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateUser(user *models.User) error {
	query := `
	INSERT INTO users 
	(id, password_hash, name, email, phone_number, balance, credit_score, credit_limit) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?);`

	_, err := s.db.Exec(query, user.ID, user.PasswordHash, user.Name, user.Email, user.PhoneNumber, user.Balance, user.CreditScore, user.CreditLimit)
	if err != nil {
		return fmt.Errorf("failed to write profile for user ID '%s' to database: %w", user.ID, err)
	}
	return nil
}

func (s *Store) GetUser(userID string) (*models.User, error) {
	query := `
	SELECT id, password_hash, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users 
	WHERE id = ?;`
	var user models.User

	err := s.db.QueryRow(query, userID).Scan(&user.ID, &user.PasswordHash, &user.Name, &user.Email, &user.PhoneNumber, &user.Balance, &user.CreditScore, &user.CreditLimit, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve full account parameters for user ID '%s': %w", userID, err)
	}

	return &user, nil
}

func (s *Store) GetProfile(id string) (*models.Profile, error) {
	query := `
	SELECT id, name, email, phone_number, created_at 
	FROM users 
	WHERE id = ?;`
	var profile models.Profile

	err := s.db.QueryRow(query, id).Scan(&profile.ID, &profile.Name, &profile.Email, &profile.PhoneNumber, &profile.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to pull minimalist public profile metadata for user ID '%s': %w", id, err)
	}

	return &profile, nil
}

func (s *Store) ListUsers() ([]models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users
	ORDER BY name ASC;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute full users table scan lookup query: %w", err)
	}
	defer rows.Close()

	users := []models.User{}

	for rows.Next() {
		var u models.User

		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PhoneNumber, &u.Balance, &u.CreditScore, &u.CreditLimit, &u.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row into user data struct during list aggregation: %w", err)
		}

		users = append(users, u)
	}
	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during users rows iteration loop: %w", err)
	}
	return users, nil
}

func (s *Store) ListProfiles() ([]models.Profile, error) {
	query := `
	SELECT id, name, email, phone_number, created_at
	FROM users
	ORDER BY name ASC;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute global public profiles query row collection: %w", err)
	}
	defer rows.Close()

	profiles := []models.Profile{}

	for rows.Next() {
		var profile models.Profile

		err := rows.Scan(&profile.ID, &profile.Name, &profile.Email, &profile.PhoneNumber, &profile.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching profile data mapping structure: %w", err)
		}

		profiles = append(profiles, profile)
	}
	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during profiles row iteration loop: %w", err)
	}
	return profiles, nil
}

func (s *Store) SearchProfiles(queryStr string) ([]models.Profile, error) {
	// Transform the query string into a SQL wildcard match format
	wildcardQuery := fmt.Sprintf("%%%s%%", queryStr)

	query := `
	SELECT id, name, email, phone_number, created_at
	FROM users
	WHERE name LIKE ? OR email LIKE ?
	ORDER BY name ASC;`

	rows, err := s.db.Query(query, wildcardQuery, wildcardQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute profiles global search query for token '%s': %w", queryStr, err)
	}
	defer rows.Close()

	profiles := []models.Profile{}

	for rows.Next() {
		var p models.Profile
		err = rows.Scan(&p.ID, &p.Name, &p.Email, &p.PhoneNumber, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching profile search record row: %w", err)
		}
		profiles = append(profiles, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor failure mid-stream during profiles search loop execution:: %w", err)
	}

	return profiles, nil
}

func (s *Store) Pay(payment *models.Payment) error {
	// Generate a unique transaction identifier inside the store so the client never controls primary keys
	if payment.ID == "" {
		payment.ID = create_new_ID()
	}

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ledger settlement aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, discard all changes
	defer transaction.Rollback()

	// Block the sender from going into negative balance
	var currentBalance float64
	balanceQuery := `
	SELECT balance
	FROM users
	WHERE id = ?;`
	err = transaction.QueryRow(balanceQuery, payment.SenderID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to verify sender funds: %w", err)
	}

	if currentBalance < payment.Amount {
		return fmt.Errorf("ledger settlement rejected: insufficient liquid funds (ID: '%s' attempted to pay $%.2f but only has $%.2f)", payment.SenderID, payment.Amount, currentBalance)
	}

	// Update the senders balance to deduct the upfront payment
	query := `
	UPDATE users 
	SET balance = balance - ? 
	WHERE id = ?;`

	_, err = transaction.Exec(query, payment.Amount, payment.SenderID)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to clear balance deduction of $%.2f from sender ID '%s': %w", payment.Amount, payment.SenderID, err)
	}

	// Update receivers balance to receive the total payment
	query = `
	UPDATE users 
	SET balance = balance + ? 
	WHERE id = ?;`

	_, err = transaction.Exec(query, payment.TotalAmount, payment.ReceiverID)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to credit balance allocation of $%.2f to receiver ID '%s': %w", payment.TotalAmount, payment.ReceiverID, err)
	}

	// Create a new row in the senders payment table
	query = `
	INSERT INTO payments 
	(id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = transaction.Exec(query, payment.ID, payment.SenderID, payment.ReceiverID, payment.Amount, payment.TotalAmount, payment.Note, payment.PaymentType, payment.TotalInstallments, payment.Status)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: historical transaction entry creation with ID '%s' rejected by database: %w", payment.ID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical engine tracking mismatch: failed to write block modifications to disk on final commit sequence: %w", err)
	}

	s.CreateNotification(&models.Notification{
		UserID:   payment.ReceiverID,
		Type:     "payment_received",
		Title:    "Payment received",
		Body:     fmt.Sprintf("@%s paid you $%.2f", payment.SenderID, payment.Amount),
		LinkView: "activity",
	})
	return nil
}

func (s *Store) GetPayment(paymentID string) (*models.Payment, error) {
	query := `
	SELECT id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status, created_at
	FROM payments
	WHERE id = ?;`

	var payment models.Payment

	err := s.db.QueryRow(query, paymentID).Scan(&payment.ID, &payment.SenderID, &payment.ReceiverID, &payment.Amount, &payment.TotalAmount, &payment.Note, &payment.PaymentType, &payment.TotalInstallments, &payment.Status, &payment.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment record for transaction ID '%s' : %w", paymentID, err)
	}
	return &payment, nil
}

func (s *Store) GetPaymentWithInstallments(paymentID string) (*models.PaymentWithInstallments, error) {
	query := `
	SELECT id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status, created_at
    FROM payments
    WHERE id = ?;`

	var pwi models.PaymentWithInstallments

	err := s.db.QueryRow(query, paymentID).Scan(
		&pwi.Payment.ID,
		&pwi.Payment.SenderID,
		&pwi.Payment.ReceiverID,
		&pwi.Payment.Amount,
		&pwi.Payment.TotalAmount,
		&pwi.Payment.Note,
		&pwi.Payment.PaymentType,
		&pwi.Payment.TotalInstallments,
		&pwi.Payment.Status,
		&pwi.Payment.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("payment transaction record ID '%s' not found: %w", paymentID, err)
		}
		return nil, fmt.Errorf("failed to retrieve master payment parameters for ID '%s': %w", paymentID, err)
	}

	query = `
    SELECT id, payment_id, user_id, amount, due_date, is_paid, created_at
    FROM installments
    WHERE payment_id = ?
    ORDER BY due_date ASC;`

	rows, err := s.db.Query(query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute installment timeline lookup for master payment ID '%s': %w", paymentID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var i models.Installment
		var isPaidInt int // Intermediary variable to scan SQLite 0/1 integer flags safely

		err = rows.Scan(&i.ID, &i.PaymentID, &i.UserWithDebt, &i.Amount, &i.DueDate, &isPaidInt, &i.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack individual installment schedule row segment: %w", err)
		}

		i.IsPaid = isPaidInt == 1
		pwi.Installments = append(pwi.Installments, i)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor failure during payment installments assembly pipeline: %w", err)
	}
	return &pwi, nil
}

func (s *Store) ListPaymentsReceived(userID string) ([]models.Payment, error) {
	query := `
    SELECT id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status, created_at
    FROM payments
    WHERE receiver_id = ?
    ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up inbound payment history ledger for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	payments := []models.Payment{}

	for rows.Next() {
		var p models.Payment

		err = rows.Scan(&p.ID, &p.SenderID, &p.ReceiverID, &p.Amount, &p.TotalAmount, &p.Note, &p.PaymentType, &p.TotalInstallments, &p.Status, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack settlement fields into internal payment history model array: %w", err)
		}

		payments = append(payments, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected an unexpected structural read failure inside inbound transaction pipeline iteration for user ID '%s': %w", userID, err)
	}
	return payments, nil
}

func (s *Store) ListPaymentsSent(userID string) ([]models.Payment, error) {
	query := `
    SELECT id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status, created_at
    FROM payments
    WHERE sender_id = ?
    ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up inbound payment history ledger for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	payments := []models.Payment{}

	for rows.Next() {
		var p models.Payment

		err = rows.Scan(&p.ID, &p.SenderID, &p.ReceiverID, &p.Amount, &p.TotalAmount, &p.Note, &p.PaymentType, &p.TotalInstallments, &p.Status, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack settlement fields into internal payment history model array: %w", err)
		}

		payments = append(payments, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected an unexpected structural read failure inside inbound transaction pipeline iteration for user ID '%s': %w", userID, err)
	}
	return payments, nil
}

func (s *Store) CreatePaymentRequest(request *models.PaymentRequest) error {
	// Generate a unique request identifier inside the store so the client never controls primary keys
	if request.ID == "" {
		request.ID = create_new_ID()
	}

	query := `
	INSERT INTO payment_requests
	(id, requester_id, payer_id, amount, note, status)
	VALUES (?,?,?,?,?,?);`

	_, err := s.db.Exec(query, request.ID, request.RequesterID, request.PayerID, request.Amount, request.Note, request.Status)
	if err != nil {
		return fmt.Errorf("failed to push open payment demand requisition with invoice ID '%s' into table ledgers: %w", request.ID, err)
	}
	s.CreateNotification(&models.Notification{
		UserID:   request.PayerID,
		Type:     "payment_request",
		Title:    "Payment request",
		Body:     fmt.Sprintf("@%s is requesting $%.2f from you", request.RequesterID, request.Amount),
		LinkView: "activity",
	})
	return nil
}

func (s *Store) ListIncomingPaymentRequests(userID string) ([]models.PaymentRequest, error) {
	query := `
	SELECT id, requester_id, payer_id, amount, note, status, created_at
	FROM payment_requests
	WHERE payer_id = ? AND status = 'pending'
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up outstanding pending payables collection dashboard records for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	requests := []models.PaymentRequest{}

	for rows.Next() {
		var r models.PaymentRequest

		err = rows.Scan(&r.ID, &r.RequesterID, &r.PayerID, &r.Amount, &r.Note, &r.Status, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack invoice variables into internal payment request structure array: %w", err)
		}
		requests = append(requests, r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected an unexpected structural read failure inside payables pipeline engine iteration for user ID '%s': %w", userID, err)
	}
	return requests, nil
}

func (s *Store) ListOutgoingPaymentRequests(userID string) ([]models.PaymentRequest, error) {
	query := `
	SELECT id, requester_id, payer_id, amount, note, status, created_at
	FROM payment_requests
	WHERE requester_id = ? AND status = 'pending'
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up outbound collectibles receivables list directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	requests := []models.PaymentRequest{}

	for rows.Next() {
		var r models.PaymentRequest

		err = rows.Scan(&r.ID, &r.RequesterID, &r.PayerID, &r.Amount, &r.Note, &r.Status, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to map outbound record fields safely to target collection model fields: %w", err)
		}

		requests = append(requests, r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected an unexpected structural read failure inside collectibles receivables pipeline iteration for user ID '%s': %w", userID, err)
	}
	return requests, nil
}

func (s *Store) UpdatePaymentRequestStatus(paymentID, newStatus string) error {
	query := `
	UPDATE payment_requests
	SET status = ?
	WHERE id = ?;`

	_, err := s.db.Exec(query, newStatus, paymentID)
	if err != nil {
		return fmt.Errorf("state machine error: failed to transition payment invoice request ID '%s' to state token '%s': %w", paymentID, newStatus, err)
	}
	return nil
}

func (s *Store) CreateBNPLLoan(payment *models.Payment) error {
	if payment.TotalInstallments == 0 {
		return fmt.Errorf("credit engine processing aborted: total plan financing installments cannot be evaluated at zero")
	}

	// Generate the master loan ID in the store
	if payment.ID == "" {
		payment.ID = create_new_ID()
	}

	// Reject the loan if the requested amount exceeds the buyer's available credit limit
	sender, err := s.GetUser(payment.SenderID)
	if err != nil {
		return fmt.Errorf("credit engine evaluation rejected: failed to query credit profile for buyer ID '%s': %w", payment.SenderID, err)
	}
	if payment.TotalAmount > sender.CreditLimit {
		return fmt.Errorf("credit engine rejected: requested loan amount of $%.2f exceeds available credit limit of $%.2f for buyer ID '%s'", payment.TotalAmount, sender.CreditLimit, payment.SenderID)
	}

	// Variable for the raw price before fees
	itemPrice := payment.TotalAmount

	var feeRate float64 = 0.00
	// Calculate the risk fee based on their credit health score iff they are paying over time
	if payment.TotalInstallments > 1 {
		feeRate = s.CalculateFeeRate(sender.CreditScore)
	}

	// Update the senders purchase amount by the fee rate
	totalDebt := itemPrice + (feeRate * itemPrice)

	baseAmount := math.Floor((totalDebt/float64(payment.TotalInstallments))*100) / 100

	// Calculate the leftover pennies to add to initial installment
	remainder := totalDebt - (baseAmount * float64(payment.TotalInstallments))

	// Step 1: Insert the master loan record to preserve historical data integrity before any fund movements
	// This anchors the loan in the payments table without touching balances directly
	payment.TotalAmount = totalDebt
	payment.Amount = itemPrice
	payment.PaymentType = "bnpl_loan_master"

	query := `
	INSERT INTO payments
	(id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = s.db.Exec(query, payment.ID, payment.SenderID, payment.ReceiverID, payment.Amount, payment.TotalAmount, payment.Note, payment.PaymentType, payment.TotalInstallments, payment.Status)
	if err != nil {
		return fmt.Errorf("loan processing aborted: failed to anchor master loan record ID '%s' into payments ledger: %w", payment.ID, err)
	}

	creditQuery := `
	UPDATE users
	SET credit_limit = credit_limit - ?
	WHERE id = ?;`

	_, err = s.db.Exec(creditQuery, itemPrice, payment.SenderID)
	if err != nil {
		return fmt.Errorf("loan processing aborted: failed to deduct $%.2f from credit limit for buyer ID '%s': %w", itemPrice, payment.SenderID, err)
	}

	// Step 2: Pay the Merchant — the app treasury injects the full item price to the seller immediately
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

	err = s.Pay(fundingPayment)
	if err != nil {
		return fmt.Errorf("loan processing aborted: upfront treasury capital injection for merchant pool failed: %w", err)
	}

	// Step 3: Collect the Down Payment — pull the first installment from the buyer back to the app treasury
	downPayment := &models.Payment{
		ID:                fmt.Sprintf("down_%s", payment.ID),
		SenderID:          payment.SenderID,
		ReceiverID:        "app_treasury",
		Amount:            baseAmount + remainder,
		TotalAmount:       baseAmount + remainder,
		PaymentType:       "installment",
		TotalInstallments: 1,
		Status:            "completed",
		Note:              fmt.Sprintf("Down payment for loan %s", payment.ID),
	}

	err = s.Pay(downPayment)
	if err != nil {
		return fmt.Errorf("loan processing aborted: down payment collection extraction failed for buyer ID '%s': %w", payment.SenderID, err)
	}

	// Step 4: Generate Installment Calendars — build the remaining debt schedule into the installments table
	currentTime := time.Now()

	for i := uint8(1); i <= payment.TotalInstallments; i++ {
		var installmentAmount float64
		var isPaid bool
		var dueDate time.Time

		if i == 1 {
			// Installment 1 is paid upfront during s.Pay(downPayment)
			installmentAmount = baseAmount + remainder
			isPaid = true
			dueDate = currentTime
		} else {
			installmentAmount = baseAmount
			isPaid = false
			// Stagger deadlines by 7 days multiplied by the installment index
			dueDate = currentTime.AddDate(0, 0, int(i-1)*7)
		}

		// Generate a structured identifier for each installment row
		installmentID := fmt.Sprintf("inst_%s_%d", payment.ID, i)

		isPaidInt := 0
		if isPaid {
			isPaidInt = 1
		}

		installmentQuery := `
		INSERT INTO installments 
		(id, payment_id, user_id, amount, due_date, is_paid)
		VALUES (?,?,?,?,?,?);`
		_, err = s.db.Exec(installmentQuery, installmentID, payment.ID, payment.SenderID, installmentAmount, dueDate.Format("2006-01-02"), isPaidInt)
		if err != nil {
			return fmt.Errorf("failed to save generated installment row segment %d for loan ID '%s' into database schedules: %w", i, payment.ID, err)
		}
	}
	return nil
}

func (s *Store) CalculateFeeRate(creditScore uint8) float64 {
	// Returns a fee multiplier based on the buyer's credit health score
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

func (s *Store) UpdateCreditScore(userID string, delta int) error {
	// Clamp the resulting score between 0 and 100 to prevent overflow or underflow
	query := `
	UPDATE users
	SET credit_score = MIN(100, MAX(0, credit_score + ?))
	WHERE id = ?`

	_, err := s.db.Exec(query, delta, userID)
	if err != nil {
		return fmt.Errorf("failed to apply credit score adjustment of %+d points to user ID '%s': %w", delta, userID, err)
	}
	return nil
}

func (s *Store) PayInstallment(installmentID, paymentID, userID string, amount float64) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("installment settlement aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, discard all changes
	defer transaction.Rollback()

	// Block the buyer from going into negative balance
	var currentBalance float64
	balanceQuery := `
	SELECT balance
	FROM users
	WHERE id = ?;`

	err = transaction.QueryRow(balanceQuery, userID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to verify buyer funds for user ID '%s': %w", userID, err)
	}
	if currentBalance < amount {
		return fmt.Errorf("installment settlement rejected: insufficient liquid funds (ID: '%s' attempted to pay $%.2f but only has $%.2f)", userID, amount, currentBalance)
	}

	// Deduct the installment amount from the buyer and credit the treasury
	deductQuery := `
	UPDATE users
	SET balance = balance - ?
	WHERE id = ?;`

	_, err = transaction.Exec(deductQuery, amount, userID)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to deduct $%.2f from buyer ID '%s': %w", amount, userID, err)
	}

	creditQuery := `
	UPDATE users
	SET balance = balance + ?
	WHERE id = ?;`

	_, err = transaction.Exec(creditQuery, amount, "app_treasury")
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to credit $%.2f to treasury: %w", amount, err)
	}

	// Mark the installment row as paid
	markPaidQuery := `
	UPDATE installments
	SET is_paid = 1
	WHERE id = ?;`

	_, err = transaction.Exec(markPaidQuery, installmentID)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to mark installment ID '%s' as paid: %w", installmentID, err)
	}

	// Check if this installment was paid late and penalize the credit score accordingly
	var dueDate string
	dueDateQuery := `
	SELECT due_date
	FROM installments
	WHERE id = ?;`

	err = transaction.QueryRow(dueDateQuery, installmentID).Scan(&dueDate)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to retrieve due date for late payment evaluation on installment ID '%s': %w", installmentID, err)
	}

	parsedDueDate, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to parse due date format for installment ID '%s': %w", installmentID, err)
	}

	// Deduct 5 credit score points for a late payment, reward 1 point for an on-time payment
	var scoreDelta int
	if time.Now().After(parsedDueDate) {
		scoreDelta = -5
	} else {
		scoreDelta = 1
	}

	scoreQuery := `
	UPDATE users
	SET credit_score = MIN(100, MAX(0, credit_score + ?))
	WHERE id = ?;`

	_, err = transaction.Exec(scoreQuery, scoreDelta, userID)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to apply credit score adjustment for user ID '%s': %w", userID, err)
	}

	// Check if all installments for this loan are now paid
	var unpaidCount int
	unpaidQuery := `
	SELECT COUNT(*)
	FROM installments
	WHERE payment_id = ? AND is_paid = 0;`

	err = transaction.QueryRow(unpaidQuery, paymentID).Scan(&unpaidCount)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to evaluate remaining debt obligations for loan ID '%s': %w", paymentID, err)
	}

	// If all installments are cleared, restore the full loan amount back to the buyer's credit limit and mark the master loan record as completed
	if unpaidCount == 0 {
		var loanAmount float64
		loanAmountQuery := `
		SELECT amount
		FROM payments
		WHERE id = ?;`

		err = transaction.QueryRow(loanAmountQuery, paymentID).Scan(&loanAmount)
		if err != nil {
			return fmt.Errorf("installment settlement failed: unable to retrieve original loan amount for credit limit restoration on loan ID '%s': %w", paymentID, err)
		}

		restoreQuery := `
		UPDATE users
		SET credit_limit = credit_limit + ?
		WHERE id = ?;`

		_, err = transaction.Exec(restoreQuery, loanAmount, userID)
		if err != nil {
			return fmt.Errorf("installment settlement failed: unable to restore $%.2f to credit limit for user ID '%s': %w", loanAmount, userID, err)
		}

		loanStatusQuery := `
		UPDATE payments
		SET status = 'completed'
		WHERE id = ?;`

		_, err = transaction.Exec(loanStatusQuery, paymentID)
		if err != nil {
			return fmt.Errorf("installment settlement failed: unable to mark loan ID '%s' as completed: %w", paymentID, err)
		}
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical engine tracking mismatch: failed to write installment block modifications to disk on final commit sequence: %w", err)
	}
	return nil
}

func (s *Store) ListInstallments(userID string) ([]models.Installment, error) {
	query := `
	SELECT id, payment_id, user_id, amount, due_date, is_paid, created_at
	FROM installments
	WHERE user_id = ?
	ORDER BY due_date ASC;`

	rows, err := s.db.Query(query, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active debt obligation schedule for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	installments := []models.Installment{}

	for rows.Next() {
		var installment models.Installment
		var isPaidInt int

		err = rows.Scan(&installment.ID, &installment.PaymentID, &installment.UserWithDebt, &installment.Amount, &installment.DueDate, &isPaidInt, &installment.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize installment row into debt schedule model structure: %w", err)
		}

		// Convert the SELite integer flag back to a Go boolean
		installment.IsPaid = isPaidInt == 1
		installments = append(installments, installment)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during installments iteration loop for user ID '%s': %w", userID, err)
	}
	return installments, nil
}

func (s *Store) ListOverdueInstallments(userID string) ([]models.Installment, error) {
	query := `
	SELECT id, payment_id, user_id, amount, due_date, is_paid, created_at
	FROM installments
	WHERE user_id = ? AND is_paid = 0 and due_date < ?
	ORDER BY due_date ASC;`

	today := time.Now().Format("2006-01-02")

	rows, err := s.db.Query(query, userID, today)
	if err != nil {
		return nil, fmt.Errorf("failed to scan overdue delinquency obligation records for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	installments := []models.Installment{}

	for rows.Next() {
		var installment models.Installment
		var isPaidInt int

		err = rows.Scan(&installment.ID, &installment.PaymentID, &installment.UserWithDebt, &installment.Amount, &installment.DueDate, &isPaidInt, &installment.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize overdue installment row into debt schedule model structure: %w", err)
		}

		// Convert the SELite integer flag back to a Go boolean
		installment.IsPaid = isPaidInt == 1
		installments = append(installments, installment)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during overdue installments iteration loop for user ID '%s': %w", userID, err)
	}
	return installments, nil
}

func (s *Store) SendFriendRequest(request *models.FriendRequest) error {
	query := `
	INSERT INTO friend_requests
	(id, sender_id, receiver_id)
	VALUES (?,?,?);`

	if request.ID == "" {
		request.ID = create_new_ID()
	}
	_, err := s.db.Exec(query, request.ID, request.SenderID, request.ReceiverID)
	if err != nil {
		return fmt.Errorf("failed to insert pending relationship record for invitation request ID '%s': %w", request.ID, err)
	}
	s.CreateNotification(&models.Notification{
		UserID:   request.ReceiverID,
		Type:     "friend_request",
		Title:    "New friend request",
		Body:     "@" + request.SenderID + " wants to be your friend",
		LinkView: "social",
	})
	return nil
}

func (s *Store) ListIncomingFriendRequests(userID string) ([]models.FriendRequest, error) {
	query := `
	SELECT id, sender_id, receiver_id, created_at
	FROM friend_requests
	WHERE receiver_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query incoming friend requests directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	// collect a list of all the requests
	requests := []models.FriendRequest{}
	for rows.Next() {
		var r models.FriendRequest

		err := rows.Scan(&r.ID, &r.SenderID, &r.ReceiverID, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse incoming connection record sequence into target friend request model: %w", err)
		}
		requests = append(requests, r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor error processing user ID '%s' incoming network request tracking loop: %w", userID, err)
	}
	return requests, nil
}

func (s *Store) ListOutgoingFriendRequests(userID string) ([]models.FriendRequest, error) {
	query := `
	SELECT id, sender_id, receiver_id, created_at
	FROM friend_requests
	WHERE sender_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outbound tracking directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	requests := []models.FriendRequest{}

	for rows.Next() {
		var r models.FriendRequest

		err = rows.Scan(&r.ID, &r.SenderID, &r.ReceiverID, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching outbound invitation row properties into model reference: %w", err)
		}
		requests = append(requests, r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor error processing user ID '%s' outgoing network request tracking loop: %w", userID, err)
	}
	return requests, nil
}

func (s *Store) AcceptFriendRequest(requestID, receiverID string) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("friend request acceptance aborted: failed to provision database transaction: %w", err)
	}
	defer transaction.Rollback()

	// Derive senderID directly from the database — never trust a client-supplied value.
	// The WHERE also confirms the caller is the actual recipient of this request.
	var senderID string
	err = transaction.QueryRow(
		`SELECT sender_id FROM friend_requests WHERE id = ? AND receiver_id = ?;`,
		requestID, receiverID,
	).Scan(&senderID)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: request ID '%s' not found or does not belong to receiver '%s': %w", requestID, receiverID, err)
	}

	// Remove the pending invitation row
	_, err = transaction.Exec(`DELETE FROM friend_requests WHERE id = ?;`, requestID)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: unable to clear invitation record ID '%s': %w", requestID, err)
	}

	// Create the bidirectional friendship
	_, err = transaction.Exec(
		`INSERT OR IGNORE INTO friends (user_id, friend_id) VALUES (?1, ?2), (?2, ?1);`,
		senderID, receiverID,
	)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: failed to generate mutual peer-to-peer mapping for user IDs '%s' and '%s': %w", senderID, receiverID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("friend request acceptance failed: database transaction failed to commit to disk: %w", err)
	}
	s.CreateNotification(&models.Notification{
		UserID:   senderID,
		Type:     "friend_accepted",
		Title:    "Friend request accepted",
		Body:     "@" + receiverID + " accepted your friend request",
		LinkView: "social",
	})
	return nil
}

func (s *Store) DeclineFriendRequest(requestID string) error {
	query := `
	DELETE FROM friend_requests
	WHERE id = ?;`

	_, err := s.db.Exec(query, requestID)
	if err != nil {
		return fmt.Errorf("failed to execute removal delete action on friend request ID '%s': %w", requestID, err)
	}
	return nil
}

func (s *Store) RemoveFriendMutual(userID, friendID string) error {
	query := `
	DELETE FROM friends
	WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?);`

	_, err := s.db.Exec(query, userID, friendID, friendID, userID)
	if err != nil {
		return fmt.Errorf("failed to sever mutual bidirectional connection map matching user IDs '%s' and '%s': %w", userID, friendID, err)
	}
	return nil
}

func (s *Store) ListFriends(userID string) ([]models.Profile, error) {
	// Look up all friends of the current user and use the friends' IDs to return their profile
	query := `
	SELECT users.id, users.name, users.email, users.phone_number, users.created_at
	FROM friends
	JOIN users ON friends.friend_id = users.id
	WHERE friends.user_id = ?
	ORDER BY users.name ASC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate network directory join mapping for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	friends := []models.Profile{}

	for rows.Next() {
		var friend models.Profile

		err := rows.Scan(&friend.ID, &friend.Name, &friend.Email, &friend.PhoneNumber, &friend.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize user profile attributes from friends table query data stream: %w", err)
		}
		friends = append(friends, friend)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected structural cursor disruption within active user friends list scanner loop for ID '%s': %w", userID, err)
	}
	return friends, nil
}

// NEW FEATURE : GROUPS

func (s *Store) CreateGroup(group *models.Group) error {
	// Generate a unique group identifier inside the store so the client never controls primary keys
	if group.ID == "" {
		group.ID = create_new_ID()
	}

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("group creation aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, roll back all pending changes
	defer transaction.Rollback()

	query := `
	INSERT INTO groups (id, name, creator_id) VALUES (?, ?, ?);`

	_, err = transaction.Exec(query, group.ID, group.Name, group.CreatorID)
	if err != nil {
		return fmt.Errorf("group creation failed: unable to write group record for group ID '%s': %w", group.ID, err)
	}

	// Immediately bind the creator to the membership table as the first member
	query = `
	INSERT INTO group_members (group_id, member_id) VALUES (?, ?);`

	_, err = transaction.Exec(query, group.ID, group.CreatorID)
	if err != nil {
		return fmt.Errorf("group creation failed: unable to register creator '%s' as initial member of group ID '%s': %w", group.CreatorID, group.ID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("group creation failed: transaction commit to disk rejected for group ID '%s': %w", group.ID, err)
	}
	return nil
}

func (s *Store) ListGroups(userID string) ([]models.Group, error) {
	query := `
	SELECT groups.id, groups.name, groups.creator_id, groups.created_at
	FROM groups
	JOIN group_members ON groups.id = group_members.group_id
	WHERE group_members.member_id = ?
	ORDER BY groups.name ASC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user group collection matrices: %w", err)
	}
	defer rows.Close()

	list := []models.Group{}

	for rows.Next() {
		var g models.Group

		err = rows.Scan(&g.ID, &g.Name, &g.CreatorID, &g.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan database group row into model properties: %w", err)
		}
		list = append(list, g)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error encountered during group record row scanning loops: %w", err)
	}
	return list, nil
}

func (s *Store) ListGroupMembers(groupID string) ([]models.Profile, error) {
	query := `
	SELECT users.id, users.name, users.email, users.phone_number, users.created_at
	FROM users
	JOIN group_members ON group_members.member_id = users.id
	WHERE group_members.group_id = ?
	ORDER BY users.name ASC;`

	rows, err := s.db.Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query member roster for group ID '%s': %w", groupID, err)
	}
	defer rows.Close()

	members := []models.Profile{}

	for rows.Next() {
		var m models.Profile

		err = rows.Scan(&m.ID, &m.Name, &m.Email, &m.PhoneNumber, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize member profile record from group roster query for group ID '%s': %w", groupID, err)
		}
		members = append(members, m)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during member roster iteration for group ID '%s': %w", groupID, err)
	}
	return members, nil
}

func (s *Store) SendGroupInvitation(groupID, senderID, receiverID string) error {
	query := `
	INSERT INTO group_invitations 
	(id, group_id, sender_id, receiver_id) 
	VALUES (?,?,?,?);`

	_, err := s.db.Exec(query, create_new_ID(), groupID, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("failed to insert group invitation from sender '%s' to receiver '%s' for group ID '%s': %w", senderID, receiverID, groupID, err)
	}
	return nil
}

func (s *Store) AcceptGroupInvitation(invitationID, callerID string) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("group invitation acceptance aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, discard all pending changes
	defer transaction.Rollback()

	// Derive the group from the database and confirm the caller is the actual receiver —
	// never trust a group ID supplied by the client
	var groupID string
	query := `
	SELECT group_id
	FROM group_invitations
	WHERE id = ? AND receiver_id = ?;`

	err = transaction.QueryRow(query, invitationID, callerID).Scan(&groupID)
	if err != nil {
		return fmt.Errorf("group invitation acceptance failed: invitation ID '%s' not found or does not belong to receiver '%s': %w", invitationID, callerID, err)
	}

	// Remove the invitation row — accepted invitations do not persist as a status change
	query = `
	DELETE FROM group_invitations
	WHERE id = ?;`

	_, err = transaction.Exec(query, invitationID)
	if err != nil {
		return fmt.Errorf("group invitation acceptance failed: unable to remove invitation record ID '%s': %w", invitationID, err)
	}

	// Register the new member in the group roster
	query = `
	INSERT OR IGNORE INTO group_members (group_id, member_id) VALUES (?, ?);`

	_, err = transaction.Exec(query, groupID, callerID)
	if err != nil {
		return fmt.Errorf("group invitation acceptance failed: unable to register user '%s' as a member of group ID '%s': %w", callerID, groupID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("group invitation acceptance failed: transaction commit to disk rejected for invitation ID '%s': %w", invitationID, err)
	}
	return nil
}

func (s *Store) DeclineGroupInvitation(invitationID, callerID string) error {
	// The WHERE clause on receiver_id ensures a user can only decline invitations sent to them
	query := `
	DELETE FROM group_invitations
	WHERE id = ? AND receiver_id = ?;`

	_, err := s.db.Exec(query, invitationID, callerID)
	if err != nil {
		return fmt.Errorf("failed to decline group invitation ID '%s' for receiver '%s': %w", invitationID, callerID, err)
	}
	return nil
}

func (s *Store) ListIncomingGroupInvitations(receiverID string) ([]models.GroupInvitation, error) {
	query := `
	SELECT id, group_id, sender_id, receiver_id, created_at
	FROM group_invitations
	WHERE receiver_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, receiverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query incoming group invitation directory for receiver ID '%s': %w", receiverID, err)
	}
	defer rows.Close()

	invitations := []models.GroupInvitation{}

	for rows.Next() {
		var inv models.GroupInvitation

		err = rows.Scan(&inv.ID, &inv.GroupID, &inv.SenderID, &inv.ReceiverID, &inv.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize incoming group invitation record into model structure: %w", err)
		}
		invitations = append(invitations, inv)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during incoming group invitation iteration for receiver ID '%s': %w", receiverID, err)
	}
	return invitations, nil
}

func (s *Store) ListOutgoingGroupInvitations(senderID string) ([]models.GroupInvitation, error) {
	query := `
	SELECT id, group_id, sender_id, receiver_id, created_at
	FROM group_invitations
	WHERE sender_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outgoing group invitation directory for sender ID '%s': %w", senderID, err)
	}
	defer rows.Close()

	invitations := []models.GroupInvitation{}

	for rows.Next() {
		var inv models.GroupInvitation

		err = rows.Scan(&inv.ID, &inv.GroupID, &inv.SenderID, &inv.ReceiverID, &inv.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize outgoing group invitation record into model structure: %w", err)
		}
		invitations = append(invitations, inv)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during outgoing group invitation iteration for sender ID '%s': %w", senderID, err)
	}
	return invitations, nil
}

func (s *Store) LeaveGroup(groupID, userID string) error {
	query := `
	DELETE FROM group_members
	WHERE group_id = ? AND member_id = ?;`

	_, err := s.db.Exec(query, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user ID '%s' from group ID '%s' membership roster: %w", userID, groupID, err)
	}
	return nil
}

func (s *Store) RemoveGroupMember(groupID, targetUserID string) error {
	query := `
	DELETE FROM group_members
	WHERE group_id = ? AND member_id = ?;`

	_, err := s.db.Exec(query, groupID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to remove target user ID '%s' from group ID '%s' membership roster: %w", targetUserID, groupID, err)
	}
	return nil
}

func (s *Store) CreateNotification(n *models.Notification) error {
	// Generate a unique notification identifier inside the store so the client never controls primary keys
	if n.ID == "" {
		n.ID = create_new_ID()
	}

	query := `
	INSERT INTO notifications 
	(id, user_id, type, title, body, link_view) 
	VALUES (?,?,?,?,?,?);`

	_, err := s.db.Exec(query, n.ID, n.UserID, n.Type, n.Title, n.Body, n.LinkView)
	if err != nil {
		return fmt.Errorf("failed to write notification record for user ID '%s': %w", n.UserID, err)
	}
	return nil
}

func (s *Store) ListNotifications(userID string) ([]models.Notification, error) {
	query := `
	SELECT id, user_id, type, title, body, link_view, is_seen, created_at
	FROM notifications
	WHERE user_id = ? AND is_seen = 0
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve unseen notification records for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	notifs := []models.Notification{}

	for rows.Next() {
		var n models.Notification
		var isSeen int

		err = rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.LinkView, &isSeen, &n.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize notification row into model structure for user ID '%s': %w", userID, err)
		}
		// Convert the SQLite integer flag back to a Go boolean
		n.IsSeen = isSeen == 1
		notifs = append(notifs, n)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during notification iteration for user ID '%s': %w", userID, err)
	}
	return notifs, nil
}

func (s *Store) MarkNotificationSeen(notifID, userID string) error {
	query := `
	UPDATE notifications
	SET is_seen = 1
	WHERE id = ? AND user_id = ?;`

	_, err := s.db.Exec(query, notifID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification ID '%s' as seen for user ID '%s': %w", notifID, userID, err)
	}
	return nil
}

func (s *Store) ClearAllNotifications(userID string) error {
	query := `
	UPDATE notifications
	SET is_seen = 1
	WHERE user_id = ?;`

	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to clear all notification records for user ID '%s': %w", userID, err)
	}
	return nil
}
