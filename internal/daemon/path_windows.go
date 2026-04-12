//go:build windows

package daemon

import (
	"os"
	"path/filepath"
)

// platformExtraBins returns Windows-specific binary directories for agent CLIs
// and Tailscale. Called from AugmentServicePath in path.go.
func platformExtraBins() []string {
	var paths []string
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		paths = append(paths, filepath.Join(appdata, "npm"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		paths = append(paths, filepath.Join(local, "pnpm"))
		paths = append(paths, filepath.Join(local, "Programs", "nodejs"))
	}
	paths = append(paths, `C:\Program Files\Tailscale`)
	return paths
}
