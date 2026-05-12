package pty

// Phase 100-02 worktree stub helpers — Plan 01 canonical implementation
// supersedes this at merge time.

import (
	"os"
	"path/filepath"
	"strings"
)

// lookupSysShellEnv returns the value of the SHELL environment variable
// at call time. Indirection allows tests to override via t.Setenv.
func lookupSysShellEnv() string {
	return os.Getenv("SHELL")
}

// shellBasenameMatches reports whether the basename of `path` (sans .exe)
// equals `name` (case-sensitive).
func shellBasenameMatches(path, name string) bool {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".exe")
	return base == name
}
