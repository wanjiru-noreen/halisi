package repository

import (
	"halisi/internal/models"
)

type UserRepository struct {
	users []models.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: []models.User{},
	}
}

func (r *UserRepository) CreateUser(user models.User) models.User {
	user.ID = len(r.users) + 1
	r.users = append(r.users, user)

	return user
}

func (r *UserRepository) GetUsers() []models.User {
	return r.users
}
