# Halisi System Architecture

This document describes the architectural design, component layers, data flows, and infrastructure topology of HALISI.

---

## 1. System Overview

HALISI is an LPG (Liquefied Petroleum Gas) marketplace and delivery management platform connecting retail gas customers, verified gas vendors, delivery riders, and administrators in Kenya.

```
                                    +-----------------------------+
                                    |       Client Browsers       |
                                    |  (Customers, Vendors,       |
                                    |   Riders, Administrators)   |
                                    +--------------+--------------+
                                                   | HTTPS
                                                   v
+---------------------------------------------------------------------------------------------------+
| Render Edge / Anycast Reverse Proxy                                                               |
|  - SSL / TLS Termination                                                                          |
|  - Forwarding to internal container port ($PORT)                                                  |
+--------------------------------------------------+------------------------------------------------+
                                                   |
                                                   v
+---------------------------------------------------------------------------------------------------+
| Halisi Web Service (Go Runtime)                                                                   |
|                                                                                                   |
|  +---------------------------------------------------------------------------------------------+  |
|  | HTTP Middleware Pipeline                                                                    |  |
|  |  - Security Headers (HSTS, CSP, X-Frame-Options, X-Content-Type-Options)                      |  |
|  |  - Attempt Limiter / Brute-Force Defense                                                      |  |
|  |  - Session Authentication & Role Context Injection                                          |  |
|  +----------------------------------------------+----------------------------------------------+  |
|                                                 |                                                 |
|  +----------------------------------------------v----------------------------------------------+  |
|  | HTTP Route Handlers                                                                         |  |
|  |  - Storefront & Catalog      - Checkout & Orders            - Authentication (Local/OAuth)  |  |
|  |  - M-Pesa Daraja Payments    - Delivery & Rider Management  - Admin & KYC Approvals         |  |
|  +----------------------------------------------+----------------------------------------------+  |
|                                                 |                                                 |
|  +----------------------------------------------v----------------------------------------------+  |
|  | Domain Services Layer                                                                       |  |
|  |  - AuthService   - OrderService   - PaymentService   - DeliveryService   - StorageService   |  |
|  +-----------------------+-------------------------------+------------------------------+------+  |
|                          |                               |                              |         |
|                          v                               v                              v         |
|             +--------------------------+    +--------------------------+   +--------------------+ |
|             | Database Layer           |    | Persistent Storage       |   | External Services  | |
|             |  - pgxpool pool (max 10) |    |  - os.Root abstraction   |   |  - Safaricom Daraja| |
|             |  - sqlc type-safe queries|    |  - Mount: /var/data      |   |  - Google OAuth 2.0| |
|             |  - Parameterized SQL     |    |  - Restricted permissions|   |  - Outbound SMTP   | |
|             +------------+-------------+    +------------+-------------+   +--------------------+ |
+--------------------------|-------------------------------|----------------------------------------+
                           |                               |
                           v                               v
             +---------------------------+   +---------------------------+
             | Render Managed PostgreSQL |   | Attached Persistent Disk  |
             | Version: 17               |   | Size: 1 GB                |
             | Automated 7-Day Backups   |   | Path: /var/data           |
             +---------------------------+   +---------------------------+
```

---

## 2. Component Layers

### 2.1. Presentation & HTTP Routing Layer
- **Embedded Web Assets**: Static HTML templates and frontend assets (JavaScript, CSS) are compiled directly into the binary using Go's `//go:embed`.
- **Router**: Go standard library `http.ServeMux` extended with lightweight pattern routing and method dispatching.
- **Middleware**:
  - `SecurityHeaders`: Injects defensive headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: same-origin`, `Content-Security-Policy`, and conditional `Strict-Transport-Security`).
  - `AttemptLimiter`: In-memory sliding-window limiter guarding against brute-force authentication.
  - `RequireAuth` & `RequireRole`: Validates session cookie, queries session state, injects current user into request context, and enforces role boundaries (`Customer`, `Vendor`, `Rider`, `Admin`).

### 2.2. Domain Services Layer
- Encapsulates core business logic, status state transitions, and coordination between storage, database, and external integrations.
- Keeps HTTP handlers thin (handling only request decoding, input validation, service invocation, and response serialization).

### 2.3. Data Access Layer
- **Driver**: `jackc/pgx/v5` paired with `pgxpool.Pool` for thread-safe high-performance database connectivity.
- **Query Generation**: `sqlc` generates type-safe Go code from pure SQL queries, guaranteeing 100% parameterization and zero dynamic SQL string formatting.
- **Migrations**: `golang-migrate` manages forward and backward schema versions (`up` / `down`), executing automatically during pre-deployment.

### 2.4. File & Document Storage Layer
- Securely stores vendor business registration certificates and identification documents.
- Uses Go's `os.Root` capability to prevent path traversal outside the persistent disk mount (`/var/data/vendor-documents`).
- Files are assigned strict directory permissions (`0700`) and file permissions (`0600`).
- Documents are never directly exposed via web server static routes; all access is guarded by authenticated handlers verifying ownership or admin privileges.

---

## 3. Security Architecture

1. **Parameterization**: All database interaction utilizes positional parameters (`$1`, `$2`, etc.), mitigating SQL injection attacks.
2. **Session Security**: Session identifiers are generated using cryptographically secure random bytes (`crypto/rand`). Cookies use `HttpOnly`, `SameSite=Lax`, and `Secure` (in production).
3. **Role-Based Access Control (RBAC)**: All sensitive routes verify context roles. Customer, vendor, and rider resources enforce strict tenant ownership checks.
4. **Credential Isolation**: Database passwords and integration API keys are supplied through environment variables, masked in logs (`config.MaskDatabaseURL`), and never checked into source control.
5. **Payment Safety**: M-Pesa Daraja callbacks are authenticated using single-use callback verification tokens; transaction amounts are validated against database order totals.

---

## 4. Production Deployment Topology

Halisi is provisioned on Render as a coordinated multi-resource blueprint (`render.yaml`):
- **Web Service**: Starter plan running in the `oregon` region with `autoDeploy: false` to allow controlled, deliberate releases from `main`.
- **Database**: Managed PostgreSQL 17 starter instance with automated daily snapshots and point-in-time recovery.
- **Disk**: 1 GB attached persistent storage volume mounted at `/var/data`.
