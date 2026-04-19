package tui

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestStyles_TokyoNight(t *testing.T) {
	// Dark mode
	s := newStyles(true)
	darkChecks := map[string]color.Color{
		"FgAccent":      lipgloss.Color("#7aa2f7"),
		"FgMuted":       lipgloss.Color("#565f89"),
		"StatusRunning": lipgloss.Color("#9ece6a"),
		"BorderNormal":  lipgloss.Color("#414868"),
		"BgSidebar":     lipgloss.Color("#16161e"),
		"BgSurface":     lipgloss.Color("#1a1b26"),
		"BadgeClaude":   lipgloss.Color("#7aa2f7"),
		"BadgeCodex":    lipgloss.Color("#bb9af7"),
	}
	vals := map[string]color.Color{
		"FgAccent":      s.FgAccent,
		"FgMuted":       s.FgMuted,
		"StatusRunning": s.StatusRunning,
		"BorderNormal":  s.BorderNormal,
		"BgSidebar":     s.BgSidebar,
		"BgSurface":     s.BgSurface,
		"BadgeClaude":   s.BadgeClaude,
		"BadgeCodex":    s.BadgeCodex,
	}
	for name, want := range darkChecks {
		got := vals[name]
		if got != want {
			t.Errorf("dark %s = %v, want %v", name, got, want)
		}
	}

	// Light mode
	sl := newStyles(false)
	if sl.FgAccent != lipgloss.Color("#2e7de9") {
		t.Errorf("light FgAccent = %v, want #2e7de9", sl.FgAccent)
	}
	if sl.BgSidebar != lipgloss.Color("#f0f0f4") {
		t.Errorf("light BgSidebar = %v, want #f0f0f4", sl.BgSidebar)
	}
}

func TestAgentBadgeColor(t *testing.T) {
	s := newStyles(true)
	tests := []struct {
		cli  string
		want color.Color
	}{
		{"claude", s.BadgeClaude},
		{"Claude", s.BadgeClaude},   // case-insensitive
		{"opencode", s.BadgeOpencode},
		{"codex", s.BadgeCodex},
		{"gemini", s.BadgeGemini},
		{"cursor", s.BadgeCursor},
		{"aider", s.BadgeAider},
		{"unknown", s.FgMuted}, // fallback
		{"", s.FgMuted},        // empty
	}
	for _, tt := range tests {
		got := agentBadgeColor(tt.cli, s)
		if got != tt.want {
			t.Errorf("agentBadgeColor(%q) = %v, want %v", tt.cli, got, tt.want)
		}
	}
}

func TestStyles_AllBadgeColorsDistinct(t *testing.T) {
	s := newStyles(true)
	badges := []color.Color{
		s.BadgeClaude, s.BadgeOpencode, s.BadgeCodex,
		s.BadgeGemini, s.BadgeCursor, s.BadgeAider,
	}
	seen := make(map[color.Color]bool)
	for i, b := range badges {
		if seen[b] {
			t.Errorf("badge[%d] color %v is a duplicate", i, b)
		}
		seen[b] = true
	}
}
