# Email verification and Google sign-in

Apply migration `000003` before starting the updated server. Configuration comes
from process environment variables; `.env` is not loaded automatically.

## SMTP configuration

Set `PUBLIC_BASE_URL` to the trusted application origin, without a path/query.
Production requires HTTPS and `SMTP_HOST`. Verification links never use request
Host headers. Configure `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USERNAME`,
`SMTP_PASSWORD`, and `SMTP_TLS_MODE`. Username/password are either both present
or both absent for an unauthenticated relay. `starttls` defaults to port 587 and
requires STARTTLS and valid certificates, with no plaintext fallback. `tls` uses
implicit TLS, default port 465. `none` is limited to development loopback servers.

For local development:

```sh
docker run --rm -p 127.0.0.1:1025:1025 -p 127.0.0.1:8025:8025 axllent/mailpit
```

Export the example SMTP values in `.env.example`, then read messages at
`http://localhost:8025`. Mailpit captures messages locally, rather than delivering
to internet inboxes. Live delivery requires your SMTP provider credentials and
an authorized sender. Never put credentials in Git or chat.

Registration sends a one-hour verification link before committing its user/token
transaction. SMTP rejection rolls back registration and returns 503. Without
SMTP configured, registration/resend fail closed; tokens are not logged as a
substitute. SMTP acceptance is not a guarantee of inbox delivery: check provider
logs for bounces. If SMTP succeeds but the database commit fails, that link is
unusable; retry registration or request another link.

## Verification endpoints

| Endpoint | JSON input | Result |
| --- | --- | --- |
| `POST /api/auth/verify-email` | `token` | 200 verified; 400 invalid/expired/reused |
| `POST /api/auth/verification/resend` | `email` | Generic 202 for unknown, verified, and unverified accounts; 503 on delivery errors |
| `GET /api/auth/options` | None | Only `google_enabled` and `verification_enabled` booleans |

Tokens contain 256 random bits; only SHA-256 digests are stored. Resending
invalidates previous links. PostgreSQL row locks serialize resend/consumption,
and each token is single-use. Tokens record their target email, so they cannot
verify a changed address. Email changes automatically clear verification.
Resend limits are five requests per IP and three per email per 15 minutes.

Links open `/verify-email#token=...`. The fragment stays out of server request
logs and referrers; JavaScript removes it from browser history and asks for an
explicit click before POSTing. GET requests/email scanners do not consume tokens.
Verification sets `users.email_verified_at` but does not log the user in.
Password login remains available before verification; public user responses add
`email_verified` for use by workflows requiring verified ownership.

## Google setup and validation

Create a Google OAuth **Web application** client and consent configuration for
`openid profile email`. Export `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`.
Register exactly `PUBLIC_BASE_URL/api/auth/google/callback` as an authorized
redirect URI, e.g. `http://localhost:8080/api/auth/google/callback` locally or the
real HTTPS origin in production. Actual Google sign-in needs these credentials
and the Google consent screen/test-user settings. Leaving both variables empty
disables Google and hides its buttons; configuring only one is a startup error.

`POST /api/auth/google/start` accepts `{}` and returns an authorization URL.
A ten-minute PostgreSQL state record binds independent random state, nonce and
S256 PKCE verifier to a random HttpOnly/SameSite browser cookie. The GET callback
atomically consumes matching unexpired state before exchanging the code.
Replays, wrong browser cookies and missing state fail. No endpoint accepts an
arbitrary client-supplied ID token.

Discovery uses Google's fixed HTTPS issuer. The OpenID verifier checks RS256
signatures against rotating Google keys, issuer, audience and expiry. Additional
checks validate nonce, issued-at, authorized party, and `at_hash` when supplied.
A nonempty subject and verified email claim are required. Provider error details,
tokens, codes and secrets are not reflected in authentication errors. Access and
refresh tokens are not persisted or exposed to the browser.

New subjects create customer accounts with password login disabled and an
unguessable bcrypt hash retained for downgrade safety. Existing subjects use
their linked user ID even if Google's email changes; HALISI's email is not
overwritten. Gmail and matching Workspace-domain emails may be marked verified.
Other third-party Google emails receive HALISI's local verification email.

## Explicit account linking

Matching email addresses **never automatically link** an existing HALISI user.
Sign in to that account, verify its email, then choose **Connect Google** on
`/account`. `POST /api/auth/google/link` accepts `{}` and requires a session less
than ten minutes old. The flow binds the exact session hash and user ID. The
callback checks that session under a database lock, that the account remains
verified, and that the verified Google email matches. Logout, switched sessions,
expiry and stale authentication prevent completion. Unique constraints prevent
another subject from taking over a linked account. One Google identity is
supported per HALISI user. Password reset/account recovery is a separate feature;
Google email matching is not used to bypass HALISI account ownership.

## Database and rollback

Migration 3 adds `email_verified_at` and `password_login_enabled` to users,
`email_verification_tokens`, `google_accounts` keyed by immutable Google subject,
and `oauth_states`. Token/state tables have expiry indexes; user/session foreign
keys cascade on deletion. PKCE verifiers are stored server-side only for the
short flow lifetime; digest-only storage is used for session, verification,
state, nonce and browser-binding tokens. No provider access/refresh tokens are
retained. Schedule cleanup of expired/used verification tokens and OAuth states.

Down removes these tables, email-verification triggers and user columns. Back up
identity data before rollback: Google links and verification history are lost;
Google-only users cannot use a password after downgrade because its random value
is never known. Migrations 1 and 2 are unchanged.

## Tests

Run `sqlc generate`, `npm run build:css`, `go test ./...`, `go build ./...`, and
`go vet ./...`. `TEST_DATABASE_URL` enables isolated PostgreSQL verification and
Google-linking integration tests. Tests cover expiry/reuse/resend, SMTP rollback,
verification changes, stable-subject login, browser/state binding and replay,
and revoked/switched/stale session linking. Local RSA-signed ID tokens exercise
issuer, audience, signature, expiry, nonce, authorized-party and access-token-hash
checks. A local SMTP protocol server tests actual message construction/delivery.
Tests do not contact a live Google account or internet mailbox.
