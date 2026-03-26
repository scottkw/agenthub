---
gsd_state_version: 1.0
milestone: v1.5
milestone_name: Bug Fixes & CLI Args
status: Ready to plan
stopped_at: Completed 32-01-PLAN.md
last_updated: "2026-03-26T05:37:53.416Z"
progress:
  total_phases: 5
  completed_phases: 3
  total_plans: 4
  completed_plans: 4
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-25)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 32 — daemon-startup-performance

## Current Position

Phase: 33
Plan: Not started

## Accumulated Context

### Decisions

- v1.4 unified binary: single `agenthub` binary dispatches GUI/CLI/daemon
- Daemon architecture: SessionEngine in `internal/daemon`, HTTP/JSON over Unix socket
- Terminal fill bug is agent-specific (Claude/Gemini), works after resize — root cause is FitAddon called before CSS layout commits
- Slow startup root cause: `pollSessionStatus` in `app.go` sleeps 2s before first poll (not EnsureDaemon)
- Args feature: `pty.CreateRequest.Args` already exists and is forwarded at PTY layer; gap is in the 5 layers above it
- [Phase 30-backend-args-wiring]: args threaded between workDir and onStatus params; json:args,omitempty ensures backward-compatible wire format; all callers pass nil
- [Phase 31-cli-arg-passthrough]: splitDashDash returns nil (not empty slice) when no -- present, matching Go idiom
- [Phase 32-daemon-startup-performance]: Runtime PATH augmentation at daemon startup: prepend nvm/volta/Homebrew dirs to process PATH before exec.LookPath, so service-mode agents resolve correctly
- [Phase 32-daemon-startup-performance]: pollSessionStatus poll-first: GetSessionStatus fires immediately, then sleep 500ms between polls (was sleep 2s before first call)

### Pending Todos

None.

### Blockers/Concerns

- PERF-03 (service-mode PATH): Gemini CLI has known 8-60s MCP initialization regression (CLI-side, not daemon-side). Profile with time.Now() deltas to distinguish.
- Phase 34: double-rAF vs. single-rAF for initial fit — test both in production binary (wails build), behavior differs from wails dev.
- Phase 33: confirm whether `wails dev` auto-regenerates TypeScript bindings on Go method signature change or requires explicit `wails generate`.

## Session Continuity

Last session: 2026-03-26T05:34:55.358Z
Stopped at: Completed 32-01-PLAN.md
Resume file: None
Next action: `/gsd:plan-phase 30`
