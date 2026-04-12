# Phase 68: Agent & Tailscale Discovery + Install Instructions — Research

**Researched:** 2026-04-11
**Domain:** Go PATH augmentation, cross-platform binary discovery, React/TSX component update
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DISC-01 | Daemon startup scans common directories for agent CLI binaries (nvm, Volta, Homebrew, snap, flatpak, cargo, pipx, native installers, system paths) per platform | `AugmentServicePath()` in `internal/daemon/path.go` is the exact function to extend; current candidates list is missing snap, flatpak, cargo, Windows npm/pnpm paths |
| DISC-02 | Detected agent paths are added to daemon PATH so agents resolve via exec.LookPath | Already implemented in `AugmentServicePath()` via `os.Setenv("PATH", ...)`; extending DISC-01 automatically satisfies DISC-02 |
| DISC-03 | Tailscale binary location detected across platforms (Homebrew, system package, Windows default install) | Tailscale health check (`CheckHealth`) uses `local.Client` (socket-based, not binary-based) and already works for all required platforms; Windows needs `C:\Program Files\Tailscale` added to PATH for CLI binary access |
| INST-01 | Welcome screen macOS install command combines `brew tap` + `brew install --cask` into single copyable command | `WelcomeTab.tsx` macOS `<code>` block contains `brew install agenthub`; change to `brew tap scottkw/agenthub && brew install --cask agenthub` |
</phase_requirements>

## Summary

Phase 68 has two independent workstreams. The first (DISC-01/02/03) extends `AugmentServicePath()` in `internal/daemon/path.go` to cover additional install locations that the current implementation misses: snap `/snap/bin`, flatpak user/system export dirs, cargo `~/.cargo/bin`, and Windows paths for npm, pnpm, and the Tailscale default installer. The second (INST-01) updates one string in `WelcomeTab.tsx` and its companion test.

The existing architecture is well-designed for extension. `AugmentServicePath()` already uses the correct pattern: build a candidate list, stat each directory, skip non-existent ones, prepend existing ones to `PATH`. New paths follow the identical pattern. No new interfaces, no new files required — both requirements are small, targeted edits to existing code.

DISC-02 is already satisfied by the existing `os.Setenv("PATH", ...)` call in `AugmentServicePath()`. Any paths added for DISC-01 are automatically included in the PATH update.

**Primary recommendation:** Extend `AugmentServicePath()` with platform-gated candidates, add a Windows-only `path_windows.go` helper for `%APPDATA%`/`%LOCALAPPDATA%` env lookups, and update the WelcomeTab string and test.

## Current Architecture

### `internal/daemon/path.go` — `AugmentServicePath()` [VERIFIED: codebase]

Called from two places:
- `main.go:57` — GUI mode startup (`runGUI`)
- `internal/daemon/process.go:29` — daemon mode startup (`runDaemonCore`)

Current candidate directories:
```go
candidates := []string{
    filepath.Join(home, ".local", "bin"),     // Anthropic native installer (macOS/Linux) + pipx
    filepath.Join(home, ".volta", "bin"),     // Volta (any platform)
    "/opt/homebrew/bin",                      // macOS ARM Homebrew
    "/usr/local/bin",                         // macOS Intel Homebrew, Tailscale
    "/home/linuxbrew/.linuxbrew/bin",         // Linux Homebrew
    nvmActiveBin(home),                       // nvm active version
}
```

The function prepends all existing directories to PATH; non-existent dirs are silently skipped. `nvmActiveBin` reads `~/.nvm/alias/default` to find the active node version directory.

### `internal/pty/detect.go` — `DetectCLIs()` [VERIFIED: codebase]

Uses `exec.LookPath` to find known CLI binaries. The four known CLIs are: `claude`, `codex`, `gemini`, `opencode`. These are found automatically once their directories are in PATH — no changes needed to `detect.go`.

### `internal/webserver/tailscale.go` — `CheckHealth()` [VERIFIED: codebase]

Uses `tailscale.com/client/local.Client` which calls `paths.DefaultTailscaledSocket()` to find the daemon socket. Socket paths by platform [VERIFIED: tailscale.com@v1.96.3/paths/paths.go]:
- macOS: `/var/run/tailscaled.socket`
- Linux: `/var/run/tailscale/tailscaled.sock`
- Windows: `\\.\pipe\ProtectedPrefix\Administrators\Tailscale\tailscaled`

All three paths align with the installation methods named in DISC-03 (Homebrew macOS, apt/dnf Linux, Windows default installer). **CheckHealth already works for all three — no socket path changes needed.**

### `frontend/src/components/WelcomeTab.tsx` [VERIFIED: codebase]

macOS install command on line 94:
```tsx
<code className="welcome-tab__code">brew install agenthub</code>
```

Companion test at `frontend/src/components/__tests__/WelcomeTab.test.tsx` line 33:
```ts
expect(raw).toContain('brew install agenthub')
```
This test still passes after the change because `brew install agenthub` is a substring of `brew tap scottkw/agenthub && brew install --cask agenthub`.

Homebrew tap `scottkw/homebrew-agenthub` exists and is public. [VERIFIED: GitHub API]

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` stdlib | Go stdlib | `exec.LookPath` for CLI detection | Already used in `detect.go` |
| `os` stdlib | Go stdlib | `os.Stat`, `os.Setenv`, `os.UserHomeDir` | Already used in `path.go` |
| `runtime` stdlib | Go stdlib | `runtime.GOOS` for platform branching | Already used throughout codebase |
| `filepath` stdlib | Go stdlib | `filepath.Join` for path construction | Already used in `path.go` |

### No New Dependencies Required
All work is standard library. The only question is how to handle Windows-specific `%APPDATA%` and `%LOCALAPPDATA%` env vars. Pattern is `os.Getenv("APPDATA")` — no import needed.

## Architecture Patterns

### Recommended Project Structure

No new files required. Changes are:
```
internal/daemon/
├── path.go          # extend AugmentServicePath candidates (unix/cross-platform paths)
├── path_windows.go  # add windowsExtraBins() helper returning Windows-only paths
└── path_test.go     # add tests for new candidates
frontend/src/components/
├── WelcomeTab.tsx                        # update macOS <code> string
└── __tests__/WelcomeTab.test.tsx         # add brew tap assertion (optional, existing still passes)
```

### Pattern 1: Platform-Gated Path Candidates

The cleanest approach is a build-tag separated helper for Windows-only paths. This avoids `runtime.GOOS` string comparisons inside `path.go` and keeps the Windows paths with their Windows-specific env var lookups.

**`path.go`** — add cross-platform candidates:
```go
// Source: codebase internal/daemon/path.go
func AugmentServicePath() {
    home, err := os.UserHomeDir()
    if err != nil { return }

    candidates := []string{
        filepath.Join(home, ".local", "bin"),       // existing
        filepath.Join(home, ".volta", "bin"),       // existing
        "/opt/homebrew/bin",                        // existing
        "/usr/local/bin",                           // existing
        "/home/linuxbrew/.linuxbrew/bin",           // existing
        nvmActiveBin(home),                         // existing
        // NEW: cross-platform
        filepath.Join(home, ".cargo", "bin"),       // cargo (any platform)
        // NEW: Linux-specific (no-op on macOS/Windows: dirs won't exist)
        "/snap/bin",                                // snap (Linux)
        "/var/lib/flatpak/exports/bin",             // flatpak system (Linux)
        filepath.Join(home, ".local", "share", "flatpak", "exports", "bin"), // flatpak user (Linux)
    }
    // append platform-specific paths (defined in path_windows.go / path_other.go)
    candidates = append(candidates, platformExtraBins()...)
    // ... rest unchanged
}
```

**`path_windows.go`** (build tag `//go:build windows`):
```go
package daemon

import (
    "os"
    "filepath"
)

func platformExtraBins() []string {
    var paths []string
    if appdata := os.Getenv("APPDATA"); appdata != "" {
        paths = append(paths, filepath.Join(appdata, "npm"))       // npm global on Windows
    }
    if local := os.Getenv("LOCALAPPDATA"); local != "" {
        paths = append(paths, filepath.Join(local, "pnpm"))        // pnpm on Windows
        paths = append(paths, filepath.Join(local, "Programs", "nodejs", "bin")) // node installer
    }
    // Tailscale Windows default installer path
    paths = append(paths, `C:\Program Files\Tailscale`)
    return paths
}
```

**`path_other.go`** (build tag `//go:build !windows`) — returns nil for non-Windows platforms.

### Pattern 2: WelcomeTab String Update

Single-line change to `WelcomeTab.tsx`:
```tsx
// Before:
<code className="welcome-tab__code">brew install agenthub</code>

// After:
<code className="welcome-tab__code">brew tap scottkw/agenthub && brew install --cask agenthub</code>
```

No copy button needed. Issue #14 explicitly says "just && them together." The `<code>` element is selectable text. No CSS changes needed.

### Anti-Patterns to Avoid

- **Do not use `runtime.GOOS` string comparisons in `path.go` for Windows paths.** The file runs on all platforms and importing env vars that don't exist on other OSes causes no harm, but separating Windows paths into `path_windows.go` keeps platform concerns clean and follows the existing codebase pattern (`process_windows.go`, `socket_windows.go`).
- **Do not add snap/flatpak paths via a snap-detection function.** Simply including `/snap/bin` and flatpak export dirs in the candidates list is sufficient — `os.Stat` will skip them on systems where they don't exist.
- **Do not add Tailscale snap socket fallback.** DISC-03 specifies Homebrew, system package, Windows installer — all handled by the existing socket path. Snap tailscale is out of scope.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tailscale health check | Custom socket path probing | `tailscale.com/client/local.Client` | Already in use; handles all socket path discovery automatically |
| CLI detection | Custom binary search | `exec.LookPath` (existing) | Already implemented in `detect.go`; PATH augmentation feeds it correctly |
| Clipboard copy | Custom clipboard API | `navigator.clipboard.writeText` (existing) | Already used in `SettingsTab.tsx`; but INST-01 doesn't need a copy button |

## Common Pitfalls

### Pitfall 1: Windows Path Separator in Hardcoded Paths
**What goes wrong:** Using `/` in Windows paths like `/Program Files/Tailscale` fails.
**Why it happens:** Go `filepath.Join` uses `\` on Windows, but string literals need `\` or raw string backtick syntax.
**How to avoid:** Use raw string literal `` `C:\Program Files\Tailscale` `` in `path_windows.go`, or `filepath.Join("C:\\Program Files", "Tailscale")`.
**Warning signs:** `os.Stat` returns false on a path that should exist.

### Pitfall 2: nvm Symlink Aliases
**What goes wrong:** `~/.nvm/alias/default` may contain a named alias like `lts/iron` instead of a version number.
**Why it happens:** nvm supports named aliases pointing to LTS releases.
**How to avoid:** The current `nvmActiveBin` implementation does a prefix match on the versions directory. Named aliases starting with `lts/` won't match `v` or a digit prefix, so they return `""` silently. This is acceptable for now — it's an existing limitation, not a regression.
**Warning signs:** Users on nvm lts aliases don't get detection.

### Pitfall 3: flatpak Exports Dir Not Created Until App Is Installed
**What goes wrong:** `/var/lib/flatpak/exports/bin` exists only when at least one flatpak app is installed. Similarly for user flatpak.
**Why it happens:** flatpak creates the exports dir lazily.
**How to avoid:** `os.Stat` skip is correct behavior — if no flatpak apps are installed, the dir doesn't exist and nothing is added. No fix needed.
**Warning signs:** None — this is correct behavior.

### Pitfall 4: PATH Deduplication Not Required
**What goes wrong:** Adding dirs already in PATH creates duplicates but no functional issue.
**Why it happens:** `AugmentServicePath` doesn't deduplicate.
**How to avoid:** No fix needed. Duplicate PATH entries are harmless. The function prepends, so user-installed paths take priority.
**Warning signs:** Very long PATH on machines with all tools installed — cosmetic only.

### Pitfall 5: WelcomeTab Test Passes Without Update
**What goes wrong:** The existing test `expect(raw).toContain('brew install agenthub')` continues to pass after the update because the new string contains the old substring.
**Why it happens:** Substring containment.
**How to avoid:** Optionally add a new test for `brew tap scottkw/agenthub` to verify the tap command is present. The test file needs updating to reflect the new expected behavior.
**Warning signs:** Tests pass but instruction is still wrong.

## Code Examples

### Verified Path Augmentation Pattern (from codebase)
```go
// Source: internal/daemon/path.go (verified)
candidates := []string{
    filepath.Join(home, ".local", "bin"),
    filepath.Join(home, ".volta", "bin"),
    "/opt/homebrew/bin",
    "/usr/local/bin",
    "/home/linuxbrew/.linuxbrew/bin",
    nvmActiveBin(home),
}
current := os.Getenv("PATH")
var extra []string
for _, dir := range candidates {
    if dir == "" { continue }
    if _, err := os.Stat(dir); err == nil {
        extra = append(extra, dir)
    }
}
if len(extra) > 0 {
    _ = os.Setenv("PATH", strings.Join(extra, string(os.PathListSeparator))+string(os.PathListSeparator)+current)
}
```

### Windows Build-Tag File Pattern (from codebase)
```go
// Source: internal/daemon/process_windows.go (verified)
//go:build windows

package daemon
```

### Test Pattern for New Path Candidates (from path_test.go)
```go
// Source: internal/daemon/path_test.go (verified)
func TestAugmentServicePath_AddsExistingDirs(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("uses Unix PATH separator")
    }
    home := t.TempDir()
    t.Setenv("HOME", home)
    original := "/usr/bin:/bin"
    t.Setenv("PATH", original)

    // Create the target dir
    targetBin := filepath.Join(home, ".cargo", "bin")
    if err := os.MkdirAll(targetBin, 0755); err != nil {
        t.Fatalf("MkdirAll: %v", err)
    }

    AugmentServicePath()

    got := os.Getenv("PATH")
    if !strings.Contains(got, targetBin) {
        t.Errorf("PATH should contain %s, got %s", targetBin, got)
    }
}
```

## Tailscale Socket Path Analysis

This section documents why no socket changes are needed for DISC-03.

| Install Method | Socket Path | Covered by DefaultTailscaledSocket()? |
|----------------|-------------|---------------------------------------|
| macOS Homebrew (brew install --cask tailscale) | `/var/run/tailscaled.socket` | YES [VERIFIED: tailscale.com@v1.96.3/paths/paths.go] |
| macOS App Store | TCP localhost via sameuserproof | YES — `local.Client.defaultDialer` handles this automatically [VERIFIED: safesocket_darwin.go] |
| Linux apt/dnf system package | `/var/run/tailscale/tailscaled.sock` | YES [VERIFIED: paths.go] |
| Windows default installer | `\\.\pipe\ProtectedPrefix\Administrators\Tailscale\tailscaled` | YES [VERIFIED: paths.go] |
| Linux snap (out of scope) | `/var/snap/tailscale/common/socket/tailscaled.sock` | NO — but not required by DISC-03 spec |

**Conclusion:** `CheckHealth()` works for all DISC-03-required install methods with no changes. DISC-03 only needs the Windows tailscale binary path added to PATH for CLI accessibility.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `brew install agenthub` | `brew tap scottkw/agenthub && brew install --cask agenthub` | Phase 68 (now) | User can copy one command instead of running two |
| PATH covers nvm/volta/Homebrew | PATH also covers snap/flatpak/cargo/Windows npm/pnpm | Phase 68 (now) | Agents installed via additional package managers are detected automatically |

## Open Questions

1. **Volta on Windows**
   - What we know: `~/.volta/bin` is already in candidates; `os.UserHomeDir()` returns `%USERPROFILE%` on Windows, so `filepath.Join(home, ".volta", "bin")` = `C:\Users\user\.volta\bin`
   - What's unclear: Whether Volta on Windows uses exactly this path or a different convention
   - Recommendation: Include it as-is; `os.Stat` will skip if it doesn't exist. [ASSUMED]

2. **npm global on Windows custom prefix**
   - What we know: Default `%APPDATA%\npm` is the standard Windows npm global bin
   - What's unclear: Users who ran `npm config set prefix` use a different location
   - Recommendation: Only add the default; non-standard prefixes require user to set PATH manually. Acceptable limitation.

3. **Agent installs via Windows WinGet**
   - What we know: WinGet installs to various locations; `@openai/codex` can be installed via `winget install Codex`
   - What's unclear: Where WinGet places the binary (depends on installer type)
   - Recommendation: Not addressed in this phase; most WinGet packages install to `%LOCALAPPDATA%\Programs\` which is covered by the node installer path, or to system PATH. [ASSUMED]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Volta on Windows stores binaries in `%USERPROFILE%\.volta\bin` | Standard Stack | Low — `os.Stat` skips non-existent dirs harmlessly |
| A2 | WinGet-installed agents end up in system PATH or in covered locations | Open Questions | Low — out of scope for this phase |
| A3 | snap tailscale on Linux uses the standard socket path via layout binding | Tailscale Socket Path Analysis | Low — snap tailscale is explicitly out of scope (DISC-03 says "system package manager") |

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go compiler | All Go changes | ✓ | (project standard) | — |
| pnpm | Frontend test run | ✓ | (project standard) | — |
| vitest | Frontend tests | ✓ | v4.1.0 | — |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` stdlib |
| Framework (Frontend) | vitest v4.1.0 |
| Quick run (Go) | `go test ./internal/daemon/...` |
| Quick run (Frontend) | `cd frontend && pnpm test` |
| Full suite | `go test ./... && cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DISC-01 | snap `/snap/bin` added when it exists | unit | `go test ./internal/daemon/ -run TestAugmentServicePath_Snap` | ❌ Wave 0 |
| DISC-01 | flatpak user dir added when it exists | unit | `go test ./internal/daemon/ -run TestAugmentServicePath_Flatpak` | ❌ Wave 0 |
| DISC-01 | cargo `~/.cargo/bin` added when it exists | unit | `go test ./internal/daemon/ -run TestAugmentServicePath_Cargo` | ❌ Wave 0 |
| DISC-01 | Windows `%APPDATA%\npm` added when it exists | unit | `go test ./internal/daemon/ -run TestPlatformExtraBins` | ❌ Wave 0 |
| DISC-02 | All detected paths prepended to PATH | unit | covered by DISC-01 tests above | — |
| DISC-03 | No socket changes needed (verified by analysis) | n/a | existing tailscale tests pass | ✅ |
| DISC-03 | Windows Tailscale path included in platformExtraBins | unit | `go test ./internal/daemon/ -run TestPlatformExtraBins` | ❌ Wave 0 |
| INST-01 | macOS command contains `brew tap scottkw/agenthub` | unit | `cd frontend && pnpm test -- WelcomeTab` | ❌ needs update |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/... && cd frontend && pnpm test`
- **Per wave merge:** `go test ./... && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/path_test.go` — add `TestAugmentServicePath_Cargo`, `TestAugmentServicePath_Snap`, `TestAugmentServicePath_FlatpakUser` following existing `TestAugmentServicePath_AddsExistingDirs` pattern
- [ ] `internal/daemon/path_windows.go` — new file with `platformExtraBins()` for Windows paths
- [ ] `internal/daemon/path_other.go` — new file with `platformExtraBins() []string { return nil }` for non-Windows
- [ ] `frontend/src/components/__tests__/WelcomeTab.test.tsx` line 33 — update to assert `brew tap scottkw/agenthub`

## Sources

### Primary (HIGH confidence)
- Codebase `internal/daemon/path.go` — full `AugmentServicePath` implementation
- Codebase `internal/pty/detect.go` — `DetectCLIs` and `knownCLIs`
- Codebase `internal/webserver/tailscale.go` — `CheckHealth` using `local.Client`
- `tailscale.com@v1.96.3/paths/paths.go` (local module cache) — `DefaultTailscaledSocket()` for all platforms
- `tailscale.com@v1.96.3/safesocket/safesocket_darwin.go` — macOS App Store socket handling
- GitHub API — `scottkw/homebrew-agenthub` tap existence confirmed
- GitHub issue #14 — "Just && them together" confirms INST-01 scope (no copy button)
- GitHub issue #15 — "look for all locations sub apps are installed in" confirms DISC-01 scope

### Secondary (MEDIUM confidence)
- WebSearch: canonical/tailscale-snap — snap socket path `/var/snap/tailscale/common/socket/tailscaled.sock` (out of scope for DISC-03)
- WebSearch: Claude Code, Codex, Gemini CLI, OpenCode install locations

## Metadata

**Confidence breakdown:**
- DISC-01 path candidates: HIGH — based on direct codebase analysis and official package manager conventions
- DISC-02 satisfaction: HIGH — code already does `os.Setenv`; this is verified
- DISC-03 socket analysis: HIGH — verified against `tailscale.com@v1.96.3` source
- INST-01 change: HIGH — trivial string change with verified tap name
- Windows paths: MEDIUM — standard conventions, not verified against actual Windows machine

**Research date:** 2026-04-11
**Valid until:** 2026-05-11 (stable domain)
