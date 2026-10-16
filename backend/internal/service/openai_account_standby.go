package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

// Group-scoped standby policy is stored on the backup account, without changing
// its priority or membership in unrelated groups. Values are primary account IDs.
const openAIStandbyGroupsKey = "openai_standby_for_groups"

type openAIStandbyRequestKey struct{}

func withOpenAIStandbyRequest(ctx context.Context, req OpenAIAccountScheduleRequest) context.Context {
	return context.WithValue(ctx, openAIStandbyRequestKey{}, req)
}

func openAIStandbyPrimaryIDs(account *Account, groupID *int64) (ids []int64, configured, valid bool) {
	if account == nil || groupID == nil || account.Extra == nil {
		return nil, false, true
	}
	raw, exists := account.Extra[openAIStandbyGroupsKey]
	if !exists {
		return nil, false, true
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, true, false
	}
	var groups map[string]json.RawMessage
	if json.Unmarshal(encoded, &groups) != nil {
		return nil, true, false
	}
	entry, exists := groups[strconv.FormatInt(*groupID, 10)]
	if !exists {
		return nil, false, true
	}
	if json.Unmarshal(entry, &ids) != nil || len(ids) == 0 || len(ids) > 64 {
		return nil, true, false
	}
	for _, id := range ids {
		if id <= 0 || id == account.ID {
			return nil, true, false
		}
	}
	return ids, true, true
}

func (s *OpenAIGatewayService) openAIStandbyAllowedFromContext(ctx context.Context, account *Account) bool {
	req, ok := ctx.Value(openAIStandbyRequestKey{}).(OpenAIAccountScheduleRequest)
	return !ok || s.openAIStandbyAllowed(ctx, account, req)
}

func (s *OpenAIGatewayService) filterOpenAIStandbyAccounts(ctx context.Context, accounts []Account) []Account {
	if _, ok := ctx.Value(openAIStandbyRequestKey{}).(OpenAIAccountScheduleRequest); !ok {
		return accounts
	}
	filtered := make([]Account, 0, len(accounts))
	for i := range accounts {
		if s.openAIStandbyAllowedFromContext(ctx, &accounts[i]) {
			filtered = append(filtered, accounts[i])
		}
	}
	return filtered
}

func (s *OpenAIGatewayService) openAIStandbyAllowed(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) bool {
	ids, configured, valid := openAIStandbyPrimaryIDs(account, req.GroupID)
	if !configured {
		return true
	}
	if !valid || s == nil || (s.schedulerSnapshot == nil && s.accountRepo == nil) {
		return false
	}
	// Never move a stateful continuation off its actual owner merely because a
	// primary recovered. Ordinary session stickiness does not bypass this gate.
	if req.GuardianParentAccountID == account.ID {
		return true
	}
	if !req.PreviousResponseCanMove && strings.TrimSpace(req.PreviousResponseID) != "" {
		if store := s.getOpenAIWSStateStore(); store != nil {
			owner, err := store.GetResponseAccount(ctx, derefGroupID(req.GroupID), req.PreviousResponseID)
			if err == nil && owner == account.ID {
				return true
			}
		}
	}
	checker := &defaultOpenAIAccountScheduler{service: s}
	for _, id := range ids {
		if _, excluded := req.ExcludedIDs[id]; excluded {
			continue
		}
		primary, err := s.getSchedulableAccount(ctx, id)
		// A failed state lookup is not evidence that the primary is unhealthy.
		if err != nil {
			return false
		}
		if primary == nil {
			continue
		}
		if _, chained, _ := openAIStandbyPrimaryIDs(primary, req.GroupID); chained {
			return false
		}
		if !s.openAIAccountMatchesSchedulingGroup(primary, req.GroupID) {
			continue
		}
		if !isOpenAICompatibleAccountEligibleForRequest(ctx, primary, req.Platform, req.RequestedModel, req.RequireCompact, req.RequiredCapability) {
			continue
		}
		if !checker.isAccountRequestCompatible(ctx, primary, req) || !checker.isAccountTransportCompatible(primary, req.RequiredTransport) {
			continue
		}
		return false
	}
	return true
}
