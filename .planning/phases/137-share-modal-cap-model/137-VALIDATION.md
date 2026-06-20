---
phase: 137
slug: share-modal-cap-model
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-20
validated: 2026-06-20
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

| Requirement | Wave | Threat Ref | Secure Behavior | Test Type | Automated Command (as delivered) | File Exists | Status |
|-------------|------|------------|-----------------|-----------|-------------------|-------------|--------|
| SHARE-01 | 1 | — | Share button on Hub card opens modal | unit (frontend) | `pnpm test SessionCard.share` | ✅ | ✅ green |
| SHARE-02 | 1 | — | "Share the session" toggle reveals RO + RW links/codes + QR | unit (frontend) | `pnpm test SessionShareModal` | ✅ | ✅ green |
| ★ SHARE-03 / D-03 | 0 | T-137-01 | Browse OFF: RO=`read`, RW=`read,write` (no file perms) | unit (Go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOff_NoFilesPerms` | ✅ | ✅ green |
| ★ SHARE-03 / D-04 RO | 0 | T-137-02 | Browse ON: RO=`read,files.read` exactly (never files.write) | unit (Go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_ROPermsExact` | ✅ | ✅ green |
| ★ SHARE-03 / D-04 RW | 0 | T-137-02 | Browse ON: RW=`read,write,files.read,files.write` exactly | unit (Go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_RWPermsExact` | ✅ | ✅ green |
| ★ SHARE-03 cross-surface | 0 | T-137-03 | RO cap + files.read → web file browse 200 (read-only) | unit (Go webserver) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_BrowseOn_FilesReadRoute200` | ✅ | ✅ green |
| ★ SHARE-03 cross-surface | 0 | T-137-03 | RO cap (browse ON) → web file WRITE 403 (no files.write) | unit (Go webserver) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_BrowseOn_WriteRoute403` | ✅ | ✅ green |
| ★ SHARE-03 cross-surface | 0 | T-137-03 | RW cap (browse ON) → web file WRITE 200 | unit (Go webserver) | `go test ./internal/webserver/... -run TestFilesRoutes_RW_BrowseOn_WriteRoute200` | ✅ | ✅ green |
| SHARE-03 cross-surface | 1 | T-137-03 | Browse OFF RO cap → web file browse 403 | unit (Go webserver) | `go test ./internal/webserver/... -run TestRequireFilesRead` | ✅ | ✅ green |
| SHARE-04 | 1 | — | LAN Basic Auth password visible in modal in local mode | unit (frontend) | `pnpm test SessionShareModal` (local-mode fixture) | ✅ | ✅ green |
| SHARE-04 | 1 | — | QR codes copyable per link row | unit (frontend) | `pnpm test SessionSharePanel` | ✅ | ✅ green |
| SHARE-05 regression | 1 | — | Server-truth seeding on modal open (webEnabled + caps) | unit (frontend) | `pnpm test SessionShareModal` (webEnabled fixture) | ✅ | ✅ green |
| SHARE-05 regression | 1 | — | Stale URL cleared on web-server restart | unit (frontend) | `pnpm test SessionShareModal` (server-restart fixture) | ✅ | ✅ green |
| SHARE-06 / D-13 | 1 | — | Remote peer card Share button disabled (lock icon + tooltip, colorblind-safe) | unit + source (frontend) | `pnpm test SessionCard.share` (remote fixture) | ✅ | ✅ green |
| ★ D-02 removal | 0 | T-137-04 | `ownerWriteEnabled` prop + CAP-05 opt-in stripped (no AllowFileEditing toggle) | unit (frontend) | `pnpm test SessionSharePanel` | ✅ | ✅ green |
| ★ D-07 removal | 0 | T-137-05 | `filesReadEnabled()` no longer called in perm injection | unit (Go) + grep | `go test ./internal/daemon/... -run TestIssueCapabilities` | ✅ | ✅ green |
| D-09 | 1 | — | Home-dir warning shown when cwd=$HOME before enabling browse | unit (frontend) | `pnpm test SessionShareModal` (homeDir fixture) | ✅ | ✅ green |
| ★ no-substring | 0 | T-137-06 | New code uses whole-token `HasPerm`, never `strings.Contains` on perms | static grep | `go test ./internal/webserver/... -run TestHasPerm_NoStringsContains_Browse` | ✅ | ✅ green |
| CSRF invariant | 1 | T-137-07 | `requireFilesWrite` still calls `originAllowedForWrite` (unchanged) | unit (Go webserver) | `go test ./internal/webserver/... -run TestRequireFilesWrite` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*★ = security-delta row, primary audit target for `/gsd:secure-phase`.*
*Name drift reconciled at validation: planned `TestFilesRoutes_RO_BrowseOn`→`…_FilesReadRoute200`; `TestFilesRoutes_RO_NoWrite`→`…_WriteRoute403`; `TestHasPerm`→`TestHasPerm_NoStringsContains_Browse`. RW-write-200 row added (delivered beyond plan).*

---

## Wave 0 Requirements

- [x] `internal/daemon/api_test.go` — retired 4 old `TestIssueCapabilities_*` global-flag tests; added `TestIssueCapabilities_BrowseOff_NoFilesPerms`, `TestIssueCapabilities_BrowseOn_ROPermsExact`, `TestIssueCapabilities_BrowseOn_RWPermsExact` (commit 657dbb1e RED → fad0275f GREEN)
- [x] `internal/webserver/files_routes_test.go` — added `TestFilesRoutes_RO_BrowseOn_FilesReadRoute200` (RO cap + files.read → 200), `TestFilesRoutes_RO_BrowseOn_WriteRoute403` (RO cap → 403 on write), `TestFilesRoutes_RW_BrowseOn_WriteRoute200` (RW cap → 200); `TestHasPerm_NoStringsContains_Browse` grep gate in `capability_test.go` (commit af89d4cb)
- [x] `frontend/src/components/__tests__/SessionCard.share.test.tsx` — Share button renders; click fires onShare without bubbling to onCardClick; disabled on remote peer (4/4 green, commit becb06cc RED → 74de9798 GREEN)
- [x] `frontend/src/components/__tests__/SessionShareModal.test.tsx` — share toggle; browse toggle; LAN password (local-mode); homeDir warning; server-truth seeding on open; stale-URL cleared on server restart (9/9 green, commit becb06cc RED → 68b10a71 GREEN)
- [x] `frontend/src/components/__tests__/SessionSharePanel.test.tsx` — removed CAP-05 opt-in tests; added simplified-panel tests (write link shown whenever sharing ON) (9/9 green, commit becb06cc → 74de9798)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| End-to-end web file browse from a real remote peer using a presented RO vs RW code | SHARE-03 | Cross-surface live behavior (web client consuming a real cap) is not fully exercisable in unit tests | Daemon-backed UAT: enable sharing + browse, present RO code in browser → confirm read-only file browse; present RW code → confirm read/write; verify per `reference_live_uat_daemon_gotchas` |
| Disabled Share button on remote peer card is colorblind-safe by appearance | SHARE-06 / D-13 | User is colorblind — verify at source (lock icon + tooltip present), not by eye | Inspect SessionCard source for non-color affordance; confirm tooltip text "Only the session owner can share" |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-20 — all 19 automated rows green; 2 manual-only rows exercised live in UAT (137-HUMAN-UAT.md, 6/6 pass)

---

## Validation Audit 2026-06-20

State A audit of the pre-execution draft against delivered tests. All commands below were re-run live during this audit and confirmed green.

| Metric | Count |
|--------|-------|
| Per-Task rows audited | 19 |
| COVERED (green, verified live) | 19 |
| PARTIAL | 0 |
| MISSING (gaps found) | 0 |
| Gaps filled by auditor | 0 (none needed) |
| Manual-only (retained) | 2 — both exercised live in UAT |

**Reconciliation notes:** The draft was authored pre-execution (all rows `⬜ pending`, `nyquist_compliant: false`). No coverage gaps existed — every planned test was delivered and passes. The audit corrected three planned→delivered test-name drifts and added the `TestFilesRoutes_RW_BrowseOn_WriteRoute200` row (delivered beyond the original plan). No `gsd-nyquist-auditor` spawn was required (zero gaps to fill). No new test files generated.

**Live re-run evidence (this audit):**
- `go test ./internal/daemon/... -run 'TestIssueCapabilities|TestKillSession_ClearsStaleBrowseEntry'` → ok
- `go test ./internal/webserver/... -run 'TestFilesRoutes_R|TestRequireFiles|TestHasPerm_NoStringsContains_Browse'` → ok
- `pnpm exec vitest run SessionShareModal SessionCard.share SessionSharePanel` → 3 files / 22 tests passed
