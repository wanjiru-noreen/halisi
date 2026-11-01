# Halisi Setup Guide

## Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| Go | 1.26+ | Application runtime and build tool |
| PostgreSQL | 12+ | Database |
| sqlc | Latest | Generate type-safe Go code from SQL |
| Git | Any | Version control |

## Quick Start

### 1. Clone the Repository

```bash
git clone <repository-url>
cd halisi
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Set Up the Database

Create a PostgreSQL database for the application:

```bash
createdb halisi
```

Or using `psql`:

```sql
CREATE DATABASE halisi;
```

### 4. Configure Environment Variables

Create a `.env` file in the project root with the following variables:

```env
# Database
DATABASE_URL=postgres://user:password@localhost:5432/halisi?sslmode=disable

# Google OAuth2
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# SMTP Email (for verification emails)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@halisi.com

# Application URL (for verification email links)
APP_URL=http://localhost:8080
```

#### Google OAuth2 Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project or select an existing one.
3. Enable the **Google+ API**.
4. Create **OAuth 2.0 Client ID** credentials (type: Web application).
5. Add `http://localhost:8080/auth/google/callback` to the **Authorized redirect URIs**.
6. Copy the Client ID and Client Secret into your `.env` file.

#### SMTP Setup

For local development, you can use:
- **Gmail**: Use an [App Password](https://myaccount.google.com/apppasswords) (not your regular password).
- **Ethereal**: A fake SMTP service for testing — creates disposable email accounts.
- **Mailhog**: A local SMTP server that catches emails for inspection at `http://localhost:8025`.

### 5. Generate sqlc Code

If you modify SQL queries, regenerate the Go code:

```bash
sqlc generate
```

### 6. Run the Application

```bash
go run ./cmd/api
```

The server starts on `http://localhost:8080`. Tables are auto-created on first run.

### 7. Create an Admin User

The first admin user must be created directly in the database. There is no self-service admin registration:

```sql
-- Generate a bcrypt hash for the password (e.g., using Go or an online tool)
-- Example: password "admin123" → hash via bcrypt

INSERT INTO users (name, email, password, role, email_verified)
VALUES ('Admin', 'admin@halisi.com', '$2a$10$<bcrypt-hash>', 'admin', TRUE);
```

To generate a bcrypt hash from the command line:

```bash
# Using a small Go program
go run -exec '' /dev/stdin <<'EOF'
package main
import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("your-password"), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
EOF
```

## Project Structure Overview

```
halisi/
├── cmd/api/main.go          # Entry point
├── internal/                # Application logic
│   ├── database/            # DB connection and schema
│   ├── email/               # Email sending
│   ├── handlers/            # HTTP handlers
│   ├── middleware/          # Auth and role middleware
│   ├── models/              # Domain models
│   ├── repository/          # Data access layer
│   └── services/            # Business logic
├── web/                     # Frontend assets
│   ├── static/              # CSS, JS, images
│   └── templates/           # HTML templates
├── .env                     # Environment variables (not in git)
├── go.mod                   # Go dependencies
├── sqlc.yaml                # sqlc configuration
└── Dockerfile               # Docker build
```

## Running Tests

```bash
go test ./...
```

## Troubleshooting

### "DATABASE_URL environment variable is required"

Ensure your `.env` file exists and contains a valid `DATABASE_URL`.

### "Google OAuth credentials are not configured"

Check that `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` are set in your `.env`.

### "SMTP configuration is incomplete"

Email verification requires SMTP settings. For local development without email, you can temporarily bypass verification by setting `email_verified = TRUE` directly in the database.

### Port 8080 already in use

Stop the process on port 8080 or change the port in `cmd/api/main.go`:

```go
http.ListenAndServe(":9090", nil)
```
