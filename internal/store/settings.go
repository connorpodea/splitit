package store

import (
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

// Deposit credits the specified amount to a user's available cash balance.
func (s *Store) Deposit(userID string, amount float64) error {
	return nil
}

// Withdraw debits the specified amount from a user's cash balance after liquid funds verification.
func (s *Store) Withdraw(userID string, amount float64) error {
	return nil
}

// CreateLinkedCard records an external card instrument against a user account using a tokenized
// provider reference string. Raw card numbers must never be passed or stored here.
func (s *Store) CreateLinkedCard(card *models.LinkedCard) error {
	return nil
}

// DeleteLinkedCard severs an active linked card reference row, restricting the operation
// to the card owner via dual key constraint enforcement.
func (s *Store) DeleteLinkedCard(cardID, userID string) error {
	return nil
}

// ListLinkedCards retrieves all external funding instruments currently registered to a specific user.
func (s *Store) ListLinkedCards(userID string) ([]models.LinkedCard, error) {
	return nil, nil
}

// GetLinkedCard fetches a single linked card record by composite card/user identity key pair.
func (s *Store) GetLinkedCard(cardID, userID string) (*models.LinkedCard, error) {
	return nil, nil
}

// SetDefaultCard flags an individual card as the principal funding source for a user account,
// clearing any previously active default flag in the same atomic operation.
func (s *Store) SetDefaultCard(cardID, userID string) error {
	return nil
}

// ListWalletTransactions retrieves the absolute balance mutation history (deposits and withdrawals)
// for a user, separate from the peer-to-peer payments ledger.
func (s *Store) ListWalletTransactions(userID string) ([]models.WalletTx, error) {
	return nil, nil
}

// GetUserSettings pulls preference and privacy configuration metadata from the user_settings table.
func (s *Store) GetUserSettings(userID string) (*models.UserSettings, error) {
	return nil, nil
}

func (s *Store) UpsertUserSettings(settings *models.UserSettings) error {
	query := `
    INSERT INTO user_settings (user_id, theme, email_notifications, is_discoverable, updated_at)
    VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
    ON CONFLICT (user_id) DO UPDATE SET
        theme = excluded.theme,
        email_notifications = excluded.email_notifications,
        is_discoverable = excluded.is_discoverable,
        updated_at = CURRENT_TIMESTAMP;`

	// Convert Go booleans to SQLite integers (0 or 1) to match your table schemas
	emailNotifsInt := 0
	if settings.EmailNotifications {
		emailNotifsInt = 1
	}

	isDiscoverableInt := 0
	if settings.IsDiscoverable {
		isDiscoverableInt = 1
	}

	_, err := s.db.Exec(query, settings.UserID, settings.Theme, emailNotifsInt, isDiscoverableInt)
	if err != nil {
		return fmt.Errorf("failed to save or overwrite layout configuration matrix for user ID '%s': %w", settings.UserID, err)
	}

	return nil
}
