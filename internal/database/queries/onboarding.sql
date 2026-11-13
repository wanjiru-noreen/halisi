-- name: LockVendor :one
SELECT * FROM vendors WHERE id=$1 FOR UPDATE;

-- name: ApplicationByOwner :one
SELECT * FROM vendors WHERE user_id=$1;

-- name: SaveApplication :one
INSERT INTO vendors(user_id,name,address,contact_email,contact_phone,city,operating_hours)
VALUES($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT(user_id) DO UPDATE SET name=EXCLUDED.name,address=EXCLUDED.address,address_id=NULL,contact_email=EXCLUDED.contact_email,contact_phone=EXCLUDED.contact_phone,city=EXCLUDED.city,operating_hours=EXCLUDED.operating_hours,status='pending',is_active=FALSE,submitted_at=NULL,review_reason='',reviewed_by=NULL,reviewed_at=NULL
WHERE vendors.status<>'suspended' RETURNING *;

-- name: ApplicationDocuments :many
SELECT id,vendor_id,document_type,status,original_name,content_type,size_bytes,sha256,created_at,updated_at FROM vendor_documents WHERE vendor_id=$1 ORDER BY document_type;

-- name: StoredVendorDocument :one
SELECT * FROM vendor_documents WHERE id=$1;

-- name: SaveUploadedDocument :one
INSERT INTO vendor_documents(vendor_id,document_type,storage_key,original_name,content_type,size_bytes,sha256)
VALUES($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT(vendor_id,document_type) DO UPDATE SET storage_key=EXCLUDED.storage_key,original_name=EXCLUDED.original_name,content_type=EXCLUDED.content_type,size_bytes=EXCLUDED.size_bytes,sha256=EXCLUDED.sha256,status='pending'
RETURNING *;

-- name: ResetApplicationReview :exec
UPDATE vendors SET status='pending',is_active=FALSE,submitted_at=NULL,reviewed_by=NULL,reviewed_at=NULL WHERE id=$1;

-- name: SubmitApplication :one
UPDATE vendors SET status='pending',submitted_at=NOW(),is_active=FALSE,review_reason='',reviewed_by=NULL,reviewed_at=NULL WHERE id=$1 RETURNING *;

-- name: ReviewApplication :one
UPDATE vendors SET status=$2,is_active=($2='approved'),review_reason=$3,reviewed_by=$4,reviewed_at=NOW() WHERE id=$1 RETURNING *;

-- name: ReviewDocuments :exec
UPDATE vendor_documents SET status=$2 WHERE vendor_id=$1;

-- name: RecordVendorReview :exec
INSERT INTO vendor_review_events(vendor_id,administrator_id,status,reason) VALUES($1,$2,$3,$4);

-- name: SearchApprovedVendors :many
SELECT id,name,address,city,contact_email,contact_phone,operating_hours FROM vendors
WHERE status='approved' AND is_active AND strpos(lower(name||' '||city),lower(sqlc.arg(search)::text))>0 ORDER BY name,id LIMIT 50;
