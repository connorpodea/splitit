package store

import (
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

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
