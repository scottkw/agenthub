---
phase: 97-serialize-addon-save-session-ux
plan: "03"
subsystem: ui
tags: [phase-97, serialize, app-saver-registry, callback-prop, wave-1, saver-pipeline]

# Dependency graph
requires:
  - phase: 97-01
    provides: Wave 0 RED scaffolds for App.saver.test.tsx
  - phase: 97-02
    provides: stripAnsi.ts and sanitizeFilename.ts helpers ready for import in App.tsx

provides:
  - "frontend/src/App.tsx — serializerRegistry state + handleRegisterSaver + handleRequestSave + saveBanner state + JSX prop passes to TerminalPanel and TabBar"
  - "frontend/src/wailsjs/go/main/App.d.ts — SaveTerminalSession type stub (runtime Call() stub lands in Plan 97-05)"
  - "frontend/src/__tests__/App.saver.test.tsx — 11 RED scaffolds flipped to GREEN source-scan assertions"

affects: [97-04, 97-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Saver-registry callback chain: React state in App.tsx holds Record<sessionId, () => string | null>; TerminalPanel registers closures via onRegisterSaver prop; TabBar triggers saves via onRequestSave prop"
    - "Wave-bridge ts-expect-error: TerminalPanel/TabBar props interfaces not yet extended (97-04 wires consumer side); @ts-expect-error comments suppress TSC errors on unknown props until 97-04 lands"
    - "saveBanner state mirrors existing localBanner one-shot pattern: {kind: 'info' | 'error'; text: string} | null, rendered inside the existing banner-stack div"
    - "Type stub pattern: App.d.ts SaveTerminalSession declaration allows compile-time resolution; runtime Call() stub deferred to Plan 97-05 hand-edit"
    - "Source-scan test convention: expect(raw).toMatch(...) assertions verify structural invariants without needing to stand up Wails/browser runtime"

key-files:
  created: []
  modified:
    - frontend/src/App.tsx
    - frontend/src/__tests__/App.saver.test.tsx
    - frontend/src/wailsjs/go/main/App.d.ts

key-decisions:
  - "Used @ts-expect-error wave-bridge (Option B) instead of `as any` casts — `as any` on the value does not suppress unknown-prop JSX errors in TypeScript strict mode; @ts-expect-error on the attribute line correctly suppresses the TS2322 error"
  - "saveBanner rendered inside existing banner-stack with gate condition extended to include `saveBanner !== null` — no new container div, consistent with existing banner rendering pattern"
  - "handleRegisterSaver has empty useCallback deps because setSerializerRegistry is stable (React useState setter identity guarantee)"
  - "handleRequestSave uses [tabs, serializerRegistry] deps — both must be current at invocation time to resolve correct sessionId and saver function"
  - "SaveTerminalSession type stub uses named params (defaultDir, defaultName, content) consistent with OpenFileDialog precedent in App.d.ts, not arg1/arg2/arg3 style used by SetImageConfig"

# Metrics
duration: ~12 min
completed: 2026-05-07
tasks_completed: 1
files_modified: 3
---

# Phase 97 Plan 03: App.tsx Saver-Registry Pipeline Summary

**One-liner:** App.tsx saver-registry wired with handleRegisterSaver + handleRequestSave callbacks, stripAnsi + sanitizeFilename + SaveTerminalSession pipeline, and 11 App.saver.test.tsx RED scaffolds flipped GREEN

## What Was Built

Plan 97-03 introduces the structurally-novel shape of Phase 97: the **saver-registry callback chain**. This is the only file in Phase 97 with genuinely new architecture (all other files are near-verbatim analogs of existing patterns).

### Saver Registry Contract

App.tsx now holds:
```typescript
const [serializerRegistry, setSerializerRegistry] = useState<
  Record<string, (() => string) | null>
>({})
```

TerminalPanel (Plan 97-04) will register closures via `handleRegisterSaver(sessionId, fn)` on addon attach and unregister via `handleRegisterSaver(sessionId, null)` on detach. The registry is keyed by `sessionId` (not `tabId`) because the closure captures the SerializeAddon instance bound to a specific terminal session.

### handleRequestSave Pipeline

```
TabBar right-click "Save Terminal As…"
  → onRequestSave(tabId)
  → tab lookup by tabId
  → serializerRegistry[tab.sessionId] → fn()
  → stripAnsi(fn())         ← T-97-03-01 mitigation (ANSI escapes stripped before disk)
  → sanitizeFilename(tab.name) + timestamp + '.txt'  ← T-97-03-05 mitigation
  → SaveTerminalSession('', fname, plainText)
```

If no saver is registered (Serialize toggled OFF), shows info banner: "Enable the Serialize plugin in Settings to save sessions." — per RESEARCH locked decision (always show menu item; toast if disabled).

### saveBanner State

Mirrors the existing `localBannerDismissed` one-shot pattern. `saveBanner: { kind: 'info' | 'error'; text: string } | null` — info kind for "Serialize disabled" affordance, error kind for write/dialog failures. Rendered inside the existing `<div className="banner-stack">` without introducing new CSS classes.

### SaveTerminalSession Type Stub

`frontend/src/wailsjs/go/main/App.d.ts` received:
```typescript
// Save terminal session (Phase 97 SER-01) — type stub; runtime Call() stub lands in Plan 97-05.
export function SaveTerminalSession(defaultDir: string, defaultName: string, content: string): Promise<void>
```
This allows App.tsx to import and reference `SaveTerminalSession` at compile time. The runtime `Call('main.App.SaveTerminalSession', [...])` stub in `App.js` lands in Plan 97-05.

### Wave-Bridge Casts

TerminalPanel and TabBar do not yet declare `onRegisterSaver` / `onRequestSave` in their props interfaces (that wiring lands in Plan 97-04). To keep `pnpm tsc --noEmit` GREEN, `@ts-expect-error` comments are placed on the JSX attribute lines:

```tsx
// @ts-expect-error — Plan 97-04 wires onRegisterSaver prop on TerminalPanel; wave-bridge cast
onRegisterSaver={handleRegisterSaver}
```

Plan 97-04 removes these comments when the props interfaces formally accept the callbacks.

## Test Results

All 11 `App.saver.test.tsx` tests pass GREEN:
1. App.tsx imports stripAnsi from ./lib/stripAnsi
2. App.tsx imports sanitizeFilename from ./lib/sanitizeFilename
3. App.tsx imports SaveTerminalSession from wailsjs binding
4. App.tsx declares serializerRegistry state
5. App.tsx declares handleRegisterSaver useCallback
6. App.tsx declares handleRequestSave useCallback (strip + sanitize + save chain)
7. handleRequestSave shows banner when registry empty (Serialize OFF)
8. App.tsx passes onRegisterSaver={handleRegisterSaver} to TerminalPanel
9. App.tsx passes onRequestSave={handleRequestSave} to TabBar
10. TerminalPanel.tsx file is loadable (defensive)
11. TabBar.tsx file is loadable (defensive)

`pnpm tsc --noEmit` exits 0 (with @ts-expect-error wave-bridge annotations).

Pre-existing failing tests (Sidebar RTL tests + TerminalPanel/TabBar/PluginsSection RED scaffolds for 97-04/97-05) are not regressions — same count as before this plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `as any` cast does not suppress unknown-prop JSX TSC error**
- **Found during:** Task 1 — running `pnpm tsc --noEmit`
- **Issue:** Plan specified Option A (`onRegisterSaver={handleRegisterSaver as any}`) but TypeScript 5.x strict mode emits TS2322 for unknown properties on JSX elements regardless of the value type cast — the cast applies to the value but TSC's unknown-prop check fires on the attribute name
- **Fix:** Switched to Option B (`@ts-expect-error` comment on the JSX attribute line) which correctly suppresses TS2322
- **Files modified:** `frontend/src/App.tsx`
- **Commit:** 90f506e

## Task Commits

1. **Task 1: Wire App.tsx saver-registry pipeline + flip RED scaffold to GREEN** — `90f506e` (feat)

## Files Created/Modified

- `frontend/src/App.tsx` — serializerRegistry + saveBanner state, handleRegisterSaver + handleRequestSave callbacks, imports, JSX prop passes, banner rendering
- `frontend/src/__tests__/App.saver.test.tsx` — 11 RED expect.fail() scaffolds replaced with real source-scan assertions; all GREEN
- `frontend/src/wailsjs/go/main/App.d.ts` — SaveTerminalSession type stub appended after SetImageConfig

## Decisions Made

- @ts-expect-error wave-bridge (not `as any`) for unknown JSX props — technically correct, does not mask the intent
- saveBanner gate condition added to existing banner-stack outer conditional rather than creating a separate always-rendered container — consistent with existing banner rendering idiom
- handleRequestSave checks `!tab.sessionId` (welcome tab guard) per the plan spec — welcome tab has `sessionId: ''` which is falsy

## Known Stubs

- `SaveTerminalSession` in `App.d.ts`: type-only stub — runtime `Call()` in `App.js` lands in Plan 97-05
- `@ts-expect-error` wave-bridge casts on `onRegisterSaver` and `onRequestSave` props — removed in Plan 97-04

## Threat Surface Scan

No new network endpoints or auth paths introduced. The plan's threat register (T-97-03-01 through T-97-03-05) is fully addressed:

| Threat ID | Mitigation | Status |
|-----------|-----------|--------|
| T-97-03-01 | `stripAnsi(fn())` before Wails RPC call — ANSI escapes stripped before bytes cross boundary | Implemented |
| T-97-03-02 | handleRequestSave checks `if (!fn)` before invoking; unregister-on-unmount in Plan 97-04 | Partial (Plan 97-04 completes) |
| T-97-03-03 | Wails dialog is modal — accepted | Accepted |
| T-97-03-04 | Error wrapping in Plan 97-05 SaveTerminalSession | Deferred to 97-05 |
| T-97-03-05 | `sanitizeFilename` collapses whitespace including `\n`/`\t` | Implemented |

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| `frontend/src/App.tsx` exists | FOUND |
| `frontend/src/__tests__/App.saver.test.tsx` exists | FOUND |
| `frontend/src/wailsjs/go/main/App.d.ts` exists | FOUND |
| `97-03-SUMMARY.md` exists | FOUND |
| Commit `90f506e` exists | FOUND |
| 11/11 App.saver.test.tsx tests GREEN | PASSED |
| `pnpm tsc --noEmit` exits 0 | PASSED |
