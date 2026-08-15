package service

import (
	"context"
	"errors"
	"time"
)

const AssistantRewardCampaignNewUser = "new_user_v1"

var (
	ErrAssistantRewardDisabled = errors.New("assistant reward is disabled")
	ErrAssistantRewardNotFound = errors.New("assistant reward decision not found")
)

type AssistantRewardDecision struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"user_id"`
	Campaign          string    `json:"campaign"`
	AISuggestedAmount float64   `json:"ai_suggested_amount"`
	AIReason          string    `json:"ai_reason"`
	AIConfidence      float64   `json:"ai_confidence"`
	Eligible          bool      `json:"eligible"`
	RuleReasons       []string  `json:"rule_reasons"`
	FinalAmount       float64   `json:"final_amount"`
	Status            string    `json:"status"`
	BalanceBefore     *float64  `json:"balance_before,omitempty"`
	BalanceAfter      *float64  `json:"balance_after,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AssistantRewardDecisionInput struct {
	UserID            int64
	Campaign          string
	IdempotencyKey    string
	AISuggestedAmount float64
	AIReason          string
	AIConfidence      float64
	Eligible          bool
	RuleReasons       []string
	FinalAmount       float64
}

type AssistantRewardDecisionList struct {
	Items    []*AssistantRewardDecision `json:"items"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}

type AssistantRewardStatus struct {
	Enabled  bool                     `json:"enabled"`
	CanApply bool                     `json:"can_apply"`
	Claimed  bool                     `json:"claimed"`
	Reason   string                   `json:"reason,omitempty"`
	Decision *AssistantRewardDecision `json:"decision,omitempty"`
}

type AssistantRewardRepository interface {
	GetByUserCampaign(ctx context.Context, userID int64, campaign string) (*AssistantRewardDecision, error)
	CreateAndApply(ctx context.Context, input AssistantRewardDecisionInput) (*AssistantRewardDecision, bool, error)
	List(ctx context.Context, page, pageSize int) (*AssistantRewardDecisionList, error)
}
