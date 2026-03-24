# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- 🚧 **v1.3 CLI + Daemon** — Phases 19-24 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED 2026-03-19</summary>

- [x] Phase 1: PTY Foundation (2/2 plans) — completed 2026-03-18
- [x] Phase 2: Session Registry + WebSocket Relay (2/2 plans) — completed 2026-03-18
- [x] Phase 3: Wails Desktop UI (3/3 plans) — completed 2026-03-18
- [x] Phase 4: Web Serving + TLS + Auth (4/4 plans) — completed 2026-03-18
- [x] Phase 5: QR Codes + Status Indicators (6/6 plans) — completed 2026-03-18
- [x] Phase 6: Distribution + Cross-Platform (2/2 plans) — completed 2026-03-19

</details>

<details>
<summary>✅ v1.1 Polish & Build (Phases 7-13) — SHIPPED 2026-03-20</summary>

- [x] Phase 7: Layout Baseline (1/1 plans) — completed 2026-03-19
- [x] Phase 8: Per-Tab Status Bar (2/2 plans) — completed 2026-03-19
- [x] Phase 9: Settings Modal Overhaul (1/1 plans) — completed 2026-03-19
- [x] Phase 10: Per-Tab Font Size (1/1 plans) — completed 2026-03-19
- [x] Phase 11: New-Session Modal (3/3 plans) — completed 2026-03-19
- [x] Phase 12: Tab Rename + Web Dashboard (3/3 plans) — completed 2026-03-20
- [x] Phase 13: Build Script (2/2 plans) — completed 2026-03-20

</details>

<details>
<summary>✅ v1.2 Tailscale-Only Networking (Phases 14-18) — SHIPPED 2026-03-23</summary>

- [x] Phase 14: Tailscale Health Check Infrastructure (2/2 plans) — completed 2026-03-20
- [x] Phase 15: Tailscale TLS + Interface Binding (2/2 plans) — completed 2026-03-20
- [x] Phase 16: Auth Layer Removal (2/2 plans) — completed 2026-03-20
- [x] Phase 17: Dead Code Cleanup (2/2 plans) — completed 2026-03-20
- [x] Phase 18: Frontend Health Modal + Status UI (2/2 plans) — completed 2026-03-22

</details>

### 🚧 v1.3 CLI + Daemon (In Progress)

**Milestone Goal:** Extract session management into a persistent background daemon; GUI and CLI are both clients to the same session pool.

- [x] **Phase 19: Daemon Core (Engine + IPC)** — Extract SessionEngine from App and establish HTTP/JSON protocol over Unix socket in-process; validates the module boundary and protocol before any process separation (completed 2026-03-23)
- [x] **Phase 20: Process Separation** — Fork daemon into a standalone process; sessions survive GUI close/reopen; daemon auto-starts from CLI when not running (completed 2026-03-23)
- [x] **Phase 21: CLI Session + Web Commands** — Standalone CLI binary with new, list, kill, rename, web, health, qr, serve, unserve commands; all thin wrappers over DaemonClient (completed 2026-03-24)
- [x] **Phase 22: CLI Attach** — Full interactive PTY proxy: raw I/O, resize propagation, Ctrl-C passthrough, scrollback replay, detach prefix state machine, terminal restore on all exit paths (completed 2026-03-24)
- [x] **Phase 23: Service Manager Integration** — Register daemon with launchd (macOS), systemd (Linux), Windows SCM; agenthub daemon install/uninstall/start/stop (completed 2026-03-24)
- [x] **Phase 24: CLI Polish** — JSON output flag on all list/status commands; agenthub settings read-only inspection (completed 2026-03-24)

## Phase Details

### Phase 19: Daemon Core (Engine + IPC)
**Goal**: Session logic lives in a self-contained `internal/daemon` package with a validated HTTP/JSON protocol over Unix socket; App delegates to DaemonClient; all existing tests pass
**Depends on**: Phase 18 (v1.2 complete)
**Requirements**: DAEMON-02
**Success Criteria** (what must be TRUE):
  1. `internal/daemon/engine.go` contains `SessionEngine` with all session state extracted from `App`; `App` holds no authoritative session state
  2. `internal/daemon/api.go` exposes HTTP routes over a Unix socket covering session CRUD, web serving, health, and settings
  3. `internal/daemon/client.go` provides a typed Go client; App delegates all session operations through it
  4. Stale socket file on startup is handled automatically (auto-remove on ECONNREFUSED); socket path length assertion fires a clear error before a cryptic bind failure
  5. All existing Go tests pass with `go test -race ./...`; GUI behavior is identical to v1.2
**Plans**: 2 plans
Plans:
- [x] 19-01-PLAN.md — Create internal/daemon package (SessionEngine, HTTP API, DaemonClient, socket utilities, tests)
- [x] 19-02-PLAN.md — Migrate App to delegate through DaemonClient, verify all tests pass, GUI regression check

### Phase 20: Process Separation
**Goal**: Sessions outlive the GUI window; closing and reopening the GUI reconnects to the same running session pool managed by a separate daemon process
**Depends on**: Phase 19
**Requirements**: DAEMON-01, DAEMON-03, DAEMON-04, DAEMON-05
**Success Criteria** (what must be TRUE):
  1. Closing the GUI window does not terminate any running sessions
  2. Reopening the GUI reconnects to the daemon and shows the exact same sessions that were running before close
  3. CLI `agenthub list` output and GUI session list are identical in all scenarios (single source of truth)
  4. The daemon auto-starts when any CLI command is run and no daemon is already listening on the socket
  5. GUI App struct holds exactly one field for daemon communication: `*daemon.DaemonClient`
**Plans**: 2 plans
Plans:
- [x] 20-01-PLAN.md — Daemon infrastructure: RunDaemon entry point, EnsureDaemon auto-start, relay/webserver API routes + client methods
- [x] 20-02-PLAN.md — App migration to pure DaemonClient shell, main.go daemon dispatch, frontend error banner, GUI regression

### Phase 21: CLI Session + Web Commands
**Goal**: Users can manage sessions and all web-serving functionality from the terminal
**Depends on**: Phase 20
**Requirements**: CLI-01, CLI-02, CLI-03, CLI-04, WEB-01, WEB-02, WEB-03, WEB-04, WEB-05
**Success Criteria** (what must be TRUE):
  1. `agenthub new <agent> <path>` creates a session and prints its ID; the session appears in both CLI list and GUI
  2. `agenthub list` shows all sessions with ID, name, agent, and status
  3. `agenthub kill <id>` terminates a session; `agenthub rename <id> <name>` renames it with the new name reflected in GUI tab bar
  4. `agenthub web start`, `agenthub web stop`, and `agenthub web status` control and report the Tailscale web server state
  5. `agenthub serve <id>`, `agenthub unserve <id>`, `agenthub health`, and `agenthub qr <id>` execute without error and produce correct output matching GUI equivalents
**Plans**: 2 plans
Plans:
- [x] 21-01-PLAN.md — CLI binary with session commands (new, list, kill, rename), daemon mode, tests
- [x] 21-02-PLAN.md — Web/utility commands (web start/stop/status, serve, unserve, health, qr), tests

### Phase 22: CLI Attach
**Goal**: Users can attach to any running session with a full interactive terminal and detach cleanly without harming the session or leaving the terminal in a broken state
**Depends on**: Phase 21
**Requirements**: CLI-05, CLI-06, CLI-07, CLI-08
**Success Criteria** (what must be TRUE):
  1. `agenthub attach <id>` connects to a running session and replays recent scrollback output before showing the live prompt
  2. Keystrokes typed in the attached terminal are received by the AI CLI running in the session (full raw I/O passthrough)
  3. Resizing the terminal window while attached propagates the new dimensions to the session PTY with no visual corruption
  4. Ctrl-C passes through to the AI CLI as a PTY byte (0x03) and does not terminate the attach process
  5. The configured detach prefix key sequence detaches cleanly; the terminal is restored to its prior mode on every exit path including normal detach, SIGTERM, SIGHUP, and abnormal exit
**Plans**: 2 plans
Plans:
- [x] 22-01-PLAN.md — Core attach implementation (cmd_attach.go, platform files, main.go integration)
- [ ] 22-02-PLAN.md — Attach command tests (unit + integration tests for all attach behaviors)

### Phase 23: Service Manager Integration
**Goal**: The daemon can be registered as a platform-native service that auto-starts on login and is controllable via CLI
**Depends on**: Phase 22
**Requirements**: SVC-01, SVC-02, SVC-03
**Success Criteria** (what must be TRUE):
  1. `agenthub daemon install` registers the daemon service on macOS (launchd), Linux (systemd user unit), and Windows (SCM)
  2. After `install`, the daemon starts automatically on next login without any manual intervention; sessions created before reboot are reconnectable after
  3. `agenthub daemon uninstall` removes the service registration; daemon no longer auto-starts after the next reboot
  4. `agenthub daemon start` and `agenthub daemon stop` control a registered service that is not already running/stopped
  5. The daemon runs in foreground mode (no double-fork on any platform); the service manager owns the process lifecycle
**Plans**: 2 plans
Plans:
- [x] 23-01-PLAN.md — Add kardianos/service dependency, create service.go with daemonSvc wrapper + ServiceControl, refactor RunDaemon
- [ ] 23-02-PLAN.md — CLI daemon subcommand dispatcher (install/uninstall/start/stop), tests, macOS verification

### Phase 24: CLI Polish
**Goal**: CLI output is machine-readable and all configuration is inspectable from the terminal without opening the GUI
**Depends on**: Phase 23
**Requirements**: POLISH-01, POLISH-02
**Success Criteria** (what must be TRUE):
  1. `agenthub list --json`, `agenthub web status --json`, `agenthub health --json`, and `agenthub daemon status --json` all emit valid JSON parseable by `jq` with no interleaved plain text
  2. `agenthub settings` prints current configuration values in a human-readable format (read-only; no modifications via this command)
**Plans**: 2 plans
Plans:
- [x] 24-01-PLAN.md — Add --json flag to list, web status, health, daemon status commands with tests
- [x] 24-02-PLAN.md — Add agenthub settings command for read-only config inspection with tests

## Progress

**Execution Order:**
Phases execute in numeric order: 19 → 20 → 21 → 22 → 23 → 24

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. PTY Foundation | v1.0 | 2/2 | Complete | 2026-03-18 |
| 2. Session Registry + WebSocket Relay | v1.0 | 2/2 | Complete | 2026-03-18 |
| 3. Wails Desktop UI | v1.0 | 3/3 | Complete | 2026-03-18 |
| 4. Web Serving + TLS + Auth | v1.0 | 4/4 | Complete | 2026-03-18 |
| 5. QR Codes + Status Indicators | v1.0 | 6/6 | Complete | 2026-03-18 |
| 6. Distribution + Cross-Platform | v1.0 | 2/2 | Complete | 2026-03-19 |
| 7. Layout Baseline | v1.1 | 1/1 | Complete | 2026-03-19 |
| 8. Per-Tab Status Bar | v1.1 | 2/2 | Complete | 2026-03-19 |
| 9. Settings Modal Overhaul | v1.1 | 1/1 | Complete | 2026-03-19 |
| 10. Per-Tab Font Size | v1.1 | 1/1 | Complete | 2026-03-19 |
| 11. New-Session Modal | v1.1 | 3/3 | Complete | 2026-03-19 |
| 12. Tab Rename + Web Dashboard | v1.1 | 3/3 | Complete | 2026-03-20 |
| 13. Build Script | v1.1 | 2/2 | Complete | 2026-03-20 |
| 14. Tailscale Health Check Infrastructure | v1.2 | 2/2 | Complete | 2026-03-20 |
| 15. Tailscale TLS + Interface Binding | v1.2 | 2/2 | Complete | 2026-03-20 |
| 16. Auth Layer Removal | v1.2 | 2/2 | Complete | 2026-03-20 |
| 17. Dead Code Cleanup | v1.2 | 2/2 | Complete | 2026-03-20 |
| 18. Frontend Health Modal + Status UI | v1.2 | 2/2 | Complete | 2026-03-22 |
| 19. Daemon Core (Engine + IPC) | v1.3 | 2/2 | Complete | 2026-03-23 |
| 20. Process Separation | v1.3 | 2/2 | Complete | 2026-03-23 |
| 21. CLI Session + Web Commands | v1.3 | 2/2 | Complete    | 2026-03-24 |
| 22. CLI Attach | v1.3 | 1/2 | Complete    | 2026-03-24 |
| 23. Service Manager Integration | v1.3 | 1/2 | Complete    | 2026-03-24 |
| 24. CLI Polish | v1.3 | 2/2 | Complete    | 2026-03-24 |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
