package store

import (
	"fmt"
	"time"

	"github.com/connorpodea/splitit/internal/models"
)

const overdueScorePenalty = 10

// GetSpendingTotals computes total sent, total received, and net cash flow
// for a user across a caller-specified time window.
func (s *Store) GetSpendingTotals(userID string, since time.Time) (*models.SpendingTotals, error) {
	// Convert Go time object to standard SQLite text format (YYYY-MM-DD)
	sinceStr := since.Format("2006-01-02")

	// Single-pass query: Sums up 'sent' vs 'received' amounts using conditional CASE statements.
	// COALESCE wraps the sums to guarantee a 0 fallback instead of a database NULL.
	query := `
    SELECT
    COALESCE(SUM(CASE WHEN sender_id = ? THEN amount_cents ELSE 0 END), 0) AS total_sent_cents,
    COALESCE(SUM(CASE WHEN receiver_id = ? THEN amount_cents ELSE 0 END), 0) AS total_received_cents
    FROM payments
    WHERE (sender_id = ? OR receiver_id = ?) AND created_at >= ?;`

	var totalSentCents int
	var totalReceivedCents int

	err := s.db.QueryRow(query, userID, userID, userID, userID, sinceStr).Scan(&totalSentCents, &totalReceivedCents)
	if err != nil {
		return nil, fmt.Errorf("failed to execute financial volume aggregation scenario for user '%s': %w", userID, err)
	}

	// Compute net cash flow
	netCents := totalReceivedCents - totalSentCents

	// Instantiate the return struct
	spendingTotals := models.SpendingTotals{
		TotalSentCents:     totalSentCents,
		TotalReceivedCents: totalReceivedCents,
		NetCents:           netCents,
	}

	return &spendingTotals, nil
}

// Deposit credits the specified amount to a user's available cash balance.
func (s *Store) Deposit(userID string, amountCents int) error {
	query := `
	UPDATE users
	SET balance_cents = balance_cents + ?
	WHERE id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction session context for deposit: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.Exec(query, amountCents, userID)
	if err != nil {
		return fmt.Errorf("failed to clear balance deposit allocation of %d cents for user ID '%s': %w", amountCents, userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read deposit execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("deposit rejected: user account ID '%s' not found in user registry", userID)
	}

	_, err = transaction.Exec(
		`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'deposit');`,
		create_new_ID(), userID, amountCents,
	)
	if err != nil {
		return fmt.Errorf("deposit failed: unable to record wallet transaction for user '%s': %w", userID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical ledger engine mismatch: failed to write deposit modifications to disk on final commit sequence: %w", err)
	}

	return nil
}

// Withdraw debits the specified amount from a user's cash balance after liquid funds verification.
func (s *Store) Withdraw(userID string, amountCents int) error {
	query := `
	UPDATE users
	SET balance_cents = balance_cents - ?
	WHERE id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction session context for withdrawal: %w", err)
	}
	defer transaction.Rollback()

	var currentBalance int
	if err = transaction.QueryRow(`SELECT balance_cents FROM users WHERE id = ?;`, userID).Scan(&currentBalance); err != nil {
		return fmt.Errorf("withdrawal rejected: unable to verify balance for user '%s': %w", userID, err)
	}
	if currentBalance < amountCents {
		return fmt.Errorf("withdrawal rejected: insufficient funds — balance is %d cents, requested %d cents", currentBalance, amountCents)
	}

	result, err := transaction.Exec(query, amountCents, userID)
	if err != nil {
		return fmt.Errorf("failed to clear balance withdraw allocation of %d cents for user ID '%s': %w", amountCents, userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read withdrawal execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("withdrawal rejected: user account ID '%s' not found in user registry", userID)
	}

	_, err = transaction.Exec(
		`INSERT INTO wallet_transactions (id, user_id, amount_cents, transaction_type) VALUES (?, ?, ?, 'withdrawal');`,
		create_new_ID(), userID, amountCents,
	)
	if err != nil {
		return fmt.Errorf("withdrawal failed: unable to record wallet transaction for user '%s': %w", userID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical ledger engine mismatch: failed to write withdraw modifications to disk on final commit sequence: %w", err)
	}

	return nil
}

// ListWalletTransactions retrieves the absolute balance mutation history (deposits and withdrawals)
// for a user, separate from the peer-to-peer payments ledger.
func (s *Store) ListWalletTransactions(userID string) ([]models.WalletTransaction, error) {
	query := `
    SELECT id, user_id, amount_cents, transaction_type, created_at
    FROM wallet_transactions
    WHERE user_id = ?
    ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract wallet transaction ledger payload for user '%s': %w", userID, err)
	}
	defer rows.Close()

	transactions := []models.WalletTransaction{}

	for rows.Next() {
		var t models.WalletTransaction

		err = rows.Scan(
			&t.ID,
			&t.UserID,
			&t.AmountCents,
			&t.TransactionType,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet transaction segment row into data struct: %w", err)
		}

		transactions = append(transactions, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected stream cursor exception during wallet history compilation for user '%s': %w", userID, err)
	}

	return transactions, nil
}

// GetMonthlySpendingSummary calculates absolute outflow, inflow, and active BNPL charge totals
// for a specific calendar billing month.
func (s *Store) GetMonthlySpendingSummary(userID string, year int, month time.Month) (*models.MonthlySummary, error) {
	startDate := fmt.Sprintf("%04d-%02d-01", year, int(month))
	endDate := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	flowQuery := `
	SELECT
		COALESCE(SUM(CASE WHEN sender_id = ? THEN amount_cents ELSE 0 END), 0) AS total_out_cents,
		COALESCE(SUM(CASE WHEN receiver_id = ? THEN amount_cents ELSE 0 END), 0) AS total_in_cents
	FROM payments
	WHERE (sender_id = ? OR receiver_id = ?) AND created_at >= ? AND created_at < ?;`

	var totalOutCents, totalInCents int
	err := s.db.QueryRow(flowQuery, userID, userID, userID, userID, startDate, endDate).
		Scan(&totalOutCents, &totalInCents)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate payment flow totals for user '%s' in %04d-%02d: %w", userID, year, int(month), err)
	}

	bnplQuery := `
	SELECT COALESCE(SUM(amount_cents), 0)
	FROM installments
	WHERE user_id = ? AND due_date >= ? AND due_date < ?;`

	var activeBNPLCents int
	err = s.db.QueryRow(bnplQuery, userID, startDate, endDate).Scan(&activeBNPLCents)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate active BNPL installment charges for user '%s' in %04d-%02d: %w", userID, year, int(month), err)
	}

	return &models.MonthlySummary{
		Year:                   year,
		Month:                  month,
		TotalOutCents:          totalOutCents,
		TotalInCents:           totalInCents,
		ActiveBNPLChargesCents: activeBNPLCents,
	}, nil
}

// GetTopRecipients ranks the user's peer-to-peer transfer targets by total funds directed toward them
// within a bounded time window, limited to the top N results.
func (s *Store) GetTopRecipients(userID string, limit int, since time.Time) ([]models.TopRecipient, error) {
	sinceStr := since.Format("2006-01-02")

	query := `
	SELECT u.id, u.name, u.email, u.phone_number, u.created_at,
		COALESCE(SUM(p.amount_cents), 0) AS total_sent_cents
	FROM payments p
	JOIN users u ON u.id = p.receiver_id
	WHERE p.sender_id = ? AND p.created_at >= ?
	GROUP BY p.receiver_id
	ORDER BY total_sent_cents DESC
	LIMIT ?;`

	rows, err := s.db.Query(query, userID, sinceStr, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to compute top recipient rankings for user '%s': %w", userID, err)
	}
	defer rows.Close()

	recipients := []models.TopRecipient{}

	for rows.Next() {
		var r models.TopRecipient
		err = rows.Scan(
			&r.Profile.ID,
			&r.Profile.Name,
			&r.Profile.Email,
			&r.Profile.PhoneNumber,
			&r.Profile.CreatedAt,
			&r.TotalSentCents,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top recipient row into result struct: %w", err)
		}
		recipients = append(recipients, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected stream cursor failure during top recipient iteration for user '%s': %w", userID, err)
	}

	return recipients, nil
}

// GetBNPLUtilization returns a ratio representing the user's current outstanding installment debt
// relative to their master credit limit ceiling.
func (s *Store) GetBNPLUtilization(userID string) (float64, error) {
	outstandingQuery := `
	SELECT COALESCE(SUM(amount_cents), 0)
	FROM installments
	WHERE user_id = ? AND is_paid = 0;`

	var outstandingCents int
	err := s.db.QueryRow(outstandingQuery, userID).Scan(&outstandingCents)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate outstanding installment balance for user '%s': %w", userID, err)
	}

	limitQuery := `SELECT credit_limit_cents FROM users WHERE id = ?;`

	var creditLimitCents int
	err = s.db.QueryRow(limitQuery, userID).Scan(&creditLimitCents)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve credit limit ceiling for user '%s': %w", userID, err)
	}

	if creditLimitCents == 0 {
		return 0, nil
	}

	return float64(outstandingCents) / float64(creditLimitCents), nil
}

// GetCreditScoreHistory fetches the chronological audit log of credit score variation events
// for a user from the credit_score_log tracking table.
func (s *Store) GetCreditScoreHistory(userID string) ([]models.CreditScoreLog, error) {
	query := `
	SELECT id, user_id, score, delta, created_at
	FROM credit_score_log
	WHERE user_id = ?
	ORDER BY created_at ASC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract credit score audit log for user '%s': %w", userID, err)
	}
	defer rows.Close()

	logs := []models.CreditScoreLog{}

	for rows.Next() {
		var entry models.CreditScoreLog
		err = rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.Score,
			&entry.Delta,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan credit score log row into audit struct: %w", err)
		}
		logs = append(logs, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected stream cursor failure during credit score history iteration for user '%s': %w", userID, err)
	}

	return logs, nil
}
