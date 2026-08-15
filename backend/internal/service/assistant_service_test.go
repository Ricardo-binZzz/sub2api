package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestAssistantStatusRequiresCompleteValidConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		enabled bool
	}{
		{name: "complete", enabled: true},
		{name: "disabled", mutate: func(cfg *config.Config) { cfg.Assistant.Enabled = false }},
		{name: "missing key", mutate: func(cfg *config.Config) { cfg.Assistant.APIKey = "" }},
		{name: "missing model", mutate: func(cfg *config.Config) { cfg.Assistant.Model = "" }},
		{name: "invalid URL", mutate: func(cfg *config.Config) { cfg.Assistant.BaseURL = "://bad" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := assistantTestConfig("https://example.com/v1")
			if tt.mutate != nil {
				tt.mutate(cfg)
			}
			status := NewAssistantService(cfg, nil, nil, nil, nil, nil).Status()
			if status.Enabled != tt.enabled {
				t.Fatalf("Enabled = %v, want %v", status.Enabled, tt.enabled)
			}
			if !tt.enabled && status.Model != "" {
				t.Fatalf("disabled status exposed model %q", status.Model)
			}
		})
	}
}

func TestAssistantValidateQuestion(t *testing.T) {
	cfg := assistantTestConfig("https://example.com/v1")
	cfg.Assistant.MaxQuestionChars = 2
	svc := NewAssistantService(cfg, nil, nil, nil, nil, nil)

	if got, err := svc.validateQuestion(" 你好 "); err != nil || got != "你好" {
		t.Fatalf("validateQuestion() = %q, %v", got, err)
	}
	if _, err := svc.validateQuestion("你好啊"); !errors.Is(err, ErrAssistantInvalidQuestion) {
		t.Fatalf("overlong Unicode question error = %v", err)
	}
	if _, err := svc.validateQuestion("  "); !errors.Is(err, ErrAssistantInvalidQuestion) {
		t.Fatalf("empty question error = %v", err)
	}

	cfg.Assistant.Enabled = false
	if _, err := svc.validateQuestion("ok"); !errors.Is(err, ErrAssistantDisabled) {
		t.Fatalf("disabled question error = %v", err)
	}
}

func TestAssistantContextIsBoundedAndSanitized(t *testing.T) {
	cfg := assistantTestConfig("https://example.com/v1")
	svc := NewAssistantService(cfg, nil, nil, nil, nil, nil)
	now := time.Now().UTC()
	data := &assistantUserContext{
		PeriodStart: now.Add(-time.Hour),
		PeriodEnd:   now,
		Usage:       &UsageStats{TotalRequests: 10},
		Errors: []assistantUserErrorContext{{
			CreatedAt: now,
			Model:     "model",
			Message:   strings.Repeat("x", 200),
		}},
	}

	full, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Assistant.MaxContextBytes = len(full) - 1
	bounded, err := svc.marshalBoundedUserContext(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(bounded) > cfg.Assistant.MaxContextBytes {
		t.Fatalf("context has %d bytes, limit is %d", len(bounded), cfg.Assistant.MaxContextBytes)
	}
	if !strings.Contains(string(bounded), `"truncated":true`) {
		t.Fatalf("bounded context does not report truncation: %s", bounded)
	}

	cfg.Assistant.MaxContextBytes = 1
	if _, err := svc.marshalBoundedUserContext(&assistantUserContext{PeriodStart: now, PeriodEnd: now}); err == nil {
		t.Fatal("expected an error when even the minimal context exceeds the limit")
	}

	safe, err := json.Marshal(assistantUserErrorContext{Message: "safe"})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ip", "user_agent", "key_name", "api_key"} {
		if strings.Contains(string(safe), forbidden) {
			t.Fatalf("sanitized context contains forbidden field %q: %s", forbidden, safe)
		}
	}
}

func TestAssistantHelpers(t *testing.T) {
	if got := truncateRunes("你好世界", 2); got != "你好..." {
		t.Fatalf("truncateRunes() = %q", got)
	}
	if got := truncateRunes("value", 0); got != "" {
		t.Fatalf("truncateRunes(limit=0) = %q", got)
	}

	groups := []*GroupAvailability{
		{GroupID: 2, GroupName: "b"},
		{GroupID: 1, GroupName: "z"},
		{GroupID: 1, GroupName: "a"},
	}
	sortGroupAvailability(groups)
	if groups[0].GroupName != "a" || groups[1].GroupName != "z" || groups[2].GroupID != 2 {
		t.Fatalf("unexpected group order: %#v", groups)
	}
}

func TestValidateAssistantHistory(t *testing.T) {
	history := make([]AssistantMessage, 0, 12)
	for i := 0; i < 12; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		history = append(history, AssistantMessage{Role: role, Content: fmt.Sprintf(" message-%d ", i)})
	}

	bounded, err := validateAssistantHistory(history)
	if err != nil {
		t.Fatal(err)
	}
	if len(bounded) != assistantHistoryMessageLimit {
		t.Fatalf("history length = %d, want %d", len(bounded), assistantHistoryMessageLimit)
	}
	if bounded[0].Content != "message-2" || bounded[len(bounded)-1].Content != "message-11" {
		t.Fatalf("unexpected bounded history: %#v", bounded)
	}

	bounded, err = validateAssistantHistory([]AssistantMessage{{Role: "user", Content: strings.Repeat("界", assistantHistoryRuneLimit+20)}})
	if err != nil {
		t.Fatal(err)
	}
	if got := len([]rune(bounded[0].Content)); got != assistantHistoryRuneLimit {
		t.Fatalf("bounded content has %d runes, want %d", got, assistantHistoryRuneLimit)
	}

	bounded, err = validateAssistantHistory([]AssistantMessage{
		{Role: "user", Content: strings.Repeat("旧", 5000)},
		{Role: "assistant", Content: strings.Repeat("新", 5000)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len([]rune(bounded[0].Content)); got != 3000 {
		t.Fatalf("older message has %d runes, want 3000", got)
	}
	if got := len([]rune(bounded[1].Content)); got != 5000 {
		t.Fatalf("newest message has %d runes, want 5000", got)
	}

	for _, role := range []string{"system", "tool", ""} {
		if _, err := validateAssistantHistory([]AssistantMessage{{Role: role, Content: "inject"}}); !errors.Is(err, ErrAssistantInvalidHistory) {
			t.Fatalf("role %q error = %v", role, err)
		}
	}
	if _, err := validateAssistantHistory([]AssistantMessage{{Role: "user", Content: "  "}}); !errors.Is(err, ErrAssistantInvalidHistory) {
		t.Fatalf("empty history content error = %v", err)
	}
}

func TestAssistantCompleteUsesReadOnlyChatCompletionsPayload(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"  useful answer  "}}]}`)
	}))
	defer server.Close()

	svc := NewAssistantService(assistantTestConfig(server.URL+"/v1"), nil, nil, nil, nil, nil)
	history := []AssistantMessage{
		{Role: "user", Content: "previous question"},
		{Role: "assistant", Content: "previous answer"},
	}
	reply, err := svc.complete(context.Background(), "user", "help", []byte(`{"total_requests":1}`), history)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Answer != "useful answer" {
		t.Fatalf("answer = %q", reply.Answer)
	}
	if _, ok := received["tools"]; ok {
		t.Fatal("read-only assistant payload must not include tools")
	}
	messages, ok := received["messages"].([]any)
	if !ok || len(messages) != 5 {
		t.Fatalf("messages = %#v", received["messages"])
	}
	wantRoles := []string{"system", "system", "user", "assistant", "user"}
	for i, want := range wantRoles {
		message, ok := messages[i].(map[string]any)
		if !ok || message["role"] != want {
			t.Fatalf("message %d = %#v, want role %q", i, messages[i], want)
		}
	}
}

func TestAssistantCompleteRejectsBadProviderResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "non-2xx", status: http.StatusBadGateway, body: `{"error":"bad"}`},
		{name: "invalid JSON", status: http.StatusOK, body: `{`},
		{name: "no choices", status: http.StatusOK, body: `{"choices":[]}`},
		{name: "empty answer", status: http.StatusOK, body: `{"choices":[{"message":{"content":" "}}]}`},
		{name: "too large", status: http.StatusOK, body: strings.Repeat("x", assistantResponseLimit+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()

			svc := NewAssistantService(assistantTestConfig(server.URL+"/v1"), nil, nil, nil, nil, nil)
			if _, err := svc.complete(context.Background(), "user", "help", []byte(`{}`), nil); err == nil {
				t.Fatal("expected provider response error")
			}
		})
	}
}

func TestAssistantRewardRecommendationStrictJSON(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{name: "valid", content: `{"recommended_amount":0.75,"reason":"Useful first visit","confidence":0.8}`},
		{name: "unknown field", content: `{"recommended_amount":0.75,"reason":"ok","confidence":0.8,"grant":true}`, wantErr: true},
		{name: "negative amount", content: `{"recommended_amount":-1,"reason":"ok","confidence":0.8}`, wantErr: true},
		{name: "confidence too high", content: `{"recommended_amount":1,"reason":"ok","confidence":1.1}`, wantErr: true},
		{name: "empty reason", content: `{"recommended_amount":1,"reason":" ","confidence":0.5}`, wantErr: true},
		{name: "trailing object", content: `{"recommended_amount":1,"reason":"ok","confidence":0.5}{}`, wantErr: true},
		{name: "markdown", content: "```json\n{\"recommended_amount\":1,\"reason\":\"ok\",\"confidence\":0.5}\n```", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"choices": []any{map[string]any{"message": map[string]any{"content": tt.content}}},
				})
			}))
			defer server.Close()

			cfg := assistantTestConfig(server.URL + "/v1")
			cfg.Assistant.RewardMaxAmount = 1
			svc := NewAssistantService(cfg, nil, nil, nil, nil, nil)
			user := &User{CreatedAt: time.Now().UTC().Add(-2 * time.Hour)}
			suggestion, err := svc.recommendReward(context.Background(), user, &UsageStats{}, time.Now().UTC())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("recommendReward() = %#v, want error", suggestion)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if suggestion.Amount != 0.75 || suggestion.Reason != "Useful first visit" || suggestion.Confidence != 0.8 {
				t.Fatalf("recommendReward() = %#v", suggestion)
			}
		})
	}
}

func TestClampAssistantReward(t *testing.T) {
	tests := []struct {
		name       string
		amount     float64
		maximum    float64
		wantAmount float64
	}{
		{name: "within limit", amount: 0.5, maximum: 1, wantAmount: 0.5},
		{name: "clamped", amount: 3, maximum: 1, wantAmount: 1},
		{name: "negative", amount: -1, maximum: 1, wantAmount: 0},
		{name: "disabled maximum", amount: 1, maximum: 0, wantAmount: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampAssistantReward(tt.amount, tt.maximum); got != tt.wantAmount {
				t.Fatalf("clampAssistantReward(%v, %v) = %v, want %v", tt.amount, tt.maximum, got, tt.wantAmount)
			}
		})
	}
}

func TestAssistantRewardRules(t *testing.T) {
	now := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	cfg := assistantTestConfig("https://example.com/v1")
	cfg.Assistant.RewardMinAccountAgeMinutes = 60
	cfg.Assistant.RewardEligibilityDays = 30
	svc := NewAssistantService(cfg, nil, nil, nil, nil, nil)

	tests := []struct {
		name       string
		user       *User
		want       bool
		wantReason string
	}{
		{name: "eligible", user: &User{Status: StatusActive, Role: RoleUser, CreatedAt: now.Add(-2 * time.Hour)}, want: true},
		{name: "inactive", user: &User{Status: StatusDisabled, Role: RoleUser, CreatedAt: now.Add(-2 * time.Hour)}, wantReason: "account_inactive"},
		{name: "admin", user: &User{Status: StatusActive, Role: RoleAdmin, CreatedAt: now.Add(-2 * time.Hour)}, wantReason: "not_standard_user"},
		{name: "too new", user: &User{Status: StatusActive, Role: RoleUser, CreatedAt: now.Add(-30 * time.Minute)}, wantReason: "account_too_new"},
		{name: "expired", user: &User{Status: StatusActive, Role: RoleUser, CreatedAt: now.Add(-31 * 24 * time.Hour)}, wantReason: "eligibility_window_expired"},
		{name: "missing registration time", user: &User{Status: StatusActive, Role: RoleUser}, wantReason: "registration_time_unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, reasons := svc.rewardRules(tt.user, now)
			if eligible != tt.want {
				t.Fatalf("eligible = %v, reasons = %v, want %v", eligible, reasons, tt.want)
			}
			if tt.wantReason != "" && !containsString(reasons, tt.wantReason) {
				t.Fatalf("reasons = %v, want %q", reasons, tt.wantReason)
			}
		})
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assistantTestConfig(baseURL string) *config.Config {
	cfg := &config.Config{}
	cfg.Assistant = config.AssistantConfig{
		Enabled:          true,
		BaseURL:          baseURL,
		APIKey:           "test-key",
		Model:            "test-model",
		TimeoutSeconds:   2,
		MaxQuestionChars: 2000,
		MaxContextBytes:  32 * 1024,
	}
	cfg.Security.URLAllowlist.AllowPrivateHosts = true
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	return cfg
}
