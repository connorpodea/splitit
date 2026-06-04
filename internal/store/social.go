package store

import (
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

func (s *Store) SendFriendRequest(request *models.FriendRequest) error {
	query := `
	INSERT INTO friend_requests
	(id, sender_id, receiver_id)
	VALUES (?,?,?);`

	if request.ID == "" {
		request.ID = create_new_ID()
	}
	_, err := s.db.Exec(query, request.ID, request.SenderID, request.ReceiverID)
	if err != nil {
		return fmt.Errorf("failed to insert pending relationship record for invitation request ID '%s': %w", request.ID, err)
	}
	s.CreateNotification(&models.Notification{
		UserID:   request.ReceiverID,
		Type:     "friend_request",
		Title:    "New friend request",
		Body:     "@" + request.SenderID + " wants to be your friend",
		LinkView: "social",
	})
	return nil
}

func (s *Store) ListIncomingFriendRequests(userID string) ([]models.FriendRequest, error) {
	query := `
	SELECT id, sender_id, receiver_id, created_at
	FROM friend_requests
	WHERE receiver_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query incoming friend requests directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	// collect a list of all the requests
	requests := []models.FriendRequest{}
	for rows.Next() {
		var r models.FriendRequest

		err := rows.Scan(&r.ID, &r.SenderID, &r.ReceiverID, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse incoming connection record sequence into target friend request model: %w", err)
		}
		requests = append(requests, r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor error processing user ID '%s' incoming network request tracking loop: %w", userID, err)
	}
	return requests, nil
}

func (s *Store) ListOutgoingFriendRequests(userID string) ([]models.FriendRequest, error) {
	query := `
	SELECT id, sender_id, receiver_id, created_at
	FROM friend_requests
	WHERE sender_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outbound tracking directory for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	requests := []models.FriendRequest{}

	for rows.Next() {
		var r models.FriendRequest

		err = rows.Scan(&r.ID, &r.SenderID, &r.ReceiverID, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching outbound invitation row properties into model reference: %w", err)
		}
		requests = append(requests, r)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor error processing user ID '%s' outgoing network request tracking loop: %w", userID, err)
	}
	return requests, nil
}

func (s *Store) AcceptFriendRequest(requestID, receiverID string) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("friend request acceptance aborted: failed to provision database transaction: %w", err)
	}
	defer transaction.Rollback()

	// Derive senderID directly from the database — never trust a client-supplied value.
	// The WHERE also confirms the caller is the actual recipient of this request.
	var senderID string
	err = transaction.QueryRow(
		`SELECT sender_id FROM friend_requests WHERE id = ? AND receiver_id = ?;`,
		requestID, receiverID,
	).Scan(&senderID)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: request ID '%s' not found or does not belong to receiver '%s': %w", requestID, receiverID, err)
	}

	// Remove the pending invitation row
	_, err = transaction.Exec(`DELETE FROM friend_requests WHERE id = ?;`, requestID)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: unable to clear invitation record ID '%s': %w", requestID, err)
	}

	// Create the bidirectional friendship
	_, err = transaction.Exec(
		`INSERT OR IGNORE INTO friends (user_id, friend_id) VALUES (?1, ?2), (?2, ?1);`,
		senderID, receiverID,
	)
	if err != nil {
		return fmt.Errorf("friend request acceptance failed: failed to generate mutual peer-to-peer mapping for user IDs '%s' and '%s': %w", senderID, receiverID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("friend request acceptance failed: database transaction failed to commit to disk: %w", err)
	}
	s.CreateNotification(&models.Notification{
		UserID:   senderID,
		Type:     "friend_accepted",
		Title:    "Friend request accepted",
		Body:     "@" + receiverID + " accepted your friend request",
		LinkView: "social",
	})
	return nil
}

func (s *Store) DeclineFriendRequest(requestID string) error {
	query := `
	DELETE FROM friend_requests
	WHERE id = ?;`

	_, err := s.db.Exec(query, requestID)
	if err != nil {
		return fmt.Errorf("failed to execute removal delete action on friend request ID '%s': %w", requestID, err)
	}
	return nil
}

func (s *Store) RemoveFriendMutual(userID, friendID string) error {
	query := `
	DELETE FROM friends
	WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?);`

	_, err := s.db.Exec(query, userID, friendID, friendID, userID)
	if err != nil {
		return fmt.Errorf("failed to sever mutual bidirectional connection map matching user IDs '%s' and '%s': %w", userID, friendID, err)
	}
	return nil
}

func (s *Store) ListFriends(userID string) ([]models.Profile, error) {
	// Look up all friends of the current user and use the friends' IDs to return their profile
	query := `
	SELECT users.id, users.name, users.email, users.phone_number, users.created_at
	FROM friends
	JOIN users ON friends.friend_id = users.id
	WHERE friends.user_id = ?
	ORDER BY users.name ASC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate network directory join mapping for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	friends := []models.Profile{}

	for rows.Next() {
		var friend models.Profile

		err := rows.Scan(&friend.ID, &friend.Name, &friend.Email, &friend.PhoneNumber, &friend.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize user profile attributes from friends table query data stream: %w", err)
		}
		friends = append(friends, friend)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected structural cursor disruption within active user friends list scanner loop for ID '%s': %w", userID, err)
	}
	return friends, nil
}
