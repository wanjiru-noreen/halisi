package services

import "testing"

func TestDeliveryLifecycle(t *testing.T) {
	allowed := map[[2]string]bool{
		{"pending", "assigned"}:    true,
		{"pending", "cancelled"}:   true,
		{"assigned", "picked_up"}:  true,
		{"assigned", "failed"}:     true,
		{"assigned", "cancelled"}:  true,
		{"picked_up", "delivered"}: true,
		{"picked_up", "failed"}:    true,
	}
	states := []string{"pending", "assigned", "picked_up", "delivered", "failed", "cancelled", "unknown", ""}
	for _, from := range states {
		for _, to := range states {
			t.Run(from+"/"+to, func(t *testing.T) {
				if got := DeliveryTransitionAllowed(from, to); got != allowed[[2]string{from, to}] {
					t.Fatalf("transition %q -> %q allowed=%v", from, to, got)
				}
			})
		}
	}
}
