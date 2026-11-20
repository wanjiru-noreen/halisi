package services

import (
	"errors"
	"math"
)

// CalculateCommission rounds the platform fee to the nearest cent, with half
// cents rounded up, and returns the fee and vendor share in cents.
func CalculateCommission(grossCents int64, rateBPS int32) (commissionCents, vendorCents int64, err error) {
	if grossCents <= 0 || rateBPS < 0 || rateBPS > 10000 {
		return 0, 0, errors.New("gross amount and commission rate are invalid")
	}
	if grossCents > (math.MaxInt64-5000)/10000 {
		return 0, 0, errors.New("gross amount is too large")
	}
	commissionCents = (grossCents*int64(rateBPS) + 5000) / 10000
	return commissionCents, grossCents - commissionCents, nil
}
