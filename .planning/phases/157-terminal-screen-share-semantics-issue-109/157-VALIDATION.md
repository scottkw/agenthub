---
phase: 157
slug: terminal-screen-share-semantics-issue-109
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-27
validated: 2026-06-27
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

> **Status legend:** ✅ COVERED = test exists, targets behavior, runs green. All rows below
> re-confirmed live during the 2026-06-27 validation audit (see Validation Audit section).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 157-01-01 | 01 | 1 | VIEW-01/VIEW-02/D-01/D-02 | T-157-01,T-157-04 | Host-authority ResizeClient: min-among-local, freeze-last-host-size, web-origin gate, broadcastResize + Rows() | unit | `go build ./internal/relay/` | ✅ `internal/relay/hub.go` | ✅ COVERED |
| 157-01-02 | 01 | 1 | VIEW-01/VIEW-02/D-01/D-02 | T-157-01 | Replace MC-06 trio → min/freeze/web-ignore/broadcast/Rows tests (6 host-authority tests) | unit | `go test -race -run 'TestHub_Resize\|TestHub_Rows' ./internal/relay/` | ✅ `internal/relay/hub_test.go` | ✅ COVERED |
| 157-02-01 | 02 | 2 | VIEW-03 | — | Relay-path join pushes 0x02 before scrollback; local host stays sole driver | unit | `go build ./internal/relay/` | ✅ `internal/relay/server.go` | ✅ COVERED |
| 157-02-02 | 02 | 2 | VIEW-02/VIEW-03 | T-157-02 | Web-path join push before scrollback + drop guest resize at read-pump | unit | `go build ./internal/webserver/` | ✅ `internal/webserver/server.go` | ✅ COVERED |
| 157-02-03 | 02 | 2 | VIEW-02/VIEW-03 | T-157-02 | Join-order + web-origin-drop integration tests (both surfaces) | integration | `go test -race -run 'TestRelayJoin_PushesResizeBeforeScrollback\|TestWebJoin_PushesResizeBeforeScrollback\|TestWebReadPump_DropsGuestResize' ./internal/relay/ ./internal/webserver/` | ✅ `internal/relay/server_test.go` · `internal/webserver/server_test.go` | ✅ COVERED |
| 157-03-01 | 03 | 2 | VIEW-04/VIEW-05 | T-157-03 | Web guest honors 0x02 → term.resize + recomputeScale; no fitAddon.fit / no 0x11 send | structural+manual | `node --check web/assets/terminal.js` (+ M-27 live UAT) | ✅ `web/assets/terminal.js` | ✅ COVERED (structural gate green + M-27 live-verified) |
| 157-03-02 | 03 | 2 | VIEW-05 | T-157-03 | terminal.css transform container (overflow:hidden + transform-origin) | structural | `grep -c transform-origin web/assets/terminal.css` | ✅ `web/assets/terminal.css` | ✅ COVERED |
| 157-04-01 | 04 | 2 | VIEW-04/VIEW-05 | T-157-03 | RelayClient onResize (un-drop 0x02) + pure computeGuestScale cap math | unit | `(cd frontend && pnpm test -- --run relayClient terminalScale)` | ✅ `frontend/src/lib/relayClient.test.ts` · `frontend/src/lib/terminalScale.test.ts` | ✅ COVERED |
| 157-04-02 | 04 | 2 | VIEW-04/VIEW-05 | T-157-03,T-157-07 | isGuest-gated honor + capped scale; host path invariant (no transform, sendResize preserved) | unit | `(cd frontend && pnpm test -- --run TerminalPanel.scale)` + `tsc --noEmit` | ✅ `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` | ✅ COVERED |
| 157-05-01 | 05 | 3 | VIEW-01..05 | T-157-08 | TESTING.md manifest §2 + traceability §4 + manual §5 (Category P); path-check green | doc gate | `bash tests/check-traceability-paths.sh` | ✅ `TESTING.md` | ✅ COVERED |

---

## Wave 0 Requirements

- [x] Go hub-level host-authority + min-among-local + freeze tests — MC-06 max-wins trio
      **replaced** with 6 host-authority tests in `internal/relay/hub_test.go`; the
      unsubscribe-shrink assertion was **inverted** into `TestHub_ResizeFreezeLastHostSize` (D-01).
- [x] Web-origin-rejection test — `TestHub_ResizeIgnoresWebOrigin` (arbiter) +
      `TestWebReadPump_DropsGuestResize` (webserver call-site drop).
- [x] Join-order test — `TestRelayJoin_PushesResizeBeforeScrollback` (relay) +
      `TestWebJoin_PushesResizeBeforeScrollback` (web): 0x02 frame precedes scrollback replay.
- [x] Viewer test(s) for server-resize-honoring + scale computation (vitest) — desktop harness
      delivered: `relayClient.test.ts` (0x02 dispatch), `terminalScale.test.ts` (cap math),
      `TerminalPanel.scale.test.tsx` (honor + host invariance). Web `terminal.js` is a vendored
      asset outside vitest — covered by `node --check` structural gate + Category P manual UAT.

*Planner refines exact file paths and confirms the vitest harness for the terminal viewers.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Issue #109 garble scenario (host + smaller-windowed guest → no overlapping/doubled chars) | VIEW-01..05 (criterion 1) | Live multi-window render; xterm pixel output not unit-assertable | Start a session as host, open web-share guest in a smaller window, run a full-screen TUI (e.g. `htop`/`vim`); confirm no garble and CSS downscale fits |
| Downscale fit + cap s≤1.0 (no upscale, no host disturbance) | VIEW-05 (criterion 4) | Visual scale behavior across viewport sizes | Resize guest window larger than host grid → padding, no stretch; smaller → downscaled, no clipping; host view unchanged throughout |
| Cross-surface parity: desktop guest path matches web guest | criterion 5 | Native webview render (parity is release-blocking) | Open the same session as a desktop **remote guest** (HubModal remote / WebShareSessionView) and confirm identical no-garble + scale behavior |

> **Manual-only status (2026-06-27):** all three rows above were LIVE-verified via dev-browser
> and recorded in `157-UAT.md` (M-27 + M-28, 2/2 pass, 0 issues). Surface correction: the
> "desktop guest" is the `/app/` React SPA (browser-drivable), not the native Wails WebView.
> These items are genuinely non-automatable (live multi-window xterm pixel render) and remain
> manual by design — supplements to the green automated coverage, not the sole coverage.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (MC-06 replacement set)
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-27 — all automatable requirements green; manual-only items live-verified (M-27/M-28).

---

## Validation Audit 2026-06-27

Audit of the draft VALIDATION.md against the executed phase (State A — existing file). All five
VIEW requirements re-confirmed: every automatable behavior has a green test; the only non-automated
portions (web `terminal.js` render + visual garble/scale checks) are vendored-asset / live-render
items already classified manual-only and live-verified via dev-browser (M-27/M-28, 2/2 pass).

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
| Tasks re-confirmed COVERED | 10/10 |

**Live re-run evidence:**
- `go test -race -run 'TestHub_Resize|TestHub_Rows|TestRelayJoin_PushesResizeBeforeScrollback|TestWebJoin_PushesResizeBeforeScrollback|TestWebReadPump_DropsGuestResize' ./internal/relay/ ./internal/webserver/` → both packages `ok`
- `(cd frontend && pnpm test -- --run relayClient terminalScale TerminalPanel.scale)` → 3 files, 76/76 pass
- `node --check web/assets/terminal.js` → clean (structural gate)

No test files generated — phase already Nyquist-complete. No new-test commit needed.
