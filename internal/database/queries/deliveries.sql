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

-- name: AssignOwnedDelivery :one
UPDATE deliveries d
SET rider_id = sqlc.arg(rider_id),
    status = 'assigned'
FROM vendors v
WHERE d.id = sqlc.arg(id)
  AND d.vendor_id = v.id
  AND v.user_id = sqlc.arg(vendor_user_id)
  AND d.status IN ('pending', 'assigned')
  AND EXISTS (
      SELECT 1 FROM users u
      JOIN rider_profiles rp ON rp.user_id = u.id
      WHERE u.id = sqlc.arg(rider_id) AND u.role = 'rider' AND rp.status = 'approved'
  )
RETURNING d.*;

-- name: AdminAssignDelivery :one
UPDATE deliveries d
SET rider_id = sqlc.arg(rider_id),
    status = 'assigned'
WHERE d.id = sqlc.arg(id)
  AND d.status IN ('pending', 'assigned')
  AND EXISTS (
      SELECT 1 FROM users u
      JOIN rider_profiles rp ON rp.user_id = u.id
      WHERE u.id = sqlc.arg(rider_id) AND u.role = 'rider' AND rp.status = 'approved'
  )
RETURNING d.*;

-- name: ListApprovedRiders :many
SELECT u.id AS user_id, rp.full_name, rp.phone, rp.city, rp.status, rp.created_at
FROM users u
JOIN rider_profiles rp ON rp.user_id = u.id
WHERE u.role = 'rider' AND rp.status = 'approved'
ORDER BY rp.full_name ASC
LIMIT 100;

-- name: DeliveryWithRider :one
SELECT d.*, rp.full_name AS rider_name, rp.phone AS rider_phone
FROM deliveries d
LEFT JOIN rider_profiles rp ON rp.user_id = d.rider_id
WHERE d.id = $1;
