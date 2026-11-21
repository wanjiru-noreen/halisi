# Halisi API Reference

This document outlines the HTTP endpoints, request payloads, response formats, and authentication requirements for HALISI.

---

## 1. Conventions & Standards

- **Base URL**: `https://halisi.onrender.com` (production) or `http://localhost:8080` (local development).
- **Content Types**:
  - Web UI routes serve `text/html; charset=utf-8`.
  - API endpoints serve and accept `application/json; charset=utf-8`.
  - Document uploads accept `multipart/form-data`.
- **Authentication**: Session cookie (`halisi_session`) transmitted with requests. Protected endpoints return `401 Unauthorized` or `403 Forbidden` if missing or unauthorized.

---

## 2. Health & System Probes

### 2.1. Liveness Probe
`GET /health`
- **Auth**: None (Public)
- **Response**: `200 OK`
```json
{
  "status": "ok"
}
```

### 2.2. Database Probe
`GET /health/db`
- **Auth**: None (Public)
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

## 3. Authentication & Sessions

### 3.1. Register Account
`POST /auth/register`
- **Auth**: Public
- **Request**:
```json
{
  "email": "customer@example.com",
  "password": "SecurePassword123!",
  "role": "customer",
  "full_name": "Jane Doe",
  "phone": "0712345678"
}
```
- **Response**: `201 Created` with session cookie or verification email triggered.

### 3.2. Login
`POST /auth/login`
- **Auth**: Public
- **Request**:
```json
{
  "email": "customer@example.com",
  "password": "SecurePassword123!"
}
```
- **Response**: `200 OK` (Sets `halisi_session` cookie).

### 3.3. Logout
`POST /auth/logout`
- **Auth**: Authenticated
- **Response**: `200 OK` (Clears session cookie).

### 3.4. Google OAuth 2.0
- `GET /auth/google`: Initiates Google OAuth consent flow with CSRF state token.
- `GET /auth/google/callback`: Handles OAuth redirect, verifies Google ID token, and provisions session.

---

## 4. Storefront & Catalog

### 4.1. List Products
`GET /api/v1/products?category=refill&brand=K-Gas`
- **Auth**: Public
- **Query Parameters**:
  - `category` (optional): `cylinder_with_gas`, `refill`, `accessory`.
  - `brand` (optional): Filter by gas brand name.
  - `limit` (optional, default 20): Maximum items to return.
- **Response**: `200 OK`
```json
[
  {
    "id": "prod_01h...",
    "vendor_id": "vend_01h...",
    "brand": "K-Gas",
    "weight_kg": 13.0,
    "category": "refill",
    "price_cents": 320000,
    "stock_quantity": 15,
    "is_active": true
  }
]
```

---

## 5. Orders & Checkout

### 5.1. Create Order (Checkout)
`POST /api/v1/checkout`
- **Auth**: Authenticated (`Customer`)
- **Request**:
```json
{
  "items": [
    {
      "product_id": "prod_01h...",
      "quantity": 1
    }
  ],
  "delivery_address": "Apartment 4B, Kilimani, Nairobi",
  "delivery_lat": -1.2921,
  "delivery_lng": 36.7845,
  "customer_phone": "0712345678"
}
```
- **Response**: `201 Created`
```json
{
  "order_id": "ord_01h...",
  "status": "pending",
  "total_amount_cents": 320000,
  "currency": "KES"
}
```

### 5.2. View Order Details
`GET /api/v1/orders/{id}`
- **Auth**: Authenticated (Order owner, assigned Vendor, assigned Rider, or Admin)
- **Response**: `200 OK`

---

## 6. M-Pesa Daraja Payments

### 6.1. Initiate STK Push
`POST /api/v1/payments/mpesa/initiate`
- **Auth**: Authenticated (`Customer` owning the order)
- **Request**:
```json
{
  "order_id": "ord_01h...",
  "phone": "254712345678"
}
```
- **Response**: `202 Accepted`
```json
{
  "payment_attempt_id": "att_01h...",
  "checkout_request_id": "ws_CO_21112026...",
  "status": "initiated"
}
```

### 6.2. Query Payment Status
`GET /api/v1/payments/mpesa/status/{attempt_id}`
- **Auth**: Authenticated (Owner of payment attempt)
- **Response**: `200 OK`
```json
{
  "attempt_id": "att_01h...",
  "status": "completed",
  "receipt_number": "QKA12345XYZ",
  "amount": 3200
}
```

### 6.3. Daraja Callback Webhook
`POST /api/v1/payments/mpesa/callback?token={callback_token}`
- **Auth**: Validated via secret single-use per-attempt query token.
- **Payload**: Standard Safaricom Daraja STK Push JSON callback.
- **Response**: `200 OK` `{"ResultCode": 0, "ResultDesc": "Accepted"}`

---

## 7. Deliveries & Rider Operations

### 7.1. List Available Deliveries
`GET /api/v1/deliveries`
- **Auth**: Authenticated (`Rider`)
- **Response**: `200 OK` list of pending unassigned delivery runs.

### 7.2. Accept Delivery Run
`POST /api/v1/deliveries/{id}/accept`
- **Auth**: Authenticated (`Rider` with verified status)
- **Response**: `200 OK` (Assigns delivery to rider and transitions order to `assigned`).

### 7.3. Update Delivery Milestone
`POST /api/v1/deliveries/{id}/milestone`
- **Auth**: Authenticated (Assigned `Rider`)
- **Request**:
```json
{
  "status": "in_transit",
  "notes": "Picked up cylinder from vendor depot"
}
```
- **Response**: `200 OK`

---

## 8. KYC Document Management

### 8.1. Upload Vendor Verification Document
`POST /api/v1/vendors/documents`
- **Auth**: Authenticated (`Vendor`)
- **Payload**: `multipart/form-data` with `document` file (PDF, PNG, JPG; max 5MB).
- **Response**: `201 Created`
```json
{
  "document_id": "doc_01h...",
  "filename": "business_permit.pdf",
  "status": "pending_review"
}
```

### 8.2. Stream Verification Document
`GET /api/v1/vendors/documents/{id}`
- **Auth**: Authenticated (Owning `Vendor` or `Admin`)
- **Response**: `200 OK` binary stream with `Content-Type` matching detected document MIME type.

---

## 9. Administrator Management

### 9.1. List Pending Vendor Approvals
`GET /api/v1/admin/vendors?status=pending`
- **Auth**: Authenticated (`Admin`)
- **Response**: `200 OK` list of vendors awaiting onboarding approval.

### 9.2. Approve or Reject Vendor
`POST /api/v1/admin/vendors/{id}/verify`
- **Auth**: Authenticated (`Admin`)
- **Request**:
```json
{
  "approved": true,
  "comments": "Business permit and EPRA license verified."
}
```
- **Response**: `200 OK`
