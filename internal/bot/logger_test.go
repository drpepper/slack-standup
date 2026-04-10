package bot

import (
	"testing"
)

func TestRedactingLogger_Redact(t *testing.T) {
	logger := NewRedactingLogger("my-secret-value", "another-secret")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"redacts known secret",
			"using token my-secret-value here",
			"using token [REDACTED] here",
		},
		{
			"redacts xoxb token pattern",
			"token=xoxb-123-456-abc",
			"token=[REDACTED]",
		},
		{
			"redacts xapp token pattern",
			"app token xapp-1-A02-xyz",
			"app token [REDACTED]",
		},
		{
			"redacts multiple secrets in one line",
			"bot=my-secret-value app=another-secret",
			"bot=[REDACTED] app=[REDACTED]",
		},
		{
			"leaves clean text unchanged",
			"no secrets here",
			"no secrets here",
		},
		{
			"handles empty secrets list",
			"xoxb-token-here should still be redacted",
			"[REDACTED] should still be redacted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logger.redact(tt.input)
			if got != tt.want {
				t.Errorf("redact(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
