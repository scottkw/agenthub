# Requirements: AgentHub v1.10

**Defined:** 2026-04-08
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.10 Requirements

Requirements for Collapsible Sidebar Navigation milestone. Each maps to roadmap phases.

### Sidebar Layout

- [x] **SIDE-01**: User sees a collapsible left sidebar with navigation icons instead of top toolbar buttons
- [x] **SIDE-02**: User can toggle sidebar between collapsed (icons only, 48px) and expanded (icons + text labels, 200px) via hamburger button
- [x] **SIDE-03**: Sidebar collapsed/expanded state persists across app restarts via localStorage

### Sidebar Navigation

- [ ] **NAV-01**: User can click Home icon to open the Welcome tab
- [ ] **NAV-02**: User can click Remote icon to open the Remote Sessions panel
- [ ] **NAV-03**: User can click Sessions icon to open the Daemon Manager panel
- [ ] **NAV-04**: User can click New Tab icon to create a new terminal session
- [ ] **NAV-05**: User can click Settings icon (pinned to sidebar bottom) to open the Settings panel

### Icons

- [x] **ICON-01**: All sidebar icons use Heroicons (MIT-licensed open-source SVGs) instead of Unicode characters
- [x] **ICON-02**: Sessions uses a distinct icon (server-stack) since hamburger is now the sidebar toggle

### Tab Bar

- [ ] **TAB-01**: Tab bar retains session tabs but no longer has action buttons on the right

## Future Requirements

None — this is a focused UI restructure milestone.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Vertical tab list in sidebar | Tabs stay horizontal on top; sidebar is for navigation only |
| Theme/color customization for sidebar | Tokyo Night palette hardcoded, consistent with rest of app |
| Sidebar drag-to-resize | Fixed widths (48px/200px) are sufficient |
| Keyboard shortcuts for sidebar items | Existing app menu shortcuts cover these actions |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SIDE-01 | Phase 55 | Complete |
| SIDE-02 | Phase 55 | Complete |
| SIDE-03 | Phase 55 | Complete |
| NAV-01 | Phase 56 | Pending |
| NAV-02 | Phase 56 | Pending |
| NAV-03 | Phase 56 | Pending |
| NAV-04 | Phase 56 | Pending |
| NAV-05 | Phase 56 | Pending |
| ICON-01 | Phase 55 | Complete |
| ICON-02 | Phase 55 | Complete |
| TAB-01 | Phase 56 | Pending |

**Coverage:**
- v1.10 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0

---
*Requirements defined: 2026-04-08*
*Last updated: 2026-04-08 after roadmap creation*
