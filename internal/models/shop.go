package models

type Shop struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Phone    string `json:"phone"`
	OwnerID  *int   `json:"owner_id"`
}
