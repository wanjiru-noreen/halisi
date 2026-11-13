-- name: AdminDashboardAccounts :one
SELECT
 (SELECT count(*) FROM users WHERE role='customer') AS customers_total,
 (SELECT count(*) FROM users WHERE role='customer' AND created_at>=sqlc.arg(start_at)::timestamptz AND created_at<sqlc.arg(end_at)::timestamptz) AS customers_added,
 (SELECT count(*) FROM vendors) AS vendors_total,
 (SELECT count(*) FROM vendors WHERE created_at>=sqlc.arg(start_at)::timestamptz AND created_at<sqlc.arg(end_at)::timestamptz) AS vendors_added,
 (SELECT count(*) FROM vendors WHERE status='pending') AS vendors_pending,
 (SELECT count(*) FROM vendors WHERE status='approved' AND is_active) AS vendors_active,
 (SELECT count(*) FROM rider_profiles) AS riders_total,
 (SELECT count(*) FROM rider_profiles WHERE created_at>=sqlc.arg(start_at)::timestamptz AND created_at<sqlc.arg(end_at)::timestamptz) AS riders_added,
 (SELECT count(*) FROM rider_profiles WHERE status='pending') AS riders_pending,
 (SELECT count(*) FROM rider_profiles rp JOIN users u ON u.id=rp.user_id WHERE rp.status='approved' AND u.role='rider') AS riders_approved;

-- name: AdminDashboardOrders :many
SELECT status,count(*) AS count FROM orders
WHERE created_at>=sqlc.arg(start_at)::timestamptz AND created_at<sqlc.arg(end_at)::timestamptz
GROUP BY status ORDER BY status;

-- name: AdminDashboardFinances :many
SELECT currency,
 count(*) AS payment_count,
 sum(amount)::text AS gross_sales,
 coalesce(sum(amount) FILTER(WHERE status='refunded'),0)::text AS refunds,
 coalesce(sum(amount) FILTER(WHERE status='completed'),0)::text AS net_collections
FROM payments
WHERE status IN ('completed','refunded')
AND created_at>=sqlc.arg(start_at)::timestamptz AND created_at<sqlc.arg(end_at)::timestamptz
GROUP BY currency ORDER BY currency;

-- name: AdminDashboardActivity :many
SELECT event_type,resource_id,status,occurred_at FROM (
 SELECT 'customer_registered'::text AS event_type,id AS resource_id,'registered'::text AS status,created_at AS occurred_at FROM users WHERE role='customer'
 UNION ALL
 SELECT 'vendor_registered',id,status,created_at FROM vendors
 UNION ALL
 SELECT 'rider_applied',id,status,created_at FROM rider_profiles
 UNION ALL
 SELECT 'order_created',id,status,created_at FROM orders
 UNION ALL
 SELECT 'payment_created',id,status,created_at FROM payments
) activity
WHERE occurred_at>=sqlc.arg(start_at)::timestamptz AND occurred_at<sqlc.arg(end_at)::timestamptz
ORDER BY occurred_at DESC,event_type,resource_id DESC LIMIT 20;
