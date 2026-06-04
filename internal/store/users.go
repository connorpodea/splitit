package store

import (
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
	SELECT id, password_hash, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users 
	WHERE id = ?;`
	var user models.User

	err := s.db.QueryRow(query, userID).Scan(&user.ID, &user.PasswordHash, &user.Name, &user.Email, &user.PhoneNumber, &user.Balance, &user.CreditScore, &user.CreditLimit, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve full account parameters for user ID '%s': %w", userID, err)
	}

	return &user, nil
}

func (s *Store) GetProfile(id string) (*models.Profile, error) {
	query := `
	SELECT id, name, email, phone_number, created_at 
	FROM users 
	WHERE id = ?;`
	var profile models.Profile

	err := s.db.QueryRow(query, id).Scan(&profile.ID, &profile.Name, &profile.Email, &profile.PhoneNumber, &profile.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to pull minimalist public profile metadata for user ID '%s': %w", id, err)
	}

	return &profile, nil
}

func (s *Store) ListUsers() ([]models.User, error) {
	query := `
	SELECT id, name, email, phone_number, balance, credit_score, credit_limit, created_at 
	FROM users
	ORDER BY name ASC;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute full users table scan lookup query: %w", err)
	}
	defer rows.Close()

	users := []models.User{}

	for rows.Next() {
		var u models.User

		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PhoneNumber, &u.Balance, &u.CreditScore, &u.CreditLimit, &u.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row into user data struct during list aggregation: %w", err)
		}

		users = append(users, u)
	}
	// Check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("detected a mid-stream cursor failure during users rows iteration loop: %w", err)
	}
	return users, nil
}

func (s *Store) ListProfiles() ([]models.Profile, error) {
	query := `
	SELECT id, name, email, phone_number, created_at
	FROM users
	ORDER BY name ASC;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute global public profiles query row collection: %w", err)
	}
	defer rows.Close()

	profiles := []models.Profile{}

	for rows.Next() {
		var profile models.Profile

		err := rows.Scan(&profile.ID, &profile.Name, &profile.Email, &profile.PhoneNumber, &profile.CreatedAt)
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
	WHERE name LIKE ? OR email LIKE ?
	ORDER BY name ASC;`

	rows, err := s.db.Query(query, wildcardQuery, wildcardQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute profiles global search query for token '%s': %w", queryStr, err)
	}
	defer rows.Close()

	profiles := []models.Profile{}

	for rows.Next() {
		var p models.Profile
		err = rows.Scan(&p.ID, &p.Name, &p.Email, &p.PhoneNumber, &p.CreatedAt)
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

func (s *Store) UpdateCreditScore(userID string, delta int) error {
	// Clamp the resulting score between 0 and 100 to prevent overflow or underflow
	query := `
	UPDATE users
	SET credit_score = MIN(100, MAX(0, credit_score + ?))
	WHERE id = ?`

	_, err := s.db.Exec(query, delta, userID)
	if err != nil {
		return fmt.Errorf("failed to apply credit score adjustment of %+d points to user ID '%s': %w", delta, userID, err)
	}
	return nil
}