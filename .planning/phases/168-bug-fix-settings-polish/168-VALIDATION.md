---
phase: 168
slug: bug-fix-settings-polish
status: ready
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-01
---

# Phase 168 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (backend) / vitest (frontend) |
| **Config file** | `frontend/vitest.config.ts` (frontend); none for Go |
| **Quick run command** | `cd frontend && pnpm vitest run <changed-spec>` |
| **Full suite command** | `go test ./... && cd frontend && pnpm vitest run && pnpm tsc --noEmit` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `{quick run command}`
- **After every plan wave:** Run `{full suite command}`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 168-01-01 | 01 | 1 | FIX-04 | T-168-01 | N/A (read-only count filter) | unit (Go) | `go test ./internal/relay/... -run TestRemoteViewerCount -v` | ❌ W0 (new test in-task) | ⬜ pending |
| 168-01-02 | 01 | 1 | FIX-04 | T-168-01 | Local-only session reads 0 viewers | unit (Go) | `go test ./internal/daemon/... -run TestListSessions_ViewerCount -v` | ❌ W0 (new test in-task) | ⬜ pending |
| 168-02-01 | 02 | 1 | FIX-01 | — | N/A (prop plumbing) | typecheck | `cd frontend && pnpm exec tsc --noEmit` | ✅ | ⬜ pending |
| 168-02-02 | 02 | 1 | FIX-01 | T-168-02 / T-168-03 | Cap in ?cap= (accepted); no CSP change | unit (vitest) | `cd frontend && pnpm vitest run WebShareSessionView.plugin-config` | ❌ W0 (new test in-task) | ⬜ pending |
| 168-03-01 | 03 | 2 | FIX-03 | T-168-04 | Per-tab cap/host isolation | typecheck | `cd frontend && pnpm exec tsc --noEmit` | ✅ | ⬜ pending |
| 168-03-02 | 03 | 2 | FIX-03 | T-168-04 | Two remotes → two isolated tabs | unit (vitest) | `cd frontend && pnpm vitest run App.open-remote` | ⚠️ extend existing | ⬜ pending |
| 168-04-01 | 04 | 3 | UX-01 | T-168-05 | Daemon-local settings RPC | unit (Go) | `go test ./internal/daemon/... -run StayOnHub -v` | ❌ W0 (new tests in-task) | ⬜ pending |
| 168-04-02 | 04 | 3 | UX-01 | T-168-05 | Default OFF, server-truth | unit (vitest) | `cd frontend && pnpm vitest run SettingsTab.stay-on-hub-toggle` | ❌ W0 (new test in-task) | ⬜ pending |
| 168-04-03 | 04 | 3 | UX-01 | — | Gate setActiveId only; tab still created | unit (vitest) | `cd frontend && pnpm vitest run App.createTab.stayOnHub` | ❌ W0 (new test in-task) | ⬜ pending |
| 168-05-01 | 05 | 4 | UX-02 | — | Single lifted modal instance (no clone) | typecheck | `cd frontend && pnpm exec tsc --noEmit` | ✅ | ⬜ pending |
| 168-05-02 | 05 | 4 | UX-02 | T-168-06 | Share button gated on tab TYPE (EoP) | unit (vitest) | `cd frontend && pnpm vitest run StatusBar.shareSession` | ⚠️ extend existing | ⬜ pending |
| 168-06-01 | 06 | 4 | FIX-02 | T-168-08 | Closes web-origin only; local untouched; no eviction | unit (Go, -race) | `go test ./internal/relay/... -run 'TestDisconnectWebViewers|TestHub_TwoWebOriginSubscribers_NoEviction' -race -v` | ❌ W0 (new tests in-task) | ⬜ pending |
| 168-06-02 | 06 | 4 | FIX-02 | T-168-07 | Disconnect RPC daemon-local only (EoP) | unit (Go) | `go test ./internal/daemon/... -run DisconnectViewers -v` | ❌ W0 (new test in-task) | ⬜ pending |
| 168-06-03 | 06 | 4 | FIX-02 | T-168-09 | Drops connections only; cap not revoked | unit (vitest) | `cd frontend && pnpm vitest run SessionShareModal.disconnect` | ❌ W0 (new test in-task) | ⬜ pending |
| 168-07-01 | 07 | 5 | FIX-01/02/03 | T-168-10 | Traceability path integrity | script | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |
| 168-07-02 | 07 | 5 | FIX-01/02/03 | T-168-10 | M-13 reflects shipped behavior | grep gate | `grep -q "in-app" TESTING.md && grep -Eq "two-(browser|viewer)|Disconnect all viewers" TESTING.md` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*(Planner fills this map from RESEARCH.md ## Validation Architecture during PLAN.md creation.)*

---

## Wave 0 Requirements

All new automated test files are created in-task by the code-producing task that owns them
(tdd="true" tasks write the test before/with the implementation), following exact in-repo
analog test files — no separate Wave 0 scaffolding plan is required:

- `internal/relay/hub_test.go` (extend) — TestRemoteViewerCount (P01), TestDisconnectWebViewers + TestHub_TwoWebOriginSubscribers_NoEviction (P06) — analog: TestHubTwoSubscribersBothReceive / TestHubSlowSubscriberGetsDisconnected
- `internal/daemon/engine_test.go` (extend) — TestListSessions_ViewerCount (P01)
- `internal/daemon/engine_stayonhub_test.go` + `api_stayonhub_test.go` (new, P04) — analog: engine_notify_test.go / api_notify_test.go
- `internal/daemon/api_disconnect_test.go` (new, P06)
- `frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx` (new, P02)
- `frontend/src/components/__tests__/App.open-remote.test.tsx` (extend, P03)
- `frontend/src/components/__tests__/App.createTab.stayOnHub.test.tsx` (new, P04)
- `frontend/src/components/__tests__/SettingsTab.stay-on-hub-toggle.test.tsx` (new, P04) — analog: SettingsTab.notify-toggle.test.tsx
- `frontend/src/components/__tests__/StatusBar.shareSession.test.tsx` (extend, P05) — analog: StatusBar.test.tsx
- `frontend/src/components/Hub/__tests__/SessionShareModal.disconnect.test.tsx` (new, P06)

Adaptation (not a passing gate): `frontend/e2e/web-plugin-hot-swap.spec.ts` targets a retired
UMD-global mechanism (RESEARCH Pitfall 5) — recorded as a reference/adaptation item in P07, NOT
blind-un-skipped.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live two-browser-viewer smoke: neither viewer is kicked; "Disconnect all viewers" drops both; Hub count returns to 0 within a poll cycle | FIX-02 | Requires two real browser clients on a live share link; A1 verdict ("no eviction in code") needs empirical confirmation | Open the same share URL in two real browsers; confirm both stream; click "Disconnect all viewers"; confirm both drop and the Hub card viewer count falls to 0 next poll (new M-NN, P07) |
| Live plugin-config hot-swap + DevTools CSP check on /app/ in a real browser | FIX-01 | The Wails WebView is not a real browser; SSE hot-swap + CSP-console inspection need a prod build in Chrome/Firefox | Open the /app/ share URL in a real browser; change plugin-config on the host; confirm live update with no reload; inspect DevTools Console for CSP errors (note the documented /app/ no-CSP-header caveat, RESEARCH Pitfall 1) (new M-NN, P07) |
| Live two-Mac remote-open opens an in-app tab (not external browser) | FIX-03 | Requires a live two-node tailnet | Reworded M-13 (Category G): open a remote tailnet session from the Hub; confirm it opens an in-app web-session tab streaming the relay, not an external browser window |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
