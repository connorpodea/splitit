package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

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

	result, err := transaction.Exec(query, payment.Amount, payment.SenderID)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to clear balance deduction of $%.2f from sender ID '%s': %w", payment.Amount, payment.SenderID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read sender deduction execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("ledger settlement rejected: sender account ID '%s' not found in user registry", payment.SenderID)
	}

	// Update receivers balance to receive the total payment
	query = `
	UPDATE users
	SET balance = balance + ?
	WHERE id = ?;`

	result, err = transaction.Exec(query, payment.TotalAmount, payment.ReceiverID)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to credit balance allocation of $%.2f to receiver ID '%s': %w", payment.TotalAmount, payment.ReceiverID, err)
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read receiver credit execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("ledger settlement rejected: receiver account ID '%s' not found in user registry", payment.ReceiverID)
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

	// Dispatch the notification in a goroutine so it doesn't block the payment response
	go s.CreateNotification(&models.Notification{
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

	err := s.db.QueryRow(query, paymentID).Scan(
		&payment.ID,
		&payment.SenderID,
		&payment.ReceiverID,
		&payment.Amount,
		&payment.TotalAmount,
		&payment.Note,
		&payment.PaymentType,
		&payment.TotalInstallments,
		&payment.Status,
		&payment.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment record not found for transaction ID '%s': %w", paymentID, err)
		}
		return nil, fmt.Errorf("failed to retrieve payment record for transaction ID '%s': %w", paymentID, err)
	}
	return &payment, nil
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

		err = rows.Scan(
			&p.ID,
			&p.SenderID,
			&p.ReceiverID,
			&p.Amount,
			&p.TotalAmount,
			&p.Note,
			&p.PaymentType,
			&p.TotalInstallments,
			&p.Status,
			&p.CreatedAt,
		)
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

		err = rows.Scan(
			&p.ID,
			&p.SenderID,
			&p.ReceiverID,
			&p.Amount,
			&p.TotalAmount,
			&p.Note,
			&p.PaymentType,
			&p.TotalInstallments,
			&p.Status,
			&p.CreatedAt,
		)
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
	// Dispatch the notification in a goroutine so it doesn't block the request response
	go s.CreateNotification(&models.Notification{
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

		err = rows.Scan(
			&r.ID,
			&r.RequesterID,
			&r.PayerID,
			&r.Amount,
			&r.Note,
			&r.Status,
			&r.CreatedAt,
		)
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

		err = rows.Scan(
			&r.ID,
			&r.RequesterID,
			&r.PayerID,
			&r.Amount,
			&r.Note,
			&r.Status,
			&r.CreatedAt,
		)
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

	result, err := s.db.Exec(query, newStatus, paymentID)
	if err != nil {
		return fmt.Errorf("state machine error: failed to transition payment invoice request ID '%s' to state token '%s': %w", paymentID, newStatus, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read payment request status transition execution metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("state machine rejected: payment invoice request ID '%s' not found in ledger registry", paymentID)
	}

	return nil
}
