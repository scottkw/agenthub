---
gsd_state_version: 1.0
milestone: v1.7
milestone_name: Daemon UX & Branding
status: verifying
stopped_at: "Completed 36-01-PLAN.md — awaiting Task 3 human visual verification (checkpoint:human-verify)"
last_updated: "2026-04-01T03:47:41.763Z"
last_activity: 2026-04-01
progress:
  total_phases: 6
  completed_phases: 1
  total_plans: 1
  completed_plans: 1
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-31)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 36 — app-icons-branding-assets

## Current Position

Phase: 37
Plan: Not started
Status: Phase complete — ready for verification
Last activity: 2026-04-01

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 0
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

*Updated after each plan completion*
| Phase 36-app-icons-branding-assets P01 | 37 | 2 tasks | 22 files |

## Accumulated Context

### Decisions

Key decisions from research (inform upcoming phases):

- Phase 36: Icon sizes — ICNS needs all 10 named files (5 sizes × 2 densities); square logomark required (wordmark at 805x208 is too wide for small sizes)
- Phase 37: Splash anti-flash — use `StartHidden: true` + static HTML splash in index.html + `OnDomReady` → `runtime.WindowShow()`; `defer setInitComplete(true)` on all code paths
- Phase 39: Web status bar — use REST polling (3s interval) not a new relay frame type; status bar must be flex sibling with fixed height to avoid FitAddon regression
- Phase 41: Tray — never add fyne.io/systray or any mainstream systray library (duplicate AppDelegate symbol confirmed failure); keep custom cgo NSStatusItem for macOS; LSUIElement in Info.plist only (not Info.dev.plist)
- [Phase 36-app-icons-branding-assets]: Phase 36: ICNS injection pattern — wails build produces 3-size ICNS (361KB); post-build cp of pre-built 10-entry iconfile.icns (590KB) into bundle Resources is required
- [Phase 36-app-icons-branding-assets]: Phase 36: Transparent background for appicon.png — macOS applies standardized rounded corners + drop shadow to transparent icons; no custom background color needed

### Pending Todos

None.

### Blockers/Concerns

- [Phase 36]: Square logomark asset needed — existing `docs/agenthub-title-logo.png` is a horizontal wordmark (805x208); icon sizes below 64px need a standalone square mark. Resolve before Phase 36 execution: extract mark, redraw, or letterbox with padding.
- [Phase 41]: LSUIElement removes app from Cmd+Tab entirely (not just Dock) — confirm this product behavior is acceptable before Phase 41 implementation.
- [Phase 41]: Tray and ICNS changes require `wails build` production test cycle (not `wails dev`) — plan for slower iteration in Phases 36 and 41.

## Session Continuity

Last session: 2026-03-31T20:49:22.746Z
Stopped at: Completed 36-01-PLAN.md — awaiting Task 3 human visual verification (checkpoint:human-verify)
Resume file: None
Next action: `/gsd:plan-phase 36`
