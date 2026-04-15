package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

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

	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		topBorder := lines[0]
		titleWidth := lipgloss.Width(title)
		borderWidth := lipgloss.Width(topBorder)
		if borderWidth > titleWidth+4 {
			insertPos := 3
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
