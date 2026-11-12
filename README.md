# halisi
LPG delivery and vendor management platform built with Go, PostgreSQL, HTML, JavaScript, and Tailwind CSS.

## Run the HTTP server

Install the Go version specified in `go.mod`, then run:

```sh
go run ./cmd/server
```

The server defaults to port `8080`. Set the `PORT` environment variable to use
another port (for example, `$env:PORT = "9090"` in PowerShell or
`export PORT=9090` in a POSIX shell), then run the same command.

`GET http://localhost:8080/health` returns HTTP 200 with
`Content-Type: application/json` and body `{"status":"ok"}`.

## Build and test

```sh
go build ./...
go test ./...
```
