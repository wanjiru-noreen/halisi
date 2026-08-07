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

- Go 1.20+ installed
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

- Build with `go build ./...` and then run the server. Use the browser to exercise the `/order` flow and confirm the confirmation page displays the map and delivery address.

