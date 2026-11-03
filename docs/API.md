# Halisi API & Route Reference

Halisi serves server-side rendered (SSR) type-safe components via **templ** and **HTMX**, complemented by asynchronous REST/JSON API endpoints for Google Maps, M-Pesa webhooks, and in-app notifications.

---

## 📌 Base URLs & Content Negotiation

* **Local Development**: `http://localhost:8080`
* **Production**: `https://api.halisi.co.ke` (or configured `APP_URL`)
* **Session Storage**: HTTP-only, SameSite-strict encrypted cookie via `gorilla/sessions`.
* **CSRF Protection**: All state-changing `POST`/`PUT`/`DELETE` requests must include a valid CSRF token via form field `gorilla.csrf.Token` or header `X-CSRF-Token`.

---

## 🔑 1. Authentication & Account Management

### `GET /`
* **Access**: Public
* **Description**: Marketing landing page with hero CTA, featured gas refill shops, and quick search.

### `GET /register` | `POST /register`
* **Access**: Public
* **Description**: Create new account. Hashes password with `bcrypt`, generates a verification token, and sends confirmation email via SMTP.
* **Form Parameters**:
  * `name` (string, required): Full name.
  * `email` (string, required): Valid email address.
  * `phone` (string, required): Format `2547XXXXXXXX`.
  * `password` (string, required): Min 8 characters.
  * `role` (string, required): `customer`, `owner` (vendor), or `rider`.

### `GET /verify-email?token=:token`
* **Access**: Public
* **Description**: Verifies email address and activates account.

### `GET /login` | `POST /login`
* **Access**: Public (Rate-limited: 5 req/min)
* **Description**: Authenticate credentials and establish user session.
* **Form Parameters**:
  * `email` (string, required)
  * `password` (string, required)

### `GET /auth/google` | `GET /auth/google/callback`
* **Access**: Public
* **Description**: Google OAuth2 OpenID Connect flow. Redirects to Google consent and provisions verified session.

### `GET /forgot-password` | `POST /forgot-password`
* **Access**: Public (Rate-limited: 3 req/min)
* **Description**: Generate 15-minute cryptographically signed password reset token and dispatch instructions via email.

### `GET /reset-password?token=:token` | `POST /reset-password`
* **Access**: Public
* **Description**: Update password using valid reset token.

### `GET /logout`
* **Access**: Authenticated
* **Description**: Invalidate cookie session and redirect to `/login`.

---

## 🛒 2. Customer Endpoints (`role: customer`)

### `GET /dashboard`
* **Description**: Customer hub showing active orders, delivery tracking shortcuts, and quick re-order actions.

### `GET /shops`
* **Description**: Browse verified LPG shops with filters for cylinder sizes (6kg, 13kg, 45kg) and brand.
* **Query Parameters**:
  * `q` (string, optional): Search keyword.
  * `cylinder_size` (string, optional): `6kg`, `13kg`, or `45kg`.

### `GET /shops/map`
* **Description**: Fullscreen interactive Google Map plotting verified LPG shops within radius, showing live prices and stock availability badges.

### `GET /api/shops/nearby`
* **Description**: JSON endpoint returning nearby shops ranked by Haversine proximity.
* **Query Parameters**:
  * `lat` (float, required): Customer latitude.
  * `lon` (float, required): Customer longitude.
  * `radius_km` (float, optional, default: 10).

### `GET /shop?id=:id`
* **Description**: Shop profile page with cylinder prices, operating hours, stock status, ratings breakdown, and customer reviews.

### `GET /order?shop_id=:id`
* **Description**: Order configuration form with delivery address autocomplete, cylinder size picker, and optional safety accessories (regulator, hosepipe).

### `GET /api/delivery-fee`
* **Description**: Calculate delivery fee based on customer address coordinates and shop location.
* **Query Parameters**: `shop_id` (int), `lat` (float), `lon` (float).
* **Response**: `{"delivery_fee": 150.00, "distance_km": 2.4}`.

### `POST /order`
* **Description**: Create new refill order and initiate checkout.
* **Form Parameters**:
  * `shop_id` (int, required)
  * `cylinder_size` (string, required)
  * `quantity` (int, required)
  * `delivery_address` (string, required)
  * `latitude` (float, required)
  * `longitude` (float, required)
  * `payment_method` (string, required): `mpesa` or `cash`.
  * `phone_number` (string, conditional): Required if `payment_method=mpesa`.

### `GET /orders`
* **Description**: List order history with status pills (Pending, Paid, Dispatched, Delivered, Cancelled).

### `GET /orders/:id/track`
* **Description**: Live delivery tracking view with 5-step timeline, assigned rider details, and estimated arrival time.

### `GET /orders/:id/receipt`
* **Description**: Tamper-evident printable and downloadable digital receipt featuring order details, M-Pesa receipt code, and verification QR code.

### `POST /orders/:id/review`
* **Description**: Submit 1–5 star rating and feedback tags for a delivered order.
* **Form Parameters**:
  * `rating` (int, required): 1–5.
  * `comment` (string, optional): Written feedback.
  * `tags` (string[], optional): `fast_delivery`, `polite_rider`, `fair_price`.

---

## 🏪 3. Shop Owner / Vendor Endpoints (`role: owner`)

### `GET /shop/manage`
* **Description**: Vendor portal for managing shop profile, operating hours, stock availability, and cylinder prices.

### `POST /shop/manage`
* **Description**: Update shop business details and pricing.
* **Form Parameters**:
  * `name`, `location`, `phone`
  * `price_6kg`, `price_13kg`, `price_45kg`
  * `is_open` (boolean): Shop operating state.
  * `stock_6kg`, `stock_13kg`, `stock_45kg` (booleans): Real-time inventory toggles.

### `GET /shop/orders`
* **Description**: Incoming orders dispatch board categorized by active, in-transit, and completed orders.

### `POST /orders/update`
* **Description**: Advance order lifecycle status.
* **Form Parameters**:
  * `order_id` (int, required)
  * `status` (string, required): `confirmed`, `out_for_delivery`, `delivered`, `cancelled`.
  * `rider_id` (int, optional): Assign delivery rider.

### `GET /shop/kyc` | `POST /shop/kyc/upload`
* **Description**: Upload regulatory compliance files (EPRA license, KRA PIN, Business Registration).
* **Form Parameters**:
  * `document_type` (string, required): `epra_license`, `business_registration`, `kra_pin`.
  * `file` (multipart/form-data, required): PDF/PNG/JPEG under 5MB.

### `GET /shop/ledger`
* **Description**: View financial settlements, M-Pesa collections, and daily revenue breakdown.

---

## 🛵 4. Delivery Rider Endpoints (`role: rider`)

### `GET /rider/orders`
* **Description**: Assigned deliveries board with customer phone numbers and Google Maps route links.

### `POST /rider/orders/:id/verify-pin`
* **Description**: Submit customer 4-digit OTP delivery verification code to complete delivery.
* **Form Parameters**:
  * `verification_pin` (string, required): 4-digit code provided by customer upon drop-off.

---

## 🛡️ 5. Platform Administrator Endpoints (`role: admin`)

### `GET /admin`
* **Description**: Platform executive dashboard with GMV, user growth, shop verification stats, and volume charts.

### `GET /admin/kyc`
* **Description**: Pending vendor regulatory document review queue.

### `POST /admin/kyc/review`
* **Description**: Approve or reject submitted KYC documentation.
* **Form Parameters**:
  * `document_id` (int, required)
  * `status` (string, required): `approved` or `rejected`.
  * `rejection_reason` (string, conditional): Mandatory when rejecting.

### `POST /admin/verify-shop`
* **Description**: Issue or revoke verified marketplace badge for an LPG shop.

### `GET /admin/users` | `POST /admin/users/:id/status`
* **Description**: Manage user accounts and suspend non-compliant users or shops.

---

## 💳 6. Payments & Webhooks

### `POST /api/payments/mpesa/stkpush`
* **Description**: Initiates Daraja STK Push to customer's mobile phone.
* **Payload**: `{"order_id": 102, "phone_number": "254712345678"}`.

### `POST /api/payments/mpesa/callback`
* **Access**: Public Webhook (Safaricom Daraja IP whitelist / signature verification)
* **Description**: Asynchronous payment confirmation webhook. Idempotently transitions payment and order records to `success`/`paid`.

### `GET /api/orders/:id/payment-status`
* **Description**: Polling endpoint returning live payment status (`initiated`, `success`, `failed`).

---

## 🔔 7. Notifications & Utilities

### `GET /api/notifications`
* **Description**: Return user's recent in-app notifications with unread counts.

### `POST /api/notifications/:id/read`
* **Description**: Mark specific notification as read.

### `GET /verify-receipt?hash=:hash`
* **Access**: Public
* **Description**: Public receipt authenticity verification page reached via QR code.

### `GET /healthz` | `GET /readyz`
* **Access**: Public
* **Description**: Liveness and readiness health probes for container orchestrators.

### `GET /metrics`
* **Access**: Internal / Monitoring
* **Description**: Prometheus telemetry metrics exposition.
