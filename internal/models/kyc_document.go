package models

type KYCDocument struct {
	ID              int
	ShopID          int
	DocumentType    string
	FilePath        string
	Status          string
	RejectionReason string
	CreatedAt       string
	ReviewedAt      *string
}
