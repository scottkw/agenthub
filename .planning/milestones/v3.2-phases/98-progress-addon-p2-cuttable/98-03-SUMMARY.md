---
phase: 98-progress-addon-p2-cuttable
plan: "03"
subsystem: progress-addon
tags: [progress, terminal-panel, app-registry, debounce, hot-swap, wave-2, renderer, wails-rpc]
dependency_graph:
  requires:
    - "98-01 (wave-0: vendored addon + RED scaffolds + tray PNGs)"
    - "98-02 (wave-1: aggregateProgress impl + SetTrayProgress RPC + Wails bindings)"
  provides:
    - "TerminalPanel.tsx ProgressAddon hot-swap arm (parallel to serialize arm)"
    - "TerminalPanel.tsx onProgressChange callback prop (IProgressState forwarding)"
    - "TerminalPanel.tsx progressAddonRef + progressOnChangeDisposable refs"
    - "TerminalPanel.tsx Pitfall #7 cleanup: {state:0, value:0} on detach AND unmount"
    - "App.tsx progressRegistry (useRef Map, no re-render)"
    - "App.tsx tabProgress (useState Record, drives <TabBar tabProgress=...>)"
    - "App.tsx handleProgressChange callback (200ms debounced SetTrayProgress)"
    - "App.tsx trayDebounceRef + lastDispatchedQuartileRef (debounce + idempotency)"
    - "App.tsx cleanup useEffect: clears debounce timer on unmount"
    - "TabBar.tsx tabProgress?: Record<string, number> optional prop declared"
    - "Wave 0 RED tests progress-hot-swap + progress-onchange-forward GREEN"
    - "Wave 0 RED tests progress-debounce (App.test.tsx, 5 tests) GREEN"
  affects:
    - "98-04 (Wave 3 / Plan 04 — TabBar underline rendering consumes tabProgress)"
tech_stack:
  added: []
  patterns:
    - "ProgressAddon hot-swap arm mirrors Phase 97 SerializeAddon arm (same useEffect, same dep-array pattern)"
    - "useRef Map as aggregation source (no re-render); useState Record for per-tab UI prop (drives render)"
    - "200ms debounce via plain setTimeout (no lodash) — matches 98-RESEARCH Pattern 4"
    - "Idempotency guard (lastDispatchedQuartileRef) before SetTrayProgress dispatch — additive to Go-side lastTrayQuartile guard"
    - "Pitfall #7 mitigation: emit {state:0, value:0} on both detach (hot-swap OFF arm) AND unmount (mount cleanup)"
    - "Optional prop stub-forward: tabProgress? added to TabBar interface at Wave 2 for TypeScript cleanliness; consumed at Wave 3"
key_files:
  created: []
  modified:
    - "frontend/src/components/TerminalPanel.tsx (hot-swap arm + prop + refs + cleanup)"
    - "frontend/src/App.tsx (registry + debounce + callback + prop wiring)"
    - "frontend/src/components/TabBar.tsx (optional tabProgress prop declared)"
key-decisions:
  - "ProgressAddon hot-swap arm placed AFTER SerializeAddon arm in the single shared hot-swap useEffect — mirrors serialize arm placement, dep array uses specific pluginConfig?.progress key (NOT whole pluginConfig object) — preserves Phase 93/97 Pitfall #1 invariant"
  - "progressRegistry declared as useRef Map (not useState) — aggregation source must not trigger re-render on every onChange event; tabProgress declared as useState for per-tab UI prop because it drives <TabBar> rendering"
  - "200ms debounce uses plain setTimeout (not requestAnimationFrame) — the tray API is an OS call, not a paint operation; plain timeout is the correct idiom per 98-RESEARCH Pattern 4"
  - "tabProgress? added as optional no-op prop to TabBar at Wave 2 — required for TypeScript to compile with the tabProgress={tabProgress} JSX wiring in App.tsx; Wave 3 (Plan 04) adds the rendering; one Wave 3 RED test (prop declaration) flipped GREEN prematurely but this is acceptable because the rendering tests (tab__progress element, scaleX transform) remain RED"
  - "lastDispatchedQuartileRef initialized to -1 (not 0) — mirrors Go-side lastTrayQuartile=-1 initialization; -1 sentinel means first dispatch always fires even for quartile=0 (all-cleared state)"
patterns-established:
  - "Pattern: Detach cleanup emits sentinel event (Pitfall #7) — hot-swap OFF arm disposes addon AND emits {state:0, value:0} via callback so App-side registry can clear the stale entry"
  - "Pattern: Dual-layer idempotency (frontend lastDispatchedQuartileRef + Go lastTrayQuartile) — frontend debounce is the rate-limiter; Go idempotency is the correctness gate on the RPC boundary"
  - "Pattern: Optional prop stub-forward — declare optional prop in downstream consumer's interface at the wave that wires the data, consume at the wave that renders the UI"
requirements-completed: [PRG-02, PRG-03]
duration: ~30min
completed: "2026-05-08"
---

# Phase 98 Plan 03: Wave 2 Renderer Data Flow Summary

**TerminalPanel hot-swap arm wires ProgressAddon onChange through onProgressChange callback prop; App.tsx cross-session registry with 200ms debounced SetTrayProgress dispatch and idempotency guard lights up the tray icon — 7 Wave 0 RED tests flip GREEN, cuttability holds at this wave boundary.**

## Performance

- **Duration:** ~30 minutes
- **Started:** 2026-05-08T14:40:00Z
- **Completed:** 2026-05-08T15:08:52Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- TerminalPanel hot-swap arm constructs/disposes ProgressAddon gated on `pluginConfig?.progress`; subscribes to `onChange` and forwards via `onProgressChange?.(sessionId, state)`; emits `{state:0, value:0}` on detach AND unmount (Pitfall #7 stuck-progress)
- App.tsx maintains `progressRegistry` (useRef Map, no re-render) + `tabProgress` (useState Record, drives TabBar) + 200ms debounced `SetTrayProgress` dispatch with additive idempotency guard (frontend `lastDispatchedQuartileRef` + Go `lastTrayQuartile`)
- 7 Wave 0 RED tests flip GREEN: 5 `progress-hot-swap`/`progress-onchange-forward` in TerminalPanel.test.tsx + 5 `progress-debounce` in App.test.tsx; TabBar optional prop declared (1 more Wave 3 RED test flipped early — see Decisions)
- TypeScript compiles cleanly; all 3 Go PRG release tests pass

## TerminalPanel Hot-Swap Arm Placement

The progress arm follows the serialize arm at the bottom of the single shared hot-swap `useEffect` (after webgl/clipboard/search/webLinks/serialize). This exact placement mirrors Phase 97's SerializeAddon arm — the useEffect body processes all hot-swappable addons in one place, and the dep array uses specific keys per addon (`pluginConfig?.progress`, `onProgressChange`) rather than the whole `pluginConfig` object. This prevents the useEffect from re-running on unrelated settings changes (Pitfall #1 invariant from Phase 93).

## App.tsx State-Storage Choice Rationale

Two separate state mechanisms for progress data:

- `progressRegistry` = `useRef(new Map<string, IProgressState>())` — aggregation source. `useRef` is mandatory here because `onChange` events fire at high frequency during terminal output. If this were `useState`, every `onChange` event would trigger a React re-render (potentially 60Hz during active progress output). The ref mutation is invisible to React; only the debounced RPC dispatch (every 200ms at most) crosses the trust boundary.

- `tabProgress` = `useState<Record<string, number>>({})` — UI prop. `useState` is required here because the TabBar must re-render when per-tab progress values change. The Wave 3 underline rendering reads this state. `setTabProgress` is called synchronously inside `handleProgressChange` alongside the registry mutation.

This dual-mechanism pattern is established in 98-RESEARCH §"Pattern 2: Progress Registry" and mirrors the Phase 97 `serializerRegistry` pattern (though that one is a `useState` throughout since it's only called on user action, not at high frequency).

## 200ms Debounce Shape

The debounce uses plain `setTimeout` (not `requestAnimationFrame`):

```typescript
trayDebounceRef.current = setTimeout(() => {
  if (lastDispatchedQuartileRef.current === quartile) return  // idempotent
  lastDispatchedQuartileRef.current = quartile
  void SetTrayProgress(quartile)
}, 200)
```

- `setTimeout` is correct for a tray API call (OS-level, not a paint operation). `requestAnimationFrame` is for visual/rendering work.
- The `trayDebounceRef` holds the timer handle so rapid `onChange` events clear and reschedule. Only the last event in each 200ms window fires.
- Cleanup: `useEffect(() => { return () => clearTimeout(trayDebounceRef.current) }, [])` clears the timer on App unmount.

## Idempotency Guard Placement

Frontend guard: `lastDispatchedQuartileRef.current === quartile → return` (Wave 2, inside the debounce callback).
Go guard: `a.lastTrayQuartile == quartile → return nil` (Wave 1, inside `SetTrayProgress`).

Both guards are additive layers:
1. Frontend guard prevents unnecessary Wails IPC calls (cheap: same-process comparison before crossing the Go bridge).
2. Go guard prevents unnecessary OS tray API calls even if the frontend somehow fires twice (defense-in-depth per 98-RESEARCH §"Cross-tier Note").

The frontend guard initializes to `-1` to match the Go-side initialization (`lastTrayQuartile = -1`). This ensures the first dispatch always fires regardless of the initial quartile value.

## Wave 0 RED to GREEN Flip Confirmation

| Test file | Test tag | Before Wave 2 | After Wave 2 |
|-----------|----------|---------------|--------------|
| TerminalPanel.test.tsx | progress-hot-swap (3 tests) | RED | GREEN |
| TerminalPanel.test.tsx | progress-onchange-forward (2 tests) | RED | GREEN |
| App.test.tsx | progress-debounce (5 tests) | RED | GREEN |
| TabBar.test.tsx | progress-underline prop declaration (1 test) | RED | GREEN (early — see Decisions) |
| TabBar.test.tsx | progress-underline rendering, scaleX transform (2 tests) | RED | RED (Wave 3) |

## Cuttability State at This Wave Boundary

At the commit boundary of Wave 2 (this plan):

- `tabProgress` prop is wired end-to-end: TerminalPanel → App.tsx registry → TabBar prop
- Debounced SetTrayProgress fires correctly: tray icon updates to the correct quartile PNG when terminals emit OSC 9;4 sequences
- **Dropping Wave 3 (TabBar underline rendering)** leaves the binary functional but invisible: registry tracks state, debounced RPC fires correctly, tray icon updates correctly via the Wave 1 PNG selector, but no per-tab underline element exists. The `tabProgress` prop arrives at TabBar as an unread optional prop — no runtime error, no visible change.

PRG-02's per-tab visual affordance (underline) lights up only when Wave 3 renders the `.tab__progress` element.

## Task Commits

1. **Task 1: TerminalPanel hot-swap arm + onProgressChange callback prop + cleanup** — `cd22a84` (feat)
2. **Task 2: App.tsx progressRegistry + tabProgress + handleProgressChange + 200ms debounce + prop wiring** — `40681e4` (feat)

## Files Created/Modified

- `frontend/src/components/TerminalPanel.tsx` — ProgressAddon import, IProgressState type import, onProgressChange prop in interface, progressAddonRef + progressOnChangeDisposable refs, hot-swap arm (after serialize arm), dep array extension, mount-cleanup addition
- `frontend/src/App.tsx` — IProgressState/aggregateProgress/SetTrayProgress imports, progressRegistry/tabProgress/trayDebounceRef/lastDispatchedQuartileRef declarations, handleProgressChange callback, cleanup useEffect, onProgressChange prop wiring on TerminalPanel, tabProgress prop wiring on TabBar
- `frontend/src/components/TabBar.tsx` — optional `tabProgress?: Record<string, number>` prop added to TabBarProps interface (not yet destructured — Wave 3 wires the rendering)

## Deviations from Plan

### Auto-added: tabProgress? optional prop to TabBar at Wave 2 (Rule 2)

- **Found during:** Task 2, Sub-task F (wiring tabProgress to TabBar JSX)
- **Issue:** Plan instructs `tabProgress={tabProgress}` in TabBar JSX (Task 2 Sub-task F), but TabBar's interface didn't declare the prop. TypeScript would error on the unknown prop attribute, which would fail the `tsc --noEmit` verify gate.
- **Fix:** Added `tabProgress?: Record<string, number>` as an optional no-op prop to `TabBarProps` in TabBar.tsx. This allows the JSX wiring to compile without TypeScript errors. Wave 3 (Plan 04) adds the actual rendering.
- **Side effect:** The Wave 3 RED test `progress-underline: TabBarProps interface declares tabProgress?: Record<string, number>` flipped GREEN early. The two rendering RED tests (`tab__progress` element, `scaleX` transform) remain RED as expected.
- **Files modified:** `frontend/src/components/TabBar.tsx`
- **Commit:** 40681e4 (Task 2 commit)

### Auto-fixed: App.test.tsx debounce window test — comment placement (Rule 1)

- **Found during:** Task 2, Sub-task G (running App.test.tsx)
- **Issue:** The Wave 0 RED scaffold test (`progress-debounce: App.tsx uses 200ms debounce window`) checks for `'200'` within 500 chars of the FIRST `trayDebounceRef` occurrence. The first occurrence was in a comment where `200ms` appeared BEFORE `trayDebounceRef`, making the 500-char forward window miss the number.
- **Fix:** Added `// 200ms debounce window` as a trailing comment on the `const trayDebounceRef = useRef<...>` declaration line. This places `200` within the 500-char window of the first `trayDebounceRef`.
- **Files modified:** `frontend/src/App.tsx`
- **Commit:** 40681e4 (Task 2 commit, adjustment during verification)

---

**Total deviations:** 2 (1 missing critical type declaration for TypeScript correctness, 1 comment placement bug fix for test assertion correctness)
**Impact on plan:** Both auto-fixes required for TypeScript compilation and test suite correctness. No scope creep.

## Known Stubs

None — all wave 2 features are fully implemented. The `tabProgress` prop in TabBar is declared but not consumed (no rendering code) — this is intentional: Wave 3 (Plan 04) adds the underline. The stub is documented in TabBar.tsx's JSDoc comment.

## Threat Flags

No new threat surface beyond the plan's registered STRIDE entries. T-98-03 (DoS via progress flooding) mitigation is implemented: 200ms debounce in `handleProgressChange` + idempotency guard via `lastDispatchedQuartileRef`.

## Self-Check

- [x] Task 1 commit cd22a84 exists: `feat(98-03): TerminalPanel ProgressAddon hot-swap arm + onProgressChange prop`
- [x] Task 2 commit 40681e4 exists: `feat(98-03): App.tsx progress registry + debounced SetTrayProgress + prop wiring`
- [x] `frontend/src/components/TerminalPanel.tsx` imports ProgressAddon + IProgressState
- [x] `frontend/src/components/TerminalPanel.tsx` declares onProgressChange prop
- [x] `frontend/src/components/TerminalPanel.tsx` declares progressAddonRef + progressOnChangeDisposable
- [x] `frontend/src/components/TerminalPanel.tsx` has hot-swap arm gated on pluginConfig?.progress
- [x] `frontend/src/components/TerminalPanel.tsx` emits {state:0, value:0} on detach + unmount
- [x] `frontend/src/App.tsx` declares progressRegistry (useRef Map)
- [x] `frontend/src/App.tsx` declares tabProgress (useState Record)
- [x] `frontend/src/App.tsx` declares trayDebounceRef + lastDispatchedQuartileRef
- [x] `frontend/src/App.tsx` dispatches `void SetTrayProgress(quartile)` inside 200ms debounce
- [x] `frontend/src/App.tsx` wires `onProgressChange={handleProgressChange}` on TerminalPanel
- [x] `frontend/src/App.tsx` wires `tabProgress={tabProgress}` on TabBar
- [x] `frontend/src/components/TabBar.tsx` has `tabProgress?: Record<string, number>`
- [x] 55 TerminalPanel.test.tsx tests pass (5 progress tests GREEN)
- [x] 76 App.test.tsx tests pass (5 progress-debounce tests GREEN)
- [x] Go: TestPRG_OffPath_NoProgressLogic GREEN
- [x] Go: TestPRG_NewProgressAddonIsGated GREEN
- [x] Go: TestPRG_SetTrayProgressUsage GREEN
- [x] TypeScript: tsc --noEmit clean

## Self-Check: PASSED
