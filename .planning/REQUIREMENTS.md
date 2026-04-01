# Requirements: AgentHub

**Defined:** 2026-03-31
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.7 Requirements

Requirements for v1.7 Daemon UX & Branding. Each maps to roadmap phases.

### System Tray

- [ ] **TRAY-01**: User sees AgentHub icon in system tray (macOS menu bar, Windows notification area, Linux tray)
- [ ] **TRAY-02**: User can right-click tray icon to see menu with "Open AgentHub" and "Quit" actions
- [ ] **TRAY-03**: Tray icon visually reflects daemon state (running vs error/disconnected)
- [ ] **TRAY-04**: Tray menu lists active sessions by name; clicking a session focuses it in the GUI
- [ ] **TRAY-05**: macOS dock icon is hidden (LSUIElement) — app lives in menu bar only
- [ ] **TRAY-06**: Tray icon tooltip shows active session count on hover

### Daemon Management

- [ ] **DMGR-01**: Closing the GUI window hides it instead of quitting — daemon and tray icon remain active
- [ ] **DMGR-02**: "Quit" from tray menu stops daemon and fully exits the application
- [ ] **DMGR-03**: Daemon management panel inside existing GUI window showing session list with status, start/stop, kill, and web-serve controls

### Remote Session Indicators

- [ ] **RMTE-01**: Web terminal (browser) displays a status bar showing session name, agent type, host machine name, and connection state
- [ ] **RMTE-02**: CLI `agenthub attach` prints connection banner to stderr showing session name, agent, hostname, and detach key before PTY stream
- [x] **RMTE-03**: Session metadata from daemon includes machine hostname (`os.Hostname()`) for remote identification

### Branding

- [x] **BRND-01**: App icon set generated from logomark: .icns (macOS), .ico (Windows), multi-size PNGs (Linux/Wails)
- [x] **BRND-02**: Splash screen shows full title logo during app startup, dismissed when daemon connection confirmed (no artificial delay, 3s timeout fallback)
- [ ] **BRND-03**: macOS tray icon uses monochrome template image that adapts to light/dark menu bar

## Future Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Tray Enhancements

- **TRAY-07**: Session count badge overlay on tray icon
- **TRAY-08**: Dark/light adaptive tray icon for Windows/Linux (beyond macOS template)

### Daemon Management Enhancements

- **DMGR-04**: Separate mini management popup window (requires Wails v3 multi-window)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Tray icon on daemon binary (separate process) | kardianos/service daemons have no UI access; tray must live in GUI binary |
| Animated/spinning tray icon | CPU-intensive on macOS, deprecated on Windows 10+ |
| Full terminal in mini management window | Would duplicate main GUI; xterm.js canvas too heavyweight for popup |
| Splash screen with fixed minimum duration | Artificial delays frustrate users; dismiss as soon as ready |
| Wails v3 migration | Staying on v2; v3 is alpha |
| Second OS-level window for management | Wails v2 is single-window only |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| BRND-01 | Phase 36 | Complete |
| BRND-02 | Phase 37 | Complete |
| RMTE-03 | Phase 38 | Complete |
| RMTE-01 | Phase 39 | Pending |
| RMTE-02 | Phase 39 | Pending |
| DMGR-03 | Phase 40 | Pending |
| TRAY-01 | Phase 41 | Pending |
| TRAY-02 | Phase 41 | Pending |
| TRAY-03 | Phase 41 | Pending |
| TRAY-04 | Phase 41 | Pending |
| TRAY-05 | Phase 41 | Pending |
| TRAY-06 | Phase 41 | Pending |
| DMGR-01 | Phase 41 | Pending |
| DMGR-02 | Phase 41 | Pending |
| BRND-03 | Phase 41 | Pending |

**Coverage:**
- v1.7 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-31*
*Last updated: 2026-03-31 after roadmap creation — all 15 requirements mapped to Phases 36-41*
