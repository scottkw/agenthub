---
phase: 175
slug: web-share-remote-viewer-windowing-bug-fixes
status: draft
nyquist_compliant: false
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
| TBD | TBD | TBD | BUG-01..04 | — | N/A | unit/manual | TBD | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] BUG-03 live timed repro (manual diagnostic — confirm/deny the exit-poll deadline root cause before writing the fix)
- [ ] Go test scaffolding for WS close-code/reason propagation (BUG-02)
- [ ] Go test scaffolding for scrollback/alt-screen replay (BUG-04)
- [ ] vitest scaffolding for `computeGuestScale` legibility floor (BUG-01)

*Planner refines against final task breakdown.*

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

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
