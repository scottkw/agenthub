---
phase: 86-tui-visual-polish
plan: "01"
subsystem: tui
tags: [tui, palette, model, keybindings, tokyonight]
dependency_graph:
  requires: []
  provides: [TokyoNight-palette, model-tab-state, model-panes-focus, key-bindings-tab-nav, version-plumbing]
  affects: [internal/tui/styles.go, internal/tui/model.go, internal/tui/keys.go, internal/tui/tui.go, cmd_tui.go]
tech_stack:
  added: []
  patterns: [lipgloss LightDark adaptive palette, Bubble Tea model extension, tabID type enum]
key_files:
  created:
    - internal/tui/styles_test.go
  modified:
    - internal/tui/styles.go
    - internal/tui/model.go
    - internal/tui/keys.go
    - internal/tui/tui.go
    - cmd_tui.go
    - internal/tui/update_test.go
decisions:
  - "tabID uses iota with explicit type on each const to avoid ambiguity between const group entries"
  - "agentBadgeColor placed in styles.go (not view.go) because it reads Styles tokens directly"
  - "tabSessions set as default open tab in newModel to preserve existing UX"
metrics:
  duration_minutes: 20
  completed_date: "2026-04-19"
  tasks_completed: 2
  files_changed: 6
---

# Phase 86 Plan 01: Foundation Types, Palette, and Key Bindings Summary

**One-liner:** TokyoNight palette with 22+ adaptive tokens, tab/pane model state, and Tab/[/] key bindings wired to version-parameterized Run().

## What Was Built

### Task 1: TokyoNight Palette and Model Foundation Types

**styles.go** — Replaced all 256-color approximations with TokyoNight hex values using the `lipgloss.LightDark(hasDark)` adaptive pattern. Added 8 new tokens: `BgSurface`, `BgSidebar`, and 6 per-agent badge colors (`BadgeClaude`, `BadgeOpencode`, `BadgeCodex`, `BadgeGemini`, `BadgeCursor`, `BadgeAider`). Added `agentBadgeColor()` function mapping 6 CLI names (case-insensitive) to badge colors with `FgMuted` fallback.

**model.go** — Added `panesFocusState` bool type (`focusContent`/`focusSidebar`), `tabID` int enum (`tabHome`, `tabSessions`, `tabRemote`, `tabSettings`). Extended `Model` struct with `panesFocus`, `openTabs`, `activeTab`, `sidebarFocus`, `version` fields. Added helper methods: `activeTabID()`, `tabName()`, `openTab()`, `cycleTab()`.

### Task 2: Key Bindings, Run() Signature, and Tests

**keys.go** — Added `TabFocus` (Tab), `PrevTab` ([), `NextTab` (]) bindings to `KeyMap` struct and `defaultKeyMap()`.

**tui.go** — Updated `Run()` and `newModel()` signatures to accept `version string`. `newModel()` now initializes with `openTabs: []tabID{tabSessions}`, `activeTab: 0`, `panesFocus: focusContent`, `sidebarFocus: 1`.

**cmd_tui.go** — Updated call site to pass `daemon.BuildVersion` as third argument to `tui.Run()`.

**update_test.go** — Fixed `testModel()` to use 3-argument `newModel(nil, nil, "")`.

**styles_test.go** (new) — Three tests: `TestStyles_TokyoNight` (validates 8 dark-mode tokens + 2 light-mode tokens), `TestAgentBadgeColor` (9 cases including case-insensitive and fallback), `TestStyles_AllBadgeColorsDistinct` (all 6 badge colors unique).

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 767a5f9 | feat(86-01): TokyoNight palette and model foundation types |
| 2 | f877a6f | feat(86-01): key bindings, Run() version param, and palette tests |

## Verification

```
go build ./...          -> exit 0
go test ./internal/tui/... -count=1 -> PASS (all existing + 3 new tests)
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — no stub patterns detected in modified files.

## Threat Flags

None — pure local TUI rendering changes with no network surface.

## Self-Check: PASSED

- internal/tui/styles.go: FOUND
- internal/tui/model.go: FOUND
- internal/tui/keys.go: FOUND
- internal/tui/tui.go: FOUND
- cmd_tui.go: FOUND
- internal/tui/styles_test.go: FOUND
- Commit 767a5f9: FOUND
- Commit f877a6f: FOUND
