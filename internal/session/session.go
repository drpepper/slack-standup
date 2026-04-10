package session

import (
	"math/rand/v2"
)

type Session struct {
	ChannelID string
	Order     []string
	Current   int
}

// ShuffleFunc defines a function that shuffles a slice of strings in place.
type ShuffleFunc func([]string)

// Store manages standup sessions keyed by channel ID.
// It is NOT concurrent-safe — callers must serialize access (e.g. via an event loop).
type Store struct {
	sessions map[string]*Session
	shuffle  ShuffleFunc
}

func defaultShuffle(a []string) {
	for i := len(a) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

// NewStore creates a new session store with the default Fisher-Yates shuffle.
func NewStore() *Store {
	return &Store{sessions: make(map[string]*Session), shuffle: defaultShuffle}
}

// NewStoreWithShuffle creates a store with an injectable shuffle for testing.
func NewStoreWithShuffle(fn ShuffleFunc) *Store {
	return &Store{sessions: make(map[string]*Session), shuffle: fn}
}

// Start creates a new standup session, shuffling the user IDs.
func (s *Store) Start(channelID string, userIDs []string) *Session {
	order := make([]string, len(userIDs))
	copy(order, userIDs)
	s.shuffle(order)
	sess := &Session{ChannelID: channelID, Order: order, Current: 0}
	s.sessions[channelID] = sess
	return sess
}

// Get returns the session for a channel, or nil if none exists.
func (s *Store) Get(channelID string) *Session {
	sess, ok := s.sessions[channelID]
	if !ok {
		return nil
	}
	return sess
}

// Next advances to the next speaker. Returns nil if the standup is over.
func (s *Store) Next(channelID string) *Session {
	sess, ok := s.sessions[channelID]
	if !ok {
		return nil
	}
	sess.Current++
	if sess.Current >= len(sess.Order) {
		delete(s.sessions, channelID)
		return nil
	}
	return sess
}

// Add inserts a user right after the current speaker. No-op if already in remaining order.
func (s *Store) Add(channelID, userID string) *Session {
	sess, ok := s.sessions[channelID]
	if !ok {
		return nil
	}
	// Check if already in remaining order
	for i := sess.Current; i < len(sess.Order); i++ {
		if sess.Order[i] == userID {
			return sess
		}
	}
	// Insert right after current position
	pos := sess.Current + 1
	sess.Order = append(sess.Order, "")
	copy(sess.Order[pos+1:], sess.Order[pos:])
	sess.Order[pos] = userID
	return sess
}

// Remove removes a user from current position onward. Returns nil if standup ends.
func (s *Store) Remove(channelID, userID string) *Session {
	sess, ok := s.sessions[channelID]
	if !ok {
		return nil
	}
	// Only remove from current or later
	idx := -1
	for i := sess.Current; i < len(sess.Order); i++ {
		if sess.Order[i] == userID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return sess
	}
	sess.Order = append(sess.Order[:idx], sess.Order[idx+1:]...)
	if sess.Current >= len(sess.Order) {
		delete(s.sessions, channelID)
		return nil
	}
	return sess
}

// End removes the session for a channel.
func (s *Store) End(channelID string) {
	delete(s.sessions, channelID)
}
