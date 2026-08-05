package repository

import (
	"database/sql"

	"halisi/internal/database"
	"halisi/internal/models"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// ---------------- USERS ----------------

func (r *UserRepository) CreateUser(user models.User) (models.User, error) {

	query := `
	INSERT INTO users(name, email, password, role)
	VALUES (?, ?, ?, ?)
	`

	result, err := database.DB.Exec(
		query,
		user.Name,
		user.Email,
		user.Password,
		user.Role,
	)

	if err != nil {
		return models.User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, err
	}

	user.ID = int(id)

	return user, nil
}

func (r *UserRepository) GetUsers() ([]models.User, error) {

	rows, err := database.DB.Query(`
		SELECT id, name, email, password, role
		FROM users
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []models.User

	for rows.Next() {

		var user models.User

		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password,
			&user.Role,
		)

		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) GetUserByEmail(email string) (models.User, error) {

	var user models.User

	query := `
	SELECT id, name, email, password, role
	FROM users
	WHERE email = ?
	`

	err := database.DB.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
	)

	if err != nil {

		if err == sql.ErrNoRows {
			return models.User{}, nil
		}

		return models.User{}, err
	}

	return user, nil
}

// ---------------- SHOPS ----------------

func (r *UserRepository) GetShops() ([]models.Shop, error) {

	rows, err := database.DB.Query(`
		SELECT id, name, location, phone, owner_id
		FROM shops
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var shops []models.Shop

	for rows.Next() {

		var shop models.Shop

		err := rows.Scan(
			&shop.ID,
			&shop.Name,
			&shop.Location,
			&shop.Phone,
			&shop.OwnerID,
		)

		if err != nil {
			return nil, err
		}

		shops = append(shops, shop)
	}

	return shops, nil
}

func (r *UserRepository) GetShopByID(id int) (models.Shop, error) {

	var shop models.Shop

	query := `
	SELECT id, name, location, phone, owner_id
	FROM shops
	WHERE id = ?
	`

	err := database.DB.QueryRow(query, id).Scan(
		&shop.ID,
		&shop.Name,
		&shop.Location,
		&shop.Phone,
		&shop.OwnerID,
	)

	if err != nil {

		if err == sql.ErrNoRows {
			return models.Shop{}, nil
		}

		return models.Shop{}, err
	}

	return shop, nil
}
