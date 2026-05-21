package tui

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"

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

	// client is the transport handle used by all files Cmds dispatched while
	// this view is active. Phase 121 used Model.client (the local Unix-socket
	// DaemonClient) implicitly; Phase 122 makes it explicit so a remote
	// HTTPS+cap RemoteFilesClient can stand in for cross-tailnet browsing
	// without touching the rest of the Update loop.
	client FilesClient

	// generation is a monotonic counter bumped on every reset
	// (newFilesModel) and on every navigation that supersedes a prior
	// in-flight request (Enter into dir, Backspace up). Each outgoing
	// loadDir / head / read cmd is stamped with the current generation;
	// apply*Msg handlers discard messages whose generation < current.
	// Closes WR-03 (reset-during-fetch race).
	generation uint64

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
//
// Phase 122: this constructor leaves filesModel.client nil — the caller in
// update.go's FilesOpen branch is responsible for wiring m.client (the local
// DaemonClient) into m.files.client immediately after newFilesModel returns.
// Prefer newFilesModelWithClient where the client is known up-front (which is
// every production path post-Phase 122).
func newFilesModel(sid string, listW, listH, previewW, previewH int) filesModel {
	return newFilesModelWithClient(sid, nil, listW, listH, previewW, previewH)
}

// newFilesModelWithClient is the explicit-client constructor introduced in
// Phase 122. Local FilesOpen passes m.client (a *daemon.DaemonClient); remote
// FilesOpen passes a freshly-constructed *RemoteFilesClient. Both satisfy
// FilesClient.
func newFilesModelWithClient(sid string, client FilesClient, listW, listH, previewW, previewH int) filesModel {
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
		client:      client,
		loading:     true,
		filterInput: fi,
		preview:     pv,
		previewKind: previewEmpty,
		generation:  1, // start at 1 so a zero-valued msg looks stale
	}
}

// isRemoteClient reports whether the active FilesClient is a remote-HTTPS
// transport. Used by the 401 → forget-cap branch in applyFilesListMsg.
func (fm filesModel) isRemoteClient() bool {
	_, ok := fm.client.(*RemoteFilesClient)
	return ok
}

// renderFilesListPane renders the left pane (file list + optional inline
// filter input). The focused pane gets BorderAccent; the other BorderNormal.
func (m Model) renderFilesListPane(w, h int, focused bool) string {
	borderColor := m.styles.BorderNormal
	if focused {
		borderColor = m.styles.BorderAccent
	}
	innerW := w - 2
	if innerW < 4 {
		innerW = 4
	}
	innerH := h - 2
	if innerH < 1 {
		innerH = 1
	}

	// WR-01: Re-sync filter input width on every render so terminal resizes
	// (and tab-switch-back-into-Files) reflow the input correctly. The
	// preview viewport applies the same pattern below. The "- 4" matches
	// the budget used in newFilesModel; clamp to 1 to avoid SetWidth(0).
	filterW := innerW - 4
	if filterW < 1 {
		filterW = 1
	}
	m.files.filterInput.SetWidth(filterW)

	entries := m.files.filteredEntries()

	// Reserve a bottom row for the filter input or static filter hint.
	hasFilterRow := m.files.filterActive || strings.TrimSpace(m.files.filterInput.Value()) != ""
	rowBudget := innerH
	if hasFilterRow {
		rowBudget--
		if rowBudget < 0 {
			rowBudget = 0
		}
	}

	var rows []string
	if len(entries) == 0 {
		empty := lipgloss.Place(innerW, max(1, rowBudget),
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("No files"))
		rows = append(rows, empty)
	} else {
		// Visible window with the selected row pinned in view.
		start := 0
		if rowBudget > 0 && m.files.selected >= rowBudget {
			start = m.files.selected - rowBudget + 1
		}
		end := start + rowBudget
		if end > len(entries) {
			end = len(entries)
		}
		for i := start; i < end; i++ {
			e := entries[i]
			name := ansi.Strip(e.Name)
			cursor := "  "
			if i == m.files.selected {
				cursor = "> "
			}
			suffix := ""
			if e.IsDir {
				suffix = "/"
			}
			text := cursor + truncate(name+suffix, innerW-2)
			if i == m.files.selected {
				text = lipgloss.NewStyle().
					Background(m.styles.BgSelected).
					Foreground(m.styles.FgSelected).
					Width(innerW).Render(text)
			} else {
				text = lipgloss.NewStyle().
					Foreground(m.styles.FgNormal).
					Width(innerW).Render(text)
			}
			rows = append(rows, text)
		}
		// Pad to row budget so the filter row stays anchored at the bottom.
		for len(rows) < rowBudget {
			rows = append(rows, lipgloss.NewStyle().Width(innerW).Render(""))
		}
	}

	if hasFilterRow {
		var filterRow string
		if m.files.filterActive {
			filterRow = lipgloss.NewStyle().Foreground(m.styles.FgAccent).
				Render("/ ") + m.files.filterInput.View()
		} else {
			filterRow = lipgloss.NewStyle().Foreground(m.styles.FgMuted).
				Render("/ " + strings.TrimSpace(m.files.filterInput.Value()))
		}
		rows = append(rows, lipgloss.NewStyle().Width(innerW).Render(filterRow))
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return m.wrapInFrame(inner, " Files ", w, borderColor)
}

// renderFilesPreviewPane renders the right pane: preview content + border.
// The frame title varies with previewKind so the user can tell at a glance
// what kind of content is being shown.
func (m Model) renderFilesPreviewPane(w, h int, focused bool) string {
	borderColor := m.styles.BorderNormal
	if focused {
		borderColor = m.styles.BorderAccent
	}
	innerW := w - 2
	if innerW < 4 {
		innerW = 4
	}
	innerH := h - 2
	if innerH < 1 {
		innerH = 1
	}

	// Match the viewport to the inner pane dimensions on every render so
	// terminal resizes don't leave a stale viewport size.
	m.files.preview.SetWidth(innerW)
	m.files.preview.SetHeight(innerH)

	var title string
	switch m.files.previewKind {
	case previewMarkdown:
		title = " Markdown "
	case previewBinary:
		title = " Binary "
	case previewOverCap:
		title = " Too large "
	case previewErr:
		title = " Error "
	case previewText:
		title = " Preview "
	default:
		title = " Preview "
	}

	var body string
	switch {
	case m.files.previewLoading:
		body = lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("Loading…"))
	default:
		body = m.files.preview.View()
	}

	return m.wrapInFrame(body, title, w, borderColor)
}

// renderFilesStatusLine renders the bottom status line shown beneath the
// two panes. Format: `{trunc cwd} • N entries[(truncated)] • i/N`. When
// `m.files.err != nil` the line is replaced by the error string in the
// StatusErrored color.
func (m Model) renderFilesStatusLine(w int) string {
	if m.files.err != nil {
		return lipgloss.NewStyle().
			Foreground(m.styles.StatusErrored).
			Width(w).
			Render(truncate("Error: "+m.files.err.Error(), w))
	}

	displayPath := m.files.cwd
	if displayPath == "" || displayPath == "." {
		displayPath = "/"
	} else if !strings.HasPrefix(displayPath, "/") {
		displayPath = "./" + displayPath
	}

	entries := m.files.filteredEntries()
	n := len(entries)
	trunc := ""
	if m.files.truncated {
		trunc = " (truncated)"
	}
	sel := 0
	if n > 0 {
		sel = m.files.selected + 1
		if sel > n {
			sel = n
		}
	}

	// WR-06: Compute the tail string first and subtract its actual width
	// instead of a magic 40-col reservation. With 9999 entries the tail
	// approaches 36 chars and "(truncated)" adds 12 — well past 40 once
	// the daemon's truncation cap is ever bumped. The magic constant
	// would silently start clipping the leaf segment from the right
	// (the opposite of truncateLeft's intent).
	tail := fmt.Sprintf(" • %d entries%s • %d/%d", n, trunc, sel, n)
	pathBudget := w - lipgloss.Width(tail)
	if pathBudget < 10 {
		pathBudget = 10
	}
	pathPart := truncateLeft(displayPath, pathBudget)

	body := pathPart + tail
	return lipgloss.NewStyle().Foreground(m.styles.FgMuted).Width(w).Render(body)
}

// renderFilesTab assembles the two-pane (list | preview) layout plus the
// status line. 40 % / 60 % split with a 1-char separator column.
func (m Model) renderFilesTab(cw, ch int) string {
	listW := cw * 40 / 100
	if listW < 10 {
		listW = 10
	}
	previewW := cw - listW - 1
	if previewW < 10 {
		previewW = 10
	}
	bodyH := ch - 1
	if bodyH < 3 {
		bodyH = 3
	}

	listPane := m.renderFilesListPane(listW, bodyH, !m.files.previewFocused)
	previewPane := m.renderFilesPreviewPane(previewW, bodyH, m.files.previewFocused)

	// WR-02: strings.Repeat("│\n", bodyH) yields bodyH glyphs PLUS a trailing
	// newline, making sepCol one row taller than listPane/previewPane. The
	// extra row either produced a phantom blank line or got silently clipped
	// — either way a fragile dependency on lipgloss-internal padding. Strip
	// the trailing newline so all three columns are exactly bodyH lines.
	sepCol := lipgloss.NewStyle().
		Foreground(m.styles.BorderNormal).
		Render(strings.TrimRight(strings.Repeat("│\n", bodyH), "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, listPane, sepCol, previewPane)
	status := m.renderFilesStatusLine(cw)
	return lipgloss.JoinVertical(lipgloss.Left, body, status)
}

// parentDir returns the parent directory of p using forward-slash path
// semantics (the daemon side normalises to '/'). Returns "" for root ("",
// ".", or "/"), and strips a single trailing slash before consulting
// path.Dir.
func parentDir(p string) string {
	p = strings.TrimRight(p, "/")
	if p == "" || p == "." {
		return ""
	}
	d := path.Dir(p)
	if d == "." || d == "/" {
		return ""
	}
	return d
}

// joinDir joins a base relative path with a child entry name. base "" or "."
// returns just name. Always uses forward slashes — daemon path semantics, not
// host filesystem.
func joinDir(base, name string) string {
	if base == "" || base == "." {
		return name
	}
	return path.Join(base, name)
}

// filteredEntries returns the visible entries given the current filter input.
// Uses ansi.Strip on entry names to defend against T-121-06 (filenames with
// embedded ANSI escapes — daemon trusts cwd, TUI must sanitise on render).
func (fm filesModel) filteredEntries() []daemon.FileEntry {
	q := strings.ToLower(strings.TrimSpace(fm.filterInput.Value()))
	if q == "" {
		return fm.entries
	}
	// WR-04: Use an explicit allocation rather than fm.entries[:0:0]. The
	// zero-capacity reslice is safe today (append always allocates fresh
	// backing storage on first growth) but a future maintainer changing
	// it to fm.entries[:0] would silently corrupt fm.entries because the
	// backing array is shared. Allocate cleanly to remove the footgun;
	// entries are capped by the daemon's truncation limit.
	out := make([]daemon.FileEntry, 0, len(fm.entries))
	for _, e := range fm.entries {
		if strings.Contains(strings.ToLower(ansi.Strip(e.Name)), q) {
			out = append(out, e)
		}
	}
	return out
}

// handleFilesKey is the full Plan 02 dispatcher for the Files tab. The
// filter-mode cascade is strict: when filterActive == true, ALL keys except
// Esc and Enter route to the textinput. This is what protects
// Pitfall TUI-PITFALL-2 (Backspace must NOT navigate-up while filtering).
func (m Model) handleFilesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	// Filter-mode capture (Pitfall TUI-PITFALL-2).
	if m.files.filterActive {
		switch s {
		case "esc":
			m.files.filterActive = false
			m.files.filterInput.SetValue("")
			m.files.filterInput.Blur()
			if m.files.selected >= len(m.files.entries) {
				m.files.selected = max(0, len(m.files.entries)-1)
			}
			return m, nil
		case "enter":
			m.files.filterActive = false
			m.files.filterInput.Blur()
			if n := len(m.files.filteredEntries()); m.files.selected >= n {
				m.files.selected = max(0, n-1)
			}
			return m, nil
		case "ctrl+c":
			// CR-01: Ctrl+C is the universal "quit" contract for this TUI
			// (see Quit binding in keys.go and matching paths in update.go).
			// The bundled textinput has no Ctrl+C handler — it would
			// silently swallow the key — so we MUST intercept it BEFORE
			// forwarding to the textinput. Otherwise the user cannot quit
			// the TUI while typing a filter.
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.files.filterInput, cmd = m.files.filterInput.Update(msg)
		// Reset selection to top so the cursor lands on the first match as
		// the user types.
		m.files.selected = 0
		return m, cmd
	}

	// Non-filter mode.
	switch {
	case key.Matches(msg, m.keys.FilterStart):
		m.files.filterActive = true
		cmd := m.files.filterInput.Focus()
		return m, cmd

	case s == "backspace" || s == "left":
		// TUI-03: Backspace at root is a hard no-op.
		if m.files.cwd == "" || m.files.cwd == "." {
			return m, nil
		}
		parent := parentDir(m.files.cwd)
		m.files.loading = true
		m.files.generation++ // WR-03: supersede any in-flight load
		return m, loadDirCmd(m.files.client, m.files.sessionID, parent, m.files.generation)

	case key.Matches(msg, m.keys.Up):
		if m.files.previewFocused {
			// WR-05: The bundled viewport only scrolls on PgUp/PgDn/Home/End/
			// MouseWheel — Up/Down are NOT in its default KeyMap. Forwarding
			// the arrow keys via Update() was a no-op, leaving "the arrow
			// keys dead in preview mode". Translate explicitly to a single-
			// line scroll so the hint bar's "Up/Down" promise is honored.
			m.files.preview.ScrollUp(1)
			return m, nil
		}
		if m.files.selected > 0 {
			m.files.selected--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.files.previewFocused {
			// WR-05: explicit ScrollDown — see Up branch above.
			m.files.preview.ScrollDown(1)
			return m, nil
		}
		n := len(m.files.filteredEntries())
		if m.files.selected < n-1 {
			m.files.selected++
		}
		return m, nil

	case s == "pgup":
		if m.files.previewFocused {
			var cmd tea.Cmd
			m.files.preview, cmd = m.files.preview.Update(msg)
			return m, cmd
		}
		m.files.selected = max(0, m.files.selected-10)
		return m, nil

	case s == "pgdown":
		if m.files.previewFocused {
			var cmd tea.Cmd
			m.files.preview, cmd = m.files.preview.Update(msg)
			return m, cmd
		}
		n := len(m.files.filteredEntries())
		target := m.files.selected + 10
		if target > n-1 {
			target = n - 1
		}
		if target < 0 {
			target = 0
		}
		m.files.selected = target
		return m, nil

	case s == "enter":
		entries := m.files.filteredEntries()
		if len(entries) == 0 || m.files.selected < 0 || m.files.selected >= len(entries) {
			return m, nil
		}
		entry := entries[m.files.selected]
		name := ansi.Strip(entry.Name)
		next := joinDir(m.files.cwd, name)
		if entry.IsDir {
			m.files.loading = true
			m.files.generation++ // WR-03: supersede any in-flight load
			return m, loadDirCmd(m.files.client, m.files.sessionID, next, m.files.generation)
		}
		m.files.previewLoading = true
		m.files.generation++ // WR-03: supersede any in-flight preview
		return m, headFileCmd(m.files.client, m.files.sessionID, next, m.files.generation)

	case key.Matches(msg, m.keys.FilesFocusToggle):
		m.files.previewFocused = !m.files.previewFocused
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil

	case key.Matches(msg, m.keys.PrevTab):
		m.cycleTab(-1)
		return m, nil

	case key.Matches(msg, m.keys.NextTab):
		m.cycleTab(+1)
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	// Swallow all other keys (do NOT fall through to handleContentKey).
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
	// WR-03: discard messages from superseded in-flight requests so a slow
	// previous-cwd reply cannot land on a freshly-reset model.
	if msg.generation < m.files.generation {
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
	// WR-03: discard messages from superseded in-flight requests.
	if msg.generation < m.files.generation {
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
	// WR-03: propagate the head's generation through to the read so a stale
	// read can't land on a fresher model.
	return m, readFileCmd(m.files.client, msg.sessionID, msg.relPath, msg.generation)
}

// applyFilesReadMsg handles the ReadFile result. Markdown (by suffix OR by
// mime) is rendered via the glamour markdown renderer; on render error we
// fall back to plain text so the user still sees something. Style picks
// "dark" vs "light" based on hasDark — matches Phase 86 lipgloss styling.
func (m Model) applyFilesReadMsg(msg filesReadMsg) (Model, tea.Cmd) {
	if msg.sessionID != m.files.sessionID {
		return m, nil
	}
	// WR-03: discard messages from superseded in-flight requests.
	if msg.generation < m.files.generation {
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
