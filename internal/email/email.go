package email

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/resend/resend-go/v3"
)

//go:embed templates/*.html
var templateFS embed.FS

var verifyTmpl = template.Must(template.ParseFS(templateFS, "templates/verify.html"))

// Sender sends transactional emails.
type Sender interface {
	SendVerificationEmail(to, token string) error
}

// LogSender logs emails to stdout (for development).
type LogSender struct {
	BaseURL string
}

func NewLogSender(baseURL string) *LogSender {
	return &LogSender{BaseURL: baseURL}
}

func (s *LogSender) SendVerificationEmail(to, token string) error {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", s.BaseURL, token)
	slog.Info("verification email", "to", to, "verify_url", verifyURL)
	return nil
}

// ResendSender sends emails via the Resend API.
type ResendSender struct {
	client *resend.Client
	from   string
	baseURL string
}

func NewResendSender(apiKey, baseURL, from string) *ResendSender {
	return &ResendSender{
		client:  resend.NewClient(apiKey),
		from:    from,
		baseURL: baseURL,
	}
}

func (s *ResendSender) SendVerificationEmail(to, token string) error {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", s.baseURL, token)

	var buf bytes.Buffer
	if err := verifyTmpl.Execute(&buf, map[string]string{"VerifyURL": verifyURL}); err != nil {
		return fmt.Errorf("rendering verification email template: %w", err)
	}

	_, err := s.client.Emails.SendWithContext(context.Background(), &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Verify your email - Colign",
		Html:    buf.String(),
	})
	if err != nil {
		slog.Error("failed to send verification email via resend", "to", to, "error", err)
		return fmt.Errorf("sending verification email: %w", err)
	}

	return nil
}
