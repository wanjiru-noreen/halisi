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

// CreateUser inserts a new user record.
func (r *UserRepository) CreateUser(user models.User) (models.User, error) {
	query := `
		INSERT INTO users (
			name,
			email,
			password,
			role,
			email_verified,
			verification_token,
			google_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		user.Name,
		user.Email,
		user.Password,
		user.Role,
		user.EmailVerified,
		user.VerificationToken,
		user.GoogleID,
	)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, fmt.Errorf(
			"failed to retrieve inserted user ID: %w",
			err,
		)
	}

	user.ID = int(id)
	return user, nil
}

// UpdateUnverifiedUser updates an existing unverified user's details.
func (r *UserRepository) UpdateUnverifiedUser(user models.User) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET
			name = ?,
			password = ?,
			role = ?,
			email_verified = 0,
			verification_token = ?
		WHERE id = ?
		  AND email_verified = 0
	`,
		user.Name,
		user.Password,
		user.Role,
		user.VerificationToken,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update unverified user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email.
func (r *UserRepository) GetUserByEmail(email string) (models.User, error) {
	query := `
		SELECT
			id,
			name,
			email,
			password,
			role,
			email_verified,
			verification_token,
			google_id
		FROM users
		WHERE email = ?
	`

	var user models.User

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.EmailVerified,
		&user.VerificationToken,
		&user.GoogleID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}

		return models.User{}, fmt.Errorf("failed to query user by email: %w", err)
	}

	return user, nil
}

// GetUserByVerificationToken retrieves a user by verification token.
func (r *UserRepository) GetUserByVerificationToken(
	token string,
) (models.User, error) {
	query := `
		SELECT
			id,
			name,
			email,
			password,
			role,
			email_verified,
			verification_token,
			google_id
		FROM users
		WHERE verification_token = ?
	`

	var user models.User

	err := r.db.QueryRow(query, token).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.EmailVerified,
		&user.VerificationToken,
		&user.GoogleID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}

		return models.User{}, fmt.Errorf(
			"failed to query user by verification token: %w",
			err,
		)
	}

	return user, nil
}

// VerifyUserEmail marks a user's email as verified and clears the token.
func (r *UserRepository) VerifyUserEmail(userID int) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET
			email_verified = 1,
			verification_token = ''
		WHERE id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to verify user email: %w", err)
	}

	return nil
}

// GetUsers returns all registered users.
func (r *UserRepository) GetUsers() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT
			id,
			name,
			email,
			password,
			role,
			email_verified,
			verification_token,
			google_id
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := []models.User{}

	for rows.Next() {
		var user models.User

		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password,
			&user.Role,
			&user.EmailVerified,
			&user.VerificationToken,
			&user.GoogleID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}

// GetUserByGoogleID retrieves a user by their Google account ID.
func (r *UserRepository) GetUserByGoogleID(
	googleID string,
) (models.User, error) {
	query := `
		SELECT
			id,
			name,
			email,
			password,
			role,
			email_verified,
			verification_token,
			google_id
		FROM users
		WHERE google_id = ?
	`

	var user models.User

	err := r.db.QueryRow(query, googleID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.EmailVerified,
		&user.VerificationToken,
		&user.GoogleID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}

		return models.User{}, fmt.Errorf(
			"failed to query user by Google ID: %w",
			err,
		)
	}

	return user, nil
}

// LinkGoogleID links a Google account to an existing verified user.
func (r *UserRepository) LinkGoogleID(
	userID int,
	googleID string,
) error {
	result, err := r.db.Exec(`
		UPDATE users
		SET google_id = ?
		WHERE id = ?
		  AND email_verified = 1
		  AND google_id = ''
	`, googleID, userID)
	if err != nil {
		return fmt.Errorf("failed to link Google account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check Google account link: %w", err)
	}

	if rows != 1 {
		return fmt.Errorf(
			"Google account could not be linked: user is unverified or already linked",
		)
	}

	return nil
}
