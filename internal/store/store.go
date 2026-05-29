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
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
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
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (sender_id) REFERENCES users (id),
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
	(id, name, email, phone_number, balance, credit_score, credit_limit) 
	VALUES (?, ?, ?, ?, ?, ?, ?);`

	_, err := s.db.Exec(query, user.ID, user.Name, user.Email, user.PhoneNumber, user.Balance, user.CreditScore, user.CreditLimit)
	if err != nil {
		return fmt.Errorf("failed to write profile for user ID '%s' to database: %w", user.ID, err)
	}
	return nil
}

func (s *Store) GetUser(userID string) (*models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users 
	WHERE id = ?;`
	var user models.User

	err := s.db.QueryRow(query, userID).Scan(&user.ID, &user.Name, &user.Email, &user.PhoneNumber, &user.Balance, &user.CreditScore, &user.CreditLimit, &user.CreatedAt)
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

func (s *Store) ListUsers() ([]*models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute full users table scan lookup query: %w", err)
	}
	defer rows.Close()

	var users []*models.User

	for rows.Next() {
		var u models.User

		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PhoneNumber, &u.Balance, &u.CreditScore, &u.CreditLimit, &u.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row into user data struct during list aggregation: %w", err)
		}

		users = append(users, &u)
	}
	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during users rows iteration loop: %w", err)
	}
	return users, nil
}

func (s *Store) ListProfiles() ([]*models.Profile, error) {
	query := `
	SELECT id, name, email, phone_number, created_at
	FROM users;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute global public profiles query row collection: %w", err)
	}
	defer rows.Close()

	var profiles []*models.Profile

	for rows.Next() {
		var profile models.Profile

		err := rows.Scan(&profile.ID, &profile.Name, &profile.Email, &profile.PhoneNumber, &profile.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching profile data mapping structure: %w", err)
		}

		profiles = append(profiles, &profile)
	}
	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during profiles row iteration loop: %w", err)
	}
	return profiles, nil
}

func (s *Store) Pay(payment *models.Payment) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ledger settlement aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, discard all changes
	defer transaction.Rollback()

	// Block the sender from going into negative balance
	var current_balance float64
	balance_query := `
	SELECT balance 
	FROM users 
	WHERE id = ?;`
	err = transaction.QueryRow(balance_query, payment.SenderID).Scan(&current_balance)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to verify sender funds: %w", err)
	}

	if current_balance < payment.Amount {
		return fmt.Errorf("ledger settlement rejected: insufficient liquid funds (ID: '%s' attempted to pay $%.2f but only has $%.2f)", payment.SenderID, payment.Amount, current_balance)
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

	return nil
}

func (s *Store) CreateBNPLLoan(payment *models.Payment) error {
	if payment.TotalInstallments == 0 {
		return fmt.Errorf("credit engine processing aborted: total plan financing installments cannot be evaluated at zero")
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

	// If all installments are cleared, restore the full loan amount back to the buyer's credit limit
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
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical engine tracking mismatch: failed to write installment block modifications to disk on final commit sequence: %w", err)
	}
	return nil
}

func (s *Store) SendFriendRequest(request *models.FriendRequest) error {
	query := `
	INSERT INTO friend_requests 
	(id, sender_id, receiver_id)
	VALUES (?,?,?);`

	_, err := s.db.Exec(query, request.ID, request.SenderID, request.ReceiverID)
	if err != nil {
		return fmt.Errorf("failed to insert pending relationship record for invitation request ID '%s': %w", request.ID, err)
	}
	return nil
}

func (s *Store) ListIncomingFriendRequests(userID string) ([]*models.FriendRequest, error) {
	query := `
	SELECT id, sender_id, receiver_id, created_at
	FROM friend_requests
	WHERE receiver_id = ?;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query incoming friend requests directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	// collect a list of all the requests
	var requests []*models.FriendRequest
	for rows.Next() {
		var r models.FriendRequest

		err := rows.Scan(&r.ID, &r.SenderID, &r.ReceiverID, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse incoming connection record sequence into target friend request model: %w", err)
		}
		requests = append(requests, &r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor error processing user ID '%s' incoming network request tracking loop: %w", userID, err)
	}
	return requests, nil
}

func (s *Store) ListOutgoingFriendRequests(userID string) ([]*models.FriendRequest, error) {
	query := `
	SELECT id, sender_id, receiver_id, created_at
	FROM friend_requests
	WHERE sender_id = ?;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outbound tracking directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	var requests []*models.FriendRequest

	for rows.Next() {
		var r models.FriendRequest

		err = rows.Scan(&r.ID, &r.SenderID, &r.ReceiverID, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching outbound invitation row properties into model reference: %w", err)
		}
		requests = append(requests, &r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor error processing user ID '%s' outgoing network request tracking loop: %w", userID, err)
	}
	return requests, nil
}

func (s *Store) AcceptFriendRequest(requestID, senderID, receiverID string) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("friend request acceptance aborted: failed to provision database transaction: %w", err)
	}
	defer transaction.Rollback()

	// update the friend requests table
	query := `
	DELETE FROM friend_requests
	WHERE id = ?;`

	_, err = transaction.Exec(query, requestID)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: unable to clear invitation record ID '%s': %w", requestID, err)
	}

	// update the friends table to show the new relationship
	// (for both users, ie user A is friends with user B, and user B is friends with user A)
	query = `
	INSERT OR IGNORE INTO friends
	(user_id, friend_id)
	VALUES (?1, ?2), (?2, ?1);`

	_, err = transaction.Exec(query, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: failed to generate mutual peer-to-peer mapping intersection for user IDs '%s' and '%s': %w", senderID, receiverID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("friend request acceptance failed: database transaction failed to commit to disk: %w", err)
	}
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

func (s *Store) ListFriends(userID string) ([]*models.Profile, error) {
	// Look up all friends of the current user and use the friends' IDs to return their profile
	query := `
	SELECT u.id, u.name, u.email, u.phone_number, u.created_at
	FROM friends AS fr
	JOIN users AS u ON fr.friend_id = u.id
	WHERE fr.user_id = ?;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate network directory join mapping for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	var friends []*models.Profile

	for rows.Next() {
		var friend models.Profile

		err := rows.Scan(&friend.ID, &friend.Name, &friend.Email, &friend.PhoneNumber, &friend.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize user profile attributes from friends table query data stream: %w", err)
		}
		friends = append(friends, &friend)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected structural cursor disruption within active user friends list scanner loop for ID '%s': %w", userID, err)
	}
	return friends, nil
}

func (s *Store) CreatePaymentRequest(request *models.PaymentRequest) error {
	query := `
	INSERT INTO payment_requests
	(id, requester_id, payer_id, amount, note, status)
	VALUES (?,?,?,?,?,?);`

	_, err := s.db.Exec(query, request.ID, request.RequesterID, request.PayerID, request.Amount, request.Note, request.Status)
	if err != nil {
		return fmt.Errorf("failed to push open payment demand requisition with invoice ID '%s' into table ledgers: %w", request.ID, err)
	}
	return nil
}

func (s *Store) ListIncomingPaymentRequests(userID string) ([]*models.PaymentRequest, error) {
	query := `
	SELECT id, requester_id, payer_id, amount, note, status, created_at
	FROM payment_requests
	WHERE payer_id = ? AND status = 'pending';`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up outstanding pending payables collection dashboard records for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	var requests []*models.PaymentRequest

	for rows.Next() {
		var r models.PaymentRequest

		err = rows.Scan(&r.ID, &r.RequesterID, &r.PayerID, &r.Amount, &r.Note, &r.Status, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack invoice variables into internal payment request structure array: %w", err)
		}
		requests = append(requests, &r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected an unexpected structural read failure inside payables pipeline engine iteration for user ID '%s': %w", userID, err)
	}
	return requests, nil
}

func (s *Store) ListOutgoingPaymentRequests(userID string) ([]*models.PaymentRequest, error) {
	query := `
	SELECT id, requester_id, payer_id, amount, note, status, created_at
	FROM payment_requests
	WHERE requester_id = ? AND status = 'pending';`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up outbound collectibles receivables list directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	var requests []*models.PaymentRequest

	for rows.Next() {
		var r models.PaymentRequest

		err = rows.Scan(&r.ID, &r.RequesterID, &r.PayerID, &r.Amount, &r.Note, &r.Status, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to map outbound record fields safely to target collection model fields: %w", err)
		}

		requests = append(requests, &r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected an unexpected structural read failure inside collectibles receivables pipeline iteration for user ID '%s': %w", userID, err)
	}
	return requests, nil
}

func (s *Store) UpdatePaymentRequestStatus(paymentID, new_status string) error {
	query := `
	UPDATE payment_requests
	SET status = ?
	WHERE id = ?;`

	_, err := s.db.Exec(query, new_status, paymentID)
	if err != nil {
		return fmt.Errorf("state machine error: failed to transition payment invoice request ID '%s' to state token '%s': %w", paymentID, new_status, err)
	}
	return nil
}
