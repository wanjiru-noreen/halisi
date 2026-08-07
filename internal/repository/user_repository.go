package repository

import (
	"database/sql"

	"halisi/internal/database"
	"halisi/internal/models"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Create user
func (r *UserRepository) CreateUser(user models.User) (models.User, error) {

	query := `
	INSERT INTO users(name, email, password, role)
	VALUES (?, ?, ?, ?)
	`

	result, err := database.DB.Exec(
		query,
		user.Name,
		user.Email,
		user.Password,
		user.Role,
	)

	if err != nil {
		return models.User{}, err
	}

	id, err := result.LastInsertId()

	if err != nil {
		return models.User{}, err
	}

	user.ID = int(id)

	return user, nil
}

// Get all users
func (r *UserRepository) GetUsers() ([]models.User, error) {

	rows, err := database.DB.Query(`
		SELECT id, name, email, password, role
		FROM users
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []models.User

	for rows.Next() {

		var user models.User

		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password,
			&user.Role,
		)

		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

// Get user by email
func (r *UserRepository) GetUserByEmail(email string) (models.User, error) {

	var user models.User

	query := `
	SELECT id, name, email, password, role
	FROM users
	WHERE email = ?
	`

	err := database.DB.QueryRow(query, email).Scan(
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

		return models.User{}, err
	}

	return user, nil
}
