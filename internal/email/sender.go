package email

import (
	"fmt"
	"net/smtp"
)

type Sender interface {
	SendEmail(to []string, subject string, body string) error
}

type SmtpSender struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func NewSmtpSender(host, port, username, password, from string) *SmtpSender {
	return &SmtpSender{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

func (s *SmtpSender) SendEmail(to []string, subject string, body string) error {
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)

	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", s.From, to[0], subject, body))

	err := smtp.SendMail(addr, auth, s.From, to, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
