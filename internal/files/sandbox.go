// Package files provides a TOCTOU-safe read-only filesystem sandbox built
// on Go 1.24+ *os.Root. A Sandbox wraps an EvalSymlinks-resolved absolute
// working directory and exposes Open/Stat/ReadDir operations that atomically
// reject any path escape (absolute paths, "..", symlink-out, Windows device
// names, alternate data streams, etc).
//
// All public operations validate the user-supplied relative path through
// validateRelativePath BEFORE delegating to os.Root, so layered defenses
// catch Windows-specific edge cases on all platforms (RESEARCH §Pattern 2).
//
// The package is dependency-free of other internal packages — daemon,
// relay, webserver, capability all consume it; it consumes none of them.
// See .planning/phases/118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi/118-RESEARCH.md
// for the architecture decisions and threat model.
package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sandbox is a per-session view of a single resolved working directory.
// All file operations are confined to this root via os.OpenRoot, which
// atomically rejects path escapes at the syscall level.
type Sandbox struct {
	rootPath string // EvalSymlinks-resolved absolute path; never empty
}

// NewSandbox returns a Sandbox rooted at workDir. workDir is resolved via
// filepath.EvalSymlinks ONCE (eliminating the TOCTOU race window — per-
// request open uses the cached resolved path via os.OpenRoot). Returns an
// error if workDir is empty, does not exist, or is not a directory.
func NewSandbox(workDir string) (*Sandbox, error) {
	if workDir == "" {
		return nil, errors.New("files: empty WorkDir")
	}
	resolved, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		return nil, fmt.Errorf("files: resolve WorkDir: %w", err)
	}
	fi, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("files: stat WorkDir: %w", err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("files: WorkDir is not a directory: %s", resolved)
	}
	return &Sandbox{rootPath: resolved}, nil
}

// RootPath returns the EvalSymlinks-resolved absolute sandbox root.
func (s *Sandbox) RootPath() string {
	return s.rootPath
}

// Open returns an *os.File for relPath within the sandbox root. The open is
// atomic at the syscall level via os.Root — symlinks that escape the root
// are rejected without any TOCTOU window. The caller is responsible for
// closing the returned file.
//
// relPath must be a non-empty path relative to the sandbox root. Absolute
// paths, paths containing "..", null bytes, Windows reserved device names,
// alternate-data-stream colons, and UNC prefixes are all rejected by
// validateRelativePath before os.Root sees them.
func (s *Sandbox) Open(relPath string) (*os.File, error) {
	cleaned, err := validateAndClean(relPath)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return nil, fmt.Errorf("files: open root: %w", err)
	}
	defer root.Close()
	return root.Open(cleaned)
}

// Stat returns os.FileInfo for relPath within the sandbox root. Symlinks
// that escape the root are rejected atomically (same security boundary as
// Open).
func (s *Sandbox) Stat(relPath string) (os.FileInfo, error) {
	cleaned, err := validateAndClean(relPath)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return nil, fmt.Errorf("files: open root: %w", err)
	}
	defer root.Close()
	return root.Stat(cleaned)
}

// validateAndClean validates relPath through the layered-defense checks and
// returns the filepath.Clean'd form (which os.Root expects). Any rejection
// returns a non-nil error.
func validateAndClean(relPath string) (string, error) {
	if err := validateRelativePath(relPath); err != nil {
		return "", err
	}
	cleaned := filepath.Clean(relPath)
	// After Clean, any leading ".." (alone or as a path prefix) is an escape
	// attempt — defense in depth before os.Root sees it.
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) ||
		strings.HasPrefix(cleaned, "../") {
		return "", errors.New("files: path traversal rejected")
	}
	return cleaned, nil
}

// windowsDeviceNames lists Windows reserved device names that must be
// rejected on ALL platforms (not just Windows builds). CVE-2025-27210 was
// exactly this class — a Linux daemon receiving ?path=NUL would happily
// look for a file named "NUL", and a Windows-written directory copied to
// Linux could contain a literal CON.txt that traps a Windows reader.
var windowsDeviceNames = map[string]bool{
	"CON": true, "NUL": true, "PRN": true, "AUX": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// validateRelativePath applies the layered-defense rejections from
// PITFALLS.md Pitfall 2 BEFORE the path reaches os.Root. The checks are
// cross-platform — Windows device names and alternate-data-stream colons
// are rejected on Linux/macOS too, because a Windows-written directory
// can contain files that exploit a non-Windows reader.
//
// Acceptance:
//   - "." is allowed (sandbox root itself; List/Stat use it for the cwd).
//   - "a", "a/b", "./a", "./a/b" are accepted (Clean normalises ./ prefix).
//
// Rejection:
//   - "" (empty)
//   - any rune == '\x00'
//   - filepath.IsAbs (catches /etc, C:\, etc.)
//   - drive letter "X:" prefix (IsAbs on non-Windows misses this)
//   - UNC \\server\share or //server/share
//   - any ':' anywhere (alternate-data-stream / ADS)
//   - any path segment whose stem (sans extension) is a Windows device name
//   - "..", "../..", or any traversal that escapes after Clean
func validateRelativePath(p string) error {
	if p == "" {
		return errors.New("files: empty path")
	}
	if strings.ContainsRune(p, 0) {
		return errors.New("files: null byte in path")
	}
	if filepath.IsAbs(p) {
		return errors.New("files: absolute path rejected")
	}
	// Windows drive letter "X:..." on non-Windows (IsAbs misses this).
	if len(p) >= 2 && p[1] == ':' {
		return errors.New("files: drive letter rejected")
	}
	// UNC: \\server\share or //server/share
	if strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//") {
		return errors.New("files: UNC path rejected")
	}
	// Alternate Data Streams — total ban on ':' anywhere (rare in
	// legitimate filenames; security trade-off favours rejection).
	if strings.ContainsRune(p, ':') {
		return errors.New("files: colon (ADS) rejected")
	}
	// Windows reserved device names — case-insensitive, with or without
	// extension, applied per path segment so "sub/CON" is also rejected.
	segments := strings.FieldsFunc(p, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, segment := range segments {
		base := strings.ToUpper(segment)
		if dot := strings.IndexByte(base, '.'); dot >= 0 {
			base = base[:dot]
		}
		if windowsDeviceNames[base] {
			return fmt.Errorf("files: Windows device name rejected: %s", segment)
		}
	}
	// Defense-in-depth: reject "../" or ".." after Clean (os.Root also
	// rejects, but explicit upfront rejection produces a more obvious
	// error message in tests).
	cleaned := filepath.Clean(p)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) ||
		strings.HasPrefix(cleaned, "../") {
		return errors.New("files: path traversal rejected")
	}
	return nil
}
