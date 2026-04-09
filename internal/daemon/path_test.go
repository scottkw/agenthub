package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAugmentServicePath_AddsExistingDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix PATH separator")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	original := "/usr/bin:/bin"
	t.Setenv("PATH", original)

	// Create volta bin dir
	voltaBin := filepath.Join(home, ".volta", "bin")
	if err := os.MkdirAll(voltaBin, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	AugmentServicePath()

	got := os.Getenv("PATH")
	if !strings.Contains(got, voltaBin) {
		t.Errorf("PATH should contain %s, got %s", voltaBin, got)
	}
	if !strings.HasSuffix(got, original) {
		t.Errorf("PATH should end with original %s, got %s", original, got)
	}
}

func TestAugmentServicePath_SkipsNonexistent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	original := "/usr/bin:/bin"
	t.Setenv("PATH", original)

	// No .volta, no .nvm, no homebrew dirs created — all non-existent
	AugmentServicePath()

	got := os.Getenv("PATH")
	// PATH may include /opt/homebrew/bin or /usr/local/bin if they actually
	// exist on this machine, so we check that neither volta nor nvm dirs
	// were added (since they don't exist under our temp home).
	voltaBin := filepath.Join(home, ".volta", "bin")
	if strings.Contains(got, voltaBin) {
		t.Errorf("PATH should not contain non-existent %s, got %s", voltaBin, got)
	}
	nvmBin := filepath.Join(home, ".nvm")
	if strings.Contains(got, nvmBin) {
		t.Errorf("PATH should not contain non-existent nvm path %s, got %s", nvmBin, got)
	}
}

func TestNvmActiveBin_ValidAlias(t *testing.T) {
	home := t.TempDir()

	// Create alias/default with short version "20"
	aliasDir := filepath.Join(home, ".nvm", "alias")
	if err := os.MkdirAll(aliasDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aliasDir, "default"), []byte("20"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create matching version directory
	nodeBin := filepath.Join(home, ".nvm", "versions", "node", "v20.11.0", "bin")
	if err := os.MkdirAll(nodeBin, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got := nvmActiveBin(home)
	if got != nodeBin {
		t.Errorf("nvmActiveBin = %q, want %q", got, nodeBin)
	}
}

func TestNvmActiveBin_FullVersionAlias(t *testing.T) {
	home := t.TempDir()

	// Create alias/default with full version "v20.11.0"
	aliasDir := filepath.Join(home, ".nvm", "alias")
	if err := os.MkdirAll(aliasDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aliasDir, "default"), []byte("v20.11.0"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create matching version directory
	nodeBin := filepath.Join(home, ".nvm", "versions", "node", "v20.11.0", "bin")
	if err := os.MkdirAll(nodeBin, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got := nvmActiveBin(home)
	if got != nodeBin {
		t.Errorf("nvmActiveBin = %q, want %q", got, nodeBin)
	}
}

func TestNvmActiveBin_NoNvm(t *testing.T) {
	home := t.TempDir()
	// No .nvm directory

	got := nvmActiveBin(home)
	if got != "" {
		t.Errorf("nvmActiveBin = %q, want empty string", got)
	}
}

func TestAugmentServicePath_AddsLocalBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix PATH separator")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	original := "/usr/bin:/bin"
	t.Setenv("PATH", original)

	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	AugmentServicePath()

	got := os.Getenv("PATH")
	if !strings.Contains(got, localBin) {
		t.Errorf("PATH should contain %s, got %s", localBin, got)
	}
	if !strings.HasSuffix(got, original) {
		t.Errorf("PATH should end with original %s, got %s", original, got)
	}
}

func TestAugmentServicePath_PrependsNotAppends(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix PATH separator")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	original := "/usr/bin:/bin"
	t.Setenv("PATH", original)

	// Create volta bin dir
	voltaBin := filepath.Join(home, ".volta", "bin")
	if err := os.MkdirAll(voltaBin, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	AugmentServicePath()

	got := os.Getenv("PATH")
	voltaIdx := strings.Index(got, voltaBin)
	origIdx := strings.Index(got, original)
	if voltaIdx == -1 {
		t.Errorf("PATH should contain %s, got %s", voltaBin, got)
	}
	if origIdx == -1 {
		t.Errorf("PATH should contain original %s, got %s", original, got)
	}
	if voltaIdx > origIdx {
		t.Errorf("volta bin (%d) should appear BEFORE original PATH (%d) in %s", voltaIdx, origIdx, got)
	}
}
