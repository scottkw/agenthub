---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: executing
stopped_at: Completed 02-session-registry-websocket-relay-02-PLAN.md
last_updated: "2026-03-18T13:45:55.165Z"
last_activity: 2026-03-18 — Plan 01-01 complete (Go module, PTY interfaces, CLI detection, session registry)
progress:
  total_phases: 6
  completed_phases: 2
  total_plans: 4
  completed_plans: 4
  percent: 50
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-17)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 1 — PTY Foundation

## Current Position

Phase: 1 of 6 (PTY Foundation)
Plan: 1 of 2 in current phase
Status: In progress
Last activity: 2026-03-18 — Plan 01-01 complete (Go module, PTY interfaces, CLI detection, session registry)

Progress: [█████░░░░░] 50%

## Performance Metrics

**Velocity:**
- Total plans completed: 1
- Average duration: 2 min
- Total execution time: 2 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-pty-foundation | 1 of 2 | 2 min | 2 min |

*Updated after each plan completion*
| Phase 01-pty-foundation P02 | 10min | 3 tasks | 14 files |
| Phase 02-session-registry-websocket-relay P01 | 3min | 2 tasks | 6 files |
| Phase 02-session-registry-websocket-relay P02 | 3min | 2 tasks | 5 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Use go-pty (aymanbagabas) not creack/pty — Windows ConPTY support required from day one
- [Roadmap]: win32-input-mode state-machine parser needed in Phase 1 — cannot be retrofitted
- [Roadmap]: CA cert pattern (not bare self-signed leaf) required for WSS — browsers silently reject untrusted wss://
- [Roadmap]: PLAT-01/02/03 assigned to Phase 6 but cross-platform validation is incremental each phase
- [Phase 01-pty-foundation]: SessionBackend is an interface so Plan 02 provides the platform implementation without touching Plan 01 types
- [Phase 01-pty-foundation]: DetectCLIs returns make([]DetectedCLI, 0) not nil — callers can range safely without nil check
- [Phase 01-pty-foundation]: Registry owns session lifetime: context cancellation does NOT remove sessions from registry
- [Phase 01-pty-foundation]: Do not combine Setpgid:true with go-pty — Setsid already creates new session (PGID==PID); combining causes EPERM on macOS
- [Phase 01-pty-foundation]: Close PTY master before cmd.Wait in killSession — prevents indefinite block when PTY slave is still referenced after child exits
- [Phase 01-pty-foundation]: win32input_parse.go has no build tag — stateless chunk parser compiles everywhere so unit tests run on all platforms
- [Phase 01-pty-foundation]: session.job stored as any — avoids Windows build tags in session.go; type assertion done in cleanup_windows.go
- [Phase 02-session-registry-websocket-relay]: Hub stores scrollback as framed bytes (MakeOutputFrame wrapped) so WebSocket clients receive identical bytes from live stream and replay without re-framing
- [Phase 02-session-registry-websocket-relay]: Scrollback.Append uses in-place copy-left on overflow (no extra allocation) to reduce GC pressure under high-throughput PTY output
- [Phase 02-session-registry-websocket-relay]: Hub.Shutdown uses sync.Once — allows Run to call it on return and external callers to call it safely without panic
- [Phase 02-session-registry-websocket-relay]: HubManager.Create is idempotent — returns existing hub if session already exists, preventing double-Run goroutines
- [Phase 02-session-registry-websocket-relay]: websocket.Accept uses InsecureSkipVerify:true — origin validation deferred to Phase 4 where CORS policy will be defined with known Electron origin

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 3]: Linux WebKitGTK fragmentation — webkit2gtk-4.0 vs 4.1 (Ubuntu 22.04 vs 24.04) needs research during Phase 3 planning
- [Phase 4]: Per-OS CA cert trust installation UX underspecified — macOS Keychain, Linux NSS, Windows certutil; needs design pass before Phase 4 execution
- [Phase 5]: Per-CLI status indicator output patterns for Codex, Gemini CLI, OpenCode undocumented — empirical testing needed during Phase 5 planning

## Session Continuity

Last session: 2026-03-18T13:45:55.162Z
Stopped at: Completed 02-session-registry-websocket-relay-02-PLAN.md
Resume file: None
