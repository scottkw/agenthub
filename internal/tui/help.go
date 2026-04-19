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

	// Inject title into top border using ANSI-safe helper.
	bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)

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
		formatBinding("Tab", "Toggle sidebar/content"),
		formatBinding("[/]", "Cycle tabs"),
	)

	// Group 2: Sessions
	sections = append(sections, "")
	sections = append(sections, groupStyle.Render("Sessions"))
	sections = append(sections,
		formatBinding("Enter", "Attach to session"),
		formatBinding("q", "QR code / URL"),
		formatBinding("n", "New session"),
		formatBinding("d", "Kill session"),
		formatBinding("r", "Rename session"),
	)

	// Group 3: General
	sections = append(sections, "")
	sections = append(sections, groupStyle.Render("General"))
	sections = append(sections,
		formatBinding("?", "Toggle help"),
		formatBinding("Q, Ctrl+C", "Quit"),
	)

	// Close hint
	sections = append(sections, "")
	closeHint := lipgloss.NewStyle().Foreground(m.styles.FgMuted).
		Align(lipgloss.Center).
		Render("Press ? or Esc to close")
	sections = append(sections, closeHint)

	return strings.Join(sections, "\n")
}
