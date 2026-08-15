package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type operationRepository struct{ db *sql.DB }

func NewOperationRepository(db *sql.DB) service.OperationRepository {
	return &operationRepository{db: db}
}

const operationConnectionColumns = `id, platform, name, enabled, encrypted_config, last_cursor,
last_error, last_success_at, created_at, updated_at`

func (r *operationRepository) ListConnections(ctx context.Context) ([]*service.OperationConnectionRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+operationConnectionColumns+` FROM assistant_operation_connections ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*service.OperationConnectionRecord, 0)
	for rows.Next() {
		item, err := scanOperationConnection(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *operationRepository) GetConnection(ctx context.Context, id int64) (*service.OperationConnectionRecord, error) {
	return scanOperationConnection(r.db.QueryRowContext(ctx, `SELECT `+operationConnectionColumns+` FROM assistant_operation_connections WHERE id=$1`, id))
}

func (r *operationRepository) SaveConnection(ctx context.Context, v *service.OperationConnectionRecord) (*service.OperationConnectionRecord, error) {
	if v.ID == 0 {
		return scanOperationConnection(r.db.QueryRowContext(ctx, `INSERT INTO assistant_operation_connections(platform,name,enabled,encrypted_config)
VALUES($1,$2,$3,$4) RETURNING `+operationConnectionColumns, v.Platform, v.Name, v.Enabled, v.EncryptedConfig))
	}
	return scanOperationConnection(r.db.QueryRowContext(ctx, `UPDATE assistant_operation_connections SET platform=$2,name=$3,enabled=$4,
encrypted_config=$5,updated_at=NOW() WHERE id=$1 RETURNING `+operationConnectionColumns,
		v.ID, v.Platform, v.Name, v.Enabled, v.EncryptedConfig))
}

func (r *operationRepository) DeleteConnection(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM assistant_operation_connections WHERE id=$1`, id)
	return operationAffected(res, err)
}
func (r *operationRepository) UpdateConnectionHealth(ctx context.Context, id int64, message string, success bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE assistant_operation_connections SET last_error=$2,
last_success_at=CASE WHEN $3 THEN NOW() ELSE last_success_at END,updated_at=NOW() WHERE id=$1`, id, message, success)
	return err
}
func (r *operationRepository) UpdateConnectionCursor(ctx context.Context, id int64, cursor string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE assistant_operation_connections SET last_cursor=$2,updated_at=NOW() WHERE id=$1`, id, cursor)
	return err
}

func (r *operationRepository) GetPolicy(ctx context.Context) (*service.OperationPolicy, error) {
	var v service.OperationPolicy
	err := r.db.QueryRowContext(ctx, `SELECT enabled,autonomous_enabled,auto_publish,require_approval,planning_interval_minutes,
min_publish_interval_minutes,max_daily_actions,quiet_hours_start,quiet_hours_end,updated_at FROM assistant_operation_policies WHERE id=1`).Scan(
		&v.Enabled, &v.AutonomousEnabled, &v.AutoPublish, &v.RequireApproval, &v.PlanningIntervalMinutes,
		&v.MinPublishIntervalMinutes, &v.MaxDailyActions, &v.QuietHoursStart, &v.QuietHoursEnd, &v.UpdatedAt)
	return &v, err
}
func (r *operationRepository) UpdatePolicy(ctx context.Context, v service.OperationPolicy) (*service.OperationPolicy, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE assistant_operation_policies SET enabled=$1,autonomous_enabled=$2,auto_publish=$3,
require_approval=$4,planning_interval_minutes=$5,min_publish_interval_minutes=$6,max_daily_actions=$7,
quiet_hours_start=$8,quiet_hours_end=$9,updated_at=NOW() WHERE id=1`, v.Enabled, v.AutonomousEnabled, v.AutoPublish,
		v.RequireApproval, v.PlanningIntervalMinutes, v.MinPublishIntervalMinutes, v.MaxDailyActions, v.QuietHoursStart, v.QuietHoursEnd)
	if err != nil {
		return nil, err
	}
	return r.GetPolicy(ctx)
}
func (r *operationRepository) SetAutonomousEnabled(ctx context.Context, enabled bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE assistant_operation_policies SET autonomous_enabled=$1,updated_at=NOW() WHERE id=1`, enabled)
	return err
}

const operationTaskColumns = `id, connection_id, kind, platform, status, content, idempotency_key, scheduled_at,
approved_by, approved_at, attempts, max_attempts, last_error, external_id, created_at, updated_at`

func (r *operationRepository) CreateTask(ctx context.Context, in service.OperationTaskInput) (*service.OperationTask, error) {
	var connection any
	if in.ConnectionID > 0 {
		connection = in.ConnectionID
	}
	return scanOperationTask(r.db.QueryRowContext(ctx, `INSERT INTO assistant_operation_tasks(connection_id,kind,platform,status,content,idempotency_key,scheduled_at)
VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+operationTaskColumns, connection, in.Kind, in.Platform, in.Status, in.Content, in.IdempotencyKey, in.ScheduledAt))
}
func (r *operationRepository) GetTask(ctx context.Context, id int64) (*service.OperationTask, error) {
	return scanOperationTask(r.db.QueryRowContext(ctx, `SELECT `+operationTaskColumns+` FROM assistant_operation_tasks WHERE id=$1`, id))
}
func (r *operationRepository) ListTasks(ctx context.Context, page, size int) ([]*service.OperationTask, int64, error) {
	page, size = normalizePage(page, size)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assistant_operation_tasks`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+operationTaskColumns+` FROM assistant_operation_tasks ORDER BY created_at DESC,id DESC LIMIT $1 OFFSET $2`, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*service.OperationTask, 0)
	for rows.Next() {
		v, e := scanOperationTask(rows)
		if e != nil {
			return nil, 0, e
		}
		items = append(items, v)
	}
	return items, total, rows.Err()
}
func (r *operationRepository) ApproveTask(ctx context.Context, id, userID int64) (*service.OperationTask, error) {
	var approver any
	if userID > 0 {
		approver = userID
	}
	return scanOperationTask(r.db.QueryRowContext(ctx, `UPDATE assistant_operation_tasks SET status='approved',approved_by=$2,approved_at=NOW(),updated_at=NOW()
WHERE id=$1 AND status='pending_approval' RETURNING `+operationTaskColumns, id, approver))
}
func (r *operationRepository) CancelTask(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE assistant_operation_tasks SET status='cancelled',updated_at=NOW() WHERE id=$1 AND status IN ('pending_approval','approved','failed')`, id)
	return operationAffected(res, err)
}
func (r *operationRepository) DeferTask(ctx context.Context, id int64, scheduledAt time.Time, message string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE assistant_operation_tasks SET scheduled_at=$2,last_error=$3,updated_at=NOW()
WHERE id=$1 AND status IN ('approved','failed')`, id, scheduledAt, message)
	return operationAffected(res, err)
}
func (r *operationRepository) BeginTaskRun(ctx context.Context, id int64) (*service.OperationTask, int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	task, err := scanOperationTask(tx.QueryRowContext(ctx, `UPDATE assistant_operation_tasks SET status='running',attempts=attempts+1,updated_at=NOW()
WHERE id=$1 AND status IN ('approved','failed') AND scheduled_at<=NOW() AND attempts<max_attempts RETURNING `+operationTaskColumns, id))
	if err != nil {
		return nil, 0, err
	}
	var runID int64
	if err := tx.QueryRowContext(ctx, `INSERT INTO assistant_operation_runs(task_id,status) VALUES($1,'running') RETURNING id`, id).Scan(&runID); err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}
	return task, runID, nil
}
func (r *operationRepository) ListDueTaskIDs(ctx context.Context, limit int) ([]int64, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM assistant_operation_tasks
WHERE status IN ('approved','failed') AND scheduled_at<=NOW() AND attempts<max_attempts ORDER BY scheduled_at,id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (r *operationRepository) CompleteTask(ctx context.Context, taskID, runID int64, externalID, message string, success bool, retryAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	status := "failed"
	if success {
		status = "succeeded"
	}
	_, err = tx.ExecContext(ctx, `UPDATE assistant_operation_runs SET status=$2,output=CASE WHEN $2='succeeded' THEN $3 ELSE '' END,
error=CASE WHEN $2='failed' THEN $3 ELSE '' END,finished_at=NOW() WHERE id=$1 AND task_id=$4`, runID, status, message, taskID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE assistant_operation_tasks SET status=$2,external_id=$3,last_error=$4,
scheduled_at=CASE WHEN $2='failed' THEN $5 ELSE scheduled_at END,updated_at=NOW() WHERE id=$1`, taskID, status, externalID, message, retryAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *operationRepository) ListRuns(ctx context.Context, page, size int) ([]*service.OperationRun, int64, error) {
	page, size = normalizePage(page, size)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assistant_operation_runs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,task_id,status,output,error,started_at,finished_at FROM assistant_operation_runs ORDER BY started_at DESC,id DESC LIMIT $1 OFFSET $2`, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*service.OperationRun, 0)
	for rows.Next() {
		var v service.OperationRun
		if err := rows.Scan(&v.ID, &v.TaskID, &v.Status, &v.Output, &v.Error, &v.StartedAt, &v.FinishedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, &v)
	}
	return items, total, rows.Err()
}
func (r *operationRepository) CountPendingTasks(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assistant_operation_tasks WHERE status IN ('pending_approval','approved','running','failed') AND attempts<max_attempts`).Scan(&n)
	return n, err
}
func (r *operationRepository) CountSucceededSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assistant_operation_tasks WHERE status='succeeded' AND updated_at >= $1`, since).Scan(&n)
	return n, err
}
func (r *operationRepository) LastSucceededAt(ctx context.Context, kind string) (*time.Time, error) {
	var t time.Time
	err := r.db.QueryRowContext(ctx, `SELECT updated_at FROM assistant_operation_tasks WHERE kind=$1 AND status='succeeded' ORDER BY updated_at DESC LIMIT 1`, kind).Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}
func (r *operationRepository) LastTaskCreatedAt(ctx context.Context, kind string) (*time.Time, error) {
	var t time.Time
	err := r.db.QueryRowContext(ctx, `SELECT created_at FROM assistant_operation_tasks WHERE kind=$1 ORDER BY created_at DESC LIMIT 1`, kind).Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

type operationScanner interface{ Scan(...any) error }

func scanOperationConnection(s operationScanner) (*service.OperationConnectionRecord, error) {
	var v service.OperationConnectionRecord
	err := s.Scan(&v.ID, &v.Platform, &v.Name, &v.Enabled, &v.EncryptedConfig, &v.LastCursor, &v.LastError, &v.LastSuccessAt, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrOperationNotFound
	}
	v.HasSecrets = v.EncryptedConfig != ""
	return &v, err
}
func scanOperationTask(s operationScanner) (*service.OperationTask, error) {
	var v service.OperationTask
	err := s.Scan(&v.ID, &v.ConnectionID, &v.Kind, &v.Platform, &v.Status, &v.Content, &v.IdempotencyKey, &v.ScheduledAt, &v.ApprovedBy, &v.ApprovedAt, &v.Attempts, &v.MaxAttempts, &v.LastError, &v.ExternalID, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrOperationNotFound
	}
	return &v, err
}
func operationAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return service.ErrOperationNotFound
	}
	return nil
}
func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

var _ service.OperationRepository = (*operationRepository)(nil)
