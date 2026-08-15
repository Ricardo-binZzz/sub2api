package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type operationTestRepo struct {
	OperationRepository
	policy        *OperationPolicy
	connection    *OperationConnectionRecord
	created       OperationTaskInput
	beginTask     *OperationTask
	count         int
	lastSucceeded *time.Time
	completed     bool
	deferred      bool
	deferAt       time.Time
	deferMessage  string
	retryAt       time.Time
}

func (r *operationTestRepo) GetPolicy(context.Context) (*OperationPolicy, error) {
	return r.policy, nil
}
func (r *operationTestRepo) GetConnection(context.Context, int64) (*OperationConnectionRecord, error) {
	return r.connection, nil
}
func (r *operationTestRepo) CreateTask(_ context.Context, in OperationTaskInput) (*OperationTask, error) {
	r.created = in
	return &OperationTask{Status: in.Status, Content: in.Content}, nil
}
func (r *operationTestRepo) CountSucceededSince(context.Context, time.Time) (int, error) {
	return r.count, nil
}
func (r *operationTestRepo) LastSucceededAt(context.Context, string) (*time.Time, error) {
	return r.lastSucceeded, nil
}
func (r *operationTestRepo) BeginTaskRun(context.Context, int64) (*OperationTask, int64, error) {
	return r.beginTask, 71, nil
}
func (r *operationTestRepo) GetTask(context.Context, int64) (*OperationTask, error) {
	return r.beginTask, nil
}
func (r *operationTestRepo) DeferTask(_ context.Context, _ int64, at time.Time, message string) error {
	r.deferred = true
	r.deferAt = at
	r.deferMessage = message
	return nil
}
func (r *operationTestRepo) CompleteTask(_ context.Context, _, _ int64, _, _ string, _ bool, retry time.Time) error {
	r.completed = true
	r.retryAt = retry
	return nil
}
func (r *operationTestRepo) UpdateConnectionHealth(context.Context, int64, string, bool) error {
	return nil
}

type operationTestEncryptor struct{}

func (operationTestEncryptor) Encrypt(value string) (string, error) { return value, nil }
func (operationTestEncryptor) Decrypt(value string) (string, error) { return value, nil }

func operationPolicy() *OperationPolicy {
	return &OperationPolicy{
		Enabled: true, RequireApproval: true, PlanningIntervalMinutes: 60,
		MinPublishIntervalMinutes: 180, MaxDailyActions: 3, QuietHoursStart: 23, QuietHoursEnd: 8,
	}
}

func TestAssistantOperationsCreateTaskRequiresApproval(t *testing.T) {
	repo := &operationTestRepo{
		policy:     operationPolicy(),
		connection: &OperationConnectionRecord{OperationConnection: OperationConnection{ID: 4, Platform: "bluesky", Enabled: true}},
	}
	svc := NewAssistantOperationsService(repo, operationTestEncryptor{}, nil, nil)

	task, err := svc.CreateTask(context.Background(), 4, "promotion", "  release update  ", time.Time{})
	require.NoError(t, err)
	require.Equal(t, "pending_approval", task.Status)
	require.Equal(t, "release update", repo.created.Content)

	repo.policy.RequireApproval = false
	task, err = svc.CreateTask(context.Background(), 4, "promotion", "second update", time.Now())
	require.NoError(t, err)
	require.Equal(t, "approved", task.Status)
}

func TestAssistantOperationsExecutionPolicyBoundaries(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	task := &OperationTask{Kind: "promotion"}
	tests := []struct {
		name  string
		hour  int
		count int
		last  *time.Time
		want  string
	}{
		{name: "daily budget", hour: 12, count: 3, want: "daily budget"},
		{name: "quiet hours", hour: 23, want: "quiet hours"},
		{name: "minimum publish interval", hour: 12, last: assistantTimePtr(now.Add(-30 * time.Minute)), want: "publish interval"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &operationTestRepo{policy: operationPolicy(), count: tc.count, lastSucceeded: tc.last}
			svc := NewAssistantOperationsService(repo, operationTestEncryptor{}, nil, nil)
			svc.now = func() time.Time { return now.Add(time.Duration(tc.hour-now.Hour()) * time.Hour) }
			err := svc.checkExecutionPolicy(context.Background(), repo.policy, task)
			require.ErrorIs(t, err, ErrOperationPolicyBlocked)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestAssistantOperationsPolicyDeferralDoesNotConsumeAttempt(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	policy := operationPolicy()
	policy.MaxDailyActions = 1
	repo := &operationTestRepo{policy: policy, beginTask: &OperationTask{ID: 8, Kind: "promotion", Attempts: 2}, count: 1}
	svc := NewAssistantOperationsService(repo, operationTestEncryptor{}, nil, nil)
	svc.now = func() time.Time { return now }

	err := svc.RunTask(context.Background(), 8)
	require.ErrorIs(t, err, ErrOperationPolicyBlocked)
	require.True(t, repo.deferred)
	require.False(t, repo.completed)
	require.Equal(t, now.Add(12*time.Hour), repo.deferAt)
	require.Contains(t, repo.deferMessage, "daily budget")
	require.Equal(t, 2, repo.beginTask.Attempts)
}

func TestAssistantOperationsValidationAndRedaction(t *testing.T) {
	require.NoError(t, validateConnectionConfig("discord", map[string]string{
		"webhook_url": "https://discord.com/api/webhooks/123/token",
	}))
	for _, raw := range []string{
		"http://discord.com/api/webhooks/123/token",
		"https://evil.example/api/webhooks/123/token",
		"https://discord.com/not-a-webhook",
	} {
		require.ErrorIs(t, validateConnectionConfig("discord", map[string]string{"webhook_url": raw}), ErrOperationInvalid)
	}

	allowed := parseChatIDs("123, -456, invalid, 0")
	require.True(t, allowed["123"])
	require.True(t, allowed["-456"])
	require.False(t, allowed["999"])
	require.False(t, allowed["0"])

	secret := "TOKEN-super-secret"
	message := externalErrorMessage(errors.New("request failed with " + secret))
	require.Equal(t, "external platform operation failed", message)
	require.False(t, strings.Contains(message, secret))
}

func assistantTimePtr(value time.Time) *time.Time { return &value }
