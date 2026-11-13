package services

import (
	"strings"
	"testing"
)

func TestRiderInputValidation(t *testing.T) {
	good := RiderInput{FullName: "  Alex Kamau ", Phone: " +254712345678 ", City: " Nairobi "}
	if err := good.Validate(); err != nil || good.FullName != "Alex Kamau" || good.Phone != "+254712345678" || good.City != "Nairobi" {
		t.Fatal("profile normalization failed")
	}
	for _, field := range []string{"name", "control", "long-name", "phone", "phone-long", "city", "city-long"} {
		t.Run(field, func(t *testing.T) {
			v := good
			switch field {
			case "name":
				v.FullName = " "
			case "control":
				v.FullName = "Alex\x00Kamau"
			case "long-name":
				v.FullName = strings.Repeat("a", 101)
			case "phone":
				v.Phone = "0712345678"
			case "phone-long":
				v.Phone = "+" + strings.Repeat("1", 16)
			case "city":
				v.City = " "
			case "city-long":
				v.City = strings.Repeat("a", 101)
			}
			if v.Validate() == nil {
				t.Fatal("invalid rider profile accepted")
			}
		})
	}
}
