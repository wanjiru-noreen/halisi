package email

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/config"
)

type SMTP struct{ Config config.Identity }

func (s *SMTP) SendVerification(ctx context.Context, to, link string) error {
	recipient, err := mail.ParseAddress(to)
	if err != nil || recipient.Address != to || strings.ContainsAny(to+link+s.Config.SMTPFrom, "\r\n") {
		return errors.New("invalid verification email")
	}
	c := s.Config
	messageID := make([]byte, 16)
	if _, err := rand.Read(messageID); err != nil {
		return errors.New("could not create email identifier")
	}
	if c.SMTPTLSMode != "none" && c.SMTPTLSMode != "tls" && c.SMTPTLSMode != "starttls" {
		return errors.New("SMTP TLS mode is not configured")
	}
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(c.SMTPHost, c.SMTPPort))
	if err != nil {
		return errors.New("SMTP connection failed")
	}
	defer conn.Close()
	rawConn := conn
	stop := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer stop()
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	tlsConfig := &tls.Config{ServerName: c.SMTPHost, MinVersion: tls.VersionTLS12}
	if c.SMTPTLSMode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return errors.New("SMTP TLS failed")
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, c.SMTPHost)
	if err != nil {
		return errors.New("SMTP greeting failed")
	}
	defer client.Close()
	if c.SMTPTLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support required STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return errors.New("SMTP STARTTLS failed")
		}
	}
	if c.SMTPUsername != "" {
		if err := client.Auth(smtp.PlainAuth("", c.SMTPUsername, c.SMTPPassword, c.SMTPHost)); err != nil {
			return errors.New("SMTP authentication failed")
		}
	}
	if err := client.Mail(c.SMTPFrom); err != nil {
		return errors.New("SMTP sender rejected")
	}
	if err := client.Rcpt(to); err != nil {
		return errors.New("SMTP recipient rejected")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("SMTP DATA failed")
	}
	_, domain, _ := strings.Cut(c.SMTPFrom, "@")
	message := "From: " + (&mail.Address{Name: "HALISI", Address: c.SMTPFrom}).String() + "\r\nTo: " + recipient.String() + "\r\nDate: " + time.Now().Format(time.RFC1123Z) + "\r\nMessage-ID: <" + hex.EncodeToString(messageID) + "@" + domain + ">\r\nSubject: Verify your HALISI email\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nConfirm your email address for HALISI:\r\n\r\n" + link + "\r\n\r\nThis link expires in one hour and can be used once. If you did not request this, ignore this email.\r\n"
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return errors.New("SMTP message failed")
	}
	if err := writer.Close(); err != nil {
		return errors.New("SMTP message was not accepted")
	}
	// Once DATA is accepted, a QUIT failure must not turn successful delivery into a failed registration.
	_ = client.Quit()
	return nil
}
