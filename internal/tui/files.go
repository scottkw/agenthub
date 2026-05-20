package tui

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/glamour"

	"github.com/scottkw/agenthub/internal/daemon"
)

// previewKind tags the rendering mode the preview pane is currently in.
type previewKind int

const (
	previewEmpty previewKind = iota
	previewText
	previewMarkdown
	previewBinary
	previewOverCap
	previewErr
)

// filesModel holds all state for the Phase 121 TUI Files view.
//
// Embedded as Model.files. ALWAYS reset on `f` keypress per RESEARCH.md
// Pitfall TUI-PITFALL-6 — never assume "same session as before". The struct
// shape is intentionally a superset of Plan 01's needs so Plan 02 can fill in
// the navigation, filter, and preview behaviour without touching the field
// list again.
type filesModel struct {
	sessionID string             // session whose cwd we're scoped to
	cwd       string             // relative path within sandbox; "" / "." = root
	entries   []daemon.FileEntry // current directory listing
	truncated bool               // ListFiles set X-Directory-Truncated
	selected  int                // cursor position in entries (after filtering)
	loading   bool               // listing in flight
	err       error              // last error (sticky until next nav)

	// Filter
	filterActive bool
	filterInput  textinput.Model

	// Preview pane
	preview        viewport.Model
	previewKind    previewKind
	previewMime    string
	previewLoading bool
	previewErr     error
	previewFocused bool // PgUp/PgDn route to viewport vs list
}

// newFilesModel zero-initialises a filesModel for the given session and pane
// dimensions. Plan 02 will populate cwd / entries on the first filesListMsg.
func newFilesModel(sid string, listW, listH, previewW, previewH int) filesModel {
	fi := textinput.New()
	fi.Prompt = "/ "
	if listW > 4 {
		fi.SetWidth(listW - 4)
	} else {
		fi.SetWidth(1)
	}
	fi.CharLimit = 128

	pv := viewport.New()
	if previewW > 0 {
		pv.SetWidth(previewW)
	}
	if previewH > 0 {
		pv.SetHeight(previewH)
	}

	_ = listH // reserved for Plan 02 list-pane sizing
	return filesModel{
		sessionID:   sid,
		loading:     true,
		filterInput: fi,
		preview:     pv,
		previewKind: previewEmpty,
	}
}

// renderFilesTab is a Plan-01 stub. Plan 02 will replace this with the
// two-pane (list | preview) layout. For now we render a single "Loading…"
// frame so the tab is wired into the dispatch table without entangling render
// logic.
func (m Model) renderFilesTab(cw, ch int) string {
	_ = ch
	return m.wrapInFrame("Loading…", " Files ", cw, m.styles.BorderNormal)
}

// handleFilesKey is a Plan-01 stub. It handles only the keys that must keep
// working from the very first commit (quit, help, tab cycling) and swallows
// everything else. Plan 02 will replace this with the full filter /
// navigation / preview dispatch.
func (m Model) handleFilesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "Q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = true
		return m, nil
	case "[":
		m.cycleTab(-1)
		return m, nil
	case "]":
		m.cycleTab(+1)
		return m, nil
	case "esc":
		// Plan 02 will route esc to clear filter or close the tab.
		return m, nil
	}
	// Swallow all other keys — Plan 02 fills in the rest.
	return m, nil
}

// previewSizeCap is the maximum byte size a file may have to be preview-able
// inline. Anything larger gets the "Too large…" refusal. Mirrors the daemon-
// side preview budget; the TUI enforces it independently so a misconfigured
// daemon can't drown the Update loop in a multi-MB body transfer.
const previewSizeCap int64 = 5 * 1024 * 1024

// applyFilesListMsg consumes a directory-listing result. Stale results (from
// a previous session ID) are silently discarded (T-121-04). A
// "session not found" error from the daemon is translated to a friendly
// "Session no longer running" message so the tab can stay open with a clear
// indicator that the underlying session is gone.
func (m Model) applyFilesListMsg(msg filesListMsg) (Model, tea.Cmd) {
	if msg.sessionID != m.files.sessionID {
		return m, nil
	}
	m.files.loading = false
	if msg.err != nil {
		if strings.Contains(msg.err.Error(), "session not found") {
			m.files.err = errors.New("Session no longer running")
		} else {
			m.files.err = msg.err
		}
		return m, nil
	}
	m.files.err = nil
	m.files.cwd = msg.relPath
	m.files.entries = msg.entries
	m.files.truncated = msg.truncated
	m.files.selected = 0
	// Reset preview when directory changes (T-121-09).
	m.files.preview.SetContent("")
	m.files.previewKind = previewEmpty
	m.files.previewErr = nil
	m.files.previewMime = ""
	m.files.previewLoading = false
	return m, nil
}

// applyFilesHeadMsg handles a HEAD preflight result. Decision tree:
//   - error                   → previewErr
//   - size > previewSizeCap   → previewOverCap (refusal message)
//   - !strings.HasPrefix(mime, "text/") → previewBinary (refusal message)
//   - otherwise               → dispatch readFileCmd
func (m Model) applyFilesHeadMsg(msg filesHeadMsg) (Model, tea.Cmd) {
	if msg.sessionID != m.files.sessionID {
		return m, nil
	}
	if msg.err != nil {
		m.files.previewKind = previewErr
		m.files.previewErr = msg.err
		m.files.previewLoading = false
		m.files.preview.SetContent("Error: " + msg.err.Error())
		return m, nil
	}
	if msg.size > previewSizeCap {
		m.files.previewKind = previewOverCap
		m.files.previewLoading = false
		m.files.preview.SetContent("Too large to preview, use desktop or web to download")
		return m, nil
	}
	if !strings.HasPrefix(msg.mime, "text/") {
		m.files.previewKind = previewBinary
		m.files.previewMime = msg.mime
		m.files.previewLoading = false
		m.files.preview.SetContent("Use desktop or web to preview")
		return m, nil
	}
	m.files.previewLoading = true
	m.files.previewMime = msg.mime
	return m, readFileCmd(m.client, msg.sessionID, msg.relPath)
}

// applyFilesReadMsg handles the ReadFile result. Markdown (by suffix OR by
// mime) is rendered via the glamour markdown renderer; on render error we
// fall back to plain text so the user still sees something. Style picks
// "dark" vs "light" based on hasDark — matches Phase 86 lipgloss styling.
func (m Model) applyFilesReadMsg(msg filesReadMsg) (Model, tea.Cmd) {
	if msg.sessionID != m.files.sessionID {
		return m, nil
	}
	m.files.previewLoading = false
	if msg.err != nil {
		m.files.previewKind = previewErr
		m.files.previewErr = msg.err
		m.files.preview.SetContent("Error: " + msg.err.Error())
		return m, nil
	}
	lower := strings.ToLower(msg.relPath)
	isMarkdown := strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".markdown") ||
		strings.HasPrefix(msg.mime, "text/markdown")
	if isMarkdown {
		style := "dark"
		if !m.hasDark {
			style = "light"
		}
		out, err := glamour.Render(string(msg.data), style)
		if err != nil {
			// Fall back to plain text on render error so the user still sees
			// the file contents (T-121-10: glamour is style-only, never
			// executes embedded HTML/JS).
			m.files.previewKind = previewText
			m.files.preview.SetContent(string(msg.data))
			return m, nil
		}
		m.files.previewKind = previewMarkdown
		m.files.preview.SetContent(out)
		return m, nil
	}
	m.files.previewKind = previewText
	m.files.preview.SetContent(string(msg.data))
	return m, nil
}
