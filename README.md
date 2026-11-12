# halisi
LPG delivery and vendor management platform built with Go, PostgreSQL, HTML, JavaScript, and Tailwind CSS.

## Run the HTTP server

Install the Go version specified in `go.mod`. Copy `.env.example` to `.env` as a
local reference and replace the placeholders. `.env` is ignored by Git.
The server reads the process environment; it does not automatically load `.env`.
Export the settings in your shell or configure your deployment to inject them.
For example, in PowerShell:

```powershell
$env:DATABASE_URL = "postgres://YOUR_USERNAME:YOUR_PASSWORD@localhost:5432/YOUR_DATABASE"
$env:APP_ENV = "development"
```

Then run:

```sh
go run ./cmd/server
```

The server defaults to port `8080`. Set the `PORT` environment variable to use
another port (for example, `$env:PORT = "9090"` in PowerShell or
`export PORT=9090` in a POSIX shell), then run the same command.

`GET http://localhost:8080/health` returns HTTP 200 with
`Content-Type: application/json` and body `{"status":"ok"}`.

Configuration is validated before the server starts:

| Variable | Default | Validation |
| --- | --- | --- |
| `PORT` | `8080` | Integer from 1 to 65535 |
| `DATABASE_URL` | Required | `postgres://` or `postgresql://` URL with a host |
| `APP_ENV` | `development` | `development`, `test`, or `production` |

Missing or invalid settings stop startup with an error naming the variable.
Configuration errors do not include database credentials. Keep real credentials
in your local environment or deployment secret store.

## PostgreSQL

Start PostgreSQL and set `DATABASE_URL` to its connection URL before running the
server. Startup creates a `pgxpool` pool and calls `Ping`; a failed connection
stops startup with a credential-safe error. No schema or migrations are applied.
TLS settings are supplied through `DATABASE_URL` according to your deployment.

The pool allows up to 10 connections, retires connections after one hour, and
closes idle connections after five minutes. Initial connectivity has a five-second
timeout. On Ctrl+C or SIGTERM the server drains HTTP requests for up to ten
seconds, then closes the pool.

`GET /health` checks the HTTP server. `GET /health/db` performs a database ping
with a two-second timeout and returns HTTP 200 with `{"status":"ok"}` or HTTP 503
with `{"status":"unavailable"}`. Driver errors and credentials are not returned.

Database integration tests require a running PostgreSQL instance. Set
`TEST_DATABASE_URL` to its connection URL, then run `go test ./...`. Without that
variable, only the live PostgreSQL integration test is skipped; failure, timeout,
configuration and HTTP health tests still run. The integration test only pings
the database and checks pool closure; it does not modify data.

## Build and test

See [database migrations](docs/DATABASE.md) for schema setup and rollback commands.
The same guide documents [sqlc generation](docs/DATABASE.md#generate-go-queries-with-sqlc).
See [CI and branch protection](docs/CI.md) for automated pull request checks.
See [authentication](docs/AUTH.md) for registration, login, sessions and frontend builds.

```sh
go build ./...
go test ./...
```
