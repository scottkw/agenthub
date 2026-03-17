# Requirements: AgentHub

**Defined:** 2026-03-17
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Terminal

- [ ] **TERM-01**: User can open multiple terminal tabs, each running an independent session
- [ ] **TERM-02**: User can name/rename terminal tabs for identification
- [ ] **TERM-03**: Terminal renders full ANSI color and Unicode/emoji output correctly
- [ ] **TERM-04**: Terminal provides 10K+ line scrollback buffer
- [ ] **TERM-05**: User can copy/paste text from terminal sessions
- [ ] **TERM-06**: Terminal resizes correctly when window is resized (SIGWINCH propagation)
- [ ] **TERM-07**: User can close/kill a session cleanly (process group cleanup)

### CLI

- [ ] **CLI-01**: App detects installed AI coding CLIs (Claude Code, Codex, Gemini CLI, OpenCode) via PATH
- [ ] **CLI-02**: User can launch a new session by selecting from detected CLIs
- [ ] **CLI-03**: User can configure custom CLI paths in app settings

### Session

- [ ] **SESS-01**: Sessions persist when the app window is closed (Go-native PTY backend)
- [ ] **SESS-02**: App runs in system tray when window is closed, keeping sessions alive
- [ ] **SESS-03**: User can reattach to a running session after reopening the window

### Web Serving

- [ ] **WEB-01**: User can toggle web serving on/off per session
- [ ] **WEB-02**: Web-served sessions use self-signed TLS with local CA cert pattern
- [ ] **WEB-03**: App provides in-app guidance for installing CA cert in OS trust store
- [ ] **WEB-04**: Web dashboard lists all web-served sessions with password authentication
- [ ] **WEB-05**: User can generate shareable token links for specific sessions
- [ ] **WEB-06**: Remote browser connects to session via xterm.js over WebSocket (full interaction, not read-only)

### Network

- [ ] **NET-01**: User can bind web server to a specific network interface
- [ ] **NET-02**: App auto-detects Tailscale interface via CGNAT range (100.64.0.0/10)
- [ ] **NET-03**: User can select other VPN interfaces (WireGuard, etc.) from a dropdown

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
| TERM-01 | — | Pending |
| TERM-02 | — | Pending |
| TERM-03 | — | Pending |
| TERM-04 | — | Pending |
| TERM-05 | — | Pending |
| TERM-06 | — | Pending |
| TERM-07 | — | Pending |
| CLI-01 | — | Pending |
| CLI-02 | — | Pending |
| CLI-03 | — | Pending |
| SESS-01 | — | Pending |
| SESS-02 | — | Pending |
| SESS-03 | — | Pending |
| WEB-01 | — | Pending |
| WEB-02 | — | Pending |
| WEB-03 | — | Pending |
| WEB-04 | — | Pending |
| WEB-05 | — | Pending |
| WEB-06 | — | Pending |
| NET-01 | — | Pending |
| NET-02 | — | Pending |
| NET-03 | — | Pending |
| QR-01 | — | Pending |
| QR-02 | — | Pending |
| STAT-01 | — | Pending |
| STAT-02 | — | Pending |
| PLAT-01 | — | Pending |
| PLAT-02 | — | Pending |
| PLAT-03 | — | Pending |

**Coverage:**
- v1 requirements: 29 total
- Mapped to phases: 0
- Unmapped: 29 ⚠️

---
*Requirements defined: 2026-03-17*
*Last updated: 2026-03-17 after initial definition*
