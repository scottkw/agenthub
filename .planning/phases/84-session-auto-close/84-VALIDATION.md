---
phase: 84
slug: session-auto-close
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-19
---

# Phase 84 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest |
| **Config file** | `vitest.config.ts` (frontend), native (Go) |
| **Quick run command** | `go test ./internal/... -run TestAutoClose -count=1` |
| **Full suite command** | `go test ./internal/... -count=1 && cd frontend && npx vitest run` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -run TestAutoClose -count=1`
- **After every plan wave:** Run `go test ./internal/... -count=1 && cd frontend && npx vitest run`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 84-01-01 | 01 | 1 | SESS-01 | — | N/A | unit | `go test ./internal/pty/ -run TestExitDetection -count=1` | ❌ W0 | ⬜ pending |
| 84-01-02 | 01 | 1 | SESS-01 | — | N/A | unit | `go test ./internal/daemon/ -run TestExitEvent -count=1` | ❌ W0 | ⬜ pending |
| 84-02-01 | 02 | 2 | SESS-02 | — | N/A | unit | `cd frontend && npx vitest run --reporter=verbose 2>&1 \| grep -i countdown` | ❌ W0 | ⬜ pending |
| 84-02-02 | 02 | 2 | SESS-03 | — | N/A | unit | `cd frontend && npx vitest run --reporter=verbose 2>&1 \| grep -i toast` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/pty/session_exit_test.go` — stubs for exit detection tests (SESS-01)
- [ ] `internal/daemon/engine_exit_test.go` — stubs for exit event emission tests (SESS-01)
- [ ] `frontend/src/components/__tests__/ExitCountdownBanner.test.tsx` — stubs for countdown UI (SESS-02)
- [ ] `frontend/src/components/__tests__/ExitToast.test.tsx` — stubs for toast notification (SESS-03)

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tab fade-out animation plays smoothly | SESS-02 | Visual animation quality cannot be automated | 1. Start agent, let it exit naturally 2. Observe tab dims/fades over countdown 3. Verify no flicker or jump |
| Toast visible from non-active tab | SESS-03 | Cross-tab visibility requires visual confirmation | 1. Start agent in Tab A 2. Switch to Tab B 3. Let agent in Tab A exit 4. Verify toast appears while on Tab B |
| Terminal remains scrollable during countdown | SESS-02 | Interactive scrolling requires manual verification | 1. Let agent exit with long output 2. During 5s countdown, scroll up/down in terminal 3. Verify smooth scrolling |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
