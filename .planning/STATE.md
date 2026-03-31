---
gsd_state_version: 1.0
milestone: v1.7
milestone_name: Daemon UX & Branding
status: active
stopped_at: Roadmap created — ready for Phase 36 planning
last_updated: "2026-03-31T19:30:00.000Z"
last_activity: 2026-03-31 -- Roadmap created for v1.7 (6 phases, 15 requirements)
progress:
  total_phases: 6
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-31)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 36 — App Icons & Branding Assets

## Current Position

Phase: 36 of 41 (App Icons & Branding Assets)
Plan: Not started
Status: Ready to plan
Last activity: 2026-03-31 — Roadmap created (Phases 36-41, 15 requirements mapped)

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

## Accumulated Context

### Decisions

Key decisions from research (inform upcoming phases):

- Phase 36: Icon sizes — ICNS needs all 10 named files (5 sizes × 2 densities); square logomark required (wordmark at 805x208 is too wide for small sizes)
- Phase 37: Splash anti-flash — use `StartHidden: true` + static HTML splash in index.html + `OnDomReady` → `runtime.WindowShow()`; `defer setInitComplete(true)` on all code paths
- Phase 39: Web status bar — use REST polling (3s interval) not a new relay frame type; status bar must be flex sibling with fixed height to avoid FitAddon regression
- Phase 41: Tray — never add fyne.io/systray or any mainstream systray library (duplicate AppDelegate symbol confirmed failure); keep custom cgo NSStatusItem for macOS; LSUIElement in Info.plist only (not Info.dev.plist)

### Pending Todos

None.

### Blockers/Concerns

- [Phase 36]: Square logomark asset needed — existing `docs/agenthub-title-logo.png` is a horizontal wordmark (805x208); icon sizes below 64px need a standalone square mark. Resolve before Phase 36 execution: extract mark, redraw, or letterbox with padding.
- [Phase 41]: LSUIElement removes app from Cmd+Tab entirely (not just Dock) — confirm this product behavior is acceptable before Phase 41 implementation.
- [Phase 41]: Tray and ICNS changes require `wails build` production test cycle (not `wails dev`) — plan for slower iteration in Phases 36 and 41.

## Session Continuity

Last session: 2026-03-31
Stopped at: Roadmap created — 6 phases defined, all 15 v1.7 requirements mapped
Resume file: None
Next action: `/gsd:plan-phase 36`
