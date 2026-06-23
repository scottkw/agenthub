---
phase: 149
slug: google-antigravity-agent
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-22
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

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 149-XX-XX | TBD | 1 | AGENT-01 | — | `knownCLIs` contains `agy` entry | unit (Go) | `go test ./internal/pty/ -run TestKnownCLIs` | ✅ extend | ⬜ pending |
| 149-XX-XX | TBD | 1 | AGENT-01 | — | DetectCLIs finds `agy` stub on PATH | unit (Go) | `go test ./internal/pty/ -run TestDetectCLIs` | ✅ extend | ⬜ pending |
| 149-XX-XX | TBD | 1 | AGENT-01 | — | stored-path override + stale-path filter for `agy` | unit (Go) | `go test ./internal/pty/ -run TestDetectCLI` | ✅ extend | ⬜ pending |
| 149-XX-XX | TBD | 1 | AGENT-01 | — | Windows PATH includes `%LOCALAPPDATA%\agy\bin` | unit (Go, Win matrix) | CI Windows matrix `go test ./internal/daemon/...` | ❌ W0 | ⬜ pending |
| 149-XX-XX | TBD | 1 | AGENT-01 | — | `agentBadgeModifier('agy') === 'agy'` | unit (TS) | `pnpm test --run agentBadge` | ✅ extend | ⬜ pending |
| 149-XX-XX | TBD | 1 | AGENT-01 | — | `.tab__agent-badge--agy` + `[data-agent="agy"]` CSS sites present | unit (TS) | `pnpm test --run style.hub` | ✅ extend | ⬜ pending |
| 149-XX-XX | TBD | 1 | AGENT-01 | — | `#ff9e64` WCAG-AA contrast documented at source (dark 8.72:1 pass; light fail noted) | source assertion | grep CSS comment + computed ratio | ✅ extend | ⬜ pending |
| M-15 | TBD | — | AGENT-01 | — | Live `agy` REPL launch (waitlist-gated) | manual | TESTING.md Section 5 checklist | n/a | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/pty/detect_test.go` — extend `TestKnownCLIs_HasExpectedEntries` to include `"agy"`
- [ ] `frontend/src/lib/agentBadge.test.ts` — add `returns 'agy' for cli='agy'` test
- [ ] `frontend/src/components/__tests__/style.hub.test.ts` — add `"agy"` to the data-agent presence assertions
- [ ] `internal/daemon/path_windows_test.go` — add test that `platformExtraBins` includes the `agy\bin` path

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live Antigravity (`agy`) REPL launch | AGENT-01 | Closed-beta/waitlist (D-03) — binary cannot be installed for live UAT this phase | M-15 in TESTING.md §5: when waitlist access is granted, run `agenthub new antigravity`, confirm `agy` launches an interactive PTY REPL, auth completes via browser-loopback/OTP, and the status badge renders `#ff9e64` |

*Per D-03, live launch is the only behavior that falls to manual; all source-level behaviors are automated above.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
