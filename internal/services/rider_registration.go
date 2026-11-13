package services

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
)

var ErrRiderDuplicate = errors.New("a rider application already exists for this account")
var ErrRiderProfileMissing = errors.New("no rider application exists for this account")
var ErrRiderProfileLocked = errors.New("only a pending rider profile can be edited")

type RiderInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	City     string `json:"city"`
}
type RiderUpdate struct {
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
	City     *string `json:"city"`
}

func validRiderText(value string) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) >= 2 && utf8.RuneCountInString(value) <= 100 && !strings.ContainsFunc(value, unicode.IsControl)
}
func (input *RiderInput) Validate() error {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Phone = strings.TrimSpace(input.Phone)
	input.City = strings.TrimSpace(input.City)
	if !validRiderText(input.FullName) {
		return &ValidationError{Message: "Full name must contain 2–100 characters without control characters."}
	}
	if !phonePattern.MatchString(input.Phone) {
		return &ValidationError{Message: "Phone must use international format, such as +254712345678."}
	}
	if !validRiderText(input.City) {
		return &ValidationError{Message: "City must contain 2–100 characters without control characters."}
	}
	return nil
}

type RiderRegistration struct{ queries *dbsqlc.Queries }

func NewRiderRegistration(pool *pgxpool.Pool) *RiderRegistration {
	return &RiderRegistration{queries: dbsqlc.New(pool)}
}
func (s *RiderRegistration) Register(ctx context.Context, userID int64, input RiderInput) (dbsqlc.RiderProfile, error) {
	if err := input.Validate(); err != nil {
		return dbsqlc.RiderProfile{}, err
	}
	profile, err := s.queries.RegisterRider(ctx, dbsqlc.RegisterRiderParams{UserID: userID, FullName: input.FullName, Phone: input.Phone, City: input.City})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return dbsqlc.RiderProfile{}, ErrRiderDuplicate
	}
	return profile, err
}
func (s *RiderRegistration) Own(ctx context.Context, userID int64) (dbsqlc.RiderProfile, error) {
	v, err := s.queries.OwnRiderProfile(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrRiderProfileMissing
	}
	return v, err
}
func (s *RiderRegistration) Update(ctx context.Context, userID int64, input RiderUpdate) (dbsqlc.RiderProfile, error) {
	params := dbsqlc.UpdateOwnPendingRiderParams{UserID: userID}
	if input.FullName == nil && input.Phone == nil && input.City == nil {
		return dbsqlc.RiderProfile{}, &ValidationError{Message: "Provide at least one profile field to update."}
	}
	if input.FullName != nil {
		v := strings.TrimSpace(*input.FullName)
		if !validRiderText(v) {
			return dbsqlc.RiderProfile{}, &ValidationError{Message: "Full name must contain 2–100 characters without control characters."}
		}
		params.FullName = pgtype.Text{String: v, Valid: true}
	}
	if input.Phone != nil {
		v := strings.TrimSpace(*input.Phone)
		if !phonePattern.MatchString(v) {
			return dbsqlc.RiderProfile{}, &ValidationError{Message: "Phone must use international format."}
		}
		params.Phone = pgtype.Text{String: v, Valid: true}
	}
	if input.City != nil {
		v := strings.TrimSpace(*input.City)
		if !validRiderText(v) {
			return dbsqlc.RiderProfile{}, &ValidationError{Message: "City must contain 2–100 characters without control characters."}
		}
		params.City = pgtype.Text{String: v, Valid: true}
	}
	v, err := s.queries.UpdateOwnPendingRider(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrRiderProfileLocked
	}
	return v, err
}
func (s *RiderRegistration) RiderApproved(ctx context.Context, userID int64) (bool, error) {
	return s.queries.RiderIsApproved(ctx, userID)
}
