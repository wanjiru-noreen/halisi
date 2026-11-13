package config

import (
	"errors"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Identity struct {
	PublicBaseURL                                                         string
	SMTPHost, SMTPPort, SMTPUsername, SMTPPassword, SMTPFrom, SMTPTLSMode string
	GoogleClientID, GoogleClientSecret                                    string
}

func LoadIdentity(appEnv string) (Identity, error) {
	c := Identity{PublicBaseURL: strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"), SMTPHost: os.Getenv("SMTP_HOST"), SMTPPort: os.Getenv("SMTP_PORT"), SMTPUsername: os.Getenv("SMTP_USERNAME"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: os.Getenv("SMTP_FROM"), SMTPTLSMode: os.Getenv("SMTP_TLS_MODE"), GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET")}
	if c.PublicBaseURL == "" && appEnv != "production" {
		c.PublicBaseURL = "http://localhost:" + Port()
	}
	u, err := url.Parse(c.PublicBaseURL)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(appEnv != "production" && u.Scheme == "http" && loopback(u.Hostname()))) {
		return Identity{}, errors.New("PUBLIC_BASE_URL must be an HTTPS origin (HTTP loopback is allowed in development)")
	}
	if (c.GoogleClientID == "") != (c.GoogleClientSecret == "") {
		return Identity{}, errors.New("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must both be configured")
	}
	if c.SMTPTLSMode == "" {
		c.SMTPTLSMode = "starttls"
	}
	if c.SMTPPort == "" {
		c.SMTPPort = "587"
		if c.SMTPTLSMode == "tls" {
			c.SMTPPort = "465"
		}
	}
	if c.SMTPHost == "" {
		if appEnv == "production" {
			return Identity{}, errors.New("SMTP_HOST is required in production for email verification")
		}
		return c, nil
	}
	port, err := strconv.Atoi(c.SMTPPort)
	if err != nil || port < 1 || port > 65535 {
		return Identity{}, errors.New("SMTP_PORT must be an integer between 1 and 65535")
	}
	sender, err := mail.ParseAddress(c.SMTPFrom)
	if err != nil || sender.Address != c.SMTPFrom || strings.ContainsAny(c.SMTPFrom, "\r\n") {
		return Identity{}, errors.New("SMTP_FROM must be a bare email address")
	}
	if strings.ContainsAny(c.SMTPHost, "/\\\r\n") {
		return Identity{}, errors.New("SMTP_HOST must be a hostname or IP address")
	}
	if (c.SMTPUsername == "") != (c.SMTPPassword == "") {
		return Identity{}, errors.New("SMTP_USERNAME and SMTP_PASSWORD must both be configured")
	}
	switch c.SMTPTLSMode {
	case "starttls", "tls":
	case "none":
		if appEnv == "production" || !loopback(c.SMTPHost) {
			return Identity{}, errors.New("SMTP_TLS_MODE=none is permitted only for local development mail servers")
		}
	default:
		return Identity{}, errors.New("SMTP_TLS_MODE must be starttls, tls, or none")
	}
	return c, nil
}

func loopback(host string) bool {
	ip := net.ParseIP(host)
	return host == "localhost" || (ip != nil && ip.IsLoopback())
}
