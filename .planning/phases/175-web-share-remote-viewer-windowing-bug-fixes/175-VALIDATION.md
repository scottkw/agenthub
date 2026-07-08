---
phase: 175
slug: web-share-remote-viewer-windowing-bug-fixes
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-08
---

# Phase 175 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `go test` (backend: internal/webserver, internal/relay) + vitest (frontend: terminalScale, TerminalPanel) |
| **Config file** | go.mod / frontend/vitest.config.ts |
| **Quick run command** | `go test ./internal/webserver/... ./internal/relay/...` · `pnpm --dir frontend test:run <file>` |
| **Full suite command** | `bash tests/run-all.sh` (or the suite groups in TESTING.md Section 2) |
| **Estimated runtime** | ~60–120 seconds |

---

## Sampling Rate

- **After every task commit:** Run the relevant quick command (Go pkg test or targeted vitest)
- **After every plan wave:** Run the full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

> Planner fills this from the final PLAN.md task IDs. One row per task.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 175-01 T1 | 175-01 | 0 | BUG-03 | T-175-01-01 | diagnosis records no tokens | manual | N/A (live timed repro → 175-01-DIAGNOSIS.md) | ❌ manual | ⬜ pending |
| 175-02 T1 | 175-02 | 0 | BUG-03 | T-175-02-01 | behavior-preserving refactor | unit (Go) | `go test ./... -run TestShouldContinuePolling -count=1` | ❌ W0 creates | ⬜ pending |
| 175-02 T2 | 175-02 | 0 | BUG-02 | T-175-02-02 | pins fixed non-leaking reason | unit (Go, RED skip) | `go test ./internal/webserver/... -run TestSessionEnd -count=1` | ❌ W0 creates | ⬜ pending |
| 175-02 T3 | 175-02 | 0 | BUG-04 | — | — | unit (Go, RED skip) | `go test ./internal/relay/... -run TestScrollbackAltScreenReplay -count=1` | ❌ W0 creates | ⬜ pending |
| 175-03 T1 | 175-03 | 1 | BUG-01 | T-175-03-01 | never upscale; no host resize | unit (vitest) | `cd frontend && pnpm exec vitest run src/lib/terminalScale.test.ts` | ✅ extend | ⬜ pending |
| 175-03 T2 | 175-03 | 1 | BUG-01 | T-175-03-01 | guest path never sendResize | unit (vitest) | `cd frontend && pnpm exec vitest run src/components/__tests__/TerminalPanel.scale.test.tsx` | ✅ extend | ⬜ pending |
| 175-04 T1 | 175-04 | 1 | BUG-04 | — | — | source grep | `grep -n "Open in tab" frontend/src/components/Hub/SessionCard.tsx` | ✅ exists | ⬜ pending |
| 175-04 T2 | 175-04 | 1 | BUG-04 | T-175-04-01/02 | lazy + bounded + non-blocking | unit (Go) | `go test ./internal/relay/... -run TestScrollbackAltScreenReplay -count=1` | ❌ W0→green | ⬜ pending |
| 175-04 T3 | 175-04 | 1 | BUG-04 | T-175-04-03 | server→client only, perms intact | unit (Go) | `go build ./... && go test ./internal/webserver/... ./internal/relay/... -count=1` | ✅ exists | ⬜ pending |
| 175-05 T1 | 175-05 | 1 | BUG-03 | T-175-05-01 | bounded loop exits preserved | unit (Go) | `go test ./... -run TestShouldContinuePolling -count=1` | ❌ W0→update | ⬜ pending |
| 175-05 T2 | 175-05 | 1 | BUG-03 | T-175-05-02 | logs no tokens/content | vet + grep | `go vet ./... && grep -n "slog" app.go` | ✅ exists | ⬜ pending |
| 175-06 T1 | 175-06 | 2 | BUG-02 | T-175-06-01 | fixed generic close reason | unit (Go) | `go test ./internal/webserver/... -run TestSessionEnd -count=1` | ❌ W0→green | ⬜ pending |
| 175-06 T2 | 175-06 | 2 | BUG-02 | T-175-06-02/03 | no raw reason render; no reconnect | unit (vitest) | `cd frontend && pnpm exec vitest run src/lib/relayClient.test.ts src/components/__tests__/SessionEndedBanner.test.tsx` | ❌ creates | ⬜ pending |
| 175-07 T1 | 175-07 | 3 | BUG-01..04 | T-175-07-01 | manifest cannot silently drift | script | `bash tests/check-traceability-paths.sh` | ✅ exists | ⬜ pending |
| 175-07 T2 | 175-07 | 3 | BUG-01..04 | — | — | grep gate | `grep -c "dead code\|/app/" TESTING.md` | ✅ exists | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] **175-01 T1** — BUG-03 live timed repro (manual diagnostic → 175-01-DIAGNOSIS.md; confirm/deny the app.go:386 exit-poll deadline root cause BEFORE 175-05 writes the fix)
- [ ] **175-02 T1** — extract `shouldContinuePolling` pure helper from app.go:386 + `app_poll_test.go` (GREEN, pins current 300s behavior; the seam 175-05 rewires)
- [ ] **175-02 T2** — Go test scaffold for WS close-code/reason propagation (`internal/webserver/session_ended_test.go`, RED skip-guarded; greened by 175-06) (BUG-02)
- [ ] **175-02 T3** — Go test scaffold for scrollback/alt-screen reconnect replay (`internal/relay/scrollback_altscreen_test.go`, RED skip-guarded; greened by 175-04) (BUG-04)

> Note: BUG-01's vitest floor scaffold is done as in-plan TDD inside 175-03 T1 (pure-function RED→GREEN in the same wave), not as a separate Wave-0 file — the pure fn is trivial to red-first alongside its implementation.

*Planner refined against the final 7-plan / 4-wave breakdown.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Mobile terminal legibility on a real phone viewport | BUG-01 | Requires a real/emulated mobile viewport rendering the live web-share surface | Web-share a session, open the link on a phone-width viewport, confirm the 80-col grid is readable (no unbounded downscale) |
| Disconnect notice on owner-ends-session | BUG-02 | Requires a live owner→guest session teardown | Owner stops a shared session; guest sees a clear disconnect banner, not a frozen terminal |
| Shared-session tab auto-closes on exit | BUG-03 | Requires live PTY exit timing | Exit from inside a shared session; its tab auto-closes matching unshared behavior |
| Long-running TUI agent replays alt-screen for late joiner | BUG-04 | Requires a live alt-screen TUI + late guest join | Start a full-screen TUI agent, join as guest after it enters alt-screen; guest sees current screen, not raw scrollback |

*Live UAT items — the Wails-coupled app shell won't render in plain Chrome; verify via web-share to a real browser + native app.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (BUG-02/03/04 Go scaffolds; BUG-01 floor is in-plan TDD)
- [x] No watch-mode flags (all `vitest run` / `go test`, no `--watch`)
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planner-approved (pending execution)
