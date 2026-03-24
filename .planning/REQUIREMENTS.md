# Requirements: AgentHub

**Defined:** 2026-03-23
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.3 Requirements

Requirements for CLI + Daemon milestone. Each maps to roadmap phases.

### Daemon Infrastructure

- [x] **DAEMON-01**: Session management runs in a standalone daemon process separate from the GUI
- [x] **DAEMON-02**: Daemon communicates with clients via HTTP/JSON over Unix socket (named pipe on Windows)
- [x] **DAEMON-03**: Sessions persist when all clients (GUI and CLI) disconnect
- [x] **DAEMON-04**: GUI app connects to the daemon as a client; GUI and CLI see the same session pool
- [x] **DAEMON-05**: Daemon auto-starts when any CLI command is run and no daemon is running

### CLI Session Commands

- [x] **CLI-01**: User can create a new session from the terminal (`agenthub new`)
- [x] **CLI-02**: User can list all sessions with status from the terminal (`agenthub list`)
- [x] **CLI-03**: User can terminate a session from the terminal (`agenthub kill`)
- [x] **CLI-04**: User can rename a session from the terminal (`agenthub rename`)
- [x] **CLI-05**: User can attach to a session with full interactive PTY proxy (`agenthub attach`)
- [x] **CLI-06**: Attached session supports raw I/O, terminal resize propagation, and ctrl-c passthrough
- [x] **CLI-07**: User can detach from an attached session via configurable prefix key
- [x] **CLI-08**: Attaching to an existing session replays recent scrollback output

### CLI Web/Utility Commands

- [x] **WEB-01**: User can start/stop the Tailscale web server from the terminal (`agenthub web start/stop`)
- [x] **WEB-02**: User can check web server status from the terminal (`agenthub web status`)
- [x] **WEB-03**: User can toggle web serving per session from the terminal (`agenthub serve/unserve`)
- [x] **WEB-04**: User can run Tailscale health check from the terminal (`agenthub health`)
- [x] **WEB-05**: User can display a session's QR code in the terminal (`agenthub qr`)

### Service Management

- [x] **SVC-01**: Daemon can be installed as a platform service (launchd/systemd/Windows Service)
- [x] **SVC-02**: Daemon auto-starts on login when installed as a service
- [x] **SVC-03**: User can install/uninstall/start/stop the service from CLI (`agenthub daemon install/uninstall/start/stop`)

### CLI Polish

- [x] **POLISH-01**: All list/status commands support `--json` flag for machine-readable output
- [ ] **POLISH-02**: User can view current settings from CLI (`agenthub settings`)

## Future Requirements

### Session Access Control

- **ACL-01**: Per-session access control for shared tailnets
- **ACL-02**: Tailscale ACL tag-based session visibility

### Advanced CLI

- **ACLI-01**: TUI session picker (fzf-style interactive selection)
- **ACLI-02**: Multiple simultaneous clients attached to one session
- **ACLI-03**: Remote daemon over TCP (not just local Unix socket)

## Out of Scope

| Feature | Reason |
|---------|--------|
| TUI session picker | v2+ — CLI commands sufficient for v1.3 |
| Multiple simultaneous attach clients | v2+ — single attach per session for v1.3 |
| Remote daemon over TCP | v2+ — local Unix socket only; Tailscale web covers remote access |
| gRPC/protobuf for IPC | HTTP/JSON over Unix socket is simpler, debuggable with curl, sufficient for ~15 command types |
| Custom daemon protocol | HTTP/JSON reuses stdlib; no need for custom binary IPC protocol |
| GUI changes for daemon migration | GUI Wails bindings stay identical; App delegates to DaemonClient internally |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DAEMON-01 | Phase 20 | Complete |
| DAEMON-02 | Phase 19 | Complete |
| DAEMON-03 | Phase 20 | Complete |
| DAEMON-04 | Phase 20 | Complete |
| DAEMON-05 | Phase 20 | Complete |
| CLI-01 | Phase 21 | Complete |
| CLI-02 | Phase 21 | Complete |
| CLI-03 | Phase 21 | Complete |
| CLI-04 | Phase 21 | Complete |
| CLI-05 | Phase 22 | Complete |
| CLI-06 | Phase 22 | Complete |
| CLI-07 | Phase 22 | Complete |
| CLI-08 | Phase 22 | Complete |
| WEB-01 | Phase 21 | Complete |
| WEB-02 | Phase 21 | Complete |
| WEB-03 | Phase 21 | Complete |
| WEB-04 | Phase 21 | Complete |
| WEB-05 | Phase 21 | Complete |
| SVC-01 | Phase 23 | Complete |
| SVC-02 | Phase 23 | Complete |
| SVC-03 | Phase 23 | Complete |
| POLISH-01 | Phase 24 | Complete |
| POLISH-02 | Phase 24 | Pending |

**Coverage:**
- v1.3 requirements: 23 total
- Mapped to phases: 23
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-23*
*Last updated: 2026-03-23 after roadmap creation (v1.3 phases 19-24)*
