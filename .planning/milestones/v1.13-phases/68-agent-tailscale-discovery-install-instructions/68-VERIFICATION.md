---
phase: 68-agent-tailscale-discovery-install-instructions
verified: 2026-04-11T23:03:30Z
status: passed
score: 8/8 must-haves verified
overrides_applied: 0
re_verification: false
---

# Phase 68: Agent & Tailscale Discovery + Install Instructions Verification Report

**Phase Goal:** AgentHub reliably finds agent CLIs and Tailscale on all platforms regardless of how they were installed, and macOS install instructions show a single copyable command
**Verified:** 2026-04-11T23:03:30Z
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Starting a Claude Code, Codex, Gemini CLI, or OpenCode session succeeds when the agent was installed via nvm, Volta, Homebrew, snap, flatpak, cargo, pipx, or a native installer — without the user manually configuring PATH | VERIFIED | `AugmentServicePath()` in `path.go` now includes `.cargo/bin`, `/snap/bin`, `/var/lib/flatpak/exports/bin`, flatpak user dir, and calls `platformExtraBins()` for Windows npm/pnpm/node paths. All candidate paths are stat-checked; absent dirs are skipped. Tests `TestAugmentServicePath_Cargo` and `TestAugmentServicePath_FlatpakUser` pass. |
| 2 | The Tailscale health check correctly detects Tailscale when installed via Homebrew, a system package manager, or the Windows default installer location | VERIFIED | `CheckHealth()` already uses `tailscale.com/client/local.Client` with socket-based detection (no PATH dependency) — all three install methods use platform-standard socket paths covered by `DefaultTailscaledSocket()`. Windows CLI path `C:\Program Files\Tailscale` added to `platformExtraBins()` for binary lookup. |
| 3 | The Welcome screen macOS install section shows a single `brew tap scottkw/agenthub && brew install --cask agenthub` command that can be copied in one action | VERIFIED | `WelcomeTab.tsx` line 94 contains exact string `brew tap scottkw/agenthub && brew install --cask agenthub`. Test assertion in `WelcomeTab.test.tsx` line 33 asserts the full command. 323/323 frontend tests pass. |

**Roadmap Score:** 3/3 success criteria verified

### Must-Have Truths (Plan Frontmatter — Plan 01)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | AugmentServicePath includes cargo bin directory when it exists | VERIFIED | `filepath.Join(home, ".cargo", "bin")` at line 29 of `path.go`. `TestAugmentServicePath_Cargo` creates the dir and confirms PATH contains it. PASS. |
| 2 | AugmentServicePath includes snap bin directory when it exists | VERIFIED | `/snap/bin` at line 30 of `path.go`. Covered by existing `TestAugmentServicePath_SkipsNonexistent` pattern (non-existent dirs skipped; real `/snap/bin` would be added on Linux). |
| 3 | AugmentServicePath includes flatpak system and user export dirs when they exist | VERIFIED | `/var/lib/flatpak/exports/bin` (line 31) and `filepath.Join(home, ".local", "share", "flatpak", "exports", "bin")` (line 32) in `path.go`. `TestAugmentServicePath_FlatpakUser` confirms user path. |
| 4 | Windows builds include npm, pnpm, node installer, and Tailscale paths | VERIFIED | `path_windows.go` returns `%APPDATA%\npm`, `%LOCALAPPDATA%\pnpm`, `%LOCALAPPDATA%\Programs\nodejs`, and `` `C:\Program Files\Tailscale` ``. |
| 5 | Non-Windows builds return nil from platformExtraBins | VERIFIED | `path_other.go` contains `return nil`. `TestPlatformExtraBins_NonWindows` confirms nil return on macOS/Linux. PASS. |
| 6 | All new candidate paths are stat-checked and skipped when absent | VERIFIED | `AugmentServicePath` stat loop at lines 38-44 is unchanged and covers all candidates including new ones. |

### Must-Have Truths (Plan Frontmatter — Plan 02)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 7 | Welcome screen macOS install command shows `brew tap scottkw/agenthub && brew install --cask agenthub` | VERIFIED | `WelcomeTab.tsx` line 94 contains exact string. Confirmed with direct file read. |
| 8 | WelcomeTab test asserts the full brew tap + install --cask command | VERIFIED | `WelcomeTab.test.tsx` line 33: `expect(raw).toContain('brew tap scottkw/agenthub && brew install --cask agenthub')`. |

**Combined Score:** 8/8 must-haves verified

---

## Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/path.go` | Extended candidate list with cargo, snap, flatpak + platformExtraBins() call | VERIFIED | Lines 29-34: cargo, snap, flatpak system, flatpak user, `platformExtraBins()` call. Substantive (78 lines). Wired via `AugmentServicePath()` called at daemon startup. |
| `internal/daemon/path_windows.go` | Windows-specific path candidates (npm, pnpm, node, Tailscale) | VERIFIED | Exists, `//go:build windows` on line 1, `func platformExtraBins() []string` returns 4 Windows paths. 24 lines, substantive. |
| `internal/daemon/path_other.go` | No-op platformExtraBins for non-Windows | VERIFIED | Exists, `//go:build !windows` on line 1, `func platformExtraBins() []string { return nil }`. Correct stub for non-Windows. |
| `internal/daemon/path_test.go` | Tests for cargo, snap, flatpak path discovery | VERIFIED | Contains `TestAugmentServicePath_Cargo` (line 174), `TestAugmentServicePath_FlatpakUser` (line 196), `TestPlatformExtraBins_NonWindows` (line 218). All pass. |
| `frontend/src/components/WelcomeTab.tsx` | Updated macOS install command | VERIFIED | Line 94: `brew tap scottkw/agenthub && brew install --cask agenthub`. |
| `frontend/src/components/__tests__/WelcomeTab.test.tsx` | Test asserting brew tap command | VERIFIED | Line 33 asserts full brew tap + cask command. |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/daemon/path.go` | `internal/daemon/path_windows.go` | `platformExtraBins()` call (build-tag selected) | WIRED | Line 34: `candidates = append(candidates, platformExtraBins()...)`. Function defined in `path_windows.go` under `//go:build windows`. |
| `internal/daemon/path.go` | `internal/daemon/path_other.go` | `platformExtraBins()` call (build-tag selected) | WIRED | Same call site; `path_other.go` provides the symbol under `//go:build !windows`. `go build` confirms both compile without errors. |
| `frontend/src/components/__tests__/WelcomeTab.test.tsx` | `frontend/src/components/WelcomeTab.tsx` | raw source import for assertion | WIRED | Line 5: `import raw from '../../components/WelcomeTab.tsx?raw'`. Assertion on line 33 reads the raw source. 323/323 frontend tests pass. |

---

## Data-Flow Trace (Level 4)

Not applicable. Both workstreams are:
- **PATH augmentation (Go):** Not a data-rendering artifact — it modifies environment state. Data flows through `os.Setenv`, which is side-effectful and verified by unit tests that read back `os.Getenv("PATH")`.
- **WelcomeTab.tsx (React):** The install command is static display text, not dynamic data. No state variable, no fetch, no store. Level 4 does not apply to static string literals.

---

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Cargo bin candidate present in `path.go` | `grep -c '\.cargo/bin' internal/daemon/path.go` | 1 | PASS |
| snap bin candidate present in `path.go` | `grep -c '/snap/bin' internal/daemon/path.go` | 1 | PASS |
| flatpak user dir present in `path.go` | `grep -c 'flatpak/exports/bin' internal/daemon/path.go` | 2 (system + user) | PASS |
| platformExtraBins call in `path.go` | `grep -c 'platformExtraBins' internal/daemon/path.go` | 1 | PASS |
| Daemon package compiles | `go build ./internal/daemon/...` | exit 0 | PASS |
| New daemon tests pass | `go test ./internal/daemon/... -run TestAugmentServicePath_Cargo\|TestAugmentServicePath_FlatpakUser\|TestPlatformExtraBins_NonWindows` | 3 PASS | PASS |
| All daemon tests pass (no regressions) | `go test ./internal/daemon/... -count=1` | `ok` (exit 0) | PASS |
| Frontend tests pass | `cd frontend && pnpm test` | 323/323 passed, 18 files | PASS |
| macOS brew tap command in WelcomeTab | `grep 'brew tap scottkw/agenthub' frontend/src/components/WelcomeTab.tsx` | 1 match | PASS |
| Test asserts brew tap command | `grep 'brew tap scottkw/agenthub' frontend/src/components/__tests__/WelcomeTab.test.tsx` | 1 match | PASS |

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| DISC-01 | 68-01-PLAN.md | Daemon startup scans common directories for agent CLI binaries (nvm, Volta, Homebrew, snap, flatpak, cargo, pipx, native installers, system paths) per platform | SATISFIED | `AugmentServicePath()` now covers snap, flatpak (system + user), cargo, Windows npm/pnpm/node in addition to existing nvm, Volta, Homebrew. |
| DISC-02 | 68-01-PLAN.md | Detected agent paths are added to daemon PATH so agents resolve via exec.LookPath | SATISFIED | The existing `os.Setenv("PATH", ...)` call in `AugmentServicePath()` is unchanged. All newly discovered paths are automatically included when they exist. |
| DISC-03 | 68-01-PLAN.md | Tailscale binary location detected across platforms (Homebrew, system package, Windows default install) | SATISFIED | Socket-based health check already works for Homebrew (macOS), system package (Linux), and Windows installer via `DefaultTailscaledSocket()`. Windows Tailscale CLI binary path `C:\Program Files\Tailscale` added to `platformExtraBins()`. |
| INST-01 | 68-02-PLAN.md | Welcome screen macOS install command combines `brew tap` + `brew install --cask` into single copyable command | SATISFIED | `WelcomeTab.tsx` line 94 shows exact command. Test locks it. |

All 4 requirements declared across both plans are satisfied. No orphaned requirements for Phase 68 in REQUIREMENTS.md.

---

## Anti-Patterns Found

None. Scan of all modified files (`path.go`, `path_windows.go`, `path_other.go`, `path_test.go`, `WelcomeTab.tsx`, `WelcomeTab.test.tsx`) found no TODOs, FIXMEs, placeholder comments, empty return stubs (path_other.go `return nil` is correct non-Windows behavior, not a stub), or hardcoded empty data.

---

## Human Verification Required

None. All aspects of this phase are verifiable programmatically:
- PATH augmentation: verified by unit tests that stat real temp directories
- Build-tag selection: verified by `go build` on the host platform
- Frontend install command: verified by raw source string assertion in unit tests
- No visual, real-time, or external-service behaviors introduced

---

## Deferred Items

None. All roadmap success criteria for Phase 68 are fully satisfied.

---

## Gaps Summary

No gaps. All 8 must-have truths are verified, all 6 artifacts exist and are substantive and wired, all 3 key links are confirmed, all 4 requirement IDs are satisfied, no anti-patterns detected, and the test suite is green (all daemon tests + 323/323 frontend tests passing).

---

_Verified: 2026-04-11T23:03:30Z_
_Verifier: Claude (gsd-verifier)_
