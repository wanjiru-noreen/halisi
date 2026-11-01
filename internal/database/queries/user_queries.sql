-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password,
    role,
    email_verified,
    verification_token,
    google_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByVerificationToken :one
SELECT * FROM users
WHERE verification_token = $1;

-- name: GetUserByGoogleID :one
SELECT * FROM users
WHERE google_id = $1;

-- name: GetUsers :many
SELECT * FROM users
ORDER BY id;

-- name: UpdateUnverifiedUser :exec
UPDATE users
SET
    name = $1,
    password = $2,
    role = $3,
    email_verified = FALSE,
    verification_token = $4
WHERE id = $5
  AND email_verified = FALSE;

-- name: VerifyUserEmail :exec
UPDATE users
SET
    email_verified = TRUE,
    verification_token = ''
WHERE id = $1;

-- name: LinkGoogleID :exec
UPDATE users
SET google_id = $1
WHERE id = $2
  AND email_verified = TRUE
  AND google_id = '';
