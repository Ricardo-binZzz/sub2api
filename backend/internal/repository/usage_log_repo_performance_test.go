package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetUpstreamPerformanceSplitsGroupsAndAccounts(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	start := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	mock.ExpectQuery("SELECT 'group' AS entity_type").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"entity_type", "entity_id", "name", "platform", "requests", "successful",
			"first_p50", "first_p95", "duration_p50", "duration_p95",
		}).
			AddRow("group", int64(6), "Plus", "openai", int64(10), int64(9), 900.0, 1800.0, 1500.0, 3200.0).
			AddRow("account", int64(137), "plus", "openai", int64(7), int64(7), 850.0, 1700.0, 1450.0, 3000.0))

	report, err := repo.GetUpstreamPerformance(context.Background(), start, end)
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	require.Len(t, report.Accounts, 1)
	require.Equal(t, 90.0, report.Groups[0].SuccessRate)
	require.Equal(t, 900.0, *report.Groups[0].FirstTokenP50Ms)
	require.Equal(t, int64(137), report.Accounts[0].ID)
	require.Equal(t, 3000.0, report.Thresholds.FirstTokenP50WarningMs)
	require.NoError(t, mock.ExpectationsWereMet())
}
