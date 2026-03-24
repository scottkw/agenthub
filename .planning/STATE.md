---
gsd_state_version: 1.0
milestone: v1.3
milestone_name: CLI + Daemon
status: Ready to execute
stopped_at: Completed 26-graceful-gui-startup-failure/26-02-PLAN.md
last_updated: "2026-03-24T21:56:36.811Z"
progress:
  total_phases: 8
  completed_phases: 7
  total_plans: 15
  completed_plans: 14
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-23)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 26 — graceful-gui-startup-failure

## Current Position

Phase: 26 (graceful-gui-startup-failure) — EXECUTING
Plan: 2 of 2

## Performance Metrics

**Velocity:**

- Total plans completed: 0 (v1.3)
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| — | — | — | — |

*Updated after each plan completion*
| Phase 19-daemon-core-engine-ipc P01 | 25min | 2 tasks | 9 files |
| Phase 19-daemon-core-engine-ipc P02 | 6min | 1 tasks | 3 files |
| Phase 19-daemon-core-engine-ipc P02 | 11min | 2 tasks | 3 files |
| Phase 20-process-separation P01 | 4min | 2 tasks | 9 files |
| Phase 21-cli-session-web-commands P01 | 15min | 2 tasks | 2 files |
| Phase 21-cli-session-web-commands P02 | 4min | 2 tasks | 2 files |
| Phase 22-cli-attach P01 | 2 | 2 tasks | 5 files |
| Phase 22-cli-attach P02 | 130 | 1 tasks | 1 files |
| Phase 23-service-manager-integration P01 | 104 | 2 tasks | 5 files |
| Phase 23-service-manager-integration P02 | 8 | 1 tasks | 3 files |
| Phase 24-cli-polish P01 | 3 | 2 tasks | 4 files |
| Phase 24-cli-polish P02 | 2min | 1 tasks | 2 files |
| Phase 26-graceful-gui-startup-failure P02 | 2 | 2 tasks | 4 files |

## Accumulated Context

### Decisions

Recent decisions affecting current work (full log in PROJECT.md):

- [v1.2]: Tailscale health gates web server startup — safe to build daemon on top of this
- [v1.3 roadmap]: Phase 19 combines SessionEngine extraction + IPC layer (in-process) to validate protocol before process separation in Phase 20
- [v1.3 roadmap]: Phase 22 (CLI Attach) is its own phase — 7 distinct correctness requirements need explicit testing gates
- [Phase 19-01]: onStatus callback injected at CreateSession call site so engine has zero Wails imports; App in Plan 02 supplies the EventsEmit wrapper
- [Phase 19-01]: CleanupStaleSocket probes with net.DialTimeout(500ms): refused/timeout=stale (remove), success=already running (error)
- [Phase 19-01]: Short socket paths in tests use /tmp/dtest{n}_{name} — macOS t.TempDir() paths exceed 103-char sun_path limit
- [Phase 19-02]: CreateSession calls engine directly (not client) — onStatus callback cannot be serialized over HTTP; this is the intentional exception to the delegation pattern
- [Phase 19-02]: testApp() uses /tmp/aht{pid}_{seq}.sock paths to stay under macOS 103-char sun_path limit (t.TempDir() produces paths > 103 chars)
- [Phase 19-02]: CreateSession calls engine directly (not client) — onStatus callback cannot be serialized over HTTP; this is the intentional exception to the delegation pattern
- [Phase 19-02]: testApp() uses /tmp/aht{pid}_{seq}.sock paths to stay under macOS 103-char sun_path limit (t.TempDir() produces paths > 103 chars)
- [Phase 20-process-separation]: Relay TCP server lives inside API struct (relayLn field), started by RunDaemon before api.Start() — daemon owns the relay port lifecycle
- [Phase 20-process-separation]: EnsureDaemon takes socketPath as argument — allows tests to inject short socket paths and avoids macOS 103-char limit
- [Phase 20-process-separation]: pollSessionStatus goroutine replaces onStatus callback — callbacks cannot be serialized over HTTP; polling is the correct pattern for out-of-process daemon
- [Phase 20-process-separation]: SetWebServerForTest added to daemon.API — enables test injection of TLS webserver without Tailscale in test environment
- [Phase 20-process-separation]: shutdown() has no daemon teardown — daemon is an independent process; GUI closing does not affect session state (DAEMON-03)
- [Phase 20-02]: PTY sessions use background context — HTTP request context cancellation kills goroutines when handler returns; PTY goroutines must outlive the request
- [Phase 20-02]: GUI regression verified end-to-end: sessions survive close/reopen, daemon auto-restarts after kill, relay port handoff confirmed working
- [Phase 21-01]: cmd functions return error instead of os.Exit — main() handles exits; makes unit testing trivial
- [Phase 21-01]: io.Writer injection on cmdNew/cmdList for testable stdout; main() passes os.Stdout, tests pass bytes.Buffer
- [Phase 21-cli-session-web-commands]: cmdWebStart gates on all 3 Tailscale checks (Connected, IP, HasCerts) before calling daemon
- [Phase 21-cli-session-web-commands]: testSetupWithWebServer injects real WebServer via SetWebServerForTest for serve/unserve tests — avoids Tailscale in CI
- [Phase 22-cli-attach]: Use MsgResize2 (0x11) not MsgResize (0x02) for client-to-server resize — server read pump only handles MsgResize2 for incoming resize frames
- [Phase 22-cli-attach]: Do NOT catch SIGINT in attach — in raw mode Ctrl-C is byte 0x03 forwarded to remote PTY, not a local signal
- [Phase 22-cli-attach]: Used polling loop with 10ms intervals for live output test over channel-based sync — simpler and tolerates scheduler jitter
- [Phase 22-cli-attach]: 30ms sleep before dial in scrollback test mirrors relay/server_test.go pattern to ensure hub processes write before client connects
- [Phase 23-service-manager-integration]: Use KeepAlive=false to allow manual daemon stop without automatic restart
- [Phase 23-service-manager-integration]: Use UserService=true for user-scope service registration (launchd/systemd/SCM)
- [Phase 23-service-manager-integration]: runDaemonCore uses fmt.Fprintf+return instead of os.Exit for clean service manager lifecycle
- [Phase Phase 23-02]: Use package-level var serviceControlFunc = daemon.ServiceControl for test injection — avoids interface overhead for single-function mocking
- [Phase Phase 23-02]: No-args and 'run' subcommand both call daemon.RunDaemon() for backward compat with EnsureDaemon spawn pattern
- [Phase 24-cli-polish]: Used flag.NewFlagSet per command (not flag package globals) to avoid state pollution between test runs
- [Phase 24-cli-polish]: daemon status intercepted in main() before early-exit block; other daemon subcommands bypass EnsureDaemon
- [Phase 24-cli-polish]: cmdSettings uses daemon.DefaultSocketPath() directly (socket path is local config, not daemon state); tests use /bin/sh for CLI path (daemon validates existence)
- [Phase 26-graceful-gui-startup-failure]: Early-return in retryInit: call RetryDaemon() first; if daemon restart fails, skip Promise.all to avoid cascading nil-client errors
- [Phase 26-graceful-gui-startup-failure]: Banner shows {daemonError} directly — Go error strings are more actionable than hardcoded generic messages

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 22 risk]: Terminal left in raw mode on crash is most visible failure mode — signal handlers for SIGTERM/SIGINT/SIGHUP must restore terminal before exit.
- [Phase 23 research flag]: Windows SCM behavior with kardianos/service is MEDIUM confidence — establish Windows CI during Phase 19 before Phase 23 makes it critical.

## Session Continuity

Last session: 2026-03-24T21:56:36.809Z
Stopped at: Completed 26-graceful-gui-startup-failure/26-02-PLAN.md
Resume file: None
