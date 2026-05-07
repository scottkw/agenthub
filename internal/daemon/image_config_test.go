package daemon

import "testing"

// TestSetImageConfigPreservesSiblings — Plan 96-02 implements.
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
	t.Skip("Pending until Plan 96-02 implements engine.SetImageConfig sub-key writer (96-VALIDATION row IMG-02 SetImageConfig persists ONLY ImageConfig sub-key).")
}
