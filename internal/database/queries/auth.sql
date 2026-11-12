-- name: GetSessionUser :one
SELECT u.id, u.name, u.email, u.phone, u.role
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW();

-- name: RevokeSessionByHash :exec
UPDATE sessions SET revoked_at = NOW()
WHERE token_hash = $1 AND revoked_at IS NULL;
