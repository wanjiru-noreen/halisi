package repository

import (
	"context"

	"halisi/internal/database"
	"halisi/internal/database/queried"
	"halisi/internal/models"
)

type OrderRepository struct {
	q *queried.QueriedDB
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{q: queried.NewQueriedDB(database.DB)}
}

// Create order
func (r *OrderRepository) CreateOrder(order models.Order) (models.Order, error) {
	ctx := context.Background()
	result, err := r.q.CreateOrder(ctx, queried.CreateOrderParams{
		UserID:          order.UserID,
		ShopID:          order.ShopID,
		CylinderSize:    order.CylinderSize,
		Quantity:        order.Quantity,
		TotalPrice:      order.TotalPrice,
		Status:          order.Status,
		DeliveryAddress: order.DeliveryAddress,
	})
	if err != nil {
		return models.Order{}, err
	}

	return models.Order{
		ID:              result.ID,
		UserID:          result.UserID,
		ShopID:          result.ShopID,
		CylinderSize:    result.CylinderSize,
		Quantity:        result.Quantity,
		TotalPrice:      result.TotalPrice,
		Status:          result.Status,
		DeliveryAddress: result.DeliveryAddress,
	}, nil
}

// Get all orders
func (r *OrderRepository) GetOrders() ([]models.Order, error) {
	ctx := context.Background()
	results, err := r.q.GetOrders(ctx)
	if err != nil {
		return nil, err
	}

	var orders []models.Order
	for _, result := range results {
		shopName := ""
		if result.ShopName.Valid {
			shopName = result.ShopName.String
		}
		orders = append(orders, models.Order{
			ID:              result.ID,
			UserID:          result.UserID,
			RecipientName:   result.RecipientName,
			ShopID:          result.ShopID,
			CylinderSize:    result.CylinderSize,
			Quantity:        result.Quantity,
			TotalPrice:      result.TotalPrice,
			Status:          result.Status,
			DeliveryAddress: result.DeliveryAddress,
			ShopName:        shopName,
		})
	}

	return orders, nil
}

// Get order by ID
func (r *OrderRepository) GetOrderByID(id int) (models.Order, error) {
	ctx := context.Background()
	result, err := r.q.GetOrderByID(ctx, id)
	if err != nil {
		return models.Order{}, err
	}

	shopName := ""
	if result.ShopName.Valid {
		shopName = result.ShopName.String
	}

	return models.Order{
		ID:              result.ID,
		UserID:          result.UserID,
		RecipientName:   result.RecipientName,
		ShopID:          result.ShopID,
		CylinderSize:    result.CylinderSize,
		Quantity:        result.Quantity,
		TotalPrice:      result.TotalPrice,
		Status:          result.Status,
		DeliveryAddress: result.DeliveryAddress,
		ShopName:        shopName,
	}, nil
}

// Get orders for a specific customer
func (r *OrderRepository) GetOrdersByUserID(userID int) ([]models.Order, error) {
	ctx := context.Background()
	results, err := r.q.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var orders []models.Order
	for _, result := range results {
		shopName := ""
		if result.ShopName.Valid {
			shopName = result.ShopName.String
		}
		orders = append(orders, models.Order{
			ID:              result.ID,
			UserID:          result.UserID,
			ShopID:          result.ShopID,
			CylinderSize:    result.CylinderSize,
			Quantity:        result.Quantity,
			TotalPrice:      result.TotalPrice,
			Status:          result.Status,
			DeliveryAddress: result.DeliveryAddress,
			ShopName:        shopName,
		})
	}

	return orders, nil
}

// Update the status of an existing order.
// This changes the same order instead of creating another order.
func (r *OrderRepository) UpdateOrderStatus(id int, status string) error {
	ctx := context.Background()
	return r.q.UpdateOrderStatus(ctx, queried.UpdateOrderStatusParams{
		Status: status,
		ID:     id,
	})
}

// Get orders belonging to a specific shop
func (r *OrderRepository) GetOrdersByShopID(shopID int) ([]models.Order, error) {
	ctx := context.Background()
	results, err := r.q.GetOrdersByShopID(ctx, shopID)
	if err != nil {
		return nil, err
	}

	var orders []models.Order
	for _, result := range results {
		shopName := ""
		if result.ShopName.Valid {
			shopName = result.ShopName.String
		}
		orders = append(orders, models.Order{
			ID:              result.ID,
			UserID:          result.UserID,
			ShopID:          result.ShopID,
			CylinderSize:    result.CylinderSize,
			Quantity:        result.Quantity,
			TotalPrice:      result.TotalPrice,
			Status:          result.Status,
			DeliveryAddress: result.DeliveryAddress,
			ShopName:        shopName,
		})
	}

	return orders, nil
}

// Delete a pending order
func (r *OrderRepository) DeleteOrder(id int) error {
	ctx := context.Background()
	return r.q.DeleteOrder(ctx, id)
}
