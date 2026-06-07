package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

// GetUserSettings pulls preference and privacy configuration metadata from the user_settings table.
func (s *Store) GetUserSettings(userID string) (*models.UserSettings, error) {
	query := `
    SELECT user_id, email_notifications, is_discoverable, updated_at
    FROM user_settings
    WHERE user_id = ?;`

	var settings models.UserSettings
	var emailNotifsInt int
	var isDiscoverableInt int

	err := s.db.QueryRow(query, userID).Scan(
		&settings.UserID,
		&emailNotifsInt,
		&isDiscoverableInt,
		&settings.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user configuration not found for user ID '%s': %w", userID, err)
		}
		return nil, fmt.Errorf("failed to read settings for user ID '%s': %w", userID, err)
	}

	settings.EmailNotifications = emailNotifsInt == 1
	settings.IsDiscoverable = isDiscoverableInt == 1

	return &settings, nil
}

// UpsertUserSettings inserts or updates the preference and privacy configuration for a user
// in a single conflict-safe operation.
func (s *Store) UpsertUserSettings(settings *models.UserSettings) error {
	query := `
	INSERT INTO user_settings (user_id, email_notifications, is_discoverable, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT (user_id) DO UPDATE SET
		email_notifications = excluded.email_notifications,
		is_discoverable = excluded.is_discoverable,
		updated_at = CURRENT_TIMESTAMP;`

	emailNotifsInt := 0
	if settings.EmailNotifications {
		emailNotifsInt = 1
	}

	isDiscoverableInt := 0
	if settings.IsDiscoverable {
		isDiscoverableInt = 1
	}

	_, err := s.db.Exec(query, settings.UserID, emailNotifsInt, isDiscoverableInt)
	if err != nil {
		return fmt.Errorf("failed to save settings for user ID '%s': %w", settings.UserID, err)
	}

	return nil
}
