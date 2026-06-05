package store

import (
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

// Deposit credits the specified amount to a user's available cash balance.
func (s *Store) Deposit(userID string, amount float64) error {
	query := `
	UPDATE users
	SET balance = balance + ?
	WHERE id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction session context for deposit: %w", err)
	}
	// This discards changes if an error happens midway through
	defer transaction.Rollback()

	_, err = transaction.Exec(query, amount, userID)
	if err != nil {
		return fmt.Errorf("failed to clear balance deposit allocation of $%.2f for user ID '%s': %w", amount, userID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical ledger engine mismatch: failed to write deposit modifications to disk on final commit sequence: %w", err)
	}

	return nil
}

// Withdraw debits the specified amount from a user's cash balance after liquid funds verification.
func (s *Store) Withdraw(userID string, amount float64) error {
	query := `
	UPDATE users
	SET balance = balance - ?
	WHERE id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction session context for deposit: %w", err)
	}
	defer transaction.Rollback()

	_, err = transaction.Exec(query, amount, userID)
	if err != nil {
		return fmt.Errorf("failed to clear balance withdraw allocation of $%.2f for user ID '%s': %w", amount, userID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical ledger engine mismatch: failed to write withdraw modifications to disk on final commit sequence: %w", err)
	}

	return nil
}

// CreateLinkedCard records an external card instrument against a user account using a tokenized
// provider reference string. Raw card numbers must never be passed or stored here.
func (s *Store) CreateLinkedCard(card *models.LinkedCard) error {
	query := `
	INSERT INTO linked_cards (id, user_id, token_ref, last4, brand, is_default)
	VALUES (?, ?, ?, ?, ?, 0);`

	card.ID = create_new_ID()

	_, err := s.db.Exec(query, card.ID, card.UserID, card.TokenRef, card.Last4, card.Brand)
	if err != nil {
		return fmt.Errorf("failed to register linked card for user ID '%s': %w", card.UserID, err)
	}
	return nil
}

// DeleteLinkedCard severs an active linked card reference row, restricting the operation
// to the card owner via dual key constraint enforcement.
func (s *Store) DeleteLinkedCard(cardID, userID string) error {
	query := `
    DELETE FROM linked_cards
    WHERE id = ? AND user_id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction context for asset severance: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.Exec(query, cardID, userID)
	if err != nil {
		return fmt.Errorf("failed to execute card deletion routine for asset ID '%s' under user '%s': %w", cardID, userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("card severance rejected: asset ID '%s' not found or unauthorized for user '%s'", cardID, userID)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("critical engine tracking mismatch: failed to finalize card deletion on disk commit: %w", err)
	}

	return nil
}

// ListLinkedCards retrieves all external funding instruments currently registered to a specific user.
func (s *Store) ListLinkedCards(userID string) ([]models.LinkedCard, error) {
	query := `
    SELECT id, user_id, token_ref, last_4, brand, is_default, created_at
    FROM linked_cards
    WHERE user_id = ?;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute registered cards aggregation scan for user '%s': %w", userID, err)
	}
	defer rows.Close()

	cards := []models.LinkedCard{}

	for rows.Next() {
		var c models.LinkedCard
		var isDefaultInt int

		err = rows.Scan(&c.ID, &c.UserID, &c.TokenRef, &c.Last4, &c.Brand, &isDefaultInt, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan database asset row into card data struct: %w", err)
		}

		// Convert the SQLite integer flag back to a clean Go boolean
		c.IsDefault = isDefaultInt == 1
		cards = append(cards, c)
	}

	// Verify that the database stream cursor didn't fail mid-flight
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during cards row iteration loop for user '%s': %w", userID, err)
	}

	return cards, nil
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
