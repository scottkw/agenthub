package daemon

import (
	"testing"
)

// TestSetPluginSettingsRoundTrip validates the engine-level Set/Get round-trip
// AND the disk persistence reload-engine pattern (mirrors TestStartMinimizedPersistence
// at engine_settings_test.go:101-148).
func TestSetPluginSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	e := &SessionEngine{
		configDir:      dir,
		cliPaths:       make(map[string]string),
		pluginSettings: defaultPluginSettings(),
	}

	// Flip every plugin: defaults are 7-ON-1-OFF; flipped is 7-OFF-1-ON.
	flipped := PluginSettings{
		WebGL:     false,
		Unicode11: false,
		Search:    false,
		WebLinks:  false,
		Image:     false,
		Serialize: false,
		Clipboard: false,
		Progress:  true,
	}
	e.SetPluginSettings(flipped)

	got := e.GetPluginSettings()
	if got != flipped {
		t.Errorf("GetPluginSettings after Set: got %+v, want %+v", got, flipped)
	}

	// Reload-engine round-trip: a fresh engine constructed against the same
	// configDir must observe the persisted values via loadSettingsFromDisk.
	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir)

	got2 := e2.GetPluginSettings()
	if got2 != flipped {
		t.Errorf("reloaded GetPluginSettings: got %+v, want %+v", got2, flipped)
	}
}
