# Requirements: AgentHub

**Defined:** 2026-04-16
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v2.1 Requirements

Requirements for v2.1 milestone. Each maps to roadmap phases.

### Settings Persistence (GitHub #26)

- [ ] **SET-01**: User-modified agent paths persist across app restarts
- [ ] **SET-02**: User-modified Tailscale path persists across app restarts
- [ ] **SET-03**: Clicking Save shows visible confirmation feedback (toast, flash, or inline indicator)

### Tailscale Detection (GitHub #27)

- [ ] **TS-01**: Tailscale is detected when installed via Homebrew, system package manager, Snap, Flatpak, or Windows default location
- [ ] **TS-02**: Tailscale connection state (connected/disconnected) is reliably reported across all platforms

### Banner Notifications (GitHub #28)

- [ ] **BAN-01**: When multiple notifications are active, they stack vertically instead of side-by-side
- [ ] **BAN-02**: Each stacked notification remains independently dismissible

### Path File Browsing (GitHub #31)

- [ ] **SET-04**: Each path entry in Settings > Paths has a browse button that opens a native file/folder picker
- [ ] **SET-05**: Selecting a path via the browser populates the corresponding input field

### Minimize to Tray (GitHub #25)

- [ ] **TRAY-01**: Settings includes a toggle for "Start minimized to system tray"
- [ ] **TRAY-02**: When enabled, launching AgentHub opens with window hidden and only tray icon visible
- [ ] **TRAY-03**: Minimize-to-tray preference persists across app restarts

## Future Requirements

Deferred to future release. Tracked but not in current roadmap.

### TUI Enhancements

- **TUI-11**: Embedded terminal preview — live session output in viewport without full attach
- **TUI-12**: Theme selection panel in TUI settings
- **TUI-13**: Per-session web serving toggle from TUI

### Multi-Client Enhancements

- **MC-07**: Input locking — host can take exclusive control, preventing observers from typing
- **MC-08**: GUI displays viewer count badge in session list

### File Management

- **FILE-01**: File browser tab for browsing session directory files locally and remotely (GitHub #24)
- **FILE-02**: Upload/download capability for session files
- **FILE-03**: File preview for common file types

### Intersession Communication

- **IPC-01**: Support for or fork of Claude Peers MCP server for cross-session orchestration (GitHub #10)

### Networking

- **NET-01**: Built-in Headscale server with embedded Tailscale client (GitHub #9)

### Mobile

- **MOB-01**: iOS and Android mobile app versions of AgentHub (GitHub #30)

### TUI Polish

- **TUIP-01**: TUI UI/UX overhaul to visually match GUI with frames and tabs (GitHub #29)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Mobile app (GitHub #30) | Major new platform — separate milestone |
| Headscale networking (GitHub #9) | Large infrastructure project — separate milestone |
| File browser tab (GitHub #24) | Significant new feature — separate milestone |
| Intersession comms (GitHub #10) | Requires MCP server research — separate milestone |
| TUI visual overhaul (GitHub #29) | Cosmetic TUI work — separate milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SET-01 | Phase 79 | Pending |
| SET-02 | Phase 79 | Pending |
| SET-03 | Phase 79 | Pending |
| SET-04 | Phase 79 | Pending |
| SET-05 | Phase 79 | Pending |
| TS-01 | Phase 80 | Pending |
| TS-02 | Phase 80 | Pending |
| BAN-01 | Phase 81 | Pending |
| BAN-02 | Phase 81 | Pending |
| TRAY-01 | Phase 82 | Pending |
| TRAY-02 | Phase 82 | Pending |
| TRAY-03 | Phase 82 | Pending |

**Coverage:**
- v2.1 requirements: 12 total
- Mapped to phases: 12
- Unmapped: 0

**GitHub Issues:**
- #26 → SET-01, SET-02, SET-03 → Phase 79
- #31 → SET-04, SET-05 → Phase 79
- #27 → TS-01, TS-02 → Phase 80
- #28 → BAN-01, BAN-02 → Phase 81
- #25 → TRAY-01, TRAY-02, TRAY-03 → Phase 82

---
*Requirements defined: 2026-04-16*
*Last updated: 2026-04-16 after roadmap creation*
