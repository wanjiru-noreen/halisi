package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"halisi/internal/database"
	"halisi/internal/database/queried"
	"halisi/internal/models"
)

type KYCDocumentRepository struct {
	q *queried.QueriedDB
}

func NewKYCDocumentRepository() *KYCDocumentRepository {
	return &KYCDocumentRepository{
		q: queried.NewQueriedDB(database.DB),
	}
}

func (r *KYCDocumentRepository) CreateDocument(
	document *models.KYCDocument,
) error {
	ctx := context.Background()
	result, err := r.q.CreateKYCDocument(ctx, queried.CreateKYCDocumentParams{
		ShopID:          document.ShopID,
		DocumentType:    document.DocumentType,
		FilePath:        document.FilePath,
		Status:          document.Status,
		RejectionReason: document.RejectionReason,
	})
	if err != nil {
		return fmt.Errorf("create KYC document: %w", err)
	}

	document.ID = result.ID
	document.CreatedAt = result.CreatedAt.Format(time.RFC3339)
	if result.ReviewedAt.Valid {
		v := result.ReviewedAt.Time.Format(time.RFC3339)
		document.ReviewedAt = &v
	}

	return nil
}

func (r *KYCDocumentRepository) GetDocumentsByShopID(
	shopID int,
) ([]models.KYCDocument, error) {
	ctx := context.Background()
	results, err := r.q.GetKYCDocumentsByShopID(ctx, shopID)
	if err != nil {
		return nil, fmt.Errorf("get KYC documents: %w", err)
	}

	var documents []models.KYCDocument
	for _, result := range results {
		doc := models.KYCDocument{
			ID:              result.ID,
			ShopID:          result.ShopID,
			DocumentType:    result.DocumentType,
			FilePath:        result.FilePath,
			Status:          result.Status,
			RejectionReason: result.RejectionReason,
			CreatedAt:       result.CreatedAt.Format(time.RFC3339),
		}
		if result.ReviewedAt.Valid {
			v := result.ReviewedAt.Time.Format(time.RFC3339)
			doc.ReviewedAt = &v
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

func (r *KYCDocumentRepository) GetDocumentByID(
	id int,
) (*models.KYCDocument, error) {
	ctx := context.Background()
	result, err := r.q.GetKYCDocumentByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("KYC document not found")
		}
		return nil, fmt.Errorf("get KYC document: %w", err)
	}

	doc := models.KYCDocument{
		ID:              result.ID,
		ShopID:          result.ShopID,
		DocumentType:    result.DocumentType,
		FilePath:        result.FilePath,
		Status:          result.Status,
		RejectionReason: result.RejectionReason,
		CreatedAt:       result.CreatedAt.Format(time.RFC3339),
	}
	if result.ReviewedAt.Valid {
		v := result.ReviewedAt.Time.Format(time.RFC3339)
		doc.ReviewedAt = &v
	}

	return &doc, nil
}

func (r *KYCDocumentRepository) UpdateDocumentStatus(
	id int,
	status string,
	rejectionReason string,
) error {
	ctx := context.Background()
	err := r.q.UpdateKYCDocumentStatus(ctx, queried.UpdateKYCDocumentStatusParams{
		Status:          status,
		RejectionReason: rejectionReason,
		ID:              id,
	})
	if err != nil {
		return fmt.Errorf("update KYC document status: %w", err)
	}

	// Check if the document was actually updated
	var count int
	err = r.q.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM kyc_documents WHERE id = $1 AND status = $2", id, status).Scan(&count)
	if err != nil {
		return fmt.Errorf("check KYC document update: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("KYC document not found or already reviewed")
	}

	return nil
}

func (r *KYCDocumentRepository) GetPendingDocuments() ([]models.KYCDocument, error) {
	ctx := context.Background()
	results, err := r.q.GetPendingKYCDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pending KYC documents: %w", err)
	}

	var documents []models.KYCDocument
	for _, result := range results {
		doc := models.KYCDocument{
			ID:              result.ID,
			ShopID:          result.ShopID,
			DocumentType:    result.DocumentType,
			FilePath:        result.FilePath,
			Status:          result.Status,
			RejectionReason: result.RejectionReason,
			CreatedAt:       result.CreatedAt.Format(time.RFC3339),
		}
		if result.ReviewedAt.Valid {
			v := result.ReviewedAt.Time.Format(time.RFC3339)
			doc.ReviewedAt = &v
		}
		documents = append(documents, doc)
	}

	return documents, nil
}
