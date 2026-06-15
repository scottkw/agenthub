---
phase: 123
slug: td-cleanup-write-sandbox-primitives-daemon-routes
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-14
---

# Phase 123 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib testing + go fuzz) |
| **Config file** | none — Go module native |
| **Quick run command** | `go test ./internal/files/... ./internal/daemon/...` |
| **Full suite command** | `go test -race ./internal/files/... ./internal/daemon/...` |
| **Estimated runtime** | ~30-90 seconds (fuzz adds 60s when `-fuzz` enabled) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/files/... ./internal/daemon/...`
- **After every plan wave:** Run `go test -race ./internal/files/... ./internal/daemon/...`
- **Before `/gsd:verify-work`:** Full suite must be green; `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/...` reports zero crashes
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Requirement | Plan | Secure Behavior | Test File · Function | Test Type | Automated Command | Status |
|-------------|------|-----------------|----------------------|-----------|-------------------|--------|
| FSW-01 | 123-01 | Atomic temp+Sync+rename; no O_TRUNC; reader never sees partial | `internal/files/write_test.go` · `TestWriteFileAtomic`, `_Overwrite`, `_Subdir`, `_ConcurrentReadNeverPartial` | unit | `go test -race ./internal/files/ -run 'TestWriteFileAtomic'` | ✅ green |
| FSW-02 | 123-01 | Rename validates BOTH src+dest; destination traversal rejected | `internal/files/write_test.go` · `TestRename_SameDir`, `_CrossDirMove`, `_DestinationTraversalRejected`, `_SourceTraversalRejected` | unit | `go test -race ./internal/files/ -run 'TestRename'` | ✅ green |
| FSW-03 | 123-01 | Mkdir/MkdirAll within sandbox; traversal rejected | `internal/files/write_test.go` · `TestMkdir`, `TestMkdirAll`, `TestMkdir_TraversalRejected` | unit | `go test -race ./internal/files/ -run 'TestMkdir'` | ✅ green |
| FSW-04 | 123-01 | Delete file or recursive subtree; stays in sandbox | `internal/files/write_test.go` · `TestDelete_File`, `_RecursiveSubtree`, `_TraversalRejected` | unit | `go test -race ./internal/files/ -run 'TestDelete'` | ✅ green |
| FSW-05 | 123-03 | Upload filename sanitized via filepath.Base + validateAndClean | `internal/files/handler_test.go` · `TestHandlerUpload_FilenameSanitized` (`../../.bashrc`), `_EmptyFilename400`, `_DotFilename400`, `_DotDotFilename400` | unit (HTTP) | `go test -race ./internal/files/ -run 'TestHandlerUpload_Filename\|TestHandlerUpload_.*400'` | ✅ green |
| FSW-06 | 123-01/03 | Denylist enforced in ALL sandbox write methods AND HTTP layer → 403 "Protected system file" | `internal/files/write_test.go` · `TestDenylist_HomeRooted` (9 targets × 4 methods), `_CaseVariation`, `_DaemonConfigDir`, `_NonHomeRootedUnaffected`; `internal/files/handler_test.go` · `TestHandlerUpload_DenylistForbidden` | unit + HTTP | `go test -race ./internal/files/ -run 'TestDenylist\|TestHandlerUpload_DenylistForbidden'` | ✅ green |
| FSW-07 | 123-01 | FuzzSandboxWrite 60s 0-crash merge gate | `internal/files/sandbox_test.go` · `FuzzSandboxWrite` | fuzz | `go test -run '^$' -fuzz='^FuzzSandboxWrite$' -fuzztime=60s ./internal/files/` | ✅ green |
| FSW-08 | 123-03 | Five auth-less method-prefixed daemon write routes | `internal/daemon/api_test.go` · `TestFilesWriteRoutes_Registered`, `_WrongVerb405`, `_WriteRoundTrip` | integration | `go test -race ./internal/daemon/ -run 'TestFilesWriteRoutes'` | ✅ green |
| FSW-09 | 123-04 | DaemonClient write methods (Write/Upload/Delete/Rename/Mkdir) | `internal/daemon/client_test.go` · `TestDaemonClientWrite_RoundTrip`, `Upload_`, `Delete_`, `Rename_`, `Mkdir_RoundTrip`, `_NonOKError`, `_ContextCancel` | integration | `go test -race ./internal/daemon/ -run 'TestDaemonClient(Write\|Upload\|Delete\|Rename\|Mkdir)'` | ✅ green |
| FSW-10 | 123-02 | TD-5: ExchangeJoinCodeAtURL parses 303 Location ?cap=<token>; shared client unmutated | `internal/daemon/client_test.go` · `TestExchangeJoinCode_303Success`, `_NoAutoFollow`, `_ErrorLocation`, `_EmptyCap`, `_SharedClientUnchanged`, `_AbsoluteLocation*` | integration | `go test -race ./internal/daemon/ -run 'TestExchangeJoinCode'` | ✅ green |
| FSW-11 | 123-02 | TD-4: WR-01..05 file-browser hardening verified/applied | `internal/daemon/client_test.go` · `TestExchangeJoinCode_WR05_AbsoluteErrorLocation` + verifier source review (server.go, FileBrowserTab.tsx, FileRow.tsx, humanSize.ts) | unit + source | `go test -race ./internal/daemon/ -run 'TestExchangeJoinCode_WR05'` | ✅ green |
| FSW-12 | 123-03 | 50 MiB cap via MaxBytesReader BEFORE ParseMultipartForm; over-cap → 413, not truncated | `internal/files/handler_test.go` · `TestHandlerUpload_OverCap413`, `TestHandlerWrite_OverCap413`, `TestHandler_Read_OverCapReturns413`; `internal/files/write_test.go` · `TestMaxUploadBytes_Is50MiB`, `TestUpload_IN05_MalformedMultipart_Returns400` | unit (HTTP) | `go test -race ./internal/files/ -run 'OverCap\|TestMaxUploadBytes_Is50MiB\|MalformedMultipart'` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] write-primitive unit tests (write, rename, delete, mkdir, upload) — delivered as `internal/files/write_test.go` + `internal/files/handler_test.go` (HTTP layer)
- [x] `FuzzSandboxWrite` harness mirroring `FuzzSandboxPath` — delivered in `internal/files/sandbox_test.go` (`FuzzSandboxWrite` at :449, directly after `FuzzSandboxPath`)
- [x] daemon write-route + client handler tests — delivered as `internal/daemon/api_test.go` (`TestFilesWriteRoutes_*`) + `internal/daemon/client_test.go` (`TestDaemonClient*_RoundTrip`, `TestExchangeJoinCode_*`)

*Framework already present (go test) — no install needed. Filenames differ from the planner placeholders (`sandbox_write_test.go`/`fuzz_write_test.go`/`*_write_test.go`): the delivered tests live in the package's existing `write_test.go`, `handler_test.go`, `sandbox_test.go`, `api_test.go`, and `client_test.go` — confirmed present and green.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| curl PUT over real `~/.agenthub/daemon.sock` | FSW (daemon route) | requires a live daemon + real session | Start daemon, create session, run the success-criterion curl, confirm 200 + read-back |

*Most behaviors have automated verification; the live-socket curl is the documented manual smoke.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated (retroactive audit 2026-06-14)

---

## Validation Audit 2026-06-14

Retroactive, audit-only Nyquist validation of an archived, already-shipped phase. The full Go test suite is confirmed green; targeted `-count=1` spot-checks were re-run for the FSW-critical tests (denylist, traversal, atomic write, HTTP cap/sanitize 413, daemon routes/clients, ExchangeJoinCode) and `FuzzSandboxWrite` was re-run (6s spot-check, 0 crashes — the 60s gate ran live during original verification).

| Metric | Count |
|--------|-------|
| Requirements audited | 12 (FSW-01..12) |
| Covered by automated tests | 12 |
| Manual-only (live socket curl) | 1 (already executed & PASSED 2026-06-14 per VERIFICATION.md) |
| Gaps found | 0 |
| Gaps resolved | 0 |
| Gaps escalated | 0 |

**Note on test filenames:** VERIFICATION.md and the original Wave 0 plan referenced placeholder filenames (`sandbox_write_test.go`, `fuzz_write_test.go`, `*_write_test.go`). The delivered tests live in the packages' existing files (`internal/files/write_test.go`, `internal/files/handler_test.go`, `internal/files/sandbox_test.go`, `internal/daemon/api_test.go`, `internal/daemon/client_test.go`). Every named test function in VERIFICATION.md was located and confirmed present and green — no fabrication, no missing coverage. The HTTP-layer FSW-05/06/12 tests (`TestHandlerUpload_FilenameSanitized`, `TestHandlerUpload_DenylistForbidden`, `TestHandlerUpload_OverCap413`, `TestHandlerWrite_OverCap413`) are in `handler_test.go`, not `write_test.go`.

All 12 FSW requirements are COVERED by passing automated tests. The single manual-only item (live daemon-socket PUT curl) was already executed and passed during human verification. `nyquist_compliant` flipped to `true` on real evidence.
