# Phase 138: Hub-First Navigation - Research

**Researched:** 2026-06-20
**Domain:** React/TSX GUI refactor — sidebar/route deletion, card indicators, parity-preserving page removal
**Confidence:** HIGH (entire surface is in-repo; no external dependencies, no new packages)

## Summary

Phase 138 is a **pure GUI consolidation** with zero backend, Go, or CLI surface. It removes two
sidebar pages (Sessions = `DaemonManagerPanel`, Remote = `RemoteSessionsPanel`) and the sidebar
"New Session" item, collapses the sidebar to Home/Hub/Settings, deletes the Hub's duplicate
`.hub__header` "New session" button (leaving `HubFilterBar`'s button as the sole creation entry),
and adds local/remote + connected/available indicators to `SessionCard`.

The single highest-risk item is **parity-before-deletion**. `DaemonManagerPanel` and
`RemoteSessionsPanel` still own real functionality (web-share toggle, file-browse on-ramp, kill,
remote open-in-browser, remote join/connect). Most of it was migrated in Phase 137 (Share modal) and
Phases 131-134 (Hub cards + interactive modal + remote-on-desktop), but **three things are NOT yet
on the Hub**: per-session **Kill**, remote **"Open Session in browser"**, and the remote **"Browse
files" on-ramp**. The plan must close (migrate) or explicitly defer each gap with sign-off — see the
Parity Inventory below. Cross-surface parity is release-blocking for this project [CITED: MEMORY.md feedback_cross_surface_parity].

There is one **non-obvious deletion hazard**: `RemoteSessionsPanel.tsx` is not just a page — it
**exports the `RemoteSession` / `RemotePeerSessions` TypeScript types** that `remoteSession.ts`,
`remoteAdapter.ts`, and `App.tsx` all import to drive the Hub's remote-session pipeline. Deleting the
file naively breaks the Hub. The types must be **relocated** (e.g. to `lib/remoteSession.ts`), not
deleted, before the component is removed.

For the card indicators (CARD-02/03), the data is **already present**: `SessionCard` already renders
a local/remote origin marker (`ComputerDesktopIcon` + "Local" vs `GlobeAltIcon` + hostname) — CARD-02
is largely satisfied today and mostly needs verification + possible visual promotion. CARD-03
("connected vs available") needs a **new prop threaded from `App.tsx`'s `remoteCapsCached` set** (the
sessions for which a join code has been exchanged = "connected"); no new backend state is required.

**Primary recommendation:** Sequence the plan as (Wave 0) update the existing structural tests RED →
(1) relocate remote types out of `RemoteSessionsPanel.tsx`, (2) thread a `connectedRemoteIds`/origin
prop into `SessionCard` for CARD-02/03 + resize for CARD-04, (3) remove `.hub__header` (CARD-01),
(4) delete the two panels + sidebar items + App routing/Tab-type/poll cleanup (NAV-02..05), closing
the Kill/remote-open/remote-browse parity gaps on the Hub first.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Sidebar nav structure | Frontend (React) | — | `Sidebar.tsx` is a pure presentational nav; props wired in `App.tsx` |
| Page routing (which panel renders) | Frontend (App.tsx) | — | `activeId === *_TAB.id` switch in `App.tsx` render body |
| Session creation entry point | Frontend (HubFilterBar) | App.tsx (`onNewSession` → `setShowNewSessionModal`) | Modal already owned by App; button is the trigger |
| Local/remote origin signal | Frontend (SessionCard) | App.tsx (provenance: `sessions` vs `remoteSessions` prop) | Data already on `SessionInfo.hostname`; provenance is the true discriminator |
| Connected-vs-available signal | Frontend (SessionCard) | App.tsx (`remoteCapsCached` set) | "Connected" = join-code exchanged; state lives in App, must be threaded down |
| Web-share / file-browse / kill controls | Frontend (Hub Share modal + card buttons) | Daemon (Wails RPC) | Already migrated to Share modal (137) except Kill + remote on-ramps |
| Remote session data pipeline | Frontend (lib/remoteAdapter + remoteSession) | Daemon (`GetRemoteSessionsWithMeta`) | Hub consumes `adaptAllRemoteSessions(remotePeers)`; types currently in panel file |

## Standard Stack

No new libraries. This phase uses the existing frontend stack exclusively.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| react | ^19.2.4 | UI | [VERIFIED: package.json] Existing app framework |
| @heroicons/react | ^2.2.0 | Indicator icons (origin, connection, lock) | [VERIFIED: package.json + node_modules] Already the card's icon source |
| vitest | ^4.1.0 | Unit/JSDOM tests | [VERIFIED: package.json] Project test runner (`npm test` → `vitest run`) |
| @playwright/test | ^1.59.1 | E2E (if needed) | [VERIFIED: package.json] Available but not required for this phase |

**Installation:** None — `git` operations and edits only. No `npm install`.

### Verified icon availability (for CARD-02/03 indicators)
`node_modules/@heroicons/react/24/outline/` confirmed to contain: `ComputerDesktopIcon`,
`GlobeAltIcon`, `LinkIcon`, `CheckCircleIcon`, `BoltIcon`, `ArrowsRightLeftIcon`, `LockClosedIcon`,
`CloudArrowDownIcon` [VERIFIED: ls node_modules]. The card **already imports** `ComputerDesktopIcon`
(local) and `GlobeAltIcon` (remote) for its origin row. `SignalIcon`/`WifiIcon` were NOT confirmed —
do not assume they exist; prefer the verified set above. [ASSUMED] `LinkIcon`/`BoltIcon` are
appropriate "connected" glyphs pending UI-spec (Phase 140); the card already uses text+shape, so
icon choice is reinforcement, not the sole signal.

## Package Legitimacy Audit

> Not applicable — this phase installs **no external packages**. All work is edits/deletes against
> existing in-repo TypeScript and CSS. No registry interaction, no `npm install`, no slopcheck target.

## Architecture Patterns

### System Architecture Diagram (data flow for the affected surface)

```
                         ┌─────────────────────────── App.tsx (root state) ───────────────────────────┐
                         │                                                                              │
  Wails RPC              │  ListSessions() ──poll(3s, Hub active)──▶ hubSessions: SessionInfo[]         │
  (daemon)   ───────────▶│  GetRemoteSessionsWithMeta() ─poll(30s)─▶ remotePeers: RemotePeerSessions[]  │
                         │                                            │                                  │
                         │                              adaptAllRemoteSessions() ▶ remoteSessions[]      │
                         │  remoteCapsCached: Set<id>  (join-code exchanged = "connected")               │
                         │                                                                              │
                         └───────┬───────────────────────────────────────┬──────────────────────────────┘
                                 │ props                                  │ props
                    ┌────────────▼───────────┐               ┌────────────▼──────────────┐
                    │ Sidebar.tsx            │               │ HubPanel                  │
                    │  Home / Hub / Settings │               │  .hub__header  (DELETE)   │
                    │  (Remote/Sessions/New  │               │  HubFilterBar [New Session]│ ◀ sole entry
                    │   Session  DELETED)    │               │  SessionCardGrid          │
                    └────────────────────────┘               │    └ SessionCard          │
                                                             │        origin row (local/ │
   DELETED PAGES (NAV-03/04):                                │        remote) CARD-02     │
   DaemonManagerPanel  ─┐  features migrate to Hub          │        +connected/available│
   RemoteSessionsPanel ─┘  (kill / remote-open / browse)    │        CARD-03 (new prop)  │
        │ but its RemoteSession/RemotePeerSessions TYPES     │        Share btn (137)     │
        │ are imported by remoteAdapter+remoteSession+App ──▶│  CARD-04 resize to fit     │
        └─ RELOCATE TYPES before deleting the component file └────────────────────────────┘
```

### Recommended approach (file-by-file)

```
frontend/src/
├── components/
│   ├── Sidebar.tsx              # NAV-02/03/04/05: remove Remote, Sessions, New Session buttons + props
│   ├── DaemonManagerPanel.tsx   # NAV-03: DELETE (after migrating Kill to Hub)
│   ├── RemoteSessionsPanel.tsx  # NAV-04: DELETE component — but RELOCATE its exported types first
│   ├── Hub/
│   │   ├── HubPanel.tsx         # CARD-01: remove .hub__header block (lines 461-467)
│   │   ├── SessionCard.tsx      # CARD-02/03/04: promote origin row; add connected/available indicator; resize
│   │   └── SessionCardGrid.tsx  # CARD-03: thread connectedRemoteIds prop through both render paths
│   └── __tests__/
│       ├── Sidebar.test.tsx     # Wave 0: update — currently asserts Sessions/New Session + items.length===6
│       ├── App.hub.test.tsx     # Wave 0: update routing expectations
│       └── style.hub.test.ts    # Wave 0: update if .hub__header rules are asserted
├── lib/
│   ├── remoteSession.ts         # NAV-04: NEW HOME for RemoteSession/RemotePeerSessions types
│   └── remoteAdapter.ts         # NAV-04: update import to new type location
└── App.tsx                      # NAV-02..05: remove DAEMON_MANAGER_TAB / REMOTE_SESSIONS_TAB consts,
                                 #   handlers, polls, render branches, Tab-type-filter lists, Sidebar props
```

### Pattern 1: Provenance-not-hostname for local/remote (CARD-02)
**What:** Local vs remote is decided by **which prop a session arrived on** (`sessions` = local,
`remoteSessions` = remote), NOT by `hostname`. Local sessions carry the machine's own
`os.Hostname()`, so a hostname check misclassifies every local session as remote.
**When to use:** Anywhere CARD-02/03 needs to know if a card is remote.
**Example (existing, authoritative):**
```typescript
// Source: frontend/src/components/Hub/HubPanel.tsx:302-305 (GAP-134-A)
const remoteIdSet = React.useMemo(
  () => new Set((remoteSessions ?? []).map((s) => s.id)),
  [remoteSessions],
)
// SessionCard currently uses: const isLocal = !hostname || hostname === ''
// CARD-02 should switch to provenance (pass an isRemote/origin prop) for correctness.
```
**Note:** `SessionCard.tsx:164` currently derives `isLocal` from `hostname` only. For remote
adapter sessions `hostname` is the peer name (non-empty) so it happens to work, but the
HubPanel comment explicitly warns hostname is not a reliable discriminator. CARD-02 should
thread an explicit origin/`isRemote` prop derived from provenance for robustness.

### Pattern 2: Connected-vs-available signal source (CARD-03)
**What:** A remote session is "connected" when the user has exchanged a join code for it — tracked
by `remoteCapsCached: Set<string>` in `App.tsx` (already passed to `HubPanel` as `remoteCapsCached`).
**When to use:** CARD-03 indicator.
**Example:**
```typescript
// Source: frontend/src/App.tsx:209 + HubPanel prop (line 1384)
// remoteCapsCached.has(sessionId) === true  → "Connected"
// remote session NOT in the set            → "Available"
// Thread remoteCapsCached → SessionCardGrid → SessionCard as a connectedRemoteIds Set.
```
This requires NO new backend state — `remoteCapsCached` already exists and is the de-facto
connection ledger (it gates the card-click modal and the file-browser proxy today).

### Pattern 3: Colorblind-safe indicators (CARD-03, release norm)
**What:** Every status carries an **icon shape + text label**; color is reinforcement only. The card
already does this consistently (`STATUS_CONFIG` with per-status icons + labels, origin row with
Computer/Globe icon + "Local"/hostname text, Share lock icon).
**When to use:** CARD-02 and CARD-03 indicators MUST follow this — never color alone.
[CITED: MEMORY.md user_colorblind — verify color-based UAT at source level (hex constants), not by eye]
**Example (existing precedent):**
```tsx
// Source: frontend/src/components/Hub/SessionCard.tsx:357-369 (origin row)
{isLocal ? (
  <><ComputerDesktopIcon className="hub-card__origin-icon" aria-hidden="true" /><span>Local</span></>
) : (
  <><GlobeAltIcon className="hub-card__origin-icon" aria-hidden="true" /><span>{hostname}</span></>
)}
```
For CARD-03, mirror this: e.g. a `LinkIcon` + "Connected" vs an outline/`GlobeAltIcon` + "Available"
text chip, plus an `aria-label` suffix on the card (the card aria-label is built at
`SessionCard.tsx:180`).

### Anti-Patterns to Avoid
- **Deleting `RemoteSessionsPanel.tsx` before relocating its types:** breaks `remoteAdapter.ts`,
  `remoteSession.ts`, and `App.tsx` imports (TypeScript compile failure — exactly the Phase 137
  Rule-1 auto-fix class). Relocate `RemoteSession`/`RemotePeerSessions` first.
- **Removing a sidebar item without removing its App.tsx handler/poll/Tab branch:** leaves dead
  `handleOpenDaemonManager`/`handleOpenRemoteSessions`, `*_TAB` consts, poll effects, and
  Tab-type-filter strings (`App.tsx:1517`, `1556`) referencing deleted types — lint/compile noise
  and dead code.
- **Using `hostname` as the local/remote discriminator** (Pattern 1) — misclassifies local cards.
- **Color-only indicators** — violates the release-blocking colorblind norm.
- **Rebuilding the New Session flow** — `HubFilterBar`'s button already calls `onNewSession` →
  `App.setShowNewSessionModal(true)`. CARD-01 is a deletion, not a rebuild.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Local/remote detection | New hostname heuristic | Provenance set (`remoteIdSet`, HubPanel:302) | Already solved; hostname is documented-unreliable |
| Connection ledger | New "isConnected" backend field | Existing `remoteCapsCached` set (App.tsx:209) | Already the source of truth for cap-gated actions |
| New Session entry | New button/modal | `HubFilterBar` button → `onNewSession` (existing) | Sole entry point is the requirement; modal exists |
| Remote session adaptation | New mapper | `adaptAllRemoteSessions` (remoteAdapter.ts) | Already feeds the Hub grid |
| Colorblind-safe status pattern | New badge system | `STATUS_CONFIG` icon+label pattern (SessionCard) | Proven, contrast-verified, reused across card |
| Card grid density / responsive reflow | New CSS grid | `.hub__card-row` `repeat(auto-fill, minmax(240px, 1fr))` (style.css:4287) | Already responsive; CARD-04 must preserve it |
| Float-to-top / attention pulse | New animation | `useFLIPAnimation` + `.hub-card--attention` (SessionCardGrid:86, style.css:4939) | Preserve, do not reimplement |

**Key insight:** Almost everything CARD-02/03/04 needs already exists in the Hub — Phases 131-137
built the card, the origin row, the remote pipeline, the attention/FLIP system, and the Share modal.
138 is mostly **deletion + threading one connection prop + verifying preservation**, not new
construction.

## Parity-Before-Deletion Inventory (HIGHEST-VALUE OUTPUT)

> NAV-03 deletes `DaemonManagerPanel` (Sessions page); NAV-04 deletes `RemoteSessionsPanel` (Remote
> page). Cross-surface parity is release-blocking. Every control below must already exist on the Hub
> or be migrated/deferred-with-sign-off.

### DaemonManagerPanel (Sessions page) — `DaemonManagerPanel.tsx`
| Feature/control on the page | Already on Hub? (where) | Gap if deleted | Required action |
|-----------------------------|-------------------------|----------------|-----------------|
| **Web-share on/off + RO/RW links/codes + QR** | YES — Share modal (Phase 137 `SessionShareModal`, opened by card Share button) | None | Verify SHARE-05 lifecycle parity (already verified in 137) |
| **LAN Basic Auth password display** | YES — Share modal (SHARE-04, `GetLocalNetworkPassword`) | None | None |
| **"Enable file writes" / browse toggle** | YES — Share modal "Enable remote file browsing" (SHARE-03, `SetSessionBrowse`) | None | None |
| **Home-dir write warning** | YES — Share modal (`HomeDirWriteWarning`, D-09) | None | None |
| **Open (re-attach terminal tab)** | YES — `SessionCard` "Open" button (`onOpenSession`, line 400) | None | None |
| **Browse files (local)** | PARTIAL — Hub card-click opens interactive modal (Phase 134); file-browser on-ramp from a card is not a dedicated button | Local file-browse via a card button | **Confirm** card-click modal + Share modal cover this; if not, add. `handleOpenFileBrowser` exists in App. |
| **Kill session** | **NO — not on the Hub** | **GAP: no per-card Kill** | **MIGRATE** — add a Kill affordance to `SessionCard` (overflow menu is the natural home; `onKill`/`handleCloseTab` already exists in App) OR explicitly defer with sign-off |
| Session status dot + name + cli + hostname | YES — card row1/row2 (richer than the panel) | None | None |

### RemoteSessionsPanel (Remote page) — `RemoteSessionsPanel.tsx`
| Feature/control on the page | Already on Hub? (where) | Gap if deleted | Required action |
|-----------------------------|-------------------------|----------------|-----------------|
| **List tailnet peers + their shareable sessions** | YES — `adaptAllRemoteSessions(remotePeers)` merged into Hub grid (HubPanel:295) | None | None |
| **Peer reachable/unreachable + empty states** | PARTIAL — Hub shows merged cards; per-peer "Unreachable"/"no shareable sessions" messaging is panel-only | Loss of explicit unreachable/peer-empty messaging | **Decide:** acceptable loss (cards simply absent) or migrate a hint. Likely **defer with sign-off** (cosmetic) |
| **"Open Session" (open remote session in browser, `BrowserOpenURL`)** | **NO dedicated equivalent** — Hub card-click opens the in-app interactive modal (Phase 134), not the browser | **GAP: "open in browser" affordance** | **MIGRATE or defer** — confirm the Phase 134 in-app modal supersedes "open in browser"; if browser-open is still wanted, add to card. `handleOpenRemoteSession`/`onOpen` exists in App |
| **"Browse Files" (remote file-browse join-code on-ramp)** | PARTIAL — `handleBrowseFilesRemote` + `RemoteJoinCodeModal` exist in App, but the only Hub trigger is card-click (cap gate). No dedicated "Browse files" button on remote cards | **GAP: remote browse on-ramp button** | **MIGRATE or confirm** — verify card-click cap flow covers remote browse; if a dedicated button is expected, add it |
| **`RemoteSession` / `RemotePeerSessions` TYPE EXPORTS** | N/A — types, not UI | **COMPILE-BREAK: imported by remoteAdapter.ts, remoteSession.ts, App.tsx** | **RELOCATE types to `lib/remoteSession.ts`** before deleting the component file |

**Parity gaps the plan MUST close or defer-with-sign-off:**
1. **Per-session Kill** (DaemonManagerPanel) → migrate to card overflow menu (recommended).
2. **Remote "Open in browser"** (RemoteSessionsPanel) → confirm superseded by Phase 134 modal, or migrate.
3. **Remote "Browse files" on-ramp** → confirm covered by card-click cap flow, or add a card button.
4. **Type relocation** (`RemoteSession`/`RemotePeerSessions`) → mandatory, non-optional.

## Routing / Dead-Link Cleanup Inventory

All navigation is **internal tab-state** in `App.tsx` (no react-router / no URL routes; no keyboard
shortcuts point to these pages). Items to remove when NAV-02/03/04 land:

| Item | Location | Action |
|------|----------|--------|
| `DAEMON_MANAGER_TAB` const | App.tsx:88 | Remove |
| `REMOTE_SESSIONS_TAB` const | App.tsx:89 | Remove |
| `handleOpenDaemonManager` | App.tsx:992 | Remove |
| `handleOpenRemoteSessions` | App.tsx:1141 | Remove |
| DaemonManager poll effect | App.tsx:891-911 | Remove |
| Remote poll: gated on `REMOTE_SESSIONS_TAB.id` **OR** `HUB_TAB.id` | App.tsx:944-971 | **KEEP — Hub still needs remote data;** just drop the `REMOTE_SESSIONS_TAB` half of the condition |
| `<DaemonManagerPanel>` render branch | App.tsx:1357-1369 | Remove |
| `<RemoteSessionsPanel>` render branch | App.tsx:1478-1486 | Remove |
| Tab-type filter strings `'daemon-manager'`/`'remote-sessions'` | App.tsx:1517, 1556 | Remove from both filter lists |
| `Tab.type` union members | TabBar.tsx:8 | Remove `'daemon-manager'` and `'remote-sessions'` |
| Sidebar props `onOpenRemoteSessions`, `onOpenDaemonManager`, `onAdd` | Sidebar.tsx:15-22 + App.tsx:1326-1328 | Remove |
| Sidebar Remote/Sessions/New Session buttons + unused icon imports | Sidebar.tsx:67-101 (GlobeAltIcon/ServerStackIcon/PlusIcon) | Remove |
| `handleAddTab` (sidebar New Session) | App.tsx:737 | Remove **only if** no other caller; Hub uses `onNewSession` → `setShowNewSessionModal` directly. **Verify** `handleAddTab` is not used elsewhere before deleting |

**Note on remote data poll:** Do NOT delete the remote-sessions poll — the Hub depends on
`remotePeers` for its remote cards. Only remove the `REMOTE_SESSIONS_TAB`-active half of the
activation guard (App.tsx:946), leaving the `HUB_TAB`-active condition.

## Card Redesign Preservation Checklist (CARD-04)

The card must keep these behaviors while adding the new indicators + (already-present) Share button:

| Behavior | Where implemented | Preservation note |
|----------|-------------------|-------------------|
| Attention pulse | `.hub-card--attention` + `@media (prefers-reduced-motion)` guard, style.css:4939-4990 | Don't remove the class or the reduced-motion fallback |
| Float-to-top reorder | `useFLIPAnimation` + `sortSessionsForDisplay`, SessionCardGrid:76-126 | Pure layout; resizing cards must not break FLIP measurement (it reads `getBoundingClientRect`) |
| Mini-preview | `<MiniPreview>` ROW 6, SessionCard:431; `usePreviewPoller` 3s shared interval, HubPanel:65 | Keep as last row; don't change the poller |
| Grid density | `.hub__card-row { grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)) }`, style.css:4287 | New indicators must fit within `min-width:240px` / `max-width:360px` card (style.css:4296) |
| Responsive reflow | Same `auto-fill`/`minmax` grid | Adding a row or chip is fine; avoid forcing card width beyond 360px |
| Share button | `.hub-card__share`, SessionCard:418-428 (Phase 137) | Already present; CARD-04 ensures layout accommodates it cleanly |
| Card-click → modal (Phase 134) | article `onClick` with `.hub-card__share`/`.hub-card__open`/menu guards, SessionCard:248-258 | New interactive children (e.g. a Kill button) MUST add a `.closest()` guard + `e.stopPropagation()` (Pitfall 6 pattern) |

## Common Pitfalls

### Pitfall 1: Deleting RemoteSessionsPanel breaks the Hub's remote pipeline
**What goes wrong:** TS compile fails — `remoteAdapter.ts`/`remoteSession.ts`/`App.tsx` import
`RemoteSession`/`RemotePeerSessions` from the deleted file.
**Why it happens:** The panel file doubles as the type-definition module.
**How to avoid:** Relocate the two interfaces to `lib/remoteSession.ts` (or a new `lib/remoteTypes.ts`)
and update all importers BEFORE deleting the component. Run `tsc --noEmit` as the gate.
**Warning signs:** `Cannot find module '../components/RemoteSessionsPanel'` after deletion.

### Pitfall 2: New card child intercepts the card-click modal
**What goes wrong:** A migrated Kill button (or new indicator that's clickable) triggers the Phase 134
card-click modal.
**Why it happens:** The article `onClick` fires for any descendant without a guard.
**How to avoid:** Follow the existing Pitfall-6 pattern — add `e.stopPropagation()` on the child AND a
`if (target.closest('.hub-card__<x>')) return` line in the article `onClick` (SessionCard:248-258).

### Pitfall 3: Stale structural tests fail RED unexpectedly
**What goes wrong:** `Sidebar.test.tsx` asserts a "Sessions" button, a "New Session" button, and
`items.length === 6`; `App.hub.test.tsx` / `style.hub.test.ts` may assert the old structure.
**Why it happens:** Tests encode the pre-138 sidebar/header.
**How to avoid:** Treat these as **Wave 0** — update the assertions to the new
Home/Hub/Settings (3-item) sidebar and the removed `.hub__header` FIRST (RED → GREEN).
**Warning signs:** `expect(items.length).toBe(6)` and `button[aria-label="Sessions"]` assertions.

### Pitfall 4: Removing the remote poll starves the Hub
**What goes wrong:** Deleting the whole remote-sessions poll effect (because its tab is gone) empties
the Hub's remote cards.
**Why it happens:** The poll is shared — its activation guard is `REMOTE_SESSIONS_TAB || HUB_TAB`.
**How to avoid:** Keep the effect; only drop the `REMOTE_SESSIONS_TAB` half of the guard.

### Pitfall 5: Color-only connection indicator
**What goes wrong:** "Connected" shown only as a green dot — invisible to the colorblind owner.
**Why it happens:** Easy default.
**How to avoid:** Icon shape + text label (e.g. `LinkIcon` + "Connected" vs outline + "Available");
verify the chosen hexes at source, not by eye. [CITED: MEMORY.md user_colorblind]

## Code Examples

### Threading the connection prop (CARD-03)
```tsx
// App.tsx — already has remoteCapsCached: Set<string> (line 209) passed to HubPanel.
// HubPanel → SessionCardGrid → SessionCard: add a connectedRemoteIds prop.
// Source pattern mirrors existing attentionIds threading (SessionCardGrid:181, 254).

// In SessionCard, render (colorblind-safe, mirrors origin row at SessionCard:357):
{isRemote && (
  <span className="hub-card__conn">
    {isConnected
      ? (<><LinkIcon aria-hidden="true" /><span>Connected</span></>)
      : (<><GlobeAltIcon aria-hidden="true" /><span>Available</span></>)}
  </span>
)}
// And append to cardAriaLabel (SessionCard:180): `${isRemote ? (isConnected ? ', connected' : ', available') : ''}`
```

### Removing .hub__header (CARD-01)
```tsx
// Source: frontend/src/components/Hub/HubPanel.tsx:461-467 — DELETE this block entirely.
<div className="hub__header">
  <span className="hub__title">Hub</span>
  <button className="hub__new-session-btn" type="button" onClick={onNewSession}>New session</button>
</div>
// HubFilterBar's onNewSession button (HubFilterBar:135-141) remains the sole entry point.
// Also remove now-dead .hub__header / .hub__title / .hub__new-session-btn CSS in style.css if present.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Separate Sessions + Remote pages | Unified Hub grid (local + remote cards) | Phases 131-134 | 138 removes the now-redundant pages |
| Per-session controls on Sessions page | Per-card Share modal | Phase 137 | Web-share/browse parity already migrated |
| Two New Session entry points (sidebar + `.hub__header`) | Single `HubFilterBar` button | Phase 138 (this) | CARD-01/NAV-02 |

**Deprecated/outdated after this phase:**
- `DaemonManagerPanel.tsx`, `RemoteSessionsPanel.tsx` (component) — deleted.
- Sidebar `onOpenDaemonManager`/`onOpenRemoteSessions`/`onAdd` props — deleted.
- `Tab.type` members `'daemon-manager'`/`'remote-sessions'` — deleted.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `LinkIcon`/`BoltIcon` are appropriate "connected" glyphs | Standard Stack / Pattern 3 | Low — final iconography is Phase 140 UI-spec; text+shape already carries the signal |
| A2 | Phase 134 in-app interactive modal supersedes RemoteSessionsPanel's "Open in browser" | Parity Inventory | Medium — if browser-open is still wanted, it's a migrated gap, not a clean deletion. **Confirm with user/UAT** |
| A3 | Remote "Browse files" is covered by card-click cap flow (no dedicated button needed) | Parity Inventory | Medium — if a dedicated remote browse button is expected, must add. **Confirm** |
| A4 | Per-session Kill is acceptable to migrate to the card overflow menu | Parity Inventory | Low-Medium — placement is a UI decision; the capability (`handleCloseTab`) exists |
| A5 | `handleAddTab` has no caller besides the sidebar New Session button | Routing Cleanup | Low — verify with grep before deleting |
| A6 | No keyboard shortcut / deep link targets the deleted pages | Routing Cleanup | Low — grep found only internal tab-state nav; no router |

## Open Questions

1. **Does the in-app interactive modal (Phase 134) fully replace "Open remote session in browser"?**
   - What we know: card-click opens an in-app `HubModal` with a live terminal (remote via daemon WS proxy).
   - What's unclear: whether users still want a "pop out to browser" affordance for remote sessions.
   - Recommendation: confirm at plan/UAT time; if yes, migrate `handleOpenRemoteSession` to a card action; if no, document as superseded (A2).

2. **Where does per-session Kill live on the card?**
   - What we know: `handleCloseTab` (App) kills; the card has an overflow menu (`.hub-card__menu`).
   - Recommendation: add "Kill session" to the overflow menu (with confirmation), guarded against card-click.

3. **Is the per-peer "Unreachable / no shareable sessions" messaging worth preserving?**
   - Recommendation: likely defer (cosmetic) — merged-card model simply omits absent sessions. Get sign-off.

## Environment Availability

> Skipped — this phase has no external tool/service dependencies. It is in-repo TypeScript/CSS edits
> plus `npm test` / `tsc --noEmit`, both already part of the existing toolchain (package.json).

## Validation Architecture

> `workflow.nyquist_validation` key is absent in `.planning/config.json` → treat as **enabled**.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest ^4.1.0 (+ @testing-library, JSDOM) |
| Config file | `frontend/vite.config.ts` / vitest defaults (no standalone vitest.config) |
| Quick run command | `cd frontend && npx vitest run src/components/__tests__/Sidebar.test.tsx` (single file) |
| Full suite command | `cd frontend && npm test` (≈ `vitest run`, 107+ files, ~1750 tests as of Phase 137) |
| Type gate | `cd frontend && npx tsc --noEmit` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NAV-02 | No "New Session" sidebar item | unit | `npx vitest run src/components/__tests__/Sidebar.test.tsx` | ⚠️ exists, asserts OPPOSITE — Wave 0 update |
| NAV-03 | No Sessions page / DaemonManagerPanel route | unit | `npx vitest run src/components/__tests__/App.hub.test.tsx` | ⚠️ update |
| NAV-04 | No Remote page route | unit | `npx vitest run src/components/__tests__/App.hub.test.tsx` | ⚠️ update |
| NAV-05 | Sidebar = exactly Home/Hub/Settings (3 items) | unit | `npx vitest run src/components/__tests__/Sidebar.test.tsx` | ⚠️ currently asserts `items.length===6` — Wave 0 |
| CARD-01 | `.hub__header` + its New session button removed | unit/style | `npx vitest run src/components/__tests__/App.hub.test.tsx` (+ style.hub.test.ts) | ⚠️ update |
| CARD-02 | Each card shows local vs remote (icon+text) | unit | `npx vitest run src/components/Hub/__tests__/SessionCard.*.test.tsx` | ❌ Wave 0 (new origin assertion; SessionCard.share.test.tsx exists as a base) |
| CARD-03 | Remote card shows connected vs available, colorblind-safe | unit | new `SessionCard` test | ❌ Wave 0 |
| CARD-04 | Resized card preserves pulse/preview/grid/Share | unit/style | `npx vitest run src/components/__tests__/style.hub.test.ts` + SessionCardGrid tests | ⚠️/❌ assert preservation |

### Sampling Rate
- **Per task commit:** the single-file `npx vitest run <touched test>` + `tsc --noEmit`.
- **Per wave merge:** `cd frontend && npm test` (full vitest suite).
- **Phase gate:** full suite green + `tsc --noEmit` clean before `/gsd:verify-work`. Then live UAT
  (dev-browser) for the 3-item sidebar, sole New Session entry, and card indicators — verify
  colorblind-safe signals at the **hex/source** level, not by eye [CITED: MEMORY.md user_colorblind].

### Wave 0 Gaps
- [ ] `Sidebar.test.tsx` — rewrite to assert Home/Hub/Settings only, `items.length === 3`, no
      Sessions/Remote/New Session buttons (covers NAV-02/03/04/05).
- [ ] `App.hub.test.tsx` — remove DaemonManager/Remote route expectations; assert no `.hub__header`
      (CARD-01) and that HubFilterBar's button is the sole creation trigger.
- [ ] `style.hub.test.ts` — drop any `.hub__header`/`.hub__title`/`.hub__new-session-btn` assertions;
      add CARD-04 preservation assertions (`.hub__card-row` grid intact, `.hub-card--attention` intact).
- [ ] New `SessionCard` test (extend `SessionCard.share.test.tsx` or new file) — CARD-02 origin
      (Computer/Local vs Globe/host) and CARD-03 connected/available indicator (icon + text +
      aria-label), provenance-driven.
- [ ] Parity-migration tests — if Kill / remote-open / remote-browse migrate to the card, add unit
      tests for those affordances (guarded against the card-click modal).

## Security Domain

> `security_enforcement` not set to `false` → included. This phase introduces **no new network
> endpoints, auth paths, capability changes, or trust boundaries** — it deletes UI and adds
> presentational indicators. The cap model (Phase 137) is untouched.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface touched |
| V3 Session Management | no | No session-token handling changed |
| V4 Access Control | no | No new permission decisions; CARD-03 only *displays* existing cap state |
| V5 Input Validation | no | No new user input parsed (search input unchanged) |
| V6 Cryptography | no | None |
| V7 Error Handling/Logging | yes (minor) | Preserve existing remote-error/empty states where Hub already handles them |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cap token leaking into React state via new "connected" indicator | Information Disclosure | CARD-03 must read only the **boolean** `remoteCapsCached.has(id)`, never the token (tokens already live in the daemon's RemoteCapStore, not React — Phase 122 T-122-03-01). Do not surface or store the token. |
| Re-share affordance appearing on remote (unowned) cards | Elevation of Privilege | Share button already disabled on remote cards (SHARE-06 / D-13). Any migrated Kill/browse action on remote cards must respect ownership (remote sessions are not owned locally). |

## Sources

### Primary (HIGH confidence)
- `frontend/src/components/Sidebar.tsx` — nav structure (NAV-02..05 targets)
- `frontend/src/App.tsx` — routing, Tab consts, polls, panel wiring (lines cited inline)
- `frontend/src/components/Hub/HubPanel.tsx` — `.hub__header` (461-467), remote provenance (302), share modal mount
- `frontend/src/components/Hub/HubFilterBar.tsx` — canonical New Session button (135-141)
- `frontend/src/components/Hub/SessionCard.tsx` — origin row (357-369), Share button (418-428), card-click guards (248-258)
- `frontend/src/components/Hub/SessionCardGrid.tsx` — FLIP/float-to-top, grid render paths
- `frontend/src/components/DaemonManagerPanel.tsx` — Sessions-page feature inventory
- `frontend/src/components/RemoteSessionsPanel.tsx` — Remote-page inventory + exported types
- `frontend/src/lib/remoteAdapter.ts`, `lib/remoteSession.ts` — remote pipeline + type importers
- `frontend/src/style.css` — `.hub__card-row` (4287), `.hub-card` (4296), `.hub-card--attention` (4939)
- `frontend/src/wailsjs/go/main/App.d.ts` — active `SessionInfo` type (no isRemote/connected field → provenance/`remoteCapsCached` are the discriminators)
- `frontend/src/components/__tests__/Sidebar.test.tsx` — current assertions (Wave 0 source of truth)
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `137-CONTEXT.md`, `137-03-SUMMARY.md`
- `grep` (Go/cmd/internal) — confirmed NO Go/CLI coupling to these GUI panels

### Secondary (MEDIUM confidence)
- MEMORY.md: `user_colorblind`, `feedback_cross_surface_parity` — release norms

### Tertiary (LOW confidence)
- None — entire surface verified in-repo this session.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; versions verified in package.json + node_modules.
- Architecture / routing cleanup: HIGH — all call sites read directly from App.tsx/Sidebar/TabBar.
- Parity inventory: HIGH for what exists; MEDIUM for whether 3 gaps (Kill / remote-open / remote-browse) need migration vs defer — needs user/UAT confirmation (A2/A3/A4).
- Card indicators (CARD-02/03/04): HIGH — data already present; only threading + preservation needed.

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable in-repo surface; no fast-moving external deps)
