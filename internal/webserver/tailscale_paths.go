package webserver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// tailscaleWellKnownPaths returns the ordered list of well-known binary paths
// for the current platform. Detection order: custom path -> these -> PATH.
func tailscaleWellKnownPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Tailscale.app/Contents/MacOS/Tailscale",
			"/opt/homebrew/bin/tailscale",
			"/usr/local/bin/tailscale",
		}
	case "linux":
		home, _ := os.UserHomeDir()
		paths := []string{
			"/usr/bin/tailscale",
			"/usr/sbin/tailscale",
			"/snap/bin/tailscale",
			"/var/lib/flatpak/exports/bin/tailscale",
		}
		if home != "" {
			paths = append(paths, filepath.Join(home, ".local", "share", "flatpak", "exports", "bin", "tailscale"))
		}
		return paths
	case "windows":
		return []string{
			`C:\Program Files\Tailscale\tailscale.exe`,
			`C:\Program Files (x86)\Tailscale\tailscale.exe`,
		}
	}
	return nil
}

// detectTailscaleBinary finds the tailscale binary. Detection order:
//  1. customPath (non-empty, must exist on disk)
//  2. Well-known platform paths (tailscaleWellKnownPaths)
//  3. PATH (exec.LookPath)
//
// Returns the resolved path or "" if not found.
func detectTailscaleBinary(customPath string) string {
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath
		}
	}
	for _, p := range tailscaleWellKnownPaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("tailscale"); err == nil {
		return p
	}
	return ""
}
