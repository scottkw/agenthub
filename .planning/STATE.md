---
gsd_state_version: 1.0
milestone: v3.3
milestone_name: Shell Sessions & Polish
status: 4 plans authored (107-01..107-04), waves assigned, awaiting execution
stopped_at: Phase 107 planning complete (4 plans authored, STATE.md updated)
last_updated: "2026-05-13T04:57:22.890Z"
last_activity: 2026-05-13 — Phase 107 planning complete
progress:
  total_phases: 1
  completed_phases: 1
  total_plans: 4
  completed_plans: 4
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-03)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 107 — shell-UX collapse + binary path picker + clean-exit handling

## Current Position

Phase: 107 — shell-ux-collapse-binary-path-picker-clean-exit-handling
Plan: —
Status: 4 plans authored (107-01..107-04), waves assigned, awaiting execution
Last activity: 2026-05-13 — Phase 107 planning complete

## Phase 107 Plan Set

| Plan | Wave | Depends on | Type | Files | Requirement |
|------|------|------------|------|-------|-------------|
| 107-01 daemon-shell-path | 0 | — | Go + TS bindings | engine/api/client/app.go + App.d.ts/App.js + tests | SHELL-11 (backend) |
| 107-02 daemon-clean-exit | 0 | — | Go | engine.go + engine_test.go | SHELL-12 (backend) |
| 107-03 frontend-shell-ux-collapse | 1 | 107-01 | React/Vitest | NewSessionModal, SettingsTab, 2 new test files | SHELL-10, SHELL-11 (UI) |
| 107-04 frontend-auto-close-tab | 1 | 107-02 | React/Vitest | App.tsx + 1 new test file | SHELL-12 (UI) |

Parallel execution: wave 0 runs 107-01 and 107-02 simultaneously (different file regions, zero overlap). Wave 1 runs 107-03 and 107-04 simultaneously after wave 0 completes (different frontend files, zero overlap).

## Performance Metrics

**Velocity:**

- v3.1 phases: 4 (Phases 87-90)
- v3.1 plans completed: 18
- v3.1 timeline: 2026-04-20 → 2026-05-03 (~13 days)
- Cumulative: 90 phases, 186 plans across 19 milestones

## Accumulated Context

### Decisions

- v3.2 scope addresses GitHub Issue #36 ("Extend xterm.js functionality with select plugins")
- Phase numbering continues from v3.1 (Phase 90 last shipped). Phase 91 is the deferred-distribution-pipeline-followups bucket from v3.1 (preserved at `.planning/deferred/91-distribution-pipeline-followups/`); v3.2 starts at Phase 92 to avoid reusing 91.
- 8-phase shape (92-99) honors the synthesized SUMMARY.md recommendation from 4 specialist researchers; ordering rationale: foundation (92) → migrate-don't-add (93) → cheapest-new (94) → security-gate (95) → CSP/perf-gate (96) → standalone (97) → optional-cuttable (98) → release-gate (99).
- Phase 92 (Foundation) ships with NO addon-loading work — establishes daemon `PluginSettings`, Wails RPC, `settings:plugins` event, Settings UI shell, migration test only.
- Phase 93 generalizes `vendor_drift_test.go` into a load-bearing CI gate enforcing `@xterm/addon-*` version parity for every addon (not just `addon-fit`); migrates webgl/unicode11/clipboard onto the new reconcile pattern AND vendors them for the web page (none vendored today).
- Phase 95 (Web-Links) is treated with v3.1-WS-Origin-allowlist rigor: scheme allowlist (`https`, `http`, `mailto` only), OSC 8 hover-href display, IDN/Punycode click confirmation, platform-aware activation (Cmd-click/Ctrl-click).
- Phase 96 (Image) starts with a mandatory pre-phase research subtask reading `addon-image.js` source for `URL.createObjectURL`/`new Worker(`/`blob:` usage; CSP amendment is conditional on findings.
- Phase 96 sets `storageLimit: 16` MB (overriding upstream 100 MB default) to prevent tab-OOM with 8+ open tabs.
- Phase 98 (Progress) is P2 / explicitly cuttable — defers to v3.3 if Phases 95 or 96 over-run.
- Phase 99 is the release gate: cross-browser CSP e2e (Chromium + Safari + Firefox), iPad Safari Tailscale UAT, settings.json migration verification.
- Phase 94 (Search) owns find-bar UI for BOTH desktop and web (user explicitly chose ambitious scope; original SUMMARY proposed deferring web UI).
- Recommendation: ship all 7 plugins ON by default except optional `addon-progress` (default OFF in v3.2, flips ON in v3.3 after field validation).
- Server-shared plugin config for buffer-interpretation plugins (Unicode 11 must match across clients to avoid scrollback divergence); per-client renderer choice (WebGL/DOM) tolerated since it doesn't affect buffer state.
- [Phase 92]: Pin wails-generated models.ts in-repo rather than regenerate per build — Project maintains hand-edited App.d.ts/App.js stubs with Call()-based convention; replacing wholesale would break vite test aliasing and lose hand-maintained inline type definitions
- [Phase 92]: PluginSettings ships as full Wails-generated class form (not bare interface) — Matches 'wails generate module' output verbatim so future regeneration is a clean diff; supports both type-only and value-construction usage in Plan 92-03
- [Phase 92 Plan 03]: Frontend imports PluginSettings via `import type { daemon } from '../wailsjs/go/models'` then `type PluginSettings = daemon.PluginSettings` — resolved Plan 92-02's documented fork in favor of the generated path (the alternative `../types/plugins` hand-written file was never created)
- [Phase 92 Plan 03]: TerminalPanel inert-prop invariant implemented via single `void pluginConfig` line outside every useEffect — keeps the prop visible to the source-inspection regex while mechanically excluding it from any addon-load consumption path until Phase 93 lifts the invariant
- [Phase ?]: Plan 94-06: animation wiring uses two-phase mount-then-RAF (entering modifier dropped on next animation frame) + parent-driven exit (TerminalPanel owns 200ms unmount timer; FindBar exiting prop applies modifier).
- [Phase 94 Plan 07]: SetSearchConfig sub-key RPC plumbing follows the existing /settings/plugins shape — new PATCH /settings/search-config + DaemonClient + App Wails facade; bindings hand-edited at wailsjs/go/main/App.{d.ts,js} (mirrors Phase 92's PLUG-03 pattern, NOT a separate wailsjs/go/daemon/SessionEngine namespace which the plan named but does not exist).
- [Phase 94 Plan 07]: seededRef one-shot useEffect pattern established (useRef(false) + early-return on already-seeded / source-null / mid-open) for seeding local UI state from an async-loaded prop without disrupting an open UI. Re-usable for any future find-bar-like component that reads from PluginSettings.
- [Phase 107 planning]: shellPath plumbing mirrors shellWebShareWarned exactly — same engine field + Get/Set methods + GET/PATCH HTTP routes + DaemonClient methods + Wails wrappers + TS bindings. Pattern is load-bearing across all settings additions; future additions should mirror it line-for-line.
- [Phase 107 planning]: resolveDefaultShellPath fallback chain ($SHELL → DiscoverShells Name=="shell" → platform hardcode) is resolved at GetShellPath()/CreateSession time, NOT at settings-load. This preserves "clear field to use system default" semantics — an empty stored value is a valid configuration meaning "use platform default", not an uninitialized state.
- [Phase 107 planning]: SHELL-12 normalization gap was at engine.go:383 (ListSessions ExitCode emission) — the natural-exit goroutine already normalized at line 334 but ListSessions read s.ExitCode() raw. Fix: insert the same `if ec == -1 { ec = 0 }` guard at the second site. Two grep hits expected post-fix (filter `^[[:space:]]*//` comments).

### Pending Todos

_All v3.2 pending items resolved 2026-05-12 — see resolution notes below:_

- ✅ Phase 92 planning — phase completed 2026-05-03 (3 plans, 6 commits)
- ✅ Phase 96 pre-phase research subtask — completed; findings in `.planning/phases/96-image-addon-csp-audit/96-RESEARCH.md`
- ✅ Phase 99 cross-browser CSP e2e — completed (99-04: playwright web-csp.spec.ts × chromium + firefox + webkit, all green); iPad Safari UAT script authored (99-iPad-UAT.md) but **deferred to v3.3** with shell-session feature
- 🔄 Phase 91 (distribution pipeline follow-ups, deferred from v3.1) remains in `.planning/deferred/91-distribution-pipeline-followups/` for v3.3+ (still applicable, carries forward)
- ⏭️ Phase 107: execute the 4-plan set via `/gsd-execute-phase 107` (waves 0 + 1 parallel-safe)

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
| Phase 107 P107-04 | 175 | 1 tasks | 2 files |

### Plan Execution Metrics

| Phase | Plan | Duration | Tasks | Files | Commits |
|-------|------|----------|-------|-------|---------|
| Phase 92 P01 | 16 min | 3 tasks | 8 files |
| Phase 92 P02 | 12 min | 2 tasks | 4 files |
| Phase 92 P03 | 10 min | 3 tasks | 6 files |
| Phase 94 P06 | 25min | 3 tasks | 9 files |
| Phase 94 P07 | 32min | 3 tasks | 9 files |

### Blockers/Concerns

- **Image addon CSP behavior** — unknown whether `addon-image.js` uses `URL.createObjectURL` / `blob:` / dynamic-Worker construction. v3.1 CSP has no `worker-src` (falls back to `default-src 'none'` — silent block). Resolved by mandatory pre-Phase-96 source inspection.
- **Web-links phishing surface on Tailscale-served sessions** — fresh phishing primitive (tailnet viewer trusts AgentHub URL, sees clickable URL emitted by arbitrary process, gets redirected). Mitigated in Phase 95 with v3.1-style rigor: click-confirmation, OSC 8 href display, IDN/Punycode warning, strict scheme allowlist.
- **Sixel storage bomb** — upstream `storageLimit` default 100 MB × 8 tabs = OOM. Phase 96 overrides to 16 MB.
- **WebGL software-renderer detection** — iPad Safari, GPU-blacklisted corp browsers, software-rasterized Linux see WebGL but worse-than-DOM performance. Phase 93 must detect (`gl.getParameter(RENDERER)`) and fall back proactively.
- **Settings.json migration zeroes plugin defaults** — naïve `json.Unmarshal` of v3.1 settings into v3.2 struct yields Go zero values (false/0). Phase 92 ships defaults-merge constructor + fixture migration test as non-negotiable.
- WinGet first-submission to microsoft/winget-pkgs deferred until first release is published (carried from v3.0; absorbed by Phase 91 deferred work).

## Deferred Items

Items acknowledged and deferred at v3.2 milestone close on 2026-05-12:

| Category | Item | Status |
|----------|------|--------|
| uat_gap | Phase 93: 93-iPad-UAT.md | deferred (0 pending) |
| uat_gap | Phase 95: 95-DESKTOP-UAT.md | deferred (0 pending) |
| uat_gap | Phase 95: 95-HUMAN-UAT.md | deferred (5 pending — iPad/Tailscale UAT) |
| uat_gap | Phase 95: 95-WEB-UAT.md | deferred (0 pending) |
| uat_gap | Phase 96: 96-HUMAN-UAT.md | deferred (0 pending) |
| uat_gap | Phase 97: 97-HUMAN-UAT.md | approved (0 pending) |
| uat_gap | Phase 98: 98-HUMAN-UAT.md | approved (0 pending) |
| uat_gap | Phase 99: 99-iPad-UAT.md | deferred (0 pending — Test 11 blocked on shell-session feature) |
| quick_task | 260406-nqy-dynamic-dock-icon-visibility-show-when-w | missing (artifact ghost from Apr 2026) |
| quick_task | 260406-op4-tray-icon-a-matches-app-icon-a-monochrom | missing (artifact ghost from Apr 2026) |
| quick_task | 260406-s0e-fix-cli-detection-app-shows-no-clis-dete | missing (artifact ghost from Apr 2026) |
| quick_task | 260407-w91-make-toolbar-icons-match-globe-icon-size | missing (artifact ghost from Apr 2026) |
| quick_task | 260408-dcv-fix-github-actions-build-and-release-pip | missing (artifact ghost from Apr 2026) |
| quick_task | 260409-vop-remove-flashing-tailscale-check-modal-an | missing (artifact ghost from Apr 2026) |
| quick_task | 260412-l7k-fix-local-network-banner-showing-when-ta | missing (artifact ghost from Apr 2026) |

UAT items align with the v3.2-MILESTONE-AUDIT deferred-to-v3.3 list (blocked on shell-session feature). Quick-task ghosts are pre-existing hygiene debt from prior milestones — slugs persist in the index but artifact files were not retained.

## Session Continuity

Last session: 2026-05-13T04:57:22.886Z
Stopped at: Phase 107 planning complete (4 plans authored, STATE.md updated)
Resume file: None
Next action: `/gsd-execute-phase 107` to execute the 4-plan set. Wave 0 runs 107-01 + 107-02 in parallel; wave 1 runs 107-03 + 107-04 in parallel after wave 0 completes.

**Active Milestone:** v3.3 Shell Sessions & Polish — 8 phases (100-107). Phases 100-106 shipped. Phase 107 planned, awaiting execute.

## Operator Next Steps

- `/gsd-execute-phase 107` to ship SHELL-10/11/12 (4 plans, 2 parallel waves).
- After Phase 107 lands, run `/gsd-verify-work 107` for the manual UAT smoke (shell session exit cleanly → tab disappears; Settings → Paths shell binary field round-trips).
- Then close v3.3 milestone.
