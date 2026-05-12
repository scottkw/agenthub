package pty

// NOTE (Phase 100-02 worktree stub): This file is a parallel-wave placeholder
// created so Plan 100-02 (engine.go shell-spawn wiring) can build and test
// independently of Plan 100-01 (the canonical shells.go authors). Plan 01 is
// expected to author the real, production-quality version of this file —
// including platform-conditional PATH/PATHEXT discovery, /etc/shells parsing
// on POSIX, Windows `powershell.exe` 5.x fallback, and a synthetic
// "shell" (system default) entry derived from $SHELL.
//
// When the wave merges, Plan 01's commit on internal/pty/shells.go must take
// precedence. The API contract preserved here is the one Plan 100-02's
// engine.go and engine_test.go depend on:
//
//   type ShellSpec struct { Name, DisplayName string; Argv []string }
//   type DetectedShell struct { Name, DisplayName, Path string; Argv []string }
//   func KnownShellSpecs() []ShellSpec   // returns 4 entries per Plan 01 M2:
//                                        //   bash, zsh, pwsh, powershell
//   func DiscoverShells() []DetectedShell
//   func DetectShell(name string) (*DetectedShell, error)
//   var ErrShellNotFound = errors.New("shell not found")
//
// Plan 01 owns the canonical contract. This stub MUST be deleted by the
// orchestrator at merge time in favor of Plan 01's version.

import (
	"errors"
	"os/exec"
	"runtime"
)

// ErrShellNotFound is returned by DetectShell when the named shell is not
// found on PATH.
var ErrShellNotFound = errors.New("shell not found")

// ShellSpec is a known shell that AgentHub can spawn.
type ShellSpec struct {
	Name        string   // canonical short name ("bash", "zsh", "pwsh", "powershell")
	DisplayName string   // human-readable label
	Argv        []string // interactive-non-login argv (e.g. ["-i"] or ["-NoLogo"])
}

// DetectedShell is a ShellSpec whose binary is present on this system.
type DetectedShell struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Path        string   `json:"path"`
	Argv        []string `json:"argv"`
}

// knownShellSpecs is the authoritative shell list. Per Plan 01 M2 lock-in,
// powershell is a first-class spec alongside pwsh (so a `cliPaths["powershell"]`
// override resolves cleanly without falling through to live discovery).
var knownShellSpecs = []ShellSpec{
	{Name: "bash", DisplayName: "bash", Argv: []string{"-i"}},
	{Name: "zsh", DisplayName: "zsh", Argv: []string{"-i"}},
	{Name: "pwsh", DisplayName: "PowerShell", Argv: []string{"-NoLogo"}},
	{Name: "powershell", DisplayName: "Windows PowerShell", Argv: []string{"-NoLogo"}},
}

// KnownShellSpecs returns a copy of the known shell specs slice.
// Read-only consumer API — callers should not mutate the returned slice.
func KnownShellSpecs() []ShellSpec {
	out := make([]ShellSpec, len(knownShellSpecs))
	copy(out, knownShellSpecs)
	return out
}

// DiscoverShells returns the subset of knownShellSpecs whose binary is found
// on PATH plus, on POSIX, a synthetic "shell" entry resolved from $SHELL.
//
// Stub semantics (Plan 100-02 placeholder): minimal PATH discovery only.
// Plan 01's canonical implementation adds /etc/shells parsing and Windows
// powershell.exe 5.x fallback when pwsh is absent.
func DiscoverShells() []DetectedShell {
	result := make([]DetectedShell, 0)
	for _, spec := range knownShellSpecs {
		// On POSIX, "powershell" is rarely on PATH — skip the discovery
		// (the override branch in engine.go still resolves the spec).
		if spec.Name == "powershell" && runtime.GOOS != "windows" {
			continue
		}
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
	// POSIX system-default synthetic entry derived from $SHELL.
	if sysShell := systemDefaultShell(); sysShell != nil {
		result = append(result, *sysShell)
	}
	return result
}

// DetectShell looks up a single shell by name. Returns ErrShellNotFound if
// not in knownShellSpecs or its binary is not on PATH.
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

// systemDefaultShell returns a synthetic DetectedShell representing the
// caller's preferred shell (`$SHELL` on POSIX, pwsh/powershell on Windows).
// Returns nil if the value cannot be resolved.
func systemDefaultShell() *DetectedShell {
	// Stub: only resolve on POSIX via $SHELL. Plan 01 canonical
	// implementation handles Windows fallback to pwsh.exe / powershell.exe.
	if runtime.GOOS == "windows" {
		return nil
	}
	// Read $SHELL via os.Getenv at call time, not import time.
	shell := lookupSysShellEnv()
	if shell == "" {
		return nil
	}
	// Verify the binary is executable.
	if _, err := exec.LookPath(shell); err != nil {
		return nil
	}
	// Determine argv: if basename matches a known spec, use its Argv,
	// otherwise default to ["-i"].
	argv := []string{"-i"}
	for _, spec := range knownShellSpecs {
		if shellBasenameMatches(shell, spec.Name) {
			argv = append([]string(nil), spec.Argv...)
			break
		}
	}
	return &DetectedShell{
		Name:        "shell",
		DisplayName: "system default",
		Path:        shell,
		Argv:        argv,
	}
}
