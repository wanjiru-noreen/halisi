package services

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrGoogle       = errors.New("Google sign-in could not be completed; please try again")
	ErrLinkRequired = errors.New("an account uses this email; log in to that account, verify your email, and connect Google from your account page")
	ErrRecentLogin  = errors.New("log in again and verify your email before connecting Google")
	ErrEmailVerificationRequired = errors.New("verify your email address before signing in with Google")
)

type GoogleIdentity struct {
	Subject, Email, Name, HostedDomain string
	EmailVerified                      bool
}
type GoogleProvider interface {
	AuthURL(state, nonce, verifier string) string
	VerifyCode(context.Context, string, string, string) (GoogleIdentity, error)
}

func WithGoogle(provider GoogleProvider) AuthOption { return func(a *Auth) { a.google = provider } }
func (a *Auth) GoogleEnabled() bool                 { return a.google != nil }

type GoogleFlow struct{ URL, BrowserToken string }

func (a *Auth) StartGoogle(ctx context.Context, sessionToken string, link bool) (GoogleFlow, error) {
	if a.google == nil {
		return GoogleFlow{}, ErrGoogle
	}
	state, err := randomToken()
	if err != nil {
		return GoogleFlow{}, err
	}
	browser, err := randomToken()
	if err != nil {
		return GoogleFlow{}, err
	}
	nonce, err := randomToken()
	if err != nil {
		return GoogleFlow{}, err
	}
	verifier, err := randomToken()
	if err != nil {
		return GoogleFlow{}, err
	}
	params := dbsqlc.CreateOAuthStateParams{StateHash: TokenHash(state), BrowserHash: TokenHash(browser), NonceHash: TokenHash(nonce), CodeVerifier: verifier}
	if link {
		if !ValidToken(sessionToken) {
			return GoogleFlow{}, ErrRecentLogin
		}
		session, err := a.queries.GetRecentSession(ctx, TokenHash(sessionToken))
		if err != nil {
			return GoogleFlow{}, ErrRecentLogin
		}
		u, err := a.queries.GetUser(ctx, session.UserID)
		if err != nil {
			return GoogleFlow{}, err
		}
		if !u.EmailVerifiedAt.Valid {
			return GoogleFlow{}, ErrRecentLogin
		}
		params.LinkUserID = pgtype.Int8{Int64: u.ID, Valid: true}
		params.LinkSessionHash = pgtype.Text{String: TokenHash(sessionToken), Valid: true}
	}
	if err := a.queries.CreateOAuthState(ctx, params); err != nil {
		return GoogleFlow{}, err
	}
	return GoogleFlow{URL: a.google.AuthURL(state, nonce, verifier), BrowserToken: browser}, nil
}

func (a *Auth) FinishGoogle(ctx context.Context, state, browser, code, sessionToken string) (LoginSession, bool, error) {
	if a.google == nil || !ValidToken(state) || !ValidToken(browser) {
		return LoginSession{}, false, ErrGoogle
	}
	flow, err := a.queries.ConsumeOAuthState(ctx, dbsqlc.ConsumeOAuthStateParams{StateHash: TokenHash(state), BrowserHash: TokenHash(browser)})
	if err != nil {
		return LoginSession{}, false, ErrGoogle
	}
	if code == "" || len(code) > 4096 {
		return LoginSession{}, false, ErrGoogle
	}
	identity, err := a.google.VerifyCode(ctx, code, flow.CodeVerifier, flow.NonceHash)
	if err != nil || !validGoogleIdentity(identity) {
		return LoginSession{}, false, ErrGoogle
	}
	identity.Email = NormalizeEmail(identity.Email)
	if flow.LinkUserID.Valid {
		if !ValidToken(sessionToken) || TokenHash(sessionToken) != flow.LinkSessionHash.String {
			return LoginSession{}, false, ErrRecentLogin
		}
		tx, err := a.pool.Begin(ctx)
		if err != nil {
			return LoginSession{}, false, err
		}
		defer tx.Rollback(ctx)
		q := a.queries.WithTx(tx)
		session, err := q.GetRecentSession(ctx, TokenHash(sessionToken))
		if err != nil || session.UserID != flow.LinkUserID.Int64 {
			return LoginSession{}, false, ErrRecentLogin
		}
		u, err := q.LockAuthUser(ctx, session.UserID)
		if err != nil {
			return LoginSession{}, false, err
		}
		if !u.EmailVerifiedAt.Valid || u.Email != identity.Email {
			return LoginSession{}, false, ErrRecentLogin
		}
		if err := q.LinkGoogleAccount(ctx, dbsqlc.LinkGoogleAccountParams{Subject: identity.Subject, UserID: u.ID}); err != nil {
			return LoginSession{}, false, ErrLinkRequired
		}
		if err := tx.Commit(ctx); err != nil {
			return LoginSession{}, false, err
		}
		return LoginSession{}, true, nil
	}
	u, err := a.queries.GetGoogleUser(ctx, identity.Subject)
	if errors.Is(err, pgx.ErrNoRows) {
		u, err = a.createGoogleUser(ctx, identity)
	}
	if err != nil {
		return LoginSession{}, false, err
	}
	if !u.EmailVerifiedAt.Valid {
		return LoginSession{}, false, ErrEmailVerificationRequired
	}
	session, err := a.createLoginSession(ctx, u, sessionToken)
	return session, false, err
}

func (a *Auth) createGoogleUser(ctx context.Context, identity GoogleIdentity) (dbsqlc.User, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return dbsqlc.User{}, err
	}
	defer tx.Rollback(ctx)
	q := a.queries.WithTx(tx)
	if _, err := q.GetUserByEmail(ctx, identity.Email); err == nil {
		return dbsqlc.User{}, ErrLinkRequired
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return dbsqlc.User{}, err
	}
	// Google-only accounts retain an unguessable hash for downgrade safety, but password login is explicitly disabled.
	random, err := randomToken()
	if err != nil {
		return dbsqlc.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(random), PasswordCost)
	if err != nil {
		return dbsqlc.User{}, err
	}
	name := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, identity.Name))
	if len([]rune(name)) < 2 {
		name = "HALISI customer"
	}
	if len([]rune(name)) > 100 {
		name = string([]rune(name)[:100])
	}
	var verified pgtype.Timestamptz
	domain := strings.Split(identity.Email, "@")[1]
	if domain == "gmail.com" || (identity.HostedDomain != "" && domain == strings.ToLower(identity.HostedDomain)) {
		verified = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}
	u, err := q.CreateGoogleUser(ctx, dbsqlc.CreateGoogleUserParams{Name: name, Email: identity.Email, PasswordHash: string(hash), EmailVerifiedAt: verified})
	if err != nil {
		return dbsqlc.User{}, ErrLinkRequired
	}
	if err := q.LinkGoogleAccount(ctx, dbsqlc.LinkGoogleAccountParams{Subject: identity.Subject, UserID: u.ID}); err != nil {
		return dbsqlc.User{}, ErrLinkRequired
	}
	if !verified.Valid {
		if err := a.sendVerification(ctx, q, u); err != nil {
			return dbsqlc.User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dbsqlc.User{}, err
	}
	return u, nil
}

func validGoogleIdentity(i GoogleIdentity) bool {
	email := NormalizeEmail(i.Email)
	address, err := mail.ParseAddress(email)
	return i.EmailVerified && len(i.Subject) > 0 && len(i.Subject) <= 255 && err == nil && address.Address == email && len(email) <= 254 && strings.Count(email, "@") == 1
}
