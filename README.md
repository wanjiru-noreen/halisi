# Halisi

A digital marketplace connecting households with gas refill shops in Kenya — with reliable delivery tracking and order management.

## Overview

Halisi helps customers find gas refill shops, compare prices, place orders, and track deliveries. Shop owners manage their prices, handle incoming orders, and update delivery status — all through one platform.

The goal is to make gas refilling easier, faster, and more accessible for every household.

## Problem

Many households experience challenges when refilling gas:

* Difficulty finding nearby refill shops
* Lack of price comparison between suppliers
* Time wasted making multiple calls
* No simple way to track refill requests

## Solution

Halisi provides a trusted marketplace where:

* Customers can discover gas shops, place orders, and track deliveries
* Shop owners can manage prices, accept orders, and update delivery status
* Administrators can oversee the platform and verify shop owners

## Features

### Customer

* User registration and login (including Google OAuth)
* Browse available gas shops and compare prices
* Place refill orders with delivery address
* Track order status and delivery progress
* View order history

### Shop Owner

* Shop account management with KYC verification
* Update gas cylinder prices
* View and manage incoming orders
* Accept or reject orders
* Update order status and delivery tracking

### Admin

* Manage users and gas shops
* Monitor KYC document verification
* Monitor system activity

## Technology Stack

**Backend:** Go 1.26 with `net/http`

**Frontend:** HTML templates with CSS

**Database:** PostgreSQL with sqlc for type-safe queries

**Authentication:** Google OAuth via `go-oidc`

**Architecture:** Layered — Handlers → Services → Repositories → Models

## Project Structure

```
halisi/
├── cmd/
│   └── api/              # Application entry point
├── internal/
│   ├── handlers/         # HTTP handlers
│   ├── models/           # Data models
│   ├── database/         # Database setup & sqlc queries
│   ├── repository/       # Data access layer
│   ├── services/         # Business logic
│   └── middleware/       # Auth and request middleware
├── web/
│   ├── templates/        # HTML templates
│   ├── templ/            # Type-safe templ components (in progress)
│   └── static/           # CSS, JS, images
├── Dockerfile
├── sqlc.yaml             # sqlc configuration
└── README.md
```

## Quickstart

**Prerequisites:**

* Go 1.26+ installed
* PostgreSQL running locally

**Local setup and run:**

```bash
# Build the project
go build ./...

# Run the API server
go run ./cmd/api

# Open http://localhost:8080
```

The application connects to PostgreSQL using environment variables (`DATABASE_URL` or individual `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`). On first run, tables are created automatically.

**Docker:**

```bash
docker build -t halisi .
docker run -p 8080:8080 halisi
```

## API Endpoints

* `GET /` — Landing page
* `GET /register`, `POST /register` — User registration
* `GET /login`, `POST /login` — Login
* `GET /dashboard` — User dashboard (requires auth)
* `GET /shops` — Browse shops (requires auth)
* `GET /order` — Order form (requires auth)
* `POST /order` — Place order (renders confirmation)
* `GET /orders` — Your orders

## Future Improvements

* Customer reviews and ratings
* Location-based shop discovery
* Mobile application

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Submit a pull request to the `development` branch

All pull requests are reviewed before merging.

## License

This project is licensed under the [MIT License](LICENSE).

## Author

Wanjiru Noreen

