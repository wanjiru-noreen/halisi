# Halisi API Reference

All endpoints serve HTML pages (server-side rendering). The application uses cookie-based sessions for authentication and role-based access control.

## Base URL

```
http://localhost:8080
```

## Authentication Endpoints

### GET `/`
Render the home page. No authentication required.

### GET `/login`
Render the login form. No authentication required.

### POST `/login`
Process login form submission. Validates email/password against the database using bcrypt. On success, sets session cookies and redirects to the appropriate dashboard based on user role.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `email` | Yes | User's email address |
| `password` | Yes | User's password |

### GET `/register`
Render the registration form. No authentication required.

### POST `/register`
Process registration. Validates input, hashes password with bcrypt, generates a verification token, and sends a verification email via SMTP.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `fullname` | Yes | User's full name |
| `email` | Yes | User's email address |
| `password` | Yes | User's password |
| `role` | Yes | Either `customer` or `owner` |

### GET `/verify-email`
Verify email address using the token from the registration email.

**Query parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `token` | Yes | The verification token sent via email |

### GET `/logout`
Clear the session cookie and redirect to the login page.

## Google OAuth2 Endpoints

### GET `/auth/google`
Initiate Google OAuth2 login flow. Redirects to Google's authorization page.

### GET `/auth/google/callback`
Handle the OAuth2 callback from Google. Verifies the ID token via OpenID Connect, creates or updates the user record, and sets the session.

## Customer Routes

All customer routes require authentication with role `customer`.

### GET `/dashboard`
Render the customer dashboard showing recent orders and activity.

### GET `/shops`
Render a list of all verified shops with their details (name, location, prices).

### GET `/shop`
Render a single shop's details including prices and location.

**Query parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `id` | Yes | Shop ID |

### GET `/order`
Render the order placement form for a specific shop.

**Query parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `shop_id` | Yes | Shop ID to order from |

### POST `/order`
Process a new order. Creates an order record with status `Pending`.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `shop_id` | Yes | Shop ID |
| `cylinder_size` | Yes | Cylinder size (e.g., `6kg`, `13kg`, `45kg`) |
| `quantity` | Yes | Number of cylinders |
| `delivery_address` | Yes | Delivery address |

### GET `/orders`
Render a list of all orders placed by the current user.

### POST `/orders/delete`
Delete an order.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Order ID to delete |

### GET `/orders/confirmation`
Render order confirmation page with order details.

**Query parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `id` | Yes | Order ID |

## Admin Routes

All admin routes require authentication with role `admin`.

### GET `/admin`
Render the admin dashboard with overview statistics.

### POST `/admin/verify-shop`
Verify or reject a shop's registration. Updates the shop's `verification_status`.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `shop_id` | Yes | Shop ID to verify |
| `action` | Yes | `approve` or `reject` |

### GET `/admin/kyc`
Render a list of pending KYC documents awaiting review.

### POST `/admin/kyc/review`
Review a KYC document — approve or reject it with an optional rejection reason.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `document_id` | Yes | KYC document ID |
| `status` | Yes | `approved` or `rejected` |
| `rejection_reason` | Conditional | Required if status is `rejected` |

### GET `/admin/kyc/document`
View a specific KYC document's details.

**Query parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `id` | Yes | KYC document ID |

## Shop Owner Routes

All owner routes require authentication with role `owner`.

### GET `/shop/orders`
Render a list of orders for the owner's shop.

### GET `/shop/manage`
Render the shop management page where the owner can update shop details (name, location, phone, prices, coordinates, business registration info).

### POST `/shop/manage`
Update shop details.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Shop name |
| `location` | Yes | Shop location |
| `phone` | Yes | Contact phone number |
| `price_6kg` | Yes | Price for 6kg cylinder |
| `price_13kg` | Yes | Price for 13kg cylinder |
| `price_45kg` | Yes | Price for 45kg cylinder |
| `latitude` | No | Shop latitude for map display |
| `longitude` | No | Shop longitude for map display |
| `business_registration_number` | No | Business registration number |
| `kra_pin` | No | KRA PIN (Kenyan tax ID) |
| `license_number` | No | Business license number |

### GET `/shop/kyc`
Render a list of KYC documents submitted by the owner's shop, with their current status.

### POST `/shop/kyc/upload`
Upload a KYC document for the owner's shop. Creates a new KYC document record with status `pending`.

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `document_type` | Yes | Type of document (e.g., `business_registration`, `kra_pin`, `license`) |
| `file` | Yes | File upload (document image/PDF) |

### POST `/orders/update`
Update the status of an order (e.g., from `Pending` to `Delivered`).

**Form fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `order_id` | Yes | Order ID |
| `status` | Yes | New status (e.g., `Pending`, `Processing`, `Out for Delivery`, `Delivered`, `Cancelled`) |

## Static Files

### GET `/static/*`
Serve static files (CSS, JavaScript, images) from `web/static/`.

## Session Management

The application uses cookie-based sessions via Gorilla Sessions. Session data stored in cookies includes:

| Key | Type | Description |
|-----|------|-------------|
| `user_id` | int | Logged-in user's ID |
| `role` | string | User's role (`customer`, `owner`, `admin`) |
| `name` | string | User's display name |
| `email` | string | User's email address |

## Error Handling

| Status Code | Meaning |
|-------------|---------|
| 303 See Other | Redirect to `/login` when not authenticated |
| 403 Forbidden | User authenticated but lacks required role |
| 405 Method Not Allowed | Invalid HTTP method for the endpoint |
| 500 Internal Server Error | Server-side error (e.g., template render failure, DB error) |
