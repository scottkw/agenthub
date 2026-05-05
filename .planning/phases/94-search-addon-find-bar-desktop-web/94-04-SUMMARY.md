---
phase: 94-search-addon-find-bar-desktop-web
plan: 04
subsystem: webserver-e2e + frontend-tests
tags: [phase-94, search, performance, cancellation, wave-3, chromedp, e2e, regression-gate]

# Dependency graph
requires:
  - phase: 94-01
    provides: vendored @xterm/addon-search@0.16.0 + Wave 0 RED scaffolds (TestFindBar_10kPerf, FindBar.cancel)
  - phase: 94-02
    provides: daemon.SearchConfig + Wails models.ts mirror (perf test exercises full /api path indirectly via the harness; not strictly required, but plan-declared dependency)
  - phase: 94-03
    provides: TerminalPanel handleSearchClose with clearTimeout(debounceTimerRef.current) + searchAddonRef.current?.clearDecorations() + debounceTimerRef.current = null reset (the source-inspection target)
provides:
  - 10,000-line scrollback perf budget regression gate (TestFindBar_10kPerf, build-tag e2e)
  - Cancellation contract regression gate (FindBar.cancel.test.tsx, jsdom source-inspection)
  - Reusable chromedp harness pattern: TLS httptest sidecar serving xterm vendor assets via webfs.WebFS sub-FS
affects:
  - 94-05 (web parity wave: same vendored xterm + addon-search bundles must keep loading at /assets/xterm/* — perf harness regression-gates that route too)
  - Future phases that change handleSearchClose: source-inspection assertions will fail loudly if the cancel symbols are removed

# Tech tracking
tech-stack:
  added: []  # chromedp + cdproto already present from Phase 89
  patterns:
    - "Sidecar TLS httptest server for chromedp e2e: webfs.WebFS sub-FS mount + custom test-only route avoids modifying production server.go"
    - "runtime.EvaluateParams.WithAwaitPromise pattern for chromedp Evaluate of async APIs (term.write, etc.) — local helper EvaluateOption rather than the higher-level CDP wrapper"
    - "Self-skip on 'exec: chrome not found' — chromedp tests degrade to manual UAT instead of false-failing in environments without Chromium"
    - "Source-inspection regression gate: jsdom can't exercise xterm runtime, so test asserts the production source string contains the required cancel symbols"

key-files:
  created:
    - internal/webserver/testdata/findbar_perf_fixture.txt  # 10,000 lines, 100 'needle' matches, ~860 KB
    - internal/webserver/testdata/findbar_perf_harness.html  # minimal xterm + addon-search harness page
  modified:
    - internal/webserver/findbar_perf_e2e_test.go            # RED scaffold → GREEN: real chromedp test
    - frontend/src/components/FindBar/__tests__/FindBar.cancel.test.tsx  # RED scaffold → GREEN: 5 cancel-on-close tests

key-decisions:
  - "Sidecar httptest TLS server (not the existing testServer helper) — perf test is self-contained: needs xterm vendor assets at /assets/xterm/ + a harness HTML route. Spinning up the full WebServer would add capability/CSP/origin gates that the perf harness doesn't need and would fight the chromedp navigation. The sidecar mirrors the production /assets/xterm/ mount exactly via fs.Sub(webfs.WebFS, 'vendor/xterm')."
  - "Discovered empirically: window.SearchAddon is a NAMESPACE OBJECT (window.SearchAddon.SearchAddon is the constructor), NOT a direct function. The harness HTML accommodates both shapes via `window.SearchAddon.SearchAddon || window.SearchAddon` so it stays robust if the addon's UMD shape changes upstream. This is a notable Phase 94 fact — the desktop FindBar uses the npm-imported `SearchAddon` named export so it never hit this; web parity (Plan 94-05) will need to handle the same namespace shape."
  - "Self-skip pattern from Phase 89 (browser_csp_e2e_test.go) reused verbatim: 'exec: chrome not found' triggers t.Skipf instead of t.Fatalf. Keeps the test useful in CI without false-failing on developer laptops missing headless-Chromium."
  - "Cancel test runs in jsdom (Vitest), not chromedp. jsdom can't construct a real Terminal/SearchAddon (no canvas), so the cancel runtime path was already exercised by Plan 94-03's TerminalPanel.search.test.tsx assertions on the same source. This plan adds a TARGETED source-inspection that names the cancel symbols specifically — clearTimeout(debounceTimerRef.current) AND clearDecorations() — so any future refactor that loses one without the other gets caught."
  - "Locked to plan-declared Option 2 (test-harness HTML fixture). No fallback to Option 3 (Vitest+jsdom perf path) — that would not exercise the same code path and would compromise the SC-3 perf budget guarantee. Per plan-checker revision."

patterns-established:
  - "Phase 94 perf-test harness shape: testdata/<feature>_harness.html + testdata/<feature>_fixture.<ext> + sidecar httptest server + chromedp navigation + performance.now() Evaluate. Reusable for any future addon perf budget."
  - "runtime.EvaluateParams.WithAwaitPromise EvaluateOption helper for chromedp Promise-returning Evaluate calls. Three lines of Go per test file; no external dependency."

requirements-completed: [SRC-03]

# Metrics
duration: ~25min
completed: 2026-05-05
---

# Phase 94 Plan 04: Perf Budget + Cancel-on-Close Summary

**SC-3 perf budget (10,000-line scrollback findNext < 1000ms) and SRC-03 cancel-on-close contract (clearTimeout debounce + clearDecorations on close) both encoded as automated regression gates. T-94-04 (regex DoS) fully mitigated via plain-string perf gate + cancellation contract.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-05-05T (post worktree-base-correction)
- **Tasks:** 2 (both auto, both TDD-style with RED scaffolds turned GREEN)
- **Files created:** 2 (findbar_perf_fixture.txt 10,000 lines / 860 KB, findbar_perf_harness.html ~60 lines)
- **Files modified:** 2 (findbar_perf_e2e_test.go RED→GREEN, FindBar.cancel.test.tsx RED→GREEN)
- **Tests turned GREEN:** 6 (1 chromedp e2e + 5 vitest cancel)
- **Measured perf:** findNext over 10,000-line scrollback completed in ~5 ms locally (vs 1000 ms budget — 200× headroom)

## Accomplishments

- **TestFindBar_10kPerf** — turns Wave 0 RED scaffold GREEN. Real chromedp e2e test that navigates headless Chromium to a sidecar TLS httptest server, instantiates a real xterm Terminal + SearchAddon, writes the 10k-line fixture, awaits term.write completion via runtime.awaitPromise, times searchAddon.findNext('needle') under performance.now(), and asserts the call returns within 1000 ms wall-clock.
- **findbar_perf_harness.html** — minimal page that loads xterm.js + addon-search.js via the production /assets/xterm/* paths. Bonus regression coverage on the Phase 94-01 vendor-drift gate (the harness assets must continue to serve correctly).
- **findbar_perf_fixture.txt** — 10,000 lines × ~80 chars (~860 KB) with 'needle' on every 100th line for a deterministic 100-match count. Generated via shell loop, committed verbatim (no generator script in repo).
- **FindBar.cancel.test.tsx** — turns Wave 0 RED scaffold GREEN with 5 tests:
  1. Close-button click invokes onClose synchronously.
  2. Esc keydown on a non-input element (case toggle) invokes onClose (Pitfall #3 mitigation gate — handler at container level, not just the input).
  3. TerminalPanel.tsx?raw contains `clearTimeout(debounceTimerRef.current)`.
  4. TerminalPanel.tsx?raw contains `searchAddonRef.current?.clearDecorations()`.
  5. TerminalPanel.tsx?raw contains `debounceTimerRef.current = null` (post-clear reset).
- **SRC-03 closed end-to-end.** Performance budget AND cancellation contract both have automated guards. T-94-04 has two regression gates: plain-string perf budget (test 1) and debounce/decoration cancellation (test 2-5). Pathological-regex remains an accepted limitation per RESEARCH Pitfall #5.

## Task Commits

Each task committed atomically:

1. **Task 1 — TestFindBar_10kPerf chromedp e2e + 10k-line fixture + harness HTML** — `00c2e0e` (feat)
2. **Task 2 — FindBar.cancel cancel-on-close (5 GREEN tests)** — `3f33740` (test)

_Plan 94-04 has no separate metadata commit — orchestrator owns STATE.md / ROADMAP.md after the wave completes._

## Files Created/Modified

**Created:**
- `internal/webserver/testdata/findbar_perf_fixture.txt` — 10,000-line scrollback fixture (deterministic 100 'needle' matches; ~860 KB).
- `internal/webserver/testdata/findbar_perf_harness.html` — minimal xterm + addon-search harness page; loads vendor assets via /assets/xterm/* and exposes window.term + window.searchAddon for chromedp.

**Modified:**
- `internal/webserver/findbar_perf_e2e_test.go` — replaced Wave 0 RED scaffold (`t.Skip(...)`) with real chromedp test (~150 lines): sidecar TLS httptest server, runtime.awaitPromise EvaluateOption helper, ignore-cert-errors flag, self-skip on missing Chromium, 1000 ms budget assertion citing SC-3 / T-94-04. Build-tagged `//go:build e2e` (only runs in the explicit e2e CI lane).
- `frontend/src/components/FindBar/__tests__/FindBar.cancel.test.tsx` — replaced Wave 0 RED scaffold (`expect.fail(...)`) with 5 GREEN tests covering close-button + Esc-on-non-input + 3 TerminalPanel source-inspection assertions.

## Decisions Made

1. **Sidecar TLS httptest server (not the existing testServer helper)** — perf test is self-contained: needs only xterm vendor assets at /assets/xterm/ + a harness HTML route. The full WebServer would add capability/CSP/origin gates that the perf test doesn't need and would fight chromedp navigation. The sidecar mirrors the production mount exactly via `fs.Sub(webfs.WebFS, "vendor/xterm")` so the asset paths are identical to production.
2. **Surfaced empirical UMD shape: `window.SearchAddon` is a namespace object, NOT a direct function.** The vendored bundle exposes `window.SearchAddon = { SearchAddon: <constructor> }`. The harness handles both shapes (`window.SearchAddon.SearchAddon || window.SearchAddon`) so it survives upstream UMD shape changes. This is a notable fact for Plan 94-05 (web parity): the web/terminal.js code path will need the same accommodation.
3. **Phase 89 self-skip pattern reused verbatim** — `t.Skipf` on "exec: chrome not found". Keeps the test useful in CI lanes that have headless Chromium and harmless on developer machines that don't.
4. **runtime.EvaluateParams.WithAwaitPromise EvaluateOption helper inline in the test file** — chromedp doesn't expose a public `EvalAwaitPromise` constant; rather than importing a private/internal helper, define a 3-line `awaitPromiseOpt` function in the test file. Reusable as a copy-paste idiom for any future Promise-returning Evaluate.
5. **Cancel test stays in jsdom (Vitest)** — consistent with Plan 94-03's existing test scaffold layout. The runtime cancel path was already exercised in TerminalPanel via the cleanup pyramid; this plan adds the targeted symbolic source-inspection (named symbols, not just "cleanup happens somewhere") so future refactors lose-one-without-the-other get caught.
6. **No fallback to Option 3 (Vitest perf path)** — plan-checker explicitly locked Option 2. Vitest+jsdom would measure renderer-agnostic SearchAddon work but would NOT exercise the same code path as production (no canvas, no real WebGL fallback). The chromedp path proves the budget end-to-end.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Worktree was at the wrong base commit (cfd0155 vs the orchestrator-expected 69c05f1)**
- **Found during:** Pre-Task-1 setup (`.planning/phases/94-search-addon-find-bar-desktop-web/` did not exist in the worktree; recent commits showed v3.1 milestone work, not Phase 94).
- **Issue:** The `<worktree_branch_check>` block ran but the harness apparently does not auto-execute the embedded shell; the worktree HEAD remained on cfd0155. All Phase 94 plan files lived only on main at 69c05f1.
- **Fix:** Ran `git fetch origin` then `git reset --hard 69c05f1eb523e462c648bc8b194329d0ff732391`. Worktree branch namespace (`worktree-agent-*`) was correct, so the destructive-git allow-list permitted the reset (matches the worktree-startup self-correction path).
- **Files modified:** none (HEAD update only, no source changes).
- **Verification:** `git log --oneline -3` shows 69c05f1 / 978b195 / 7a239c2 (Phase 94-03 prior commits visible).
- **Committed in:** N/A (HEAD reset side-effect).

**2. [Rule 3 - Blocking] Frontend node_modules missing in the worktree**
- **Found during:** Pre-Task-2 (Vitest invocation needed for FindBar.cancel.test.tsx).
- **Issue:** `frontend/node_modules` did not exist in the freshly-reset worktree.
- **Fix:** Ran `pnpm --filter ./frontend install` (1.9 s; all 126 packages resolved from local cache, zero downloads).
- **Files modified:** none committed.
- **Committed in:** N/A.

**3. [Rule 1 - Bug] Initial harness instantiated SearchAddon as a direct constructor and silently failed**
- **Found during:** First chromedp run — `chromedp.Run: waiting for function failed: timeout` after 15 s. Probe revealed `window.__findbarPerfHarnessReady` was `undefined` even though both `window.Terminal` and `window.SearchAddon` were defined.
- **Issue:** I assumed `window.SearchAddon` was the constructor itself (matching the typical `Terminal` shape). Empirical probe (`Object.keys(window.SearchAddon)`) showed it's actually a namespace object: `{ SearchAddon: <constructor> }`. `new window.SearchAddon()` throws TypeError silently inside the IIFE; readiness flag never set; chromedp poll timed out.
- **Fix:** Updated harness to use `window.SearchAddon.SearchAddon || window.SearchAddon` (handles both UMD shapes). Wrapped IIFE in try/catch and assigned `window.__findbarPerfHarnessError` on failure so future regressions surface a useful message.
- **Files modified:** `internal/webserver/testdata/findbar_perf_harness.html`.
- **Verification:** Test now passes in 2.91 s; findNext measured at 4.60 ms vs the 1000 ms budget.
- **Committed in:** `00c2e0e` (Task 1 commit — single-commit fix; ahead-of-time discovery).

---

**Total deviations:** 3 auto-fixed (2 environment / 1 authoring slip).
**Impact on plan:** None on scope. The first two were worktree-environment concerns that did not change source. The third was an authoring slip discovered and fixed within the same task before committing — the harness is now more robust against upstream UMD shape changes than the plan-skeleton would have produced.

## Issues Encountered

- **Pre-existing Sidebar.test.tsx 20 jsdom localStorage failures** (per parallel-execution prompt + memory note `feedback_verify_test_env_before_declaring_failure.md`) — unrelated to Phase 94, intentionally not touched.
- **Expected-RED FindBar.themeMatrix.test.tsx remains RED** — reserved for Plan 94-05 (web parity). Confirmed via `pnpm exec vitest run src/components/FindBar`: 6 of 7 test files pass; only themeMatrix fails with the planned `expect.fail('RED scaffold — Plan 94-05')` message.
- **chromedp output noise** — chromedp prints `runtime.callFrame` and `securitypolicyviolation` warnings to stderr during navigation; not failures, expected for any Phase 89/93/94 e2e test that loads JS bundles.

## Test Outcomes

- **TestFindBar_10kPerf** — PASS (2.91 s, ~5 ms findNext over 10,000 lines, 200× under budget).
- **FindBar.cancel.test.tsx** — 5/5 PASS (1.27 s).
- **`go build -tags e2e ./internal/webserver/`** — exits 0.
- **`go test ./internal/webserver/`** (default lane, no e2e tag) — PASS (1.296 s, no regressions).
- **`pnpm exec vitest run src/components/FindBar`** — 6 of 7 test files pass (29/30 tests); the 1 file failure is the expected RED themeMatrix scaffold for Plan 94-05.

## Threat Mitigation Status

| Threat ID | Disposition | Status After This Plan |
|-----------|-------------|------------------------|
| T-94-04 (DoS — regex backtracking on 10k-line scrollback) | mitigate | **DONE** — plain-string perf budget gated by TestFindBar_10kPerf at 1000 ms wall-clock; cancellation contract gated by FindBar.cancel.test.tsx (clearTimeout + clearDecorations + null reset). Pathological regex remains accepted limitation per RESEARCH Pitfall #5. |
| T-94-01 / T-94-02 / T-94-03 / T-94-05 | (inherited) | UNCHANGED — no new surface in this plan. |

## Acceptance Criteria Status

### Task 1 (TestFindBar_10kPerf + fixture + harness)

- [x] `wc -l internal/webserver/testdata/findbar_perf_fixture.txt` returns `10000`
- [x] `grep -c needle internal/webserver/testdata/findbar_perf_fixture.txt` returns `100`
- [x] First non-blank line of `findbar_perf_e2e_test.go` is `//go:build e2e`
- [x] Test contains explicit budget assertion citing 1000 ms (4 grep hits)
- [x] Test cites SC-3 / T-94-04 / 10,000 (17 grep hits — abundantly)
- [x] `go build -tags e2e ./internal/webserver/` exits 0
- [x] No `t.Skip` on the assertion path (1 hit is the chromium-unavailable self-skip, mirroring browser_csp_e2e_test.go pattern — NOT an assertion-path skip)

### Task 2 (FindBar.cancel.test.tsx)

- [x] `grep -c 'RED scaffold' FindBar.cancel.test.tsx` returns 0
- [x] Close-button + Esc assertions present (9 grep hits)
- [x] `TerminalPanel.tsx?raw` source-inspection (1 import)
- [x] `clearTimeout` AND `clearDecorations` both asserted (6 grep hits)
- [x] `Pitfall #10` cited in comments (4 hits)
- [x] `pnpm --filter ./frontend test -- "FindBar\.cancel"` exits 0 (5/5 pass)

## Next Phase Readiness

- **Plan 94-05 (web parity wave):** the chromedp harness pattern and the empirical UMD shape (`window.SearchAddon.SearchAddon`) are the primary gifts to web parity. The web `web/assets/terminal.js` will need to instantiate SearchAddon using the same `(window.SearchAddon.SearchAddon || window.SearchAddon)` shape. Plan 94-05's FindBar.themeMatrix.test.tsx is the last RED scaffold; cancel + perf are now closed and OUT of scope for 94-05.
- **Plan 99 (PUI-03 advanced disclosure):** unchanged — daemon SearchConfig already round-trips; PluginsSection just needs a `<details>` block.
- **Future addon perf budgets:** the `testdata/<feature>_fixture` + `testdata/<feature>_harness.html` + sidecar httptest pattern is now the established convention for Phase 94+ chromedp e2e perf gates.

## Self-Check: PASSED

- `internal/webserver/testdata/findbar_perf_fixture.txt` — exists ✓ (10,000 lines, 100 needles, ~860 KB)
- `internal/webserver/testdata/findbar_perf_harness.html` — exists ✓
- `internal/webserver/findbar_perf_e2e_test.go` — modified ✓ (Wave 0 RED → real chromedp test)
- `frontend/src/components/FindBar/__tests__/FindBar.cancel.test.tsx` — modified ✓ (Wave 0 RED → 5 GREEN tests)
- Commit `00c2e0e` (Task 1) — verified in git log ✓
- Commit `3f33740` (Task 2) — verified in git log ✓
- `go build -tags e2e ./internal/webserver/` — exits 0 ✓
- `go test -tags e2e ./internal/webserver/ -run TestFindBar_10kPerf` — PASS, 4.60 ms vs 1000 ms budget ✓
- `pnpm exec vitest run src/components/FindBar/__tests__/FindBar.cancel.test.tsx` — 5/5 PASS ✓
- Default-lane `go test ./internal/webserver/` — no regressions ✓
- Only RED test remaining in Phase 94: FindBar.themeMatrix.test.tsx (Plan 94-05) ✓

---
*Phase: 94-search-addon-find-bar-desktop-web*
*Plan: 04*
*Completed: 2026-05-05*
