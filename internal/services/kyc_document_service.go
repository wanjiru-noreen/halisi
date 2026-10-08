package services

import (
	"fmt"
	"strings"

	"halisi/internal/models"
	"halisi/internal/repository"
)

type KYCDocumentService struct {
	repository *repository.KYCDocumentRepository
}

func NewKYCDocumentService(
	repository *repository.KYCDocumentRepository,
) *KYCDocumentService {
	return &KYCDocumentService{
		repository: repository,
	}
}

func (s *KYCDocumentService) SubmitDocument(
	shopID int,
	documentType string,
	filePath string,
) (*models.KYCDocument, error) {
	documentType = strings.TrimSpace(documentType)
	filePath = strings.TrimSpace(filePath)

	if shopID <= 0 {
		return nil, fmt.Errorf("invalid shop ID")
	}

	if documentType == "" {
		return nil, fmt.Errorf("document type is required")
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	document := &models.KYCDocument{
		ShopID:          shopID,
		DocumentType:    documentType,
		FilePath:        filePath,
		Status:          "pending",
		RejectionReason: "",
	}

	err := s.repository.CreateDocument(document)
	if err != nil {
		return nil, err
	}

	return document, nil
}

func (s *KYCDocumentService) GetShopDocuments(
	shopID int,
) ([]models.KYCDocument, error) {
	if shopID <= 0 {
		return nil, fmt.Errorf("invalid shop ID")
	}

	return s.repository.GetDocumentsByShopID(shopID)
}

func (s *KYCDocumentService) GetDocument(
	id int,
) (*models.KYCDocument, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid document ID")
	}

	return s.repository.GetDocumentByID(id)
}

func (s *KYCDocumentService) ReviewDocument(
	id int,
	status string,
	rejectionReason string,
) error {
	if id <= 0 {
		return fmt.Errorf("invalid document ID")
	}

	status = strings.ToLower(strings.TrimSpace(status))
	rejectionReason = strings.TrimSpace(rejectionReason)

	if status != "approved" && status != "rejected" {
		return fmt.Errorf("status must be approved or rejected")
	}

	if status == "rejected" && rejectionReason == "" {
		return fmt.Errorf("rejection reason is required")
	}

	if status == "approved" {
		rejectionReason = ""
	}

	return s.repository.UpdateDocumentStatus(
		id,
		status,
		rejectionReason,
	)
}

func (s *KYCDocumentService) GetPendingDocuments() ([]models.KYCDocument, error) {
	return s.repository.GetPendingDocuments()
}
