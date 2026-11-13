package services

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestValidateRegistration(t *testing.T) {
	valid := RegisterInput{Name: "  Amina Wanjiru  ", Email: " AMINA@Example.com ", Phone: " +254712345678 ", Password: "a long passphrase"}
	normalized, err := ValidateRegistration(valid)
	if err != nil || normalized.Name != "Amina Wanjiru" || normalized.Email != "amina@example.com" || normalized.Phone != "+254712345678" || normalized.Password != valid.Password {
		t.Fatal("registration values were not normalized correctly")
	}
	for _, field := range []string{"name", "control-name", "email", "display-email", "phone", "short-password", "long-password", "unicode-password"} {
		t.Run(field, func(t *testing.T) {
			input := valid
			switch field {
			case "name":
				input.Name = " "
			case "control-name":
				input.Name = "Amina\nWanjiru"
			case "email":
				input.Email = "invalid"
			case "display-email":
				input.Email = "Amina <amina@example.com>"
			case "phone":
				input.Phone = "0712345678"
			case "short-password":
				input.Password = "short"
			case "long-password":
				input.Password = strings.Repeat("a", 73)
			case "unicode-password":
				input.Password = strings.Repeat("界", 25)
			}
			if _, err := ValidateRegistration(input); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestSessionTokens(t *testing.T) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatal(err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	if !ValidToken(token) || len(TokenHash(token)) != 64 || TokenHash(token) == token {
		t.Fatal("invalid session token/hash")
	}
	for _, invalid := range []string{"", "raw-token", strings.Repeat("a", 42), strings.Repeat("!", 43), token + "="} {
		if ValidToken(invalid) {
			t.Fatal("invalid token accepted")
		}
	}
}
