package store

import (
	"fmt"
	"time"

	"github.com/connorpodea/splitit/internal/models"
)

// GetSpendingTotals computes total sent, total received, and net cash flow
// for a user across a caller-specified time window.
func (s *Store) GetSpendingTotals(userID string, since time.Time) (*models.SpendingTotals, error) {
	// Convert Go time object to standard SQLite text format (YYYY-MM-DD)
	sinceStr := since.Format("2006-01-02")

	// Single-pass query: Sums up 'sent' vs 'received' amounts using conditional CASE statements.
	// COALESCE wraps the sums to guarantee a 0.0 fallback instead of a database NULL.
	query := `
    SELECT 
    COALESCE(SUM(CASE WHEN sender_id = ? THEN amount ELSE 0 END), 0.0) AS total_sent,
    COALESCE(SUM(CASE WHEN receiver_id = ? THEN amount ELSE 0 END), 0.0) AS total_received
    FROM payments
    WHERE (sender_id = ? OR receiver_id = ?) AND created_at >= ?;`

	var totalSent float64
	var totalReceived float64

	err := s.db.QueryRow(query, userID, userID, userID, userID, sinceStr).Scan(&totalSent, &totalReceived)
	if err != nil {
		return nil, fmt.Errorf("failed to execute financial volume aggregation scenario for user '%s': %w", userID, err)
	}

	// Compute net cash flow
	netFlow := totalReceived - totalSent

	// Instantiate the return struct
	spendingTotals := models.SpendingTotals{
		TotalSent:     totalSent,
		TotalReceived: totalReceived,
		Net:           netFlow,
	}

	return &spendingTotals, nil
}

// ListPaymentsByUser aggregates a unified, chronologically ordered transaction feed
// containing both inbound and outbound payments for use in an activity feed view.
func (s *Store) ListPaymentsByUser(userID string) ([]models.Payment, error) {
	query := `
    SELECT id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status, created_at
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
			&p.Amount,
			&p.TotalAmount,
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
    SELECT id, sender_id, receiver_id, amount, total_amount, note, payment_type, total_installments, status, created_at
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
			&p.Amount,
			&p.TotalAmount,
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

// ListWalletTransactions retrieves the absolute balance mutation history (deposits and withdrawals)
// for a user, separate from the peer-to-peer payments ledger.
func (s *Store) ListWalletTransactions(userID string) ([]models.WalletTx, error) {
	return nil, nil
}

// GetMonthlySpendingSummary calculates absolute outflow, inflow, and active BNPL charge totals
// for a specific calendar billing month.
func (s *Store) GetMonthlySpendingSummary(userID string, year int, month time.Month) (*models.MonthlySummary, error) {
	return nil, nil
}

// GetTopRecipients ranks the user's peer-to-peer transfer targets by total funds directed toward them
// within a bounded time window, limited to the top N results.
func (s *Store) GetTopRecipients(userID string, limit int, since time.Time) ([]models.TopRecipient, error) {
	return nil, nil
}

// GetBNPLUtilization returns a ratio representing the user's current outstanding installment debt
// relative to their master credit limit ceiling.
func (s *Store) GetBNPLUtilization(userID string) (float64, error) {
	return 0, nil
}

// GetCreditScoreHistory fetches the chronological audit log of credit score variation events
// for a user from the credit_score_log tracking table.
func (s *Store) GetCreditScoreHistory(userID string) ([]models.CreditScoreLog, error) {
	return nil, nil
}

// GetInstallmentSummary pulls combined installment portfolio metrics — total settled,
// total outstanding, and total overdue amounts — in a single query execution.
func (s *Store) GetInstallmentSummary(userID string) (*models.InstallmentSummary, error) {
	return nil, nil
}
