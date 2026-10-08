package email

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendVerificationEmail(to, token string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	baseURL := os.Getenv("APP_URL")

	if host == "" || port == "" || username == "" || password == "" || from == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	verificationURL := fmt.Sprintf(
		"%s/verify-email?token=%s",
		baseURL,
		token,
	)

	subject := "Verify your Halisi email address"

	body := fmt.Sprintf(
		"Hello,\n\n"+
			"Thank you for registering with Halisi.\n\n"+
			"Please verify your email address by clicking the link below:\n\n"+
			"%s\n\n"+
			"If you did not create this account, you can ignore this email.\n\n"+
			"Regards,\n"+
			"Halisi",
		verificationURL,
	)

	message := []byte(
		"From: " + from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	auth := smtp.PlainAuth(
		"",
		username,
		password,
		host,
	)

	return smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{to},
		message,
	)
}
