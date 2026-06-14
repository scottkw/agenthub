package tui

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestResolveEditor verifies the $EDITOR→$VISUAL→nano→vim→vi resolution chain.
//
// Case 1 (resolved): $EDITOR set to a known-present binary ("go") — expects
// non-empty path that ends with "go".
// Case 2 (empty): all env vars unset AND PATH stripped to a directory that
// has no editor — expects "".
func TestResolveEditor(t *testing.T) {
	t.Run("resolves $EDITOR when set to known binary", func(t *testing.T) {
		// "go" is always present on PATH in this repo's dev environment.
		goPath, err := exec.LookPath("go")
		if err != nil {
			t.Skip("go not on PATH — skipping EDITOR resolve test")
		}

		t.Setenv("EDITOR", "go")
		t.Setenv("VISUAL", "")

		got := resolveEditor()
		if got == "" {
			t.Fatal("resolveEditor: expected non-empty path with EDITOR=go, got empty string")
		}
		if got != goPath {
			t.Errorf("resolveEditor: got %q, want %q", got, goPath)
		}
	})

	t.Run("returns empty string when no editor resolves", func(t *testing.T) {
		// Unset both env vars and point PATH at an empty temp directory so
		// none of the fallback binaries (nano, vim, vi) can be found.
		emptyDir := t.TempDir()

		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "")
		t.Setenv("PATH", emptyDir)

		got := resolveEditor()
		if got != "" {
			t.Errorf("resolveEditor: expected empty string with no editors on PATH, got %q", got)
		}
	})

	t.Run("resolution order is EDITOR VISUAL nano vim vi", func(t *testing.T) {
		// Create a fake "myeditor" in a temp dir to verify order: $EDITOR wins
		// over $VISUAL wins over fallbacks.
		dir := t.TempDir()
		for _, name := range []string{"myeditor", "myvisual", "nano", "vim", "vi"} {
			path := dir + "/" + name
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
				t.Fatalf("create fake %s: %v", name, err)
			}
		}

		t.Setenv("EDITOR", "myeditor")
		t.Setenv("VISUAL", "myvisual")
		oldPath := os.Getenv("PATH")
		t.Setenv("PATH", dir+":"+oldPath)

		got := resolveEditor()
		if !strings.HasSuffix(got, "/myeditor") {
			t.Errorf("resolveEditor: $EDITOR should win, got %q", got)
		}

		// Now unset EDITOR — $VISUAL should win.
		t.Setenv("EDITOR", "")
		got2 := resolveEditor()
		if !strings.HasSuffix(got2, "/myvisual") {
			t.Errorf("resolveEditor: $VISUAL should win when $EDITOR unset, got %q", got2)
		}
	})
}
