package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSanitizeCodexRequestBodyMapRemovesClientEnvironmentMetadata(t *testing.T) {
	ids := &codexFingerprintIDs{
		accountID:           42,
		mode:                codexFingerprintSession,
		installationID:      "00000000-0000-4000-8000-000000000042",
		sessionID:           "00000000-0000-4000-8000-000000000043",
		threadID:            "00000000-0000-4000-8000-000000000044",
		turnID:              "00000000-0000-4000-8000-000000000045",
		windowID:            "00000000-0000-4000-8000-000000000046",
		turnStartedAtUnixMs: 1700000000000,
	}
	body := map[string]any{
		"input":     []any{map[string]any{"type": "message", "text": "hello"}},
		"cwd":       "C:\\client\\repo",
		"workspace": "C:\\client\\repo",
		"git":       map[string]any{"branch": "private-branch"},
		"os":        "windows",
		"arch":      "amd64",
		"terminal":  "powershell",
		"plugin":    map[string]any{"name": "private-plugin"},
		"skills":    []any{"private-skill"},
		"mcp":       map[string]any{"server": "private-mcp"},
		"trace":     "client-trace",
		"metadata":  map[string]any{"user": "client-user"},
		"client_metadata": map[string]any{
			"x-codex-installation-id":  "raw-installation",
			"session_id":               "raw-session",
			"trace":                    "raw-trace",
			"unknown_client_field":     "raw-secret",
			responsesLiteWSMetadataKey: "true",
		},
	}

	require.True(t, sanitizeCodexRequestBodyMap(body, ids))
	for _, key := range []string{
		"cwd", "workspace", "git", "os", "arch", "terminal", "plugin",
		"skills", "mcp", "trace", "metadata",
	} {
		require.NotContains(t, body, key, "client environment field leaked: %s", key)
	}
	clientMetadata, ok := body["client_metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, ids.installationID, clientMetadata["x-codex-installation-id"])
	require.Equal(t, ids.sessionID, clientMetadata["session_id"])
	require.Equal(t, ids.threadID, clientMetadata["thread_id"])
	require.Equal(t, ids.turnID, clientMetadata["turn_id"])
	require.Equal(t, ids.windowID, clientMetadata["x-codex-window-id"])
	require.Equal(t, "true", clientMetadata[responsesLiteWSMetadataKey])
	require.NotContains(t, clientMetadata, "trace")
	require.NotContains(t, clientMetadata, "unknown_client_field")
	require.NotContains(t, jsonValueStringForCodexTest(clientMetadata), "raw-secret")
}

func TestSanitizeCodexRequestBodyRawRemovesClientEnvironmentMetadata(t *testing.T) {
	ids := &codexFingerprintIDs{
		accountID:           7,
		mode:                codexFingerprintSession,
		installationID:      "00000000-0000-4000-8000-000000000007",
		sessionID:           "00000000-0000-4000-8000-000000000008",
		threadID:            "00000000-0000-4000-8000-000000000009",
		turnID:              "00000000-0000-4000-8000-000000000010",
		windowID:            "00000000-0000-4000-8000-000000000011",
		turnStartedAtUnixMs: 1700000000001,
	}
	raw := []byte("{\"input\":[],\"cwd\":\"C:\\\\private\",\"workspace\":\"private-workspace\",\"git\":{\"remote\":\"private-remote\"},\"os\":\"private-os\",\"arch\":\"private-arch\",\"terminal\":\"private-terminal\",\"plugin\":\"private-plugin\",\"skill\":\"private-skill\",\"mcp\":{\"token\":\"private-mcp\"},\"trace\":\"private-trace\",\"metadata\":{\"secret\":\"private-metadata\"},\"client_metadata\":{\"session_id\":\"raw-session\",\"trace\":\"raw-trace\",\"unknown\":\"raw-secret\"}}")

	rewritten, changed, err := sanitizeCodexRequestBodyRaw(raw, ids)
	require.NoError(t, err)
	require.True(t, changed)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rewritten, &body))
	for _, key := range []string{
		"cwd", "workspace", "git", "os", "arch", "terminal", "plugin",
		"skill", "mcp", "trace", "metadata",
	} {
		require.NotContains(t, body, key, "client environment field leaked: %s", key)
	}
	clientMetadata, ok := body["client_metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, ids.installationID, clientMetadata["x-codex-installation-id"])
	require.Equal(t, ids.sessionID, clientMetadata["session_id"])
	require.NotContains(t, clientMetadata, "trace")
	require.NotContains(t, clientMetadata, "unknown")
	require.NotContains(t, jsonValueStringForCodexTest(clientMetadata), "raw-secret")
}

func TestSanitizeCodexTurnMetadataValueRebuildsClosedSchema(t *testing.T) {
	raw := "{\"installation_id\":\"raw-installation\",\"session_id\":\"raw-session\",\"thread_id\":\"raw-thread\",\"cwd\":\"C:\\\\private\",\"trace\":\"private-trace\",\"sandbox\":{\"secret\":\"private-secret\"}}"
	fields := map[string]any{
		"installation_id":         "gateway-installation",
		"session_id":              "gateway-session",
		"thread_id":               "gateway-thread",
		"turn_id":                 "gateway-turn",
		"window_id":               "gateway-window",
		"turn_started_at_unix_ms": int64(1700000000002),
	}
	rewritten := sanitizeCodexTurnMetadataValue(raw, fields)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal([]byte(rewritten), &metadata))
	require.Len(t, metadata, len(fields))
	for key, value := range fields {
		if key == "turn_started_at_unix_ms" {
			require.EqualValues(t, value, metadata[key], "field %s was not rebuilt", key)
			continue
		}
		require.Equal(t, value, metadata[key], "field %s was not rebuilt", key)
	}
	require.NotContains(t, metadata, "cwd")
	require.NotContains(t, metadata, "trace")
	require.NotContains(t, jsonValueStringForCodexTest(metadata), "private-secret")
}

func TestDeleteCodexPrivateHeadersRemovesClientChannels(t *testing.T) {
	headers := make(http.Header)
	for name, value := range map[string]string{
		"Accept-Language":             "zh-CN",
		"Cookie":                      "session=private",
		"Baggage":                     "sentry-trace=private",
		"Traceparent":                 "00-private",
		"Tracestate":                  "private",
		"X-Request-ID":                "private-request",
		"X-Forwarded-For":             "192.0.2.1",
		"X-Forwarded-Host":            "private.example",
		"X-Forwarded-Proto":           "http",
		"X-Real-IP":                   "192.0.2.2",
		"X-Locale":                    "zh-CN",
		"Locale":                      "zh-CN",
		"X-Stainless-Timeout":         "1",
		"X-Stainless-Read-Timeout":    "1",
		"X-Stainless-Connect-Timeout": "1",
		"X-Request-Timeout":           "1",
		"Request-Timeout":             "1",
		"Grpc-Timeout":                "1",
		liveAttestationHeader:         "private-attestation",
	} {
		headers.Set(name, value)
	}
	deleteCodexPrivateHeaders(headers)
	for _, name := range codexPrivateHeaderNames {
		require.Empty(t, headers.Get(name), "private header survived: %s", name)
	}
}

func TestFinalizeOpenAICodexRequestHeadersRebuildsStrictly(t *testing.T) {
	account := &Account{
		ID:       77,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			codexFingerprintModeExtraKey: codexFingerprintSession,
			codexFingerprintSeedExtraKey: testCodexFingerprintSeed,
			"openai_device_id":           "raw-device-id",
		},
	}
	ids := resolveCodexFingerprintIDs(account, "client-session", codexFingerprintSession)
	require.NotNil(t, ids)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://example.test/v1/responses", nil)
	c.Request.Header.Set("session-id", "client-session")
	stageCodexFingerprintIDs(c, ids)

	headers := make(http.Header)
	headers.Set("Accept", "text/private")
	headers.Set("Accept-Language", "private-locale")
	headers.Set("Cookie", "private-cookie")
	headers.Set("OpenAI-Beta", "assistants=v2")
	headers.Set("X-Codex-Beta-Features", "private-beta")
	headers.Set("X-Codex-Installation-ID", "raw-installation")
	headers.Set("X-Codex-Turn-Metadata", "{\"session_id\":\"raw-session\",\"cwd\":\"C:\\\\private\",\"trace\":\"private-trace\"}")
	headers.Set("X-Codex-Turn-State", "opaque-client-state")
	headers.Set("Traceparent", "00-private")
	headers.Set("X-Request-Timeout", "1")

	svc := &OpenAIGatewayService{}
	svc.finalizeOpenAICodexRequestHeaders(c, account, headers, false, true)
	require.Equal(t, "text/event-stream", headers.Get("Accept"))
	require.Equal(t, "application/json", headers.Get("Content-Type"))
	require.Equal(t, "responses=experimental", headers.Get("OpenAI-Beta"))
	require.Equal(t, openAIRemoteCompactionV2Feature, headers.Get("X-Codex-Beta-Features"))
	require.Equal(t, ids.installationID, headers.Get("X-Codex-Installation-ID"))
	require.Equal(t, ids.sessionID, headers.Get("Session-ID"))
	require.NotContains(t, headers.Get("X-Codex-Turn-Metadata"), "raw-session")
	require.NotContains(t, headers.Get("X-Codex-Turn-Metadata"), "private-trace")
	require.Empty(t, headers.Get("Accept-Language"))
	require.Empty(t, headers.Get("Cookie"))
	require.Empty(t, headers.Get("Traceparent"))
	require.Empty(t, headers.Get("X-Request-Timeout"))
	require.NotContains(t, headers.Get("X-Codex-Installation-ID"), "raw-device-id")
}

func TestFinalizeOpenAICodexRequestHeadersDropsTurnStateFromAnotherAccount(t *testing.T) {
	account := &Account{
		ID:       88,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			codexFingerprintModeExtraKey: codexFingerprintSession,
			codexFingerprintSeedExtraKey: testCodexFingerprintSeed,
		},
	}
	ids := resolveCodexFingerprintIDs(account, "client-session", codexFingerprintSession)
	require.NotNil(t, ids)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://example.test/v1/responses", nil)
	c.Request.Header.Set("session-id", "client-session")
	stageCodexFingerprintIDs(c, ids)
	headers := make(http.Header)
	headers.Set(openAICodexTurnStateHeader, "opaque-state-from-other-account")
	svc := &OpenAIGatewayService{}
	seed := openAICodexTurnStateSeed(c)
	svc.openaiCodexTurnStateOrigins.Store(seed, openAICodexTurnStateOrigin{
		accountID: 999,
		expiresAt: time.Now().Add(time.Minute),
	})
	svc.finalizeOpenAICodexRequestHeaders(c, account, headers, false, false)
	require.Empty(t, headers.Get(openAICodexTurnStateHeader))
}

func TestCodexFingerprintModeDefaultsToSession(t *testing.T) {
	for _, extra := range []map[string]any{
		nil,
		map[string]any{codexFingerprintModeExtraKey: ""},
		map[string]any{codexFingerprintModeExtraKey: "invalid"},
	} {
		account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: extra}
		require.Equal(t, codexFingerprintSession, account.GetCodexFingerprintMode())
	}
	account := &Account{
		ID:       2,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{codexFingerprintModeExtraKey: "off"},
	}
	require.Equal(t, codexFingerprintOff, account.GetCodexFingerprintMode())
}

func TestResolveConvergedInstallationIDIsDeploymentScopedAndNeverRaw(t *testing.T) {
	previous := codexFingerprintDeploymentNamespace()
	t.Cleanup(func() {
		codexFingerprintNamespaceState.Lock()
		codexFingerprintNamespaceState.value = previous
		codexFingerprintNamespaceState.Unlock()
	})
	account := &Account{
		ID:       3,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{"openai_device_id": "raw-device-id"},
	}
	SetCodexFingerprintDeploymentNamespace("deployment-a")
	first := resolveConvergedInstallationID(account, testCodexFingerprintSeed)
	second := resolveConvergedInstallationID(account, testCodexFingerprintSeed)
	require.Equal(t, first, second)
	require.NotEqual(t, "raw-device-id", first)
	parsed, err := uuid.Parse(first)
	require.NoError(t, err)
	require.Equal(t, uuid.Version(4), parsed.Version())
	SetCodexFingerprintDeploymentNamespace("deployment-b")
	third := resolveConvergedInstallationID(account, testCodexFingerprintSeed)
	require.NotEqual(t, first, third)
	require.NotEqual(t, "raw-device-id", third)
}

func jsonValueStringForCodexTest(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
