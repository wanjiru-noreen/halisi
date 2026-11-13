# User authentication

Apply migrations before starting the server (`make migrate-up` or
`go run ./cmd/migrate up`), then `go run ./cmd/server`.
Visit `/register`, `/login`, and the protected `/account` page.

## API

Requests use `Content-Type: application/json`. Unknown fields, multiple JSON
values, and bodies over 4 KiB are rejected. Responses contain a public `user`
object (`id`, `name`, `email`, `phone`, `role`) or an `error` message. Password
hashes, session hashes, and raw session tokens are never returned in JSON.

| Endpoint | Input | Result |
| --- | --- | --- |
| `POST /api/auth/register` | `name`, `email`, `phone`, `password` | 201; creates customer account; 409 for duplicate email; 422 for invalid values |
| `POST /api/auth/login` | `email`, `password` | 200 and session cookie; 401 with the same message for missing account or wrong password |
| `GET /api/auth/me` | Session cookie | 200 public user or 401 |
| `POST /api/auth/logout` | `{}` and session cookie | 204; revokes session and expires cookie; safe to repeat |

Registration trims names/phones and lowercases/trims email. Names require 2–100
characters and no control characters. Emails must parse as a bare email address
and contain at most 254 bytes. Phones require international format such as
`+254712345678`. Passwords are not trimmed: use at least 12 Unicode characters,
with at most 72 UTF-8 bytes (bcrypt's limit). Bcrypt cost is 12 with a fresh salt.
Public registration always creates role `customer`; clients cannot select a
privileged role. The case-insensitive database unique index handles concurrent
duplicate registrations. Registration does not automatically log in.

## Sessions and browser security

Login generates a fresh 256-bit cryptographically random token. The cookie
contains its base64url value; PostgreSQL stores only its SHA-256 digest. Sessions
expire after 24 hours. Authentication checks expiry and revocation in PostgreSQL
on every protected request, so sessions survive process restarts. User data is
read from the current user record. Login revokes the previously presented session
and creates the new one in a single transaction. It does not log out other devices.
Logout invalidates the database session before clearing the cookie. Database
failures return 503 and do not pretend logout succeeded.

Cookies are host-only, `Path=/`, `HttpOnly`, and `SameSite=Lax`, with matching
expiry and Max-Age. Set **`APP_ENV=production` and serve the entire application
over HTTPS** for deployment: production uses a `Secure`, `__Host-halisi_session`
cookie. Local development/test uses `halisi_session` without Secure so that
`http://localhost:8080` works. Never deploy using the development cookie settings.

All routes pass through Go's `http.CrossOriginProtection`. State changes are
POST-only and require JSON; cross-origin browser submissions are rejected.
No CORS allowlist is enabled. The embedded pages use same-origin requests and
never store tokens in localStorage. Content Security Policy allows only local
assets; framing is denied, and responses containing user information are not
cached. Protected browser page requests redirect to `/login`; API requests get
401. Static assets and login/registration pages remain public.

## Rate limiting

Login permits 10 attempts per peer IP and 5 per normalized email per 15-minute
fixed window, including successful attempts. Registration permits 10 attempts
per peer IP in the same window. Exceeding a limit returns 429 and `Retry-After`.
Malformed login bodies still consume the IP budget. The concurrency-safe limiter
holds at most 10,000 keys and rejects new keys at capacity until entries expire.
Missing accounts perform a dummy bcrypt comparison to reduce timing differences.

Limits are held per process and reset on restart. For multiple application
instances, add a shared limiter or enforce equivalent limits at the ingress.
The application uses the TCP peer address and deliberately ignores untrusted
`X-Forwarded-For`. Behind a reverse proxy this groups requests by proxy IP; use
ingress rate limiting or explicitly implement trusted proxy handling before a
multi-user deployment behind that proxy.

Expired/revoked sessions cannot authenticate. Schedule removal of expired sessions
as an operational retention task; no background deletion job is installed here.
Email verification, password reset, MFA, and privileged-user provisioning are
separate workflows and are not implemented by these endpoints.

## Frontend assets

HTML pages, JavaScript, and generated Tailwind CSS are embedded in the Go binary.
They work without a CDN or a particular runtime working directory. After editing
HTML or `web/static/css/auth.input.css`, regenerate the stylesheet:

```sh
npm ci
npm run build:css
```

Use `npm.cmd` in PowerShell if script execution policy blocks `npm.ps1`.
Commit the generated `web/static/css/auth.css` alongside the sources. After SQL
changes, run `sqlc generate` and commit its output too.

## Verification

```sh
go test ./...
go build ./...
go vet ./...
```

Set `TEST_DATABASE_URL` to a disposable PostgreSQL database to run the complete
HTTPS authentication integration test. It creates an isolated schema, applies
migrations, tests registration/duplicate rejection, bcrypt storage, invalid
credentials, secure cookies, protected routes, session rotation/restart/expiry,
and logout invalidation, then removes that schema. Existing CI already sets this
variable. Unit tests cover validation, CSRF/JSON rejection, pages/assets and
concurrent rate limiting without PostgreSQL.
