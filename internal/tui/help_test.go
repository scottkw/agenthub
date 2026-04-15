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

	groups := []string{"Navigation", "Sessions", "General"}
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
		"QR code / URL",
		"New session",
		"Kill session",
		"Rename session",
		"Toggle help",
		"Quit",
	}
	for _, binding := range bindings {
		if !strings.Contains(content, binding) {
			t.Errorf("help content missing binding description %q", binding)
		}
	}
}

func TestHelp_QRBinding(t *testing.T) {
	m := testModel()
	content := m.buildHelpContent()
	if !strings.Contains(content, "QR code / URL") {
		t.Error("help missing 'QR code / URL' binding")
	}
	if !strings.Contains(content, "Q, Ctrl+C") {
		t.Error("help missing 'Q, Ctrl+C' quit binding")
	}
	// Old quit binding should not be present
	if strings.Contains(content, "q, Ctrl+C") {
		t.Error("help still contains old 'q, Ctrl+C' quit binding")
	}
}

func TestHelpOverlay_ContainsCloseHint(t *testing.T) {
	m := testModel()

	content := m.buildHelpContent()

	if !strings.Contains(content, "Press ? or Esc to close") {
		t.Error("help content missing close hint")
	}
}

func TestHelp_UpdatedBindings(t *testing.T) {
	m := testModel()
	content := m.buildHelpContent()

	required := []string{"Kill session", "Rename session", "Sessions"}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Errorf("help content missing %q", want)
		}
	}
	if strings.Contains(content, "Actions") {
		t.Error("help should use 'Sessions' group, not 'Actions'")
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
