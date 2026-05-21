package tui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/scottkw/agenthub/internal/daemon"
)

// errNilClient is returned by the files Cmds when DaemonClient is nil. The
// existing testModel() (update_test.go) injects a nil client to keep tests
// off the network — these guards keep that path panic-free.
var errNilClient = errors.New("tui: nil DaemonClient")

// filesListMsg carries the result of a ListFiles round-trip. Plan 02's
// Update handler uses sessionID + relPath to ignore stale results from a
// previous session (T-121-04). The generation field (WR-03) lets the
// handler discard messages from a superseded in-flight request even
// inside the same session.
type filesListMsg struct {
	sessionID  string
	generation uint64
	relPath    string
	entries    []daemon.FileEntry
	truncated  bool
	err        error
}

// filesReadMsg carries the result of a ReadFile round-trip.
type filesReadMsg struct {
	sessionID  string
	generation uint64
	relPath    string
	data       []byte
	mime       string
	err        error
}

// filesHeadMsg carries the result of a HeadFile preflight. The daemon's
// modtime is intentionally dropped at this layer — the TUI status line uses
// the cwd-relative path, not the modtime.
type filesHeadMsg struct {
	sessionID  string
	generation uint64
	relPath    string
	size       int64
	mime       string
	err        error
}

// filesErrMsg is a generic error envelope reserved for Plan 02 (e.g. for
// non-route errors like preview-render failures). Plan 01 does not emit it.
type filesErrMsg struct {
	sessionID string
	relPath   string
	err       error
}

// loadDirCmd returns a tea.Cmd that lists a directory inside the session
// sandbox. 5-second timeout — a hung daemon must not freeze the Update loop
// (T-121-01). The gen value (WR-03) is stamped on the result so the
// Update-side handler can discard messages from a superseded request.
func loadDirCmd(client *daemon.DaemonClient, sid, relPath string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return filesListMsg{sessionID: sid, generation: gen, relPath: relPath, err: errNilClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		entries, truncated, err := client.ListFiles(ctx, sid, relPath)
		return filesListMsg{
			sessionID:  sid,
			generation: gen,
			relPath:    relPath,
			entries:    entries,
			truncated:  truncated,
			err:        err,
		}
	}
}

// readFileCmd returns a tea.Cmd that fetches file bytes from the daemon.
// 10-second timeout to accommodate transferring a body up to the 5 MiB cap.
func readFileCmd(client *daemon.DaemonClient, sid, relPath string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return filesReadMsg{sessionID: sid, generation: gen, relPath: relPath, err: errNilClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		data, mime, err := client.ReadFile(ctx, sid, relPath)
		return filesReadMsg{
			sessionID:  sid,
			generation: gen,
			relPath:    relPath,
			data:       data,
			mime:       mime,
			err:        err,
		}
	}
}

// headFileCmd returns a tea.Cmd that preflights a file via HEAD — size +
// content-type without transferring the body. 5-second timeout matches
// loadDirCmd.
func headFileCmd(client *daemon.DaemonClient, sid, relPath string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return filesHeadMsg{sessionID: sid, generation: gen, relPath: relPath, err: errNilClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		size, mime, _, err := client.HeadFile(ctx, sid, relPath)
		return filesHeadMsg{
			sessionID:  sid,
			generation: gen,
			relPath:    relPath,
			size:       size,
			mime:       mime,
			err:        err,
		}
	}
}
