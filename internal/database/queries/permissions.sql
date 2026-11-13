-- name: CustomerOrders :many
SELECT * FROM orders WHERE customer_id=$1 ORDER BY id DESC LIMIT 100;

-- name: CustomerOrder :one
SELECT * FROM orders WHERE id=$1 AND customer_id=$2;

-- name: CustomerAddress :one
SELECT * FROM addresses WHERE id=$1 AND user_id=$2;

-- name: CustomerPayments :many
SELECT p.* FROM payments p JOIN orders o ON o.id=p.order_id WHERE o.customer_id=$1 ORDER BY p.id DESC LIMIT 100;

-- name: CustomerPayment :one
SELECT p.* FROM payments p JOIN orders o ON o.id=p.order_id WHERE p.id=$1 AND o.customer_id=$2;

-- name: OwnedShop :one
SELECT * FROM vendors WHERE user_id=$1;

-- name: CreateOwnedShop :one
INSERT INTO vendors (user_id,name,address) VALUES ($1,$2,$3) ON CONFLICT (user_id) DO NOTHING RETURNING *;

-- name: UpdateOwnedShop :one
UPDATE vendors SET name=$2,address=$3,address_id=NULL,status='pending',is_active=FALSE,submitted_at=NULL WHERE user_id=$1 AND status<>'suspended' RETURNING *;

-- name: OwnedProducts :many
SELECT p.* FROM products p JOIN vendors v ON v.id=p.vendor_id WHERE v.user_id=$1 ORDER BY p.id DESC LIMIT 100;

-- name: OwnedProduct :one
SELECT p.* FROM products p JOIN vendors v ON v.id=p.vendor_id WHERE p.id=$1 AND v.user_id=$2;

-- name: CreateOwnedProduct :one
INSERT INTO products (vendor_id,name,price)
SELECT id,sqlc.arg(name)::text,sqlc.arg(price)::numeric FROM vendors WHERE user_id=sqlc.arg(user_id)
RETURNING *;

-- name: UpdateOwnedProduct :one
UPDATE products p SET name=sqlc.arg(name),price=sqlc.arg(price)
FROM vendors v WHERE p.id=sqlc.arg(id) AND v.id=p.vendor_id AND v.user_id=sqlc.arg(user_id)
RETURNING p.*;

-- name: DeleteOwnedProduct :execrows
DELETE FROM products p USING vendors v WHERE p.id=$1 AND v.id=p.vendor_id AND v.user_id=$2;

-- name: VendorOrders :many
SELECT o.* FROM orders o JOIN vendors v ON v.id=o.vendor_id WHERE v.user_id=$1 ORDER BY o.id DESC LIMIT 100;

-- name: VendorOrder :one
SELECT o.* FROM orders o JOIN vendors v ON v.id=o.vendor_id WHERE o.id=$1 AND v.user_id=$2;

-- name: AssignedDeliveries :many
SELECT d.id,d.order_id,d.status,d.delivered_at,d.created_at,d.updated_at,o.delivery_address,u.phone
FROM deliveries d JOIN orders o ON o.id=d.order_id JOIN users u ON u.id=o.customer_id
WHERE d.rider_id=$1 ORDER BY d.id DESC LIMIT 100;

-- name: AssignedDelivery :one
SELECT d.id,d.order_id,d.status,d.delivered_at,d.created_at,d.updated_at,o.delivery_address,u.phone
FROM deliveries d JOIN orders o ON o.id=d.order_id JOIN users u ON u.id=o.customer_id
WHERE d.id=$1 AND d.rider_id=$2;

-- name: UpdateAssignedDelivery :one
UPDATE deliveries SET status=sqlc.arg(status)::text,
delivered_at=CASE WHEN sqlc.arg(status)::text='delivered' THEN NOW() ELSE NULL END
WHERE id=sqlc.arg(id) AND rider_id=sqlc.arg(rider_id)
AND ((status='assigned' AND sqlc.arg(status)::text IN ('picked_up','failed')) OR (status='picked_up' AND sqlc.arg(status)::text IN ('delivered','failed')))
RETURNING id,order_id,status,delivered_at,updated_at;

-- name: ManagementUsers :many
SELECT id,name,email,phone,role,email_verified_at,created_at,updated_at FROM users ORDER BY id LIMIT 100;

-- name: ManagementVendors :many
SELECT * FROM vendors ORDER BY id LIMIT 100;

-- name: ManagementVendorStatus :one
UPDATE vendors SET status=$2 WHERE id=$1 RETURNING *;
