package relay

import (
	"encoding/json"
	"io"
	"testing"
	"time"
)

// makePresenceTestHub creates a Hub backed by an io.Pipe for presence/typing tests.
func makePresenceTestHub(t *testing.T) *Hub {
	t.Helper()
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close(); pr.Close() })
	return NewHub("presence-test", pr, pw, 64*1024, nil)
}

// makeTestSub creates a buffered-channel Subscriber for in-process hub tests.
func makeTestSub(personKey, tailnetID, origin, alias string) *Subscriber {
	return &Subscriber{
		Msgs:      make(chan []byte, 32),
		CloseSlow: func() {},
		PersonKey: personKey,
		TailnetID: tailnetID,
		Origin:    origin,
		Alias:     alias,
	}
}

// drainPresence reads one frame from sub.Msgs, asserts it is MsgPresence, and
// returns the decoded PresencePayload. Fails the test on timeout or wrong type.
func drainPresence(t *testing.T, sub *Subscriber) PresencePayload {
	t.Helper()
	select {
	case frame := <-sub.Msgs:
		if len(frame) == 0 {
			t.Fatal("drainPresence: received empty frame")
		}
		if frame[0] != MsgPresence {
			t.Fatalf("drainPresence: expected MsgPresence (0x%02x), got 0x%02x", MsgPresence, frame[0])
		}
		var p PresencePayload
		if err := json.Unmarshal(frame[1:], &p); err != nil {
			t.Fatalf("drainPresence: unmarshal error: %v", err)
		}
		return p
	case <-time.After(2 * time.Second):
		t.Fatal("drainPresence: timeout waiting for MsgPresence frame")
		return PresencePayload{}
	}
}

// drainTyping reads one frame from sub.Msgs, asserts it is MsgTyping, and
// returns the decoded TypingPayload. Fails the test on timeout or wrong type.
func drainTyping(t *testing.T, sub *Subscriber, timeout time.Duration) TypingPayload {
	t.Helper()
	select {
	case frame := <-sub.Msgs:
		if len(frame) == 0 {
			t.Fatal("drainTyping: received empty frame")
		}
		if frame[0] != MsgTyping {
			t.Fatalf("drainTyping: expected MsgTyping (0x%02x), got 0x%02x", MsgTyping, frame[0])
		}
		var p TypingPayload
		if err := json.Unmarshal(frame[1:], &p); err != nil {
			t.Fatalf("drainTyping: unmarshal error: %v", err)
		}
		return p
	case <-time.After(timeout):
		t.Fatalf("drainTyping: timeout waiting for MsgTyping frame (waited %v)", timeout)
		return TypingPayload{}
	}
}

// ---------------------------------------------------------------------------
// Task 1: Subscriber identity fields + reference-counted presence roster
// ---------------------------------------------------------------------------

// TestPresenceCollapse verifies D-03: two subscribers sharing a PersonKey collapse
// to a single presence entry with ConnCount == 2.
func TestPresenceCollapse(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub1 := makeTestSub("k:web", "k", "web", "alice")
	sub2 := makeTestSub("k:web", "k", "web", "alice")

	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	roster := hub.CurrentPresence()
	if len(roster) != 1 {
		t.Fatalf("expected 1 presence entry, got %d: %+v", len(roster), roster)
	}
	if roster[0].ConnCount != 2 {
		t.Errorf("expected ConnCount=2, got %d", roster[0].ConnCount)
	}
	if roster[0].PersonKey != "k:web" {
		t.Errorf("expected PersonKey=%q, got %q", "k:web", roster[0].PersonKey)
	}
}

// TestPresenceRefCount verifies that ConnCount increments on Subscribe and
// decrements on Unsubscribe, and the entry is removed only when the last
// connection drops.
func TestPresenceRefCount(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub1 := makeTestSub("k:web", "k", "web", "alice")
	sub2 := makeTestSub("k:web", "k", "web", "alice")

	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	// Unsubscribe first — entry should remain with ConnCount 1.
	changed := hub.Unsubscribe(sub1)
	if changed {
		t.Error("Unsubscribe of non-last connection should return presenceChanged=false")
	}

	roster := hub.CurrentPresence()
	if len(roster) != 1 {
		t.Fatalf("expected 1 entry after first unsub, got %d", len(roster))
	}
	if roster[0].ConnCount != 1 {
		t.Errorf("expected ConnCount=1 after first unsub, got %d", roster[0].ConnCount)
	}

	// Unsubscribe last — entry should be removed.
	changed = hub.Unsubscribe(sub2)
	if !changed {
		t.Error("Unsubscribe of last connection should return presenceChanged=true")
	}

	roster = hub.CurrentPresence()
	if len(roster) != 0 {
		t.Fatalf("expected 0 entries after last unsub, got %d", len(roster))
	}
}

// TestCompositePersonKey verifies D-04: two subscribers with different PersonKeys
// produce two distinct presence entries.
func TestCompositePersonKey(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub1 := makeTestSub("local:local", "local", "local", "ken")
	sub2 := makeTestSub("k:web", "k", "web", "ken-web")

	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	roster := hub.CurrentPresence()
	if len(roster) != 2 {
		t.Fatalf("expected 2 distinct presence entries, got %d: %+v", len(roster), roster)
	}

	keys := make(map[string]bool)
	for _, e := range roster {
		keys[e.PersonKey] = true
	}
	if !keys["local:local"] {
		t.Error("expected presence entry for local:local")
	}
	if !keys["k:web"] {
		t.Error("expected presence entry for k:web")
	}
}

// TestUnsubscribePresenceChanged verifies the bool return from Unsubscribe:
// false for non-last connection, true for last connection.
func TestUnsubscribePresenceChanged(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub1 := makeTestSub("k:web", "k", "web", "alice")
	sub2 := makeTestSub("k:web", "k", "web", "alice")

	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	if changed := hub.Unsubscribe(sub1); changed {
		t.Error("expected presenceChanged=false for first of two connections")
	}
	if changed := hub.Unsubscribe(sub2); !changed {
		t.Error("expected presenceChanged=true for last connection")
	}
}

// TestBroadcastPresence verifies that BroadcastPresence delivers a MsgPresence
// frame to every subscriber's Msgs channel.
func TestBroadcastPresence(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub := makeTestSub("local:local", "local", "local", "ken")
	hub.Subscribe(sub)

	frame := MakePresenceFrame(PresencePayload{
		Participants: []PresenceEntry{{PersonKey: "local:local", Alias: "ken", ConnCount: 1}},
	})
	hub.BroadcastPresence(frame)

	p := drainPresence(t, sub)
	if len(p.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(p.Participants))
	}
	if p.Participants[0].PersonKey != "local:local" {
		t.Errorf("expected PersonKey=%q, got %q", "local:local", p.Participants[0].PersonKey)
	}
}

// TestUpdateAlias verifies that UpdateAlias updates the roster entry so the next
// CurrentPresence reflects the new alias.
func TestUpdateAlias(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub := makeTestSub("k:web", "k", "web", "alice")
	hub.Subscribe(sub)

	hub.UpdateAlias("k:web", "sam")

	roster := hub.CurrentPresence()
	if len(roster) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(roster))
	}
	if roster[0].Alias != "sam" {
		t.Errorf("expected Alias=%q after UpdateAlias, got %q", "sam", roster[0].Alias)
	}
}

// TestEmptyPersonKeyNoPresenceEntry verifies that a Subscriber with empty
// PersonKey does not create a presence entry.
func TestEmptyPersonKeyNoPresenceEntry(t *testing.T) {
	hub := makePresenceTestHub(t)

	// Subscriber with no PersonKey — legacy subscriber, should not affect roster.
	sub := &Subscriber{
		Msgs:      make(chan []byte, 16),
		CloseSlow: func() {},
	}
	hub.Subscribe(sub)

	roster := hub.CurrentPresence()
	if len(roster) != 0 {
		t.Fatalf("expected 0 presence entries for empty-PersonKey subscriber, got %d", len(roster))
	}
}

// TestNotifyPresence verifies the NotifyPresence package-level function
// builds and broadcasts a MsgPresence frame to all subscribers.
func TestNotifyPresence(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub := makeTestSub("local:local", "local", "local", "ken")
	hub.Subscribe(sub)

	NotifyPresence(hub)

	p := drainPresence(t, sub)
	if len(p.Participants) != 1 {
		t.Fatalf("expected 1 participant from NotifyPresence, got %d", len(p.Participants))
	}
	if p.Participants[0].PersonKey != "local:local" {
		t.Errorf("unexpected PersonKey: %q", p.Participants[0].PersonKey)
	}
}

// ---------------------------------------------------------------------------
// Task 2: Typing TTL via time.AfterFunc with sender-exclusion and rate limit
// ---------------------------------------------------------------------------

// TestTypingSenderExclusion verifies that UpdateTyping broadcasts typing:true
// to all OTHER subscribers, not the sender.
func TestTypingSenderExclusion(t *testing.T) {
	hub := makePresenceTestHub(t)
	hub.typingTTL = 100 * time.Millisecond // short TTL for test

	sender := makeTestSub("local:local", "local", "local", "ken")
	receiver := makeTestSub("k:web", "k", "web", "alice")
	hub.Subscribe(sender)
	hub.Subscribe(receiver)

	hub.UpdateTyping(sender, true)

	// Receiver must get typing:true.
	p := drainTyping(t, receiver, 2*time.Second)
	if !p.Typing {
		t.Error("receiver: expected Typing=true")
	}
	if p.PersonKey != "local:local" {
		t.Errorf("receiver: expected PersonKey=%q, got %q", "local:local", p.PersonKey)
	}

	// Sender must NOT receive their own typing frame.
	select {
	case frame := <-sender.Msgs:
		// Allow MsgPresence frames (from subscribe), reject MsgTyping.
		if frame[0] == MsgTyping {
			t.Error("sender: received their own typing frame (self-echo)")
		}
	default:
		// Expected — no frame for sender.
	}
}

// TestTypingTTL verifies that after UpdateTyping(true), a typing:false frame
// is broadcast within ~100ms when typingTTL is set to 5ms.
func TestTypingTTL(t *testing.T) {
	hub := makePresenceTestHub(t)
	hub.typingTTL = 5 * time.Millisecond

	sender := makeTestSub("local:local", "local", "local", "ken")
	receiver := makeTestSub("k:web", "k", "web", "alice")
	hub.Subscribe(sender)
	hub.Subscribe(receiver)

	hub.UpdateTyping(sender, true)

	// Drain the initial typing:true frame from receiver.
	initial := drainTyping(t, receiver, 2*time.Second)
	if !initial.Typing {
		t.Error("expected initial Typing=true")
	}

	// Now wait for the TTL-fired typing:false (up to 100ms).
	auto := drainTyping(t, receiver, 100*time.Millisecond)
	if auto.Typing {
		t.Error("expected TTL-fired Typing=false")
	}
	if auto.PersonKey != "local:local" {
		t.Errorf("expected PersonKey=%q, got %q", "local:local", auto.PersonKey)
	}
}

// TestTypingTimerReset verifies that a second UpdateTyping(true) before the TTL
// resets the timer so typing stays active.
func TestTypingTimerReset(t *testing.T) {
	hub := makePresenceTestHub(t)
	hub.typingTTL = 30 * time.Millisecond

	sender := makeTestSub("local:local", "local", "local", "ken")
	receiver := makeTestSub("k:web", "k", "web", "alice")
	hub.Subscribe(sender)
	hub.Subscribe(receiver)

	hub.UpdateTyping(sender, true)
	// Drain initial broadcast.
	drainTyping(t, receiver, 2*time.Second)

	// After 15ms (< TTL), send another typing:true — should reset the timer.
	time.Sleep(15 * time.Millisecond)
	hub.UpdateTyping(sender, true)
	// Drain the re-broadcast (rate-limiting may suppress this — allow either path).
	select {
	case frame := <-receiver.Msgs:
		if frame[0] == MsgTyping {
			// OK — either a re-broadcast or timer reset with new broadcast.
		}
	default:
		// Also OK — rate limiter suppressed the re-broadcast.
	}

	// At this point the TTL should have been reset. Wait slightly less than the TTL
	// from the second call — the false broadcast should NOT arrive yet.
	time.Sleep(15 * time.Millisecond)

	select {
	case frame := <-receiver.Msgs:
		if frame[0] == MsgTyping {
			var p TypingPayload
			if err := json.Unmarshal(frame[1:], &p); err == nil && !p.Typing {
				t.Error("typing:false arrived too early — timer was not reset")
			}
		}
	default:
		// Expected — false has not arrived yet.
	}
}

// TestTypingExplicitStop verifies that UpdateTyping(false) cancels the timer
// and broadcasts typing:false immediately.
func TestTypingExplicitStop(t *testing.T) {
	hub := makePresenceTestHub(t)
	hub.typingTTL = 5 * time.Second // long TTL so it won't fire on its own

	sender := makeTestSub("local:local", "local", "local", "ken")
	receiver := makeTestSub("k:web", "k", "web", "alice")
	hub.Subscribe(sender)
	hub.Subscribe(receiver)

	hub.UpdateTyping(sender, true)
	drainTyping(t, receiver, 2*time.Second) // drain typing:true

	hub.UpdateTyping(sender, false)

	// Must receive typing:false quickly.
	p := drainTyping(t, receiver, 500*time.Millisecond)
	if p.Typing {
		t.Error("expected Typing=false after explicit stop")
	}
}

// TestUnsubscribeCancelsTypingTimer verifies that Unsubscribe of the sender's
// last connection cancels the typing timer so no subsequent TTL broadcast fires.
func TestUnsubscribeCancelsTypingTimer(t *testing.T) {
	hub := makePresenceTestHub(t)
	hub.typingTTL = 30 * time.Millisecond

	sender := makeTestSub("local:local", "local", "local", "ken")
	receiver := makeTestSub("k:web", "k", "web", "alice")
	hub.Subscribe(sender)
	hub.Subscribe(receiver)

	hub.UpdateTyping(sender, true)
	drainTyping(t, receiver, 2*time.Second) // drain typing:true

	// Unsubscribe sender before TTL fires.
	hub.Unsubscribe(sender)

	// Wait longer than the TTL — no typing:false frame should arrive.
	time.Sleep(60 * time.Millisecond)

	select {
	case frame := <-receiver.Msgs:
		if frame[0] == MsgTyping {
			t.Error("received typing frame after sender unsubscribed — timer not cancelled")
		}
	default:
		// Expected — no typing frame.
	}
}

// TestHubShutdownWithActiveTypingTimer verifies that calling Shutdown with an
// active typing timer does not panic (h.closed guard in timer callback).
func TestHubShutdownWithActiveTypingTimer(t *testing.T) {
	hub := makePresenceTestHub(t)
	hub.typingTTL = 20 * time.Millisecond

	sender := makeTestSub("local:local", "local", "local", "ken")
	receiver := makeTestSub("k:web", "k", "web", "alice")
	hub.Subscribe(sender)
	hub.Subscribe(receiver)

	hub.UpdateTyping(sender, true)
	// Don't drain — we're testing for no panic, not frame delivery.

	// Shutdown before the TTL fires.
	hub.Shutdown()

	// Wait for the timer to fire. Must not panic under -race.
	time.Sleep(60 * time.Millisecond)
	// If we reach here without panic, the test passes.
}
