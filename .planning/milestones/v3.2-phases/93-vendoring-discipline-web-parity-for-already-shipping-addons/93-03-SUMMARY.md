---
phase: 93-vendoring-discipline-web-parity-for-already-shipping-addons
plan: 03
subsystem: ui
tags: [react, xterm, webgl, addon-clipboard, addon-unicode11, hot-swap, banner-stack, vitest]

# Dependency graph
requires:
  - phase: 92-plugin-settings-foundation
    provides: pluginConfig prop drilling pipeline (daemon → Wails → React → TerminalPanel) with PLUG-03 inert-prop contract; PluginsSection 8-toggle UI with three-state Save; settings:plugins event subscription on App.tsx
provides:
  - "TerminalPanel two-useEffect architecture: mount [sessionId] for next-session-only addons (Unicode 11), hot-swap [pluginConfig?.webgl, pluginConfig?.clipboard, onWebGLContextLost, sessionId] for live attach/detach (WebGL + Clipboard)"
  - "Phase 92 inert-prop invariant lifted: TerminalPanel.tsx no longer contains `void pluginConfig`; pluginConfig is genuinely consumed inside the hot-swap useEffect"
  - "WebGLRecoveryBanner component (parallel to UpdateBanner) — two reason variants ('context-loss' auto-dismiss 8000ms, 'software-rasterized' persistent), verbatim copy from 93-UI-SPEC, role=status + aria-live=polite, dismiss button aria-label='Dismiss notification'"
  - "isSoftwareWebGL() probe — boolean-only return; renderer string never leaves the function (T-93-WGL-03 information-disclosure mitigation)"
  - "App.tsx WebGL recovery state + render-condition extension: webglContextLost / webglSoftwareDetected / webglBannerDismissed; onWebGLContextLost callback wired stable via useCallback([])"
  - "PluginsSection italic caption affordance: 'Applies to new sessions you create.' under Unicode 11 row using new .settings-panel__description--italic CSS modifier"
affects:
  - 93-04-PLAN  # web/terminal.html parity — same hot-swap vocabulary, will reuse WebGLRecoveryBanner copy verbatim
  - 93-05-PLAN  # web push channel for plugin-config — runtime contract these toggles enforce on web
  - 99-PUI-02   # post-Save BannerStack toast for Unicode 11 (deferred from this phase by UI-SPEC)

# Tech tracking
tech-stack:
  added:
    - "@xterm/addon-clipboard imported in TerminalPanel.tsx (existing dependency, newly used)"
    - "frontend/src/lib/webglProbe.ts standalone module (new file)"
  patterns:
    - "Two-useEffect TerminalPanel architecture: mount-once [sessionId] for next-session addons; hot-swap [specific pluginConfig keys] for live addons. Pitfall #1: never put the whole pluginConfig object in the dep array — always reference specific keys."
    - "Stable useCallback identity (empty dep array) for callbacks threaded into child useEffect dep arrays — prevents re-runs on unrelated parent re-renders."
    - "One-shot per-session toast pattern: independent state flags for trigger condition and dismissal, gated together (`(triggerA || triggerB) && !dismissed`). Implicit-dismiss path: setting trigger flag again while dismissed=true keeps toast hidden."
    - "Information-disclosure mitigation for software-renderer probes: return boolean only, internal regex literals never reach user-visible messages."

key-files:
  created:
    - frontend/src/lib/webglProbe.ts
    - frontend/src/lib/__tests__/webglProbe.test.ts
    - frontend/src/components/WebGLRecoveryBanner.tsx
    - frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx
    - frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx
  modified:
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/PluginsSection.tsx
    - frontend/src/components/__tests__/PluginsSection.test.tsx
    - frontend/src/App.tsx
    - frontend/src/__tests__/App.plugin-event.test.tsx
    - frontend/src/style.css

key-decisions:
  - "Unicode 11 was kept out of the hot-swap dep array intentionally — UI-SPEC mandates next-session-only semantics; the italic caption 'Applies to new sessions you create.' is the user-visible affordance. Toggling Unicode 11 on an open terminal would re-flow the buffer."
  - "WebGL initial load happens inside the hot-swap useEffect (first run on mount), not the mount useEffect. This unifies initial-load and live-toggle code paths — single source of truth for WebGL lifecycle."
  - "Hot-swap useEffect dep array uses specific pluginConfig keys (`pluginConfig?.webgl`, `pluginConfig?.clipboard`) rather than the whole pluginConfig object — Pitfall #1 from 93-RESEARCH. Otherwise every save would re-run even for unrelated toggles like Unicode 11."
  - "Software-rasterizer detection fires the SAME callback as context-loss but with reason='software-rasterized', distinguished at the App.tsx level. This keeps TerminalPanel's surface area minimal (one callback prop, two literal-union reasons)."
  - "WebGLRecoveryBanner reuses the existing #7aa2f7 accent (info tone), NOT #f59e0b (warn tone). Per UI-SPEC: by the time the user sees the toast, recovery has already happened — info, not warning."

patterns-established:
  - "Two-useEffect TerminalPanel: mount-once vs. dep-tracked hot-swap. Future addons follow this split — next-session semantics in mount, live semantics in hot-swap."
  - "Stable callback identity for child-useEffect deps: useCallback with empty deps; setters from useState are stable by React's contract."
  - "Verbatim copy locked in UI-SPEC: production code, tests, and SUMMARY.md all reference the same canonical strings. Tests use `toMatch(/literal copy/)` to pin invariants against drift."
  - "Information-disclosure mitigation in renderer probes: regex matches stay inside the probe; user-visible messages are pre-canned per branch."
  - "Phase 92 → Phase 93 invariant-lift contract: a Phase-N inert-prop assertion in tests becomes a flipped assertion in Phase-N+1 (consumesInEffect=false → consumesInEffect=true), with a corresponding `not.toMatch(/^\\s*void pluginConfig\\s*$/m)` removal assertion. This pattern allows future plans to make 'is consumed' vs. 'is inert' a tested invariant."

requirements-completed:
  - WGL-01
  - WGL-02
  - WGL-03
  - U11-01
  - CLIP-02

# Metrics
duration: 13min
completed: 2026-05-04
---

# Phase 93 Plan 03: Lift Phase 92 inert-prop invariant — TerminalPanel hot-swap useEffects, WebGLRecoveryBanner, isSoftwareWebGL probe, italic Unicode 11 caption

**Two-useEffect TerminalPanel pattern wires WebGL/Clipboard live-swap and Unicode 11 next-session honoring; new WebGLRecoveryBanner with reason='context-loss' (8s auto-dismiss) and reason='software-rasterized' (persistent) renders in the existing .banner-stack via App.tsx state + stable useCallback handler.**

## Performance

- **Duration:** ~13 min
- **Started:** 2026-05-04T19:35:00Z
- **Completed:** 2026-05-04T19:48:02Z
- **Tasks:** 3 (RED → GREEN-1 → GREEN-2)
- **Files modified:** 6 (frontend/src/components/TerminalPanel.tsx, App.tsx, PluginsSection.tsx, style.css, plus the two existing test files App.plugin-event.test.tsx + PluginsSection.test.tsx)
- **Files created:** 5 (frontend/src/lib/webglProbe.ts, components/WebGLRecoveryBanner.tsx, plus three test files: components/__tests__/TerminalPanel.hot-swap.test.tsx, components/__tests__/WebGLRecoveryBanner.test.tsx, lib/__tests__/webglProbe.test.ts)

## Accomplishments

- **Phase 92 inert-prop invariant lifted.** `void pluginConfig` line removed from TerminalPanel.tsx; pluginConfig is now genuinely consumed inside the hot-swap useEffect. The Phase 92 test assertion `expect(consumesInEffect).toBe(false)` is flipped to `toBe(true)` and a new assertion pins the absence of `void pluginConfig`.
- **TerminalPanel two-useEffect architecture shipped.** Mount useEffect ([sessionId]) handles next-session-only Unicode 11 (gated on `pluginConfig?.unicode11 !== false`). Hot-swap useEffect ([pluginConfig?.webgl, pluginConfig?.clipboard, onWebGLContextLost, sessionId]) handles WebGL + Clipboard live attach/detach. Disposes addons cleanly on toggle-off and on session teardown.
- **WGL-02 context-loss recovery wired end-to-end.** WebglAddon.onContextLoss disposes the addon, clears the ref, and fires `onWebGLContextLost?.('context-loss')` — App.tsx renders the WebGLRecoveryBanner once per session.
- **WGL-03 software-rasterizer preemption wired.** New `isSoftwareWebGL()` probe (matches SwiftShader/llvmpipe/ANGLE-software/ANGLE-SwiftShader against `gl.RENDERER`) is checked at WebGL load time; on positive match the addon is NOT loaded and `onWebGLContextLost?.('software-rasterized')` fires. Renderer string never leaves the probe (T-93-WGL-03 information-disclosure mitigation).
- **WebGLRecoveryBanner component shipped** with both reason variants. Verbatim copy from 93-UI-SPEC § Copywriting Contract; role="status", aria-live="polite", dismiss button aria-label="Dismiss notification". Auto-dismiss timer (8000ms) for context-loss; persistent for software-rasterized.
- **U11-01 italic caption affordance.** PluginsSection renders "Applies to new sessions you create." directly under the Unicode 11 row description using the new `.settings-panel__description--italic` modifier class.
- **CLIP-01 / CLIP-02 desktop hot-swap.** ClipboardAddon attaches/detaches live based on `pluginConfig?.clipboard`. (Web read-only viewer gate is Plan 93-04 scope.)

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — write tests** — `6ee3478` (test)
2. **Task 2: GREEN — webglProbe + WebGLRecoveryBanner + style.css + italic caption** — `7429928` (feat)
3. **Task 3: GREEN — TerminalPanel hot-swap + App.tsx wiring** — `58130a9` (feat)

_TDD plan-level cycle: RED (Task 1) → GREEN (Tasks 2–3). No REFACTOR commit — production code is in its final shape on first GREEN; no cleanup pass was needed._

## Files Created/Modified

**Created:**
- `frontend/src/lib/webglProbe.ts` — `isSoftwareWebGL()` boolean probe; matches SwiftShader/llvmpipe/ANGLE-software/ANGLE-SwiftShader against `gl.RENDERER`; safe-false on any error path.
- `frontend/src/lib/__tests__/webglProbe.test.ts` — 4 source-inspection assertions (export, return type, regex content, getParameter/RENDERER usage).
- `frontend/src/components/WebGLRecoveryBanner.tsx` — Two-variant banner (context-loss auto-dismiss 8000ms; software-rasterized persistent); verbatim copy from UI-SPEC; XMarkIcon dismiss button (16px) with aria-label="Dismiss notification".
- `frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx` — 6 render-based assertions including auto-dismiss timing via `vi.useFakeTimers()`.
- `frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx` — 10 source-inspection assertions including the top-of-file scrollback-survival rationale comment block (ROADMAP SC#3) and the `term.clear()` / `term.reset()` absence assertions.

**Modified:**
- `frontend/src/components/TerminalPanel.tsx` — Lifted Phase 92 inert-prop invariant; restructured into mount + hot-swap useEffects; added webglAddonRef + clipboardAddonRef; added onWebGLContextLost prop; imports ClipboardAddon and isSoftwareWebGL.
- `frontend/src/App.tsx` — Imported WebGLRecoveryBanner; added webglContextLost / webglSoftwareDetected / webglBannerDismissed state; added stable handleWebGLContextLost useCallback; threaded callback into TerminalPanel render site; extended banner-stack render condition with WebGL conditions.
- `frontend/src/components/PluginsSection.tsx` — Extended `renderRow` with optional `caption` parameter; passed "Applies to new sessions you create." to the unicode11 row only.
- `frontend/src/components/__tests__/PluginsSection.test.tsx` — Added 3 new assertions for italic caption (verbatim copy, modifier class, ordering).
- `frontend/src/__tests__/App.plugin-event.test.tsx` — Flipped `consumesInEffect=false` → `true`; added `void pluginConfig` absence assertion; updated comment to reflect Phase 93 invariant lift.
- `frontend/src/style.css` — Appended `.settings-panel__description--italic` (one-line modifier) and parallel `.webgl-recovery-banner` block (mirroring `.update-banner` shape; same TokyoNight palette and #7aa2f7 accent border for info tone). Added `prefers-reduced-motion: reduce` media query to disable transitions.

## Decisions Made

- **Unicode 11 stays out of hot-swap dep array.** Per UI-SPEC: toggling Unicode 11 mid-session would re-flow the scrollback buffer — the italic caption is the explicit affordance.
- **WebGL initial load lives in hot-swap useEffect, not mount useEffect.** This unifies the initial-load and toggle-on code paths into one place. The hot-swap effect runs once at mount because pluginConfig?.webgl is in its dep array.
- **Hot-swap dep array uses SPECIFIC pluginConfig keys, not the whole object.** Pitfall #1 from 93-RESEARCH: `[pluginConfig]` would re-run even for unrelated toggles like Unicode 11.
- **Stable callback identity for `handleWebGLContextLost`.** `useCallback([])` keeps its identity stable across App re-renders; otherwise hot-swap useEffect would re-run unnecessarily.
- **#7aa2f7 (info accent), not #f59e0b (warn).** Per UI-SPEC: by the time user sees the toast, recovery already happened.
- **Phase 92 patterns preserved:**
  - **Optional pluginConfig prop** (Pitfall #4) — TerminalPanel.test.tsx 36 existing tests still mount the component without pluginConfig and remain green.
  - **Three-state Save button** in PluginsSection — untouched.
  - **Wails event subscription model** for `settings:plugins` — untouched on App.tsx.
- **Phase 92 invariants lifted:**
  - **Inert-prop invariant** — `void pluginConfig` line deleted from TerminalPanel.tsx; the corresponding test assertion in App.plugin-event.test.tsx flipped (`consumesInEffect=false` → `true`).

## Deviations from Plan

None — plan executed exactly as written. All three tasks landed in TDD order, all 41 task-1 test assertions passed after Task 3, the existing 36 TerminalPanel.test.tsx tests still pass (Pitfall #4 optional-prop preserved), and `pnpm exec tsc --noEmit` exits clean.

The only adjustment was a minor TypeScript import correction during Task 2: `Root` is exported from `react-dom/client`, not `react-dom`. This was caught by `tsc --noEmit` and fixed in the same Task-2 commit before commit. Not a deviation — a typo fix during normal verification.

## Issues Encountered

**Pre-existing test environment issue (out of scope):** `frontend/src/components/__tests__/Sidebar.test.tsx` has 20 failing tests with `TypeError: localStorage.clear is not a function`. Verified on the base commit `e858261` (before any Plan 93-03 changes) — same 20 tests fail with the same error. NOT a 93-03 regression. Logged to `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/deferred-items.md` for a future infra plan to add a Vitest setup file polyfilling `localStorage` on `window`.

Frontend test suite totals (before Plan 93-03): 506 passing, 20 failing (Sidebar). After Plan 93-03: 526 passing, 20 failing (same Sidebar set). Net delta: +20 passing, 0 new failures.

## User Setup Required

None — no external service configuration. All changes are pure-frontend React/CSS plus existing xterm addon dependencies already present in package.json.

## Threat Model Compliance

All four STRIDE threats from the plan's `<threat_model>` are mitigated:

- **T-93-CLIP-01 (Tampering, OSC 52)** — ClipboardAddon loaded only when `pluginConfig?.clipboard === true`; browser Clipboard API gesture/permission requirement still applies. CLIP-02 viewer read-only gate is Plan 93-04 scope.
- **T-93-WGL-02 (DoS, silent context-loss)** — onContextLoss disposes addon, clears ref, fires onWebGLContextLost callback exactly once per session (gated by `webglBannerDismissed`).
- **T-93-WGL-03 (Information Disclosure, renderer string)** — `isSoftwareWebGL()` returns boolean only; renderer string never appears in user-visible messages. UI-SPEC copy verified verbatim against probe-internal regex literals.
- **T-93-WGL-XSS (XSS, reason values)** — `reason` is a TypeScript-narrowed literal union (`'context-loss' | 'software-rasterized'`). No user input is interpolated into JSX.

No new trust boundaries introduced. No new security-relevant surface beyond what was already declared in the plan's threat model.

## Next Phase Readiness

- Plan 93-04 (web parity for terminal.html) can reuse the WebGLRecoveryBanner verbatim copy and the same `isSoftwareWebGL` probe pattern in the web-side terminal — no contract changes needed on the desktop side.
- Plan 93-05 (web push channel for plugin-config) can rely on the `pluginConfig?.webgl` / `pluginConfig?.clipboard` keys as the canonical hot-swap surface — no rename needed.
- Phase 99 PUI-02 (post-Save BannerStack toast for Unicode 11) can be implemented later without changing the Phase 93 italic-caption affordance — they are complementary, not exclusive.

---

## TDD Gate Compliance

This plan is type=execute (not type=tdd at the plan level), but each task is `tdd="true"` and follows RED → GREEN cycles internally:

- **Task 1 RED gate:** `6ee3478` (`test(93-03): RED — hot-swap, WebGLRecoveryBanner, webglProbe, italic-caption tests`) — 13 tests asserted RED before any production code existed.
- **Task 2 GREEN gate (partial):** `7429928` (`feat(93-03): GREEN — webglProbe + WebGLRecoveryBanner + italic caption`) — 23 tests turned GREEN.
- **Task 3 GREEN gate (final):** `58130a9` (`feat(93-03): GREEN — TerminalPanel hot-swap useEffect + App.tsx wiring`) — remaining tests (TerminalPanel.hot-swap + App.plugin-event flipped) turned GREEN; total 41/41 plan-defined tests pass plus 36/36 pre-existing TerminalPanel.test.tsx unaffected.

No REFACTOR commit needed — production code is in its final shape on first GREEN.

---

## Self-Check: PASSED

**Files created (all present):**
- `frontend/src/lib/webglProbe.ts` — FOUND
- `frontend/src/lib/__tests__/webglProbe.test.ts` — FOUND
- `frontend/src/components/WebGLRecoveryBanner.tsx` — FOUND
- `frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx` — FOUND
- `frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx` — FOUND
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/deferred-items.md` — FOUND

**Commits (all present in git log):**
- `6ee3478` (Task 1 RED) — FOUND
- `7429928` (Task 2 GREEN-1) — FOUND
- `58130a9` (Task 3 GREEN-2) — FOUND

**Acceptance criteria (sampled):**
- `grep -c 'void pluginConfig' frontend/src/components/TerminalPanel.tsx` → 0 (PASS)
- `grep -c 'webglAddonRef' frontend/src/components/TerminalPanel.tsx` → ≥ 2 (PASS)
- `grep -c 'WebGLRecoveryBanner' frontend/src/App.tsx` → ≥ 2 (PASS)
- `grep -c 'Applies to new sessions you create.' frontend/src/components/PluginsSection.tsx` → 1 (PASS)
- `grep -c '\.settings-panel__description--italic' frontend/src/style.css` → 1 (PASS)
- `pnpm exec tsc --noEmit` exit 0 (PASS)
- 5 task-1 test files all GREEN (PASS)
- 36 existing TerminalPanel.test.tsx tests still GREEN (PASS)

---
*Phase: 93-vendoring-discipline-web-parity-for-already-shipping-addons*
*Completed: 2026-05-04*
