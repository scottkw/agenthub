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
	if s.SearchConfig.Regex {
		t.Error("expected SearchConfig.Regex=false (Phase 94 SRC-02 default OFF per UI-SPEC §\"Toggle Persistence\")")
	}
	if s.SearchConfig.CaseSensitive {
		t.Error("expected SearchConfig.CaseSensitive=false (Phase 94 SRC-02 default OFF)")
	}
	if s.SearchConfig.WholeWord {
		t.Error("expected SearchConfig.WholeWord=false (Phase 94 SRC-02 default OFF)")
	}
	// Phase 95 LNK-02/LNK-03/LNK-05: WebLinksConfig defaults are
	// platform-correct + ALL confirmations ON (security-first posture).
	if got := s.WebLinksConfig.Modifier; got != "platform" {
		t.Errorf("WebLinksConfig.Modifier = %q, want \"platform\"", got)
	}
	if !s.WebLinksConfig.ConfirmOSC8 {
		t.Error("WebLinksConfig.ConfirmOSC8 should default true")
	}
	if !s.WebLinksConfig.ConfirmIDN {
		t.Error("WebLinksConfig.ConfirmIDN should default true")
	}
	if !s.WebLinksConfig.ConfirmTyposquat {
		t.Error("WebLinksConfig.ConfirmTyposquat should default true")
	}
}
