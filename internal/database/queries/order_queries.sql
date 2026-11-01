-- name: CreateOrder :one
INSERT INTO orders (
    user_id,
    shop_id,
    cylinder_size,
    quantity,
    total_price,
    status,
    delivery_address
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetOrderByID :one
SELECT
    o.id,
    o.user_id,
    o.shop_id,
    o.cylinder_size,
    o.quantity,
    o.total_price,
    o.status,
    o.delivery_address,
    s.name AS shop_name,
    COALESCE(u.name, '') AS recipient_name
FROM orders o
LEFT JOIN shops s ON o.shop_id = s.id
LEFT JOIN users u ON o.user_id = u.id
WHERE o.id = $1;

-- name: GetOrders :many
SELECT
    o.id,
    o.user_id,
    o.shop_id,
    o.cylinder_size,
    o.quantity,
    o.total_price,
    o.status,
    o.delivery_address,
    s.name AS shop_name,
    COALESCE(u.name, '') AS recipient_name
FROM orders o
LEFT JOIN shops s ON o.shop_id = s.id
LEFT JOIN users u ON o.user_id = u.id
ORDER BY o.id DESC;

-- name: GetOrdersByUserID :many
SELECT
    o.id,
    o.user_id,
    o.shop_id,
    o.cylinder_size,
    o.quantity,
    o.total_price,
    o.status,
    o.delivery_address,
    s.name AS shop_name
FROM orders o
LEFT JOIN shops s ON o.shop_id = s.id
WHERE o.user_id = $1
ORDER BY o.id DESC;

-- name: GetOrdersByShopID :many
SELECT
    o.id,
    o.user_id,
    o.shop_id,
    o.cylinder_size,
    o.quantity,
    o.total_price,
    o.status,
    o.delivery_address,
    s.name AS shop_name
FROM orders o
LEFT JOIN shops s ON o.shop_id = s.id
WHERE o.shop_id = $1
ORDER BY o.id DESC;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $1
WHERE id = $2;

-- name: DeleteOrder :exec
DELETE FROM orders
WHERE id = $1
AND status = 'Pending';
