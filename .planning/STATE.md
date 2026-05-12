---
gsd_state_version: 1.0
milestone: v3.3
milestone_name: Shell Sessions & Polish
status: planning
last_updated: "2026-05-12T22:44:26.235Z"
last_activity: 2026-05-12
progress:
  total_phases: 7
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-12)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 100 — shell-session-backend-and-discovery (v3.3 entry phase, unblocks deferred v3.2 UAT)

## Current Position

Phase: Phase 100 — Shell Session Backend & Discovery
Plan: — (planning not yet started; run `/gsd-plan-phase 100`)
Status: Roadmap created, awaiting phase planning
Last activity: 2026-05-12 — v3.3 roadmap created (7 phases, 100-106, 28/28 requirements mapped)

## Performance Metrics

**Velocity:**

- v3.2 phases: 8 (Phases 92-99)
- v3.2 plans completed: 44
- v3.2 timeline: 2026-05-03 → 2026-05-12 (~9 days)
- Cumulative: 98 phases, 230 plans across 20 milestones (Phase 91 deferred, absorbed by v3.3 Phase 106)

## Accumulated Context

### Decisions

- v3.3 scope addresses GitHub Issues #44 (shell agent) + #45 (Settings hyperlinked index), absorbs v3.2 polish carry-over (P-1..P-6), re-runs 9 deferred v3.2 UAT scenarios, and absorbs the deferred Phase 91 v3.1 distribution-pipeline followups.
- Phase numbering continues from v3.2 (Phase 99 last shipped). v3.3 spans Phases 100-106 (7 phases). The Phase 91 deferred directory (`.planning/deferred/91-distribution-pipeline-followups/`) is absorbed into Phase 106 — the number 91 is NOT reused.
- 7-phase shape (100-106) follows the recommended target: SHELL split into two phases (backend then surfaces+gating), POLISH split into two phases (web-links cluster vs find-bar/test-env/IIP cluster), one phase each for SETUI, UAT closure, DIST. SHELL is sequenced first because it unblocks the UAT batch in Phase 105.
- Phase 100 (Shell backend) is the foundation — daemon-side PTY plumbing and platform shell discovery; no surface work, no web-share considerations. Lays the groundwork that 101 (surfaces+gating) and 105 (UAT) both depend on.
- Phase 101 (Shell surfaces + web-share gating) combines GUI/CLI/TUI surface work with web-share-disabled-by-default override + one-time arbitrary-execution confirmation banner. Combined because the gating UI (SHELL-07/08) is meaningless without the surfaces (SHELL-01/02/03) being present first.
- Phase 102 (Web-Links polish) clusters POLISH-01 (mailto) and POLISH-02 (Cyrillic IDN) because both live in the same addon-web-links URL matcher / `LinkConfirmPopover` wiring path; one edit window touches both.
- Phase 103 (Find-bar + test-env + IIP polish) clusters POLISH-03 (Esc-after-toggle), POLISH-04 (slide-out animation), POLISH-05 (iTerm2 IIP investigation), and POLISH-06 (Vitest localStorage). 03+04 live in the same `FindBar.tsx` / `web/assets/terminal.js` files; 05 is a single-investigation item that doesn't justify its own phase; 06 is a tiny vitest setupFile change. Folding all four into one polish phase avoids fragmentation.
- Phase 104 (Settings hyperlinked index) is independent of all other v3.3 work — pure additive UI within `SettingsTab.tsx`. Can run in parallel with shell phases if planning chooses.
- Phase 105 (Deferred UAT re-run) is sequenced AFTER 100/101/102/103 because every deferred UAT depends on a raw shell PTY (to print fixtures) and/or the polish fixes landing first (LNK chain on iPad needs POLISH-01/02 shipped; find-bar UAT cannot pass without POLISH-03/04 shipped).
- Phase 106 (Distribution pipeline followups) is independent of product code; sequenced last so the next release tag (after v3.3 product changes ship) exercises the fixed pipeline end-to-end.
- Shell session web-share security posture: SHELL-07 makes new shell sessions opt-in for web serving (overriding the agent-session auto-enable default); SHELL-08 surfaces a one-time confirmation banner explaining arbitrary-execution risk on first toggle. Defense-in-depth on top of v3.1 capability tokens + WS Origin allowlist.
- Shell sessions use interactive (non-login) mode by default per requirements out-of-scope decision (login-shell semantics deferred until users request). Custom shell binary path picker explicitly out-of-scope; system-discovery (SHELL-04) is sufficient.

### Pending Todos

- 🔄 `/gsd-plan-phase 100` to begin Phase 100 (Shell Session Backend & Discovery)
- 🔄 Phase 105 UAT prerequisites: confirm physical iPad availability + Tailscale tailnet access before Phase 105 starts
- 🔄 Phase 106: archive `.planning/deferred/91-distribution-pipeline-followups/` into the Phase 106 directory once 91-A/B/C land
- 🔄 Phase 103 POLISH-05 investigation: read `web/vendor/xterm/addons/addon-image/lib/addon-image.js` for `iipSupport` default and IIP parser path to decide between fix vs sixel-only documentation

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260406-nqy | Dynamic dock icon visibility (show when window present) | 2026-04-06 | 5a90f5a | `.planning/quick/260406-nqy-dynamic-dock-icon-visibility-show-when-w/` |
| 260406-op4 | Tray icon matches app icon (monochrome) | 2026-04-06 | a3332df | `.planning/quick/260406-op4-tray-icon-a-matches-app-icon-a-monochrom/` |
| 260406-s0e | Fix CLI detection — app showed "No CLIs detected" despite agents installed | 2026-04-06 | 56ccc97 | `.planning/quick/260406-s0e-fix-cli-detection-app-shows-no-clis-dete/` |
| 260407-w91 | Toolbar icons match globe icon size + brightness | 2026-04-07 | 3045a0a | `.planning/quick/260407-w91-make-toolbar-icons-match-globe-icon-size/` |
| 260408-dcv | Fix GitHub Actions build + release pipeline failures | 2026-04-08 | c1511b3 | `.planning/quick/260408-dcv-fix-github-actions-build-and-release-pip/` |
| 260409-vop | Remove flashing Tailscale check modal; Settings → tab (not modal) | 2026-04-09 | 3bc0560 | `.planning/quick/260409-vop-remove-flashing-tailscale-check-modal-an/` |
| 260412-l7k | Fix local-network banner showing while Tailscale connected | 2026-04-12 | 0db6ade | `.planning/quick/260412-l7k-fix-local-network-banner-showing-when-ta/` |

### Plan Execution Metrics

(empty — v3.3 phase planning not yet started)

### Blockers/Concerns

- **Shell web-share attack surface** — Shells expose arbitrary command execution. A tailnet viewer with write capability could execute commands as the host user. Mitigated in Phase 101 with web-share-disabled-by-default override (SHELL-07) + one-time arbitrary-execution confirmation banner (SHELL-08). Acknowledge: this only reduces accidental enablement; deliberate web-share of a shell session by an authenticated user remains by design.
- **POLISH-05 iTerm2 IIP — unknown root cause** — Three hypotheses listed in `v3.2-RELEASE-BLOCKERS.md` (addon construction missing `iipSupport`, fixture syntax issue, or sixel-only-by-intent). Phase 103 must do source inspection of `addon-image.js` before deciding whether to fix vs document.
- **iPad + Tailscale UAT logistics** — Phase 105 UAT-02, UAT-04, UAT-07 require a physical iPad + a real Tailscale tailnet. If hardware/network unavailable, those three UAT items may need to defer again. Mitigation: confirm hardware before Phase 105 planning begins.
- **DIST-03 first-submission risk** — `wingetcreate new` against microsoft/winget-pkgs is a real PR to an external repo; failure modes (manifest validation, naming conflicts) only surface at PR time. Phase 106 plan should include a local `wingetcreate validate` dry-run before the upstream PR.
- **Shell discovery on edge platforms** — `/etc/shells` parsing on minimal Linux containers, missing PowerShell on stock Windows installs without `pwsh.exe`, BSD-derived shells. Phase 100 acceptance limits scope to bash/zsh/pwsh/system-default; exotic shells deferred.

## Deferred Items

Carry-over from v3.2 close (2026-05-12) — all absorbed by v3.3 phases:

| Category | Item | Absorbed by | Status |
|----------|------|-------------|--------|
| uat_gap | Phase 93: 93-iPad-UAT.md (UAT-1/2) | Phase 105 (UAT-01, UAT-02) | Pending — Phase 105 |
| uat_gap | Phase 94: 10K scrollback perf (UAT-3) | Phase 105 (UAT-03) | Pending — Phase 105 |
| uat_gap | Phase 95: iPad LNK chain (UAT-4/5) | Phase 105 (UAT-04) | Pending — Phase 102 + 105 |
| uat_gap | Phase 96: chafa fidelity + 2-client image (UAT-5/6) | Phase 105 (UAT-05, UAT-06) | Pending — Phase 105 |
| uat_gap | Phase 99: 99-iPad-UAT.md (UAT-7) | Phase 105 (UAT-07) | Pending — Phase 105 |
| polish | P-1 mailto | Phase 102 (POLISH-01) | Pending — Phase 102 |
| polish | P-2 IDN Cyrillic | Phase 102 (POLISH-02) | Pending — Phase 102 |
| polish | P-3 find bar Esc-after-toggle | Phase 103 (POLISH-03) | Pending — Phase 103 |
| polish | P-4 find bar slide-out anim | Phase 103 (POLISH-04) | Pending — Phase 103 |
| polish | P-5 iTerm2 IIP | Phase 103 (POLISH-05) | Pending — Phase 103 |
| polish | P-6 Vitest localStorage | Phase 103 (POLISH-06) | Pending — Phase 103 |
| dist | 91-A PAT credential in release.yml | Phase 106 (DIST-01) | Pending — Phase 106 |
| dist | 91-B RELEASE_TAG env var | Phase 106 (DIST-02) | Pending — Phase 106 |
| dist | 91-C wingetcreate new | Phase 106 (DIST-03) | Pending — Phase 106 |

(Quick-task ghost entries from prior milestones omitted — they remain ghosts; not v3.3 work.)

## Session Continuity

Last session: 2026-05-12 — v3.3 roadmap created
Stopped at: Roadmap definition complete; awaiting phase 100 planning
Resume file: None
Next action: `/gsd-plan-phase 100` to decompose Phase 100 (Shell Session Backend & Discovery) into plans. Optional: `/gsd-discuss-phase 100` first if any gray areas surface during plan review.

**Active Milestone:** v3.3 Shell Sessions & Polish — 7 phases (100-106), targeting Issue #44 + Issue #45 closure, absorbing v3.2 polish carry-over (6 items), v3.2 deferred UAT (9 scenarios), and v3.1 deferred Phase 91 distribution-pipeline followups (3 items).

## Operator Next Steps

- `/gsd-plan-phase 100` to start Phase 100 (Shell Session Backend & Discovery)
