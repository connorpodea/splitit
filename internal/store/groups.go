package store

import (
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

// CreateGroup inserts a new group record and immediately registers the creator as its first member
// in a single atomic transaction.
func (s *Store) CreateGroup(group *models.Group) error {
	// Generate a unique group identifier inside the store so the client never controls primary keys
	if group.ID == "" {
		group.ID = create_new_ID()
	}

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("group creation aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, roll back all pending changes
	defer transaction.Rollback()

	query := `
	INSERT INTO groups (id, name, creator_id) VALUES (?, ?, ?);`

	_, err = transaction.Exec(query, group.ID, group.Name, group.CreatorID)
	if err != nil {
		return fmt.Errorf("group creation failed: unable to write group record for group ID '%s': %w", group.ID, err)
	}

	// Immediately bind the creator to the membership table as the first member
	query = `
	INSERT INTO group_members (group_id, member_id) VALUES (?, ?);`

	_, err = transaction.Exec(query, group.ID, group.CreatorID)
	if err != nil {
		return fmt.Errorf("group creation failed: unable to register creator '%s' as initial member of group ID '%s': %w", group.CreatorID, group.ID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("group creation failed: transaction commit to disk rejected for group ID '%s': %w", group.ID, err)
	}
	return nil
}

// ListGroups retrieves all groups that a user is a member of, ordered by name.
func (s *Store) ListGroups(userID string) ([]models.Group, error) {
	query := `
	SELECT groups.id, groups.name, groups.creator_id, groups.created_at
	FROM groups
	JOIN group_members ON groups.id = group_members.group_id
	WHERE group_members.member_id = ?
	ORDER BY groups.name ASC;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user group collection matrices: %w", err)
	}
	defer rows.Close()

	list := []models.Group{}

	for rows.Next() {
		var g models.Group

		err = rows.Scan(
			&g.ID,
			&g.Name,
			&g.CreatorID,
			&g.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan database group row into model properties: %w", err)
		}
		list = append(list, g)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error encountered during group record row scanning loops: %w", err)
	}
	return list, nil
}

// ListGroupMembers returns the public profile of every active member in a group, ordered by name.
func (s *Store) ListGroupMembers(groupID string) ([]models.Profile, error) {
	query := `
	SELECT users.id, users.name, users.email, users.phone_number, users.profile_color, users.created_at
	FROM users
	JOIN group_members ON group_members.member_id = users.id
	WHERE group_members.group_id = ? AND users.is_active = 1
	ORDER BY users.name ASC;`

	rows, err := s.db.Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query member roster for group ID '%s': %w", groupID, err)
	}
	defer rows.Close()

	members := []models.Profile{}

	for rows.Next() {
		var m models.Profile

		err = rows.Scan(
			&m.ID,
			&m.Name,
			&m.Email,
			&m.PhoneNumber,
			&m.ProfileColor,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize member profile record from group roster query for group ID '%s': %w", groupID, err)
		}
		members = append(members, m)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during member roster iteration for group ID '%s': %w", groupID, err)
	}
	return members, nil
}

// SendGroupInvitation creates a pending invitation record from a sender to a receiver for a specific group,
// then dispatches a notification to the receiver so it appears in their Notification Center.
func (s *Store) SendGroupInvitation(groupID, senderID, receiverID string) error {
	var groupName string
	if err := s.db.QueryRow(`SELECT name FROM groups WHERE id = ?;`, groupID).Scan(&groupName); err != nil {
		groupName = "a group"
	}

	query := `
	INSERT INTO group_invitations
	(id, group_id, sender_id, receiver_id)
	VALUES (?,?,?,?);`

	_, err := s.db.Exec(query, create_new_ID(), groupID, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("failed to insert group invitation from sender '%s' to receiver '%s' for group ID '%s': %w", senderID, receiverID, groupID, err)
	}

	go s.CreateNotification(&models.Notification{
		UserID:   receiverID,
		Type:     "group_invitation",
		Title:    "Group invitation",
		Body:     fmt.Sprintf("@%s invited you to join %s", senderID, groupName),
		LinkView: "groups",
	})

	return nil
}

// AcceptGroupInvitation removes the invitation row and registers the caller as a group member
// in a single atomic transaction. The groupID is derived from the database to prevent client spoofing.
func (s *Store) AcceptGroupInvitation(invitationID, callerID string) error {
	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("group invitation acceptance aborted: failed to open transaction session context: %w", err)
	}
	// If the function exits early due to an error, discard all pending changes
	defer transaction.Rollback()

	// Derive the group from the database and confirm the caller is the actual receiver —
	// never trust a group ID supplied by the client
	var groupID string
	query := `
	SELECT group_id
	FROM group_invitations
	WHERE id = ? AND receiver_id = ?;`

	err = transaction.QueryRow(query, invitationID, callerID).Scan(&groupID)
	if err != nil {
		return fmt.Errorf("group invitation acceptance failed: invitation ID '%s' not found or does not belong to receiver '%s': %w", invitationID, callerID, err)
	}

	// Remove the invitation row — accepted invitations do not persist as a status change
	query = `
	DELETE FROM group_invitations
	WHERE id = ?;`

	_, err = transaction.Exec(query, invitationID)
	if err != nil {
		return fmt.Errorf("group invitation acceptance failed: unable to remove invitation record ID '%s': %w", invitationID, err)
	}

	// Register the new member in the group roster
	query = `
	INSERT OR IGNORE INTO group_members (group_id, member_id) VALUES (?, ?);`

	_, err = transaction.Exec(query, groupID, callerID)
	if err != nil {
		return fmt.Errorf("group invitation acceptance failed: unable to register user '%s' as a member of group ID '%s': %w", callerID, groupID, err)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("group invitation acceptance failed: transaction commit to disk rejected for invitation ID '%s': %w", invitationID, err)
	}
	return nil
}

// DeclineGroupInvitation deletes a pending group invitation, restricted to the intended receiver
// via a dual-key WHERE constraint.
func (s *Store) DeclineGroupInvitation(invitationID, callerID string) error {
	// The WHERE clause on receiver_id ensures a user can only decline invitations sent to them
	query := `
	DELETE FROM group_invitations
	WHERE id = ? AND receiver_id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction context for group invitation decline: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.Exec(query, invitationID, callerID)
	if err != nil {
		return fmt.Errorf("failed to decline group invitation ID '%s' for receiver '%s': %w", invitationID, callerID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("decline rejected: invitation ID '%s' not found or does not belong to receiver '%s'", invitationID, callerID)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("failed to finalize group invitation decline on disk commit: %w", err)
	}
	return nil
}

// ListIncomingGroupInvitations returns all pending group invitations sent to a specific receiver.
func (s *Store) ListIncomingGroupInvitations(receiverID string) ([]models.GroupInvitation, error) {
	query := `
	SELECT id, group_id, sender_id, receiver_id, created_at
	FROM group_invitations
	WHERE receiver_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, receiverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query incoming group invitation directory for receiver ID '%s': %w", receiverID, err)
	}
	defer rows.Close()

	invitations := []models.GroupInvitation{}

	for rows.Next() {
		var inv models.GroupInvitation

		err = rows.Scan(
			&inv.ID,
			&inv.GroupID,
			&inv.SenderID,
			&inv.ReceiverID,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize incoming group invitation record into model structure: %w", err)
		}
		invitations = append(invitations, inv)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during incoming group invitation iteration for receiver ID '%s': %w", receiverID, err)
	}
	return invitations, nil
}

// ListOutgoingGroupInvitations returns all pending group invitations that a sender has dispatched.
func (s *Store) ListOutgoingGroupInvitations(senderID string) ([]models.GroupInvitation, error) {
	query := `
	SELECT id, group_id, sender_id, receiver_id, created_at
	FROM group_invitations
	WHERE sender_id = ?
	ORDER BY created_at DESC;`

	rows, err := s.db.Query(query, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outgoing group invitation directory for sender ID '%s': %w", senderID, err)
	}
	defer rows.Close()

	invitations := []models.GroupInvitation{}

	for rows.Next() {
		var inv models.GroupInvitation

		err = rows.Scan(
			&inv.ID,
			&inv.GroupID,
			&inv.SenderID,
			&inv.ReceiverID,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize outgoing group invitation record into model structure: %w", err)
		}
		invitations = append(invitations, inv)
	}

	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected mid-stream cursor failure during outgoing group invitation iteration for sender ID '%s': %w", senderID, err)
	}
	return invitations, nil
}

// RemoveGroupMember forcibly removes a target user from a group's membership roster.
// Intended for group admin or moderation actions.
func (s *Store) RemoveGroupMember(groupID, targetUserID string) error {
	query := `
	DELETE FROM group_members
	WHERE group_id = ? AND member_id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction context for group member removal: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.Exec(query, groupID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to remove target user ID '%s' from group ID '%s' membership roster: %w", targetUserID, groupID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("removal rejected: user ID '%s' is not a member of group ID '%s'", targetUserID, groupID)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("failed to finalize group member removal on disk commit: %w", err)
	}
	return nil
}

// ListGroupActivity returns all payments where both the sender and receiver are members of the
// specified group, ordered by most recent first. Limited to 50 records.
func (s *Store) ListGroupActivity(groupID string) ([]models.Payment, error) {
	query := `
	SELECT id, sender_id, receiver_id, amount_cents, total_amount_cents, note, payment_type, total_installments, status, created_at
	FROM payments
	WHERE sender_id   IN (SELECT member_id FROM group_members WHERE group_id = ?)
	  AND receiver_id IN (SELECT member_id FROM group_members WHERE group_id = ?)
	ORDER BY created_at DESC
	LIMIT 50;`

	rows, err := s.db.Query(query, groupID, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query activity log for group ID '%s': %w", groupID, err)
	}
	defer rows.Close()

	payments := []models.Payment{}

	for rows.Next() {
		var p models.Payment
		err = rows.Scan(
			&p.ID,
			&p.SenderID,
			&p.ReceiverID,
			&p.AmountCents,
			&p.TotalAmountCents,
			&p.Note,
			&p.PaymentType,
			&p.TotalInstallments,
			&p.Status,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment row into model for group activity query: %w", err)
		}
		payments = append(payments, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error encountered during group activity row scanning for group ID '%s': %w", groupID, err)
	}
	return payments, nil
}

// GetGroupMemberCounts returns the member count for every group the user belongs to,
// resolved in a single subquery to avoid N+1 scans per group.
func (s *Store) GetGroupMemberCounts(userID string) (map[string]int, error) {
	query := `
	SELECT gm.group_id, COUNT(*) AS member_count
	FROM group_members gm
	WHERE gm.group_id IN (
		SELECT group_id FROM group_members WHERE member_id = ?
	)
	GROUP BY gm.group_id;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query member count index for user ID '%s': %w", userID, err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var groupID string
		var count int
		if err = rows.Scan(&groupID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan member count row: %w", err)
		}
		counts[groupID] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during member count iteration for user ID '%s': %w", userID, err)
	}
	return counts, nil
}

// LeaveGroup removes the calling user from a group's membership roster voluntarily.
func (s *Store) LeaveGroup(groupID, userID string) error {
	query := `
	DELETE FROM group_members
	WHERE group_id = ? AND member_id = ?;`

	transaction, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to open transaction context for group leave: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.Exec(query, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user ID '%s' from group ID '%s' membership roster: %w", userID, groupID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("leave rejected: user ID '%s' is not a member of group ID '%s'", userID, groupID)
	}

	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("failed to finalize group leave on disk commit: %w", err)
	}
	return nil
}
