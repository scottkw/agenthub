package status_test

import (
	"sync"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/status"
	"github.com/scottkw/agenthub/internal/testutil"
)

// --- mock hub for Watch tests ---

type mockHub struct {
	msgs chan []byte
	done chan struct{}
	sub  *relay.Subscriber
	mu   sync.Mutex
}

func newMockHub() *mockHub {
	return &mockHub{
		msgs: make(chan []byte, 64),
		done: make(chan struct{}),
	}
}

func (m *mockHub) Subscribe(sub *relay.Subscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sub = sub
}

func (m *mockHub) Unsubscribe(_ *relay.Subscriber) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sub = nil
}

func (m *mockHub) Done() <-chan struct{} {
	return m.done
}

func (m *mockHub) ScrollbackSnapshot() []byte {
	return nil
}

func (m *mockHub) send(b []byte) {
	m.mu.Lock()
	sub := m.sub
	m.mu.Unlock()
	if sub != nil {
		sub.Msgs <- b
	}
}

func (m *mockHub) close() {
	close(m.done)
}

// subscribed reports whether Watch has registered its subscriber yet. Tests
// must wait for this before send(), otherwise the frame is silently dropped.
func (m *mockHub) subscribed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sub != nil
}

// pending reports the number of frames queued but not yet consumed by the
// detector goroutine, or -1 if not yet subscribed.
func (m *mockHub) pending() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sub == nil {
		return -1
	}
	return len(m.sub.Msgs)
}

// --- helper: newDetector creates a Detector with Claude patterns ---

func newClaudeDetector(sessionID string, onTransit func(string, status.SessionStatus)) *status.Detector {
	return status.NewDetector(sessionID, status.DefaultClaudePatterns(), onTransit)
}

// --- helper: newAgyDetector creates a Detector with agy patterns ---

func newAgyDetector(sessionID string, onTransit func(string, status.SessionStatus)) *status.Detector {
	return status.NewDetector(sessionID, status.DefaultAgyPatterns(), onTransit)
}

// --- TestANSIStrip ---

func TestANSIStrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text passes through unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "CSI color code stripped",
			input: "\x1b[1;32mgreen bold\x1b[0m",
			want:  "green bold",
		},
		{
			name:  "prompt with ANSI wrapped",
			input: "\x1b[1;32m❯\x1b[0m ",
			want:  "❯ ",
		},
		{
			name:  "ctrl+c message with color",
			input: "\x1b[33mctrl+c to interrupt\x1b[0m",
			want:  "ctrl+c to interrupt",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := string(status.StripANSI([]byte(tc.input)))
			if got != tc.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- Detector classification tests ---

func TestDetector_ClaudeRunning(t *testing.T) {
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("Some output... ctrl+c to interrupt"))
	if got != status.StatusRunning {
		t.Errorf("expected StatusRunning, got %q", got)
	}
}

func TestDetector_ClaudeIdle(t *testing.T) {
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("Output done\n❯ "))
	if got != status.StatusIdle {
		t.Errorf("expected StatusIdle, got %q", got)
	}
}

func TestDetector_Waiting(t *testing.T) {
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("Apply changes? [y/n]"))
	if got != status.StatusWaiting {
		t.Errorf("expected StatusWaiting, got %q", got)
	}
}

func TestDetector_WaitingSelectMenu(t *testing.T) {
	// Modern Claude Code prompts for input with an interactive SELECT MENU (numbered
	// options + a footer), not a "[y/n]" string. The detector must classify this as
	// Waiting so the Hub flags the card "Needs input" and the briefing modal is reachable.
	//
	// CRITICAL: this is the REAL byte shape captured from a live `claude` session's tail
	// (#95). The TUI positions text with ANSI cursor-movement escapes, not spaces, so after
	// StripANSI the footer words are COLLAPSED, and trailing kitty-keyboard/mouse private
	// CSI sequences (\x1b[<u, \x1b[>1u, \x1b[>4;2m) that reANSI does not strip sit AFTER the
	// footer in the suffix. The pattern must still match through that trailing junk.
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("What would you like to work on in this session?\r\r" +
		"6.Chataboutthis\r\r\r" +
		"Entertoselect·↑/↓tonavigate·Esctocancel\r\r" +
		"\x1b(B\x0f\x1b[<u\x1b[>1u\x1b[>4;2m\x1b(B\x0f\x1b[<u\x1b[>1u\x1b[>4;2m"))
	if got != status.StatusWaiting {
		t.Errorf("expected StatusWaiting for a collapsed select-menu prompt, got %q", got)
	}
}

func TestDetector_WaitingSelectMenuSpacedForm(t *testing.T) {
	// The spaced form (should the TUI ever emit literal spaces) must also classify Waiting.
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("Enter to select · ↑/↓ to navigate · Esc to cancel"))
	if got != status.StatusWaiting {
		t.Errorf("expected StatusWaiting for a spaced select-menu prompt, got %q", got)
	}
}

func TestDetector_WaitingTakesPriority(t *testing.T) {
	// Waiting has higher priority than Running
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("ctrl+c to interrupt\nApply changes? [y/n]"))
	if got != status.StatusWaiting {
		t.Errorf("expected StatusWaiting (higher priority), got %q", got)
	}
}

func TestDetector_DefaultRunning(t *testing.T) {
	// Default: no pattern matched -> running
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("random text without any markers"))
	if got != status.StatusRunning {
		t.Errorf("expected StatusRunning (default), got %q", got)
	}
}

func TestDetector_TransitionCallback(t *testing.T) {
	// Callback fires only on state transitions, not every Feed call.
	var transitions []status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) {
		transitions = append(transitions, s)
	})

	// First feed: default is running; feed idle -> transition to idle
	d.Feed([]byte("Output done\n❯ "))
	// Second feed: still idle -> no transition
	d.Feed([]byte("More output\n❯ "))
	// Third feed: running -> transition to running
	d.Feed([]byte("ctrl+c to interrupt"))

	if len(transitions) != 2 {
		t.Errorf("expected 2 transitions (idle, running), got %d: %v", len(transitions), transitions)
	}
	if len(transitions) >= 1 && transitions[0] != status.StatusIdle {
		t.Errorf("first transition: expected StatusIdle, got %q", transitions[0])
	}
	if len(transitions) >= 2 && transitions[1] != status.StatusRunning {
		t.Errorf("second transition: expected StatusRunning, got %q", transitions[1])
	}
}

func TestDetector_IdleOverridesStaleWorking(t *testing.T) {
	// Real scenario: "ctrl+c to interrupt" lingers in scrollback from the
	// welcome banner, but the prompt "❯" appears at the end.  The idle prompt
	// at the tail end should win over the stale working indicator.
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("Welcome to Claude Code\nctrl+c to interrupt\n"))
	if got != status.StatusRunning {
		t.Fatalf("setup: expected StatusRunning, got %q", got)
	}
	d.Feed([]byte("some output...\n❯ "))
	if got != status.StatusIdle {
		t.Errorf("expected StatusIdle (prompt at end overrides stale working), got %q", got)
	}
}

func TestDetector_ANSIInPrompt(t *testing.T) {
	// ANSI-wrapped prompt should still be classified as idle
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("\x1b[1;32m❯\x1b[0m "))
	if got != status.StatusIdle {
		t.Errorf("expected StatusIdle for ANSI-wrapped prompt, got %q", got)
	}
}

func TestDetector_RollingTail(t *testing.T) {
	// Feed >4KB of junk then an idle prompt. Only the tail should remain.
	var got status.SessionStatus
	d := newClaudeDetector("s1", func(_ string, s status.SessionStatus) { got = s })

	junk := make([]byte, 5000)
	for i := range junk {
		junk[i] = 'x'
	}
	d.Feed(junk)
	d.Feed([]byte("❯ "))

	if got != status.StatusIdle {
		t.Errorf("expected StatusIdle after rolling tail, got %q", got)
	}
}

// --- TestWatch_IdleTransition ---

// TestWatch_IdleTransition verifies that a framed MsgOutput message containing
// the Claude Code prompt triggers a transition to StatusIdle.
func TestWatch_IdleTransition(t *testing.T) {
	hub := newMockHub()

	var (
		mu  sync.Mutex
		got status.SessionStatus
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.Watch(hub, "test-session", "claude", func(id string, s status.SessionStatus) {
			mu.Lock()
			got = s
			mu.Unlock()
		})
	}()

	// Wait for Watch to subscribe before sending; otherwise the frame is
	// dropped by the mock hub's nil-subscriber guard (issue #80).
	testutil.WaitFor(t, 2*time.Second, hub.subscribed, "Watch did not subscribe")

	// Send a framed MsgOutput containing the Claude Code idle prompt.
	prompt := relay.MakeOutputFrame([]byte("❯ "))
	hub.send(prompt)

	// Poll for the detector to report the transition instead of sleeping.
	testutil.WaitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return got == status.StatusIdle
	}, "expected StatusIdle after framed prompt")

	// Close hub so Watch exits.
	hub.close()
	wg.Wait()
}

// TestWatch_NonOutputFrameIgnored verifies that MsgResize frames are not fed
// to the detector (do not change status beyond initial seed).
func TestWatch_NonOutputFrameIgnored(t *testing.T) {
	hub := newMockHub()

	var (
		mu          sync.Mutex
		transitions int
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.Watch(hub, "test-session", "claude", func(id string, s status.SessionStatus) {
			mu.Lock()
			transitions++
			mu.Unlock()
		})
	}()

	// Wait for the subscription so the resize frame isn't dropped (issue #80).
	testutil.WaitFor(t, 2*time.Second, hub.subscribed, "Watch did not subscribe")

	// Send a resize frame — should not trigger additional transitions.
	resize := relay.MakeResizeFrame(80, 24)
	hub.send(resize)

	// Wait until the resize frame is consumed; a resize cannot produce a
	// transition, so once it's drained the count is authoritative (issue #80).
	testutil.WaitFor(t, 2*time.Second, func() bool {
		return hub.pending() == 0
	}, "resize frame was not consumed by the detector")

	hub.close()
	wg.Wait()

	mu.Lock()
	count := transitions
	mu.Unlock()

	// Only the initial transition (from empty sentinel) should fire.
	if count > 1 {
		t.Errorf("expected at most 1 transition, got %d", count)
	}
}

// TestStripANSI_OSC verifies that OSC sequences are stripped by StripANSI.
func TestStripANSI_OSC(t *testing.T) {
	input := []byte("\x1b]0;Claude Code\x07❯ ")
	got := status.StripANSI(input)
	if string(got) != "❯ " {
		t.Errorf("StripANSI OSC: expected %q, got %q", "❯ ", string(got))
	}
}

// --- agy status pattern tests ---

// TestDetector_AgyIdle verifies that an agy session's `> ` prompt classifies as idle.
func TestDetector_AgyIdle(t *testing.T) {
	var got status.SessionStatus
	d := newAgyDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("Output done\n> "))
	if got != status.StatusIdle {
		t.Errorf("expected StatusIdle for agy `> ` prompt, got %q", got)
	}
}

// TestDetector_AgyIdleNotBroadAngleBracket guards against the over-broad `>\s*$` idle
// pattern (Phase 149 WR-01): an angle bracket that is NOT a bare prompt at the start of a
// line (markup close tags, arrows, redirects) must NOT classify ordinary working output as
// idle. With the anchored `(?m)^>\s*$` pattern these all fall through to the running default.
func TestDetector_AgyIdleNotBroadAngleBracket(t *testing.T) {
	cases := []string{
		"rendering </div>\n", // HTML close tag ends in '>'
		"const f = () =>\n",  // arrow ends a line
		"<span></span>\n",    // markup ending in '>'
	}
	for _, in := range cases {
		var got status.SessionStatus
		d := newAgyDetector("s1", func(_ string, s status.SessionStatus) { got = s })
		d.Feed([]byte(in))
		if got == status.StatusIdle {
			t.Errorf("input %q must NOT classify as idle (over-broad angle-bracket match)", in)
		}
	}
}

// TestDetector_AgyWaiting verifies that an agy session's [y/n] prompt classifies as waiting.
func TestDetector_AgyWaiting(t *testing.T) {
	var got status.SessionStatus
	d := newAgyDetector("s1", func(_ string, s status.SessionStatus) { got = s })
	d.Feed([]byte("Apply changes? [y/n]"))
	if got != status.StatusWaiting {
		t.Errorf("expected StatusWaiting for agy [y/n] prompt, got %q", got)
	}
}

// TestPatternsForCLI_AgyNotFallback verifies that PatternsForCLI("agy") returns
// DefaultAgyPatterns (non-empty Idle) rather than FallbackPatterns (empty).
func TestPatternsForCLI_AgyNotFallback(t *testing.T) {
	patterns := status.PatternsForCLI("agy")
	if len(patterns.Idle) == 0 {
		t.Error("expected PatternsForCLI(\"agy\").Idle to be non-empty (not FallbackPatterns)")
	}
}

// --- TestDetectorShutdown ---

func TestDetectorShutdown(t *testing.T) {
	// Watch goroutine must exit when hub.Done() closes.
	hub := newMockHub()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.Watch(hub, "s1", "claude", func(string, status.SessionStatus) {})
	}()

	// Wait until Watch has subscribed so we know the goroutine is running
	// before we close (issue #80).
	testutil.WaitFor(t, 2*time.Second, hub.subscribed, "Watch did not subscribe")

	// Close the hub done channel.
	hub.close()

	// Watch should return promptly.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Watch goroutine did not exit within 500ms after hub.Done()")
	}
}
