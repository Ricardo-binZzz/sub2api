CREATE TABLE IF NOT EXISTS assistant_operation_connections (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(24) NOT NULL,
    name VARCHAR(120) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    encrypted_config TEXT NOT NULL,
    last_cursor VARCHAR(160) NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    last_success_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assistant_operation_connection_platform CHECK (platform IN ('bluesky', 'telegram', 'discord'))
);

CREATE INDEX IF NOT EXISTS idx_assistant_operation_connections_enabled
    ON assistant_operation_connections (enabled, platform);

CREATE TABLE IF NOT EXISTS assistant_operation_policies (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    autonomous_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    auto_publish BOOLEAN NOT NULL DEFAULT FALSE,
    require_approval BOOLEAN NOT NULL DEFAULT TRUE,
    planning_interval_minutes INTEGER NOT NULL DEFAULT 360,
    min_publish_interval_minutes INTEGER NOT NULL DEFAULT 180,
    max_daily_actions INTEGER NOT NULL DEFAULT 3,
    quiet_hours_start SMALLINT NOT NULL DEFAULT 23,
    quiet_hours_end SMALLINT NOT NULL DEFAULT 8,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assistant_operation_policy_limits CHECK (
        planning_interval_minutes BETWEEN 30 AND 10080 AND
        min_publish_interval_minutes BETWEEN 15 AND 10080 AND
        max_daily_actions BETWEEN 0 AND 50 AND
        quiet_hours_start BETWEEN 0 AND 23 AND quiet_hours_end BETWEEN 0 AND 23
    )
);

INSERT INTO assistant_operation_policies (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS assistant_operation_tasks (
    id BIGSERIAL PRIMARY KEY,
    connection_id BIGINT REFERENCES assistant_operation_connections(id) ON DELETE SET NULL,
    kind VARCHAR(32) NOT NULL,
    platform VARCHAR(24) NOT NULL,
    status VARCHAR(24) NOT NULL,
    content TEXT NOT NULL,
    idempotency_key VARCHAR(180) NOT NULL UNIQUE,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    last_error TEXT NOT NULL DEFAULT '',
    external_id VARCHAR(240) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assistant_operation_task_kind CHECK (kind IN ('promotion', 'notification')),
    CONSTRAINT assistant_operation_task_platform CHECK (platform IN ('bluesky', 'telegram', 'discord')),
    CONSTRAINT assistant_operation_task_status CHECK (status IN ('pending_approval', 'approved', 'running', 'succeeded', 'failed', 'cancelled')),
    CONSTRAINT assistant_operation_task_attempts CHECK (attempts >= 0 AND max_attempts BETWEEN 1 AND 10)
);

CREATE INDEX IF NOT EXISTS idx_assistant_operation_tasks_due
    ON assistant_operation_tasks (status, scheduled_at, id);

CREATE TABLE IF NOT EXISTS assistant_operation_runs (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES assistant_operation_tasks(id) ON DELETE CASCADE,
    status VARCHAR(24) NOT NULL,
    output TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    CONSTRAINT assistant_operation_run_status CHECK (status IN ('running', 'succeeded', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_assistant_operation_runs_task
    ON assistant_operation_runs (task_id, started_at DESC);

COMMENT ON TABLE assistant_operation_connections IS 'Encrypted Bluesky and instant-messaging integration settings';
COMMENT ON TABLE assistant_operation_policies IS 'Server-enforced budgets, cadence, quiet hours, and approval policy';
COMMENT ON TABLE assistant_operation_tasks IS 'Auditable autonomous-operation queue';
COMMENT ON TABLE assistant_operation_runs IS 'Immutable external action attempt history';
