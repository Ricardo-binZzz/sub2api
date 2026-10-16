//go:build unit

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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Exact provider refusal from incident 626572; contains no user prompt or secret.
const cyberWrapperMessage = "This content was flagged for possible cybersecurity risk. If this seems wrong, try rephrasing your request. If you’re doing authorized security work that requires more cyber permissive safeguards, apply for Daybreak access via https://platform.openai.com/settings/organization/status-and-access before retrying."

func cyberWrapperBody() string {
	b, _ := json.Marshal(map[string]any{"error": map[string]any{"type": "content_policy_violation", "message": cyberWrapperMessage}})
	return string(b)
}

func cyberWrapperEvent(kind string) string {
	errObject := gjson.Get(cyberWrapperBody(), "error").Raw
	usage := `"usage":{"input_tokens":9,"output_tokens":2}`
	if kind == "error" {
		return `{"type":"error","error":` + errObject + `,` + usage + `}`
	}
	return `{"type":"response.failed","response":{"id":"resp_cyber","object":"response","status":"failed","model":"gpt-5.5","output":[],"error":` + errObject + `,` + usage + `}}`
}

func TestCyberWrapperDetectionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		hit        bool
	}{
		{"incident", cyberWrapperBody(), true},
		{"failed", cyberWrapperEvent("response.failed"), true},
		{"bare error", cyberWrapperEvent("error"), true},
		{"code wrapper", strings.Replace(cyberWrapperBody(), `"type":`, `"code":`, 1), true},
		{"case whitespace", `{"error":{"type":" CONTENT_POLICY_VIOLATION ","message":" THIS CONTENT WAS FLAGGED FOR POSSIBLE CYBERSECURITY RISK. "}}`, true},
		{"generic content policy", `{"error":{"type":"content_policy_violation","message":"Content violates safety policy"}}`, false},
		{"auth with same message", strings.Replace(cyberWrapperBody(), "content_policy_violation", "authentication_error", 1), false},
		{"message without classification", `{"error":{"message":` + fmt.Sprintf("%q", cyberWrapperMessage) + `}}`, false},
		{"echoed input", `{"error":{"type":"content_policy_violation","message":"Other refusal"},"input":` + fmt.Sprintf("%q", cyberWrapperMessage) + `}`, false},
		{"cross object fields", `{"error":{"type":"content_policy_violation"},"response":{"error":{"message":` + fmt.Sprintf("%q", cyberWrapperMessage) + `}}}`, false},
		{"embedded quotation", `{"error":{"type":"content_policy_violation","message":"Quoted input: This content was flagged for possible cybersecurity risk."}}`, false},
		{"malformed", `{"error":{"code":"cyber_policy"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hit, code, message := detectOpenAICyberPolicy([]byte(tc.body))
			require.Equal(t, tc.hit, hit)
			if hit {
				require.Equal(t, "cyber_policy", code)
				require.NotEmpty(t, message)
			}
		})
	}
}

func TestCyberWrapperNoRetryClassification(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := rawChatCompletionsTestAccount()
	for _, status := range []int{400, 403, 429, 502, 503} {
		require.False(t, svc.shouldFailoverOpenAIUpstreamResponse(account, status, cyberWrapperMessage, []byte(cyberWrapperBody())))
		require.False(t, shouldFailoverOpenAIPassthroughResponse(account, status, []byte(cyberWrapperBody())))
	}
	for _, kind := range []string{"error", "response.failed"} {
		payload := []byte(cyberWrapperEvent(kind))
		require.False(t, openAIStreamFailedEventShouldFailover(payload, cyberWrapperMessage))
		require.False(t, openAIStreamErrorEventShouldFailover(payload, cyberWrapperMessage))
	}
	// Genuine credential errors keep their failover behavior.
	denied := []byte(`{"error":{"code":"account_disabled","message":"Your account is disabled"}}`)
	require.True(t, svc.shouldFailoverOpenAIUpstreamResponse(account, 403, "Your account is disabled", denied))
}

func TestCyberWrapperHTTPForwardPreservesMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, passthrough := range []bool{false, true} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("passthrough=%v/stream=%v", passthrough, stream), func(t *testing.T) {
				c, rec := newOpenAIUpstreamErrorTestContext(t)
				body := []byte(fmt.Sprintf(`{"model":"gpt-5.5","input":"hello","stream":%v}`, stream))
				up := &httpUpstreamRecorder{resp: newOpenAIUpstreamErrorResponse(403, cyberWrapperBody())}
				svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: up, toolCorrector: NewCodexToolCorrector()}
				account := rawChatCompletionsTestAccount()
				account.Extra = map[string]any{"openai_passthrough": passthrough}
				_, err := svc.Forward(context.Background(), c, account, body)
				require.Error(t, err)
				var failover *UpstreamFailoverError
				require.False(t, errors.As(err, &failover))
				require.Len(t, up.requests, 1, "refusal must not retry upstream")
				require.Equal(t, 403, rec.Code)
				require.Equal(t, cyberWrapperMessage, gjson.Get(rec.Body.String(), "error.message").String())
				require.NotNil(t, GetOpsCyberPolicy(c))
			})
		}
	}
}

func TestCyberWrapperCompatHTTPPreservesMessage(t *testing.T) {
	for _, writer := range []compatErrorWriter{writeChatCompletionsError, writeAnthropicError} {
		c, rec := newOpenAIUpstreamErrorTestContext(t)
		svc := &OpenAIGatewayService{cfg: &config.Config{}}
		_, err := svc.handleCompatErrorResponse(newOpenAIUpstreamErrorResponse(403, cyberWrapperBody()), c, rawChatCompletionsTestAccount(), writer)
		require.Error(t, err)
		require.Equal(t, 403, rec.Code)
		require.Equal(t, cyberWrapperMessage, gjson.Get(rec.Body.String(), "error.message").String())
		require.NotNil(t, GetOpsCyberPolicy(c))
	}
}

func TestCyberWrapperResponsesSSEPreservesMessage(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, buffered := range []bool{false, true} {
			for _, kind := range []string{"error", "response.failed"} {
				t.Run(fmt.Sprintf("passthrough=%v/buffered=%v/%s", passthrough, buffered, kind), func(t *testing.T) {
					c, rec := newOpenAIUpstreamErrorTestContext(t)
					body := "event: " + kind + "\ndata: " + cyberWrapperEvent(kind) + "\n\n"
					resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
					svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}}
					var err error
					account := rawChatCompletionsTestAccount()
					if buffered {
						if passthrough {
							_, err = svc.handlePassthroughSSEToJSON(resp, c, account, []byte(body), "gpt-5.5", "gpt-5.5")
						} else {
							_, err = svc.handleSSEToJSON(resp, c, account, []byte(body), "gpt-5.5", "gpt-5.5")
						}
						require.Equal(t, 403, rec.Code)
						require.Equal(t, cyberWrapperMessage, gjson.Get(rec.Body.String(), "error.message").String())
					} else {
						if passthrough {
							_, err = svc.handleStreamingResponsePassthrough(c.Request.Context(), resp, c, account, time.Now(), "gpt-5.5", "gpt-5.5")
						} else {
							_, err = svc.handleStreamingResponse(c.Request.Context(), resp, c, account, time.Now(), "gpt-5.5", "gpt-5.5")
						}
						require.Contains(t, rec.Body.String(), cyberWrapperMessage)
					}
					require.Error(t, err)
					var failover *UpstreamFailoverError
					require.False(t, errors.As(err, &failover))
					mark := GetOpsCyberPolicy(c)
					require.NotNil(t, mark)
					require.Equal(t, cyberWrapperMessage, mark.Message)
					require.Equal(t, 9, mark.UpstreamInTok)
					require.Equal(t, 2, mark.UpstreamOutTok)
				})
			}
		}
	}
}

func TestCyberWrapperCompatSSEPreservesMessage(t *testing.T) {
	for _, kind := range []string{"error", "response.failed"} {
		for _, anthropic := range []bool{false, true} {
			for _, stream := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/anthropic=%v/stream=%v", kind, anthropic, stream), func(t *testing.T) {
					c, rec := newOpenAIUpstreamErrorTestContext(t)
					body := []byte(fmt.Sprintf(`{"model":"gpt-5.5","max_tokens":32,"messages":[{"role":"user","content":"hi"}],"stream":%v}`, stream))
					up := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("event: " + kind + "\ndata: " + cyberWrapperEvent(kind) + "\n\n"))}}
					svc := &OpenAIGatewayService{httpUpstream: up}
					var err error
					if anthropic {
						_, err = svc.ForwardAsAnthropic(context.Background(), c, compatCyberOAuthAccount(), body, "", "gpt-5.5")
					} else {
						_, err = svc.ForwardAsChatCompletions(context.Background(), c, compatCyberOAuthAccount(), body, "", "gpt-5.5")
					}
					require.Error(t, err)
					var failover *UpstreamFailoverError
					require.False(t, errors.As(err, &failover))
					require.Contains(t, rec.Body.String(), cyberWrapperMessage)
					require.NotNil(t, GetOpsCyberPolicy(c))
				})
			}
		}
	}
}

func TestCyberWrapperCompactKeepalivePreservesMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, passthrough := range []bool{false, true} {
		for _, started := range []bool{false, true} {
			t.Run(fmt.Sprintf("passthrough=%v/keepalive=%v", passthrough, started), func(t *testing.T) {
				c, rec := newOpenAIUpstreamErrorTestContext(t)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
				MarkOpenAICompactClientStream(c)
				if started {
					stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
					defer stop()
					waitForKeepaliveBeats()
				}
				svc := &OpenAIGatewayService{cfg: &config.Config{}}
				resp := newOpenAIUpstreamErrorResponse(403, cyberWrapperBody())
				if passthrough {
					require.Error(t, svc.handleErrorResponsePassthrough(context.Background(), resp, c, rawChatCompletionsTestAccount(), nil, []byte(cyberWrapperBody())))
				} else {
					_, err := svc.handleErrorResponse(context.Background(), resp, c, rawChatCompletionsTestAccount(), nil)
					require.Error(t, err)
				}
				if started {
					events := parseCompactBridgeSSE(t, stripKeepaliveComments(rec.Body.String()))
					require.Len(t, events, 1)
					require.Equal(t, "response.failed", events[0][0])
					require.Equal(t, cyberWrapperMessage, gjson.Get(events[0][1], "response.error.message").String())
				} else {
					require.Equal(t, 403, rec.Code)
					require.Equal(t, cyberWrapperMessage, gjson.Get(rec.Body.String(), "error.message").String())
				}
			})
		}
	}
}

func TestCyberWrapperPassthroughDoesNotExposeProviderMetadata(t *testing.T) {
	c, rec := newOpenAIUpstreamErrorTestContext(t)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	body := strings.TrimSuffix(cyberWrapperBody(), "}") + `,"provider_debug":"private metadata"}`
	resp := newOpenAIUpstreamErrorResponse(403, body)
	resp.Header.Set("X-Provider-Secret", "private header")
	err := svc.handleErrorResponsePassthrough(context.Background(), resp, c, rawChatCompletionsTestAccount(), nil, []byte(body))
	require.Error(t, err)
	require.Equal(t, 403, rec.Code)
	require.Equal(t, cyberWrapperMessage, gjson.Get(rec.Body.String(), "error.message").String())
	require.NotContains(t, rec.Body.String(), "private metadata")
	require.Empty(t, rec.Header().Get("X-Provider-Secret"))
}

func TestCyberWrapperWSForwardPreservesMessage(t *testing.T) {
	for _, kind := range []string{"error", "response.failed"} {
		t.Run(kind, func(t *testing.T) {
			c, rec := newOpenAIUpstreamErrorTestContext(t)
			cfg := newOpenAIWSV2TestConfig()
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
			conn := &openAIWSCaptureConn{events: [][]byte{[]byte(cyberWrapperEvent(kind))}}
			pool := newOpenAIWSConnPool(cfg)
			pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: conn})
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &httpUpstreamRecorder{}, cache: &stubGatewayCache{}, openaiWSResolver: NewOpenAIWSProtocolResolver(cfg), toolCorrector: NewCodexToolCorrector(), openaiWSPool: pool}
			account := rawChatCompletionsTestAccount()
			account.Status = StatusActive
			account.Schedulable = true
			account.Extra = map[string]any{"responses_websockets_v2_enabled": true}
			_, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.5","stream":false,"input":"hello"}`))
			var failover *UpstreamFailoverError
			require.False(t, errors.As(err, &failover))
			require.NotNil(t, GetOpsCyberPolicy(c))
			require.Contains(t, rec.Body.String(), cyberWrapperMessage)
		})
	}
}

func TestCyberWrapperWSHTTPBridgePreservesMessage(t *testing.T) {
	for _, kind := range []string{"http", "error", "response.failed"} {
		t.Run(kind, func(t *testing.T) {
			status, body := 403, cyberWrapperBody()
			if kind != "http" {
				status = 200
				body = "data: " + cyberWrapperEvent(kind) + "\n\n"
			}
			up := &httpUpstreamRecorder{resp: newOpenAIUpstreamErrorResponse(status, body)}
			svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: up}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
			payload := []byte(`{"type":"response.create","model":"gpt-5.5","input":"hi"}`)
			var writes []string
			_, err := svc.proxyOpenAIWSHTTPBridgeTurn(context.Background(), c, rawChatCompletionsTestAccount(), "sk-test", payload, len(payload), "gpt-5.5", "", "", "", "", 1, func(b []byte) error { writes = append(writes, string(b)); return nil })
			var failover *UpstreamFailoverError
			require.False(t, errors.As(err, &failover))
			require.NotNil(t, GetOpsCyberPolicy(c))
			require.Contains(t, strings.Join(writes, "\n"), cyberWrapperMessage)
		})
	}
}
