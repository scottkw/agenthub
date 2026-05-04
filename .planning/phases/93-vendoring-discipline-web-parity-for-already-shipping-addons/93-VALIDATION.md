---
phase: 93
slug: vendoring-discipline-web-parity-for-already-shipping-addons
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-04
---

# Phase 93 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend), `go test` (backend), Playwright (web e2e) |
| **Config file** | `frontend/vitest.config.ts`, `go.mod`, `frontend/playwright.config.ts` |
| **Quick run command** | `cd frontend && pnpm vitest run --reporter=dot` (frontend) / `go test ./internal/... -count=1` (backend) |
| **Full suite command** | `cd frontend && pnpm vitest run && pnpm playwright test` && `go test ./... -count=1` |
| **Estimated runtime** | ~30s frontend unit, ~15s go, ~60s e2e |

---

## Sampling Rate

- **After every task commit:** Run the relevant `quick run command` for the touched stack (frontend or backend).
- **After every plan wave:** Run `full suite command`.
- **Before `/gsd-verify-work`:** Full suite must be green AND `vendor_drift_test.go` must pass.
- **Max feedback latency:** 30 seconds (frontend unit), 15 seconds (go).

---

## Per-Task Verification Map

> Filled in by gsd-planner when PLAN.md task IDs are assigned. Below are the validation surfaces the planner must map onto.

| Surface | Requirement | Test Type | Command |
|---------|-------------|-----------|---------|
| TerminalPanel hot-swap WebGL on/off across open terminals | WGL-01 | unit | `pnpm vitest run TerminalPanel` |
| Unicode 11 italic caption renders + honored at next session | U11-01 | unit | `pnpm vitest run SettingsPanel` / `pnpm vitest run TerminalPanel` |
| Clipboard (OSC 52) addon attaches under desktop reconcile | CLIP-01, CLIP-02 | unit | `pnpm vitest run TerminalPanel` |
| WebGL onContextLoss → DOM fallback + one-shot toast | WGL-02 | unit | `pnpm vitest run TerminalPanel` (mocked context-loss) |
| Software-rasterized renderer detection at startup | WGL-03 | unit | `pnpm vitest run` (renderer-detection helper) |
| Web `terminal.html` renders WebGL/U11/clipboard same-origin | WEB-01, WGL-04, U11-02 | e2e | `pnpm playwright test web-vendor-parity.spec.ts` |
| Web devtools shows zero CDN requests + zero CSP violations | WEB-02 | e2e | `pnpm playwright test web-csp-csv.spec.ts` |
| `/api/plugin-config` SSE/WS pushes hot-swap to web clients | PLUG-04, WEB-03 | integration | `go test ./internal/webserver/... -run PluginConfig` + `pnpm playwright test web-plugin-hot-swap.spec.ts` |
| `vendor_drift_test.go` fails on any addon-* drift | (CI gate covering all WGL/U11/CLIP) | go test (CI gate) | `go test ./internal/... -run TestVendorDrift` |

The planner MUST emit one row per task in the final PLAN.md, citing one of these surfaces or adding a new one.

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx` — hot-swap useEffect skeleton tests (WGL-01, CLIP-02)
- [ ] `frontend/src/components/__tests__/TerminalPanel.context-loss.test.tsx` — `onContextLoss` mock harness (WGL-02)
- [ ] `frontend/src/components/__tests__/TerminalPanel.unicode11.test.tsx` — italic caption + new-session-only honoring (U11-01)
- [ ] `frontend/src/lib/__tests__/renderer-detect.test.ts` — software-rasterizer detection (WGL-03)
- [ ] `frontend/e2e/web-vendor-parity.spec.ts` — Playwright fixture for `web/terminal.html` w/ vendored assets
- [ ] `frontend/e2e/web-csp-csp.spec.ts` — DevTools network + CSP violation assertion
- [ ] `internal/webserver/plugin_config_test.go` — `/api/plugin-config` capability + push test
- [ ] Existing `internal/.../vendor_drift_test.go` — extend regex matrix (no new file)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| iPad Safari background/foreground triggers WebGL context loss → DOM fallback + toast | WGL-02 | Real iPad Safari behavior cannot be fully simulated headlessly | (1) Open AgentHub web terminal on iPad Safari over Tailscale. (2) Switch to home screen and wait 30s. (3) Reopen Safari tab. (4) Confirm scrollback intact, DOM renderer active, one-shot info toast visible in `.banner-stack`. |
| System sleep/wake triggers WebGL context loss recovery | WGL-02 | Requires real OS sleep cycle | (1) Open desktop terminal with WebGL enabled. (2) Sleep Mac for 60s. (3) Wake. (4) Confirm renderer fell back to DOM (not retried into WebGL), scrollback intact, one-shot toast visible. |
| Real Tailscale-served session shows zero CDN requests | WEB-02 | Confirms vendor wiring against real network egress | Open `https://<tailnet>.ts.net/terminal.html` in Chrome over Tailscale. Open DevTools → Network. Filter for `cdn.jsdelivr.net OR unpkg.com OR esm.sh`. Confirm zero matches across full attach/resize/scrollback session. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
