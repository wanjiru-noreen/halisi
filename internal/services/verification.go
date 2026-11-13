package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
)

var ErrInvalidVerification = errors.New("verification link is invalid or expired; request a new one")
var ErrEmailDelivery = errors.New("verification email could not be sent; please try again")

type VerificationMailer interface {
	SendVerification(context.Context, string, string) error
}
type AuthOption func(*Auth)

func WithVerification(mailer VerificationMailer, baseURL string) AuthOption {
	return func(a *Auth) { a.mailer = mailer; a.baseURL = baseURL }
}
func (a *Auth) VerificationEnabled() bool { return a.mailer != nil }

func (a *Auth) sendVerification(ctx context.Context, q *dbsqlc.Queries, u dbsqlc.User) error {
	if a.mailer == nil {
		return ErrEmailDelivery
	}
	token, err := randomToken()
	if err != nil {
		return err
	}
	if err := q.InvalidateVerificationTokens(ctx, u.ID); err != nil {
		return err
	}
	if err := q.CreateVerificationToken(ctx, dbsqlc.CreateVerificationTokenParams{UserID: u.ID, Email: u.Email, TokenHash: TokenHash(token)}); err != nil {
		return err
	}
	if err := a.mailer.SendVerification(ctx, u.Email, a.baseURL+"/verify-email#token="+token); err != nil {
		return ErrEmailDelivery
	}
	return nil
}

func (a *Auth) ResendVerification(ctx context.Context, email string) error {
	if a.mailer == nil {
		return ErrEmailDelivery
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := a.queries.WithTx(tx)
	u, err := q.GetUserByEmail(ctx, NormalizeEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	u, err = q.LockAuthUser(ctx, u.ID)
	if err != nil {
		return err
	}
	if u.EmailVerifiedAt.Valid {
		return nil
	}
	if err := a.sendVerification(ctx, q, u); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *Auth) VerifyEmail(ctx context.Context, token string) error {
	if !ValidToken(token) {
		return ErrInvalidVerification
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := a.queries.WithTx(tx)
	stored, err := q.GetVerificationToken(ctx, TokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidVerification
	}
	if err != nil {
		return err
	}
	u, err := q.LockAuthUser(ctx, stored.UserID)
	if err != nil {
		return err
	}
	if u.EmailVerifiedAt.Valid || u.Email != stored.Email {
		return ErrInvalidVerification
	}
	_, err = q.ConsumeVerificationToken(ctx, dbsqlc.ConsumeVerificationTokenParams{TokenHash: TokenHash(token), UserID: u.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidVerification
	}
	if err != nil {
		return err
	}
	if _, err := q.MarkEmailVerified(ctx, u.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
