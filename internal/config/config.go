package config

import "os"

// Port returns the configured HTTP port, defaulting to 8080.
func Port() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}
