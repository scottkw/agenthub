# Requirements: AgentHub

**Defined:** 2026-03-23
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.3 Requirements

Requirements for CLI + Daemon milestone. Each maps to roadmap phases.

### Daemon Infrastructure

- [ ] **DAEMON-01**: Session management runs in a standalone daemon process separate from the GUI
- [ ] **DAEMON-02**: Daemon communicates with clients via HTTP/JSON over Unix socket (named pipe on Windows)
- [ ] **DAEMON-03**: Sessions persist when all clients (GUI and CLI) disconnect
- [ ] **DAEMON-04**: GUI app connects to the daemon as a client; GUI and CLI see the same session pool
- [ ] **DAEMON-05**: Daemon auto-starts when any CLI command is run and no daemon is running

### CLI Session Commands

- [ ] **CLI-01**: User can create a new session from the terminal (`agenthub new`)
- [ ] **CLI-02**: User can list all sessions with status from the terminal (`agenthub list`)
- [ ] **CLI-03**: User can terminate a session from the terminal (`agenthub kill`)
- [ ] **CLI-04**: User can rename a session from the terminal (`agenthub rename`)
- [ ] **CLI-05**: User can attach to a session with full interactive PTY proxy (`agenthub attach`)
- [ ] **CLI-06**: Attached session supports raw I/O, terminal resize propagation, and ctrl-c passthrough
- [ ] **CLI-07**: User can detach from an attached session via configurable prefix key
- [ ] **CLI-08**: Attaching to an existing session replays recent scrollback output

### CLI Web/Utility Commands

- [ ] **WEB-01**: User can start/stop the Tailscale web server from the terminal (`agenthub web start/stop`)
- [ ] **WEB-02**: User can check web server status from the terminal (`agenthub web status`)
- [ ] **WEB-03**: User can toggle web serving per session from the terminal (`agenthub serve/unserve`)
- [ ] **WEB-04**: User can run Tailscale health check from the terminal (`agenthub health`)
- [ ] **WEB-05**: User can display a session's QR code in the terminal (`agenthub qr`)

### Service Management

- [ ] **SVC-01**: Daemon can be installed as a platform service (launchd/systemd/Windows Service)
- [ ] **SVC-02**: Daemon auto-starts on login when installed as a service
- [ ] **SVC-03**: User can install/uninstall/start/stop the service from CLI (`agenthub daemon install/uninstall/start/stop`)

### CLI Polish

- [ ] **POLISH-01**: All list/status commands support `--json` flag for machine-readable output
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
| DAEMON-01 | — | Pending |
| DAEMON-02 | — | Pending |
| DAEMON-03 | — | Pending |
| DAEMON-04 | — | Pending |
| DAEMON-05 | — | Pending |
| CLI-01 | — | Pending |
| CLI-02 | — | Pending |
| CLI-03 | — | Pending |
| CLI-04 | — | Pending |
| CLI-05 | — | Pending |
| CLI-06 | — | Pending |
| CLI-07 | — | Pending |
| CLI-08 | — | Pending |
| WEB-01 | — | Pending |
| WEB-02 | — | Pending |
| WEB-03 | — | Pending |
| WEB-04 | — | Pending |
| WEB-05 | — | Pending |
| SVC-01 | — | Pending |
| SVC-02 | — | Pending |
| SVC-03 | — | Pending |
| POLISH-01 | — | Pending |
| POLISH-02 | — | Pending |

**Coverage:**
- v1.3 requirements: 23 total
- Mapped to phases: 0
- Unmapped: 23 ⚠️

---
*Requirements defined: 2026-03-23*
*Last updated: 2026-03-23 after initial definition*
