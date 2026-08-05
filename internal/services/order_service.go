package services

import (
	"halisi/internal/models"
	"halisi/internal/repository"
)

type OrderService struct {
	repo *repository.OrderRepository
}

func NewOrderService(repo *repository.OrderRepository) *OrderService {
	return &OrderService{
		repo: repo,
	}
}

func (s *OrderService) CreateOrder(order models.Order) (models.Order, error) {

	// Default status for new orders
	if order.Status == "" {
		order.Status = "Pending"
	}

	return s.repo.CreateOrder(order)
}

func (s *OrderService) GetOrders() ([]models.Order, error) {
	return s.repo.GetOrders()
}

func (s *OrderService) GetOrderByID(id int) (models.Order, error) {
	return s.repo.GetOrderByID(id)
}
