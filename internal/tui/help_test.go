package tui

import (
	"strings"
	"testing"
)

func TestHelpOverlay_ContainsGroups(t *testing.T) {
	m := testModel()
	m.showHelp = true
	m.sessions = nil

	content := m.buildHelpContent()

	groups := []string{"Navigation", "Actions", "General"}
	for _, group := range groups {
		if !strings.Contains(content, group) {
			t.Errorf("help content missing group %q", group)
		}
	}
}

func TestHelpOverlay_ContainsBindings(t *testing.T) {
	m := testModel()

	content := m.buildHelpContent()

	bindings := []string{
		"Move up/down",
		"Jump to first",
		"Jump to last",
		"Refresh list",
		"Attach to session",
		"New session",
		"Toggle help",
		"Quit",
	}
	for _, binding := range bindings {
		if !strings.Contains(content, binding) {
			t.Errorf("help content missing binding description %q", binding)
		}
	}
}

func TestHelpOverlay_ContainsCloseHint(t *testing.T) {
	m := testModel()

	content := m.buildHelpContent()

	if !strings.Contains(content, "Press ? or Esc to close") {
		t.Error("help content missing close hint")
	}
}

func TestHelpOverlay_NoReservedHiddenKeys(t *testing.T) {
	m := testModel()

	content := m.buildHelpContent()

	// d and e should NOT appear in help (reserved and hidden per UI-SPEC)
	// The safest check: neither "Kill" nor "Rename" should appear
	if strings.Contains(content, "Kill") {
		t.Error("help should not contain Kill action (Phase 77)")
	}
	if strings.Contains(content, "Rename") {
		t.Error("help should not contain Rename action (Phase 77)")
	}
}

func TestHelpOverlay_Rendered(t *testing.T) {
	m := testModel()
	m.showHelp = true

	rendered := m.renderHelpOverlay()

	if !strings.Contains(rendered, "Keybindings") {
		t.Error("rendered help overlay missing title 'Keybindings'")
	}
	if !strings.Contains(rendered, "Navigation") {
		t.Error("rendered help overlay missing 'Navigation' group")
	}
}
