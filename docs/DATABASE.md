# Database migrations

HALISI uses golang-migrate v4 with its pgx/v5 driver. The version is pinned in
`go.mod`; `go mod download` installs the dependencies. The Go migration command
embeds SQL from `internal/database/migrations/`, so no global migrate CLI is
required and compiled binaries do not depend on the working directory.

## Configuration and commands

Set `DATABASE_URL` in the process environment using the database's actual
credentials. See `.env.example` for the URL format. `.env` is not automatically
loaded. The command uses `internal/config` and its validation; `PORT` and
`APP_ENV` retain their defaults. Use a database account allowed to create and
drop tables in the target schema. TLS and other connection options come from
the URL. Connection establishment defaults to a five-second timeout.

Run from the repository root:

```sh
go mod download
make migrate-up
make migrate-version
make migrate-down
```

Without Make (including PowerShell):

```sh
go run ./cmd/migrate up
go run ./cmd/migrate version
go run ./cmd/migrate down
```

`up` applies all pending versions. Repeating it succeeds without reapplying
existing migrations. `down` rolls back exactly one version and deletes the
tables/data removed by that version. Back up any data you need before rollback.
The HTTP server does not automatically apply migrations; run them explicitly
before deploying code that needs the schema.

## Initial schema

Version `000001` creates users, vendors, products, orders, order_items,
payments and deliveries. IDs are generated bigint identities. Foreign keys
preserve references; prices and stock cannot be negative and item quantities
and payment amounts must be positive. Monetary values use `NUMERIC(12,2)`.
The migration contains no accounts, credentials or seed data. Role/status
values and business workflows remain application responsibilities.

The up and down SQL each run in a transaction. Down removes tables in reverse
dependency order without CASCADE. The `schema_migrations` table records the
current version and dirty flag; it remains after rollback. Advisory locking
prevents concurrent migration execution. Do not edit a migration after it has
been applied: add a new numbered pair, e.g. `000002_description.up.sql` and
`000002_description.down.sql`.

On failure the command exits nonzero without printing driver errors that could
contain credentials. Check `migrate-version` and database logs. A dirty version
blocks further migrations: investigate and repair the failed migration before
using golang-migrate's administrative force command to reconcile the recorded
version. Force changes tracking only; it does not execute or undo SQL.

## Verification

```sh
go test ./...
go build ./...
```

Set `TEST_MIGRATION_DATABASE_URL` to a disposable PostgreSQL database to enable
the migration integration test. It creates and removes its own isolated schema,
verifies version tracking and repeat-up behavior, then tests rollback and reapply.
Without that variable the live migration test is skipped.

## Generate Go queries with sqlc

Use sqlc **v1.31.1** for reproducible output. Install the CLI with:

```sh
make sqlc-install
# Or without Make:
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
```

Add the Go binary directory (`go env GOBIN`, or `go env GOPATH` plus `/bin` when
GOBIN is empty) to PATH. On Windows this is typically `%USERPROFILE%\go\bin`.
Confirm `sqlc version` prints `v1.31.1`.

`sqlc.yaml` configures PostgreSQL and `pgx/v5`. It reads schema migrations from
`internal/database/migrations/` and named SQL queries from
`internal/database/queries/`. sqlc understands the golang-migrate filename
convention and ignores down migrations. Generation needs no running database,
credentials, or `DATABASE_URL`.

After adding a migration or changing queries, run from the repository root:

```sh
make sqlc-generate
# Or without Make:
sqlc generate
go test ./...
go build ./...
```

Commit the SQL sources, `sqlc.yaml`, and generated files together. Go code in
`internal/database/sqlc/` is generated; change SQL and regenerate instead of
editing it by hand. To check reproducibility, regenerate and confirm there is
no diff in that directory.

Initial queries provide create, lookup by ID/email, paginated listing, profile
update and delete for users. Values use PostgreSQL `$1`, `$2`, etc. parameters.
Profile updates leave password hashes and roles unchanged. List results omit
password hashes. Lookup/create/update results are internal records that contain
password hashes: map them to response DTOs before exposing them via HTTP.
Delete returns the affected row count and obeys existing foreign keys.

Use the generated package with the application's pool:

```go
import dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"

queries := dbsqlc.New(pool)
user, err := queries.GetUser(ctx, userID)
```

The generated `DBTX` interface accepts a pgx pool, connection, or transaction;
`queries.WithTx(tx)` binds queries to a transaction. Missing single-row results
return `pgx.ErrNoRows`. Nullable columns use pgx types such as `pgtype.Text`.
