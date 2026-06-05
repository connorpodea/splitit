package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/connorpodea/splitit/internal/models"
)

func (s *Store) CreateUser(user *models.User) error {
	query := `
	INSERT INTO users 
	(id, password_hash, name, email, phone_number, balance, credit_score, credit_limit) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?);`

	_, err := s.db.Exec(query, user.ID, user.PasswordHash, user.Name, user.Email, user.PhoneNumber, user.Balance, user.CreditScore, user.CreditLimit)
	if err != nil {
		return fmt.Errorf("failed to write profile for user ID '%s' to database: %w", user.ID, err)
	}
	return nil
}

func (s *Store) GetUser(userID string) (*models.User, error) {
	query := `
	SELECT id, password_hash, name, email, phone_number, balance, credit_score, credit_limit, is_active, created_at
	FROM users
	WHERE id = ?;`
	var user models.User
	var isActiveInt int

	err := s.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.PasswordHash,
		&user.Name,
		&user.Email,
		&user.PhoneNumber,
		&user.Balance,
		&user.CreditScore,
		&user.CreditLimit,
		&isActiveInt,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user account not found for ID '%s': %w", userID, err)
		}
		return nil, fmt.Errorf("failed to retrieve full account parameters for user ID '%s': %w", userID, err)
	}

	user.IsActive = isActiveInt == 1
	return &user, nil
}

func (s *Store) GetProfile(id string) (*models.Profile, error) {
	query := `
	SELECT id, name, email, phone_number, created_at
	FROM users
	WHERE id = ? AND is_active = 1;`
	var profile models.Profile

	err := s.db.QueryRow(query, id).Scan(
		&profile.ID,
		&profile.Name,
		&profile.Email,
		&profile.PhoneNumber,
		&profile.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("profile not found for user ID '%s': %w", id, err)
		}
		return nil, fmt.Errorf("failed to pull minimalist public profile metadata for user ID '%s': %w", id, err)
	}

	return &profile, nil
}

func (s *Store) ListUsers() ([]models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, is_active, created_at
	FROM users
	WHERE is_active = 1
	ORDER BY name ASC;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute full users table scan lookup query: %w", err)
	}
	defer rows.Close()

	users := []models.User{}

	for rows.Next() {
		var u models.User
		var isActiveInt int

		err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.PhoneNumber,
			&u.Balance,
			&u.CreditScore,
			&u.CreditLimit,
			&isActiveInt,
			&u.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row into user data struct during list aggregation: %w", err)
		}

		u.IsActive = isActiveInt == 1
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during users rows iteration loop: %w", err)
	}
	return users, nil
}

func (s *Store) ListProfiles() ([]models.Profile, error) {
	query := `
	SELECT id, name, email, phone_number, created_at
	FROM users
	WHERE is_active = 1
	ORDER BY name ASC;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute global public profiles query row collection: %w", err)
	}
	defer rows.Close()

	profiles := []models.Profile{}

	for rows.Next() {
		var profile models.Profile

		err := rows.Scan(
			&profile.ID,
			&profile.Name,
			&profile.Email,
			&profile.PhoneNumber,
			&profile.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching profile data mapping structure: %w", err)
		}

		profiles = append(profiles, profile)
	}
	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during profiles row iteration loop: %w", err)
	}
	return profiles, nil
}

func (s *Store) SearchProfiles(queryStr string) ([]models.Profile, error) {
	// Transform the query string into a SQL wildcard match format
	wildcardQuery := fmt.Sprintf("%%%s%%", queryStr)

	query := `
	SELECT id, name, email, phone_number, created_at
	FROM users
	WHERE is_active = 1 AND (name LIKE ? OR email LIKE ?)
	ORDER BY name ASC;`

	rows, err := s.db.Query(query, wildcardQuery, wildcardQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute profiles global search query for token '%s': %w", queryStr, err)
	}
	defer rows.Close()

	profiles := []models.Profile{}

	for rows.Next() {
		var p models.Profile
		err = rows.Scan(
			&p.ID,
			&p.Name,
			&p.Email,
			&p.PhoneNumber,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan matching profile search record row: %w", err)
		}
		profiles = append(profiles, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected cursor failure mid-stream during profiles search loop execution:: %w", err)
	}

	return profiles, nil
}

// UpdatePassword overwrites the stored credential hash. Callers are responsible for securely
// hashing the new password externally before passing the digest to this method.
func (s *Store) UpdatePassword(userID, newHashedPassword string) error {
	query := `
	UPDATE users
	SET password_hash = ?
	WHERE id = ?;`

	result, err := s.db.Exec(query, newHashedPassword, userID)
	if err != nil {
		return fmt.Errorf("failed to execute credential mutation segment for user ID '%s': %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read credential update metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("credential mutation rejected: user ID '%s' not found", userID)
	}

	return nil
}

// UpdateEmail replaces the email address on record for a user account.
func (s *Store) UpdateEmail(userID, newEmail string) error {
	query := `
    UPDATE users
    SET email = ?
    WHERE id = ?;`

	result, err := s.db.Exec(query, newEmail, userID)
	if err != nil {
		return fmt.Errorf("failed to execute email mutation segment for user ID '%s': %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read email update execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("email mutation rejected: user ID '%s' not found", userID)
	}

	return nil
}

// UpdatePhoneNumber replaces the phone number on record for a user account.
func (s *Store) UpdatePhoneNumber(userID, newPhoneNumber string) error {
	query := `
    UPDATE users
    SET phone_number = ?
    WHERE id = ?;`

	result, err := s.db.Exec(query, newPhoneNumber, userID)
	if err != nil {
		return fmt.Errorf("failed to execute phone number mutation segment for user ID '%s': %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read phone number update execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("phone number mutation rejected: user ID '%s' not found", userID)
	}

	return nil
}

// UpdateDisplayName replaces the display name for a user account.
func (s *Store) UpdateDisplayName(userID, newName string) error {
	query := `
	UPDATE users
	SET name = ?
	WHERE id = ?;`

	result, err := s.db.Exec(query, newName, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("")
	}
	return nil
}

// DeactivateAccount performs a defensive soft-delete state transition by setting the is_active flag
// to false rather than purging the row, preserving ledger integrity.
func (s *Store) DeactivateAccount(userID string) error {
	query := `
	UPDATE users
	SET is_active = 0
	WHERE id = ?;`

	result, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to execute account deactivation state transition for user ID '%s': %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read account deactivation execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("account deactivation rejected: user ID '%s' not found in user registry", userID)
	}

	return nil
}

func (s *Store) UpdateCreditScore(userID string, delta int) error {
	// Clamp the resulting score between 0 and 100 to prevent overflow or underflow
	query := `
	UPDATE users
	SET credit_score = MIN(100, MAX(0, credit_score + ?))
	WHERE id = ?`

	result, err := s.db.Exec(query, delta, userID)
	if err != nil {
		return fmt.Errorf("failed to apply credit score adjustment of %+d points to user ID '%s': %w", delta, userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read credit score adjustment execution state metrics: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("credit score adjustment rejected: user account ID '%s' not found in user registry", userID)
	}

	return nil
}
