package email

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogSender_SendVerificationEmail(t *testing.T) {
	sender := NewLogSender("https://app.colign.dev")
	err := sender.SendVerificationEmail("user@example.com", "abc123token")
	require.NoError(t, err)
}

func TestVerifyTemplate_Renders(t *testing.T) {
	var buf strings.Builder
	err := verifyTmpl.Execute(&buf, map[string]string{
		"VerifyURL": "https://app.colign.dev/auth/verify-email?token=test123",
	})
	require.NoError(t, err)

	html := buf.String()
	assert.Contains(t, html, "https://app.colign.dev/auth/verify-email?token=test123")
	assert.Contains(t, html, "Verify Email")
	assert.Contains(t, html, "Colign")
	assert.Contains(t, html, "24 hours")
}
