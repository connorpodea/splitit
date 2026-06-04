package store

import (
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

func (s *Store) CreateNotification(n *models.Notification) error {
	// Generate a unique notification identifier inside the store so the client never controls primary keys
	if n.ID == "" {
		n.ID = create_new_ID()
	}

	query := `
	INSERT INTO notifications 
	(id, user_id, type, title, body, link_view) 
	VALUES (?,?,?,?,?,?);`

	_, err := s.db.Exec(query, n.ID, n.UserID, n.Type, n.Title, n.Body, n.LinkView)
	if err != nil {
		return fmt.Errorf("failed to write notification record for user ID '%s': %w", n.UserID, err)
	}
	return nil
}

func (s *Store) ListNotifications(userID string) ([]models.Notification, error) {
	query := `
	SELECT id, user_id, type, title, body, link_view, is_seen, created_at
	FROM notifications
	WHERE user_id = ? AND is_seen = 0
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve unseen notification records for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	notifs := []models.Notification{}

	for rows.Next() {
		var n models.Notification
		var isSeen int

		err = rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.LinkView, &isSeen, &n.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize notification row into model structure for user ID '%s': %w", userID, err)
		}
		// Convert the SQLite integer flag back to a Go boolean
		n.IsSeen = isSeen == 1
		notifs = append(notifs, n)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during notification iteration for user ID '%s': %w", userID, err)
	}
	return notifs, nil
}

func (s *Store) CountUnseenNotifications(userID string) (int, error) {
	query := `
	SELECT COUNT(*)
	FROM notifications
	WHERE user_id = ? AND is_seen = 0;`

	var count int

	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate unseen notification count metric for user ID '%s': %w", userID, err)
	}
	return count, nil
}

func (s *Store) MarkNotificationSeen(notifID, userID string) error {
	query := `
	UPDATE notifications
	SET is_seen = 1
	WHERE id = ? AND user_id = ?;`

	_, err := s.db.Exec(query, notifID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification ID '%s' as seen for user ID '%s': %w", notifID, userID, err)
	}
	return nil
}

func (s *Store) ClearAllNotifications(userID string) error {
	query := `
	UPDATE notifications
	SET is_seen = 1
	WHERE user_id = ?;`

	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to clear all notification records for user ID '%s': %w", userID, err)
	}
	return nil
}
