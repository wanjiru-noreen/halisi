package config

import "testing"

func TestIdentityConfiguration(t *testing.T) {
	keys := []string{"PUBLIC_BASE_URL", "SMTP_HOST", "SMTP_PORT", "SMTP_TLS_MODE", "SMTP_FROM", "SMTP_USERNAME", "SMTP_PASSWORD", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"}
	for _, name := range []string{"local", "production", "insecure production", "insecure remote SMTP", "missing Google secret", "unsafe public URL", "unsafe sender"} {
		t.Run(name, func(t *testing.T) {
			for _, key := range keys {
				t.Setenv(key, "")
			}
			t.Setenv("PUBLIC_BASE_URL", "http://localhost:8080")
			t.Setenv("SMTP_HOST", "localhost")
			t.Setenv("SMTP_PORT", "1025")
			t.Setenv("SMTP_TLS_MODE", "none")
			t.Setenv("SMTP_FROM", "no-reply@example.com")
			env := "development"
			switch name {
			case "production":
				env = "production"
				t.Setenv("PUBLIC_BASE_URL", "https://halisi.example")
				t.Setenv("SMTP_TLS_MODE", "starttls")
			case "insecure production":
				env = "production"
			case "insecure remote SMTP":
				t.Setenv("SMTP_HOST", "smtp.example.com")
			case "missing Google secret":
				t.Setenv("GOOGLE_CLIENT_ID", "client")
			case "unsafe public URL":
				t.Setenv("PUBLIC_BASE_URL", "https://user:secret@halisi.example/path")
			case "unsafe sender":
				t.Setenv("SMTP_FROM", "sender@example.com\r\nBCC: other@example.com")
			}
			_, err := LoadIdentity(env)
			valid := name == "local" || name == "production"
			if valid && err != nil {
				t.Fatal(err)
			}
			if !valid && err == nil {
				t.Fatal("unsafe identity settings accepted")
			}
		})
	}
}
