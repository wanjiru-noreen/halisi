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

## Build and test

```sh
go build ./...
go test ./...
```
