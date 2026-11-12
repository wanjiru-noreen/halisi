-- name: CreateUser :one
INSERT INTO users (name, email, phone, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, email, phone, password_hash, role, created_at, updated_at, email_verified_at, password_login_enabled;

-- name: GetUser :one
SELECT id, name, email, phone, password_hash, role, created_at, updated_at, email_verified_at, password_login_enabled
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, phone, password_hash, role, created_at, updated_at, email_verified_at, password_login_enabled
FROM users
WHERE lower(email) = lower($1);

-- name: ListUsers :many
SELECT id, name, email, phone, role, created_at, updated_at, email_verified_at, password_login_enabled
FROM users
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET name = $2, email = $3, phone = $4
WHERE id = $1
RETURNING id, name, email, phone, password_hash, role, created_at, updated_at, email_verified_at, password_login_enabled;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id = $1;
