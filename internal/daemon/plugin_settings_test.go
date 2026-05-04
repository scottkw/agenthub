package daemon

import "testing"

// TestDefaultPluginSettings asserts the v3.2 default plugin enable/disable state
// per ROADMAP `## Decisions` and UI-SPEC §"Default ON / OFF on first launch":
// all plugins default ON except Progress (default OFF, flips ON in v3.3).
//
// Each field is asserted individually so a future regression points exactly at
// the broken default rather than failing on a single struct-equality check.
func TestDefaultPluginSettings(t *testing.T) {
	s := defaultPluginSettings()
	if !s.WebGL {
		t.Error("expected WebGL=true (UI-SPEC default ON)")
	}
	if !s.Unicode11 {
		t.Error("expected Unicode11=true (UI-SPEC default ON)")
	}
	if !s.Search {
		t.Error("expected Search=true (UI-SPEC default ON)")
	}
	if !s.WebLinks {
		t.Error("expected WebLinks=true (UI-SPEC default ON)")
	}
	if !s.Image {
		t.Error("expected Image=true (UI-SPEC default ON)")
	}
	if !s.Serialize {
		t.Error("expected Serialize=true (UI-SPEC default ON)")
	}
	if !s.Clipboard {
		t.Error("expected Clipboard=true (UI-SPEC default ON)")
	}
	if s.Progress {
		t.Error("expected Progress=false (UI-SPEC default OFF; flips ON in v3.3 per ROADMAP)")
	}
}
