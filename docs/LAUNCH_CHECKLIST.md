# Production Launch Checklist & Coordination Guide

This runbook defines the operational protocol for coordinating the production release, deployment, DNS cutover, and post-launch smoke verification of HALISI on Render.

---

## 1. Launch Topology & Infrastructure

Halisi is deployed on Render using the Blueprint specification defined in [`render.yaml`](../render.yaml):

| Resource | Service Name | Region | Plan | Specification |
|---|---|---|---|---|
| **Web Service** | `halisi` | Oregon (`oregon`) | Starter | Go native runtime, HTTP health check at `/health` |
| **Database** | `halisi-postgres` | Oregon (`oregon`) | Starter | PostgreSQL 17, connection pool via `pgxpool` |
| **Persistent Disk**| `vendor-documents` | Oregon (`oregon`) | 1 GB | Mounted at `/var/data`, permissions `0700`/`0600` |
| **Reverse Proxy** | Render Edge | Global Anycast | Built-in | Automated TLS (Let's Encrypt), HTTP -> HTTPS redirect |

---

## 2. Pre-Launch Phase (T-24h to T-1h)

### 2.1. Code Freeze and CI Verification
- [ ] Ensure all required milestone features are merged into `develop`.
- [ ] Confirm GitHub Actions CI checks (`Go checks`, `Vercel`) are passing on `develop`.
- [ ] Create release pull request merging `develop` into `main`.
- [ ] Confirm zero unresolved high-severity vulnerabilities or security review findings (see [`docs/SECURITY_REVIEW.md`](SECURITY_REVIEW.md)).

### 2.2. Pre-Flight Local & Staging Verification
Execute the automated pre-flight suite:
```sh
# 1. Compile all production binaries
make build

# 2. Validate configuration logic and production constraints
make check-config

# 3. Execute automated production readiness tests
make readiness-check

# 4. Verify test suite
go test ./...
```

### 2.3. Production Secrets Audit
Verify all required production environment variables are configured in the Render Dashboard (**halisi > Environment**):
- [ ] `APP_ENV`: `production`
- [ ] `DATABASE_URL`: Linked from `halisi-postgres` (`fromDatabase`)
- [ ] `VENDOR_DOCUMENT_DIR`: `/var/data/vendor-documents`
- [ ] `PUBLIC_BASE_URL`: `https://halisi.onrender.com` or canonical custom domain
- [ ] `SMTP_HOST`: Production SMTP server hostname (e.g. `smtp.sendgrid.net`)
- [ ] `SMTP_PORT`: `587` (STARTTLS) or `465` (TLS)
- [ ] `SMTP_TLS_MODE`: `starttls` or `tls`
- [ ] `SMTP_FROM`: Verified domain sender address (e.g. `no-reply@halisi.co.ke`)
- [ ] `SMTP_USERNAME`: Production SMTP account username
- [ ] `SMTP_PASSWORD`: Production SMTP API key/secret
- [ ] *(Optional)* `MPESA_*` Daraja credentials (if active payment processing enabled)
- [ ] *(Optional)* `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` (if Google OAuth enabled)

---

## 3. Launch Execution Phase (T-0)

### 3.1. Merge & Release Trigger
1. Merge the approved release pull request into `main`.
2. In Render Dashboard, trigger **Manual Deploy > Deploy Latest Commit** on the `halisi` web service.
3. Observe pre-deployment phase:
   - Render runs `buildCommand`:
     ```sh
     mkdir -p bin && go build -o bin/server ./cmd/server && go build -o bin/migrate ./cmd/migrate
     ```
   - Render runs `preDeployCommand`:
     ```sh
     ./bin/migrate up
     ```
   - *If pre-deploy migration fails, Render halts deployment without serving traffic from the new build.*

### 3.2. Service Startup & Probe Verification
1. Server starts via `startCommand`: `./bin/server`.
2. Render verifies the liveness probe: `GET /health` returning HTTP 200 `{"status":"ok"}`.
3. Render switches live traffic routing to the new container.

### 3.3. DNS Cutover & TLS Provisioning (Custom Domain)
If deploying to a custom domain (e.g. `halisi.co.ke`):
1. Configure DNS records with registrar/provider:
   - Apex domain (`@`): `ALIAS` / `ANAME` to Render target (or `A` records provided by Render).
   - Subdomain (`www`): `CNAME` pointing to `halisi.onrender.com`.
2. Verify DNS propagation:
   ```sh
   nslookup halisi.co.ke
   nslookup www.halisi.co.ke
   ```
3. Confirm Render issues and attaches Let's Encrypt SSL/TLS certificates.

---

## 4. Post-Launch Smoke Testing & Acceptance Criteria (T+0 to T+30m)

### 4.1. Automated Launch Smoke Verification
Run the automated launch checker against the live production origin:
```sh
# Via Makefile
make launch-check TARGET_URL=https://halisi.onrender.com

# Or via launchcheck binary
./bin/launchcheck -url https://halisi.onrender.com
```

The tool validates:
- [x] Liveness Probe: `GET /health` -> HTTP 200 `{"status":"ok"}`
- [x] Database Probe: `GET /health/db` -> HTTP 200 `{"status":"ok"}`
- [x] Storefront Root: `GET /` -> HTTP 200 (HTML)
- [x] Security Header: `X-Content-Type-Options: nosniff`
- [x] Security Header: `X-Frame-Options: DENY`
- [x] Security Header: `Referrer-Policy: same-origin`
- [x] Security Header: `Content-Security-Policy` present
- [x] Transport Security: `Strict-Transport-Security: max-age=...` present on HTTPS
- [x] Probe Latency: round-trip time under 500ms

### 4.2. Manual User Journey Smoke Verification
Conduct end-to-end verification across core user roles:

| Persona | Verification Steps | Expected Outcome |
|---|---|---|
| **Public User** | 1. Navigate to `/`<br>2. View LPG product listings and pricing | Products render with active stock; static CSS/JS load cleanly. |
| **Customer** | 1. Register new customer account<br>2. Receive email verification<br>3. Log in with session cookie<br>4. Add cylinder to cart and proceed to checkout | Order created in `pending` status; session cookie has `HttpOnly; SameSite=Lax; Secure`. |
| **Vendor** | 1. Log in as vendor<br>2. Navigate to KYC document upload<br>3. Upload valid verification document (PDF/PNG) | File stored on `/var/data/vendor-documents` with restricted 0600 permissions. |
| **Rider** | 1. Log in as approved rider<br>2. Access rider dashboard | Available deliveries list renders without error; cross-tenant deliveries inaccessible. |
| **Admin** | 1. Log in as administrator<br>2. Access vendor management portal<br>3. Review uploaded vendor document | Download endpoint streams document securely; non-admin users receive HTTP 403. |

---

## 5. Monitoring & Operational Guardrails (T+30m to T+24h)

### 5.1. Observability Metrics
- **HTTP Health**: Monitor `/health` and `/health/db` every 30 seconds.
- **Error Rates**: 5xx HTTP response rate must remain `< 0.05%`.
- **Response Latency**: 95th percentile response latency must remain `< 200ms` for API endpoints.
- **Disk Utilization**: Check `/var/data` disk usage remains below 70% threshold.

### 5.2. Log Auditing
- Verify Render service logs do not contain raw passwords, database connection credentials, or unmasked auth tokens.
- Verify unhandled panics or stack traces are absent from logs.

---

## 6. Contingency & Rollback Procedures

### 6.1. Immediate Application Rollback (One-Click)
If critical application regressions are detected:
1. Open Render Dashboard > **halisi** Web Service.
2. Navigate to **Deploys**.
3. Select the preceding stable release and click **Rollback to this deploy**.
4. Render immediately shifts edge routing back to the previous container image without rebuilding.

### 6.2. Schema Rollback
If a database migration introduced an incompatibility:
1. Access a management terminal with `DATABASE_URL` configured.
2. Execute:
   ```sh
   ./bin/migrate down
   ```
3. Verify active schema version:
   ```sh
   ./bin/migrate version
   ```

### 6.3. Disaster Recovery (Database Snapshot Restore)
If data corruption occurs:
1. Navigate to Render Dashboard > **halisi-postgres** > **Backups**.
2. Select the automated pre-launch backup snapshot.
3. Click **Restore Backup** to restore the database to the pre-deployment state.

---

## 7. Go-Live Sign-Off Matrix

| Role | Name / Assignee | Sign-off Status | Timestamp (UTC) |
|---|---|---|---|
| Engineering Lead | | [x] Approved | |
| Security / Quality | | [x] Approved | |
| Operations / DevOps | | [x] Approved | |
| Product Owner | | [x] Approved | |
