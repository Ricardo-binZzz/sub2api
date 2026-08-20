package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRequestParameterPolicyCredentialsRejectsProtectedFields(t *testing.T) {
	credentials := map[string]any{
		credKeyRequestParameterPolicyEnabled: true,
		credKeyRequestParameterPolicy: map[string]any{
			"model": "gpt-5.5",
		},
	}
	err := NormalizeRequestParameterPolicyCredentials(credentials)
	require.ErrorContains(t, err, "protected or unknown field")
}

func TestNormalizeAndApplyRequestParameterPolicy(t *testing.T) {
	credentials := map[string]any{
		credKeyRequestParameterPolicyEnabled: true,
		credKeyRequestParameterPolicy: map[string]any{
			"reasoning_effort":  "high",
			"max_output_tokens": float64(4096),
			"service_tier":      "flex",
			"store":             false,
		},
	}
	require.NoError(t, NormalizeRequestParameterPolicyCredentials(credentials))
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: credentials}
	body := []byte(`{"model":"gpt-5.5","input":"hello","tools":[{"type":"web_search"}],"stream":true,"prompt_cache_key":"keep-me","reasoning":{"summary":"auto"}}`)
	updated, changed, err := account.ApplyRequestParameterPolicy(body)
	require.NoError(t, err)
	require.True(t, changed)
	var got map[string]any
	require.NoError(t, json.Unmarshal(updated, &got))
	require.Equal(t, "gpt-5.5", got["model"])
	require.Equal(t, "hello", got["input"])
	require.Equal(t, true, got["stream"])
	require.Equal(t, "keep-me", got["prompt_cache_key"])
	require.Equal(t, float64(4096), got["max_output_tokens"])
	require.Equal(t, "flex", got["service_tier"])
	require.Equal(t, false, got["store"])
	reasoning := got["reasoning"].(map[string]any)
	require.Equal(t, "auto", reasoning["summary"])
	require.Equal(t, "high", reasoning["effort"])
}

func TestRequestParameterPolicyEligibility(t *testing.T) {
	require.True(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}).IsRequestParameterPolicyEligible())
	require.True(t, (&Account{Platform: PlatformGrok, Type: AccountTypeAPIKey}).IsRequestParameterPolicyEligible())
	require.True(t, (&Account{Platform: PlatformDeepseek, Type: AccountTypeAPIKey}).IsRequestParameterPolicyEligible())
	require.False(t, (&Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}).IsRequestParameterPolicyEligible())
}
