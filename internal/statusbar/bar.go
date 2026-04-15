package statusbar

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

// ANSI escape sequence constants. Exactly 8 — no others per UI-SPEC.
const (
	setScrollRegion   = "\033[%d;%dr" // DECSTBM: rows [top, bottom]
	resetScrollRegion = "\033[r"      // DECSTBM reset to full terminal
	cursorSave        = "\033[s"      // Save cursor position
	cursorRestore     = "\033[u"      // Restore cursor position
	moveCursor        = "\033[%d;1H"  // Move to row r, column 1
	eraseLineEntire   = "\033[2K"     // Erase entire current line
	reverseVideo      = "\033[7m"     // Enable reverse video
	sgrReset          = "\033[m"      // Reset all SGR attributes
)

// Position controls where the status bar is rendered.
type Position int

const (
	Bottom Position = iota // default — bar on last row, scroll region rows 1..(rows-1)
	Top                    // --status-top — bar on row 1, scroll region rows 2..rows
)

// Options configures the status bar content and placement.
type Options struct {
	SessionName  string
	AgentType    string
	Hostname     string
	CreatedAt    time.Time // session creation time for elapsed display
	Position     Position  // Bottom (default) or Top
	Fd           uintptr   // file descriptor for term.GetSize (os.Stdout.Fd())
	FallbackCols int       // fallback terminal width when GetSize fails (e.g. in tests); 0 = skip draw
	FallbackRows int       // fallback terminal height when GetSize fails; 0 = skip draw
}

// Bar renders a persistent one-row ANSI status bar using DECSTBM scroll regions.
// It is safe for concurrent use: SetViewerCount and SetConnectionState may be
// called from any goroutine. All writes to w are serialized by the caller
// (lockedWriter in cmd_attach.go).
type Bar struct {
	w    io.Writer
	opts Options

	mu          sync.Mutex
	cols        int
	rows        int
	viewerCount int
	connState   string // "" = connected (omitted), "reconnecting" = shown

	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// New creates a new status bar. The writer w should be a lockedWriter wrapping
// os.Stdout to serialize draws with PTY output. Call Start() to begin rendering.
func New(w io.Writer, opts Options) *Bar {
	return &Bar{
		w:    w,
		opts: opts,
	}
}

// sanitize strips all bytes < 0x20 (control characters including ESC) from s
// to prevent terminal injection via user-controlled session names or hostnames.
func sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// getSize returns the terminal dimensions. When term.GetSize fails (e.g. in
// non-TTY test environments), it falls back to FallbackCols/FallbackRows if
// both are non-zero. Returns ok=false when no size is available.
func (b *Bar) getSize() (cols, rows int, ok bool) {
	c, r, err := term.GetSize(int(b.opts.Fd))
	if err == nil {
		return c, r, true
	}
	if b.opts.FallbackCols > 0 && b.opts.FallbackRows > 0 {
		return b.opts.FallbackCols, b.opts.FallbackRows, true
	}
	return 0, 0, false
}

// Start sets up the DECSTBM scroll region and begins the 1-second ticker goroutine.
func (b *Bar) Start() {
	cols, rows, ok := b.getSize()
	if !ok {
		return // graceful no-op if terminal size unavailable
	}
	b.mu.Lock()
	b.cols = cols
	b.rows = rows
	b.mu.Unlock()

	if b.opts.Position == Top {
		fmt.Fprintf(b.w, setScrollRegion, 2, rows)
	} else {
		fmt.Fprintf(b.w, setScrollRegion, 1, rows-1)
	}

	// Save cursor before initial draw so we can restore on first tick.
	fmt.Fprint(b.w, cursorSave)
	b.draw()

	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.wg.Add(1)
	go b.tickLoop()
}

func (b *Bar) tickLoop() {
	defer b.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.draw()
		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Bar) draw() {
	cols, rows, ok := b.getSize()
	if !ok {
		return // skip draw cycle if terminal size unavailable
	}

	b.mu.Lock()
	if cols != b.cols || rows != b.rows {
		b.cols = cols
		b.rows = rows
		// Re-issue DECSTBM on terminal resize (self-healing).
		if b.opts.Position == Top {
			fmt.Fprintf(b.w, setScrollRegion, 2, rows)
		} else {
			fmt.Fprintf(b.w, setScrollRegion, 1, rows-1)
		}
	}
	viewerCount := b.viewerCount
	connState := b.connState
	localCols := b.cols
	localRows := b.rows
	b.mu.Unlock()

	barRow := localRows // bottom (default)
	if b.opts.Position == Top {
		barRow = 1
	}

	text := b.format(viewerCount, connState, localCols)

	fmt.Fprint(b.w, cursorSave)
	fmt.Fprintf(b.w, moveCursor, barRow)
	fmt.Fprint(b.w, eraseLineEntire)
	fmt.Fprint(b.w, text)
	fmt.Fprint(b.w, cursorRestore)
}

// format assembles the bar text for the given viewer count, connection state,
// and terminal width.
//
// All width comparisons use utf8.RuneCountInString (character/cell count), NOT len
// (byte count). The separator U+2502 is 3 bytes in UTF-8 but occupies 1 terminal
// column. With 4+ separators, len() would overcount by 8+ bytes, causing line
// wrapping instead of proper truncation.
func (b *Bar) format(viewerCount int, connState string, cols int) string {
	elapsed := time.Since(b.opts.CreatedAt)
	h := int(elapsed.Hours())
	m := int(elapsed.Minutes()) % 60
	s := int(elapsed.Seconds()) % 60

	parts := []string{
		sanitize(b.opts.SessionName),
		sanitize(b.opts.AgentType),
		sanitize(b.opts.Hostname),
		`Ctrl-\ to detach`,
	}
	if viewerCount > 1 {
		parts = append(parts, fmt.Sprintf("%d viewers", viewerCount))
	}
	if connState != "" {
		parts = append(parts, fmt.Sprintf("[%s]", connState))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%d:%02d:%02d", h, m, s))
	} else {
		parts = append(parts, fmt.Sprintf("%d:%02d", m, s))
	}

	text := " " + strings.Join(parts, " \u2502 ") + " "

	// Truncate to terminal width to prevent line wrapping.
	// Use utf8.RuneCountInString for character-cell count, not len() which
	// returns byte count. The separator U+2502 is 3 bytes but 1 column.
	runeLen := utf8.RuneCountInString(text)
	if runeLen > cols {
		if cols > 1 {
			text = string([]rune(text)[:cols-1]) + "\u2026"
		} else {
			text = "\u2026"
		}
		runeLen = utf8.RuneCountInString(text)
	}

	// Pad to full width so reverse video fills the entire row.
	if runeLen < cols {
		text = text + strings.Repeat(" ", cols-runeLen)
	}

	return reverseVideo + text + sgrReset
}

// Stop tears down the scroll region and clears the bar line. Safe to call
// multiple times (sync.Once guard). Must be called to restore terminal state.
func (b *Bar) Stop() {
	b.stopOnce.Do(func() {
		if b.cancel != nil {
			b.cancel()
		}
		b.wg.Wait()

		// Reset scroll region to full terminal.
		fmt.Fprint(b.w, resetScrollRegion)

		// Clear the bar line.
		b.mu.Lock()
		rows := b.rows
		pos := b.opts.Position
		b.mu.Unlock()

		barRow := rows
		if pos == Top {
			barRow = 1
		}
		fmt.Fprintf(b.w, moveCursor, barRow)
		fmt.Fprint(b.w, eraseLineEntire)
		fmt.Fprint(b.w, cursorRestore)
	})
}

// SetViewerCount updates the viewer count displayed on the next tick.
// Thread-safe: called from wsOutputPump goroutine.
func (b *Bar) SetViewerCount(n int) {
	b.mu.Lock()
	b.viewerCount = n
	b.mu.Unlock()
}

// SetConnectionState updates the connection state string displayed on the next tick.
// Pass "" to clear (connected state — field omitted). Pass "reconnecting" to show.
// Thread-safe: called from wsOutputPump goroutine.
func (b *Bar) SetConnectionState(state string) {
	b.mu.Lock()
	b.connState = state
	b.mu.Unlock()
}
