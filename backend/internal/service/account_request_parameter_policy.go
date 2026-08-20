package service

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	credKeyRequestParameterPolicyEnabled = "request_parameter_policy_enabled"
	credKeyRequestParameterPolicy        = "request_parameter_policy"
	maxRequestParameterPolicyTokens      = 1_000_000
	// RequestParameterPolicyContextKey is set on the active request when an
	// account policy actually changes an outbound request body.
	RequestParameterPolicyContextKey = "request_parameter_policy_applied"
	RequestParameterPolicyFieldsKey  = "request_parameter_policy_fields"
)

// RequestParameterPolicy is intentionally small. Fields that alter routing,
// request identity, prompt content, tools, streaming, or cache isolation are
// not representable and therefore cannot be overridden by account config.
type RequestParameterPolicy struct {
	ReasoningEffort string
	MaxOutputTokens *int
	ServiceTier     string
	Store           *bool
}

func (p *RequestParameterPolicy) OverriddenFields() []string {
	if p == nil {
		return nil
	}
	fields := make([]string, 0, 4)
	if p.ReasoningEffort != "" {
		fields = append(fields, "reasoning.effort")
	}
	if p.MaxOutputTokens != nil {
		fields = append(fields, "max_output_tokens")
	}
	if p.ServiceTier != "" {
		fields = append(fields, "service_tier")
	}
	if p.Store != nil {
		fields = append(fields, "store")
	}
	sort.Strings(fields)
	return fields
}

func (a *Account) IsRequestParameterPolicyEligible() bool {
	if a == nil {
		return false
	}
	switch a.Platform {
	case PlatformOpenAI, PlatformGrok:
		return a.Type == AccountTypeAPIKey || a.Type == AccountTypeOAuth
	case PlatformKimi, PlatformZhipu, PlatformDeepseek:
		return a.Type == AccountTypeAPIKey
	default:
		return false
	}
}

func (a *Account) GetRequestParameterPolicy() *RequestParameterPolicy {
	if !a.IsRequestParameterPolicyEligible() || a.Credentials == nil {
		return nil
	}
	enabled, _ := a.Credentials[credKeyRequestParameterPolicyEnabled].(bool)
	if !enabled {
		return nil
	}
	raw, ok := anyMapping(a.Credentials[credKeyRequestParameterPolicy])
	if !ok {
		return nil
	}
	policy := &RequestParameterPolicy{}
	if value, ok := raw["reasoning_effort"].(string); ok {
		policy.ReasoningEffort = strings.TrimSpace(value)
	}
	if value, ok := integerFromAny(raw["max_output_tokens"]); ok && value > 0 && value <= maxRequestParameterPolicyTokens {
		policy.MaxOutputTokens = &value
	}
	if value, ok := raw["service_tier"].(string); ok {
		policy.ServiceTier = strings.TrimSpace(value)
	}
	if value, ok := raw["store"].(bool); ok {
		policy.Store = &value
	}
	if policy.ReasoningEffort == "" && policy.MaxOutputTokens == nil && policy.ServiceTier == "" && policy.Store == nil {
		return nil
	}
	return policy
}

// ApplyRequestParameterPolicy applies the allowlisted policy to a Responses
// JSON body. It never adds arbitrary keys and leaves invalid/non-object bodies
// unchanged so the normal request validator remains authoritative.
func (a *Account) ApplyRequestParameterPolicy(body []byte) ([]byte, bool, error) {
	policy := a.GetRequestParameterPolicy()
	if policy == nil {
		return body, false, nil
	}
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil || object == nil {
		return body, false, nil
	}
	if policy.ReasoningEffort != "" {
		reasoning, _ := object["reasoning"].(map[string]any)
		if reasoning == nil {
			reasoning = make(map[string]any)
		}
		reasoning["effort"] = policy.ReasoningEffort
		object["reasoning"] = reasoning
	}
	if policy.MaxOutputTokens != nil {
		object["max_output_tokens"] = *policy.MaxOutputTokens
	}
	if policy.ServiceTier != "" {
		object["service_tier"] = policy.ServiceTier
	}
	if policy.Store != nil {
		object["store"] = *policy.Store
	}
	updated, err := json.Marshal(object)
	if err != nil {
		return body, false, err
	}
	return updated, true, nil
}

func NormalizeRequestParameterPolicyCredentials(credentials map[string]any) error {
	if credentials == nil {
		return nil
	}
	if raw, ok := credentials[credKeyRequestParameterPolicyEnabled]; ok && raw != nil {
		if _, isBool := raw.(bool); !isBool {
			return invalidRequestParameterPolicy("request_parameter_policy_enabled must be a boolean")
		}
	}
	raw, exists := credentials[credKeyRequestParameterPolicy]
	if !exists || raw == nil {
		return nil
	}
	entries, ok := anyMapping(raw)
	if !ok {
		return invalidRequestParameterPolicy("request_parameter_policy must be an object")
	}
	allowed := map[string]struct{}{
		"reasoning_effort": {}, "max_output_tokens": {}, "service_tier": {}, "store": {},
	}
	for key := range entries {
		if _, ok := allowed[key]; !ok {
			return invalidRequestParameterPolicy("request_parameter_policy contains protected or unknown field %q", key)
		}
	}
	normalized := make(map[string]any, len(entries))
	if rawValue, ok := entries["reasoning_effort"]; ok && rawValue != nil && rawValue != "" {
		value, ok := rawValue.(string)
		if !ok || !isAllowedRequestPolicyValue(strings.TrimSpace(value), "none", "minimal", "low", "medium", "high", "xhigh") {
			return invalidRequestParameterPolicy("reasoning_effort must be one of none/minimal/low/medium/high/xhigh")
		}
		normalized["reasoning_effort"] = strings.TrimSpace(value)
	}
	if rawValue, ok := entries["max_output_tokens"]; ok && rawValue != nil && rawValue != "" {
		value, ok := integerFromAny(rawValue)
		if !ok || value < 1 || value > maxRequestParameterPolicyTokens {
			return invalidRequestParameterPolicy("max_output_tokens must be an integer between 1 and %d", maxRequestParameterPolicyTokens)
		}
		normalized["max_output_tokens"] = value
	}
	if rawValue, ok := entries["service_tier"]; ok && rawValue != nil && rawValue != "" {
		value, ok := rawValue.(string)
		if !ok || !isAllowedRequestPolicyValue(strings.TrimSpace(value), "auto", "default", "flex", "priority", "scale") {
			return invalidRequestParameterPolicy("service_tier must be one of auto/default/flex/priority/scale")
		}
		normalized["service_tier"] = strings.TrimSpace(value)
	}
	if rawValue, ok := entries["store"]; ok && rawValue != nil {
		value, ok := rawValue.(bool)
		if !ok {
			return invalidRequestParameterPolicy("store must be a boolean")
		}
		normalized["store"] = value
	}
	credentials[credKeyRequestParameterPolicy] = normalized
	return nil
}

func anyMapping(raw any) (map[string]any, bool) {
	switch value := raw.(type) {
	case map[string]any:
		return value, true
	case map[string]string:
		result := make(map[string]any, len(value))
		for key, item := range value {
			result[key] = item
		}
		return result, true
	default:
		return nil, false
	}
}

func integerFromAny(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), int64(int(value)) == value
	case float64:
		converted := int(value)
		return converted, float64(converted) == value
	case json.Number:
		parsed, err := value.Int64()
		return int(parsed), err == nil && int64(int(parsed)) == parsed
	default:
		return 0, false
	}
}

func isAllowedRequestPolicyValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func invalidRequestParameterPolicy(format string, args ...any) error {
	return infraerrors.Newf(http.StatusBadRequest, "INVALID_REQUEST_PARAMETER_POLICY", format, args...)
}
