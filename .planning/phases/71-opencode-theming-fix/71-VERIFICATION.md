---
phase: 71-opencode-theming-fix
verified: 2026-04-13T21:15:00Z
status: gaps_found
score: 2/3 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Selecting a theme in Settings > Appearance applies the theme colors to an ACTIVE OpenCode terminal session"
    status: failed
    reason: "OpenCode only re-queries terminal palette on SIGUSR2 or dark/light mode change. AgentHub does not send either signal when user changes theme. Active sessions retain prior theme palette."
    artifacts:
      - path: "internal/daemon/engine.go"
        issue: "CreateSession injects OPENCODE_TUI_CONFIG at spawn time only; no mechanism to signal running sessions"
    missing:
      - "SessionEngine.NotifyThemeChange() method that sends SIGUSR2 to active opencode sessions"
      - "Wails binding from frontend theme-change handler to NotifyThemeChange()"
      - "Test coverage for SIGUSR2 delivery to running opencode processes"
human_verification:
  - test: "After implementing SIGUSR2 broadcasting, switch theme in Settings > Appearance while an OpenCode session is active"
    expected: "OpenCode session repaints with new theme colors without restart"
    why_human: "Visual theme repaint cannot be verified programmatically; requires observing actual terminal color rendering"
---

# Phase 71: OpenCode Theming Fix Verification Report

**Phase Goal:** OpenCode terminal sessions respect the globally selected theme, matching the behavior of the other three supported agents
**Verified:** 2026-04-13T21:15:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Selecting a theme in Settings > Appearance applies the theme colors to an ACTIVE OpenCode terminal session | FAILED | UAT (Plan 04) confirmed: active sessions retain prior theme. No SIGUSR2 or signal mechanism exists in codebase. `grep -r SIGUSR2 internal/` returns 0 matches. |
| 2 | A newly created OpenCode session starts with the currently selected theme applied | VERIFIED | (a) `engine.go:89-91` injects `OPENCODE_TUI_CONFIG` env var for `cli == "opencode"` sessions. (b) `engine.go:61` calls `ensureOpenCodeTUIConfig(daemonConfigDir())` at engine init, writing `{"theme":"system"}` to managed config. (c) `native.go:46` merges `req.Env` into child process env via `mergeEnv`. (d) UAT (Plan 04) confirmed: new sessions start with current theme. (e) Tests pass: `TestCreateSession_OpenCodeEnv` GREEN, `TestOpenCodeTUIConfig` GREEN. |
| 3 | OpenCode theme behavior matches Claude Code, Codex, and Gemini CLI -- same theme produces visually consistent results across all four agents | VERIFIED | (a) Same xterm.js theme palette is queried by OpenCode at startup via OSC 10/11/4 (documented in 71-03-SUMMARY.md corrected interpretation). (b) UAT (Plan 04) confirmed SC-3 at session-start time. (c) `TestCreateSession_OpenCodeEnv` asserts non-opencode CLIs (claude, codex) do NOT receive `OPENCODE_TUI_CONFIG`, preventing interference with their existing theming. |

**Score:** 2/3 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/engine.go` | ensureOpenCodeTUIConfig helper + per-CLI env injection in CreateSession | VERIFIED | L1: EXISTS (245 lines). L2: SUBSTANTIVE -- `ensureOpenCodeTUIConfig` (line 51), `daemonConfigDir` (line 37), `OPENCODE_TUI_CONFIG` injection (line 90), `opencodeTUIConfig` struct field (line 21), `Env` field populated in CreateRequest (line 96). No TODOs or placeholders. L3: WIRED -- called by `NewSessionEngine` (line 61), consumed by `native.go:46` mergeEnv via `CreateRequest.Env`. |
| `internal/daemon/engine_test.go` | GREEN tests for env injection and managed tui.json | VERIFIED | L1: EXISTS (345 lines). L2: SUBSTANTIVE -- `TestCreateSession_OpenCodeEnv` (line 258, tests opencode/claude/codex env behavior), `TestOpenCodeTUIConfig` (line 320, tests file content and idempotency), `spyBackend` (line 237). L3: WIRED -- tests import `engine.go` functions in `package daemon`. All tests pass (`go test ./internal/daemon/ -short` exits 0). |
| `internal/daemon/opencode_ansi_test.go` | Integration test validating ANSI capture | VERIFIED | L1: EXISTS (137 lines). L2: SUBSTANTIVE -- regex patterns for 24-bit RGB and ANSI palette, go-pty PTY allocation, skip conditions for CI/short mode. L3: WIRED -- calls `ensureOpenCodeTUIConfig` from engine.go. Compiles and runs (diagnostic mode, no hard assertions by design). |
| `internal/pty/detect_test.go` | Fixed test for 5 known CLIs | VERIFIED | L1: EXISTS (91 lines). L2: SUBSTANTIVE -- `expected` slice includes `"tailscale"` (line 74). L3: WIRED -- tests `knownCLIs` from `detect.go`. Test passes: `go test ./internal/pty/ -run TestKnownCLIs_HasExpectedEntries` exits 0. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `engine.go` CreateSession | `native.go` mergeEnv | `CreateRequest.Env` field | WIRED | `engine.go:96` sets `Env: env`; `native.go:46` consumes `req.Env` in `mergeEnv(os.Environ(), req.Env, ...)` |
| `engine.go` NewSessionEngine | `~/.config/agenthub/opencode-tui.json` | `ensureOpenCodeTUIConfig` writes managed config | WIRED | `engine.go:61` calls `ensureOpenCodeTUIConfig(daemonConfigDir())`; file confirmed on disk at `~/Library/Application Support/agenthub/opencode-tui.json` with content `{"$schema":"https://opencode.ai/tui.json","theme":"system"}` |
| `engine_test.go` tests | `engine.go` SessionEngine | Tests assert on SessionEngine behavior | WIRED | Tests in `package daemon` access unexported `backend` field (line 261), `opencodeTUIConfig` field (line 274), and `ensureOpenCodeTUIConfig` function (line 322) |
| Settings > Appearance theme picker | OpenCode terminal panel | xterm.js theme propagation for active sessions | NOT_WIRED | No mechanism to signal active OpenCode sessions on theme change. SIGUSR2 broadcasting not implemented. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `engine.go` CreateSession | `env []string` | `e.opencodeTUIConfig` (set at init from `ensureOpenCodeTUIConfig`) | Yes -- path to real file written to disk | FLOWING |
| `opencode-tui.json` | JSON content | Hardcoded constant in `ensureOpenCodeTUIConfig` | Yes -- `{"$schema":"...","theme":"system"}` verified on disk | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Daemon tests pass | `go test ./internal/daemon/ -count=1 -short` | All 27 tests PASS, exit 0 | PASS |
| PTY tests pass | `go test ./internal/pty/ -count=1 -run TestKnownCLIs_HasExpectedEntries` | PASS, exit 0 | PASS |
| Daemon package builds | `go build ./internal/daemon/` | BUILD OK | PASS |
| Managed config file exists on disk | `cat ~/Library/Application Support/agenthub/opencode-tui.json` | `{"$schema":"https://opencode.ai/tui.json","theme":"system"}` | PASS |
| All commits verified | `git log --oneline cc32c5d 63da45a aa9544f dc73386 a02dd75` | All 5 commits exist | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| THM-05 | 71-01, 71-02, 71-03, 71-04 | The theme selected in Settings > Appearance is applied to OpenCode terminal sessions, matching the behavior for Claude Code, Codex, and Gemini CLI sessions | PARTIAL | New sessions inherit theme (SC-2, SC-3 pass). Active sessions do not update on theme change (SC-1 fails). THM-05 does not distinguish new vs active sessions, but ROADMAP SC-1 explicitly requires active session repainting. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `engine.go` | 54 | `_ = os.WriteFile(path, content, 0644)` -- silent error discard | Warning | If write fails, env var points to missing file; OpenCode falls back to default theme silently. Code review WR-01 flagged this. Not a phase blocker -- function is called at engine init and failure mode is graceful degradation (no crash). |
| `opencode_ansi_test.go` | 125-127 | Diagnostic-only test with `t.Logf` instead of `t.Errorf` for A1 violation | Info | By design -- go-pty lacks OSC responder, making hard assertions inappropriate. Plan 04 UAT is the authoritative validator. |

### Human Verification Required

### 1. Live Theme Switch on Active OpenCode Session (SC-1)

**Test:** After implementing SIGUSR2 broadcasting, launch AgentHub, create an OpenCode session, switch theme in Settings > Appearance, and return to the OpenCode tab.
**Expected:** OpenCode session repaints with the newly selected theme colors without needing a session restart.
**Why human:** Visual theme repaint requires observing actual terminal color rendering in the running application. No programmatic test can verify the user-visible outcome.

### Gaps Summary

**One gap blocks full phase goal achievement: SC-1 (live theme switching on active sessions).**

The implementation successfully delivers new-session theme inheritance (SC-2) and cross-agent consistency (SC-3) through the `OPENCODE_TUI_CONFIG` env injection + managed `opencode-tui.json` mechanism. However, active sessions cannot respond to theme changes because:

1. OpenCode's `"system"` theme queries the terminal palette via OSC escape sequences only at startup or on `SIGUSR2` signal.
2. `generateSystem()` in OpenCode emits 24-bit RGB colors (not ANSI palette indices), so xterm.js cannot retroactively remap colors at the terminal level.
3. AgentHub has no mechanism to send SIGUSR2 to running OpenCode processes when the user changes theme.

**Closure path (documented in 71-04-SUMMARY.md):**
- Add `SessionEngine.NotifyThemeChange()` that walks active sessions where `cli == "opencode"` and sends `syscall.SIGUSR2` to the child process PID.
- Wire frontend theme-change handler through Wails bindings to call `NotifyThemeChange()`.
- Add test coverage for signal delivery.
- This is a discrete follow-up phase, not an extension of the current implementation.

**No later phases in this milestone address this gap.** Phases 72 (UI contrast) and 73 (theme audit) do not cover live OpenCode theme switching. The gap requires a dedicated follow-up phase.

---

_Verified: 2026-04-13T21:15:00Z_
_Verifier: Claude (gsd-verifier)_
