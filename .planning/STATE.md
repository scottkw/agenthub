---
gsd_state_version: 1.0
milestone: v1.9
milestone_name: Remote Sessions & App Polish
status: verifying
stopped_at: Completed 49-02-PLAN.md
last_updated: "2026-04-07T16:26:02.093Z"
last_activity: 2026-04-07
progress:
  total_phases: 6
  completed_phases: 1
  total_plans: 2
  completed_plans: 2
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-06)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 49 — app-menus-version-injection

## Current Position

Phase: 50
Plan: Not started
Status: Phase complete — ready for verification
Last activity: 2026-04-07

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- v1.8 plans completed: 9
- v1.8 phases: 5
- v1.8 timeline: 2026-04-03 → 2026-04-06 (3 days)
- Cumulative: 48 phases, 86 plans across 9 milestones

## Accumulated Context

### Decisions

- No binary self-replacement on macOS/Windows — update flow is detect → notify → open browser (Gatekeeper/file-lock constraint)
- Daemon API is Unix-socket-only — remote peer probing targets web server HTTPS `/api/sessions`, not daemon port
- Brew subprocess auto-install excluded from scope — show copyable commands only (TTY/sudo/PATH issues)
- `creativeprojects/go-selfupdate@v1.5.2` chosen over rhysd predecessor (active maintenance, Dec 2025)
- Menu ordering required: `NewMenu()` → `AppMenu()` → `EditMenu()` → custom submenus (Wails pitfall)
- [Phase 49]: Package-level appCtx for menu callback context avoids closure complexity in Wails
- [Phase 49]: appMenu() ordering: AppMenu first per STATE.md Wails pitfall; FileMenu/HelpSubMenu commented out in v2.10.2 — use AddSubmenu() instead
- [Phase 49]: CSS ?raw import returns empty string in Vitest jsdom — use readFileSync for CSS assertions (matches TabBar/TerminalPanel test convention)

### Pending Todos

None.

### Blockers/Concerns

- WinGet first submission to microsoft/winget-pkgs deferred until first release is published (tracked in 48-HUMAN-UAT.md)
- Validate `go-selfupdate ParseSlug("scottkw/agenthub")` matches v1.8 artifact naming before finalizing Phase 51
- Confirm Tailscale Let's Encrypt certs are FQDN-only (no IP SANs) before finalizing Phase 50 probe TLS mode

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260406-nqy | Dynamic dock icon visibility - show when window visible, hide when minimized/closed to tray | 2026-04-07 | 82e501c | [260406-nqy-dynamic-dock-icon-visibility-show-when-w](./quick/260406-nqy-dynamic-dock-icon-visibility-show-when-w/) |
| 260406-op4 | Tray icon A matches app icon A - monochrome for macOS, full color for other OSes | 2026-04-07 | 45ffbd2 | [260406-op4-tray-icon-a-matches-app-icon-a-monochrom](./quick/260406-op4-tray-icon-a-matches-app-icon-a-monochrom/) |
| 260406-s0e | Fix CLI detection - export AugmentServicePath and call in GUI startup | 2026-04-07 | eb90fa6 | [260406-s0e-fix-cli-detection-app-shows-no-clis-dete](./quick/260406-s0e-fix-cli-detection-app-shows-no-clis-dete/) |
| Phase 49 P01 | 155 | 2 tasks | 4 files |
| Phase 49 P02 | 390 | 2 tasks | 5 files |

## Session Continuity

Last session: 2026-04-07T16:22:20.459Z
Stopped at: Completed 49-02-PLAN.md
Resume file: None
Next action: `/gsd:plan-phase 49`
