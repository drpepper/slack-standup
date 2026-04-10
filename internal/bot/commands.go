package bot

import (
	"log"
	"strings"

	"github.com/drpepper/slack-standup/internal/blocks"
	"github.com/drpepper/slack-standup/internal/parse"
	"github.com/drpepper/slack-standup/internal/session"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// handleSlashCommand handles /standup in Socket Mode.
func (b *Bot) handleSlashCommand(evt *socketmode.Event, client *socketmode.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		return
	}
	client.Ack(*evt.Request)
	b.handleSlashCommandDirect(cmd)
}

// handleSlashCommandDirect handles the /standup slash command logic.
func (b *Bot) handleSlashCommandDirect(cmd slack.SlashCommand) {
	channelID := cmd.ChannelID
	userID := cmd.UserID
	raw := strings.TrimSpace(cmd.Text)
	lower := strings.ToLower(raw)

	log.Printf("/standup channel=%s user=%s text=%q", channelID, userID, raw)

	// ── next ──
	if lower == "next" {
		b.handleNext(channelID, userID, cmd.ResponseURL)
		return
	}

	// ── end ──
	if lower == "end" {
		b.handleEnd(channelID, cmd.ResponseURL)
		return
	}

	// ── status ──
	if lower == "status" {
		b.handleStatus(channelID, userID)
		return
	}

	// ── add @user ──
	if strings.HasPrefix(lower, "add ") {
		b.handleAdd(channelID, userID, raw[4:])
		return
	}

	// ── remove @user ──
	if strings.HasPrefix(lower, "remove ") {
		b.handleRemove(channelID, userID, raw[7:])
		return
	}

	// ── start ──
	if lower == "start" || strings.HasPrefix(lower, "start ") {
		startText := ""
		if len(raw) > 5 {
			startText = raw[6:]
		}
		b.handleStart(channelID, userID, startText, cmd.ResponseURL)
		return
	}

	// ── help (default) ──
	b.handleHelp(channelID, userID)
}

const helpText = `:clipboard: *Standup Commands*

• ` + "`/standup start`" + ` — Start standup with active channel members
• ` + "`/standup start @alice @bob`" + ` — Start with specific people
• ` + "`/standup next`" + ` — Advance to next speaker
• ` + "`/standup add @dave`" + ` — Add someone to the remaining order
• ` + "`/standup remove @bob`" + ` — Remove someone from the remaining order
• ` + "`/standup status`" + ` — Re-post the current order
• ` + "`/standup end`" + ` — End the standup early`

func (b *Bot) handleHelp(channelID, userID string) {
	b.api.PostEphemeral(channelID, userID,
		slack.MsgOptionText(helpText, false))
}

func (b *Bot) handleNext(channelID, userID, responseURL string) {
	var sess struct {
		exists  bool
		updated bool
		ts      string
	}

	// Check session state
	b.loop.Do(func() {
		s := b.store.Get(channelID)
		if s == nil {
			sess.exists = false
			return
		}
		sess.exists = true
		updated := b.store.Next(channelID)
		sess.updated = updated != nil
		sess.ts = b.messageTs[channelID]
	})

	if !sess.exists {
		b.api.PostEphemeral(channelID, userID,
			slack.MsgOptionText(blocks.ErrorText("No standup in progress. Start one with `/standup start`."), false))
		return
	}

	// Get current session state for rendering
	var currentSess interface{}
	done := !sess.updated
	b.loop.Do(func() {
		if sess.updated {
			currentSess = b.store.Get(channelID)
		}
	})

	if sess.ts != "" {
		updateStandupMessage(b.api, channelID, sess.ts, sessionPtr(currentSess), done)
	} else {
		ts, err := postStandupMessage(b.api, channelID, sessionPtr(currentSess), done)
		if err == nil {
			b.loop.Do(func() {
				if done {
					delete(b.messageTs, channelID)
				} else {
					b.messageTs[channelID] = ts
				}
			})
		}
	}

	if done {
		b.loop.Do(func() {
			delete(b.messageTs, channelID)
		})
	}
}

func (b *Bot) handleEnd(channelID, responseURL string) {
	log.Printf("/standup end: channel=%s", channelID)

	var ts string
	b.loop.Do(func() {
		b.store.End(channelID)
		ts = b.messageTs[channelID]
		delete(b.messageTs, channelID)
	})

	if ts != "" {
		updateStandupMessage(b.api, channelID, ts, nil, true)
	} else {
		postStandupMessage(b.api, channelID, nil, true)
	}
}

func (b *Bot) handleStatus(channelID, userID string) {
	var hasSess bool
	b.loop.Do(func() {
		hasSess = b.store.Get(channelID) != nil
	})

	if !hasSess {
		b.api.PostEphemeral(channelID, userID,
			slack.MsgOptionText(blocks.ErrorText("No standup in progress."), false))
		return
	}

	log.Printf("/standup status: channel=%s", channelID)

	// Get session and post
	var sessForRender interface{}
	b.loop.Do(func() {
		sessForRender = b.store.Get(channelID)
	})

	ts, err := postStandupMessage(b.api, channelID, sessionPtr(sessForRender), false)
	if err == nil {
		b.loop.Do(func() {
			b.messageTs[channelID] = ts
		})
	}
}

func (b *Bot) handleAdd(channelID, userID, text string) {
	var hasSess bool
	b.loop.Do(func() {
		hasSess = b.store.Get(channelID) != nil
	})

	if !hasSess {
		b.api.PostEphemeral(channelID, userID,
			slack.MsgOptionText(blocks.ErrorText("No standup in progress."), false))
		return
	}

	uids := parse.ParseParticipants(text)
	if len(uids) == 0 {
		b.api.PostEphemeral(channelID, userID,
			slack.MsgOptionText(blocks.ErrorText("Please specify a user, e.g. `/standup add @alice` or `/standup add j`."), false))
		return
	}

	log.Printf("/standup add: channel=%s users=%v", channelID, uids)

	var sessForRender interface{}
	var ts string
	b.loop.Do(func() {
		for _, uid := range uids {
			b.store.Add(channelID, uid)
		}
		sessForRender = b.store.Get(channelID)
		ts = b.messageTs[channelID]
	})

	if ts != "" {
		updateStandupMessage(b.api, channelID, ts, sessionPtr(sessForRender), false)
	} else {
		newTS, err := postStandupMessage(b.api, channelID, sessionPtr(sessForRender), false)
		if err == nil {
			b.loop.Do(func() {
				b.messageTs[channelID] = newTS
			})
		}
	}
}

func (b *Bot) handleRemove(channelID, userID, text string) {
	var hasSess bool
	b.loop.Do(func() {
		hasSess = b.store.Get(channelID) != nil
	})

	if !hasSess {
		b.api.PostEphemeral(channelID, userID,
			slack.MsgOptionText(blocks.ErrorText("No standup in progress."), false))
		return
	}

	uids := parse.ParseParticipants(text)
	if len(uids) == 0 {
		b.api.PostEphemeral(channelID, userID,
			slack.MsgOptionText(blocks.ErrorText("Please specify a user, e.g. `/standup remove @alice` or `/standup remove j`."), false))
		return
	}

	log.Printf("/standup remove: channel=%s users=%v", channelID, uids)

	var sessForRender interface{}
	var done bool
	var ts string
	b.loop.Do(func() {
		for _, uid := range uids {
			result := b.store.Remove(channelID, uid)
			if result == nil {
				done = true
				break
			}
		}
		if !done {
			sessForRender = b.store.Get(channelID)
		}
		ts = b.messageTs[channelID]
	})

	if ts != "" {
		updateStandupMessage(b.api, channelID, ts, sessionPtr(sessForRender), done)
	} else {
		newTS, err := postStandupMessage(b.api, channelID, sessionPtr(sessForRender), done)
		if err == nil && !done {
			b.loop.Do(func() {
				b.messageTs[channelID] = newTS
			})
		}
	}

	if done {
		b.loop.Do(func() {
			delete(b.messageTs, channelID)
		})
	}
}

func (b *Bot) handleStart(channelID, userID, raw, responseURL string) {
	log.Printf("/standup start: channel=%s", channelID)

	// If there's already a session, clean up the old message
	var oldTS string
	b.loop.Do(func() {
		if b.store.Get(channelID) != nil {
			oldTS = b.messageTs[channelID]
			delete(b.messageTs, channelID)
		}
	})
	if oldTS != "" {
		updateStandupMessage(b.api, channelID, oldTS, nil, true)
	}

	userIDs := parse.ParseParticipants(raw)

	if len(userIDs) == 0 {
		// No users listed — grab active channel members
		slack.PostWebhook(responseURL, &slack.WebhookMessage{
			Text:         ":hourglass: Fetching active channel members…",
			ResponseType: "ephemeral",
		})

		members, err := fetchChannelMembers(b.api, channelID)
		if err != nil {
			if err.Error() == "not_in_channel" {
				slack.PostWebhook(responseURL, &slack.WebhookMessage{
					Text:         ":wave: I'm not in this channel. Please invite me with `/invite @<bot-name>`, then run `/standup` again.",
					ResponseType: "ephemeral",
				})
				return
			}
			log.Printf("/standup handler failed: %v", err)
			slack.PostWebhook(responseURL, &slack.WebhookMessage{
				Text:         blocks.ErrorText("Something went wrong: " + err.Error()),
				ResponseType: "ephemeral",
			})
			return
		}

		userIDs = filterActiveMembers(b.api, members)
		if len(userIDs) == 0 {
			b.api.PostEphemeral(channelID, userID,
				slack.MsgOptionText(blocks.ErrorText("No active members found in this channel."), false))
			return
		}
	}

	log.Printf("/standup start: channel=%s users=%v", channelID, userIDs)

	// Create session via event loop
	b.loop.Do(func() {
		b.store.Start(channelID, userIDs)
	})

	// Get session for rendering
	var sessForRender interface{}
	b.loop.Do(func() {
		sessForRender = b.store.Get(channelID)
	})

	ts, err := postStandupMessage(b.api, channelID, sessionPtr(sessForRender), false)
	if err != nil {
		if err.Error() == "not_in_channel" {
			slack.PostWebhook(responseURL, &slack.WebhookMessage{
				Text:         ":wave: I'm not in this channel. Please invite me with `/invite @<bot-name>`, then run `/standup` again.",
				ResponseType: "ephemeral",
			})
			b.loop.Do(func() {
				b.store.End(channelID)
			})
			return
		}
		log.Printf("/standup handler failed: %v", err)
		slack.PostWebhook(responseURL, &slack.WebhookMessage{
			Text:         blocks.ErrorText("Something went wrong: " + err.Error()),
			ResponseType: "ephemeral",
		})
		return
	}

	b.loop.Do(func() {
		b.messageTs[channelID] = ts
	})
}

// sessionPtr safely converts an interface{} back to *session.Session.
func sessionPtr(v interface{}) *session.Session {
	if v == nil {
		return nil
	}
	return v.(*session.Session)
}
