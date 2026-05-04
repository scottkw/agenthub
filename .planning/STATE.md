---
gsd_state_version: 1.0
milestone: v3.2
milestone_name: Plugin Suite
status: executing
stopped_at: Phase 92 UI-SPEC approved
last_updated: "2026-05-04T15:36:51.777Z"
last_activity: 2026-05-04
progress:
  total_phases: 12
  completed_phases: 0
  total_plans: 3
  completed_plans: 1
  percent: 33
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-03)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 92 — Plugin Settings Foundation

## Current Position

Phase: 92 (Plugin Settings Foundation) — EXECUTING
Plan: 2 of 3
Status: Ready to execute
Last activity: 2026-05-04

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

### Pending Todos

- Phase 92 planning (`/gsd-plan-phase 92`) — Foundation phase, no addon work
- Phase 96 pre-phase research subtask: audit `frontend/node_modules/@xterm/addon-image/lib/addon-image.js` for `URL.createObjectURL` / `new Worker(` / `blob:` / `data:` script construction; document findings in phase RESEARCH.md before any wiring work
- Phase 99 cross-browser CSP e2e: extend existing Chromium-only suite to cover Safari and Firefox; new iPad Safari Tailscale UAT script
- Phase 91 (distribution pipeline follow-ups, deferred from v3.1) remains in `.planning/deferred/` for a future milestone (v3.2.x patch or v3.3) — not in v3.2 scope

### Quick Tasks Completed

(carried from v3.1)

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|

### Plan Execution Metrics

| Phase | Plan | Duration | Tasks | Files | Commits |
|-------|------|----------|-------|-------|---------|
| Phase 92 P01 | 16 min | 3 tasks | 8 files |

### Blockers/Concerns

- **Image addon CSP behavior** — unknown whether `addon-image.js` uses `URL.createObjectURL` / `blob:` / dynamic-Worker construction. v3.1 CSP has no `worker-src` (falls back to `default-src 'none'` — silent block). Resolved by mandatory pre-Phase-96 source inspection.
- **Web-links phishing surface on Tailscale-served sessions** — fresh phishing primitive (tailnet viewer trusts AgentHub URL, sees clickable URL emitted by arbitrary process, gets redirected). Mitigated in Phase 95 with v3.1-style rigor: click-confirmation, OSC 8 href display, IDN/Punycode warning, strict scheme allowlist.
- **Sixel storage bomb** — upstream `storageLimit` default 100 MB × 8 tabs = OOM. Phase 96 overrides to 16 MB.
- **WebGL software-renderer detection** — iPad Safari, GPU-blacklisted corp browsers, software-rasterized Linux see WebGL but worse-than-DOM performance. Phase 93 must detect (`gl.getParameter(RENDERER)`) and fall back proactively.
- **Settings.json migration zeroes plugin defaults** — naïve `json.Unmarshal` of v3.1 settings into v3.2 struct yields Go zero values (false/0). Phase 92 ships defaults-merge constructor + fixture migration test as non-negotiable.
- WinGet first-submission to microsoft/winget-pkgs deferred until first release is published (carried from v3.0; absorbed by Phase 91 deferred work).

## Session Continuity

Last session: 2026-05-04T15:36:48.395Z
Stopped at: Phase 92 UI-SPEC approved
Resume file: None
Next action: `/gsd-plan-phase 92` to begin planning the Plugin Settings Foundation phase.

**Active Milestone:** v3.2 Plugin Suite — 8 phases (92-99), targeting Issue #36 closure. Phase 92 not started.
