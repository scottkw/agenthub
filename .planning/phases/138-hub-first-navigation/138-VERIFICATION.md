---
phase: 138-hub-first-navigation
verified: 2026-06-20T23:25:00Z
status: human_needed
score: 8/8 must-haves verified
overrides_applied: 0
human_verification:
  - test: "3-item sidebar visible in live app"
    expected: "Sidebar shows exactly Home, Hub, Settings — no Sessions, Remote, or New Session entry"
    why_human: "DOM structure is verified by unit tests; live visual layout and responsive collapse behavior require dev-browser"
  - test: "HubFilterBar is the sole New Session entry point"
    expected: "No other button or affordance creates a new session; clicking HubFilterBar's button opens the new session modal"
    why_human: "Session creation is a live interaction with daemon state; unit tests cannot cover the full modal → creation path"
  - test: "Remote card shows 'Open in browser' and 'Browse files' in overflow menu"
    expected: "Overflow menu on a remote card shows both items; clicking 'Open in browser' opens the real peer URL in the system browser"
    why_human: "Requires a live connected remote peer; URL resolution and BrowserOpenURL cannot be exercised in JSDOM"
  - test: "Kill two-step confirm on a live local session"
    expected: "Overflow menu on a live local session shows 'Kill session'; clicking it once shows 'Confirm kill' / 'This will stop the session'; second click terminates the session"
    why_human: "Requires a running local session; daemon interaction and session termination cannot be verified without the daemon"
  - test: "Remote card does NOT show Kill session"
    expected: "Overflow menu on a remote card shows 'Open in browser' and 'Browse files' only — no Kill option"
    why_human: "Requires a live remote card in the Hub; unit tests cover isLocal guard at source level"
  - test: "Colorblind-safe card indicators render correctly with icon + text"
    expected: "Local cards show ComputerDesktopIcon + 'Local'; remote connected cards show LinkIcon + 'Connected'; remote available cards show GlobeAltIcon + 'Available' — verified at source level; dev-browser confirms visual rendering"
    why_human: "User is colorblind — visual rendering check in live app confirms icons render alongside text (hex-source already verified); no color-only signals"
  - test: "Attention pulse, mini-preview, and grid reflow are preserved"
    expected: "Hub cards still animate attention pulse, show mini-preview text, and reflow responsively at 240–360px grid density after new affordances were added"
    why_human: "CARD-04 visual preservation contract; animation and layout require a live browser (JSDOM has no layout engine)"
---

# Phase 138: Hub-First Navigation Verification Report

**Phase Goal:** Remove Sessions page, Remote page, sidebar New Session item; add local/remote (origin) and connected/available indicators on Hub session cards; collapse sidebar to Home/Hub/Settings. Migrate Remote/Sessions parity affordances (Open-in-browser, Browse-files, Kill) onto the Hub card overflow menu BEFORE deleting those pages (cross-surface parity is release-blocking).
**Verified:** 2026-06-20T23:25:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Sidebar renders exactly Home, Hub, Settings — no Sessions, Remote, or New Session items | VERIFIED | `Sidebar.tsx` imports only `HomeIcon`, `Cog6ToothIcon`, `Squares2X2Icon`; SidebarProps has no `onOpenRemoteSessions`/`onOpenDaemonManager`/`onAdd`; `Sidebar.test.tsx` `toBe(3)` + 3 null-checks pass (166/166 tests green) |
| 2 | DaemonManagerPanel.tsx and RemoteSessionsPanel.tsx are deleted and not imported anywhere | VERIFIED | Both files missing from filesystem; `grep -rn "DaemonManagerPanel\|RemoteSessionsPanel" frontend/src/` returns only comments (no import statements) |
| 3 | App.tsx has no DAEMON_MANAGER_TAB/REMOTE_SESSIONS_TAB consts, handlers, or render branches; Hub remote poll is PRESERVED gated on `HUB_TAB.id` | VERIFIED | `grep DAEMON_MANAGER_TAB App.tsx` → 0 matches; `grep REMOTE_SESSIONS_TAB App.tsx` → 0 matches; remote poll at line 901 has `if (activeId !== HUB_TAB.id) return` — no REMOTE_SESSIONS_TAB clause; `remotePeers` state retained |
| 4 | App.tsx wires onKill/onOpenInBrowser/onBrowseFiles/remotePeers into HubPanel | VERIFIED | Lines 1313-1316: `onKill={(id) => void handleCloseTab(id)}`, `onOpenInBrowser={handleOpenRemoteSession}`, `onBrowseFiles={handleBrowseFilesRemote}`, `remotePeers={remotePeers}` |
| 5 | Hub cards show provenance-based origin (CARD-02) with colorblind-safe icon+text | VERIFIED | `SessionCard.tsx` line 223: `const isLocal = isRemote !== undefined ? !isRemote : (!hostname || hostname === '')` (provenance priority); origin row: `ComputerDesktopIcon`+"Local" vs `GlobeAltIcon`+`{hostname}` — never color alone |
| 6 | Remote cards show colorblind-safe Connected/Available chip (CARD-03) | VERIFIED | Lines 475-485: `{isRemote && (<div className="hub-card__row2b">...)}` with `LinkIcon`+"Connected" / `GlobeAltIcon`+"Available"; `style.css` has `.hub-card__conn { color: var(--hub-text-muted) }` and `.hub-card__conn--connected { color: var(--hub-accent) }` — custom properties only, not raw hex |
| 7 | Card overflow Kill (two-step, local-only), Open-in-browser, Browse-files items are present and guarded (CARD-01/CARD-04) | VERIFIED | Line 415: `{isLocal && session.state !== 'stopped' &&` (Kill local-only guard — CR-02 fix); line 388: `{isRemote && (...)}` (Open-in-browser + Browse-files remote-only); line 517: `{isLocal && onOpenSession && session.state !== 'stopped' &&` (re-attach Open button local-only — WR-01 fix); all menu items call `e.stopPropagation()` |
| 8 | adaptRemoteSession carries `url` so "Open in browser" forwards the real peer URL (CR-01 fix) | VERIFIED | `remoteAdapter.ts` line 27: `url: session.url`; `AdaptedRemoteSessionInfo = SessionInfo & { url: string }` type exported; `SessionCard.tsx` line 229 reads `(session as { url?: string }).url ?? ''` — now resolves to real URL from adapter |

**Score:** 8/8 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/Sidebar.tsx` | 3-item sidebar (Home/Hub/Settings) | VERIFIED | Exactly 3 `.sidebar__item` buttons; no `ServerStackIcon`, `GlobeAltIcon`, `PlusIcon` |
| `frontend/src/components/Hub/SessionCard.tsx` | Kill+Open+Browse overflow items; CARD-02 origin; CARD-03 chip | VERIFIED | `KillConfirmItem` component defined; `isRemote &&` guards on remote items; `LinkIcon`+`GlobeAltIcon` chip rendered |
| `frontend/src/components/Hub/HubPanel.tsx` | Header removed; peer hint rendered | VERIFIED | `grep hub__header HubPanel.tsx` → 0 matches; `hub__peer-hint` renders from `remotePeers` filter |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | Threads isRemote/isConnected/onKill/onOpenInBrowser/onBrowseFiles | VERIFIED | Props declared; BOTH SessionCard invocations (lines 270-274, 319-323) pass all 5 props |
| `frontend/src/App.tsx` | Cleaned routing; HubPanel wired | VERIFIED | Contains `onKill={`, `onOpenInBrowser={`, `onBrowseFiles={`, `remotePeers={` |
| `frontend/src/lib/remoteAdapter.ts` | `url` field carried through adapter | VERIFIED | `url: session.url` at line 27; `AdaptedRemoteSessionInfo` type enforces it |
| `frontend/src/lib/remoteSession.ts` | `RemoteSession`/`RemotePeerSessions` type home | VERIFIED | Both interfaces exported from this file; no import from `RemoteSessionsPanel` |
| `frontend/src/style.css` | CARD-03 chip CSS; destructive menu CSS; hub__header removed (commented); peer-hint added | VERIFIED | `.hub-card__conn`, `.hub-card__conn--connected`, `.hub-card__conn-icon`, `.hub-card__menu-item--destructive`, `.hub__peer-hint` all present; `.hub__header` block inside `/* */` comment |
| `frontend/src/components/DaemonManagerPanel.tsx` | DELETED | VERIFIED | File does not exist |
| `frontend/src/components/RemoteSessionsPanel.tsx` | DELETED | VERIFIED | File does not exist |
| `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` | DELETED | VERIFIED | File does not exist |
| `frontend/src/components/TabBar.tsx` | `Tab.type` union excludes `'daemon-manager'`/`'remote-sessions'` | VERIFIED | `type?: 'terminal' \| 'welcome' \| 'settings' \| 'file-browser' \| 'hub'` — neither removed type present |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `App.tsx` remote poll | Hub remote cards | `activeId !== HUB_TAB.id` guard only | VERIFIED | Line 901; REMOTE_SESSIONS_TAB clause removed; poll retained |
| `App.tsx` | `HubPanel` parity handlers | `onKill`/`onOpenInBrowser`/`onBrowseFiles`/`remotePeers` | VERIFIED | Lines 1313-1316 |
| `HubPanel.tsx` | `SessionCardGrid.tsx` | `connectedRemoteIds` + `remoteIdSet` | VERIFIED | Lines 472-473; both Sets passed |
| `SessionCardGrid.tsx` | `SessionCard.tsx` | `isRemote={remoteIdSet?.has(s.id)}` in BOTH render paths | VERIFIED | Lines 270+319 (two invocations) |
| `adaptRemoteSession` | `SessionCard` "Open in browser" | `url: session.url` → `remoteUrl` → `onOpenInBrowser?.(remoteUrl)` | VERIFIED | CR-01 fix confirmed in both files |
| `KillConfirmItem` | card-click modal guard | `e.stopPropagation()` + `closest('.hub-card__menu')` | VERIFIED | stopPropagation in KillConfirmItem (line 101) + all new menu items |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `SessionCard.tsx` connection chip | `isConnected` boolean | `remoteCapsCached.has(session.id)` → `connectedRemoteIds` Set → card prop | Yes — Set populated from daemon cap store | FLOWING |
| `SessionCard.tsx` remote URL | `remoteUrl` | `adaptRemoteSession(peer, session).url = session.url` from peer API data | Yes — real peer URL | FLOWING |
| `HubPanel.tsx` peer hint | `remotePeers` | `setRemotePeers` populated by 30s poll via App.tsx remote effect | Yes — live API data | FLOWING |
| `SessionCard.tsx` origin | `isLocal` / `originText` | `remoteIdSet.has(session.id)` provenance | Yes — derived from real peer session list | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full vitest suite (1709 tests) | `cd frontend && npx vitest run` | 1709 passed, 0 failed, 104 test files | PASS |
| 5 core Wave 0 tests (166 tests) | `cd frontend && npx vitest run` *(5 spec files)* | 166 passed | PASS |
| TypeScript type check | `cd frontend && npx tsc --noEmit` | exit 0, no output | PASS |
| Deleted panels not imported | `grep -rn "DaemonManagerPanel\|RemoteSessionsPanel" frontend/src/` | Only comments in `remoteSession.ts` and `remoteSession.test.ts`; no import statements | PASS |
| Remote poll guard | `grep -n "activeId !== HUB_TAB.id" App.tsx` | Line 874 + line 901; no REMOTE_SESSIONS_TAB reference | PASS |
| Orphaned CSS removed | `grep -n "remote-panel" frontend/src/style.css` | 0 matches — WR-02 fix confirmed | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| NAV-02 | 138-01, 138-04 | Sidebar New Session item removed; sessions created from Hub | SATISFIED | No `aria-label="New Session"` in Sidebar.tsx; HubFilterBar is sole entry |
| NAV-03 | 138-01, 138-04 | Sessions sidebar page (DaemonManagerPanel) removed | SATISFIED | DaemonManagerPanel.tsx deleted; no import in App.tsx |
| NAV-04 | 138-01, 138-04 | Remote sidebar page removed | SATISFIED | RemoteSessionsPanel.tsx deleted; no import in App.tsx |
| NAV-05 | 138-01, 138-04 | Sidebar contains exactly Home / Hub / Settings | SATISFIED | Sidebar.tsx renders 3 items; `toBe(3)` test passes |
| CARD-01 | 138-03 | `.hub__header` removed; HubFilterBar is sole creation entry | SATISFIED | `grep hub__header HubPanel.tsx` → 0; hub__header CSS commented out |
| CARD-02 | 138-02 | Each card indicates local or remote origin | SATISFIED | Provenance-based `isLocal = !isRemote`; icon+text in origin row |
| CARD-03 | 138-02 | Remote cards indicate connected/available state, colorblind-safe | SATISFIED | `LinkIcon`+"Connected" / `GlobeAltIcon`+"Available"; color via `var(--hub-accent)` only |
| CARD-04 | 138-02, 138-03 | Card accommodates Share + indicators; preserves attention/preview/grid | SATISFIED (automated portion) | `.hub__card-row`, `.hub-card--attention`, `240px` anti-regression tests pass; visual preservation requires human UAT |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `SessionCard.tsx` | 229 | `as { url?: string }` cast (WR-03 residual) | Info | Type-safety smell; harmless at runtime since `adaptRemoteSession` now guarantees the field. Noted in code review as info-level; not a blocker |
| `SessionCard.tsx` | 223 | `const isLocal = isRemote !== undefined ? !isRemote : (!hostname...)` — fallback hostname-heuristic | Info | IN-03 residual: dual discriminators for "is remote". They agree for all current sessions. Flagged in code review |
| `App.tsx` | 1001-1003 | `handleOpenRemoteSession` calls `BrowserOpenURL(url)` without empty-URL guard | Info | If peer URL is empty string, `BrowserOpenURL('')` is called. Mitigated by adapter always carrying real peer URL; guard would add safety. Not a blocker |
| `style.css` | 3965-3999 | `.hub__header` CSS left as commented-out dead code | Info | Plan explicitly accepted this ("safe to delete in a future cleanup pass"); WR-02 `.remote-panel*` CSS was fully removed |

No TBD, FIXME, or XXX markers found in any phase-modified file.

---

### Human Verification Required

#### 1. 3-Item Sidebar Visual Layout

**Test:** Open the app in dev-browser. Inspect the left sidebar.
**Expected:** Exactly three nav buttons are visible (Home, Hub, Settings). No Sessions, Remote, or New Session button appears anywhere in the sidebar. Collapse toggle works and all three items remain.
**Why human:** Unit tests confirm the DOM structure; live visual layout and collapse animation require a browser.

#### 2. HubFilterBar is Sole Session Creation Entry Point

**Test:** With a live daemon running, confirm there is no other button or affordance that creates a new session other than the HubFilterBar "New Session" button.
**Expected:** Clicking "New Session" in the Hub filter bar opens the new session modal; no other creation path exists in the GUI.
**Why human:** End-to-end session creation requires daemon interaction; unit tests only assert `setShowNewSessionModal(true)` at source level.

#### 3. Remote Card "Open in Browser" Forwards Real URL

**Test:** With at least one reachable remote peer connected, open its session card overflow menu and click "Open in browser".
**Expected:** The system browser opens the peer session URL (not an empty tab or error).
**Why human:** Requires a live remote peer with a URL; `BrowserOpenURL` is a Wails runtime call that cannot be tested in JSDOM.

#### 4. Kill Two-Step Confirm on Live Local Session

**Test:** With a running local session, open its overflow menu. Click "Kill session".
**Expected:** Label changes to "Confirm kill" with subtext "This will stop the session". Second click terminates the session.
**Why human:** Requires a running local daemon session; session termination involves daemon RPC.

#### 5. Remote Card Has No Kill Option

**Test:** With a remote card visible in the Hub, open its overflow menu.
**Expected:** Only "Open in browser" and "Browse files" are shown — no "Kill session" item.
**Why human:** Requires a live remote card to confirm at runtime that the `isLocal` guard works end-to-end.

#### 6. Colorblind-Safe Visual Rendering (Source-Level Verified; Visual Confirmation Needed)

**Test:** In the Hub, observe a remote connected card and a remote available card.
**Expected:** Connected card shows a chip with a link/chain icon AND the text "Connected". Available card shows a globe icon AND the text "Available". No status relies on color alone.
**Why human:** Hex source confirmed (`var(--hub-accent)` / `var(--hub-text-muted)`; icon + text present in code). Visual rendering confirms icons actually appear alongside text labels.

#### 7. Attention Pulse, Mini-Preview, and Grid Reflow Preserved (CARD-04)

**Test:** Verify Hub cards still animate attention pulse on errored sessions, show mini-preview text, and reflow at 240–360px grid density.
**Expected:** No regression to existing Hub card behavior after the new affordances were added.
**Why human:** JSDOM has no layout engine; animation and CSS grid density require a live browser rendering.

---

### Gaps Summary

No automated gaps found. All 8 must-have truths are VERIFIED in the codebase. The phase's code-review criticals (CR-01, CR-02) and the major warning (WR-01, WR-02) are confirmed fixed in the live code:

- **CR-01 FIXED:** `adaptRemoteSession` carries `url: session.url`; `AdaptedRemoteSessionInfo` type enforces it
- **CR-02 FIXED:** `KillConfirmItem` gated on `isLocal && session.state !== 'stopped'` (line 415)
- **WR-01 FIXED:** Re-attach Open button gated on `isLocal && onOpenSession && session.state !== 'stopped'` (line 517)
- **WR-02 FIXED:** `.remote-panel*` CSS block completely removed from `style.css`

Remaining info-level items (IN-01/IN-02/IN-03/WR-03) are not blockers and do not affect goal achievement.

Status is `human_needed` because CARD-04 visual preservation, live remote card behavior, Kill confirm, and BrowserOpenURL all require dev-browser UAT with a live daemon and (for remote tests) a reachable peer.

---

_Verified: 2026-06-20T23:25:00Z_
_Verifier: Claude (gsd-verifier)_
