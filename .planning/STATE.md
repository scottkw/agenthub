---
gsd_state_version: 1.0
milestone: v1.4
milestone_name: Unified Binary
status: Ready to plan
stopped_at: Completed 28-01-PLAN.md
last_updated: "2026-03-25T19:00:14.798Z"
progress:
  total_phases: 3
  completed_phases: 2
  total_plans: 2
  completed_plans: 2
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-25)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 28 — cli-package-removal

## Current Position

Phase: 29
Plan: Not started

## Performance Metrics

| Metric | v1.3 | v1.4 Target |
|--------|------|-------------|
| Phases | 8 | 3 |
| Requirements | ~20 | 12 |
| Binary targets | 2 (agenthub + agenthub-cli) | 1 (agenthub) |
| Phase 27-unified-entrypoint P01 | 18 | 2 tasks | 11 files |
| Phase 28 P01 | 2min | 2 tasks | 9 files |

## Accumulated Context

### Decisions

- v1.4 is merge-only: no new CLI commands, no new features
- Phase 28 (cleanup) must follow Phase 27 (dispatch works) to avoid breaking the build
- Phase 29 (build/CI) must follow Phase 28 so CI validates the final state, not intermediate
- Unified dispatch strategy: `len(os.Args) == 1 || os.Args[1] starts with "-"` → GUI; otherwise → CLI
- [Phase 27-unified-entrypoint]: Dispatch rule: no args or flag (not --help/-h) routes to GUI; --help/-h routes to usage(); any word command routes to runCLI()
- [Phase 27-unified-entrypoint]: daemon status fall-through: most daemon subcommands go directly to cmdDaemon; only status needs EnsureDaemon first
- [Phase 27-unified-entrypoint]: cmd/agenthub-cli/ left untouched — deletion is Phase 28
- [Phase 28]: Deleted cmd/agenthub-cli/ entirely — all CLI logic now lives in root package (main.go) per Phase 27 unification

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-03-25T18:55:11.035Z
Stopped at: Completed 28-01-PLAN.md
Resume file: None
Next action: `/gsd:plan-phase 27`
