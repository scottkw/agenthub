package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/scottkw/agenthub/internal/pty"
)

// agentEntry is a unified picker entry covering both AI CLIs and shells.
// cliKey is the value passed to daemon.CreateSession (e.g. "claude", "shell",
// "bash"). displayLabel is the human-readable string shown in the picker
// (e.g. "Claude Code", "Shell — system default", "Shell — bash").
//
// Phase 101 SHELL-03: introduced to merge detectedCLIs + detectedShells into
// a single index space that agentIdx walks.
type agentEntry struct {
	cliKey       string
	displayLabel string
}

// sortShellsForPicker returns a copy of shells sorted by the canonical UI-SPEC
// order: shell (system default) → bash → zsh → pwsh → powershell → unknown.
// Stable sort preserves caller order for entries with the same priority.
//
// Phase 101 SHELL-03 — mirrors the GUI Plan 02 sort so picker order is the
// same across all three surfaces (GUI modal, TUI picker, CLI --help).
func sortShellsForPicker(shells []pty.DetectedShell) []pty.DetectedShell {
	priority := map[string]int{
		"shell":      0,
		"bash":       1,
		"zsh":        2,
		"pwsh":       3,
		"powershell": 4,
	}
	out := append([]pty.DetectedShell(nil), shells...)
	sort.SliceStable(out, func(i, j int) bool {
		pi, oki := priority[out[i].Name]
		if !oki {
			pi = 99
		}
		pj, okj := priority[out[j].Name]
		if !okj {
			pj = 99
		}
		return pi < pj
	})
	return out
}

// agentEntries returns the unified picker entry slice: AI CLIs first (in
// detectedCLIs order), then shells (in sortShellsForPicker order). Shell
// entries get the locked "Shell — " prefix per UI-SPEC §Copywriting.
//
// Returns nil if both detectedCLIs and detectedShells are empty (caller
// checks len(entries) == 0 for the empty-state branch).
func (m Model) agentEntries() []agentEntry {
	if len(m.detectedCLIs) == 0 && len(m.detectedShells) == 0 {
		return nil
	}
	entries := make([]agentEntry, 0, len(m.detectedCLIs)+len(m.detectedShells))
	for _, c := range m.detectedCLIs {
		entries = append(entries, agentEntry{cliKey: c.Name, displayLabel: c.DisplayName})
	}
	for _, sh := range sortShellsForPicker(m.detectedShells) {
		entries = append(entries, agentEntry{
			cliKey:       sh.Name,
			displayLabel: "Shell — " + sh.DisplayName,
		})
	}
	return entries
}

// renderNewSessionModal renders the new-session modal as a centered bordered overlay.
func (m Model) renderNewSessionModal() string {
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
// Phase 101 SHELL-03: shows AI CLI labels followed by "Shell — <name>" entries
// for each discovered shell (per UI-SPEC §Interaction TUI flow).
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
// Phase 101 SHELL-03: walks the unified agentEntries slice (AI CLIs + shells).
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
