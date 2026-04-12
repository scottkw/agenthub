//go:build linux

package main

import (
	_ "embed"
	"testing"
)

//go:embed assets/tray_icon.png
var testTrayIconBytes []byte

//go:embed assets/tray_icon_error.png
var testTrayIconErrorBytes []byte

// TestPngToARGB32Pixmap verifies that the connected tray icon PNG decodes
// correctly to an ARGB32 pixmap with the expected dimensions and pixel count.
func TestPngToARGB32Pixmap(t *testing.T) {
	w, h, pixels, err := pngToARGB32Pixmap(testTrayIconBytes)
	if err != nil {
		t.Fatalf("pngToARGB32Pixmap failed: %v", err)
	}
	if w != 18 {
		t.Errorf("width = %d, want 18", w)
	}
	if h != 18 {
		t.Errorf("height = %d, want 18", h)
	}
	wantLen := int(w) * int(h) * 4 // ARGB32 = 4 bytes per pixel
	if len(pixels) != wantLen {
		t.Errorf("pixel data length = %d, want %d (18*18*4)", len(pixels), wantLen)
	}
}

// TestPngToARGB32PixmapInvalid verifies that invalid PNG bytes return an error.
func TestPngToARGB32PixmapInvalid(t *testing.T) {
	_, _, _, err := pngToARGB32Pixmap([]byte("not a png"))
	if err == nil {
		t.Error("expected error for invalid PNG bytes, got nil")
	}
}

// TestDbusMenuLayout verifies that a D-Bus menu layout built from 2 sessions
// has the correct child count, sequential IDs, and correct item types.
func TestDbusMenuLayout(t *testing.T) {
	sessions := []SessionInfo{
		{ID: "sess-1", Name: "Alpha"},
		{ID: "sess-2", Name: "Beta"},
	}
	menuItems := BuildMenuItems(sessions)
	layout := buildDbusMenuLayout(menuItems)

	// With 2 sessions: Open AgentHub, sep, Alpha, Beta, sep, Quit = 6 items
	children, ok := layout.props["children-display"]
	_ = children
	_ = ok

	if len(layout.children) != 6 {
		t.Fatalf("expected 6 children, got %d", len(layout.children))
	}

	// Verify sequential IDs starting from 1
	for i, child := range layout.children {
		wantID := int32(i + 1)
		if child.id != wantID {
			t.Errorf("child[%d].id = %d, want %d", i, child.id, wantID)
		}
	}

	// Verify separator items have type "separator"
	// items[1] and items[4] should be separators
	for _, sepIdx := range []int{1, 4} {
		child := layout.children[sepIdx]
		typ, hasType := child.props["type"]
		if !hasType {
			t.Errorf("separator child[%d] missing 'type' property", sepIdx)
			continue
		}
		if typ != "separator" {
			t.Errorf("child[%d].props[type] = %q, want separator", sepIdx, typ)
		}
	}

	// Verify action items have label property
	// items[0] = Open AgentHub, items[2] = Alpha, items[3] = Beta, items[5] = Quit
	wantLabels := map[int]string{0: "Open AgentHub", 2: "Alpha", 3: "Beta", 5: "Quit"}
	for idx, label := range wantLabels {
		got, ok := layout.children[idx].props["label"]
		if !ok {
			t.Errorf("child[%d] missing 'label' property", idx)
			continue
		}
		if got != label {
			t.Errorf("child[%d].props[label] = %q, want %q", idx, got, label)
		}
	}
}

// TestDbusMenuLayoutEmpty verifies that a D-Bus menu with no sessions has
// 4 children (Open AgentHub, separator, separator, Quit).
func TestDbusMenuLayoutEmpty(t *testing.T) {
	menuItems := BuildMenuItems(nil)
	layout := buildDbusMenuLayout(menuItems)

	if len(layout.children) != 4 {
		t.Fatalf("expected 4 children with no sessions, got %d", len(layout.children))
	}

	// First item: Open AgentHub
	label, ok := layout.children[0].props["label"]
	if !ok || label != "Open AgentHub" {
		t.Errorf("children[0].label = %q, want Open AgentHub", label)
	}

	// Last item: Quit
	last := layout.children[len(layout.children)-1]
	label, ok = last.props["label"]
	if !ok || label != "Quit" {
		t.Errorf("last child.label = %q, want Quit", label)
	}
}
