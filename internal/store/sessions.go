package store

import (
	"crypto/rand"
	"encoding/hex"
)

// CreateSession generates a cryptographically random 64-hex-char token, persists
// the token→userID mapping in the sessions table, and returns the token.
func (s *Store) CreateSession(userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	if _, err := s.db.Exec(`INSERT INTO sessions (token, user_id) VALUES (?, ?)`, token, userID); err != nil {
		return "", err
	}
	return token, nil
}

// sessionTTL is the maximum lifetime of a session token. Tokens older than this are
// treated as expired and deleted from the database on their first use attempt.
const sessionTTL = "-30 days"

// LookupSession returns the userID bound to token, or false if the token is
// unknown, expired, or the owning user has been cascade-deleted.
// Expired tokens are deleted inline so the sessions table self-prunes over time.
func (s *Store) LookupSession(token string) (string, bool) {
	var userID string
	err := s.db.QueryRow(
		`SELECT user_id FROM sessions WHERE token = ? AND datetime(created_at) > datetime('now', ?)`,
		token, sessionTTL,
	).Scan(&userID)
	if err != nil {
		// If the token exists but is expired, remove it so stale rows don't accumulate.
		s.db.Exec(`DELETE FROM sessions WHERE token = ? AND datetime(created_at) <= datetime('now', ?)`, token, sessionTTL)
		return "", false
	}
	return userID, true
}

// DeleteSession removes a single token, signing the user out of that device.
func (s *Store) DeleteSession(token string) {
	s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
}

// DeleteAllSessionsForUser revokes every active session for userID — used when
// an account is deactivated to prevent any existing cookie from still working.
func (s *Store) DeleteAllSessionsForUser(userID string) {
	s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
}
