package organization

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	organizationv1 "github.com/gobenpark/colign/gen/proto/organization/v1"
	"github.com/gobenpark/colign/internal/auth"
)

func makeAuthHeader(t *testing.T, jwtManager *auth.JWTManager, userID, orgID int64) string {
	t.Helper()
	tok, err := jwtManager.GenerateAccessToken(userID, "user@example.com", "Test User", orgID)
	require.NoError(t, err)
	return "Bearer " + tok
}

func reqWithAuth[T any](msg *T, authHeader string) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", authHeader)
	return req
}

func TestDeleteOrganization_OrgIDMismatch(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret")
	db, _ := setupTestDB(t)
	svc := NewService(db)
	h := NewConnectHandler(svc, jwtManager, nil, nil)

	// JWT org_id = 10, but request org_id = 20
	authHeader := makeAuthHeader(t, jwtManager, 1, 10)
	req := reqWithAuth(&organizationv1.DeleteOrganizationRequest{
		OrganizationId: 20,
	}, authHeader)

	_, err := h.DeleteOrganization(context.Background(), req)
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
}

func TestDeleteOrganization_Unauthenticated(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret")
	db, _ := setupTestDB(t)
	svc := NewService(db)
	h := NewConnectHandler(svc, jwtManager, nil, nil)

	req := connect.NewRequest(&organizationv1.DeleteOrganizationRequest{
		OrganizationId: 1,
	})

	_, err := h.DeleteOrganization(context.Background(), req)
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}
