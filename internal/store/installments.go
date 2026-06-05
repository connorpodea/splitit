package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/connorpodea/splitit/internal/models"
)

// CreateBNPLLoan initiates a full buy-now-pay-later loan: validates the buyer's credit limit,
// calculates the fee-adjusted repayment schedule, funds the merchant via the app treasury,
// collects the down payment, and generates the installment calendar rows.
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
	if payment.TotalAmountCents > sender.CreditLimitCents {
		return fmt.Errorf("credit engine rejected: requested loan amount of %d cents exceeds available credit limit of %d cents for buyer ID '%s'", payment.TotalAmountCents, sender.CreditLimitCents, payment.SenderID)
	}

	// Variable for the raw price before fees
	itemPriceCents := payment.TotalAmountCents

	var feeRate float64 = 0.00
	// Calculate the risk fee based on their credit health score iff they are paying over time
	if payment.TotalInstallments > 1 {
		feeRate = s.CalculateFeeRate(sender.CreditScore)
	}

	// Update the purchase amount by the fee rate
	totalDebtCents := itemPriceCents + int(feeRate*float64(itemPriceCents))

	baseAmountCents := totalDebtCents / int(payment.TotalInstallments) // integer division; remainder distributed to first installment
	remainderCents := totalDebtCents - (baseAmountCents * int(payment.TotalInstallments))

	// Step 1: Insert the master loan record to preserve historical data integrity before any fund movements
	// This anchors the loan in the payments table without touching balances directly
	payment.TotalAmountCents = totalDebtCents
	payment.AmountCents = itemPriceCents
	payment.PaymentType = "bnpl_loan_master"

	query := `
	INSERT INTO payments
	(id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = s.db.Exec(query, payment.ID, payment.SenderID, payment.ReceiverID, payment.AmountCents, payment.TotalAmountCents, payment.Note, payment.PaymentType, payment.TotalInstallments, payment.Status)
	if err != nil {
		return fmt.Errorf("loan processing aborted: failed to anchor master loan record ID '%s' into payments ledger: %w", payment.ID, err)
	}

	creditQuery := `
	UPDATE users
	SET credit_limit_cents = credit_limit_cents - ?
	WHERE id = ?;`

	_, err = s.db.Exec(creditQuery, itemPriceCents, payment.SenderID)
	if err != nil {
		return fmt.Errorf("loan processing aborted: failed to deduct %d cents from credit limit for buyer ID '%s': %w", itemPriceCents, payment.SenderID, err)
	}

	// Step 2: Pay the Merchant — the app treasury injects the full item price to the seller immediately
	fundingPayment := &models.Payment{
		ID:                fmt.Sprintf("fund_%s", payment.ID),
		SenderID:          "app_treasury",
		ReceiverID:        payment.ReceiverID,
		AmountCents:       itemPriceCents,
		TotalAmountCents:  itemPriceCents,
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
		AmountCents:       baseAmountCents + remainderCents,
		TotalAmountCents:  baseAmountCents + remainderCents,
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
		var installmentAmountCents int
		var isPaid bool
		var dueDate time.Time

		if i == 1 {
			// Installment 1 is paid upfront during s.Pay(downPayment)
			installmentAmountCents = baseAmountCents + remainderCents
			isPaid = true
			dueDate = currentTime
		} else {
			installmentAmountCents = baseAmountCents
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
		(id, payment_id, user_id, amount_cents, due_date, is_paid)
		VALUES (?,?,?,?,?,?);`
		_, err = s.db.Exec(installmentQuery, installmentID, payment.ID, payment.SenderID, installmentAmountCents, dueDate.Format("2006-01-02"), isPaidInt)
		if err != nil {
			return fmt.Errorf("failed to save generated installment row segment %d for loan ID '%s' into database schedules: %w", i, payment.ID, err)
		}
	}
	return nil
}

// CalculateFeeRate returns the interest fee multiplier for a BNPL loan based on the buyer's
// credit score. Higher scores qualify for lower rates.
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

// PayInstallment settles a single installment payment: verifies buyer funds, deducts the balance,
// credits the treasury, marks the installment paid, adjusts the credit score for on-time or late
// payment, and closes the loan if all installments are now cleared — all in one atomic transaction.
func (s *Store) PayInstallment(installmentID, paymentID, userID string, amountCents int) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("installment settlement aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, discard all changes
	defer transaction.Rollback()

	// Block the buyer from going into negative balance
	var currentBalanceCents int
	balanceQuery := `
	SELECT balance_cents
	FROM users
	WHERE id = ?;`

	err = transaction.QueryRow(balanceQuery, userID).Scan(&currentBalanceCents)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to verify buyer funds for user ID '%s': %w", userID, err)
	}
	if currentBalanceCents < amountCents {
		return fmt.Errorf("installment settlement rejected: insufficient liquid funds (ID: '%s' attempted to pay %d cents but only has %d cents)", userID, amountCents, currentBalanceCents)
	}

	// Deduct the installment amount from the buyer and credit the treasury
	deductQuery := `
	UPDATE users
	SET balance_cents = balance_cents - ?
	WHERE id = ?;`

	deductResult, err := transaction.Exec(deductQuery, amountCents, userID)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to deduct %d cents from buyer ID '%s': %w", amountCents, userID, err)
	}
	deductRows, err := deductResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read buyer deduction execution state metrics: %w", err)
	}
	if deductRows == 0 {
		return fmt.Errorf("installment settlement rejected: buyer account ID '%s' not found in user registry", userID)
	}

	creditQuery := `
	UPDATE users
	SET balance_cents = balance_cents + ?
	WHERE id = ?;`

	creditResult, err := transaction.Exec(creditQuery, amountCents, "app_treasury")
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to credit %d cents to treasury: %w", amountCents, err)
	}
	creditRows, err := creditResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read treasury credit execution state metrics: %w", err)
	}
	if creditRows == 0 {
		return fmt.Errorf("installment settlement rejected: treasury account not found in user registry")
	}

	// Mark the installment row as paid
	markPaidQuery := `
	UPDATE installments
	SET is_paid = 1
	WHERE id = ?;`

	markResult, err := transaction.Exec(markPaidQuery, installmentID)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to mark installment ID '%s' as paid: %w", installmentID, err)
	}
	markRows, err := markResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read installment paid-state execution metrics: %w", err)
	}
	if markRows == 0 {
		return fmt.Errorf("installment settlement rejected: installment ID '%s' not found in debt schedule registry", installmentID)
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

	scoreResult, err := transaction.Exec(scoreQuery, scoreDelta, userID)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to apply credit score adjustment for user ID '%s': %w", userID, err)
	}
	scoreRows, err := scoreResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read credit score adjustment execution state metrics: %w", err)
	}
	if scoreRows == 0 {
		return fmt.Errorf("installment settlement rejected: credit score update target account ID '%s' not found in user registry", userID)
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
		var loanAmountCents int
		loanAmountQuery := `
		SELECT amount_cents
		FROM payments
		WHERE id = ?;`

		err = transaction.QueryRow(loanAmountQuery, paymentID).Scan(&loanAmountCents)
		if err != nil {
			return fmt.Errorf("installment settlement failed: unable to retrieve original loan amount for credit limit restoration on loan ID '%s': %w", paymentID, err)
		}

		restoreQuery := `
		UPDATE users
		SET credit_limit_cents = credit_limit_cents + ?
		WHERE id = ?;`

		restoreResult, err := transaction.Exec(restoreQuery, loanAmountCents, userID)
		if err != nil {
			return fmt.Errorf("installment settlement failed: unable to restore %d cents to credit limit for user ID '%s': %w", loanAmountCents, userID, err)
		}
		restoreRows, err := restoreResult.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to read credit limit restoration execution state metrics: %w", err)
		}
		if restoreRows == 0 {
			return fmt.Errorf("installment settlement rejected: credit limit restoration target account ID '%s' not found in user registry", userID)
		}

		loanStatusQuery := `
		UPDATE payments
		SET status = 'completed'
		WHERE id = ?;`

		loanStatusResult, err := transaction.Exec(loanStatusQuery, paymentID)
		if err != nil {
			return fmt.Errorf("installment settlement failed: unable to mark loan ID '%s' as completed: %w", paymentID, err)
		}
		loanStatusRows, err := loanStatusResult.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to read loan completion status execution state metrics: %w", err)
		}
		if loanStatusRows == 0 {
			return fmt.Errorf("installment settlement rejected: master loan record ID '%s' not found in payments ledger", paymentID)
		}
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical engine tracking mismatch: failed to write installment block modifications to disk on final commit sequence: %w", err)
	}
	return nil
}

// ListInstallments retrieves the full installment debt schedule for a user, ordered by due date ascending.
func (s *Store) ListInstallments(userID string) ([]models.Installment, error) {
	query := `
	SELECT id, payment_id, user_id, amount_cents, due_date, is_paid, created_at
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

		err = rows.Scan(
			&installment.ID,
			&installment.PaymentID,
			&installment.UserWithDebt,
			&installment.AmountCents,
			&installment.DueDate,
			&isPaidInt,
			&installment.CreatedAt,
		)
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

// ListOverdueInstallments retrieves all unpaid installments whose due date has already passed,
// ordered by due date ascending.
func (s *Store) ListOverdueInstallments(userID string) ([]models.Installment, error) {
	query := `
	SELECT id, payment_id, user_id, amount_cents, due_date, is_paid, created_at
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

		err = rows.Scan(
			&installment.ID,
			&installment.PaymentID,
			&installment.UserWithDebt,
			&installment.AmountCents,
			&installment.DueDate,
			&isPaidInt,
			&installment.CreatedAt,
		)
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

// GetPaymentWithInstallments fetches a master payment record alongside its full installment
// schedule in a single composite response.
func (s *Store) GetPaymentWithInstallments(paymentID string) (*models.PaymentWithInstallments, error) {
	query := `
	SELECT id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status, created_at
    FROM payments
    WHERE id = ?;`

	var pwi models.PaymentWithInstallments

	err := s.db.QueryRow(query, paymentID).Scan(
		&pwi.Payment.ID,
		&pwi.Payment.SenderID,
		&pwi.Payment.ReceiverID,
		&pwi.Payment.AmountCents,
		&pwi.Payment.TotalAmountCents,
		&pwi.Payment.Note,
		&pwi.Payment.PaymentType,
		&pwi.Payment.TotalInstallments,
		&pwi.Payment.Status,
		&pwi.Payment.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment transaction record ID '%s' not found: %w", paymentID, err)
		}
		return nil, fmt.Errorf("failed to retrieve master payment parameters for ID '%s': %w", paymentID, err)
	}

	query = `
    SELECT id, payment_id, user_id, amount_cents, due_date, is_paid, created_at
    FROM installments
    WHERE payment_id = ?
    ORDER BY due_date ASC;`

	rows, err := s.db.Query(query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute installment timeline lookup for master payment ID '%s': %w", paymentID, err)
	}
	defer rows.Close()

	pwi.Installments = []models.Installment{}

	for rows.Next() {
		var i models.Installment
		var isPaidInt int

		err = rows.Scan(
			&i.ID,
			&i.PaymentID,
			&i.UserWithDebt,
			&i.AmountCents,
			&i.DueDate,
			&isPaidInt,
			&i.CreatedAt,
		)
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
