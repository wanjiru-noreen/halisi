# Halisi 🚀

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Database](https://img.shields.io/badge/Database-PostgreSQL%2016-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Query Layer](https://img.shields.io/badge/Query%20Engine-sqlc-blue?style=flat)](https://sqlc.dev/)
[![Templating](https://img.shields.io/badge/UI-templ%20%2B%20HTMX-black?style=flat)](https://templ.guide/)
[![Payments](https://img.shields.io/badge/Payments-M--Pesa%20Daraja-00A651?style=flat)](https://developer.safaricom.co.ke/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Halisi** is an on-demand liquefied petroleum gas (LPG) marketplace and delivery tracking ecosystem engineered for households and commercial kitchens in Kenya. Halisi connects consumers with verified local gas vendors through intelligent proximity matching, real-time Google Maps visualization, automated Safaricom M-Pesa STK payments, tamper-evident digital receipts, and end-to-end order tracking.

---

## 📌 Table of Contents

- [Overview & Value Proposition](#overview--value-proposition)
- [Key Features by User Role](#key-features-by-user-role)
- [System Architecture](#system-architecture)
- [Core Application Services](#core-application-services)
- [Technology Stack](#technology-stack)
- [Project Directory Structure](#project-directory-structure)
- [Getting Started & Local Development](#getting-started--local-development)
- [Environment Configuration](#environment-configuration)
- [API & Route Reference](#api--route-reference)
- [Security & Compliance](#security--compliance)
- [Testing & Quality Assurance](#testing--quality-assurance)
- [Contributing](#contributing)
- [License & Authors](#license--authors)

---

## 🌟 Overview & Value Proposition

In urban and peri-urban Kenya, sourcing cooking gas refills is plagued by friction:
- **Opaque Pricing & Counterfeit Risk**: Lack of price transparency and fear of underfilled or illicit cylinders.
- **Delivery Inefficiency**: Disorganized telephone orders with ambiguous delivery ETAs and no driver tracking.
- **Payment Reconciliation**: Manual payment verification causing dispatch bottlenecks for vendors.

**Halisi eliminates these barriers through a trusted multi-sided platform:**
1. **Nearby Discovery**: Instant geolocation matching customers with the nearest certified LPG vendors.
2. **Interactive Map**: Google Maps interface plotting verified vendors, prices, and stock availability.
3. **One-Tap Checkout**: Direct Safaricom M-Pesa Daraja STK push with automated instant payment reconciliation.
4. **Transparent Governance**: Mandatory regulatory KYC verification (EPRA license, KRA PIN, Business Registration) before vendors can list cylinders.

---

## 👥 Key Features by User Role

Halisi implements a strict **Role-Based Access Control (RBAC)** architecture supporting four specialized user personas:

### 1. 🛒 Customer (Household & Commercial Consumers)
* **Location-Based Discovery**: Automatically finds closest shops with live distance badges (km) and estimated delivery times.
* **Interactive Google Maps View**: Visual map markers showing open vendors, pricing per cylinder size (6kg, 13kg, 45kg), and ratings.
* **Streamlined Cart & Ordering**: Single or multi-cylinder refills with optional accessories (regulators, hosepipes, safety valves).
* **Instant M-Pesa STK Push**: Seamless mobile checkout without manual paybill/account number entry.
* **Real-Time Order Tracking**: 5-stage live status timeline (Pending → Paid → Confirmed → Out for Delivery → Delivered).
* **In-App Notification Center**: Instant visual alerts on order status changes, driver dispatch, and promotions.
* **Digital Receipts**: Instant downloadable and printable receipts with QR code and M-Pesa reference.
* **Verified Ratings & Feedback**: Post-delivery 1–5 star reviews with structured tags and written feedback.

### 2. 🏪 Shop Owner / Vendor
* **Vendor Operations Dashboard**: Real-time sales metrics, incoming order alerts with audible notifications, and dispatch controls.
* **Inventory & Operating Controls**: One-click toggles for cylinder stock availability and shop status (Open / Closed).
* **Price & Catalog Management**: Real-time cylinder price updates across 6kg, 13kg, and 45kg models.
* **Rider Assignment & Dispatch**: Assign internal riders or delivery drivers to accepted orders.
* **Regulatory KYC Portal**: Direct upload of EPRA licenses, business registration certificates, and KRA PIN documents.
* **Financial Ledger & Settlement**: Complete audit log of M-Pesa payments, cash-on-delivery collections, and daily revenue.

### 3. 🛵 Delivery Rider / Agent
* **Rider Task Board**: Assigned deliveries with optimal delivery route link via Google Maps navigation.
* **Delivery PIN Verification**: Secure one-time delivery verification PIN confirmation with customer upon cylinder drop-off.
* **Status Updates**: One-tap toggles for "Arrived at Customer" and "Delivered".

### 4. 🛡️ Platform Administrator
* **KYC Compliance Desk**: Visual inspection of uploaded vendor regulatory documentation with approve/reject workflows.
* **Vendor Governance**: Instant suspension or verification badge issuance for gas shops.
* **Platform Analytics**: Global GMV (Gross Merchandise Value), order completion rates, average delivery durations, and customer retention.
* **Audit & Moderation**: Review dispute resolution and moderation of customer reviews.

---

## 🏗️ System Architecture

Halisi follows clean architecture and idiomatic Go design patterns, separating transport, business domain logic, and data persistence:

```mermaid
graph TD
    subgraph ClientLayer["Frontend & Client Layer"]
        B["Browser (Desktop & Mobile)"]
        HTMX["HTMX Dynamic Interactivity"]
        GMaps["Google Maps JavaScript API"]
    end

    subgraph TransportLayer["Transport & HTTP Layer (cmd/api & internal/handlers)"]
        Router["Go net/http Server + RBAC Middleware"]
        Handlers["HTTP Handlers"]
        TemplComp["templ Component Engine (Type-Safe SSR)"]
    end

    subgraph ServiceLayer["Core Domain Services (internal/services)"]
        UserSvc["UserService & Auth"]
        ShopSvc["ShopService"]
        OrderSvc["OrderService"]
        LocSvc["LocationService (Geo/Maps)"]
        PaySvc["PaymentService (M-Pesa Daraja)"]
        NotifSvc["NotificationService (In-App/SSE)"]
        RcptSvc["ReceiptService (Digital PDF)"]
        RevSvc["FeedbackService (Ratings)"]
        KYCSvc["KYCDocumentService"]
    end

    subgraph PersistenceLayer["Data & External Infrastructure"]
        SQLC["sqlc Query Engine"]
        Postgres[("PostgreSQL 16 DB")]
        S3["Cloud Object Storage (S3/R2)"]
        Daraja["Safaricom Daraja API"]
        SMTP["Mailpit / SMTP Gateway"]
    end

    B --> Router
    HTMX --> Router
    GMaps --> LocSvc
    Router --> Handlers
    Handlers --> TemplComp
    Handlers --> ServiceLayer
    ServiceLayer --> SQLC
    SQLC --> Postgres
    PaySvc --> Daraja
    KYCSvc --> S3
    UserSvc --> SMTP
```

---

## ⚙️ Core Application Services

| Service | Package | Primary Responsibilities |
|---|---|---|
| **`UserService`** | `internal/services` | Registration, bcrypt hashing, Google OIDC OAuth2, email verification, password reset, and RBAC enforcement. |
| **`ShopService`** | `internal/services` | Vendor profile management, cylinder price indexes, stock toggle, and business hours. |
| **`LocationService`** | `internal/services` | Haversine proximity calculations, Google Maps geocoding, boundary validation, and nearest vendor sorting. |
| **`OrderService`** | `internal/services` | Multi-item checkout, price calculation, finite-state order transitions, and cancellation enforcement. |
| **`PaymentService`** | `internal/services` | Daraja STK Push generation, webhook processing, signature validation, idempotency, and financial ledgering. |
| **`NotificationService`** | `internal/services` | In-app notification feed, real-time status dispatch, unread counters, and transactional email triggers. |
| **`ReceiptService`** | `internal/services` | Generation of tamper-evident digital receipts, printable order confirmations, and QR code verification. |
| **`FeedbackService`** | `internal/services` | Verified buyer review verification, 1-5 star ratings aggregation, and sentiment moderation. |
| **`KYCDocumentService`** | `internal/services` | Upload handling, file magic-byte validation, secure cloud storage (S3/R2), and admin audit review. |

---

## 💻 Technology Stack

* **Language**: Go 1.26+ (strictly typed, zero-overhead concurrency)
* **Web Framework**: Standard library `net/http` with custom composable middleware
* **Server-Side Templating**: [templ](https://templ.guide/) (compiled, type-safe HTML components with compile-time checking)
* **Frontend Interactivity**: [HTMX](https://htmx.org/) for reactive SPA-like partial DOM updates and CSS3 responsive layout
* **Mapping & Geolocation**: Google Maps JavaScript API + Google Places / Geocoding API
* **Database & ORM**: PostgreSQL 16 managed with [sqlc](https://sqlc.dev/) (type-safe compile-time SQL generation)
* **Payments Engine**: Safaricom M-Pesa Daraja API (Lipa Na M-Pesa Online / STK Push)
* **Authentication**: Session cookies via `gorilla/sessions`, Passwords with `bcrypt`, Google OAuth2 via `go-oidc`
* **Logging & Observability**: Standard library `log/slog` structured JSON, Prometheus metrics exposition
* **Containerization**: Multi-stage `Dockerfile` with non-root runtime + `docker-compose.yml`

---

## 📂 Project Directory Structure

```text
halisi/
├── cmd/
│   └── api/                      # Application entrypoint & dependency wiring
├── docs/                         # In-depth architectural & API specifications
│   ├── ARCHITECTURE.md           # System design & module boundaries
│   ├── API.md                    # REST/HTTP API endpoint catalog
│   ├── DATABASE.md               # Schema diagrams & migration guides
│   └── DEPLOYMENT.md             # Production hosting & container guidelines
├── internal/
│   ├── database/                 # Database connection & migrations
│   │   ├── migrations/           # Versioned golang-migrate SQL scripts
│   │   ├── queries/              # Raw SQL queries for sqlc compilation
│   │   └── queried/              # sqlc-generated type-safe database code
│   ├── email/                    # SMTP transactional email client
│   ├── handlers/                 # HTTP request handlers & route controllers
│   ├── middleware/               # Auth, RBAC, CSRF, Rate-limit, Security headers, Tracing
│   ├── models/                   # Core business domain models & entities
│   ├── repository/               # Data access layer interfacing with sqlc
│   ├── services/                 # Pure domain business logic & third-party clients
│   └── templdata/                # Strongly-typed view structs passed to templ
├── web/
│   ├── static/                   # Static assets
│   │   ├── css/                  # Responsive stylesheets & modern tokens
│   │   ├── js/                   # HTMX, Google Maps helper scripts, toast alerts
│   │   └── images/               # Logos, empty state graphics, icons
│   └── templ/                    # Compiled, type-safe templ components (.templ)
├── Dockerfile                    # Hardened multi-stage container build
├── docker-compose.yml            # Local PostgreSQL 16 + Mailpit development cluster
├── Makefile                      # Standardized build, lint, test, and run scripts
├── sqlc.yaml                     # sqlc query generation configuration
└── go.mod                        # Go module definition and dependencies
```

---

## 🚀 Getting Started & Local Development

### Prerequisites
* **Go**: `1.26+` installed ([Download Go](https://golang.org/dl/))
* **Docker & Docker Compose**: Installed and running ([Download Docker](https://www.docker.com/))
* **templ CLI**: `go install github.com/a-h/templ/cmd/templ@latest`
* **sqlc CLI**: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

### 1. Clone & Set Up Environment
```bash
git clone https://github.com/wanjiru-noreen/halisi.git
cd halisi
cp .env.example .env
```

### 2. Start Supporting Services (PostgreSQL & Mailpit)
```bash
docker compose up -d postgres mailpit
```
* PostgreSQL is available on `localhost:5432`.
* Mailpit Web UI (inspect all sent emails) is accessible at `http://localhost:8025`.

### 3. Generate Code (Templ & sqlc)
```bash
# Compile templ templates into Go code
templ generate

# Compile sql queries into type-safe Go repository code
sqlc generate
```

### 4. Build and Run Server
```bash
go run ./cmd/api
```
The application will start on `http://localhost:8080`.

---

## 🔐 Environment Configuration

Create a `.env` file at the root of the project with the following keys:

```ini
# Application Configuration
PORT=8080
APP_ENV=development
APP_URL=http://localhost:8080
SESSION_KEY=your-32-byte-secret-session-key

# Database Connection
DATABASE_URL=postgres://postgres:postgres@localhost:5432/halisi?sslmode=disable

# Google OAuth2 Authentication (Optional for offline dev)
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Google Maps API
GOOGLE_MAPS_API_KEY=AIzaSyYourGoogleMapsApiKey

# Safaricom M-Pesa Daraja Credentials
MPESA_ENVIRONMENT=sandbox
MPESA_CONSUMER_KEY=your_daraja_consumer_key
MPESA_CONSUMER_SECRET=your_daraja_consumer_secret
MPESA_PASSKEY=your_daraja_stk_passkey
MPESA_SHORTCODE=174379
MPESA_CALLBACK_URL=https://your-domain.com/api/payments/mpesa/callback

# Transactional Email (Mailpit for local dev)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=no-reply@halisi.co.ke

# Cloud Storage (AWS S3 or Cloudflare R2 for KYC documents)
STORAGE_PROVIDER=local # or s3
S3_BUCKET=halisi-documents
S3_REGION=eu-west-1
S3_ACCESS_KEY=
S3_SECRET_KEY=
```

---

## 📡 API & Route Reference

### Public Routes
* `GET /` — Marketing landing page with hero CTA and nearby gas shop locator.
* `GET /register`, `POST /register` — User account creation (supports Customer and Vendor roles).
* `GET /verify-email` — Email verification link confirmation.
* `GET /login`, `POST /login` — Password authentication.
* `GET /auth/google`, `GET /auth/google/callback` — Google OAuth2 authentication.
* `GET /logout` — Session termination.

### Customer Endpoints (`middleware.RequireRole("customer")`)
* `GET /dashboard` — Customer hub with active orders, recent activity, and saved addresses.
* `GET /shops` — Discover verified gas shops, filter by proximity and cylinder sizes.
* `GET /shops/map` — Fullscreen interactive Google Map showing nearby shops with live markers.
* `GET /shop?id=:id` — Detailed shop profile, inventory, opening hours, and customer reviews.
* `GET /order`, `POST /order` — Multi-item order configuration and M-Pesa checkout initiation.
* `GET /orders` — Order history and tracking overview.
* `GET /orders/confirmation?id=:id` — Order confirmation view.
* `GET /orders/:id/track` — Real-time delivery progress timeline.
* `GET /orders/:id/receipt` — Printable and downloadable digital receipt.
* `POST /orders/:id/review` — Submit verified customer rating and written feedback.
* `GET /api/notifications` — In-app notification feed.

### Shop Owner / Vendor Endpoints (`middleware.RequireRole("owner")`)
* `GET /shop/manage` — Vendor management portal: inventory stock toggle, pricing, operating hours.
* `GET /shop/orders` — Incoming orders dispatch board.
* `POST /orders/update` — Update order status (Confirm, Dispatch, Deliver).
* `GET /shop/kyc`, `POST /shop/kyc/upload` — Regulatory license upload and compliance status.
* `GET /shop/ledger` — Financial reconciliation and M-Pesa payout summary.

### Admin Endpoints (`middleware.RequireRole("admin")`)
* `GET /admin` — Executive dashboard (platform GMV, user statistics, shop distribution).
* `GET /admin/kyc` — Pending KYC document review queue.
* `POST /admin/kyc/review` — Approve or reject shop regulatory submissions.
* `POST /admin/verify-shop` — Issue or revoke verified badge for gas shops.

### System & Health Endpoints
* `GET /healthz` — Liveness probe (HTTP 200).
* `GET /readyz` — Readiness probe (validates database connection and upstream health).
* `GET /metrics` — Prometheus telemetry exposition.
* `POST /api/payments/mpesa/callback` — Daraja STK Push asynchronous callback webhook.

---

## 🛡️ Security & Compliance

* **Data Protection & KYC Confidentiality**: Vendor business documents are stored in private cloud buckets with pre-signed ephemeral URLs, strictly accessible by authorized admins.
* **Double-Submit CSRF Defense**: All state-changing POST forms require cryptographic CSRF tokens.
* **Brute-Force Rate Limiting**: Exponential backoff and token-bucket rate limiting across login and checkout endpoints.
* **Security Headers**: Production middleware enforces `Strict-Transport-Security`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and fine-grained `Content-Security-Policy`.
* **Idempotent Webhooks**: M-Pesa webhook endpoints guarantee exactly-once processing with replay attack mitigation.

---

## 🧪 Testing & Quality Assurance

Halisi adheres to rigorous automated testing standards:

```bash
# Run unit and integration tests with race detection
make test

# Run static analysis and linter checks
make quality

# Run vulnerability check on all dependencies
make security

# Build and verify Docker container
make docker-build
```

---

## 🤝 Contributing

We welcome contributions from the community! Please follow our established workflow:

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes following [Conventional Commits](https://www.conventionalcommits.org/) (`git commit -m 'feat: add Google Maps shop radius filter'`).
4. Push to your branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request targeting the `develop` branch.

All pull requests run automated CI testing, vulnerability scanning, and code formatting checks.

---

## 📄 License & Authors

Distributed under the **MIT License**. See [LICENSE](LICENSE) for details.

* **Project Lead & Author**: Wanjiru Noreen (`mynahnoreen@gmail.com`)
* **Maintainers**: Bramwel Mutugi (`mutugibramwel@gmail.com`) & Contributors
