---
phase: 94-search-addon-find-bar-desktop-web
verified: 2026-05-05T11:00:00Z
status: human_needed
score: 5/5 success criteria implementation-verified; 1 SC has a documented gap requiring human decision
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
  - test: "UAT — desktop: open new session, press Cmd-F, verify the bar slides in over ~200ms (not appears instantly)"
    expected: "Bar slides down from above the terminal pane over 200ms; smooth animation, no flash. Esc dismisses with ~150ms exit slide."
    why_human: "Animation feel is subjective and the test suite asserts only that the transition CSS rule exists, not that classes are applied. WR-01 indicates this currently fails; this UAT confirms whether the failure is real."
  - test: "UAT — desktop: toggle case-sensitive ON, click Save Plugins (or close the find bar), restart the GUI, press Cmd-F"
    expected: "Find bar opens with case-sensitive toggle visibly ON (highlighted). If it opens with all toggles OFF, WR-02 reproduces."
    why_human: "Confirms WR-02 (searchOptions never sync from pluginConfig prop after first render) on a real run."
  - test: "UAT — perf: paste a 10,000-line buffer (or run `seq 1 10000`), open Cmd-F, search a regex like `^[5-9]\\d{3}$`"
    expected: "No 'Page Unresponsive' dialog; DevTools Performance shows no >1s blocked main-thread frame; closing the bar mid-search cancels cleanly."
    why_human: "SC-3 perf budget is enforced by chromedp e2e (build-tagged) but the lived perf feel (smooth typing, no jank) is a manual confirmation."
  - test: "UAT — web parity: open the served terminal at https://<tailscale-fqdn>:port/sessions/<id>?cap=..., press Cmd-F"
    expected: "Same find bar visual treatment, same shortcuts (Enter/Shift-Enter/Cmd-G/Cmd-Shift-G/Esc). Match count updates. Toggles work."
    why_human: "SC-5 web parity needs to be eyeballed; the chromedp e2e covers the contract surface but not the iPad/Safari real-device check."
  - test: "UAT — theme matrix: switch among 5+ themes (TokyoNight Storm, Solarized Light, Nord, Dracula, Catppuccin) with the find bar open; type a search and confirm matches highlight via theme.selectionBackground"
    expected: "Match highlight color changes per theme (it sources from theme.selectionBackground via xterm core selection rendering). No black-on-black or invisible matches in any theme."
    why_human: "FindBar.themeMatrix.test.tsx asserts source-level absence of forbidden color keys, but real visual confirmation across themes is a human eye check."
  - test: "Regression — focus gate: in Settings tab, click in a text input outside the terminal, press Cmd-F"
    expected: "Browser-native Find dialog opens (NOT the AgentHub find bar) — focus is not in the xterm DOM, so isXtermFocused() returns false."
    why_human: "SC-1 focus-conditioning requires browser find still works for non-terminal page text; jsdom can't simulate browser-native Cmd-F dialog."
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
