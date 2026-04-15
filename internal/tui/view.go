package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	header := m.renderHeader()
	colHeaders := m.renderColumnHeaders()
	list := m.renderSessionList()
	separator := ""
	footer := m.renderFooter()

	// Join all sections vertically
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		colHeaders,
		list,
		separator,
		footer,
	)

	// Modal overlays (rendered on top of content)
	if m.modal == modalNewSession {
		return m.renderNewSessionModal()
	}
	if m.modal == modalKillConfirm {
		return m.renderKillConfirmModal()
	}

	return content
}

// STUB: replaced by Plan 77-04 with full new-session modal rendering
func (m Model) renderNewSessionModal() string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "New Session (loading...)")
}

// renderHeader renders "AgentHub" (bold) + session count, right-aligned.
func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().Bold(true).
		Foreground(m.styles.FgNormal).
		Render("AgentHub")

	count := fmt.Sprintf("%d sessions", len(m.sessions))
	if len(m.sessions) == 1 {
		count = "1 session"
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

	// Empty state
	if len(m.sessions) == 0 {
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
		rows = append(rows, m.renderSessionRow(m.sessions[i], i))
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
	total := len(m.sessions)
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

	agent := truncate(s.CLI, 12)
	host := truncate(s.Hostname, 20)
	viewers := ""
	if s.ViewerCount > 0 {
		viewers = fmt.Sprintf("%d", s.ViewerCount)
	}

	row := fmt.Sprintf("%s%s %-*s  %-12s  %-20s  %7s",
		cursor, styledGlyph, nameWidth, name, agent, host, viewers)

	if isSelected {
		return lipgloss.NewStyle().
			Background(m.styles.BgSelected).
			Foreground(m.styles.FgSelected).
			Width(m.width).
			Render(row)
	}
	return lipgloss.NewStyle().
		Foreground(m.styles.FgNormal).
		Width(m.width).
		Render(row)
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
	quitHint := lipgloss.NewStyle().Foreground(m.styles.FgAccent).Render("q Quit")

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
	gap := m.width - lipgloss.Width(webPart) - len(sep) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return webPart + sep + strings.Repeat(" ", gap) + right
}

// renderHintBar renders the keybinding hint bar (bottom-most footer line).
func (m Model) renderHintBar() string {
	hint := "j/k Up/Down  Enter Attach  n New  d Kill  r Rename  ? Help  q Quit"
	return lipgloss.NewStyle().Foreground(m.styles.FgMuted).
		Width(m.width).Render(hint)
}

// nameColWidth calculates the flexible Name column width.
// Fixed columns: cursor(2) + status(2) + agent(12) + host(20) + viewers(7) + gaps(5*2=10) = 53
func (m Model) nameColWidth() int {
	w := m.width - 53
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
