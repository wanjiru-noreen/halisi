package handlers_test

import (
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/handlers"
)

func TestNormalizeMPesaPhone(t *testing.T) {
	for raw, want := range map[string]string{
		"0712 345 678":      "254712345678",
		"+254 (712) 345678": "254712345678",
		"254712345678":      "254712345678",
		"0112-345-678":      "254112345678",
	} {
		got, err := handlers.NormalizeMPesaPhone(raw)
		if err != nil || got != want {
			t.Errorf("NormalizeMPesaPhone(%q) = (%q, %v), want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "12345", "071234567", "07123456789", "+254912345678"} {
		if _, err := handlers.NormalizeMPesaPhone(raw); err == nil {
			t.Errorf("NormalizeMPesaPhone(%q) unexpectedly succeeded", raw)
		}
	}
}
