package email

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/smtp"

	_ "embed"

	"github.com/amirullazmi0/web-profile-backend/config"
)

type EmailService struct {
	config  *config.SMTPConfig
	appName string
}

type EmailData struct {
	To      string
	Subject string
	Body    string
}

func NewEmailService(cfg *config.SMTPConfig, appName string) *EmailService {
	return &EmailService{config: cfg, appName: appName}
}

//go:embed templates/verification_email.html
var verificationEmailTmpl string

//go:embed templates/verification_email_admin.html
var verificationEmailAdminTmpl string

//go:embed templates/reset_password_email.html
var resetPasswordEmailTmpl string

// SendEmail sends an email using SMTP
func (s *EmailService) SendEmail(to, subject, htmlBody string) error {
	auth := smtp.PlainAuth("", s.config.Email, s.config.Password, s.config.Host)

	// Email headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Build message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody

	// Send email
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	err := smtp.SendMail(addr, auth, s.config.FromEmail, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendVerificationEmail sends email verification
func (s *EmailService) SendVerificationEmail(to, name, verificationToken, baseURL string) error {
	verificationLink := fmt.Sprintf("%s/auth/verify-email?token=%s", baseURL, verificationToken)

	t, err := template.New("verification").Parse(verificationEmailTmpl)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	data := map[string]string{
		"Name":             name,
		"VerificationLink": verificationLink,
		"AppName":          s.appName,
	}

	if err := t.Execute(&body, data); err != nil {
		return err
	}

	return s.SendEmail(to, "Verify Your Email Address", body.String())
}

// SendVerificationEmailAdmin sends email verification for admin created users (with password)
func (s *EmailService) SendVerificationEmailAdmin(to, name, verificationToken, password, baseURL string) error {
	verificationLink := fmt.Sprintf("%s/auth/verify-email?token=%s", baseURL, verificationToken)

	t, err := template.New("verification_admin").Parse(verificationEmailAdminTmpl)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	data := map[string]string{
		"Name":             name,
		"VerificationLink": verificationLink,
		"AppName":          s.appName,
		"Password":         password,
	}

	if err := t.Execute(&body, data); err != nil {
		return err
	}

	return s.SendEmail(to, "Account Created - Verify Your Email", body.String())
}

// SendPasswordResetEmail sends password reset instructions.
func (s *EmailService) SendPasswordResetEmail(to, name, resetToken, baseURL string) error {
	resetLink := fmt.Sprintf("%s/auth/reset-password?token=%s", baseURL, resetToken)

	t, err := template.New("reset_password").Parse(resetPasswordEmailTmpl)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	data := map[string]string{
		"Name":      name,
		"ResetLink": resetLink,
		"AppName":   s.appName,
	}

	if err := t.Execute(&body, data); err != nil {
		return err
	}

	return s.SendEmail(to, "Reset Your Password", body.String())
}

// GenerateVerificationToken generates a random verification token
func GenerateVerificationToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
