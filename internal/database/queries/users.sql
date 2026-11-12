-- name: CreateUser :one
INSERT INTO users (name, email, phone, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, email, phone, password_hash, role, created_at;

-- name: GetUser :one
SELECT id, name, email, phone, password_hash, role, created_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, phone, password_hash, role, created_at
FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT id, name, email, phone, role, created_at
FROM users
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET name = $2, email = $3, phone = $4
WHERE id = $1
RETURNING id, name, email, phone, password_hash, role, created_at;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id = $1;
