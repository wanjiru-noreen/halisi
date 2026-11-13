package email

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/config"
)

func TestSMTPVerificationMessage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	messages := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		write := func(value string) { _, _ = conn.Write([]byte(value + "\r\n")) }
		write("220 localhost test SMTP")
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				write("250 localhost")
			case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				write("250 OK")
			case strings.HasPrefix(line, "DATA"):
				write("354 send message")
				var body strings.Builder
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if line == ".\r\n" {
						break
					}
					body.WriteString(line)
				}
				messages <- body.String()
				write("250 accepted")
			case strings.HasPrefix(line, "QUIT"):
				write("221 bye")
				return
			default:
				write("500 unexpected")
			}
		}
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	sender := &SMTP{Config: config.Identity{SMTPHost: host, SMTPPort: port, SMTPFrom: "no-reply@example.com", SMTPTLSMode: "none"}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	link := "https://halisi.example/verify-email#token=test-token"
	if err := sender.SendVerification(ctx, "user@example.com", link); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-messages:
		if !strings.Contains(message, link) || !strings.Contains(message, "To: <user@example.com>") {
			t.Fatal("verification email was malformed")
		}
	case <-ctx.Done():
		t.Fatal("SMTP message not delivered")
	}
	if err := sender.SendVerification(ctx, "user@example.com\r\nBCC: attacker@example.com", link); err == nil {
		t.Fatal("SMTP header injection accepted")
	}
}
