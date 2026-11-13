BEGIN;
DROP TABLE oauth_states;
DROP TABLE google_accounts;
DROP TABLE email_verification_tokens;
DROP TRIGGER users_email_verification_reset ON users;
DROP FUNCTION reset_email_verification();
ALTER TABLE users DROP COLUMN password_login_enabled, DROP COLUMN email_verified_at;
COMMIT;
