package pty

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestDetectCLIs_FindsInstalledCLIs verifies that DetectCLIs returns a CLI
// when its binary is present on PATH.
func TestDetectCLIs_FindsInstalledCLIs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell script stubs not executable on Windows")
	}
	dir := t.TempDir()

	// Write a stub "claude" executable.
	stubPath := filepath.Join(dir, "claude")
	if err := os.WriteFile(stubPath, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}

	t.Setenv("PATH", dir)

	result := DetectCLIs()

	var found bool
	for _, cli := range result {
		if cli.Name == "claude" {
			found = true
			if cli.Path == "" {
				t.Error("expected non-empty Path for claude")
			}
		}
	}
	if !found {
		t.Error("expected DetectCLIs to find claude, but it was not in results")
	}
}

// TestDetectCLIs_SkipsMissing verifies that a CLI not on PATH is excluded.
func TestDetectCLIs_SkipsMissing(t *testing.T) {
	dir := t.TempDir()
	// No stubs placed — PATH contains only an empty temp dir.
	t.Setenv("PATH", dir)

	result := DetectCLIs()

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

// TestDetectCLIs_AllMissing verifies that when nothing is on PATH the returned
// slice is empty and non-nil.
func TestDetectCLIs_AllMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	result := DetectCLIs()

	if result == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected zero entries, got %d", len(result))
	}
}

// TestKnownCLIs_HasExpectedEntries verifies the package-level knownCLIs list
// contains exactly the expected CLI names.
func TestKnownCLIs_HasExpectedEntries(t *testing.T) {
	expected := []string{"claude", "codex", "gemini", "opencode"}

	if len(knownCLIs) != len(expected) {
		t.Fatalf("expected %d known CLIs, got %d", len(expected), len(knownCLIs))
	}

	nameSet := make(map[string]bool, len(knownCLIs))
	for _, spec := range knownCLIs {
		nameSet[spec.Name] = true
	}

	for _, name := range expected {
		if !nameSet[name] {
			t.Errorf("expected knownCLIs to contain %q", name)
		}
	}
}
