# Phase 82: Minimize to Tray - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-16
**Phase:** 82-minimize-to-tray
**Areas discussed:** Toggle placement, Startup behavior, Platform scope

---

## Toggle Placement

| Option | Description | Selected |
|--------|-------------|----------|
| Appearance tab | Add to existing Appearance sub-section alongside theme selection | |
| New General tab | Create a new "General" sub-section at top of Settings | |
| Top of Settings | Place outside any sub-section as standalone toggle | |

**User's choice:** New "Behavior" section in the Settings tab (free-text response)
**Notes:** User specified a "Behavior" section rather than any of the presented options. Preferred a dedicated section for app-level behavior toggles.

### Follow-up: Section Ordering

| Option | Description | Selected |
|--------|-------------|----------|
| Before Paths | Behavior at top, then Paths, then Appearance | |
| Between Paths and Appearance | Paths first, then Behavior, then Appearance | |
| After Appearance | Paths, Appearance, then Behavior at bottom | |

**User's choice:** Claude's discretion
**Notes:** User was indifferent — deferred to Claude for best design judgment.

---

## Startup Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Skip window entirely | Window never shows — domReady skips WindowShow, only tray icon appears | ✓ |
| Flash splash then hide | Show splash briefly (~1s) then auto-hide for visual confirmation | |
| You decide | Claude picks approach that fits Wails lifecycle | |

**User's choice:** Skip window entirely
**Notes:** Cleanest minimized experience — no flash, no splash, just tray icon.

---

## Platform Scope

| Option | Description | Selected |
|--------|-------------|----------|
| All platforms | macOS, Linux, Windows all get the toggle from the start | ✓ |
| macOS only first | Ship macOS first, add Linux/Windows in follow-up | |

**User's choice:** All platforms
**Notes:** Each platform already has tray support — change is in domReady and settings, not tray code.

---

## Claude's Discretion

- Behavior section ordering within Settings tab
- Toggle component style (checkbox, switch, etc.)
- How domReady reads the start-minimized setting
- Edge case handling when daemon is unreachable at startup

## Deferred Ideas

None — discussion stayed within phase scope
