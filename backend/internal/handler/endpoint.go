package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────────────────
// Canonical inbound / upstream endpoint paths.
// All normalization and derivation reference this single set
// of constants — add new paths HERE when a new API surface
// is introduced.
// ──────────────────────────────────────────────────────────

const (
	EndpointMessages             = "/v1/messages"
	EndpointChatCompletions      = "/v1/chat/completions"
	EndpointEmbeddings           = "/v1/embeddings"
	EndpointAlphaSearch          = "/v1/alpha/search"
	EndpointResponses            = "/v1/responses"
	EndpointResponsesCompact     = "/v1/responses/compact"
	EndpointResponsesInputTokens = "/v1/responses/input_tokens"
	EndpointImagesGenerations    = "/v1/images/generations"
	EndpointImagesEdits          = "/v1/images/edits"
	EndpointImageTasks           = "/v1/images/tasks"
	EndpointVideosGenerations    = "/v1/videos/generations"
	EndpointVideosEdits          = "/v1/videos/edits"
	EndpointVideosExtensions     = "/v1/videos/extensions"
	EndpointVideos               = "/v1/videos"
	EndpointGeminiModels         = "/v1beta/models"
)

const EndpointAntigravityGenerateContent = "/v1internal:streamGenerateContent"

// gin.Context keys used by the middleware and helpers below.
const (
	ctxKeyInboundEndpoint        = "_gateway_inbound_endpoint"
	ctxKeyActualUpstreamEndpoint = "_gateway_actual_upstream_endpoint"
)

// ──────────────────────────────────────────────────────────
// Normalization functions
// ──────────────────────────────────────────────────────────

// NormalizeInboundEndpoint maps a raw request path (which may carry
// prefixes like /antigravity, /openai) to its canonical form.
//
//	"/antigravity/v1/messages"   → "/v1/messages"
//	"/v1/chat/completions"       → "/v1/chat/completions"
//	"/openai/v1/responses/compact" → "/v1/responses/compact"
//	"/v1beta/models/gemini:gen"  → "/v1beta/models"
//
// The OpenAI Responses API is also exposed via a few bare/alias
// routes that do not carry a "/v1/" prefix (top-level bare route and
// the Codex direct route). "/responses/compact" (and "/backend-api/
// codex/responses/compact") is a distinct client endpoint — the
// "compact" client — and is normalized to its OWN canonical inbound
// endpoint, EndpointResponsesCompact, rather than being folded into
// the root Responses endpoint. Only the explicitly supported
// "/compact" and "/input_tokens" subpaths are recognized; unknown
// descendants are left untouched and rejected by the route guard.
//
//	"/v1/responses/compact"                         → EndpointResponsesCompact
//	"/openai/v1/responses/compact"                  → EndpointResponsesCompact
//	"/responses/compact"                            → EndpointResponsesCompact
//	"/backend-api/codex/responses/compact"          → EndpointResponsesCompact
//	"/v1/responses"                                 → EndpointResponses
//	"/openai/v1/responses"                          → EndpointResponses
//	"/responses"                                    → EndpointResponses
//	"/backend-api/codex/responses"                  → EndpointResponses
//
// The compact check MUST be evaluated before the root Responses check,
// otherwise "/v1/responses" (a prefix of "/v1/responses/compact")
// would erroneously match first.
func NormalizeInboundEndpoint(path string) string {
	path = strings.TrimSpace(path)
	switch {
	case isResponsesInputTokensPath(path):
		return EndpointResponsesInputTokens
	case strings.Contains(path, EndpointEmbeddings):
		return EndpointEmbeddings
	case strings.Contains(path, EndpointAlphaSearch) || isBareOrSubpathOf(strings.TrimRight(path, "/"), "/alpha/search") || isBareOrSubpathOf(strings.TrimRight(path, "/"), "/backend-api/codex/alpha/search"):
		return EndpointAlphaSearch
	case strings.Contains(path, EndpointChatCompletions):
		return EndpointChatCompletions
	case strings.Contains(path, EndpointMessages):
		return EndpointMessages
	case strings.Contains(path, EndpointImagesGenerations) || strings.Contains(path, "/images/generations"):
		return EndpointImagesGenerations
	case strings.Contains(path, EndpointImagesEdits) || strings.Contains(path, "/images/edits"):
		return EndpointImagesEdits
	case strings.Contains(path, EndpointImageTasks) || strings.Contains(path, "/images/tasks/"):
		return EndpointImageTasks
	case strings.Contains(path, EndpointVideosGenerations) || strings.Contains(path, "/videos/generations"):
		return EndpointVideosGenerations
	case strings.Contains(path, EndpointVideosEdits) || strings.Contains(path, "/videos/edits"):
		return EndpointVideosEdits
	case strings.Contains(path, EndpointVideosExtensions) || strings.Contains(path, "/videos/extensions"):
		return EndpointVideosExtensions
	case strings.Contains(path, EndpointVideos) || strings.Contains(path, "/videos/"):
		return EndpointVideos
	case isResponsesCompactPath(path):
		return EndpointResponsesCompact
	case isResponsesRootPath(path):
		return EndpointResponses
	case strings.Contains(path, EndpointGeminiModels):
		return EndpointGeminiModels
	default:
		return path
	}
}

// responsesPathRoots is the closed set of inbound route roots that expose the
// OpenAI Responses API. Keep this list explicit so an unrelated path containing
// "/responses" cannot be normalized or forwarded accidentally.
var responsesPathRoots = [...]string{
	EndpointResponses,
	"/openai/v1/responses",
	"/responses",
	"/backend-api/codex/responses",
}

// normalizedExactPath trims whitespace and at most one conventional trailing
// slash. Repeated trailing slashes are deliberately not canonicalized: they
// remain outside the exact route allowlist.
func normalizedExactPath(path string) (string, bool) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false
	}
	if strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimSuffix(trimmed, "/")
		if trimmed == "" || strings.HasSuffix(trimmed, "/") {
			return "", false
		}
	}
	return trimmed, true
}

func isExactResponsesPath(path, suffix string) bool {
	normalized, ok := normalizedExactPath(path)
	if !ok {
		return false
	}
	for _, root := range responsesPathRoots {
		if normalized == root+suffix {
			return true
		}
	}
	return false
}

func isResponsesRootPath(path string) bool {
	return isExactResponsesPath(path, "")
}

func isResponsesCompactPath(path string) bool {
	return isExactResponsesPath(path, "/compact")
}

func isResponsesInputTokensPath(path string) bool {
	return isExactResponsesPath(path, "/input_tokens")
}

func isResponsesInputTokensAliasPath(path string) bool {
	return isResponsesInputTokensPath(path)
}

// isResponsesCompactAliasPath reports whether path is one of the exact
// "compact" client endpoints. It intentionally does not accept descendants.
//
//   - "/responses/compact"                   (bare route, compact client)
//   - "/backend-api/codex/responses/compact" (Codex direct route, compact client)
//
// This MUST be checked before isResponsesRootAliasPath, since
// "/responses" is a prefix of "/responses/compact".
func isResponsesCompactAliasPath(path string) bool {
	return isResponsesCompactPath(path)
}

// isResponsesRootAliasPath reports whether path is one of the exact bare/alias
// routes that serve the root OpenAI Responses API without a "/v1/" prefix.
//
//   - "/responses"                    (top-level bare route)
//   - "/backend-api/codex/responses"  (Codex direct route)
//
// Only the two exact bare/alias roots are recognized here; this deliberately
// does NOT generalize to any path merely ending in "/responses" (e.g. an
// unrelated "/foo/responses" must not match).
func isResponsesRootAliasPath(path string) bool {
	return isResponsesRootPath(path)
}

// isBareOrSubpathOf reports whether path is exactly root, or a subpath rooted
// at root. It remains used by the non-Responses alias routes (for example
// alpha/search); Responses itself uses the stricter exact matcher above.
func isBareOrSubpathOf(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"/")
}

// DeriveUpstreamEndpoint determines the upstream endpoint from the
// account platform and the normalized inbound endpoint.
//
// Platform-specific rules:
//   - OpenAI and Grok text compatibility routes forward to /v1/responses
//     (with only the allowlisted /compact or /input_tokens suffix preserved
//     from the raw URL); native endpoints such as embeddings and alpha search
//     retain their paths. Grok raw Chat requests override this through the
//     forwarding result consumed by resolveOpenAIUpstreamEndpoint.
//   - Anthropic  → /v1/messages
//   - Gemini     → /v1beta/models
//   - Antigravity → /v1/messages (Claude) or gemini (Gemini)
//   - Antigravity routes may target either Claude or Gemini, so the
//     inbound endpoint is used to distinguish.
func DeriveUpstreamEndpoint(inbound, rawRequestPath, platform string) string {
	inbound = strings.TrimSpace(inbound)

	switch platform {
	case service.PlatformOpenAI, service.PlatformGrok:
		if inbound == EndpointEmbeddings || inbound == EndpointAlphaSearch || inbound == EndpointResponsesInputTokens || inbound == EndpointImagesGenerations || inbound == EndpointImagesEdits || inbound == EndpointVideosGenerations || inbound == EndpointVideosEdits || inbound == EndpointVideosExtensions || inbound == EndpointVideos {
			return inbound
		}
		// OpenAI forwards compatibility requests to the Responses API. Preserve
		// only the allowlisted suffixes (compact/input_tokens) derived from the
		// raw path.
		if suffix := responsesSubpathSuffix(rawRequestPath); suffix != "" {
			return EndpointResponses + suffix
		}
		// The raw path carried no derivable suffix (e.g. it was already
		// normalized upstream, or the caller only has the canonical
		// inbound endpoint available) — fall back to the canonical
		// compact endpoint when that's what the inbound request was
		// recognized as, so it isn't silently treated as the root
		// Responses endpoint.
		if inbound == EndpointResponsesCompact {
			return EndpointResponsesCompact
		}
		return EndpointResponses

	case service.PlatformAnthropic:
		return EndpointMessages

	case service.PlatformGemini:
		return EndpointGeminiModels

	case service.PlatformAntigravity:
		// Antigravity accounts serve both Claude and Gemini.
		if inbound == EndpointGeminiModels {
			return EndpointGeminiModels
		}
		return EndpointMessages
	}

	// Unknown platform — fall back to inbound.
	return inbound
}

// responsesSubpathSuffix extracts only an allowlisted part after "/responses"
// in a raw request path. Unknown descendants return "" so they cannot alter
// the upstream URL even if a caller forgets to run the route guard.
func responsesSubpathSuffix(rawPath string) string {
	trimmed, ok := normalizedExactPath(rawPath)
	if !ok {
		return ""
	}
	for _, root := range responsesPathRoots {
		if trimmed == root {
			return ""
		}
		if !strings.HasPrefix(trimmed, root+"/") {
			continue
		}
		suffix := strings.TrimPrefix(trimmed, root)
		switch suffix {
		case "/compact", "/input_tokens":
			return suffix
		default:
			return ""
		}
	}
	return ""
}

// ──────────────────────────────────────────────────────────
// Middleware
// ──────────────────────────────────────────────────────────

// InboundEndpointMiddleware normalizes the request path and stores the
// canonical inbound endpoint in gin.Context so that every handler in
// the chain can read it via GetInboundEndpoint.
//
// Apply this middleware to all gateway route groups.
func InboundEndpointMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := ""
		if c.Request != nil && c.Request.URL != nil {
			path = c.Request.URL.Path
		}
		if path == "" {
			path = c.FullPath()
		}
		c.Set(ctxKeyInboundEndpoint, NormalizeInboundEndpoint(path))
		c.Next()
	}
}

// ──────────────────────────────────────────────────────────
// Context helpers — used by handlers before building
// RecordUsageInput / RecordUsageLongContextInput.
// ──────────────────────────────────────────────────────────

// GetInboundEndpoint returns the canonical inbound endpoint stored by
// InboundEndpointMiddleware. If the middleware did not run (e.g. in
// tests), it falls back to normalizing c.Request.URL.Path on the fly
// (preferring the raw request path over c.FullPath(), which collapses
// wildcard route patterns such as "/v1/responses/*subpath" and would
// otherwise mis-normalize concrete requests like "/v1/responses/compact"
// to the root Responses endpoint).
func GetInboundEndpoint(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyInboundEndpoint); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	// Fallback: normalize on the fly.
	path := ""
	if c != nil {
		if c.Request != nil && c.Request.URL != nil {
			path = c.Request.URL.Path
		}
		if path == "" {
			path = c.FullPath()
		}
	}
	return NormalizeInboundEndpoint(path)
}

// GetUpstreamEndpoint derives the upstream endpoint from the context
// and the account platform. Handlers call this after scheduling an
// account, passing account.Platform.
func GetUpstreamEndpoint(c *gin.Context, platform string) string {
	if c != nil {
		if value, ok := c.Get(ctxKeyActualUpstreamEndpoint); ok {
			if endpoint, ok := value.(string); ok && endpoint != "" {
				return endpoint
			}
		}
	}
	inbound := GetInboundEndpoint(c)
	rawPath := ""
	if c != nil && c.Request != nil && c.Request.URL != nil {
		rawPath = c.Request.URL.Path
	}
	return DeriveUpstreamEndpoint(inbound, rawPath, platform)
}

func setActualUpstreamEndpoint(c *gin.Context, endpoint string) {
	if c != nil {
		c.Set(ctxKeyActualUpstreamEndpoint, strings.TrimSpace(endpoint))
	}
}

func shouldUseAntigravityCompat(account *service.Account) bool {
	return account != nil &&
		account.Platform == service.PlatformAntigravity &&
		account.Type == service.AccountTypeOAuth
}
