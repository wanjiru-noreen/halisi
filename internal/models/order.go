package models

type Order struct {
	ID           int
	UserID       int
	ShopID       int
	CylinderSize string
	Quantity     int
	TotalPrice   float64
	Status       string
}
