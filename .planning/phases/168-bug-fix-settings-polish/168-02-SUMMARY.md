---
phase: 168-bug-fix-settings-polish
plan: 02
subsystem: frontend
tags: [react, vitest, sse, eventsource, fetch, plugin-config, web-share]

# Dependency graph
requires:
  - phase: 155
    provides: WebShareSessionView's wsURL/apiBaseURL construction (Phase 155-03, PARITY-01) that this plan parameterizes on baseURL
provides:
  - "WebShareSessionView baseURL?: string prop — apiBaseURL/wsURL derive from a resolved origin (default window.location.origin) instead of a hardcoded window.location reference"
  - "WebShareSessionView web-guest livePluginConfig self-fetch (/api/plugin-config?cap=) + EventSource hot-swap (/api/plugin-config/stream?cap=), applied to TerminalPanel"
affects: [168-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "isWebGuest = pluginConfig === undefined: absence of a Wails-provided pluginConfig prop is the signal to self-fetch/subscribe; explicit null opts a caller out"
    - "baseURL seam: apiBaseURL/wsURL always derive from a resolved origin (baseURL ?? window.location.origin), never a bare window.location reference, so remote-peer tabs (FIX-03) can reuse the same component"

key-files:
  created:
    - frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx
  modified:
    - frontend/src/components/Hub/WebShareSessionView.tsx
    - frontend/src/components/Hub/WebShareSessionView.test.tsx
    - TESTING.md

key-decisions:
  - "Effective config (livePluginConfig once populated, else the incoming pluginConfig prop) is forwarded to TerminalPanel only — ChatPanel has no pluginConfig prop in this codebase to receive it, despite the plan's must_haves phrasing implying both."
  - "wsURL's ws/wss scheme now derives from the resolved origin's http/https scheme instead of being hardcoded to wss — a direct, intended consequence of the baseURL-derivation task, not a new decision outside the plan."
  - "T-168-03 (CSP gap on /app/) filed as a GitHub follow-up (scottkw/agenthub#123) per the plan's threat-model disposition (transfer, out of locked scope) rather than fixed inline."

requirements-completed: [FIX-01]

coverage:
  - id: D1
    description: "WebShareSessionView accepts an optional baseURL prop; apiBaseURL and wsURL derive from it (default window.location.origin), reproducing today's behavior when omitted"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/WebShareSessionView.test.tsx#WebShareSessionView — wsURL construction (Pitfall 6 guard)"
        status: pass
      - kind: other
        ref: "cd frontend && pnpm exec tsc --noEmit"
        status: pass
    human_judgment: false
  - id: D2
    description: "A web guest (no Wails pluginConfig prop) self-fetches /api/plugin-config?cap=<capToken> on mount and applies the config to TerminalPanel"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx#WebShareSessionView — web-guest plugin-config self-fetch (FIX-01)"
        status: pass
    human_judgment: false
  - id: D3
    description: "An EventSource 'plugin-config' event updates the live plugin config without remount/page reload"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx#WebShareSessionView — SSE hot-swap (FIX-01)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Desktop path (Wails pluginConfig prop provided) creates no fetch/EventSource; unmount aborts the fetch AbortController and closes the EventSource"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx#WebShareSessionView — desktop (Wails pluginConfig prop) path (FIX-01)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx#WebShareSessionView — cleanup on unmount (FIX-01)"
        status: pass
    human_judgment: false
  - id: D5
    description: "A web-share guest opening a session URL in a real browser sees live plugin-config changes without page reload (end-to-end, real browser + real backend)"
    requirement: "FIX-01"
    verification: []
    human_judgment: true
    rationale: "Backend SSE endpoints (/api/plugin-config, /api/plugin-config/stream) and the guest-side EventSource wiring are each unit-proven in isolation (D2-D4, plus pre-existing Go tests in internal/webserver/plugin_config_test.go and plugin_config_stream_test.go), but an actual live browser reconnecting to a real capability-gated session over the webserver has never been exercised end-to-end since the Phase 159 /sessions->/app redirect broke this path (#112). Genuine live-UAT judgment call, consistent with this phase's other deferred manual-UAT items."

# Metrics
duration: 11min
completed: 2026-07-01
status: complete
---

# Phase 168 Plan 02: Web-guest plugin-config self-fetch + SSE hot-swap Summary

**WebShareSessionView self-fetches /api/plugin-config and subscribes to its SSE stream when acting as a web guest, restoring live plugin hot-swap lost after the Phase 159 /sessions->/app redirect (#112), and gains a baseURL prop seam for FIX-03's remote-peer reuse.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-07-01T20:14:07Z
- **Completed:** 2026-07-01T20:25:02Z
- **Tasks:** 2
- **Files modified:** 3 (1 created, 2 modified) + TESTING.md

## Accomplishments

- Added an optional `baseURL?: string` prop to `WebShareSessionView`; `apiBaseURL`/`wsURL` now derive from a resolved origin (`baseURL ?? window.location.origin`) instead of a hardcoded `window.location` reference, with the ws/wss scheme converted from the resolved origin's http/https scheme.
- Web guests (no Wails-provided `pluginConfig` prop) now self-fetch `/api/plugin-config?cap=<capToken>` on mount and subscribe via `EventSource` to `/api/plugin-config/stream?cap=<capToken>`, applying each `plugin-config` push to the terminal without a page reload.
- The desktop path (a Wails `pluginConfig` prop is provided) creates no fetch/EventSource; unmount aborts the fetch `AbortController` and closes the `EventSource`.
- New dedicated vitest suite (`WebShareSessionView.plugin-config.test.tsx`, 5 tests) proves the full behavior contract with a fake `EventSource` and mocked `fetch`.
- Filed a GitHub follow-up issue (scottkw/agenthub#123) for the `/app/` CSP gap flagged by the plan's threat model (T-168-03), per its "transfer" disposition — not fixed inline, per locked scope (D-16).

## Task Commits

Each task was committed atomically:

1. **Task 1: Add baseURL prop and derive apiBaseURL/wsURL from it** - `17e9ab45` (feat)
2. **Task 2: Web-guest plugin-config self-fetch + SSE hot-swap** - `99b0e423` (test, RED) -> `d7ca219f` (feat, GREEN)

**Additional commit (standing convention, TESTING.md):** `2ae59b21` (docs)

**Plan metadata:** commit to follow (docs: complete plan)

## Files Created/Modified

- `frontend/src/components/Hub/WebShareSessionView.tsx` - Added `baseURL` prop, `resolvedOrigin`/`wsOrigin` derivation, `livePluginConfig` state + self-fetch/EventSource `useEffect`, effective-config forwarding to `TerminalPanel`
- `frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx` - New vitest suite (5 tests) covering self-fetch, SSE hot-swap, desktop no-op, and unmount cleanup
- `frontend/src/components/Hub/WebShareSessionView.test.tsx` - Updated wsURL-scheme assertions to derive `ws`/`wss` from `window.location.protocol`; `mountView` now passes `pluginConfig={null}` explicitly so this suite's wsURL/rendering assertions don't trigger the new web-guest effect
- `TESTING.md` - Section 2 (Suite Manifest) count + Note, Section 4 (traceability) new `FIX-01` row (disambiguated from the pre-existing Phase ~100 `FIX-01` row)

## Decisions Made

- **Effective config forwarded to TerminalPanel only.** The plan's must_haves text says "applies the returned config to TerminalPanel/ChatPanel", but `ChatPanel`'s props interface (`frontend/src/components/Hub/ChatPanel.tsx:64-89`) has no `pluginConfig` prop at all — it never consumed plugin settings (those govern xterm addon loading: webgl/unicode11/clipboard/weblinks, not chat). Forwarding only to `TerminalPanel` (the actual, sole consumer) matches the codebase's real prop surface rather than inventing an unused prop on `ChatPanel`.
- **`isWebGuest = pluginConfig === undefined`** (not `== null`). This lets a caller opt out of the self-fetch/SSE path by passing `pluginConfig={null}` explicitly, distinct from simply omitting the prop. Matches the plan's literal wording ("absence of a Wails-provided pluginConfig prop").
- **wsURL scheme now derives from the resolved origin** (http->ws, https->wss) instead of being hardcoded to `wss`, per Task 1's explicit instruction ("instead of the hardcoded wss://${window.location.host}"). In real deployment (web-share is always served over HTTPS) this is behavior-identical to before; it only differs in the jsdom test environment (default `http://localhost:3000`), which required updating the pre-existing test's expectations (see Deviations).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing `WebShareSessionView.test.tsx` wsURL assertions broken by the intended scheme-derivation change**
- **Found during:** Task 1
- **Issue:** The plan's Task 1 explicitly directs replacing the hardcoded `wss://${window.location.host}` with a scheme derived from the resolved origin. jsdom's default test origin is `http://localhost:3000`, so the pre-existing suite's hardcoded `wss://` expectations failed (4/11 tests) once the derivation landed.
- **Fix:** Updated the 4 assertions in `WebShareSessionView.test.tsx` to compute the expected scheme from `window.location.protocol` (`https:` -> `wss`, else `ws`) instead of a literal `wss`.
- **Files modified:** `frontend/src/components/Hub/WebShareSessionView.test.tsx`
- **Verification:** `pnpm vitest run WebShareSessionView.test.tsx` — 11/11 pass
- **Committed in:** `17e9ab45` (part of Task 1 commit)

**2. [Rule 1 - Bug] Pre-existing `WebShareSessionView.test.tsx` crashed after Task 2's self-fetch effect landed**
- **Found during:** Task 2
- **Issue:** `mountView`'s default props omit `pluginConfig`, which Task 2 now interprets as "web guest" (`isWebGuest = pluginConfig === undefined`), triggering the new self-fetch/EventSource effect. jsdom has no global `EventSource`, so every test in that suite crashed with `ReferenceError: EventSource is not defined`.
- **Fix:** Updated `mountView`'s default render to pass `pluginConfig={null}` explicitly, opting that suite (which tests wsURL/rendering, not plugin-config behavior) out of the new effect. The dedicated behavior suite (`WebShareSessionView.plugin-config.test.tsx`) covers the self-fetch/SSE contract directly.
- **Files modified:** `frontend/src/components/Hub/WebShareSessionView.test.tsx`
- **Verification:** `pnpm vitest run WebShareSessionView` — 16/16 pass; full suite `pnpm vitest run` — 137 files / 2280 tests pass
- **Committed in:** `d7ca219f` (part of Task 2 GREEN commit)

**3. [Standing convention, ./CLAUDE.md + TESTING.md] Registered the new test file in TESTING.md**
- **Found during:** post-Task-2 (before final commit)
- **Issue:** ./CLAUDE.md's Regression Test Convention requires every new test file to be added to TESTING.md's Suite Manifest (Section 2) and Traceability map (Section 4).
- **Fix:** Added a Section 2 Note + count bump (vitest 136->137, total 518->519) and a Section 4 `FIX-01` row for `WebShareSessionView.plugin-config.test.tsx`, disambiguated from the pre-existing Phase ~100 `FIX-01` row (same requirement-ID text, different milestone/meaning — IDs are scoped per-milestone in this project, not globally unique).
- **Files modified:** `TESTING.md`
- **Verification:** `bash tests/check-traceability-paths.sh` exits 0
- **Committed in:** `2ae59b21`

---

**Total deviations:** 3 auto-fixed (2 Rule-1 bug fixes directly caused by this plan's intended changes, 1 standing-convention doc update)
**Impact on plan:** All three fixes are direct, necessary consequences of the plan's own instructions (scheme derivation, self-fetch effect) plus a mandatory project convention. No scope creep — no unrelated code was touched.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The `baseURL` prop seam is in place and proven not to change default (web-guest) behavior — 168-03 (FIX-03) can pass a remote peer's origin directly.
- The `isWebGuest` self-fetch effect will also fire transiently for 168-03's in-app remote-peer tabs (App.tsx's own `pluginConfig` state is irrelevant to a different peer's session), which is the intended behavior — App.tsx's `handleOpenRemoteSession` rewrite (168-03) is unaffected by anything in this plan.
- **Deferred:** live end-to-end UAT of the self-fetch/SSE hot-swap against a real capability-gated session (D5 above) — tracked alongside this phase's other deferred manual-UAT items in STATE.md.
- **Deferred:** `/app/` CSP gap (T-168-03) — tracked as scottkw/agenthub#123, out of this plan's locked scope.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-01*

## Self-Check: PASSED

- FOUND: frontend/src/components/Hub/WebShareSessionView.tsx
- FOUND: frontend/src/components/Hub/WebShareSessionView.test.tsx
- FOUND: frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx
- FOUND commit: 17e9ab45
- FOUND commit: 99b0e423
- FOUND commit: d7ca219f
- FOUND commit: 2ae59b21
