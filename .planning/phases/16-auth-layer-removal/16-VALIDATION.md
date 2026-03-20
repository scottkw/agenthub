---
phase: 16
slug: auth-layer-removal
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 16 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) + Vitest |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/webserver/... -race` |
| **Full suite command** | `go test ./... -race && cd frontend && pnpm test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/... -race`
- **After every plan wave:** Run `go test ./... -race && cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 16-01-01 | 01 | 1 | AUTH-01 | unit | `go test ./internal/webserver/... -run TestWebServerDashboardNoAuthRequired -race` | ✅ (needs update) | ⬜ pending |
| 16-01-02 | 01 | 1 | AUTH-01 | unit | `go test ./internal/webserver/... -run TestWebServerSessionListAPI -race` | ✅ (needs update) | ⬜ pending |
| 16-01-03 | 01 | 1 | AUTH-01 | unit | `go test ./internal/webserver/... -run TestLoginRouteNotRegistered` | ❌ W0 | ⬜ pending |
| 16-01-04 | 01 | 1 | AUTH-02 | unit | `go test ./internal/webserver/... -run TestTokenRouteNotRegistered` | ❌ W0 | ⬜ pending |
| 16-01-05 | 01 | 1 | AUTH-03 | unit | `go test ./internal/webserver/... -run TestSessionAccessWithoutAuth -race` | ❌ W0 | ⬜ pending |
| 16-01-06 | 01 | 1 | AUTH-03 | unit | `go test ./internal/webserver/... -run TestWebServerToggle -race` | ✅ | ⬜ pending |
| 16-01-07 | 01 | 1 | AUTH-03 | unit | `go test . -run TestStartWebServerNoPasswordRequired -race` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/server_test.go` — rewrite `testServer()` to not call `SetPassword`; delete `login()` helper; rewrite auth-dependent tests as open-access tests
- [ ] `internal/webserver/server_test.go` — add `TestLoginRouteNotRegistered`, `TestTokenRouteNotRegistered`, `TestSessionAccessWithoutAuth`
- [ ] `app_test.go` — delete `TestStartWebServerErrorsWhenPasswordNotSet`, `TestSetWebPasswordPersistsAndReloads`; add `TestStartWebServerNoPasswordRequired`
- [ ] `frontend/src/components/__tests__/SettingsPanel.test.tsx` — delete `Security tab shows CA certificate path` (already failing), delete Security tab test cases, update mock to remove `SetWebPassword`/`IsWebPasswordSet`
- [ ] `frontend/src/components/__tests__/StatusBar.test.tsx` — remove `onCopyTokenLink` prop references if prop is removed from `StatusBarProps`

*(Pre-existing failure: `SettingsPanel.test.tsx > Security tab shows CA certificate path` — 1 test failing before Phase 16 starts. This test must be deleted in Phase 16.)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dashboard loads in browser without login prompt | AUTH-01, AUTH-03 | Visual browser verification | Open dashboard URL in browser, confirm no login form appears |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
