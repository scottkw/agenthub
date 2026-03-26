---
gsd_state_version: 1.0
milestone: v1.5
milestone_name: Bug Fixes & CLI Args
status: Ready to plan
stopped_at: null
last_updated: "2026-03-25"
progress:
  total_phases: 5
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-25)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 30 — Backend Args Wiring

## Current Position

Phase: 30 of 34 (Backend Args Wiring)
Plan: 0 of ? in current phase
Status: Ready to plan
Last activity: 2026-03-25 — v1.5 roadmap created; phases 30-34 defined

Progress: [░░░░░░░░░░] 0% (v1.5)

## Accumulated Context

### Decisions

- v1.4 unified binary: single `agenthub` binary dispatches GUI/CLI/daemon
- Daemon architecture: SessionEngine in `internal/daemon`, HTTP/JSON over Unix socket
- Terminal fill bug is agent-specific (Claude/Gemini), works after resize — root cause is FitAddon called before CSS layout commits
- Slow startup root cause: `pollSessionStatus` in `app.go` sleeps 2s before first poll (not EnsureDaemon)
- Args feature: `pty.CreateRequest.Args` already exists and is forwarded at PTY layer; gap is in the 5 layers above it

### Pending Todos

None.

### Blockers/Concerns

- PERF-03 (service-mode PATH): Gemini CLI has known 8-60s MCP initialization regression (CLI-side, not daemon-side). Profile with time.Now() deltas to distinguish.
- Phase 34: double-rAF vs. single-rAF for initial fit — test both in production binary (wails build), behavior differs from wails dev.
- Phase 33: confirm whether `wails dev` auto-regenerates TypeScript bindings on Go method signature change or requires explicit `wails generate`.

## Session Continuity

Last session: 2026-03-25
Stopped at: Roadmap created — Phase 30 ready to plan
Resume file: None
Next action: `/gsd:plan-phase 30`
