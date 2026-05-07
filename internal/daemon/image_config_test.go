package daemon

import (
	"testing"
)

// TestSetImageConfigPreservesSiblings — Phase 96 IMG-02.
//  1. ImageConfig sub-key updates round-trip through GetPluginSettings.
//  2. All other PluginSettings fields are PRESERVED (T-96-02-04 mitigation).
//  3. The pluginSettingsListener fires exactly once (T-96-02-05 mitigation).
//  4. The change persists to disk (reload reflects the new ImageConfig).
//
// Mirrors search_config_test.go TestSetSearchConfig and
// web_links_config_test.go TestSetWebLinksConfigPreservesSiblings:
// write an ImageConfig with non-default StorageLimit (e.g. 32);
// assert all OTHER PluginSettings fields (WebGL/Unicode11/Search/
// SearchConfig/WebLinks/WebLinksConfig/Image bool/Serialize/Clipboard/
// Progress) are preserved; assert listener fires exactly once.
//
// Phase 96 IMG-02 sub-key writer concurrency contract per
// 96-PATTERNS.md §"Sub-key writer concurrency contract".
func TestSetImageConfigPreservesSiblings(t *testing.T) {
	dir := t.TempDir()

	// Baseline PluginSettings: every non-image-config field set to a
	// non-default sentinel value so any stomp by SetImageConfig is
	// detectable. ImageConfig itself starts at StorageLimit=99 (the OLD
	// value) so we can detect the post-call diff to 32 (the NEW value).
	baseline := PluginSettings{
		WebGL:        false,
		Unicode11:    false,
		Search:       true,
		SearchConfig: SearchConfig{Regex: true, CaseSensitive: true, WholeWord: true},
		WebLinks:     false,
		WebLinksConfig: WebLinksConfig{
			Modifier:         "cmd",
			ConfirmOSC8:      false,
			ConfirmIDN:       false,
			ConfirmTyposquat: false,
		},
		Image:       false,
		ImageConfig: ImageConfig{StorageLimit: 99},
		Serialize:   false,
		Clipboard:   false,
		Progress:    true,
	}

	e := &SessionEngine{
		configDir:      dir,
		cliPaths:       make(map[string]string),
		pluginSettings: baseline,
	}

	// Listener counter — SetImageConfig must invoke it exactly once.
	var listenerCount int
	e.SetPluginSettingsListener(func() { listenerCount++ })

	// Apply the image-config sub-key change.
	newCfg := ImageConfig{StorageLimit: 32}
	e.SetImageConfig(newCfg)

	// Listener fired exactly once (T-96-02-05).
	if listenerCount != 1 {
		t.Errorf("listener invocations: got %d, want 1", listenerCount)
	}

	got := e.GetPluginSettings()

	// ImageConfig sub-key updated.
	if got.ImageConfig != newCfg {
		t.Errorf("ImageConfig: got %+v, want %+v", got.ImageConfig, newCfg)
	}
	if got.ImageConfig.StorageLimit != 32 {
		t.Errorf("ImageConfig.StorageLimit: got %d, want 32", got.ImageConfig.StorageLimit)
	}

	// Every other field preserved (T-96-02-04 mitigation — no full-PluginSettings stomp).
	if got.WebGL != baseline.WebGL {
		t.Errorf("WebGL stomped: got %v, want %v", got.WebGL, baseline.WebGL)
	}
	if got.Unicode11 != baseline.Unicode11 {
		t.Errorf("Unicode11 stomped: got %v, want %v", got.Unicode11, baseline.Unicode11)
	}
	if got.Search != baseline.Search {
		t.Errorf("Search stomped: got %v, want %v", got.Search, baseline.Search)
	}
	if got.SearchConfig != baseline.SearchConfig {
		t.Errorf("SearchConfig stomped: got %+v, want %+v", got.SearchConfig, baseline.SearchConfig)
	}
	if got.WebLinks != baseline.WebLinks {
		t.Errorf("WebLinks (boolean) stomped: got %v, want %v", got.WebLinks, baseline.WebLinks)
	}
	if got.WebLinksConfig != baseline.WebLinksConfig {
		t.Errorf("WebLinksConfig stomped: got %+v, want %+v", got.WebLinksConfig, baseline.WebLinksConfig)
	}
	if got.Image != baseline.Image {
		t.Errorf("Image (boolean) stomped: got %v, want %v", got.Image, baseline.Image)
	}
	if got.Serialize != baseline.Serialize {
		t.Errorf("Serialize stomped: got %v, want %v", got.Serialize, baseline.Serialize)
	}
	if got.Clipboard != baseline.Clipboard {
		t.Errorf("Clipboard stomped: got %v, want %v", got.Clipboard, baseline.Clipboard)
	}
	if got.Progress != baseline.Progress {
		t.Errorf("Progress stomped: got %v, want %v", got.Progress, baseline.Progress)
	}

	// Persistence: reload via a fresh engine pointed at the same configDir.
	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir)

	got2 := e2.GetPluginSettings()
	if got2.ImageConfig != newCfg {
		t.Errorf("reloaded ImageConfig: got %+v, want %+v", got2.ImageConfig, newCfg)
	}
	// Spot-check that the non-image-config fields also persisted (defaults-merge
	// could otherwise mask a stomp at the on-disk JSON layer).
	if got2.SearchConfig != baseline.SearchConfig {
		t.Errorf("reloaded SearchConfig: got %+v, want %+v", got2.SearchConfig, baseline.SearchConfig)
	}
	if got2.WebLinksConfig != baseline.WebLinksConfig {
		t.Errorf("reloaded WebLinksConfig: got %+v, want %+v", got2.WebLinksConfig, baseline.WebLinksConfig)
	}
	if got2.Progress != baseline.Progress {
		t.Errorf("reloaded Progress: got %v, want %v", got2.Progress, baseline.Progress)
	}
}
