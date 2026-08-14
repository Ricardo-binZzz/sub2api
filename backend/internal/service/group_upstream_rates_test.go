package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func groupUpstreamRateAccount(id int64, name string, declared float64, freshUntil time.Time) Account {
	return Account{
		ID:          id,
		Name:        name,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":          UpstreamBillingProbeStatusOK,
				"received_at":     freshUntil.Add(-time.Hour),
				"fresh_until":     freshUntil,
				"last_attempt_at": freshUntil.Add(-time.Hour),
				"data": map[string]any{
					"billing_scope":             "token",
					"resolved_rate_multiplier":  declared,
					"effective_rate_multiplier": declared,
					"peak_rate_enabled":         false,
				},
			},
		},
	}
}

func TestBuildGroupUpstreamRatesShowsDeclarationsWithoutConfiguredRateFallback(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	grok := groupUpstreamRateAccount(1, "Grok", 0.13, now.Add(time.Hour))
	configured := 1.0
	grok.RateMultiplier = &configured
	aiHub := groupUpstreamRateAccount(2, "ai hub", 0.06, now.Add(time.Hour))
	configured = 1.5
	aiHub.RateMultiplier = &configured

	got := buildGroupUpstreamRates(10, []Account{grok, aiHub}, now)

	require.Equal(t, int64(10), got.GroupID)
	require.Len(t, got.Accounts, 2)
	require.Equal(t, 0.13, *got.Accounts[0].DeclaredRateMultiplier)
	require.Equal(t, 0.13, *got.Accounts[0].EffectiveRateMultiplier)
	require.Equal(t, 0.06, *got.Accounts[1].DeclaredRateMultiplier)
	require.Equal(t, 0.06, *got.Accounts[1].EffectiveRateMultiplier)
}

func TestBuildGroupUpstreamRatesShowsPeakMaximum(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	account := groupUpstreamRateAccount(1, "peak", 0.2, now.Add(time.Hour))
	probe := account.Extra[UpstreamBillingProbeExtraKey].(map[string]any)
	data := probe["data"].(map[string]any)
	data["peak_rate_enabled"] = true
	data["peak_rate_multiplier"] = 1.5
	data["peak_start"] = "00:00"
	data["peak_end"] = "23:59"
	data["timezone"] = "UTC"

	got := buildGroupUpstreamRates(10, []Account{account}, now)

	require.Equal(t, 0.2, *got.Accounts[0].DeclaredRateMultiplier)
	require.InDelta(t, 0.3, *got.Accounts[0].EffectiveRateMultiplier, 1e-12)
	require.InDelta(t, 0.3, *got.Accounts[0].PeakMaximumRateMultiplier, 1e-12)
	require.Equal(t, 1.5, *got.Accounts[0].PeakRateMultiplier)
}

func TestBuildGroupUpstreamRatesKeepsMissingAndFailedAccountsVisible(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	notProbed := Account{ID: 1, Name: "new", Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	failed := Account{
		ID: 2, Name: "failed", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Extra: map[string]any{UpstreamBillingProbeExtraKey: map[string]any{
			"status": UpstreamBillingProbeStatusFailed, "last_attempt_at": now, "last_error": "timeout",
		}},
	}

	got := buildGroupUpstreamRates(10, []Account{notProbed, failed}, now)

	require.Len(t, got.Accounts, 2)
	require.Equal(t, "not_probed", got.Accounts[0].ProbeStatus)
	require.Nil(t, got.Accounts[0].DeclaredRateMultiplier)
	require.Equal(t, UpstreamBillingProbeStatusFailed, got.Accounts[1].ProbeStatus)
	require.Equal(t, "timeout", got.Accounts[1].LastError)
}
