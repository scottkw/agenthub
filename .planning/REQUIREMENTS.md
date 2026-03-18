# Requirements: AgentHub

**Defined:** 2026-03-17
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Terminal

- [x] **TERM-01**: User can open multiple terminal tabs, each running an independent session
- [x] **TERM-02**: User can name/rename terminal tabs for identification
- [x] **TERM-03**: Terminal renders full ANSI color and Unicode/emoji output correctly
- [x] **TERM-04**: Terminal provides 10K+ line scrollback buffer
- [x] **TERM-05**: User can copy/paste text from terminal sessions
- [x] **TERM-06**: Terminal resizes correctly when window is resized (SIGWINCH propagation)
- [x] **TERM-07**: User can close/kill a session cleanly (process group cleanup)

### CLI

- [x] **CLI-01**: App detects installed AI coding CLIs (Claude Code, Codex, Gemini CLI, OpenCode) via PATH
- [x] **CLI-02**: User can launch a new session by selecting from detected CLIs
- [x] **CLI-03**: User can configure custom CLI paths in app settings

### Session

- [x] **SESS-01**: Sessions persist when the app window is closed (Go-native PTY backend)
- [x] **SESS-02**: App runs in system tray when window is closed, keeping sessions alive
- [x] **SESS-03**: User can reattach to a running session after reopening the window

### Web Serving

- [x] **WEB-01**: User can toggle web serving on/off per session
- [x] **WEB-02**: Web-served sessions use self-signed TLS with local CA cert pattern
- [x] **WEB-03**: App provides in-app guidance for installing CA cert in OS trust store
- [x] **WEB-04**: Web dashboard lists all web-served sessions with password authentication
- [x] **WEB-05**: User can generate shareable token links for specific sessions
- [x] **WEB-06**: Remote browser connects to session via xterm.js over WebSocket (full interaction, not read-only)

### Network

- [x] **NET-01**: User can bind web server to a specific network interface
- [x] **NET-02**: App auto-detects Tailscale interface via CGNAT range (100.64.0.0/10)
- [x] **NET-03**: User can select other VPN interfaces (WireGuard, etc.) from a dropdown

### QR

- [ ] **QR-01**: App generates QR codes for all web-served session URLs
- [ ] **QR-02**: QR codes are displayed in the desktop app and on the web dashboard

### Status

- [ ] **STAT-01**: Each tab shows session status: running, waiting for input, idle, or errored
- [ ] **STAT-02**: Status detection uses heuristic parsing of CLI output patterns

### Platform

- [ ] **PLAT-01**: App builds and runs on macOS
- [ ] **PLAT-02**: App builds and runs on Linux
- [ ] **PLAT-03**: App builds and runs on Windows

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Session Backend

- **TMUX-01**: User can choose real tmux as session backend (macOS/Linux only)
- **TMUX-02**: Sessions managed by real tmux are attachable via `tmux attach` from any terminal

### Polish

- **POL-01**: Per-session token expiry and revocation
- **POL-02**: Tab color coding per CLI type
- **POL-03**: Font and theme customization for terminal sessions

## Out of Scope

| Feature | Reason |
|---------|--------|
| CLI installation/management | Version management, permissions, update handling are each a project; adds liability |
| Tailscale/VPN installation or management | VPN setup requires privilege escalation and is a separate concern |
| Let's Encrypt / ACME cert management | Requires domain name and internet-accessible endpoint; this is local-first |
| User accounts / registration system | Single-user per installation; password + tokens handles all auth needs |
| Cloud hosting / SaaS deployment | Contradicts local-first design; adds billing, data residency, support burden |
| Plugin system for new CLIs | Premature abstraction; hardcode initial set, extend via code contributions |
| Session output search / replay | High complexity; tools like agent-sessions already serve this niche |
| MCP server management | Agent-deck targets this; requires per-CLI MCP config understanding |
| Split panes / tiling within a tab | Each AI session gets its own tab; split panes add complexity without value |
| Mobile app | Web UI served from desktop app is the remote access mechanism |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| TERM-01 | Phase 3 | Complete |
| TERM-02 | Phase 3 | Complete |
| TERM-03 | Phase 3 | Complete |
| TERM-04 | Phase 3 | Complete |
| TERM-05 | Phase 3 | Complete |
| TERM-06 | Phase 1 | Complete |
| TERM-07 | Phase 1 | Complete |
| CLI-01 | Phase 1 | Complete |
| CLI-02 | Phase 1 | Complete |
| CLI-03 | Phase 3 | Complete |
| SESS-01 | Phase 1 | Complete |
| SESS-02 | Phase 3 | Complete |
| SESS-03 | Phase 2 | Complete |
| WEB-01 | Phase 4 | Complete |
| WEB-02 | Phase 4 | Complete |
| WEB-03 | Phase 4 | Complete |
| WEB-04 | Phase 4 | Complete |
| WEB-05 | Phase 4 | Complete |
| WEB-06 | Phase 4 | Complete |
| NET-01 | Phase 4 | Complete |
| NET-02 | Phase 4 | Complete |
| NET-03 | Phase 4 | Complete |
| QR-01 | Phase 5 | Pending |
| QR-02 | Phase 5 | Pending |
| STAT-01 | Phase 5 | Pending |
| STAT-02 | Phase 5 | Pending |
| PLAT-01 | Phase 6 | Pending |
| PLAT-02 | Phase 6 | Pending |
| PLAT-03 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 29 total
- Mapped to phases: 29
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-17*
*Last updated: 2026-03-17 after roadmap creation — all 29 requirements mapped*
