package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiagnoseRequestParameterPolicyRejection(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{name: "service tier", msg: "service_tier=priority is not allowed for model gpt-5.5", want: "service_tier"},
		{name: "reasoning effort", msg: "reasoning.effort high is not supported", want: "reasoning effort"},
		{name: "unrelated", msg: "invalid api key", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiagnoseRequestParameterPolicyRejection(tt.msg)
			if tt.want == "" && got != "" {
				t.Fatalf("got unexpected diagnostic %q", got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Fatalf("diagnostic %q does not contain %q", got, tt.want)
			}
		})
	}
}

func TestSafeUpstreamURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"strips query", "https://api.anthropic.com/v1/messages?beta=true", "https://api.anthropic.com/v1/messages"},
		{"strips fragment", "https://api.openai.com/v1/responses#frag", "https://api.openai.com/v1/responses"},
		{"strips both", "https://host/path?token=secret#x", "https://host/path"},
		{"no query or fragment", "https://host/path", "https://host/path"},
		{"empty string", "", ""},
		{"whitespace only", "  ", ""},
		{"query before fragment", "https://h/p?a=1#f", "https://h/p"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, safeUpstreamURL(tt.input))
		})
	}
}
