package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

// Pay executes a direct peer-to-peer payment: verifies the sender has sufficient funds,
// deducts the sender's balance, credits the receiver, records the transaction, and
// dispatches a payment notification — all within a single atomic database transaction.
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
	var currentBalanceCents int
	balanceQuery := `
	SELECT balance_cents
	FROM users
	WHERE id = ?;`
	err = transaction.QueryRow(balanceQuery, payment.SenderID).Scan(&currentBalanceCents)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to verify sender funds: %w", err)
	}

	if currentBalanceCents < payment.AmountCents {
		return fmt.Errorf("ledger settlement rejected: insufficient liquid funds (ID: '%s' attempted to pay %d cents but only has %d cents)", payment.SenderID, payment.AmountCents, currentBalanceCents)
	}

	// Update the senders balance to deduct the upfront payment
	query := `
	UPDATE users
	SET balance_cents = balance_cents - ?
	WHERE id = ?;`

	result, err := transaction.Exec(query, payment.AmountCents, payment.SenderID)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to clear balance deduction of %d cents from sender ID '%s': %w", payment.AmountCents, payment.SenderID, err)
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
	SET balance_cents = balance_cents + ?
	WHERE id = ?;`

	result, err = transaction.Exec(query, payment.TotalAmountCents, payment.ReceiverID)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: unable to credit balance allocation of %d cents to receiver ID '%s': %w", payment.TotalAmountCents, payment.ReceiverID, err)
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
	(id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = transaction.Exec(query, payment.ID, payment.SenderID, payment.ReceiverID, payment.AmountCents, payment.TotalAmountCents, payment.Note, payment.PaymentType, payment.TotalInstallments, payment.Status)
	if err != nil {
		return fmt.Errorf("ledger settlement failed: historical transaction entry creation with ID '%s' rejected by database: %w", payment.ID, err)
	}

	// Record wallet activity for real users only (skip app_treasury internal transfers)
	if payment.SenderID != "app_treasury" {
		_, err = transaction.Exec(
			`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'withdrawal');`,
			create_new_ID(), payment.SenderID, payment.AmountCents,
		)
		if err != nil {
			return fmt.Errorf("ledger settlement failed: unable to record withdrawal for sender '%s': %w", payment.SenderID, err)
		}
	}
	if payment.ReceiverID != "app_treasury" {
		_, err = transaction.Exec(
			`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'deposit');`,
			create_new_ID(), payment.ReceiverID, payment.TotalAmountCents,
		)
		if err != nil {
			return fmt.Errorf("ledger settlement failed: unable to record deposit for receiver '%s': %w", payment.ReceiverID, err)
		}
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical engine tracking mismatch: failed to write block modifications to disk on final commit sequence: %w", err)
	}

	// Dispatch the notification in a goroutine so it doesn't block the payment response
	go s.CreateNotification(&models.Notification{
		UserID:   payment.ReceiverID,
		Type:     "payment_received",
		Title:    "Payment received",
		Body:     fmt.Sprintf("@%s sent you $%.2f", payment.SenderID, float64(payment.AmountCents)/100),
		LinkView: "activity",
	})
	return nil
}

// GetPayment retrieves a single payment record by its transaction ID.
func (s *Store) GetPayment(paymentID string) (*models.Payment, error) {
	query := `
	SELECT id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status, created_at
	FROM payments
	WHERE id = ?;`

	var payment models.Payment

	err := s.db.QueryRow(query, paymentID).Scan(
		&payment.ID,
		&payment.SenderID,
		&payment.ReceiverID,
		&payment.AmountCents,
		&payment.TotalAmountCents,
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

// ListPaymentsReceived returns all inbound payment records for a user, ordered newest first.
func (s *Store) ListPaymentsReceived(userID string) ([]models.Payment, error) {
	query := `
    SELECT id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status, created_at
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
			&p.AmountCents,
			&p.TotalAmountCents,
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

// ListPaymentsSent returns all outbound payment records for a user, ordered newest first.
func (s *Store) ListPaymentsSent(userID string) ([]models.Payment, error) {
	query := `
    SELECT id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status, created_at
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
			&p.AmountCents,
			&p.TotalAmountCents,
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

// ListPaymentsByUser aggregates a unified, chronologically ordered transaction feed
// containing both inbound and outbound payments for use in an activity feed view.
func (s *Store) ListPaymentsByUser(userID string) ([]models.Payment, error) {
	query := `
    SELECT id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status, created_at
    FROM payments
    WHERE sender_id = ? OR receiver_id = ?
    ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to compile transaction history ledger for user '%s': %w", userID, err)
	}
	defer rows.Close()

	payments := []models.Payment{}

	for rows.Next() {
		var p models.Payment

		err = rows.Scan(
			&p.ID,
			&p.SenderID,
			&p.ReceiverID,
			&p.AmountCents,
			&p.TotalAmountCents,
			&p.Note,
			&p.PaymentType,
			&p.TotalInstallments,
			&p.Status,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger row segment into payment struct: %w", err)
		}

		payments = append(payments, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected stream cursor failure during transaction history compilation for user '%s': %w", userID, err)
	}

	return payments, nil
}

// ListPaymentsBetweenUsers retrieves the isolated transaction stream shared exclusively
// between two specific user identities.
func (s *Store) ListPaymentsBetweenUsers(userID, otherID string) ([]models.Payment, error) {
	query := `
    SELECT id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status, created_at
    FROM payments
    WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
    ORDER BY created_at DESC;` // Explicit grouping guarantees safety if filters are added later

	rows, err := s.db.Query(query, userID, otherID, otherID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract isolated transaction ledger between '%s' and '%s': %w", userID, otherID, err)
	}
	defer rows.Close()

	payments := []models.Payment{}

	for rows.Next() {
		var p models.Payment

		err = rows.Scan(
			&p.ID,
			&p.SenderID,
			&p.ReceiverID,
			&p.AmountCents,
			&p.TotalAmountCents,
			&p.Note,
			&p.PaymentType,
			&p.TotalInstallments,
			&p.Status,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger row segment into payment struct: %w", err)
		}

		payments = append(payments, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected stream cursor failure during shared history compilation for context pair ('%s', '%s'): %w", userID, otherID, err)
	}

	return payments, nil
}

// CreatePaymentRequest inserts a pending payment request record and notifies the payer.
func (s *Store) CreatePaymentRequest(request *models.PaymentRequest) error {
	// Generate a unique request identifier inside the store so the client never controls primary keys
	if request.ID == "" {
		request.ID = create_new_ID()
	}
	request.Status = "pending"

	query := `
	INSERT INTO payment_requests
	(id, requester_id, payer_id, amount_cents, note, status)
	VALUES (?,?,?,?,?,?);`

	_, err := s.db.Exec(query, request.ID, request.RequesterID, request.PayerID, request.AmountCents, request.Note, request.Status)
	if err != nil {
		return fmt.Errorf("failed to push open payment demand requisition with invoice ID '%s' into table ledgers: %w", request.ID, err)
	}
	// Dispatch the notification in a goroutine so it doesn't block the request response
	go s.CreateNotification(&models.Notification{
		UserID:   request.PayerID,
		Type:     "payment_request",
		Title:    "Payment request",
		Body:     fmt.Sprintf("@%s is requesting $%.2f from you", request.RequesterID, float64(request.AmountCents)/100),
		LinkView: "activity",
	})
	return nil
}

// ListIncomingPaymentRequests returns all pending payment requests where the user is the payer.
func (s *Store) ListIncomingPaymentRequests(userID string) ([]models.PaymentRequest, error) {
	query := `
	SELECT id, requester_id, payer_id, amount_cents, note, status, created_at
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
			&r.AmountCents,
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

// ListOutgoingPaymentRequests returns all pending payment requests that the user has sent out.
func (s *Store) ListOutgoingPaymentRequests(userID string) ([]models.PaymentRequest, error) {
	query := `
	SELECT id, requester_id, payer_id, amount_cents, note, status, created_at
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
			&r.AmountCents,
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

// UpdatePaymentRequestStatus transitions a payment request to a new status string (e.g. "accepted", "declined").
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

// PayToGroup fans a single payment out to every group member except the sender,
// splitting the total equally. The entire operation is a single atomic transaction —
// if any member credit fails, all balance changes are rolled back.
func (s *Store) PayToGroup(payment *models.Payment) error {
	if payment.ID == "" {
		payment.ID = create_new_ID()
	}

	members, err := s.ListGroupMembers(payment.ReceiverID)
	if err != nil {
		return fmt.Errorf("group payment failed: unable to resolve member roster for group '%s': %w", payment.ReceiverID, err)
	}

	recipients := make([]models.Profile, 0, len(members))
	for _, m := range members {
		if m.ID != payment.SenderID {
			recipients = append(recipients, m)
		}
	}
	if len(recipients) == 0 {
		return fmt.Errorf("group payment rejected: no eligible recipients (sender may be the only member)")
	}

	perPersonCents := payment.AmountCents / len(recipients)
	remainderCents := payment.AmountCents % len(recipients)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("group payment failed: unable to open transaction context: %w", err)
	}
	defer tx.Rollback()

	var senderBalance int
	if err = tx.QueryRow(`SELECT balance_cents FROM users WHERE id = ?;`, payment.SenderID).Scan(&senderBalance); err != nil {
		return fmt.Errorf("group payment failed: unable to verify sender funds for '%s': %w", payment.SenderID, err)
	}
	if senderBalance < payment.AmountCents {
		return fmt.Errorf("group payment rejected: insufficient funds (balance: %d cents, total required: %d cents)", senderBalance, payment.AmountCents)
	}

	if _, err = tx.Exec(`UPDATE users SET balance_cents = balance_cents - ? WHERE id = ?;`, payment.AmountCents, payment.SenderID); err != nil {
		return fmt.Errorf("group payment failed: unable to deduct %d cents from sender '%s': %w", payment.AmountCents, payment.SenderID, err)
	}
	if _, err = tx.Exec(`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'withdrawal');`, create_new_ID(), payment.SenderID, payment.AmountCents); err != nil {
		return fmt.Errorf("group payment failed: unable to record sender withdrawal for '%s': %w", payment.SenderID, err)
	}

	for i, recipient := range recipients {
		amount := perPersonCents
		if i == 0 {
			amount += remainderCents
		}

		result, err := tx.Exec(`UPDATE users SET balance_cents = balance_cents + ? WHERE id = ?;`, amount, recipient.ID)
		if err != nil {
			return fmt.Errorf("group payment failed: unable to credit member '%s': %w", recipient.ID, err)
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("group payment failed: member account '%s' not found in user registry", recipient.ID)
		}

		splitID := fmt.Sprintf("split_%s_%d", payment.ID, i)
		if _, err = tx.Exec(
			`INSERT INTO payments (id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status) VALUES (?, ?, ?, ?, ?, ?, 'group_split', 1, 'completed');`,
			splitID, payment.SenderID, recipient.ID, amount, amount, payment.Note,
		); err != nil {
			return fmt.Errorf("group payment failed: unable to record split payment record for member '%s': %w", recipient.ID, err)
		}

		if _, err = tx.Exec(`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'deposit');`, create_new_ID(), recipient.ID, amount); err != nil {
			return fmt.Errorf("group payment failed: unable to record deposit for member '%s': %w", recipient.ID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("group payment failed: transaction commit rejected: %w", err)
	}

	for _, recipient := range recipients {
		r := recipient
		go s.CreateNotification(&models.Notification{
			UserID:   r.ID,
			Type:     "payment_received",
			Title:    "Group payment received",
			Body:     fmt.Sprintf("@%s paid you as part of a group split", payment.SenderID),
			LinkView: "activity",
		})
	}

	return nil
}

// RequestFromGroup creates one payment request per group member (excluding the requester),
// splitting the total amount equally in a single atomic transaction.
func (s *Store) RequestFromGroup(request *models.PaymentRequest) error {
	// Fully consume the member cursor before opening a write transaction to avoid
	// SQLite "cannot start a transaction within a transaction" on single-connection mode.
	members, err := s.ListGroupMembers(request.PayerID)
	if err != nil {
		return fmt.Errorf("group request failed: unable to resolve member roster for group '%s': %w", request.PayerID, err)
	}

	recipients := make([]models.Profile, 0, len(members))
	for _, m := range members {
		if m.ID != request.RequesterID {
			recipients = append(recipients, m)
		}
	}
	if len(recipients) == 0 {
		return fmt.Errorf("group request rejected: no eligible payers (requester may be the only member)")
	}

	perPersonCents := request.AmountCents / len(recipients)
	remainderCents := request.AmountCents % len(recipients)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("group request failed: unable to open transaction context: %w", err)
	}
	defer tx.Rollback()

	type inserted struct {
		payerID     string
		amountCents int
	}
	rows := make([]inserted, 0, len(recipients))

	for i, recipient := range recipients {
		amount := perPersonCents
		if i == 0 {
			amount += remainderCents
		}
		id := create_new_ID()
		if _, err = tx.Exec(
			`INSERT INTO payment_requests (id, requester_id, payer_id, amount_cents, note, status) VALUES (?, ?, ?, ?, ?, 'pending');`,
			id, request.RequesterID, recipient.ID, amount, request.Note,
		); err != nil {
			return fmt.Errorf("group request failed: unable to insert split request for member '%s': %w", recipient.ID, err)
		}
		rows = append(rows, inserted{payerID: recipient.ID, amountCents: amount})
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("group request failed: transaction commit rejected: %w", err)
	}

	for _, r := range rows {
		r := r
		go s.CreateNotification(&models.Notification{
			UserID:   r.payerID,
			Type:     "payment_request",
			Title:    "Payment request",
			Body:     fmt.Sprintf("@%s is requesting $%.2f from you", request.RequesterID, float64(r.amountCents)/100),
			LinkView: "activity",
		})
	}

	return nil
}

// FulfillPaymentRequest atomically pays a pending payment request: transfers funds from
// the payer to the requester, records the payment and wallet transactions, and marks the
// request as completed — all in a single flat transaction (no nested calls to Pay).
func (s *Store) FulfillPaymentRequest(requestID, payerID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("request fulfillment aborted: failed to open transaction context: %w", err)
	}
	defer tx.Rollback()

	var requesterID string
	var amountCents int
	var status string
	err = tx.QueryRow(
		`SELECT requester_id, amount_cents, status FROM payment_requests WHERE id = ? AND payer_id = ?;`,
		requestID, payerID,
	).Scan(&requesterID, &amountCents, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("request fulfillment rejected: request ID '%s' not found or does not belong to payer '%s'", requestID, payerID)
		}
		return fmt.Errorf("request fulfillment failed: unable to validate payment request: %w", err)
	}
	if status != "pending" {
		return fmt.Errorf("request fulfillment rejected: request '%s' is not pending (current status: %s)", requestID, status)
	}

	var balance int
	if err = tx.QueryRow(`SELECT balance_cents FROM users WHERE id = ?;`, payerID).Scan(&balance); err != nil {
		return fmt.Errorf("request fulfillment failed: unable to verify payer balance: %w", err)
	}
	if balance < amountCents {
		return fmt.Errorf("request fulfillment rejected: insufficient liquid funds (balance: %d cents, required: %d cents)", balance, amountCents)
	}

	if _, err = tx.Exec(`UPDATE users SET balance_cents = balance_cents - ? WHERE id = ?;`, amountCents, payerID); err != nil {
		return fmt.Errorf("request fulfillment failed: unable to deduct balance from payer '%s': %w", payerID, err)
	}
	if _, err = tx.Exec(`UPDATE users SET balance_cents = balance_cents + ? WHERE id = ?;`, amountCents, requesterID); err != nil {
		return fmt.Errorf("request fulfillment failed: unable to credit balance to requester '%s': %w", requesterID, err)
	}

	paymentID := create_new_ID()
	if _, err = tx.Exec(
		`INSERT INTO payments (id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status) VALUES (?, ?, ?, ?, ?, '', 'direct', 1, 'completed');`,
		paymentID, payerID, requesterID, amountCents, amountCents,
	); err != nil {
		return fmt.Errorf("request fulfillment failed: unable to record payment: %w", err)
	}

	if _, err = tx.Exec(
		`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'withdrawal');`,
		create_new_ID(), payerID, amountCents,
	); err != nil {
		return fmt.Errorf("request fulfillment failed: unable to record withdrawal for payer '%s': %w", payerID, err)
	}
	if _, err = tx.Exec(
		`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'deposit');`,
		create_new_ID(), requesterID, amountCents,
	); err != nil {
		return fmt.Errorf("request fulfillment failed: unable to record deposit for requester '%s': %w", requesterID, err)
	}

	if _, err = tx.Exec(`UPDATE payment_requests SET status = 'completed' WHERE id = ?;`, requestID); err != nil {
		return fmt.Errorf("request fulfillment failed: unable to mark request '%s' as completed: %w", requestID, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("request fulfillment failed: transaction commit rejected: %w", err)
	}

	go s.CreateNotification(&models.Notification{
		UserID:   requesterID,
		Type:     "payment_received",
		Title:    "Payment received",
		Body:     fmt.Sprintf("@%s paid your $%.2f request", payerID, float64(amountCents)/100),
		LinkView: "activity",
	})

	return nil
}

// GetUnifiedActivity returns a single chronologically ordered feed combining payments
// sent, payments received, requests sent, requests received, and pending BNPL installments.
func (s *Store) GetUnifiedActivity(userID string, limit int) ([]models.ActivityItem, error) {
	if limit <= 0 {
		limit = 50
	}

	// Every branch must emit 10 columns in the same order:
	//   kind, id, peer_id, peer_name, peer_color, amount_cents, payment_id, note, status, created_at
	// peer_color is read live from users at query time so any profile update is immediately
	// reflected in all historical entries on the next fetch — no caching layer required.
	// payment_id is only meaningful for installment_due rows; all other branches emit ''.
	query := `
	SELECT kind, id, peer_id, peer_name, peer_color, amount_cents, payment_id, note, status, created_at FROM (
	  SELECT 'payment_sent' AS kind, p.id, p.receiver_id AS peer_id,
	         COALESCE(NULLIF(u.name,''), p.receiver_id) AS peer_name,
	         COALESCE(NULLIF(u.profile_color,''), '#4ade80') AS peer_color,
	         p.amount_cents, '' AS payment_id, COALESCE(p.note,'') AS note, p.status, p.created_at
	  FROM payments p LEFT JOIN users u ON u.id = p.receiver_id
	  WHERE p.sender_id = ? AND p.payment_type NOT IN ('bnpl','group_split')

	  UNION ALL

	  SELECT 'payment_received', p.id, p.sender_id,
	         COALESCE(NULLIF(u.name,''), p.sender_id),
	         COALESCE(NULLIF(u.profile_color,''), '#4ade80'),
	         p.amount_cents, '', COALESCE(p.note,''), p.status, p.created_at
	  FROM payments p LEFT JOIN users u ON u.id = p.sender_id
	  WHERE p.receiver_id = ? AND p.payment_type NOT IN ('bnpl','group_split')

	  UNION ALL

	  SELECT 'request_sent', pr.id, pr.payer_id,
	         COALESCE(NULLIF(u.name,''), pr.payer_id),
	         COALESCE(NULLIF(u.profile_color,''), '#4ade80'),
	         pr.amount_cents, '', COALESCE(pr.note,''), pr.status, pr.created_at
	  FROM payment_requests pr LEFT JOIN users u ON u.id = pr.payer_id
	  WHERE pr.requester_id = ?

	  UNION ALL

	  SELECT 'request_received', pr.id, pr.requester_id,
	         COALESCE(NULLIF(u.name,''), pr.requester_id),
	         COALESCE(NULLIF(u.profile_color,''), '#4ade80'),
	         pr.amount_cents, '', COALESCE(pr.note,''), pr.status, pr.created_at
	  FROM payment_requests pr LEFT JOIN users u ON u.id = pr.requester_id
	  WHERE pr.payer_id = ?

	  UNION ALL

	  SELECT 'installment_due', i.id, p.receiver_id,
	         COALESCE(NULLIF(u.name,''), p.receiver_id),
	         COALESCE(NULLIF(u.profile_color,''), '#4ade80'),
	         i.amount_cents, p.id AS payment_id, COALESCE(p.note,''), 'pending', i.due_date
	  FROM installments i
	  JOIN payments p ON p.id = i.payment_id
	  LEFT JOIN users u ON u.id = p.receiver_id
	  WHERE i.user_id = ? AND i.is_paid = 0
	)
	ORDER BY created_at DESC
	LIMIT ?;`

	rows, err := s.db.Query(query, userID, userID, userID, userID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute unified activity query for user '%s': %w", userID, err)
	}
	defer rows.Close()

	items := []models.ActivityItem{}
	for rows.Next() {
		var item models.ActivityItem
		if err = rows.Scan(
			&item.Kind, &item.ID, &item.PeerID, &item.PeerName, &item.PeerColor,
			&item.AmountCents, &item.PaymentID, &item.Note, &item.Status, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan activity item: %w", err)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("cursor error during unified activity fetch for user '%s': %w", userID, err)
	}

	return items, nil
}
