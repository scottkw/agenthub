# Phase 164: Web-Share Chat Layout Polish — Context

**Gathered:** 2026-06-28
**Status:** Ready for planning
**Source:** Direct scoping conversation during /gsd-plan-phase 164 (no separate discuss-phase)

<domain>
## Phase Boundary

Two web-share chat layout issues discovered during Phase 161 live UAT, both fixed in the
**shared** `ChatPanel` / `ChatMessage` components + `style.css` so all surfaces benefit
(GUI session tab, Hub interactive modal, web-share guest):

1. **CHAT-LAYOUT-01 (bug):** A web peer's raw `authorID` (`nodekey:456d9361bab4eb7…`)
   renders un-truncated as the `.chat-msg__tailnet-id` secondary label, forcing horizontal
   scroll on the web-share `/app/` surface.
2. **CHAT-LAYOUT-02 (feature):** The chat drawer is a fixed 360px width — make it resizable.

This phase is **frontend-only**. No Go / relay / protocol / server changes.
</domain>

<decisions>
## Implementation Decisions (LOCKED)

### CHAT-LAYOUT-01 — secondary-label overflow

**Root cause (confirmed by code trace):** The WEBCHAT-06 ellipsis CSS is correct
(`.chat-msg__header { min-width:0 }`, `.chat-msg__alias` + `.chat-msg__tailnet-id` carry
`overflow:hidden; text-overflow:ellipsis; min-width:0; flex-shrink`). It does NOT engage
because the **virtualizer row that wraps each `ChatMessage` has no width constraint**.
In `ChatPanel.tsx` `getRowStyle()`, non-separator rows return
`{ position:'absolute', top:0, left:0, transform:translateY(...) }` — an absolutely
positioned box with **no width** sizes shrink-to-fit to its content, so it grows as wide
as the longest header line and the flex ellipsis never has a bounded container to shrink
within. Long `nodekey:` authorIDs only appear on web-share, which is why it surfaced there.

**Decision 1 (root-cause fix):** Constrain the absolute virtualizer row to the thread
width so the existing WEBCHAT-06 ellipsis engages. (e.g. add `width:'100%'` / `right:0`
to the non-separator branch of `getRowStyle`.) This is the actual reported bug fix and is
belt-and-suspenders for the alias label too.

**Decision 2 (secondary label = short fingerprint):** Instead of rendering the full raw
`(nodekey:456d…)` as the `.chat-msg__tailnet-id` secondary label, render a **short
fingerprint — the last 6 characters of `authorID`** — as a compact disambiguator.
- Derived **client-side** in `ChatMessage.tsx` from the existing `authorID` prop. **No
  server change, no protocol change.**
- The friendly Tailscale name is ALREADY the primary `.chat-msg__alias` label
  (server resolves `who.Node.ComputedName` → `AuthorAlias`, `server.go:1126` / `hub.go:622`),
  so the secondary slot only needs to be a short stable disambiguator, not a full name.
- `authorID` itself MUST NOT change — it is the identity/dedup key used by
  `tailnetIdToHue()` and mention matching (`currentUserTailnetID`). Only the *displayed*
  secondary string is shortened.
- Keep the `"local"` desktop sentinel readable (it is short already; the fingerprint
  transform must degrade gracefully for short / non-`nodekey:` IDs).

### CHAT-LAYOUT-02 — resizable chat width

**Decision 3 (drag-to-resize):** Add a left-edge drag handle on the chat drawer that
resizes its width.
**Decision 4 (persist + bounds):** Persist the chosen width to `localStorage` and restore
it on reload; clamp to a **min/max of ~280–640px** (final exact bounds at planner
discretion, but must be bounded both ways).
**Decision 5 (all surfaces):** Implement in the shared `ChatPanel` + `style.css` so the
GUI session tab, Hub interactive modal, AND web-share guest all get the resize affordance
from one implementation. The drawer width is currently set in two CSS rules
(`.chat-panel { width:360px }` base + `.hub-modal__body--interactive .chat-panel { width:360px }`)
— the resize must drive both / be reconciled so a runtime width override wins on every surface.
- Overlay mode (D-02) is preserved: resizing the drawer must NOT resize the terminal /
  trigger a PTY `sendResize` — it remains an overlay over the terminal's right edge.
- The `.chat-panel--open ~ .hub-modal__chat-toggle { right: 372px }` offset (CHAT-FIX-01)
  is hard-coded to the 360px drawer; it must track the live drawer width so the toggle
  stays clear of the resized drawer.

### Claude's Discretion
- Exact resize-handle implementation (pointer events vs CSS `resize`), DOM placement of the
  handle, and how the live width is applied (inline style / CSS custom property on the panel).
- Exact min/max px bounds within the ~280–640px envelope and localStorage key name.
- Fingerprint format details (separator/parens) as long as it is the last 6 chars of the key
  and visually reads as a short disambiguator.
- Whether to drive both CSS width rules via a single CSS custom property (e.g. `--chat-panel-width`).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Shared chat components (all changes land here)
- `frontend/src/components/Hub/ChatMessage.tsx` — secondary-label render (line ~94); add the
  client-side fingerprint transform here.
- `frontend/src/components/Hub/ChatPanel.tsx` — `getRowStyle()` (lines ~235-249, the
  unconstrained-width root cause); drawer root render + header; add resize handle + width state here.
- `frontend/src/components/Hub/WebShareSessionView.tsx` — web-share mount of ChatPanel
  (reference: confirms shared component path; no per-surface code expected).
- `frontend/src/style.css` — `.chat-msg__header` / `.chat-msg__alias` / `.chat-msg__tailnet-id`
  (WEBCHAT-06 ellipsis, lines ~6360-6408); `.chat-panel` base width (line ~6676); 
  `.hub-modal__body--interactive .chat-panel` width (line ~6009); 
  `.chat-panel--open ~ .hub-modal__chat-toggle` offset (line ~6059).

### Reference only — DO NOT modify (proves friendly name already exists server-side)
- `internal/webserver/server.go` (lines ~1116-1145) — `WhoIs` → `ComputedName` → `defaultAlias`.
- `internal/relay/hub.go` (lines ~620-622, 680-681) — `AuthorID = sub.TailnetID` (raw nodekey),
  `AuthorAlias = sub.Alias` (friendly name).

### Tests
- `frontend/src/components/Hub/ChatPanel.test.tsx`, `ChatMessage` unit tests — extend for the
  fingerprint helper and `getRowStyle` width; add resize/persistence coverage.
- `TESTING.md` — update Suite Manifest + Traceability map per the standing regression convention.
</canonical_refs>

<specifics>
## Specific Ideas

- The fingerprint helper should be a small pure exported function (testable like the existing
  `tailnetIdToHue` / `formatHHMM` helpers) so it gets unit coverage.
- Resize width is cleanest as a single source of truth (one CSS custom property the JS sets),
  avoiding divergence between the two `width:360px` CSS rules.
- `currentUserTailnetID` mention matching and `tailnetIdToHue` must keep using the full
  `authorID` — only the rendered secondary string changes.
</specifics>

<deferred>
## Deferred Ideas

- Showing a *distinct* friendly name (e.g. full FQDN `who.Node.Name`) in the secondary slot —
  not needed; the friendly `ComputedName` is already the primary alias. Would require a new
  chat-message field if ever wanted; explicitly out of scope.
- Vertical (height) resizing of the drawer — only width is in scope (CHAT-LAYOUT-02).
</deferred>

<notes>
## Process Notes

- **Requirement IDs not yet in REQUIREMENTS.md:** `CHAT-LAYOUT-01` and `CHAT-LAYOUT-02` exist in
  ROADMAP.md Phase 164 but are not present in `.planning/REQUIREMENTS.md` (same gap pattern noted
  for ROCHAT IDs). Both IDs MUST still appear in each plan's `requirements` field. Consider adding
  them to REQUIREMENTS.md during execution so the milestone audit reconciles.
</notes>

---

*Phase: 164-web-share-chat-layout-polish*
*Context gathered: 2026-06-28 via direct scoping conversation*
