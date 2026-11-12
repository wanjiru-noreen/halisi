package config

import "testing"

func TestPort(t *testing.T) {
	for _, port := range []string{"", "9090"} {
		t.Run("PORT="+port, func(t *testing.T) {
			t.Setenv("PORT", port)
			want := port
			if want == "" {
				want = "8080"
			}
			if got := Port(); got != want {
				t.Fatalf("Port() = %q, want %q", got, want)
			}
		})
	}
}
