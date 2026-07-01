---
phase: 168-bug-fix-settings-polish
fixed_at: 2026-07-01T23:02:05Z
review_path: .planning/phases/168-bug-fix-settings-polish/168-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 168: Code Review Fix Report

**Fixed at:** 2026-07-01T23:02:05Z
**Source review:** .planning/phases/168-bug-fix-settings-polish/168-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (CR-01, CR-02, CR-03, WR-01, WR-02, WR-03)
- Fixed: 6
- Skipped: 0 (IN-01, IN-02 intentionally out of scope per instructions — not attempted)

## Fixed Issues

### CR-01: Footer "Share Session" button silently no-ops for a just-created session

**Files modified:** `frontend/src/App.tsx`
**Commit:** `e5391299`
**Applied fix:** `createTab`'s success path now appends a `SessionInfo`-shaped entry to `hubSessions` (keyed by the new `sessionId`) immediately after `CreateSession` resolves, before the auto-switch `setActiveId` call and before the `catch`. This mirrors the mount-time seed already documented at `App.tsx:~511` so `openShareModalForActiveSession`'s `hubSessions.find(s => s.id === activeId)` lookup succeeds for a session created via "New Session" even before the Hub-tab-gated 3s poll ever runs.

### CR-02: SessionShareModal receives no live updates when opened from a non-Hub tab

**Files modified:** `frontend/src/App.tsx`
**Commit:** `c6255942`
**Applied fix:** Added a second, narrowly-scoped `useEffect` poll (`ListSessions()` every 3s) that runs only while `shareModalSession` is set AND the Hub tab is not already polling (`activeId !== HUB_TAB.id`). This decouples the modal's live-truth feed from Hub-tab-active gating per the review's suggested approach, while preserving the T-131-10 DoS-guard intent — no always-on poll is introduced; polling only happens while a Share modal is actually open. The effect is keyed on `shareModalSession?.id` (not the whole object) to avoid tearing down/restarting the interval on every tick (the sibling sync effect replaces the object reference on each successful poll).

### CR-03: Missing `key` on `<WebShareSessionView>` leaks chat/plugin-config state between two open remote-peer session tabs

**Files modified:** `frontend/src/App.tsx`
**Commit:** `554f09b9`
**Applied fix:** Added `key={wsSessionId}` to the `<WebShareSessionView>` element at the `__websession__` render branch, exactly as suggested in the review. This forces React to remount the component (rather than reuse the instance and its `useState` `chatOpen`/`unreadCount`/`hasMention`/`livePluginConfig`) whenever the active web-session tab switches to a different remote session.

### WR-01: SessionShareModal shows "sharing ON" even when the shell web-share confirm actually failed on the backend

**Files modified:** `frontend/src/App.tsx`, `frontend/src/components/Hub/SessionShareModal.tsx`, `frontend/src/components/__tests__/SessionShareModal.test.tsx`
**Commit:** `5820fa6d`
**Applied fix:** `handleShellWebShareConfirm` now returns `Promise<boolean>` (`true` on success, `false` on the caught error path) instead of `Promise<void>`. `SessionShareModal`'s banner `onConfirm` handler only calls `setShareEnabled(true)` when the awaited result is truthy. Updated the `SessionShareModalProps` type and the modal's local test-harness `ModalOpts` type accordingly (`() => Promise<boolean>`); updated all `onShellWebShareConfirm: vi.fn().mockResolvedValue(undefined)` mocks in the existing shell-warning test suite to `.mockResolvedValue(true)` for type correctness (these tests don't reach the confirm click path, so behavior is unaffected) and strengthened the one test that does reach it plus added a new failure-path regression test.

### WR-02: `internal/daemon/engine.go` fails `gofmt -l`

**Files modified:** `internal/daemon/engine.go`
**Commit:** `794ca22c`
**Applied fix:** `gofmt -w internal/daemon/engine.go`. Single-line diff realigning the `NotifyOnWaiting` trailing comment to match the wider `StayOnHubAfterCreate` tag column.

### WR-03: `internal/daemon/types.go` also fails `gofmt -l`

**Files modified:** `internal/daemon/types.go`
**Commit:** `1e070a94`
**Applied fix:** `gofmt -w internal/daemon/types.go`. Merges the two previously separately-aligned comment blocks in the `SessionInfo` struct into one gofmt-aligned block.

## Additional test coverage (not a separate finding — verification requirement)

**Commit:** `1204d189`
Per the task's verification instructions, added/extended regression tests that would have caught CR-01 and CR-03, following the codebase's established source-inspection convention for `App.tsx` logic (App is not fully mounted in this test suite — see `App.createTab.stayOnHub.test.tsx`, `App.open-remote.test.tsx`, `App.wiring.test.tsx`):
- New file `frontend/src/components/__tests__/App.createTab.hubSessionsSeed.test.tsx` — asserts `createTab` calls `setHubSessions` inside its `try` block, keyed by `sessionId`, before the `catch`.
- Extended `frontend/src/components/__tests__/App.open-remote.test.tsx`'s existing per-tab-isolation describe block with an assertion that `<WebShareSessionView>` carries `key={wsSessionId}`.
- Extended `frontend/src/components/__tests__/SessionShareModal.test.tsx` with a WR-01 failure-path test (toggle stays OFF when `onShellWebShareConfirm` resolves `false`) and strengthened the existing success-path test to assert the toggle actually flips ON.

A full-`App`-mount behavioral test for CR-01 (e.g. actually creating a session then clicking the footer button and observing the modal open) was not attempted — this repo's established convention for `App.tsx`-level logic is source-inspection via `App.tsx?raw` string assertions, not full mounting (multiple prior phase test files document this explicitly, citing Wails RPC coupling as the reason). CR-02's live-poll timing (the new interval effect) does not have a dedicated fake-timer test; it is exercised indirectly by the existing `SessionShareModal.test.tsx` Funnel warm-up suite via the parent's `hubSessions` prop threading, but a focused App-level timer test was judged out of scope for this fix pass and is noted in `TESTING.md` as a follow-up.

Per `TESTING.md`'s standing regression-test convention (Section 6, `.planning`/repo-level `CLAUDE.md`), updated `TESTING.md`'s Suite Manifest counts (vitest 140→141, Total 525→526), added a traceability row for the new `App.createTab.hubSessionsSeed.test.tsx` file under requirement ID `UX-02`, and added a Note summarizing this fix pass's test changes. Ran `bash tests/check-traceability-paths.sh` — passes.

## Skipped Issues

None — all 6 in-scope findings (CR-01, CR-02, CR-03, WR-01, WR-02, WR-03) were fixed. IN-01 and IN-02 were explicitly out of scope per the task instructions and were not attempted.

## Verification Output

**TypeScript (`cd frontend && npx tsc --noEmit`):** clean, exit 0 (no errors).

**Targeted vitest suite** (`StatusBar.shareSession`, `App.open-remote`, `SessionShareModal.disconnect`, `SessionShareModal`, `App.createTab.stayOnHub`, `App.shellWebShare`, plus the new `App.createTab.hubSessionsSeed`):
```
Test Files  7 passed (7)
     Tests  103 passed (103)
```

**Full frontend vitest suite** (sanity check beyond the required subset):
```
Test Files  141 passed (141)
     Tests  2320 passed (2320)
```
(jsdom canvas `getContext` warnings are pre-existing environment noise, not failures.)

**Go:**
```
$ gofmt -l internal/daemon/engine.go internal/daemon/types.go
(empty)

$ go build ./...
(clean, exit 0)

$ go vet ./...
(clean, exit 0)

$ go test ./internal/daemon/... ./internal/relay/...
ok  	github.com/scottkw/agenthub/internal/daemon	26.881s
ok  	github.com/scottkw/agenthub/internal/relay	3.243s
```

**Traceability check:**
```
$ bash tests/check-traceability-paths.sh
OK: all traceability paths exist
```

---

_Fixed: 2026-07-01T23:02:05Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
