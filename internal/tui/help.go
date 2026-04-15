package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderHelpOverlay renders the centered bordered help modal.
func (m Model) renderHelpOverlay() string {
	content := m.buildHelpContent()

	overlayWidth := max(40, min(60, m.width-10))

	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.BorderNormal).
		Width(overlayWidth-2). // subtract border width
		Padding(1, 2).
		Render(content)

	// Add title to top border
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.BorderAccent).
		Render(" Keybindings ")

	// Replace first line's center with title
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		topBorder := lines[0]
		titleWidth := lipgloss.Width(title)
		borderWidth := lipgloss.Width(topBorder)
		if borderWidth > titleWidth+4 {
			insertPos := 3 // after "---"
			runes := []rune(topBorder)
			titleRunes := []rune(title)
			copy(runes[insertPos:insertPos+len(titleRunes)], titleRunes)
			lines[0] = string(runes)
		}
		bordered = strings.Join(lines, "\n")
	}

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, bordered)
}

// buildHelpContent builds the formatted help text with keybinding groups.
func (m Model) buildHelpContent() string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgAccent)
	descStyle := lipgloss.NewStyle().Foreground(m.styles.FgMuted)
	groupStyle := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgNormal)

	formatBinding := func(k, desc string) string {
		return fmt.Sprintf("  %s  %s",
			keyStyle.Render(fmt.Sprintf("%-16s", k)),
			descStyle.Render(desc))
	}

	var sections []string

	// Group 1: Navigation
	sections = append(sections, groupStyle.Render("Navigation"))
	sections = append(sections,
		formatBinding("j/k, Up/Down", "Move up/down"),
		formatBinding("g/Home", "Jump to first"),
		formatBinding("G/End", "Jump to last"),
		formatBinding("R", "Refresh list"),
	)

	// Group 2: Sessions
	sections = append(sections, "")
	sections = append(sections, groupStyle.Render("Sessions"))
	sections = append(sections,
		formatBinding("Enter", "Attach to session"),
		formatBinding("n", "New session"),
		formatBinding("d", "Kill session"),
		formatBinding("r", "Rename session"),
	)

	// Group 3: General
	sections = append(sections, "")
	sections = append(sections, groupStyle.Render("General"))
	sections = append(sections,
		formatBinding("?", "Toggle help"),
		formatBinding("q, Ctrl+C", "Quit"),
	)

	// Close hint
	sections = append(sections, "")
	closeHint := lipgloss.NewStyle().Foreground(m.styles.FgMuted).
		Align(lipgloss.Center).
		Render("Press ? or Esc to close")
	sections = append(sections, closeHint)

	return strings.Join(sections, "\n")
}
