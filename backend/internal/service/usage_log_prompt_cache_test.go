package service

import "testing"

func TestPromptCacheRate(t *testing.T) {
	tests := []struct {
		name                  string
		input, creation, read int
		want                  float64
		ok                    bool
	}{
		{name: "hit and miss input", input: 300, creation: 100, read: 600, want: 0.6, ok: true},
		{name: "no cache data", input: 0, creation: 0, read: 0, want: 0, ok: false},
		{name: "cache only", input: 0, creation: 0, read: 10, want: 1, ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PromptCacheRate(int64(tt.input), int64(tt.creation), int64(tt.read))
			if ok != tt.ok || got != tt.want {
				t.Fatalf("PromptCacheRate() = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}
