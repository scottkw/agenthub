---
phase: 95
slug: web-links-addon-security-hardening
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-05-06
revised: 2026-05-06
---

# Phase 95 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: 95-RESEARCH.md `## Validation Architecture` + the 6 PLAN.md `<verify>` blocks.
> RED test files consolidated per plan structure (one urlSafety.test.ts covers all four
> detectors; one TerminalPanel.web-links.test.tsx covers modifier + hover; one
> web_links_test.go covers source-inspection regression suite).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend), Playwright (web-served E2E), Go test (backend RPC + source-inspection) |
| **Config file** | `frontend/vitest.config.ts`, `frontend/e2e/playwright.config.ts`, project-root `go test ./...` |
| **Quick run command** | `pnpm --dir frontend test --run -- src/lib/__tests__/urlSafety.test.ts src/lib/__tests__/openLink.test.ts` |
| **Full suite command** | `pnpm --dir frontend test --run && go test ./internal/daemon/... ./internal/webserver/... && pnpm --dir frontend exec playwright test e2e/web-links-live-toggle.spec.ts` |
| **Estimated runtime** | ~90 seconds (vitest ~25s, go ~10s, Playwright web-links spec ~55s) |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green; Cyrillic / OSC 8 / `javascript:` / `_blank,noopener,noreferrer` regression tests all green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 95-01-T1 | 01 | 0 | LNK-01..LNK-06 | T-95-W0 | Wave 0 spike: validate `@xterm/addon-web-links@^0.12.0` OSC 8 hover/validator API; record `**Selected:** Plan A \| Plan B` in RESEARCH.md `## Wave 0 Spike Outcome` | spike + setup | `pnpm --dir frontend install && go test ./internal/daemon/... -run TestPluginSettings_WebLinksConfig` | ❌ W0 | ⬜ pending |
| 95-01-T2 | 01 | 0 | LNK-01..LNK-06 | — | RED scaffolds for every consolidated test file referenced downstream; metatest asserts Cyrillic spoof fixture preserves U+043E codepoints | scaffold | `pnpm --dir frontend test --run -- src/lib/__tests__/urlSafety.test.ts --reporter=verbose` (expect RED) | ❌ W0 | ⬜ pending |
| 95-02-T1 | 02 | 1 | LNK-01, LNK-04 | T-95-01, T-95-04 | `https://`/`http://`/`mailto:` allowed; `javascript:`, `file://`, `data:`, custom schemes rejected; OSC 8 mismatch / IDN / typosquat detected via first-match-wins `getRisk()` (osc8 > idn > typosquat) | unit | `pnpm --dir frontend test --run -- src/lib/__tests__/urlSafety.test.ts` | ❌ W0 | ⬜ pending |
| 95-02-T2 | 02 | 1 | LNK-04 | T-95-04 | `openLink()` routes desktop → Wails `BrowserOpenURL`; web → `window.open(url, '_blank', 'noopener,noreferrer')`; the literal string `'_blank', 'noopener,noreferrer'` appears verbatim in `frontend/src/lib/openLink.ts` | unit + grep | `pnpm --dir frontend test --run -- src/lib/__tests__/openLink.test.ts && grep -F "'_blank', 'noopener,noreferrer'" frontend/src/lib/openLink.ts` | ❌ W0 | ⬜ pending |
| 95-03-T1 | 03 | 2 | LNK-03 | T-95-03 | `LinkConfirmPopover` portal-renders with full resolved URL via React text content (no `dangerouslySetInnerHTML`); ARIA dialog role; reduced-motion CSS guard | unit | `pnpm --dir frontend test --run -- src/components/__tests__/LinkConfirmPopover.test.tsx` | ❌ W0 | ⬜ pending |
| 95-04-T1 | 04 | 3 | LNK-01, LNK-02, LNK-03 | T-95-01, T-95-02, T-95-03 | TerminalPanel hot-swap mounts `WebLinksAddon` with scheme-allowlist validator + Cmd/Ctrl modifier requirement + hover tooltip + popover-on-risk; reuses existing TerminalPanel.tsx hot-swap pattern (`useEffect` ~lines 259-339); single-click never activates without modifier | unit + integration | `pnpm --dir frontend test --run -- src/components/__tests__/TerminalPanel.web-links.test.tsx` | ❌ W0 | ⬜ pending |
| 95-05-T1 | 05 | 3 | LNK-06 | T-95-06 | `engine.SetWebLinksConfig` mirrors `SetSearchConfig` sub-key writer pattern (`e.pluginSettings.WebLinksConfig = cfg`, no full-settings overwrite); persisted across daemon restart | unit | `go test ./internal/daemon/... -run Test.*WebLinksConfig` | ❌ W0 | ⬜ pending |
| 95-05-T2 | 05 | 3 | LNK-05, LNK-06 | T-95-05, T-95-06 | Wails RPC `SetWebLinksConfig` bound; SSE plugin-event delivers updated config; live toggle re-mounts addon on next refresh without session restart | unit + integration | `pnpm --dir frontend test --run -- src/__tests__/App.plugin-event.test.tsx` | ❌ W0 | ⬜ pending |
| 95-06-T1 | 06 | 4 | LNK-01..LNK-05 (web) | T-95-01..05 | Vendor `@xterm/addon-web-links` UMD into `web/vendor/xterm/web-links`; bump `vendor_drift_test.go` `len(pnpmVersions) >= 7` (was 6); extend `embed.go` to ship asset; web `terminal.html`/`terminal.js`/`terminal.css` includes inline urlSafety + openLink + plain-DOM popover | unit + drift | `go test ./internal/webserver/... -run TestVendorDrift` | ❌ W0 | ⬜ pending |
| 95-06-T2 | 06 | 4 | LNK-01, LNK-04 (web) | T-95-04 | Source-inspection regression: `web/assets/terminal.js` contains the literal `'_blank', 'noopener,noreferrer'` and contains NO current-tab `location.href = ` / `location.assign(` against link URLs; `@xterm/addon-web-links` UMD is byte-identical to vendored copy | grep + go test | `go test ./internal/webserver/... -run "TestSecurity_NoCurrentTabNavigation\|TestTerminalJS_WebLinksOpener\|TestAssets_AddonWebLinks"` | ❌ W0 | ⬜ pending |
| 95-06-T3 | 06 | 4 | LNK-02, LNK-03, LNK-05 (web) | T-95-02..05 | Playwright e2e: web-served session opens link in new tab only (regression: never current-tab); modifier-click required on web; live toggle applies on next refresh; `95-DESKTOP-UAT.md` and `95-WEB-UAT.md` runbooks present and runnable via `/gsd-verify-work 95` | e2e + UAT | `pnpm --dir frontend exec playwright test e2e/web-links-live-toggle.spec.ts` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

The following RED scaffold files MUST exist (failing) before downstream waves run:

- [ ] `frontend/src/lib/__tests__/urlSafety.test.ts` — RED stubs for scheme allowlist (`javascript:`/`file://`/`data:` rejected; `https:`/`http:`/`mailto:` allowed) + OSC 8 mismatch detection + IDN/Cyrillic detect + typosquat patterns + `getRisk()` priority order (osc8 > idn > typosquat). Includes metatest: Cyrillic fixture preserves `host.charCodeAt(i) > 127` for U+043E codepoints (asserts file I/O did not normalize to Latin).
- [ ] `frontend/src/lib/__tests__/openLink.test.ts` — RED stub for desktop (Wails `BrowserOpenURL`) vs web (`window.open(url, '_blank', 'noopener,noreferrer')`) routing; metatest grep asserts the security string is verbatim in `openLink.ts`.
- [ ] `frontend/src/components/__tests__/LinkConfirmPopover.test.tsx` — RED stub for full resolved URL render via React text content (no innerHTML), ARIA dialog, reduced-motion guard.
- [ ] `frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx` — RED stubs for modifier-click semantics (Cmd on darwin / Ctrl on linux+win), hover tooltip rendering, hot-swap re-mount on `pluginConfig?.webLinks` change, single-click rejection.
- [ ] `frontend/src/__tests__/App.plugin-event.test.tsx` — RED stub for SSE plugin-event delivering updated `WebLinksConfig` and re-rendering all open terminals.
- [ ] `internal/daemon/web_links_config_test.go` — RED stub for `SetWebLinksConfig` sub-key writer (mirror `search_config_test.go` verbatim).
- [ ] `internal/webserver/web_links_test.go` — RED stubs for `TestSecurity_NoCurrentTabNavigation`, `TestTerminalJS_WebLinksOpener`, `TestAssets_AddonWebLinks` source-inspection regression suite.
- [ ] `internal/webserver/vendor_drift_test.go` — bump `len(pnpmVersions) >= 7` (was 6) — RED until UMD lands.
- [ ] `frontend/e2e/web-links-live-toggle.spec.ts` — Playwright spec scaffold for web-served `_blank+noopener,noreferrer` regression and live-toggle parity.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Default-browser opens correctly via Wails on macOS / Linux / Windows native builds | LNK-05 | Cross-OS native shell behavior cannot be exercised reliably in CI | Build per-OS via `wails build -tags wailsassets`; in each OS run `echo https://example.com` in a terminal pane; Cmd/Ctrl-click; verify the system default browser opens example.com. Captured in `95-DESKTOP-UAT.md`. |
| Real Tailscale-served session opens link in user's external browser, not the WebView/PWA shell | LNK-05 | Real Tailscale topology required; CI cannot replicate ts-net | Bring up agenthub on host A, join from host B over tailnet, click an `https://` URL on B, verify it opens host B's browser only. Captured in `95-WEB-UAT.md`. |
| Multi-line OSC 8 hyperlink wrap (URL spans across an 80-column wrap point) | LNK-03 | Plan A/B spike outcome may defer multi-line OSC 8 traversal to v3.3; if Plan A traverses `line.isWrapped`, manual fixture confirms; if Plan B is selected, this is documented as a known v3.2 limitation | After Wave 0 spike resolves, in a terminal force-wrap an OSC 8 hyperlink (long display text + long href) and confirm popover triggers (Plan A) OR confirm the limitation is captured in `95-DESKTOP-UAT.md` known-issues (Plan B). |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (RED tests for every LNK-0X requirement)
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter (set when this revision approved)

**Approval:** approved 2026-05-06 (revised to match plan-level test file paths)
