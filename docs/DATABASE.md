# Halisi Database

## Overview

Halisi uses **PostgreSQL** as its database. The application connects via the `lib/pq` driver and uses **sqlc** to generate type-safe Go code from SQL queries. Tables are auto-created on application startup via `CREATE TABLE IF NOT EXISTS` statements in `internal/database/database.go`.

## Connection

The database connection is configured via the `DATABASE_URL` environment variable:

```
DATABASE_URL=postgres://user:password@host:port/dbname?sslmode=disable
```

The connection is established in `internal/database/database.go` on application startup. If the variable is missing or the connection fails, the application exits with `log.Fatal`.

## Schema

### `users`

Stores registered users (customers, shop owners, and admins).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | SERIAL | PRIMARY KEY | Unique user identifier |
| `name` | TEXT | NOT NULL | User's full name |
| `email` | TEXT | NOT NULL, UNIQUE | User's email address |
| `password` | TEXT | NOT NULL | bcrypt-hashed password |
| `role` | TEXT | NOT NULL | User role: `customer`, `owner`, or `admin` |
| `email_verified` | BOOLEAN | NOT NULL, DEFAULT FALSE | Whether the email has been verified |
| `verification_token` | TEXT | DEFAULT '' | Random token for email verification |
| `google_id` | TEXT | NOT NULL, DEFAULT '' | Google OAuth2 sub (empty for non-Google users) |

**Notes:**
- Passwords are hashed using bcrypt with default cost before storage.
- The `verification_token` is a 32-byte random token hex-encoded, generated during registration.
- Google sign-in users have their `google_id` populated; they are auto-verified.

### `shops`

Stores gas shop information and pricing.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | SERIAL | PRIMARY KEY | Unique shop identifier |
| `name` | TEXT | NOT NULL | Shop name |
| `location` | TEXT | NOT NULL | Shop location/description |
| `phone` | TEXT | NOT NULL | Contact phone number |
| `price_6kg` | DOUBLE PRECISION | NOT NULL, DEFAULT 0 | Price for 6kg cylinder |
| `price_13kg` | DOUBLE PRECISION | NOT NULL, DEFAULT 0 | Price for 13kg cylinder |
| `price_45kg` | DOUBLE PRECISION | NOT NULL, DEFAULT 0 | Price for 45kg cylinder |
| `owner_id` | INTEGER | Nullable | FK to `users.id` (the shop owner) |
| `latitude` | DOUBLE PRECISION | DEFAULT 0 | GPS latitude for map display |
| `longitude` | DOUBLE PRECISION | DEFAULT 0 | GPS longitude for map display |
| `business_registration_number` | TEXT | DEFAULT '' | Business registration number |
| `kra_pin` | TEXT | DEFAULT '' | KRA PIN (Kenyan Revenue Authority) |
| `license_number` | TEXT | DEFAULT '' | Business license number |
| `verification_status` | TEXT | NOT NULL, DEFAULT 'pending' | Admin verification: `pending`, `approved`, `rejected` |

**Notes:**
- Shops start with `verification_status = 'pending'` and must be approved by an admin before appearing in customer listings.
- Pricing is stored per cylinder size (6kg, 13kg, 45kg).

### `orders`

Stores gas cylinder orders placed by customers.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | SERIAL | PRIMARY KEY | Unique order identifier |
| `user_id` | INTEGER | Nullable | FK to `users.id` (the customer who placed the order) |
| `shop_id` | INTEGER | Nullable | FK to `shops.id` (the shop fulfilling the order) |
| `cylinder_size` | TEXT | NOT NULL | Size of cylinder ordered (e.g., `6kg`, `13kg`, `45kg`) |
| `quantity` | INTEGER | NOT NULL | Number of cylinders |
| `total_price` | DOUBLE PRECISION | NOT NULL | Total price for the order |
| `status` | TEXT | NOT NULL | Order status (e.g., `Pending`, `Processing`, `Out for Delivery`, `Delivered`, `Cancelled`) |
| `delivery_address` | TEXT | NOT NULL, DEFAULT '' | Delivery address for the order |

**Notes:**
- New orders default to `status = 'Pending'`.
- Shop owners can update order status via `/orders/update`.
- Customers can delete their own orders.

### `kyc_documents`

Stores KYC (Know Your Customer) documents submitted by shop owners for admin review.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | SERIAL | PRIMARY KEY | Unique document identifier |
| `shop_id` | INTEGER | NOT NULL | FK to `shops.id` (the shop the document belongs to) |
| `document_type` | TEXT | NOT NULL | Type of document (e.g., `business_registration`, `kra_pin`, `license`) |
| `file_path` | TEXT | NOT NULL | Server file path where the uploaded document is stored |
| `status` | TEXT | NOT NULL, DEFAULT 'pending' | Review status: `pending`, `approved`, `rejected` |
| `rejection_reason` | TEXT | DEFAULT '' | Reason for rejection (populated when status is `rejected`) |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | When the document was submitted |
| `reviewed_at` | TIMESTAMP | Nullable | When the document was reviewed by an admin |

**Notes:**
- Documents are uploaded by shop owners via `/shop/kyc/upload`.
- Admins review documents via `/admin/kyc/review`.
- Rejection requires a reason; approval clears the rejection reason.

## Entity Relationships

```
users (1) ──── (M) orders
   │
   │ owner_id
   ▼
shops (1) ──── (M) orders
shops (1) ──── (M) kyc_documents
```

- A **user** can place many **orders** (as a customer).
- A **user** can own one **shop** (as an owner).
- A **shop** can have many **orders** (from different customers).
- A **shop** can have many **KYC documents** (for verification).

## sqlc Configuration

The project uses **sqlc** to generate type-safe Go code from SQL queries. Configuration is in `sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "internal/database/schema.sql"
    queries: "internal/database/queries"
    gen:
      go:
        package: "queried"
        out: "internal/database/queried"
```

### Generated Files

sqlc generates these files in `internal/database/queried/`:

| File | Contents |
|------|----------|
| `models.go` | Go structs matching database tables |
| `user_queries.go` | Generated functions for user queries |
| `shop_queries.go` | Generated functions for shop queries |
| `order_queries.go` | Generated functions for order queries |
| `kyc_queries.go` | Generated functions for KYC document queries |

### Regenerating Code

After modifying SQL queries in `internal/database/queries/`, regenerate the Go code:

```bash
sqlc generate
```

## Auto-Migration

On startup, `internal/database/database.go` runs `CREATE TABLE IF NOT EXISTS` for all four tables. This means:
- No separate migration tool is needed.
- New columns added to the schema require manual `ALTER TABLE` statements added to `createTables()`.
- For production use, consider replacing this with a proper migration tool like `golang-migrate` or `atlas`.
