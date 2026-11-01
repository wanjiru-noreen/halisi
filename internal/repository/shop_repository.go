package repository

import (
	"context"
	"database/sql"

	"halisi/internal/database"
	"halisi/internal/database/queried"
	"halisi/internal/models"
)

type ShopRepository struct {
	q *queried.QueriedDB
}

func NewShopRepository() *ShopRepository {
	return &ShopRepository{q: queried.NewQueriedDB(database.DB)}
}

// Create a new shop
func (r *ShopRepository) CreateShop(shop models.Shop) (models.Shop, error) {
	ctx := context.Background()
	ownerID := sql.NullInt64{}
	if shop.OwnerID != nil {
		ownerID = sql.NullInt64{Int64: int64(*shop.OwnerID), Valid: true}
	}

	result, err := r.q.CreateShop(ctx, queried.CreateShopParams{
		Name:                       shop.Name,
		Location:                   shop.Location,
		Phone:                      shop.Phone,
		Price6kg:                   shop.Price6kg,
		Price13kg:                  shop.Price13kg,
		Price45kg:                  shop.Price45kg,
		OwnerID:                    ownerID,
		Latitude:                   shop.Latitude,
		Longitude:                  shop.Longitude,
		BusinessRegistrationNumber: shop.BusinessRegistrationNumber,
		KRAPin:                     shop.KRAPin,
		LicenseNumber:              shop.LicenseNumber,
		VerificationStatus:         shop.VerificationStatus,
	})
	if err != nil {
		return models.Shop{}, err
	}

	ownerIDPtr := (*int)(nil)
	if result.OwnerID.Valid {
		v := int(result.OwnerID.Int64)
		ownerIDPtr = &v
	}

	return models.Shop{
		ID:                         result.ID,
		Name:                       result.Name,
		Location:                   result.Location,
		Phone:                      result.Phone,
		Price6kg:                   result.Price6kg,
		Price13kg:                  result.Price13kg,
		Price45kg:                  result.Price45kg,
		OwnerID:                    ownerIDPtr,
		Latitude:                   result.Latitude,
		Longitude:                  result.Longitude,
		BusinessRegistrationNumber: result.BusinessRegistrationNumber,
		KRAPin:                     result.KRAPin,
		LicenseNumber:              result.LicenseNumber,
		VerificationStatus:         result.VerificationStatus,
	}, nil
}

// Update shop prices
func (r *ShopRepository) UpdateShopPrices(
	id int,
	price6kg float64,
	price13kg float64,
	price45kg float64,
) error {
	ctx := context.Background()
	return r.q.UpdateShopPrices(ctx, queried.UpdateShopPricesParams{
		Price6kg:  price6kg,
		Price13kg: price13kg,
		Price45kg: price45kg,
		ID:        id,
	})
}

// Get all shops
func (r *ShopRepository) GetShops() ([]models.Shop, error) {
	ctx := context.Background()
	results, err := r.q.GetShops(ctx)
	if err != nil {
		return nil, err
	}

	var shops []models.Shop
	for _, result := range results {
		ownerIDPtr := (*int)(nil)
		if result.OwnerID.Valid {
			v := int(result.OwnerID.Int64)
			ownerIDPtr = &v
		}
		shops = append(shops, models.Shop{
			ID:                         result.ID,
			Name:                       result.Name,
			Location:                   result.Location,
			Phone:                      result.Phone,
			Price6kg:                   result.Price6kg,
			Price13kg:                  result.Price13kg,
			Price45kg:                  result.Price45kg,
			OwnerID:                    ownerIDPtr,
			Latitude:                   result.Latitude,
			Longitude:                  result.Longitude,
			BusinessRegistrationNumber: result.BusinessRegistrationNumber,
			KRAPin:                     result.KRAPin,
			LicenseNumber:              result.LicenseNumber,
			VerificationStatus:         result.VerificationStatus,
		})
	}

	return shops, nil
}

// Get only verified shops
func (r *ShopRepository) GetVerifiedShops() ([]models.Shop, error) {
	ctx := context.Background()
	results, err := r.q.GetVerifiedShops(ctx)
	if err != nil {
		return nil, err
	}

	var shops []models.Shop
	for _, result := range results {
		ownerIDPtr := (*int)(nil)
		if result.OwnerID.Valid {
			v := int(result.OwnerID.Int64)
			ownerIDPtr = &v
		}
		shops = append(shops, models.Shop{
			ID:                         result.ID,
			Name:                       result.Name,
			Location:                   result.Location,
			Phone:                      result.Phone,
			Price6kg:                   result.Price6kg,
			Price13kg:                  result.Price13kg,
			Price45kg:                  result.Price45kg,
			OwnerID:                    ownerIDPtr,
			Latitude:                   result.Latitude,
			Longitude:                  result.Longitude,
			BusinessRegistrationNumber: result.BusinessRegistrationNumber,
			KRAPin:                     result.KRAPin,
			LicenseNumber:              result.LicenseNumber,
			VerificationStatus:         result.VerificationStatus,
		})
	}

	return shops, nil
}

// Get shop by ID
func (r *ShopRepository) GetShopByID(id int) (models.Shop, error) {
	ctx := context.Background()
	result, err := r.q.GetShopByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Shop{}, nil
		}
		return models.Shop{}, err
	}

	ownerIDPtr := (*int)(nil)
	if result.OwnerID.Valid {
		v := int(result.OwnerID.Int64)
		ownerIDPtr = &v
	}

	return models.Shop{
		ID:                         result.ID,
		Name:                       result.Name,
		Location:                   result.Location,
		Phone:                      result.Phone,
		Price6kg:                   result.Price6kg,
		Price13kg:                  result.Price13kg,
		Price45kg:                  result.Price45kg,
		OwnerID:                    ownerIDPtr,
		Latitude:                   result.Latitude,
		Longitude:                  result.Longitude,
		BusinessRegistrationNumber: result.BusinessRegistrationNumber,
		KRAPin:                     result.KRAPin,
		LicenseNumber:              result.LicenseNumber,
		VerificationStatus:         result.VerificationStatus,
	}, nil
}

// Get shop by owner ID
func (r *ShopRepository) GetShopByOwnerID(ownerID int) (models.Shop, error) {
	ctx := context.Background()
	result, err := r.q.GetShopByOwnerID(ctx, ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Shop{}, nil
		}
		return models.Shop{}, err
	}

	ownerIDPtr := (*int)(nil)
	if result.OwnerID.Valid {
		v := int(result.OwnerID.Int64)
		ownerIDPtr = &v
	}

	return models.Shop{
		ID:                         result.ID,
		Name:                       result.Name,
		Location:                   result.Location,
		Phone:                      result.Phone,
		Price6kg:                   result.Price6kg,
		Price13kg:                  result.Price13kg,
		Price45kg:                  result.Price45kg,
		OwnerID:                    ownerIDPtr,
		Latitude:                   result.Latitude,
		Longitude:                  result.Longitude,
		BusinessRegistrationNumber: result.BusinessRegistrationNumber,
		KRAPin:                     result.KRAPin,
		LicenseNumber:              result.LicenseNumber,
		VerificationStatus:         result.VerificationStatus,
	}, nil
}

// Update shop verification status
func (r *ShopRepository) UpdateShopVerificationStatus(
	id int,
	status string,
) error {
	ctx := context.Background()
	return r.q.UpdateShopVerificationStatus(ctx, queried.UpdateShopVerificationStatusParams{
		VerificationStatus: status,
		ID:                 id,
	})
}
