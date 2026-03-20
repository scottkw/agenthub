---
phase: 12
slug: tab-rename-web-dashboard
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.x (frontend), Go test (backend) |
| **Config file** | `frontend/vite.config.ts` (frontend), Go stdlib (backend) |
| **Quick run command** | `cd frontend && pnpm test` + `go test ./internal/webserver/... -count=1` |
| **Full suite command** | `cd frontend && pnpm test` + `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test` + `go test ./internal/webserver/... -count=1`
- **After every plan wave:** Run `cd frontend && pnpm test` + `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 12-01-01 | 01 | 1 | UILAY-04 | unit (DOM) | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |
| 12-01-02 | 01 | 1 | UILAY-04 | unit (DOM) | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |
| 12-02-01 | 02 | 1 | UILAY-05, WEBUI-02 | unit (Go) | `go test ./internal/webserver/... -run TestWebServerSessionListAPI` | ✅ (needs update) | ⬜ pending |
| 12-03-01 | 03 | 2 | WEBUI-01 | manual | n/a | n/a | ⬜ pending |
| 12-03-02 | 03 | 2 | WEBUI-02 | unit (Go) | `go test ./internal/webserver/... -run TestWebServerSessionListAPI` | ✅ (needs update) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — add right-click context menu + double-click rename tests for UILAY-04
- [ ] `internal/webserver/server_test.go` — update `TestWebServerSessionListAPI` to decode `[]struct{ID,Name,CLIType string}` instead of `[]string`

*Existing test infrastructure covers all other phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dashboard visual redesign with status dots and CLI badges | WEBUI-01 | CSS visual styling cannot be verified by unit tests | Open browser to `/dashboard`, confirm status color dots and CLI badge pills render next to session names |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
