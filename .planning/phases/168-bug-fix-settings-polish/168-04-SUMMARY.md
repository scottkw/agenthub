---
phase: 168-bug-fix-settings-polish
plan: 04
subsystem: settings
tags: [go, react, vitest, wails, daemon-settings, hub]

# Dependency graph
requires:
  - phase: 168-01
    provides: "SettingsTab.tsx / App.tsx baseline this plan extends (pre-existing settings-session-behavior and settings-behavior sections, createTab implementation)"
  - phase: 168-03
    provides: "openWebSessionTab / per-tab web-session param pattern (sibling App.tsx work in the same wave, no direct code dependency but shared file)"
provides:
  - "daemonSettings.StayOnHubAfterCreate bool (engine.go) + GetStayOnHubAfterCreate/SetStayOnHubAfterCreate (engine.go, client.go, app.go bound methods)"
  - "GET/PATCH /settings/stay-on-hub-after-create daemon-local routes (api.go)"
  - "Settings -> Session Behavior 'Stay on Hub after creating a session' toggle (SettingsTab.tsx), default OFF"
  - "App.tsx createTab gates its single setActiveId(sessionId) call on stayOnHubAfterCreateRef.current — the only auto-switch in the app"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Settings persistence chain (5-hop mirror of Phase 167 NotifyOnWaiting): engine.go daemonSettings field + Get/Set -> api.go 2 routes + 2 handlers -> client.go Get/Set wrappers -> app.go bound methods -> SettingsTab.tsx state/load/save. Zero-value default, no defaults-merge entry."
    - "App-level daemon-setting-into-ref pattern (autoCloseRef precedent): a setting read once via GetX().then(val => ref.current = val) on mount, avoiding useCallback dep churn and stale closures inside createTab."

key-files:
  created:
    - internal/daemon/engine_stayonhub_test.go
    - internal/daemon/api_stayonhub_test.go
    - frontend/src/components/__tests__/SettingsTab.stay-on-hub-toggle.test.tsx
    - frontend/src/components/__tests__/App.createTab.stayOnHub.test.tsx
  modified:
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - app.go
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/App.tsx
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx
    - frontend/src/components/__tests__/SettingsTab.notify-permission-hint.test.tsx
    - frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx
    - frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx
    - frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx
    - TESTING.md

key-decisions:
  - "GetStayOnHubAfterCreate/SetStayOnHubAfterCreate on App is a plain client passthrough (like SetStartMinimized), not an atomic-cached value like NotifyOnWaiting — there is no background reader (no tray poller) that needs a hot in-process cache."
  - "createTab reads the setting via a ref (stayOnHubAfterCreateRef), not React state, mirroring the existing autoCloseRef pattern — avoids adding the setting to createTab's useCallback deps and avoids stale-closure risk."
  - "No fromHub flag introduced (D-11 confirmed by 168-CONTEXT.md): createTab's single setActiveId(sessionId) call is the ONLY auto-switch in the app, so gating it there is sufficient — no per-call-site plumbing needed."

requirements-completed: [UX-01]

coverage:
  - id: D1
    description: "Daemon persistence chain for stayOnHubAfterCreate (engine.go/api.go/client.go/app.go) — defaults OFF, persists across daemon restart, GET/PATCH routes round-trip"
    requirement: "UX-01"
    verification:
      - kind: unit
        ref: "internal/daemon/engine_stayonhub_test.go#TestStayOnHubAfterCreate_Default, TestStayOnHubAfterCreate_Persists, TestStayOnHubAfterCreate_RoundTrip, TestStayOnHubAfterCreate_NoSchemaBump"
        status: pass
      - kind: unit
        ref: "internal/daemon/api_stayonhub_test.go#TestAPIGetStayOnHubAfterCreate_Default, TestAPIPatchStayOnHubAfterCreate_FlipsTrue, TestAPIPatchStayOnHubAfterCreate_BadBody, TestDaemonClient_GetSetStayOnHubAfterCreate_RoundTrip"
        status: pass
      - kind: other
        ref: "go test -race -short ./internal/daemon/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "Settings -> Session Behavior 'Stay on Hub after creating a session' toggle, default OFF, in the correct section (not the Behavior section where notifyOnWaiting lives)"
    requirement: "UX-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SettingsTab.stay-on-hub-toggle.test.tsx (15 tests: source contract + DOM render default-OFF + DOM render loaded-ON, including the section-placement guard)"
        status: pass
      - kind: other
        ref: "cd frontend && pnpm exec tsc --noEmit"
        status: pass
    human_judgment: false
  - id: D3
    description: "createTab gates its single setActiveId(sessionId) auto-switch on the setting; the tab is still always created"
    requirement: "UX-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/App.createTab.stayOnHub.test.tsx (5 tests: setTabs always runs before the gate, gate wraps setActiveId only, no fromHub flag introduced)"
        status: pass
    human_judgment: true
    rationale: "createTab is not exported and App.tsx is not fully mounted in this codebase's established test convention (source-inspection via App.tsx?raw, consistent with App.open-remote.test.tsx). The source-inspection tests prove the mechanism (ordering of setTabs vs. the gated setActiveId call, ref-based gating) but a live end-to-end confirmation — create a session from the Hub with the toggle ON and observe the active view stays on the Hub while the new tab appears in the strip — was not exercised. Recommend folding into the phase's live-UAT pass."

# Metrics
duration: 12min
completed: 2026-07-01
status: complete
---

# Phase 168 Plan 04: Stay on Hub after creating a session Summary

**A persisted `stayOnHubAfterCreate` daemon setting (5-hop chain mirroring Phase 167's NotifyOnWaiting) backs a new Settings → Session Behavior toggle that, when ON, skips `App.tsx`'s single `setActiveId(sessionId)` auto-switch in `createTab` so the user stays on the Hub after creating a session.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-01T20:45:21Z
- **Completed:** 2026-07-01T20:57:44Z
- **Tasks:** 3
- **Files modified:** 8 core (4 backend, 2 frontend + 2 wailsjs bindings) + 5 pre-existing test files (Rule 3 mock fixes) + TESTING.md

## Accomplishments

- `internal/daemon/engine.go`: new `StayOnHubAfterCreate bool` field on `daemonSettings` (json `stayOnHubAfterCreate,omitempty`, no defaults-merge entry, zero-value default OFF) plus `GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` engine methods mirroring `GetNotifyOnWaiting`/`SetNotifyOnWaiting` exactly.
- `internal/daemon/api.go`: `GET`/`PATCH /settings/stay-on-hub-after-create` daemon-local routes with handlers returning/accepting `{"stayOnHubAfterCreate": bool}`, 400 on decode error, 204 on set.
- `internal/daemon/client.go`: `GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` `DaemonClient` wrappers over `doJSON`.
- `app.go`: `App.GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` bound methods — a plain client passthrough (like `SetStartMinimized`), not an atomic-cached value, since there is no background reader that needs a hot cache.
- `frontend/src/components/SettingsTab.tsx`: new toggle rendered under `id="settings-session-behavior"` (next to `autoCloseSession`) — NOT `id="settings-behavior"` where `notifyOnWaiting` lives (D-08) — with label "Stay on Hub after creating a session", default OFF (D-09), instant toggle (no confirm dialog, mirrors `handleToggleNotifyOnWaiting`).
- `frontend/src/App.tsx`: `stayOnHubAfterCreateRef` loaded via `GetStayOnHubAfterCreate()` on mount (same ref-based pattern as `autoCloseRef`); `createTab`'s single `setActiveId(sessionId)` call — the only auto-switch in the app (D-11) — is now gated on `!stayOnHubAfterCreateRef.current`, while `setTabs` (tab creation) still always runs (D-10). No `fromHub` flag introduced.
- `frontend/src/wailsjs/go/main/App.js`/`App.d.ts`: new `GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` Wails bindings (Rule 3 — required for the new SettingsTab/App.tsx imports to resolve; these files are hand-maintained/checked-in, not gitignored generated output).
- Full daemon persistence chain, Settings toggle, and `createTab` gate covered by new Go and vitest tests (20 new test cases across 4 new files); `go test -race -short ./internal/daemon/...` and `pnpm exec tsc --noEmit` both clean; full frontend suite (139 files / 2306 tests) passes.

## Task Commits

Each task was committed atomically:

1. **Task 1: Daemon persistence chain for stayOnHubAfterCreate** - `0787d5d1` (feat)
2. **Task 2: Settings → Session Behavior toggle** - `9d340376` (test, RED) → `21583ad9` (feat, GREEN)
3. **Task 3: Gate createTab auto-switch on the setting** - `07688025` (feat, includes the App.createTab.stayOnHub.test.tsx test + the Rule 3 mock fixes to 5 pre-existing SettingsTab test files)

**Additional commit (standing convention, TESTING.md):** `5a3676db` (docs)

**Plan metadata:** commit to follow (docs: complete plan)

## Files Created/Modified

- `internal/daemon/engine.go` - `daemonSettings.StayOnHubAfterCreate` field, backing engine field, load-from-disk wiring, `GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` methods.
- `internal/daemon/api.go` - route registration + `handleGetStayOnHubAfterCreate`/`handleSetStayOnHubAfterCreate`.
- `internal/daemon/client.go` - `GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` client wrappers.
- `app.go` - `App.GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` bound methods.
- `internal/daemon/engine_stayonhub_test.go` (new) - default/persist/round-trip/no-schema-bump tests, copied from `engine_notify_test.go`.
- `internal/daemon/api_stayonhub_test.go` (new) - GET/PATCH route handler tests + client round-trip, copied from `api_notify_test.go`.
- `frontend/src/components/SettingsTab.tsx` - `stayOnHubAfterCreate` state/load/save + toggle markup in the Session Behavior section.
- `frontend/src/components/__tests__/SettingsTab.stay-on-hub-toggle.test.tsx` (new) - source-contract + DOM render tests (15 tests), copied from `SettingsTab.notify-toggle.test.tsx`.
- `frontend/src/App.tsx` - `GetStayOnHubAfterCreate` import, `stayOnHubAfterCreateRef`, load-on-mount effect, `createTab` gate.
- `frontend/src/components/__tests__/App.createTab.stayOnHub.test.tsx` (new) - source-inspection tests (5 tests) proving the gate shape and ordering.
- `frontend/src/wailsjs/go/main/App.js` / `App.d.ts` - new Wails bindings.
- `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx`, `SettingsTab.notify-permission-hint.test.tsx`, `SettingsTab.notify-toggle.test.tsx`, `SettingsTab.shell-warn-toggle.test.tsx`, `SettingsTab.shellPath.test.tsx` - added `GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` mocks (Rule 3 fix, see Deviations).
- `TESTING.md` - Section 2 (Suite Manifest, counts 371→373 Go / 137→139 vitest / 519→523 total) and Section 4 (traceability, 4 new UX-01 rows).

## Decisions Made

- **`GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` on `App` are a plain client passthrough**, not an atomic-cached value like `NotifyOnWaiting`. `NotifyOnWaiting`'s cache exists because the tray-poller goroutine reads it every tick; `stayOnHubAfterCreate` has no such background reader, so a direct `a.client.Get/Set...` call (mirroring `SetStartMinimized`) is sufficient and simpler — matches the plan's explicit guidance.
- **`createTab` reads the setting via a `useRef`, not `useState`** — mirrors the existing `autoCloseRef` pattern exactly, avoiding both a `useCallback` dependency-array churn (which would recreate `createTab` on every toggle) and stale-closure risk.
- **No `fromHub` flag introduced.** 168-CONTEXT.md's D-11 confirms `createTab`'s single `setActiveId(sessionId)` call is the only auto-switch in the entire app (CLI-created sessions never route through `createTab`), so gating that one call site is complete — no additional call-site plumbing was needed or added.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added Wails bindings for the new App methods**
- **Found during:** Task 2
- **Issue:** `SettingsTab.tsx` needs to import `GetStayOnHubAfterCreate`/`SetStayOnHubAfterCreate` from `frontend/src/wailsjs/go/main/App`, but these hand-maintained/checked-in binding files (`App.js`/`App.d.ts`) had no entries for the new `app.go` methods — the app would not build without them (Wails bindings are normally regenerated by `wails build`, unavailable in this environment).
- **Fix:** Added the two binding pairs, mirroring the exact `GetNotifyOnWaiting`/`SetNotifyOnWaiting` shape immediately above them in both files.
- **Files modified:** `frontend/src/wailsjs/go/main/App.js`, `frontend/src/wailsjs/go/main/App.d.ts`
- **Verification:** `pnpm exec tsc --noEmit` passes; `pnpm vitest run SettingsTab.stay-on-hub-toggle` passes.
- **Committed in:** `21583ad9` (Task 2 GREEN commit)

**2. [Rule 3 - Blocking] Added missing Wails binding mocks to 5 pre-existing SettingsTab test files**
- **Found during:** Task 3 (running the full frontend suite as part of the wave-merge verification)
- **Issue:** `SettingsTab.appearance-theme.test.tsx`, `SettingsTab.notify-permission-hint.test.tsx`, `SettingsTab.notify-toggle.test.tsx`, `SettingsTab.shell-warn-toggle.test.tsx`, and `SettingsTab.shellPath.test.tsx` each `vi.mock('../../wailsjs/go/main/App', ...)` with an explicit binding list (no `importOriginal` passthrough). Task 2's new `SettingsTab.tsx` import of `GetStayOnHubAfterCreate` was `undefined` in the mocked module for these 5 files, throwing inside the new `useEffect` and crashing the component render (5 test files / 31 tests failing with "An error occurred in the SettingsTab component").
- **Fix:** Added `GetStayOnHubAfterCreate: vi.fn().mockResolvedValue(false)` and `SetStayOnHubAfterCreate: vi.fn().mockResolvedValue(undefined)` to each file's mock, immediately after the existing `GetNotifyOnWaiting`/`SetNotifyOnWaiting` mocks — same remediation pattern as Phase 167-07's `EventsOn` fix for the same 4 files (this phase added a 5th, `SettingsTab.appearance-theme.test.tsx`, which also needed it).
- **Files modified:** `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx`, `SettingsTab.notify-permission-hint.test.tsx`, `SettingsTab.notify-toggle.test.tsx`, `SettingsTab.shell-warn-toggle.test.tsx`, `SettingsTab.shellPath.test.tsx`
- **Verification:** `pnpm vitest run` — 139 files / 2306 tests, all pass.
- **Committed in:** `07688025` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues directly caused by this plan's own new Wails bindings/imports, necessary for the build and existing tests to keep working).
**Impact on plan:** Both fixes are direct, necessary consequences of the plan's own instructions (mirror the NotifyOnWaiting chain, which requires the same binding surface). No scope creep — no unrelated code was touched.

## TDD Gate Compliance

- **Task 1** (daemon persistence chain): implementation was written before the mirrored test files (`engine_stayonhub_test.go`, `api_stayonhub_test.go`), so no RED state was captured for this task — a single `feat` commit (`0787d5d1`) contains both the implementation and tests. This deviates from strict RED-then-GREEN, but the task is a byte-for-byte mirror of an already-proven, already-tested pattern (Phase 167 `NotifyOnWaiting`), so the risk of skipping the RED confirmation is low. All 8 tests pass; `go vet`/`go build` clean.
- **Task 2** (Settings toggle): proper RED→GREEN sequence followed — `9d340376` (test, confirmed failing against pre-change `SettingsTab.tsx`) → `21583ad9` (feat, GREEN, 15/15 pass).
- **Task 3** (createTab gate): the `App.tsx` implementation was written before `App.createTab.stayOnHub.test.tsx`, so — like Task 1 — no RED state was captured; both land in a single `feat` commit (`07688025`). Same low-risk rationale as Task 1 (mirrors the established `autoCloseRef` pattern). All 5 tests pass on first run.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- UX-01 (#116) is code-complete: the daemon persistence chain, Settings toggle, and `createTab` gate are all implemented and unit-proven (20 new tests across 4 new files, all passing), `tsc --noEmit` clean, full frontend suite (139 files / 2306 tests) passes, `go test -race -short ./internal/daemon/...` passes.
- **Deferred (human judgment, D3 above):** live end-to-end confirmation that toggling "Stay on Hub after creating a session" ON in Settings and then creating a session from the Hub actually keeps the active view on the Hub (with the new tab visible in the strip but not focused) — `createTab` is internal to `App.tsx` and not exercised by a full-App-mount test in this codebase's established testing convention. Recommend folding into this phase's other deferred live-UAT items or the next `/gsd-verify-work 168` pass.
- No manual-checklist (M-NN) item was added — this behavior is a pure React/daemon-settings toggle with no native GUI, remote-peer, live-PTY, or physical-hardware dependency, so it is fully covered by the automated suite per the standing convention's carve-out.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-01*

## Self-Check: PASSED

- FOUND: internal/daemon/engine.go
- FOUND: internal/daemon/api.go
- FOUND: internal/daemon/client.go
- FOUND: app.go
- FOUND: internal/daemon/engine_stayonhub_test.go
- FOUND: internal/daemon/api_stayonhub_test.go
- FOUND: frontend/src/components/SettingsTab.tsx
- FOUND: frontend/src/components/__tests__/SettingsTab.stay-on-hub-toggle.test.tsx
- FOUND: frontend/src/App.tsx
- FOUND: frontend/src/components/__tests__/App.createTab.stayOnHub.test.tsx
- FOUND: frontend/src/wailsjs/go/main/App.js
- FOUND: frontend/src/wailsjs/go/main/App.d.ts
- FOUND: TESTING.md
- FOUND commit: 0787d5d1
- FOUND commit: 9d340376
- FOUND commit: 21583ad9
- FOUND commit: 07688025
- FOUND commit: 5a3676db
- FOUND commit: ce1eb8a9
