package store

import (
	"time"

	"github.com/connorpodea/splitit/internal/models"
)

// GetSpendingTotals computes total sent, total received, and net cash flow
// for a user across a caller-specified time window.
func (s *Store) GetSpendingTotals(userID string, since time.Time) (*models.SpendingTotals, error) {
	return nil, nil
}

// ListPaymentsByUser aggregates a unified, chronologically ordered transaction feed
// containing both inbound and outbound payments for use in an activity feed view.
func (s *Store) ListPaymentsByUser(userID string) ([]models.Payment, error) {
	return nil, nil
}

// ListPaymentsBetweenUsers retrieves the isolated transaction stream shared exclusively
// between two specific user identities.
func (s *Store) ListPaymentsBetweenUsers(userID, otherID string) ([]models.Payment, error) {
	return nil, nil
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
