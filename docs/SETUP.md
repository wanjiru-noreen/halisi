# Halisi Local Development & Setup Guide

## 1. Prerequisites

Before setting up Halisi, ensure your workstation has the following tools installed:

| Tool | Version | Purpose | Installation Guide |
|---|---|---|---|
| **Go** | `1.26+` | Application compiler and runtime | [golang.org/dl](https://golang.org/dl/) |
| **Docker & Docker Compose** | Latest | Container runtime for PostgreSQL & Mailpit | [docker.com](https://www.docker.com/) |
| **templ CLI** | `v0.3.1100+` | Type-safe HTML component compiler | `go install github.com/a-h/templ/cmd/templ@latest` |
| **sqlc CLI** | `v1.26+` | Compile-time SQL-to-Go query generator | `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` |
| **Git** | `2.40+` | Source control management | [git-scm.com](https://git-scm.com/) |

Verify your installation:
```bash
go version
docker compose version
templ version
sqlc version
```

---

## 2. Quickstart Setup (Under 3 Minutes)

### Step 1: Clone Repository & Create Environment File
```bash
git clone https://github.com/wanjiru-noreen/halisi.git
cd halisi
cp .env.example .env
```

### Step 2: Start PostgreSQL & Mailpit via Docker Compose
Halisi includes a turnkey local cluster with PostgreSQL 16 and Mailpit (local SMTP mail catcher):

```bash
docker compose up -d postgres mailpit
```
* **PostgreSQL**: Bound to `localhost:5432` (`user: postgres`, `password: postgres`, `db: halisi`).
* **Mailpit SMTP**: Bound to `localhost:1025` (handles transactional emails).
* **Mailpit Web UI**: Open `http://localhost:8025` in your browser to inspect verification and password reset emails instantly.

### Step 3: Install Go Module Dependencies
```bash
go mod download
```

### Step 4: Generate Code (Templ Components & sqlc Queries)
```bash
# Compile .templ files into type-safe Go components
templ generate

# Compile internal/database/queries into internal/database/queried
sqlc generate
```

### Step 5: Start Local Server
```bash
go run ./cmd/api
```
The server will boot on `http://localhost:8080`.

---

## 3. Environment Variables Configuration (`.env`)

```ini
# Application Server
PORT=8080
APP_ENV=development
APP_URL=http://localhost:8080
SESSION_KEY=32-character-random-secret-key-goes-here

# PostgreSQL Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/halisi?sslmode=disable

# Google OAuth2 (Optional for offline dev)
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Google Maps API (For Geocoding & Interactive Shop Map)
GOOGLE_MAPS_API_KEY=AIzaSyYourGoogleMapsApiKey

# Safaricom M-Pesa Daraja (Sandbox)
MPESA_ENVIRONMENT=sandbox
MPESA_CONSUMER_KEY=your_sandbox_consumer_key
MPESA_CONSUMER_SECRET=your_sandbox_consumer_secret
MPESA_PASSKEY=bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919
MPESA_SHORTCODE=174379
MPESA_CALLBACK_URL=https://your-ngrok-or-localtunnel-url.loca.lt/api/payments/mpesa/callback

# SMTP Email Configuration (Mailpit for local dev)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=no-reply@halisi.co.ke

# Object Storage (Local disk for dev, AWS S3 or Cloudflare R2 for prod)
STORAGE_PROVIDER=local
S3_BUCKET=halisi-documents
S3_REGION=eu-west-1
S3_ACCESS_KEY=
S3_SECRET_KEY=
```

---

## 4. Setting Up Test Accounts & Admin User

### 1. Register a Normal Account
1. Open `http://localhost:8080/register` in your browser.
2. Sign up as a `customer` or `owner` (vendor).
3. Open the Mailpit Web UI at `http://localhost:8025`.
4. Click the verification link sent in the confirmation email to activate your account.

### 2. Promote User to Platform Administrator
To test admin governance and KYC approval features, promote an existing verified user via `psql`:

```bash
docker compose exec postgres psql -U postgres -d halisi -c "UPDATE users SET role = 'admin' WHERE email = 'your-email@example.com';"
```

### 3. Promote User to Delivery Rider
```bash
docker compose exec postgres psql -U postgres -d halisi -c "UPDATE users SET role = 'rider' WHERE email = 'rider@example.com';"
```

---

## 5. Development Workflow & Live Reload

### Watch Mode with `templ` and Air
For automatic template recompilation on file changes:

```bash
# Watch templ components in one terminal
templ generate --watch

# In a second terminal, run air or go run
go run ./cmd/api
```

---

## 6. Running Tests & Quality Checks

Run the automated test suite and linter before committing changes:

```bash
# Run unit and repository tests with race detector
make test

# Run formatting, dependency verification, and vet analysis
make quality

# Run vulnerability check across all Go dependencies
make security
```
