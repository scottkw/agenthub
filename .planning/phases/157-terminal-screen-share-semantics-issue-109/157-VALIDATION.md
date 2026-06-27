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

> Task IDs map to `{phase}-{plan}-{task}`. Plan split (standard granularity): 01 = hub arbiter,
> 02 = server call sites + join push, 03 = web viewer, 04 = desktop viewer parity, 05 = TESTING.md.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 157-01-01 | 01 | 1 | VIEW-01/VIEW-02/D-01/D-02 | T-157-01,T-157-04 | Host-authority ResizeClient: min-among-local, freeze-last-host-size, web-origin gate, broadcastResize + Rows() | unit | `go build ./internal/relay/` | ❌ W0 (rewrite hub.go) | ⬜ pending |
| 157-01-02 | 01 | 1 | VIEW-01/VIEW-02/D-01/D-02 | T-157-01 | Replace MC-06 trio → min/freeze/web-ignore/broadcast/Rows tests | unit | `go test -race -run 'TestHub_Resize\|TestHub_Rows' ./internal/relay/` | ❌ W0 (replaces hub_test.go:354/403/439) | ⬜ pending |
| 157-02-01 | 02 | 2 | VIEW-03 | — | Relay-path join pushes 0x02 before scrollback; local host stays sole driver | unit | `go build ./internal/relay/` | ❌ W0 | ⬜ pending |
| 157-02-02 | 02 | 2 | VIEW-02/VIEW-03 | T-157-02 | Web-path join push before scrollback + drop guest resize at read-pump | unit | `go build ./internal/webserver/` | ❌ W0 | ⬜ pending |
| 157-02-03 | 02 | 2 | VIEW-02/VIEW-03 | T-157-02 | Join-order + web-origin-drop integration tests (both surfaces) | integration | `go test -race -run 'Join\|DropsGuestResize' ./internal/relay/ ./internal/webserver/` | ❌ W0 | ⬜ pending |
| 157-03-01 | 03 | 2 | VIEW-04/VIEW-05 | T-157-03 | Web guest honors 0x02 → term.resize + recomputeScale; no fitAddon.fit / no 0x11 send | structural+manual | `node --check web/assets/terminal.js` | ❌ W0 | ⬜ pending |
| 157-03-02 | 03 | 2 | VIEW-05 | T-157-03 | terminal.css transform container (overflow:hidden + transform-origin) | structural | `grep -c transform-origin web/assets/terminal.css` | ❌ W0 | ⬜ pending |
| 157-04-01 | 04 | 2 | VIEW-04/VIEW-05 | T-157-03 | RelayClient onResize (un-drop 0x02) + pure computeGuestScale cap math | unit | `(cd frontend && pnpm test -- --run relayClient terminalScale)` | ❌ W0 | ⬜ pending |
| 157-04-02 | 04 | 2 | VIEW-04/VIEW-05 | T-157-03,T-157-07 | isGuest-gated honor + capped scale; host path invariant (no transform, sendResize preserved) | unit | `(cd frontend && pnpm test -- --run TerminalPanel.scale)` + `tsc --noEmit` | ❌ W0 | ⬜ pending |
| 157-05-01 | 05 | 3 | VIEW-01..05 | T-157-08 | TESTING.md manifest §2 + traceability §4 + manual §5 (Category P); path-check green | doc gate | `bash tests/check-traceability-paths.sh` | ⚠️ extend TESTING.md | ⬜ pending |

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
