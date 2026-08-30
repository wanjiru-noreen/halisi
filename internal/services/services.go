package services

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

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

// RegisterUser creates a new customer or shop owner.
func (s *UserService) RegisterUser(user models.User) (models.User, error) {
	// Clean user input
	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.TrimSpace(strings.ToLower(user.Email))

	// Validate name
	if user.Name == "" {
		return models.User{}, errors.New("name is required")
	}

	// Validate email
	if user.Email == "" {
		return models.User{}, errors.New("email is required")
	}

	// Validate password
	if user.Password == "" {
		return models.User{}, errors.New("password is required")
	}

	// Validate account type
	if user.Role != "customer" && user.Role != "owner" {
		return models.User{}, errors.New("invalid account type")
	}

	// Check if email already exists
	existingUser, err := s.repo.GetUserByEmail(user.Email)
	if err != nil {
		return models.User{}, err
	}

	if existingUser.ID != 0 {
		return models.User{}, errors.New("email is already registered")
	}

	// Hash password before storing it
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return models.User{}, err
	}

	user.Password = string(hashedPassword)

	// Save user
	return s.repo.CreateUser(user)
}

// GetAllUsers returns all registered users.
func (s *UserService) GetAllUsers() ([]models.User, error) {
	return s.repo.GetUsers()
}

// LoginUser authenticates a user using email and password.
func (s *UserService) LoginUser(email, password string) (models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return models.User{}, err
	}

	// User does not exist
	if user.ID == 0 {
		return models.User{}, nil
	}

	// Compare entered password with hashed password
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

// CreateShop creates a new gas shop.
func (s *ShopService) CreateShop(shop models.Shop) (models.Shop, error) {
	return s.repo.CreateShop(shop)
}

// GetShops returns all shops.
func (s *ShopService) GetShops() ([]models.Shop, error) {
	return s.repo.GetShops()
}

// GetShopByID returns a shop by its ID.
func (s *ShopService) GetShopByID(id int) (models.Shop, error) {
	return s.repo.GetShopByID(id)
}

// GetShopByOwnerID returns the shop belonging to an owner.
func (s *ShopService) GetShopByOwnerID(ownerID int) (models.Shop, error) {
	return s.repo.GetShopByOwnerID(ownerID)
}

// UpdateShopPrices updates all gas cylinder prices for a shop.
func (s *ShopService) UpdateShopPrices(
	id int,
	price6kg float64,
	price13kg float64,
	price45kg float64,
) error {
	return s.repo.UpdateShopPrices(
		id,
		price6kg,
		price13kg,
		price45kg,
	)
}
