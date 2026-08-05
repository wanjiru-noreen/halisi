package models

type Shop struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Location  string  `json:"location"`
	Phone     string  `json:"phone"`
	Price6kg  float64 `json:"price_6kg"`
	Price13kg float64 `json:"price_13kg"`
	OwnerID   *int    `json:"owner_id"`
}
