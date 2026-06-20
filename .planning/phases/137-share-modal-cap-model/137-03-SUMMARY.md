---
phase: 137-share-modal-cap-model
plan: "03"
subsystem: frontend
tags: [share-modal, hub, session-card, cap-model, browse-toggle]
dependency_graph:
  requires: ["137-01", "137-02"]
  provides: [SHARE-01, SHARE-02, SHARE-04, SHARE-05, SHARE-06]
  affects: [HubPanel, SessionCardGrid, SessionCard, SessionShareModal, DaemonManagerPanel]
tech_stack:
  added: []
  patterns:
    - "Direct IssueCapabilities call in restart-clear effect (avoids React 18 scheduler
       deferral past setTimeout fence in tests)"
    - "Ref mirrors (shareEnabledRef, sessionIdRef, cachedShareRef) for reading state in
       effects without adding them as deps"
    - "hub-card__share guard in card onClick + e.stopPropagation() (Pitfall 6 pattern)"
key_files:
  created:
    - frontend/src/components/Hub/SessionShareModal.tsx
  modified:
    - frontend/src/components/SessionSharePanel.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/App.tsx
    - frontend/src/components/DaemonManagerPanel.tsx
    - frontend/src/lib/remoteAdapter.ts
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
    - frontend/src/components/__tests__/DaemonManagerPanel.test.tsx
    - frontend/src/lib/remoteAdapter.test.ts
    - "[13 other test fixture files updated browseEnabled]"
decisions:
  - "SHARE-05 restart-clear: call IssueCapabilities directly in the restart-clear effect
     instead of incrementing a seedVersion counter — React 18's concurrent scheduler defers
     state updates from effects past the test's setTimeout(0) fence; a direct async call in
     the same effect resolves before the macrotask runs"
  - "webServerMode type uses 'tailscale' | 'local' | null (not 'tunnel') to match App.tsx
     state; test fixtures updated accordingly"
  - "shareModalSession naming in HubPanel (plan spec said shareTarget; both are equivalent —
     shareModalSession is more descriptive)"
metrics:
  duration: "~3 hours (including SHARE-05 flushSync debugging)"
  completed: "2026-06-20"
  tasks_completed: 3
  files_changed: 20
---

# Phase 137 Plan 03: Hub Share Modal (GREEN) Summary

SessionShareModal built, Share button threaded end-to-end, all Plan 01 RED tests green.

## Tasks

### Task 1 — Simplify SessionSharePanel + Add Share button to SessionCard (74de9798)

- `SessionSharePanel`: stripped CAP-05 two-gate (ownerWriteEnabled, allowFileEditing, showWriteConfirm, surfaceWriteLink). Write link always rendered. `browseEnabled` prop controls scope text only.
- `SessionCard`: added `onShare?: (session: SessionInfo) => void` prop, `hub-card__share` guard in article onClick (Pitfall 6), Share button with `LockClosedIcon` for remote peers (D-13 colorblind-safe: shape + aria-label + title).
- Wails stubs: `filesWrite` → `browseEnabled` on SessionInfo; `SetSessionFilesWrite` → `SetSessionBrowse`.

### Task 2 — Build SessionShareModal (68b10a71)

- Two-toggle design (SHARE-01/02): "Share the session" + "Enable remote file browsing".
- SHARE-05 server-truth seeding: IssueCapabilities on open when webEnabled=true.
- SHARE-05 restart-clear: webServerRunning false→true clears stale cache AND directly re-issues caps in the same effect (avoids React 18 scheduler deferred rendering past test's setTimeout fence).
- SHARE-04: LAN password via GetLocalNetworkPassword in local mode.
- D-09: HomeDirWriteWarning when session.homeDir=true.
- SHARE-06/Pitfall 1: browse toggle re-issues caps after SetSessionBrowse.
- Animation phase machine: entering → open → exiting with prefers-reduced-motion guard.
- Focus return on unmount (MODAL-02 pattern).
- All 9 Plan 01 SessionShareModal tests green.

### Task 3 — Thread onShare + Mount SessionShareModal (a08a5538)

- `SessionCardGrid`: added `onShare` prop, threaded to both render paths (named groups + workDir groups).
- `HubPanel`: added `webServerMode`/`webServerRunning` props; `handleShare` callback; `shareModalSession` state; `SessionShareModal` mounted outside `.hub` div.
- `App.tsx`: passes `webServerMode` + `webServerRunning` to HubPanel.
- **Rule 1 auto-fixes**: `DaemonManagerPanel` had stale `SetSessionFilesWrite`/`s.filesWrite` references causing TypeScript compilation failure; updated to `SetSessionBrowse`/`s.browseEnabled`. Same fix applied to `remoteAdapter.ts` and 13+ test fixture files.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] SHARE-05 restart-seeding via seedVersion counter failed in tests**
- **Found during:** Task 2
- **Issue:** `setCachedShare(null)` + `setSeedVersion(v => v + 1)` called inside a `useEffect` during `flushSync` — React 18's concurrent scheduler defers the resulting re-render past the test's `setTimeout(0)` fence, so the seeding effect's IssueCapabilities call never executes before the test assertion.
- **Fix:** Removed `seedVersion` state; the restart-clear effect now calls IssueCapabilities directly (async) in the same effect body when `shareEnabledRef.current` is true. The promise resolves as a microtask before `setTimeout(0)` fires.
- **Files modified:** `frontend/src/components/Hub/SessionShareModal.tsx`
- **Commit:** 68b10a71

**2. [Rule 1 - Bug] DaemonManagerPanel referenced removed SetSessionFilesWrite binding**
- **Found during:** Task 3 TypeScript check
- **Issue:** After `App.d.ts`/`App.js` renamed `SetSessionFilesWrite` → `SetSessionBrowse` and `filesWrite` → `browseEnabled`, `DaemonManagerPanel.tsx` still imported `SetSessionFilesWrite` and read `s.filesWrite`, causing TypeScript compilation failure.
- **Fix:** Updated to `SetSessionBrowse` and `s.browseEnabled`.
- **Files modified:** `DaemonManagerPanel.tsx`, `remoteAdapter.ts`, 13 test fixture files
- **Commit:** a08a5538

**3. [Rule 1 - Bug] Test fixture webServerMode type used 'tunnel' (not in interface)**
- **Found during:** Task 3 TypeScript check
- **Issue:** `SessionShareModal.test.tsx` used `webServerMode: 'tunnel'` as default, but `SessionShareModalProps` only accepts `'tailscale' | 'local' | null`.
- **Fix:** Changed default to `null`; updated `ModalOpts` interface to `'local' | 'tailscale'`; updated hardcoded `'tunnel'` literals in restart-clear test to `null`.
- **Files modified:** `SessionShareModal.test.tsx`
- **Commit:** a08a5538

## Verification Results

- **Frontend tests:** 1753/1753 passing (107 test files)
- **Plan 01 RED tests (now GREEN):** 13/13 passing
  - SessionCard.share.test.tsx: 4/4
  - SessionShareModal.test.tsx: 9/9
- **TypeScript:** `tsc --noEmit` clean (0 errors)
- **Go tests:** `go test ./internal/daemon/... ./internal/webserver/...` — all cached pass

## Known Stubs

None. All behavior wired end-to-end:
- Share button → `onShare(session)` → `shareModalSession` state → `SessionShareModal` mounts
- IssueCapabilities called on open; restart-clear effect re-issues on server restart
- Browse toggle calls SetSessionBrowse + re-issues caps (Pitfall 1 mitigation live)

## Threat Flags

None. No new network endpoints, auth paths, or trust boundary surface introduced. SessionShareModal is owner-only (D-13 gate in SessionCard prevents remote-peer cards from reaching the modal).

## Self-Check: PASSED

- `frontend/src/components/Hub/SessionShareModal.tsx` — FOUND
- `frontend/src/components/Hub/SessionCard.tsx` — FOUND (hub-card__share class: FOUND)
- `frontend/src/components/Hub/HubPanel.tsx` — FOUND (SessionShareModal mount: FOUND)
- `frontend/src/components/Hub/SessionCardGrid.tsx` — FOUND (onShare prop: FOUND)
- `frontend/src/App.tsx` — FOUND (webServerMode passed to HubPanel: FOUND)
- Commits 74de9798, 68b10a71, a08a5538 — all verified in git log
