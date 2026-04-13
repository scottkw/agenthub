---
phase: 71-opencode-theming-fix
plan: 04
subsystem: uat
tags: [uat, opencode, theming, human-verified]

# Dependency graph
requires:
  - phase: 71-02
    provides: "OPENCODE_TUI_CONFIG env injection + managed tui.json"
  - phase: 71-03
    provides: "Empirical verification of color output behavior"
provides:
  - "Human-verified status of Phase 71 success criteria"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "Live theme switch (SC-1) requires follow-up work — not solvable by env injection alone"
  - "OpenCode only re-queries terminal palette on SIGUSR2 or dark/light mode change; AgentHub does not yet signal either"

patterns-established: []

requirements-completed: []

# Metrics
duration: 5min
completed: 2026-04-13
---

# Phase 71 Plan 04: UAT Verification Summary

**Human-verified status of three Phase 71 success criteria via live app testing.**

## Results

| SC | Description | Result | Notes |
|----|-------------|--------|-------|
| SC-1 | Theme switch repaints active OpenCode session | ❌ FAIL | Existing sessions retain prior theme after switch |
| SC-2 | New OpenCode session starts with current theme | ✅ PASS | Newly spawned sessions inherit active theme |
| SC-3 | Cross-agent visual consistency | ✅ PASS (implicit) | Same mechanism as SC-2; confirmed working for new sessions |

## Interpretation

Plan 02's env injection delivers **session-start theme inheritance** (SC-2, SC-3) but does **not** deliver **live theme switching on active sessions** (SC-1).

**Why SC-2/SC-3 work:** On PTY spawn, OpenCode reads `OPENCODE_TUI_CONFIG` → loads `{"theme":"system"}` → queries xterm.js via OSC 10/11/4 escape sequences → xterm.js responds with the currently active theme's palette → `generateSystem()` builds a theme object from those hex values → OpenCode renders using that theme.

**Why SC-1 fails:** OpenCode only re-queries the terminal palette under two conditions (from binary analysis of `opencode 1.4.0`):
1. `process.on("SIGUSR2", refresh4)` — where `refresh4` calls `renderer.clearPaletteCache()` then `init4()` which re-runs `resolveSystemTheme`
2. `CliRenderEvents.THEME_MODE` event — fires when OpenCode's own dark/light mode toggles

AgentHub currently does neither when the user changes theme in `Settings > Appearance`. OpenCode sessions therefore keep rendering with the palette that was active when they were spawned.

The generated theme is encoded as 24-bit RGB escape sequences (`\033[38;2;R;G;Bm`), not ANSI palette indices, so xterm.js cannot retroactively remap colors at the terminal level either — OpenCode itself must re-emit with new values.

## Gap

**SC-1 gap → closure path:** Implement SIGUSR2 broadcasting. When the user changes theme in AgentHub, the daemon should signal all active OpenCode sessions so they re-query xterm.js and re-render.

Sketch:
- Add `SessionEngine.NotifyThemeChange()` method that walks sessions where `cli == "opencode"` and sends `syscall.SIGUSR2` to the process (via `pty.SessionBackend` extension or direct PID lookup).
- Wire frontend theme-change handler through Wails bindings to call this method.
- Verification: live SC-1 test — switch theme, observe OpenCode repaints without restart.

This work should live in a follow-up phase (`71.1` or similar) rather than extending Plan 04, since it requires new backend/frontend integration and its own test coverage.

## What Shipped (Phase 71 as executed)

- ✅ Managed `~/.config/agenthub/opencode-tui.json` written on daemon start with `{"theme":"system"}`
- ✅ `OPENCODE_TUI_CONFIG` env var injected into OpenCode PTY sessions
- ✅ New OpenCode sessions inherit active AgentHub theme
- ✅ Cross-agent consistency at session start time
- ❌ Live theme switching on active OpenCode sessions (deferred to gap-closure phase)

## Self-Check: PASSED (partial — SC-1 gap documented)

- UAT completed by human tester
- Predictions from binary analysis matched observed behavior exactly
- Gap closure path identified with concrete mechanism (SIGUSR2)

---
*Phase: 71-opencode-theming-fix*
*Completed: 2026-04-13*
