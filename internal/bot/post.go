package bot

import (
	"github.com/drpepper/slack-standup/internal/blocks"
	"github.com/drpepper/slack-standup/internal/session"
	"github.com/slack-go/slack"
)

// postStandupMessage posts a new standup message and returns the message timestamp.
func postStandupMessage(api *slack.Client, channelID string, sess *session.Session, done bool) (string, error) {
	blks := blocks.StandupBlocks(sess, done)
	fallbackText := "Standup order started"
	if done {
		fallbackText = "Standup complete!"
	}
	_, ts, err := api.PostMessage(channelID,
		slack.MsgOptionBlocks(blks...),
		slack.MsgOptionText(fallbackText, false),
	)
	return ts, err
}

// updateStandupMessage updates an existing standup message in place.
func updateStandupMessage(api *slack.Client, channelID, ts string, sess *session.Session, done bool) error {
	blks := blocks.StandupBlocks(sess, done)
	fallbackText := "Standup order"
	if done {
		fallbackText = "Standup complete!"
	}
	_, _, _, err := api.UpdateMessage(channelID, ts,
		slack.MsgOptionBlocks(blks...),
		slack.MsgOptionText(fallbackText, false),
	)
	return err
}
