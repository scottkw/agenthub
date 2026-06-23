---
phase: 149
slug: google-antigravity-agent
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-22
validated: 2026-06-23
---

# Phase 149 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib) + vitest (frontend) |
| **Config file** | `frontend/vitest.config.ts` (TS); none needed for Go |
| **Quick run command** | `go test ./internal/pty/... && cd frontend && pnpm test --run agentBadge` |
| **Full suite command** | `go test ./... && cd frontend && pnpm test --run && tsc --noEmit && vite build` |
| **Estimated runtime** | ~60 seconds (quick ~10s, full ~60s) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/pty/ && cd frontend && pnpm test --run agentBadge`
- **After every plan wave:** Run `go test ./... && cd frontend && pnpm test --run && tsc --noEmit`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Test File | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-----------|--------|
| 149-01-T1 | 149-01 | 1 | AGENT-01 | — | `knownCLIs` contains `agy` entry (5 entries) | unit (Go) | `go test ./internal/pty/ -run TestKnownCLIs_HasExpectedEntries` | `internal/pty/detect_test.go` | ✅ green |
| 149-01-T1 | 149-01 | 1 | AGENT-01 | — | DetectCLIs finds `agy` stub on PATH | unit (Go) | `go test ./internal/pty/ -run TestDetectCLIs_FindsAgy` | `internal/pty/detect_test.go` | ✅ green |
| 149-01-T1 | 149-01 | 1 | AGENT-01 | — | DetectCLI("agy") returns ErrCLINotFound when absent | unit (Go) | `go test ./internal/pty/ -run TestDetectCLI_AgyNotFound` | `internal/pty/detect_test.go` | ✅ green |
| 149-01-T2 | 149-01 | 1 | AGENT-01 | — | Windows PATH includes `%LOCALAPPDATA%\agy\bin` | unit (Go, Win matrix) | CI Windows matrix `go test ./internal/daemon/ -run TestPlatformExtraBins`; host: `GOOS=windows go build ./internal/daemon/` exits 0 | `internal/daemon/path_windows_test.go` | ✅ CI-matrix (cross-compiles clean) |
| 149-01-T3 | 149-01 | 1 | AGENT-01 | — | agy session classifies idle at line-anchored `>` prompt (not perma-running), waiting at `[y/n]` | unit (Go) | `go test ./internal/status/ -run 'Agy\|PatternsForCLI'` | `internal/status/detector_test.go` | ✅ green |
| 149-02-T1 | 149-02 | 1 | AGENT-01 | — | `agentBadgeModifier('agy') === 'agy'` | unit (TS) | `pnpm test --run agentBadge` | `frontend/src/lib/agentBadge.test.ts` | ✅ green |
| 149-02-T2 | 149-02 | 1 | AGENT-01 | — | `.tab__agent-badge--agy` + `[data-agent="agy"]` spine/chip CSS sites present in lockstep | unit (TS) | `pnpm test --run style.hub` | `frontend/src/components/__tests__/style.hub.test.ts` | ✅ green |
| 149-02-T2 | 149-02 | 1 | AGENT-01 | — | `#ff9e64` WCAG contrast documented at source (dark 8.72:1 pass; light 2.03:1 fail noted) | source assertion (TS) | `pnpm test --run style.hub` | `frontend/src/components/__tests__/style.hub.test.ts` | ✅ green |
| M-15 | 149-03 | — | AGENT-01 | — | Live `agy` REPL launch (waitlist-gated) | manual | TESTING.md Section 5 Category I | n/a | 🔒 manual-only |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · 🔒 manual-only*

---

## Wave 0 Requirements

- [x] `internal/pty/detect_test.go` — `TestKnownCLIs_HasExpectedEntries` (5 entries inc. `"agy"`), `TestDetectCLIs_FindsAgy`, `TestDetectCLI_AgyNotFound`
- [x] `frontend/src/lib/agentBadge.test.ts` — `returns 'agy' for cli='agy'` test (14/14 pass)
- [x] `frontend/src/components/__tests__/style.hub.test.ts` — 7 agy assertions across the three data-agent CSS sites + WCAG comment gate
- [x] `internal/daemon/path_windows_test.go` — `TestPlatformExtraBins_WindowsIncludesAgyBin` asserts `platformExtraBins` includes the `agy\bin` path
- [x] `internal/status/detector_test.go` — `TestDetector_AgyIdle`, `TestDetector_AgyIdleNotBroadAngleBracket`, `TestDetector_AgyWaiting`, `TestPatternsForCLI_AgyNotFallback` (added during audit reconciliation — was absent from the original draft map despite covering VERIFICATION truth #4)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live Antigravity (`agy`) REPL launch | AGENT-01 | Closed-beta/waitlist (D-03) — binary cannot be installed for live UAT this phase | M-15 in TESTING.md §5: when waitlist access is granted, run `agenthub new antigravity`, confirm `agy` launches an interactive PTY REPL, auth completes via browser-loopback/OTP, and the status badge renders `#ff9e64` |

*Per D-03, live launch is the only behavior that falls to manual; all source-level behaviors are automated above.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-23 (retroactive audit)

---

## Validation Audit 2026-06-23

Retroactive audit of the pre-execution draft against the executed phase and `149-VERIFICATION.md` (9/9 truths verified, 2026-06-22). All requirement behaviors carry an automated test that runs green; the Windows-PATH row is covered by the CI Windows matrix and cross-compiles clean on the host. No test code was generated — every gap was a stale-draft documentation gap (placeholder task IDs, an unlisted detector row, `nyquist_compliant: false`).

| Metric | Count |
|--------|-------|
| Gaps found | 3 |
| Resolved | 3 |
| Escalated | 0 |

**Gaps resolved (documentation reconciliation, no test generation):**
1. Placeholder task IDs (`149-XX-XX` / `TBD`) resolved to real plan/task IDs (149-01-T1…149-02-T2) and concrete test-file paths.
2. Added the missing status-detector row (149-01-T3 → `detector_test.go`) — it covered VERIFICATION truth #4 but was absent from the original draft map.
3. Flipped frontmatter to `status: validated`, `nyquist_compliant: true`, `wave_0_complete: true`; checked Wave 0 + sign-off.

**Verification commands re-run during audit (all green):**
- `go test ./internal/pty/ -run 'TestKnownCLIs|TestDetectCLI'` → ok
- `go test ./internal/status/ -run 'Agy|PatternsForCLI'` → ok
- `GOOS=windows go vet ./internal/daemon/` + `GOOS=windows go build ./internal/daemon/` → exit 0
- `pnpm exec vitest run agentBadge` → 14/14 · `pnpm exec vitest run style.hub` → 100/100
