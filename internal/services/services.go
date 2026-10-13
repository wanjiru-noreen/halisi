package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"halisi/internal/models"
	"halisi/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) RegisterUser(user models.User) (models.User, error) {
	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.TrimSpace(strings.ToLower(user.Email))

	if user.Name == "" {
		return models.User{}, errors.New("name is required")
	}

	if user.Email == "" {
		return models.User{}, errors.New("email is required")
	}

	if _, err := mail.ParseAddress(user.Email); err != nil {
		return models.User{}, errors.New("invalid email address")
	}

	if user.Password == "" {
		return models.User{}, errors.New("password is required")
	}

	if user.Role != "customer" && user.Role != "owner" {
		return models.User{}, errors.New("invalid account type")
	}

	existingUser, err := s.repo.GetUserByEmail(user.Email)
	if err != nil {
		return models.User{}, err
	}

	if existingUser.ID != 0 && existingUser.EmailVerified {
		return models.User{}, errors.New("email is already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return models.User{}, err
	}

	user.Password = string(hashedPassword)

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return models.User{}, errors.New(
			"failed to generate verification token",
		)
	}

	user.EmailVerified = false
	user.VerificationToken = hex.EncodeToString(tokenBytes)

	if existingUser.ID != 0 {
		user.ID = existingUser.ID

		if err := s.repo.UpdateUnverifiedUser(user); err != nil {
			return models.User{}, err
		}

		return user, nil
	}

	return s.repo.CreateUser(user)
}

func (s *UserService) RegisterGoogleUser(
	name, email, googleID string,
) (models.User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))
	googleID = strings.TrimSpace(googleID)

	if email == "" || googleID == "" {
		return models.User{}, errors.New(
			"Google account information is incomplete",
		)
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return models.User{}, errors.New("invalid email address")
	}

	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}

	existingUser, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return models.User{}, err
	}

	if existingUser.ID != 0 {
		if existingUser.GoogleID == googleID {
			return existingUser, nil
		}

		return models.User{}, errors.New(
			"an account with this email already exists; sign in using your existing method",
		)
	}

	existingGoogleUser, err := s.repo.GetUserByGoogleID(googleID)
	if err != nil {
		return models.User{}, err
	}

	if existingGoogleUser.ID != 0 {
		return existingGoogleUser, nil
	}

	user := models.User{
		Name:          name,
		Email:         email,
		Role:          "customer",
		EmailVerified: true,
		GoogleID:      googleID,
	}

	return s.repo.CreateUser(user)
}

func (s *UserService) GetAllUsers() ([]models.User, error) {
	return s.repo.GetUsers()
}

func (s *UserService) LoginUser(email, password string) (models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return models.User{}, err
	}

	if user.ID == 0 {
		return models.User{}, nil
	}

	if user.Password == "" {
		return models.User{}, nil
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	)
	if err != nil {
		return models.User{}, nil
	}

	if !user.EmailVerified {
		return models.User{}, errors.New("email address is not verified")
	}

	return user, nil
}

func (s *UserService) VerifyEmail(token string) error {
	token = strings.TrimSpace(token)

	if token == "" {
		return errors.New("verification token is required")
	}

	user, err := s.repo.GetUserByVerificationToken(token)
	if err != nil {
		return err
	}

	if user.ID == 0 {
		return errors.New("invalid or expired verification token")
	}

	if user.EmailVerified {
		return errors.New("email is already verified")
	}

	return s.repo.VerifyUserEmail(user.ID)
}

type ShopService struct {
	repo *repository.ShopRepository
}

func NewShopService(repo *repository.ShopRepository) *ShopService {
	return &ShopService{repo: repo}
}
func (s *ShopService) CreateShop(shop models.Shop) (models.Shop, error) {
	shop.Name = strings.TrimSpace(shop.Name)
	shop.Location = strings.TrimSpace(shop.Location)
	shop.Phone = strings.TrimSpace(shop.Phone)

	if shop.Name == "" {
		return models.Shop{}, errors.New("shop name is required")
	}

	if shop.Location == "" {
		return models.Shop{}, errors.New("shop location is required")
	}

	if shop.Phone == "" {
		return models.Shop{}, errors.New("shop phone is required")
	}

	if shop.OwnerID == nil || *shop.OwnerID <= 0 {
		return models.Shop{}, errors.New("shop owner is required")
	}

	if shop.VerificationStatus == "" {
		shop.VerificationStatus = "pending"
	}

	return s.repo.CreateShop(shop)
}

func (s *ShopService) GetShops() ([]models.Shop, error) {
	return s.repo.GetShops()
}

func (s *ShopService) GetAllShops() ([]models.Shop, error) {
	return s.repo.GetShops()
}

func (s *ShopService) GetVerifiedShops() ([]models.Shop, error) {
	return s.repo.GetVerifiedShops()
}

func (s *ShopService) GetShopByID(id int) (models.Shop, error) {
	if id <= 0 {
		return models.Shop{}, errors.New("invalid shop ID")
	}

	return s.repo.GetShopByID(id)
}

func (s *ShopService) GetShopByOwnerID(ownerID int) (models.Shop, error) {
	if ownerID <= 0 {
		return models.Shop{}, errors.New("invalid owner ID")
	}

	return s.repo.GetShopByOwnerID(ownerID)
}

func (s *ShopService) UpdateShopPrices(
	id int,
	price6kg float64,
	price13kg float64,
	price45kg float64,
) error {
	if id <= 0 {
		return errors.New("invalid shop ID")
	}

	if price6kg < 0 || price13kg < 0 || price45kg < 0 {
		return errors.New("prices cannot be negative")
	}

	return s.repo.UpdateShopPrices(
		id,
		price6kg,
		price13kg,
		price45kg,
	)
}

func (s *ShopService) UpdateShopVerificationStatus(
	id int,
	status string,
) error {
	if id <= 0 {
		return errors.New("invalid shop ID")
	}

	status = strings.TrimSpace(strings.ToLower(status))

	if status != "pending" &&
		status != "approved" &&
		status != "rejected" {
		return errors.New("invalid verification status")
	}

	return s.repo.UpdateShopVerificationStatus(id, status)
}
