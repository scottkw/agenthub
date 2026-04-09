# Requirements: AgentHub

**Defined:** 2026-04-08
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.11 Requirements

Requirements for milestone v1.11: Local Network & UX Polish.

### CLI Detection

- [x] **DET-01**: User can launch Claude Code sessions when Claude is installed via Anthropic native installer (~/.local/bin/claude on macOS/Linux, %USERPROFILE%\.local\bin\claude.exe on Windows)

### UI Polish

- [x] **UI-01**: Sidebar displays "New Session" instead of "New Tab"
- [x] **UI-02**: User can access Settings as a sidebar tab (not a modal), consistent with Home/Remote/Sessions panels

### Auto-Serve

- [x] **SERVE-01**: Web server starts automatically when daemon launches (no manual start required)
- [x] **SERVE-02**: New sessions have web serving enabled automatically when the web server is running

### Local Network

- [ ] **NET-01**: User can serve sessions over the local network with self-signed TLS when Tailscale is not available
- [ ] **NET-02**: Local network mode generates a random password for all web connections via HTTP Basic Auth
- [ ] **NET-03**: Web server binds to a local network interface when operating in local mode
- [ ] **NET-04**: User sees a persistent nudge banner on each launch recommending Tailscale installation when in local mode
- [ ] **NET-05**: User can view the generated password in the UI (settings/status area)

## Future Requirements

Deferred to future milestones. Tracked but not in current roadmap.

### Cross-Platform

- **PLAT-01**: Linux system tray icon and functionality
- **PLAT-02**: Windows system tray icon and functionality

### Advanced

- **ADV-01**: TUI option mimicking the GUI
- **ADV-02**: Inter-session Claude Peers MCP server support
- **ADV-03**: Headscale + built-in Tailscale client option
- **ADV-04**: Theme support

## Out of Scope

| Feature | Reason |
|---------|--------|
| Configurable Tailscale vs Headscale | Future milestone — v1.11 adds local fallback only |
| Theme customization | Future milestone — per-tab font size covers immediate need |
| Tab color coding per CLI type | Deferred to future milestone |
| GitHub Actions build/release fixes | Tracked separately as quick tasks |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DET-01 | Phase 57 | Complete |
| UI-01 | Phase 57 | Complete |
| UI-02 | Phase 58 | Complete |
| SERVE-01 | Phase 59 | Complete |
| SERVE-02 | Phase 59 | Complete |
| NET-01 | Phase 60 | Pending |
| NET-02 | Phase 60 | Pending |
| NET-03 | Phase 60 | Pending |
| NET-04 | Phase 60 | Pending |
| NET-05 | Phase 60 | Pending |

**Coverage:**
- v1.11 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0

---
*Requirements defined: 2026-04-08*
*Last updated: 2026-04-08 after roadmap creation (v1.11 Phases 57-60)*
