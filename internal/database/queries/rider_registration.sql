-- name: RegisterRider :one
INSERT INTO rider_profiles(user_id,full_name,phone,city,status)
VALUES($1,$2,$3,$4,'pending') RETURNING *;

-- name: OwnRiderProfile :one
SELECT * FROM rider_profiles WHERE user_id=$1;

-- name: UpdateOwnPendingRider :one
UPDATE rider_profiles
SET full_name=COALESCE(sqlc.narg(full_name)::text,full_name),
phone=COALESCE(sqlc.narg(phone)::text,phone),
city=COALESCE(sqlc.narg(city)::text,city)
WHERE user_id=sqlc.arg(user_id) AND status='pending' RETURNING *;

-- name: RiderIsApproved :one
SELECT EXISTS(SELECT 1 FROM rider_profiles WHERE user_id=$1 AND status='approved');
