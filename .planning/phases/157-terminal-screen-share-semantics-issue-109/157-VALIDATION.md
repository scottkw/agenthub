---
phase: 157
slug: terminal-screen-share-semantics-issue-109
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-27
---

# Phase 157 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (hub arbitration) + vitest (web/desktop viewer) |
| **Config file** | `go.mod` (Go) · `frontend/vitest.config.ts` (TS) — confirm during planning |
| **Quick run command** | `go test ./internal/relay/...` |
| **Full suite command** | `go test ./... && (cd frontend && pnpm test)` |
| **Estimated runtime** | ~30–90 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/relay/...` (or the touched package's vitest)
- **After every plan wave:** Run `go test ./... && (cd frontend && pnpm test)`
- **Before `/gsd-verify-work`:** Full suite must be green + `bash tests/check-traceability-paths.sh`
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

> Populated by the planner. Anchor rows below reflect the six VIEW change-layers; the planner
> maps concrete task IDs to these requirements.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 157-01-01 | 01 | 1 | VIEW-01 | — | Host-authority: PTY grid tracks local-origin subscriber; min-among-local | unit | `go test ./internal/relay/ -run Resize` | ❌ W0 (replaces MC-06 tests) | ⬜ pending |
| 157-01-02 | 01 | 1 | VIEW-01/D-01 | — | Freeze last host size when no local subscriber (no shrink-to-guest) | unit | `go test ./internal/relay/ -run Resize` | ❌ W0 | ⬜ pending |
| 157-01-03 | 01 | 1 | VIEW-02 | — | Web-origin MsgResize ignored by arbiter; loopback host is sole driver | unit | `go test ./internal/relay/ -run Origin` | ❌ W0 | ⬜ pending |
| 157-01-04 | 01 | 1 | VIEW-03 | — | Join pushes MakeResizeFrame(Cols,Rows) before scrollback replay | unit | `go test ./internal/relay/ -run Join` | ❌ W0 | ⬜ pending |
| 157-02-01 | 02 | 2 | VIEW-04 | — | Guest viewer honors server 0x02 → term.resize; host viewer ignores inbound resize | unit | `(cd frontend && pnpm test)` | ❌ W0 | ⬜ pending |
| 157-02-02 | 02 | 2 | VIEW-05 | — | CSS scale s=min(cW/gW,cH/gH), cap s≤1.0, recompute on resize + 0x02 | unit | `(cd frontend && pnpm test)` | ❌ W0 | ⬜ pending |

---

## Wave 0 Requirements

- [ ] Go hub-level host-authority + min-among-local + freeze tests — **replace** the three MC-06
      max-wins tests at `internal/relay/hub_test.go` (~:354/:403/:439); the unsubscribe-shrink
      assertion must be **inverted** into the D-01 freeze assertion (it currently contradicts D-01).
- [ ] Web-origin-rejection test (arbiter ignores web `MsgResize`/`MsgResize2`).
- [ ] Join-order test (resize frame precedes scrollback replay).
- [ ] Viewer test(s) for server-resize-honoring + scale computation (vitest) — confirm a harness
      exists for `web/assets/terminal.js` / `TerminalPanel.tsx`; if not, scope a minimal one.

*Planner refines exact file paths and confirms the vitest harness for the terminal viewers.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Issue #109 garble scenario (host + smaller-windowed guest → no overlapping/doubled chars) | VIEW-01..05 (criterion 1) | Live multi-window render; xterm pixel output not unit-assertable | Start a session as host, open web-share guest in a smaller window, run a full-screen TUI (e.g. `htop`/`vim`); confirm no garble and CSS downscale fits |
| Downscale fit + cap s≤1.0 (no upscale, no host disturbance) | VIEW-05 (criterion 4) | Visual scale behavior across viewport sizes | Resize guest window larger than host grid → padding, no stretch; smaller → downscaled, no clipping; host view unchanged throughout |
| Cross-surface parity: desktop guest path matches web guest | criterion 5 | Native webview render (parity is release-blocking) | Open the same session as a desktop **remote guest** (HubModal remote / WebShareSessionView) and confirm identical no-garble + scale behavior |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (MC-06 replacement set)
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
