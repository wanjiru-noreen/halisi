package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
	"golang.org/x/crypto/bcrypt"
)

const PasswordCost = 12
const SessionTTL = 24 * time.Hour

var (
	ErrEmailExists        = errors.New("an account with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthenticated    = errors.New("authentication required")
	phonePattern          = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// User is the only user representation returned to HTTP clients.
type User struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"email_verified"`
}

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func ValidateRegistration(input RegisterInput) (RegisterInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = NormalizeEmail(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	if !utf8.ValidString(input.Name) || utf8.RuneCountInString(input.Name) < 2 || utf8.RuneCountInString(input.Name) > 100 || strings.ContainsFunc(input.Name, unicode.IsControl) {
		return input, &ValidationError{"Name must contain 2–100 characters without control characters."}
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 {
		return input, &ValidationError{"Enter a valid email address."}
	}
	if !phonePattern.MatchString(input.Phone) {
		return input, &ValidationError{"Enter a phone number in international format, such as +254712345678."}
	}
	if !utf8.ValidString(input.Password) || utf8.RuneCountInString(input.Password) < 12 || len(input.Password) > 72 {
		return input, &ValidationError{"Password must have at least 12 characters and at most 72 bytes."}
	}
	return input, nil
}

type Auth struct {
	pool      *pgxpool.Pool
	queries   *dbsqlc.Queries
	dummyHash []byte
	mailer    VerificationMailer
	baseURL   string
	google    GoogleProvider
}

func NewAuth(pool *pgxpool.Pool, options ...AuthOption) (*Auth, error) {
	dummy := make([]byte, 32)
	if _, err := rand.Read(dummy); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword(dummy, PasswordCost)
	if err != nil {
		return nil, err
	}
	a := &Auth{pool: pool, queries: dbsqlc.New(pool), dummyHash: hash}
	for _, option := range options {
		option(a)
	}
	return a, nil
}

func (a *Auth) Register(ctx context.Context, input RegisterInput) (User, error) {
	input, err := ValidateRegistration(input)
	if err != nil {
		return User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), PasswordCost)
	if err != nil {
		return User{}, err
	}
	if a.mailer == nil {
		return User{}, ErrEmailDelivery
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	q := a.queries.WithTx(tx)
	u, err := q.CreateUser(ctx, dbsqlc.CreateUserParams{Name: input.Name, Email: input.Email, Phone: pgtype.Text{String: input.Phone, Valid: true}, PasswordHash: string(hash), Role: "customer"})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailExists
		}
		return User{}, err
	}
	if err := a.sendVerification(ctx, q, u); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return publicUser(u), nil
}

type LoginSession struct {
	Token     string
	ExpiresAt time.Time
	User      User
}

func (a *Auth) Login(ctx context.Context, email, password, oldToken string) (LoginSession, error) {
	if len(email) > 254 || len(password) > 72 || password == "" {
		return LoginSession{}, ErrInvalidCredentials
	}
	u, err := a.queries.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return LoginSession{}, err
	}
	hash := a.dummyHash
	if err == nil && u.PasswordLoginEnabled {
		hash = []byte(u.PasswordHash)
	}
	match := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil || match != nil || !u.PasswordLoginEnabled {
		return LoginSession{}, ErrInvalidCredentials
	}
	return a.createLoginSession(ctx, u, oldToken)
}

func (a *Auth) createLoginSession(ctx context.Context, u dbsqlc.User, oldToken string) (LoginSession, error) {
	token, err := randomToken()
	if err != nil {
		return LoginSession{}, err
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return LoginSession{}, err
	}
	defer tx.Rollback(ctx)
	q := a.queries.WithTx(tx)
	if ValidToken(oldToken) {
		if err := q.RevokeSessionByHash(ctx, TokenHash(oldToken)); err != nil {
			return LoginSession{}, err
		}
	}
	session, err := q.CreateSession(ctx, dbsqlc.CreateSessionParams{UserID: u.ID, TokenHash: TokenHash(token), ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(SessionTTL), Valid: true}})
	if err != nil {
		return LoginSession{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LoginSession{}, err
	}
	return LoginSession{Token: token, ExpiresAt: session.ExpiresAt.Time, User: publicUser(u)}, nil
}

func (a *Auth) Authenticate(ctx context.Context, token string) (User, error) {
	if !ValidToken(token) {
		return User{}, ErrUnauthenticated
	}
	u, err := a.queries.GetSessionUser(ctx, TokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, err
	}
	return User{ID: u.ID, Name: u.Name, Email: u.Email, Phone: u.Phone.String, Role: u.Role, EmailVerified: u.EmailVerifiedAt.Valid}, nil
}

func (a *Auth) Logout(ctx context.Context, token string) error {
	if !ValidToken(token) {
		return nil
	}
	return a.queries.RevokeSessionByHash(ctx, TokenHash(token))
}

func ValidToken(token string) bool {
	if len(token) != 43 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(decoded) == 32 && base64.RawURLEncoding.EncodeToString(decoded) == token
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func publicUser(u dbsqlc.User) User {
	return User{ID: u.ID, Name: u.Name, Email: u.Email, Phone: u.Phone.String, Role: u.Role, EmailVerified: u.EmailVerifiedAt.Valid}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
