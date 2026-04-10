package blocks

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/drpepper/slack-standup/internal/session"
	"github.com/slack-go/slack"
)

var slackIDRe = regexp.MustCompile(`^[A-Z][A-Z0-9]+$`)

func fmtUser(id string) string {
	if slackIDRe.MatchString(id) {
		return "<@" + id + ">"
	}
	return id
}

// StandupBlocks builds the Block Kit message for a standup session.
// If done is true, returns a "standup complete" message.
func StandupBlocks(sess *session.Session, done bool) []slack.Block {
	if done {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, ":tada: *Standup complete!* Everyone has spoken.", false, false),
				nil, nil,
			),
		}
	}

	remaining := len(sess.Order) - sess.Current
	var lines []string
	for i, id := range sess.Order {
		switch {
		case i < sess.Current:
			lines = append(lines, fmt.Sprintf(":white_check_mark: ~%s~", fmtUser(id)))
		case i == sess.Current:
			lines = append(lines, fmt.Sprintf(":speaking_head_in_silhouette: *%s* ← up now", fmtUser(id)))
		default:
			lines = append(lines, fmt.Sprintf("%d. %s", i+1-sess.Current, fmtUser(id)))
		}
	}

	text := fmt.Sprintf("*Standup order* (%d remaining)\n\n%s", remaining, strings.Join(lines, "\n"))

	nextBtn := slack.NewButtonBlockElement("standup_next", "", slack.NewTextBlockObject(slack.PlainTextType, "Next ▶", false, false))
	nextBtn.WithStyle(slack.StylePrimary)

	endBtn := slack.NewButtonBlockElement("standup_end", "", slack.NewTextBlockObject(slack.PlainTextType, "End standup", false, false))
	endBtn.WithStyle(slack.StyleDanger)
	endBtn.WithConfirm(slack.NewConfirmationBlockObject(
		slack.NewTextBlockObject(slack.PlainTextType, "End standup?", false, false),
		slack.NewTextBlockObject(slack.MarkdownType, "This will cancel the current standup.", false, false),
		slack.NewTextBlockObject(slack.PlainTextType, "Yes, end it", false, false),
		slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
	))

	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
			nil, nil,
		),
		slack.NewDividerBlock(),
		slack.NewActionBlock("standup_actions", nextBtn, endBtn),
	}
}

// ErrorText returns an error message prefixed with :x: emoji.
func ErrorText(msg string) string {
	return ":x: " + msg
}
