package repository

import (
	"database/sql"

	"halisi/internal/database"
	"halisi/internal/models"
)

type ShopRepository struct{}

func NewShopRepository() *ShopRepository {
	return &ShopRepository{}
}

// Create a new shop
func (r *ShopRepository) CreateShop(shop models.Shop) (models.Shop, error) {
	result, err := database.DB.Exec(`
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		shop.Name,
		shop.Location,
		shop.Phone,
		shop.Price6kg,
		shop.Price13kg,
		shop.Price45kg,
		shop.OwnerID,
		shop.Latitude,
		shop.Longitude,
		shop.BusinessRegistrationNumber,
		shop.KRAPin,
		shop.LicenseNumber,
		shop.VerificationStatus,
	)

	if err != nil {
		return models.Shop{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Shop{}, err
	}

	shop.ID = int(id)

	return shop, nil
}

// Update shop prices
func (r *ShopRepository) UpdateShopPrices(
	id int,
	price6kg float64,
	price13kg float64,
	price45kg float64,
) error {
	_, err := database.DB.Exec(`
		UPDATE shops
		SET
			price_6kg = ?,
			price_13kg = ?,
			price_45kg = ?
		WHERE id = ?
	`,
		price6kg,
		price13kg,
		price45kg,
		id,
	)

	return err
}

// Get all shops
func (r *ShopRepository) GetShops() ([]models.Shop, error) {
	rows, err := database.DB.Query(`
		SELECT
			id,
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
		FROM shops
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var shops []models.Shop

	for rows.Next() {
		var shop models.Shop

		err := rows.Scan(
			&shop.ID,
			&shop.Name,
			&shop.Location,
			&shop.Phone,
			&shop.Price6kg,
			&shop.Price13kg,
			&shop.Price45kg,
			&shop.OwnerID,
			&shop.Latitude,
			&shop.Longitude,
			&shop.BusinessRegistrationNumber,
			&shop.KRAPin,
			&shop.LicenseNumber,
			&shop.VerificationStatus,
		)

		if err != nil {
			return nil, err
		}

		shops = append(shops, shop)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return shops, nil
}

// Get only verified shops
func (r *ShopRepository) GetVerifiedShops() ([]models.Shop, error) {
	rows, err := database.DB.Query(`
		SELECT
			id,
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
		FROM shops
		WHERE verification_status = 'approved'
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var shops []models.Shop

	for rows.Next() {
		var shop models.Shop

		err := rows.Scan(
			&shop.ID,
			&shop.Name,
			&shop.Location,
			&shop.Phone,
			&shop.Price6kg,
			&shop.Price13kg,
			&shop.Price45kg,
			&shop.OwnerID,
			&shop.Latitude,
			&shop.Longitude,
			&shop.BusinessRegistrationNumber,
			&shop.KRAPin,
			&shop.LicenseNumber,
			&shop.VerificationStatus,
		)

		if err != nil {
			return nil, err
		}

		shops = append(shops, shop)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return shops, nil
}

// Get shop by ID
func (r *ShopRepository) GetShopByID(id int) (models.Shop, error) {
	var shop models.Shop

	query := `
		SELECT
			id,
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
		FROM shops
		WHERE id = ?
	`

	err := database.DB.QueryRow(query, id).Scan(
		&shop.ID,
		&shop.Name,
		&shop.Location,
		&shop.Phone,
		&shop.Price6kg,
		&shop.Price13kg,
		&shop.Price45kg,
		&shop.OwnerID,
		&shop.Latitude,
		&shop.Longitude,
		&shop.BusinessRegistrationNumber,
		&shop.KRAPin,
		&shop.LicenseNumber,
		&shop.VerificationStatus,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Shop{}, nil
		}

		return models.Shop{}, err
	}

	return shop, nil
}

// Get shop by owner ID
func (r *ShopRepository) GetShopByOwnerID(ownerID int) (models.Shop, error) {
	var shop models.Shop

	query := `
		SELECT
			id,
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
		FROM shops
		WHERE owner_id = ?
	`

	err := database.DB.QueryRow(query, ownerID).Scan(
		&shop.ID,
		&shop.Name,
		&shop.Location,
		&shop.Phone,
		&shop.Price6kg,
		&shop.Price13kg,
		&shop.Price45kg,
		&shop.OwnerID,
		&shop.Latitude,
		&shop.Longitude,
		&shop.BusinessRegistrationNumber,
		&shop.KRAPin,
		&shop.LicenseNumber,
		&shop.VerificationStatus,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Shop{}, nil
		}

		return models.Shop{}, err
	}

	return shop, nil
}

// Update shop verification status
func (r *ShopRepository) UpdateShopVerificationStatus(
	id int,
	status string,
) error {
	_, err := database.DB.Exec(`
		UPDATE shops
		SET verification_status = ?
		WHERE id = ?
	`,
		status,
		id,
	)

	return err
}
