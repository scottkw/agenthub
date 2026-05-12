---
phase: 93
slug: vendoring-discipline-web-parity-for-already-shipping-addons
status: approved
nyquist_compliant: true
wave_0_complete: true
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

> Concrete per-task rows naming the Plan 93-XX-Task-N source. Populated 2026-05-04 after Plans 93-01..05 landed.

| Surface | Plan-Task | Requirement | Test Type | Command |
|---------|-----------|-------------|-----------|---------|
| vendor_drift_test.go generalized regex + min-count | 93-01-T1 | WEB-02 | go test (CI gate) | `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock -count=1` |
| Three vendored UMD bundles + VERSION + embed.go + terminal.html script tags | 93-02-T1, 93-02-T2 | WEB-01, WGL-04, U11-02, CLIP-01 | go test + grep | `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock -count=1 && go test ./internal/webserver/... -run TestSecurity_NoInlineScriptOrStyleInHTML -count=1` |
| TerminalPanel hot-swap useEffects, isSoftwareWebGL, onWebGLContextLost wiring | 93-03-T1, 93-03-T3 | WGL-01, WGL-02, WGL-03, CLIP-02 (desktop) | unit (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.hot-swap.test.tsx src/__tests__/App.plugin-event.test.tsx` |
| WebGLRecoveryBanner + isSoftwareWebGL probe + italic caption | 93-03-T2 | WGL-02, WGL-03, U11-01 | unit (vitest) | `pnpm exec vitest run src/components/__tests__/WebGLRecoveryBanner.test.tsx src/lib/__tests__/webglProbe.test.ts src/components/__tests__/PluginsSection.test.tsx` |
| /api/plugin-config endpoint + capability gate + 503 fallback | 93-04-T1, 93-04-T2 | PLUG-04, WEB-03 | go unit | `go test ./internal/webserver/... -run "TestPluginConfig" -count=1` |
| Web vendored addon assets reachable via /assets/xterm/addons/ | 93-04-T1 | WEB-01 | go integration | `go test ./internal/webserver/... -run TestAssets_VendoredAddons -count=1` |
| Web terminal.js conditional addon loading + context-loss banner | 93-04-T3 | PLUG-04, WEB-03, U11-02, CLIP-02 | playwright e2e | `pnpm playwright test e2e/web-plugin-hot-swap.spec.ts` |
| Web vendor parity + zero CDN | 93-05-T1 | WEB-01, WGL-04, U11-02 | playwright e2e | `pnpm playwright test e2e/web-vendor-parity.spec.ts` |
| Web CSP zero-violation | 93-05-T1 | WEB-02 | playwright e2e | `pnpm playwright test e2e/web-csp.spec.ts` |
| iPad Safari context-loss + software-rasterizer + zero-CDN real network | 93-05-T2 | WGL-02, WGL-03, WEB-02 | manual UAT | follow `93-iPad-UAT.md` UAT-1..5 |

---

## Wave 0 Requirements

The original "Wave 0" entries in this document called for skeletal test files that bootstrap the assertions Plans 93-01..05 would later prove. Those skeletal tests were folded into the RED-step deliverables of each implementation plan rather than executed as a separate Wave 0 plan (see 93-05-SUMMARY.md "Wave 0 rationalization"). All originally-listed Wave 0 items are now satisfied by the corresponding implementation plan's TDD RED commit:

- [x] `frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx` — Plan 93-03 Task 1 (RED)
- [x] `frontend/src/components/__tests__/TerminalPanel.context-loss.test.tsx` — folded into `TerminalPanel.hot-swap.test.tsx` Plan 93-03 Task 1 (context-loss assertion lives in the same hot-swap useEffect surface)
- [x] `frontend/src/components/__tests__/TerminalPanel.unicode11.test.tsx` — folded into `PluginsSection.test.tsx` Plan 93-03 Task 1 (italic-caption assertion belongs with the row that renders it)
- [x] `frontend/src/lib/__tests__/renderer-detect.test.ts` — Plan 93-03 Task 1 (RED) shipped as `webglProbe.test.ts`
- [x] `frontend/e2e/web-vendor-parity.spec.ts` — Plan 93-05 Task 1 (live)
- [x] `frontend/e2e/web-csp-csp.spec.ts` (renamed to `web-csp.spec.ts`) — Plan 93-05 Task 1 (live)
- [x] `internal/webserver/plugin_config_test.go` — Plan 93-04 Task 1 (RED) + Task 2 (GREEN)
- [x] Existing `internal/webserver/vendor_drift_test.go` — Plan 93-01 Task 1 (regex generalization)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| iPad Safari background/foreground triggers WebGL context loss → DOM fallback + toast | WGL-02 | Real iPad Safari behavior cannot be fully simulated headlessly | See `93-iPad-UAT.md` UAT-2 |
| System sleep/wake triggers WebGL context loss recovery | WGL-02 | Requires real OS sleep cycle | See `93-iPad-UAT.md` UAT-2 (extension: substitute Mac sleep for the loseContext() trigger) |
| Real Tailscale-served session shows zero CDN requests | WEB-02 | Confirms vendor wiring against real network egress | See `93-iPad-UAT.md` UAT-5 |
| iPad Safari software-rasterizer preemption banner | WGL-03 | iPad Safari context-loss model differs from desktop | See `93-iPad-UAT.md` UAT-1 |
| Hot-swap across open desktop terminals | WGL-01 | Multi-tab visual confirm | See `93-iPad-UAT.md` UAT-3 |
| Unicode 11 italic caption + next-session-only honoring | U11-01 | Visual + emoji-render confirm | See `93-iPad-UAT.md` UAT-4 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (rolled into Plan 93-01..04 Task 1 RED steps; documented in 93-05-SUMMARY)
- [x] No watch-mode flags
- [x] Feedback latency < 30s for unit tests; < 60s for e2e
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-05-04
