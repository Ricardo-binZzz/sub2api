package service

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestInjectExplicitResponsesPromptCacheKey(t *testing.T) {
	t.Run("injects explicit session", func(t *testing.T) {
		body, changed, err := injectExplicitResponsesPromptCacheKey([]byte(`{"model":"gpt-5.6-sol","input":"hi"}`), " session-1 ")
		if err != nil || !changed {
			t.Fatalf("injectExplicitResponsesPromptCacheKey() changed=%v err=%v", changed, err)
		}
		if got := gjson.GetBytes(body, "prompt_cache_key").String(); got != "session-1" {
			t.Fatalf("prompt_cache_key=%q, want session-1", got)
		}
	})

	t.Run("preserves body key", func(t *testing.T) {
		original := []byte(`{"model":"gpt-5.6-sol","prompt_cache_key":"body-key","input":"hi"}`)
		body, changed, err := injectExplicitResponsesPromptCacheKey(original, "header-key")
		if err != nil || changed || string(body) != string(original) {
			t.Fatalf("existing key changed: changed=%v err=%v body=%s", changed, err, body)
		}
	})

	t.Run("ignores empty session", func(t *testing.T) {
		original := []byte(`{"model":"gpt-5.6-sol","input":"hi"}`)
		body, changed, err := injectExplicitResponsesPromptCacheKey(original, " ")
		if err != nil || changed || string(body) != string(original) {
			t.Fatalf("empty session changed body: changed=%v err=%v body=%s", changed, err, body)
		}
	})
}
