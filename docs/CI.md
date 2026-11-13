# Continuous integration

`.github/workflows/ci.yml` runs on pull requests targeting `develop` or `main`,
and on pushes to either branch. Each run builds the Go packages, runs all tests
with PostgreSQL, runs `go vet`, and regenerates sqlc output to detect drift.
The required GitHub check is named **Go checks**; keep that name in sync with
branch protection settings when changing the workflow.

The PostgreSQL 17 service is disposable. Its username/password are test-only
values, not deployment credentials. A dynamically allocated port is used after
the service passes its readiness check. `TEST_DATABASE_URL` enables connection
tests; `TEST_MIGRATION_DATABASE_URL` enables migration/rollback tests. CI uses no
production database or repository secrets, so fork pull requests can run safely
with a read-only GitHub token.

Go follows `go.mod`; sqlc is pinned to v1.31.1. Regeneration checks tracked
modifications, deletions and untracked files under `internal/database/sqlc/`.
To fix a drift failure, run `sqlc generate` and commit the resulting files.

## Local checks

Set `TEST_DATABASE_URL` and `TEST_MIGRATION_DATABASE_URL` to a disposable
PostgreSQL instance, then run:

```sh
sqlc generate
git status --short --untracked-files=all -- internal/database/sqlc
go build ./...
go test ./... -count=1
go vet ./...
```

Generated output should have no diff. Without the database variables, live
integration tests are skipped locally; CI always sets them.

## Branch protection

Repository settings must protect both `main` and `develop` independently of the
workflow file. Require pull requests and the **Go checks** check from GitHub
Actions, require the branch to be up to date, enforce rules for administrators,
and disallow force pushes and deletion. No additional reviewer approval is
required by this setup. These rules restrict direct pushes and prevent merging
failed or pending checks. Administrators must preserve the rules when editing
repository settings.

Workflow changes are validated in a pull request before merging. A new required
check can remain pending on older pull requests until their branches include
the workflow and GitHub runs it.
