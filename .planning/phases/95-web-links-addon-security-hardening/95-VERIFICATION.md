---
phase: 95-web-links-addon-security-hardening
verified: 2026-05-06T20:55:00Z
status: human_needed
score: 5/5 must-haves verified (automated); 5 SCs require manual UAT for full sign-off
overrides_applied: 0
human_verification:
  - test: "macOS Cmd-click activates link; Linux/Windows Ctrl-click activates link; single-click never activates (LNK-02 / SC-2)"
    expected: "Cmd-click on macOS opens URL in default browser via Wails BrowserOpenURL; single-click does nothing; hover tooltip shows resolved href"
    why_human: "Real-OS modifier semantics + Wails native browser invocation cannot be automated in jsdom; deferred to 95-DESKTOP-UAT.md §2"
  - test: "Cyrillic spoof URL https://gооgle.com triggers click-confirmation popover before navigation (LNK-03 / SC-3)"
    expected: "Cmd-click on Cyrillic URL renders popover with idn copy + full resolved URL; Continue opens browser; Cancel dismisses without navigation"
    why_human: "Visual popover behavior + real terminal output rendering not feasible in jsdom (xterm canvas/WebGL); deferred to 95-DESKTOP-UAT.md §3"
  - test: "Web-served terminal page on Tailscale: window.open with '_blank' + 'noopener,noreferrer'; window.opener === null in opened tab (LNK-04 / SC-4 web side)"
    expected: "Click https URL on web session, new tab opens, DevTools shows window.opener === null"
    why_human: "Requires real browser + Tailscale-served session; deferred to 95-WEB-UAT.md §5"
  - test: "Live toggle: disable web-links in Settings, already-rendered links lose underline on next refresh; re-enable, links return — no session restart (LNK-06 / SC-5)"
    expected: "settings:plugins event propagates via Wails EventsEmit (desktop) and SSE /api/plugin-config/stream (web); applyPluginConfig disposes/loads addon without restart"
    why_human: "Real settings:plugins SSE round-trip + DOM observation; deferred to 95-DESKTOP-UAT.md §6 + 95-WEB-UAT.md §7"
  - test: "iPad Safari Tailscale walkthrough: full LNK-01..05 chain on iOS Safari with paired keyboard for Cmd-modifier (Phase 99 release gate co-verification)"
    expected: "All gates fire correctly on iPad Safari; no console errors; popover renders; window.open spawns new tab"
    why_human: "Cross-OS / iOS Safari quirks not testable from CI; deferred to 95-WEB-UAT.md §9"
deferred:
  - truth: "OSC 8 hyperlink display-vs-href divergence triggers click-confirmation popover (SC-3 OSC 8 slice)"
    addressed_in: "v3.3 (post-roadmap)"
    evidence: "Plan B selected per 95-RESEARCH.md Wave 0 Spike Outcome: IBufferCell.getHyperlinkId not in @xterm/xterm@6.0.0 public typings. osc8 branch ships dormant in LinkConfirmPopover and showLinkConfirmPopover; getRisk(uri, uri) for plain-text URLs cannot fire osc8 branch since displayText === uri. Documented across all 6 plan SUMMARYs and in 95-DESKTOP-UAT.md §5b. NOTE: LNK-OSC8-FUT-01 was promised in plan SUMMARYs but is NOT yet in REQUIREMENTS.md ## Future Requirements — see warning below."
---

# Phase 95: Web-Links Addon Security Hardening — Verification Report

**Phase Goal:** Clickable URLs ship with v3.1-WS-Origin-allowlist rigor: no scheme outside an explicit allowlist becomes clickable, no link can be activated by accidental single-click, and OSC 8 / IDN / typosquat phishing primitives are detected and surfaced before navigation.

**Verified:** 2026-05-06T20:55:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (cross-referenced against ROADMAP Success Criteria)

| #   | Truth (ROADMAP SC)                                                                                                              | Status     | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| --- | ------------------------------------------------------------------------------------------------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| SC-1 | Scheme allowlist (https/http/mailto only); javascript:/file:/etc never clickable; regression test fails on attacker-supplied javascript: | ✓ VERIFIED | `frontend/src/lib/urlSafety.ts:11` — `ALLOWED_SCHEMES = ['https:', 'http:', 'mailto:'] as const`. `urlSafety.test.ts` 20/20 GREEN including `rejects javascript:`/`file:`/`data:`. Defense-in-depth: addon's default urlRegex + `isAllowedScheme(uri)` in handler at `TerminalPanel.tsx:388` + scheme regex in `openLink.ts:49`. Web mirror in `web/assets/terminal.js`. |
| SC-2 | Modifier-click required (Cmd on macOS / Ctrl on Linux+Win); single-click never activates; hover tooltip shows resolved href     | ✓ VERIFIED (auto) / ? HUMAN | `isModifierPressed` at `openLink.ts:28-37` — platform mode dispatches metaKey on Mac, ctrlKey elsewhere; tested in `openLink.test.ts` (5 GREEN tests). Handler at `TerminalPanel.tsx:398` rejects un-modifier'd clicks. Hover/leave tooltip via `event.target.setAttribute('title', uri)` at `TerminalPanel.tsx:419-431`. Real-OS modifier behavior + Wails BrowserOpenURL invocation requires manual UAT. |
| SC-3 | OSC 8 mismatch / IDN / typosquat trigger popover; Cyrillic gооgle.com fixture + OSC 8 mismatch must trigger                     | ⚠️ PARTIAL  | IDN + typosquat: ✓ wired and tested (Cyrillic fixture U+043E codepoints preserved per metatest in `urlSafety.test.ts`; popover renders for `risk='idn'` and `risk='typosquat'`). **OSC 8 mismatch: ⚠️ DEFERRED to v3.3** per Plan B Wave 0 spike outcome (IBufferCell.getHyperlinkId not in @xterm/xterm@6.0.0 public typings). osc8 branch ships dormant in popover; `getRisk(uri, uri)` cannot fire it for plain-text URLs. Listed in `deferred:` frontmatter. |
| SC-4 | Desktop: BrowserOpenURL; Web: window.open with noopener,noreferrer (no current-tab navigation, regression test enforced)        | ✓ VERIFIED | `openLink.ts:48-59` — Wails branch calls `BrowserOpenURL`, web branch calls `window.open(url, '_blank', 'noopener,noreferrer')`. `web/assets/terminal.js:338` — same literal. `TestSecurity_NoCurrentTabNavigation` GREEN — source-inspects both files for location.href/window.location patterns. `TestTerminalJS_WebLinksOpener` GREEN — asserts BrowserOpenURL absent from web. |
| SC-5 | User can enable/disable in Settings; toggling applies live without session restart                                              | ✓ VERIFIED (auto) / ? HUMAN | Hot-swap arm in `TerminalPanel.tsx:382-446` extends useEffect deps with `pluginConfig?.webLinks` (NOT `webLinksConfig` — sub-config flows via ref per Pitfall #8). Toggle off disposes addon; toggle on constructs. App.tsx subscribes to `EventsOn('settings:plugins')` at `App.tsx:351`. Engine `SetWebLinksConfig` at `internal/daemon/engine.go:526-535` invokes `pluginSettingsListener` for SSE push. `TestSetWebLinksConfigPreservesSiblings` GREEN. Real round-trip needs human UAT. |

**Score:** 5/5 truths verified at the source/automated level; 5 require human UAT for end-user flow confirmation (visual + real-OS + Tailscale).

### Deferred Items

Items not yet met but explicitly addressed as future work.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | OSC 8 hyperlink display-vs-href divergence triggers popover (SC-3 OSC 8 slice) | v3.3 (post-roadmap) | Plan B selected per 95-RESEARCH.md Wave 0 Spike Outcome: `IBufferCell.getHyperlinkId` not in @xterm/xterm@6.0.0 public typings. osc8 branch ships **dormant** in `LinkConfirmPopover.tsx:39-43` and web `showLinkConfirmPopover`. The presentation surface is complete; only the trigger is dormant. A v3.3 wiring-only PR can flip the slice GREEN by adding a custom OSC 8 handler via `Terminal.parser.registerOscHandler(8, ...)`. |

### Required Artifacts

| Artifact                                                                  | Expected                                                                       | Status     | Details                                                                          |
| ------------------------------------------------------------------------- | ------------------------------------------------------------------------------ | ---------- | -------------------------------------------------------------------------------- |
| `frontend/src/lib/urlSafety.ts`                                           | 7 exports (ALLOWED_SCHEMES, RiskKind, isAllowedScheme, osc8Mismatch, hasIDN, isTypoSquat, getRisk) + 30-entry TYPOSQUAT_LIST | ✓ VERIFIED | All 7 exports present; TYPOSQUAT_LIST has 30 entries; 20/20 tests GREEN |
| `frontend/src/lib/openLink.ts`                                            | openLink + isModifierPressed + ModifierMode; literal `'_blank', 'noopener,noreferrer'` | ✓ VERIFIED | All 3 exports; literal options string at line 57; 12/12 tests GREEN |
| `frontend/src/components/LinkConfirmPopover.tsx`                          | Portal-rendered ARIA dialog with osc8/idn/typosquat copy; textContent (no innerHTML) | ✓ VERIFIED | createPortal, role="dialog", aria-modal="true"; no dangerouslySetInnerHTML; 11/11 tests GREEN |
| `frontend/src/components/TerminalPanel.tsx` (modified)                    | WebLinksAddon imported + handler with allowlist/modifier/risk/openLink + popover render | ✓ VERIFIED | All wiring patterns present; dep array correct (webLinks IN, webLinksConfig OUT — Pitfall #8); 25/25 web-links tests GREEN |
| `internal/daemon/plugin_settings.go`                                      | WebLinksConfig struct + nested field on PluginSettings + platform/all-confirm-on defaults | ✓ VERIFIED | type WebLinksConfig at line 33; default at lines 84-87; TestDefaultPluginSettings GREEN |
| `internal/daemon/engine.go`                                               | (*SessionEngine).SetWebLinksConfig sub-key writer | ✓ VERIFIED | Lines 526-535; mutex/saveSettingsToDisk/listener pattern; TestSetWebLinksConfigPreservesSiblings GREEN |
| `internal/daemon/api.go`                                                  | PATCH /settings/web-links-config + handleSetWebLinksConfig | ✓ VERIFIED | Route registered at line 77; handler at line 590 |
| `internal/daemon/client.go`                                               | DaemonClient.SetWebLinksConfig HTTP wrapper                                    | ✓ VERIFIED | Method exists (added in Plan 95-05 deviation #3) |
| `app.go`                                                                  | (*App).SetWebLinksConfig — client write + readback + EventsEmit settings:plugins | ✓ VERIFIED | Line 541; client call at 545; EventsEmit at 557 |
| `frontend/src/wailsjs/go/main/App.d.ts`                                   | export function SetWebLinksConfig(arg1: daemon.WebLinksConfig): Promise<void> | ✓ VERIFIED | Line 137 |
| `frontend/src/wailsjs/go/main/App.js`                                     | Runtime stub for SetWebLinksConfig                                              | ✓ VERIFIED | Line 83 — `Call('main.App.SetWebLinksConfig', [cfg])` (project convention; Plan 95-05 deviation #2 documented) |
| `frontend/src/wailsjs/go/models.ts`                                       | daemon.WebLinksConfig class + webLinksConfig field on PluginSettings           | ✓ VERIFIED | Hand-edit per Phase 92 STATE.md pin (Plan 95-01) |
| `web/vendor/xterm/addons/addon-web-links.js`                              | Byte-identical UMD copy from frontend/node_modules                              | ✓ VERIFIED | `cmp -s` confirms byte-identical |
| `web/embed.go`                                                            | //go:embed extended to include addon-web-links.js                              | ✓ VERIFIED | Line 10 includes `vendor/xterm/addons/addon-web-links.js` |
| `web/vendor/xterm/VERSION`                                                | 7 entries including @xterm/addon-web-links@0.12.0                              | ✓ VERIFIED | 7 lines, addon-web-links@0.12.0 present |
| `web/terminal.html`                                                       | <script src=...addon-web-links.js> + #link-confirm-popover dialog DOM           | ✓ VERIFIED | Line 49 script tag; line 54 popover DOM with role/aria-modal |
| `web/assets/terminal.js`                                                  | applyPluginConfig webLinks arm + inline urlSafety/openLink + plain-DOM popover | ✓ VERIFIED | 8 helper functions; namespace constructor `new WebLinksAddon.WebLinksAddon(...)` (Pitfall #7); literal noopener,noreferrer at line 338; NO BrowserOpenURL |
| `web/assets/terminal.css`                                                 | #link-confirm-popover styles + reduced-motion guard                            | ✓ VERIFIED | Block present; @media reduced-motion targets popover class |
| `internal/webserver/vendor_drift_test.go`                                 | Min-count guard 6 → 7                                                          | ✓ VERIFIED | TestXtermVendorVersionsMatchPnpmLock GREEN at 7 |
| `internal/webserver/web_links_test.go`                                    | 3 GREEN tests (assets reachable, no current-tab nav, opener literal)           | ✓ VERIFIED | All 3 GREEN; no t.Skip remaining |
| `frontend/e2e/web-links-live-toggle.spec.ts`                              | Playwright spec (test.skip until plumbed) with documented walks                | ✓ VERIFIED (test.skip) | File exists; 3 documented test.skip cases per plan deviation (manual UAT is verification path until Playwright plumbed for web embed) |
| `.planning/phases/95-web-links-addon-security-hardening/95-DESKTOP-UAT.md` | Per-OS Cmd/Ctrl-click + Cyrillic + Plan A/B + window.opener gate              | ✓ VERIFIED | 8 sections; mentions U+043E Cyrillic; Plan A/B branches present |
| `.planning/phases/95-web-links-addon-security-hardening/95-WEB-UAT.md`     | Tailscale + iPad Safari + dev-browser skill walkthrough                        | ✓ VERIFIED | 9 sections; mentions iPad Safari, Tailscale, dev-browser |

### Key Link Verification

| From                                          | To                                                                  | Via                                       | Status     | Details                                                                          |
| --------------------------------------------- | ------------------------------------------------------------------- | ----------------------------------------- | ---------- | -------------------------------------------------------------------------------- |
| TerminalPanel webLinks hot-swap arm           | WebLinksAddon constructor with custom handler (LNK-01..04)          | `useEffect dep [..., pluginConfig?.webLinks, ...]` | ✓ WIRED    | `new WebLinksAddon(handler, {hover, leave})` at TerminalPanel.tsx:415; deps include `pluginConfig?.webLinks` at line 448 |
| TerminalPanel handler                         | openLink (LNK-04 platform-aware opener)                             | `import from ../lib/openLink`             | ✓ WIRED    | Import at line 17; call at line 413 (non-risky path) and 663 (Continue path)    |
| TerminalPanel risky-click branch              | <LinkConfirmPopover> portal                                          | `linkConfirmState useState + setLinkConfirmState` | ✓ WIRED | Render at line 655-668; Continue calls openLink+clear; Cancel clears |
| (*App).SetWebLinksConfig                      | engine.SetWebLinksConfig via DaemonClient.SetWebLinksConfig PATCH    | client → HTTP → daemon engine             | ✓ WIRED    | app.go:545 calls a.client.SetWebLinksConfig(cfg) → PATCH /settings/web-links-config → engine sub-key writer |
| engine.SetWebLinksConfig                      | pluginSettingsListener (SSE broadcast)                               | engine mutex + listener slot              | ✓ WIRED    | engine.go:531-534 captures listener post-Unlock and invokes (Phase 93 PLUG-04 SSE channel) |
| App.tsx EventsOn('settings:plugins')          | setPluginConfig → TerminalPanel prop drill                           | Wails EventsEmit                          | ✓ WIRED    | App.tsx:351; pluginConfig threaded to TerminalPanel at line 923 |
| web/assets/terminal.js applyPluginConfig      | new WebLinksAddon.WebLinksAddon(handler, options) (UMD namespace)    | window.WebLinksAddon UMD global           | ✓ WIRED    | Line 522 — Pitfall #7 namespace constructor verified |
| web openLink helper                           | window.open(url, '_blank', 'noopener,noreferrer')                    | browser-native                            | ✓ WIRED    | Line 338 literal; NEVER BrowserOpenURL on web (asserted by TestTerminalJS_WebLinksOpener) |

### Data-Flow Trace (Level 4)

| Artifact                                    | Data Variable                | Source                                                    | Produces Real Data | Status      |
| ------------------------------------------- | ---------------------------- | --------------------------------------------------------- | ------------------ | ----------- |
| TerminalPanel WebLinksAddon                 | uri (passed to handler)      | xterm WebLinksAddon urlRegex match against terminal output | Yes (real terminal text) | ✓ FLOWING  |
| LinkConfirmPopover                          | url, risk                    | linkConfirmState set from handler with real uri + getRisk result | Yes               | ✓ FLOWING  |
| TerminalPanel pluginConfig                  | pluginConfig (incl. webLinksConfig) | App.tsx EventsOn settings:plugins from real PluginSettings | Yes (full struct round-trips) | ✓ FLOWING  |
| Engine WebLinksConfig                       | e.pluginSettings.WebLinksConfig | SetWebLinksConfig + saveSettingsToDisk to disk + load on next start | Yes (TestSetWebLinksConfigPreservesSiblings asserts disk persistence) | ✓ FLOWING  |
| Web terminal.js currentWebLinksConfig       | currentWebLinksConfig        | applyPluginConfig from SSE settings:plugins frame          | Yes               | ✓ FLOWING  |

### Behavioral Spot-Checks

| Behavior                                                          | Command                                                                                                                                                | Result      | Status   |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------- | -------- |
| All Phase 95 frontend tests pass                                  | `cd frontend && pnpm exec vitest run src/lib/__tests__/urlSafety.test.ts src/lib/__tests__/openLink.test.ts src/components/__tests__/LinkConfirmPopover.test.tsx src/components/__tests__/TerminalPanel.web-links.test.tsx` | 68/68 passed in 4 files | ✓ PASS  |
| App plugin-event prop drill test passes (webLinksConfig nested)   | `pnpm exec vitest run src/__tests__/App.plugin-event.test.tsx`                                                                                          | 13/13 passed | ✓ PASS  |
| Daemon Phase 95 Go tests pass                                     | `go test ./internal/daemon/ -run "TestSetWebLinksConfigPreservesSiblings\|TestPluginSettingsMigration_WebLinksConfig\|TestDefaultPluginSettings"`        | 3/3 PASS    | ✓ PASS  |
| Webserver Phase 95 Go tests pass                                  | `go test ./internal/webserver/ -run "TestAssets_AddonWebLinks\|TestSecurity_NoCurrentTabNavigation\|TestTerminalJS_WebLinksOpener\|TestXtermVendorVersionsMatchPnpmLock"` | 4/4 PASS | ✓ PASS  |
| Full daemon + webserver Go suite green                            | `go test ./internal/daemon/... ./internal/webserver/... -count=1`                                                                                       | both `ok`   | ✓ PASS  |
| Vendored UMD byte-identical to source                             | `cmp -s web/vendor/xterm/addons/addon-web-links.js frontend/node_modules/@xterm/addon-web-links/lib/addon-web-links.js`                                 | byte-identical | ✓ PASS |
| Full frontend test sweep — only documented pre-existing failures  | `pnpm exec vitest run`                                                                                                                                 | 687 passed / 20 failed; 20 failures are Sidebar.test.tsx localStorage env issue (pre-existing per MEMORY.md "Verify test-env before declaring failure" + all 6 plan SUMMARYs) | ✓ PASS (no new regressions) |

### Requirements Coverage

| Requirement | Source Plans                                              | Description                                                                                                       | Status      | Evidence                                                                                                                                                                                       |
| ----------- | --------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| LNK-01      | 95-01, 95-02, 95-04, 95-06                                | Plain https/http/mailto detected + clickable; non-allowlist schemes never clickable                                | ✓ SATISFIED | ALLOWED_SCHEMES const + isAllowedScheme + handler defense-in-depth + web mirror; 20+ tests GREEN                                                                                                |
| LNK-02      | 95-01, 95-02, 95-04, 95-06                                | Cmd/Ctrl-click required; single-click never activates; hover tooltip                                              | ✓ SATISFIED (auto) / ? HUMAN | isModifierPressed + handler check + setAttribute('title')/removeAttribute. Real-OS modifier confirmation needs human UAT.                                                                       |
| LNK-03      | 95-01, 95-02, 95-03, 95-04, 95-06                         | OSC 8 mismatch / IDN / typosquat trigger popover                                                                  | ⚠️ PARTIAL  | IDN + typosquat: ✓. **OSC 8 mismatch: deferred to v3.3** per Plan B (Wave 0 spike). osc8 popover branch ships dormant; LNK-OSC8-FUT-01 future req tracked in plan SUMMARYs but **NOT yet in REQUIREMENTS.md ## Future Requirements** (see WARNING below). |
| LNK-04      | 95-01, 95-02, 95-04, 95-06                                | IDN/Punycode + typosquat trigger click-confirmation popover (REQUIREMENTS.md mapping)                              | ✓ SATISFIED | hasIDN + isTypoSquat + getRisk + LinkConfirmPopover with idn/typosquat copy + TerminalPanel handler routes risky → popover. Web mirror in plain-DOM popover.                                  |
| LNK-05      | 95-01, 95-02, 95-04, 95-05, 95-06                         | Desktop BrowserOpenURL; Web noopener,noreferrer; no current-tab nav                                               | ✓ SATISFIED | openLink Wails branch + web window.open literal; TestSecurity_NoCurrentTabNavigation regression gate                                                                                            |
| LNK-06      | 95-01, 95-04, 95-05                                       | Settings toggle applies live to all open terminals                                                                | ✓ SATISFIED (auto) / ? HUMAN | Hot-swap arm + EventsEmit('settings:plugins') + SSE pluginSettingsListener + TestSetWebLinksConfigPreservesSiblings GREEN. Real round-trip needs human UAT.                                    |

All 6 requirement IDs are claimed by ≥1 plan and have implementation evidence. NO orphaned requirements detected.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | — | — | All forbidden patterns checked: no `dangerouslySetInnerHTML` in popover, no `location.href = ` in openLink/terminal.js, no `BrowserOpenURL` in web/assets/terminal.js, no `expect.fail` remaining, no `t.Skip` remaining. |

### Human Verification Required

5 items require human UAT (deferred to 95-DESKTOP-UAT.md and 95-WEB-UAT.md):

#### 1. Cross-OS Modifier Click + Hover Tooltip (LNK-02 / SC-2)

**Test:** macOS Cmd-click activates link; Linux/Windows Ctrl-click activates link; single-click never activates; hover shows resolved href as title attribute.
**Expected:** Cmd-click on macOS opens URL in default browser via Wails BrowserOpenURL; single-click does nothing; hover tooltip shows resolved href including any future OSC 8 mismatches.
**Why human:** Real-OS modifier semantics + Wails native browser invocation cannot be automated in jsdom. Runbook: 95-DESKTOP-UAT.md §2.

#### 2. Cyrillic Spoof Popover (LNK-03 / SC-3)

**Test:** Run `echo https://gооgle.com` (Cyrillic U+043E for the two 'o's after 'g') in a terminal session; Cmd-click the URL.
**Expected:** Click-confirmation popover renders with idn copy ("contains internationalized characters that can spoof familiar domains") + full resolved URL displayed; Continue invokes openLink → BrowserOpenURL; Cancel dismisses.
**Why human:** Visual popover behavior + real terminal output rendering not feasible in jsdom (xterm canvas/WebGL). Runbook: 95-DESKTOP-UAT.md §3.

#### 3. Web window.opener Defense (LNK-04 / SC-4)

**Test:** Open a Tailscale-served terminal session in a real browser; Cmd-click an https URL.
**Expected:** New tab opens; DevTools console `window.opener` evaluates to `null`; original tab does not navigate.
**Why human:** Requires real browser + Tailscale-served session; window.opener pivot can only be observed in a real browser. Runbook: 95-WEB-UAT.md §5.

#### 4. Live Toggle Without Session Restart (LNK-06 / SC-5)

**Test:** With a session running and a URL visible, disable web-links in Settings → links lose underline on next refresh; re-enable → links return underlined; no session restart.
**Expected:** settings:plugins event propagates via Wails EventsEmit (desktop) and SSE /api/plugin-config/stream (web); applyPluginConfig disposes/loads addon without restart.
**Why human:** Real settings:plugins SSE round-trip + DOM observation. Runbook: 95-DESKTOP-UAT.md §6 + 95-WEB-UAT.md §7.

#### 5. iPad Safari Tailscale Walkthrough (Phase 99 release gate co-verification)

**Test:** Open Tailscale URL on iPad Safari with paired keyboard; walk LNK-01..05 chain (scheme allowlist, modifier-click via paired keyboard Cmd, hover tooltip, popover for Cyrillic, window.open).
**Expected:** All gates fire correctly on iPad Safari; no console errors; popover renders; window.open spawns new tab.
**Why human:** Cross-OS / iOS Safari quirks not testable from CI; iPad keyboard event semantics; touch-vs-click differences. Runbook: 95-WEB-UAT.md §9.

### Gaps Summary

**No automated gaps found.** All implementation surfaces verified at source/test level:

- All 6 requirement IDs (LNK-01..LNK-06) have implementation evidence with passing tests.
- All 5 ROADMAP Success Criteria have automated coverage at the source/regression-test level.
- 20 Sidebar.test.tsx test failures are documented pre-existing localStorage env issues (per project MEMORY.md and all 6 plan SUMMARYs); not caused by Phase 95.

**5 human-UAT items** are intentional and listed above — visual + real-OS + Tailscale + iPad behaviors that cannot be automated in CI. Status is `human_needed`, not `passed`, because the Success Criteria explicitly require human-observable end-user flows (Cmd-click activates, popover renders, link opens in real browser, toggle applies without restart).

### Warnings (informational; do not block status)

1. **LNK-OSC8-FUT-01 not in REQUIREMENTS.md ## Future Requirements** — All 6 plan SUMMARYs (95-01 through 95-06) state that an `LNK-OSC8-FUT-01` row will be added to `.planning/REQUIREMENTS.md ## Future Requirements` to track the v3.3 OSC 8 mismatch wiring. The current REQUIREMENTS.md `## Future Requirements` section (line 108) lists SER-FUT-01, SRC-FUT-01, PROG-FUT-01, GRAPHEMES-FUT-01 but **NOT LNK-OSC8-FUT-01**. The deferral is well-documented in plan SUMMARYs, in `95-RESEARCH.md ## Wave 0 Spike Outcome`, in `95-DESKTOP-UAT.md §5b`, and is grep-discoverable via the dormant osc8 branch in popovers. **Recommendation:** add the row to REQUIREMENTS.md to close the bookkeeping loop. This does not block phase verification — the deferral itself is sound and traceable.

2. **Playwright e2e is `test.skip`** — `frontend/e2e/web-links-live-toggle.spec.ts` exists with 3 documented `test.skip` cases. Per Plan 95-06 deviation #3, real-browser e2e infrastructure for the web embed is not plumbed (Phase 94 used chromedp, not Playwright). Manual UAT (95-WEB-UAT.md) is the v3.2 verification path. The documented bodies are a turnkey checklist for a future Playwright-plumbed PR. Acceptable per plan boundary; not a regression.

---

_Verified: 2026-05-06T20:55:00Z_
_Verifier: Claude (gsd-verifier)_
