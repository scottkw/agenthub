# Phase 173: Share modal three-tab segmented redesign - Research

**Researched:** 2026-07-08
**Domain:** React/TypeScript frontend component refactor (Wails desktop app), CSS design-token migration
**Confidence:** HIGH (all claims code-grounded against HEAD commit `40331fa2`; no external libraries involved)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All decisions below are LOCKED by spec #129. Line refs are HEAD-relative starting points — verify before editing.

- **D-01 — Fixed control strip (SM-01):** The three toggles (Share the session / Enable remote file browsing / Enable internet sharing) + the divider live at the top of `.hub-share-modal__body` as a non-scrolling control strip. Toggling a control NEVER reflows or pushes down content already on screen. `Enable remote file browsing` disabled until `shareEnabled`.
- **D-02 — Bounded height, single scroll region (SM-02):** No state causes the whole dialog to scroll. Dialog fits `max-height: calc(100vh - 64px)`; scroll confined to one region (active tab body).
- **D-03 — Three-tab segmented control (SM-03):** New `ShareSegmentedControl`, rendered under the divider, shown only when `shareEnabled`. Segments: `Tailnet`/`Private`, `Internet`/`Read-only`, `Internet`/`⚠ Full access`. Selecting a tab swaps the panel body only. CSS classes: `.share-segbar`, `.share-seg`, `.share-seg__main`, `.share-seg__sub`, `.share-seg.is-active`, `.share-seg.is-danger`.
- **D-04 — Full-access / command-execution flow walled off (SM-04):** Public-write flow (warning → enable → hold-to-confirm gate with expiry select + `HoldToConfirmButton` → armed summary) lives ONLY in `InternetFullAccessTab`. Nothing about public write appears anywhere else.
- **D-05 — Tab state machine + transient confirm view (SM-05):** State inputs: `shareEnabled`, `internetEnabled`, `internetConfirmed`, `publicWriteArmed`.
  - `!shareEnabled` → no tabs; empty hint.
  - `shareEnabled && internetEnabled && !internetConfirmed` → transient confirm view (repurposed `FunnelRiskPanel`) REPLACES the panel; no tabs.
  - `shareEnabled && (!internetEnabled || internetConfirmed)` → segmented control + active tab body.
  - Internet tabs enabled only when `internetConfirmed` (else dimmed `aria-disabled`). On confirm, default active tab = Internet·Read-only. Disabling internet resets active tab to Tailnet and disables both Internet tabs.
- **D-06 — One reusable ShareLinkCard (SM-06):** title · truncated URL (full URL in `title=`) · Copy/Open/QR · join code (wraps `CodeDisplay`) · scope description directly beneath. Used by ALL tailnet + internet link rows. CSS: `.share-linkcard`, `__top`, `__title`, `__url`, `__actions`, `__join`, `__desc`.
- **D-07 — Colorblind-safe + keyboard-operable (SM-07):** Owner is colorblind. Full-access tab distinguished by ⚠ glyph + red inset ring when active (not color alone). Toggles read state by knob position + "On/Off/N/A" text label. Segmented control is real `role="tablist"`/`role="tab"`/`aria-selected`, arrow-key roving tabindex, visible `:focus-visible`. Hold-to-confirm keeps text label + works under `prefers-reduced-motion` (plain confirm fallback, not timed fill).
- **D-08 — Modal width + preserved behavior + tests (SM-08):** Widen `.hub-share-modal` from `width: min(480px, calc(100vw - 48px))` to `width: min(520px, calc(100vw - 48px))`. Capabilities/tokens/TTL/Funnel-teardown/3s hold-gate all UNCHANGED. Tests updated with attribute-based, non-hue assertions.
- **D-09 — Components & preservation:** `SessionShareModal.tsx` (shell) keeps the 3 toggles as fixed control strip; owns tab/confirm state (`tab`, `internetConfirmed`, `publicWriteArmed`); renders `ShareSegmentedControl` + active-tab dispatch. `ShareSegmentedControl` NEW. `ShareLinkCard` NEW/extracted. `SessionSharePanel.tsx` refactored into `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab`. PRESERVE as-is: `HoldToConfirmButton`, `CodeDisplay`. `FunnelRiskPanel` repurposed into the transient confirm view. Migrate inline `style={}` to CSS classes in `frontend/src/style.css` (no CSS module).

### Claude's Discretion
- Exact new-component file paths/locations (follow existing `components/` + `components/Hub/` conventions), prop signatures, and internal helper structure.
- How much of `SessionSharePanel` becomes shared helpers vs per-tab code, as long as the three-tab-renderer decomposition holds.
- Wave/plan decomposition and test-file organization details.
- Whether QR toggle lives on the card vs a shared modal-level popover, provided each card exposes a QR affordance.

### Deferred Ideas (OUT OF SCOPE)
- Rejected layout alternatives (accordion, wizard/stepper, fixed-height-inner-scroll) — documented in #129 as considered-and-rejected; fixed-height inner-scroll noted only as a fallback if the segmented rework can't land in one phase.
- No changes to capabilities/tokens/TTL/Funnel teardown, and no changes to the Sharing Guide Help content (explicitly out of scope).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SM-01 | Fixed control strip — toggles pinned, toggling never reflows/pushes content | See "Current State Map — SessionShareModal.tsx §Control Strip" below: exact JSX block (lines 566-687) to keep at top of `__body`, unmodified except removing the inline `FunnelRiskPanel` mount from that region |
| SM-02 | Bounded height — no state scrolls the whole dialog | `.hub-share-modal` (max-height, HEAD line 6400) + `.hub-share-modal__body` (overflow-y:auto, HEAD line 6440) already implement the outer bound; new tab-body wrapper needs its own `overflow-y:auto`/`min-height:0` so only IT scrolls, not `__body` |
| SM-03 | Three-tab segmented control that swaps panel body | New `ShareSegmentedControl` — no existing tablist precedent in codebase (grep confirmed); build from scratch per spec code sketch |
| SM-04 | Public-write flow lives ONLY in Full-access tab | `SessionSharePanel.tsx` lines 603-724 (`hub-funnel-write-gate` block) is the exact code to extract verbatim into `InternetFullAccessTab` |
| SM-05 | Tab state machine + transient confirm view | Existing `riskPanelOpen`/`funnelOn` state in `SessionShareModal.tsx` (lines 323-353) already implements the semantics of spec's `internetEnabled`/`internetConfirmed` — see mapping table below; reuse, do not reinvent |
| SM-06 | One reusable ShareLinkCard | Four ad-hoc rows identified: Read-Only (`SessionSharePanel.tsx:438-480`), Full Access (`:489-525`), Public URL (`:545-591`), Public-write result (`:651-698`) — all share the same `session-share-panel__link-row` + `CodeDisplay` shape |
| SM-07 | Colorblind-safe + keyboard-operable | No existing On/Off text label today (verified — toggles show only the row label, not state text); no existing `--danger`/`--danger-line` tokens (verified — must map to `--hub-destructive`); `.share-seg` a11y pattern is net-new |
| SM-08 | Modal width bump + preserved behavior + tests | Actual HEAD line numbers differ from spec's stale refs — verified below; capability/RPC call sites are 100% inside `SessionShareModal.tsx` handlers, none inside `SessionSharePanel.tsx`, so tab extraction cannot touch RPC logic if handlers stay in the shell |
</phase_requirements>

## Summary

This is a pure frontend refactor of three already-shipped, well-tested components (`SessionShareModal.tsx`, `SessionSharePanel.tsx`, `FunnelRiskPanel.tsx`) plus a CSS token migration. No new packages, no backend changes, no new RPC calls. The codebase is smaller and simpler than the design spec's code sketches assume in two important ways: (1) the spec's abstract `internetEnabled`/`internetConfirmed` state pair is **already implemented** today as `riskPanelOpen`/`funnelOn` in `SessionShareModal.tsx` — the planner should rename/reuse these, not add parallel state; and (2) the design tokens named in the spec (`--accent`, `--danger`, `--danger-line`, `--modal-border`, `--btn-bg`, `--m-text`, `--m-dim`) **do not exist** in `style.css` — the app's actual token vocabulary is `--hub-*` (e.g. `--hub-accent`, `--hub-destructive`, `--hub-border`, `--hub-text-primary/secondary/muted`), defined once for dark theme (`:root`, ~line 4642) and once for light theme (`[data-ui-theme="light"]`, ~line 4723). Any new CSS must extend the `--hub-*` vocabulary in both blocks, not invent new names.

All four ad-hoc link rows share the same underlying markup (`.session-share-panel__link-row` + `<CodeDisplay>` + a `<p className="session-share-panel__scope">`), confirming `ShareLinkCard` is a straightforward extraction. The public-write gate (`hub-funnel-write-gate` block, `SessionSharePanel.tsx:603-724`) is already a self-contained block with its own local state (`gateExpirySeconds`, QR toggles, countdown) — it can move into `InternetFullAccessTab` as one unit. `HoldToConfirmButton` and `CodeDisplay` are private (non-exported) function components defined inside `SessionSharePanel.tsx` (lines 29 and 114) — they must either be exported from that file or extracted to their own module if `InternetFullAccessTab`/`ShareLinkCard` become separate files, since D-09 requires preserving them "as-is."

**Primary recommendation:** Do the refactor in three passes — (1) extract `ShareLinkCard` + hoist `HoldToConfirmButton`/`CodeDisplay` to exported/shared status with zero behavior change (regression-safe scaffolding), (2) build `ShareSegmentedControl` + wire the tab state machine into `SessionShareModal.tsx` reusing existing `riskPanelOpen`/`funnelOn`/`writeGate*` state, (3) split `SessionSharePanel.tsx` into `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab` and delete the old single-column JSX. Update tests in lockstep with each pass rather than as a final pass, since the existing 37+ SessionShareModal tests and panel tests already pass on HEAD and are the regression net.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Tab selection / active-tab state | Browser/Client (React state in `SessionShareModal.tsx`) | — | Pure UI state, no server round-trip; matches existing pattern where `riskPanelOpen`/`funnelOn` already live in the shell |
| Internet-risk confirm gesture | Browser/Client | API/Backend (`SetSessionFunnel` RPC) | The transient confirm view is presentational (mirrors current `FunnelRiskPanel`); the actual commit RPC call stays in `SessionShareModal.tsx`'s `handleFunnelEnable`, unchanged |
| Public-write hold-to-confirm gate | Browser/Client (`HoldToConfirmButton` timer) | API/Backend (`SetSessionFunnelWrite` RPC) | Timer logic is client-only; RPC call (`handleGateConfirm`) stays in the shell — do not move into a tab component that can't reach Wails bindings without prop drilling, which is fine since props already flow this way |
| Capability URLs / join codes / QR generation | API/Backend (`IssueCapabilities`, `GetCapabilityQRCode`) | Browser/Client (rendering) | Unchanged — `cachedShare` state and `IssueCapabilities` calls stay exactly where they are in `SessionShareModal.tsx`; only the rendering of the resulting URLs moves into tab components |
| CSS design tokens | Browser/Client (`style.css` custom properties) | — | Must extend the existing `--hub-*` `:root` + `[data-ui-theme="light"]` blocks, not introduce a parallel token namespace |
| Funnel teardown / TTL / capability model | API/Backend (Go daemon, `internal/daemon`, `internal/capability`) | — | Explicitly OUT OF SCOPE — no changes this phase |

## Current State Map (code-grounded, HEAD `40331fa2`)

### `SessionShareModal.tsx` (733 lines) — state ownership

| State variable | Line | Purpose | Maps to spec concept |
|---|---|---|---|
| `shareEnabled` | 146 | "Share the session" toggle, seeded from `session.webEnabled` | `shareEnabled` (same name) |
| `browseEnabled` | 151 | "Enable remote file browsing" toggle | not tab-related; stays in control strip |
| `cachedShare` | 153 | `{readURL, writeURL, readCode, writeCode}` from `IssueCapabilities` | feeds `TailnetTab` |
| `funnelOn` | 323 | server-truth: internet share actually active | **≈ `internetConfirmed`** (true only after `SetSessionFunnel(id, true, …)` resolves) |
| `riskPanelOpen` | 324 | risk panel is open awaiting user decision (not yet committed) | **the "confirm-view-showing" condition** — `riskPanelOpen && !funnelOn` is exactly spec's `internetEnabled && !internetConfirmed` |
| `expirySeconds` | 325 | selected auto-expire preset for the risk panel | unchanged, passed to `FunnelRiskPanel` |
| `warmingUp` / `warmupTimedOut` | 331-332 | TLS warm-up UX between confirm and `funnelActive` flip | new: must be handled inside `InternetReadOnlyTab`/confirm-view transition |
| `funnelUrl` / `publicReadCode` | 333/336 | the public Internet·Read-only URL + reusable code | feeds `InternetReadOnlyTab` |
| `writeGateUrl/Code/ExpiresAt/Used` | 450-453 | public-write armed state | **= spec's `publicWriteArmed`** (armed when `writeGateUrl \|\| writeGateCode` truthy) — feeds `InternetFullAccessTab` |
| `funnelDisabled` | 340 | `webServerMode !== 'tailscale'` fail-closed gate | must disable/hide BOTH Internet tabs, not just the toggle |

**Key finding — the toggle checkbox for "Enable internet sharing" is currently `checked={funnelOn || riskPanelOpen}` (line 670).** This single boolean IS the spec's `internetEnabled`. Concretely:
- `internetEnabled` (spec) = `funnelOn || riskPanelOpen` (current code)
- `internetConfirmed` (spec) = `funnelOn` (current code)
- Confirm-view-showing condition = `(funnelOn || riskPanelOpen) && !funnelOn` = `riskPanelOpen && !funnelOn`, which simplifies to just `riskPanelOpen` (since `riskPanelOpen` is only ever set true while `funnelOn` is false — `handleFunnelToggle` at line 344 returns early `if (funnelOn) return` before it would ever set `riskPanelOpen` while already on). **The planner can gate the transient confirm view on the existing `riskPanelOpen` flag directly — no new state needed.**

**Control strip block to preserve verbatim at top of `.hub-share-modal__body`:** lines 547-648 (shell warning banner, Share toggle, LAN password, HomeDirWriteWarning, Browse toggle, Disconnect-viewers button). This is SM-01's "never reflows" strip.

**Currently misplaced (must move per D-05):** the `FunnelRiskPanel` inline mount at lines 691-700 sits BETWEEN the toggle strip and where `SessionSharePanel` mounts (line 707-728) — this is the exact "injects a block that pushes content down" defect named in the design spec's Motivation section. Confirmed structurally: `funnelError` (line 701-703) also currently renders in this same interstitial position and must move with it (it belongs with the transient confirm view / wherever `SetSessionFunnel` failures surface — likely inside the repurposed confirm view, since D-05 doesn't mention a separate error slot).

**`onOpenHelp` cross-link (FUI-06):** `FunnelRiskPanel`'s `onOpenHelp` prop is wired to `handleOpenHelp` (line 497-501), which calls `handleClose()` then `onOpenHelp?.()`. This must be preserved unchanged when `FunnelRiskPanel` becomes the transient confirm view — it is a modal-level concern (closes the whole modal), not tab-scoped.

**Props passed down to `SessionSharePanel` today (lines 707-727):** `sessionId, readURL, writeURL, readCode, writeCode, browseEnabled, funnelActive(=session.funnelActive, NOT funnelOn — see landmine below), funnelUrl, warmingUp, warmupTimedOut, onDisableFunnel, publicReadCode, onGateConfirm, writeGateUrl, writeGateCode, writeGateExpiresAt, writeGateUsed, onDisableGateWrite`. All 17 must be redistributed across `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab` — none can be dropped without checking whether a test asserts on it.

### `SessionSharePanel.tsx` (727 lines) — block boundaries for tab extraction

| Block | Lines | Becomes |
|---|---|---|
| `HoldToConfirmButton` (private fn component) | 29-109 | Preserve as-is; must be exported (or moved to a shared module) since D-09 requires it usable from `InternetFullAccessTab` |
| `CodeDisplay` (private fn component) | 114-163 | Preserve as-is; same export requirement — used by Tailnet AND both Internet tabs, and by `ShareLinkCard` internally per D-06 |
| Read-Only link row + scope text | 438-480 | → `ShareLinkCard` instance inside `TailnetTab` |
| Full Access link row + scope text | 482-525 | → `ShareLinkCard` instance inside `TailnetTab` |
| `qrError` shared error text | 527 | Currently ONE shared QR-error slot for both read+write tailnet rows — if each becomes an independent `ShareLinkCard`, this must become per-card local state (each card owns its own QR fetch/error), a behavior change implied but not explicitly stated by D-06; flag as an assumption |
| Internet (public) section (`hub-share-internet-section`) | 531-601 | → `InternetReadOnlyTab` body (`ShareLinkCard` for the public URL + `CodeDisplay` for `publicReadCode` + "Disable internet share" button) |
| Danger / public-write gate (`hub-funnel-write-gate`) | 603-724 | → `InternetFullAccessTab` body verbatim, including `HoldToConfirmButton` |

**`funnelEngaged` (line 310) = `warmingUp || warmupTimedOut || funnelActive`** currently gates BOTH the Internet section AND the Danger section identically. Under the new tab model this becomes: the Internet·Read-only tab body is only reachable once `internetConfirmed` (spec) = `funnelOn` (current), and `funnelEngaged` still separately controls whether to show "Starting up…"/"timed out"/live-URL sub-states inside that tab. The Full-access tab's availability gate is IDENTICAL (`internetConfirmed`), but its internal warm-up gating on the `HoldToConfirmButton`'s `disabled` prop is `!(funnelActive && !warmingUp)` (line 645) — this nuance must be preserved: the tab is reachable once confirmed, but the hold button itself stays disabled until the connection is actually warm.

**Local state that must move with its block:** `gateExpirySeconds`, `gateCopied`, `showGateQR`, `gateQRb64`, `gateQRError`, `gateNowSec`, `disableGateBtnRef`, `hadGateResultRef` (all Danger-section-local, lines 282-306) move into `InternetFullAccessTab`. `funnelCopied`, `showFunnelQR`, `funnelQRb64`, `funnelQRError` (lines 276-279) move into `InternetReadOnlyTab`. `readCopied/writeCopied/showReadQR/showWriteQR/readQRb64/writeQRb64` (lines 268-273) move into `TailnetTab` (or into per-card state if `ShareLinkCard` owns its own QR/copy state — recommended, see below).

**Focus management landmine:** `disableGateBtnRef`/`hadGateResultRef` (lines 298-306) move focus to the "Disable public write" button the instant the gate completes. This `useEffect` must move intact into `InternetFullAccessTab` — if the tab unmounts/remounts on tab-switch, verify this effect still fires correctly when the tab becomes active with an already-armed gate (e.g., user switches away then back).

### `FunnelRiskPanel.tsx` (104 lines) — props for repurposing as transient confirm view

Current props: `open, expirySeconds, onExpiryChange, onEnable, onCancel, onOpenHelp` — ALL of these are still needed as the transient confirm view (nothing to remove). The only structural change: today it's rendered inline (`open` prop drives a CSS max-height expand via `.hub-funnel-risk-panel--open`, `style.css:7318`) sitting BETWEEN the toggle strip and the panel. Per D-05 it must instead be rendered as a REPLACEMENT for the segmented-control region (conditionally, instead of `<ShareSegmentedControl>` + tab body). **Recommendation:** keep the component's internal CSS class names for now (`.hub-funnel-risk-panel`, no rename needed structurally) — moving its position in the JSX tree costs nothing; renaming its CSS class is optional cosmetic cleanup, not required by any locked decision. `FunnelRiskPanel.test.tsx` tests the component in isolation and does not test its parent-mount position, so this test file needs no structural changes, only possibly a new test asserting it is NOT nested alongside link content (optional).

### CSS — verified line numbers (HEAD `40331fa2`, style.css is 7611 lines)

The design spec's line numbers are stale — actual current locations:

| Selector | Spec's stated line | Actual HEAD line |
|---|---|---|
| `.hub-share-modal` (width/max-height) | `:6318` | **`:6391`** (width `min(480px,...)` on line 6393, `max-height` on line 6400) |
| `.hub-share-modal__header` | `:6339` | **`:6412`** |
| `.hub-share-modal__body` | `:6361` | **`:6434`** |
| Responsive/reduced-motion override | `:6405` | **`:6478`** (`@media (prefers-reduced-motion: reduce) { .hub-share-modal { ... } }`) — NOTE: this is NOT a viewport-width responsive override, it's a motion-preference override; there is no separate narrow-viewport media query for `.hub-share-modal` today. The width already clamps via `calc(100vw - 48px)` inline in the base rule (no breakpoint needed) — verify this still holds at 520px on a narrow window (e.g. 400px viewport) since `min(520px, calc(100vw-48px))` degrades gracefully by CSS `min()` alone. |

**Design tokens — CRITICAL MISMATCH, verified:** none of the token names in CONTEXT.md's "Specific Ideas" section (`--accent`, `--link`, `--danger`, `--danger-line`, `--warn`, `--chip-bg`, `--chip-border`, `--modal-border`, `--mono`) exist anywhere in `style.css` (grep across the whole file returned zero matches). The actual existing token vocabulary, defined in `:root` (dark, ~line 4642-4712) and `[data-ui-theme="light"]` (light, ~line 4723+):

| Spec token name | Actual equivalent | Dark value | Light value |
|---|---|---|---|
| `--accent` | `--hub-accent` | `#7aa2f7` | `#3d6fe8` |
| `--link` | *(none — currently hardcoded `#7aa2f7` inline in `.session-share-panel__url`, style.css:3058)* | — | — |
| `--danger` | `--hub-destructive` | `#f7768e` | `#c0394f` |
| `--danger-line` | *(none exists — `.hub-funnel-write-gate` currently uses `border-top: 2px solid var(--hub-destructive)` directly, no separate "line" shade)* | — | — |
| `--warn` | `--hub-warning` | `#fbbf24` | `#b45309` |
| `--modal-border` | `--hub-border` | `#41454f` | `#d1d1db` |
| `--btn-bg` | *(none — no generic button-background token; `.share-seg.is-active` background needs a new decision, e.g. reuse `--hub-sidebar-item-active-bg` pattern or a new dedicated token)* | — | — |
| `--m-text` | `--hub-text-primary` | `#f4f5f8` | `#1a1b26` |
| `--m-dim` | `--hub-text-muted` | `#9398a8` | `#5c5d80` |
| `--mono` | `--hub-font-mono` | `'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace` | (same) |
| `--chip-bg`/`--chip-border` | *(none — new tokens needed if adopted, or reuse existing `rgba(255,255,255,...)` literal pattern already used elsewhere, e.g. `--hub-sidebar-item-hover-bg: rgba(122,162,247,0.07)`)* | — | — |

**Planner action required:** any new CSS for `.share-segbar`/`.share-seg*`/`.share-linkcard*` MUST use `--hub-*` names (either existing ones per the mapping above, or newly-added `--hub-*` tokens defined in BOTH the dark `:root` block and the light `[data-ui-theme="light"]` block — this codebase is dual-themed and every existing color token has a light-theme override). Do not introduce the spec's literal token names as CSS variables; they don't exist and there is no light-theme value to pair with the dark hex the spec lists.

**Colorblind precedent already in the codebase:** `--hub-fullaccess-badge-text: #f7768e` (dark) / equivalent light value already exists with an explicit code comment: `"COLORBLIND-SAFE: full-access badge ... reinforcement only; LockOpenIcon shape + 'FULL ACCESS' text + notched badge geometry carry the state"` (style.css ~4713-4718). This is a directly reusable precedent for the `.share-seg.is-danger` treatment — same hue family, same "glyph+text carries the state, color reinforces" philosophy already established in this codebase for the exact same Full-Access concept. Reuse `--hub-destructive`/`--hub-fullaccess-badge-text`, do not invent a new red.

**Hardcoded hex literals found that violate the "extend style.css, don't hardcode" principle (existing debt, worth cleaning up alongside this refactor since D-09 already calls for migrating inline `style={}`):**
- `.session-share-panel__url { color: #7aa2f7; }` (style.css:3058) — should become `var(--hub-accent)` (same value, verified numerically identical).
- `.session-share-panel__label { color: #a9b1d6; }` (style.css:3050) — no exact `--hub-*` match found; either introduce a new token or leave as-is (out of scope unless the card extraction touches this rule anyway).
- Inline `style={{ margin: '2px 0 10px', fontSize: 12, color: '#9aa5ce', ... }}` on scope-text `<p>` elements (`SessionSharePanel.tsx:467, 484`) — exactly the "orphaned scope descriptions" + inline-style defect named in the design spec's Motivation section; migrate to a `.share-linkcard__desc` class.
- Inline `style={{ cursor: 'pointer' }}` / `style={{ opacity: 0.6, pointerEvents: 'none' }}` on toggle rows in `SessionShareModal.tsx` (lines 569, 607, 611, 658, 662, 682) — D-09 says migrate inline `style={}` layout to classes; these are candidates but are NOT called out by name in any locked decision, so treat as "nice to have, don't block the phase on it" unless time permits.

### No existing tablist / roving-tabindex precedent

Grepped the entire `frontend/src` tree for `role="tablist"` / `role="tab"` — the only hit is a NEGATIVE test assertion in `SettingsTab.test.tsx` (`expect(raw).not.toContain('role="tablist"')`, confirming Settings deliberately avoids a tab pattern). **`ShareSegmentedControl` is genuinely new UI infrastructure for this codebase** — there is no existing roving-tabindex helper/hook to reuse. Build directly from the design spec's code sketch (arrow-key handling is NOT shown in the spec's sketch and must be added — the sketch only shows `tabIndex={active===t.id?0:-1}` static roving-index wiring, not the keydown handler that moves it. Standard WAI-ARIA APG tablist pattern: `ArrowLeft`/`ArrowRight` move focus + selection between enabled segments, wrapping at the ends, skipping `disabled` segments).

### No existing QR/Copy/Open helper module

`copy`, `openExternal`/`BrowserOpenURL`, `truncate` are NOT centralized — each is reimplemented inline per-row in `SessionSharePanel.tsx` today (`handleCopy`, direct `BrowserOpenURL(...)` calls, no `truncate()` helper exists — URLs are truncated purely via CSS `text-overflow: ellipsis` on `.session-share-panel__url`, not JS truncation). The design spec's code sketch shows a JS `truncate(url)` helper that **does not exist** — either introduce one, or (recommended, matches existing pattern and needs zero new code) keep using CSS ellipsis truncation with the full URL in the `title=` attribute, which is what D-06 actually asks for ("truncated URL (with full URL in `title=`)") and is already how `session-share-panel__url` behaves today.

## Standard Stack

No new libraries are needed for this phase — it is a pure refactor within the existing stack.

### Core (existing, unchanged)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| react / react-dom | ^19.2.4 | Component model | Already the app's framework |
| typescript | ^5.9.3 | Type safety | Already enforced via `tsc && vite build` |
| @heroicons/react | ^2.2.0 | `ExclamationTriangleIcon`, `XMarkIcon` icons already used in `FunnelRiskPanel`/`SessionSharePanel`/`SessionShareModal` | Already the icon library; reuse for any new warning glyph if an icon (vs. plain `⚠` text glyph) is wanted — spec explicitly wants a **glyph in text**, not necessarily an SVG icon, so plain `⚠` character is sufficient and simplest |
| vite | ^8.0.0 | Build | unchanged |
| vitest | ^4.1.0 | Test runner | unchanged |

### Supporting
None — no new runtime dependencies. No testing-library packages present or needed; existing tests use raw `react-dom/client` (`createRoot`) + `flushSync`, a pattern to continue for new/updated tests (do not introduce `@testing-library/react` mid-phase — it would be an unrelated dependency addition and inconsistent with all 4 existing share test files).

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled roving-tabindex on `ShareSegmentedControl` | Radix UI `Tabs` primitive or similar | Adds a new dependency for ~30 lines of arrow-key handling; codebase has zero existing Radix/headless-UI dependency, and every other interactive control in this app (toggles, hold-button) is hand-rolled — stay consistent, do not introduce a UI library for one control |
| CSS `text-overflow: ellipsis` truncation (current) | JS `truncate()` helper (per spec's non-normative sketch) | CSS approach already works, needs zero new code, and correctly handles arbitrary container widths; JS truncation would need to be re-measured on resize — not worth the complexity for this phase |

**Installation:** None required.

## Package Legitimacy Audit

Not applicable — this phase installs no external packages. All work is confined to existing first-party `frontend/src/components/**` files and `frontend/src/style.css`.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ SessionShareModal.tsx (shell — owns ALL RPC calls + state)      │
│                                                                   │
│  Fixed control strip (unchanged, D-01):                         │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Share toggle → ToggleWebServing()                         │  │
│  │ Browse toggle → SetSessionBrowse() + IssueCapabilities()  │  │
│  │ Internet toggle → sets riskPanelOpen (NO RPC yet)         │  │
│  │ [Disconnect viewers] → DisconnectViewers()                │  │
│  └───────────────────────────────────────────────────────────┘  │
│                            │                                     │
│                            ▼  (swappable region — D-05)          │
│        ┌──────────────────────────────────────┐                 │
│        │ if !shareEnabled → empty hint          │                │
│        │ elif riskPanelOpen && !funnelOn         │                │
│        │   → <FunnelRiskPanel> (transient       │                │
│        │      confirm view)                      │                │
│        │      onEnable → SetSessionFunnel(true)  │                │
│        │      onCancel → close, no RPC           │                │
│        │ else → <ShareSegmentedControl>           │                │
│        │        + active tab body:                │                │
│        │   ┌─────────────┬────────────┬────────┐  │                │
│        │   │ TailnetTab  │ InternetRO │InternetFA│ │                │
│        │   │ (always     │ Tab        │ Tab      │ │                │
│        │   │  enabled    │ (enabled   │ (enabled │ │                │
│        │   │  when       │  iff       │  iff     │ │                │
│        │   │  shareEnabled)│ funnelOn) │ funnelOn)│ │                │
│        │   └─────────────┴────────────┴────────┘  │                │
│        └──────────────────────────────────────────┘                │
│                            │                                     │
│                            ▼ props: cachedShare, funnelUrl,      │
│                              writeGate*, onGateConfirm, etc.     │
│              (all RPC results flow DOWN as props;                │
│               tabs never call Wails bindings directly)           │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
frontend/src/components/
├── Hub/
│   ├── SessionShareModal.tsx        # shell — MODIFIED (tab state, dispatch)
│   ├── FunnelRiskPanel.tsx          # repurposed transient confirm view — position changes, props don't
│   └── ShareSegmentedControl.tsx    # NEW — role=tablist component
├── SessionSharePanel.tsx            # gutted to a thin re-export or removed — see discretion note
├── SessionShare/                    # NEW subfolder (discretion — or flatten into components/)
│   ├── TailnetTab.tsx               # NEW — extracted from SessionSharePanel.tsx
│   ├── InternetReadOnlyTab.tsx      # NEW — extracted from SessionSharePanel.tsx
│   ├── InternetFullAccessTab.tsx    # NEW — extracted from SessionSharePanel.tsx (hosts HoldToConfirmButton)
│   ├── ShareLinkCard.tsx            # NEW — extracted reusable row
│   └── shared.tsx                   # NEW — hoisted CodeDisplay (+ HoldToConfirmButton if not co-located with InternetFullAccessTab)
```
This is a discretionary layout suggestion (Claude's Discretion per CONTEXT.md) — the only hard constraint is "follow existing `components/` + `components/Hub/` conventions." `SessionSharePanel.tsx` as a named export/file may or may not survive; if all its content moves into three tab files + `ShareLinkCard` + `shared.tsx`, the planner should decide whether to keep a thin `SessionSharePanel.tsx` (e.g. as a barrel or deleted entirely with imports repointed) — **flag this file's fate explicitly in the plan**, since 3 test files (`SessionSharePanel.test.tsx` + indirectly `SessionShareModal.test.tsx`/`.disconnect.test.tsx`) import it by path today.

### Pattern 1: Roving tabindex tablist
**What:** `role="tablist"` container, each segment `role="tab"` with `aria-selected` + `tabIndex` (0 for active, -1 for others), `aria-disabled` for gated segments, arrow-key navigation.
**When to use:** `ShareSegmentedControl` (SM-03/SM-07).
**Example (adapted from design spec's code sketch, WAI-ARIA APG roving-tabindex added):**
```tsx
// Source: design spec 173-DESIGN-129.md (non-normative sketch) + WAI-ARIA Authoring
// Practices Guide tablist pattern (arrow-key nav not in the spec's sketch — added here)
function ShareSegmentedControl({ tabs, active, onSelect }: Props) {
  const enabled = tabs.filter(t => !t.disabled)
  function move(delta: 1 | -1) {
    const idx = enabled.findIndex(t => t.id === active)
    const next = enabled[(idx + delta + enabled.length) % enabled.length]
    onSelect(next.id)
  }
  return (
    <div className="share-segbar" role="tablist" aria-label="Share access tier">
      {tabs.map(t => (
        <button
          key={t.id} role="tab" aria-selected={active === t.id}
          aria-disabled={t.disabled} disabled={t.disabled}
          tabIndex={active === t.id ? 0 : -1}
          className={cx('share-seg', active === t.id && 'is-active', t.danger && 'is-danger')}
          onClick={() => !t.disabled && onSelect(t.id)}
          onKeyDown={(e) => {
            if (e.key === 'ArrowRight') { e.preventDefault(); move(1) }
            if (e.key === 'ArrowLeft')  { e.preventDefault(); move(-1) }
          }}
        >
          <span className="share-seg__main">{t.main}</span>
          <span className="share-seg__sub">{t.disabled ? 'N/A' : t.sub}</span>
        </button>
      ))}
    </div>
  )
}
```

### Anti-Patterns to Avoid
- **Re-deriving `internetEnabled`/`internetConfirmed` as new state:** they already exist as `riskPanelOpen`/`funnelOn` — adding parallel booleans risks the two falling out of sync (a real bug class, not hypothetical, given this codebase's history of careful single-source-of-truth comments like "Single warned authority from App.tsx — do NOT fork into local state" at line 251-252).
- **Giving each `ShareLinkCard` its own independent QR-fetch cache keyed only by URL without considering the existing shared-error-slot removal:** today `qrError` is ONE shared string for both tailnet rows (line 274/527); decomposing into per-card state changes error-display behavior (two cards could show independent errors simultaneously) — this is arguably an improvement but is a behavior change beyond pure layout, worth calling out to the user/tests explicitly.
- **Hardcoding new hex colors for `.share-seg`/`.share-linkcard`:** must extend `--hub-*` in both theme blocks (dark AND light) — a single-theme addition will look broken in light mode (`[data-ui-theme="light"]` is a real, tested surface — HUB-04 requirement).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Copy-to-clipboard | New clipboard helper | Existing `ClipboardSetText` (Wails runtime binding), already wrapped consistently as `handleCopy(url, setter)` in `SessionSharePanel.tsx:353` | Already battle-tested across 4 call sites; just reuse the function when hoisting into `ShareLinkCard` |
| QR code generation | Client-side QR library | Existing `GetCapabilityQRCode` Go binding (server-generated base64 PNG) | Already the pattern for all 3 existing QR rows; no reason to duplicate client-side |
| Hold-to-confirm timing gesture | New timer/progress component | Existing `HoldToConfirmButton` (`SessionSharePanel.tsx:29-109`) | D-09 explicitly says preserve as-is; it already handles pointer + keyboard, `prefers-reduced-motion` is only PARTIALLY implemented today — see Common Pitfalls below |

**Key insight:** Every piece of "plumbing" this phase touches (copy, QR, hold gesture, join-code display) already has a single well-tested implementation in this codebase. The entire phase is about **rearranging where these pieces render**, not rebuilding any of them.

## Common Pitfalls

### Pitfall 1: `HoldToConfirmButton` does not yet implement the `prefers-reduced-motion` fallback that D-07/SM-07 requires
**What goes wrong:** D-07 states "Hold-to-confirm keeps a text label and works under `prefers-reduced-motion` (falls back to a plain confirm rather than a timed fill)." Reading `HoldToConfirmButton` (`SessionSharePanel.tsx:29-109`) today: the `progress` fill bar has NO `prefers-reduced-motion` branch in the component itself — style.css DOES have one for the fill's CSS transition (`.hub-funnel-write-gate__hold-fill { transition: none; }` under `@media (prefers-reduced-motion: reduce)`, style.css:7536), but the component still requires a full 3000ms hold even with reduced motion — there is no "plain confirm" (e.g. click-to-open-a-real-confirm-dialog) fallback path.
**Why it happens:** This requirement (a full behavioral fallback, not just a CSS transition removal) appears to be NEW in this phase's spec — it wasn't part of Phase 171's original hold-gate implementation.
**How to avoid:** This needs an explicit plan task: add a `prefersReducedMotion` check inside `HoldToConfirmButton` that renders a plain `<button onClick={onConfirm}>Confirm public write access</button>` (single click, no timer) when reduced motion is active, instead of just softening the CSS transition. Confirm with the user/spec that this is in-scope — D-09 says "PRESERVE as-is: HoldToConfirmButton" which is in tension with D-07 requiring new reduced-motion behavior inside it. **Flag this contradiction explicitly to the planner** — likely resolution: D-09's "preserve as-is" refers to the 3s-hold SAFETY GATE BEHAVIOR (never changing the underlying `SetSessionFunnelWrite` semantics/duration), not to zero code changes in the component; the reduced-motion fallback is additive, not a change to the gate's timing contract.

### Pitfall 2: Shared QR-error state assumption breaks silently if cards go independent
**What goes wrong:** if `ShareLinkCard` is built with fully independent internal QR state (recommended for cleanliness), the existing test assertions that look for a SHARED `qrError`/`session-share-panel__error` element by class name across both read+write rows may need updating — check `SessionSharePanel.test.tsx` QR-related assertions before assuming the current shared-error DOM structure.
**Why it happens:** `ShareLinkCard`'s natural encapsulation boundary (each card owns Copy/Open/QR) conflicts with the current single shared error slot.
**Warning signs:** A test asserting `.session-share-panel__error` exists exactly once, or asserting error text appears regardless of which row's QR button was clicked.

### Pitfall 3: `funnelActive` prop vs `funnelOn` state — do not conflate
**What goes wrong:** `SessionShareModal.tsx` passes `funnelActive={session.funnelActive}` (server-truth from the Hub poll) to `SessionSharePanel`, which is DIFFERENT from the shell's own `funnelOn` local state (set optimistically the instant `SetSessionFunnel` resolves, line 365, BEFORE the poll confirms). `SessionSharePanel.tsx`'s internal `funnelEngaged` (line 310) is derived from the PROP `funnelActive`, not from `funnelOn`. When restructuring into tabs, the "is this tab enabled" gate (spec's `internetConfirmed`) should key off `funnelOn` (immediate, shell-local, matches "on successful internet confirm the active tab defaults to Internet·Read-only" firing right after the CTA click) — but the "show warming/live sub-state inside the tab" logic should still key off the `funnelActive`/`warmingUp`/`warmupTimedOut` trio exactly as today. Mixing these up will make the tab-availability transition either premature (before the daemon confirms) or laggy (waiting for the poll when the UI should react instantly to the user's own click).
**Warning signs:** Tests `FUI-05: warm-up → live URL` (SessionShareModal.test.tsx:595-627) exercise exactly this distinction — they must still pass unchanged since they don't touch tab UI, only content within whatever renders the Internet·Read-only body.

### Pitfall 4: The "reset active tab to Tailnet on internet-disable" requirement needs a new effect
**What goes wrong:** D-05 says "Disabling internet sharing resets the active tab to Tailnet and disables both Internet tabs." No existing code does this today (there's no `tab` state yet). This needs a NEW `useEffect` in `SessionShareModal.tsx`: `useEffect(() => { if (!funnelOn) setTab('tailnet') }, [funnelOn])` or equivalent — easy to add, easy to forget, and there's no existing test coverage to catch its absence (it's entirely new behavior). Add an explicit test for it.

### Pitfall 5: Line-number drift will bite naive plan tasks
**What goes wrong:** Every line number in `173-CONTEXT.md`/`173-DESIGN-129.md` for `style.css` is off by 60-73 lines from actual HEAD (see CSS table above); `SessionShareModal.tsx`/`SessionSharePanel.tsx` internal line refs in the spec (`:541`, `:567`, `:605`, `:654`, `:691`, `:707` for the modal; `:438`, `:482`, `:531`, `:609`, `:79`, `:131` for the panel) were independently re-verified in this research and found to be reasonably close/exact matches for the CURRENT HEAD used in this research session (`40331fa2`) — but the codebase moves fast (multiple phases landed since #129 was written). **The planner must re-grep every line reference at plan-authoring time, not trust either the spec or this document's numbers blindly** — treat all line numbers everywhere as "as of this research/spec's HEAD," subject to drift from any commits between now and execution.

## Code Examples

### Existing hold-to-confirm pattern (preserve gesture, verified working)
```tsx
// Source: frontend/src/components/SessionSharePanel.tsx:53-68 (verified HEAD 40331fa2)
function startHold(): void {
  if (disabled || holding) return
  setHolding(true)
  setProgress(0)
  startRef.current = Date.now()
  intervalRef.current = setInterval(() => {
    const elapsed = Date.now() - startRef.current
    setProgress(Math.min(100, (elapsed / HOLD_DURATION_MS) * 100))
  }, HOLD_TICK_MS)
  timeoutRef.current = setTimeout(() => {
    clearTimers()
    setProgress(100)
    setHolding(false)
    onConfirm()
  }, HOLD_DURATION_MS)
}
```

### Existing join-code-exchange URL derivation (reuse for ShareLinkCard, do not reinvent)
```tsx
// Source: frontend/src/components/SessionSharePanel.tsx:387-393 (verified HEAD 40331fa2)
// QR encodes the join-code exchange URL, NOT the capability token —
// a photographed QR is worthless after single-use exchange.
function joinURLFor(capURL: string, code: string): string {
  const u = new URL(capURL)
  return `${u.protocol}//${u.host}/join?code=${code}`
}
```

### Existing toggle markup (D-07 will need an "On/Off/N/A" text span added alongside this)
```tsx
// Source: frontend/src/components/Hub/SessionShareModal.tsx:567-584 (verified HEAD 40331fa2)
// NOTE: no state-text span exists today — only the row label ("Share the session").
// D-07 requires adding something like <span className="...__state">{shareEnabled ? 'On' : 'Off'}</span>
<label className={`settings-panel__toggle-row${shareEnabled ? ' settings-panel__toggle-row--checked' : ''}`}>
  <input type="checkbox" role="switch" aria-checked={shareEnabled}
         aria-label="Share the session" checked={shareEnabled}
         onChange={() => void handleShareToggle()} />
  <span className="settings-panel__toggle-track"><span className="settings-panel__toggle-thumb" /></span>
  <span className="settings-panel__toggle-label">Share the session</span>
  {/* NEW: On/Off text label goes here per D-07/SM-07 */}
</label>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Single-column stacking panel with inline-injected risk panel | Fixed control strip + 3-tab segmented panel with transient confirm view | This phase (173) | Eliminates the "disheveled/reflow" defect named in the design spec; no capability-model change |

**Deprecated/outdated:** N/A — no prior art in this codebase to deprecate beyond the single-column layout itself.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Per-card independent QR-error state (vs. today's single shared `qrError`) is an acceptable, even desirable, side effect of `ShareLinkCard` extraction | Architecture Patterns / Anti-Patterns, Pitfall 2 | If the user/plan-checker wants byte-identical error-display behavior, this needs an explicit decision — low risk (UX improvement, not regression) but IS a behavior change beyond pure layout that D-08 says must otherwise be "unchanged" |
| A2 | The `prefers-reduced-motion` "plain confirm" fallback for `HoldToConfirmButton` (D-07) is intended as an ADDITIVE change to the component's internals, not blocked by D-09's "preserve as-is" | Common Pitfalls, Pitfall 1 | If wrong (D-09 takes precedence and no reduced-motion behavioral fallback is wanted), this becomes a no-op/descope item — low risk either way, but the plan must make an explicit call rather than silently picking one interpretation |
| A3 | `funnelError` (SessionShareModal.tsx:701-703) should render inside/alongside wherever the repurposed `FunnelRiskPanel` transient confirm view lives, since it currently sits in that same interstitial region and reports `SetSessionFunnel` failures from the confirm gesture | Current State Map, "Currently misplaced" note | If wrong, error text could end up orphaned or duplicated; low risk, easy to spot in a UAT pass |
| A4 | New CSS tokens needed to fill the `--hub-*` gaps identified (e.g. no `--danger-line`, no `--btn-bg`, no `--chip-bg/border`) should be ADDED as new `--hub-*`-prefixed custom properties in both theme blocks, rather than introducing a differently-prefixed token family | CSS section | If wrong (e.g. team wants a scoped-to-this-modal token prefix instead), only a naming/location change — no functional risk |

**If this table is empty:** N/A — see above; all four assumptions are low-risk UX/behavior nuances, not architectural risks.

## Open Questions

1. **Does the confirm-view's `funnelError` need to persist visibly if the user then navigates away/back before retrying?**
   - What we know: today it's a simple `useState<string|null>` in the shell, cleared only on `handleFunnelCancel`/successful enable.
   - What's unclear: whether switching tabs (once past confirm) should clear a stale error from a PRIOR failed confirm attempt.
   - Recommendation: keep current behavior (clear on cancel/success only) — this is an edge case unlikely to be hit in practice (the confirm view isn't reachable once past confirm), low priority for explicit plan coverage.

2. **Should `SessionSharePanel.tsx` be deleted or kept as a re-export shim?**
   - What we know: 3 test files import from it by path (`SessionSharePanel.test.tsx` directly; `SessionShareModal.test.tsx`/`.disconnect.test.tsx` import `SessionShareModal` which internally imports `SessionSharePanel` — those tests don't import `SessionSharePanel` directly, only the modal).
   - What's unclear: whether the planner wants the file deleted entirely (with `SessionSharePanel.test.tsx` renamed/split across new test files for `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab`) or kept as a thin composed wrapper for backward-compat/minimal-diff.
   - Recommendation: delete `SessionSharePanel.tsx` and its export; the component is being reorganized, not layered — carrying a dead re-export forward adds confusion. Split `SessionSharePanel.test.tsx` into per-tab test files matching the new component boundaries (update TESTING.md Suite Manifest count + Traceability map per the standing convention in `agenthub/CLAUDE.md`).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Frontend build/test | ✓ | v24.14.1 | — |
| pnpm | Package manager (project convention: pnpm preferred) | ✓ | 9.15.9 | — |
| vitest (`pnpm test`) | Unit/component tests | ✓ | 4.1.0 (confirmed: `SessionShareModal.test.tsx` — 37/37 passed in isolation) | — |
| `tsc && vite build` (`pnpm build`) | Full build gate (per project memory: vitest alone tolerates TS errors that this gate rejects) | ✓ (tsc ^5.9.3, vite ^8.0.0 present) | — | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest ^4.1.0, `environment: jsdom`, `react-dom/client` + `flushSync` pattern (no `@testing-library/react`) |
| Config file | `frontend/vitest.config.ts` (via `defineConfig` from `vitest/config`, plugin `@vitejs/plugin-react`) |
| Quick run command | `cd frontend && pnpm test -- --run <TestFile>.test.tsx` (verified: `SessionShareModal.test.tsx` runs in 1.16s) |
| Full suite command | `cd frontend && pnpm test` (142 `*.test.ts/tsx` files per TESTING.md Suite Manifest) — full build gate is `cd frontend && pnpm build` (`tsc && vite build`) which vitest alone does NOT cover |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SM-01 | Toggling a control never reflows content already on screen | unit (structural — assert control-strip DOM position stable across toggle) | `pnpm test -- --run SessionShareModal.test.tsx` | ✅ existing file, ❌ new assertion needed |
| SM-02 | No state causes the whole `.hub-share-modal__body` to scroll (only the tab-body region does) | unit (CSS-class/structural assertion — e.g. assert `.hub-share-modal__body` has no `overflow-y:auto` win, or assert a new inner scroll wrapper class exists) | `pnpm test -- --run SessionShareModal.test.tsx` | ❌ Wave 0 — new test |
| SM-03 | `role="tablist"`/`role="tab"`, `aria-selected`, roving `tabIndex`, arrow-key nav | unit | `pnpm test -- --run ShareSegmentedControl.test.tsx` | ❌ Wave 0 — new file |
| SM-04 | Public-write UI ONLY inside Full-access tab (never elsewhere in the DOM) | unit (assert `hub-funnel-write-gate`/hold-button absent from Tailnet/Internet-RO tab renders) | `pnpm test -- --run InternetFullAccessTab.test.tsx` (or retained `SessionSharePanel.test.tsx` if kept) | ❌ Wave 0 — new/renamed file |
| SM-05 | Tab state machine (empty hint / confirm view / segmented+body) + default-to-Read-only on confirm + reset-to-Tailnet on disable | unit | `pnpm test -- --run SessionShareModal.test.tsx` | ✅ existing file (FUI-01/02 tests already cover confirm gesture — extend, don't replace), ❌ new reset-to-Tailnet assertion (Pitfall 4) |
| SM-06 | `ShareLinkCard` renders title/URL/actions/join-code/description consistently across all 3 usages | unit | `pnpm test -- --run ShareLinkCard.test.tsx` | ❌ Wave 0 — new file |
| SM-07 | On/Off/N-A text labels; `.share-seg.is-danger` class + `aria-selected`; `:focus-visible` (CSS-only, not unit-testable) | unit (attribute/text assertions only, per D-08's "non-hue" mandate) | `pnpm test -- --run SessionShareModal.test.tsx` + `ShareSegmentedControl.test.tsx` | ❌ Wave 0 — extend existing + new file |
| SM-08 | Modal width literal `min(520px, calc(100vw - 48px))`; capabilities/tokens/TTL/teardown/hold-gate behaviorally unchanged | unit (CSS source-string assertion, same pattern as existing "no hardcoded color" gates per TESTING.md) + full existing regression suite must stay green | `pnpm test` (full suite) + `pnpm build` (tsc gate) | ✅ regression net already exists (37+ modal tests, panel tests, disconnect tests, FunnelRiskPanel tests) |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test -- --run <changed-file>.test.tsx`
- **Per wave merge:** `cd frontend && pnpm test && pnpm build` (vitest full suite + the `tsc && vite build` gate — per project memory, vitest alone is insufficient to catch TS errors the build gate rejects)
- **Phase gate:** Full suite + build green before `/gsd-verify-work`; per project memory, also run a live dev-browser UAT pass (component-harness style, per `reference_browser_uat_help_harness.md`) since colorblind-source-level verification and CSS-computed-style checks are called for by user preference, not just DOM assertions.

### Wave 0 Gaps
- [ ] `frontend/src/components/Hub/__tests__/ShareSegmentedControl.test.tsx` (or co-located `__tests__` per existing convention) — covers SM-03/SM-07 tablist contract
- [ ] `frontend/src/components/__tests__/ShareLinkCard.test.tsx` — covers SM-06
- [ ] New/renamed test file(s) for `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab` — covers SM-04 (walled-off assertion is the highest-value new test in this phase: assert public-write markup is ABSENT from the other two tabs' rendered output)
- [ ] Extend `SessionShareModal.test.tsx` — new assertions for: control-strip DOM stability (SM-01), single-scroll-region structural check (SM-02), default-tab-after-confirm = Internet·Read-only (SM-05), reset-to-Tailnet on internet-disable (SM-05, Pitfall 4), On/Off/N-A text labels present (SM-07)
- [ ] Framework install: none — vitest/jsdom already configured

## Security Domain

`security_enforcement` is not set to `false` in `.planning/config.json` (key absent → treat as enabled), so this section is included per protocol, though the phase is explicitly scoped to exclude any capability/token/crypto changes.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Unchanged — capability-token model (Go daemon) is out of scope this phase |
| V3 Session Management | No | Unchanged — TTL/expiry semantics untouched (D-08 explicit) |
| V4 Access Control | No | Unchanged — Funnel teardown / RW-gate authorization logic untouched (D-08 explicit); UI reorg does not alter which RPCs exist or when they're callable, only the layout around calling them |
| V5 Input Validation | Marginal | The expiry `<select>` dropdowns (`FunnelRiskPanel`, public-write gate) are already fixed-enum (no free-text input) — this pattern must be preserved; do not introduce a free-text expiry input as part of any "cleanup" |
| V6 Cryptography | No | No crypto touched — QR/copy/URL generation is unchanged, only re-homed into `ShareLinkCard` |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Public-write flow becoming reachable outside the Full-access tab due to a refactor mistake (e.g. leftover conditional bug exposing `hub-funnel-write-gate` in `InternetReadOnlyTab`) | Elevation of Privilege (UI-level — real backend authorization is unaffected, but a UI leak that makes command-execution discoverable in the "safe" tab would be a serious usability/trust regression even though the backend RPC itself still requires the correct capability) | Explicit negative test (SM-04 in the Requirements→Test Map above): assert the write-gate DOM is ABSENT from `TailnetTab`/`InternetReadOnlyTab` renders, not just present in `InternetFullAccessTab` |
| QR code encoding the raw capability token instead of the join-code exchange URL, if `ShareLinkCard`'s QR logic is rewritten instead of reusing `joinURLFor()` | Information Disclosure | Reuse the existing `joinURLFor(capURL, code)` helper (verified at `SessionSharePanel.tsx:387-393`) verbatim — do not let `ShareLinkCard` re-derive its own QR-target URL logic |

## Sources

### Primary (HIGH confidence — direct code inspection at HEAD `40331fa2`)
- `frontend/src/components/Hub/SessionShareModal.tsx` (733 lines, read in full)
- `frontend/src/components/SessionSharePanel.tsx` (727 lines, read in full)
- `frontend/src/components/Hub/FunnelRiskPanel.tsx` (104 lines, read in full)
- `frontend/src/style.css` (7611 lines; targeted sections read: `.hub-share-modal*` ~6380-6484, `:root`/`[data-ui-theme="light"]` token blocks ~4635-4745, `.session-share-panel__*` ~3040-3065, `.hub-funnel-risk-panel*` ~7297-7391, `.hub-funnel-write-gate*` ~7437-7572)
- `frontend/src/components/__tests__/SessionShareModal.test.tsx` (812 lines, read in full)
- `frontend/src/components/__tests__/SessionSharePanel.test.tsx` (first 90 lines read; structure confirmed)
- `frontend/src/components/__tests__/FunnelRiskPanel.test.tsx` (150 lines, read in full)
- `frontend/src/components/__tests__/SessionShareModal.disconnect.test.tsx` (148 lines, read in full)
- `frontend/package.json` (dependency versions)
- `frontend/vitest.config.ts` (test framework config)
- Live command execution: `pnpm test -- --run SessionShareModal.test.tsx` → 37/37 passed (verified working test infra)
- `.planning/ROADMAP.md` lines 658-669 (Phase 173 entry, SM-01..08 canonical definitions)
- `.planning/STATE.md` line 86 (phase status)
- `TESTING.md` (Suite Manifest, standing convention)
- `git log --oneline -- frontend/src/components/Hub/SessionShareModal.tsx frontend/src/components/SessionSharePanel.tsx` (confirms most recent touch was Phase 171-03, `40331fa2`)

### Secondary (MEDIUM confidence)
- `.planning/phases/173-share-modal-three-tab-segmented-redesign/173-CONTEXT.md` and `173-DESIGN-129.md` — authoritative for WHAT to build (locked design), but several of their embedded line numbers and CSS token names were found stale/inaccurate against actual HEAD during this research (documented above) — treat the design intent as authoritative, the literal line/token references as approximate.

### Tertiary (LOW confidence)
- None — no WebSearch or training-data-only claims were needed for this phase; it is entirely code-groundable.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all existing versions read directly from `package.json`
- Architecture: HIGH — every component boundary, prop, and state variable cited above was read directly from HEAD source, not inferred
- Pitfalls: HIGH for pitfalls 1-4 (each traced to a specific verified line range); MEDIUM for pitfall 5's forward-looking "will bite naive plan tasks" framing (inherently about future drift, not a current-state fact)

**Research date:** 2026-07-08
**Valid until:** Line-number references are valid only against commit `40331fa2` — re-verify before any plan task execution if HEAD has moved. Architectural/behavioral findings (state ownership, component boundaries, token vocabulary) are stable until the next phase that touches these same files.
