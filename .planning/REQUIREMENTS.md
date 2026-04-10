# Requirements: AgentHub

**Defined:** 2026-04-10
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.12 Requirements

Requirements for v1.12 UI/UX Polish milestone. Each maps to roadmap phases.

### Terminal Padding

- [ ] **PAD-01**: User sees terminal content inset from the edges with consistent padding

### Web Server Links

- [ ] **WEB-01**: User can open the web server dashboard URL in their system browser
- [ ] **WEB-02**: User can copy the web server dashboard URL to clipboard
- [ ] **WEB-03**: User can view a QR code for the web server dashboard URL

### Terminal Theming

- [ ] **THM-01**: User can select a terminal color theme from the full xterm-theme library
- [ ] **THM-02**: User's selected theme persists across app restarts
- [ ] **THM-03**: Theme change applies immediately to all open terminal sessions

### Sidebar Polish

- [ ] **SBR-01**: Sidebar icons are visually centered when sidebar is collapsed

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
| SBR-01 | Phase 63 | Pending |
| PAD-01 | Phase 64 | Pending |
| THM-01 | Phase 65 | Pending |
| THM-02 | Phase 65 | Pending |
| THM-03 | Phase 65 | Pending |
| WEB-01 | Phase 66 | Pending |
| WEB-02 | Phase 66 | Pending |
| WEB-03 | Phase 66 | Pending |

**Coverage:**
- v1.12 requirements: 8 total
- Mapped to phases: 8
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-10*
*Last updated: 2026-04-10 after roadmap creation*
