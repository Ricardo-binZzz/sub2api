package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func standbyTestAccounts() []Account {
	primary := Account{ID: 1, Name: "primary", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 100, Priority: 1, GroupIDs: []int64{99, 104}, Credentials: map[string]any{"api_key": "test", "model_mapping": map[string]any{"gpt-6-sol": "gpt-6-sol"}}}
	second := primary
	second.ID = 2
	backup := primary
	backup.ID = 3
	backup.Name = "standby"
	backup.Priority = 0
	backup.GroupIDs = []int64{82, 99, 104}
	backup.Extra = map[string]any{openAIStandbyGroupsKey: map[string]any{"99": []int64{1, 2}, "104": []int64{1, 2}}}
	return []Account{primary, second, backup}
}

func TestOpenAIStandbyConditions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func([]Account, *OpenAIAccountScheduleRequest)
		allowed bool
	}{
		{"healthy primary", func(a []Account, r *OpenAIAccountScheduleRequest) {}, false},
		{"one primary remains", func(a []Account, r *OpenAIAccountScheduleRequest) { a[0].Schedulable = false }, false},
		{"all disabled", func(a []Account, r *OpenAIAccountScheduleRequest) { a[0].Schedulable = false; a[1].Schedulable = false }, true},
		{"cooldown and rate limit", func(a []Account, r *OpenAIAccountScheduleRequest) {
			until := time.Now().Add(time.Hour)
			a[0].TempUnschedulableUntil = &until
			a[1].RateLimitResetAt = &until
		}, true},
		{"request failover", func(a []Account, r *OpenAIAccountScheduleRequest) { r.ExcludedIDs = map[int64]struct{}{1: {}, 2: {}} }, true},
		{"one failed request", func(a []Account, r *OpenAIAccountScheduleRequest) { r.ExcludedIDs = map[int64]struct{}{1: {}} }, false},
		{"other group unchanged", func(a []Account, r *OpenAIAccountScheduleRequest) { group := int64(82); r.GroupID = &group }, true},
		{"wrong primary group", func(a []Account, r *OpenAIAccountScheduleRequest) {
			a[0].GroupIDs = []int64{80}
			a[1].GroupIDs = []int64{80}
		}, true},
		{"model unavailable on primary", func(a []Account, r *OpenAIAccountScheduleRequest) { r.RequestedModel = "gpt-6-astra" }, true},
		{"sticky does not bypass", func(a []Account, r *OpenAIAccountScheduleRequest) { r.StickyAccountID = 3; r.StickyWeighted = true }, false},
		{"invalid self reference", func(a []Account, r *OpenAIAccountScheduleRequest) {
			a[2].Extra[openAIStandbyGroupsKey] = map[string]any{"99": []int64{3}}
		}, false},
		{"chained standby denied", func(a []Account, r *OpenAIAccountScheduleRequest) {
			a[0].Extra = map[string]any{openAIStandbyGroupsKey: map[string]any{"99": []int64{3}}}
		}, false},
		{"empty config denied", func(a []Account, r *OpenAIAccountScheduleRequest) {
			a[2].Extra[openAIStandbyGroupsKey] = map[string]any{"99": []int64{}}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accounts := standbyTestAccounts()
			group := int64(99)
			req := OpenAIAccountScheduleRequest{GroupID: &group, Platform: PlatformOpenAI, RequestedModel: "gpt-6-sol", RequiredTransport: OpenAIUpstreamTransportAny}
			tc.change(accounts, &req)
			svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts}}
			require.Equal(t, tc.allowed, svc.openAIStandbyAllowed(context.Background(), &accounts[2], req))
		})
	}
}

func TestOpenAIStandbySchedulerRecoveryAndFailover(t *testing.T) {
	for _, advanced := range []bool{false, true} {
		for _, batch := range []bool{false, true} {
			t.Run(fmt.Sprintf("advanced=%v/batch=%v", advanced, batch), func(t *testing.T) {
				resetOpenAIAdvancedSchedulerSettingCacheForTest()
				t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)
				openAIAdvancedSchedulerSettingCache.Store(&cachedOpenAIAdvancedSchedulerSetting{enabled: advanced, stickyWeightedEnabled: true, lbTopKOverride: 3, expiresAt: time.Now().Add(time.Hour).UnixNano()})
				accounts := standbyTestAccounts()
				cfg := &config.Config{}
				cfg.Gateway.Scheduling.LoadBatchEnabled = batch
				cache := &schedulerTestGatewayCache{}
				svc := &OpenAIGatewayService{cfg: cfg, accountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts}, cache: cache, concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{})}
				group := int64(99)
				selectID := func(excluded map[int64]struct{}) int64 {
					selection, _, err := svc.SelectAccountWithScheduler(context.Background(), &group, "", "test-standby-session", "gpt-6-sol", excluded, OpenAIUpstreamTransportAny, false)
					require.NoError(t, err)
					require.NotNil(t, selection)
					require.NotNil(t, selection.Account)
					if selection.ReleaseFunc != nil {
						defer selection.ReleaseFunc()
					}
					return selection.Account.ID
				}
				for i := 0; i < 10; i++ {
					require.NotEqual(t, int64(3), selectID(nil), "healthy primary must win even against backup priority/sticky weight")
				}
				require.Equal(t, int64(3), selectID(map[int64]struct{}{1: {}, 2: {}}))
				accounts[0].Schedulable = false
				accounts[1].Schedulable = false
				require.Equal(t, int64(3), selectID(nil))
				accounts[0].Schedulable = true
				require.Equal(t, int64(1), selectID(nil), "recovery must defeat standby sticky session")
			})
		}
	}
}

func TestOpenAIStandbyPreservesOwnedContinuation(t *testing.T) {
	accounts := standbyTestAccounts()
	group := int64(99)
	svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts}, cache: &schedulerTestGatewayCache{}}
	ctx := context.Background()
	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, group, "resp_standby", 3, time.Hour))
	req := OpenAIAccountScheduleRequest{GroupID: &group, Platform: PlatformOpenAI, RequestedModel: "gpt-6-sol", PreviousResponseID: "resp_standby"}
	require.True(t, svc.openAIStandbyAllowed(ctx, &accounts[2], req))
	req.PreviousResponseCanMove = true
	require.False(t, svc.openAIStandbyAllowed(ctx, &accounts[2], req))
	req.PreviousResponseCanMove = false
	req.PreviousResponseID = "unknown"
	require.False(t, svc.openAIStandbyAllowed(ctx, &accounts[2], req))
}
