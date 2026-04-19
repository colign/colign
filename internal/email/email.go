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

var (
	verifyTmpl = template.Must(template.ParseFS(templateFS, "templates/verify.html"))
	inviteTmpl = template.Must(template.ParseFS(templateFS, "templates/invite.html"))
)

// InviteParams holds the data needed to render an invitation email.
type InviteParams struct {
	Token       string
	OrgName     string
	InviterName string
	Role        string
}

// Sender sends transactional emails.
type Sender interface {
	SendVerificationEmail(to, token string) error
	SendInvitationEmail(to string, params InviteParams) error
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

func (s *LogSender) SendInvitationEmail(to string, params InviteParams) error {
	inviteURL := fmt.Sprintf("%s/invite/%s", s.BaseURL, params.Token)
	slog.Info("invitation email", "to", to, "org", params.OrgName, "inviter", params.InviterName, "role", params.Role, "invite_url", inviteURL)
	return nil
}

// ResendSender sends emails via the Resend API.
type ResendSender struct {
	client  *resend.Client
	from    string
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

func (s *ResendSender) SendInvitationEmail(to string, params InviteParams) error {
	inviteURL := fmt.Sprintf("%s/invite/%s", s.baseURL, params.Token)

	var buf bytes.Buffer
	if err := inviteTmpl.Execute(&buf, map[string]string{
		"InviteURL":   inviteURL,
		"OrgName":     params.OrgName,
		"InviterName": params.InviterName,
		"Role":        params.Role,
	}); err != nil {
		return fmt.Errorf("rendering invitation email template: %w", err)
	}

	_, err := s.client.Emails.SendWithContext(context.Background(), &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: fmt.Sprintf("You're invited to %s - Colign", params.OrgName),
		Html:    buf.String(),
	})
	if err != nil {
		slog.Error("failed to send invitation email via resend", "to", to, "org", params.OrgName, "error", err)
		return fmt.Errorf("sending invitation email: %w", err)
	}

	return nil
}
