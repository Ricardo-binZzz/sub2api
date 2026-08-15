package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestOperationRepositoryBeginTaskRunClaimsAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	columns := []string{
		"id", "connection_id", "kind", "platform", "status", "content", "idempotency_key", "scheduled_at",
		"approved_by", "approved_at", "attempts", "max_attempts", "last_error", "external_id", "created_at", "updated_at",
	}
	mock.ExpectBegin()
	claim := `UPDATE assistant_operation_tasks SET status='running',attempts=attempts+1,updated_at=NOW()
WHERE id=$1 AND status IN ('approved','failed') AND scheduled_at<=NOW() AND attempts<max_attempts RETURNING ` + operationTaskColumns
	mock.ExpectQuery(regexp.QuoteMeta(claim)).WithArgs(int64(9)).WillReturnRows(
		sqlmock.NewRows(columns).AddRow(9, 3, "promotion", "bluesky", "running", "hello", "key", now, nil, nil, 1, 3, "", "", now, now),
	)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO assistant_operation_runs(task_id,status) VALUES($1,'running') RETURNING id")).
		WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(22))
	mock.ExpectCommit()

	task, runID, err := NewOperationRepository(db).BeginTaskRun(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, "running", task.Status)
	require.Equal(t, 1, task.Attempts)
	require.Equal(t, int64(22), runID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOperationRepositoryDefersWithoutIncrementingAttempts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	at := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	query := `UPDATE assistant_operation_tasks SET scheduled_at=$2,last_error=$3,updated_at=NOW()
WHERE id=$1 AND status IN ('approved','failed')`
	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(int64(9), at, "daily budget exhausted").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = NewOperationRepository(db).DeferTask(context.Background(), 9, at, "daily budget exhausted")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
