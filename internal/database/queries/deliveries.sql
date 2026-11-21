-- name: DeliveryForUser :one
SELECT d.* FROM deliveries d JOIN vendors v ON v.id=d.vendor_id
WHERE d.id=sqlc.arg(id) AND (
    (sqlc.arg(role)::text='customer' AND d.customer_id=sqlc.arg(user_id)) OR
    (sqlc.arg(role)::text='vendor' AND v.user_id=sqlc.arg(user_id)) OR
    (sqlc.arg(role)::text='admin') OR
    (sqlc.arg(role)::text='rider' AND d.rider_id=sqlc.arg(user_id)
     AND EXISTS(SELECT 1 FROM rider_profiles rp WHERE rp.user_id=sqlc.arg(user_id) AND rp.status='approved'))
);

-- name: ConfirmOwnedOrder :one
UPDATE orders o SET status='confirmed'
FROM vendors v WHERE o.id=sqlc.arg(id) AND o.vendor_id=v.id AND v.user_id=sqlc.arg(user_id)
AND v.status='approved' AND v.is_active AND o.address_id IS NOT NULL AND o.status='pending'
RETURNING o.*;
