# Phase 99: Settings UI Polish + Migration + Final CSP Audit (Release Gate) — Research

**Researched:** 2026-05-08
**Domain:** Settings UI polish (PUI-02 / PUI-03 / PUI-04), v3.1→v3.2 settings.json migration verification, cross-browser CSP zero-violation e2e (Chromium + WebKit + Firefox + iPad Safari Tailscale UAT)
**Confidence:** HIGH

---

## Summary

Phase 99 is the v3.2 release gate. It is **mostly mechanical reuse** because Phases 92–98 have already shipped every primitive this phase needs. The work is three workstreams stitched together:

1. **Settings UI Polish (PUI-02 / PUI-03 / PUI-04).** PUI-02 is **already partially shipped** — `PluginsSection.tsx:130,136` already renders italic captions ("Applies to new sessions you create.") under the Unicode 11 and Inline Images rows via the existing `settings-panel__description--italic` CSS class (Phase 93 U11-01). What Phase 99 still owes for PUI-02 is the *one-shot BannerStack confirmation toast* on toggle-change. PUI-03 is brand-new UI surface: three `<details>` disclosures (Search, Web-Links, Inline Images) under their respective toggles, wired to the **already-shipped sub-key RPCs** (`SetSearchConfig`, `SetWebLinksConfig`, `SetImageConfig` in `app.go` and `engine.go`). The CSS class `.settings-panel__details` already exists (`style.css:573-583`). PUI-04 is satisfied by re-using the existing three-state Save button on `PluginsSection.tsx:154-159` — no new save infrastructure.

2. **Migration verification.** **Already shipped, already green.** `internal/daemon/engine_migration_test.go` already tests the v3.1 fixture (`tests/fixtures/settings_v3.1.json` exists and is realistic — `cliPaths` + `startMinimized` + `autoCloseSession`), asserts the `schemaVersion: 2` rewrite, and asserts idempotency via mtime comparison. Phase 99's job for SC-3 is to **expand** the assertions to cover all v3.2 sub-configs (SearchConfig defaults, WebLinksConfig defaults, ImageConfig.StorageLimit=16, all 8 plugin booleans), not re-architect the test.

3. **Cross-browser CSP e2e + iPad Safari Tailscale UAT.** Existing `frontend/e2e/web-csp.spec.ts` (Phase 93) is Chromium-only via `playwright.config.ts:31-37`. Phase 99 adds two more `projects[]` entries (`webkit`, `firefox`). The CSP under test is now Phase 96-amended: `script-src 'self' 'wasm-unsafe-eval'` (universally supported across Chromium 102+, Firefox 102+, Safari 16+, iPad Safari 16+ — verified in Phase 96 RESEARCH). iPad Safari Tailscale UAT precedent already exists at `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-iPad-UAT.md` — Phase 99's runbook clones that shape and runs the **all-plugins-enabled** attach/render/scrollback/detach flow.

**Primary recommendation:** Plan 99 as **3 parallel workstreams in 4 waves**: Wave 1 = PUI-02 BannerStack one-shot wiring (depends on existing `localBanner` pattern in `App.tsx:114`); Wave 2 = PUI-03 three `<details>` disclosure components in PluginsSection (depends on existing sub-key RPCs); Wave 3 = migration test assertion expansion + cross-browser Playwright project additions; Wave 4 = iPad Safari Tailscale UAT runbook + final phase verification. **Zero new architectural primitives.** Every dependency this phase has is already shipped.

---

## Project Constraints (from CLAUDE.md)

- **JS/TS:** `camelCase` vars, `PascalCase` components, ESLint + Prettier, TypeScript types — applies to PluginsSection extensions and new `<details>` sub-components.
- **Node:** `pnpm` preferred (the project default).
- **Go:** `go fmt`, context-aware functions. Migration test extension in `engine_migration_test.go`.
- **NEVER `kill node.exe`** — Claude Code runs as Node.js.
- **LSP first** for code navigation — applies to discovering existing PluginsSection refs and the three sub-key RPC bindings.
- **UAT via dev-browser skill** for browser-based verifications — Phase 99 has the most browser-UAT touchpoints of any v3.2 phase (Chromium, WebKit, Firefox automated; iPad Safari Tailscale manual).
- **Wails build requires `-tags wailsassets`** for production builds (project memory feedback).
- **Don't delete test artifacts early** (project memory feedback) — wait for user to confirm UAT is complete before cleanup.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PUI-02 | Toggles for plugins that cannot hot-swap (Unicode 11, Inline Images) display an inline italic caption "Applies to new sessions you create" directly under the toggle; toggling them surfaces a one-shot BannerStack confirmation telling the user to open a new session to see the change. | **Italic caption: ALREADY SHIPPED** in `PluginsSection.tsx:130` (Unicode 11) and `:136` (Inline Images) via the `caption` arg of `renderRow` and the `settings-panel__description--italic` CSS class (Phase 93 U11-01). **One-shot BannerStack: NEW.** Reuse the `localBanner` / `saveBanner` pattern in `App.tsx:114,890-933` — a `useState<{kind, text} \| null>` paired with `.banner-stack` rendering. Trigger from `handleSavePlugins` after successful save when the saved-state's Unicode 11 or Image bool *differs* from the previous saved value AND that diff is true→false or false→true (i.e., user actually toggled it). |
| PUI-03 | Plugins with meaningful runtime configuration — Search (defaults regex/case/word), Web-Links (Cmd-vs-Ctrl click modifier and confirmation policy), Inline Images (`storageLimit`) — expose those options via an inline `<details>` disclosure under their toggle. | **All three sub-key RPCs SHIPPED.** `SetSearchConfig(daemon.SearchConfig)`, `SetWebLinksConfig(daemon.WebLinksConfig)`, `SetImageConfig(daemon.ImageConfig)` exist in `app.go` (lines 527, 566, 606) and `engine.go` (lines 497, 526, 554). Daemon HTTP routes: `PATCH /settings/search-config`, `PATCH /settings/web-links-config`, `PATCH /settings/image-config` (`api.go:76-78`). Wails TS bindings exist at `wailsjs/go/main/App.d.ts:131,137,143`. CSS `.settings-panel__details` exists at `style.css:573-583`. **What's new:** the React UI inside each `<details>` (form controls bound to `pluginConfig.searchConfig` / `pluginConfig.webLinksConfig` / `pluginConfig.imageConfig` and dispatching the sub-key RPCs). |
| PUI-04 | Plugins section reuses existing three-state Save button (idle/saving/saved) and existing `daemonSettings` persistence mechanism — no new save infrastructure. | **Already satisfied by `PluginsSection.tsx:152-160`.** The three-state Save button is in place. PUI-04 is a *constraint* on PUI-03 implementation: the new `<details>` disclosures must NOT introduce per-row Save buttons, per-field auto-save spinners, or any new save indicator. **The pattern:** sub-key form controls dispatch their sub-key RPC immediately on change (`SetSearchConfig` / `SetWebLinksConfig` / `SetImageConfig`), exactly like FindBar's `onSearchOptionsChange` already does (`TerminalPanel.tsx:719-728`). The full-snapshot Save button at the bottom of the Plugins section continues to handle the 8 boolean toggles. |
</phase_requirements>

---

<user_constraints>
## User Constraints

> No `99-CONTEXT.md` will be authored. Per [skip-discuss-when-research-complete] memory: when ROADMAP / REQUIREMENTS / earlier-phase research already pre-answer the gray areas, skip `/gsd-discuss-phase` and proceed to `/gsd-plan-phase`. STATE.md `## Decisions` + REQUIREMENTS PUI-02..PUI-04 + the additional context in the orchestrator prompt leave only mechanical questions.

### Locked Decisions (from STATE.md `## Decisions`, ROADMAP Phase 99 SC, prior-phase RESEARCH/UI-SPEC)

- **Phase 99 is the release gate** for v3.2. Sequential dependency on Phases 92–98 (all complete except 90-06 and 94-VERIFICATION re-run, which are unrelated to v3.2 release readiness). [STATE.md `## Decisions` Phase 99]
- **Italic caption affordance is already shipped.** `PluginsSection.tsx:130` (Unicode 11) and `:136` (Inline Images) render the verbatim copy "Applies to new sessions you create." via the existing `settings-panel__description--italic` CSS class. Phase 99 does NOT re-author this; it asserts the caption is correct. [Phase 93 U11-01 / Phase 96 IMG-01 / `PluginsSection.tsx`]
- **One-shot BannerStack is the existing `.banner-stack` vocabulary.** Phase 81 BAN-01/BAN-02 / Phase 93 WGL-02 / Phase 97 SER-01 all share the same banner-stack pattern: a `useState<{kind, text} \| null>` in `App.tsx`, rendered inside `.banner-stack` with a × dismiss button. [App.tsx:114,890-933 — `saveBanner` precedent]
- **PUI-03 disclosures are next-session-aware where appropriate.** Inline Images storageLimit takes effect on next session (matches the italic caption). Web-Links modifier and confirmation policy take effect immediately on already-rendered links via `webLinksConfigRef.current` (`TerminalPanel.tsx:191-192`) — these are read at click time, NOT addon-construction time. Search regex/case/word defaults seed FindBar's `searchOptions` state via the `seededRef` pattern at NEXT bar-open (`TerminalPanel.tsx:166-178`). [Phase 95 LNK-02/05/06; Phase 94 SRC-02; Phase 96 IMG-02]
- **Sub-key RPCs are non-negotiable for PUI-03.** Use `SetSearchConfig`, `SetWebLinksConfig`, `SetImageConfig` — NOT `SetPluginSettings(full snapshot)`. Phase 94-07 already documented why: full-snapshot writes from a stale local prop race in-flight Plugins-tab edits. [Phase 94-07 WR-03; STATE.md decision "SetSearchConfig sub-key RPC plumbing follows the existing /settings/plugins shape"]
- **Migration test is already a CI gate, already green.** `engine_migration_test.go` covers SC-3. Phase 99 expands the assertion surface (all 8 plugins, all 3 sub-configs) but does NOT introduce a new test file. [Phase 92 PLUG-02; existing test]
- **CSP under test is `script-src 'self' 'wasm-unsafe-eval'`** (Phase 96 Amendment 2). All other directives unchanged from Phase 89 D-09 baseline. [Phase 96 RESEARCH §"Mandatory Pre-Phase CSP Audit"; `csp_mw.go:107-113`]
- **Cross-browser scope: Chromium + WebKit + Firefox automated, iPad Safari Tailscale manual.** Edge / IE / older browsers explicitly NOT in scope (consistent with Phase 89 SEC-08 supported-browser matrix). [ROADMAP Phase 99 SC-4]
- **iPad Safari Tailscale UAT is manual + real device, not emulator.** [ROADMAP Phase 99 SC-4 verbatim] iOS Simulator is NOT a substitute — Safari WebKit on iOS has subtle CSP / WebAssembly / network handling differences from desktop WebKit that the simulator hides.
- **All v3.2 plugins enabled** during the iPad Safari UAT session [ROADMAP Phase 99 SC-4: "with all v3.2 plugins enabled"]. Default state: 7 ON / Progress OFF. Plus a second pass with all 8 ON.
- **No autonomy on iPad UAT plan** — final wave plan must have `autonomous: false` (matches Phase 93 Plan 05, Phase 96 Plan 06, Phase 97 Plan 06, Phase 98 Plan 05 precedent: human checkpoint required for UAT-completion plans).

### Claude's Discretion

- **BannerStack toast lifetime.** WebGL recovery banner uses 8000ms auto-dismiss for `context-loss`, persistent for `software-rasterized`. Save banner is dismiss-only (no auto-clear). **Recommendation:** the PUI-02 toggle-confirmation banner auto-dismisses after **6000ms** (long enough to read "Open a new session to see the change", short enough to not pollute the banner stack). User can dismiss earlier via the × button. Mirror the WebGL recovery banner's `useEffect(setTimeout(onDismiss, 8000))` pattern with a 6000ms timeout.
- **BannerStack copy verbatim.** **Recommendation:** *"Open a new terminal session to apply the Unicode 11 change."* / *"Open a new terminal session to apply the Inline Images change."* Two distinct strings rather than a single generic ("Open a new session to apply your changes") because the user may toggle both in one save and we want them to know specifically which one. If both toggled in the same save, **show two banners stacked** (existing banner-stack supports multiple children — e.g., LocalNetwork + Update + WebGL all coexist in `App.tsx:894-933`).
- **Disclosure summary copy.** **Recommendation:** verbatim:
  - Search: `Search defaults`
  - Web Links: `Link click behavior`
  - Inline Images: `Storage limit`
- **`<details>` `open` attribute default.** **Recommendation:** all three default closed (browser default `<details>` initial state). Users who care about advanced config click to open; users who don't never see it. Avoids visual noise in the Plugins section default view.
- **PUI-03 form control choices:**
  - **Search regex/case/word:** three checkboxes, identical visual to existing PluginsSection toggles BUT with smaller compact form (these aren't 44px hit targets — they're advanced config). **Pragmatic alternative:** use the same `.settings-panel__toggle-row` class — visual consistency wins over space efficiency.
  - **Web Links modifier:** a `<select>` with options `platform` / `cmd` / `ctrl` / `none` (matches the daemon's string union from `WebLinksConfig.Modifier`). **Default UI value:** `platform` (per `defaultPluginSettings()`). Three checkboxes for `confirmOSC8`, `confirmIDN`, `confirmTyposquat`.
  - **Inline Images storageLimit:** a `<input type="number" min="1" max="1000" step="1">` (matches the daemon's `[1, 1000]` range gate at `api.go` `handleSetImageConfig`). Suffix label "MB". Default 16.
- **Migration test fixture extension.** **Recommendation:** keep the existing `tests/fixtures/settings_v3.1.json` AS-IS (it's deliberately minimal — proves zero-merge of v3.2 keys). Phase 99 may optionally add a SECOND fixture `tests/fixtures/settings_v3.2_partial.json` representing a returning *v3.2-rc* user with some plugins set but no sub-configs (hypothetical mid-development upgrade) to exercise the sub-config defaults-merge path. Optional — the existing fixture already covers the load-bearing v3.1 case.
- **Playwright cross-browser project order.** **Recommendation:** `chromium` (existing, fastest), `firefox` (gecko, distinct CSP impl), `webkit` (closest proxy to iPad Safari). Run sequentially because the Go fixture binary is single-instance (`fullyParallel: false, workers: 1` already set in `playwright.config.ts:20-21`).
- **iPad UAT runbook scope.** **Recommendation:** clone Phase 93's `93-iPad-UAT.md` shape verbatim, but extend with: zero-CDN audit (Safari Web Inspector remote debugging), CSP zero-violation audit (Safari Web Inspector console), all-plugins-enabled flow (attach → type → emit OSC 9;4 progress → emit sixel image → scroll back through history → detach → re-attach → confirm scrollback intact). 5 scenarios total.

### Deferred Ideas (OUT OF SCOPE)

- **PluginConfigSearch.tsx / PluginConfigWebLinks.tsx as separate component files.** [STATE.md → Phase 92 UI-SPEC §"Explicitly deferred to Phase 99"] The orchestrator originally suggested splitting, but the actual implementation can be inline render functions in PluginsSection.tsx (mirrors how `renderRow` is inline). Component extraction is a future refactor candidate if PluginsSection grows past 300 lines. Phase 99 may inline.
- **Per-plugin info icons / "learn more" affordances.** [Phase 92 UI-SPEC §"Explicitly deferred to Phase 99: do NOT design here"] Phase 99 ALSO defers — out of scope for v3.2 release. Future v3.3 candidate.
- **Web-side Settings UI for sub-configs.** Web-served Tailscale terminal page does NOT expose a Settings UI (read-only view of plugin config via `/api/plugin-config`). PUI-03 is desktop-only. Web parity for sub-configs is "the SSE stream propagates changes" — not "the web page lets you edit them". [Phase 93 PLUG-04 / WEB-03 architecture]
- **Migration test for v3.0 → v3.2.** v3.0 had no plugins block at all (same shape as v3.1 from a missing-keys perspective). The existing test handles this case implicitly — v3.0 settings.json is a strict subset of v3.1 (no `autoCloseSession` either). If a v3.0 fixture is desired, add it as a future hardening pass; not load-bearing for v3.2 release.
- **PROG-FUT-01 (Progress default-ON in v3.3).** Already deferred per REQUIREMENTS `## Future Requirements`. Phase 99 italic caption "Default OFF in v3.2 — flips ON in v3.3 after field validation" already in place at `PluginsSection.tsx:144`.
- **`addon-canvas`, `addon-ligatures`, `addon-attach`, third-party-plugin extensibility.** All in REQUIREMENTS `## Out of Scope`.
- **Reduced-motion override audit.** Pre-existing v3.1 concern per Phase 92 UI-SPEC accessibility action item; not a Phase 99 introduction. Defer to a future a11y pass.

</user_constraints>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Italic caption rendering | Browser (PluginsSection.tsx) | — | Pure presentation; CSS class `.settings-panel__description--italic` already exists. No daemon involvement. |
| One-shot BannerStack toggle confirmation | Browser (App.tsx state + PluginsSection emits the trigger) | — | UX affordance only. PluginsSection's `handleSavePlugins` post-save callback compares pre/post Unicode11 and Image booleans; calls a new `onPluginToggleSideEffect(kind)` prop wired to App.tsx's `pluginToggleBanner` setter. |
| `<details>` disclosure form rendering | Browser (PluginsSection.tsx inline render functions or sub-components) | — | Pure presentation. Form controls bound to `pluginConfig.{searchConfig,webLinksConfig,imageConfig}`. |
| Sub-key RPC dispatch on form change | Browser → Wails RPC → Daemon engine | API/Backend (engine.SetSearchConfig / SetWebLinksConfig / SetImageConfig) | Already shipped end-to-end in Phases 94-07, 95-05, 96-02. Phase 99 only adds the new UI dispatch site. |
| Sub-key state propagation to web SSE subscribers | API/Backend (engine listener fires `pluginSettingsListener` after sub-key write) | CDN/Static (web terminal.js consumes the SSE frame) | Already shipped; Phase 99 inherits unchanged. |
| `pluginConfig` prop drill from App.tsx → PluginsSection | Browser | — | Already shipped (Phase 92). Phase 99 only adds a 9th render branch (the disclosure forms) that reads existing prop fields. |
| Migration verification | API/Backend (Go test) | — | `engine_migration_test.go` already exists. Phase 99 adds assertions for SearchConfig defaults, WebLinksConfig defaults, ImageConfig defaults, all 8 plugin booleans. |
| Cross-browser CSP zero-violation e2e | Browser (Playwright spec running against fixture) + CI (matrix runner) | — | `frontend/e2e/web-csp.spec.ts` exists. Phase 99 adds 2 `projects[]` entries to `playwright.config.ts`. |
| iPad Safari Tailscale UAT | Real device (Safari on iPad over Tailnet) | Documentation (`99-iPad-UAT.md` runbook) | Manual checkpoint. Precedent: `93-iPad-UAT.md`. |

**Cross-tier integrity:** Every PUI-03 form control writes through a sub-key RPC that atomically updates one slice of PluginSettings under `e.mu.Lock()` and fires the listener (`engine.go:497-563`). The web SSE consumer receives the new full snapshot on every sub-key change. Concurrency between PluginsSection's full-snapshot Save and a sub-key RPC is bounded by the daemon's mutex — sub-key writes don't race full-snapshot writes.

---

## Standard Stack

### Core (all already installed / shipped)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@playwright/test` | `^1.59.1` | Cross-browser e2e CSP zero-violation suite | Already a project devDep; native multi-browser support via `projects[]` array; same engine for chromium/firefox/webkit so test code is portable. |
| `vitest` | `^4.1.0` | Component-level unit tests for new disclosure forms | Already a project devDep; existing `__tests__/PluginsSection.test.tsx` precedent. |

**Verified:** `frontend/package.json` declares `@playwright/test ^1.59.1` and `vitest ^4.1.0` (read 2026-05-08). [VERIFIED: codebase grep]

**Version verification (note from research protocol):**
```bash
cd frontend && pnpm view @playwright/test version  # confirm latest stable still in 1.x
```
**Recommendation:** do NOT bump Playwright version in Phase 99 — release-gate phase, freeze versions. Use the locked `^1.59.1`.

### Supporting (already shipped)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (no new deps) | — | — | Phase 99 introduces zero new dependencies. Every primitive needed is already imported. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Playwright multi-project | Separate Cypress / WebdriverIO suites for Firefox/Safari | Project already standardized on Playwright (Phase 93/96 e2e). Adding a new framework for Phase 99 alone is dependency churn at the release gate. **Don't.** |
| Real iPad UAT | iOS Simulator UAT | Simulator hides Safari/WebKit/Tailscale-network differences. ROADMAP SC-4 explicitly says "real device, not emulator". |
| New PluginConfigSearch.tsx file | Inline render functions in PluginsSection.tsx | Both work; inline is the path of least churn. Component extraction can be a future refactor. |
| New BannerStack component | Reuse `.banner-stack` div with the existing kind/text useState pattern | Existing pattern is proven (Phase 81/93/97); a new component would duplicate it. |

**Installation:** No new packages.

---

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│ User Settings interaction                                               │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
       (toggle 8 plugins)            (open <details>, edit sub-config)
                │                             │
                ▼                             ▼
   PluginsSection.tsx                  PluginsSection.tsx (inline disclosures)
   ─ local edit buffer                 ─ controlled form controls
   ─ Save Plugins button               ─ onChange → sub-key RPC immediately
                │                             │
                ▼                             ▼
   SetPluginSettings(full)             SetSearchConfig(SearchConfig)
                │                      SetWebLinksConfig(WebLinksConfig)
                │                      SetImageConfig(ImageConfig)
                │                             │
                └──────────────┬──────────────┘
                               │
                               ▼   (Wails RPC on Unix socket)
                       app.go (App methods)
                               │
                               ▼   (DaemonClient over Unix socket)
                       internal/daemon/api.go
                               │
                               ▼
                  engine.SetPluginSettings / SetSearchConfig /
                  SetWebLinksConfig / SetImageConfig
                               │
                               ▼   (mutate under e.mu.Lock())
                       saveSettingsToDisk
                               │
                               ├──────────────► settings.json (with schemaVersion: 2)
                               │
                               ▼
                       pluginSettingsListener fires
                               │
                ┌──────────────┴──────────────┐
                │                             │
                ▼                             ▼
   runtime.EventsEmit("settings:plugins")  ws.BroadcastPluginConfig
                │                             │
                ▼                             ▼
   App.tsx EventsOn handler           /api/plugin-config/stream SSE
                │                             │
                ▼                             ▼
   App-level pluginConfig state       web/assets/terminal.js consumer
                │                             │
                ▼                             ▼
   prop drill into <PluginsSection>   web terminal hot-swap or next-session
   prop drill into <TerminalPanel>


┌─────────────────────────────────────────────────────────────────────────┐
│ One-shot BannerStack on Unicode 11 / Image toggle (PUI-02)              │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
   PluginsSection.handleSavePlugins (after successful SetPluginSettings)
                               │
                               ▼   (compute diff vs prior saved state)
   if (priorUnicode11 !== savedUnicode11) emit('unicode11')
   if (priorImage !== savedImage) emit('image')
                               │
                               ▼
   App.tsx pluginToggleBanner state (similar shape to saveBanner)
                               │
                               ▼
   <div className="banner-stack">
     <PluginToggleBanner kind="unicode11" onDismiss={...}/>
     <PluginToggleBanner kind="image" onDismiss={...}/>
   </div>


┌─────────────────────────────────────────────────────────────────────────┐
│ Cross-browser CSP zero-violation e2e (SC-4)                             │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
   playwright.config.ts:31  projects: [chromium, firefox, webkit]
                               │
                               ▼
   global-setup boots the Go fixture (cmd/playwright-fixture)
                               │
                               ▼
   frontend/e2e/web-csp.spec.ts runs against each browser project
   ─ goto BASE_URL/sessions/<id>?cap=<token>
   ─ listen page.on('console') for 'csp' / 'content security policy'
   ─ listen page.on('weberror')
   ─ exercise scroll, type, image emit
   ─ assert cspViolations.length === 0


┌─────────────────────────────────────────────────────────────────────────┐
│ iPad Safari Tailscale UAT (SC-4 manual)                                 │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
   real iPad on Tailnet, Safari, attach Tailscale URL
                               │
                               ▼
   Safari Web Inspector (remote debugging from dev Mac)
   ─ Console tab: zero CSP violations
   ─ Network tab: zero CDN requests
                               │
                               ▼
   Manual flow: attach → type → OSC 9;4 progress → sixel image →
                scrollback → detach → re-attach → scrollback intact
```

### Recommended Project Structure (no new directories)

```
frontend/src/
├── components/
│   ├── PluginsSection.tsx        # MODIFIED — add 3 inline disclosure render fns + diff-detect props
│   └── __tests__/
│       └── PluginsSection.test.tsx  # MODIFIED — assert disclosure rendering, sub-key RPC dispatch
├── App.tsx                       # MODIFIED — add pluginToggleBanner state + 1 banner-stack child
└── style.css                     # MODIFIED — minor — disclosure form input cosmetic rules (if any)

frontend/e2e/
├── playwright.config.ts          # MODIFIED — add firefox + webkit projects
└── web-csp.spec.ts               # UNCHANGED (runs unchanged against new projects)

internal/daemon/
└── engine_migration_test.go      # MODIFIED — expand assertions to all 8 plugins + 3 sub-configs

tests/fixtures/
└── settings_v3.1.json            # UNCHANGED (already realistic)

.planning/phases/99-.../
├── 99-iPad-UAT.md                # NEW — runbook for real-device manual UAT
├── 99-VALIDATION.md              # NEW — generated by orchestrator after this RESEARCH
└── 99-RESEARCH.md                # THIS FILE
```

### Pattern 1: Existing italic caption (PUI-02 partial — already shipped)

**What:** Verbatim copy in italic muted text under the toggle row, indicating next-session-only semantics.
**When to use:** Only for plugins whose runtime change does NOT hot-swap (Unicode 11, Inline Images).
**Already implemented in:**
```typescript
// Source: frontend/src/components/PluginsSection.tsx:107-111
{caption && (
  <p className="settings-panel__description settings-panel__description--italic">
    {caption}
  </p>
)}
```
**CSS:**
```css
/* Source: frontend/src/style.css:1605-1607 */
.settings-panel__description--italic {
  font-style: italic;
}
```
**Phase 99 action:** Verify the captions are present (UAT-1 / UAT-4 of the iPad runbook); do NOT re-author. Add an inline component test asserting the verbatim string for both Unicode 11 and Inline Images rows.

### Pattern 2: One-shot BannerStack toast on toggle save (PUI-02 — NEW)

**What:** A transient banner appearing in the existing `.banner-stack` after the user saves a Unicode 11 or Image toggle change, with copy directing them to open a new session.
**When to use:** Only after PluginsSection's `handleSavePlugins` succeeds AND the saved value of either bool differs from the pre-save value. Both can fire in the same save (Unicode 11 + Image both toggled and saved → two banners stacked).
**Pattern (from existing `saveBanner` precedent in `App.tsx:114,890-933`):**
```typescript
// Pattern source: frontend/src/App.tsx:114 and 890-933 (saveBanner from Phase 97 SER-01)
type PluginToggleKind = 'unicode11' | 'image'
const [pluginToggleBanners, setPluginToggleBanners] = useState<PluginToggleKind[]>([])

// Wired into PluginsSection via a new prop:
const handlePluginToggleSideEffect = useCallback((kinds: PluginToggleKind[]) => {
  setPluginToggleBanners(prev => [...new Set([...prev, ...kinds])])
}, [])

// Render inside the existing .banner-stack:
{pluginToggleBanners.map(kind => (
  <PluginToggleBanner
    key={kind}
    kind={kind}
    onDismiss={() => setPluginToggleBanners(prev => prev.filter(k => k !== kind))}
  />
))}
```
**Verbatim copy (locked decision per Claude's Discretion above):**
- `unicode11`: *"Open a new terminal session to apply the Unicode 11 change."*
- `image`: *"Open a new terminal session to apply the Inline Images change."*

**Auto-dismiss:** 6000ms via `useEffect(setTimeout(onDismiss, 6000))` in the new `PluginToggleBanner` component.
**Visual treatment:** Reuse `.webgl-recovery-banner` BEM class verbatim — same TokyoNight info-tone palette, same 53px banner-stack height, same × button styling (`style.css:1612-1659`). The component differs only in its message copy and timeout.

### Pattern 3: `<details>` disclosure with sub-key RPC dispatch (PUI-03 — NEW)

**What:** An `<details>` block under the toggle row whose `<summary>` is the disclosure label and whose body holds form controls bound to `pluginConfig.{searchConfig,webLinksConfig,imageConfig}`.
**When to use:** Search, Web Links, Inline Images — the three plugins with meaningful runtime config.
**Pattern (from existing `<details>` in `SettingsTab.tsx:441-473`):**
```typescript
// Source pattern: frontend/src/components/SettingsTab.tsx:441-473 (Tailscale diagnostics disclosure)
<details className="settings-panel__details">
  <summary>Search defaults</summary>
  <div className="settings-panel__details-body">
    {/* Three checkboxes bound to pluginConfig.searchConfig */}
    <label>
      <input type="checkbox"
             checked={pluginConfig?.searchConfig?.regex ?? false}
             onChange={(e) => SetSearchConfig(new daemon.SearchConfig({
               ...pluginConfig.searchConfig,
               regex: e.target.checked
             }))} />
      Regex
    </label>
    {/* ... case + word ... */}
  </div>
</details>
```

**Sub-key RPC dispatch site (already shipped in TerminalPanel for FindBar):**
```typescript
// Source: frontend/src/components/TerminalPanel.tsx:719-728
SetSearchConfig(new daemon.SearchConfig(opts)).catch(() => {
  // The next GetPluginSettings call will reconcile.
})
```

**Three disclosure shapes:**

#### Search defaults (under "Find in scrollback" row)
- Three `<input type="checkbox">` for `regex`, `caseSensitive`, `wholeWord`
- Visual: reuse `.settings-panel__toggle-row` shape OR a more compact form (Claude's discretion — recommend reusing for visual consistency)
- onChange → `SetSearchConfig(new daemon.SearchConfig({...prior, [field]: value}))`
- **Persistence shape:** the daemon's `engine.SetSearchConfig` writes ONLY the SearchConfig sub-key under `e.mu.Lock()`. App-level `pluginConfig` updates via the `settings:plugins` event re-fetch in `app.go:535-540`. PluginsSection re-renders with the new value.

#### Link click behavior (under "Clickable web links" row)
- `<select>` bound to `pluginConfig.webLinksConfig.modifier` with options `platform` / `cmd` / `ctrl` / `none`
- Three `<input type="checkbox">` for `confirmOSC8`, `confirmIDN`, `confirmTyposquat`
- onChange → `SetWebLinksConfig(new daemon.WebLinksConfig({...prior, [field]: value}))`
- **Live-applies on already-rendered links** via `webLinksConfigRef.current` read at click time (`TerminalPanel.tsx:191-192`). No new-session requirement.

#### Storage limit (under "Inline images" row)
- `<input type="number" min="1" max="1000" step="1">` suffix label "MB"
- Bound to `pluginConfig.imageConfig.storageLimit`
- onChange → `SetImageConfig(new daemon.ImageConfig({storageLimit: value}))` (debounce recommended — `setTimeout` 500ms — to avoid spamming the daemon while user types digits)
- **Next-session-only:** matches the italic caption above. Daemon RPC fires immediately, but TerminalPanel's mount useEffect (`TerminalPanel.tsx`) reads the new value only on next session-mount per Phase 96 IMG-01.

### Pattern 4: Migration test assertion expansion (SC-3 — extend existing)

**What:** Existing `TestSettingsMigrationV3_1ToV3_2` already loads the v3.1 fixture and asserts `schemaVersion == 2` + `Plugins.WebGL == true`. Phase 99 expands to assert all 8 plugin booleans + all 3 sub-configs.
**Pattern (extending existing test):**
```go
// Source: internal/daemon/engine_migration_test.go (extending TestSettingsMigrationV3_1ToV3_2)
got := e.GetPluginSettings()
want := defaultPluginSettings()  // already returns the full PluginSettings — assertion below is exhaustive
if got != want {
    t.Errorf("GetPluginSettings after v3.1 load: got %+v, want %+v", got, want)
}

// Existing test already does this exhaustive comparison via the struct-equals check.
// Phase 99 adds explicit per-field error messages for clearer test output:
if got.WebGL != want.WebGL { t.Errorf("WebGL: got %v, want %v", got.WebGL, want.WebGL) }
// ...repeat for all 8 + 3 sub-configs
if got.ImageConfig.StorageLimit != 16 {
    t.Errorf("ImageConfig.StorageLimit: got %d, want 16 (Phase 96 default)", got.ImageConfig.StorageLimit)
}
```

**Idempotency assertion already exists** as `TestSettingsMigrationIdempotent` (mtime comparison, `engine_migration_test.go:92-127`). No change needed.

### Pattern 5: Cross-browser Playwright config (SC-4 — extend existing)

**What:** Add `firefox` and `webkit` projects to the existing `projects[]` array.
**Pattern (extending `playwright.config.ts:31-37`):**
```typescript
// Source: frontend/playwright.config.ts:31-37 (extending)
projects: [
  { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
  { name: 'webkit', use: { ...devices['Desktop Safari'] } },
],
```
**No spec changes needed.** `web-csp.spec.ts` already uses `page.on('console')` and `page.on('weberror')` — both APIs are supported across all three browser engines. The CSP detection regex `/content security policy|csp/i` works for all three (Chromium logs "Refused to ... because of Content Security Policy", Firefox logs "Content Security Policy: ...", WebKit logs "Refused to load ... because it appears in neither the script-src directive ...").
**Required CLI install:** `npx playwright install --with-deps firefox webkit` (Playwright bundles browser binaries; the `--with-deps` flag also installs OS-level shared libs on Linux CI).

**Known browser CSP differences (cross-checked):**

| Difference | Chromium | Firefox | WebKit | Mitigation |
|------------|----------|---------|--------|------------|
| `'wasm-unsafe-eval'` support | 102+ (May 2022) | 102+ (June 2022) | 16.0+ (Sep 2022) | All three meet the v3.2 supported floor; no fallback needed. [VERIFIED: Phase 96 RESEARCH §"Browser support"] |
| Console violation message format | "Refused to ... because of Content Security Policy" | "Content Security Policy: The page's settings blocked..." | "Refused to ... because it appears in neither the X directive..." | Regex `/content security policy\|csp/i` is case-insensitive and matches all three. |
| `frame-ancestors 'none'` enforcement | Yes (CSP2) | Yes (CSP2) | Yes (CSP2) | Already in v3.1 D-09 baseline; no Phase 99 change. |
| Report-only header (`Content-Security-Policy-Report-Only`) | Yes | Yes | Yes | NOT used (D-11 — no report-uri). |
| `blob:` URL handling | Same-origin via `'self'` | Same-origin via `'self'` | Stricter — sometimes requires explicit `blob:` source | Phase 96 audit confirmed addon-image's blob: URLs flow through `Image.src` (governed by `img-src`), not script-src. Modern browsers (incl. WebKit 15+) support `createImageBitmap` so the blob: fallback is dead code. [Phase 96 RESEARCH Detailed Finding 1] |
| `'unsafe-inline'` for style-src | Standard | Standard | Standard | Already in v3.1 D-09 amendment for xterm.js runtime style injection. |
| WebSocket from CSP `connect-src` | Yes | Yes | Yes | v3.1 D-09: `connect-src 'self' wss://<host>`. No Phase 99 change. |

**Conclusion:** No browser-specific test guards needed. The same spec runs unchanged against all three projects.

### Pattern 6: iPad Safari Tailscale UAT runbook (SC-4 — clone Phase 93 shape)

**What:** A markdown runbook with numbered scenarios, each with setup, steps, expected behavior, and a sign-off checkbox.
**Pattern source:** `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-iPad-UAT.md` (5 scenarios + sign-off section).
**Phase 99 scenario plan:**
1. **UAT-1: Italic caption verbatim** — confirm "Applies to new sessions you create." renders under both Unicode 11 and Inline Images on the desktop app.
2. **UAT-2: One-shot BannerStack** — toggle Unicode 11 OFF, save; confirm banner appears with verbatim copy; auto-dismisses after 6s; toggle Image OFF + save → second banner; both stack correctly.
3. **UAT-3: PUI-03 disclosure round-trip** — open Search disclosure, toggle Regex ON, close app, reopen, confirm Regex ON; open Web Links disclosure, change modifier to `cmd`, click a link in an already-open terminal, confirm Cmd-only required; open Image disclosure, change storageLimit to 32, create new session, confirm new session uses 32 MB cap (verified via existing `imageAddonRef.current.storageLimit` getter — DevTools console).
4. **UAT-4: iPad Safari Tailscale all-plugins-enabled flow (real device)** — clone of Phase 93 UAT-1+UAT-5 + extension. Open Tailscale URL on iPad. Verify zero CSP violations in Safari Web Inspector console. Verify zero CDN requests in Safari Web Inspector network. Run a full attach → type → emit OSC 9;4 progress (e.g. `for i in 1 2 3 4 5 6 7 8 9 10; do printf "\\033]9;4;1;%d\\033\\\\\\n" $((i*10)); sleep 1; done`) → emit sixel (e.g. `cat fixture.sixel`) → scroll back through history → detach → re-attach → confirm scrollback intact.
5. **UAT-5: Cross-browser desktop CSP audit** — open Tailscale URL in Chrome, Firefox, Safari (desktop) on the dev Mac; confirm zero CSP violations in DevTools console for each (this complements the Playwright automated test with a manual smoke).

**Sign-off:** 5 checkboxes; all 5 must PASS before flipping `99-VALIDATION.md` checkboxes.

### Anti-Patterns to Avoid

- **Per-field Save buttons in disclosures.** Forbidden by PUI-04. Sub-key RPCs dispatch immediately on change; the bottom-of-section three-state Save button stays for the 8 booleans only. Mixing dispatch styles confuses users.
- **Re-fetching `pluginConfig` after sub-key dispatch.** Don't. The daemon's `app.go:535-540` already re-fetches and emits `settings:plugins` after each sub-key write; App.tsx's existing listener handles it.
- **Storing disclosure form state in PluginsSection local state.** Don't. Bind directly to `pluginConfig` props (controlled components). Local state would need a re-seed pattern like FindBar's `seededRef`, which is brittle for advanced config. Controlled-from-prop matches PUI-04's "no new save infrastructure" intent.
- **Calling `SetPluginSettings(full)` from a disclosure's onChange.** Forbidden — races the user's in-flight Save button click. Use the sub-key RPCs.
- **Snapshotting `pluginConfig` at PluginsSection mount and editing the snapshot.** Bug! `pluginConfig` is propagated via the `settings:plugins` event; the mounted snapshot would go stale after a sub-key dispatch. Use the live prop.
- **Adding new CSS classes for the disclosure forms.** Reuse `.settings-panel__details`, `.settings-panel__toggle-row`, `.settings-panel__description`. Phase 99 should not introduce visual novelty at the release gate.
- **Bumping Playwright version for cross-browser support.** `^1.59.1` already supports all three engines via `projects[]`. Version bump = unnecessary risk at the release gate.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Sub-key persistence | A custom `PartialUpdate` JSON-merge endpoint | `SetSearchConfig` / `SetWebLinksConfig` / `SetImageConfig` (already shipped) | Surgical sub-key writers preserve concurrent boolean edits in the Plugins-tab edit buffer (Phase 94-07 WR-03 lesson). |
| Multi-browser e2e harness | Custom WebDriver / Selenium / Cypress matrix | Playwright `projects[]` array | Existing Playwright infrastructure handles all three engines uniformly. |
| BannerStack one-shot UI | New banner component file | Existing `.webgl-recovery-banner` BEM class + `useEffect(setTimeout)` pattern | Visual + behavioral consistency with Phase 81/93/97 banners. |
| Migration test framework | Custom golden-file diff harness | Standard Go `testing` + struct-equality + `os.Stat` mtime check | Already proven in Phase 92. |
| Disclosure UI | A `<dialog>`, modal, or popover | Native `<details>`/`<summary>` elements | Browser-native, accessible, no JS state to manage, matches existing SettingsTab.tsx Tailscale-diagnostics pattern. |
| iPad Safari remote debug | Custom debug HTTP endpoint | Safari Web Inspector remote debugging from dev Mac | Built-in iOS feature; no app changes needed. |
| Cross-browser CSP message normalization | Per-browser regex variants | Single case-insensitive regex `/content security policy\|csp/i` | All three engines emit the literal "content security policy" or "csp" in violation messages. |

**Key insight:** Phase 99 is **release-gate hygiene**, not new feature work. The right move at every fork in the road is "use the existing v3.2 primitive," not "design a Phase 99 special case."

---

## Runtime State Inventory

> Phase 99 is a polish + verification phase. It modifies UI rendering paths, adds a banner state, and adds Playwright projects. It does NOT rename, rebrand, refactor, or migrate runtime state. The 5-category inventory below is brief.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — `settings.json` schema is unchanged from Phase 92's `schemaVersion: 2`. PUI-03 disclosure forms write into existing sub-key fields (SearchConfig, WebLinksConfig, ImageConfig) that are already part of the persisted struct. | None — verified by reading `internal/daemon/plugin_settings.go` (no new fields added in Phase 99). |
| Live service config | None — Phase 99 does not touch external service configuration. | None. |
| OS-registered state | None — no Windows Task Scheduler, pm2, launchd, systemd interactions. Tray progress glyphs (Phase 98) are unaffected. | None. |
| Secrets/env vars | None — no new env vars, no secret rotation, no SOPS keys touched. | None. |
| Build artifacts / installed packages | Playwright browser binaries (`firefox`, `webkit`) need `npx playwright install --with-deps firefox webkit` on local dev machines and CI runners that previously ran `chromium` only. This is a one-time CI workflow update. | **Code edit only:** add `npx playwright install --with-deps firefox webkit` step to whichever GitHub Actions workflow runs `pnpm exec playwright test` (likely `.github/workflows/build.yml` or a new e2e workflow — confirm during planning). |

**The canonical question — after every file is updated, what runtime systems still have stale state?**
- iPad Safari `sessionStorage` for the WebGL banner one-shot dismiss flag (Phase 93 / `webglBannerDismissed`) is per-tab and clears naturally. Phase 99 should NOT introduce new sessionStorage keys — the new `pluginToggleBanners` state is in-memory React state, dismissed-only-during-session, no persistence.
- The new Playwright `firefox` + `webkit` browser binaries on CI runners — if a runner is reused across PRs, the browsers persist (idempotent), so no cleanup needed. First run on each runner downloads ~300 MB of browser binaries; subsequent runs are cached.

---

## Common Pitfalls

### Pitfall 1: Disclosure form re-renders disrupt FindBar's `seededRef` invariant

**What goes wrong:** PluginsSection's Search disclosure dispatches `SetSearchConfig`. The daemon emits `settings:plugins` with the new SearchConfig. App.tsx's listener updates `pluginConfig`. TerminalPanel re-renders with new `pluginConfig.searchConfig`. The seededRef useEffect at `TerminalPanel.tsx:166-178` checks `findBarOpen` to avoid mid-open re-seeding — but if the find bar is closed, the seed fires and the user's NEXT find-bar open shows the new defaults. **This is correct behavior.** The pitfall is if a future Phase 99 task accidentally clears `seededRef.current = false` on `pluginConfig.searchConfig` change, then mid-open re-seeds would happen.
**Why it happens:** Misreading Pitfall #2 from Phase 94 — the rule is "never re-seed mid-open", not "always re-seed on prop change".
**How to avoid:** Do NOT touch `seededRef` in Phase 99. The disclosure form writes through to `pluginConfig.searchConfig`; the find bar reads its own local `searchOptions` state. They are visually in sync once the user closes and re-opens the find bar.
**Warning signs:** Find bar's regex/case/word toggles flip from under the user mid-search. Test: open find bar, then in another window/tab go to Settings, change Search defaults, watch the find bar — toggles should NOT change while bar is open.

### Pitfall 2: PUI-02 BannerStack double-fires

**What goes wrong:** User opens Settings, toggles Unicode 11 OFF, presses Save → banner appears. They press Save AGAIN (idle re-click) → banner appears AGAIN. The banner is annoyingly repetitive.
**Why it happens:** Naive diff: `if (priorUnicode11 !== savedUnicode11) emit`. After the first save, `priorUnicode11` becomes the new saved value, so a second click with no toggle change has `priorUnicode11 === savedUnicode11` and does NOT fire — **but if the user toggles it back ON and saves, then OFF and saves**, two banners stack quickly.
**How to avoid:** Diff against the value AT THE START of the current Save click (not against whatever was last fed by `settings:plugins`). The PluginsSection's local edit buffer (`pluginConfig` state in PluginsSection.tsx:21) IS this snapshot — diff `local.unicode11 !== priorSaved.unicode11` where `priorSaved` is captured at handleSavePlugins entry. Idempotent.
**Warning signs:** Toggle thrash produces banner thrash. Test: rapidly toggle Unicode 11 ON / OFF / ON / OFF / Save → exactly one banner if the final state differs from the initial state, zero if it matches.

### Pitfall 3: Playwright `webkit` project on Linux CI lacks codecs

**What goes wrong:** `webkit` on Linux CI runners can fail to launch or fails to render fonts (the bundled WebKit lacks system fonts). CSP violations can spuriously fire if a font fails to load and triggers a `font-src` violation.
**Why it happens:** Playwright's `webkit` is a stripped Mac-Safari-equivalent on Linux; it doesn't have the macOS system font fallback chain.
**How to avoid:** Run `npx playwright install --with-deps webkit` on CI to pull down the OS shared libs. Use the Playwright-bundled fonts. If a `font-src` violation appears in the test, it's a v3.1 CSP gap (font-src is `'self'` only, no Google Fonts), NOT a Phase 99 regression — investigate before suppressing.
**Warning signs:** Webkit-only CSP violations on CI that don't reproduce on the dev Mac running real Safari. Mitigation: also run a manual desktop-Safari smoke (UAT-5) to distinguish "CI environment artifact" from "real Safari issue".

### Pitfall 4: Migration test fixture drifts from current default schema

**What goes wrong:** A future phase adds a new sub-config (e.g., `ClipboardConfig`) to `PluginSettings` and bumps `CurrentSchemaVersion = 3`. The v3.1 fixture (`tests/fixtures/settings_v3.1.json`) STILL has no plugins block, so the test passes (defaults-merge still populates everything). But a v3.2-rc fixture (if added) would NOT have the new sub-config, and the test would falsely pass because struct equality compares the merged-in default to itself.
**Why it happens:** Defaults-merge is one-directional (missing keys → defaults). A test fixture that's "almost current schema" can hide regressions where defaults aren't merged.
**How to avoid:** **Do not add intermediate-schema fixtures in Phase 99.** Keep only `settings_v3.1.json` (load-bearing). Future schema bumps should add `settings_v3.{Y}.json` fixtures at THAT phase, not retroactively.
**Warning signs:** A new sub-config added in Phase 100 shows zero values in production for v3.2-rc users despite the migration test being green. Mitigation: each schema bump's phase research must inventory ALL prior fixtures and consciously decide which to upgrade.

### Pitfall 5: iPad Safari Web Inspector requires "Web Inspector" enabled in iPad Settings

**What goes wrong:** UAT-4 instructs the tester to "open Safari Web Inspector from the dev Mac" but the iPad has Web Inspector OFF by default. The tester sees no inspector option in dev Mac Safari's Develop menu.
**Why it happens:** Apple gates remote debugging behind a Settings → Safari → Advanced → Web Inspector toggle.
**How to avoid:** The `99-iPad-UAT.md` runbook MUST include a "Prerequisites" step with: "On iPad: Settings → Safari → Advanced → Web Inspector → ON. On dev Mac Safari: Settings → Advanced → Show Develop menu → ON."
**Warning signs:** Tester reports "I can't see the iPad in the Develop menu". Solution: prerequisite check.

### Pitfall 6: PUI-03 number input accepts non-integer storageLimit

**What goes wrong:** User types `16.5` in the storageLimit number input. The daemon's `handleSetImageConfig` validates `[1, 1000]` but accepts non-integer (Go `int` truncates: `16.5` → `16`). Confusing UX.
**Why it happens:** HTML `<input type="number">` allows decimals by default unless `step="1"` is set.
**How to avoid:** Set `step="1"` AND `min="1"` AND `max="1000"` on the input. Additionally, JS guard: `Math.floor(Number(e.target.value))` before dispatch. The daemon's range gate is the load-bearing validator (defense-in-depth).
**Warning signs:** `imageConfig.storageLimit` ends up as a fractional value in the daemon-side `pluginSettings.json`. Mitigation: assert in the unit test that the dispatched value is always an integer.

### Pitfall 7: BannerStack copy violates v3.1 information-disclosure mitigation pattern

**What goes wrong:** A naive banner copy might say "Toggle Unicode 11 changed; restart your terminal" or include version numbers. Phase 93's WebGL recovery banner explicitly avoids exposing implementation details ("SwiftShader", "llvmpipe", "ANGLE") per the information-disclosure mitigation pattern documented in `WebGLRecoveryBanner.tsx:23`.
**Why it happens:** Implementation details in user-facing copy invite reconnaissance for attackers and confuse users.
**How to avoid:** Use the locked verbatim copy: *"Open a new terminal session to apply the Unicode 11 change."* / *"Open a new terminal session to apply the Inline Images change."* — plain English, no jargon, no version numbers, no internal class names.
**Warning signs:** Banner text contains words like "addon", "xterm", "WebAssembly", "decoder". Mitigation: locked-copy assertion in component test.

---

## Code Examples

Verified patterns from official sources / existing codebase:

### Example 1: Existing italic caption (PUI-02 partial — read but do not modify)
```tsx
// Source: frontend/src/components/PluginsSection.tsx:107-111 (existing — Phase 93 U11-01)
{caption && (
  <p className="settings-panel__description settings-panel__description--italic">
    {caption}
  </p>
)}
```

### Example 2: One-shot BannerStack (PUI-02 — extend existing pattern)
```tsx
// Pattern source: frontend/src/components/WebGLRecoveryBanner.tsx (Phase 93)
// Adapted for plugin-toggle confirmation
import React, { useEffect } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

export type PluginToggleKind = 'unicode11' | 'image'

const COPY: Record<PluginToggleKind, string> = {
  unicode11: 'Open a new terminal session to apply the Unicode 11 change.',
  image: 'Open a new terminal session to apply the Inline Images change.',
}

export function PluginToggleBanner(props: {
  kind: PluginToggleKind
  onDismiss: () => void
}): React.ReactElement {
  useEffect(() => {
    const id = window.setTimeout(props.onDismiss, 6000)
    return () => window.clearTimeout(id)
  }, [props.onDismiss])
  return (
    <div className="webgl-recovery-banner" role="status" aria-live="polite">
      <span className="webgl-recovery-banner__message">{COPY[props.kind]}</span>
      <button
        type="button"
        className="webgl-recovery-banner__dismiss"
        aria-label="Dismiss notification"
        onClick={props.onDismiss}
      >
        <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
      </button>
    </div>
  )
}
```

### Example 3: Sub-key RPC dispatch from disclosure form (PUI-03)
```tsx
// Pattern source: frontend/src/components/TerminalPanel.tsx:719-728 (Phase 94-07 WR-03)
import { SetSearchConfig, SetWebLinksConfig, SetImageConfig } from '../wailsjs/go/main/App'
import { daemon } from '../wailsjs/go/models'

// Inside PluginsSection, in a Search disclosure render:
<details className="settings-panel__details">
  <summary>Search defaults</summary>
  <label className="settings-panel__toggle-row">
    <input
      type="checkbox"
      checked={pluginConfig?.searchConfig?.regex ?? false}
      onChange={(e) => {
        const next = new daemon.SearchConfig({
          ...(pluginConfig?.searchConfig ?? {}),
          regex: e.target.checked,
        })
        SetSearchConfig(next).catch(() => {
          // The next settings:plugins event will reconcile the prop.
        })
      }}
    />
    <span className="settings-panel__toggle-label">Regex</span>
  </label>
  {/* ...caseSensitive, wholeWord ... */}
</details>
```

### Example 4: Cross-browser Playwright config (SC-4)
```typescript
// Source: extending frontend/playwright.config.ts:31-37
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  // ...existing config unchanged...
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox',  use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit',   use: { ...devices['Desktop Safari'] } },
  ],
})
```

### Example 5: Migration test assertion expansion (SC-3)
```go
// Source: extending internal/daemon/engine_migration_test.go (Phase 92)
func TestSettingsMigrationV3_1ToV3_2_AllDefaultsPopulated(t *testing.T) {
    dir := t.TempDir()
    copyFixtureToTempDir(t, dir)

    e := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
    e.loadSettingsFromDisk(dir)

    got := e.GetPluginSettings()
    want := defaultPluginSettings()

    // 8 plugin booleans
    if got.WebGL != true     { t.Errorf("WebGL: got %v, want true", got.WebGL) }
    if got.Unicode11 != true { t.Errorf("Unicode11: got %v, want true", got.Unicode11) }
    if got.Search != true    { t.Errorf("Search: got %v, want true", got.Search) }
    if got.WebLinks != true  { t.Errorf("WebLinks: got %v, want true", got.WebLinks) }
    if got.Image != true     { t.Errorf("Image: got %v, want true", got.Image) }
    if got.Serialize != true { t.Errorf("Serialize: got %v, want true", got.Serialize) }
    if got.Clipboard != true { t.Errorf("Clipboard: got %v, want true", got.Clipboard) }
    if got.Progress != false { t.Errorf("Progress: got %v, want false (P2 v3.2 OFF)", got.Progress) }

    // 3 sub-config defaults
    if got.SearchConfig != (SearchConfig{}) {
        t.Errorf("SearchConfig: got %+v, want zero-value", got.SearchConfig)
    }
    if got.WebLinksConfig.Modifier != "platform" {
        t.Errorf("WebLinksConfig.Modifier: got %q, want \"platform\"", got.WebLinksConfig.Modifier)
    }
    if !got.WebLinksConfig.ConfirmOSC8 || !got.WebLinksConfig.ConfirmIDN || !got.WebLinksConfig.ConfirmTyposquat {
        t.Errorf("WebLinksConfig confirm flags: got %+v, want all true", got.WebLinksConfig)
    }
    if got.ImageConfig.StorageLimit != 16 {
        t.Errorf("ImageConfig.StorageLimit: got %d, want 16 (Phase 96 default)", got.ImageConfig.StorageLimit)
    }

    // Belt-and-braces struct-equality
    if got != want {
        t.Errorf("PluginSettings != defaultPluginSettings(): got %+v, want %+v", got, want)
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Chromium-only e2e CSP suite (`projects: [chromium]`) | Multi-browser via Playwright `projects[]` | Phase 99 (this phase) | Surfaces WebKit/Firefox-specific CSP regressions before iPad Safari UAT. |
| Full-snapshot `SetPluginSettings` from every UI surface | Sub-key RPCs (`SetSearchConfig` etc) for in-place advanced config | Phase 94-07 WR-03 (pattern) → Phase 99 (UI consumption) | Prevents Plugins-tab edit buffer races (lesson from Phase 94-07). |
| Italic caption added at Phase 93 (U11-01) and Phase 96 (IMG-01 — already shipped) | Same caption + new BannerStack toast | Phase 99 (this phase) | Reinforces next-session-only affordance with a transient confirmation, reducing user confusion. |
| `<details>` disclosures absent from Plugins section | Three new disclosures (Search, Web Links, Image) | Phase 99 (this phase) | Exposes runtime config without overwhelming the default Plugins view. |

**Deprecated/outdated:**
- **Naive `json.Unmarshal` for settings load** — replaced with defaults-merge in Phase 92. Never revert.
- **CDN-loaded xterm assets** — replaced with vendored same-origin in Phase 89. Phase 93/94/95/96/97/98 each vendor their addon bundle. Phase 93's `vendor_drift_test.go` is the load-bearing CI gate; if any addon shows a CDN reference at the Phase 99 cross-browser audit, that's a Phase 93 regression, not a Phase 99 issue.

---

## Assumptions Log

> All claims tagged `[ASSUMED]` in this research. The planner and discuss-phase use this section to identify decisions that need user confirmation before execution.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The user wants the BannerStack copy verbatim as: *"Open a new terminal session to apply the Unicode 11 change."* / *"Open a new terminal session to apply the Inline Images change."* | Pattern 2; Claude's Discretion | Low — copy can be tweaked in plan execution; user can review during UAT-2. |
| A2 | The user wants the BannerStack to auto-dismiss after 6000ms (vs. WebGL banner's 8000ms or save banner's never). | Claude's Discretion | Low — the timing is empirical; tweak if UAT feedback says too short or too long. |
| A3 | The user wants both Unicode 11 and Image banners to stack (multiple at once) when both toggled in one save, rather than a single combined banner. | Pattern 2 | Medium — combined "Open a new session to apply your changes" is one alternative; stacked is more explicit. Honor the user's preference if discussed. |
| A4 | The user wants `<details>` disclosures default-CLOSED (browser default). | Claude's Discretion | Low — `open` attribute is a one-line change. |
| A5 | The user accepts inline render functions in PluginsSection.tsx rather than separate `PluginConfigSearch.tsx` / `PluginConfigWebLinks.tsx` / `PluginConfigImage.tsx` files. | Deferred Ideas | Low — file extraction is a future refactor; either approach satisfies PUI-03. |
| A6 | The user wants the iPad Safari UAT to test ALL plugins enabled (default 7 ON + Progress ON via toggle), not just default state. | Locked Decisions; UAT plan | Low — both are needed; "all-plugins" is the stricter test and matches ROADMAP SC-4 verbatim. |
| A7 | The user wants the storageLimit number input as `<input type="number" step="1" min="1" max="1000">` rather than a slider or radio group of presets. | Claude's Discretion | Low — UX choice; the daemon range gate is `[1, 1000]` regardless. |

**If this table needs review:** The planner should re-confirm A1, A2, A3 before locking PUI-02 banner copy in plan tasks.

---

## Open Questions

1. **Should the migration test be parametrized over multiple v3.1 fixture variants (no plugins / partial plugins / all plugins)?**
   - What we know: Existing `settings_v3.1.json` is minimal (no plugins block, no schemaVersion). Defaults-merge populates everything correctly.
   - What's unclear: Whether a "v3.2-rc partial" fixture (some plugins toggled, no sub-configs) would catch a regression that the minimal fixture misses.
   - Recommendation: **Skip parametrization for Phase 99.** The minimal fixture is load-bearing; a v3.2-rc fixture is theoretical (no real users have it). Add only if a real bug surfaces.

2. **Should the cross-browser CSP suite run on every PR or only on release tags?**
   - What we know: Existing chromium-only run is on every PR (CI-cheap).
   - What's unclear: Whether adding firefox + webkit triples CI time per PR.
   - Recommendation: **Run all three on every PR.** The browser binaries are cached after first install; subsequent runs are fast (~30s for the spec across 3 projects). If CI time becomes an issue, gate firefox + webkit on `release/*` branches only — but start with always-on.

3. **Does the iPad Safari Tailscale UAT happen pre-tag (rc) or post-tag (final)?**
   - What we know: ROADMAP SC-4 says "iPad Safari Tailscale UAT (real device) reports zero CSP violations" — implies pre-tag (gate on UAT pass before tagging final).
   - What's unclear: Whether the `99-iPad-UAT.md` runbook should reference an rc tag or a build-from-main.
   - Recommendation: **Pre-tag.** Run UAT against an rc build (`v3.2.0-rc1.dmg` produced by the existing Phase 90 release pipeline). UAT pass → cut final tag. Failure → fix → rc2.

4. **Are Playwright `firefox` and `webkit` browser binaries already pre-installed on the v3.1 CI runners?**
   - What we know: Phase 93 Plan 05 added `chromium` install (likely via `npx playwright install chromium`).
   - What's unclear: Whether the install step pulls all three by default or only chromium.
   - Recommendation: **Assume not.** Add `npx playwright install --with-deps firefox webkit` explicitly in the workflow file. Idempotent; no harm if already cached.

5. **Should the disclosure form's onChange dispatch be debounced for the storageLimit number input?**
   - What we know: User typing `1`, `16`, `160` in sequence would fire 3 RPCs. Each writes to disk and broadcasts SSE.
   - What's unclear: Whether the daemon-side write rate is a problem.
   - Recommendation: **Yes, debounce 500ms** for the number input only. Toggle-style fields (regex/case/word/confirm*/modifier) are single-event and don't need debounce.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js + pnpm | Frontend build, vitest, Playwright | ✓ | (per project lockfile) | — |
| Go 1.21+ | Daemon test (engine_migration_test) | ✓ | (per go.mod) | — |
| Playwright `chromium` browser | Existing e2e | ✓ | bundled with `@playwright/test ^1.59.1` | — |
| Playwright `firefox` browser | Phase 99 cross-browser e2e | TBD on each runner | — | Run `npx playwright install --with-deps firefox` (auto-pulls; ~150 MB) |
| Playwright `webkit` browser | Phase 99 cross-browser e2e | TBD on each runner | — | Run `npx playwright install --with-deps webkit` (auto-pulls; ~150 MB) |
| Real iPad with iOS 16+ | UAT-4 (iPad Safari Tailscale) | Available (per CLAUDE.md / project memory — Tailscale Mac+iPad setup) | — | iOS Simulator is NOT a substitute (per ROADMAP SC-4). If iPad unavailable: defer UAT-4 sign-off and flag in 99-VERIFICATION. |
| Tailscale on iPad joined to dev tailnet | UAT-4 | Available (Phase 93 UAT-1..UAT-5 prerequisite, already proven) | — | — |
| Safari Web Inspector (remote debug from dev Mac) | UAT-4 zero-CSP, zero-CDN audit | Built-in macOS feature | — | Use mobile Safari console (less convenient but works). |
| `chafa` CLI | UAT-4 sixel image emit (optional) | TBD on dev Mac | — | Hand-craft a synthetic sixel byte stream (see Phase 96 RESEARCH) |

**Missing dependencies with no fallback:** None blocking. iPad availability is the only single-point-of-failure; runbook will document graceful deferral if hardware unavailable.

**Missing dependencies with fallback:** Playwright `firefox` + `webkit` (auto-installed via CLI on first test run).

---

## Validation Architecture

> Required because `workflow.nyquist_validation` is not explicitly false in `.planning/config.json` (treat as enabled).

### Test Framework

| Property | Value |
|----------|-------|
| Frontend framework | `vitest@^4.1.0` (component tests for PluginsSection extensions) |
| E2E framework | `@playwright/test@^1.59.1` (cross-browser CSP zero-violation) |
| Backend framework | Go `testing` (standard library) |
| Frontend config file | `frontend/vitest.config.ts` (existing) |
| E2E config file | `frontend/playwright.config.ts` (existing — Phase 99 modifies `projects[]`) |
| Backend config file | `go.mod` (root) |
| Quick run command (frontend) | `cd frontend && pnpm test` |
| Quick run command (backend) | `go test ./internal/daemon/... -count=1 -run TestSettingsMigration` |
| Full suite command (frontend) | `cd frontend && pnpm test && pnpm exec playwright test` |
| Full suite command (backend) | `go test ./... -count=1` |
| Cross-browser e2e command | `cd frontend && pnpm exec playwright test --project=chromium --project=firefox --project=webkit` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PUI-02 (caption) | Italic caption "Applies to new sessions you create." renders verbatim under Unicode 11 and Inline Images rows | unit (vitest) | `cd frontend && pnpm test -- PluginsSection.test.tsx` | ✅ (test file exists; assertion may need extension) |
| PUI-02 (banner) | One-shot BannerStack appears after Unicode 11 / Image toggle save with verbatim copy + auto-dismiss after 6000ms | unit (vitest) + manual UAT-2 | `cd frontend && pnpm test -- App.test.tsx` (assert pluginToggleBanner state); manual = UAT-2 in `99-iPad-UAT.md` | ❌ Wave 0 (assertion needs new component test for `PluginToggleBanner.test.tsx`) |
| PUI-03 (Search disclosure) | `<details>` opens; toggling regex dispatches SetSearchConfig with new value; pluginConfig prop reflects new value | unit (vitest) — mock SetSearchConfig + assert dispatch payload | `cd frontend && pnpm test -- PluginsSection.test.tsx` | ❌ Wave 0 (test extension) |
| PUI-03 (Web Links disclosure) | `<details>` opens; modifier select changes dispatch SetWebLinksConfig; confirm checkboxes dispatch correctly | unit (vitest) | same | ❌ Wave 0 |
| PUI-03 (Image disclosure) | `<details>` opens; storageLimit number input within [1,1000] dispatches SetImageConfig; debounced 500ms | unit (vitest) | same | ❌ Wave 0 |
| PUI-04 (Save button reuse) | The bottom three-state Save button continues to function for the 8 booleans; sub-key dispatches do NOT trigger the Save button | unit (vitest) — assert button state independence | same | ✅ existing assertions; Phase 99 may extend |
| SC-3 (migration) | v3.1 fixture loads with all 8 plugin defaults + 3 sub-config defaults populated; schemaVersion: 2 written; idempotent on second load | unit (Go) | `go test ./internal/daemon/... -run TestSettingsMigration -count=1` | ✅ existing tests; Phase 99 expands assertions |
| SC-4 (cross-browser CSP) | Zero CSP violations on chromium, firefox, webkit during attach + scroll session | e2e (Playwright × 3 projects) | `cd frontend && pnpm exec playwright test web-csp.spec.ts --project=chromium --project=firefox --project=webkit` | ✅ spec exists; Phase 99 adds two browser projects to config |
| SC-4 (iPad Safari Tailscale UAT) | Zero CSP violations + zero CDN requests during real-device attach/render/scrollback/detach with all plugins enabled | manual UAT (real iPad) | `99-iPad-UAT.md` UAT-4 | ❌ Wave 4 (new runbook file) |

### Sampling Rate

- **Per task commit:** `cd frontend && pnpm test` + `go test ./internal/daemon/... -count=1`
- **Per wave merge:** above + `cd frontend && pnpm exec playwright test --project=chromium` (smoke; defer multi-browser to wave 3)
- **Phase gate (before `/gsd-verify-work 99`):** full suite green across all 3 browsers + iPad UAT-4 signed off + UAT-1/2/3/5 signed off

### Wave 0 Gaps

- [ ] `frontend/src/components/PluginToggleBanner.tsx` — new component (mirrors `WebGLRecoveryBanner.tsx`)
- [ ] `frontend/src/components/__tests__/PluginToggleBanner.test.tsx` — new component test (verbatim copy + 6000ms auto-dismiss + dismiss button)
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` — extend with disclosure-render assertions and sub-key RPC dispatch assertions (mock `SetSearchConfig`, `SetWebLinksConfig`, `SetImageConfig`)
- [ ] `frontend/src/components/__tests__/App.test.tsx` — extend with `pluginToggleBanner` state-management assertions (a banner enters the stack after a save with a Unicode 11 / Image toggle change; multiple banners stack; dismissal works)
- [ ] `internal/daemon/engine_migration_test.go` — extend `TestSettingsMigrationV3_1ToV3_2` with explicit per-field assertions for all 8 booleans + 3 sub-configs (current test does struct-equality only)
- [ ] `frontend/playwright.config.ts` — add `firefox` + `webkit` projects
- [ ] `.github/workflows/<e2e workflow>` — add `npx playwright install --with-deps firefox webkit` step (confirm exact workflow file during planning)
- [ ] `.planning/phases/99-.../99-iPad-UAT.md` — new runbook (5 scenarios; clones Phase 93 shape)

---

## Security Domain

> Required when `security_enforcement` is enabled (absent in config; treat as enabled).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | partial | Capability tokens (Phase 87) gate web access; Phase 99 does not modify the auth surface. |
| V3 Session Management | no | Phase 99 does not introduce new sessions. |
| V4 Access Control | partial | Sub-key RPCs are exposed via `(*App)` Wails methods (desktop privileged), and via `PATCH /settings/*-config` on the daemon's Unix-socket API (loopback only). The `/api/plugin-config` web endpoint is read-only (capability-gated); web users CANNOT write sub-configs. |
| V5 Input Validation | yes | `handleSetImageConfig` already validates `[1, 1000]` range. PUI-03 form must additionally validate client-side (`min`/`max`/`step="1"` attributes + `Math.floor`) for UX, but the daemon range gate is the load-bearing validator. |
| V6 Cryptography | no | No new crypto in Phase 99. Capability signing keys (Phase 87) unchanged. |
| V11 Business Logic | partial | Sub-key RPCs preserve concurrent-edit semantics (Phase 94-07 WR-03 lesson). Disclosure form must NOT call `SetPluginSettings(full snapshot)` from a stale local buffer. |
| V14 Configuration | yes | CSP `script-src 'self' 'wasm-unsafe-eval'` (Phase 96 amendment) verified across all three browser engines. No new directives in Phase 99. |

### Known Threat Patterns for Settings UI + cross-browser e2e

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| User-typed XSS in storageLimit input | Tampering | Input is `type="number"` and writes to a Go `int`; XSS impossible. Defense-in-depth: daemon range gate. |
| CSP bypass via WebKit-specific quirk | Tampering / Information Disclosure | Cross-browser e2e suite asserts zero violations on all three engines; iPad UAT confirms real-device behavior. |
| Cross-tab Settings race (two windows editing simultaneously) | Tampering | Sub-key RPCs serialize through `e.mu.Lock()`. Last-write-wins on the daemon is the documented semantic. SSE broadcast updates other windows' `pluginConfig` prop. |
| Phishing via misleading banner copy | Information Disclosure | Locked verbatim copy; no implementation details (matches Phase 93 information-disclosure mitigation pattern). |
| Capability token leak via Playwright trace | Information Disclosure | Phase 93 e2e fixture already isolates the cap token in a temp file (`.playwright/fixture-env.json`); not committed. CI artifacts must NOT include this file. |
| OSC 8 spoofing via disclosure-changed modifier | Spoofing | Phase 95's LinkConfirmPopover handles this; modifier change does not weaken the OSC 8 confirmation gate. |

---

## Sources

### Primary (HIGH confidence — verified by direct codebase read or prior-phase RESEARCH)

- `frontend/src/components/PluginsSection.tsx:107-111,130,136,144,152-160` — italic caption already shipped, three-state Save button already shipped
- `frontend/src/components/SettingsTab.tsx:441-473` — existing `<details>` disclosure pattern
- `frontend/src/style.css:573-583` — `.settings-panel__details` CSS rules
- `frontend/src/style.css:1605-1607` — `.settings-panel__description--italic` CSS rule (Phase 93 U11-01)
- `frontend/src/style.css:1612-1659` — `.webgl-recovery-banner` CSS for one-shot banner reuse
- `frontend/src/App.tsx:114,890-933` — saveBanner / localBanner / webglBanner pattern for one-shot BannerStack
- `frontend/src/App.tsx:464-470` — `settings:plugins` event subscription
- `frontend/src/components/WebGLRecoveryBanner.tsx` — banner component pattern with auto-dismiss + verbatim copy
- `frontend/src/components/TerminalPanel.tsx:166-178,191-192,719-728` — `seededRef` pattern for FindBar; `webLinksConfigRef` live-read pattern; `SetSearchConfig` dispatch site precedent
- `frontend/playwright.config.ts:31-37` — existing `projects[]` array with chromium only
- `frontend/e2e/web-csp.spec.ts` — existing CSP zero-violation spec (browser-engine-agnostic)
- `internal/daemon/plugin_settings.go` — `PluginSettings`, `SearchConfig`, `WebLinksConfig`, `ImageConfig` struct definitions; `defaultPluginSettings()` source of truth
- `internal/daemon/engine.go:480-563` — `SetSearchConfig`, `SetWebLinksConfig`, `SetImageConfig` sub-key writers (mutate-under-lock + listener)
- `internal/daemon/engine_migration_test.go` — `TestSettingsMigrationV3_1ToV3_2` and `TestSettingsMigrationIdempotent` (load-bearing CI gates)
- `internal/daemon/api.go:74-78` — daemon HTTP routes for sub-key PATCH endpoints
- `internal/webserver/csp_mw.go:107-113` — CSP `script-src 'self' 'wasm-unsafe-eval'` (Phase 96 Amendment 2)
- `internal/webserver/server.go:83-130` — pluginSettingsProvider + SSE subscriber registry
- `app.go:472-606` — Wails App methods for GetPluginSettings, SetPluginSettings, SetSearchConfig, SetWebLinksConfig, SetImageConfig
- `frontend/src/wailsjs/go/main/App.d.ts:131,137,143` — TypeScript bindings for the three sub-key RPCs
- `tests/fixtures/settings_v3.1.json` — minimal v3.1 fixture (cliPaths + startMinimized + autoCloseSession)
- `.planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-iPad-UAT.md` — UAT runbook precedent (5 scenarios + sign-off)
- `.planning/phases/96-image-addon-csp-audit/96-RESEARCH.md` — CSP `'wasm-unsafe-eval'` browser support matrix and audit
- `.planning/phases/92-plugin-settings-foundation/92-UI-SPEC.md` — Plugins section design contract; deferred items list (PUI-02 BannerStack + PUI-03 disclosures explicitly deferred to Phase 99)
- `.planning/phases/92-plugin-settings-foundation/92-RESEARCH.md` — Defaults-merge pattern; migration test approach
- `.planning/REQUIREMENTS.md` — PUI-02, PUI-03, PUI-04 verbatim
- `.planning/STATE.md` — Phase 94-07, 95, 96 sub-key RPC pattern decisions
- `.planning/ROADMAP.md` — Phase 99 success criteria (SC-1..SC-4) verbatim

### Secondary (MEDIUM confidence — verified by prior-phase research cross-reference)

- Playwright multi-browser project syntax — `@playwright/test ^1.59.1` package version verified in `frontend/package.json`; `projects[]` API has been stable since Playwright 1.0.
- `'wasm-unsafe-eval'` browser support matrix (Chrome 102+, Firefox 102+, Safari 16+, iPad Safari 16+) — cross-verified by Phase 96 RESEARCH against caniuse and CSP3 spec; multiple primary sources agree.
- `<details>` element support — universal across all three browser engines for ~10+ years.

### Tertiary (LOW confidence — none required for Phase 99)

(no LOW-confidence claims; this phase reuses verified primitives end-to-end)

---

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — every dependency already installed; no new packages.
- Architecture (sub-key RPC reuse, BannerStack pattern, `<details>` disclosure): HIGH — all primitives shipped in prior phases; verified in current codebase.
- Migration verification: HIGH — existing test green; assertions need expansion only.
- Cross-browser CSP: MEDIUM-HIGH — Playwright cross-browser semantics are well-documented; the only unknown is whether CI runners need OS-level browser deps (mitigated by `--with-deps`).
- iPad Safari UAT: HIGH — Phase 93 precedent proves the runbook shape works.
- Pitfalls: HIGH — drawn from Phase 92/93/94/95/96/97/98 lessons learned; every pitfall has a documented prior-phase root cause.

**Research date:** 2026-05-08
**Valid until:** 2026-06-07 (30 days; v3.2 release-gate phase, low volatility — primitives are frozen by definition).

---

## RESEARCH COMPLETE
