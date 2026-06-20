# Phase 138: Hub-First Navigation - Context

**Gathered:** 2026-06-20
**Status:** Ready for UI-SPEC + planning
**Source:** Parity sign-off (research-driven; discuss-phase skipped per `skip_discuss: true`)

<domain>
## Phase Boundary

GUI-only React/TSX refactor. Restructure the sidebar to exactly Home / Hub / Settings,
delete the Sessions (`DaemonManagerPanel`) and Remote (`RemoteSessionsPanel`) pages, route
all session creation through the `HubFilterBar` "New Session" button, and add colorblind-safe
local/remote + connection-state indicators to session cards. No backend/Go/CLI changes —
grep of `cmd/` and `internal/` confirms zero references to the deleted panels.

**In scope:** NAV-02, NAV-03, NAV-04, NAV-05, CARD-01, CARD-02, CARD-03, CARD-04.
**Not in scope:** CARD-05 (mini-preview legibility, Phase 139), the RDS redesign (Phases 140–141),
TAB strip (Phase 139).

</domain>

<decisions>
## Implementation Decisions

### Parity-before-deletion sign-offs (LOCKED — user, 2026-06-20)

The Sessions and Remote pages are deleted (NAV-03/NAV-04). Per the release-blocking
cross-surface-parity rule, every control they exposed is either already on the Hub or
explicitly migrated. The user signed off on **full parity preservation — nothing deferred**:

1. **Per-session Kill → MIGRATE to card overflow menu.** `DaemonManagerPanel`'s Kill control
   has no Hub equivalent. Add a per-card overflow menu containing "Kill session" (with
   confirmation). Capability `handleCloseTab`/`onKill` already exists in App; only the card
   affordance is new. MUST be guarded against the Phase 134 card-click modal
   (`.closest()` guard + `e.stopPropagation()`).

2. **Remote "Open in browser" → KEEP IT (migrate alongside the in-app modal).** The Remote
   page's "Open Session" opened the remote session in the system browser (`BrowserOpenURL`).
   The Hub's card-click opens the Phase 134 in-app interactive modal. The user wants BOTH:
   keep an explicit "open in browser" affordance on remote cards in addition to the in-app
   modal. `handleOpenRemoteSession`/`onOpen` exists in App.

3. **Remote "Browse files" on-ramp → ADD a dedicated browse button.** The Remote page had a
   dedicated "Browse Files" button (join-code cap flow). The user wants an explicit
   "Browse files" affordance on remote cards, not only the implicit card-click cap gate.
   `handleBrowseFilesRemote` + `RemoteJoinCodeModal` exist in App.

4. **Per-peer "Unreachable" / "no shareable sessions" → MIGRATE a status hint.** Preserve a
   lightweight indicator on the Hub so users can tell an unreachable/offline peer from a peer
   with nothing shared (vs. cards simply being absent with no message).

### Card scope implication (drives CARD-04 + the UI-SPEC)

The card must now accommodate, without losing attention pulse / float-to-top, mini-preview,
grid density, or responsive reflow:
- Share button (from Phase 137)
- Local vs remote origin indicator (CARD-02) — icon/label/badge, **never color alone**
- Remote connected-vs-available indicator (CARD-03) — colorblind-safe
- Overflow menu (Kill session) — decision 1
- Remote-only: "open in browser" affordance — decision 2
- Remote-only: "Browse files" affordance — decision 3
- A per-peer unreachable/empty status hint surfaced on the Hub — decision 4

How these are laid out (placement, overflow vs inline, icon set, badge vs label) is a
**UI-SPEC decision** (`/gsd:ui-phase 138`), to be made before planning.

### Data sources (from research — no new backend state)

- **Local vs remote origin (CARD-02):** already rendered on the card; provenance is on the
  session model (do NOT infer from hostname — see RESEARCH anti-patterns).
- **Connected vs available (CARD-03):** `remoteCapsCached: Set<string>` already lives in App
  and is passed to `HubPanel`; thread it down to `SessionCard` as a new prop. No new state.

### Deletion hazard (LOCKED — non-negotiable, from research)

`RemoteSessionsPanel.tsx` currently EXPORTS the `RemoteSession` / `RemotePeerSessions` **types**
consumed by `remoteAdapter.ts`, `remoteSession.ts`, and `App.tsx` to feed the Hub's remote-card
pipeline. The plan MUST relocate those types to `lib/remoteSession.ts` BEFORE deleting the panel,
or the Hub breaks (TS compile failure). The remote-session poll the panel owned must not be
starved either (RESEARCH Pitfall 4).

### Stale structural tests are Wave 0 (from research)

`Sidebar.test.tsx` asserts a Sessions button, a New Session button, and `items.length === 6`;
`App.hub.test.tsx` / `style.hub.test.ts` encode the old `.hub__header`. These RED-fail until
updated to the 3-item sidebar and header-less Hub — update them as Wave 0 work.

### Claude's Discretion

- Exact overflow-menu component/pattern and icon choices (subject to UI-SPEC).
- Test file organization for new card affordances.
- Whether "open in browser" + "Browse files" live in the overflow menu or inline (UI-SPEC).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Research + validation (this phase)
- `.planning/phases/138-hub-first-navigation/138-RESEARCH.md` — file-by-file routing-cleanup map,
  parity inventory, card preservation checklist, connection-state data source, Wave 0 test gaps
- `.planning/phases/138-hub-first-navigation/138-VALIDATION.md` — Nyquist validation contract

### Source of truth (React/TSX frontend)
- `frontend/src/components/Sidebar.tsx` — sidebar nav (NAV-02/03/04/05)
- `frontend/src/components/DaemonManagerPanel.tsx` — Sessions page (NAV-03 DELETE; migrate Kill first)
- `frontend/src/components/RemoteSessionsPanel.tsx` — Remote page (NAV-04 DELETE; relocate types first)
- `frontend/src/lib/remoteSession.ts`, `frontend/src/lib/remoteAdapter.ts` — remote types/pipeline
- `frontend/src/components/Hub/HubFilterBar.tsx` — canonical "New Session" button (CARD-01)
- `frontend/src/components/Hub/HubPanel.tsx` — Hub page, owns `.hub__header` (CARD-01)
- `frontend/src/components/Hub/SessionCard.tsx` — card (CARD-02/03/04)
- `frontend/src/App.tsx` — routing + sidebar wiring + handler call sites (Kill/open/browse handlers)

</canonical_refs>

<specifics>
## Specific Ideas

- Card-click → Phase 134 interactive modal is preserved; any new clickable card child (Kill,
  open-in-browser, browse, overflow trigger) MUST add a `.closest()` guard + `e.stopPropagation()`
  so it doesn't trigger the modal (RESEARCH Pitfall 2).
- Colorblind-safe is a hard release norm: indicators use icon/text/shape, never color alone
  (user is colorblind; verify at source level — hex/icon constants in code, not by eye).

</specifics>

<deferred>
## Deferred Ideas

None deferred for parity — user chose full preservation on all four decisions. (CARD-05,
TAB-*, and the RDS redesign are separate phases, not deferrals from this phase.)

</deferred>

---

*Phase: 138-hub-first-navigation*
*Context captured: 2026-06-20 via research-driven parity sign-off (discuss-phase skipped)*
