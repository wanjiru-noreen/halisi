# Halisi

A gas refill ordering and management system that connects customers with nearby gas refill shops.

## Overview

Halisi helps users find gas refill shops, view available prices, and place refill orders. Shop owners can manage their gas prices and handle customer orders through the system.

The goal is to make gas refilling easier, faster, and more accessible.

## Problem

Many households experience challenges when refilling gas:

* Difficulty finding nearby refill shops
* Lack of price comparison between suppliers
* Time wasted making multiple calls
* No simple way to track refill requests

## Solution

Halisi provides a digital platform where:

* Customers can discover gas shops and request refills
* Shop owners can manage prices and orders
* Administrators can manage the platform

## Features

### Customer

* User registration and login
* View available gas shops
* View gas refill prices
* Place refill orders
* View order history

### Shop Owner

* Shop account management
* Update gas prices
* View incoming orders
* Accept or reject orders
* Update order status

### Admin

* Manage users
* Manage gas shops
* Monitor system activity

## Technology Stack

**Backend**

* Go (Golang)

**Frontend**

* HTML
* CSS

**Database**

* SQLite

**Architecture**

* Layered architecture:

  * Handlers
  * Services
  * Repository
  * Models

## Project Structure

```
Halisi/
│
├── cmd/
│   └── api/
│
├── internal/
│   ├── handlers/
│   ├── models/
│   ├── database/
│   ├── repository/
│   ├── services/
│   └── middleware/
│
├── web/
│   ├── templates/
│   └── static/
│
└── README.md
```

## Future Improvements

* Mobile application
* Location-based shop discovery
* Online payments
* Delivery tracking
* Customer reviews

## Author

Wanjiru Noreen

## Quickstart

Prerequisites:

- Go version specified in `go.mod` installed, plus GCC for the SQLite CGO driver
- (Optional) `sqlite3` for exploring the database file

Local setup and run:

```bash
# fetch deps and build
go build ./...

# run the API server
go run ./cmd/api

# open http://localhost:8080
```

Database:

- The project uses `halisi.db` (SQLite). On first run the app will create tables and apply lightweight migrations.

Useful endpoints (after running):

- `GET /` — Landing page
- `GET /register`, `POST /register` — User registration
- `GET /login`, `POST /login` — Login
- `GET /dashboard` — User dashboard (requires auth)
- `GET /shops` — Browse shops (requires auth)
- `GET /order` — Order form (requires auth)
- `POST /order` — Place order (renders confirmation)
- `GET /orders` — Your orders

Notes / recent UI behavior

- The order form shows a single dropdown to select a shop (the previous tap/grid was removed).
- After placing an order the server renders a confirmation page that includes a small static roadmap graphic and the delivery address.
- Price mapping implemented in handlers:
  - 6kg => KSh 1400
  - 13kg => KSh 2800
  - 45kg => KSh 6000

Templates

- Templates are in `web/templates`. Partials (like the navbar) are parsed alongside page templates — if you add new partials, include them in `template.ParseFiles` or use a template `FuncMap` as needed.

Testing

- Run `make check` for all Go quality, build, test, coverage, and security checks.
- Run `make ci` to include the Docker image build, or use `make help` to see individual commands such as `make fmt`, `make test`, and `make docker-build`.
- To run the container locally, create `.env` with the required Google OAuth settings and run `make docker-run`. Override defaults when needed, for example `make docker-run DOCKER_PORT=9090 DOCKER_ENV_FILE=.env.local`.
- Run the server and use the browser to exercise the `/order` flow and confirm the confirmation page displays the map and delivery address.

## Continuous integration

`.github/workflows/ci.yml` runs on every branch push and on pull requests targeting `development` or `main`. It can also be run manually from GitHub Actions.

- **Quality:** Go formatting, module integrity and tidiness, and `go vet`.
- **Build and test:** compilation, tests with race detection, and coverage reports retained for seven days. Coverage is reported without a minimum percentage gate.
- **Security:** reachable vulnerability checks using a pinned `govulncheck` version.
- **Docker build:** builds the application image on pull requests without publishing it.

The Go jobs run independently on Linux with CGO enabled and cached dependencies/build outputs. Superseded runs are cancelled per branch or PR. No application secrets are required. Push and PR runs are kept separate so the PR merge result is always checked.

Configure the `development` and `main` branch rules to require **Quality**, **Build and test**, **Security**, and **Docker build** before merging. The workflow file does not configure repository branch rules itself.

Add PostgreSQL integration/migration checks and separate frontend jobs when those components are introduced. Current automated tests do not cover all handlers, services, templates, or production database migrations.

