---
phase: 53
slug: remote-sessions-cli
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-07
---

# Phase 53 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library) |
| **Config file** | none — Go built-in test runner |
| **Quick run command** | `go test -run "TestParseRemoteID\|TestResolveRemotePeer\|TestFetchPeerSessions\|TestCmdList\|TestCmdAttach\|TestUsage_RemoteSessionDocs" -v -count=1 -race .` |
| **Full suite command** | `go test -v -count=1 -race .` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -run "TestParseRemoteID\|TestResolveRemotePeer\|TestFetchPeerSessions\|TestCmdList\|TestCmdAttach" -v -count=1 -race .`
- **After every plan wave:** Run `go test -v -count=1 -race .`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 2 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 53-01-01 | 01 | 1 | REM-04 | unit | `go test -run "TestParseRemoteID\|TestResolveRemotePeer\|TestFetchPeerSessions" -v -count=1 -race .` | ✅ cmd_remote_test.go | ✅ green |
| 53-01-02 | 01 | 1 | REM-04 | unit+integration | `go test -run "TestCmdList" -v -count=1 -race .` | ✅ cmd_cli_test.go | ✅ green |
| 53-02-01 | 02 | 2 | REM-05 | unit | `go test -run "TestCmdAttach\|TestPrintAttachBanner" -v -count=1 -race .` | ✅ cmd_attach_test.go | ✅ green |
| 53-02-02 | 02 | 2 | REM-05 | unit | `go test -run "TestUsage_RemoteSessionDocs" -v -count=1 .` | ✅ cmd_cli_test.go | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Remote session list display with real tailnet peers | REM-04 | Requires real tailnet peers with running daemons | Run `agenthub list` on a machine with tailnet peers running agenthub daemons |
| Remote session attach via WSS | REM-05 | Requires real tailnet connectivity and running remote daemon | Run `agenthub attach macbook:session-id` using a real session ID |
| Remote attach banner clarity | REM-05 | Visual clarity assessment | After attaching to a remote session, verify the banner clearly indicates the remote hostname |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 2s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-07

## Validation Audit 2026-04-07

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 20+ automated tests pass with race detector. go vet clean. Build succeeds.
