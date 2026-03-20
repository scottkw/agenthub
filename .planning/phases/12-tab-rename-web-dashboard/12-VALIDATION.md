---
phase: 12
slug: tab-rename-web-dashboard
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-20
audited: 2026-03-20
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
| 12-01-01 | 01 | 1 | UILAY-05, WEBUI-02 | unit (Go) | `go test ./internal/webserver/... -run TestWebServerSessionListAPI` | ✅ | ✅ green |
| 12-01-02 | 01 | 1 | UILAY-05, WEBUI-02 | unit (Go) | `go test ./internal/webserver/... -run TestWebServerSessionListAPIWithResolver` | ✅ | ✅ green |
| 12-02-01 | 02 | 1 | UILAY-04 | unit (DOM) | `cd frontend && pnpm test` | ✅ | ✅ green |
| 12-03-01 | 03 | 2 | WEBUI-01 | manual | n/a | n/a | ✅ manual-verified |
| 12-03-02 | 03 | 2 | WEBUI-02 | unit (Go) | `go test ./internal/webserver/... -run TestWebServerSessionListAPIWithResolver` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `frontend/src/components/__tests__/TabBar.test.tsx` — 4 context menu tests for UILAY-04 (right-click shows menu, Rename menuitem, clicking Rename starts editing, title includes right-click)
- [x] `internal/webserver/server_test.go` — `TestWebServerSessionListAPI` decodes `[]struct{ID,Name string}`, `TestWebServerSessionListAPIWithResolver` validates name/cli_type/status

*All Wave 0 requirements satisfied during execution.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dashboard visual redesign with status dots and CLI badges | WEBUI-01 | CSS visual styling cannot be verified by unit tests | Open browser to `/dashboard`, confirm status color dots and CLI badge pills render next to session names |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-03-20

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All requirements verified against actual test files and test run output. Frontend: 85/85 tests green. Backend: all webserver tests green.
