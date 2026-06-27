package relay

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Phase 155-05 (PARITY-01): server-side broadcast race fix
//
// The web WSS handler must register a subscriber in the broadcast fan-out set
// IMMEDIATELY after the WebSocket Accept, BEFORE the (latency-bearing) WhoIs
// identity lookup. To make that safe, the subscriber is added with an empty
// PersonKey first (delivery-only) and its presence-roster entry is registered
// AFTER its identity fields are set, via the new two-phase RegisterPresence.
//
// These tests pin the load-bearing invariant: a subscriber that is in the
// broadcast set but has NOT yet had its identity/presence registered still
// receives broadcast frames, and presence registers correctly afterward.
// ---------------------------------------------------------------------------

// drainChatFrame reads one frame from sub.Msgs, asserts it is MsgChat, and
// returns it. Fails the test on timeout or wrong type.
func drainChatFrame(t *testing.T, sub *Subscriber, timeout time.Duration) []byte {
	t.Helper()
	select {
	case frame := <-sub.Msgs:
		if len(frame) == 0 {
			t.Fatal("drainChatFrame: received empty frame")
		}
		if frame[0] != MsgChat {
			t.Fatalf("drainChatFrame: expected MsgChat (0x%02x), got 0x%02x", MsgChat, frame[0])
		}
		return frame
	case <-time.After(timeout):
		t.Fatalf("drainChatFrame: timeout waiting for MsgChat frame (waited %v)", timeout)
		return nil
	}
}

// TestBroadcastDeliversBeforeIdentity proves the two-phase subscribe ordering:
// a subscriber added to the broadcast set BEFORE its identity/presence is
// registered (empty PersonKey, simulating the WhoIs window) still receives a
// broadcast chat frame. If delivery were gated behind identity/presence, this
// would deadlock on the drain and fail.
func TestBroadcastDeliversBeforeIdentity(t *testing.T) {
	hub := makePresenceTestHub(t)

	// Phase 1: subscribe with WhoIs-independent fields only — empty identity.
	// This mirrors handleWSSRelay subscribing right after Accept, before WhoIs.
	sub := &Subscriber{
		Msgs:      make(chan []byte, 256),
		CloseSlow: func() {},
		Origin:    "web",
		// TailnetID / PersonKey / Alias deliberately empty for now.
	}
	hub.Subscribe(sub)

	// While still "inside the WhoIs window" (no PersonKey yet), the roster must
	// be empty — Subscribe with empty PersonKey registers delivery only.
	if roster := hub.CurrentPresence(); len(roster) != 0 {
		t.Fatalf("expected empty presence roster before identity, got %d entries: %+v", len(roster), roster)
	}

	// A message broadcast in this window MUST reach the subscriber.
	hub.BroadcastChat(MakeChatFrame(ChatMessage{ID: "m1", Content: "live-during-whois"}))
	drainChatFrame(t, sub, 2*time.Second)

	// Phase 2: identity resolves — set fields, THEN register presence.
	sub.TailnetID = "nodekeyAAA"
	sub.PersonKey = "nodekeyAAA:web"
	sub.Alias = "alice"
	hub.RegisterPresence(sub)

	roster := hub.CurrentPresence()
	if len(roster) != 1 {
		t.Fatalf("expected 1 presence entry after RegisterPresence, got %d: %+v", len(roster), roster)
	}
	if roster[0].PersonKey != "nodekeyAAA:web" {
		t.Errorf("PersonKey = %q, want %q", roster[0].PersonKey, "nodekeyAAA:web")
	}
	if roster[0].TailnetID != "nodekeyAAA" {
		t.Errorf("TailnetID = %q, want %q", roster[0].TailnetID, "nodekeyAAA")
	}
	if roster[0].Alias != "alice" {
		t.Errorf("Alias = %q, want %q", roster[0].Alias, "alice")
	}
	if roster[0].Origin != "web" {
		t.Errorf("Origin = %q, want %q", roster[0].Origin, "web")
	}
	if roster[0].ConnCount != 1 {
		t.Errorf("ConnCount = %d, want 1", roster[0].ConnCount)
	}
}

// TestRegisterPresenceRefCounts verifies that RegisterPresence increments
// ConnCount for a second connection sharing the same PersonKey (mirroring the
// reference-counting Subscribe does in one shot for the relay local path).
func TestRegisterPresenceRefCounts(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub1 := &Subscriber{Msgs: make(chan []byte, 32), CloseSlow: func() {}, Origin: "web"}
	sub2 := &Subscriber{Msgs: make(chan []byte, 32), CloseSlow: func() {}, Origin: "web"}

	// Both subscribe early (delivery-only), then both register identity.
	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	sub1.TailnetID, sub1.PersonKey, sub1.Alias = "k", "k:web", "alice"
	hub.RegisterPresence(sub1)

	sub2.TailnetID, sub2.PersonKey, sub2.Alias = "k", "k:web", "alice"
	hub.RegisterPresence(sub2)

	roster := hub.CurrentPresence()
	if len(roster) != 1 {
		t.Fatalf("expected 1 collapsed presence entry, got %d: %+v", len(roster), roster)
	}
	if roster[0].ConnCount != 2 {
		t.Errorf("ConnCount = %d, want 2 (two conns share PersonKey)", roster[0].ConnCount)
	}

	// Unsubscribe symmetry: dropping one leaves the entry with ConnCount 1.
	if changed := hub.Unsubscribe(sub1); changed {
		t.Error("Unsubscribe of non-last connection should return presenceChanged=false")
	}
	roster = hub.CurrentPresence()
	if len(roster) != 1 || roster[0].ConnCount != 1 {
		t.Fatalf("after one unsub: expected 1 entry ConnCount=1, got %+v", roster)
	}
}

// TestUnsubscribeEmptyPersonKeyIsClean verifies the symmetric drop path: if a
// connection dies inside the WhoIs window (PersonKey never set), Unsubscribe
// cleanly removes it from the broadcast set and touches no roster entry.
func TestUnsubscribeEmptyPersonKeyIsClean(t *testing.T) {
	hub := makePresenceTestHub(t)

	sub := &Subscriber{Msgs: make(chan []byte, 32), CloseSlow: func() {}, Origin: "web"}
	hub.Subscribe(sub)
	if hub.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after Subscribe, got %d", hub.SubscriberCount())
	}

	changed := hub.Unsubscribe(sub)
	if changed {
		t.Error("Unsubscribe with empty PersonKey should return presenceChanged=false")
	}
	if hub.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after Unsubscribe, got %d", hub.SubscriberCount())
	}
	if roster := hub.CurrentPresence(); len(roster) != 0 {
		t.Errorf("expected empty roster, got %+v", roster)
	}
}
