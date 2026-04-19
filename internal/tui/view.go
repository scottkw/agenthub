package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/scottkw/agenthub/internal/daemon"
)

// View returns the Bubble Tea view. Uses alt-screen, no mouse mode.
func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	// NO mouse mode per REQUIREMENTS.md

	// Handle missing dimensions (before first WindowSizeMsg)
	if m.width == 0 || m.height == 0 {
		v.SetContent("Loading sessions...")
		return v
	}

	// Minimum terminal size check
	if m.width < 60 || m.height < 10 {
		v.SetContent(lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			"Terminal too small (need 60x10)"))
		return v
	}

	v.SetContent(m.renderFull())
	return v
}

// renderFull composes the complete screen content.
// When help overlay is open, it replaces the entire content.
func (m Model) renderFull() string {
	if m.showHelp {
		return m.renderHelpOverlay()
	}
	// Modal overlays (rendered on top of entire screen — unchanged)
	if m.modal == modalNewSession {
		return m.renderNewSessionModal()
	}
	if m.modal == modalKillConfirm {
		return m.renderKillConfirmModal()
	}
	if m.qrSession != nil {
		return m.renderQROverlay()
	}

	// Two-pane layout: sidebar | separator | (tabBar + content + footer)
	sidebar := m.renderSidebar()

	tabBar := m.renderTabBar()
	content := m.renderContentPane()
	footer := m.renderFooter()

	right := lipgloss.JoinVertical(lipgloss.Left, tabBar, content, footer)

	// Vertical separator between sidebar and content
	sepHeight := m.height
	sep := lipgloss.NewStyle().
		Foreground(m.styles.BorderNormal).
		Render(strings.Repeat("\u2502\n", sepHeight))

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep, right)
}

// renderSidebar renders the vertical sidebar with section labels.
func (m Model) renderSidebar() string {
	items := []struct {
		id    int
		label string
	}{
		{0, "Home"},
		{1, "Sessions"},
		{2, "Remote"},
		{3, "Settings"},
	}

	sideW := m.sidebarWidth()
	var rows []string
	for _, item := range items {
		style := lipgloss.NewStyle().Width(sideW)
		if item.id == m.sidebarFocus {
			// Active item: accent foreground, bold
			style = style.Bold(true).Foreground(m.styles.FgAccent)
			if m.panesFocus == focusSidebar {
				// Focused: add selected background
				style = style.Background(m.styles.BgSelected)
			}
		} else {
			style = style.Foreground(m.styles.FgNormal)
		}
		rows = append(rows, style.Render("  "+item.label))
	}

	content := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Width(sideW).
		Height(m.height).
		Render(content)
}

// renderTabBar renders the horizontal tab bar above the content pane.
func (m Model) renderTabBar() string {
	if len(m.openTabs) == 0 {
		return ""
	}

	var parts []string
	for i, tab := range m.openTabs {
		label := "  " + tabName(tab) + "  "
		if i == m.activeTab {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(m.styles.FgAccent).
				Underline(true).
				Render(label))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(m.styles.FgMuted).
				Render(label))
		}
	}
	return lipgloss.NewStyle().
		Width(m.contentWidth()).
		Render(strings.Join(parts, ""))
}

// renderContentPane dispatches rendering to the active tab's content renderer.
func (m Model) renderContentPane() string {
	// Content pane height: total height minus tab bar (1 line) minus footer (2 lines)
	contentHeight := m.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	cw := m.contentWidth()

	var content string
	switch m.activeTabID() {
	case tabHome:
		content = m.renderHomeTab(cw, contentHeight)
	case tabSessions:
		content = m.renderSessionFrame(cw, contentHeight)
	case tabRemote:
		content = m.renderRemoteTab(cw, contentHeight)
	case tabSettings:
		content = m.renderSettingsTab(cw, contentHeight)
	default:
		content = m.renderSessionFrame(cw, contentHeight)
	}

	return lipgloss.NewStyle().
		Width(cw).
		Height(contentHeight).
		Render(content)
}

// renderSessionFrame renders the session list inside a bordered frame with title.
func (m Model) renderSessionFrame(cw, ch int) string {
	// Determine border color based on pane focus
	borderColor := m.styles.BorderNormal
	if m.panesFocus == focusContent {
		borderColor = m.styles.BorderAccent
	}

	// Error state
	if m.err != nil {
		errMsg := "Cannot connect to daemon. Is it running?"
		inner := lipgloss.Place(cw-4, ch-2,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(m.styles.StatusErrored).Render(errMsg))
		return m.wrapInFrame(inner, " Sessions ", cw, borderColor)
	}

	// Loading state
	if m.loading {
		inner := lipgloss.Place(cw-4, ch-2,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("Loading sessions..."))
		return m.wrapInFrame(inner, " Sessions ", cw, borderColor)
	}

	// Empty state
	if len(m.sessions) == 0 && len(m.remoteSessions) == 0 {
		emptyText := lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgNormal).Render("No sessions"),
			lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("Press n to create a new session"),
		)
		inner := lipgloss.Place(cw-4, ch-2, lipgloss.Center, lipgloss.Center, emptyText)
		return m.wrapInFrame(inner, " Sessions ", cw, borderColor)
	}

	// Column headers (per D-10: first row inside frame)
	nameWidth := m.nameColWidth()
	colHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgMuted)
	colHeaders := colHeaderStyle.Render(fmt.Sprintf("    %-*s  %-12s  %-20s  %7s",
		nameWidth, "NAME", "AGENT", "HOST", "VIEWERS"))

	// Separator line below column headers
	sepLine := lipgloss.NewStyle().Foreground(m.styles.BorderNormal).
		Render(strings.Repeat("\u2500", cw-4))

	// Session rows (reuse existing visibleRange logic)
	listHeight := ch - 4 // frame border(2) + colHeaders(1) + separator(1)
	if listHeight < 1 {
		listHeight = 1
	}
	start, end := m.visibleRange(listHeight)
	var rows []string
	for i := start; i < end; i++ {
		entry := m.unifiedList[i]
		switch entry.kind {
		case entryLocal:
			if entry.session != nil {
				rows = append(rows, m.renderSessionRow(*entry.session, i))
			}
		case entryRemote:
			if entry.remote != nil {
				rows = append(rows, m.renderRemoteSessionRow(entry.remote, i))
			}
		case entryDivider:
			rows = append(rows, m.renderDividerRow(entry.divider))
		}
	}
	rendered := strings.Join(rows, "\n")
	lineCount := end - start
	for lineCount < listHeight {
		rendered += "\n"
		lineCount++
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, colHeaders, sepLine, rendered)
	return m.wrapInFrame(inner, " Sessions ", cw, borderColor)
}

// wrapInFrame wraps content in a bordered lipgloss frame with a title.
func (m Model) wrapInFrame(inner, title string, width int, borderColor color.Color) string {
	frameWidth := width - 2 // outer width minus frame chars
	if frameWidth < 10 {
		frameWidth = 10
	}
	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(frameWidth).
		Render(inner)

	styledTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.FgAccent).
		Render(title)

	return injectBorderTitle(bordered, styledTitle, borderColor)
}

// renderHomeTab renders the Home tab with branding, stats, and quick-action hints.
func (m Model) renderHomeTab(cw, ch int) string {
	// Title line: "AgentHub  v{version}"
	title := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgNormal).Render("AgentHub")
	ver := lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("  v" + m.version)
	titleLine := title + ver

	// Tagline
	tagline := lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("AI coding terminal sessions")

	// Separator
	sep := lipgloss.NewStyle().Foreground(m.styles.BorderNormal).
		Render(strings.Repeat("\u2500", min(cw-4, 40)))

	// Session counts
	running, idle, waiting, errored := 0, 0, 0, 0
	for _, s := range m.sessions {
		switch s.Status {
		case "running":
			running++
		case "idle":
			idle++
		case "waiting":
			waiting++
		case "errored":
			errored++
		}
	}
	label := lipgloss.NewStyle().Foreground(m.styles.FgMuted)
	value := lipgloss.NewStyle().Foreground(m.styles.FgNormal)
	var statParts []string
	if running > 0 {
		g := lipgloss.NewStyle().Foreground(m.styles.StatusRunning).Render("\u25CF")
		statParts = append(statParts, fmt.Sprintf("%s %d running", g, running))
	}
	if idle > 0 {
		g := lipgloss.NewStyle().Foreground(m.styles.StatusIdle).Render("\u25CB")
		statParts = append(statParts, fmt.Sprintf("%s %d idle", g, idle))
	}
	if waiting > 0 {
		g := lipgloss.NewStyle().Foreground(m.styles.StatusWaiting).Render("\u25CE")
		statParts = append(statParts, fmt.Sprintf("%s %d waiting", g, waiting))
	}
	if errored > 0 {
		g := lipgloss.NewStyle().Foreground(m.styles.StatusErrored).Render("\u2717")
		statParts = append(statParts, fmt.Sprintf("%s %d errored", g, errored))
	}
	sessLine := label.Render("Sessions  ") + value.Render(strings.Join(statParts, "  "))
	if len(statParts) == 0 {
		sessLine = label.Render("Sessions  ") + value.Render("none")
	}

	// Web server status
	var webLine string
	if m.webStatus.Running {
		g := lipgloss.NewStyle().Foreground(m.styles.WebOn).Render("\u25CF")
		webLine = label.Render("Web       ") + fmt.Sprintf("%s Running \u2014 %s", g, value.Render(m.webStatus.URL))
	} else {
		g := lipgloss.NewStyle().Foreground(m.styles.WebOff).Render("\u25CB")
		webLine = label.Render("Web       ") + fmt.Sprintf("%s Stopped", g)
	}

	// Tailscale status (derive from remoteSessions per RESEARCH.md recommendation)
	var tsLine string
	if m.fetchRemoteFn != nil && len(m.remoteSessions) > 0 {
		g := lipgloss.NewStyle().Foreground(m.styles.StatusRunning).Render("\u25CF")
		hostCount := len(m.remoteSessions)
		tsLine = label.Render("Tailscale ") + fmt.Sprintf("%s Connected \u2014 %d peer(s)", g, hostCount)
	} else if m.fetchRemoteFn != nil {
		g := lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("\u25CB")
		tsLine = label.Render("Tailscale ") + fmt.Sprintf("%s No peers found", g)
	} else {
		g := lipgloss.NewStyle().Foreground(m.styles.StatusErrored).Render("\u2717")
		tsLine = label.Render("Tailscale ") + fmt.Sprintf("%s not configured", g)
	}

	// Quick-action hints
	hintKey := lipgloss.NewStyle().Foreground(m.styles.FgAccent)
	hintDesc := lipgloss.NewStyle().Foreground(m.styles.FgMuted)
	hints := hintKey.Render("n") + hintDesc.Render(" new session   ") +
		hintKey.Render("Enter") + hintDesc.Render(" attach   ") +
		hintKey.Render("q") + hintDesc.Render(" QR   ") +
		hintKey.Render("?") + hintDesc.Render(" help")

	lines := []string{
		"  " + titleLine,
		"  " + tagline,
		"  " + sep,
		"  " + sessLine,
		"  " + webLine,
		"  " + tsLine,
		"  " + sep,
		"  " + hints,
	}

	return strings.Join(lines, "\n")
}

// renderSettingsTab renders a read-only settings display.
func (m Model) renderSettingsTab(cw, ch int) string {
	label := lipgloss.NewStyle().Foreground(m.styles.FgMuted)
	value := lipgloss.NewStyle().Foreground(m.styles.FgNormal)

	header := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgNormal).Render("Settings")
	note := lipgloss.NewStyle().Italic(true).Foreground(m.styles.FgMuted).
		Render("read-only \u2014 run 'agenthub settings' to edit")
	sep := lipgloss.NewStyle().Foreground(m.styles.BorderNormal).
		Render(strings.Repeat("\u2500", min(cw-4, 40)))

	var webPort string
	if m.webStatus.Running {
		webPort = m.webStatus.URL
	} else {
		webPort = "stopped"
	}

	lines := []string{
		"  " + header + "  " + note,
		"  " + sep,
		"  " + label.Render("Web Server    ") + value.Render(webPort),
	}

	return strings.Join(lines, "\n")
}

// renderRemoteTab renders remote sessions in the same bordered frame pattern.
func (m Model) renderRemoteTab(cw, ch int) string {
	borderColor := m.styles.BorderNormal
	if m.panesFocus == focusContent {
		borderColor = m.styles.BorderAccent
	}

	if len(m.remoteSessions) == 0 {
		inner := lipgloss.Place(cw-4, ch-2,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("No remote sessions"))
		return m.wrapInFrame(inner, " Remote ", cw, borderColor)
	}

	// Column headers (same as sessions, minus VIEWERS)
	nameWidth := m.nameColWidth()
	colHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgMuted)
	colHeaders := colHeaderStyle.Render(fmt.Sprintf("    %-*s  %-12s  %-20s",
		nameWidth, "NAME", "AGENT", "HOST"))
	sepLine := lipgloss.NewStyle().Foreground(m.styles.BorderNormal).
		Render(strings.Repeat("\u2500", cw-4))

	// Build rows from remote groups only
	var rows []string
	for _, g := range m.remoteSessions {
		if len(g.Sessions) == 0 {
			continue
		}
		rows = append(rows, m.renderDividerRow(&peerDivider{
			Hostname: g.Hostname, SessionCount: len(g.Sessions),
		}))
		for i := range g.Sessions {
			rows = append(rows, m.renderRemoteSessionRow(&g.Sessions[i], -1))
		}
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, colHeaders, sepLine, strings.Join(rows, "\n"))
	return m.wrapInFrame(inner, " Remote ", cw, borderColor)
}

// renderHeader renders "AgentHub" (bold) + session count, right-aligned.
func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().Bold(true).
		Foreground(m.styles.FgNormal).
		Render("AgentHub")

	// Count local and remote sessions separately
	localCount := len(m.sessions)
	remoteCount := 0
	for _, e := range m.unifiedList {
		if e.kind == entryRemote {
			remoteCount++
		}
	}

	var count string
	if remoteCount > 0 {
		count = fmt.Sprintf("%d local, %d remote", localCount, remoteCount)
	} else if localCount == 1 {
		count = "1 session"
	} else {
		count = fmt.Sprintf("%d sessions", localCount)
	}

	countStyled := lipgloss.NewStyle().
		Foreground(m.styles.FgMuted).
		Render(count)

	// Right-align the count
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(countStyled)
	if gap < 1 {
		gap = 1
	}

	return title + strings.Repeat(" ", gap) + countStyled
}

// renderColumnHeaders renders the column label row with muted bold text.
func (m Model) renderColumnHeaders() string {
	nameWidth := m.nameColWidth()
	style := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgMuted)

	// Cursor(2) + Status(2) + Name + Agent(12) + Host(20) + Viewers(7)
	// with 2-char gaps between columns
	row := fmt.Sprintf("    %-*s  %-12s  %-20s  %7s",
		nameWidth, "NAME", "AGENT", "HOST", "VIEWERS")

	return style.Render(row)
}

// renderSessionList renders the scrollable session rows.
// Row budget: terminal_rows - 5 (header + col headers + separator + 2 footer lines).
func (m Model) renderSessionList() string {
	listHeight := m.height - 5
	if listHeight < 1 {
		listHeight = 1
	}

	// Error state
	if m.err != nil {
		errMsg := "Cannot connect to daemon. Is it running?"
		return lipgloss.Place(m.width, listHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(m.styles.StatusErrored).Render(errMsg))
	}

	// Loading state
	if m.loading {
		return lipgloss.Place(m.width, listHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("Loading sessions..."))
	}

	// Empty state: show when no local sessions and no remote sessions
	if len(m.sessions) == 0 && len(m.remoteSessions) == 0 {
		emptyText := lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgNormal).Render("No sessions"),
			lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("Press n to create a new session"),
		)
		return lipgloss.Place(m.width, listHeight,
			lipgloss.Center, lipgloss.Center, emptyText)
	}

	// Calculate visible window for scrolling
	start, end := m.visibleRange(listHeight)

	var rows []string
	for i := start; i < end; i++ {
		entry := m.unifiedList[i]
		switch entry.kind {
		case entryLocal:
			if entry.session != nil {
				rows = append(rows, m.renderSessionRow(*entry.session, i))
			}
		case entryRemote:
			if entry.remote != nil {
				rows = append(rows, m.renderRemoteSessionRow(entry.remote, i))
			}
		case entryDivider:
			rows = append(rows, m.renderDividerRow(entry.divider))
		}
	}

	// Pad remaining lines to fill the list area
	rendered := strings.Join(rows, "\n")
	lineCount := end - start
	for lineCount < listHeight {
		rendered += "\n"
		lineCount++
	}

	return rendered
}

// visibleRange calculates the start and end indices for the visible session window.
func (m Model) visibleRange(listHeight int) (int, int) {
	total := len(m.unifiedList)
	if total <= listHeight {
		return 0, total
	}

	// Keep selected item visible
	start := 0
	if m.selected >= listHeight {
		start = m.selected - listHeight + 1
	}
	end := start + listHeight
	if end > total {
		end = total
		start = end - listHeight
	}
	return start, end
}

// renderSessionRow renders a single session row with column layout per UI-SPEC.
func (m Model) renderSessionRow(s daemon.SessionInfo, idx int) string {
	isSelected := idx == m.selected

	// Cursor column (2 chars)
	cursor := "  "
	if isSelected {
		cursor = "> "
	}

	// Status glyph (2 chars: glyph + space)
	glyph, glyphColor := statusGlyph(s.Status, m.styles)
	styledGlyph := lipgloss.NewStyle().Foreground(glyphColor).Render(glyph)

	// Column widths per UI-SPEC
	nameWidth := m.nameColWidth()

	// Inline rename: replace name with textinput view
	var name string
	if m.editing && s.ID == m.editSessionID {
		name = m.editInput.View()
	} else {
		name = truncate(s.Name, nameWidth)
	}

	// Colored agent badge (per D-11, D-12)
	badgeColor := agentBadgeColor(s.CLI, m.styles)
	badgeText := "[" + truncate(strings.ToLower(s.CLI), 10) + "]"
	agent := lipgloss.NewStyle().Foreground(badgeColor).Render(fmt.Sprintf("%-12s", badgeText))
	host := truncate(s.Hostname, 20)
	viewers := ""
	if s.ViewerCount > 0 {
		viewers = fmt.Sprintf("%d", s.ViewerCount)
	}

	row := fmt.Sprintf("%s%s %-*s  %s  %-20s  %7s",
		cursor, styledGlyph, nameWidth, name, agent, host, viewers)

	if isSelected {
		return lipgloss.NewStyle().
			Background(m.styles.BgSelected).
			Foreground(m.styles.FgSelected).
			Width(m.contentWidth()).
			Render(row)
	}
	return lipgloss.NewStyle().
		Foreground(m.styles.FgNormal).
		Width(m.contentWidth()).
		Render(row)
}

// renderRemoteSessionRow renders a single remote session row with the same column layout
// as renderSessionRow, reading from a RemoteSessionEntry instead of daemon.SessionInfo.
func (m Model) renderRemoteSessionRow(r *RemoteSessionEntry, idx int) string {
	isSelected := idx == m.selected

	cursor := "  "
	if isSelected {
		cursor = "> "
	}

	glyph, glyphColor := statusGlyph(r.Status, m.styles)
	styledGlyph := lipgloss.NewStyle().Foreground(glyphColor).Render(glyph)

	nameWidth := m.nameColWidth()
	name := truncate(r.Name, nameWidth)
	badgeColor := agentBadgeColor(r.CLIType, m.styles)
	badgeText := "[" + truncate(strings.ToLower(r.CLIType), 10) + "]"
	agent := lipgloss.NewStyle().Foreground(badgeColor).Render(fmt.Sprintf("%-12s", badgeText))
	host := truncate(r.Hostname, 20)

	// Remote sessions do not expose viewer count — leave blank
	row := fmt.Sprintf("%s%s %-*s  %s  %-20s  %7s",
		cursor, styledGlyph, nameWidth, name, agent, host, "")

	if isSelected {
		return lipgloss.NewStyle().
			Background(m.styles.BgSelected).
			Foreground(m.styles.FgSelected).
			Width(m.contentWidth()).
			Render(row)
	}
	return lipgloss.NewStyle().
		Foreground(m.styles.FgNormal).
		Width(m.contentWidth()).
		Render(row)
}

// renderDividerRow renders a section divider row between peer groups per UI-SPEC.
// Format: "  ── Remote: {hostname} ({N} session/sessions) ──────..."
func (m Model) renderDividerRow(d *peerDivider) string {
	label := "session"
	if d.SessionCount != 1 {
		label = "sessions"
	}
	text := fmt.Sprintf("Remote: %s (%d %s)", d.Hostname, d.SessionCount, label)
	prefix := "  \u2500\u2500 " // two leading spaces + box-drawing horizontal chars
	suffix := " "
	labelStr := prefix + text + suffix

	// Accent-colored label portion
	labelPart := lipgloss.NewStyle().Foreground(m.styles.FgAccent).Render(labelStr)

	// Fill remaining width with box-drawing chars in muted color
	fillLen := m.contentWidth() - lipgloss.Width(labelStr)
	if fillLen > 0 {
		fillPart := lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render(strings.Repeat("\u2500", fillLen))
		return labelPart + fillPart
	}
	return labelPart
}

// renderFooter renders the two footer lines: web status and keybinding hints.
func (m Model) renderFooter() string {
	webLine := m.renderWebStatus()
	hintLine := m.renderHintBar()
	return lipgloss.JoinVertical(lipgloss.Left, webLine, hintLine)
}

// renderWebStatus renders the web server status footer line.
func (m Model) renderWebStatus() string {
	var webPart string
	if m.webStatus.Running {
		glyph := lipgloss.NewStyle().Foreground(m.styles.WebOn).Render("\u25CF")
		webPart = fmt.Sprintf("Web: %s Running -- %s", glyph, m.webStatus.URL)
	} else {
		glyph := lipgloss.NewStyle().Foreground(m.styles.WebOff).Render("\u25CB")
		webPart = fmt.Sprintf("Web: %s Stopped", glyph)
	}

	helpHint := lipgloss.NewStyle().Foreground(m.styles.FgAccent).Render("? Help")
	quitHint := lipgloss.NewStyle().Foreground(m.styles.FgAccent).Render("Q Quit")

	right := helpHint + "  " + quitHint

	// Toast message (if active and not expired) with kind-based coloring
	if m.toast != "" && time.Now().Before(m.toastExp) {
		var toastColor color.Color
		switch m.toastKind {
		case toastSuccess:
			toastColor = m.styles.StatusRunning
		case toastError:
			toastColor = m.styles.StatusErrored
		default:
			toastColor = m.styles.FgMuted
		}
		webPart = lipgloss.NewStyle().Foreground(toastColor).Render(m.toast)
	}

	sep := " | "
	gap := m.contentWidth() - lipgloss.Width(webPart) - lipgloss.Width(sep) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return webPart + sep + strings.Repeat(" ", gap) + right
}

// renderHintBar renders the keybinding hint bar (bottom-most footer line).
func (m Model) renderHintBar() string {
	hint := "j/k Up/Down  Enter Attach  q QR  n New  d Kill  r Rename  ? Help  Q Quit"
	return lipgloss.NewStyle().Foreground(m.styles.FgMuted).
		Width(m.contentWidth()).Render(hint)
}

// sidebarWidth returns the fixed sidebar column width.
func (m Model) sidebarWidth() int {
	return 16
}

// contentWidth returns the available width for the content pane.
func (m Model) contentWidth() int {
	w := m.width - m.sidebarWidth() - 1 // -1 for separator char
	if w < 20 {
		return 20
	}
	return w
}

// nameColWidth calculates the flexible Name column width.
// Fixed columns: cursor(2) + status(2) + agent(12) + host(20) + viewers(7) + gaps(5*2=10) = 53
func (m Model) nameColWidth() int {
	w := m.contentWidth() - 53
	if w < 8 {
		return 8
	}
	return w
}

// truncate truncates a string to maxWidth characters, appending "..." if truncated.
func truncate(s string, maxWidth int) string {
	if utf8.RuneCountInString(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return string([]rune(s)[:maxWidth])
	}
	return string([]rune(s)[:maxWidth-3]) + "..."
}

// injectBorderTitle splices a styled title into the top border line of a
// lipgloss-rendered box. It strips ANSI codes from the border before computing
// the insertion position so that escape sequences are not corrupted, then
// re-applies borderColor to the non-title portions of the line.
func injectBorderTitle(bordered string, title string, borderColor color.Color) string {
	lines := strings.Split(bordered, "\n")
	if len(lines) == 0 {
		return bordered
	}

	topBorder := lines[0]
	titleWidth := lipgloss.Width(title)
	borderWidth := lipgloss.Width(topBorder)
	if borderWidth <= titleWidth+4 {
		return bordered
	}

	// Strip ANSI to get clean border runes for safe splicing.
	clean := ansi.Strip(topBorder)
	runes := []rune(clean)
	insertPos := 3 // after corner + 2 border chars (e.g., "╭──")
	if insertPos+titleWidth > len(runes) {
		return bordered
	}

	// Build: prefix border chars + title + suffix border chars.
	// Each segment gets its own styling so ANSI codes don't overlap.
	prefix := string(runes[:insertPos])
	suffix := string(runes[insertPos+titleWidth:])

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	lines[0] = borderStyle.Render(prefix) + title + borderStyle.Render(suffix)
	return strings.Join(lines, "\n")
}

// statusGlyph maps a session status string to a Unicode glyph and color.
func statusGlyph(status string, s Styles) (string, color.Color) {
	switch status {
	case "idle":
		return "\u25CB", s.StatusIdle // WHITE CIRCLE (blue)
	case "waiting":
		return "\u25CF", s.StatusWaiting // BLACK CIRCLE (yellow)
	case "errored":
		return "\u2716", s.StatusErrored // HEAVY MULTIPLICATION X (red)
	default: // "running" or unrecognized
		return "\u25CF", s.StatusRunning // BLACK CIRCLE (green)
	}
}
