# Halisi Architecture

## Overview

Halisi is a web application for managing cooking gas cylinder orders. It connects customers with verified gas shop owners, enabling customers to browse shops, place orders, and track deliveries, while shop owners manage their inventory and orders, and admins verify shop KYC (Know Your Customer) documents.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26 |
| Database | PostgreSQL (via `lib/pq` driver) |
| ORM/Query Layer | sqlc (code generation from SQL) |
| Session Management | Gorilla Sessions (cookie-based) |
| Authentication | Google OAuth2 + OpenID Connect (go-oidc) |
| Password Hashing | bcrypt (golang.org/x/crypto) |
| Email | SMTP (stdlib `net/smtp`) |
| Templating | Go `html/template` (server-side rendering) |
| Deployment | Docker, Railway (nixpacks) |

## Project Structure

```
halisi/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point; wiring and route registration
├── internal/
│   ├── database/
│   │   ├── database.go          # DB connection and auto-migration (CREATE TABLE IF NOT EXISTS)
│   │   ├── schema.sql           # Canonical SQL schema (source for sqlc)
│   │   └── queried/             # sqlc-generated query code
│   │       ├── models.go        # Generated DB models
│   │       ├── user_queries.go
│   │       ├── shop_queries.go
│   │       ├── order_queries.go
│   │       └── kyc_queries.go
│   ├── email/
│   │   └── email.go             # SMTP email sender (verification emails)
│   ├── handlers/
│   │   ├── handlers.go          # HTTP handlers for all routes
│   │   ├── google_auth.go       # Google OAuth2 login and callback
│   │   └── kyc_document_handler.go # KYC document upload and review handlers
│   ├── middleware/
│   │   ├── auth.go              # RequireAuth middleware
│   │   ├── role.go              # RequireRole middleware (RBAC)
│   │   ├── session.go           # Gorilla session store
│   │   └── session_helper.go    # GetCurrentUserID helper
│   ├── models/
│   │   ├── user.go              # User struct
│   │   ├── shop.go              # Shop struct
│   │   ├── order.go             # Order struct
│   │   └── kyc_document.go      # KYCDocument struct
│   ├── repository/
│   │   ├── user_repository.go
│   │   ├── shop_repository.go
│   │   ├── order_repository.go
│   │   ├── kyc_document_repository.go
│   │   └── repository_test.go   # Repository tests
│   └── services/
│       ├── services.go          # UserService (registration, login, email verification)
│       ├── order_service.go     # OrderService (CRUD operations)
│       └── kyc_document_service.go # KYCDocumentService (submit, review, pending)
├── web/
│   ├── static/
│   │   ├── css/                 # Stylesheets
│   │   ├── js/                  # Client-side JavaScript
│   │   └── images/              # Static images
│   └── templates/               # HTML templates (Golang html/template)
├── sqlc.yaml                    # sqlc configuration
├── Dockerfile                   # Docker build instructions
├── nixpacks.toml                # Railway deployment config
└── go.mod                       # Go module dependencies
```

## Architectural Layers

The application follows a **layered architecture** with clear separation of concerns:

### 1. Handler Layer (`internal/handlers/`)
- Receives HTTP requests and parses input (form values, query params, JSON).
- Delegates business logic to the service layer.
- Renders HTML templates or returns JSON responses.
- Manages session interactions (login/logout).

### 2. Service Layer (`internal/services/`)
- Contains business logic and validation rules.
- Orchestrates calls to the repository layer.
- Enforces domain rules (e.g., password hashing, email format, role validation).

### 3. Repository Layer (`internal/repository/`)
- Data access abstraction over the database.
- Uses sqlc-generated query code for type-safe SQL execution.
- Returns domain models.

### 4. Database Layer (`internal/database/`)
- Manages the PostgreSQL connection pool.
- Handles schema auto-migration on startup (`CREATE TABLE IF NOT EXISTS`).
- Provides the global `DB *sql.DB` instance.

## Authentication & Authorization

### Authentication
- **Email/Password**: Users register with email and password. A verification token is sent via SMTP. On login, bcrypt verifies the password hash.
- **Google OAuth2**: Users can log in via Google. The flow uses `golang.org/x/oauth2` and `go-oidc` for OpenID Connect discovery and ID token verification.

### Session Management
- Cookie-based sessions via Gorilla Sessions.
- Session store uses a hardcoded secret key (`halisi-secret-key`).
- Session values include `user_id`, `role`, `name`, and `email`.

### Authorization (RBAC)
Three roles are enforced via the `RequireRole` middleware:

| Role | Description |
|------|-----------|
| `customer` | End users who browse shops and place gas cylinder orders |
| `owner` | Shop owners who manage their shop, view orders, and submit KYC documents |
| `admin` | Administrators who verify shops and review KYC documents |

## Key Design Decisions

1. **Server-side rendering**: All pages are rendered via Go's `html/template` — no client-side framework. This keeps the app simple and fast.
2. **sqlc for type safety**: SQL queries are written in `.sql` files and compiled to Go code, catching query errors at compile time.
3. **Auto-migration**: Tables are created on startup via `CREATE TABLE IF NOT EXISTS`, eliminating the need for a separate migration tool.
4. **Cookie sessions**: No external session store (Redis, etc.) — sessions are stored in signed cookies, making the app stateless and easy to scale horizontally.
5. **Single-port deployment**: The app serves both HTML pages and static files on port 8080, simplifying deployment.
