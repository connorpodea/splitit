package store

import (
	"database/sql"

	// This driver connects our Go code to the SQLite database file system
	"github.com/you/p2p-bnpl/internal/models"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

// New initializes our database file
func New() (*Store, error) {

	// This creates a file called "app.db" in your folder if it doesn't exist
	db, err := sql.Open("sqlite", "app.db")
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}

	// Tell the database to create our users table if it isn't there
	err = s.createTables()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT,
		email TEXT,
		balance REAL
	);`

	_, err := s.db.Exec(query)
	return err
}

func (s *Store) CreateUser(u *models.User) error {
	query := `
	INSERT INTO users (id, name, email, balance) 
	values(?,?,?,?);`

	_, err := s.db.Exec(query, u.ID, u.Name, u.Email, u.Balance)
	return err
}

func (s *Store) GetUser(id string) (*models.User, error) {
	query := `
	SELECT id, name, email, balance FROM users WHERE id = ?;`

	row := s.db.QueryRow(query, id)

	var u models.User

	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Balance)

	if err != nil {
		return nil, err
	}

	return &u, nil
}
