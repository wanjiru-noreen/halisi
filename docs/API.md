# Halisi API Reference

This document outlines the HTTP endpoints, request payloads, response formats, and authentication requirements for HALISI. All endpoints listed below correspond to routes actually registered and wired into the application server.

---

## 1. Conventions & Standards

- **Base URL**: `https://halisi.onrender.com` (production) or `http://localhost:8080` (local development).
- **Content Types**:
  - Web UI routes serve `text/html; charset=utf-8`.
  - API endpoints serve and accept `application/json; charset=utf-8`.
  - Document uploads accept `multipart/form-data`.
- **Authentication**: Session cookie (`halisi_session`) transmitted with requests. Protected endpoints return `401 Unauthorized` when unauthenticated, or `403 Forbidden` when the role is insufficient.

---

## 2. Health & System Probes

### 2.1. Liveness Probe
`GET /health`
- **Auth**: Public
- **Response**: `200 OK`
```json
{
  "status": "ok"
}
```

### 2.2. Database Probe
`GET /health/db`
- **Auth**: Public
- **Response**:
  - `200 OK` when PostgreSQL database is reachable:
    ```json
    {
      "status": "ok"
    }
    ```
  - `503 Service Unavailable` when database ping fails (no credentials leaked):
    ```json
    {
      "status": "unavailable"
    }
    ```

---

## 3. Storefront & Public Catalog

### 3.1. List Verified Vendors
`GET /api/storefront/vendors?q={search}`
- **Auth**: Public
- **Query Parameters**:
  - `q` (optional): Filter by vendor name or service area (max 100 characters).
- **Response**: `200 OK`
```json
{
  "vendors": [
    {
      "id": 1,
      "name": "Nairobi Gas Depot",
      "service_area": "Kilimani, Nairobi",
      "product_count": 5
    }
  ]
}
```

### 3.2. List Storefront Products
`GET /api/storefront/products?vendor_id={id}&q={search}`
- **Auth**: Public
- **Query Parameters**:
  - `vendor_id` (optional): Positive integer filtering products by vendor.
  - `q` (optional): Search query matching product name or vendor name (max 100 characters).
- **Response**: `200 OK`
```json
{
  "products": [
    {
      "id": 1,
      "vendor_id": 1,
      "vendor_name": "Nairobi Gas Depot",
      "name": "13kg Refill",
      "price": "2800.00",
      "available_quantity": 25
    }
  ]
}
```

---

## 4. Authentication & Identity

### 4.1. Register Account
`POST /api/auth/register`
- **Auth**: Public (Rate-limited)
- **Request**:
```json
{
  "email": "customer@example.com",
  "password": "SecurePassword123!",
  "role": "customer"
}
```
- **Response**: `201 Created`
```json
{
  "user": {
    "id": 1,
    "email": "customer@example.com",
    "role": "customer"
  }
}
```

### 4.2. Login
`POST /api/auth/login`
- **Auth**: Public (Rate-limited)
- **Request**:
```json
{
  "email": "customer@example.com",
  "password": "SecurePassword123!"
}
```
- **Response**: `200 OK` (Sets `halisi_session` HTTP-only cookie)

### 4.3. Logout
`POST /api/auth/logout`
- **Auth**: Authenticated
- **Response**: `200 OK` (Clears session cookie)

### 4.4. Current User Session
`GET /api/auth/me`
- **Auth**: Authenticated
- **Response**: `200 OK`
```json
{
  "user": {
    "id": 1,
    "email": "customer@example.com",
    "role": "customer"
  }
}
```

### 4.5. Auth Configuration Options
`GET /api/auth/options`
- **Auth**: Public
- **Response**: `200 OK`
```json
{
  "verification_required": true,
  "google_enabled": true
}
```

### 4.6. Email Verification
- `POST /api/auth/verification/resend`: Resend email verification code.
- `POST /api/auth/verify-email`: Submit verification token (`{"token": "..."}`).

### 4.7. Google OAuth 2.0
- `POST /api/auth/google/start`: Initiate OAuth redirect for login.
- `POST /api/auth/google/link`: Initiate OAuth redirect to link Google account to logged-in user.
- `GET /api/auth/google/callback`: Handle OAuth redirect and exchange code.

---

## 5. Customer Orders & Checkout

### 5.1. Customer Checkout
`POST /api/customer/checkout`
- **Auth**: Authenticated (`Customer`)
- **Request**:
```json
{
  "address_id": 1,
  "items": [
    {
      "product_id": 1,
      "quantity": 1
    }
  ]
}
```
- **Response**: `201 Created`
```json
{
  "order": {
    "id": 101,
    "status": "pending",
    "currency": "KES",
    "total": "2800.00"
  }
}
```

### 5.2. Customer Orders
- `GET /api/customer/orders`: List authenticated customer's orders.
- `GET /api/customer/orders/{id}`: View specific order details.

### 5.3. Customer Address Management
- `GET /api/customer/addresses`: List customer saved addresses.
- `POST /api/customer/addresses`: Create new address.
- `GET /api/customer/addresses/{id}`: Get address.
- `PATCH /api/customer/addresses/{id}`: Update address.
- `DELETE /api/customer/addresses/{id}`: Delete address.

---

## 6. M-Pesa Daraja Payments

### 6.1. Gateway Availability
`GET /api/mpesa/availability`
- **Auth**: Public
- **Response**: `200 OK`
```json
{
  "enabled": true
}
```

### 6.2. Initiate STK Push Payment
`POST /api/customer/orders/{id}/payments/mpesa`
- **Auth**: Authenticated (`Customer` owning the order)
- **Headers**: `Idempotency-Key: <unique-request-key>`
- **Request**:
```json
{
  "phone_number": "254712345678"
}
```
- **Response**: `202 Accepted`
```json
{
  "status": "initiated",
  "request_key": "<unique-request-key>"
}
```

### 6.3. Daraja Callback Webhook
`POST /api/mpesa/callback/{token}`
- **Auth**: Validated via secret per-attempt query token in route path.
- **Payload**: Standard Safaricom Daraja STK Push JSON callback.
- **Response**: `200 OK` `{"ResultCode": 0, "ResultDesc": "Accepted"}`

---

## 7. Deliveries & Rider Operations

### 7.1. Rider Registration
- `POST /api/rider/registration`: Submit rider onboarding profile.
- `GET /api/rider/registration`: Get rider profile status.
- `PATCH /api/rider/registration`: Update rider profile.

### 7.2. Rider Deliveries
- `GET /api/rider/deliveries`: List assigned deliveries.
- `GET /api/rider/deliveries/{id}`: Get delivery run details.
- `PATCH /api/rider/deliveries/{id}`: Update delivery run status.

---

## 8. Vendor Management & KYC

### 8.1. Vendor Application
- `GET /api/vendor/application`: View current onboarding application.
- `PUT /api/vendor/application`: Save draft application.
- `POST /api/vendor/application/submit`: Submit application for administrative verification.

### 8.2. KYC Documents
- `POST /api/vendor/documents/{type}`: Upload document (`multipart/form-data`). Types: `business_permit`, `epra_license`, `id_card`.
- `GET /api/vendor/documents/{id}`: Download document stream (restricted to owner or admin).

### 8.3. Vendor Shop & Inventory
- `GET /api/vendor/shop`: Get vendor shop details.
- `POST /api/vendor/shop`: Create vendor shop.
- `PATCH /api/vendor/shop`: Update vendor shop.
- `GET /api/vendor/products`: List vendor-owned products.
- `POST /api/vendor/products`: Add new product.
- `PATCH /api/vendor/products/{id}`: Update product.
- `DELETE /api/vendor/products/{id}`: Delete product.
- `GET /api/vendor/orders`: List orders assigned to vendor.
- `POST /api/vendor/orders/{id}/confirm`: Confirm order fulfillment.
- `GET /api/vendor/commissions`: View earned commissions and balance.

---

## 9. Administrative Operations

### 9.1. Vendor Review & Verification
- `GET /api/admin/vendor-applications`: List vendor applications by status.
- `GET /api/admin/vendor-applications/{id}`: View specific application and attached documents.
- `POST /api/admin/vendor-applications/{id}/review`: Approve or reject vendor application.
- `PATCH /api/admin/vendors/{id}/status`: Enable, disable, or suspend vendor account.
- `PATCH /api/admin/vendors/{id}/commission-rate`: Set custom commission rate.

### 9.2. Delivery & Settlement Management
- `GET /api/admin/deliveries/{id}`: View delivery details.
- `POST /api/admin/deliveries/{id}/assign`: Assign or reassign rider.
- `GET /api/admin/settlements`: List platform settlements.
- `POST /api/admin/vendors/{id}/settlements`: Record vendor settlement payout.
- `PATCH /api/admin/settlements/{id}`: Update settlement status.
