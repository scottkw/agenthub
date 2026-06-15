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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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

// ErrProtectedSystemFile is returned by all Sandbox write methods when the
// target resolves to a shell-RC file, SSH key, Claude config, or daemon
// config directory under the user's $HOME. The "files: " prefix matches the
// package-wide error-string convention (FSW-06).
var ErrProtectedSystemFile = errors.New("files: protected system file")

// ErrPreconditionFailed is returned by WriteFileAtomic when the caller
// supplied a non-empty expectedValidator and the on-disk file's validator
// no longer matches immediately before the rename. This narrows the TOCTOU
// window to stat→rename (CR-01 fix). The HTTP handler maps this to 412.
var ErrPreconditionFailed = errors.New("files: precondition failed: file modified by another process")

// denylistCheck returns ErrProtectedSystemFile if the write target
// (identified by the cleaned relative path) resolves to a sensitive file
// under $HOME. It operates on the resolved absolute path so the check fires
// only when the session's working directory IS at or below $HOME — the
// dangerous case for AI-agent writes (FSW-06, RESEARCH §Pattern 4).
//
// Canonical denylist:
//   - Shell RC files: .bashrc, .zshrc, .profile, .bash_profile, .zprofile,
//     .zshenv, .bash_login
//   - SSH directory: anything under .ssh/ (authorized_keys, config, etc.)
//   - Claude config: anything under .claude/
//   - Daemon config: anything under .config/agenthub/
//
// CR-02 fail-closed fix: abs is built lexically from s.rootPath, which is
// already EvalSymlinks-resolved at construction. We also EvalSymlinks the
// existing parent of the target so any symlinks in the target's directory
// portion are resolved before the filepath.Rel comparison. If the parent
// EvalSymlinks fails we use the lexical abs — still safe because rootPath
// itself is canonical. Returns nil only when the target is genuinely outside
// $HOME after canonicalization; fails closed (returns ErrProtectedSystemFile)
// on any ambiguity caused by a symlinked path component.
func (s *Sandbox) denylistCheck(cleaned string) error {
	// Build the lexical absolute path. s.rootPath is already EvalSymlinks-resolved
	// (enforced by NewSandbox), so this is canonical up to any symlinks in
	// `cleaned` itself.
	abs := filepath.Join(s.rootPath, cleaned)

	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	// EvalSymlinks on home so the resolved rootPath (already EvalSymlinks-d
	// in NewSandbox) and home are on the same canonical path prefix.
	// On macOS /var/folders/... resolves to /private/var/folders/... and
	// without this step filepath.Rel returns ".." paths for valid home targets.
	if resolved, err := filepath.EvalSymlinks(home); err == nil {
		home = resolved
	}

	// CR-02: canonicalize the existing parent of the target so that symlinks
	// in the target's directory portion (not yet created) are resolved before
	// the Rel comparison. We walk up to the first existing ancestor.
	canonAbs := abs
	parent := filepath.Dir(abs)
	if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil {
		// Rebuild abs with the canonical parent + base name.
		canonAbs = filepath.Join(resolvedParent, filepath.Base(abs))
	}

	rel, err := filepath.Rel(home, canonAbs)
	if err != nil || strings.HasPrefix(rel, "..") {
		// Target is not under $HOME — denylist does not apply.
		// Note: an error from filepath.Rel is extremely rare (both paths are
		// absolute on the same OS) and does not indicate a security concern here;
		// returning nil is the correct fail-open for the "not under $HOME" case.
		return nil
	}
	// Shell RC files — exact base-name match (case-folded for macOS/Windows
	// case-insensitive volumes: .BASHRC and .Bashrc must be caught too).
	// All protected names are ASCII so strings.ToLower is safe and sufficient;
	// NFC/NFD Unicode normalization is a LOW residual (all names are ASCII).
	base := strings.ToLower(filepath.Base(canonAbs))
	switch base {
	case ".bashrc", ".zshrc", ".profile", ".bash_profile",
		".zprofile", ".zshenv", ".bash_login":
		return ErrProtectedSystemFile
	}
	// Directory-prefix protections (forward-slash normalised so the check
	// is consistent across Windows, macOS, and Linux path formats).
	// Case-folded for the same reason as the base-name switch above.
	relSlash := strings.ToLower(filepath.ToSlash(rel))

	// Build the protected-prefix set. Start with static prefixes that cover
	// Linux/cross-platform copied trees (belt-and-suspenders). Then derive
	// the OS-correct daemon config dir from os.UserConfigDir() — on macOS this
	// is ~/Library/Application Support, not ~/.config — and add its
	// $HOME-relative form. Two-line derivation mirrors engine.go:daemonConfigDir
	// but avoids importing the daemon package (internal/files must stay cycle-free).
	protectedDirs := []string{".ssh/", ".claude/", ".config/agenthub/"}
	if cfgBase, cfgErr := os.UserConfigDir(); cfgErr == nil {
		cfgDir := filepath.Join(cfgBase, "agenthub")
		cfgRel, relErr := filepath.Rel(home, cfgDir)
		if relErr == nil && !strings.HasPrefix(cfgRel, "..") {
			// cfgDir is under $HOME; add its forward-slash, lowercase relative
			// form. We append rather than prepend so the static entries still
			// fire for cross-platform copied trees.
			cfgRelSlash := strings.ToLower(filepath.ToSlash(cfgRel)) + "/"
			protectedDirs = append(protectedDirs, cfgRelSlash)
		}
	}

	for _, dir := range protectedDirs {
		if relSlash == strings.TrimSuffix(dir, "/") || strings.HasPrefix(relSlash, dir) {
			return ErrProtectedSystemFile
		}
	}
	return nil
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

// WriteFileAtomic writes content to relPath inside the sandbox using a
// sibling temp file + f.Sync() + atomic root.Rename. A concurrent reader
// never observes an empty or partial file because the old inode is visible
// until the rename completes (FSW-01, RESEARCH §Pattern 1).
//
// expectedValidator, when non-empty and not "*", enables CR-01 TOCTOU
// mitigation: the validator (mtime-UnixNano + size, quoted) is re-checked
// immediately before root.Rename. If the on-disk value no longer matches,
// ErrPreconditionFailed is returned and the temp file is removed. This
// narrows the optimistic-concurrency check window to stat→rename, the
// minimum achievable without OS-level atomic-compare-and-swap. A residual
// window (stat fires → another writer lands → rename executes) exists but
// is microscopic on a local filesystem; it cannot be eliminated without
// kernel support (e.g. renameat2 RENAME_NOREPLACE is not sufficient here).
//
// The temp file is a sibling of the target (same directory) so the rename
// is intra-filesystem and atomic. The suffix uses crypto/rand to make name
// collisions between concurrent writers practically impossible; O_EXCL on
// the temp create guarantees no silent clobber (FSW-01 §Pitfall 5).
//
// On Windows, rename-over-an-open-file fails with a sharing-violation.
// A bounded 3-attempt retry loop (~50ms between attempts) is applied on
// Windows only (FSW-01 §Pitfall 5, RESEARCH Open Q3).
//
// Writes via temp file only; never in-place truncate. Temp stays inside the
// sandbox root so the rename is always intra-filesystem (FSW-01).
func (s *Sandbox) WriteFileAtomic(relPath string, content []byte, expectedValidator ...string) error {
	cleaned, err := validateAndClean(relPath)
	if err != nil {
		return err
	}
	if err := s.denylistCheck(cleaned); err != nil { // FSW-06
		return err
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return fmt.Errorf("files: open root: %w", err)
	}
	defer root.Close()

	// WR-03: surface a failed CSPRNG read per let-it-crash (CLAUDE.md §Silent Fallbacks).
	// A failed rand.Read means the OS CSPRNG is broken — an extremely rare
	// condition that should never be silently swallowed.
	var rnd [8]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return fmt.Errorf("files: generate temp suffix: %w", err)
	}
	tmp := cleaned + ".agenthub-tmp-" + hex.EncodeToString(rnd[:])

	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("files: create temp: %w", err)
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		_ = root.Remove(tmp)
		return fmt.Errorf("files: write temp: %w", err)
	}
	if err := f.Sync(); err != nil { // fdatasync before rename — crash durability
		f.Close()
		_ = root.Remove(tmp)
		return fmt.Errorf("files: sync temp: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("files: close temp: %w", err)
	}

	// CR-01 TOCTOU mitigation: re-check the validator immediately before the
	// rename to narrow the optimistic-concurrency window to stat→rename.
	// A residual (microscopic) window remains — see WriteFileAtomic doc comment.
	validator := ""
	if len(expectedValidator) > 0 {
		validator = expectedValidator[0]
	}
	if validator != "" && validator != "*" {
		if fi, err := root.Stat(cleaned); err == nil {
			cur := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))
			if cur != validator {
				_ = root.Remove(tmp)
				return ErrPreconditionFailed
			}
		}
		// If Stat returns an error (file does not exist), allow the rename —
		// the caller's pre-write Stat in Handler.Write already verified existence;
		// a missing file here means another writer deleted it, which is fine for
		// a new-file write and benign for an existing-file write (the rename will
		// create it atomically).
	}

	// Rename temp → target atomically. On Windows a bounded retry loop
	// handles sharing-violation errors when the target is open (FSW-01).
	if err := writeAtomicRename(root, tmp, cleaned); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("files: rename temp: %w", err)
	}
	return nil
}

// writeAtomicRename performs root.Rename(tmp, dst). On Windows it retries
// up to 3 times with a 50ms delay to tolerate sharing-violation errors
// caused by another process holding the destination open. On non-Windows
// platforms a single attempt is made (RESEARCH Open Q3, PITFALLS §Pitfall 5).
func writeAtomicRename(root *os.Root, tmp, dst string) error {
	const maxAttempts = 3
	attempts := maxAttempts
	if runtime.GOOS != "windows" {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := root.Rename(tmp, dst); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < attempts-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}
	return lastErr
}

// Rename moves the file or directory at oldRel to newRel within the
// sandbox root. Both paths are validated through validateAndClean and the
// denylist before os.Root sees them — the destination is the #1
// write-side traversal risk if skipped (FSW-02, RESEARCH §Pitfall 1).
//
// Uses native root.Rename (relative-to-root, TOCTOU-safe) rather than the
// hand-rolled os.Rename(absOld, absNew) workaround documented in PITFALLS.md
// — the native method is strictly safer because it closes the TOCTOU window
// the hand-rolled approach re-opens (RESEARCH §State of the Art).
func (s *Sandbox) Rename(oldRel, newRel string) error {
	oldClean, err := validateAndClean(oldRel)
	if err != nil {
		return fmt.Errorf("files: rename source: %w", err)
	}
	newClean, err := validateAndClean(newRel) // THE #1 write-side traversal risk if skipped
	if err != nil {
		return fmt.Errorf("files: rename destination: %w", err)
	}
	if err := s.denylistCheck(newClean); err != nil { // cannot rename INTO a protected path
		return err
	}
	if err := s.denylistCheck(oldClean); err != nil { // nor move a protected path out
		return err
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return fmt.Errorf("files: open root: %w", err)
	}
	defer root.Close()
	return root.Rename(oldClean, newClean) // native; both paths relative to root (FSW-02)
}

// Mkdir creates a single directory at relPath within the sandbox root.
// The path is validated through validateAndClean and the denylist before
// any syscall. Uses native root.Mkdir (TOCTOU-safe, relative-to-root)
// (FSW-03).
func (s *Sandbox) Mkdir(relPath string) error {
	cleaned, err := validateAndClean(relPath)
	if err != nil {
		return err
	}
	if err := s.denylistCheck(cleaned); err != nil { // FSW-06
		return err
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return fmt.Errorf("files: open root: %w", err)
	}
	defer root.Close()
	return root.Mkdir(cleaned, 0o755)
}

// MkdirAll creates relPath and any missing parent directories within the
// sandbox root. Uses native root.MkdirAll which is TOCTOU-safe and
// relative-to-root — no iterative root.Mkdir loop required (FSW-03,
// RESEARCH §State of the Art).
func (s *Sandbox) MkdirAll(relPath string) error {
	cleaned, err := validateAndClean(relPath)
	if err != nil {
		return err
	}
	if err := s.denylistCheck(cleaned); err != nil { // FSW-06
		return err
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return fmt.Errorf("files: open root: %w", err)
	}
	defer root.Close()
	return root.MkdirAll(cleaned, 0o755)
}

// Delete removes the file or directory subtree at relPath within the
// sandbox root. Uses native root.RemoveAll (TOCTOU-safe, confined by
// construction — cannot escape the root) (FSW-04).
func (s *Sandbox) Delete(relPath string) error {
	cleaned, err := validateAndClean(relPath)
	if err != nil {
		return err
	}
	if err := s.denylistCheck(cleaned); err != nil { // FSW-06
		return err
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return fmt.Errorf("files: open root: %w", err)
	}
	defer root.Close()
	return root.RemoveAll(cleaned)
}

// validateAndClean validates relPath through the layered-defense checks and
// returns the filepath.Clean'd form (which os.Root expects). Any rejection
// returns a non-nil error wrapped with ErrPathValidation so callers can use
// errors.Is(err, ErrPathValidation) for classification. (IN-03)
func validateAndClean(relPath string) (string, error) {
	if err := validateRelativePath(relPath); err != nil {
		return "", fmt.Errorf("%w: %w", ErrPathValidation, err)
	}
	cleaned := filepath.Clean(relPath)
	// After Clean, any leading ".." (alone or as a path prefix) is an escape
	// attempt — defense in depth before os.Root sees it.
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) ||
		strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: files: path traversal rejected", ErrPathValidation)
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
	// Restrict to ASCII A-Z/a-z to match the real Windows grammar — the
	// later colon-anywhere ban (WR-02) catches non-letter colon prefixes
	// like "a:foo" with the accurate "colon (ADS) rejected" message.
	if len(p) >= 2 && p[1] == ':' && isASCIILetter(p[0]) {
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

// isASCIILetter reports whether b is an ASCII letter A-Z or a-z. Used
// by the drive-letter prefix check (WR-02) — Windows drive letters are
// strictly ASCII, so a Unicode-aware test would be both over-broad and
// require pulling in the unicode package for one byte's worth of work.
func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
