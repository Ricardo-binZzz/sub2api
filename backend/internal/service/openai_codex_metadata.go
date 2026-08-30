package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const maxCodexMetadataBytes = 64 << 10

// client_metadata is an identity transport, not an arbitrary application metadata
// bag. Rebuild it from a closed set so new client telemetry cannot silently become
// a new fingerprint channel.
var codexClientMetadataAllowedKeys = map[string]struct{}{
	"x-codex-installation-id":  {},
	"session_id":               {},
	"thread_id":                {},
	"turn_id":                  {},
	"x-codex-window-id":        {},
	openAIWSTurnMetadataHeader: {},
	responsesLiteWSMetadataKey: {},
}

var codexRequestMetadataKeys = map[string]struct{}{
	"metadata":    {},
	"cwd":         {},
	"workspace":   {},
	"git":         {},
	"os":          {},
	"arch":        {},
	"terminal":    {},
	"plugin":      {},
	"plugins":     {},
	"skill":       {},
	"skills":      {},
	"mcp":         {},
	"trace":       {},
	"traceparent": {},
	"tracestate":  {},
}

var codexTurnMetadataFieldOrder = []string{
	"installation_id",
	"session_id",
	"thread_id",
	"turn_id",
	"window_id",
	"turn_started_at_unix_ms",
}

func codexTurnMetadataFields(ids *codexFingerprintIDs) map[string]any {
	if ids == nil {
		return nil
	}
	fields := make(map[string]any, len(codexTurnMetadataFieldOrder))
	if ids.installationID != "" {
		fields["installation_id"] = ids.installationID
	}
	if ids.mode == codexFingerprintDevice {
		return fields
	}
	if ids.sessionID != "" {
		fields["session_id"] = ids.sessionID
	}
	if ids.threadID != "" {
		fields["thread_id"] = ids.threadID
	}
	if ids.turnID != "" {
		fields["turn_id"] = ids.turnID
	}
	if ids.windowID != "" {
		fields["window_id"] = ids.windowID
	}
	if ids.turnStartedAtUnixMs > 0 {
		fields["turn_started_at_unix_ms"] = ids.turnStartedAtUnixMs
	}
	return fields
}

func encodeStrictCodexTurnMetadata(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}
	strict := make(map[string]any, len(fields))
	for _, key := range codexTurnMetadataFieldOrder {
		if value, ok := fields[key]; ok {
			strict[key] = value
		}
	}
	if len(strict) == 0 {
		return ""
	}
	rebuilt, err := json.Marshal(strict)
	if err != nil {
		return ""
	}
	return string(rebuilt)
}

func sanitizeCodexTurnMetadataValue(raw string, fields map[string]any) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > maxCodexMetadataBytes {
		return encodeStrictCodexTurnMetadata(fields)
	}
	// Parse the old value only to enforce object syntax. No client-provided field is
	// retained; all outbound fields come from the gateway snapshot.
	if raw != "" {
		var existing map[string]any
		if err := json.Unmarshal([]byte(raw), &existing); err != nil || existing == nil {
			return encodeStrictCodexTurnMetadata(fields)
		}
	}
	return encodeStrictCodexTurnMetadata(fields)
}

func sanitizeCodexClientMetadataMap(metadata map[string]any, ids *codexFingerprintIDs) bool {
	if metadata == nil {
		return false
	}
	changed := false
	liteValue, litePresent := metadata[responsesLiteWSMetadataKey]
	for key := range metadata {
		if _, allowed := codexClientMetadataAllowedKeys[key]; !allowed {
			delete(metadata, key)
			changed = true
			continue
		}
		if key != responsesLiteWSMetadataKey {
			delete(metadata, key)
			changed = true
		}
	}
	if litePresent {
		validLite := false
		switch value := liteValue.(type) {
		case string:
			validLite = isOpenAIResponsesLiteHeader(value)
		case bool:
			validLite = value
		}
		if validLite {
			metadata[responsesLiteWSMetadataKey] = "true"
		} else {
			delete(metadata, responsesLiteWSMetadataKey)
			changed = true
		}
	}
	if ids == nil {
		return changed
	}
	if ids.installationID != "" {
		metadata["x-codex-installation-id"] = ids.installationID
	}
	if ids.mode != codexFingerprintDevice {
		metadata["session_id"] = ids.sessionID
		metadata["thread_id"] = ids.threadID
		metadata["turn_id"] = ids.turnID
		metadata["x-codex-window-id"] = ids.windowID
	}
	if embedded := encodeStrictCodexTurnMetadata(codexTurnMetadataFields(ids)); embedded != "" {
		metadata[openAIWSTurnMetadataHeader] = embedded
	}
	return true
}

func sanitizeCodexMetadataTree(value any, ids *codexFingerprintIDs, root bool) bool {
	changed := false
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			if _, sensitive := codexRequestMetadataKeys[lowerKey]; sensitive {
				delete(node, key)
				changed = true
				continue
			}
			if lowerKey == "client_metadata" {
				// client_metadata is valid only at the request root. Nested copies are
				// opaque extension channels and are rejected as unknown metadata.
				if !root || key != "client_metadata" {
					delete(node, key)
					changed = true
					continue
				}
				metadata, ok := child.(map[string]any)
				if !ok {
					delete(node, key)
					changed = true
					continue
				}
				if sanitizeCodexClientMetadataMap(metadata, ids) {
					changed = true
				}
				if len(metadata) == 0 {
					delete(node, key)
					changed = true
				}
				continue
			}
			if sanitizeCodexMetadataTree(child, nil, false) {
				changed = true
			}
		}
		if root && ids != nil {
			if _, exists := node["client_metadata"]; !exists {
				metadata := make(map[string]any, 7)
				sanitizeCodexClientMetadataMap(metadata, ids)
				if len(metadata) > 0 {
					node["client_metadata"] = metadata
					changed = true
				}
			}
		}
	case []any:
		for _, child := range node {
			if sanitizeCodexMetadataTree(child, nil, false) {
				changed = true
			}
		}
	}
	return changed
}

func sanitizeCodexRequestBodyMap(requestBody map[string]any, ids *codexFingerprintIDs) bool {
	if requestBody == nil {
		return false
	}
	changed := sanitizeCodexMetadataTree(requestBody, ids, true)
	if applyCodexFingerprintPromptCacheKey(requestBody, ids) {
		changed = true
	}
	return changed
}

func sanitizeCodexRequestBodyRaw(body []byte, ids *codexFingerprintIDs) ([]byte, bool, error) {
	if len(body) == 0 {
		return body, false, nil
	}
	var decoded any
	if err := decodeOpenAIJSONUseNumber(body, &decoded); err != nil {
		return body, false, fmt.Errorf("decode Codex request for metadata policy: %w", err)
	}
	requestBody, ok := decoded.(map[string]any)
	if !ok {
		return body, false, nil
	}
	if ids != nil && !ids.originalBodySessionIDCaptured {
		captureCodexFingerprintOriginalBodySessionID(ids, requestBody["client_metadata"])
	}
	changed := sanitizeCodexRequestBodyMap(requestBody, ids)
	if !changed {
		return body, false, nil
	}
	next, err := marshalOpenAIUpstreamJSON(requestBody)
	if err != nil {
		return body, false, fmt.Errorf("encode sanitized Codex request: %w", err)
	}
	return next, changed, nil
}

// sanitizeCodexDiagnosticBody removes client environment and metadata channels
// before an upstream error is persisted in logs, ops events, or failover state.
// Invalid/non-JSON responses are represented by a bounded marker instead of
// echoing arbitrary provider text that may contain the original request.
func sanitizeCodexDiagnosticBody(body []byte) []byte {
	if len(body) == 0 {
		return nil
	}
	var decoded any
	if err := decodeOpenAIJSONUseNumber(body, &decoded); err != nil {
		return []byte("<redacted non-json upstream body>")
	}
	if root, ok := decoded.(map[string]any); ok {
		if sanitizeCodexMetadataTree(root, nil, true) {
			if encoded, err := marshalOpenAIUpstreamJSON(root); err == nil {
				return encoded
			}
		}
	}
	return body
}

// sanitizeCodexWebSocketFrameRaw applies the request metadata policy to a
// Responses WebSocket frame.  response.create carries the identity projection
// in its root object; session.update carries a nested session object and must
// not receive an extra client_metadata member that the realtime protocol does
// not define.  Both levels are stripped of private metadata channels.
func sanitizeCodexWebSocketFrameRaw(body []byte, ids *codexFingerprintIDs) ([]byte, bool, error) {
	if len(body) == 0 || !gjson.ParseBytes(body).IsObject() {
		return body, false, nil
	}
	eventType := strings.TrimSpace(gjson.GetBytes(body, "type").String())
	rootIDs := ids
	if eventType != "response.create" {
		rootIDs = nil
	}
	next, changed, err := sanitizeCodexRequestBodyRaw(body, rootIDs)
	if err != nil {
		return body, false, err
	}
	// Realtime/WS session updates put the client session configuration below
	// `session`; sanitize that object independently without injecting response
	// identity fields into it.
	session := gjson.GetBytes(next, "session")
	if session.IsObject() {
		sanitizedSession, sessionChanged, sessionErr := sanitizeCodexRequestBodyRaw([]byte(session.Raw), nil)
		if sessionErr != nil {
			return body, false, sessionErr
		}
		if sessionChanged {
			rewritten, setErr := sjson.SetRawBytes(next, "session", sanitizedSession)
			if setErr != nil {
				return body, false, fmt.Errorf("write sanitized Codex websocket session: %w", setErr)
			}
			next = rewritten
			changed = true
		}
	}
	return next, changed, nil
}

// sanitizeCodexOutboundIdentityHeaders removes client-supplied identity and
// transport metadata from a Codex request before the caller applies the
// canonical account identity.  It intentionally leaves authentication and
// account routing headers untouched.  A non-nil fingerprint snapshot is then
// projected consistently into the flat headers and strict turn metadata.
func sanitizeCodexOutboundIdentityHeaders(headers http.Header, ids *codexFingerprintIDs) {
	if headers == nil {
		return
	}
	turnMetadata := strings.TrimSpace(headers.Get(openAIWSTurnMetadataHeader))
	deleteCodexPrivateHeaders(headers)
	for _, name := range []string{
		"accept", "accept-language", "openai-beta", "x-codex-beta-features",
		"x-codex-installation-id", "x-codex-window-id", "x-client-request-id",
		"session-id", "session_id", "thread-id", "thread_id", "turn-id", "turn_id",
		"x-codex-turn-state", openAICodexTurnStateHeader, openAIWSTurnMetadataHeader,
		responsesLiteHeaderKey,
	} {
		deleteOpenAIHeaderEqualFold(headers, name)
	}
	if ids == nil {
		return
	}
	applyCodexFingerprintHeaders(headers, ids)
	if strict := sanitizeCodexTurnMetadataValue(turnMetadata, codexTurnMetadataFields(ids)); strict != "" {
		headers.Set(openAIWSTurnMetadataHeader, strict)
	}
}

var codexPrivateHeaderNames = []string{
	"accept-language",
	"cookie",
	"baggage",
	"traceparent",
	"tracestate",
	"x-request-id",
	"x-forwarded-for",
	"x-forwarded-host",
	"x-forwarded-proto",
	"x-real-ip",
	"x-locale",
	"locale",
	"x-stainless-timeout",
	"x-stainless-read-timeout",
	"x-stainless-connect-timeout",
	"x-request-timeout",
	"request-timeout",
	"grpc-timeout",
	liveAttestationHeader,
}

func deleteCodexPrivateHeaders(headers http.Header) {
	if headers == nil {
		return
	}
	for _, name := range codexPrivateHeaderNames {
		deleteOpenAIHeaderEqualFold(headers, name)
	}
}

func (s *OpenAIGatewayService) finalizeOpenAICodexRequestHeaders(
	c *gin.Context,
	account *Account,
	headers http.Header,
	compact bool,
	stream bool,
) {
	if headers == nil || account == nil || !account.UsesOpenAICodexProtocol() {
		return
	}
	ids := stagedCodexFingerprintIDs(c, account)
	sessionID := strings.TrimSpace(headers.Get("session_id"))
	conversationID := strings.TrimSpace(headers.Get("conversation_id"))
	turnState := strings.TrimSpace(headers.Get(openAICodexTurnStateHeader))
	turnMetadata := strings.TrimSpace(headers.Get(openAIWSTurnMetadataHeader))
	deleteCodexPrivateHeaders(headers)
	deleteOpenAIHeaderEqualFold(headers, "accept")
	deleteOpenAIHeaderEqualFold(headers, "openai-beta")
	deleteOpenAIHeaderEqualFold(headers, "x-codex-beta-features")
	deleteOpenAIHeaderEqualFold(headers, openAICodexTurnStateHeader)
	deleteOpenAIHeaderEqualFold(headers, openAIWSTurnMetadataHeader)
	for _, name := range []string{
		"x-codex-installation-id", "x-codex-window-id", "x-client-request-id",
		"session-id", "session_id", "thread-id", "thread_id", "turn-id", "turn_id",
		"conversation-id", "conversation_id", responsesLiteHeaderKey,
	} {
		deleteOpenAIHeaderEqualFold(headers, name)
	}

	if compact {
		headers.Set("accept", "application/json")
	} else if stream {
		headers.Set("accept", "text/event-stream")
	} else {
		headers.Set("accept", "application/json")
	}
	headers.Set("content-type", "application/json")
	headers.Set("x-codex-beta-features", openAIRemoteCompactionV2Feature)
	ensureCodexIdentityHeaders(headers)
	enforceCodexIdentityHeadersWithUA(headers, s.codexIdentityOverrideUA(account))

	if ids == nil || ids.mode == codexFingerprintDevice {
		if sessionID != "" {
			headers.Set("session_id", sessionID)
		}
		if conversationID != "" {
			headers.Set("conversation_id", conversationID)
		}
	}
	if ids != nil {
		applyCodexFingerprintHeaders(headers, ids)
		turnMetadata = sanitizeCodexTurnMetadataValue(turnMetadata, codexTurnMetadataFields(ids))
		if turnMetadata != "" {
			headers.Set(openAIWSTurnMetadataHeader, turnMetadata)
		}
	}
	if turnState != "" {
		headers.Set(openAICodexTurnStateHeader, turnState)
		s.guardOpenAICodexTurnStateEcho(c, account, headers)
	}
}
