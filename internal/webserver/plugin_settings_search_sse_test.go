package webserver

import "testing"

// Phase 94 Wave 0 RED scaffold — Plan 94-02 implements (SRC-02 SSE broadcast).
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md row 02-daemon wave 1.
// RESEARCH §"SSE broadcast covered automatically by pluginSettingsProvider indirection".

func TestPluginSettingsSSE_Search(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-02 verifies SSE 'settings:plugins' event payload includes " +
		"nested searchConfig{regex,caseSensitive,wholeWord} after a daemon SetPluginSettings call. " +
		"Reuses Phase 93 plugin_config_stream pluginSettingsProvider func() []byte indirection — " +
		"no SSE infrastructure changes required. See 94-VALIDATION.md row 02-daemon wave 1.")
}
