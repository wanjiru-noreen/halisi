package repository

import (
	"halisi/internal/database"
	"halisi/internal/models"
)

type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

// Create order
func (r *OrderRepository) CreateOrder(order models.Order) (models.Order, error) {

	query := `
	INSERT INTO orders(
		user_id,
		shop_id,
		cylinder_size,
		quantity,
		total_price,
		status
	)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := database.DB.Exec(
		query,
		order.UserID,
		order.ShopID,
		order.CylinderSize,
		order.Quantity,
		order.TotalPrice,
		order.Status,
	)

	if err != nil {
		return models.Order{}, err
	}

	id, err := result.LastInsertId()

	if err != nil {
		return models.Order{}, err
	}

	order.ID = int(id)

	return order, nil
}

// Get all orders
func (r *OrderRepository) GetOrders() ([]models.Order, error) {

	rows, err := database.DB.Query(`
		SELECT 
			id,
			user_id,
			shop_id,
			cylinder_size,
			quantity,
			total_price,
			status
		FROM orders
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var orders []models.Order

	for rows.Next() {

		var order models.Order

		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.ShopID,
			&order.CylinderSize,
			&order.Quantity,
			&order.TotalPrice,
			&order.Status,
		)

		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// Get order by ID
func (r *OrderRepository) GetOrderByID(id int) (models.Order, error) {

	var order models.Order

	query := `
	SELECT
		id,
		user_id,
		shop_id,
		cylinder_size,
		quantity,
		total_price,
		status
	FROM orders
	WHERE id = ?
	`

	err := database.DB.QueryRow(query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.ShopID,
		&order.CylinderSize,
		&order.Quantity,
		&order.TotalPrice,
		&order.Status,
	)

	if err != nil {
		return models.Order{}, err
	}

	return order, nil
}
