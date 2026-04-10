package bot

import (
	"log"
	"sync"

	"github.com/slack-go/slack"
)

// fetchChannelMembers returns all human, non-bot members of a channel.
func fetchChannelMembers(api *slack.Client, channelID string) ([]string, error) {
	var allMembers []string
	cursor := ""
	for {
		params := &slack.GetUsersInConversationParameters{
			ChannelID: channelID,
			Cursor:    cursor,
			Limit:     200,
		}
		members, nextCursor, err := api.GetUsersInConversation(params)
		if err != nil {
			return nil, err
		}
		allMembers = append(allMembers, members...)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	log.Printf("fetchChannelMembers: %d raw members in %s", len(allMembers), channelID)

	// Filter out bots and deleted users
	var human []string
	for _, uid := range allMembers {
		info, err := api.GetUserInfo(uid)
		if err != nil {
			continue
		}
		if !info.IsBot && !info.Deleted {
			human = append(human, uid)
		}
	}

	log.Printf("fetchChannelMembers: %d human members after filtering", len(human))
	return human, nil
}

// filterActiveMembers returns only active (presence=active) members.
// Falls back to all members if none are active or presence checks fail.
func filterActiveMembers(api *slack.Client, userIDs []string) []string {
	type result struct {
		uid    string
		active bool
	}

	results := make([]result, len(userIDs))
	var wg sync.WaitGroup
	for i, uid := range userIDs {
		wg.Add(1)
		go func(i int, uid string) {
			defer wg.Done()
			p, err := api.GetUserPresence(uid)
			results[i] = result{uid: uid, active: err == nil && p.Presence == "active"}
		}(i, uid)
	}
	wg.Wait()

	var active []string
	for _, r := range results {
		if r.active {
			active = append(active, r.uid)
		}
	}

	log.Printf("filterActiveMembers: %d/%d active", len(active), len(userIDs))
	if len(active) > 0 {
		return active
	}
	return userIDs
}
