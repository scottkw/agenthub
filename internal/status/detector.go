// Package status implements a heuristic PTY output classifier for AI coding CLIs.
// It maintains a per-session rolling tail of stripped text and emits status transitions
// (running / waiting / idle / errored) to a caller-supplied callback.
package status

import (
	"regexp"
	"sync"

	"github.com/scottkw/agenthub/internal/relay"
)

// SessionStatus is an opaque string representing the current state of a CLI session.
type SessionStatus string

const (
	// StatusRunning indicates the CLI is actively processing (producing output).
	StatusRunning SessionStatus = "running"
	// StatusWaiting indicates the CLI is waiting for user confirmation (y/n prompt).
	StatusWaiting SessionStatus = "waiting"
	// StatusIdle indicates the CLI has presented its prompt and is idle.
	StatusIdle SessionStatus = "idle"
	// StatusErrored indicates the session has exited with a non-zero status or
	// the hub has closed unexpectedly.
	StatusErrored SessionStatus = "errored"
)

// reANSI matches CSI escape sequences, a subset of other common ANSI codes,
// and OSC sequences (\x1b]...\x07 or \x1b]...\x1b\) that Claude Code emits
// for window title updates before the prompt.
// Compiled once at init; safe for concurrent use.
var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b[()][AB012]|\x1b[=>]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)

// StripANSI removes ANSI/VT escape sequences from b and returns the cleaned bytes.
// The original slice is not modified.
func StripANSI(b []byte) []byte {
	return reANSI.ReplaceAll(b, nil)
}

// appendTail appends new bytes to existing, trimming from the front if the
// combined length exceeds maxLen.  The returned slice shares no backing array
// with the inputs.
func appendTail(existing, new []byte, maxLen int) []byte {
	combined := append(existing, new...)
	if len(combined) > maxLen {
		combined = combined[len(combined)-maxLen:]
	}
	// Return a copy so callers can't accidentally mutate the backing array.
	out := make([]byte, len(combined))
	copy(out, combined)
	return out
}

// PatternSet holds compiled regexps for a single CLI's status detection.
// Priority order: Waiting > Working (Running) > Idle > default Running.
type PatternSet struct {
	Working []*regexp.Regexp // output patterns that indicate the CLI is running
	Idle    []*regexp.Regexp // output patterns that indicate the CLI is idle
	Waiting []*regexp.Regexp // output patterns that indicate the CLI is waiting for input
}

// DefaultClaudePatterns returns the empirically-known patterns for Claude Code.
func DefaultClaudePatterns() PatternSet {
	return PatternSet{
		Working: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ctrl\+c to interrupt`),
		},
		Idle: []*regexp.Regexp{
			regexp.MustCompile(`❯\s*$`),
		},
		Waiting: []*regexp.Regexp{
			regexp.MustCompile(`\[y/n\]|\[Y/n\]|\[y/N\]`),
		},
	}
}

// FallbackPatterns returns an empty PatternSet that produces a conservative
// "running" default for unknown CLIs.
func FallbackPatterns() PatternSet {
	return PatternSet{}
}

// PatternsForCLI returns DefaultClaudePatterns for the "claude" CLI name and
// FallbackPatterns for everything else.
func PatternsForCLI(cliName string) PatternSet {
	if cliName == "claude" {
		return DefaultClaudePatterns()
	}
	return FallbackPatterns()
}

// Detector maintains a rolling tail of stripped PTY output for a single session
// and classifies its current state.
type Detector struct {
	sessionID string
	patterns  PatternSet
	tail      []byte
	current   SessionStatus
	onTransit func(string, SessionStatus)
}

const maxTailBytes = 4096

// NewDetector creates a Detector for the given session.
// onTransit is called (from the same goroutine as Feed) whenever the classified
// status changes.  It is also called on the very first Feed to emit the initial status.
func NewDetector(sessionID string, patterns PatternSet, onTransit func(string, SessionStatus)) *Detector {
	return &Detector{
		sessionID: sessionID,
		patterns:  patterns,
		current:   "", // empty sentinel so the first Feed always fires onTransit
		onTransit: onTransit,
	}
}

// Feed processes a raw PTY output chunk: strips ANSI sequences, appends to the
// rolling tail, reclassifies, and fires onTransit if the status changed.
func (d *Detector) Feed(raw []byte) {
	stripped := StripANSI(raw)
	d.tail = appendTail(d.tail, stripped, maxTailBytes)
	next := d.classify()
	if next != d.current {
		d.current = next
		if d.onTransit != nil {
			d.onTransit(d.sessionID, next)
		}
	}
}

// tailSuffix is the number of bytes from the end of the tail used for
// recency-sensitive pattern matching (idle, waiting).  Patterns that indicate
// "what the CLI is doing right now" should match near the end of the output,
// not deep in scrollback history.
const tailSuffix = 256

// classify determines the current status from the tail contents.
// Recency-aware priority: patterns that match the last tailSuffix bytes of
// output take precedence, because they reflect the CLI's current state rather
// than stale output still sitting in the scrollback buffer.
//
// Order: Waiting (end) > Idle (end) > Working (full tail) > default Running.
func (d *Detector) classify() SessionStatus {
	tail := string(d.tail)

	// Extract the suffix for recency-sensitive checks.
	suffix := tail
	if len(suffix) > tailSuffix {
		suffix = suffix[len(suffix)-tailSuffix:]
	}

	// Highest priority: waiting for user confirmation (checked against suffix).
	for _, re := range d.patterns.Waiting {
		if re.MatchString(suffix) {
			return StatusWaiting
		}
	}

	// Second: idle prompt visible at end of output — CLI is done and waiting
	// for a new command.  This must beat Working because "ctrl+c to interrupt"
	// lingers in scrollback long after the CLI has returned to idle.
	for _, re := range d.patterns.Idle {
		if re.MatchString(suffix) {
			return StatusIdle
		}
	}

	// Third: actively working (checked against full tail — working indicators
	// like spinner frames may appear anywhere in recent output).
	for _, re := range d.patterns.Working {
		if re.MatchString(tail) {
			return StatusRunning
		}
	}

	// Conservative default: assume running.
	return StatusRunning
}

// HubLike is the interface required by Watch. *relay.Hub satisfies this interface.
type HubLike interface {
	Subscribe(sub *relay.Subscriber)
	Unsubscribe(sub *relay.Subscriber)
	Done() <-chan struct{}
	ScrollbackSnapshot() []byte
}

// Watch subscribes to hub, feeds all incoming frames through a Detector for
// the given session, and calls onTransit on each status transition.
// Watch blocks until hub.Done() is closed, then unsubscribes and returns.
// It is designed to be run as a goroutine.
func Watch(hub HubLike, sessionID, cli string, onTransit func(string, SessionStatus)) {
	patterns := PatternsForCLI(cli)
	detector := NewDetector(sessionID, patterns, onTransit)

	sub := &relay.Subscriber{
		Msgs: make(chan []byte, 256),
		// CloseSlow is a no-op for the status detector: if the channel backs up,
		// we simply drop the frame via the default branch in the select below.
		CloseSlow: func() {},
	}

	// Subscribe before reading the snapshot to avoid a race where output
	// arrives between Snapshot and Subscribe.
	hub.Subscribe(sub)

	// Feed existing scrollback so initial status is accurate.
	// The scrollback stores concatenated MakeOutputFrame bytes (each prefixed
	// with 0x01). Strip the leading MsgOutput byte before feeding the detector
	// so binary framing does not pollute the rolling tail.
	if snap := hub.ScrollbackSnapshot(); len(snap) > 0 {
		if snap[0] == relay.MsgOutput {
			snap = snap[1:]
		}
		detector.Feed(snap)
	}

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() { hub.Unsubscribe(sub) })
	}
	defer unsubscribe()

	for {
		select {
		case frame := <-sub.Msgs:
			msgType, payload, err := relay.ParseFrame(frame)
			if err != nil || msgType != relay.MsgOutput {
				continue
			}
			detector.Feed(payload)
		case <-hub.Done():
			return
		}
	}
}
