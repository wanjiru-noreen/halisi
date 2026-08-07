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

// Get all shops
func (r *ShopRepository) GetShops() ([]models.Shop, error) {

	rows, err := database.DB.Query(`
		SELECT id, name, location, phone, owner_id
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
			&shop.OwnerID,
		)

		if err != nil {
			return nil, err
		}

		shops = append(shops, shop)
	}

	return shops, nil
}

// Get shop by ID
func (r *ShopRepository) GetShopByID(id int) (models.Shop, error) {

	var shop models.Shop

	query := `
	SELECT id, name, location, phone, owner_id
	FROM shops
	WHERE id = ?
	`

	err := database.DB.QueryRow(query, id).Scan(
		&shop.ID,
		&shop.Name,
		&shop.Location,
		&shop.Phone,
		&shop.OwnerID,
	)

	if err != nil {

		if err == sql.ErrNoRows {
			return models.Shop{}, nil
		}

		return models.Shop{}, err
	}

	return shop, nil
}
