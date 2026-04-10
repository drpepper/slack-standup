package bot

import (
	"log"
	"regexp"
	"strings"
)

// secretPatterns matches Slack tokens and other secrets that may appear in debug output.
var secretPatterns = regexp.MustCompile(`(xoxb-[A-Za-z0-9\-]+|xoxp-[A-Za-z0-9\-]+|xapp-[A-Za-z0-9\-]+|xoxe[.\-][A-Za-z0-9\-]+)`)

// RedactingLogger implements the slack.logger interface (Output(int, string) error)
// and redacts secrets before writing to the standard logger.
type RedactingLogger struct {
	secrets []string
}

// NewRedactingLogger creates a logger that redacts the given secret values
// as well as any Slack token patterns found in log output.
func NewRedactingLogger(secrets ...string) *RedactingLogger {
	// Filter out empty strings
	var filtered []string
	for _, s := range secrets {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return &RedactingLogger{secrets: filtered}
}

func (l *RedactingLogger) Output(calldepth int, s string) error {
	log.Print(l.redact(s))
	return nil
}

func (l *RedactingLogger) redact(s string) string {
	// Redact known secret values
	for _, secret := range l.secrets {
		s = strings.ReplaceAll(s, secret, "[REDACTED]")
	}
	// Redact any Slack token patterns
	s = secretPatterns.ReplaceAllString(s, "[REDACTED]")
	return s
}
