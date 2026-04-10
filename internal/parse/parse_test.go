package parse

import (
	"reflect"
	"testing"
)

func TestParseParticipants(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty string", "", []string{}},
		{"plain text names", "j r", []string{"j", "r"}},
		{"single plain name", "alice", []string{"alice"}},
		{"slack mention", "<@U123ABC>", []string{"U123ABC"}},
		{"mention with display name", "<@U123ABC|alice>", []string{"U123ABC"}},
		{"mixed mentions and plain names", "<@U123ABC> j r", []string{"U123ABC", "j", "r"}},
		{"extra whitespace", "  j   r  ", []string{"j", "r"}},
		{"multiple mentions", "<@U111> <@U222>", []string{"U111", "U222"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseParticipants(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseParticipants(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMentionsOnly(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"plain text returns empty", "j r", []string{}},
		{"extracts slack user IDs", "<@U123> <@U456>", []string{"U123", "U456"}},
		{"handles mentions with display names", "<@U123|alice>", []string{"U123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMentionsOnly(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseMentionsOnly(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
