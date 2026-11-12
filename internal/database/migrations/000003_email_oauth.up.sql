BEGIN;
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN password_login_enabled BOOLEAN NOT NULL DEFAULT TRUE;
CREATE FUNCTION reset_email_verification() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF lower(NEW.email) <> lower(OLD.email) THEN NEW.email_verified_at = NULL; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER users_email_verification_reset BEFORE UPDATE OF email ON users FOR EACH ROW EXECUTE FUNCTION reset_email_verification();

CREATE TABLE email_verification_tokens (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX email_verification_active_user_idx ON email_verification_tokens(user_id) WHERE used_at IS NULL;
CREATE INDEX email_verification_expires_idx ON email_verification_tokens(expires_at);
CREATE INDEX email_verification_user_idx ON email_verification_tokens(user_id);
CREATE TRIGGER email_verification_updated_at BEFORE UPDATE ON email_verification_tokens FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE google_accounts (
    subject TEXT PRIMARY KEY CHECK (length(subject) BETWEEN 1 AND 255),
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER google_accounts_updated_at BEFORE UPDATE ON google_accounts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE oauth_states (
    state_hash TEXT PRIMARY KEY CHECK (state_hash ~ '^[0-9a-f]{64}$'),
    browser_hash TEXT NOT NULL CHECK (browser_hash ~ '^[0-9a-f]{64}$'),
    nonce_hash TEXT NOT NULL CHECK (nonce_hash ~ '^[0-9a-f]{64}$'),
    code_verifier TEXT NOT NULL CHECK (length(code_verifier) = 43),
    link_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    link_session_hash TEXT REFERENCES sessions(token_hash) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((link_user_id IS NULL) = (link_session_hash IS NULL)),
    CHECK (expires_at > created_at)
);
CREATE INDEX oauth_states_expires_idx ON oauth_states(expires_at);
CREATE INDEX oauth_states_link_user_idx ON oauth_states(link_user_id);
CREATE INDEX oauth_states_link_session_idx ON oauth_states(link_session_hash);
COMMIT;
