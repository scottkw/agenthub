# Requirements: AgentHub

**Defined:** 2026-04-14
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v2.0 Requirements

Requirements for v2.0 milestone. Each maps to roadmap phases.

### Multi-Client Sessions (GitHub #13)

- [ ] **MC-01**: Multiple WebSocket clients can connect to the same session and receive live output simultaneously
- [ ] **MC-02**: Each connected client maintains independent scrollback position
- [ ] **MC-03**: User can attach in read-only mode via `--readonly` flag (input suppressed, output received)
- [ ] **MC-04**: Daemon tracks connection count per session and exposes it via session metadata API
- [ ] **MC-05**: Clients can provide an identity name at connection (e.g. `?client=macbook`) visible in session metadata
- [ ] **MC-06**: PTY resize arbitration prevents dimension thrash when multiple clients have different terminal sizes

### CLI Status Bar (GitHub #8)

- [ ] **SB-01**: CLI attach displays a persistent tmux-style bottom bar with session name, agent type, hostname, detach hint, and elapsed time
- [ ] **SB-02**: Status bar refreshes on a timer without corrupting terminal output (DECSTBM scroll region)
- [ ] **SB-03**: Status bar is suppressed when stdout is not a TTY
- [ ] **SB-04**: Status bar shows viewer count when multiple clients are connected
- [ ] **SB-05**: Status bar shows connection state (connected/reconnecting/latency) for remote sessions
- [ ] **SB-06**: User can place status bar at top via `--status-top` flag (bottom is default)
- [ ] **SB-07**: Status bar cleans up on detach/exit — clears bar line and restores terminal state

### TUI Mode (GitHub #7)

- [ ] **TUI-01**: `agenthub tui` command launches a Bubble Tea terminal UI
- [ ] **TUI-02**: Session list panel shows all sessions with status indicators, agent type, hostname, and viewer count
- [x] **TUI-03**: User can attach to a session from the list (TUI suspends, raw PTY attach, TUI resumes on detach)
- [x] **TUI-04**: User can create a new session via modal (agent picker, working directory, extra args)
- [x] **TUI-05**: User can kill a session with confirmation dialog
- [x] **TUI-06**: User can rename a session via inline edit or modal
- [ ] **TUI-07**: Remote sessions panel shows tailnet peer sessions with same grouping as GUI
- [ ] **TUI-08**: Web server status displayed in footer/status area
- [ ] **TUI-09**: Help overlay (? key) shows all keybindings for current view
- [ ] **TUI-10**: ASCII QR code display for session web URL

## Future Requirements

Deferred to future release. Tracked but not in current roadmap.

### Multi-Client Enhancements

- **MC-07**: Input locking — host can take exclusive control, preventing observers from typing
- **MC-08**: GUI displays viewer count badge in session list

### TUI Enhancements

- **TUI-11**: Embedded terminal preview — live session output in viewport without full attach
- **TUI-12**: Theme selection panel in TUI settings
- **TUI-13**: Per-session web serving toggle from TUI

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Synchronized scrollback across clients | Violates universal terminal-sharing contract — independent scrollback is expected |
| Mouse-driven TUI navigation | Interferes with AI agent mouse usage during attach |
| Split-pane tiling in TUI | Already out of scope for GUI; doubly complex for TUI |
| Full xterm.js rendering in TUI | Requires browser/WebView canvas — cannot render in terminal; raw PTY attach is the correct pattern |
| TUI daemon management (install/uninstall) | CLI subcommands already handle this; rare operation doesn't justify TUI surface area |
| Rich color themes for status bar | Functional, not decorative — ship one well-chosen scheme; add configurability only on demand |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| MC-01 | Phase 74 | Pending |
| MC-02 | Phase 74 | Pending |
| MC-03 | Phase 74 | Pending |
| MC-04 | Phase 74 | Pending |
| MC-05 | Phase 74 | Pending |
| MC-06 | Phase 74 | Pending |
| SB-01 | Phase 75 | Pending |
| SB-02 | Phase 75 | Pending |
| SB-03 | Phase 75 | Pending |
| SB-04 | Phase 75 | Pending |
| SB-05 | Phase 75 | Pending |
| SB-06 | Phase 75 | Pending |
| SB-07 | Phase 75 | Pending |
| TUI-01 | Phase 76 | Pending |
| TUI-02 | Phase 76 | Pending |
| TUI-08 | Phase 76 | Pending |
| TUI-09 | Phase 76 | Pending |
| TUI-03 | Phase 77 | Complete |
| TUI-04 | Phase 77 | Complete |
| TUI-05 | Phase 77 | Complete |
| TUI-06 | Phase 77 | Complete |
| TUI-07 | Phase 78 | Pending |
| TUI-10 | Phase 78 | Pending |

**Coverage:**
- v2.0 requirements: 23 total
- Mapped to phases: 23
- Unmapped: 0

**GitHub Issues:**
- #13 → MC-01 through MC-06 → Phase 74
- #8 → SB-01 through SB-07 → Phase 75
- #7 → TUI-01 through TUI-10 → Phases 76-78

---
*Requirements defined: 2026-04-14*
*Last updated: 2026-04-14 — traceability filled after roadmap creation*
