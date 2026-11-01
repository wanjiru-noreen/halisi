package repository

import (
	"context"
	"database/sql"
	"fmt"

	"halisi/internal/database"
	"halisi/internal/database/queried"
	"halisi/internal/models"
)

type UserRepository struct {
	q *queried.QueriedDB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{q: queried.NewQueriedDB(database.DB)}
}

// CreateUser inserts a new user record.
func (r *UserRepository) CreateUser(user models.User) (models.User, error) {
	ctx := context.Background()
	result, err := r.q.CreateUser(ctx, struct {
		Name              string
		Email             string
		Password          string
		Role              string
		EmailVerified     bool
		VerificationToken string
		GoogleID          string
	}{
		Name:              user.Name,
		Email:             user.Email,
		Password:          user.Password,
		Role:              user.Role,
		EmailVerified:     user.EmailVerified,
		VerificationToken: user.VerificationToken,
		GoogleID:          user.GoogleID,
	})
	if err != nil {
		return models.User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	return models.User{
		ID:                result.ID,
		Name:              result.Name,
		Email:             result.Email,
		Password:          result.Password,
		Role:              result.Role,
		EmailVerified:     result.EmailVerified,
		VerificationToken: result.VerificationToken,
		GoogleID:          result.GoogleID,
	}, nil
}

// UpdateUnverifiedUser updates an existing unverified user's details.
func (r *UserRepository) UpdateUnverifiedUser(user models.User) error {
	ctx := context.Background()
	err := r.q.UpdateUnverifiedUser(ctx, queried.UpdateUnverifiedUserParams{
		Name:              user.Name,
		Password:          user.Password,
		Role:              user.Role,
		VerificationToken: user.VerificationToken,
		ID:                user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update unverified user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email.
func (r *UserRepository) GetUserByEmail(email string) (models.User, error) {
	ctx := context.Background()
	result, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}

		return models.User{}, fmt.Errorf("failed to query user by email: %w", err)
	}

	return models.User{
		ID:                result.ID,
		Name:              result.Name,
		Email:             result.Email,
		Password:          result.Password,
		Role:              result.Role,
		EmailVerified:     result.EmailVerified,
		VerificationToken: result.VerificationToken,
		GoogleID:          result.GoogleID,
	}, nil
}

// GetUserByVerificationToken retrieves a user by verification token.
func (r *UserRepository) GetUserByVerificationToken(
	token string,
) (models.User, error) {
	ctx := context.Background()
	result, err := r.q.GetUserByVerificationToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}

		return models.User{}, fmt.Errorf(
			"failed to query user by verification token: %w",
			err,
		)
	}

	return models.User{
		ID:                result.ID,
		Name:              result.Name,
		Email:             result.Email,
		Password:          result.Password,
		Role:              result.Role,
		EmailVerified:     result.EmailVerified,
		VerificationToken: result.VerificationToken,
		GoogleID:          result.GoogleID,
	}, nil
}

// VerifyUserEmail marks a user's email as verified and clears the token.
func (r *UserRepository) VerifyUserEmail(userID int) error {
	ctx := context.Background()
	err := r.q.VerifyUserEmail(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to verify user email: %w", err)
	}

	return nil
}

// GetUsers returns all registered users.
func (r *UserRepository) GetUsers() ([]models.User, error) {
	ctx := context.Background()
	results, err := r.q.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}

	users := []models.User{}
	for _, result := range results {
		users = append(users, models.User{
			ID:                result.ID,
			Name:              result.Name,
			Email:             result.Email,
			Password:          result.Password,
			Role:              result.Role,
			EmailVerified:     result.EmailVerified,
			VerificationToken: result.VerificationToken,
			GoogleID:          result.GoogleID,
		})
	}

	return users, nil
}

// GetUserByGoogleID retrieves a user by their Google account ID.
func (r *UserRepository) GetUserByGoogleID(
	googleID string,
) (models.User, error) {
	ctx := context.Background()
	result, err := r.q.GetUserByGoogleID(ctx, googleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}

		return models.User{}, fmt.Errorf(
			"failed to query user by Google ID: %w",
			err,
		)
	}

	return models.User{
		ID:                result.ID,
		Name:              result.Name,
		Email:             result.Email,
		Password:          result.Password,
		Role:              result.Role,
		EmailVerified:     result.EmailVerified,
		VerificationToken: result.VerificationToken,
		GoogleID:          result.GoogleID,
	}, nil
}

// LinkGoogleID links a Google account to an existing verified user.
func (r *UserRepository) LinkGoogleID(
	userID int,
	googleID string,
) error {
	ctx := context.Background()
	err := r.q.LinkGoogleID(ctx, queried.LinkGoogleIDParams{
		GoogleID: googleID,
		ID:       userID,
	})
	if err != nil {
		return fmt.Errorf("failed to link Google account: %w", err)
	}

	// Check if the user was actually updated
	var count int
	err = r.q.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = $1 AND google_id = $2", userID, googleID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check Google account link: %w", err)
	}

	if count != 1 {
		return fmt.Errorf(
			"Google account could not be linked: user is unverified or already linked",
		)
	}

	return nil
}
