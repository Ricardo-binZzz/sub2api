//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAssistantRewardRepository_CreateAndApply(t *testing.T) {
	ctx := context.Background()
	repo := NewAssistantRewardRepository(integrationDB)

	createUser := func(t *testing.T, balance, totalRecharged float64) *service.User {
		t.Helper()
		user := mustCreateUser(t, integrationEntClient, &service.User{
			Email:   fmt.Sprintf("assistant-reward-%d@example.com", time.Now().UnixNano()),
			Balance: balance,
		})
		_, err := integrationDB.ExecContext(ctx,
			"UPDATE users SET total_recharged = $1 WHERE id = $2", totalRecharged, user.ID)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
		})
		return user
	}

	readBalances := func(t *testing.T, userID int64) (float64, float64) {
		t.Helper()
		var balance, totalRecharged float64
		err := integrationDB.QueryRowContext(ctx, `SELECT balance::double precision,
	total_recharged::double precision FROM users WHERE id = $1`, userID).
			Scan(&balance, &totalRecharged)
		require.NoError(t, err)
		return balance, totalRecharged
	}

	t.Run("grants once and records balance snapshots", func(t *testing.T) {
		user := createUser(t, 2.5, 9)
		input := service.AssistantRewardDecisionInput{
			UserID:            user.ID,
			Campaign:          service.AssistantRewardCampaignNewUser,
			IdempotencyKey:    fmt.Sprintf("assistant-reward:%d", user.ID),
			AISuggestedAmount: 1.25,
			AIReason:          "first visit",
			AIConfidence:      0.8,
			Eligible:          true,
			RuleReasons:       []string{"eligible"},
			FinalAmount:       1.25,
		}

		decision, applied, err := repo.CreateAndApply(ctx, input)
		require.NoError(t, err)
		require.True(t, applied)
		require.Equal(t, "granted", decision.Status)
		require.InDelta(t, 1.25, decision.FinalAmount, 1e-9)
		require.NotNil(t, decision.BalanceBefore)
		require.NotNil(t, decision.BalanceAfter)
		require.InDelta(t, 2.5, *decision.BalanceBefore, 1e-9)
		require.InDelta(t, 3.75, *decision.BalanceAfter, 1e-9)

		balance, totalRecharged := readBalances(t, user.ID)
		require.InDelta(t, 3.75, balance, 1e-9)
		require.InDelta(t, 9, totalRecharged, 1e-9, "assistant gifts must not count as recharge")

		duplicate, duplicateApplied, err := repo.CreateAndApply(ctx, input)
		require.NoError(t, err)
		require.False(t, duplicateApplied)
		require.Equal(t, decision.ID, duplicate.ID)

		balance, totalRecharged = readBalances(t, user.ID)
		require.InDelta(t, 3.75, balance, 1e-9)
		require.InDelta(t, 9, totalRecharged, 1e-9)

		var decisionCount int
		require.NoError(t, integrationDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM assistant_reward_decisions WHERE user_id = $1", user.ID).
			Scan(&decisionCount))
		require.Equal(t, 1, decisionCount)
	})

	t.Run("records rejection without changing balance", func(t *testing.T) {
		user := createUser(t, 4.5, 7)
		decision, applied, err := repo.CreateAndApply(ctx, service.AssistantRewardDecisionInput{
			UserID:            user.ID,
			Campaign:          service.AssistantRewardCampaignNewUser,
			IdempotencyKey:    fmt.Sprintf("assistant-reward:%d", user.ID),
			AISuggestedAmount: 1,
			AIReason:          "account too new",
			AIConfidence:      0.7,
			Eligible:          false,
			RuleReasons:       []string{"account_too_new"},
			FinalAmount:       1,
		})
		require.NoError(t, err)
		require.False(t, applied)
		require.Equal(t, "rejected", decision.Status)
		require.Zero(t, decision.FinalAmount)
		require.Equal(t, []string{"account_too_new"}, decision.RuleReasons)
		require.NotNil(t, decision.BalanceBefore)
		require.NotNil(t, decision.BalanceAfter)
		require.InDelta(t, 4.5, *decision.BalanceBefore, 1e-9)
		require.InDelta(t, 4.5, *decision.BalanceAfter, 1e-9)

		balance, totalRecharged := readBalances(t, user.ID)
		require.InDelta(t, 4.5, balance, 1e-9)
		require.InDelta(t, 7, totalRecharged, 1e-9)
	})
}
