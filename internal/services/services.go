package services

import (
	"halisi/internal/models"
	"halisi/internal/repository"

	"golang.org/x/crypto/bcrypt"
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

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return models.User{}, err
	}

	user.Password = string(hashedPassword)

	return s.repo.CreateUser(user)
}

func (s *UserService) GetAllUsers() ([]models.User, error) {
	return s.repo.GetUsers()
}

func (s *UserService) LoginUser(email, password string) (models.User, error) {

	user, err := s.repo.GetUserByEmail(email)

	if err != nil {
		return models.User{}, err
	}

	if user.ID == 0 {
		return models.User{}, nil
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	)

	if err != nil {
		return models.User{}, nil
	}

	return user, nil
}


// ---------------- SHOP SERVICE ----------------

type ShopService struct {
	repo *repository.ShopRepository
}

func NewShopService(repo *repository.ShopRepository) *ShopService {
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
