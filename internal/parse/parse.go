package parse

import (
	"regexp"
	"strings"
)

var mentionRe = regexp.MustCompile(`<@([A-Z0-9]+)(?:\|[^>]+)?>`)

// ParseParticipants extracts Slack UIDs from <@UID> / <@UID|name> mentions,
// then collects remaining whitespace-separated tokens as plain names.
func ParseParticipants(text string) []string {
	var participants []string

	// Extract mentions and remove them from text
	withoutMentions := mentionRe.ReplaceAllStringFunc(text, func(match string) string {
		m := mentionRe.FindStringSubmatch(match)
		if m != nil {
			participants = append(participants, m[1])
		}
		return ""
	})

	// Remaining whitespace-separated tokens are plain names
	for _, token := range strings.Fields(withoutMentions) {
		if token != "" {
			participants = append(participants, token)
		}
	}

	if participants == nil {
		return []string{}
	}
	return participants
}

// ParseMentionsOnly extracts only Slack user IDs from <@...> mentions.
func ParseMentionsOnly(text string) []string {
	matches := mentionRe.FindAllStringSubmatch(text, -1)
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}
