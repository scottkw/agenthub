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
		{"Claude", s.BadgeClaude}, // case-insensitive
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

// TestAgentBadgeColor_Shell verifies the Phase 101 SHELL-06 TUI half: agentBadgeColor
// returns the new BadgeShell color for every shell cli identifier (shell + 4 known specs).
// Each cli string MUST map to s.BadgeShell — NOT fall through to FgMuted.
func TestAgentBadgeColor_Shell(t *testing.T) {
	s := newStyles(true)
	for _, cli := range []string{"shell", "bash", "zsh", "pwsh", "powershell"} {
		got := agentBadgeColor(cli, s)
		if got != s.BadgeShell {
			t.Errorf("agentBadgeColor(%q) = %v, want s.BadgeShell %v", cli, got, s.BadgeShell)
		}
	}
}

// TestAgentBadgeColor_AICLIs_Unchanged regression-locks that the 6 AI CLI agents still
// resolve to their respective BadgeXxx colors after the shell case is added.
func TestAgentBadgeColor_AICLIs_Unchanged(t *testing.T) {
	s := newStyles(true)
	tests := []struct {
		cli  string
		want color.Color
	}{
		{"claude", s.BadgeClaude},
		{"opencode", s.BadgeOpencode},
		{"codex", s.BadgeCodex},
		{"gemini", s.BadgeGemini},
		{"cursor", s.BadgeCursor},
		{"aider", s.BadgeAider},
	}
	for _, tt := range tests {
		if got := agentBadgeColor(tt.cli, s); got != tt.want {
			t.Errorf("agentBadgeColor(%q) = %v, want %v", tt.cli, got, tt.want)
		}
	}
}

// TestAgentBadgeColor_Unknown regression-locks that unknown cli strings still fall
// through to FgMuted (existing default).
func TestAgentBadgeColor_Unknown(t *testing.T) {
	s := newStyles(true)
	if got := agentBadgeColor("unknown-tool", s); got != s.FgMuted {
		t.Errorf("agentBadgeColor(\"unknown-tool\") = %v, want s.FgMuted %v", got, s.FgMuted)
	}
}

// TestBadgeShell_LightDark_Variants verifies the BadgeShell color uses the locked
// TokyoNight slate-cyan adaptive palette: #3d5a80 (light) / #89ddff (dark).
// Cross-checks that the color is distinct from all six existing badge colors and
// resolves to the literal lipgloss.Color hex per UI-SPEC §Color.
func TestBadgeShell_LightDark_Variants(t *testing.T) {
	// Dark mode: BadgeShell must equal lipgloss.Color("#89ddff").
	dark := newStyles(true)
	wantDark := lipgloss.Color("#89ddff")
	if dark.BadgeShell != wantDark {
		t.Errorf("dark BadgeShell = %v, want %v (#89ddff)", dark.BadgeShell, wantDark)
	}
	// Light mode: BadgeShell must equal lipgloss.Color("#3d5a80").
	light := newStyles(false)
	wantLight := lipgloss.Color("#3d5a80")
	if light.BadgeShell != wantLight {
		t.Errorf("light BadgeShell = %v, want %v (#3d5a80)", light.BadgeShell, wantLight)
	}
	// Cross-check: distinct from all 6 existing badges in both modes.
	others := []color.Color{
		dark.BadgeClaude, dark.BadgeOpencode, dark.BadgeCodex,
		dark.BadgeGemini, dark.BadgeCursor, dark.BadgeAider,
	}
	for i, other := range others {
		if dark.BadgeShell == other {
			t.Errorf("dark BadgeShell collides with existing badge[%d]: %v", i, other)
		}
	}
}
