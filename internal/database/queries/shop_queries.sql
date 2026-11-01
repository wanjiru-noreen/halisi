-- name: CreateShop :one
INSERT INTO shops (
    name,
    location,
    phone,
    price_6kg,
    price_13kg,
    price_45kg,
    owner_id,
    latitude,
    longitude,
    business_registration_number,
    kra_pin,
    license_number,
    verification_status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetShopByID :one
SELECT * FROM shops
WHERE id = $1;

-- name: GetShopByOwnerID :one
SELECT * FROM shops
WHERE owner_id = $1;

-- name: GetShops :many
SELECT * FROM shops;

-- name: GetVerifiedShops :many
SELECT * FROM shops
WHERE verification_status = 'approved';

-- name: UpdateShopPrices :exec
UPDATE shops
SET
    price_6kg = $1,
    price_13kg = $2,
    price_45kg = $3
WHERE id = $4;

-- name: UpdateShopVerificationStatus :exec
UPDATE shops
SET verification_status = $1
WHERE id = $2;
