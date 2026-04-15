package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

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

	// Add title to top border: "New Session" in accent color
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.BorderAccent).
		Render(" New Session ")

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
func (m Model) renderAgentPicker() string {
	if len(m.detectedCLIs) == 0 {
		return lipgloss.NewStyle().Foreground(m.styles.FgDanger).Render("(none found)")
	}

	name := m.detectedCLIs[m.agentIdx].DisplayName
	arrows := lipgloss.NewStyle().Foreground(m.styles.FgMuted)
	return fmt.Sprintf("%s %s %s", arrows.Render("<"), name, arrows.Render(">"))
}

// cycleAgent cycles the agent picker index forward or backward.
func (m Model) cycleAgent(forward bool) Model {
	if len(m.detectedCLIs) == 0 {
		return m
	}
	if forward {
		m.agentIdx = (m.agentIdx + 1) % len(m.detectedCLIs)
	} else {
		m.agentIdx = (m.agentIdx + len(m.detectedCLIs) - 1) % len(m.detectedCLIs)
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
