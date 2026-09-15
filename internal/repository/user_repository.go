package repository

import (
	"database/sql"
	"fmt"

	"halisi/internal/database"
	"halisi/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: database.DB}
}

// CreateUser inserts a new user record into the SQLite database, including their role
func (r *UserRepository) CreateUser(user models.User) (models.User, error) {
	query := `INSERT INTO users (name, email, password, role) VALUES (?, ?, ?, ?)`

	result, err := r.db.Exec(query, user.Name, user.Email, user.Password, user.Role)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, fmt.Errorf("failed to retrieve inserted user ID: %w", err)
	}

	user.ID = int(id)
	return user, nil
}

// GetUserByEmail retrieves a user by their email address for authentication
func (r *UserRepository) GetUserByEmail(email string) (models.User, error) {
	query := `SELECT id, name, email, password, role FROM users WHERE email = ?`

	user := models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}
		return models.User{}, fmt.Errorf("failed to query user by email: %w", err)
	}

	return user, nil
}

// GetUsers returns all registered users.
func (r *UserRepository) GetUsers() ([]models.User, error) {
	rows, err := r.db.Query(`SELECT id, name, email, password, role FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Role); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}
