package models

type Shop struct {
	ID                         int     `json:"id"`
	Name                       string  `json:"name"`
	Location                   string  `json:"location"`
	Phone                      string  `json:"phone"`
	Price6kg                   float64 `json:"price_6kg"`
	Price13kg                  float64 `json:"price_13kg"`
	Price45kg                  float64 `json:"price_45kg"`
	OwnerID                    *int    `json:"owner_id"`
	Latitude                   float64 `json:"latitude"`
	Longitude                  float64 `json:"longitude"`
	BusinessRegistrationNumber string  `json:"business_registration_number"`
	KRAPin                     string  `json:"kra_pin"`
	LicenseNumber              string  `json:"license_number"`
	VerificationStatus         string  `json:"verification_status"`
}
