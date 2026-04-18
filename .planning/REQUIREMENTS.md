# Requirements: AgentHub

**Defined:** 2026-04-17
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v3.0 Requirements

Requirements for v3.0 milestone. Each maps to roadmap phases.

**GitHub Issues:** #34, #33, #32, #29

### Settings UI (#34)

- [ ] **SET-01**: Path column header and entry boxes for Tailscale align with CLI path entries in Settings > Paths section
- [ ] **SET-02**: All Settings sections audited for visual consistency (alignment, spacing, headers)

### Session Lifecycle (#33)

- [ ] **SESS-01**: Session tab automatically closes when the agent process exits
- [ ] **SESS-02**: Brief delay before auto-close allows final output to flush to terminal
- [ ] **SESS-03**: Toast or indicator notifies user that the agent exited before the tab closes

### Application Lifecycle (#32)

- [ ] **APP-01**: Quit action (GUI close / tray Quit) shows a confirmation modal
- [ ] **APP-02**: Modal offers two choices: quit GUI only (daemon stays running) or quit both GUI and daemon
- [ ] **APP-03**: Modal displays count of currently active sessions as context for the decision

### TUI Polish (#29)

- [ ] **TUI-01**: Session list displayed in bordered lipgloss frames with section headers
- [ ] **TUI-02**: Tabbed navigation mimicking GUI sidebar (Home, Sessions, Remote, Settings)
- [ ] **TUI-03**: Styled session rows with agent type, status glyphs, hostname, and viewer count matching GUI aesthetic
- [ ] **TUI-04**: Consistent color palette and typography between TUI and GUI (TokyoNight-derived)

## Future Requirements

Deferred to future release. Tracked but not in current roadmap.

### File Browser (#24)

- **FILE-01**: User can browse files in any session's working directory
- **FILE-02**: User can browse files on remote sessions
- **FILE-03**: User can upload/download files
- **FILE-04**: User can preview common file types

### Intersession Communication (#10)

- **IPC-01**: Claude Peers MCP server support or fork for cross-session orchestration

### Networking Enhancement (#9)

- **NET-01**: Embedded Tailscale client with Headscale server option
- **NET-02**: Cloud-based Headscale server with Docker Compose stack
- **NET-03**: Supabase auth integration
- **NET-04**: ACL enforcement for network isolation

### Mobile App (#30)

- **MOB-01**: iOS and Android versions of AgentHub (PWA + native wrapper)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| TUI daemon management (install/uninstall) | CLI subcommands already handle this; rare operation |
| TUI remote attach | Deferred — WSS relay attach is future scope (toast displayed) |
| Custom TUI theme editor | TokyoNight-derived palette covers the need; custom editing deferred |
| Quit modal for CLI/TUI | Desktop GUI only — CLI/TUI users expect Ctrl-C behavior |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SET-01 | Phase 83 | Pending |
| SET-02 | Phase 83 | Pending |
| SESS-01 | Phase 84 | Pending |
| SESS-02 | Phase 84 | Pending |
| SESS-03 | Phase 84 | Pending |
| APP-01 | Phase 85 | Pending |
| APP-02 | Phase 85 | Pending |
| APP-03 | Phase 85 | Pending |
| TUI-01 | Phase 86 | Pending |
| TUI-02 | Phase 86 | Pending |
| TUI-03 | Phase 86 | Pending |
| TUI-04 | Phase 86 | Pending |

**Coverage:**
- v3.0 requirements: 12 total
- Mapped to phases: 12
- Unmapped: 0

---
*Requirements defined: 2026-04-17*
*Last updated: 2026-04-17 — traceability filled after roadmap creation*
