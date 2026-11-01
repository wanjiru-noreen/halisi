-- name: CreateKYCDocument :one
INSERT INTO kyc_documents (
    shop_id,
    document_type,
    file_path,
    status,
    rejection_reason
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetKYCDocumentByID :one
SELECT * FROM kyc_documents
WHERE id = $1;

-- name: GetKYCDocumentsByShopID :many
SELECT * FROM kyc_documents
WHERE shop_id = $1
ORDER BY created_at DESC, id DESC;

-- name: GetPendingKYCDocuments :many
SELECT * FROM kyc_documents
WHERE status = 'pending'
ORDER BY created_at ASC, id ASC;

-- name: UpdateKYCDocumentStatus :exec
UPDATE kyc_documents
SET
    status = $1,
    rejection_reason = $2,
    reviewed_at = CURRENT_TIMESTAMP
WHERE id = $3
AND status = 'pending';
