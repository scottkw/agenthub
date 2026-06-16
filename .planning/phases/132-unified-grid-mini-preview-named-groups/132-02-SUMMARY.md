---
phase: 132
plan: "02"
subsystem: frontend-lib
tags: [hub-groups, remote-adapter, localStorage, vitest, tdd, GROUP-01, GROUP-03, GROUP-04, GRID-07]
dependency_graph:
  requires: []
  provides:
    - frontend/src/lib/hubGroups.ts (HubGroupDef CRUD + memberKey + persistence)
    - frontend/src/lib/remoteAdapter.ts (adaptRemoteSession + adaptAllRemoteSessions)
  affects:
    - Wave-2 components: GroupSidebar, SessionCard, SessionCardGrid, HubPanel
tech_stack:
  added: []
  patterns:
    - localStorage CRUD with try/catch JSON.parse resilience (mirrors Sidebar.tsx pattern)
    - TDD red-green cycle for pure TypeScript utility functions
key_files:
  created:
    - frontend/src/lib/hubGroups.ts
    - frontend/src/lib/hubGroups.test.ts
    - frontend/src/lib/remoteAdapter.ts
    - frontend/src/lib/remoteAdapter.test.ts
  modified: []
decisions:
  - memberKey empty workDir maps to __nodir__ sentinel (avoids empty string ambiguity in membership keys)
  - adaptAllRemoteSessions filters reachable===false peers at adapter boundary (T-132-06)
  - adaptRemoteSession sets workDir='' so remote sessions fall into default "Other" group (GROUP-04 design)
metrics:
  duration: "~2 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  files_created: 4
  tests_added: 27
---

# Phase 132 Plan 02: hubGroups + remoteAdapter Lib Modules Summary

Pure TypeScript utility library for Hub named groups (localStorage CRUD with memberKey derivation) and remote session adaptation (RemoteSession → SessionInfo mapping with reachable-peer filtering).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 RED | hubGroups.test.ts — failing tests | 4ade6d2 | frontend/src/lib/hubGroups.test.ts |
| 1 GREEN | hubGroups.ts — CRUD + localStorage | 2048691 | frontend/src/lib/hubGroups.ts |
| 2 RED | remoteAdapter.test.ts — failing tests | 2cac6d6 | frontend/src/lib/remoteAdapter.test.ts |
| 2 GREEN | remoteAdapter.ts — adapt functions | 12a7656 | frontend/src/lib/remoteAdapter.ts |

## What Was Built

### hubGroups.ts (GROUP-01/03/04)

- `HubGroupDef` interface: `{ id, name, memberKeys }` (uuid + user name + member key array)
- `memberKey(name, workDir)` — derives `${name}:::${workDir || '__nodir__'}` (GROUP-04)
- `loadGroups()` — reads `agenthub:hubGroups:v1` from localStorage; returns `[]` on absent or malformed JSON (T-132-04 resilience)
- `saveGroups(groups)` — serializes and writes to localStorage
- `createGroup(groups, name)` — appends new group with `crypto.randomUUID()` id, persists
- `assignToGroup(groups, groupId, key)` — atomically moves key from all other groups into target; no duplicate membership
- `removeFromGroup(groups, key)` — strips key from all groups, persists
- `deleteGroup(groups, groupId)` — removes by id, persists

### remoteAdapter.ts (GRID-07)

- `adaptRemoteSession(peer, session)` — maps `RemoteSession + RemotePeerSessions` → `SessionInfo`
  - `hostname` = `peer.hostname` (non-empty → GlobeAltIcon + hostname rendering in Wave-2 SessionCard)
  - `workDir = ''` → falls into default "Other" group (GROUP-04 design)
  - `status` defaults to `'running'` if `session.status` is empty
  - `webEnabled=true`, `viewerCount=0`, `homeDir=false`, `filesWrite=false`
- `adaptAllRemoteSessions(peers)` — filters `reachable===false` peers, flatMaps sessions (T-132-06)

## Verification Results

```
Test Files  2 passed (2)
Tests       27 passed (27)
pnpm tsc --noEmit  → 0 errors
```

## Acceptance Criteria Check

- [x] `pnpm vitest run src/lib/hubGroups.test.ts` — 13/13 green
- [x] `grep -c "agenthub:hubGroups:v1" hubGroups.ts` — 2 (STORAGE_KEY + HUB-GROUPS-V1 comment)
- [x] `grep -c "__nodir__" hubGroups.ts` — 1
- [x] `grep -c "GROUP-04: membership key" hubGroups.ts` — 1
- [x] `pnpm vitest run src/lib/remoteAdapter.test.ts` — 14/14 green
- [x] `grep -c "adaptRemoteSession" remoteAdapter.ts` — 3
- [x] `grep -c "GRID-07" remoteAdapter.ts` — 1
- [x] Unreachable-peer exclusion test present and green

## TDD Gate Compliance

- RED commit (hubGroups): 4ade6d2 — `test(132-02): add failing tests for hubGroups CRUD + memberKey + persistence`
- GREEN commit (hubGroups): 2048691 — `feat(132-02): implement hubGroups CRUD + memberKey + localStorage persistence`
- RED commit (remoteAdapter): 2cac6d6 — `test(132-02): add failing tests for remoteAdapter adaptRemoteSession + adaptAllRemoteSessions`
- GREEN commit (remoteAdapter): 12a7656 — `feat(132-02): implement remoteAdapter adaptRemoteSession + adaptAllRemoteSessions`

## Threat Model Compliance

| Threat ID | Mitigation Status |
|-----------|-------------------|
| T-132-04 | MITIGATED — loadGroups() wraps JSON.parse in try/catch; returns [] on malformed JSON |
| T-132-05 | DEFERRED — Pure data mapping only; downstream React text escaping enforced in Plan 04/05 |
| T-132-06 | MITIGATED — adaptAllRemoteSessions filters `reachable === false` peers; test case green |
| T-132-SC | N/A — Zero new npm dependencies |

## Deviations from Plan

None — plan executed exactly as written. Full file bodies from 132-PATTERNS.md followed verbatim.

## Known Stubs

None — these are pure utility functions with no UI rendering.

## Threat Flags

None — no new network endpoints, auth paths, or trust boundary crossings introduced.

## Self-Check: PASSED

- [x] frontend/src/lib/hubGroups.ts — FOUND
- [x] frontend/src/lib/hubGroups.test.ts — FOUND
- [x] frontend/src/lib/remoteAdapter.ts — FOUND
- [x] frontend/src/lib/remoteAdapter.test.ts — FOUND
- [x] commit 4ade6d2 — FOUND
- [x] commit 2048691 — FOUND
- [x] commit 2cac6d6 — FOUND
- [x] commit 12a7656 — FOUND
