package service

import (
	"context"
	"errors"
	"time"
)

var (
	ErrOperationNotFound      = errors.New("operation resource not found")
	ErrOperationInvalid       = errors.New("operation request is invalid")
	ErrOperationPolicyBlocked = errors.New("operation blocked by policy")
)

type OperationConnection struct {
	ID            int64      `json:"id"`
	Platform      string     `json:"platform"`
	Name          string     `json:"name"`
	Enabled       bool       `json:"enabled"`
	HasSecrets    bool       `json:"has_secrets"`
	LastError     string     `json:"last_error,omitempty"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type OperationConnectionRecord struct {
	OperationConnection
	EncryptedConfig string
	LastCursor      string
}

type OperationConnectionInput struct {
	Platform string            `json:"platform"`
	Name     string            `json:"name"`
	Enabled  bool              `json:"enabled"`
	Config   map[string]string `json:"config"`
}

type OperationPolicy struct {
	Enabled                   bool      `json:"enabled"`
	AutonomousEnabled         bool      `json:"autonomous_enabled"`
	AutoPublish               bool      `json:"auto_publish"`
	RequireApproval           bool      `json:"require_approval"`
	PlanningIntervalMinutes   int       `json:"planning_interval_minutes"`
	MinPublishIntervalMinutes int       `json:"min_publish_interval_minutes"`
	MaxDailyActions           int       `json:"max_daily_actions"`
	QuietHoursStart           int       `json:"quiet_hours_start"`
	QuietHoursEnd             int       `json:"quiet_hours_end"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type OperationTask struct {
	ID             int64      `json:"id"`
	ConnectionID   *int64     `json:"connection_id,omitempty"`
	Kind           string     `json:"kind"`
	Platform       string     `json:"platform"`
	Status         string     `json:"status"`
	Content        string     `json:"content"`
	IdempotencyKey string     `json:"idempotency_key"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	ApprovedBy     *int64     `json:"approved_by,omitempty"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	Attempts       int        `json:"attempts"`
	MaxAttempts    int        `json:"max_attempts"`
	LastError      string     `json:"last_error,omitempty"`
	ExternalID     string     `json:"external_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type OperationRun struct {
	ID         int64      `json:"id"`
	TaskID     int64      `json:"task_id"`
	Status     string     `json:"status"`
	Output     string     `json:"output,omitempty"`
	Error      string     `json:"error,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type OperationTaskInput struct {
	ConnectionID   int64
	Kind           string
	Platform       string
	Status         string
	Content        string
	IdempotencyKey string
	ScheduledAt    time.Time
}

type OperationRepository interface {
	ListConnections(context.Context) ([]*OperationConnectionRecord, error)
	GetConnection(context.Context, int64) (*OperationConnectionRecord, error)
	SaveConnection(context.Context, *OperationConnectionRecord) (*OperationConnectionRecord, error)
	DeleteConnection(context.Context, int64) error
	UpdateConnectionHealth(context.Context, int64, string, bool) error
	UpdateConnectionCursor(context.Context, int64, string) error
	GetPolicy(context.Context) (*OperationPolicy, error)
	UpdatePolicy(context.Context, OperationPolicy) (*OperationPolicy, error)
	SetAutonomousEnabled(context.Context, bool) error
	CreateTask(context.Context, OperationTaskInput) (*OperationTask, error)
	GetTask(context.Context, int64) (*OperationTask, error)
	ListTasks(context.Context, int, int) ([]*OperationTask, int64, error)
	ApproveTask(context.Context, int64, int64) (*OperationTask, error)
	CancelTask(context.Context, int64) error
	DeferTask(context.Context, int64, time.Time, string) error
	BeginTaskRun(context.Context, int64) (*OperationTask, int64, error)
	ListDueTaskIDs(context.Context, int) ([]int64, error)
	CompleteTask(context.Context, int64, int64, string, string, bool, time.Time) error
	ListRuns(context.Context, int, int) ([]*OperationRun, int64, error)
	CountPendingTasks(context.Context) (int64, error)
	CountSucceededSince(context.Context, time.Time) (int, error)
	LastTaskCreatedAt(context.Context, string) (*time.Time, error)
	LastSucceededAt(context.Context, string) (*time.Time, error)
}
