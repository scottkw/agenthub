package statusbar_test

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/statusbar"
)

type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// testOpts returns a base Options with fallback dimensions for non-TTY test environments.
// Tests that need specific fields override them after calling this helper.
func testOpts() statusbar.Options {
	return statusbar.Options{
		SessionName:  "test",
		AgentType:    "claude",
		Hostname:     "host",
		CreatedAt:    time.Now(),
		Position:     statusbar.Bottom,
		Fd:           os.Stdout.Fd(),
		FallbackCols: 120,
		FallbackRows: 24,
	}
}

// TestBar_FormatContainsRequiredFields verifies SB-01: all required fields appear
// in the bar output (session name, agent type, hostname, detach hint, elapsed time,
// and reverse video ANSI code).
func TestBar_FormatContainsRequiredFields(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	opts.SessionName = "my-session"
	opts.AgentType = "claude"
	opts.Hostname = "macbook-pro"
	opts.CreatedAt = time.Now().Add(-65 * time.Second) // 1:05 elapsed
	bar := statusbar.New(&buf, opts)

	// Manually trigger a draw by starting and immediately stopping.
	// The initial draw in Start() writes to buf.
	bar.Start()
	bar.Stop()

	output := buf.String()
	checks := []string{
		"my-session",
		"claude",
		"macbook-pro",
		`Ctrl-\ to detach`,
		"1:0", // elapsed time starts with 1:0x
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("bar output missing %q; got: %q", want, output)
		}
	}
	// Verify reverse video ANSI code is present.
	if !strings.Contains(output, "\033[7m") {
		t.Errorf("bar output missing reverse video escape; got: %q", output)
	}
}

// TestBar_ScrollRegionSetOnStart verifies SB-02: DECSTBM scroll region is set on Start.
func TestBar_ScrollRegionSetOnStart(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.Stop()

	output := buf.String()
	// DECSTBM should be present: ESC[1;Nr where N = rows-1.
	// We cannot know the exact terminal rows in CI, but the pattern ESC[1; must appear.
	if !strings.Contains(output, "\033[1;") {
		t.Errorf("bar output missing DECSTBM for bottom position; got: %q", output)
	}
}

// TestBar_SetViewerCountUpdates verifies SB-04: viewer count is shown when > 1.
func TestBar_SetViewerCountUpdates(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.SetViewerCount(3)

	// Wait for at least one tick to fire so the updated viewer count is drawn.
	time.Sleep(1500 * time.Millisecond)
	bar.Stop()

	output := buf.String()
	if !strings.Contains(output, "3 viewers") {
		t.Errorf("bar output missing '3 viewers'; got: %q", output)
	}
}

// TestBar_ConnStateDisplay verifies SB-05: connection state is shown when non-empty.
func TestBar_ConnStateDisplay(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.SetConnectionState("reconnecting")

	time.Sleep(1500 * time.Millisecond)
	bar.Stop()

	output := buf.String()
	if !strings.Contains(output, "[reconnecting]") {
		t.Errorf("bar output missing '[reconnecting]'; got: %q", output)
	}
}

// TestBar_TopPosition verifies SB-06: top placement uses DECSTBM starting at row 2
// and moves the cursor to row 1 for bar drawing.
func TestBar_TopPosition(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	opts.Position = statusbar.Top
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.Stop()

	output := buf.String()
	// Top position uses DECSTBM starting at row 2: ESC[2;Nr
	if !strings.Contains(output, "\033[2;") {
		t.Errorf("bar output missing DECSTBM for top position; got: %q", output)
	}
	// Bar row should be row 1: ESC[1;1H
	if !strings.Contains(output, "\033[1;1H") {
		t.Errorf("bar output missing cursor move to row 1 for top position; got: %q", output)
	}
}

// TestBar_StopClearsBarAndResetsScrollRegion verifies SB-07: Stop resets the scroll
// region and clears the bar line.
func TestBar_StopClearsBarAndResetsScrollRegion(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.Stop()

	output := buf.String()
	// Reset scroll region: ESC[r
	if !strings.Contains(output, "\033[r") {
		t.Errorf("bar output missing scroll region reset; got: %q", output)
	}
	// Erase line: ESC[2K
	if !strings.Contains(output, "\033[2K") {
		t.Errorf("bar output missing erase line in cleanup; got: %q", output)
	}
}

// TestBar_StopIdempotent verifies SB-07: Stop is safe to call multiple times (sync.Once).
func TestBar_StopIdempotent(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.Stop()
	bar.Stop() // second call must not panic
	bar.Stop() // third call must not panic
}

// TestBar_SanitizeSessionName verifies T-75-03: control characters in session names
// are stripped before rendering to prevent terminal injection.
func TestBar_SanitizeSessionName(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	opts.SessionName = "evil\033[2Jname" // contains ESC + clear screen
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.Stop()

	output := buf.String()
	// The session name should appear sanitized (no ESC[2J).
	if strings.Contains(output, "\033[2J") {
		t.Errorf("bar output contains unsanitized escape sequence ESC[2J")
	}
	// The sanitized name "evil[2Jname" should appear (only \033 stripped).
	if !strings.Contains(output, "evil") {
		t.Errorf("bar output missing sanitized session name; got: %q", output)
	}
}

// TestBar_ViewerCountHiddenWhenOne verifies SB-04: viewer count is NOT shown when <= 1.
func TestBar_ViewerCountHiddenWhenOne(t *testing.T) {
	var buf safeBuf
	opts := testOpts()
	bar := statusbar.New(&buf, opts)
	bar.Start()
	bar.SetViewerCount(1)
	time.Sleep(1500 * time.Millisecond)
	bar.Stop()

	output := buf.String()
	if strings.Contains(output, "viewers") {
		t.Errorf("bar output should not show viewer count when count=1; got: %q", output)
	}
}
