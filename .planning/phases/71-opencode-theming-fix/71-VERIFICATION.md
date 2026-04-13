---
phase: 71-opencode-theming-fix
verified: 2026-04-13T22:30:00Z
status: human_needed
score: 3/3 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 2/3
  gaps_closed:
    - "Selecting a theme in Settings > Appearance applies the theme colors to an active OpenCode terminal session (SC-1 closed by Plan 05 SIGUSR2 broadcast pipeline)"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Launch AgentHub, create an OpenCode session, wait for colored TUI to appear, switch theme in Settings > Appearance, return to the OpenCode tab"
    expected: "OpenCode session repaints with the newly selected theme colors without needing a session restart"
    why_human: "Visual theme repaint requires observing actual terminal color rendering in the running application. The SIGUSR2 delivery is verified by automated integration test (TestNotifyThemeChange_RealProcess_Integration passes), but whether OpenCode's re-query of xterm.js produces the correct visual output requires a human with eyes on the terminal."
---

# Phase 71: OpenCode Theming Fix Verification Report

**Phase Goal:** OpenCode terminal sessions respect the globally selected theme, matching the behavior of the other three supported agents
**Verified:** 2026-04-13T22:30:00Z
**Status:** human_needed
**Re-verification:** Yes -- after Plan 05 gap closure (SIGUSR2 broadcast pipeline)

## Summary

Plan 05 implemented the complete SIGUSR2 broadcast pipeline that was missing when the initial verification found SC-1 failing. All automated evidence now supports SC-1: the signal chain from frontend theme picker through Wails binding through daemon HTTP API through engine broadcast to per-session SIGUSR2 delivery is fully implemented, wired, and tested. One human verification step remains to confirm the visual repaint works as expected in the live application.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Selecting a theme in Settings > Appearance applies the theme colors to an active OpenCode terminal session | VERIFIED (automated) | (a) `frontend/src/App.tsx:98` calls `NotifyThemeChange().catch(...)` fire-and-forget inside `handleThemeChange`. (b) `app.go:389-394` Wails binding nil-safely calls `a.client.NotifyThemeChange()`. (c) `client.go:169-179` posts `POST /theme/notify` over Unix socket. (d) `api.go:226-232` handler calls `engine.NotifyThemeChange()` returning 204. (e) `engine.go:251-270` walks registry, filters `sessionCLIs[id] == "opencode"`, calls `signalThemeChange(sess)`. (f) `notify_theme_unix.go:14` calls `sess.Signal(syscall.SIGUSR2)`. (g) `session.go:88-95` nil-safely delivers signal to child process. (h) Integration test `TestNotifyThemeChange_RealProcess_Integration` spawns `/bin/sh` with USR2 trap, calls `NotifyThemeChange`, asserts `SIGUSR2_RECEIVED` appears in PTY output -- PASSES in 0.11s. Visual outcome requires human verification. |
| 2 | A newly created OpenCode session starts with the currently selected theme applied | VERIFIED | (a) `engine.go:89-91` injects `OPENCODE_TUI_CONFIG` env var for `cli == "opencode"` sessions at PTY spawn. (b) `engine.go:61` calls `ensureOpenCodeTUIConfig(daemonConfigDir())` at engine init, writing `{"theme":"system"}` to managed config. (c) `native.go:46` merges `req.Env` into child process env via `mergeEnv`. (d) UAT (Plan 04) confirmed: new sessions start with current theme. (e) `TestCreateSession_OpenCodeEnv` and `TestOpenCodeTUIConfig` both GREEN. |
| 3 | OpenCode theme behavior matches Claude Code, Codex, and Gemini CLI -- same theme produces visually consistent results across all four agents | VERIFIED | (a) Same xterm.js theme palette queried by OpenCode at startup via OSC 10/11/4 (documented in 71-03-SUMMARY.md corrected interpretation). (b) UAT (Plan 04) confirmed SC-3 at session-start time. (c) SIGUSR2 mechanism (Plan 05) extends consistency to live theme switching for active sessions. (d) `TestCreateSession_OpenCodeEnv` asserts non-opencode CLIs (claude, codex) do NOT receive `OPENCODE_TUI_CONFIG`. |

**Score:** 3/3 truths verified (automated); 1 human verification needed for SC-1 visual confirmation

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/pty/session.go` | `Session.Signal(sig os.Signal) error` method | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- nil-safe guard on `cmd` and `cmd.Process`, calls `cmd.Process.Signal(sig)`. L3: WIRED -- called by `notify_theme_unix.go:14` via `signalThemeChange(sess)` which is invoked from `engine.go:NotifyThemeChange`. Tests: `TestSession_Signal_NilCmd`, `TestSession_Signal_NilProcess` both PASS. |
| `internal/pty/session_test.go` | Signal edge-case tests | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- covers nil-cmd and nil-process branches. L3: WIRED -- tests in `package pty`. Both tests PASS. |
| `internal/daemon/engine.go` | `NotifyThemeChange` method + `sessionCLIs` tracking | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- `sessionCLIs` field (line 30), populated in `CreateSession` (line 118), cleaned in `KillSession` (line 180), `NotifyThemeChange` method (line 251-270) walks registry, filters to opencode, calls `signalThemeChange`. L3: WIRED -- called by `api.go:227`. |
| `internal/daemon/notify_theme_unix.go` | POSIX SIGUSR2 signal helper | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- build tag `!windows`, imports `syscall`, calls `sess.Signal(syscall.SIGUSR2)`. L3: WIRED -- called by `engine.go:NotifyThemeChange` on POSIX platforms. |
| `internal/daemon/notify_theme_windows.go` | Windows no-op helper | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- `//go:build windows`, no-op body, correct comment explaining SIGUSR2 doesn't exist on Windows. L3: WIRED -- provides same `signalThemeChange` signature as unix variant via build tags. |
| `internal/daemon/api.go` | `POST /theme/notify` handler | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- route registered at line 61, handler at line 226-232, calls `engine.NotifyThemeChange`, returns 204 on success. L3: WIRED -- registered in mux via `a.mux.HandleFunc`. Tests `TestHandleNotifyThemeChange`, `TestHandleNotifyThemeChange_WithSessions` PASS. |
| `internal/daemon/client.go` | `DaemonClient.NotifyThemeChange()` method | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- posts `POST /theme/notify` over Unix socket client. L3: WIRED -- called by `app.go:393`. Test `TestClientNotifyThemeChange` PASS. |
| `app.go` | `App.NotifyThemeChange()` Wails binding | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- nil-safe (`if a.client == nil { return nil }`), delegates to `a.client.NotifyThemeChange()`. L3: WIRED -- called from `frontend/src/App.tsx:98` via Wails. `TestNotifyThemeChange` PASS. |
| `frontend/src/App.tsx` | `handleThemeChange` calls `NotifyThemeChange` | VERIFIED | L1: EXISTS. L2: SUBSTANTIVE -- imports `NotifyThemeChange` from wailsjs (line 24), calls `NotifyThemeChange().catch(err => console.warn(...))` fire-and-forget inside `handleThemeChange` (line 98). L3: WIRED -- `handleThemeChange` used as `onThemeChange` prop at line 625. |
| `frontend/src/wailsjs/go/main/App.d.ts` | TypeScript declaration for NotifyThemeChange | VERIFIED | `NotifyThemeChange(): Promise<void>` at line 98. |
| `frontend/src/wailsjs/go/main/App.js` | Wails runtime stub for NotifyThemeChange | VERIFIED | `NotifyThemeChange` exported at line 59 as `Call('main.App.NotifyThemeChange', [])`. |
| `internal/daemon/engine.go` | `ensureOpenCodeTUIConfig` + per-CLI env injection (Plan 02) | VERIFIED | Unchanged from initial verification -- still present and wired. |
| `internal/daemon/engine_test.go` | All theme-related tests GREEN | VERIFIED | `TestCreateSession_OpenCodeEnv`, `TestOpenCodeTUIConfig`, `TestNotifyThemeChange_BroadcastsToOpenCodeOnly`, `TestNotifyThemeChange_NoOpenCodeSessions`, `TestNotifyThemeChange_EmptyEngine`, `TestNotifyThemeChange_RealProcess_Integration` -- all PASS. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `frontend/src/App.tsx handleThemeChange` | `app.go App.NotifyThemeChange` | Wails binding `Call('main.App.NotifyThemeChange', [])` | WIRED | `App.tsx:24` imports; `App.tsx:98` calls fire-and-forget; `App.js:59` stub routes to Wails runtime |
| `app.go App.NotifyThemeChange` | `internal/daemon/client.go DaemonClient.NotifyThemeChange` | `a.client.NotifyThemeChange()` method call | WIRED | `app.go:393` calls; nil-guard at line 390 |
| `internal/daemon/client.go` | `POST /theme/notify` daemon HTTP endpoint | Unix socket HTTP client | WIRED | `client.go:169-179` posts; confirmed by `TestClientNotifyThemeChange` PASS |
| `internal/daemon/api.go handleNotifyThemeChange` | `internal/daemon/engine.go SessionEngine.NotifyThemeChange` | `a.engine.NotifyThemeChange(r.Context())` | WIRED | `api.go:227` |
| `internal/daemon/engine.go NotifyThemeChange` | `internal/pty/session.go Session.Signal` | `signalThemeChange(sess)` -> `sess.Signal(syscall.SIGUSR2)` | WIRED | `engine.go:266` calls `signalThemeChange`; `notify_theme_unix.go:14` calls `sess.Signal(syscall.SIGUSR2)` |
| `engine.go` CreateSession | `native.go` mergeEnv | `CreateRequest.Env` field | WIRED | Unchanged from Plan 02; `engine.go:96` sets `Env: env`; `native.go:46` consumes |
| `engine.go` NewSessionEngine | `~/.config/agenthub/opencode-tui.json` | `ensureOpenCodeTUIConfig` writes managed config | WIRED | Unchanged from Plan 02 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `engine.go NotifyThemeChange` | `e.registry.List()` sessions | Real session registry populated by `CreateSession` | Yes -- real PTY sessions | FLOWING |
| `engine.go NotifyThemeChange` | `e.sessionCLIs[sess.ID]` | Populated in `CreateSession` when real sessions are created | Yes -- raw CLI name strings | FLOWING |
| `engine.go CreateSession` | `env []string` | `e.opencodeTUIConfig` (set at init from `ensureOpenCodeTUIConfig`) | Yes -- path to real file | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All Go packages pass (short mode) | `go test ./... -count=1 -short` | 8 packages PASS, exit 0 | PASS |
| NotifyThemeChange unit tests | `go test ./internal/daemon/ -run TestNotifyThemeChange -v` | 4 tests PASS (0.11s, incl. real-process integration) | PASS |
| HTTP route + client tests | `go test ./internal/daemon/ -run "TestHandleNotifyThemeChange\|TestClientNotifyThemeChange" -v` | 3 tests PASS | PASS |
| Wails binding test | `go test github.com/scottkw/agenthub -run TestNotifyThemeChange -v` | PASS | PASS |
| Session.Signal nil-safety tests | `go test ./internal/pty/ -run TestSession_Signal -v` | 2 tests PASS | PASS |
| Frontend vitest suite | `cd frontend && npx vitest run` | 353 tests PASS (18 test files) | PASS |
| THM-05 frontend describe block | `grep "THM-05" frontend/src/components/__tests__/App.test.tsx` | 3 assertions for NotifyThemeChange import, call, and .catch wiring | PASS |
| Plan 05 commits in history | `git log --oneline 0e1a50d 6ae4aa0 a7128c1` | All 3 commits present | PASS |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| THM-05 | 71-01, 71-02, 71-03, 71-04, 71-05 | The theme selected in Settings > Appearance is applied to OpenCode terminal sessions, matching the behavior for Claude Code, Codex, and Gemini CLI sessions | SATISFIED (pending visual confirmation) | New sessions: OPENCODE_TUI_CONFIG env injection + managed tui.json (Plan 02). Active sessions: SIGUSR2 broadcast pipeline (Plan 05) delivers signal verified by real-process integration test. Cross-agent consistency: verified by UAT (Plan 04) and same xterm.js palette mechanism. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `engine.go` | 54 | `_ = os.WriteFile(path, content, 0644)` -- silent error discard | Warning | If write fails, env var points to missing file; OpenCode falls back to default theme silently. Not a blocker -- failure mode is graceful degradation (no crash). Carried forward from Plan 02 code review. |
| `opencode_ansi_test.go` | 125-127 | Diagnostic-only with `t.Logf` instead of `t.Errorf` | Info | By design -- go-pty lacks OSC responder, making hard assertions inappropriate. Plan 04 UAT is authoritative. |

### Human Verification Required

### 1. Live Theme Switch on Active OpenCode Session (SC-1 Visual Confirmation)

**Test:** Launch AgentHub. Create an OpenCode session and wait for the colored TUI to appear. Navigate to Settings > Appearance and switch to a visually distinct theme (e.g., from the default to Solarized Light or Dracula). Return to the OpenCode session tab.
**Expected:** The OpenCode session repaints with the newly selected theme colors without requiring a session restart.
**Why human:** The SIGUSR2 signal delivery is proven by `TestNotifyThemeChange_RealProcess_Integration` (real `/bin/sh` process receives USR2 and prints marker). However, whether OpenCode's `refresh4() -> clearPaletteCache() -> resolveSystemTheme()` cycle correctly re-queries xterm.js's current palette and produces the visually correct output requires observing the actual terminal render in the live application with xterm.js responding to OSC queries. This cannot be verified programmatically without a full xterm.js headless environment.

---

_Verified: 2026-04-13T22:30:00Z_
_Verifier: Claude (gsd-verifier)_
_Re-verification: Yes (Plan 05 gap closure)_
