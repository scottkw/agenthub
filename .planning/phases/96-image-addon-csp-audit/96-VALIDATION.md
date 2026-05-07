---
phase: 96
slug: image-addon-csp-audit
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-05-07
---

# Phase 96 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Frontend Framework** | Vitest (via `pnpm exec vitest run`) |
| **Go Framework** | `go test ./...` |
| **Frontend config file** | `frontend/vite.config.ts` |
| **Quick frontend run** | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` |
| **Full frontend suite** | `pnpm test` |
| **Go unit run (relay)** | `go test ./internal/relay/... -count=1` |
| **Go unit run (webserver CSP)** | `go test ./internal/webserver/... -run TestCSP -count=1` |
| **Go unit run (daemon plugin settings)** | `go test ./internal/daemon/... -run TestPluginSettings -count=1` |
| **Go full run** | `go test ./internal/...` |
| **Estimated runtime** | ~45 seconds for the per-task quick suite; ~3 min for full |

---

## Sampling Rate

- **After every task commit:** `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx && go test ./internal/webserver/... ./internal/daemon/... ./internal/relay/... -count=1`
- **After every plan wave:** `pnpm test && go test ./internal/...`
- **Before `/gsd-verify-work`:** Full suite green; chromedp e2e green; manual UAT signed off
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Req ID | Behavior | Test Type | Automated Command | File Exists | Status |
|--------|----------|-----------|-------------------|-------------|--------|
| IMG-01 | PluginsSection renders italic caption under Image toggle | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/PluginsSection.test.tsx` | ✅ extend | ⬜ pending |
| IMG-01 | TerminalPanel constructs ImageAddon when pluginConfig.image is true | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ extend | ⬜ pending |
| IMG-01 | Toggling pluginConfig.image at runtime does NOT re-attach addon (next-session-only) | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ extend | ⬜ pending |
| IMG-02 | ImageConfig.StorageLimit defaults to 16 in defaultPluginSettings | unit (Go) | `go test ./internal/daemon/... -run TestPluginSettings_Defaults -count=1` | ✅ extend | ⬜ pending |
| IMG-02 | SetImageConfig persists ONLY ImageConfig sub-key | unit (Go) | `go test ./internal/daemon/... -run TestSetImageConfig -count=1` | ❌ Wave 0 | ⬜ pending |
| IMG-02 | PATCH /settings/image-config validates StorageLimit ∈ [1, 1000] | unit (Go) | `go test ./internal/daemon/... -run TestHandleSetImageConfig -count=1` | ❌ Wave 0 | ⬜ pending |
| IMG-02 | TerminalPanel passes pluginConfig.imageConfig.storageLimit to ImageAddon constructor | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ extend | ⬜ pending |
| IMG-03 | CSP middleware adds 'wasm-unsafe-eval' to script-src | unit (Go) | `go test ./internal/webserver/... -run TestCSP_WasmUnsafeEval -count=1` | ❌ Wave 0 | ⬜ pending |
| IMG-03 | CSP middleware does NOT contain 'unsafe-eval' (defense regression) | unit (Go) | `go test ./internal/webserver/... -run TestCSP_NoUnsafeEval -count=1` | ❌ Wave 0 | ⬜ pending |
| IMG-03 | Vendored addon-image.js served at /assets/xterm/addons/ | Go integration | `go test ./internal/webserver/... -run TestAssets_Addons -count=1` | ✅ extend | ⬜ pending |
| IMG-03 | vendor_drift_test min-count guard bumped to 8 | Go unit | `go test ./internal/webserver/... -run TestXtermVendorVersions -count=1` | ✅ modify | ⬜ pending |
| IMG-03 | chromedp CSP-zero-violation on Chromium with addon-image loaded + sixel emitted | e2e (Go //go:build e2e) | `go test ./internal/webserver/... -tags=e2e -run TestBrowserCSP_TerminalImage -count=1` | ❌ Wave 3 | ⬜ pending |
| IMG-04 | Synthetic sixel byte stream passes verbatim through relay scrollback | unit (Go) | `go test ./internal/relay/... -run TestImage_ByteFidelity_MultiClient -count=1` | ❌ Wave 0 | ⬜ pending |
| IMG-04 | Two subscribers receive byte-identical fan-out for sixel input | unit (Go) | (same test as above) | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/api_image_test.go` (or extension to `api_test.go`) — covers `handleSetImageConfig` validation
- [ ] `internal/daemon/engine_image_test.go` (or extension) — covers `SetImageConfig` sub-key writer (preserves other fields)
- [ ] `internal/webserver/csp_mw_test.go` extension — `'wasm-unsafe-eval'` present + `'unsafe-eval'` absent assertions
- [ ] `internal/relay/image_byte_fidelity_test.go` (new file) OR extension to `scrollback_test.go` — IMG-04 multi-client byte-fidelity
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` extension — ImageAddon import + construction + storageLimit pass-through + next-session-only invariant
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` extension — italic caption present under Image row
- [ ] `frontend/src/__tests__/App.plugin-event.test.tsx` extension — PluginSettings shape includes imageConfig

---

## Wave 3 Requirements

- [ ] `internal/webserver/browser_csp_image_e2e_test.go` (//go:build e2e) — chromedp loads `/sessions/{id}` with addon-image enabled, emits sixel via test fixture, asserts zero CSP violations + image canvas layer present in DOM
- [ ] `96-HUMAN-UAT.md` runbook — `chafa --format=iterm2 chart.png` desktop + web; two-client mid-stream image join; toggle Image in Settings → italic caption + no live re-attach; 50 MB sixel fixture FIFO eviction

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `chafa --format=iterm2 chart.png` renders identically on first and second mid-stream-joining client | IMG-04 | Visual byte-fidelity at the renderer layer is hard to assert in unit tests; the structural Go test covers the byte stream, but visual confirmation closes the loop | Runbook in `96-HUMAN-UAT.md` |
| 50 MB sixel fixture FIFO-evicts at 16 MB cap with no tab OOM | IMG-02 SC-3 | Tab-OOM is browser-side; addon's eviction is internal to the WASM decoder | Runbook in `96-HUMAN-UAT.md` |
| Toggle Image in Settings → italic caption visible + new sessions render images, existing sessions unchanged | IMG-01 | Live next-session-only semantics need real sessions | Runbook in `96-HUMAN-UAT.md` |

---

## Out-of-Phase (Cross-Phase Gates)

These items are required by Phase 96's ROADMAP success criteria but are explicitly owned by another phase. `/gsd-verify-work 96` MUST NOT mark the listed requirement GREEN until the gating phase lands.

| Behavior | Requirement | Owning Phase | Why Deferred | Verifier Instruction |
|----------|-------------|--------------|--------------|----------------------|
| User-facing Advanced `<details>` disclosure exposing `storageLimit` slider/input under the Inline Images toggle | IMG-02 SC-3 (the user-adjustability clause) | Phase 99 / PUI-03 | ROADMAP Phase 99 SC-2 explicitly owns the `<details>` disclosure UI for Search, Web-Links, and Inline Images together. Phase 96 ships the daemon `ImageConfig` struct + sub-key RPC + hard-coded 16 MB default; Phase 99 wires the UI control. | `/gsd-verify-work 96` should mark IMG-02 SC-3 PARTIAL (daemon-side ✓, UI-side gated by Phase 99). Do NOT mark IMG-02 fully GREEN until Phase 99 lands. |
| Cross-browser CSP zero-violation suite (Safari + Firefox) for the `'wasm-unsafe-eval'` amendment | IMG-03 SC-1 (the cross-browser clause) | Phase 99 SC-4 | Phase 96 ships Chromium-only chromedp e2e (matching Phase 89 precedent). Phase 99 owns the Chromium + Safari + Firefox + iPad Safari release-gate run. | `/gsd-verify-work 96` should mark IMG-03 SC-1 PARTIAL on cross-browser; full release gate is Phase 99. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-05-07 (post plan-checker structural review)
