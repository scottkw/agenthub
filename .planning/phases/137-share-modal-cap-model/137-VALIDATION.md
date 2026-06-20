---
phase: 137
slug: share-modal-cap-model
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-20
---

# Phase 137 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from 137-RESEARCH.md §Validation Architecture. Security deltas (perms matrix
> D-03/D-04, removed gates D-02/D-07) are the audit target for the later `/gsd:secure-phase`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `go test` + frontend `vitest` |
| **Config file** | `frontend/vitest.config.ts` (existing); Go uses standard `testing` |
| **Quick run command** | `go test ./internal/daemon/... -run TestIssueCapabilities && go test ./internal/webserver/... -run TestRequireFiles` |
| **Full suite command** | `go test ./... && cd frontend && pnpm test` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/daemon/... -run TestIssueCapabilities && go test ./internal/webserver/... -run TestRequireFiles && cd frontend && pnpm test SessionSharePanel SessionShareModal`
- **After every plan wave:** `go test ./... && cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

> Task IDs are assigned by the planner; this map is requirement-keyed and the planner
> must attach each row to the task that delivers it. Security-delta rows (★) are mandatory.

| Requirement | Wave | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|-------------|------|------------|-----------------|-----------|-------------------|-------------|--------|
| SHARE-01 | 1 | — | Share button on Hub card opens modal | unit (frontend) | `pnpm test SessionCard` | ❌ W0 | ⬜ pending |
| SHARE-02 | 1 | — | "Share the session" toggle reveals RO + RW links/codes + QR | unit (frontend) | `pnpm test SessionShareModal` | ❌ W0 | ⬜ pending |
| ★ SHARE-03 / D-03 | 0 | T-137-01 | Browse OFF: RO=`read`, RW=`read,write` (no file perms) | unit (Go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOff_NoFilesPerms` | ❌ W0 | ⬜ pending |
| ★ SHARE-03 / D-04 RO | 0 | T-137-02 | Browse ON: RO=`read,files.read` exactly (never files.write) | unit (Go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_ROPermsExact` | ❌ W0 | ⬜ pending |
| ★ SHARE-03 / D-04 RW | 0 | T-137-02 | Browse ON: RW=`read,write,files.read,files.write` exactly | unit (Go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_RWPermsExact` | ❌ W0 | ⬜ pending |
| ★ SHARE-03 cross-surface | 0 | T-137-03 | RO cap + files.read → web file browse 200 (read-only) | unit (Go webserver) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_BrowseOn` | ❌ W0 | ⬜ pending |
| ★ SHARE-03 cross-surface | 0 | T-137-03 | RO cap (browse ON) → web file WRITE 403 (no files.write) | unit (Go webserver) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_NoWrite` | ❌ W0 | ⬜ pending |
| SHARE-03 cross-surface | 1 | T-137-03 | Browse OFF RO cap → web file browse 403 | unit (Go webserver) | `go test ./internal/webserver/... -run TestRequireFilesRead` | ✅ | ⬜ pending |
| SHARE-04 | 1 | — | LAN Basic Auth password visible in modal in local mode | unit (frontend) | `pnpm test SessionShareModal` (local-mode fixture) | ❌ W0 | ⬜ pending |
| SHARE-04 | 1 | — | QR codes copyable per link row | unit (frontend) | `pnpm test SessionSharePanel` | ✅ | ⬜ pending |
| SHARE-05 regression | 1 | — | Server-truth seeding on modal open (webEnabled + caps) | unit (frontend) | `pnpm test SessionShareModal` (webEnabled fixture) | ❌ W0 | ⬜ pending |
| SHARE-05 regression | 1 | — | Stale URL cleared on web-server restart | unit (frontend) | `pnpm test SessionShareModal` (server-restart fixture) | ❌ W0 | ⬜ pending |
| SHARE-06 / D-13 | 1 | — | Remote peer card Share button disabled (lock icon + tooltip, colorblind-safe) | unit + source (frontend) | `pnpm test SessionCard` (remote fixture) | ❌ W0 | ⬜ pending |
| ★ D-02 removal | 0 | T-137-04 | `ownerWriteEnabled` prop + CAP-05 opt-in stripped (no AllowFileEditing toggle) | unit (frontend) | `pnpm test SessionSharePanel` | ⚠️ retire-old | ⬜ pending |
| ★ D-07 removal | 0 | T-137-05 | `filesReadEnabled()` no longer called in perm injection | unit (Go) + grep | `go test ./internal/daemon/... -run TestIssueCapabilities` | ⚠️ retire-old | ⬜ pending |
| D-09 | 1 | — | Home-dir warning shown when cwd=$HOME before enabling browse | unit (frontend) | `pnpm test SessionShareModal` (homeDir fixture) | ❌ W0 | ⬜ pending |
| ★ no-substring | 0 | T-137-06 | New code uses whole-token `HasPerm`, never `strings.Contains` on perms | static grep | `go test ./internal/webserver/... -run TestHasPerm` | ✅ (extend) | ⬜ pending |
| CSRF invariant | 1 | T-137-07 | `requireFilesWrite` still calls `originAllowedForWrite` (unchanged) | unit (Go webserver) | `go test ./internal/webserver/... -run TestRequireFilesWrite` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*★ = security-delta row, primary audit target for `/gsd:secure-phase`.*

---

## Wave 0 Requirements

- [ ] `internal/daemon/api_test.go` — retire 4 old `TestIssueCapabilities_*` global-flag tests (lines ~1982-2115); add `TestIssueCapabilities_BrowseOff_NoFilesPerms`, `TestIssueCapabilities_BrowseOn_ROPermsExact`, `TestIssueCapabilities_BrowseOn_RWPermsExact`
- [ ] `internal/webserver/files_routes_test.go` — add `TestFilesRoutes_RO_BrowseOn` (RO cap + files.read → 200) and `TestFilesRoutes_RO_NoWrite` (RO cap + files.read, no files.write → 403 on write route)
- [ ] `frontend/src/components/__tests__/SessionCard.share.test.tsx` — Share button renders; click fires onShare without bubbling to onCardClick; disabled on remote peer
- [ ] `frontend/src/components/__tests__/SessionShareModal.test.tsx` — share toggle; browse toggle; LAN password (local-mode); homeDir warning; server-truth seeding on open; stale-URL cleared on server restart
- [ ] `frontend/src/components/__tests__/SessionSharePanel.test.tsx` — update existing: remove CAP-05 opt-in tests; add simplified-panel tests (write link shown whenever sharing ON)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| End-to-end web file browse from a real remote peer using a presented RO vs RW code | SHARE-03 | Cross-surface live behavior (web client consuming a real cap) is not fully exercisable in unit tests | Daemon-backed UAT: enable sharing + browse, present RO code in browser → confirm read-only file browse; present RW code → confirm read/write; verify per `reference_live_uat_daemon_gotchas` |
| Disabled Share button on remote peer card is colorblind-safe by appearance | SHARE-06 / D-13 | User is colorblind — verify at source (lock icon + tooltip present), not by eye | Inspect SessionCard source for non-color affordance; confirm tooltip text "Only the session owner can share" |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
