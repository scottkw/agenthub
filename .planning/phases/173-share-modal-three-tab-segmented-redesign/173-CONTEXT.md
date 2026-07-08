# Phase 173: Share modal three-tab segmented redesign - Context

**Gathered:** 2026-07-08
**Status:** Ready for planning
**Source:** GitHub design spec #129 (full spec — reviewed against 4 candidate layouts + interactive mockup before writing). Saved verbatim at `173-DESIGN-129.md`. This is a frontend-only UX/IA + polish redesign; no change to the sharing capability model.

<domain>
## Phase Boundary

Reorganize the session **Share** modal from a single growing column — where every toggle *injects* a block inline that pushes content down until the dialog overflows and scrolls — into a **stable frame**: a fixed toggle control strip + a three-tab **segmented access panel** (Tailnet·Private / Internet·Read-only / Internet·⚠ Full access) whose panels **swap instead of stacking**.

**In scope:** layout/IA reorganization, one reusable `ShareLinkCard`, a new `ShareSegmentedControl`, refactor of `SessionSharePanel` into three tab-body renderers, converting the inline-injected `FunnelRiskPanel` into a transient confirm view, migrating inline `style={}` layout to CSS classes in `style.css`, modal width bump, colorblind-safe + keyboard-accessible affordances, and updating the two share test suites.

**Explicitly NOT in scope (must remain unchanged):** which capabilities/tokens are issued, TTL semantics, Funnel teardown logic, the 3s hold-to-confirm safety gate behavior, and the already-shipped Sharing Guide Help content. This is layout/IA only.
</domain>

<decisions>
## Implementation Decisions

All decisions below are LOCKED by spec #129. Line refs are HEAD-relative starting points — verify before editing.

- **D-01 — Fixed control strip (SM-01)**
The three toggles (Share the session / Enable remote file browsing / Enable internet sharing) + the divider live at the top of `.hub-share-modal__body` as a **non-scrolling control strip**. Toggling a control NEVER reflows or pushes down content already on screen — detail appears in the swappable panel region below, not injected above the toggles. `Enable remote file browsing` is disabled until `shareEnabled`.

- **D-02 — Bounded height, single scroll region (SM-02)**
No state causes the **whole dialog** to scroll. The dialog fits `max-height: calc(100vh - 64px)`; any scroll is confined to one region (the active tab body), not the whole `.hub-share-modal__body`.

- **D-03 — Three-tab segmented control (SM-03)**
New `ShareSegmentedControl` component rendered directly under the divider, shown only when `shareEnabled`. Three segments with stacked two-line labels: `Tailnet` / `Private`, `Internet` / `Read-only`, `Internet` / `⚠ Full access`. Selecting a tab **swaps the panel body** — it is the ONLY region that changes. CSS classes: `.share-segbar`, `.share-seg`, `.share-seg__main`, `.share-seg__sub`, `.share-seg.is-active`, `.share-seg.is-danger`.

- **D-04 — Full-access / command-execution flow walled off (SM-04)**
The public-write flow (terminal-exposure warning → "Enable public write access…" → hold-to-confirm gate with expiry select + `HoldToConfirmButton` → collapses to armed summary: write URL / single-use code / countdown / "Disable public write") lives **ONLY** in the Internet·⚠ Full-access tab (`InternetFullAccessTab`). Nothing about public write appears anywhere else in the modal.

- **D-05 — Tab state machine + transient confirm view (SM-05)**
State inputs: `shareEnabled`, `internetEnabled`, `internetConfirmed`, `publicWriteArmed`.
- `!shareEnabled` → no tabs; empty hint "Turn on 'Share the session' to get a link."
- `shareEnabled && internetEnabled && !internetConfirmed` → **transient confirm view** (the repurposed `FunnelRiskPanel`: risk ack + auto-expire + "Keep local only" / "Enable internet share") REPLACES the panel — NOT injected above the links; no tabs shown.
- `shareEnabled && (!internetEnabled || internetConfirmed)` → segmented control + active tab body.

Tab availability: Tailnet enabled when `shareEnabled`; both Internet tabs enabled only when `internetConfirmed` (else rendered as dimmed `aria-disabled` segments so the tier structure is visible before reachable). On successful internet confirm the active tab **defaults to Internet·Read-only** (safer landing; Full-access is always a deliberate second click). Disabling internet sharing **resets the active tab to Tailnet** and disables both Internet tabs.

- **D-06 — One reusable ShareLinkCard (SM-06)**
New/extracted `ShareLinkCard`: title · truncated URL (with full URL in `title=`) · Copy/Open/QR buttons · join code (wraps existing `CodeDisplay`) · scope description attached **directly beneath** the link it describes. Used by ALL tailnet + internet link rows (Read-Only, Full Access, Public URL). Replaces the four ad-hoc hand-laid rows. CSS: `.share-linkcard`, `__top`, `__title`, `__url`, `__actions`, `__join`, `__desc`.

- **D-07 — Colorblind-safe + keyboard-operable (SM-07)**
Owner is colorblind — state must NOT rely on hue.
- Full-access tab distinguished by **⚠ glyph in label + red inset ring when active** (`box-shadow: inset 0 0 0 1px var(--danger-line)`), not color alone.
- Toggles read state by knob position + an **"On/Off/N/A" text label**, not just track color.
- Segmented control is a real `role="tablist"` with `role="tab"` / `aria-selected`, **arrow-key roving tabindex** (active tab `tabIndex=0`, others `-1`), and visible `:focus-visible` rings.
- Hold-to-confirm keeps a text label and works under `prefers-reduced-motion` (falls back to a plain confirm rather than a timed fill).

- **D-08 — Modal width + preserved behavior + tests (SM-08)**
Widen `.hub-share-modal` from `width: min(480px, calc(100vw - 48px))` to **`width: min(520px, calc(100vw - 48px))`** (`style.css` ~`:6318`); verify the responsive override (~`:6405`) still clamps on narrow viewports. Capabilities / tokens / TTL / Funnel-teardown / 3s hold-gate all UNCHANGED. `SessionShareModal.test.tsx` + `SessionSharePanel.test.tsx` updated to the new structure with **attribute-based, non-hue assertions** (verify state via source/attributes like `aria-selected`, `aria-disabled`, text labels, class presence — never by computed color).

- **D-09 — Components & preservation**
- `SessionShareModal.tsx` (shell) — keeps the 3 toggles as the fixed control strip; owns tab/confirm state (`tab`, `internetConfirmed`, `publicWriteArmed`); renders `ShareSegmentedControl` + active-tab dispatch.
- `ShareSegmentedControl` — NEW.
- `ShareLinkCard` — NEW/extracted.
- `SessionSharePanel.tsx` — refactored from one long column into `TailnetTab` / `InternetReadOnlyTab` / `InternetFullAccessTab` renderers.
- PRESERVE as-is: `HoldToConfirmButton`, `CodeDisplay`. `FunnelRiskPanel` is repurposed into the transient confirm view (no longer injected inline above links).
- Migrate current inline `style={}` layout to CSS classes in `frontend/src/style.css` (no CSS module for this modal).

### Claude's Discretion
- Exact new-component file paths/locations (follow existing `components/` + `components/Hub/` conventions), prop signatures, and internal helper structure.
- How much of `SessionSharePanel` becomes shared helpers vs per-tab code, as long as the three-tab-renderer decomposition holds.
- Wave/plan decomposition and test-file organization details.
- Whether QR toggle lives on the card vs a shared modal-level popover, provided each card exposes a QR affordance.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design spec (authoritative)
- `.planning/phases/173-share-modal-three-tab-segmented-redesign/173-DESIGN-129.md` — full #129 spec: IA diagram, tab state-machine table, design tokens, a11y affordances, component breakdown, code sketches, CSS, rejected alternatives, acceptance criteria, out-of-scope, and per-file line refs.

### Modal shell & inner panel (to modify)
- `frontend/src/components/Hub/SessionShareModal.tsx` (~733 lines) — body `.hub-share-modal__body` ~`:541`; toggles ~`:567`/`:605`/`:654`; `FunnelRiskPanel` inline ~`:691`; `SessionSharePanel` mounted ~`:707`.
- `frontend/src/components/SessionSharePanel.tsx` (~727 lines) — Read-Only ~`:438`, Full Access ~`:482`, INTERNET (PUBLIC) ~`:531`, PUBLIC WRITE gate ~`:609`; `HoldToConfirmButton` ~`:79`, `CodeDisplay` ~`:131`.
- `frontend/src/components/Hub/FunnelRiskPanel.tsx` — becomes the transient confirm view.

### CSS
- `frontend/src/style.css` — `.hub-share-modal` ~`:6318`, `.hub-share-modal__body` ~`:6361`, header ~`:6339`, responsive override ~`:6405`. New classes (`.share-segbar`, `.share-seg*`, `.share-linkcard*`) go here.

### Tests (to update)
- `frontend/src/components/__tests__/SessionShareModal.test.tsx`
- `frontend/src/components/__tests__/SessionSharePanel.test.tsx`
- Related suites that touch this surface: `FunnelRiskPanel.test.tsx`, `SessionShareModal.disconnect.test.tsx`.

### Regression convention
- `TESTING.md` (repo root) — updated share test files must be reflected in Suite Manifest / Traceability map per the standing rule in `agenthub/CLAUDE.md`.
</canonical_refs>

<specifics>
## Specific Ideas

- Design tokens (extend `style.css`, do NOT hardcode literals in components): accent `#7c8cf8`, link `#8ba0ff`, danger `#f2617a`, danger-line `#e5484d`, warn `#e3b341`, chip-bg/border `rgba(255,255,255,0.06)`/`0.10`, modal-border `rgba(255,255,255,0.09)`, mono `ui-monospace, "SF Mono", Menlo`. Reuse existing dark-theme vars where they exist (`--accent`, `--danger`, `--danger-line`, `--modal-border`, `--btn-bg`, `--m-text`, `--m-dim`).
- Segmented control keeps stacked two-line labels ("Internet" / "Read-only") as a belt-and-suspenders fit on narrow viewports.
- Full-access danger box copy (from spec): "⚠ You are exposing a terminal to the internet" + "Full access grants command execution on this machine, running as your account, until you disable it or it expires (max 1 hour). A leaked link = remote code execution."
- Code sketches in `173-DESIGN-129.md` are non-normative (illustrate structure, not final).
</specifics>

<deferred>
## Deferred Ideas

- Rejected layout alternatives (accordion, wizard/stepper, fixed-height-inner-scroll) — documented in #129 as considered-and-rejected; the fixed-height inner-scroll is noted only as a fallback if the segmented rework can't land in one phase.
- No changes to capabilities/tokens/TTL/Funnel teardown, and no changes to the Sharing Guide Help content (explicitly out of scope).
</deferred>

---

*Phase: 173-share-modal-three-tab-segmented-redesign*
*Context gathered: 2026-07-08 from GitHub design spec #129*
