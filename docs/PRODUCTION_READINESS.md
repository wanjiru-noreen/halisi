# Production Readiness Checks and Recovery Runbooks

This document outlines the mandatory production readiness checks, verification procedures, observability benchmarks, and incident recovery runbooks for Halisi before promoting changes to `main` and deploying to production.

---

## 1. Readiness Checklist Matrix

| Domain | Readiness Gate | Verification Mechanism | Status Requirement |
| --- | --- | --- | --- |
| **Migrations** | Forward migrations apply cleanly | `make migrate-up` / CI pipeline | Must apply without error; `schema_migrations` dirty flag must be `false`. |
| **Migrations** | Rollbacks drop schema in reverse order | `make migrate-down` / integration tests | Must reverse cleanly without orphan tables or foreign key errors. |
| **Database** | Pool connectivity and timeouts | `make migrate-ping` / `TestConnectTimeout` | Must connect within 5s; ping timeout enforced at 2s. |
| **Health Probes** | HTTP Liveness (`/health`) | `curl -f http://host/health` | Must return HTTP 200 with `{"status":"ok"}`. |
| **Health Probes** | Database Readiness (`/health/db`) | `curl -f http://host/health/db` | Must return HTTP 200 when healthy; HTTP 503 when down. |
| **Security** | Production Environment (`APP_ENV`) | `make check-config` | Must be strictly set to `production`. |
| **Security** | Public HTTPS Base URL | `make check-config` | Must be an HTTPS origin (`https://...`). HTTP loopback rejected. |
| **Security** | Security Headers (HSTS, CSP, Frames) | `make readiness-check` | HSTS on HTTPS, CSP default-src 'self', X-Frame-Options DENY, nosniff. |
| **Security** | Secure Session Cookies | Auth handler tests | Cookies must enforce `Secure: true`, `HttpOnly: true`, `SameSite: Lax`. |
| **Security** | Persistent Document Storage | `make check-config` | `VENDOR_DOCUMENT_DIR` must be an absolute path on `/var/data`. |
| **Observability** | Bounded Server Timeouts | Code review / server config | Header timeout 5s, Read/Write 30s, Idle 60s. |
| **Observability** | Credential Masking in Logs | Migration CLI / health tests | Driver connection errors must never expose passwords or tokens. |

---

## 2. Verification Commands and Automation

Run the following checks from the repository root prior to cutting a release:

```sh
# 1. Verify build artifacts compile matching Render's buildCommand
make build

# 2. Run automated production readiness integration suite
make readiness-check

# 3. Perform pre-flight configuration validation
make check-config

# 4. Check database connectivity
make migrate-ping

# 5. Check database schema version and dirty state
make migrate-version

# 6. Execute full test suite
go test -v ./...

# 7. Static analysis
go vet ./...
```

---

## 3. Observability and Runtime Boundaries

### HTTP Server Limits
Halisi configures explicit timeout boundaries in `cmd/server/main.go` to prevent slowloris attacks and connection resource starvation:

```go
&http.Server{
    Addr:              ":" + cfg.Port,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

### PostgreSQL Connection Pool Boundaries
Database connections are bounded via `pgxpool` configuration in `internal/database/postgres.go`:

```go
cfg.ConnConfig.ConnectTimeout = 5 * time.Second
cfg.MaxConns = 10
cfg.MinConns = 0
cfg.MinIdleConns = 0
cfg.MaxConnLifetime = time.Hour
cfg.MaxConnIdleTime = 5 * time.Minute
```

### Health Check Endpoints
- **`/health`**: Liveness probe polled every 10–30s by Render. If the web process hangs or panics, Render terminates the container and boots a replacement.
- **`/health/db`**: Deep database readiness probe. Performs `Ping` with an enforced 2-second timeout (`context.WithTimeout(ctx, 2*time.Second)`). Fails safe without leaking driver connection credentials.

---

## 4. Disaster Recovery Runbooks

### Runbook 1: Pre-Deploy Migration Failure
- **Symptom**: Deployment fails during pre-deploy step `./bin/migrate up`.
- **System Behavior**: Render aborts the deployment before traffic shifts. The previous release container continues serving live users without disruption.
- **Action Steps**:
  1. Inspect the pre-deploy log in Render Dashboard.
  2. If the error indicates a syntax or schema constraint issue:
     - Reproduce locally with a disposable database (`TEST_MIGRATION_DATABASE_URL`).
     - Fix the migration file or write a new forward migration.
     - Never edit already-applied migrations on production.
  3. If the schema was marked dirty (`dirty=true`):
     - Follow Runbook 2.

### Runbook 2: Reconciling Dirty Migration State
- **Symptom**: `make migrate-version` reports `version=X dirty=true`. Further migrations fail with `migration failed: dirty database`.
- **Cause**: A migration failed mid-execution or connection was severed during DDL.
- **Action Steps**:
  1. Connect to production database via management shell.
  2. Inspect the partially applied tables or constraints in the target schema.
  3. Manually clean up or complete the partial DDL operations in PostgreSQL.
  4. Use the migration tool or direct query on `schema_migrations` to clear the dirty flag:
     ```sql
     UPDATE schema_migrations SET dirty = false WHERE version = X;
     ```
  5. Run `make migrate-version` to confirm `dirty=false`.
  6. Re-run `make migrate-up` to ensure clean forward progression.

### Runbook 3: Emergency Database Restoration
- **Symptom**: Critical data corruption or catastrophic schema failure requiring full state restoration.
- **Action Steps (Automated Snapshot Restore)**:
  1. Open Render Dashboard and navigate to the managed PostgreSQL database (`halisi-postgres`).
  2. Click **Backups**.
  3. Select the latest healthy snapshot (Render retains automated daily snapshots for 7 days on Starter).
  4. Click **Restore Backup to New Database**.
  5. Update the web service's `DATABASE_URL` or database reference to point to the restored instance.
- **Action Steps (Logical Restore from pg_dump)**:
  1. Restore the custom-format dump into target database:
     ```sh
     pg_restore --clean --if-exists -d "$DATABASE_URL" halisi_backup_YYYYMMDD.dump
     ```
  2. Verify connectivity with `make migrate-ping`.
  3. Verify schema state with `make migrate-version`.

### Runbook 4: Emergency Application Rollback
- **Symptom**: Newly deployed application version introduces a critical runtime bug.
- **Action Steps**:
  1. In Render Dashboard, open the web service `halisi`.
  2. Navigate to **Deploys**.
  3. Locate the previous stable deployment and click **Rollback**.
  4. Render instantly re-activates the previous container image without rebuilding.
  5. If the new release applied a schema migration that is incompatible with the old code:
     - Run `./bin/migrate down` on a management shell to rollback one schema version.

### Runbook 5: Secret Leak or Compromised Credentials
- **Symptom**: An API key, database password, or secret was accidentally shared or compromised.
- **Action Steps**:
  1. In Render Dashboard, go to **Environment**.
  2. Update the affected secret variable with a newly generated key (e.g. `SMTP_PASSWORD`, `GOOGLE_CLIENT_SECRET`, `MPESA_PASSKEY`).
  3. For database credentials: rotate the password in PostgreSQL and update `DATABASE_URL`.
  4. Click **Save Changes**; Render performs a rolling redeployment with zero downtime.
  5. Invalidate the compromised token with the upstream provider.
