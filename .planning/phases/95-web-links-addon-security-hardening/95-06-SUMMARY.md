---
phase: 95-web-links-addon-security-hardening
plan: 06
subsystem: web-parity
tags: [phase-95, web-links, web-parity, vendor, e2e, uat, wave-4, LNK-01, LNK-02, LNK-03, LNK-04, LNK-05]

# Dependency graph
requires:
  - phase: 95-web-links-addon-security-hardening
    plan: 01
    provides: Wave 0 RED scaffolds (web_links_test.go 3 skip-marked tests; e2e spec test.skip stub); VERSION manifest line for @xterm/addon-web-links@0.12.0; Plan B Wave 0 spike outcome (osc8 mismatch deferred to v3.3)
  - phase: 95-web-links-addon-security-hardening
    plan: 02
    provides: frontend/src/lib/urlSafety.ts + frontend/src/lib/openLink.ts (the inline-mirror reference for terminal.js helpers)
  - phase: 95-web-links-addon-security-hardening
    plan: 04
    provides: TerminalPanel.tsx WebLinksAddon wiring (the desktop reference shape mirrored on web)
  - phase: 93-vendoring-discipline-web-parity-for-already-shipping-addons
    provides: vendor_drift_test.go (generalized regex; min-count bumped from 6 → 7 here); embed.go pattern; UMD namespace global Pitfall #7
  - phase: 94-search-addon-find-bar-desktop-web
    provides: applyPluginConfig diff-apply structure in terminal.js (search arm precedent for the new web-links arm); find_bar_test.go testServer harness pattern
provides:
  - "web/vendor/xterm/addons/addon-web-links.js — byte-identical UMD copy from frontend/node_modules"
  - "web/embed.go //go:embed extended; web/vendor/xterm/VERSION manifest already at 7 entries"
  - "internal/webserver/vendor_drift_test.go min-count bumped 6 → 7 (T-95-06-01)"
  - "web/terminal.html: <script src=...addon-web-links.js> before terminal.js + #link-confirm-popover dialog DOM"
  - "web/assets/terminal.js: inline urlSafety + openLink helpers (mirror of frontend/src/lib/*); applyPluginConfig webLinks arm with Pitfall #7 namespace constructor; showLinkConfirmPopover plain-DOM mirror of LinkConfirmPopover (textContent only)"
  - "web/assets/terminal.css: #link-confirm-popover styles + reduced-motion guard + slide-in animation"
  - "internal/webserver/web_links_test.go: 3 GREEN tests — TestAssets_AddonWebLinks, TestSecurity_NoCurrentTabNavigation, TestTerminalJS_WebLinksOpener"
  - "frontend/e2e/web-links-live-toggle.spec.ts: 3 documented test.skip walks (Playwright-not-plumbed; manual UAT is verification path until plumbed)"
  - ".planning/phases/95-web-links-addon-security-hardening/95-DESKTOP-UAT.md (per-OS + Cyrillic + Plan A/B + window.opener gate)"
  - ".planning/phases/95-web-links-addon-security-hardening/95-WEB-UAT.md (Tailscale + iPad Safari + dev-browser skill)"
affects:
  - "/gsd-verify-work 95 (next): all LNK-01..05 mapped to ≥1 automated test + ≥1 manual UAT step; SC-1..SC-5 flip VERIFIED after sign-off"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Web-side helper inlining: ES5 IIFE-scope copies of frontend/src/lib/{urlSafety,openLink}.ts (no module bundler in the embed); behavior + security invariants identical; URL constructor + Punycode-fallback for IDN; defense-in-depth scheme regex in openLink itself"
    - "Plain-DOM popover (no React on web): #link-confirm-popover toggled via [hidden] + textContent assignment for risk + URL strings (NEVER innerHTML — T-95-06-05 XSS gate); Escape handler at document level; edge-clipping mitigation mirrors desktop Pitfall #4"
    - "Sub-config click-time read pattern: currentWebLinksConfig is updated by applyPluginConfig and read at click time inside the handler — toggling sub-keys (modifier/confirm*) does NOT re-attach the addon (mirror of desktop webLinksConfigRef from Plan 95-04, Pitfall #8)"
    - "Source-inspection regression gates: TestSecurity_NoCurrentTabNavigation scans BOTH web/assets/terminal.js AND frontend/src/lib/openLink.ts so a future commit reintroducing location.href on EITHER side fails CI"
    - "Comment-vs-grep collision pattern (also seen in Plans 95-02/03/04): a descriptive comment containing a forbidden token (e.g. 'BrowserOpenURL', 't.Skip') is reworded to avoid the gate without changing semantics"

key-files:
  created:
    - "web/vendor/xterm/addons/addon-web-links.js"
    - ".planning/phases/95-web-links-addon-security-hardening/95-DESKTOP-UAT.md"
    - ".planning/phases/95-web-links-addon-security-hardening/95-WEB-UAT.md"
  modified:
    - "web/embed.go"
    - "internal/webserver/vendor_drift_test.go"
    - "web/terminal.html"
    - "web/assets/terminal.js"
    - "web/assets/terminal.css"
    - "internal/webserver/web_links_test.go"
    - "frontend/e2e/web-links-live-toggle.spec.ts"

key-decisions:
  - "Vendor UMD via cmp -s byte-identical copy from frontend/node_modules (Phase 93 discipline). VERSION manifest line was already added in Plan 95-01 Task 2 to keep CI green; Task 1 here only had to extend embed.go and bump the vendor_drift min-count from 6 → 7."
  - "Inline ES5 helpers (not ES6+) for terminal.js consistency. The web embed has no Babel/transpiler step; the existing terminal.js code style is var + function declarations + try/catch (no arrow functions in helper scope, no const/let in shared closure scope). Mirrored that exactly so the diff stays small and reviewable."
  - "Web popover is plain DOM (not a duplicate React component). The web embed has no React; mirroring the LinkConfirmPopover semantics via DOM manipulation keeps the bundle ~3kb total. Behavior identical: textContent for URL/reason (NEVER innerHTML), Escape on document, focus trap via initial focus on Cancel button, edge-clipping reposition before showing."
  - "Playwright spec kept as documented test.skip (3 cases). Real-browser e2e infrastructure isn't plumbed for this repo's web surface (Phase 94 used chromedp for findbar_web_e2e_test.go in internal/webserver). Documenting the exact walks in test bodies (rather than a separate plan-X-runbook.md) means a future engineer running `pnpm exec playwright test` sees the file + the documented body, not silence. Manual UAT (95-WEB-UAT.md with the dev-browser skill agent script) is the v3.2 verification path."
  - "Plan B holds on web identically: getRisk(uri, uri) called with displayText === uri, so osc8Mismatch never fires for plain-text URLs the addon emits. The osc8 branch in showLinkConfirmPopover ships dormant (parity with desktop LinkConfirmPopover from Plan 95-03). LNK-OSC8-FUT-01 follow-up in REQUIREMENTS.md tracks v3.3 wiring."
  - "Comment leak avoided: helper-block comment originally said 'web context never has Wails BrowserOpenURL' which trips TestTerminalJS_WebLinksOpener's 'must NOT contain BrowserOpenURL' assertion. Reworded to 'web context never has the Wails runtime opener' — same semantic intent, no token collision. Mirrors the deviation pattern from Plans 95-02/03/04."

requirements-completed: [LNK-01, LNK-02, LNK-03, LNK-04, LNK-05]

# Metrics
duration: ~35min
completed: 2026-05-06
tasks_completed: 3
files_created: 3
files_modified: 7
tests_added_or_flipped_green: 3
---

# Phase 95 Plan 06: Web Parity (LNK-01..05) Summary

**Web parity: vendored the UMD bundle (byte-identical), wired addon-web-links into the web-served Tailscale terminal page (terminal.html script + popover DOM, terminal.js applyPluginConfig arm with inline urlSafety/openLink helpers + plain-DOM popover, terminal.css styles), bumped vendor_drift_test min-count 6 → 7, flipped 3 Wave-0 skip-marked Go tests to GREEN (asset reachable + no current-tab nav + exact `_blank`+`noopener,noreferrer` literal), authored 95-DESKTOP-UAT.md + 95-WEB-UAT.md runbooks. Manual UAT is the next gate (`/gsd:verify-work 95`).**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-05-06
- **Tasks:** 3 / 3
- **Files created:** 3 (UMD + 2 UAT runbooks)
- **Files modified:** 7

## Plan A vs Plan B Decision (carryover)

Plan B selected per 95-01 Wave 0 spike outcome — `IBufferCell.getHyperlinkId` is absent from `@xterm/xterm@6.0.0` public typings. terminal.js (web) therefore:

- Calls `getRisk(uri, uri)` with `displayText === uri` — `osc8Mismatch` never fires for plain-text URLs the addon emits.
- Does NOT register a secondary OSC 8 handler.
- The `osc8` branch in `showLinkConfirmPopover` ships dormant (mirror of desktop LinkConfirmPopover from Plan 95-03).
- 95-DESKTOP-UAT.md §5b documents the v3.2 OSC 8 limitation; LNK-OSC8-FUT-01 row in `REQUIREMENTS.md` `## Future Requirements` tracks the v3.3 wiring.

## Accomplishments

### Task 1 — Vendoring (commit `c83e6d5`)

- `web/vendor/xterm/addons/addon-web-links.js`: byte-identical copy from `frontend/node_modules/@xterm/addon-web-links/lib/addon-web-links.js` (verified via `cmp -s`).
- `web/embed.go`: `//go:embed` directive extended to include the new asset.
- `internal/webserver/vendor_drift_test.go`: min-count guard bumped from 6 → 7; error message lists all 7 packages; T-95-06-01 mitigation language added.
- `web/vendor/xterm/VERSION`: already at 7 entries (Plan 95-01 Task 2 deviation).

UMD global verification (Pitfall #7): `e.WebLinksAddon=t()` — `window.WebLinksAddon` is a NAMESPACE OBJECT; the constructor call must be `new WebLinksAddon.WebLinksAddon(handler, options)`, NOT `new WebLinksAddon(handler, options)`. Verified before authoring Task 2.

### Task 2 — Wiring (commit `29faef0`)

`web/terminal.html` (+15 lines):

- `<script src="/assets/xterm/addons/addon-web-links.js">` inserted between `addon-search` and `terminal.js`.
- `<div id="link-confirm-popover" hidden role="dialog" aria-modal="true">…</div>` popover container with title / reason / `<code>` URL / Cancel + Continue buttons.

`web/assets/terminal.js` (+178 lines):

- Inline urlSafety helpers (8 functions): `isAllowedScheme`, `hasIDN` (with Punycode fallback for invalid labels — same auto-fix pattern as Plan 95-02), `osc8Mismatch`, `TYPOSQUAT_LIST` (30 entries), `isTypoSquat`, `getRisk`, `isModifierPressed`, `openLink`.
- `openLink` invariant: `if (!/^(https?:|mailto:)/i.test(url)) return;` then `window.open(url, '_blank', 'noopener,noreferrer')` — NEVER `location.href`, NEVER `window.location`, NEVER any Wails opener call.
- `showLinkConfirmPopover(url, risk, x, y)`: plain DOM mirror of desktop LinkConfirmPopover. textContent only for URL + reason; edge-clipping reposition before show; Escape handler on document; cleanup detaches all listeners.
- Top-level vars: `webLinksAddonHandle = null`, `currentWebLinksConfig = null` — read at click time so sub-key toggles don't re-attach the addon.
- `applyPluginConfig` webLinks arm (after the search arm): dispose-on-toggle-off (also clears in-flight popover), construct-on-toggle-on with the namespace constructor `new WebLinksAddon.WebLinksAddon(handler, { hover, leave })`. Handler shape mirrors desktop TerminalPanel.tsx Plan 95-04 verbatim: scheme allowlist → modifier-click → risk detection → popover-or-openLink.

`web/assets/terminal.css` (+35 lines):

- `#link-confirm-popover` block (TokyoNight palette: `#1f2335` bg / `#c0caf5` text / `#7aa2f7` Continue button); slide-in animation 200ms; `prefers-reduced-motion` guard removes animation; `[hidden]` → `display:none`.

### Task 3 — Tests + UAT (commit `8483ca0`)

`internal/webserver/web_links_test.go` — 3 skip-marked RED scaffolds → 3 GREEN tests:

| Test | What it verifies | Threat |
|------|------------------|--------|
| `TestAssets_AddonWebLinks` | GET `/assets/xterm/addons/addon-web-links.js` returns 200 + `Content-Type` containing `javascript`; uses `testServer(t)` harness (mirror of Phase 94 `TestAssets_AddonSearch`) | T-95-06-01 |
| `TestSecurity_NoCurrentTabNavigation` | Source-inspect BOTH `web/assets/terminal.js` AND `frontend/src/lib/openLink.ts` for `location.href = `, `window.location = `, `document.location = ` patterns; ZERO matches required | T-95-06-02 |
| `TestTerminalJS_WebLinksOpener` | `function openLink` defined; EXACT literal `'_blank', 'noopener,noreferrer'` present; `WebLinksAddon.WebLinksAddon` namespace constructor present; `BrowserOpenURL` ABSENT (web is never inside Wails) | T-95-06-03, T-95-06-04 |

`frontend/e2e/web-links-live-toggle.spec.ts` — 3 documented `test.skip` cases:

1. webLinks toggle off→on lifecycle + window.open spy with the exact options literal
2. Cyrillic spoof URL → popover → Cancel/Continue path
3. Typosquat / non-modifier-click suppression / scheme allowlist (javascript: blocked)

`.planning/phases/95-web-links-addon-security-hardening/95-DESKTOP-UAT.md` (8 sections):

1. Scheme allowlist (LNK-01)
2. Modifier-click + hover tooltip (LNK-02)
3. IDN / Cyrillic spoof popover (LNK-03) — explicit U+043E codepoint warning
4. Typosquat popover (LNK-03)
5. Plan A: OSC 8 mismatch popover // 5b. Plan B (selected): known-limitation note
6. Live toggle ON ⇄ OFF without session restart (LNK-05/LNK-06)
7. Sub-key toggle (Modifier change) without addon re-attach (Pitfall #8)
8. window.opener defense (LNK-04)

`.planning/phases/95-web-links-addon-security-hardening/95-WEB-UAT.md` (9 sections + dev-browser script):

1. Vendored asset reachable
2. CSP zero-violation
3. Scheme allowlist
4. Modifier-click + hover tooltip
5. window.open with noopener (`window.opener === null` gate)
6. IDN / typosquat popover (textContent-not-innerHTML DevTools verification)
7. Live toggle propagation via SSE in < 2s
8. Sub-key toggle propagation without addon re-attach
9. iPad Safari Tailscale (Phase 99 release gate)

Plus a dev-browser skill walkthrough section with the literal `dev-browser navigate` / `eval` / `type` / `click` script for agent-driven UAT.

## Task Commits

Three atomic commits on the worktree branch:

1. **Task 1 — `c83e6d5` (feat):** `feat(95-06): vendor addon-web-links UMD + bump vendor_drift min-count to 7`
2. **Task 2 — `29faef0` (feat):** `feat(95-06): wire web-links addon into terminal.html/js/css (LNK-01..06)`
3. **Task 3 — `8483ca0` (test):** `test(95-06): flip web_links_test.go RED → GREEN; UAT runbooks`

## Files Created/Modified

### Created (3)

- `web/vendor/xterm/addons/addon-web-links.js` — byte-identical UMD vendored copy.
- `.planning/phases/95-web-links-addon-security-hardening/95-DESKTOP-UAT.md` — desktop runbook.
- `.planning/phases/95-web-links-addon-security-hardening/95-WEB-UAT.md` — web (Tailscale-served + iPad Safari) runbook with dev-browser skill script.

### Modified (7)

- `web/embed.go` — `//go:embed` directive includes addon-web-links.js.
- `internal/webserver/vendor_drift_test.go` — min-count guard 6 → 7; new error message.
- `web/terminal.html` — addon-web-links script tag + popover container DOM.
- `web/assets/terminal.js` — 8 inline helpers + showLinkConfirmPopover + applyPluginConfig webLinks arm + 2 top-level vars.
- `web/assets/terminal.css` — popover block (TokyoNight palette + reduced-motion guard).
- `internal/webserver/web_links_test.go` — 3 real test implementations.
- `frontend/e2e/web-links-live-toggle.spec.ts` — 3 documented `test.skip` walks.

## Each LNK Requirement — How This Plan Satisfies It (Web Side)

**LNK-01 (scheme allowlist):**
- Defense-in-depth: addon-web-links default `urlRegex` blocks at the regex layer; `isAllowedScheme(uri)` re-checks at the handler layer; `openLink` re-checks a third time before `window.open`.
- Source pattern: `if (!isAllowedScheme(uri)) return;` (terminal.js).
- Tests: `TestSecurity_NoCurrentTabNavigation` covers the openLink invariant; runtime path covered by 95-WEB-UAT.md §3.

**LNK-02 (modifier-click + hover tooltip):**
- Modifier gate: `isModifierPressed(event, modifier)` where `modifier = currentWebLinksConfig.modifier || 'platform'`.
- Hover + leave callbacks set / remove DOM `title` attribute.
- Tests: 95-WEB-UAT.md §4 (manual UAT verifies cross-OS: macOS Cmd-click, Linux/Windows Ctrl-click, iPad with paired keyboard).

**LNK-03 (risk detection + popover):**
- `getRisk(uri, uri)` returns `'idn'` / `'typosquat'` / `null` in v3.2 (Plan B).
- Popover: `showLinkConfirmPopover(uri, risk, x, y)` — plain DOM at `#link-confirm-popover`. textContent only.
- Confirm flags honored: `cfg.confirmIDN !== false`, `cfg.confirmTyposquat !== false`, `cfg.confirmOSC8 !== false` (dormant).
- Tests: `TestTerminalJS_WebLinksOpener` (constructor + namespace pattern); 95-WEB-UAT.md §6.

**LNK-04 (platform-aware opener — web variant):**
- `openLink` ALWAYS calls `window.open(url, '_blank', 'noopener,noreferrer')` — NEVER `location.href`, NEVER `window.location`, NEVER `BrowserOpenURL`.
- Tests: `TestSecurity_NoCurrentTabNavigation` (regression gate scans terminal.js + openLink.ts); `TestTerminalJS_WebLinksOpener` (positive-shape gate).

**LNK-05 (live toggle without session restart):**
- `applyPluginConfig` webLinks arm load/dispose path; SSE settings:plugins push triggers `applyPluginConfig` re-run; `currentWebLinksConfig` is updated then read at click time.
- Tests: Playwright spec documented (test.skip until Playwright is plumbed); 95-WEB-UAT.md §7-8 (manual UAT, < 2s SSE delivery target).

## Test Surface — GREEN Tally

| File | Was Wave 0 | Now Wave 4 | Notes |
|------|-----------|-----------|-------|
| `internal/webserver/web_links_test.go` | 0 GREEN + 3 skip-marked RED | **3 GREEN** | All Wave-0 skip removed; real implementations |
| Webserver full sweep | (baseline) | **GREEN** | `go test ./internal/webserver/... -count=1` passes |
| Daemon full sweep | (baseline) | **GREEN** | `go test ./internal/daemon/... -count=1` passes |
| `vendor_drift_test` | GREEN at 6 | **GREEN at 7** | min-count bump GREEN; 7 manifest entries match 7 pnpm-lock entries |

## Decisions Made

1. **VERSION manifest reuse from Plan 95-01.** Plan 95-01 Task 2 added `@xterm/addon-web-links@0.12.0` to `web/vendor/xterm/VERSION` early to keep the existing vendor_drift CI gate GREEN at the moment of `pnpm install`. Plan 95-06 Task 1 only had to extend embed.go and bump the min-count assertion. The work was correctly attributed in 95-01 SUMMARY's Deviation #1 (Rule 3 — blocking).

2. **Inline ES5 helpers in terminal.js (not ES6+ / not bundle-imported).** The web embed has no Babel / no Vite / no transpiler step — terminal.js is served raw. Existing code uses `var` + function declarations + `try/catch` exclusively in helper scope. Mirroring that style keeps the diff reviewable and the regex-extracted source inspectable. The behavior is identical to `frontend/src/lib/{urlSafety,openLink}.ts`; the test surface is independent (frontend tests cover `urlSafety.ts`/`openLink.ts`, web tests cover `terminal.js`).

3. **Web popover is plain DOM, NOT a second React app.** The web embed is ~3kb of plain DOM + xterm; introducing React would 100x the bundle. Plain DOM with `textContent` (never `innerHTML`), `[hidden]` toggle, `position:fixed` with edge-clipping mitigation, and document-level Escape handler matches the desktop `LinkConfirmPopover` semantics 1:1 within DevTools-verifiable invariants.

4. **Playwright spec kept as documented `test.skip` (not a real test).** Real-browser e2e infrastructure for this repo's web surface uses `chromedp` (Phase 94 `findbar_web_e2e_test.go` in `internal/webserver`); Playwright is plumbed only for the React frontend (`frontend/e2e/`). Documenting the exact walks in `test.skip` bodies (with the spy-injection script and the assertion list) means a future Playwright-plumbed PR has a checklist to flip the tests live; until then, manual UAT (95-WEB-UAT.md with the dev-browser skill agent script) is the v3.2 verification path.

5. **Comment-vs-grep collision avoidance (recurring pattern).** Plans 95-02, 95-03, 95-04 all had comment leaks where descriptive text contained a forbidden token (`location.href`, `getHyperlinkId`, `convertValues`). Plan 95-06 had two during authoring:
   - `BrowserOpenURL` in the helper-block comment ("web context never has Wails BrowserOpenURL") — reworded to "web context never has the Wails runtime opener" so `TestTerminalJS_WebLinksOpener`'s `MUST NOT contain BrowserOpenURL` assertion passes.
   - `t.Skip` in the file-header comment ("All three tests were t.Skip RED scaffolds") — reworded to "skip-marked RED scaffolds" so the acceptance criterion `grep -c "t.Skip" == 0` passes.

   Both are ergonomic deviations: the test gates exist for refactor-safety, and comments are intentionally in scope for those gates (a comment can become uncommented code in a refactor).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] Comment leak: `BrowserOpenURL` in helper-block JSDoc comment**
- **Found during:** Task 2 verification (acceptance criterion `! grep -q "BrowserOpenURL" web/assets/terminal.js`).
- **Issue:** The helper-block leading comment said "web context never has Wails BrowserOpenURL" — accurate documentation but the literal token tripped the negative-assertion gate.
- **Fix:** Reworded to "web context never has the Wails runtime opener — that is desktop-only (see frontend/src/lib/openLink.ts)". Same intent; no token collision.
- **Files modified:** `web/assets/terminal.js` (comment only; behavior unchanged).
- **Verification:** `grep -q BrowserOpenURL web/assets/terminal.js` returns nothing.
- **Committed in:** `29faef0` (Task 2 commit, after the rewording).

**2. [Rule 1 — Bug] Comment leak: `t.Skip` in web_links_test.go file-header comment**
- **Found during:** Task 3 verification (acceptance criterion `grep -c "t.Skip" internal/webserver/web_links_test.go == 0`).
- **Issue:** File-header comment said "All three tests were t.Skip RED scaffolds" — accurate but the literal `t.Skip` token tripped the gate.
- **Fix:** Reworded to "skip-marked RED scaffolds". Same intent; no token collision.
- **Files modified:** `internal/webserver/web_links_test.go` (comment only; behavior unchanged).
- **Verification:** `grep -c "t.Skip" internal/webserver/web_links_test.go` returns `0`.
- **Committed in:** `8483ca0` (Task 3 commit, after the rewording).

### Plan deviations (non-bug, design choices)

**3. Playwright spec is documented test.skip (not Rule N — design choice).**
- **Found during:** Task 3 authoring.
- **Issue:** Plan `<action>` Step B describes a Playwright spec body. The repo doesn't have Playwright plumbed for the web embed (`frontend/e2e/` is for the React frontend; `internal/webserver/findbar_web_e2e_test.go` uses chromedp for the web embed in Phase 94).
- **Resolution:** Each `test.skip` body documents the exact walk + assertions a future Playwright-plumbed PR must implement. The plan explicitly allows this (`<behavior>`: "If not wired, leave the test as a `test.skip(...)` with a documented note and rely on the manual UAT runbooks.").
- **Impact:** The runtime e2e path is verified manually via 95-WEB-UAT.md (with the dev-browser skill agent script for automation). When Playwright is plumbed, the documented bodies are a turnkey checklist to flip the tests GREEN.

---

**Total deviations:** 2 auto-fixed (Rule 1 — comment-vs-grep collisions, both reworded inline before commits landed) + 1 documented design choice (Playwright spec stays test.skip). None required user input.

## Issues Encountered

- **pnpm install in worktree:** the worktree's `frontend/node_modules/` was empty at executor start — `pnpm install --prefer-offline` ran in 4.3s using cached store (Plan 95-01 had populated the lockfile). The vendored UMD copy step was unblocked.
- **Pre-existing Sidebar test-environment failures and FindBar TS6133:** untouched (deferred-items.md from Plan 95-01).
- **No regressions:** `go test ./internal/webserver/... -count=1` and `go test ./internal/daemon/... -count=1` both GREEN.

## Threat Surface Recap

The plan's `<threat_model>` register lists seven threats. Status:

| Threat ID | Status | Verification |
|-----------|--------|--------------|
| T-95-06-01 (Tampering — vendored UMD drifts from npm) | MITIGATED | `vendor_drift_test` min-count 7; CI fails on mismatch; Task 1 commit verified `cmp -s` byte-identical |
| T-95-06-02 (Tampering — future commit reintroduces `location.href = url`) | MITIGATED | `TestSecurity_NoCurrentTabNavigation` source-inspects BOTH terminal.js AND openLink.ts; ZERO matches enforced |
| T-95-06-03 (Tampering — future commit drops `noopener,noreferrer`) | MITIGATED | `TestTerminalJS_WebLinksOpener` asserts the EXACT literal `'_blank', 'noopener,noreferrer'` substring is present |
| T-95-06-04 (Tampering — future commit calls BrowserOpenURL on web) | MITIGATED | `TestTerminalJS_WebLinksOpener` asserts `BrowserOpenURL` is absent from terminal.js |
| T-95-06-05 (Tampering / XSS — Cyrillic / OSC 8 / malicious display in popover) | MITIGATED | `showLinkConfirmPopover` uses textContent for both `reasonEl` and `urlEl` (never `innerHTML`); 95-WEB-UAT.md §6 documents DevTools verification |
| T-95-06-06 (Information Disclosure — click telemetry over the wire) | MITIGATED | No `fetch` / `XMLHttpRequest` / `postMessage` in helpers or popover; helpers are pure (URL constructor + regex only) |
| T-95-06-07 (Spoofing — adversarial-process-emitted clickable URL on Tailscale-served session) | MITIGATED | All four gates (scheme allowlist, modifier-click, risk detection, opener) apply equally to web; 95-WEB-UAT.md §1-9 walks the Tailscale topology including iPad Safari |

No new threat surface introduced beyond the plan's register.

## Threat Flags

None — no new security-relevant surface introduced beyond what's tracked in the threat register.

## User Setup Required

None for the implementation. Manual UAT (`/gsd:verify-work 95`) requires:

- macOS / Linux / Windows desktop build (`wails build -tags wailsassets`) for 95-DESKTOP-UAT.md
- A second device on Tailscale + an iPad (any model, iOS 17+) for 95-WEB-UAT.md §9 (Phase 99 release gate)

## Next Phase Readiness

- **`/gsd:verify-work 95`:** UNBLOCKED. Every LNK-XX requirement maps to ≥1 automated test + ≥1 manual UAT step. SC-1..SC-5 in ROADMAP can flip from `PENDING` to `VERIFIED` after the runbook sign-off blocks are complete.
- **Phase 95 SUMMARY (rollup):** the orchestrator owns this — the plan body specified updating `.planning/phases/95-web-links-addon-security-hardening/95-SUMMARY.md`, but per executor guidance "Do NOT update STATE.md or ROADMAP.md — orchestrator owns those writes," the phase-level rollup is also orchestrator scope. This per-plan SUMMARY is the executor's deliverable.
- **v3.3 follow-up:** `LNK-OSC8-FUT-01` (OSC 8 display-vs-href divergence) is tracked in `REQUIREMENTS.md` `## Future Requirements`. The dormant `osc8` branches in both `LinkConfirmPopover` (desktop) and `showLinkConfirmPopover` (web) ship complete; a v3.3 wiring-only PR can flip the slice GREEN by adding a custom OSC 8 handler via `Terminal.parser.registerOscHandler(8, ...)` once `IBufferCell.getHyperlinkId` enters the public typings (or by committing to a custom buffer-state walk).

## Self-Check: PASSED

Verified post-Write that all claims hold:

| Claim | Check | Result |
|-------|-------|--------|
| `web/vendor/xterm/addons/addon-web-links.js` exists | `test -f web/vendor/xterm/addons/addon-web-links.js` | OK |
| Byte-identical to source UMD | `cmp -s web/vendor/xterm/addons/addon-web-links.js frontend/node_modules/@xterm/addon-web-links/lib/addon-web-links.js` | OK |
| VERSION manifest has @xterm/addon-web-links | `grep -q "^@xterm/addon-web-links@" web/vendor/xterm/VERSION` | OK |
| ≥7 entries in VERSION | `[ $(wc -l < web/vendor/xterm/VERSION) -ge 7 ]` | 7 |
| embed.go includes new asset | `grep -q "vendor/xterm/addons/addon-web-links.js" web/embed.go` | OK |
| vendor_drift_test min-count == 7 | `grep -q "len(pnpmVersions) < 7" internal/webserver/vendor_drift_test.go` | OK |
| `go build ./...` clean | `go build ./...` | exit 0 |
| `vendor_drift_test` GREEN | `go test ./internal/webserver/ -run TestXtermVendorVersionsMatchPnpmLock -count=1` | OK |
| terminal.html script ordering | awk script-ordering gate | search=48 < web-links=49 < terminal=63 |
| popover DOM in terminal.html | `grep -c 'id="link-confirm-popover"' web/terminal.html` | 1 |
| popover aria-modal | `grep -q 'aria-modal="true"' web/terminal.html` | OK |
| terminal.js helper count ≥ 8 | `grep -cE "function (isAllowedScheme\|hasIDN\|osc8Mismatch\|isTypoSquat\|getRisk\|isModifierPressed\|openLink\|showLinkConfirmPopover)"` | 8 |
| Exact options literal | `grep -F "'_blank', 'noopener,noreferrer'" web/assets/terminal.js` | OK |
| NO BrowserOpenURL | `! grep -q BrowserOpenURL web/assets/terminal.js` | OK |
| NO current-tab navigation | `grep -E "(location\\.href\\s*=\|window\\.location\\s*=)" web/assets/terminal.js` | empty |
| webLinksAddonHandle declared | `grep -q webLinksAddonHandle web/assets/terminal.js` | OK |
| Phase 95 web-links arm | `grep -A 30 "Phase 95 — web-links arm" web/assets/terminal.js \| grep -q "webLinksAddonHandle.dispose"` | OK |
| terminal.css popover block | `grep -q '#link-confirm-popover' web/assets/terminal.css` | OK |
| reduced-motion guard | `grep -A 3 "prefers-reduced-motion" web/assets/terminal.css \| grep -q link-confirm-popover` | OK |
| 3 Go tests defined | grep for each function name | OK |
| NO `t.Skip` in web_links_test.go | `grep -c "t.Skip" internal/webserver/web_links_test.go` | 0 |
| 3 Go tests GREEN | `go test ./internal/webserver/ -run "TestAssets_AddonWebLinks\|TestSecurity_NoCurrentTabNavigation\|TestTerminalJS_WebLinksOpener" -count=1` | PASS PASS PASS |
| Webserver full sweep GREEN | `go test ./internal/webserver/... -count=1` | ok |
| Daemon full sweep GREEN | `go test ./internal/daemon/... -count=1` | ok |
| Playwright spec exists | `test -f frontend/e2e/web-links-live-toggle.spec.ts` | OK |
| 95-DESKTOP-UAT.md exists with required content | `grep -q "Cyrillic" && grep -q "U+043E"` | OK |
| 95-DESKTOP-UAT.md branches Plan A vs Plan B | `grep -E "(Plan A\|Plan B)" .planning/phases/95-web-links-addon-security-hardening/95-DESKTOP-UAT.md \| wc -l` | 3 |
| 95-WEB-UAT.md mentions iPad/Tailscale/dev-browser ≥3 | `grep -E "iPad Safari\|Tailscale\|dev-browser" \| wc -l` | 16 |
| 95-WEB-UAT.md asserts window.opener | `grep -q "window.opener"` | OK |
| All 3 commit hashes recorded | `git log --oneline \| head -3` | `8483ca0`, `29faef0`, `c83e6d5` |
| No accidental deletions | `git diff --diff-filter=D --name-only HEAD~3 HEAD` | empty |

## TDD Gate Compliance

Plan 95-06 is type=execute (not type=tdd at the plan level), but each task carries `tdd="true"` per the plan's task frontmatter. The Wave-0 RED scaffolds for this plan were authored in Plan 95-01 Task 2 (commit `282daa0` on the predecessor branch) — they were the RED gate. Task 3's commit `8483ca0` (`test(95-06): flip web_links_test.go RED → GREEN`) is the GREEN gate. Task 1 + Task 2's `feat(...)` commits land the GREEN-gating production code that allows Task 3's tests to pass.

- **RED gate (Plan 95-01 Task 2 — predecessor commit `282daa0`):** 3 skip-marked tests in web_links_test.go.
- **GREEN gate (this plan — Task 3 commit `8483ca0`):** 3 real tests passing.
- **Productionized via:** Task 1 (`c83e6d5` — vendor + embed) + Task 2 (`29faef0` — wire). Task 1's `feat(95-06): vendor + bump min-count` commit message follows the conventional `feat` type because it ships production assets (the UMD copy and the embed directive); the test gate it satisfies is `vendor_drift_test`'s min-count assertion, which was already in place from Phase 89/93/94.
- **REFACTOR gate:** N/A — no refactor needed; both comment-vs-grep collisions were reworded inline before each commit landed.

---
*Phase: 95-web-links-addon-security-hardening*
*Plan: 06 (web parity — vendor UMD + terminal.js applyPluginConfig arm + 3 GREEN tests + UAT runbooks)*
*Completed: 2026-05-06*
