# Production Deployment on Render

This guide outlines how Halisi is configured, built, and deployed to Render for production.

---

## 1. Architecture Overview

Halisi deploys to Render as a native Go web service paired with PostgreSQL and an attached persistent disk:

```
                      +---------------------------------------+
                      |          Client / Browser             |
                      +-------------------+-------------------+
                                          | HTTPS
                                          v
                      +---------------------------------------+
                      |     Render Edge / Reverse Proxy       |
                      |  - SSL/TLS Termination (HTTPS)        |
                      |  - Port forwarding to internal $PORT  |
                      +-------------------+-------------------+
                                          |
                                          v
+---------------------------------------------------------------------------------+
| Halisi Web Service (Render Starter Plan)                                       |
|                                                                                 |
|   +-----------------------+     Pre-deploy:      +--------------------------+  |
|   |      bin/migrate      | -------------------> |    PostgreSQL 17 DB      |  |
|   |  (Database Migrator)  |                      |  - Parameterized queries |  |
|   +-----------------------+                      |  - Connection pooling    |  |
|                                                  +--------------------------+  |
|   +-----------------------+                                  ^                  |
|   |      bin/server       | ---------------------------------+                  |
|   |   (HTTP Web Server)   |                                                     |
|   +-----------+-----------+                                                     |
|               |                                                                 |
|               | File I/O (os.Root)                                              |
|               v                                                                 |
|   +-----------------------+                                                     |
|   | Attached Persistent   |                                                     |
|   | Disk (/var/data)      |                                                     |
|   |  - KYC Verification   |                                                     |
|   |  - Mode 0700 / 0600   |                                                     |
|   +-----------------------+                                                     |
+---------------------------------------------------------------------------------+
```

Key architectural properties:
- **Zero credential leaks**: All secrets and database URLs are passed as environment variables and never logged or exposed in error messages.
- **Pre-deploy migrations**: Migrations run during pre-deployment before the new container receives traffic. If migrations fail, the deployment is aborted.
- **Embedded web assets**: HTML templates and static CSS/JS assets are compiled directly into `bin/server` via `//go:embed`.
- **Private persistent storage**: Vendor identity documents are written to an isolated persistent disk at `/var/data/vendor-documents` and accessed exclusively via `os.Root`.

---

## 2. Render Blueprint Specification (`render.yaml`)

The repository root includes `render.yaml`, defining the production web service and environment configuration:

```yaml
# Render Blueprint specification for Halisi production service.
# Provisioning requires an explicit Render blueprint deployment or service connection.
databases:
  - name: halisi-postgres
    databaseName: halisi
    user: halisi
    plan: 0.5c-1g
    region: oregon
    postgresMajorVersion: "17"

services:
  - type: web
    name: halisi
    runtime: go
    plan: starter
    region: oregon
    branch: main
    autoDeploy: false
    buildCommand: go build -o bin/server ./cmd/server && go build -o bin/migrate ./cmd/migrate
    preDeployCommand: ./bin/migrate up
    startCommand: ./bin/server
    healthCheckPath: /health
    disk:
      name: vendor-documents
      mountPath: /var/data
      sizeGB: 1
    envVars:
      - key: APP_ENV
        value: production
      - key: VENDOR_DOCUMENT_DIR
        value: /var/data/vendor-documents
      - key: DATABASE_URL
        fromDatabase:
          name: halisi-postgres
          property: connectionString
      - key: PUBLIC_BASE_URL
        sync: false
      - key: SMTP_HOST
        sync: false
      - key: SMTP_PORT
        value: "587"
      - key: SMTP_TLS_MODE
        value: starttls
      - key: SMTP_FROM
        sync: false
      - key: SMTP_USERNAME
        sync: false
      - key: SMTP_PASSWORD
        sync: false
      - key: GOOGLE_CLIENT_ID
        sync: false
      - key: GOOGLE_CLIENT_SECRET
        sync: false
      - key: MPESA_ENV
        sync: false
      - key: MPESA_CONSUMER_KEY
        sync: false
      - key: MPESA_CONSUMER_SECRET
        sync: false
      - key: MPESA_SHORT_CODE
        sync: false
      - key: MPESA_PASSKEY
        sync: false
      - key: MPESA_TRANSACTION_TYPE
        sync: false
```

### Blueprint Field Explanations

| Field | Value | Description |
| --- | --- | --- |
| `type` | `web` | Web service accessible over HTTP/HTTPS. |
| `name` | `halisi` | Identifier for the web service in Render. |
| `runtime` | `go` | Native Go runtime environment (reads `go.mod`). |
| `plan` | `starter` | Starter plan or higher is required to support persistent disk mounts. |
| `region` | `oregon` | Geographic hosting region for web service and co-located database. |
| `branch` | `main` | Production releases deploy from the `main` branch. |
| `autoDeploy` | `false` | Automatic deployment disabled; releases deployed after readiness checks. |
| `buildCommand` | `go build ...` | Compiles both `bin/server` and `bin/migrate`. |
| `preDeployCommand` | `./bin/migrate up` | Runs database migrations before traffic switch. |
| `startCommand` | `./bin/server` | Launches the production HTTP server. |
| `healthCheckPath` | `/health` | HTTP probe path for deployment health verification. |
| `disk` | `vendor-documents` | 1 GB persistent disk mounted at `/var/data`. |

---

## 3. Build & Deployment Lifecycle

Render follows a three-stage lifecycle for every deployment:

### Stage 1: Build Phase
```sh
go build -o bin/server ./cmd/server && go build -o bin/migrate ./cmd/migrate
```
- Compiles the Go server binary (`bin/server`) and migration utility (`bin/migrate`).
- Static assets and templates from `web/` are embedded into the binaries.
- Executables are placed into `./bin/` (which is excluded from Git via `.gitignore`).

### Stage 2: Pre-Deploy Phase
```sh
./bin/migrate up
```
- Connects to the database specified by `DATABASE_URL`.
- Applies any pending schema migrations up to the latest version.
- If migration fails:
  - Error logs are printed (with credentials masked).
  - The deployment stops immediately with a non-zero exit code.
  - The existing live instance continues serving traffic uninterrupted.

### Stage 3: Start & Health Check Phase
```sh
./bin/server
```
- Starts the HTTP web server listening on `PORT` (assigned by Render, defaults to 10000 on Render).
- **Persistent Disk Lifecycle Notice**: Because the service uses an attached persistent disk (`/var/data`), Render cannot run two instances concurrently with the disk attached. During redeployments, Render stops the active instance before detaching the disk and mounting it to the new instance, causing a brief maintenance downtime during deployment rather than a zero-downtime traffic shift.
- Render polls `GET /health` until it returns HTTP `200 OK` before marking the new deployment healthy.

---

## 4. Health Checks and Monitoring

Halisi exposes two dedicated health check endpoints:

### Liveness Probe (`GET /health`)
- **Purpose**: Fast HTTP liveness check used by Render to verify the process is accepting requests.
- **Status Code**: `200 OK`
- **Response**: `{"status":"ok"}`
- **Render Configuration**: Set as `healthCheckPath: /health` in `render.yaml`.

### Readiness Probe (`GET /health/db`)
- **Purpose**: Deep database connectivity verification.
- **Behavior**: Pings PostgreSQL with an enforced 2-second timeout.
- **Status Codes**:
  - `200 OK` with `{"status":"ok"}` when the connection pool is healthy.
  - `503 Service Unavailable` with `{"status":"unavailable"}` when PostgreSQL is unreachable or ping times out.
- **Security**: Never exposes database error messages or credentials in the response body.

---

## 5. Persistent Storage for Vendor KYC Documents

Halisi securely stores vendor registration documents (certificates of incorporation, licenses, tax PINs):

- **Disk Mount**: Mounted at `/var/data` as configured in `render.yaml`.
- **Directory Path**: Set via `VENDOR_DOCUMENT_DIR=/var/data/vendor-documents`.
- **Validation**: In `APP_ENV=production`, the application enforces that `VENDOR_DOCUMENT_DIR` is an absolute path (`filepath.IsAbs`). Operators must ensure this path resides on the configured persistent disk mount (`/var/data`) so uploaded documents persist across service deployments.
- **Security Model**:
  - Files are given cryptographically random 256-bit filenames.
  - Directory permissions are restricted to `0700` and file permissions to `0600`.
  - Storage is isolated outside public HTTP routes using Go's `os.Root` jailed filesystem primitives.
  - Documents are accessible only to verified administrators via authenticated session-guarded handlers.
- **Single Instance Constraint**: Render persistent disks attach to a single service instance. If scaling horizontally beyond one instance in the future, migrate to shared private object storage (e.g. S3 / GCS).

---

## 6. Domain, HTTPS, and Secrets Configuration

### Custom Domain and DNS Configuration
Halisi can be served directly from Render's default domain (`https://halisi.onrender.com`) or from custom domains (such as `https://halisi.co.ke` and `https://www.halisi.co.ke`).

1. **Adding Domains in Render**:
   - In the Render Dashboard, navigate to **Settings > Custom Domains**.
   - Add both the apex domain `halisi.co.ke` and the `www.halisi.co.ke` subdomain.
2. **Configuring DNS Records**:
   - **Apex Domain (`halisi.co.ke`)**:
     - Option A (Recommended): Create an **ALIAS**, **ANAME**, or flattened CNAME record pointing to `halisi.onrender.com`.
     - Option B: Create an **A** record pointing to Render's anycast IPv4 address: `216.24.57.1`.
   - **Subdomain (`www.halisi.co.ke`)**:
     - Create a **CNAME** record pointing to `halisi.onrender.com`.
3. **Automated SSL/TLS Certificates**:
   - Render automatically provisions and renews free SSL/TLS certificates via Let's Encrypt.
   - HTTP traffic is automatically upgraded to HTTPS at the Render edge proxy.
   - Halisi security middleware enforces HTTP Strict Transport Security (`Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`).

### Aligning `PUBLIC_BASE_URL`
The environment variable `PUBLIC_BASE_URL` must match the public HTTPS origin used by clients (e.g. `https://halisi.co.ke` or `https://halisi.onrender.com`). It directly configures:
- **Email Verification**: Generates valid links such as `${PUBLIC_BASE_URL}/verify-email#token=...`.
- **Google OAuth**: Must strictly match the authorized redirect URI registered in Google Cloud Console (`${PUBLIC_BASE_URL}/api/auth/google/callback`).
- **M-Pesa STK Push**: Must provide the public callback URL for Safaricom Daraja notifications (`${PUBLIC_BASE_URL}/api/payments/mpesa/callback?token=...`).

### Production Secrets and Credential Hygiene
All sensitive production settings are declared with `sync: false` in `render.yaml`. This ensures that real credentials are never stored or exposed in Git.

- **Configuring Secrets**:
  - In Render Dashboard under **Environment**, set each variable (`SMTP_PASSWORD`, `GOOGLE_CLIENT_SECRET`, `MPESA_PASSKEY`, etc.).
  - Database credentials for the managed database (`halisi-postgres`) are automatically managed and injected via `fromDatabase`.
- **Rotating Secrets**:
  1. Update credentials with the upstream provider (e.g., generate a new SendGrid API key, Google OAuth Secret, or Daraja Passkey).
  2. Update the environment variable in Render Dashboard.
  3. Trigger a redeployment. Render applies the new secrets with zero downtime.
- **Pre-Flight Configuration Verification**:
  - Validate all production configuration settings without starting network servers:
    ```sh
    make check-config
    # Or directly:
    go run ./cmd/server --check-config
    ```

### Environment Variables Reference

| Variable | Required | Production Value | Description |
| --- | --- | --- | --- |
| `APP_ENV` | Yes | `production` | Enables production security rules (HTTPS-only cookies, strict TLS, absolute storage path). |
| `PORT` | Auto | (Render assigns e.g. `10000`) | Listening port for the HTTP server. |
| `DATABASE_URL` | Auto | Managed via `fromDatabase` | PostgreSQL connection URL with TLS enabled. |
| `PUBLIC_BASE_URL` | Yes | `https://halisi.onrender.com` or custom domain | Canonical public HTTPS origin for OAuth redirects and payment webhooks. |
| `VENDOR_DOCUMENT_DIR` | Yes | `/var/data/vendor-documents` | Absolute directory path on the attached persistent disk. |

### Email & Verification (SMTP)
| Variable | Required | Example | Description |
| --- | --- | --- | --- |
| `SMTP_HOST` | Yes | `smtp.sendgrid.net` | Outbound SMTP hostname. Required in production. |
| `SMTP_PORT` | Yes | `587` | Outbound SMTP port (`587` for STARTTLS, `465` for TLS). |
| `SMTP_TLS_MODE` | Yes | `starttls` | Encryption mode: `starttls` or `tls`. `none` is rejected in production. |
| `SMTP_FROM` | Yes | `no-reply@halisi.co.ke` | Verified sender address. |
| `SMTP_USERNAME` | Yes | `apikey` | SMTP authentication username. |
| `SMTP_PASSWORD` | Yes | `[secret]` | SMTP authentication password or API key. |

### Optional Integrations
| Variable | Required | Description |
| --- | --- | --- |
| `GOOGLE_CLIENT_ID` | Optional | Google OAuth 2.0 Web Client ID. Leave empty if disabled. |
| `GOOGLE_CLIENT_SECRET` | Optional | Google OAuth 2.0 Web Client Secret. |
| `MPESA_ENV` | Optional | `production` or `sandbox`. Leave empty if payments disabled. |
| `MPESA_CONSUMER_KEY` | Optional | Safaricom Daraja Consumer Key. |
| `MPESA_CONSUMER_SECRET` | Optional | Safaricom Daraja Consumer Secret. |
| `MPESA_SHORT_CODE` | Optional | Business shortcode (PayBill or Till). |
| `MPESA_PASSKEY` | Optional | Daraja STK Push passkey. |
| `MPESA_TRANSACTION_TYPE` | Optional | `CustomerPayBillOnline` or `CustomerBuyGoodsOnline`. |

---

## 7. Local Build & Verification

Before pushing configuration changes or initiating a deployment, verify that the binaries build, database connectivity checks pass, readiness checks pass, and tests succeed:

```sh
# Build binaries matching Render build command
make build

# Verify database connectivity without executing migrations
make migrate-ping

# Execute automated production readiness check suite
make readiness-check

# Run all test suites
go test ./...

# Verify code analysis
go vet ./...
```

See [production readiness](PRODUCTION_READINESS.md) for the complete checklist, timeout boundaries, and recovery runbooks.

---

## 8. Rollback and Disaster Recovery

### Pre-Deploy Migration Abort
If `./bin/migrate up` encounters a failing migration during pre-deploy, Render halts deployment. The running application remains on the previous release.

### Database Backups and Restoration
Render managed PostgreSQL Starter tier includes automated daily backups with 7-day retention.
1. **Automated Daily Backups**: Managed by Render with point-in-time recovery window (e.g. 3 days for Hobby workspaces, extending up to 7–30 days on paid team plans).
2. **On-Demand Snapshots**: In Render Dashboard under **PostgreSQL > Backups**, create a manual backup immediately before applying new schema versions.
3. **Offsite Logical Backups**:
   ```sh
   pg_dump "$DATABASE_URL" -Fc -f halisi_backup_$(date +%Y%m%d).dump
   ```
4. **Database Restoration**:
   To restore from a logical backup into an isolated staging/recovery database:
   > **Warning**: `--clean` drops existing tables and schema objects before recreating them. Never run this directly against an active production database while it is serving live traffic. Always restore into an isolated recovery instance first (e.g. `$TARGET_DATABASE_URL`) to validate data integrity before cutting over.
   ```sh
   pg_restore --clean --if-exists -d "$TARGET_DATABASE_URL" halisi_backup_YYYYMMDD.dump
   ```

### Manual Schema Rollback
To revert the last applied migration:
```sh
# On a management shell with DATABASE_URL configured:
./bin/migrate down
```

To verify the current database migration version:
```sh
./bin/migrate version
```

### Application Rollback
If a deployed application release experiences unforeseen issues after routing traffic:
1. Open the service dashboard in Render.
2. Navigate to **Deploys**.
3. Select the previous successful deployment and click **Rollback**.
4. Render immediately redeploys the previous release without rebuilding.
