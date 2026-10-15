package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Exercise the actual /responses Forward path used by API-key accounts, including
// the bytes sent upstream and the tier handed to billing. Policy-helper-only
// tests do not catch a separate forwarding path hardcoding priority.
func TestForward_GroupAstraUltrafastRegression(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, model, requested, astraTier, want string
		force                                   bool
	}{
		{"astra_missing", "gpt-6-astra", "", "ultrafast", "ultrafast", true},
		{"astra_priority", "gpt-6-astra", "priority", "ultrafast", "ultrafast", true},
		{"astra_explicit_ultrafast", "gpt-6-astra", "ultrafast", "ultrafast", "ultrafast", true},
		{"other_model_fast", "gpt-6.1-sol", "ultrafast", "ultrafast", "priority", true},
		{"astra_auto_fast", "gpt-6-astra", "ultrafast", "auto", "priority", true},
		{"force_off_preserves", "gpt-6-astra", "ultrafast", "ultrafast", "ultrafast", false},
	} {
		for _, stream := range []bool{false, true} {
			name := tc.name + "/json"
			if stream {
				name = tc.name + "/sse"
			}
			t.Run(name, func(t *testing.T) {
				group := &Group{ID: 106, Platform: PlatformOpenAI, Status: StatusActive,
					Hydrated: true, ForceOpenAIFast: tc.force, ForceOpenAIAstraTier: tc.astraTier}
				ctx := context.WithValue(context.Background(), ctxkey.Group, group)
				payload := map[string]any{"model": tc.model, "input": "hello", "stream": stream}
				if tc.requested != "" {
					payload["service_tier"] = tc.requested
				}
				body, err := json.Marshal(payload)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body)).WithContext(ctx)
				c.Request.Header.Set("Content-Type", "application/json")
				response := `{"id":"resp_regression","object":"response","status":"completed","model":"` + tc.model + `","service_tier":"priority","output":[],"usage":{"input_tokens":1,"output_tokens":1}}`
				contentType := "application/json"
				if stream {
					response = "data: {\"type\":\"response.completed\",\"response\":" + response + "}\n\ndata: [DONE]\n\n"
					contentType = "text/event-stream"
				}
				upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK,
					Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(response))}}
				svc := &OpenAIGatewayService{
					cfg:            &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
					httpUpstream:   upstream,
					settingService: NewSettingService(&openAIFastPolicyRepoStub{values: map[string]string{}}, &config.Config{}),
				}
				account := &Account{ID: 287, Name: "test-api-key", Platform: PlatformOpenAI,
					Type: AccountTypeAPIKey, Concurrency: 1, Status: StatusActive, Schedulable: true,
					Credentials: map[string]any{"api_key": "sk-test"},
					Extra:       map[string]any{"openai_responses_supported": true}}
				result, err := svc.Forward(ctx, c, account, body)
				require.NoError(t, err)
				require.Equal(t, tc.want, gjson.GetBytes(upstream.lastBody, "service_tier").String(), "actual outbound tier")
				require.NotNil(t, result.ServiceTier)
				require.Equal(t, tc.want, *result.ServiceTier, "tier passed to billing")
				if tc.force && tc.want == "ultrafast" {
					require.True(t, groupProtectsAstraUltrafastTier(&APIKey{Group: group}, account, result), "forced Astra billing must remain protected from a lower response tier")
					// Replay the screenshot's token counts through the actual billing
					// calculator with the tier emitted by Forward.
					billing := NewBillingService(&config.Config{}, nil)
					cost, err := billing.CalculateCostWithServiceTier(tc.model, UsageTokens{
						InputTokens: 3, OutputTokens: 170, CacheReadTokens: 18086, CacheCreationTokens: 487,
					}, 0.20, *result.ServiceTier)
					require.NoError(t, err)
					require.InDelta(t, 0.19622100, cost.TotalCost, 1e-10)
					require.InDelta(t, 0.03924420, cost.ActualCost, 1e-10)
				}
			})
		}
	}
}
