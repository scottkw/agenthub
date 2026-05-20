package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)

// TestTruncateLeft covers the documented behavior for the new
// left-truncating helper used by the Files status line.
func TestTruncateLeft(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
		// optional predicate checks
		startsWith string
		endsWith   string
		runeWidth  int // expected []rune width when > 0
	}{
		{name: "shorter than maxWidth returns unchanged", input: "short", maxWidth: 20, want: "short"},
		{name: "empty string returns empty", input: "", maxWidth: 10, want: ""},
		{
			name:       "long path is truncated with ellipsis prefix",
			input:      "a/b/c/d/utils/helper.ts",
			maxWidth:   18,
			startsWith: "…/",
			endsWith:   "helper.ts",
			runeWidth:  18,
		},
		{name: "tiny maxWidth returns rightmost runes without panic", input: "abc", maxWidth: 2, want: "bc"},
		{name: "zero maxWidth returns empty", input: "abc", maxWidth: 0, want: ""},
		{name: "negative maxWidth returns empty", input: "abc", maxWidth: -3, want: ""},
		{name: "multibyte runes counted by rune not byte", input: "αβγδεζηθικ", maxWidth: 4, runeWidth: 4, startsWith: "…/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateLeft(tc.input, tc.maxWidth)
			if tc.want != "" || (tc.startsWith == "" && tc.endsWith == "" && tc.runeWidth == 0) {
				if got != tc.want {
					t.Errorf("truncateLeft(%q, %d) = %q, want %q", tc.input, tc.maxWidth, got, tc.want)
				}
			}
			if tc.startsWith != "" && !strings.HasPrefix(got, tc.startsWith) {
				t.Errorf("truncateLeft(%q, %d) = %q, expected prefix %q", tc.input, tc.maxWidth, got, tc.startsWith)
			}
			if tc.endsWith != "" && !strings.HasSuffix(got, tc.endsWith) {
				t.Errorf("truncateLeft(%q, %d) = %q, expected suffix %q", tc.input, tc.maxWidth, got, tc.endsWith)
			}
			if tc.runeWidth > 0 {
				if w := len([]rune(got)); w != tc.runeWidth {
					t.Errorf("truncateLeft(%q, %d) = %q, rune-width %d, want %d", tc.input, tc.maxWidth, got, w, tc.runeWidth)
				}
			}
		})
	}
}

// TestFiles_TabID_Distinct asserts the new tabFiles iota value is distinct from
// every preceding tab ID.
func TestFiles_TabID_Distinct(t *testing.T) {
	if tabFiles == tabHome {
		t.Error("tabFiles must differ from tabHome")
	}
	if tabFiles == tabSessions {
		t.Error("tabFiles must differ from tabSessions")
	}
	if tabFiles == tabRemote {
		t.Error("tabFiles must differ from tabRemote")
	}
	if tabFiles == tabSettings {
		t.Error("tabFiles must differ from tabSettings")
	}
}

// TestFiles_KeyMap_BindsF asserts the FilesOpen binding catches the 'f' key.
func TestFiles_KeyMap_BindsF(t *testing.T) {
	km := defaultKeyMap()
	if !key.Matches(tea.KeyPressMsg{Code: 'f'}, km.FilesOpen) {
		t.Error("FilesOpen binding does not match 'f' key")
	}
	if !key.Matches(tea.KeyPressMsg{Code: '/'}, km.FilterStart) {
		t.Error("FilterStart binding does not match '/' key")
	}
	if len(km.FilesFocusToggle.Keys()) == 0 {
		t.Error("FilesFocusToggle binding has no keys configured")
	}
	if len(km.FilterEsc.Keys()) == 0 {
		t.Error("FilterEsc binding has no keys configured")
	}
}

// silence unused-import warnings if no daemon-using test exists in this file yet
var _ = daemon.SessionInfo{}
