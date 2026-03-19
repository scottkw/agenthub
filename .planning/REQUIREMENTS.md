# Requirements: AgentHub

**Defined:** 2026-03-19
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.1 Requirements

Requirements for v1.1 Polish & Build milestone. Each maps to roadmap phases.

### Build System

- [ ] **BUILD-01**: User can run `build.sh --platform macos` to compile for macOS only
- [ ] **BUILD-02**: User can run `build.sh --platform linux` to compile for Linux only
- [ ] **BUILD-03**: User can run `build.sh --platform windows` to compile for Windows only
- [ ] **BUILD-04**: User can run `build.sh --all` to compile for all platforms
- [ ] **BUILD-05**: User can run `build.sh --platform macos --sign` to code-sign and notarize the macOS build

### Terminal Core

- [x] **TERM-01**: Terminal content fills all available space in each tab with no dead space
- [x] **TERM-02**: User can press SHIFT+ to increase font size in the active terminal tab
- [x] **TERM-03**: User can press SHIFT- to decrease font size in the active terminal tab
- [x] **TERM-04**: Font size changes persist per tab and do not leak characters to the PTY

### UI Layout

- [x] **UILAY-01**: Toolbar buttons are visually larger and easy to click
- [x] **UILAY-02**: Each tab displays a status bar at the bottom showing web serving state, URL, and controls
- [x] **UILAY-03**: Web status/URL header overlay is removed from tab content area
- [ ] **UILAY-04**: User can rename a tab by double-clicking or right-clicking the tab label
- [ ] **UILAY-05**: Renamed tab name is used as the session name in the web dashboard

### Session Management

- [x] **SESS-01**: Clicking + opens a modal (not a dropdown) for creating a new session
- [x] **SESS-02**: New-session modal includes an agent picker showing available CLIs
- [x] **SESS-03**: New-session modal includes a native folder browser for selecting the working directory
- [x] **SESS-04**: Folder browser defaults to home directory, or last-used folder if one exists

### Web Dashboard

- [ ] **WEBUI-01**: Web dashboard has an improved visual design with better styling
- [ ] **WEBUI-02**: Web dashboard displays session names (from tab renames) instead of raw session IDs

### Settings

- [x] **SETT-01**: Settings modal uses a tabbed layout to reduce crowding (e.g., CLI Paths | Web Serving)
- [x] **SETT-02**: Settings modal has improved styling and visual organization

## Future Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Session Backend

- **BACK-01**: Configurable session backend: real tmux (when available) or Go-native PTY multiplexer
- **BACK-02**: Real tmux mode: sessions attachable from any external terminal via `tmux attach`

### Security

- **SEC-01**: Per-session token expiry and revocation

### Visual

- **VIS-01**: Tab color coding per CLI type
- **VIS-02**: Status heuristic patterns for non-Claude CLIs (Codex, Gemini CLI, OpenCode)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Font/theme customization beyond size | Deferred — per-tab font size covers the immediate need |
| Split panes / tiling | Each AI session gets its own tab |
| Session output search / replay | Tools like agent-sessions serve this niche |
| Plugin system for adding CLIs | Hardcoded set for now |
| Mobile app | PWA via web serving covers mobile needs |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| BUILD-01 | Phase 13 | Pending |
| BUILD-02 | Phase 13 | Pending |
| BUILD-03 | Phase 13 | Pending |
| BUILD-04 | Phase 13 | Pending |
| BUILD-05 | Phase 13 | Pending |
| TERM-01 | Phase 7 | Complete |
| TERM-02 | Phase 10 | Complete |
| TERM-03 | Phase 10 | Complete |
| TERM-04 | Phase 10 | Complete |
| UILAY-01 | Phase 7 | Complete |
| UILAY-02 | Phase 8 | Complete |
| UILAY-03 | Phase 8 | Complete |
| UILAY-04 | Phase 12 | Pending |
| UILAY-05 | Phase 12 | Pending |
| SESS-01 | Phase 11 | Complete |
| SESS-02 | Phase 11 | Complete |
| SESS-03 | Phase 11 | Complete |
| SESS-04 | Phase 11 | Complete |
| WEBUI-01 | Phase 12 | Pending |
| WEBUI-02 | Phase 12 | Pending |
| SETT-01 | Phase 9 | Complete |
| SETT-02 | Phase 9 | Complete |

**Coverage:**
- v1.1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0

---
*Requirements defined: 2026-03-19*
*Last updated: 2026-03-19 after roadmap creation*
