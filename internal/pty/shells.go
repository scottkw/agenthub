package pty

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// ErrShellNotFound is returned by DetectShell when the named shell is not
// found on PATH (or not a member of knownShellSpecs).
var ErrShellNotFound = errors.New("shell not found")

// ShellSpec describes a known interactive shell that AgentHub can launch as
// a raw shell-session. Argv carries the conventional interactive-launch flags
// for the shell (e.g. "-i" for POSIX shells, "-NoLogo" for PowerShell).
type ShellSpec struct {
	Name        string
	DisplayName string
	Argv        []string
}

// DetectedShell is a ShellSpec whose binary was found on PATH (or, for the
// synthetic "shell" entry, points at the user's $SHELL value).
type DetectedShell struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Path        string   `json:"path"`
	Argv        []string `json:"argv"`
}

// knownShellSpecs is the authoritative list of interactive shells that
// AgentHub supports for the raw shell-session feature. Order is canonical and
// load-bearing: TestKnownShellSpecs_HasExpectedEntries asserts the exact set,
// and Plan 02's argv resolution walks this list in order.
//
// Per M2: `powershell` is a first-class spec (alongside `pwsh`) so that
// Plan 02's override branch (`cliPaths["powershell"] = "..."`) resolves to a
// clean match without a runtime fallback codepath. On POSIX hosts the
// `powershell` entry will not surface in DiscoverShells results because
// exec.LookPath("powershell") fails — keeping the table platform-agnostic
// avoids build-tag fragmentation.
var knownShellSpecs = []ShellSpec{
	{Name: "bash", DisplayName: "bash", Argv: []string{"-i"}},
	{Name: "zsh", DisplayName: "zsh", Argv: []string{"-i"}},
	{Name: "pwsh", DisplayName: "PowerShell", Argv: []string{"-NoLogo"}},
	{Name: "powershell", DisplayName: "Windows PowerShell", Argv: []string{"-NoLogo"}},
}

// DiscoverShells scans PATH for every entry in knownShellSpecs and returns
// the subset that is actually installed. On POSIX it may additionally append
// a synthetic "shell" (DisplayName="system default") entry when $SHELL is set
// to a basename that matches one of knownShellSpecs (cross-checked against
// /etc/shells when readable).
//
// The returned slice is always non-nil (load-bearing for slim Linux
// containers where no shells are present — callers can range with confidence).
func DiscoverShells() []DetectedShell {
	return discoverShells("/etc/shells")
}

// discoverShells is the parameterised implementation of DiscoverShells. The
// etcShellsPath argument exists so tests can inject an /etc/shells fixture
// path without mutating a package-level variable (WR-04: the previous
// approach used a `testEtcShellsPath` package-level mutable read by
// production code, which was a latent data race under -race + parallel
// tests). Production callers go through DiscoverShells which always passes
// "/etc/shells".
func discoverShells(etcShellsPath string) []DetectedShell {
	result := make([]DetectedShell, 0)

	// Pass 1 — known specs via PATH (mirrors detect.go).
	for _, spec := range knownShellSpecs {
		path, err := exec.LookPath(spec.Name)
		if err != nil {
			continue
		}
		result = append(result, DetectedShell{
			Name:        spec.Name,
			DisplayName: spec.DisplayName,
			Path:        path,
			Argv:        append([]string(nil), spec.Argv...),
		})
	}

	// Pass 2 — POSIX synthetic system-default ($SHELL).
	if runtime.GOOS != "windows" {
		shellEnv := os.Getenv("SHELL")
		if shellEnv == "" {
			// H4 contract: empty $SHELL never produces a synthetic entry,
			// regardless of /etc/shells contents.
			return result
		}
		if !isEndorsedShellBasename(shellEnv) {
			return result
		}
		// /etc/shells cross-check (optional — silent skip on read error).
		shells := readEtcShells(etcShellsPath)
		if len(shells) > 0 && !slices.Contains(shells, shellEnv) {
			// /etc/shells is readable but does not list $SHELL — refuse to
			// surface a synthetic entry.
			return result
		}
		result = append(result, DetectedShell{
			Name:        "shell",
			DisplayName: "system default",
			Path:        shellEnv,
			Argv:        argvForShellBasename(filepath.Base(shellEnv)),
		})
	}

	return result
}

// DetectShell looks up a single shell by name. Returns ErrShellNotFound if
// the shell is not in knownShellSpecs or its binary is not on PATH.
func DetectShell(name string) (*DetectedShell, error) {
	for _, spec := range knownShellSpecs {
		if spec.Name != name {
			continue
		}
		path, err := exec.LookPath(spec.Name)
		if err != nil {
			return nil, ErrShellNotFound
		}
		return &DetectedShell{
			Name:        spec.Name,
			DisplayName: spec.DisplayName,
			Path:        path,
			Argv:        append([]string(nil), spec.Argv...),
		}, nil
	}
	return nil, ErrShellNotFound
}

// KnownShellSpecs returns the package-level knownShellSpecs slice. Callers
// must NOT mutate the returned slice (it is the same backing array used by
// DiscoverShells); copy if mutation is required.
func KnownShellSpecs() []ShellSpec {
	return knownShellSpecs
}

// isEndorsedShellBasename reports whether the basename of p is one of the
// allowlisted interactive-shell basenames. This guards the synthetic
// system-default entry from surfacing arbitrary $SHELL values.
//
// WR-03: "sh" is allowlisted as a synthetic-system-default basename only —
// it is intentionally NOT added to knownShellSpecs (out of scope per
// REQUIREMENTS.md as a selectable shell). Slim Linux containers (Alpine,
// distroless — RESEARCH.md Pitfall 2) commonly have $SHELL=/bin/sh as
// the only viable interactive shell; surfacing the synthetic entry there
// honors the actual user environment without expanding the "selectable
// shells" surface.
func isEndorsedShellBasename(p string) bool {
	base := filepath.Base(p)
	switch base {
	case "sh", "bash", "zsh", "pwsh", "powershell":
		return true
	}
	return false
}

// argvForShellBasename returns the conventional interactive-launch flags for
// the given shell basename. Defaults to {"-i"} for unknown basenames (safe
// for any POSIX-style shell).
//
// IN-01: the default branch is intentionally retained as a safety net.
// Current production callers always pass a basename that
// isEndorsedShellBasename has already accepted (sh/bash/zsh/pwsh/powershell),
// so the default is unreachable today. It is preserved so future endorsed
// basenames (or direct library/test callers) get a sensible fallback rather
// than a nil slice — the "-i" flag is a safe interactive default for any
// POSIX-style shell.
func argvForShellBasename(base string) []string {
	switch base {
	case "sh", "bash", "zsh":
		// Explicit "sh" arm (WR-03): /bin/sh accepts -i as an interactive
		// flag; equivalent to the default fallback below, but listed
		// explicitly so the contract is auditable.
		return []string{"-i"}
	case "pwsh", "powershell":
		return []string{"-NoLogo"}
	}
	return []string{"-i"}
}

// readEtcShells reads /etc/shells (or the override path) and returns the
// non-comment, non-empty lines as a slice. Returns an empty slice on any
// read error (silent skip per Pitfall 2 — missing /etc/shells must not be
// fatal on slim containers).
func readEtcShells(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return []string{}
	}
	defer f.Close()

	out := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return []string{}
	}
	return out
}
