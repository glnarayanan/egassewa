package email

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func FromEnv() Config {
	return Config{
		Host:     getenv("SMTP_HOST", ""),
		Port:     getenv("SMTP_PORT", "587"),
		User:     getenv("SMTP_USER", ""),
		Password: getenv("SMTP_PASS", ""),
		From:     getenv("SMTP_FROM", "gasmate@example.com"),
	}
}

func (c Config) Send(to, subject, body string) error {
	if c.Host == "" {
		log.Printf("[email] SMTP not configured — would send to %s: %s", to, subject)
		return nil
	}
	addr := c.Host + ":" + c.Port
	auth := smtp.PlainAuth("", c.User, c.Password, c.Host)
	msg := fmt.Sprintf("From: GasMate <%s>\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		c.From, to, subject, body)
	if err := smtp.SendMail(addr, auth, c.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
