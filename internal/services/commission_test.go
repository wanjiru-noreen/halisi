package services

import "testing"

func TestCalculateCommission(t *testing.T) {
	for _, test := range []struct {
		name        string
		gross, fee  int64
		rate        int32
		vendorShare int64
	}{
		{"ten percent exact", 10000, 1000, 1000, 9000},
		{"round half cent up", 1, 1, 5000, 0},
		{"zero commission", 999, 0, 0, 999},
		{"full commission", 101, 101, 10000, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			fee, vendor, err := CalculateCommission(test.gross, test.rate)
			if err != nil || fee != test.fee || vendor != test.vendorShare || fee+vendor != test.gross {
				t.Fatalf("CalculateCommission(%d, %d) = %d, %d, %v", test.gross, test.rate, fee, vendor, err)
			}
		})
	}
}

func TestCalculateCommissionRejectsInvalidInputs(t *testing.T) {
	for _, input := range []struct {
		gross int64
		rate  int32
	}{{0, 1000}, {-1, 1000}, {100, -1}, {100, 10001}} {
		if _, _, err := CalculateCommission(input.gross, input.rate); err == nil {
			t.Fatalf("CalculateCommission(%d, %d) unexpectedly succeeded", input.gross, input.rate)
		}
	}
}
