package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles holds all semantic color tokens for the TUI.
// Rebuilt on tea.BackgroundColorMsg via newStyles(hasDark).
type Styles struct {
	FgNormal      color.Color
	FgMuted       color.Color
	FgAccent      color.Color
	BgSelected    color.Color
	FgSelected    color.Color
	StatusRunning color.Color
	StatusIdle    color.Color
	StatusWaiting color.Color
	StatusErrored color.Color
	WebOn         color.Color
	WebOff        color.Color
	BorderNormal  color.Color
	BorderAccent  color.Color

	// Phase 77 modal + danger tokens
	BgModal        color.Color
	FgDanger       color.Color
	FgInput        color.Color
	BgInput        color.Color
	FgPlaceholder  color.Color
	FgFocusedLabel color.Color
}

// newStyles creates a Styles with adaptive colors for light or dark terminals.
// LightDark(hasDark) returns func(light, dark) -- light BG value first, dark BG value second.
func newStyles(hasDark bool) Styles {
	ld := lipgloss.LightDark(hasDark)
	return Styles{
		FgNormal:      ld(lipgloss.Color("#303030"), lipgloss.Color("#C6C6C6")),
		FgMuted:       ld(lipgloss.Color("#949494"), lipgloss.Color("#626262")),
		FgAccent:      ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),
		BgSelected:    ld(lipgloss.Color("#E4E4E4"), lipgloss.Color("#303030")),
		FgSelected:    ld(lipgloss.Color("#000000"), lipgloss.Color("#FFFFFF")),
		StatusRunning: ld(lipgloss.Color("#008700"), lipgloss.Color("#5FAF5F")),
		StatusIdle:    ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),
		StatusWaiting: ld(lipgloss.Color("#AF8700"), lipgloss.Color("#FFAF00")),
		StatusErrored: ld(lipgloss.Color("#D70000"), lipgloss.Color("#FF5F5F")),
		WebOn:         ld(lipgloss.Color("#008700"), lipgloss.Color("#5FAF5F")),
		WebOff:        ld(lipgloss.Color("#949494"), lipgloss.Color("#626262")),
		BorderNormal:  ld(lipgloss.Color("#BCBCBC"), lipgloss.Color("#444444")),
		BorderAccent:  ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),

		// Phase 77 modal + danger tokens
		BgModal:        ld(lipgloss.Color("#F5F5F5"), lipgloss.Color("#1C1C1C")),
		FgDanger:       ld(lipgloss.Color("#D70000"), lipgloss.Color("#FF5F5F")),
		FgInput:        ld(lipgloss.Color("#000000"), lipgloss.Color("#FFFFFF")),
		BgInput:        ld(lipgloss.Color("#E4E4E4"), lipgloss.Color("#303030")),
		FgPlaceholder:  ld(lipgloss.Color("#949494"), lipgloss.Color("#626262")),
		FgFocusedLabel: ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),
	}
}
