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
	pathBudget := w - 40
	if pathBudget < 10 {
		pathBudget = 10
	}
	pathPart := truncateLeft(displayPath, pathBudget)

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

	body := fmt.Sprintf("%s • %d entries%s • %d/%d", pathPart, n, trunc, sel, n)
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

	sepCol := lipgloss.NewStyle().
		Foreground(m.styles.BorderNormal).
		Render(strings.Repeat("│\n", bodyH))

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
	out := fm.entries[:0:0]
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
		return m, loadDirCmd(m.client, m.files.sessionID, parent)

	case key.Matches(msg, m.keys.Up):
		if m.files.previewFocused {
			var cmd tea.Cmd
			m.files.preview, cmd = m.files.preview.Update(msg)
			return m, cmd
		}
		if m.files.selected > 0 {
			m.files.selected--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.files.previewFocused {
			var cmd tea.Cmd
			m.files.preview, cmd = m.files.preview.Update(msg)
			return m, cmd
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
			return m, loadDirCmd(m.client, m.files.sessionID, next)
		}
		m.files.previewLoading = true
		return m, headFileCmd(m.client, m.files.sessionID, next)

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
