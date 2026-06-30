---
phase: 157-terminal-screen-share-semantics-issue-109
plan: "04"
subsystem: desktop-viewer
tags: [typescript, react, xterm, relay, resize, scale, view-04, view-05]

requires:
  - phase: 157-01
    provides: "Hub.broadcastResize: MsgResize 0x02 fan-out on every host resize"
  - phase: 157-02
    provides: "Server join-push: MsgResize 0x02 before scrollback on both relay and web paths"
  - phase: 157-03
    provides: "Web guest viewer 0x02 dispatch + recomputeScale + CSS scale-to-fit (reference implementation)"

provides:
  - "RelayClientCallbacks.onResize?: (cols,rows) — optional callback; 0x02 dispatch wired (was dropped); host callers unaffected"
  - "computeGuestScale(cw,ch,gW,gH): number — pure scale helper capped at 1.0 with zero-grid guard"
  - "TerminalPanel isGuest = remote || !!wsURL gate: guest honors onResize+recomputeScale, host unchanged"
  - "Desktop CSS: .xterm { transform-origin: top left } for scale-to-fit guest panels (VIEW-05)"

affects:
  - 157-05 (TESTING.md — Category P manual UAT for visual garble + scale; traceability rows added)

tech-stack:
  added: []
  patterns:
    - "onResize dispatch: 0x02 case in RelayClient.onmessage → callbacks.onResize?.() (was dropped)"
    - "isGuestRef: useRef updated every render so effects read current value without dep-array churn"
    - "recomputeScale: useCallback([]) captures termRef/containerRef for stable cross-effect sharing"
    - "Cell-metric scale: gridW = term.cols * cellW from _renderService.dimensions.css.cell (Pitfall 6 guard)"
    - "Pitfall 5 order: term.resize(cols,rows) BEFORE recomputeScale — scale reads new term.cols/rows"
    - "Clamp to >= 1 via Math.max(1, cols/rows): xterm rejects resize(0,0) (T-157-03 mitigation)"
    - "isActive guest branch: single rAF+recomputeScale; ResizeObserver → recomputeScale (not fitTerminal)"
    - "isActive host branch: unchanged (rAF tryFit loop + ResizeObserver → fitTerminal)"

key-files:
  created:
    - frontend/src/lib/terminalScale.ts
    - frontend/src/lib/terminalScale.test.ts
    - frontend/src/components/__tests__/TerminalPanel.scale.test.tsx
  modified:
    - frontend/src/lib/relayClient.ts
    - frontend/src/lib/relayClient.test.ts
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/style.css
    - TESTING.md

key-decisions:
  - "onResize optional (?) for backward compat: TerminalPanel host path passes no onResize; existing callers unaffected"
  - "isGuestRef pattern: ref updated at render time (vs useCallback dep-array) to avoid triggering extra effect runs"
  - "recomputeScale as useCallback([]): reads only from refs (stable); called from both mount effect and isActive effect"
  - "Pitfall 5: term.resize(cols,rows) called BEFORE recomputeScale so scale math reads new term.cols/rows"
  - "isActive guest branch skips rAF tryFit loop entirely: server 0x02 (not fitTerminal) sizes the grid for guests"

requirements-completed: [VIEW-04, VIEW-05]

coverage:
  - id: D1
    description: "RelayClient: 0x02 dispatch invokes onResize?(cols,rows); optional for host compat"
    requirement: VIEW-04
    verification:
      - kind: automated
        ref: "frontend/src/lib/relayClient.test.ts: Phase 157 — 0x02 MsgResize dispatch (6 tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "computeGuestScale: min-axis, cap at 1.0, zero-grid guard"
    requirement: VIEW-05
    verification:
      - kind: automated
        ref: "frontend/src/lib/terminalScale.test.ts: computeGuestScale — VIEW-05 scale math (10 tests)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TerminalPanel guest path: honors onResize → term.resize + CSS scale; never calls sendResize"
    requirement: VIEW-04
    verification:
      - kind: automated
        ref: "frontend/src/components/__tests__/TerminalPanel.scale.test.tsx: guest path behavioral (8 tests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "TerminalPanel host path: sendResize preserved, no transform applied"
    requirement: VIEW-05
    verification:
      - kind: automated
        ref: "frontend/src/components/__tests__/TerminalPanel.scale.test.tsx: host path invariance (3 tests)"
        status: pass
    human_judgment: false
  - id: D5
    description: "tsc --noEmit clean: no TypeScript errors introduced"
    requirement: VIEW-04
    verification:
      - kind: automated
        ref: "cd frontend && npx tsc --noEmit -p tsconfig.json"
        status: pass
    human_judgment: false

duration: ~9 minutes
completed: 2026-06-27
status: complete
---

# Phase 157 Plan 04: Desktop Guest Viewer — isGuest Gate + Scale Parity Summary

**Desktop guest terminal (remote || wsURL) now honors server-pushed MsgResize 0x02 via RelayClient onResize callback, applies CSS scale-to-fit via computeGuestScale, and never drives PTY resize — achieving cross-surface parity with the web guest (VIEW-04/05)**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-06-27T13:20:50Z
- **Completed:** 2026-06-27T13:30:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

### Task 1: RelayClient onResize + computeGuestScale helper

- Added `onResize?: (cols: number, rows: number) => void` to `RelayClientCallbacks` (optional for backward compat — host callers pass no onResize).
- Replaced the `// resize frames from server are informational` drop comment with an actual dispatch: `case 'resize': this.callbacks.onResize?.(frame.cols, frame.rows)`. Frame is already decoded by `parseServerFrame` — no extra decode needed.
- Created `frontend/src/lib/terminalScale.ts` exporting `computeGuestScale(cw, ch, gW, gH): number` — safe zero-grid guard, `s = min(cw/gW, ch/gH)`, cap at 1.0.
- `terminalScale.test.ts`: 10 tests covering cap-at-1, min-axis selection, zero gridW, zero gridH, both-zero, negative gridW, exact-fit, non-integer.
- `relayClient.test.ts`: extended with 6 new 0x02 dispatch tests (MSG_RESIZE=0x02, onResize called with correct big-endian decoded cols/rows, short frame guard, host backward compat no-throw).

### Task 2: TerminalPanel isGuest gate + CSS

- Added `import { computeGuestScale } from '../lib/terminalScale'`.
- `isGuestRef` pattern: `const isGuestRef = useRef<boolean>(false); isGuestRef.current = remote || !!wsURL` — updated every render so effects read the current value without dep-array churn.
- `recomputeScale` as `useCallback([], ...)`: reads `termRef.current._core._renderService.dimensions.css.cell` (same API as `fitTerminal`; Pitfall 6 — avoids getBoundingClientRect oscillation), computes scale, sets `term.element.style.transform = 'scale(s)'`.
- Mount effect guest path: `onOpen: isGuest ? undefined : ...sendResize` (host unchanged); `onResize: isGuest ? (cols,rows) => { term.resize(Math.max(1,cols), Math.max(1,rows)); recomputeScale() } : undefined` (Pitfall 5 order, T-157-03 clamp).
- Mount effect host path: `disposeResize = term.onResize(({cols,rows}) => client.sendResize(cols,rows))` — unchanged. Guest path: `{ dispose: () => {} }` no-op to satisfy IDisposable interface.
- isActive effect: guest branch uses single rAF + `ResizeObserver → recomputeScale()` (no fitTerminal); host branch unchanged (rAF tryFit loop + `ResizeObserver → fitTerminal`).
- `style.css`: added `.xterm { transform-origin: top left; }` — pins scale anchor to top-left corner, consistent with `terminal.css` web surface (VIEW-05 cross-surface parity).
- `TerminalPanel.scale.test.tsx`: 19 tests — source-inspection (isGuest gate, computeGuestScale import, Pitfall 5 order, guest/host ResizeObserver paths, CSS rule), guest behavioral (onResize→term.resize, zero clamp, transform set, sendResize never called), host invariance (sendResize from onOpen, no transform, onResize undefined).
- `TESTING.md`: vitest count 127→129, traceability rows for VIEW-04/VIEW-05 (4 rows), section-2 note.

## Task Commits

1. **Task 1: Add onResize to RelayClient + computeGuestScale helper** - `67c8e40c` (feat)
2. **Task 2: isGuest-gated honor + scale in TerminalPanel + desktop CSS** - `7b0870ed` (feat)

## Files Created/Modified

- `frontend/src/lib/terminalScale.ts` — computeGuestScale pure helper (new)
- `frontend/src/lib/terminalScale.test.ts` — 10 unit tests for scale math (new)
- `frontend/src/lib/relayClient.ts` — onResize callback + 0x02 dispatch (modified)
- `frontend/src/lib/relayClient.test.ts` — 6 new 0x02 dispatch tests (modified)
- `frontend/src/components/TerminalPanel.tsx` — isGuestRef, recomputeScale, guest gate (modified)
- `frontend/src/style.css` — .xterm { transform-origin: top left } (modified)
- `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` — 19 tests (new)
- `TESTING.md` — count + traceability update (modified)

## Decisions Made

- `onResize` is optional (`?`) so all existing `RelayClient` callers (TerminalPanel host path, tests with `{ onOutput: vi.fn() }`) continue to work without change. This follows the established pattern for all other optional callbacks (onPresence, onChat, etc.).
- `isGuestRef` pattern (vs useCallback dep-array): `isGuest` is derived from `remote || !!wsURL`, both mount-time props that don't change while the panel is mounted. Reading from a ref inside the effect avoids adding `isGuest` to the dep array of the isActive effect (which would otherwise require eslint-disable). Pattern mirrors the existing `pendingThemeRef` usage.
- `recomputeScale` as `useCallback([])`: stable reference (only reads from refs) — safe to use in both mount effect and isActive effect dep arrays without triggering extra runs.
- Guest isActive branch skips the rAF tryFit loop: the loop exists because fitTerminal needs cell dims to be ready before computing cols/rows. For guests, the PTY grid comes from the server via 0x02 (not from pixel-measuring), so there's no reason to measure before the server pushes. A single rAF triggers the initial recomputeScale (handles the layout-committed timing requirement).
- `Math.max(1, cols/rows)`: mirrors the web guest's `|| 1` clamp (T-157-03 — xterm rejects resize(0,0); a crafted zero-dim frame cannot crash the renderer).

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — `recomputeScale` is fully wired: called on 0x02 dispatch and on container ResizeObserver. The CSS transform-origin rule is in place. Behavioral proof (host + smaller guest window, no garble, correct pixel scale) is the manual Category P UAT item tracked in Plan 05.

## Threat Flags

No new security surface beyond the plan's threat model.

- **T-157-03 (Tampering):** `data.length < 5` guard in `parseServerFrame` (existing) + `Math.max(1, cols/rows)` clamp in TerminalPanel. A crafted zero-dim frame from a compromised server cannot crash the renderer.
- **T-157-07 (Mode leak):** `isGuest = remote || !!wsURL` gate applied at both RelayClient construction (onResize only provided for guest) and isActive effect (recomputeScale branch only executes for guest). Host path asserted by `TerminalPanel.scale.test.tsx` host invariance tests.

## Next Phase Readiness

Plan 05 (TESTING.md + manual UAT) can rely on:
- RelayClient dispatches 0x02 → onResize (6 automated tests)
- Desktop guest honors onResize → term.resize + transform (behavioral tests)
- Desktop guest never calls sendResize (behavioral test)
- Desktop host preserves sendResize + no transform (behavioral tests)
- CSS transform-origin: top left in style.css
- TESTING.md Section 4 traceability rows for VIEW-04/VIEW-05 already added

---
*Phase: 157-terminal-screen-share-semantics-issue-109*
*Completed: 2026-06-27*

## Self-Check: PASSED

- `frontend/src/lib/terminalScale.ts` — FOUND (created)
- `frontend/src/lib/terminalScale.test.ts` — FOUND (created)
- `frontend/src/lib/relayClient.ts` — FOUND (modified, onResize callback wired)
- `frontend/src/lib/relayClient.test.ts` — FOUND (modified, 6 new 0x02 tests)
- `frontend/src/components/TerminalPanel.tsx` — FOUND (modified, isGuest gate)
- `frontend/src/style.css` — FOUND (modified, transform-origin)
- `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` — FOUND (created)
- Commit `67c8e40c` — verified (Task 1 feat)
- Commit `7b0870ed` — verified (Task 2 feat)
- `pnpm test -- --run relayClient terminalScale TerminalPanel.scale` — 76/76 PASS
- `tsc --noEmit` — CLEAN (no output = no errors)
- `bash tests/check-traceability-paths.sh` — OK: all traceability paths exist
