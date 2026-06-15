---
phase: 126
slug: tui-write-parity-editor-shell-out
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-14
---

# Phase 126 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (internal/tui + internal/daemon) incl. the TestFiles_NoSyncFSCalls static-grep gate and httptest.TLSServer for RemoteFilesClient |
| **Config file** | none (Go native) |
| **Quick run command** | `go test ./internal/tui/... ./internal/daemon/...` |
| **Full suite command** | `go test -race ./internal/tui/... ./internal/daemon/...` |
| **Estimated runtime** | ~30-60 seconds |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/tui/...`
- **After every plan wave:** `go test -race ./internal/tui/... ./internal/daemon/...`
- **Before `/gsd:verify-work`:** full suite green; `TestFiles_NoSyncFSCalls` passes with write commands included; `FilesClient` interface-satisfaction compile-checks for both implementers
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Secure Behavior | Test Type | Automated Command | File / Function | Status |
|---------|------|------|-------------|-----------------|-----------|-------------------|-----------------|--------|
| TUIW-01 | 126-01 | 1 | TUIW-01 | FilesClient = exactly 8 methods (no UploadFile); `*daemon.DaemonClient` + `*RemoteFilesClient` both satisfy via compile-time `var _` guards; 4 write methods round-trip | compile/unit | `go test ./internal/tui/ -run 'TestRemoteFilesClient_(SatisfiesInterface\|Write\|Delete\|Rename\|Mkdir\|WriteCapLeak)' -count=1` | `files_client.go:45`, `remote_files_client.go:64`; `remote_files_client_test.go` TestRemoteFilesClient_SatisfiesInterface/Write/Delete/Rename/Mkdir/WriteCapLeak | ✅ green |
| TUIW-02 | 126-02 | 2 | TUIW-02 | `e` dispatches editFetchCmd via tea.Cmd (generation bump); dir = no-op; write-back never synchronous in Update | unit | `go test ./internal/tui/ -run TestHandleFilesKey_Edit -count=1` | `files_edit_test.go:97` TestHandleFilesKey_Edit | ✅ green |
| TUIW-03 | 126-02 | 2 | TUIW-03 | resolveEditor chain $EDITOR→$VISUAL→nano→vim→vi; no-editor sets EXACT locked error + nil cmd | unit | `go test ./internal/tui/ -run 'TestResolveEditor\|TestHandleFilesKey_Edit' -count=1` | `files_edit_test.go:24` TestResolveEditor; `:145` exact-error sub-test | ✅ green |
| TUIW-04 | 126-02 | 2 | TUIW-04 | editorExitMsg batches tea.ClearScreen + write-back UNCONDITIONALLY (incl. exitErr!=nil); CR-01 staleness-guard regression (write-back error surfaces) | unit | `go test ./internal/tui/ -run 'TestEditorExit_RefreshesUnconditionally\|TestEditWriteBack_ErrorSurfaces' -count=1` | `files_edit_test.go:183` TestEditorExit_RefreshesUnconditionally; `:324` TestEditWriteBack_ErrorSurfaces | ✅ green |
| TUIW-05 | 126-03 | 3 | TUIW-05 | d delete-confirm modal (reuses kill-session pattern, colorblind-safe), r inline rename, m inline mkdir; modal keys dispatch at priority above tab-cycling; listing refresh on completion | unit | `go test ./internal/tui/ -run 'TestFilesOpCmd\|TestFilesDelete\|TestFilesRename\|TestFilesMkdir\|TestFilesNameInput_DispatchPriority' -count=1` | `files_ops_test.go` TestFilesOpCmd/Delete_ModalStateSet/Delete_ConfirmHandler/Delete_DispatchPriority/Rename/Mkdir/DeleteModal_ColorblindSafeText/NameInput_DispatchPriority | ✅ green |
| TUIW-06 | 126-04 | 4 | TUIW-06 (descoped) | `u` → verbatim "Use desktop or web to upload files." + nil cmd (no write); the one intentional parity gap, follow-up GitHub issue #82 filed | unit | `go test ./internal/tui/ -run TestFilesUpload_Descoped -count=1` | `files_test.go:889` TestFilesUpload_Descoped | ✅ green |
| TUIW-07 | 126-04 | 4 | TUIW-07 | TestFiles_NoSyncFSCalls broadened regex (ReadDir\|Open\|OpenFile\|Stat\|Create\|Remove\|ReadFile\|WriteFile) over files.go + files_cmds.go — all write FS I/O routes through tea.Cmd | unit/grep | `go test ./internal/tui/ -run 'TestFiles_NoSyncFSCalls\|TestLoadDirCmd_DispatchesAsync' -count=1` | `files_test.go:850` TestFiles_NoSyncFSCalls; `:174` TestLoadDirCmd_DispatchesAsync | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> **Traceability roll-up:** `TestFiles_Phase126_Requirements` (`files_test.go:1190`) maps all 7 TUIW IDs → named tests; passes with 7 green sub-rows.

---

## Wave 0 Requirements

- [x] FilesClient interface extension to 8 methods (matching the actual DaemonClient response-struct signatures)
- [x] RemoteFilesClient write-method implementations + httptest.TLSServer tests
- [x] $EDITOR resolution chain test ($EDITOR→$VISUAL→nano→vim→vi; no-editor error)
- [x] d/r/m command tests (confirm dialog, inline rename/mkdir, listing refresh)
- [x] TestFiles_NoSyncFSCalls extended to cover write commands

*Framework present (go test) — no install.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `e` suspends TUI → real $EDITOR → resume + clean terminal | TUIW-02 | requires a real terminal + interactive editor; tea.ExecProcess suspend/resume not unit-drivable | Run the TUI, press `e` on a file, edit in $EDITOR, save+exit, confirm clean redraw + refreshed listing |
| Two-machine remote write edit | TUIW-01 | requires two hosts | Deferred to Phase 128 (its stated gate) |

*Most behavior is unit-testable (command dispatch, interface satisfaction, editor resolution); the live suspend-resume terminal restore is the manual residue.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated (audit 2026-06-14)

---

## Validation Audit 2026-06-14

Retroactive audit-only Nyquist validation of archived, shipped Phase 126. Full Go suite is GREEN; spot-checks re-run read-only with `-count=1`.

**Counts:** 7 requirements total — 6 COVERED + 1 DESCOPED (TUIW-06, upload → GitHub issue #82) · 0 PARTIAL · 0 MISSING.

**Gaps:** none.
**Resolved:** all 7 TUIW rows map to existing, passing automated tests; the requirements-matrix roll-up `TestFiles_Phase126_Requirements` confirms 7/7 mapped and green.
**Escalated:** none.

**Verdict:** COMPLIANT — `nyquist_compliant: true` set.

**Notes:**
- Coverage is genuinely fail-capable, not soft: TUIW-03 asserts the EXACT locked no-editor error string; TUIW-04 asserts unconditional write-back on the exitErr!=nil path plus the CR-01 staleness-guard regression (`TestEditWriteBack_ErrorSurfaces`); TUIW-07 broadens the no-sync grep gate to the write verbs (Create/Remove/ReadFile/WriteFile) over files.go + files_cmds.go.
- **Manual-Only (operator-deferred, non-blocking):** live `$EDITOR` suspend-resume terminal restore (tea.ExecProcess requires a real interactive TTY; logic/wiring is fully covered by `TestHandleFilesKey_Edit` + `TestEditorExit_RefreshesUnconditionally`), and the two-machine remote write edit (deferred to Phase 128 as its stated gate; RemoteFilesClient write methods are unit-covered via httptest.TLSServer). Neither blocks compliance.
- **Descoped:** TUI upload (TUIW-06) is the one sanctioned cross-surface parity gap, documented on-screen and tracked in GitHub issue #82 — counted as descoped, not a missing gap.
