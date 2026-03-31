package workflow

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdvance_CrossTenantBlocked(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	stage, err := svc.Advance(ctx, 1, 1, wrongOrgID)
	require.ErrorIs(t, err, ErrChangeNotFound)
	require.Equal(t, "", string(stage))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRevert_CrossTenantBlocked(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	err := svc.Revert(ctx, 1, 1, "reason", wrongOrgID)
	require.ErrorIs(t, err, ErrChangeNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEvaluateAndAdvance_CrossTenantBlocked(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	advanced, err := svc.EvaluateAndAdvance(ctx, 1, wrongOrgID)
	require.ErrorIs(t, err, ErrChangeNotFound)
	require.False(t, advanced)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStatus_CrossTenantBlocked(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	stage, subStatus, conditions, err := svc.GetStatus(ctx, 1, wrongOrgID)
	require.ErrorIs(t, err, ErrChangeNotFound)
	require.Equal(t, "", string(stage))
	require.Equal(t, "", string(subStatus))
	require.Nil(t, conditions)
	require.NoError(t, mock.ExpectationsWereMet())
}
