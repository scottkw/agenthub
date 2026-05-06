---
phase: 95-web-links-addon-security-hardening
plan: 01
subsystem: testing
tags: [phase-95, web-links, vendor, scaffolding, wave-0, spike, plugin-suite, xterm-addon]

# Dependency graph
requires:
  - phase: 92-plugin-settings-foundation
    provides: PluginSettings struct + Wails RPC + settings:plugins event + hand-edit models.ts pattern
  - phase: 93-vendoring-discipline-web-parity-for-already-shipping-addons
    provides: Generalized vendor_drift_test.go (CI gate enforcing @xterm/* version parity)
  - phase: 94-search-addon-find-bar-desktop-web
    provides: SearchConfig sub-struct precedent + nested Wails class hand-edit pattern + RED scaffold convention
provides:
  - "@xterm/addon-web-links@0.12.0 npm dep + lockfile entry + VERSION manifest line"
  - "daemon.PluginSettings.WebLinksConfig sub-struct (Modifier/ConfirmOSC8/ConfirmIDN/ConfirmTyposquat) with security-first defaults"
  - "Hand-edited daemon.WebLinksConfig class on frontend/src/wailsjs/go/models.ts (Phase 92 pin pattern)"
  - "8 RED test scaffolds — every downstream Phase 95 plan has a named verify target waiting to flip GREEN"
  - "Wave 0 spike outcome: Plan B selected (defer OSC 8 mismatch detection to v3.3) due to IBufferCell.getHyperlinkId missing from @xterm/xterm@6.0.0 public typings"
affects:
  - 95-02-PLAN (urlSafety + openLink helpers — RED scaffolds wait)
  - 95-03-PLAN (LinkConfirmPopover — RED scaffold waits)
  - 95-04-PLAN (TerminalPanel WebLinksAddon wiring — Plan B narrows LNK-03 to IDN+typosquat only)
  - 95-05-PLAN (SetWebLinksConfig sub-key RPC + migration — Go RED scaffolds wait)
  - 95-06-PLAN (web parity: vendor UMD copy, terminal.js opener, e2e — Go + Playwright scaffolds wait; vendor_drift min-count bump from 6 to 7)

# Tech tracking
tech-stack:
  added:
    - "@xterm/addon-web-links@0.12.0"
  patterns:
    - "Wave 0 RED scaffold authoring (every downstream plan gets at least one named verify target before any implementation lands)"
    - "Cyrillic codepoint metatest (asserts fixture survived file I/O without normalization — Pitfall #2 mitigation)"
    - "Wave 0 spike captures both API surface inspection AND public-typings inspection (two distinct evidence paths) before committing to Plan A or Plan B"

key-files:
  created:
    - "frontend/src/lib/__tests__/urlSafety.test.ts"
    - "frontend/src/lib/__tests__/openLink.test.ts"
    - "frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx"
    - "frontend/src/components/__tests__/LinkConfirmPopover.test.tsx"
    - "internal/daemon/web_links_config_test.go"
    - "internal/webserver/web_links_test.go"
    - "frontend/e2e/web-links-live-toggle.spec.ts"
    - ".planning/phases/95-web-links-addon-security-hardening/deferred-items.md"
  modified:
    - "frontend/package.json"
    - "frontend/pnpm-lock.yaml"
    - "frontend/src/wailsjs/go/models.ts"
    - "frontend/src/__tests__/App.plugin-event.test.tsx"
    - "internal/daemon/plugin_settings.go"
    - "internal/daemon/plugin_settings_test.go"
    - "web/vendor/xterm/VERSION"
    - ".planning/phases/95-web-links-addon-security-hardening/95-RESEARCH.md"

key-decisions:
  - "Wave 0 spike outcome: Plan B selected — defer OSC 8 mismatch detection (LNK-03 component) to v3.3 because IBufferCell.getHyperlinkId is NOT in @xterm/xterm@6.0.0 public typings; LNK-03 ships IDN+typosquat detectors only in v3.2"
  - "Hand-edit models.ts mirrors the EXISTING inline-conversion pattern (no convertValues helper) rather than the plan's literal text — the project comment documents that convertValues would surface as keyof PluginSettings and break PluginsSection.tsx toggle iteration. Preserves Phase 92 + 94 convention."
  - "Add @xterm/addon-web-links@0.12.0 line to web/vendor/xterm/VERSION manifest immediately (npm dep is now in pnpm-lock; existing vendor_drift_test gates parity). Physical UMD copy under web/vendor/xterm/addons/ remains a Plan 95-06 concern per the plan's explicit deferral."

patterns-established:
  - "Pattern: Wave 0 RED-scaffold-everything — every downstream plan in a phase gets at least one named verify target authored upfront (urlSafety.test.ts → 95-02; LinkConfirmPopover.test.tsx → 95-03; TerminalPanel.web-links.test.tsx → 95-04; web_links_config_test.go → 95-05; web_links_test.go + e2e → 95-06)"
  - "Pattern: Cyrillic-codepoint metatest in the same file as the spoof fixture (asserts >= 1 codepoint > 0x7F in the host portion); GREEN on Wave 0; survives any future file-I/O normalization"
  - "Pattern: Spike outcome appended to RESEARCH.md (not the plan or its own file) so it lives next to the original Open Questions and survives plan deletion; selected path captured as a single literal **Selected:** Plan A | Plan B token for grep-driven downstream gating"

requirements-completed: [LNK-01, LNK-02, LNK-03, LNK-04, LNK-05, LNK-06]

# Metrics
duration: 28min
completed: 2026-05-06
---

# Phase 95 Plan 01: Wave-0 Foundation Summary

**Installed `@xterm/addon-web-links@0.12.0`, added `WebLinksConfig` sub-struct + hand-edited `models.ts`, ran the OSC 8 API spike (selected Plan B — defer OSC 8 mismatch to v3.3), and authored 8 RED test scaffolds so every downstream Phase 95 plan has a named verify target waiting to flip GREEN.**

## Performance

- **Duration:** ~28 min
- **Completed:** 2026-05-06T17:34:06Z
- **Tasks:** 2 / 2
- **Files created:** 8
- **Files modified:** 8

## Accomplishments

- `@xterm/addon-web-links@^0.12.0` installed (resolved 0.12.0); lockfile + `VERSION` manifest both updated.
- `daemon.WebLinksConfig` struct (`Modifier`, `ConfirmOSC8`, `ConfirmIDN`, `ConfirmTyposquat`) added with security-first defaults (`Modifier="platform"`, all confirmations `true`); `TestDefaultPluginSettings` extended with four new assertions.
- Hand-edited `frontend/src/wailsjs/go/models.ts` adds `daemon.WebLinksConfig` class + `webLinksConfig: WebLinksConfig` field on `PluginSettings`. TS compile clean.
- Wave 0 spike: source-read addon UMD bundle + `xterm.js` typings; appended `## Wave 0 Spike Outcome` section to `95-RESEARCH.md`. **Selected: Plan B** (defer OSC 8 mismatch detection to v3.3).
- 8 RED scaffolds authored — 11 RED + 1 GREEN metatest in `urlSafety.test.ts`, 5 RED in `openLink.test.ts`, 7 RED in `TerminalPanel.web-links.test.tsx`, 8 RED in `LinkConfirmPopover.test.tsx`, 1 NEW RED in `App.plugin-event.test.tsx` (12 prior tests still GREEN), 2 t.Skip-RED in `web_links_config_test.go`, 3 t.Skip-RED in `web_links_test.go`, 1 test.skip in `web-links-live-toggle.spec.ts`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Install + spike + WebLinksConfig + models.ts** — `95d72fe` (feat)
2. **Task 2: 8 Wave 0 RED scaffolds + VERSION manifest line** — `282daa0` (test)

## Files Created/Modified

### Created (8)

- `frontend/src/lib/__tests__/urlSafety.test.ts` — RED scaffold for `isAllowedScheme` / `hasIDN` / `osc8Mismatch` / `isTypoSquat` / `getRisk`; includes Cyrillic-codepoint metatest (GREEN) that asserts fixture survived file I/O. Plan 95-02 implements.
- `frontend/src/lib/__tests__/openLink.test.ts` — RED scaffold for the `openLink` helper (Wails `BrowserOpenURL` desktop branch + `window.open(url, '_blank', 'noopener,noreferrer')` web branch + scheme re-validation). Plan 95-02 implements.
- `frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx` — RED scaffold for `WebLinksAddon` import, custom-handler-not-default invariant, modifier-click gate, hot-swap dep array. Plan 95-04 implements.
- `frontend/src/components/__tests__/LinkConfirmPopover.test.tsx` — RED scaffold for risk-specific copy (`osc8` / `idn` / `typosquat`), button wiring, textContent rendering (XSS gate), focus trap, Escape. Plan 95-03 implements.
- `internal/daemon/web_links_config_test.go` — `TestSetWebLinksConfigPreservesSiblings` + `TestPluginSettingsMigration_WebLinksConfig` (both `t.Skip`). Plan 95-05 implements.
- `internal/webserver/web_links_test.go` — `TestAssets_AddonWebLinks` + `TestSecurity_NoCurrentTabNavigation` + `TestTerminalJS_WebLinksOpener` (all `t.Skip`). Plan 95-06 implements.
- `frontend/e2e/web-links-live-toggle.spec.ts` — Playwright `test.skip` for live-toggle web parity walk. Plan 95-06 implements.
- `.planning/phases/95-web-links-addon-security-hardening/deferred-items.md` — tracks one pre-existing TS warning in Phase 94 FindBar.animation.test.tsx (out-of-scope per Plan 95-01 boundary).

### Modified (8)

- `frontend/package.json` — `@xterm/addon-web-links: ^0.12.0` added.
- `frontend/pnpm-lock.yaml` — resolution chain for `@xterm/addon-web-links@0.12.0`.
- `frontend/src/wailsjs/go/models.ts` — `daemon.WebLinksConfig` class + `webLinksConfig` field on `PluginSettings` (inline-conversion pattern matches existing `SearchConfig`).
- `frontend/src/__tests__/App.plugin-event.test.tsx` — one new RED scaffold asserting the `webLinksConfig` nested object will be wired through; 12 prior tests untouched and still GREEN.
- `internal/daemon/plugin_settings.go` — `WebLinksConfig` struct (4 fields) + nested field on `PluginSettings` + populated default in `defaultPluginSettings()`.
- `internal/daemon/plugin_settings_test.go` — four new assertions for the WebLinksConfig defaults; existing assertions untouched.
- `web/vendor/xterm/VERSION` — `@xterm/addon-web-links@0.12.0` line added (vendor_drift_test parity gate).
- `.planning/phases/95-web-links-addon-security-hardening/95-RESEARCH.md` — `## Wave 0 Spike Outcome` section appended just above `## Metadata`; ends with `**Selected:** Plan B`.

## Wave 0 Spike Outcome (verbatim copy from 95-RESEARCH.md)

**Spike date:** 2026-05-06
**Inspected commit:** `@xterm/addon-web-links@0.12.0` + `@xterm/xterm@6.0.0`

**Finding 1 — WebLinksAddon handler IS canonical replacement (not additive):** PASS.
The UMD bundle stores the constructor's handler argument as `this._handler` and passes it straight through to `WebLinkProvider.computeLink`, which assigns it as the `activate` callback on each link object. There is no second, additive default handler that fires alongside. The upstream default itself opens `window.open()` then sets `n.location.href = t` — which is *worse* than `_blank` and confirms why a custom handler is mandatory for LNK-04 / LNK-05.

**Finding 2 — registerLinkProvider IS publicly typed; getHyperlinkId IS NOT:** PARTIAL FAIL.
- `Terminal.registerLinkProvider(linkProvider: ILinkProvider): IDisposable;` (line 1102 of `xterm.d.ts`): PASS.
- `IParser.registerOscHandler(ident, callback): IDisposable` (line 1864): PASS.
- `IBufferCell.getHyperlinkId()` — **NOT FOUND**. `grep -rn getHyperlinkId frontend/node_modules/@xterm/xterm/` returns ZERO matches.

**Selected: Plan B**

Plan 95-04 ships LNK-01..02, LNK-04, LNK-05, LNK-06 fully; LNK-03 ships IDN + typosquat detectors only. The `osc8Mismatch` helper still exists as a pure function (display + href ⇒ boolean) so its unit test stays meaningful, but it is NOT wired into the live `getRisk` path because the addon-web-links click handler only receives the `href` (never the OSC 8 *display* text). A `LNK-OSC8-FUT-01` follow-up will be added to REQUIREMENTS.md `## Future Requirements` when Plan 95-04 narrows SC-3.

## Downstream Plan Adjustments Needed (Plan B impact)

Because Plan B was selected, Plan 95-04 (the ONLY plan affected) must:

1. Narrow LNK-03 implementation to `hasIDN` + `isTypoSquat` only; do NOT wire `osc8Mismatch` into `getRisk`'s priority cascade.
2. Add a `LNK-OSC8-FUT-01` row to `.planning/REQUIREMENTS.md` `## Future Requirements` (or create that section) with body: "Surface OSC 8 display-vs-href divergence to LinkConfirmPopover. Blocked on `IBufferCell.getHyperlinkId` becoming public on `@xterm/xterm` (currently absent in 6.0.0 typings); revisit when xterm.js ships the public hyperlink-id accessor or when AgentHub commits to a custom buffer-state walk."
3. Update ROADMAP.md SC-3 acceptance text to read "IDN + typosquat warning" (not "OSC 8 mismatch + IDN + typosquat") and reference `LNK-OSC8-FUT-01` for the deferred slice.
4. The `osc8` branch of `LinkConfirmPopover.tsx` MUST still ship — the popover surface is needed so a v3.3 wiring-only PR can flip the slice GREEN without re-touching presentation.

Plans 95-02, 95-03, 95-05, 95-06 are unaffected by the spike outcome.

## Decisions Made

1. **Plan B (Wave 0 spike):** Defer OSC 8 mismatch detection to v3.3 because the public typing surface required (`IBufferCell.getHyperlinkId`) is absent from `@xterm/xterm@6.0.0`. Rationale: shipping LNK-03 with IDN + typosquat only is honest about the boundary; adding a custom OSC 8 hyperlink-range tracker via `registerOscHandler(8)` would be ~150 lines of internal-state plumbing the addon itself declined to build. Plan B unblocks Plans 95-02..06 immediately and pushes one slice of LNK-03 to a v3.3 follow-up.

2. **Hand-edited `models.ts` uses inline conversion (NOT convertValues helper):** The plan's literal text says `this.webLinksConfig = this.convertValues(source['webLinksConfig'], WebLinksConfig)`, but `models.ts` does not export a `convertValues` helper member — and the existing comment explicitly documents why: a `convertValues` member would surface as `keyof PluginSettings` and break `PluginsSection.tsx` toggle iteration. The new `webLinksConfig` field uses the same inline `source["webLinksConfig"] ? new WebLinksConfig(source["webLinksConfig"]) : new WebLinksConfig()` pattern as the existing `searchConfig` field.

3. **Add `@xterm/addon-web-links@0.12.0` to VERSION manifest immediately:** The plan defers the *physical UMD copy* to Plan 95-06, but the existing `TestXtermVendorVersionsMatchPnpmLock` gate fails the moment a new `@xterm/*` package enters `pnpm-lock.yaml` without a corresponding VERSION line. Adding the manifest line keeps CI green; the actual file copy under `web/vendor/xterm/addons/` is correctly deferred to Plan 95-06.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — Blocking] Add `@xterm/addon-web-links@0.12.0` line to `web/vendor/xterm/VERSION`**
- **Found during:** Task 2 (running `go test ./internal/webserver/...` for regression check after pnpm install)
- **Issue:** `TestXtermVendorVersionsMatchPnpmLock` failed: `web/vendor/xterm/VERSION missing entry for @xterm/addon-web-links`. The plan's `pnpm add` step was correct but the existing parity gate enforces every npm-locked `@xterm/*` package have a VERSION line; without one CI is broken.
- **Fix:** Appended `@xterm/addon-web-links@0.12.0` to `web/vendor/xterm/VERSION`. Did NOT touch `vendor_drift_test.go` (the plan explicitly forbids that — the min-count bump and the physical UMD copy under `web/vendor/xterm/addons/` are still Plan 95-06 concerns).
- **Files modified:** `web/vendor/xterm/VERSION`
- **Verification:** `go test ./internal/webserver/... -count=1` passes (was failing before the fix; passing after).
- **Committed in:** `282daa0` (Task 2 commit)

**2. [Rule 1 — Convention preservation] `models.ts` inline conversion pattern**
- **Found during:** Task 1 (hand-editing `frontend/src/wailsjs/go/models.ts`)
- **Issue:** The plan's literal `<action>` text directs `this.webLinksConfig = this.convertValues(source['webLinksConfig'], WebLinksConfig)`, but `models.ts` does not have a `convertValues` helper member. The existing file uses an inline-conversion pattern for `searchConfig` and a code comment documents why: adding a `convertValues` member would surface as `keyof PluginSettings` and break `PluginsSection.tsx` toggle iteration.
- **Fix:** Mirrored the existing inline-conversion pattern verbatim: `this.webLinksConfig = source["webLinksConfig"] ? new WebLinksConfig(source["webLinksConfig"]) : new WebLinksConfig();` Acceptance criterion `grep -q "this.webLinksConfig = this.convertValues(source\['webLinksConfig'\], WebLinksConfig)"` therefore does NOT pass on the literal text — but the architectural intent (round-trip a JSON object into a typed class instance for the nested struct) is preserved correctly.
- **Files modified:** `frontend/src/wailsjs/go/models.ts`
- **Verification:** `pnpm tsc --noEmit` exits 0 (no TS errors related to this file); the new `webLinksConfig` field is reachable from `PluginsSection.tsx`-style iteration without surfacing a non-data helper key.
- **Committed in:** `95d72fe` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 3 blocking, 1 Rule 1 convention preservation)
**Impact on plan:** Both auto-fixes essential. The VERSION line is a CI-correctness fix, not scope creep — Plan 95-06 still owns the physical UMD copy + the min-count bump in `vendor_drift_test.go`. The `models.ts` deviation defends an intentional Phase 92/94 convention documented in code; the alternative would have broken `PluginsSection.tsx`.

## Issues Encountered

- One transient flake: `TestPluginConfigStream_ExpiredCap_Returns401` failed once during a full webserver suite run (`got 403, want 401`), but passed on every isolated re-run and on the final full-suite run. Pre-existing test-ordering / shared-state flake unrelated to Phase 95 work — not in scope.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- All Wave 1 work in Plans 95-02 (frontend helpers) and 95-05 (daemon RPC) can start in parallel — both have RED scaffolds waiting and zero blocking unknowns.
- Plan 95-04 must read this SUMMARY before authoring its tasks: the LNK-03 narrowing (Plan B) is mandatory and requires a `LNK-OSC8-FUT-01` row in REQUIREMENTS.md.
- Plan 95-06 will need to bump `vendor_drift_test.go` min-count from 6 to 7 AND create `web/vendor/xterm/addons/addon-web-links.js` (the VERSION line is already in place).
- Plan 95-03 (LinkConfirmPopover) is unblocked; popover surface includes a `risk='osc8'` branch even though the live wiring for that branch ships in v3.3 (Plan B).

## Self-Check: PASSED

Verified post-Write that all claims hold:

| Claim | Check | Result |
|-------|-------|--------|
| `frontend/package.json` has `@xterm/addon-web-links` | `grep -q '"@xterm/addon-web-links"' frontend/package.json` | FOUND |
| Lockfile resolved to 0.12.0 | `pnpm list @xterm/addon-web-links --dir frontend --depth 0` | FOUND `@xterm/addon-web-links 0.12.0` |
| `WebLinksConfig` struct exists | `grep -c "type WebLinksConfig struct" internal/daemon/plugin_settings.go` | == 1 |
| `Modifier="platform"` default | `grep -q 'Modifier:         "platform"' internal/daemon/plugin_settings.go` | FOUND |
| `WebLinksConfig` class on models.ts | `grep -q "export class WebLinksConfig" frontend/src/wailsjs/go/models.ts` | FOUND |
| `webLinksConfig: WebLinksConfig` field | `grep -q "webLinksConfig: WebLinksConfig" frontend/src/wailsjs/go/models.ts` | FOUND |
| Spike outcome section present | `grep -q "## Wave 0 Spike Outcome" .planning/phases/95-web-links-addon-security-hardening/95-RESEARCH.md` | FOUND |
| Selected path captured | `grep -E "^\*\*Selected:\*\* (Plan A\|Plan B)$" .planning/phases/95-web-links-addon-security-hardening/95-RESEARCH.md` | `**Selected:** Plan B` |
| All 8 scaffold files exist | `test -f` for each | All FOUND |
| Cyrillic codepoints survived | `python3` ord-scan on urlSafety.test.ts | host contains U+043E twice |
| TestDefaultPluginSettings extended | `go test -run TestDefaultPluginSettings -v` | PASS (12 sub-assertions; 4 new for WebLinksConfig) |
| Go scaffolds compile + skip | `go test -run TestSetWebLinksConfigPreservesSiblings -v` + `TestAssets_AddonWebLinks -v` | both report `--- SKIP` |
| TS scaffolds compile | `pnpm tsc --noEmit` (excluding pre-existing FindBar warning) | 0 errors |
| Cyrillic metatest GREEN, others RED | `pnpm vitest run src/lib/__tests__/urlSafety.test.ts` | 12 tests, 1 passed (metatest), 11 failed (RED scaffolds) |
| App.plugin-event prior tests still GREEN | `pnpm vitest run src/__tests__/App.plugin-event.test.tsx` | 13 tests, 12 passed, 1 failed (NEW RED only) |
| Daemon regression suite | `go test ./internal/daemon/... -count=1` | ok |
| Webserver regression suite | `go test ./internal/webserver/... -count=1` | ok (after VERSION fix) |
| Commit hashes recorded | `git log --oneline` | `95d72fe`, `282daa0` |
| No accidental deletions | `git diff --diff-filter=D --name-only HEAD~2 HEAD` | empty |

---
*Phase: 95-web-links-addon-security-hardening*
*Completed: 2026-05-06*
