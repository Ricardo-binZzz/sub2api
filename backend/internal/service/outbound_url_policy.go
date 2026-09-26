package service

import (
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

// validateOutboundURL applies scheme and private-host checks independently of
// the optional hostname allowlist.
func validateOutboundURL(raw string, cfg *config.Config, allowedHosts []string) (string, error) {
	if cfg == nil {
		return "", errors.New("config is not available")
	}

	policy := cfg.Security.URLAllowlist
	opts := urlvalidator.ValidationOptions{
		AllowPrivate: policy.AllowPrivateHosts,
	}
	allowInsecureHTTP := policy.AllowInsecureHTTP
	if policy.Enabled {
		opts.AllowedHosts = allowedHosts
		opts.RequireAllowlist = true
		// Preserve strict allowlist behavior: allowlisted upstreams must use HTTPS.
		allowInsecureHTTP = false
	}

	return urlvalidator.ValidateHTTPURL(raw, allowInsecureHTTP, opts)
}
