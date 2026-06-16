---
phase: 129
slug: write-concurrency-fix-dns-error-ux
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-15
---

# Phase 129 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60s full suite; race test ~5s |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100` (or the package touched)
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| W0 | — | 0 | DNS-01/02 | — | DNS message only on MagicDNS failure | unit | `go test ./internal/daemon/... -run TestProxyRemoteFiles_AcceptDNSMessage` | ❌ W0 | ⬜ pending |
| W0 | — | 0 | DNS-03 | — | AcceptDNS populated when connected | unit | `go test ./internal/webserver/... -run TestCheckHealth_AcceptDNS` | ❌ W0 | ⬜ pending |
| W0 | — | 0 | RACE-01 (relay) | — | Two-writer race serialized on relay surface | unit | `go test ./internal/daemon/... -run TestRemoteFiles_TwoWriterRace_RelaySurface` | ❌ W0 | ⬜ pending |
| — | — | 1 | RACE-01 | — | Single-winner: exactly one writer succeeds | unit | `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100` | ✅ `internal/files/write_test.go:1153` | ⬜ pending |
| — | — | 1 | RACE-02 | — | No last-writer-wins language in proxy | static/grep | `grep -c "last-writer-wins" internal/daemon/remote_files.go` (expect 0) | ✅ | ⬜ pending |
| — | — | 1 | RACE-03 | — | No interleaved content, no leftover temp files | unit | `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/remote_files_test.go` — `TestProxyRemoteFiles_AcceptDNSMessage`: covers DNS-01 + DNS-02 (specific actionable message on DNS failure for `.ts.net` MagicDNS hostname; NOT triggered for connection-refused or TLS errors)
- [ ] `internal/webserver/tailscale_test.go` — `TestCheckHealth_AcceptDNS`: covers DNS-03 (`AcceptDNS` field populated when `Connected` and `GetPrefs` returns `CorpDNS=false`)
- [ ] `internal/daemon/relay_remote_files_test.go` — `TestRemoteFiles_TwoWriterRace_RelaySurface`: relay-surface coverage for RACE-01 (two concurrent PUT writes through the relay to the same path; assert single-winner) — **mandatory relay-surface coverage per REQUIREMENTS.md**

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| DNS-03 proactive warning banner on-screen | DNS-03 (UX surface) | Live app + real tailnet with `accept-dns=false` peer required | Set `accept-dns=false` on a client; open Remote Sessions; confirm banner names the fix before the failure path |

*Automated coverage exists for the daemon/health detection of all three DNS requirements; only the on-screen banner render is manual.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
