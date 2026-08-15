package service

import (
	"context"
	"encoding/json"
	"errors"
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
			status := NewAssistantService(cfg, nil, nil).Status()
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
	svc := NewAssistantService(cfg, nil, nil)

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
	svc := NewAssistantService(cfg, nil, nil)
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

	svc := NewAssistantService(assistantTestConfig(server.URL+"/v1"), nil, nil)
	reply, err := svc.complete(context.Background(), "user", "help", []byte(`{"total_requests":1}`))
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
	if !ok || len(messages) != 3 {
		t.Fatalf("messages = %#v", received["messages"])
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

			svc := NewAssistantService(assistantTestConfig(server.URL+"/v1"), nil, nil)
			if _, err := svc.complete(context.Background(), "user", "help", []byte(`{}`)); err == nil {
				t.Fatal("expected provider response error")
			}
		})
	}
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
