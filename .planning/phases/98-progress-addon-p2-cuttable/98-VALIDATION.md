---
phase: 98
slug: progress-addon-p2-cuttable
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-08
---

# Phase 98 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `98-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Frontend framework** | vitest (existing — Phase 92+) |
| **Backend framework** | go test (existing — `internal/...`, `app.go`) |
| **E2E framework** | Playwright (existing — Phase 95/96/97 precedent) |
| **Config files** | `frontend/vitest.config.ts`, `frontend/playwright.config.ts`, repo-root `go.mod` |
| **Quick run command** | `cd frontend && pnpm vitest run --reporter=basic && cd .. && go test ./...` |
| **Full suite command** | `cd frontend && pnpm test && cd .. && go test ./... && cd frontend && pnpm playwright test` |
| **Estimated runtime** | quick ~30 s · full ~3–4 min |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm vitest run --reporter=basic` + `go test ./...` (skip e2e for speed)
- **After every plan wave:** Quick run + `pnpm playwright test` for the wave's covered features
- **Before `/gsd-verify-work`:** Full suite green + `wails build -tags wailsassets` succeeds + manual UAT smoke (toggle ON, OSC 9;4 fixture → underline + tray glyph)
- **Cuttability gate:** Verify Progress=OFF (default) produces a binary behaviorally identical to Phase-97-state — addon not loaded, no onChange subscription, no SetTrayProgress, no `.tab__progress` rendered.
- **Max feedback latency:** ~30 s

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 98-01-01 | 01 | 0 | PRG-01 | V14 / CSP unchanged | Toggle persists default OFF; flips via Settings save | unit (frontend) | `pnpm vitest run components/__tests__/PluginsSection.test.tsx -t progress` | ❌ W0 | ⬜ pending |
| 98-01-02 | 01 | 0 | PRG-01 | — | Italic v3.3-flip caption rendered under Progress toggle | unit (frontend) | `pnpm vitest run components/__tests__/PluginsSection.test.tsx -t v3.3-flip` | ❌ W0 | ⬜ pending |
| 98-01-03 | 01 | 0 | PRG-01 | — | OFF path leaves zero progress addon construction (regression) | integration (Go grep + behavior) | `go test ./internal/release -run TestPRG_OffPath_NoProgressLogic` | ❌ W0 | ⬜ pending |
| 98-01-04 | 01 | 0 | PRG-02 | V5 / clamp [0,100] | aggregateProgress helper buckets correctly (empty/single/multi/boundary) | unit (frontend) | `pnpm vitest run lib/__tests__/aggregateProgress.test.ts` | ❌ W0 | ⬜ pending |
| 98-01-05 | 01 | 0 | PRG-03 | — | 200ms debounce drops within-window bursts | unit (frontend) | `pnpm vitest run lib/__tests__/progressDebounce.test.ts` | ❌ W0 | ⬜ pending |
| 98-02-01 | 02 | 1 | PRG-02 | — | ProgressAddon attaches/detaches on pluginConfig.progress flip | unit (frontend) | `pnpm vitest run components/__tests__/TerminalPanel.test.tsx -t progress-hot-swap` | ❌ W0 | ⬜ pending |
| 98-02-02 | 02 | 1 | PRG-02 | — | onChange events forwarded via onProgressChange callback prop | unit (frontend) | `pnpm vitest run components/__tests__/TerminalPanel.test.tsx -t progress-onchange-forward` | ❌ W0 | ⬜ pending |
| 98-02-03 | 02 | 1 | PRG-02 | — | TabBar renders underline width based on tabProgress; uses transform (not width) for animation | unit (frontend) | `pnpm vitest run components/__tests__/TabBar.test.tsx -t "progress-(underline\|transform)"` | ❌ W0 | ⬜ pending |
| 98-03-01 | 03 | 2 | PRG-03 | V5 / quartile bounds | SetTrayProgress idempotency (no-op on identical quartile) | unit (backend) | `go test . -run TestApp_SetTrayProgress_Idempotent` | ❌ W1 | ⬜ pending |
| 98-03-02 | 03 | 2 | PRG-03 | V5 | SetTrayProgress quartile bounds check | unit (backend) | `go test . -run TestApp_SetTrayProgress_BoundsCheck` | ❌ W1 | ⬜ pending |
| 98-03-03 | 03 | 2 | PRG-03 | — | SetTrayProgress error precedence (disconnected → error icon, ignore quartile) | unit (backend) | `go test . -run TestApp_SetTrayProgress_ErrorPrecedence` | ❌ W1 | ⬜ pending |
| 98-04-01 | 04 | 3 | PRG-02 | — | OSC 9;4 sequence drives addon → underline (e2e desktop) | e2e | `pnpm playwright test tests-e2e/progress.spec.ts` | ❌ W4 | ⬜ pending |
| 98-04-02 | 04 | 3 | PRG-02 | — | Web-served terminal page renders progress affordance from OSC 9;4 | e2e | `pnpm playwright test tests-e2e/progress-web.spec.ts` | ❌ W4 | ⬜ pending |
| 98-05-01 | 05 | 3 | PRG-03 | — | Manual UAT: 3 terminals, progress in 2, observe tray glyph quartile transitions without flicker | manual | (manual UAT runbook) | ❌ W4 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*File-exists key: ✅ already exists · ❌ W0/W1/W4 = wave that creates it*

---

## Wave 0 Requirements

- [ ] `frontend/src/lib/aggregateProgress.ts` — pure helper for cross-session mean → quartile bucketing
- [ ] `frontend/src/lib/__tests__/aggregateProgress.test.ts` — vitest cases (empty / single / multi / all-state-0 / boundary)
- [ ] `frontend/src/lib/progressDebounce.ts` — 200ms debounce primitive (or reuse if existing)
- [ ] `frontend/src/lib/__tests__/progressDebounce.test.ts`
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — extend with `progress-hot-swap` + `progress-onchange-forward`
- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — extend with `progress-underline` + `progress-transform`
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` — extend with `progress` + `v3.3-flip`
- [ ] `internal/release/no_progress_when_off_test.go` — negative regression test (mirror Phase 97 SER-03 pattern)
- [ ] `frontend/tests-e2e/progress.spec.ts` — Playwright OSC 9;4 → underline e2e (Wave 4 / cuttable last)
- [ ] `tests/fixtures/osc94-progress-fixture.sh` — 4-line shell script for manual + e2e
- [ ] `assets/tray_icon_progress_{25,50,75,100}.png` — 4 × 18×18 quartile glyph PNGs (designer-supplied or `build/gen_progress_icons.go` generated)
- [ ] `pnpm install @xterm/addon-progress@0.2.0` (lockfile update)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tray-glyph aggregate quartile transitions reflect cross-session progress without flicker | PRG-03 | macOS/Windows/Linux tray rendering behavior is OS-specific and not exercisable in headless CI | (1) Toggle Progress=ON in Settings; (2) Open 3 terminals; (3) In two of them, run `bash tests/fixtures/osc94-progress-fixture.sh`; (4) Observe tray icon transitions through 25→50→75→100% quartile glyphs; (5) Verify no flicker / no rapid icon swap thrashing during the burst phase. |
| Default-OFF cuttability — toggling OFF mid-session unloads the addon cleanly | PRG-01, PRG-02, PRG-03 | Hot-swap unload behavior under live usage is not e2e-deterministic | (1) Enable Progress; (2) Open a terminal emitting OSC 9;4; (3) Toggle Progress=OFF; (4) Verify `.tab__progress` element disappears, tray reverts to base icon, no console errors. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30 s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
