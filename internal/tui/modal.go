package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// agentEntry is a unified picker entry covering AI CLIs plus the single static
// shell row. cliKey is the value passed to daemon.CreateSession (e.g.
// "claude", "shell"). displayLabel is the human-readable string shown in the
// picker (e.g. "Claude Code", "Shell").
//
// Phase 108 PARITY-TUI-01: collapsed to one static "Shell" entry — daemon
// resolves the binary via shellPath (engine.go:500-530), so the TUI no longer
// fans out per discovered shell.
type agentEntry struct {
	cliKey       string
	displayLabel string
}

// agentEntries returns the unified picker entry slice: AI CLIs first (in
// detectedCLIs order), then exactly one trailing "Shell" entry with
// cliKey="shell". The daemon resolves the actual binary from Settings'
// shellPath when it sees cli=="shell" (with a discovered-shell fallback
// for fresh installs — see resolveShellSpawn branch (4) in
// internal/daemon/engine.go).
//
// Always returns at least the single Shell entry. The Phase 108-era
// "return nil when detectedCLIs is empty" gate was reverted during the
// v3.3.1 Phase 109 Windows UAT after two failures were observed on a
// fresh Windows install with no AI CLIs configured:
//   - TUI new-session modal showed "Agent: (none found)" with no way to
//     pick Shell, blocking shell-session creation on Windows.
//   - lipgloss.Place panicked on the empty modal content (modal.go:69
//     "runtime error: index out of range [0] with length 0").
// Mirrors the parallel GUI fix to handleAddTab in App.tsx.
//
// Phase 108 PARITY-TUI-01: replaces the Phase 101 multi-row per-shell
// fan-out with a single static "Shell" entry. Mirrors the Phase 107 GUI
// NewSessionModal collapse.
func (m Model) agentEntries() []agentEntry {
	entries := make([]agentEntry, 0, len(m.detectedCLIs)+1)
	for _, c := range m.detectedCLIs {
		entries = append(entries, agentEntry{cliKey: c.Name, displayLabel: c.DisplayName})
	}
	entries = append(entries, agentEntry{cliKey: "shell", displayLabel: "Shell"})
	return entries
}

// renderNewSessionModal renders the new-session modal as a centered bordered overlay.
//
// Phase 117 / PAPER-01: defensive guard against degenerate dimensions.
// lipgloss.Place panics with "index out of range [0] with length 0" if width
// or height is zero (rendered before WindowSizeMsg arrives, or if some future
// code path leaves them unset). The Phase 109 agentEntries fix closed the
// "no detected CLIs → empty content → panic" upstream cause; this guard
// closes the broader class of "any zero-dimension render → panic".
func (m Model) renderNewSessionModal() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	overlayWidth := max(50, min(70, m.width-10))

	content := m.buildNewSessionContent()

	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.BorderNormal).
		Width(overlayWidth-2).
		Padding(1, 2).
		Background(m.styles.BgModal).
		Render(content)

	// Inject title into top border using ANSI-safe helper.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.BorderAccent).
		Render(" New Session ")
	bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, bordered)
}

// buildNewSessionContent builds the form fields for the new-session modal.
func (m Model) buildNewSessionContent() string {
	labelStyle := lipgloss.NewStyle().Foreground(m.styles.FgNormal)
	focusedLabelStyle := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgFocusedLabel)

	var sections []string

	// Agent field (focusedField == 0)
	lbl := labelStyle
	if m.focusedField == 0 {
		lbl = focusedLabelStyle
	}
	agentDisplay := m.renderAgentPicker()
	sections = append(sections, fmt.Sprintf("  %s  %s", lbl.Render("Agent:"), agentDisplay))

	// Directory field (focusedField == 1)
	lbl = labelStyle
	if m.focusedField == 1 {
		lbl = focusedLabelStyle
	}
	sections = append(sections, "") // field-gap
	sections = append(sections, fmt.Sprintf("  %s  %s", lbl.Render("Directory:"), m.dirInput.View()))

	// Arguments field (focusedField == 2)
	lbl = labelStyle
	if m.focusedField == 2 {
		lbl = focusedLabelStyle
	}
	sections = append(sections, "") // field-gap
	sections = append(sections, fmt.Sprintf("  %s  %s", lbl.Render("Arguments:"), m.argsInput.View()))

	// Hint row
	sections = append(sections, "") // gap
	hint := lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("Tab: next field  Enter: create  Esc: cancel")
	sections = append(sections, "          "+hint)

	return strings.Join(sections, "\n")
}

// renderAgentPicker displays the current agent with Left/Right cycle arrows.
// Phase 108 PARITY-TUI-02: shows AI CLI labels followed by a single static
// "Shell" entry (no em-dash, no per-shell variant). Daemon resolves the
// actual binary from Settings' shellPath when cli=="shell".
func (m Model) renderAgentPicker() string {
	entries := m.agentEntries()
	if len(entries) == 0 {
		return lipgloss.NewStyle().Foreground(m.styles.FgDanger).Render("(none found)")
	}
	// Defensive clamp: agentIdx might be stale after a discovery refresh.
	idx := m.agentIdx
	if idx < 0 || idx >= len(entries) {
		idx = 0
	}
	label := entries[idx].displayLabel
	arrows := lipgloss.NewStyle().Foreground(m.styles.FgMuted)
	return fmt.Sprintf("%s %s %s", arrows.Render("<"), label, arrows.Render(">"))
}

// cycleAgent cycles the agent picker index forward or backward.
// Phase 108 PARITY-TUI-01: walks the unified agentEntries slice (AI CLIs +
// one trailing static "Shell" entry).
func (m Model) cycleAgent(forward bool) Model {
	entries := m.agentEntries()
	n := len(entries)
	if n == 0 {
		return m
	}
	if forward {
		m.agentIdx = (m.agentIdx + 1) % n
	} else {
		m.agentIdx = (m.agentIdx + n - 1) % n
	}
	return m
}

// renderKillConfirmModal renders the kill confirmation dialog as a centered bordered overlay.
func (m Model) renderKillConfirmModal() string {
	overlayWidth := max(40, min(55, m.width-20))
	innerWidth := overlayWidth - 6 // subtract border (2) + padding (4)

	// Question line: Kill session "name"?
	name := m.killTarget.Name
	question := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.FgDanger).
		Render(fmt.Sprintf("Kill session %q?", name))

	// Detail line
	detail := lipgloss.NewStyle().
		Foreground(m.styles.FgMuted).
		Render("This will terminate the running process.")

	// Button rendering
	var noBtn, yesBtn string
	if !m.killFocusYes {
		// No is focused (default)
		noBtn = lipgloss.NewStyle().Bold(true).Reverse(true).Render("[ No ]")
		yesBtn = lipgloss.NewStyle().Foreground(m.styles.FgDanger).Render("[ Yes ]")
	} else {
		// Yes is focused
		noBtn = lipgloss.NewStyle().Foreground(m.styles.FgNormal).Render("[ No ]")
		yesBtn = lipgloss.NewStyle().Bold(true).Reverse(true).Render("[ Yes ]")
	}

	// Button row with 2-space gap, centered
	buttonRow := noBtn + "  " + yesBtn
	btnWidth := lipgloss.Width(buttonRow)
	leftPad := 0
	if innerWidth > btnWidth {
		leftPad = (innerWidth - btnWidth) / 2
	}
	buttonRow = strings.Repeat(" ", leftPad) + buttonRow

	// Compose content
	content := strings.Join([]string{
		question,
		detail,
		"",
		buttonRow,
	}, "\n")

	// Modal border (same technique as renderHelpOverlay)
	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.BorderNormal).
		Width(overlayWidth-2).
		Padding(1, 2).
		Background(m.styles.BgModal).
		Render(content)

	// Add title to top border: "Kill Session" in danger color
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.FgDanger).
		Render(" Kill Session ")

	bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, bordered)
}

// renderJoinCodePromptModal renders the Phase 122 join-code prompt modal as
// a centered bordered overlay. The prompt sub-model owns the inner content
// (textinput, status line, hint line); this wrapper supplies the standard
// modal frame so visual style stays consistent with the kill-confirm and
// new-session modals.
func (m Model) renderJoinCodePromptModal() string {
	overlayWidth := max(50, min(70, m.width-10))
	innerWidth := overlayWidth - 6
	content := m.joinCodePrompt.View(innerWidth, 0)

	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.BorderAccent).
		Width(overlayWidth - 2).
		Padding(1, 2).
		Background(m.styles.BgModal).
		Render(content)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.FgAccent).
		Render(" Join Remote Session ")

	bordered = injectBorderTitle(bordered, title, m.styles.BorderAccent)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, bordered)
}
