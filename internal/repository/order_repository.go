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
		status,
		delivery_address
	)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := database.DB.Exec(
		query,
		order.UserID,
		order.ShopID,
		order.CylinderSize,
		order.Quantity,
		order.TotalPrice,
		order.Status,
		order.DeliveryAddress,
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
			o.id,
			o.user_id,
			o.shop_id,
			o.cylinder_size,
			o.quantity,
			o.total_price,
			o.status,
			o.delivery_address,
			s.name
		FROM orders o
		LEFT JOIN shops s ON o.shop_id = s.id
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
			&order.DeliveryAddress,
			&order.ShopName,
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
		o.id,
		o.user_id,
		o.shop_id,
		o.cylinder_size,
		o.quantity,
		o.total_price,
		o.status,
		o.delivery_address,
		s.name
	FROM orders o
	LEFT JOIN shops s ON o.shop_id = s.id
	WHERE o.id = ?
	`

	err := database.DB.QueryRow(query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.ShopID,
		&order.CylinderSize,
		&order.Quantity,
		&order.TotalPrice,
		&order.Status,
		&order.DeliveryAddress,
		&order.ShopName,
	)

	if err != nil {
		return models.Order{}, err
	}

	return order, nil
}

func (r *OrderRepository) GetOrdersByUserID(userID int) ([]models.Order, error) {

	rows, err := database.DB.Query(`
		SELECT o.id,
		       o.user_id,
		       o.shop_id,
		       o.cylinder_size,
		       o.quantity,
		       o.total_price,
		       o.status,
		       o.delivery_address,
		       s.name
		FROM orders o
		LEFT JOIN shops s ON o.shop_id = s.id
		WHERE o.user_id = ?
	`, userID)

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
			&order.DeliveryAddress,
			&order.ShopName,
		)

		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) UpdateOrderStatus(id int, status string) error {

	_, err := database.DB.Exec(`
		UPDATE orders
		SET status = ?
		WHERE id = ?
	`, status, id)

	return err
}

func (r *OrderRepository) GetOrdersByShopID(shopID int) ([]models.Order, error) {

	rows, err := database.DB.Query(`
		SELECT o.id,
		       o.user_id,
		       o.shop_id,
		       o.cylinder_size,
		       o.quantity,
		       o.total_price,
		       o.status,
		       o.delivery_address,
		       s.name
		FROM orders o
		LEFT JOIN shops s ON o.shop_id = s.id
		WHERE o.shop_id = ?
	`, shopID)

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
			&order.DeliveryAddress,
			&order.ShopName,
		)
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (r *OrderRepository) DeleteOrder(id int) error {

	_, err := database.DB.Exec(`
		DELETE FROM orders
		WHERE id = ?
		  AND status = 'Pending'
	`, id)

	return err
}
