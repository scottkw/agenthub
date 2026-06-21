# AgentHub v4.0 UI Spec (Phase 140 decision gate)

**Status:** Locked  
**Authors:** Phase 140 execution agent  
**Date:** 2026-06-21  
**Feeds:** Phase 141 (Redesign Implementation)

This document is the Phase 141 contract. All decisions below are locked — Phase 141 builds
against this spec without re-litigating direction, accent, or structural scope.

---

## Chosen Direction

**Direction 01 — Refined Native** is the chosen redesign direction (D-01).

Key characteristics:
- Sidebar navigation + tabs layout
- Comfortable density (not ultra-compact)
- Flat/native aesthetic — not glassy, not rounded-bubble (that was Direction 03 Mission Control)
- tmux-style footer status bar
- Both light and dark themes supported

This direction was authored/exported in Claude Design by the user. It is the direction built
out in the standalone HTML comp (see Canonical Visual Source below).

---

## Canonical Visual Source

The file `agenthub-v4.0-redesign/AgentHub Redesign (standalone).html` is the **canonical
visual source of truth for Phase 141** (D-02).

Key notes:
- It is a JS-bundled single-page application (SPA) that renders only in a browser — open in
  Chrome/Firefox to see the rendered output
- The standalone HTML represents Direction 01 Refined Native fully built out
- **IMPORTANT:** The standalone's CSS accent variable (`#7C8CFF` periwinkle violet) is
  **REJECTED** — see the Locked Accent & Visual Feel section below
- The comp covers Welcome, terminal/session, File Browser, Editor, and Settings surfaces.
  The Hub surface has no comp — see "Hub Has No Comp" section

---

## Review Provenance

Design provenance (D-03):

- **Authored in:** Claude Design (user-authored, exported as standalone HTML)
- **Reviewed via:** `c-*.png` screenshots (`c-welcome.png`, `c-session.png`, `c-settings.png`,
  `c-filebrowser.png`, `c-sessions.png`, `c-remote.png`) reviewed during the Phase 140 discussion
- **Live browser review:** No additional live-browser pass was required to lock the direction.
  A live walkthrough remains available in Phase 141 if needed (the standalone renders in any
  modern browser)

This satisfies ROADMAP Phase 140 success criterion #1: "A specific redesign direction is
recorded in a UI spec artifact after the standalone HTML comps are reviewed."

---

## Rejected Directions

No cross-direction mix (D-04). The following directions are **NOT** adopted, in whole or in part:

- **Direction 02 — Command Workspace:** violet accent, ⌘K command bar, live share rail
- **Direction 03 — Mission Control:** gradient/glassy card deck, rounded cards, heavy visual
  depth effects

Phase 141 must not harvest any distinctive UI element from Direction 02 (Command Workspace)
or Direction 03 (Mission Control). Those directions remain available as future-phase inspiration
but are out of v4.0 scope.

---

## Locked Accent & Visual Feel

### Blue Accent — Preserved (D-05)

The canonical blue accent tokens are **preserved as-is**. Phase 141 must NOT change the accent
color. The canonical tokens are defined in `frontend/src/style.css`:

- **Dark theme:** `--hub-accent: #7aa2f7`
- **Light theme:** `--hub-accent: #3d6fe8` (WCAG AA verified on `#ffffff`, contrast ≥ 4.5:1)

The standalone HTML's CSS accent `#7C8CFF` (periwinkle violet) is **REJECTED** and treated
as incidental to the Claude Design export — it is NOT the intended accent. Phase 141 must NOT
change the accent to `#7C8CFF` or any other periwinkle/violet value.

**Colorblind verification instruction:** The user is colorblind. Accent compliance must be
verified at the **hex constant level in source code** (grep for `#7aa2f7` / `#3d6fe8`), never
by eye. Do not infer accent correctness from rendered screenshots — trust the hex tokens, not
visual appearance.

### Visual Feel — Locked (D-06)

Phase 141 applies:
- **Comfortable density** — readable spacing, not ultra-compact
- **Flat/native aesthetics** — no glassy, gradient, or heavy-rounded treatment (that was
  Direction 03 and is rejected)
- **tmux-style footer status bar** — retained from the comp
- **Both light and dark themes** — the comp shows both; both must be restyled

### D-07 Carry-Forward Constraints

Phase 141 inherits these constraints without re-litigation:
- **`prefers-reduced-motion`:** All animations/transitions in restyled surfaces must respect
  this media query
- **Colorblind-safe semantics throughout:** Status, origin, and connection state must be
  conveyed via icon / label / badge / shape — never color alone (CARD-02/03 pattern carries
  forward into the restyle)

---

## Conflict Reconciliation (structural decisions win — RDS-03)

The standalone HTML was built on the **pre-Hub-first** navigation structure. Several comp
elements conflict with already-shipped Phase 138 decisions. Structural decisions WIN every
conflict. The following are resolved and locked:

### D-08: Sidebar Structure

**Comp shows:** `Home · Remote · Sessions · + New Session · Settings`

**Resolution — winning shipped structure (Phase 138, NAV-02/03/04/05):**
- Sidebar collapses to exactly **`Home · Hub · Settings`**
- The `Remote` and `Sessions` sidebar entries are **dropped** — they no longer exist as pages
- The `+ New Session` sidebar entry is **dropped** — session creation stays on the Hub's
  `HubFilterBar` "New Session" button (CARD-01)
- Phase 141 must NOT re-introduce `Remote`, `Sessions`, or `+ New Session` as sidebar items

### D-09: Sessions Page

**Comp shows:** A dedicated Sessions page (`c-sessions.png`) with per-session controls:
Web on/off, Browse files, Kill, Read-Only / Full-Access links, Copy/Open/QR, file-write/edit
toggles.

**Resolution:** The Sessions page is **NOT reintroduced** as a page.

Its controls already live on the shipped Hub surface:
- Kill, Browse files → per-card overflow menu (Phase 138 card overflow)
- Web-serve on/off, share links, QR codes, LAN password → Share modal (Phase 137 cap model)

Phase 141 must NOT harvest the Sessions-page layout or controls from `c-sessions.png`.

### D-10: Remote Page

**Comp shows:** A dedicated Remote page (`c-remote.png`) listing remote peers' sessions.

**Resolution:** The Remote page is **NOT reintroduced**.

Remote sessions live in the unified local+remote Hub grid. Remote provenance is shown via
colorblind-safe card indicators (CARD-02: icon/label/badge, never color alone). Phase 141
must NOT add a Remote sidebar page.

### D-11: Share Copy

**Comp footer text:** "Share links are on the Sessions tab"

**Resolution:** This copy must be reworded or removed — there is no Sessions tab. Sharing is
via the Hub / Share modal (Phase 137). Any reference to a "Sessions tab" in share dialogs
must be updated.

### D-12: Hub Has No Comp — See dedicated section below

### D-13: Restyle Depth — See "Restyle Depth" section below

This conflict resolution satisfies ROADMAP Phase 140 success criterion #2: "The chosen
direction is reconciled against Hub-first structure decisions (structural decisions win
conflicts)."

---

## Hub Has No Comp

**D-12:** The standalone HTML contains **no Hub surface** (D-12).

The Hub post-dates the comp — it was designed and shipped in Phase 131-135 (v3.6). The
standalone was authored before the Hub existed.

**Implication for Phase 141:**
- Phase 141 derives the Hub's restyle by **applying Direction 01 Refined Native's visual
  language** (typography, spacing, blue accent, card treatment) to the existing shipped Hub
- There is **no pixel comp to copy** for the Hub — Phase 141 must extrapolate from the
  Refined Native language applied to the other surfaces
- The Hub's session cards, group sidebar, filter bar, attention pulse, and grid layout are
  preserved structurally — only visual treatment (color, type, spacing) is updated

This explicit flag prevents Phase 141 from treating the Hub restyle as blocked on a comp
that does not exist.

---

## Restyle Depth

**D-13: Recolor only — keep current UX.**

Phase 141 applies the Refined Native visual language (color / typography / spacing) to the
existing surfaces. It does NOT:
- Change layout or interaction model of the share dialog
- Change layout or interaction model of Hub cards
- Harvest the Sessions-page share-control layout from the comp
- Alter the shipped Phase 137 share model (toggle → links/codes → QR → LAN password)
- Alter the shipped card interactions (click → modal, overflow menu, attention pulse,
  mini-preview, float-to-top)

The shipped Phase 137 share model and shipped card interactions are **preserved as-is**.

**Lowest-risk instruction:** Phase 141 is a visual restyle — not a structural redesign.
When in doubt, keep the existing component structure and update only color/type/spacing tokens.

---

## Surface Map

Phase 141 restyles the following surviving surfaces. All other surfaces (Sessions page, Remote
page) were removed in Phase 138 and are out of scope:

| Comp Name | Shipped Name | Comp File | Notes |
|-----------|-------------|-----------|-------|
| Welcome | Home | `c-welcome.png` | The comp "Welcome" maps to the shipped "Home" sidebar item |
| Hub | Hub | (no comp) | Derive from Refined Native language; see Hub Has No Comp |
| Session terminal | terminal/session | `c-session.png` | Full terminal surface with status bar |
| File Browser | File Browser | `c-filebrowser.png` | Phase 123-128 file browser |
| Editor | Editor | (inferred from filebrowser) | CodeMirror 6 editor (Phase 125) |
| Settings | Settings | `c-settings.png` | Settings panel |

The word "terminal" and "session" refer to the same surface in this context — the terminal tab
with an active session.

Phase 141 References (for downstream planning):
- Phase 137: Share modal & cap model (preserved, recolored only)
- Phase 138: Hub-First Navigation (sidebar Home/Hub/Settings structure — must not regress)
- Phase 139: Card rendering (headless VT mini-preview and tab strip — must not regress)

---

## Phase 141 Hand-off

This spec is the Phase 141 entry point. The following checklist ties the spec to requirements:

**Requirements delivered by Phase 141:**
- [ ] **RDS-02** — Chosen redesign applied across all surviving surfaces (Welcome→Home, Hub,
  terminal/session, File Browser, Editor, Settings)
- [ ] **RDS-03** — Reconciled with Hub-first structure; conflicts resolved (this spec); no
  Sessions/Remote pages reintroduced; structural decisions win (CARRY-02 feeds here)
- [ ] **RDS-04** — `prefers-reduced-motion` honored; colorblind-safe semantics maintained
  across all restyled surfaces in both light and dark themes; verified at hex-constant level
- [ ] **CARRY-01** — Hub GroupSidebar ARIA model made internally consistent (#97): either
  `listbox`/`option` with roving-tabindex correctly implemented, or both replaced with a plain
  focusable control list; no mismatched role/interaction pattern

**Key constraints Phase 141 must honor:**
1. Blue accent tokens `#7aa2f7` (dark) / `#3d6fe8` (light) are preserved; `#7C8CFF` is
   rejected; colorblind verification at hex level
2. Recolor-only depth (D-13) — no structural UX changes
3. Hub-first structure not regressed (D-08..D-11) — no Sessions/Remote pages, no sidebar
   `+ New Session`
4. Hub derived from Refined Native language (no pixel comp) — D-12
5. `prefers-reduced-motion` on all animations (D-07)
6. Both light + dark themes (D-06)
7. Colorblind-safe semantics throughout — icon/label/shape, never color alone (D-07)

**Canonical visual source:** `agenthub-v4.0-redesign/AgentHub Redesign (standalone).html`
(open in browser; reject its `#7C8CFF` accent — use canonical blue tokens)

---

*Spec locked: 2026-06-21*  
*Phase: 140-ui-spec-gate*  
*Next phase: 141-redesign-implementation*
