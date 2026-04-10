package session

import (
	"reflect"
	"sort"
	"testing"
)

func noShuffle(a []string) {} // identity — preserves input order

func newTestStore() *Store {
	return NewStoreWithShuffle(noShuffle)
}

const ch = "C_TEST"

func TestStart(t *testing.T) {
	t.Run("creates a session with shuffled order", func(t *testing.T) {
		s := NewStore() // real shuffle
		sess := s.Start(ch, []string{"a", "b", "c"})
		if sess.ChannelID != ch {
			t.Errorf("ChannelID = %q, want %q", sess.ChannelID, ch)
		}
		if sess.Current != 0 {
			t.Errorf("Current = %d, want 0", sess.Current)
		}
		if len(sess.Order) != 3 {
			t.Errorf("Order length = %d, want 3", len(sess.Order))
		}
		sorted := make([]string, len(sess.Order))
		copy(sorted, sess.Order)
		sort.Strings(sorted)
		if !reflect.DeepEqual(sorted, []string{"a", "b", "c"}) {
			t.Errorf("sorted Order = %v, want [a b c]", sorted)
		}
	})

	t.Run("is retrievable via Get", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a"})
		sess := s.Get(ch)
		if sess == nil {
			t.Fatal("Get returned nil")
		}
		if !reflect.DeepEqual(sess.Order, []string{"a"}) {
			t.Errorf("Order = %v, want [a]", sess.Order)
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("returns nil for unknown channel", func(t *testing.T) {
		s := newTestStore()
		if s.Get("C_UNKNOWN") != nil {
			t.Error("expected nil for unknown channel")
		}
	})
}

func TestNext(t *testing.T) {
	t.Run("advances the current index", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a", "b", "c"})
		sess := s.Next(ch)
		if sess == nil {
			t.Fatal("Next returned nil")
		}
		if sess.Current != 1 {
			t.Errorf("Current = %d, want 1", sess.Current)
		}
	})

	t.Run("returns nil when standup is over", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a"})
		result := s.Next(ch)
		if result != nil {
			t.Error("expected nil when standup is over")
		}
		if s.Get(ch) != nil {
			t.Error("session should be deleted")
		}
	})

	t.Run("returns nil for unknown channel", func(t *testing.T) {
		s := newTestStore()
		if s.Next("C_UNKNOWN") != nil {
			t.Error("expected nil for unknown channel")
		}
	})
}

func TestAdd(t *testing.T) {
	t.Run("inserts user after current speaker", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a", "b"})
		s.Add(ch, "c")
		sess := s.Get(ch)
		if !reflect.DeepEqual(sess.Order, []string{"a", "c", "b"}) {
			t.Errorf("Order = %v, want [a c b]", sess.Order)
		}
	})

	t.Run("is a no-op if user already in remaining order", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a", "b"})
		s.Add(ch, "b")
		sess := s.Get(ch)
		if !reflect.DeepEqual(sess.Order, []string{"a", "b"}) {
			t.Errorf("Order = %v, want [a b]", sess.Order)
		}
	})

	t.Run("returns nil for unknown channel", func(t *testing.T) {
		s := newTestStore()
		if s.Add("C_UNKNOWN", "x") != nil {
			t.Error("expected nil for unknown channel")
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("removes user from remaining order", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a", "b", "c"})
		s.Remove(ch, "b")
		sess := s.Get(ch)
		if !reflect.DeepEqual(sess.Order, []string{"a", "c"}) {
			t.Errorf("Order = %v, want [a c]", sess.Order)
		}
	})

	t.Run("returns nil when removing the last remaining user", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a"})
		result := s.Remove(ch, "a")
		if result != nil {
			t.Error("expected nil when last user removed")
		}
		if s.Get(ch) != nil {
			t.Error("session should be deleted")
		}
	})

	t.Run("is a no-op if user not found", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a", "b"})
		s.Remove(ch, "z")
		sess := s.Get(ch)
		if !reflect.DeepEqual(sess.Order, []string{"a", "b"}) {
			t.Errorf("Order = %v, want [a b]", sess.Order)
		}
	})

	t.Run("does not remove already-spoken users", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a", "b", "c"})
		s.Next(ch) // current = 1 (b is speaking)
		s.Remove(ch, "a") // a already spoke — should be no-op
		sess := s.Get(ch)
		if !reflect.DeepEqual(sess.Order, []string{"a", "b", "c"}) {
			t.Errorf("Order = %v, want [a b c]", sess.Order)
		}
	})
}

func TestEnd(t *testing.T) {
	t.Run("removes the session", func(t *testing.T) {
		s := newTestStore()
		s.Start(ch, []string{"a"})
		s.End(ch)
		if s.Get(ch) != nil {
			t.Error("session should be deleted")
		}
	})

	t.Run("is a no-op for unknown channel", func(t *testing.T) {
		s := newTestStore()
		s.End("C_UNKNOWN") // should not panic
	})
}
