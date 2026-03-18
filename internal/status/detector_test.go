package status_test

import (
	"sync"
	"testing"
	"time"

	"github.com/agenthub/agenthub/internal/relay"
	"github.com/agenthub/agenthub/internal/status"
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

// --- helper: newDetector creates a Detector with Claude patterns ---

func newClaudeDetector(sessionID string, onTransit func(string, status.SessionStatus)) *status.Detector {
	return status.NewDetector(sessionID, status.DefaultClaudePatterns(), onTransit)
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

	// Give Watch time to start and subscribe.
	time.Sleep(10 * time.Millisecond)

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
