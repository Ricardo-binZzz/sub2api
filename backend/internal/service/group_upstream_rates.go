package service

import (
	"context"
	"math"
	"time"
)

type GroupUpstreamRateAccount struct {
	AccountID                 int64      `json:"account_id"`
	AccountName               string     `json:"account_name"`
	Platform                  string     `json:"platform"`
	Status                    string     `json:"status"`
	Schedulable               bool       `json:"schedulable"`
	ProbeSupported            bool       `json:"probe_supported"`
	ProbeStatus               string     `json:"probe_status"`
	DeclaredRateMultiplier    *float64   `json:"declared_rate_multiplier,omitempty"`
	EffectiveRateMultiplier   *float64   `json:"effective_rate_multiplier,omitempty"`
	PeakMaximumRateMultiplier *float64   `json:"peak_maximum_rate_multiplier,omitempty"`
	PeakRateEnabled           bool       `json:"peak_rate_enabled"`
	PeakRateMultiplier        *float64   `json:"peak_rate_multiplier,omitempty"`
	ReceivedAt                *time.Time `json:"received_at,omitempty"`
	FreshUntil                *time.Time `json:"fresh_until,omitempty"`
	LastAttemptAt             *time.Time `json:"last_attempt_at,omitempty"`
	Stale                     bool       `json:"stale"`
	LastError                 string     `json:"last_error,omitempty"`
}

type GroupUpstreamRates struct {
	GroupID  int64                      `json:"group_id"`
	Accounts []GroupUpstreamRateAccount `json:"accounts"`
}

func (s *adminServiceImpl) GetGroupUpstreamRates(ctx context.Context, groupID int64) (*GroupUpstreamRates, error) {
	if _, err := s.GetGroup(ctx, groupID); err != nil {
		return nil, err
	}
	accounts, err := s.accountRepo.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return buildGroupUpstreamRates(groupID, accounts, time.Now()), nil
}

func buildGroupUpstreamRates(groupID int64, accounts []Account, now time.Time) *GroupUpstreamRates {
	result := &GroupUpstreamRates{
		GroupID:  groupID,
		Accounts: make([]GroupUpstreamRateAccount, 0, len(accounts)),
	}
	for i := range accounts {
		account := &accounts[i]
		item := GroupUpstreamRateAccount{
			AccountID:      account.ID,
			AccountName:    account.Name,
			Platform:       account.Platform,
			Status:         account.Status,
			Schedulable:    account.Schedulable,
			ProbeSupported: isUpstreamBillingProbeAccount(account),
			ProbeStatus:    "not_probed",
		}
		snapshot := decodeUpstreamBillingProbeSnapshot(account.Extra)
		if snapshot == nil {
			result.Accounts = append(result.Accounts, item)
			continue
		}
		item.ProbeStatus = snapshot.Status
		item.ReceivedAt = snapshot.ReceivedAt
		item.FreshUntil = snapshot.FreshUntil
		if !snapshot.LastAttemptAt.IsZero() {
			lastAttemptAt := snapshot.LastAttemptAt
			item.LastAttemptAt = &lastAttemptAt
		}
		item.Stale = snapshot.FreshUntil == nil || now.After(*snapshot.FreshUntil)
		item.LastError = snapshot.LastError
		if snapshot.Status != UpstreamBillingProbeStatusOK || snapshot.Data == nil {
			result.Accounts = append(result.Accounts, item)
			continue
		}

		base, ok := resolveAccountExtraNumber(snapshot.Data, "resolved_rate_multiplier")
		if !ok || base < 0 || math.IsNaN(base) || math.IsInf(base, 0) {
			result.Accounts = append(result.Accounts, item)
			continue
		}
		item.DeclaredRateMultiplier = groupUpstreamRatePtr(base)
		if effective, effectiveOK := upstreamBillingRateAt(snapshot.Data, now); effectiveOK {
			item.EffectiveRateMultiplier = groupUpstreamRatePtr(effective)
		}
		item.PeakMaximumRateMultiplier = groupUpstreamRatePtr(base)
		if peakEnabled, peakOK := snapshot.Data["peak_rate_enabled"].(bool); peakOK && peakEnabled {
			item.PeakRateEnabled = true
			if peak, multiplierOK := resolveAccountExtraNumber(snapshot.Data, "peak_rate_multiplier"); multiplierOK && peak >= 0 && !math.IsNaN(peak) && !math.IsInf(peak, 0) {
				item.PeakRateMultiplier = groupUpstreamRatePtr(peak)
				if peak > 1 {
					item.PeakMaximumRateMultiplier = groupUpstreamRatePtr(base * peak)
				}
			}
		}
		result.Accounts = append(result.Accounts, item)
	}
	return result
}

func groupUpstreamRatePtr(value float64) *float64 { return &value }
