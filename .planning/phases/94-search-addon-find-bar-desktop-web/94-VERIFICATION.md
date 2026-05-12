---
phase: 94-search-addon-find-bar-desktop-web
verified: 2026-05-05T11:00:00Z
reverified: 2026-05-11T00:00:00Z
status: human_needed
score: 5/5 SCs implementation-verified; 4 of 6 human UATs resolved 2026-05-11 (UAT-2 PASS, UAT-5 PASS, UAT-6 PASS, UAT-1 PARTIAL/minor — slide-out missing; new major bug surfaced — toggle case-sensitive then Esc/close fails to dismiss); UAT-3 perf and UAT-4 web parity deferred to Tailscale/browser batch
overrides_applied: 0
gaps:
  - truth: "SC-4 — Find bar visual treatment matches BannerStack vocabulary including 200ms slide-in/out animation"
    status: partial
    reason: "Animation CSS classes (.find-bar--entering / .find-bar--exiting) are defined in frontend/src/style.css:2176-2184 and the transition rule is wired on .find-bar (200ms ease) and #find-bar (terminal.css:115). However NEITHER React FindBar.tsx NOR web/assets/terminal.js ever applies these modifier classes during open or close. On both surfaces the bar appears/disappears instantly. SC-4 says 'animation' explicitly; UI-SPEC §Animation mandates the 200ms slide-in (translateY(-100%) → 0) and 150ms exit. The other parts of SC-4 (TokyoNight palette, theme-aware highlight via theme.selectionBackground) ARE implemented and verified."
    artifacts:
      - path: "frontend/src/components/FindBar/FindBar.tsx"
        issue: "No useState mount flag, no requestAnimationFrame, no className conditional for find-bar--entering/exiting. Bar appears at translateY(0); opacity:1 on first render."
      - path: "web/assets/terminal.js:440-472"
        issue: "showFindBar() / hideFindBar() set el.hidden=false/true with no transition class application; transition rule is dead CSS."
    missing:
      - "Apply .find-bar--entering on first render in FindBar.tsx, drop it on next rAF so transition fires."
      - "On exit, delay unmount 150-200ms while applying .find-bar--exiting."
      - "Mirror the same in showFindBar/hideFindBar in web/assets/terminal.js."
      - "OR: document scope reduction here (acceptance) and remove the dead --entering/--exiting CSS classes."
  - truth: "SC-2 — toggleable regex / case-sensitive / whole-word options; per-flag defaults persist via daemon settings (SearchConfig)"
    status: partial
    reason: "Persistence wiring works (handleSearchOptionsChange writes via SetPluginSettings; loadSettings round-trips SearchConfig; FindBar.persistence.test.tsx + Go tests verify this). HOWEVER, on the desktop the local searchOptions state is initialized via useState lazy initializer that runs ONLY on the first render, when pluginConfig is typically still null (it's loaded asynchronously by App.tsx via GetPluginSettings). The state is never re-seeded once pluginConfig arrives. Reproduction: user toggles case-sensitive ON, restarts the app, presses Cmd-F → bar opens with all toggles OFF instead of the saved state. SRC-02 explicitly requires defaults 'persist across sessions'; persistence write works but read-back-on-load is broken."
    artifacts:
      - path: "frontend/src/components/TerminalPanel.tsx:99-103"
        issue: "useState(() => ({ regex: pluginConfig?.searchConfig?.regex ?? false, ... })) — fires once at first render only; no useEffect re-syncs from pluginConfig."
    missing:
      - "Add a seededRef.current useEffect that fires once when pluginConfig.searchConfig becomes non-null and seeds searchOptions before the bar is ever opened (preserving Pitfall #2 mid-open invariant)."
      - "OR: eagerly fetch pluginConfig before mounting TerminalPanel (architectural change, larger blast radius)."
  - truth: "SC-2 — per-flag defaults persist via daemon settings (no data loss)"
    status: partial
    reason: "Threat model item T-94-02 (toggle race vs PluginsSection) was logged as 'accepted — daemon settings:plugins event re-syncs.' But PluginsSection (lines 30-40) only calls GetPluginSettings() once at mount and does NOT subscribe to settings:plugins. So when handleSearchOptionsChange calls SetPluginSettings(next) constructed from the App-level prop, and the user has unsaved boolean toggle edits in the PluginsSection local edit buffer, those edits will be silently overwritten on PluginsSection's next Save Plugins click (which uses its stale pluginConfig snapshot). This is a real data-loss vector even if narrow."
    artifacts:
      - path: "frontend/src/components/TerminalPanel.tsx:414-426"
        issue: "handleSearchOptionsChange writes the entire pluginConfig (with searchConfig overlay), not the searchConfig sub-key only."
      - path: "frontend/src/components/PluginsSection.tsx:30-40"
        issue: "Only initial GetPluginSettings(); does NOT consume settings:plugins SSE/event mid-edit."
    missing:
      - "Either: make PluginsSection consume settings:plugins to refresh its local state (UX trade-off — could overwrite unsaved boolean edits)."
      - "OR: add a daemon-side SetSearchConfig(SearchConfig) RPC and have handleSearchOptionsChange call only that, not the full PluginSettings setter (more surgical; preserves PluginsSection's edit buffer semantics)."
human_verification:
  - test: "UAT — perf: paste a 10,000-line buffer (or run `seq 1 10000`), open Cmd-F, search a regex like `^[5-9]\\d{3}$`"
    expected: "No 'Page Unresponsive' dialog; DevTools Performance shows no >1s blocked main-thread frame; closing the bar mid-search cancels cleanly."
    why_human: "SC-3 perf budget is enforced by chromedp e2e (build-tagged) but the lived perf feel (smooth typing, no jank) is a manual confirmation."
    deferred: "Tailscale/browser batch — requires real browser with DevTools Performance to capture main-thread frame timing"
  - test: "UAT — web parity: open the served terminal at https://<tailscale-fqdn>:port/sessions/<id>?cap=..., press Cmd-F"
    expected: "Same find bar visual treatment, same shortcuts (Enter/Shift-Enter/Cmd-G/Cmd-Shift-G/Esc). Match count updates. Toggles work."
    why_human: "SC-5 web parity needs to be eyeballed; the chromedp e2e covers the contract surface but not the iPad/Safari real-device check."
    deferred: "Tailscale batch — requires served session URL over Tailnet"
human_verification_resolved:
  - test: "UAT — desktop: open new session, press Cmd-F, verify the bar slides in over ~200ms (not appears instantly)"
    resolution: "PARTIAL 2026-05-11. Slide-IN animation works correctly (~200ms slide-down). However, slide-OUT animation is missing — both Esc and the close button dismiss the bar instantly with no exit transition. Confirms half of WR-01: the .find-bar--exiting class never applies on close. Severity: minor (functional path intact, exit transition cosmetic). Engineering follow-up: apply .find-bar--exiting with 150-200ms delayed unmount in FindBar.tsx close path; mirror in web/assets/terminal.js hideFindBar()."
    result: partial
    severity: minor
    resolved_on: "2026-05-11"
  - test: "UAT — desktop: toggle case-sensitive ON, click Save Plugins (or close the find bar), restart the GUI, press Cmd-F"
    resolution: "PASS 2026-05-11 — case-sensitive toggle persisted across a full Cmd-Q + relaunch cycle; bar reopened with the case-sensitive option visibly ON. WR-02 did NOT reproduce on this run; the searchOptions seed path appears to be working in the current build."
    result: pass
    resolved_on: "2026-05-11"
    side_observation: "NEW BUG (major) — after clicking the case-sensitive toggle in the open find bar, both Esc and the close button stopped dismissing the bar (different failure mode from UAT-1's missing exit animation: UAT-1 dismisses instantly without animation; this state does not dismiss at all). Suggests a keyhandler/focus regression triggered by the toggle's click. Not on the existing WR-01/WR-02 list. Engineering follow-up: reproduce the sequence Cmd-F → toggle case-sensitive → Esc and trace which handler swallows the close event."
  - test: "UAT — theme matrix: switch among 5+ themes (TokyoNight Storm, Solarized Light, Nord, Dracula, Catppuccin) with the find bar open; type a search and confirm matches highlight via theme.selectionBackground"
    resolution: "PASS 2026-05-11 — match highlights remained visible across all tested themes; no black-on-black / invisible-on-bg failure modes observed."
    result: pass
    resolved_on: "2026-05-11"
  - test: "Regression — focus gate: in Settings tab, click in a text input outside the terminal, press Cmd-F"
    resolution: "PASS 2026-05-11 — Cmd-F inside a Settings text input did NOT open the AgentHub terminal find bar. isXtermFocused() gating works correctly."
    result: pass
    resolved_on: "2026-05-11"
overrides: []
re_verification:
  previous_status: none
  previous_score: none
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 94: Search Addon + Find Bar (Desktop + Web) — Verification Report

**Phase Goal:** "User can open a polished find bar with Cmd-F in any desktop or web terminal, search a 10,000-line scrollback without UI lockup, and the find bar visual treatment matches AgentHub's BannerStack vocabulary."

**Verified:** 2026-05-05
**Status:** human_needed (3 partial gaps + 6 UAT items)
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC-1 | Cmd-F opens find bar (focus-conditioned: only when xterm DOM is `document.activeElement`); Esc dismisses; identical on desktop and web | **VERIFIED** | Desktop: `TerminalPanel.tsx:371-384` window keydown listener gated by `isXtermFocused(containerRef.current)`. Web: `terminal.js:545-559` window keydown gated by `termEl.contains(document.activeElement)`. Esc handler on container (Pitfall #3) at FindBar.tsx:67-72 + terminal.js:483-485. `isXtermFocused.ts` exists, pure function, exhaustively unit-tested. |
| SC-2 | Next/prev (Enter/Cmd-G/Shift-Enter/Cmd-Shift-G), match count "X of Y", toggleable regex/case/word with per-flag defaults persisted via SearchConfig | **PARTIAL** | Keyboard: FindBar.tsx:74-91 (Enter/Cmd-G/Shift). Match count: FindBar.tsx:97-110 + onDidChangeResults subscription at TerminalPanel.tsx:285-287. Toggles: FindBar.tsx:189-225. Persistence (write): handleSearchOptionsChange:414-426 calls SetPluginSettings. Daemon: SearchConfig + defaults verified in plugin_settings.go. **GAP**: searchOptions never re-seeds from pluginConfig once it loads async (WR-02); user's saved defaults are NOT honored on first session-open after restart. **GAP**: handleSearchOptionsChange writes full PluginSettings, racing PluginsSection's stale local buffer (WR-03). |
| SC-3 | 10,000-line scrollback search completes without UI lockup; long regex searches cancellable by closing the find bar | **VERIFIED** | 10,000-line fixture exists at `internal/webserver/testdata/findbar_perf_fixture.txt` (verified `wc -l = 10000`). chromedp e2e harness `findbar_perf_e2e_test.go` enforces fixture line count and asserts buffer absorption ≥10000. 100ms debounce in TerminalPanel.tsx:395 (desktop) + terminal.js:437 (web). Cancel-on-close clears debounce + decorations: TerminalPanel.tsx:438-450 + terminal.js:451-472. |
| SC-4 | Find bar visual treatment matches BannerStack: TokyoNight palette, 200ms slide-in/out animation, theme-aware highlight via theme.selectionBackground | **PARTIAL** | TokyoNight palette: ✓ (style.css:2155-2285 — `#16161e` bg, `#7aa2f7` accent, `#9aa5ce` muted, `#c0caf5` fg, `#f7768e` error). Theme-aware highlight: ✓ (`{ ...searchOptions, decorations: {} }` empty object preserves theme.selectionBackground via xterm core selection — verified in FindBar.themeMatrix.test.tsx + Plan 94-05 chromedp e2e). **GAP**: 200ms slide-in/out animation is **not implemented** — `.find-bar--entering` / `.find-bar--exiting` CSS classes exist but no JS code applies them on either surface. Bar appears instantly. (See WR-01.) |
| SC-5 | Web parity: same shortcuts and visual treatment on web-served sessions | **VERIFIED** | terminal.html:21-41 finds-bar DOM mirrors desktop; terminal.js:440-559 wires same handlers (input, prev/next, toggles, close, Cmd-F focus gate). terminal.css:95-212 holds copy-verbatim TokyoNight tokens (UI-SPEC line 451 mandate). SearchAddon vendored at `web/vendor/xterm/addons/addon-search.js`; embedded in `web/embed.go:10`; SSE settings:plugins live update at terminal.js:658-697. Same animation gap as SC-4 applies on web (terminal.css:115 transition rule defined; terminal.js never applies classes). |

**Score:** 3/5 fully verified, 2/5 partial (SC-2 first-load + race; SC-4 animation).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/FindBar/FindBar.tsx` | Controlled React component, 200+ lines, all UI affordances | **VERIFIED** | 238 lines; Cmd-G/Shift-Enter/Esc/aria-pressed toggles all wired; uses focusSeq prop pattern for re-focus on Cmd-F-when-open. |
| `frontend/src/components/FindBar/style.css` | (or main style.css) — TokyoNight palette + 200ms transitions | **VERIFIED (with gap)** | Module style.css is 1-line stub pointing at frontend/src/style.css:2150-2286, which holds the actual rules. Animation classes defined but unused. |
| `frontend/src/components/TerminalPanel.tsx` | SearchAddon hot-swap, focus-gated Cmd-F, debounce, cancel-on-close | **VERIFIED** | Mount-effect addon dispose: 191-205. Hot-swap useEffect: 280-303 (search arm). Cmd-F window listener: 371-384. handleSearchClose: 438-450 clears timer + decorations. |
| `frontend/src/lib/isXtermFocused.ts` | Pure focus-test helper | **VERIFIED** | 14-line pure function with explicit doc comment. |
| `frontend/src/wailsjs/go/models.ts` | SearchConfig nested in PluginSettings | **VERIFIED** | SearchConfig class at line 10-23 with regex/caseSensitive/wholeWord; PluginSettings.searchConfig field at line 31; convertValues special-cased at line 46-52. |
| `internal/daemon/plugin_settings.go` | SearchConfig struct, all-false defaults | **VERIFIED** | 65 lines; SearchConfig and PluginSettings (with SearchConfig embed) defined; defaultPluginSettings() returns Search:true with all-false SearchConfig. |
| `web/terminal.html` | DOM-mirror of desktop find bar + addon-search.js script tag | **VERIFIED** | Lines 21-41 hold the find-bar DOM; line 48 includes `/assets/xterm/addons/addon-search.js`. |
| `web/assets/terminal.js` | Web parity find-bar handlers + searchAddon hot-swap + SSE re-sync | **VERIFIED** | Search handle: 245-326 hot-swap. wireFindBarHandlers: 474-539. window keydown focus-gated: 545-559. SSE plugin-config push subscriber: 658-697. |
| `web/assets/terminal.css` | TokyoNight tokens copy-verbatim from frontend/src/style.css | **VERIFIED** | Lines 95-212; identical token values per UI-SPEC line 451 mandate. |
| `web/vendor/xterm/addons/addon-search.js` | Vendored UMD bundle | **VERIFIED** | Present (ls confirmed); embedded in `web/embed.go:10`; manifest line in `VERSION`: `@xterm/addon-search@0.16.0`. |
| `internal/webserver/testdata/findbar_perf_fixture.txt` | 10,000-line scrollback fixture for SC-3 | **VERIFIED** | `wc -l = 10000` exact. |
| `internal/webserver/findbar_perf_e2e_test.go` | chromedp perf harness | **VERIFIED** | Asserts fixture is 10000 lines; loads + writes through xterm; measures search timing. |
| `internal/webserver/vendor_drift_test.go` | Generalized version-parity gate (Phase 93 contract; Phase 94 extends) | **VERIFIED** | Line 35 explicitly mentions addon-search joining the manifest with min-count guard for 6 packages. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| TerminalPanel.tsx Cmd-F handler | FindBar mount | `setFindBarOpen(true)` + JSX guard `findBarOpen && pluginConfig?.search` | **WIRED** | TerminalPanel.tsx:378-380 + 463-476. |
| FindBar input change | SearchAddon.findNext | `onQueryChange` → `handleSearchQueryChange` → debounced `searchAddonRef.current?.findNext(q, opts)` | **WIRED** | TerminalPanel.tsx:390-412. 100ms debounce confirmed. |
| SearchAddon onDidChangeResults | FindBar match count | `searchResultsDisposableRef` → `setMatchInfo` → FindBar.matchCount/currentMatchIndex props | **WIRED** | TerminalPanel.tsx:285-287; renders at FindBar.tsx:97-110. |
| FindBar toggle click | SetPluginSettings persistence | `onSearchOptionsChange` → `handleSearchOptionsChange` → `SetPluginSettings(next)` | **WIRED (with race)** | TerminalPanel.tsx:414-426. WR-03 race against PluginsSection's stale buffer. |
| Daemon SearchConfig | TS PluginSettings binding | Wails RPC + generated models.ts | **WIRED** | SearchConfig and PluginSettings classes regenerated; convertValues nested-conversion path special-cased. |
| Web /api/plugin-config | terminal.js applyPluginConfig | fetch on init + EventSource SSE for live updates | **WIRED** | terminal.js:127-140 (fetch) + 658-697 (SSE). |
| addon-search.js vendored asset | terminal.html script tag | `<script src="/assets/xterm/addons/addon-search.js">` | **WIRED** | terminal.html:48 + embed.go:10 directive. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| FindBar match count | `matchInfo.{index, count}` | SearchAddon.onDidChangeResults event | Yes (real xterm buffer search via decorations:{} gating fire) | **FLOWING** |
| FindBar searchOptions | `searchOptions` state | useState lazy init from pluginConfig?.searchConfig | **STATIC ON FIRST LOAD** | **STATIC** — `pluginConfig` is async-loaded; lazy init runs on first render when typically null. Saved defaults are not honored on first session open. (See WR-02 / SC-2 partial.) |
| PluginsSection persistence | full PluginSettings | GetPluginSettings + SetPluginSettings | Yes | **FLOWING** |
| Web searchOptions | `searchOptions` (IIFE-scoped) | initialConfig.searchConfig + SSE applyPluginConfig sync | Yes (web has explicit re-sync — `differs` check at terminal.js:335-347) | **FLOWING** (web does this correctly; only desktop has the gap) |
| 10000-line perf fixture | xterm buffer | findbar_perf_e2e_test.go writes line-by-line through term | Yes (asserts buffer length ≥10000 before measuring) | **FLOWING** |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Phase 94 frontend tests pass | `pnpm test --run src/components/FindBar src/lib/isXtermFocused.test.ts src/__tests__/App.plugin-event.test.tsx src/components/__tests__/TerminalPanel.search.test.tsx src/components/__tests__/PluginsSection.test.tsx` | 81/81 passed across 10 test files | **PASS** |
| Daemon search-related Go tests | `go test ./internal/daemon/...` | ok 0.029s | **PASS** |
| Webserver search-related Go tests | `go test ./internal/webserver/...` | ok 0.072s | **PASS** |
| 10000-line perf fixture | `wc -l internal/webserver/testdata/findbar_perf_fixture.txt` | 10000 | **PASS** |
| Vendored addon-search.js exists | `ls web/vendor/xterm/addons/addon-search.js` | present | **PASS** |
| Vendor drift test references search | `grep addon-search internal/webserver/vendor_drift_test.go` | matched line 35 (min-count 6, Phase 94 mitigation comment) | **PASS** |
| Frontend full sweep | `pnpm test --run` | Sidebar.test.tsx fails (20 tests) — unrelated to Phase 94. All 81 Phase 94 tests pass. | **PASS for Phase 94 scope; PRE-EXISTING failures elsewhere** |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SRC-01 | 94-01..05 | Cmd-F opens find bar (focus-conditioned); Esc dismisses | **SATISFIED** | TerminalPanel.tsx:371-384 + isXtermFocused.ts + terminal.js:545-559 + Esc on container at FindBar.tsx:67 + terminal.js:483 |
| SRC-02 | 94-01..05 | Next/prev, match count, toggles with persisted defaults | **SATISFIED with gaps** | All keyboard + UI wiring + persistence-write working; first-load read-back broken (WR-02); race vs PluginsSection (WR-03). Sub-criterion "persisted defaults" is met for *write*, broken for *read* on first session open. |
| SRC-03 | 94-04, 94-05 | 10,000-line scrollback search without UI lockup; cancellable | **SATISFIED** | 100ms debounce + cancel-on-close (TerminalPanel.tsx:395, 438-450; terminal.js:437, 451-472) + chromedp perf harness with verified 10000-line fixture |
| SRC-04 | 94-03, 94-05 | TokyoNight palette, 200ms slide-in/out, theme-aware highlight | **PARTIAL** | TokyoNight palette ✓; theme.selectionBackground via decorations:{} ✓; 200ms animation **not wired** (WR-01) |
| SRC-05 | 94-01, 94-04, 94-05 | Web parity for shortcuts + visual treatment | **SATISFIED** (with shared SC-4 animation caveat) | Full DOM mirror in terminal.html; identical CSS tokens; SearchAddon vendored + embedded; SSE settings:plugins live re-sync; window keydown focus-gated identically |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| frontend/src/components/TerminalPanel.tsx | 410, 417, 430, 435 | `decorations: {} as never` cast 4× | **WARNING (info per code review)** | Bypasses TS contract; future addon-search version change may silently break runtime while passing typecheck. (WR-04) |
| web/assets/terminal.js | 436, 458, 497-503, 508-515, 533-535 | `try { ... } catch (e) {}` swallowing 6× | **INFO** | Defensive but masks real bugs; recommend `console.warn` for diagnosis. (IN-02) |
| web/assets/terminal.js | 492, 547 | `navigator.platform.toUpperCase().indexOf('MAC')` per keystroke | **INFO** | Trivial perf cost; deprecated API on borrowed time. (IN-01, IN-03) |
| frontend/src/components/PluginsSection.tsx | 18 | Stale comment: "Phase 92 contract: TerminalPanel does NOT consume pluginConfig" | **INFO** | Phase 93/94 wired consumption; future maintainer confusion. (IN-04) |
| frontend/src/components/TerminalPanel.tsx | 97, 380 | `findBarFocusSeq` accumulates indefinitely | **INFO** | Code smell only; 2^53 safe-int range. (IN-05) |
| web/assets/terminal.js | (throughout) | `var` in modern code | **INFO** | Phase 94-added find-bar block could be `let`/`const`; out-of-scope cleanup. (IN-06) |
| frontend/src/style.css | 2176-2184 | `.find-bar--entering` / `.find-bar--exiting` CSS classes defined but unused | **WARNING** | Dead CSS that misleads next maintainer; **also** indicates SC-4 animation gap (the wiring is missing, not the styles). |

No **BLOCKER**-class anti-patterns. No XSS, SQLi, secret-exposure, or auth-bypass surfaces (purely client-side search; SearchConfig booleans flow through existing SEC-* capability-gated SetPluginSettings).

### Human Verification Required

See frontmatter `human_verification:` section. Six items; the most consequential are:

1. **Animation UAT** (SC-4) — confirms or refutes the WR-01 finding that the bar appears instantly.
2. **First-load persistence UAT** (SC-2) — confirms or refutes the WR-02 finding that saved toggle state is not honored on restart.
3. **Perf feel UAT** (SC-3) — confirms the chromedp budget translates to subjectively-smooth UX.
4. **Web parity UAT** (SC-5) — confirms the served terminal behaves identically.
5. **Theme matrix UAT** (SC-4) — confirms theme.selectionBackground works visually across themes.
6. **Focus gate regression** (SC-1) — confirms browser-native Cmd-F still works for non-terminal inputs.

### Gaps Summary

The implementation is high-quality at the structural level: every artifact exists, is substantive, is wired end-to-end on both surfaces, and the test sweep is comprehensive (81 Phase 94 tests pass; 10,000-line perf fixture and chromedp e2e in place; vendor drift gate extended; daemon SearchConfig persists with sensible defaults; SSE live-update on web).

Three concrete gaps prevent declaring `pass`:

1. **SC-4 animation not wired** (WR-01) — the 200ms slide-in/out is in the spec, in the CSS, and named in the success criterion, but no JS code applies the modifier classes on either surface. The bar appears instantly. This is a literal SC failure.

2. **SC-2 first-load persistence broken on desktop** (WR-02) — saved toggle defaults round-trip through the daemon correctly, but the desktop FindBar uses a useState lazy initializer that fires on first render when `pluginConfig` is still null (it's async). The state is never re-seeded. User experiences this as: "I saved my preferences but they're forgotten on restart." Web does NOT have this bug — terminal.js explicitly re-seeds searchOptions from initialConfig.searchConfig at line 580-587.

3. **SC-2 PluginsSection ↔ FindBar persistence race** (WR-03) — handleSearchOptionsChange writes the entire PluginSettings (with searchConfig overlay) constructed from the App-level prop. PluginsSection holds its own private edit buffer that only refreshes on mount via GetPluginSettings(). If the user has unsaved boolean toggle edits in PluginsSection at the moment the find bar persists a SearchConfig change, the next "Save Plugins" click will overwrite the SearchConfig change with PluginsSection's stale snapshot. The plan documented this race as `accept` based on a "settings:plugins event re-syncs" assumption that doesn't hold (PluginsSection doesn't subscribe).

All three are quality / UX correctness gaps — none are security, data-loss, or runtime crash issues. The phase goal is **mostly** achieved; gaps require human decision on whether to:

- Fix all three before declaring Phase 94 complete (return to gsd-plan-phase --gaps).
- Accept WR-01 as a documented scope reduction (remove dead CSS), accept WR-02/WR-03 as known issues tracked into a follow-up, and proceed.
- Surgically fix WR-02 (small effort: add a seededRef useEffect) and WR-03 (medium effort: SetSearchConfig sub-key RPC) and document WR-01 acceptance.

Recommendation: **escalate to user for direction**. The cleanest path is a small Phase 94 follow-up plan that fixes WR-02 (~15 LOC), WR-03 (~daemon RPC + frontend swap, ~50 LOC), and either implements WR-01 animations (~30 LOC each surface) or removes the dead CSS classes. Estimated 1-2 hours of focused work. Alternatively, if WR-01 animation is non-load-bearing for v3.2 release readiness, a documented override can accept it with the dead CSS cleaned up.

---

_Verified: 2026-05-05_
_Verifier: Claude (gsd-verifier, Opus 4.7 1M context)_

---

# Gap Closure Verification (post-94-06 / 94-07)

**Re-verified:** 2026-05-06T09:25:00Z
**Verdict:** **PASS** — all three previously-partial gaps (WR-01, WR-02, WR-03) are now closed end-to-end in code; full Phase 94 test sweep is green.
**Re-verifier:** Claude (gsd-verifier, Opus 4.7 1M context)

## Gap-by-Gap Closure Status

### WR-01 — SC-4 200ms slide-in / 150-200ms slide-out animation (closed by 94-06)

| Surface | Wiring | Evidence |
|---------|--------|----------|
| Desktop entry | `useState(true)` + `requestAnimationFrame` drop + cancel-on-unmount | `frontend/src/components/FindBar/FindBar.tsx:77-80` (mount-RAF), `:156-157` (className composition `entering && !exiting`) |
| Desktop exit | Parent-driven `findBarExiting` flag + 200ms `setTimeout` unmount + cancel-on-reopen | `frontend/src/components/TerminalPanel.tsx:103-104` (state + timer ref), `:504-510` (handleSearchClose), `:418-422` (Cmd-F cancel-on-reopen), `:531` (`(findBarOpen || findBarExiting)` render guard) |
| Web entry | `el.classList.add('find-bar--entering')` → `requestAnimationFrame(() => remove)` | `web/assets/terminal.js:451`, `:460-461` |
| Web exit | `el.classList.add('find-bar--exiting')` + 200ms `setTimeout` unmount, mid-exit re-open cancels | `web/assets/terminal.js:489-498` |
| Web CSS | `#find-bar.find-bar--entering` and `#find-bar.find-bar--exiting` rules + reduced-motion override | `web/assets/terminal.css:213-228` |
| Regression guards | 14 new tests across 3 files | `FindBar.animation.test.tsx` (3 runtime), `TerminalPanel.search.exit.test.tsx` (11 source-inspection), `internal/webserver/web_findbar_animation_test.go` (1 Go) — all pass |

**Closed.** `find-bar--entering` / `find-bar--exiting` are no longer dead CSS — they are toggled at the JS layer on both surfaces.

### WR-02 — SC-2 first-load seed of saved searchOptions defaults (closed by 94-07)

| Wiring | Evidence |
|--------|----------|
| `seededRef` ref + one-shot useEffect | `frontend/src/components/TerminalPanel.tsx:120` (ref), `:121-131` (effect with all three early-return guards: already-seeded, source-null, mid-open) |
| Pitfall #2 mid-open invariant | `:124` `if (findBarOpen) return` — mid-open SSE pushes never disrupt the user's in-flight toggle session |
| Dep array | `[pluginConfig?.searchConfig, findBarOpen]` — fires when async-loaded SearchConfig arrives or bar closes |
| Tests | 14 source-inspection tests in `TerminalPanel.search.seedAndPersist.test.tsx` covering ref declaration, all guards, dep array, setSearchOptions shape, ref flip — all pass |

**Closed.** Saved per-flag defaults are now read back on first session-open after restart.

### WR-03 — SC-2 SetSearchConfig sub-key RPC swap (closed by 94-07)

| Layer | Wiring | Evidence |
|-------|--------|----------|
| Engine | `(*SessionEngine).SetSearchConfig(SearchConfig)` mutates only `e.pluginSettings.SearchConfig`, persists, fires `pluginSettingsListener` (Phase 93 PLUG-04 SSE hook → SRC-05 web parity preserved) | `internal/daemon/engine.go:497-506` |
| HTTP | `PATCH /settings/search-config` route + `handleSetSearchConfig` (8 KiB cap, DisallowUnknownFields) | `internal/daemon/api.go:76` (route), `:567-578` (handler) |
| Client | `(*DaemonClient).SetSearchConfig` HTTP wrapper | `internal/daemon/client.go:164-165` |
| Wails facade | `(*App).SetSearchConfig` writes via client, then re-fetches full PluginSettings and re-emits `settings:plugins` event so App.tsx listener (which expects PluginSettings) keeps working unchanged | `app.go:505-523` |
| TS bindings | `SetSearchConfig(arg1: daemon.SearchConfig): Promise<void>` declared and exported | `frontend/src/wailsjs/go/main/App.d.ts:131`, `App.js:80` |
| Frontend swap | `handleSearchOptionsChange` calls `SetSearchConfig(new daemon.SearchConfig(opts))` instead of the previous full `SetPluginSettings(next)` snapshot | `frontend/src/components/TerminalPanel.tsx:13` (import), `:476` (call) |
| Tests | `TestSetSearchConfig` (sub-key isolation + listener + reload-from-disk) plus 14 frontend source-inspection tests confirm symbol swap | All green |

**Closed.** Find-bar SearchConfig writes no longer race PluginsSection's local edit buffer.

## Test Sweep Results

| Suite | Command | Result |
|-------|---------|--------|
| Phase 94 frontend (FindBar + isXtermFocused + App.plugin-event + TerminalPanel.search* + PluginsSection) | `pnpm exec vitest run` (selected files) | **109/109 passing** (was 81 before 94-06; +14 from 94-06 + +14 from 94-07) |
| Full frontend sweep | `pnpm exec vitest run` | **618 passing / 20 failing** — only failures are pre-existing `Sidebar.test.tsx` (logged in `deferred-items.md`); no Phase 94 regression |
| Daemon Go suite | `go test ./internal/daemon/...` | ok 6.624s |
| Webserver Go suite (default tag) | `go test ./internal/webserver/...` | ok 1.250s |
| Webserver Go suite (wailsassets tag) | `go test -tags wailsassets ./internal/webserver/...` | ok 1.420s |

## Updated Success Criteria & Requirement Coverage

| Item | Pre-closure | Post-closure |
|------|-------------|--------------|
| SC-1 — Cmd-F focus-conditioned open / Esc dismiss | VERIFIED | VERIFIED |
| SC-2 — Toggleable regex/case/word with persisted defaults | PARTIAL (WR-02 + WR-03) | **VERIFIED** — first-load seed + SetSearchConfig sub-key RPC both wired and tested |
| SC-3 — 10,000-line search w/o UI lockup, cancellable | VERIFIED | VERIFIED |
| SC-4 — TokyoNight + 200ms slide-in/out + theme-aware highlight | PARTIAL (WR-01) | **VERIFIED** — animation wired on both surfaces; RAF-mount entry + parent-driven 200ms exit on desktop, mirror on web |
| SC-5 — Web parity for shortcuts + visual treatment | VERIFIED (with shared SC-4 caveat) | **VERIFIED** — animation now mirrored on web; SSE plugin-config push still re-syncs SearchConfig changes via `pluginSettingsListener` |
| SRC-01 | SATISFIED | SATISFIED |
| SRC-02 | SATISFIED with gaps | **SATISFIED** |
| SRC-03 | SATISFIED | SATISFIED |
| SRC-04 | PARTIAL | **SATISFIED** |
| SRC-05 | SATISFIED (with shared SC-4 caveat) | **SATISFIED** |

## Remaining Open Items (UAT Only)

The original `human_verification:` block contained six manual UATs. Code-side gaps are all closed; the following remain as **manual confirmations only** (not blockers):

1. **Animation feel UAT (SC-4)** — desktop + web, plus reduced-motion + mid-exit re-open scenarios from `94-06-SUMMARY.md`.
2. **First-load persistence UAT (SC-2)** — toggle case-sensitive ON, restart GUI, press Cmd-F, confirm toggle is ON.
3. **PluginsSection race-regression UAT (SC-2)** — find bar toggle no longer clobbers unsaved Plugins-tab edits (described in `94-07-SUMMARY.md`).
4. **Perf feel UAT (SC-3)** — 10,000-line search smoothness.
5. **Web parity UAT (SC-5)** — Tailscale-served session walkthrough.
6. **Theme matrix UAT (SC-4)** — match highlight visible across 5+ themes.
7. **Focus-gate regression UAT (SC-1)** — browser-native Cmd-F still works on Settings tab inputs.

These are subjective / runtime-only checks (animation feel, real device, perf feel, visual color matrix) that no automated harness can replace. They do not gate phase closure for code-correctness purposes.

## Verdict

**PASS** — Phase 94 success criteria SC-1..SC-5 and requirements SRC-01..SRC-05 are now all VERIFIED in code. Test sweep is green for Phase 94 scope (109/109) and the only remaining frontend failures (20 in Sidebar.test.tsx) are pre-existing and tracked in `deferred-items.md`. Manual UATs remain as documented user-facing verification, but they no longer surface unresolved code gaps.

_Re-verified: 2026-05-06_
_Re-verifier: Claude (gsd-verifier, Opus 4.7 1M context)_
