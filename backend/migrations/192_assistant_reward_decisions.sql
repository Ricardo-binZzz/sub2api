CREATE TABLE IF NOT EXISTS assistant_reward_decisions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    campaign VARCHAR(64) NOT NULL,
    idempotency_key VARCHAR(160) NOT NULL UNIQUE,
    ai_suggested_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    ai_reason TEXT NOT NULL DEFAULT '',
    ai_confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    eligible BOOLEAN NOT NULL DEFAULT FALSE,
    rule_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    final_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    status VARCHAR(24) NOT NULL,
    balance_before DECIMAL(20,8),
    balance_after DECIMAL(20,8),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assistant_reward_status_check CHECK (status IN ('granted', 'rejected')),
    CONSTRAINT assistant_reward_confidence_check CHECK (ai_confidence >= 0 AND ai_confidence <= 1),
    CONSTRAINT assistant_reward_amount_check CHECK (ai_suggested_amount >= 0 AND final_amount >= 0),
    CONSTRAINT assistant_reward_user_campaign_unique UNIQUE (user_id, campaign)
);

CREATE INDEX IF NOT EXISTS idx_assistant_reward_decisions_created
    ON assistant_reward_decisions (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_assistant_reward_decisions_status_created
    ON assistant_reward_decisions (status, created_at DESC);

COMMENT ON TABLE assistant_reward_decisions IS 'AI reward suggestions, deterministic rule decisions, and balance snapshots';
COMMENT ON COLUMN assistant_reward_decisions.ai_suggested_amount IS 'Untrusted model suggestion before deterministic clamping';
COMMENT ON COLUMN assistant_reward_decisions.rule_reasons IS 'Deterministic eligibility and risk rule result';
COMMENT ON COLUMN assistant_reward_decisions.final_amount IS 'Amount actually credited; gifts do not increase users.total_recharged';
