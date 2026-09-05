// Package mail delivers the few messages the site sends by itself.
package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const dialTimeout = 10 * time.Second

var ErrNoSender = errors.New("mail: no from address is configured")

type Sender interface {
	Send(ctx context.Context, to []string, subject, body string) error
}

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	UseTLS   bool
	From     string
}

type Console struct {
	Log *slog.Logger
}

var _ Sender = (*Console)(nil)

func (c *Console) Send(_ context.Context, to []string, subject, body string) error {
	log := c.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("mail not sent, no mail host configured", "to", strings.Join(to, ", "), "subject", subject, "body", body)
	return nil
}

type SMTP struct {
	Config
}

var _ Sender = (*SMTP)(nil)

func New(c Config) Sender {
	if c.Host == "" {
		return &Console{}
	}
	return &SMTP{Config: c}
}

func (s *SMTP) Send(ctx context.Context, to []string, subject, body string) error {
	if s.From == "" {
		return ErrNoSender
	}
	address := net.JoinHostPort(s.Host, s.Port)
	dialer := net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial mail host %s: %w", address, err)
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("open mail session with %s: %w", address, err)
	}
	defer client.Close()

	if s.UseTLS {
		if err := client.StartTLS(&tls.Config{ServerName: s.Host}); err != nil {
			return fmt.Errorf("start TLS with %s: %w", address, err)
		}
	}
	if s.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("authenticate with %s: %w", address, err)
		}
	}
	if err := client.Mail(s.From); err != nil {
		return fmt.Errorf("send from %s: %w", s.From, err)
	}
	for _, one := range to {
		if err := client.Rcpt(one); err != nil {
			return fmt.Errorf("send to %s: %w", one, err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if _, err := writer.Write(message(s.From, to, subject, body)); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return client.Quit()
}

func message(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return []byte(b.String())
}
