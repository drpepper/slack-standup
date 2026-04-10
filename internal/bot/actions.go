package bot

import (
	"log"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// handleNextAction handles the standup_next button in Socket Mode.
func (b *Bot) handleNextAction(evt *socketmode.Event, client *socketmode.Client) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		return
	}
	client.Ack(*evt.Request)
	b.handleNextDirect(callback.Channel.ID, callback.Message.Timestamp, callback.User.ID)
}

// handleEndAction handles the standup_end button in Socket Mode.
func (b *Bot) handleEndAction(evt *socketmode.Event, client *socketmode.Client) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		return
	}
	client.Ack(*evt.Request)
	b.handleEndDirect(callback.Channel.ID, callback.Message.Timestamp, callback.User.ID)
}

// handleNextDirect advances to the next speaker and updates the message.
func (b *Bot) handleNextDirect(channelID, ts, userID string) {
	log.Printf("action standup_next: channel=%s user=%s", channelID, userID)

	var hasSess bool
	var done bool
	b.loop.Do(func() {
		s := b.store.Get(channelID)
		if s == nil {
			hasSess = false
			return
		}
		hasSess = true
		updated := b.store.Next(channelID)
		done = updated == nil
	})

	if !hasSess {
		updateStandupMessage(b.api, channelID, ts, nil, true)
		return
	}

	var sessForRender interface{}
	b.loop.Do(func() {
		if !done {
			sessForRender = b.store.Get(channelID)
		}
	})

	updateStandupMessage(b.api, channelID, ts, sessionPtr(sessForRender), done)

	if done {
		b.loop.Do(func() {
			delete(b.messageTs, channelID)
		})
	}
}

// handleEndDirect ends the standup and updates the message to done state.
func (b *Bot) handleEndDirect(channelID, ts, userID string) {
	log.Printf("action standup_end: channel=%s user=%s", channelID, userID)

	b.loop.Do(func() {
		b.store.End(channelID)
		delete(b.messageTs, channelID)
	})

	updateStandupMessage(b.api, channelID, ts, nil, true)
}
