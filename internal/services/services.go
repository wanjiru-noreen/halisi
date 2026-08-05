package services

import (
	"halisi/internal/models"
	"halisi/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) RegisterUser(user models.User) models.User {
	return s.repo.CreateUser(user)
}

func (s *UserService) GetAllUsers() []models.User {
	return s.repo.GetUsers()
}
