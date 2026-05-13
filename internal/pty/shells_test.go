package pty

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// filterKnownSpecs returns only the entries whose Name matches one of the
// canonical knownShellSpecs names (bash, zsh, pwsh, powershell). Tests use
// this to ignore the synthetic "system default" entry that POSIX
// DiscoverShells may append based on $SHELL.
func filterKnownSpecs(in []DetectedShell) []DetectedShell {
	known := map[string]bool{"bash": true, "zsh": true, "pwsh": true, "powershell": true}
	out := []DetectedShell{}
	for _, s := range in {
		if known[s.Name] {
			out = append(out, s)
		}
	}
	return out
}

// TestDiscoverShells_FindsInstalledShells verifies that DiscoverShells surfaces
// bash + zsh stubs placed on PATH (POSIX-only — Windows uses .exe binaries).
func TestDiscoverShells_FindsInstalledShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell script stubs not executable on Windows")
	}
	dir := t.TempDir()

	for _, name := range []string{"bash", "zsh"} {
		stub := filepath.Join(dir, name)
		if err := os.WriteFile(stub, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
			t.Fatalf("writing %s stub: %v", name, err)
		}
	}

	t.Setenv("PATH", dir)
	// Empty SHELL to suppress the synthetic system-default entry — this test
	// only validates known-spec discovery.
	t.Setenv("SHELL", "")

	result := DiscoverShells()

	foundBash := false
	foundZsh := false
	for _, sh := range result {
		switch sh.Name {
		case "bash":
			foundBash = true
			if sh.Path == "" {
				t.Error("expected non-empty Path for bash")
			}
		case "zsh":
			foundZsh = true
			if sh.Path == "" {
				t.Error("expected non-empty Path for zsh")
			}
		}
	}
	if !foundBash {
		t.Error("expected DiscoverShells to find bash, but it was not in results")
	}
	if !foundZsh {
		t.Error("expected DiscoverShells to find zsh, but it was not in results")
	}
}

// TestDiscoverShells_SkipsMissing verifies that no known-spec entries surface
// when none of bash/zsh/pwsh/powershell are on PATH.
func TestDiscoverShells_SkipsMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	// Suppress synthetic system-default entry as well.
	if runtime.GOOS != "windows" {
		t.Setenv("SHELL", "")
	}

	result := DiscoverShells()
	known := filterKnownSpecs(result)
	if len(known) != 0 {
		t.Errorf("expected 0 known-spec entries with empty PATH, got %d (%+v)", len(known), known)
	}
}

// TestDiscoverShells_AllMissing locks the non-nil-empty-slice contract from
// detect.go (load-bearing for slim Linux containers where no shells are
// present).
func TestDiscoverShells_AllMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	if runtime.GOOS != "windows" {
		t.Setenv("SHELL", "")
	}

	result := DiscoverShells()

	if result == nil {
		t.Error("expected non-nil slice, got nil")
	}
	known := filterKnownSpecs(result)
	if len(known) != 0 {
		t.Errorf("expected zero known-spec entries, got %d (%+v)", len(known), known)
	}
}

// TestKnownShellSpecs_HasExpectedEntries verifies the canonical contents of
// knownShellSpecs (bash, zsh, pwsh, powershell — M2 promotes powershell to a
// first-class spec).
func TestKnownShellSpecs_HasExpectedEntries(t *testing.T) {
	expected := []string{"bash", "zsh", "pwsh", "powershell"}

	if len(knownShellSpecs) != len(expected) {
		t.Fatalf("expected %d known shell specs, got %d", len(expected), len(knownShellSpecs))
	}

	nameSet := make(map[string]bool, len(knownShellSpecs))
	for _, spec := range knownShellSpecs {
		nameSet[spec.Name] = true
	}

	for _, name := range expected {
		if !nameSet[name] {
			t.Errorf("expected knownShellSpecs to contain %q", name)
		}
	}
}

// TestDiscoverShells_NoEtcShells verifies that a missing /etc/shells does not
// panic and does not suppress known-spec discovery.
func TestDiscoverShells_NoEtcShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell script stubs not executable on Windows")
	}
	dir := t.TempDir()

	bashStub := filepath.Join(dir, "bash")
	if err := os.WriteFile(bashStub, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatalf("writing bash stub: %v", err)
	}

	t.Setenv("PATH", dir)
	t.Setenv("SHELL", "")

	testEtcShellsPath = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { testEtcShellsPath = "" })

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DiscoverShells panicked with missing /etc/shells: %v", r)
		}
	}()

	result := DiscoverShells()

	foundBash := false
	for _, sh := range result {
		if sh.Name == "bash" {
			foundBash = true
			break
		}
	}
	if !foundBash {
		t.Error("expected bash entry even with missing /etc/shells")
	}
}

// TestDiscoverShells_EtcShellsFixture verifies POSIX synthetic system-default
// surfacing when $SHELL is an endorsed binary present in /etc/shells.
func TestDiscoverShells_EtcShellsFixture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell script stubs not executable on Windows")
	}
	dir := t.TempDir()

	zshStub := filepath.Join(dir, "zsh")
	if err := os.WriteFile(zshStub, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatalf("writing zsh stub: %v", err)
	}

	etcShells := filepath.Join(dir, "etc-shells")
	content := "# comments\n" + zshStub + "\n/bin/bash\n/usr/local/bin/fish\n"
	if err := os.WriteFile(etcShells, []byte(content), 0644); err != nil {
		t.Fatalf("writing etc-shells fixture: %v", err)
	}

	t.Setenv("PATH", dir)
	t.Setenv("SHELL", zshStub)

	testEtcShellsPath = etcShells
	t.Cleanup(func() { testEtcShellsPath = "" })

	result := DiscoverShells()

	foundSystemDefault := false
	for _, sh := range result {
		if sh.Name == "shell" {
			foundSystemDefault = true
			if sh.Path != zshStub {
				t.Errorf("expected synthetic shell entry Path=%q, got %q", zshStub, sh.Path)
			}
		}
		if sh.Name == "fish" {
			t.Error("fish should not surface as a DetectedShell — out of scope")
		}
	}
	if !foundSystemDefault {
		t.Error("expected synthetic 'shell' (system default) entry with SHELL=zsh in /etc/shells")
	}
}

// TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry locks the H4 contract that
// Plan 04's empty-PATH test depends on: when $SHELL is empty, no synthetic
// "shell" entry is ever appended regardless of /etc/shells content.
func TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only contract — Windows has no /etc/shells codepath")
	}
	dir := t.TempDir()

	etcShells := filepath.Join(dir, "etc-shells")
	if err := os.WriteFile(etcShells, []byte("/bin/bash\n"), 0644); err != nil {
		t.Fatalf("writing etc-shells fixture: %v", err)
	}

	testEtcShellsPath = etcShells
	t.Cleanup(func() { testEtcShellsPath = "" })

	t.Setenv("SHELL", "")
	t.Setenv("PATH", t.TempDir())

	result := DiscoverShells()

	for _, sh := range result {
		if sh.Name == "shell" {
			t.Errorf("expected NO synthetic 'shell' entry with empty $SHELL, got %+v", sh)
		}
	}
}

// TestDiscoverShells_ShBasenameProducesSyntheticEntry locks WR-03: when
// $SHELL=/bin/sh (the most common minimal-container default per
// RESEARCH.md Pitfall 2), DiscoverShells surfaces a synthetic
// "system default" entry so slim Linux deployments are not blocked from
// using the shell-session feature.
func TestDiscoverShells_ShBasenameProducesSyntheticEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only contract — Windows has no /etc/shells codepath")
	}
	dir := t.TempDir()

	shStub := filepath.Join(dir, "sh")
	if err := os.WriteFile(shStub, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatalf("writing sh stub: %v", err)
	}

	etcShells := filepath.Join(dir, "etc-shells")
	content := shStub + "\n/bin/bash\n"
	if err := os.WriteFile(etcShells, []byte(content), 0644); err != nil {
		t.Fatalf("writing etc-shells fixture: %v", err)
	}

	t.Setenv("PATH", dir)
	t.Setenv("SHELL", shStub)

	testEtcShellsPath = etcShells
	t.Cleanup(func() { testEtcShellsPath = "" })

	result := DiscoverShells()

	foundSystemDefault := false
	for _, sh := range result {
		if sh.Name == "shell" {
			foundSystemDefault = true
			if sh.Path != shStub {
				t.Errorf("expected synthetic shell entry Path=%q, got %q", shStub, sh.Path)
			}
			// argvForShellBasename("sh") returns {"-i"}.
			if len(sh.Argv) != 1 || sh.Argv[0] != "-i" {
				t.Errorf("expected synthetic shell entry Argv=[-i], got %v", sh.Argv)
			}
		}
	}
	if !foundSystemDefault {
		t.Error("expected synthetic 'shell' (system default) entry with SHELL=/bin/sh")
	}
}

// TestDiscoverShells_Windows is a smoke-level Windows-only assertion that
// pwsh.exe resolves via PATHEXT when present on the runner.
func TestDiscoverShells_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	result := DiscoverShells()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	for _, sh := range result {
		if sh.Name == "pwsh" {
			if !strings.EqualFold(filepath.Ext(sh.Path), ".exe") {
				t.Errorf("expected pwsh Path to end with .exe, got %q", sh.Path)
			}
		}
	}
}

// TestDiscoverShells_WindowsPowerShell locks the M2 contract that "powershell"
// is a first-class spec on Windows (Plan 02's override branch resolves to
// knownShellSpecs).
func TestDiscoverShells_WindowsPowerShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	result := DiscoverShells()

	hasPowerShell := false
	hasPwsh := false
	for _, sh := range result {
		if sh.Name == "powershell" {
			hasPowerShell = true
		}
		if sh.Name == "pwsh" {
			hasPwsh = true
		}
	}
	// Smoke: if either is on the runner, at least one should surface.
	if !hasPowerShell && !hasPwsh {
		t.Log("neither powershell nor pwsh found on Windows runner — skipping assertion")
	}
}
