package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestValidateOutboundURLWithoutHostnameAllowlist(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = false
	cfg.Security.URLAllowlist.AllowPrivateHosts = false
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}

	normalized, err := validateOutboundURL(
		"https://new-public-upstream.example/v1/",
		cfg,
		cfg.Security.URLAllowlist.UpstreamHosts,
	)
	require.NoError(t, err)
	require.Equal(t, "https://new-public-upstream.example/v1", normalized)

	for _, raw := range []string{
		"http://new-public-upstream.example",
		"https://localhost",
		"https://127.0.0.1",
		"https://10.0.0.1",
		"https://169.254.169.254",
		"https://[::1]",
	} {
		_, err := validateOutboundURL(raw, cfg, cfg.Security.URLAllowlist.UpstreamHosts)
		require.Error(t, err, raw)
	}
}

func TestValidateOutboundURLWithHostnameAllowlist(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.AllowPrivateHosts = false
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.example.com"}

	_, err := validateOutboundURL("https://api.example.com", cfg, cfg.Security.URLAllowlist.UpstreamHosts)
	require.NoError(t, err)

	_, err = validateOutboundURL("https://other.example.com", cfg, cfg.Security.URLAllowlist.UpstreamHosts)
	require.Error(t, err)

	_, err = validateOutboundURL("http://api.example.com", cfg, cfg.Security.URLAllowlist.UpstreamHosts)
	require.Error(t, err)
}
