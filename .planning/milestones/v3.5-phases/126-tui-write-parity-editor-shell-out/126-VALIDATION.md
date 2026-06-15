---
phase: 126
slug: tui-write-parity-editor-shell-out
status: draft
nyquist_compliant: false
wave_0_complete: false
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

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner fills) | — | 1 | TUIW-01 | — | FilesClient 8 methods; both impls satisfy | compile/unit | `go build ./... ` + `go test ./internal/tui/...` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 2 | TUIW-02/03/04 | — | $EDITOR shell-out via tea.ExecProcess; no-editor inline error | unit | `go test ./internal/tui/...` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 3 | TUIW-05 | — | d/r/m affordances refresh listing | unit | `go test ./internal/tui/...` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 4 | TUIW-06/07 | — | upload descope message; no-sync gate w/ writes | unit/grep | `go test ./internal/tui/ -run TestFiles_NoSyncFSCalls` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] FilesClient interface extension to 8 methods (matching the actual DaemonClient response-struct signatures)
- [ ] RemoteFilesClient write-method implementations + httptest.TLSServer tests
- [ ] $EDITOR resolution chain test ($EDITOR→$VISUAL→nano→vim→vi; no-editor error)
- [ ] d/r/m command tests (confirm dialog, inline rename/mkdir, listing refresh)
- [ ] TestFiles_NoSyncFSCalls extended to cover write commands

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

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
