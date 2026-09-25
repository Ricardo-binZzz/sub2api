//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekTimingPreservesStreamingAndUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, enabled := range []string{"false", "true"} {
		t.Run(enabled, func(t *testing.T) {
			t.Setenv("DEEPSEEK_UPSTREAM_TIMING", enabled)
			body := []byte(`{"model":"deepseek-reasoner","messages":[{"role":"user","content":"hello"}],"stream":true}`)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
			payload := "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n" +
				"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":9,\"completion_tokens\":4,\"total_tokens\":13}}\n\n" +
				"data: [DONE]\n\n"
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}},
				Body: io.NopCloser(strings.NewReader(payload)),
			}}
			svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
			account := rawChatCompletionsTestAccount()
			account.Platform = "deepseek"
			result, err := svc.forwardAsRawChatCompletions(context.Background(), c, account, body, "")
			require.NoError(t, err)
			require.Equal(t, 9, result.Usage.InputTokens)
			require.Equal(t, 4, result.Usage.OutputTokens)
			require.Equal(t, payload, rec.Body.String())
			require.Equal(t, enabled == "true", httptrace.ContextClientTrace(upstream.lastReq.Context()) != nil)
			_, exists := c.Get("deepseek_upstream_timing")
			require.Equal(t, enabled == "true", exists)
		})
	}
}
