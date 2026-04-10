package blocks

import (
	"strings"
	"testing"

	"github.com/drpepper/slack-standup/internal/session"
	"github.com/slack-go/slack"
)

func sectionText(blocks []slack.Block) string {
	if len(blocks) == 0 {
		return ""
	}
	section, ok := blocks[0].(*slack.SectionBlock)
	if !ok || section.Text == nil {
		return ""
	}
	return section.Text.Text
}

func TestStandupBlocks_Done(t *testing.T) {
	blocks := StandupBlocks(nil, true)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	text := sectionText(blocks)
	if !strings.Contains(text, "Standup complete") {
		t.Errorf("expected done message, got %q", text)
	}
}

func TestStandupBlocks_SlackIDMentions(t *testing.T) {
	sess := &session.Session{Order: []string{"U123ABC", "U456DEF"}, Current: 0}
	blocks := StandupBlocks(sess, false)
	text := sectionText(blocks)
	if !strings.Contains(text, "<@U123ABC>") {
		t.Error("should mention first user")
	}
	if !strings.Contains(text, "<@U456DEF>") {
		t.Error("should mention second user")
	}
}

func TestStandupBlocks_PlainNames(t *testing.T) {
	sess := &session.Session{Order: []string{"alice", "bob"}, Current: 0}
	blocks := StandupBlocks(sess, false)
	text := sectionText(blocks)
	if !strings.Contains(text, "alice") {
		t.Error("should contain alice")
	}
	if !strings.Contains(text, "bob") {
		t.Error("should contain bob")
	}
	if strings.Contains(text, "<@alice>") {
		t.Error("plain names should not be wrapped in mentions")
	}
}

func TestStandupBlocks_CurrentSpeaker(t *testing.T) {
	sess := &session.Session{Order: []string{"a", "b"}, Current: 0}
	blocks := StandupBlocks(sess, false)
	text := sectionText(blocks)
	if !strings.Contains(text, "*a*") {
		t.Error("current speaker should be bold")
	}
	if !strings.Contains(text, "up now") {
		t.Error("should show 'up now' indicator")
	}
}

func TestStandupBlocks_CompletedSpeakers(t *testing.T) {
	sess := &session.Session{Order: []string{"a", "b", "c"}, Current: 1}
	blocks := StandupBlocks(sess, false)
	text := sectionText(blocks)
	if !strings.Contains(text, ":white_check_mark:") {
		t.Error("should show checkmark")
	}
	if !strings.Contains(text, "~a~") {
		t.Error("completed speaker should be struck through")
	}
}

func TestStandupBlocks_RemainingCount(t *testing.T) {
	sess := &session.Session{Order: []string{"a", "b", "c"}, Current: 1}
	blocks := StandupBlocks(sess, false)
	text := sectionText(blocks)
	if !strings.Contains(text, "2 remaining") {
		t.Errorf("expected '2 remaining', got %q", text)
	}
}

func TestStandupBlocks_ActionButtons(t *testing.T) {
	sess := &session.Session{Order: []string{"a"}, Current: 0}
	blocks := StandupBlocks(sess, false)
	var actions *slack.ActionBlock
	for _, b := range blocks {
		if a, ok := b.(*slack.ActionBlock); ok {
			actions = a
			break
		}
	}
	if actions == nil {
		t.Fatal("expected an actions block")
	}
	ids := make(map[string]bool)
	for _, e := range actions.Elements.ElementSet {
		if btn, ok := e.(*slack.ButtonBlockElement); ok {
			ids[btn.ActionID] = true
		}
	}
	if !ids["standup_next"] {
		t.Error("missing standup_next action")
	}
	if !ids["standup_end"] {
		t.Error("missing standup_end action")
	}
}

func TestStandupBlocks_MixedIDsAndNames(t *testing.T) {
	sess := &session.Session{Order: []string{"U123", "alice", "U456"}, Current: 0}
	blocks := StandupBlocks(sess, false)
	text := sectionText(blocks)
	if !strings.Contains(text, "<@U123>") {
		t.Error("should mention U123")
	}
	if !strings.Contains(text, "alice") {
		t.Error("should contain alice")
	}
	if strings.Contains(text, "<@alice>") {
		t.Error("alice should not be wrapped in mentions")
	}
	if !strings.Contains(text, "<@U456>") {
		t.Error("should mention U456")
	}
}

func TestErrorText(t *testing.T) {
	if ErrorText("oops") != ":x: oops" {
		t.Errorf("ErrorText('oops') = %q", ErrorText("oops"))
	}
}
