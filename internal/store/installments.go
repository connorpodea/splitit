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
// Every write operation happens inside a single atomic transaction — a failure at any step
// rolls back all prior writes so the database is never left in a partially-created state.
func (s *Store) CreateBNPLLoan(payment *models.Payment) error {
	if payment.TotalInstallments == 0 {
		return fmt.Errorf("credit engine processing aborted: total plan financing installments cannot be evaluated at zero")
	}
	if payment.ID == "" {
		payment.ID = newID()
	}

	// Pre-flight reads executed outside the transaction so the single connection
	// is not held while we decide whether to proceed.
	sender, err := s.GetUser(payment.SenderID)
	if err != nil {
		return fmt.Errorf("credit engine evaluation rejected: failed to query credit profile for buyer ID '%s': %w", payment.SenderID, err)
	}
	if payment.TotalAmountCents > sender.CreditLimitCents {
		return fmt.Errorf("credit engine rejected: requested loan amount of %d cents exceeds available credit limit of %d cents for buyer ID '%s'", payment.TotalAmountCents, sender.CreditLimitCents, payment.SenderID)
	}

	itemPriceCents := payment.TotalAmountCents

	var feeRate float64
	if payment.TotalInstallments > 1 {
		feeRate = s.CalculateFeeRate(sender.CreditScore)
	}

	totalDebtCents := itemPriceCents + int(feeRate*float64(itemPriceCents))
	baseAmountCents := totalDebtCents / int(payment.TotalInstallments)
	remainderCents := totalDebtCents - (baseAmountCents * int(payment.TotalInstallments))
	firstInstallmentCents := baseAmountCents + remainderCents

	payment.TotalAmountCents = totalDebtCents
	payment.AmountCents = itemPriceCents
	payment.PaymentType = "bnpl_loan_master"

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("loan processing aborted: failed to open transaction context: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Anchor the master loan record.
	_, err = tx.Exec(
		`INSERT INTO payments (id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		payment.ID, payment.SenderID, payment.ReceiverID, payment.AmountCents, payment.TotalAmountCents, payment.Note, payment.PaymentType, payment.TotalInstallments, payment.Status,
	)
	if err != nil {
		return fmt.Errorf("loan processing aborted: failed to anchor master loan record ID '%s': %w", payment.ID, err)
	}

	// Step 2: Deduct the item price from the buyer's credit limit.
	if _, err = tx.Exec(`UPDATE users SET credit_limit_cents = credit_limit_cents - ? WHERE id = ?;`, itemPriceCents, payment.SenderID); err != nil {
		return fmt.Errorf("loan processing aborted: failed to deduct %d cents from credit limit for buyer ID '%s': %w", itemPriceCents, payment.SenderID, err)
	}

	// Step 3: Treasury funds the merchant — full item price, immediate.
	if err = s.transfer(tx, &models.Payment{
		ID:                fmt.Sprintf("fund_%s", payment.ID),
		SenderID:          treasuryID,
		ReceiverID:        payment.ReceiverID,
		AmountCents:       itemPriceCents,
		TotalAmountCents:  itemPriceCents,
		PaymentType:       "treasury_funding",
		TotalInstallments: 1,
		Status:            "completed",
		Note:              fmt.Sprintf("Treasury funded purchase for payment %s", payment.ID),
	}); err != nil {
		return fmt.Errorf("loan processing aborted: treasury-to-merchant funding failed: %w", err)
	}

	// Step 4: Collect the down payment — first installment from buyer to treasury.
	if err = s.transfer(tx, &models.Payment{
		ID:                fmt.Sprintf("down_%s", payment.ID),
		SenderID:          payment.SenderID,
		ReceiverID:        treasuryID,
		AmountCents:       firstInstallmentCents,
		TotalAmountCents:  firstInstallmentCents,
		PaymentType:       "installment",
		TotalInstallments: 1,
		Status:            "completed",
		Note:              fmt.Sprintf("Down payment for loan %s", payment.ID),
	}); err != nil {
		return fmt.Errorf("loan processing aborted: down payment collection failed for buyer ID '%s': %w", payment.SenderID, err)
	}

	// Step 5: Generate the bi-weekly installment schedule.
	currentTime := time.Now()
	for i := uint8(1); i <= payment.TotalInstallments; i++ {
		amount := baseAmountCents
		isPaidInt := 0
		dueDate := currentTime.AddDate(0, 0, int(i-1)*14)
		if i == 1 {
			amount = firstInstallmentCents
			isPaidInt = 1
			dueDate = currentTime
		}
		installmentID := fmt.Sprintf("inst_%s_%d", payment.ID, i)
		if _, err = tx.Exec(
			`INSERT INTO installments (id, payment_id, user_id, amount_cents, due_date, is_paid) VALUES (?,?,?,?,?,?);`,
			installmentID, payment.ID, payment.SenderID, amount, dueDate.Format("2006-01-02"), isPaidInt,
		); err != nil {
			return fmt.Errorf("loan processing aborted: failed to create installment %d for loan ID '%s': %w", i, payment.ID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("loan processing aborted: failed to commit transaction: %w", err)
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

	creditResult, err := transaction.Exec(creditQuery, amountCents, treasuryID)
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

	// Mark the installment row as paid — AND user_id = ? ensures only the actual
	// borrower can settle their own installment; third-party callers are rejected here.
	markPaidQuery := `
	UPDATE installments
	SET is_paid = 1
	WHERE id = ? AND user_id = ?;`

	markResult, err := transaction.Exec(markPaidQuery, installmentID, userID)
	if err != nil {
		return fmt.Errorf("installment settlement failed: unable to mark installment ID '%s' as paid: %w", installmentID, err)
	}
	markRows, err := markResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read installment paid-state execution metrics: %w", err)
	}
	if markRows == 0 {
		return fmt.Errorf("installment settlement rejected: installment ID '%s' not found or does not belong to user '%s'", installmentID, userID)
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

// ListInstallmentDetailsForUser retrieves the full installment schedule for a user with live
// counterparty name, profile color, and note resolved via JOIN at query time.
func (s *Store) ListInstallmentDetailsForUser(userID string) ([]models.InstallmentDetail, error) {
	query := `
	SELECT i.id, i.payment_id, i.user_id, i.amount_cents, i.due_date, i.is_paid, i.created_at,
	       p.receiver_id,
	       COALESCE(NULLIF(u.name,''), p.receiver_id),
	       COALESCE(NULLIF(u.profile_color,''), '#4ade80'),
	       COALESCE(p.note, '')
	FROM installments i
	JOIN payments p ON p.id = i.payment_id
	LEFT JOIN users u ON u.id = p.receiver_id
	WHERE i.user_id = ?
	ORDER BY i.due_date ASC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve installment details for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	details := []models.InstallmentDetail{}
	for rows.Next() {
		var d models.InstallmentDetail
		var isPaidInt int
		if err = rows.Scan(
			&d.ID, &d.PaymentID, &d.UserWithDebt, &d.AmountCents, &d.DueDate, &isPaidInt, &d.CreatedAt,
			&d.PeerID, &d.PeerName, &d.PeerColor, &d.Note,
		); err != nil {
			return nil, fmt.Errorf("failed to scan installment detail row: %w", err)
		}
		d.IsPaid = isPaidInt == 1
		details = append(details, d)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("cursor error during installment details iteration for user ID '%s': %w", userID, err)
	}
	return details, nil
}

// ListOverdueInstallmentDetailsForUser retrieves unpaid installments past their due date with
// live counterparty name, profile color, and note resolved via JOIN at query time.
func (s *Store) ListOverdueInstallmentDetailsForUser(userID string) ([]models.InstallmentDetail, error) {
	query := `
	SELECT i.id, i.payment_id, i.user_id, i.amount_cents, i.due_date, i.is_paid, i.created_at,
	       p.receiver_id,
	       COALESCE(NULLIF(u.name,''), p.receiver_id),
	       COALESCE(NULLIF(u.profile_color,''), '#4ade80'),
	       COALESCE(p.note, '')
	FROM installments i
	JOIN payments p ON p.id = i.payment_id
	LEFT JOIN users u ON u.id = p.receiver_id
	WHERE i.user_id = ? AND i.is_paid = 0 AND i.due_date < ?
	ORDER BY i.due_date ASC;`

	today := time.Now().Format("2006-01-02")
	rows, err := s.db.Query(query, userID, today)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve overdue installment details for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	details := []models.InstallmentDetail{}
	for rows.Next() {
		var d models.InstallmentDetail
		var isPaidInt int
		if err = rows.Scan(
			&d.ID, &d.PaymentID, &d.UserWithDebt, &d.AmountCents, &d.DueDate, &isPaidInt, &d.CreatedAt,
			&d.PeerID, &d.PeerName, &d.PeerColor, &d.Note,
		); err != nil {
			return nil, fmt.Errorf("failed to scan overdue installment detail row: %w", err)
		}
		d.IsPaid = isPaidInt == 1
		details = append(details, d)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("cursor error during overdue installment details iteration for user ID '%s': %w", userID, err)
	}
	return details, nil
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

// ApplyMonthlyOverduePenalties checks for past-due unpaid installments, reduces the user's credit score,
// and records the penalty events in the credit score log history.
func (s *Store) ApplyMonthlyOverduePenalties() error {
	today := time.Now().Format("2006-01-02")

	// Collect the distinct set of users who have new overdue installments not yet penalized.
	userQuery := `
	SELECT DISTINCT user_id
	FROM installments
	WHERE is_paid = 0 AND due_date < ? AND penalty_applied = 0;`

	userRows, err := s.db.Query(userQuery, today)
	if err != nil {
		return fmt.Errorf("failed to scan delinquent user population for overdue penalty cycle: %w", err)
	}
	defer userRows.Close()

	var affectedUserIDs []string
	for userRows.Next() {
		var uid string
		if err = userRows.Scan(&uid); err != nil {
			return fmt.Errorf("failed to scan delinquent user ID row: %w", err)
		}
		affectedUserIDs = append(affectedUserIDs, uid)
	}
	if err = userRows.Err(); err != nil {
		return fmt.Errorf("detected stream cursor failure during delinquent user population scan: %w", err)
	}

	for _, userID := range affectedUserIDs {
		if err := s.applyOverduePenalty(userID, today); err != nil {
			return err
		}
	}
	return nil
}

// applyOverduePenalty applies the monthly credit score deduction and penalty markers for a
// single user in an isolated transaction. Extracted from the loop in ApplyMonthlyOverduePenalties
// so that defer tx.Rollback() fires at the end of each iteration, not at function return.
func (s *Store) applyOverduePenalty(userID, today string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction context for overdue penalty on user '%s': %w", userID, err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`UPDATE users SET credit_score = MAX(0, credit_score - ?) WHERE id = ?;`, overdueScorePenalty, userID); err != nil {
		return fmt.Errorf("failed to apply overdue credit score penalty for user '%s': %w", userID, err)
	}

	var newScore int
	if err = tx.QueryRow(`SELECT credit_score FROM users WHERE id = ?`, userID).Scan(&newScore); err != nil {
		return fmt.Errorf("failed to read post-penalty credit score for user '%s': %w", userID, err)
	}

	if _, err = tx.Exec(`INSERT INTO credit_score_log (id, user_id, score, delta) VALUES (?, ?, ?, ?);`, newID(), userID, newScore, -overdueScorePenalty); err != nil {
		return fmt.Errorf("failed to write overdue penalty audit event to credit score log for user '%s': %w", userID, err)
	}

	if _, err = tx.Exec(`UPDATE installments SET penalty_applied = 1 WHERE user_id = ? AND is_paid = 0 AND due_date < ? AND penalty_applied = 0;`, userID, today); err != nil {
		return fmt.Errorf("failed to mark overdue installments as penalized for user '%s': %w", userID, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit overdue penalty transaction for user '%s': %w", userID, err)
	}
	return nil
}

// GetInstallmentSummary pulls combined installment portfolio metrics — total settled,
// total outstanding, and total overdue amounts — in a single query execution.
func (s *Store) GetInstallmentSummary(userID string) (*models.InstallmentSummary, error) {
	today := time.Now().Format("2006-01-02")

	query := `
	SELECT
		COALESCE(SUM(CASE WHEN is_paid = 1 THEN amount_cents ELSE 0 END), 0) AS total_settled_cents,
		COALESCE(SUM(CASE WHEN is_paid = 0 AND due_date >= ? THEN amount_cents ELSE 0 END), 0) AS total_outstanding_cents,
		COALESCE(SUM(CASE WHEN is_paid = 0 AND due_date < ? THEN amount_cents ELSE 0 END), 0) AS total_overdue_cents
	FROM installments
	WHERE user_id = ?;`

	var summary models.InstallmentSummary
	err := s.db.QueryRow(query, today, today, userID).Scan(
		&summary.TotalSettledCents,
		&summary.TotalOutstandingCents,
		&summary.TotalOverdueCents,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate installment portfolio metrics for user '%s': %w", userID, err)
	}

	return &summary, nil
}
