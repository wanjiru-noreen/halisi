package services

import (
	"halisi/internal/models"
	"halisi/internal/repository"
)

// ---------------- USER SERVICE ----------------

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) RegisterUser(user models.User) (models.User, error) {
	return s.repo.CreateUser(user)
}

func (s *UserService) GetAllUsers() ([]models.User, error) {
	return s.repo.GetUsers()
}

func (s *UserService) LoginUser(email string) (models.User, error) {
	return s.repo.GetUserByEmail(email)
}

// ---------------- SHOP SERVICE ----------------

type ShopService struct {
	repo *repository.UserRepository
}

func NewShopService(repo *repository.UserRepository) *ShopService {
	return &ShopService{
		repo: repo,
	}
}

func (s *ShopService) GetShops() ([]models.Shop, error) {
	return s.repo.GetShops()
}

func (s *ShopService) GetShopByID(id int) (models.Shop, error) {
	return s.repo.GetShopByID(id)
}
