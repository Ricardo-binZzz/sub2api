package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEnsureJSONObjectPromptInjectsJSONInstruction(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"Extract the fields"}],"response_format":{"type":"json_object"},"request_id":9007199254740993}`)

	got, err := ensureJSONObjectPrompt(body)
	require.NoError(t, err)
	require.Equal(t, int64(2), gjson.GetBytes(got, "messages.#").Int())
	require.Equal(t, "system", gjson.GetBytes(got, "messages.0.role").String())
	require.Contains(t, gjson.GetBytes(got, "messages.0.content").String(), "JSON")
	require.Equal(t, "Extract the fields", gjson.GetBytes(got, "messages.1.content").String())
	require.Contains(t, string(got), `"request_id":9007199254740993`)
}

func TestEnsureJSONObjectPromptLeavesExistingJSONInstruction(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"Return JSON"}],"response_format":{"type":" JSON_OBJECT "}}`)

	got, err := ensureJSONObjectPrompt(body)
	require.NoError(t, err)
	require.Equal(t, string(body), string(got))
}

func TestEnsureJSONObjectPromptIgnoresOtherFormats(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"Extract the fields"}]}`)

	got, err := ensureJSONObjectPrompt(body)
	require.NoError(t, err)
	require.Equal(t, string(body), string(got))
}
