package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type assistantRewardRepository struct{ db *sql.DB }

func NewAssistantRewardRepository(db *sql.DB) service.AssistantRewardRepository {
	return &assistantRewardRepository{db: db}
}

const assistantRewardColumns = `id, user_id, campaign, ai_suggested_amount::double precision,
ai_reason, ai_confidence, eligible, rule_reasons, final_amount::double precision,
status, balance_before::double precision, balance_after::double precision, created_at, updated_at`

func (r *assistantRewardRepository) GetByUserCampaign(ctx context.Context, userID int64, campaign string) (*service.AssistantRewardDecision, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+assistantRewardColumns+`
FROM assistant_reward_decisions WHERE user_id = $1 AND campaign = $2`, userID, campaign)
	return scanAssistantReward(row)
}

func (r *assistantRewardRepository) CreateAndApply(ctx context.Context, input service.AssistantRewardDecisionInput) (*service.AssistantRewardDecision, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("begin assistant reward transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var balance float64
	if err := tx.QueryRowContext(ctx, `SELECT balance::double precision FROM users
WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, input.UserID).Scan(&balance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, service.ErrUserNotFound
		}
		return nil, false, err
	}

	existing, err := scanAssistantReward(tx.QueryRowContext(ctx, `SELECT `+assistantRewardColumns+`
FROM assistant_reward_decisions WHERE user_id = $1 AND campaign = $2`, input.UserID, input.Campaign))
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(err, service.ErrAssistantRewardNotFound) {
		return nil, false, err
	}

	status := "rejected"
	balanceAfter := balance
	if input.Eligible && input.FinalAmount > 0 {
		status = "granted"
		if err := tx.QueryRowContext(ctx, `UPDATE users SET balance = balance + $1, updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL RETURNING balance::double precision`, input.FinalAmount, input.UserID).Scan(&balanceAfter); err != nil {
			return nil, false, fmt.Errorf("credit assistant reward: %w", err)
		}
	} else {
		input.FinalAmount = 0
	}

	reasons, err := json.Marshal(input.RuleReasons)
	if err != nil {
		return nil, false, err
	}
	decision, err := scanAssistantReward(tx.QueryRowContext(ctx, `INSERT INTO assistant_reward_decisions (
user_id, campaign, idempotency_key, ai_suggested_amount, ai_reason, ai_confidence,
eligible, rule_reasons, final_amount, status, balance_before, balance_after)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING `+assistantRewardColumns,
		input.UserID, input.Campaign, input.IdempotencyKey, input.AISuggestedAmount,
		input.AIReason, input.AIConfidence, input.Eligible, reasons, input.FinalAmount,
		status, balance, balanceAfter))
	if err != nil {
		return nil, false, fmt.Errorf("insert assistant reward decision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit assistant reward transaction: %w", err)
	}
	return decision, status == "granted", nil
}

func (r *assistantRewardRepository) List(ctx context.Context, page, pageSize int) (*service.AssistantRewardDecisionList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assistant_reward_decisions`).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+assistantRewardColumns+`
FROM assistant_reward_decisions ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*service.AssistantRewardDecision, 0)
	for rows.Next() {
		item, err := scanAssistantReward(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &service.AssistantRewardDecisionList{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

type assistantRewardScanner interface{ Scan(...any) error }

func scanAssistantReward(scanner assistantRewardScanner) (*service.AssistantRewardDecision, error) {
	var item service.AssistantRewardDecision
	var reasons []byte
	var before, after sql.NullFloat64
	err := scanner.Scan(&item.ID, &item.UserID, &item.Campaign, &item.AISuggestedAmount,
		&item.AIReason, &item.AIConfidence, &item.Eligible, &reasons, &item.FinalAmount,
		&item.Status, &before, &after, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrAssistantRewardNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(reasons, &item.RuleReasons); err != nil {
		return nil, err
	}
	if before.Valid {
		item.BalanceBefore = &before.Float64
	}
	if after.Valid {
		item.BalanceAfter = &after.Float64
	}
	return &item, nil
}
