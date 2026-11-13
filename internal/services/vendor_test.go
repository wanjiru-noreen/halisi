package services

import "testing"

func TestVendorInput(t *testing.T) {
	valid := func() VendorInput {
		hours := map[string]Hours{}
		for _, d := range []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"} {
			hours[d] = Hours{Open: "08:00", Close: "18:00"}
		}
		return VendorInput{Name: "HALISI LPG", ContactEmail: "shop@example.com", ContactPhone: "+254712345678", Address: "Kenyatta Avenue", City: "Nairobi", OperatingHours: hours}
	}
	if err := func() error { v := valid(); return v.Validate() }(); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"email", "phone", "name", "hours", "closed", "overnight", "unknown-day"} {
		t.Run(field, func(t *testing.T) {
			v := valid()
			switch field {
			case "email":
				v.ContactEmail = "not-email"
			case "phone":
				v.ContactPhone = "0712345678"
			case "name":
				v.Name = " "
			case "hours":
				delete(v.OperatingHours, "sun")
			case "closed":
				for d := range v.OperatingHours {
					v.OperatingHours[d] = Hours{Closed: true}
				}
			case "overnight":
				v.OperatingHours["mon"] = Hours{Open: "20:00", Close: "06:00"}
			case "unknown-day":
				delete(v.OperatingHours, "mon")
				v.OperatingHours["funday"] = Hours{Closed: true}
			}
			if v.Validate() == nil {
				t.Fatal("invalid application accepted")
			}
		})
	}
}
