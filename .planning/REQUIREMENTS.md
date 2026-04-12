# Requirements: AgentHub

**Defined:** 2026-04-10
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.12 Requirements

Requirements for v1.12 UI/UX Polish milestone. Each maps to roadmap phases.

### Terminal Padding

- [x] **PAD-01**: User sees terminal content inset from the edges with consistent padding

### Web Server Links

- [x] **WEB-01**: User can open the web server dashboard URL in their system browser
- [x] **WEB-02**: User can copy the web server dashboard URL to clipboard
- [x] **WEB-03**: User can view a QR code for the web server dashboard URL

### Terminal Theming

- [x] **THM-01**: User can select a terminal color theme from the full xterm-theme library
- [x] **THM-02**: User's selected theme persists across app restarts
- [x] **THM-03**: Theme change applies immediately to all open terminal sessions

### Sidebar Polish

- [x] **SBR-01**: Sidebar icons are visually centered when sidebar is collapsed

## Future Requirements

Deferred to future release. Tracked but not in current roadmap.

### Appearance

- **APP-01**: User can select terminal font family from available system/web fonts
- **APP-02**: User can create custom color themes
- **APP-03**: User can set per-tab theme overrides

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Font family selection | Web font loading race adds complexity; validate demand first |
| Custom theme editor | High complexity, marginal benefit for v1.12 |
| Per-tab theme overrides | Global theme sufficient; per-tab adds UI complexity |
| Configurable padding value | Fixed value covers the need; settings bloat otherwise |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SBR-01 | Phase 63 | Satisfied |
| PAD-01 | Phase 64 | Satisfied |
| THM-01 | Phase 65 | Satisfied |
| THM-02 | Phase 65 | Satisfied |
| THM-03 | Phase 65 | Satisfied |
| WEB-01 | Phase 66 | Satisfied |
| WEB-02 | Phase 66 | Satisfied |
| WEB-03 | Phase 66 | Satisfied |

**Coverage:**
- v1.12 requirements: 8 total
- Mapped to phases: 8
- Unmapped: 0
- Satisfied: 8/8

---
*Requirements defined: 2026-04-10*
*Last updated: 2026-04-11 after milestone audit gap closure*
