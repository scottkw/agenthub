---
phase: 145
slug: windows-files-test-fixes
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-22
---

# Phase 145 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | none — invoked via `go test` |
| **Quick run command** | `go test -race -short -run "TestHandlerUpload_FilenameSanitized\|TestDenylist_NonHomeRootedUnaffected\|TestWriteFileAtomic_ConcurrentReadNeverPartial" ./internal/files/` |
| **Full suite command** | `go test -race -short ./internal/files/` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd:verify-work`:** Full suite must be green locally AND Windows CI matrix job green
- **Max feedback latency:** ~15 seconds (local); CI roundtrip for Windows ground truth

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 145-01-* | 01 | 1 | FIX-02 | T-145-01 | Upload of `../../.bashrc` sanitizes to sandbox root; no escape | unit | `go test -race -short -run TestHandlerUpload_FilenameSanitized ./internal/files/` | ✅ | ⬜ pending |
| 145-01-* | 01 | 1 | FIX-02 | T-145-01 | Non-home-rooted sandbox allows `.bashrc` write | unit | `go test -race -short -run TestDenylist_NonHomeRootedUnaffected ./internal/files/` | ✅ | ⬜ pending |
| 145-02-* | 02 | 1 | FIX-02 | T-145-02 | Concurrent read never observes empty/partial/mixed file during atomic write on Windows | unit + race | `go test -race -short -run TestWriteFileAtomic_ConcurrentReadNeverPartial ./internal/files/` | ❌ W0 (helper) | ⬜ pending |
| 145-* | * | 1 | FIX-02 | — | Full `internal/files` suite green with race detector, no regression on macOS/Linux | unit + race | `go test -race -short ./internal/files/` | ✅ | ⬜ pending |
| 145-* | * | 1 | FIX-02 | — | Windows test files cross-compile | build | `GOOS=windows GOARCH=amd64 go test -c ./internal/files/ -o /dev/null` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/files/concurrent_read_windows_test.go` — Windows build-tagged `readFilePlatformSafe` helper opening with `FILE_SHARE_DELETE` (covers FIX-02)
- [ ] `internal/files/concurrent_read_unix_test.go` — non-Windows companion delegating to `os.ReadFile` (covers FIX-02)

*Existing test files (`handler_test.go`, `write_test.go`) are modified in place — not moved or renamed. The two helper files are the only new files.*

> **Executor note:** After the two `concurrent_read_*_test.go` helper files are created and cross-vet/compile clean (end of plan 145-02), flip this file's frontmatter to `wave_0_complete: true`, and set `nyquist_compliant: true` once the full `internal/files` suite is green locally. The phase is only TRULY verified once the Windows CI matrix job is green (manual checkpoint, plan 145-03).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Three target tests pass on `windows/amd64, windows-latest` | FIX-02 | No local Windows environment (dev is macOS); CI runner is the only Windows ground truth | Push branch; confirm `build (agenthub, windows/amd64, windows-latest)` job is green in GitHub Actions |

*Local cross-compile (`GOOS=windows go vet` / `go test -c`) gives compile confidence but does NOT prove Windows runtime correctness.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (the two build-tagged helper files)
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s (local)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
