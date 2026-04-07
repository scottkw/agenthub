package daemon

import (
	"os"
	"path/filepath"
	"strings"
)

// AugmentServicePath prepends well-known user tool directories to the process
// PATH so that CLIs installed via nvm, volta, or Homebrew are found when the
// daemon runs as a launchd/systemd user service (or when the GUI app is
// launched from Finder/Dock), which do not source shell init files. Called
// once at startup before any exec.LookPath or session is created.
func AugmentServicePath() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	candidates := []string{
		filepath.Join(home, ".volta", "bin"),
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/home/linuxbrew/.linuxbrew/bin",
		nvmActiveBin(home),
	}

	current := os.Getenv("PATH")
	var extra []string
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(dir); err == nil {
			extra = append(extra, dir)
		}
	}
	if len(extra) > 0 {
		_ = os.Setenv("PATH", strings.Join(extra, string(os.PathListSeparator))+string(os.PathListSeparator)+current)
	}
}

// nvmActiveBin reads ~/.nvm/alias/default to find the active node version
// and returns the bin directory path, or "" if nvm is not installed.
func nvmActiveBin(home string) string {
	aliasFile := filepath.Join(home, ".nvm", "alias", "default")
	data, err := os.ReadFile(aliasFile)
	if err != nil {
		return ""
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	nvmDir := filepath.Join(home, ".nvm", "versions", "node")
	entries, err := os.ReadDir(nvmDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), version) {
			return filepath.Join(nvmDir, e.Name(), "bin")
		}
	}
	return ""
}
