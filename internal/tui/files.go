package tui

import (
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

// glamour is imported now so Plan 02 can use it for markdown rendering.
// Anchoring the dependency here keeps `go mod tidy` happy while leaving the
// renderer wiring for Plan 02. The reference is unexported and inert.
var _ = glamour.WithAutoStyle
