---
phase: 94
slug: search-addon-find-bar-desktop-web
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-04
---

# Phase 94 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: 94-RESEARCH.md `## Validation Architecture`

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `go test` |
| **Framework (TS unit)** | vitest |
| **Framework (e2e)** | chromedp (`//go:build e2e`) |
| **Config files** | `frontend/vitest.config.ts`, `internal/web/cdp_test.go` build tag |
| **Quick run command (Go)** | `go test ./internal/daemon/... ./internal/web/... -run "Search\|Plugin" -short` |
| **Quick run command (TS)** | `pnpm --filter ./frontend test -- FindBar` |
| **Full suite command** | `go test ./... && pnpm --filter ./frontend test && go test -tags e2e ./internal/web/...` |
| **Estimated runtime** | ~45 s (quick) / ~3 min (full + e2e) |

---

## Sampling Rate

- **After every task commit:** Run scoped quick command for the touched layer (Go quick if `internal/`, TS quick if `frontend/`).
- **After every plan wave:** Run full suite for the layer plus the cross-layer integration test (`pluginSettingsProvider` round-trip).
- **Before `/gsd-verify-work`:** Full suite + e2e must be green; manual UAT checklist signed.
- **Max feedback latency:** 60 s for unit, 180 s for full e2e.

---

## Per-Task Verification Map

> Plan IDs are placeholders the planner will assign; the validation rows pin the **requirement → test type** mapping. The planner MUST attach an `<automated>` verify command (or a Wave 0 dependency) to each task that maps to a row below.

| Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-vendor | 0 | SRC-01..05 (foundation) | T-94-01 (vendor drift) | Web bundle byte-matches `frontend/node_modules` copy | drift | `go test ./internal/web/ -run TestVendorDrift` | ❌ W0 (extend min-count to 6) | ⬜ pending |
| 01-vendor | 0 | — | — | UMD global `SearchAddon.SearchAddon` resolves on web | smoke | `grep -q 'SearchAddon\.SearchAddon' web/assets/terminal.js` | ❌ W0 | ⬜ pending |
| 02-daemon | 1 | SRC-02 (persistence) | T-94-02 (settings tamper) | `SearchConfig` round-trips through `daemon.PluginSettings` JSON | unit | `go test ./internal/daemon -run TestSearchConfig` | ❌ W0 | ⬜ pending |
| 02-daemon | 1 | SRC-02 | — | Defaults merge applies on missing field load | unit | `go test ./internal/daemon -run TestPluginSettings_DefaultsMerge` | ❌ W0 | ⬜ pending |
| 02-daemon | 1 | SRC-02 | — | SSE broadcast emits `settings:plugins` with nested SearchConfig | integration | `go test ./internal/web -run TestPluginSettingsSSE_Search` | ❌ W0 | ⬜ pending |
| 03-findbar | 2 | SRC-01 (focus-cond) | T-94-03 (browser-find pre-empt) | Cmd-F prevented only when `xtermRef.contains(document.activeElement)` | unit (jsdom) | `pnpm --filter ./frontend test -- FindBar.focus` | ❌ W0 | ⬜ pending |
| 03-findbar | 2 | SRC-01 | — | Esc dismisses + restores xterm focus | unit | `pnpm --filter ./frontend test -- FindBar.dismiss` | ❌ W0 | ⬜ pending |
| 03-findbar | 2 | SRC-02 (nav + count) | — | `onDidChangeResults` updates "{i} of {n}" label | unit | `pnpm --filter ./frontend test -- FindBar.matchCount` | ❌ W0 | ⬜ pending |
| 03-findbar | 2 | SRC-02 (toggles) | — | regex / case / wholeWord toggles round-trip to daemon SearchConfig | integration | `pnpm --filter ./frontend test -- FindBar.persistence` | ❌ W0 | ⬜ pending |
| 03-findbar | 2 | SRC-04 (visual) | — | Slide-in animation 200 ms; selectionBackground used | snapshot | `pnpm --filter ./frontend test -- FindBar.visual` | ❌ W0 | ⬜ pending |
| 04-perf | 2 | SRC-03 (10k perf) | T-94-04 (regex DoS) | 10,000-line scrollback search ≤ 1 s frame budget | perf | `go test -tags e2e ./internal/web -run TestFindBar_10kPerf` | ❌ W0 | ⬜ pending |
| 04-perf | 2 | SRC-03 (cancel) | — | Closing find bar clears decorations + cancels debounce | unit | `pnpm --filter ./frontend test -- FindBar.cancel` | ❌ W0 | ⬜ pending |
| 05-web | 3 | SRC-05 (web parity) | T-94-05 (web origin) | Web terminal page exposes Cmd-F + persists via daemon | e2e | `go test -tags e2e ./internal/web -run TestFindBar_Web` | ❌ W0 | ⬜ pending |
| 05-web | 3 | SRC-04 across themes | — | Highlight contrast valid on representative TokyoNight + light theme | snapshot | `pnpm --filter ./frontend test -- FindBar.themeMatrix` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/FindBar/__tests__/FindBar.focus.test.tsx` — stubs for SRC-01
- [ ] `frontend/src/components/FindBar/__tests__/FindBar.matchCount.test.tsx` — stubs for SRC-02
- [ ] `frontend/src/components/FindBar/__tests__/FindBar.persistence.test.tsx` — stubs for SRC-02 round-trip
- [ ] `frontend/src/components/FindBar/__tests__/FindBar.cancel.test.tsx` — stubs for SRC-03 cancel
- [ ] `frontend/src/components/FindBar/__tests__/FindBar.visual.test.tsx` — snapshot stub for SRC-04
- [ ] `frontend/src/components/FindBar/__tests__/FindBar.themeMatrix.test.tsx` — snapshot stub for SRC-04
- [ ] `internal/daemon/search_config_test.go` — stubs for SearchConfig round-trip + defaults merge
- [ ] `internal/web/plugin_settings_search_sse_test.go` — stubs for SSE broadcast
- [ ] `internal/web/findbar_perf_e2e_test.go` (build tag `e2e`) — stub 10k-line perf harness
- [ ] `internal/web/findbar_web_e2e_test.go` (build tag `e2e`) — stub web parity harness
- [ ] `internal/web/vendor_drift_test.go` — bump min-count from 5 to 6 (addon-search joins the manifest)

*All scaffolds RED on Wave 0; turn GREEN as later waves implement.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Browser-find still works when terminal unfocused | SRC-01 | Tests cross-page UA behavior; jsdom can't replicate browser Cmd-F | Open dashboard, click outside terminal, press Cmd-F → browser find toolbar appears (NOT app find bar). Re-focus terminal, Cmd-F → app find bar. |
| 200 ms slide-in / slide-out feels right | SRC-04 | Subjective animation pacing | Open + close find bar 5× on dark + light theme; record perceptual judgment in HUMAN-UAT.md |
| Web parity on iPad Safari (Tailscale) | SRC-05 | Real browser + virtual keyboard interaction | Open Tailscale-served terminal on iPad; trigger find via gesture (TBD per UI-SPEC); verify navigation + count + dismiss |
| Highlight contrast across 138 themes | SRC-04 | Snapshot tests cover only representative themes | Spot-check 5 themes (default, Dracula, Solarized Light, Solarized Dark, GitHub Dark) — confirm match highlight visible |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (11 scaffolds above)
- [ ] No watch-mode flags
- [ ] Feedback latency < 60 s for unit, 180 s for e2e
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
