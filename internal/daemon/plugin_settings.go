package daemon

// CurrentSchemaVersion is the on-disk daemonSettings schema version.
// v3.1 settings.json files have no schemaVersion field (treated as 0);
// v3.2 introduces the plugins block and bumps to 2. Future v3.3+
// migrations bump and re-save via the same defaults-merge load path.
const CurrentSchemaVersion = 2

// PluginSettings is the persisted user choice for each xterm.js addon
// shipped by the v3.2 plugin suite. Field names match UI-SPEC ordering.
// JSON tags are camelCase to match daemonSettings vocabulary (cliPaths,
// startMinimized).
//
// NO omitempty: missing keys must round-trip as the user's saved choice
// (which may legitimately be `false`), and Pitfall #14 mandates the
// parent `plugins` key always serialize so future loads observe it.
type PluginSettings struct {
	WebGL     bool `json:"webgl"`
	Unicode11 bool `json:"unicode11"`
	Search    bool `json:"search"`
	WebLinks  bool `json:"webLinks"`
	Image     bool `json:"image"`
	Serialize bool `json:"serialize"`
	Clipboard bool `json:"clipboard"`
	Progress  bool `json:"progress"`
}

// defaultPluginSettings returns the v3.2 default plugin enable/disable
// state per ROADMAP `## Decisions` and UI-SPEC §"Default ON / OFF on
// first launch": all plugins ON except Progress (default OFF, flips
// ON in v3.3 after field validation).
//
// This function is the single source of truth for plugin defaults.
// Returning v3.1 users hit it via the defaults-merge in
// loadSettingsFromDisk (Pitfall #14 mitigation).
func defaultPluginSettings() PluginSettings {
	return PluginSettings{
		WebGL:     true,
		Unicode11: true,
		Search:    true,
		WebLinks:  true,
		Image:     true,
		Serialize: true,
		Clipboard: true,
		Progress:  false,
	}
}
