---
phase: 95
slug: web-links-addon-security-hardening
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-06
---

# Phase 95 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: 95-RESEARCH.md `## Validation Architecture` — fixtures cover scheme allowlist,
> OSC 8 mismatch, IDN/Cyrillic spoof, web-served `_blank+noopener,noreferrer`,
> single-click rejection, and live Settings toggle.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend), Playwright (web-served E2E), Go test (backend RPC) |
| **Config file** | `frontend/vitest.config.ts`, `e2e/playwright.config.ts`, project-root `go test ./...` |
| **Quick run command** | `pnpm --dir frontend test --run -- src/components/__tests__/WebLinks*.test.tsx` |
| **Full suite command** | `pnpm --dir frontend test --run && go test ./internal/daemon/... ./app/... && pnpm --dir e2e test web-links` |
| **Estimated runtime** | ~90 seconds (vitest ~25s, go ~10s, Playwright web-links spec ~55s) |

---

## Sampling Rate

- **After every task commit:** Run `pnpm --dir frontend test --run -- src/components/__tests__/WebLinks*.test.tsx`
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green; Cyrillic/OSC 8/javascript: fixtures all green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 95-00-01 | 00 | 0 | LNK-01..06 | T-95-W0 | Wave 0 spike: confirm `@xterm/addon-web-links@^0.12.0` exposes the OSC 8 hover/validator hooks needed; document Plan-B if not | spike | `pnpm --dir frontend test --run -- src/components/__tests__/WebLinks.spike.test.tsx` | ❌ W0 | ⬜ pending |
| 95-01-01 | 01 | 1 | LNK-01 | T-95-01 | `javascript:`/`file://`/custom-scheme URLs are NEVER made clickable | unit | `pnpm --dir frontend test --run -- src/lib/__tests__/isAllowedScheme.test.ts` | ❌ W0 | ⬜ pending |
| 95-01-02 | 01 | 1 | LNK-01 | T-95-01 | `https://`, `http://`, `mailto:` URLs ARE made clickable | unit | same file | ❌ W0 | ⬜ pending |
| 95-02-01 | 02 | 2 | LNK-02 | T-95-02 | Single-click does not activate a link without modifier | unit | `pnpm --dir frontend test --run -- src/components/__tests__/WebLinksModifier.test.tsx` | ❌ W0 | ⬜ pending |
| 95-02-02 | 02 | 2 | LNK-02 | T-95-02 | Cmd-click on darwin / Ctrl-click on linux+win activates a link | unit | same file | ❌ W0 | ⬜ pending |
| 95-02-03 | 02 | 2 | LNK-02 | — | Modifier requirement is configurable via `WebLinksConfig.activationModifier` and persisted | unit | `pnpm --dir frontend test --run -- src/state/__tests__/webLinksConfig.test.ts` | ❌ W0 | ⬜ pending |
| 95-03-01 | 03 | 2 | LNK-03 | T-95-03 | Hover on a link shows a tooltip with the resolved href | unit | `pnpm --dir frontend test --run -- src/components/__tests__/WebLinksHoverTooltip.test.tsx` | ❌ W0 | ⬜ pending |
| 95-03-02 | 03 | 2 | LNK-03 | T-95-03 | OSC 8 hyperlink with display="click here" / href="https://evil.example" surfaces mismatch warning in tooltip and triggers popover on activation | unit | `pnpm --dir frontend test --run -- src/lib/__tests__/osc8Mismatch.test.ts` | ❌ W0 | ⬜ pending |
| 95-04-01 | 04 | 3 | LNK-04 | T-95-04 | `https://gооgle.com` (Cyrillic spoof) triggers click-confirmation popover before navigation | unit | `pnpm --dir frontend test --run -- src/lib/__tests__/idnDetect.test.ts` | ❌ W0 | ⬜ pending |
| 95-04-02 | 04 | 3 | LNK-04 | T-95-04 | Known typosquat patterns (`gооgle.com`, `g00gle.com`, `paypa1.com`, mixed-script) trigger popover | unit | `pnpm --dir frontend test --run -- src/lib/__tests__/typosquat.test.ts` | ❌ W0 | ⬜ pending |
| 95-04-03 | 04 | 3 | LNK-04 | T-95-04 | Confirmation popover renders the FULL resolved URL (incl. punycode form) | unit | `pnpm --dir frontend test --run -- src/components/__tests__/LinkConfirmPopover.test.tsx` | ❌ W0 | ⬜ pending |
| 95-05-01 | 05 | 4 | LNK-05 | T-95-05 | Desktop link activation calls Wails `BrowserOpenURL`; never opens inside the WebView | unit | `pnpm --dir frontend test --run -- src/lib/__tests__/openLink.test.ts` | ❌ W0 | ⬜ pending |
| 95-05-02 | 05 | 4 | LNK-05 | T-95-05 | Web-served session opens link via `window.open(url, '_blank', 'noopener,noreferrer')` only — never current-tab navigation | e2e | `pnpm --dir e2e test web-links -- --grep="web-served navigation"` | ❌ W0 | ⬜ pending |
| 95-05-03 | 05 | 4 | LNK-05 | T-95-05 | Web vendor parity: `web/vendor/xterm/web-links` matches frontend version (vendor_drift_test) | unit | `go test ./web/...` | ❌ W0 | ⬜ pending |
| 95-06-01 | 06 | 4 | LNK-06 | T-95-06 | Settings toggle persists via `SetWebLinksConfig` (sub-key, not full settings overwrite) | unit | `go test ./internal/daemon/... -run Test.*WebLinksConfig` | ❌ W0 | ⬜ pending |
| 95-06-02 | 06 | 4 | LNK-06 | T-95-06 | Toggling web-links applies live to all open terminals on next refresh; no session restart required | integration | `pnpm --dir frontend test --run -- src/components/__tests__/TerminalPanel.weblinks.live.test.tsx` | ❌ W0 | ⬜ pending |
| 95-06-03 | 06 | 4 | LNK-06 | T-95-06 | Web-served live-toggle parity (Playwright) | e2e | `pnpm --dir e2e test web-links -- --grep="live toggle"` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/WebLinks.spike.test.tsx` — confirm `@xterm/addon-web-links@^0.12.0` API surface (OSC 8 hooks, hover validator, custom click handler signature). Document Plan-B in commit message if missing.
- [ ] `frontend/src/lib/__tests__/isAllowedScheme.test.ts` — RED stubs for LNK-01 (positive + negative scheme cases including `javascript:`, `data:`, `file://`)
- [ ] `frontend/src/lib/__tests__/osc8Mismatch.test.ts` — RED stub for OSC 8 display-vs-href divergence detector
- [ ] `frontend/src/lib/__tests__/idnDetect.test.ts` — RED stub for IDN/Punycode + mixed-script detection (Cyrillic `gооgle.com` fixture)
- [ ] `frontend/src/lib/__tests__/typosquat.test.ts` — RED stub for typosquat pattern matcher
- [ ] `frontend/src/lib/__tests__/openLink.test.ts` — RED stub for desktop-vs-web routing decision
- [ ] `frontend/src/components/__tests__/WebLinksModifier.test.tsx` — RED stub for Cmd/Ctrl-click semantics by platform
- [ ] `frontend/src/components/__tests__/WebLinksHoverTooltip.test.tsx` — RED stub for hover tooltip rendering
- [ ] `frontend/src/components/__tests__/LinkConfirmPopover.test.tsx` — RED stub for popover with full resolved URL
- [ ] `frontend/src/components/__tests__/TerminalPanel.weblinks.live.test.tsx` — RED stub for live-toggle integration
- [ ] `frontend/src/state/__tests__/webLinksConfig.test.ts` — RED stub for sub-key persistence (mirror Phase 94 SearchConfig precedent)
- [ ] `internal/daemon/plugin_settings_weblinks_test.go` — RED stub for `SetWebLinksConfig` RPC (sub-key writer pattern from Phase 94)
- [ ] `e2e/web-links.spec.ts` — Playwright spec scaffold for web-served `_blank+noopener,noreferrer` and live-toggle parity

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Default-browser opens correctly via Wails on macOS/Linux/Windows native builds | LNK-05 | Cross-OS native shell behavior cannot be exercised reliably in CI | Build per-OS via `wails build`; in each OS run `echo https://example.com` in a terminal pane; Cmd/Ctrl-click; verify the system default browser opens example.com |
| Real Tailscale-served session opens link in user's external browser, not the WebView/PWA shell | LNK-05 | Real Tailscale topology required; CI cannot replicate ts-net | Bring up agenthub on host A, join from host B over tailnet, click an `https://` URL on B, verify it opens host B's browser only |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (RED tests for every LNK-0X requirement)
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
