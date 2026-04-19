package tui

import (
	"image/color"
	"strings"

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

	// Phase 86 surface + sidebar + per-agent badge tokens
	BgSurface     color.Color
	BgSidebar     color.Color
	BadgeClaude   color.Color
	BadgeOpencode color.Color
	BadgeCodex    color.Color
	BadgeGemini   color.Color
	BadgeCursor   color.Color
	BadgeAider    color.Color
}

// newStyles creates a Styles with adaptive colors for light or dark terminals.
// LightDark(hasDark) returns func(light, dark) -- light BG value first, dark BG value second.
func newStyles(hasDark bool) Styles {
	ld := lipgloss.LightDark(hasDark)
	return Styles{
		FgNormal:      ld(lipgloss.Color("#3b4261"), lipgloss.Color("#c0caf5")),
		FgMuted:       ld(lipgloss.Color("#9699b0"), lipgloss.Color("#565f89")),
		FgAccent:      ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),
		BgSelected:    ld(lipgloss.Color("#d0d5e3"), lipgloss.Color("#283457")),
		FgSelected:    ld(lipgloss.Color("#3b4261"), lipgloss.Color("#c0caf5")),
		StatusRunning: ld(lipgloss.Color("#485e30"), lipgloss.Color("#9ece6a")),
		StatusIdle:    ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),
		StatusWaiting: ld(lipgloss.Color("#8c6c3e"), lipgloss.Color("#e0af68")),
		StatusErrored: ld(lipgloss.Color("#c64343"), lipgloss.Color("#f7768e")),
		WebOn:         ld(lipgloss.Color("#485e30"), lipgloss.Color("#9ece6a")),
		WebOff:        ld(lipgloss.Color("#9699b0"), lipgloss.Color("#565f89")),
		BorderNormal:  ld(lipgloss.Color("#c4c8da"), lipgloss.Color("#414868")),
		BorderAccent:  ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),

		// Phase 77 modal + danger tokens
		BgModal:        ld(lipgloss.Color("#e1e2e7"), lipgloss.Color("#1f2335")),
		FgDanger:       ld(lipgloss.Color("#c64343"), lipgloss.Color("#f7768e")),
		FgInput:        ld(lipgloss.Color("#3b4261"), lipgloss.Color("#c0caf5")),
		BgInput:        ld(lipgloss.Color("#d0d5e3"), lipgloss.Color("#283457")),
		FgPlaceholder:  ld(lipgloss.Color("#9699b0"), lipgloss.Color("#565f89")),
		FgFocusedLabel: ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),

		// Phase 86 surface + sidebar + per-agent badge tokens
		BgSurface:     ld(lipgloss.Color("#e9e9ec"), lipgloss.Color("#1a1b26")),
		BgSidebar:     ld(lipgloss.Color("#f0f0f4"), lipgloss.Color("#16161e")),
		BadgeClaude:   ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),
		BadgeOpencode: ld(lipgloss.Color("#485e30"), lipgloss.Color("#9ece6a")),
		BadgeCodex:    ld(lipgloss.Color("#7847bd"), lipgloss.Color("#bb9af7")),
		BadgeGemini:   ld(lipgloss.Color("#118c9e"), lipgloss.Color("#2ac3de")),
		BadgeCursor:   ld(lipgloss.Color("#8c6c3e"), lipgloss.Color("#e0af68")),
		BadgeAider:    ld(lipgloss.Color("#c64343"), lipgloss.Color("#f7768e")),
	}
}

// agentBadgeColor returns the badge color for the given CLI agent name.
// Falls back to FgMuted for unknown agent names.
func agentBadgeColor(cli string, s Styles) color.Color {
	switch strings.ToLower(cli) {
	case "claude":
		return s.BadgeClaude
	case "opencode":
		return s.BadgeOpencode
	case "codex":
		return s.BadgeCodex
	case "gemini":
		return s.BadgeGemini
	case "cursor":
		return s.BadgeCursor
	case "aider":
		return s.BadgeAider
	default:
		return s.FgMuted
	}
}
