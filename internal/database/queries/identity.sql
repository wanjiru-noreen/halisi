-- name: LockAuthUser :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- name: InvalidateVerificationTokens :exec
UPDATE email_verification_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL;

-- name: CreateVerificationToken :exec
INSERT INTO email_verification_tokens (user_id,email,token_hash,expires_at)
VALUES ($1,$2,$3,NOW()+INTERVAL '1 hour');

-- name: GetVerificationToken :one
SELECT * FROM email_verification_tokens WHERE token_hash = $1;

-- name: ConsumeVerificationToken :one
UPDATE email_verification_tokens SET used_at=NOW()
WHERE token_hash=$1 AND user_id=$2 AND used_at IS NULL AND expires_at>NOW()
RETURNING *;

-- name: MarkEmailVerified :one
UPDATE users SET email_verified_at=NOW() WHERE id=$1 RETURNING *;

-- name: CreateOAuthState :exec
INSERT INTO oauth_states (state_hash,browser_hash,nonce_hash,code_verifier,link_user_id,link_session_hash)
VALUES ($1,$2,$3,$4,$5,$6);

-- name: ConsumeOAuthState :one
DELETE FROM oauth_states WHERE state_hash=$1 AND browser_hash=$2 AND expires_at>NOW() RETURNING *;

-- name: GetRecentSession :one
SELECT * FROM sessions WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>NOW() AND created_at>NOW()-INTERVAL '10 minutes' FOR UPDATE;

-- name: GetGoogleUser :one
SELECT u.* FROM users u JOIN google_accounts g ON g.user_id=u.id WHERE g.subject=$1;

-- name: LinkGoogleAccount :exec
INSERT INTO google_accounts (subject,user_id) VALUES ($1,$2);

-- name: CreateGoogleUser :one
INSERT INTO users (name,email,password_hash,role,password_login_enabled,email_verified_at)
VALUES ($1,$2,$3,'customer',FALSE,$4) RETURNING *;
