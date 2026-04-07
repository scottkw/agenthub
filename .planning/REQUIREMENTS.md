# Requirements: AgentHub

**Defined:** 2026-04-06
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.9 Requirements

Requirements for milestone v1.9: Remote Sessions & App Polish. Each maps to roadmap phases.

### Remote Sessions

- [x] **REM-01**: User can discover AgentHub instances running on other tailnet peers automatically
- [ ] **REM-02**: User can view a list of remote sessions with host, session name, agent type, and status in a dedicated GUI panel
- [ ] **REM-03**: User can open a remote session in the web browser directly from the GUI remote panel
- [ ] **REM-04**: CLI `agenthub list` shows local and remote sessions grouped by host by default
- [ ] **REM-05**: User can attach to a remote session from the CLI via `agenthub attach <id>` using the WebSocket relay

### Auto-Update

- [x] **UPD-01**: App checks GitHub releases for newer versions on startup and periodically
- [x] **UPD-02**: User sees a notification banner when an update is available with version info
- [x] **UPD-03**: User can trigger update with one click (opens download page for macOS DMG, or triggers platform-appropriate action)
- [x] **UPD-04**: User can check for updates manually from the Help menu

### Tailscale Onboarding

- [ ] **TS-01**: User sees enhanced install guidance with platform-specific instructions and download links when Tailscale is not installed
- [ ] **TS-02**: User can attempt auto-install of Tailscale from the health modal (brew install on macOS, with manual fallback)
- [ ] **TS-03**: User sees step-by-step configuration guidance after Tailscale install (enable HTTPS certs, etc.)

### App Menus & Polish

- [x] **MENU-01**: App has standard menus (File, Edit, Window, Help) with platform-appropriate keyboard shortcuts
- [x] **MENU-02**: Edit menu enables Cmd+C/Cmd+V clipboard operations in terminal tabs (currently silently broken on macOS)
- [x] **VER-01**: App version is injected at build time via ldflags (no hardcoded VERSION constant)
- [x] **VER-02**: Welcome screen displays the build-injected version automatically
- [x] **UI-01**: Welcome logo/title graphic has slightly rounded corners

## Future Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Remote Sessions (v2+)

- **REM-F01**: User can create a new session on a remote host from the GUI
- **REM-F02**: User can kill/rename remote sessions from the GUI
- **REM-F03**: Remote session output search and filtering

### Auto-Update (v2+)

- **UPD-F01**: Silent background download with restart prompt (VS Code style)
- **UPD-F02**: Automatic rollback on failed update

## Out of Scope

| Feature | Reason |
|---------|--------|
| In-place binary self-replacement on macOS | Breaks notarized code signing; open download page instead |
| Tailscale full management (login, ACLs, DNS) | App assists with install/setup only; ongoing management is Tailscale's domain |
| Brew subprocess auto-install from GUI | TTY detection, sudo, and PATH issues make it unreliable; copy-paste commands instead |
| Real-time push notifications for remote session changes | HTTP polling is simpler and sufficient for v1.9 |
| Tailscale Services API for peer discovery | Alpha API, not stable; HTTP probe pattern is proven |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| MENU-01 | Phase 49 | Complete |
| MENU-02 | Phase 49 | Complete |
| VER-01 | Phase 49 | Complete |
| VER-02 | Phase 49 | Complete |
| UI-01 | Phase 49 | Complete |
| REM-01 | Phase 50 | Complete |
| UPD-01 | Phase 51 | Complete |
| UPD-02 | Phase 51 | Complete |
| UPD-03 | Phase 51 | Complete |
| UPD-04 | Phase 51 | Complete |
| REM-02 | Phase 52 | Pending |
| REM-03 | Phase 52 | Pending |
| REM-04 | Phase 53 | Pending |
| REM-05 | Phase 53 | Pending |
| TS-01 | Phase 54 | Pending |
| TS-02 | Phase 54 | Pending |
| TS-03 | Phase 54 | Pending |

**Coverage:**
- v1.9 requirements: 17 total
- Mapped to phases: 17
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-06*
*Last updated: 2026-04-06 after roadmap creation (v1.9 phases 49-54)*
