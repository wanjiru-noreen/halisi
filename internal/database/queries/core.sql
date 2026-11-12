-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetActiveSession :one
SELECT * FROM sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: RevokeSession :execrows
UPDATE sessions SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: CreateVendor :one
INSERT INTO vendors (user_id, name, address) VALUES ($1, $2, $3) RETURNING *;

-- name: GetVendor :one
SELECT * FROM vendors WHERE id = $1;

-- name: SetVendorAddress :one
UPDATE vendors SET address_id = $2, address = $3 WHERE id = $1 RETURNING *;

-- name: SaveVendorDocument :one
INSERT INTO vendor_documents (vendor_id, document_type, storage_key)
VALUES ($1, $2, $3)
ON CONFLICT (vendor_id, document_type) DO UPDATE
SET storage_key = EXCLUDED.storage_key, status = 'pending'
RETURNING *;

-- name: CreateProduct :one
INSERT INTO products (vendor_id, name, price) VALUES ($1, $2, $3) RETURNING *;

-- name: SetInventory :one
INSERT INTO vendor_inventory (product_id, vendor_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (product_id) DO UPDATE SET quantity = EXCLUDED.quantity
WHERE vendor_inventory.vendor_id = EXCLUDED.vendor_id
RETURNING *;

-- name: ReserveInventory :one
UPDATE vendor_inventory
SET reserved_quantity = reserved_quantity + sqlc.arg(quantity)::integer
WHERE product_id = sqlc.arg(product_id) AND vendor_id = sqlc.arg(vendor_id)
    AND sqlc.arg(quantity)::integer > 0
    AND quantity - reserved_quantity >= sqlc.arg(quantity)::integer
RETURNING *;

-- name: CreateAddress :one
INSERT INTO addresses (user_id, label, address_line, city, latitude, longitude)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: ListUserAddresses :many
SELECT * FROM addresses WHERE user_id = $1 ORDER BY id;

-- name: CreateOrder :one
INSERT INTO orders (customer_id, vendor_id, address_id, delivery_address, total, currency)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders WHERE id = $1;

-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, vendor_id, product_id, quantity, unit_price)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListOrderItems :many
SELECT * FROM order_items WHERE order_id = $1 ORDER BY id;

-- name: CreatePayment :one
INSERT INTO payments (order_id, amount, currency, reference)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: CreateDelivery :one
INSERT INTO deliveries (order_id, rider_id) VALUES ($1, $2) RETURNING *;

-- name: CreateReview :one
INSERT INTO reviews (order_id, customer_id, vendor_id, rating, comment)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListVendorReviews :many
SELECT * FROM reviews WHERE vendor_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3;
