package repository

import (
	"database/sql"
	"fmt"

	"halisi/internal/database"
	"halisi/internal/models"
)

type KYCDocumentRepository struct {
	db *sql.DB
}

func NewKYCDocumentRepository() *KYCDocumentRepository {
	return &KYCDocumentRepository{
		db: database.DB,
	}
}

func (r *KYCDocumentRepository) CreateDocument(
	document *models.KYCDocument,
) error {
	query := `
		INSERT INTO kyc_documents (
			shop_id,
			document_type,
			file_path,
			status,
			rejection_reason
		)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		document.ShopID,
		document.DocumentType,
		document.FilePath,
		document.Status,
		document.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("create KYC document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get KYC document ID: %w", err)
	}

	document.ID = int(id)

	return nil
}

func (r *KYCDocumentRepository) GetDocumentsByShopID(
	shopID int,
) ([]models.KYCDocument, error) {
	query := `
		SELECT
			id,
			shop_id,
			document_type,
			file_path,
			status,
			rejection_reason,
			created_at,
			reviewed_at
		FROM kyc_documents
		WHERE shop_id = ?
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.Query(query, shopID)
	if err != nil {
		return nil, fmt.Errorf("get KYC documents: %w", err)
	}
	defer rows.Close()

	var documents []models.KYCDocument

	for rows.Next() {
		var document models.KYCDocument

		err := rows.Scan(
			&document.ID,
			&document.ShopID,
			&document.DocumentType,
			&document.FilePath,
			&document.Status,
			&document.RejectionReason,
			&document.CreatedAt,
			&document.ReviewedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan KYC document: %w", err)
		}

		documents = append(documents, document)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate KYC documents: %w", err)
	}

	return documents, nil
}

func (r *KYCDocumentRepository) GetDocumentByID(
	id int,
) (*models.KYCDocument, error) {
	query := `
		SELECT
			id,
			shop_id,
			document_type,
			file_path,
			status,
			rejection_reason,
			created_at,
			reviewed_at
		FROM kyc_documents
		WHERE id = ?
	`

	var document models.KYCDocument

	err := r.db.QueryRow(query, id).Scan(
		&document.ID,
		&document.ShopID,
		&document.DocumentType,
		&document.FilePath,
		&document.Status,
		&document.RejectionReason,
		&document.CreatedAt,
		&document.ReviewedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("KYC document not found")
		}

		return nil, fmt.Errorf("get KYC document: %w", err)
	}

	return &document, nil
}

func (r *KYCDocumentRepository) UpdateDocumentStatus(
	id int,
	status string,
	rejectionReason string,
) error {
	query := `
		UPDATE kyc_documents
		SET
			status = ?,
			rejection_reason = ?,
			reviewed_at = CURRENT_TIMESTAMP
		WHERE id = ?
		AND status = 'pending'
	`

	result, err := r.db.Exec(
		query,
		status,
		rejectionReason,
		id,
	)
	if err != nil {
		return fmt.Errorf("update KYC document status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check KYC document update: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("KYC document not found or already reviewed")
	}

	return nil
}

func (r *KYCDocumentRepository) GetPendingDocuments() ([]models.KYCDocument, error) {
	query := `
		SELECT
			id,
			shop_id,
			document_type,
			file_path,
			status,
			rejection_reason,
			created_at,
			reviewed_at
		FROM kyc_documents
		WHERE status = 'pending'
		ORDER BY created_at ASC, id ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("get pending KYC documents: %w", err)
	}
	defer rows.Close()

	var documents []models.KYCDocument

	for rows.Next() {
		var document models.KYCDocument

		err := rows.Scan(
			&document.ID,
			&document.ShopID,
			&document.DocumentType,
			&document.FilePath,
			&document.Status,
			&document.RejectionReason,
			&document.CreatedAt,
			&document.ReviewedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pending KYC document: %w", err)
		}

		documents = append(documents, document)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending KYC documents: %w", err)
	}

	return documents, nil
}
