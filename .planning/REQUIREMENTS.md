# Requirements: AgentHub

**Defined:** 2026-04-11
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Milestone:** v1.13 Cross-Platform Fixes & UX

## GitHub Issues

Issues targeted for this milestone. Close after all linked requirements are satisfied.

| Issue | Title | Requirements |
|-------|-------|--------------|
| [#11](https://github.com/scottkw/agenthub/issues/11) | System tray icons bugfix | TRAY-01 through TRAY-06 |
| [#14](https://github.com/scottkw/agenthub/issues/14) | macOS install instructions | INST-01 |
| [#15](https://github.com/scottkw/agenthub/issues/15) | Application searching | DISC-01 through DISC-03 |
| [#16](https://github.com/scottkw/agenthub/issues/16) | New Settings tab layout | SETT-01 through SETT-03 |

## v1.13 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### System Tray (GitHub #11)

- [ ] **TRAY-01**: Linux system tray icon visible in supported desktop environments (GNOME, KDE, XFCE)
- [ ] **TRAY-02**: Windows system tray icon visible in notification area
- [ ] **TRAY-03**: Linux/Windows tray shows dynamic session list matching macOS menu
- [ ] **TRAY-04**: Linux/Windows tray shows status icon states (connected, error) matching macOS
- [ ] **TRAY-05**: Linux/Windows tray supports hide-on-close (window hides, daemon stays active)
- [ ] **TRAY-06**: Linux/Windows tray Quit option stops daemon and fully exits

### Install Instructions (GitHub #14)

- [ ] **INST-01**: Welcome screen macOS install command combines `brew tap` + `brew install --cask` into single copyable command

### Agent Discovery (GitHub #15)

- [ ] **DISC-01**: Daemon startup scans common directories for agent CLI binaries (nvm, Volta, Homebrew, snap, flatpak, cargo, pipx, native installers, system paths) per platform
- [ ] **DISC-02**: Detected agent paths are added to daemon PATH so agents resolve via exec.LookPath
- [ ] **DISC-03**: Tailscale binary location detected across platforms (Homebrew, system package, Windows default install)

### Settings Layout (GitHub #16)

- [ ] **SETT-01**: Settings tab replaced with single scrollable page containing all settings groups
- [ ] **SETT-02**: Each settings group has a visible section header (Appearance, Web Server, Paths, etc.)
- [ ] **SETT-03**: Existing settings functionality preserved (theme picker, web actions, save paths)

## Future Requirements

None — all scoped issues included in this milestone.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Multiple remote session connections (#13) | Larger architectural change — deferred to future milestone |
| Inter-session communication (#10) | Claude Peers MCP server integration — separate milestone scope |
| Networking enhancement (#9) | Built-in Headscale is a major feature — separate milestone |
| CLI remote session UX (#8) | CLI status bar — can be bundled with future CLI improvements |
| TUI mode (#7) | Major new interface — separate milestone |
| OpenCode theming (#17) | Bug fix — can be addressed as a quick task or next milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| TRAY-01 | Phase 67 | Pending |
| TRAY-02 | Phase 67 | Pending |
| TRAY-03 | Phase 67 | Pending |
| TRAY-04 | Phase 67 | Pending |
| TRAY-05 | Phase 67 | Pending |
| TRAY-06 | Phase 67 | Pending |
| INST-01 | Phase 68 | Pending |
| DISC-01 | Phase 68 | Pending |
| DISC-02 | Phase 68 | Pending |
| DISC-03 | Phase 68 | Pending |
| SETT-01 | Phase 69 | Pending |
| SETT-02 | Phase 69 | Pending |
| SETT-03 | Phase 69 | Pending |

**Coverage:**
- v1.13 requirements: 13 total
- Mapped to phases: 13
- Unmapped: 0

---
*Requirements defined: 2026-04-11*
*Last updated: 2026-04-11 after roadmap creation*
