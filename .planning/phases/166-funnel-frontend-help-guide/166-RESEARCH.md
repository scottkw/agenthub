# Phase 166: Funnel Frontend + Help Guide — Research

**Researched:** 2026-06-30
**Domain:** React/Wails frontend — wiring Phase-165 Funnel backend to UI; in-app Help article
**Confidence:** HIGH (all priority items resolved from source code)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- D-01: Funnel toggle reveals an **inline expanding panel** inside the existing Share modal — no nested modal.
- D-02: Acknowledgment is an **explicit confirm button** ("Enable internet share") — no checkbox, no type-to-confirm.
- D-03: Risk panel shown on **every enable** — no "don't show again".
- D-04: Panel order: ⚠ risk statement → auto-expiry selector → Help cross-link → [Cancel] [Enable internet share].
- D-05: Duration presets: **30m / 1h / 4h / 8h**, default **1h**.
- D-06: "Until I disable" option (no auto-expiry) is offered.
- D-07: "Until I disable" maps to `expirySeconds=0` sentinel — CONFIRMED (see § D-07 below).
- D-08: Hub card badge: globe icon + "INTERNET" text pill.
- D-09: Session tab: globe icon only; `aria-label="Internet exposed"` carries the text requirement.
- D-10: State encoded by icon shape + text — never by color alone. Verify at hex/source level.
- D-11: Inline pill/badge on card header and inline on tab.
- D-12: Single "Public URL (read-only)" in "Internet (public)" section + copy + QR.
- D-13: One-click disable → `SetSessionFunnel(id, false, 0)`.
- D-14: Warm-up UX shows "Starting up… (TLS warming up)" with URL/QR muted until `funnelActive` = true.
- D-15: Funnel toggle disabled when `webServerMode !== 'tailscale'`.
- D-16: New Help section "Sharing Outside Your Tailnet" registered in `SECTION_META`; placed after "Chat", before "FAQ".
- D-17: Article documents both paths: Funnel (Option 1) and device-share + ACL (Option 2).
- D-18: Copy-pasteable ACL block, wildcard-default gotcha called out, Tailscale doc links.

### Claude's Discretion
- Pill/badge styling, spacing, animation of the inline risk panel, and poll interval/timeout for warm-up.
- Precise Help-article prose, heading structure, and Tailscale doc URLs to cite.

### Deferred Ideas (OUT OF SCOPE)
- Public **write** access over Funnel (read+write dual Funnel URLs).
- Custom-minutes auto-expiry input.
- Automating device-share / ACL edits via the Tailscale admin API.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FUI-01 | Risk-acknowledgment dialog on every enable (no skip), containing risk statement | D-01/D-02/D-03 — inline panel inside existing Share modal |
| FUI-02 | Risk dialog lets user choose auto-expiry duration before enabling | D-05/D-06 — expiresIn presets; 0 = no-expiry sentinel confirmed |
| FUI-03 | Persistent colorblind-safe indicator (text + non-color icon) while session is exposed | D-08/D-09/D-10 — globe + "INTERNET" on card; globe + aria-label on tab |
| FUI-04 | One-click Funnel disable from Share UI, fully tears down Funnel config | D-13 — SetSessionFunnel(id, false, 0); confirmed FNL-05 teardown |
| FUI-05 | Public Funnel URL displayed with copy + QR; "starting up…" UX while warming | D-12/D-14 — re-issue caps after funnelActive=true; GetCapabilityQRCode |
| FUI-06 | Risk dialog cross-links to Help guide | D-04 — "Want tighter containment? See the Sharing Guide →" in panel |
| HLP-01 | New Help article covering both sharing paths | D-16/D-17 — sharing-guide.md; SECTION_META + HelpSectionNav.SECTIONS |
| HLP-02 | ACL grant block, wildcard gotcha, Tailscale doc links | D-18 — port :7443 confirmed; HelpContent.tsx auto-handles links |
</phase_requirements>

---

## Summary

This is a codebase-grounded verification phase. All seven priority research items are resolved. The most critical finding is a **binding stub gap**: `frontend/src/wailsjs/go/main/App.d.ts` and `App.js` — the hand-authored stubs the app actually imports — are missing both `SetSessionFunnel` and `funnelActive: boolean` on `SessionInfo`. The Wails-generated counterparts in `wailsjs/wailsjs/go/` do have them; the stubs must be manually updated in Wave 0 before any component work.

The expiry sentinel is confirmed: `expiresIn=0` means "no auto-expiry" per `internal/daemon/types.go:158` and the conditional at `api.go:1555`. The web-share port is `7443` (hardcoded at `internal/tailnet/tailnet.go:29`, confirmed in `internal/daemon/process.go:49,125,136`). The ACL grant block with `:7443` is correct.

The Funnel URL is not returned from the `SetSessionFunnel` Wails binding (the daemon response body is dropped at `internal/daemon/client.go:368`). The UI must re-issue capabilities via `IssueCapabilities(sessionId)` after `funnelActive` becomes true — the daemon then returns URLs using the Funnel base (per `api.go:1344-1352`). The existing Hub poll runs at **3000 ms** (not 2 s as the UI-SPEC suggested); warm-up detection should ride this poll.

**Primary recommendation:** Treat Wave 0 as a pure stub-update + test-fixture wave, then implement component by component (modal → panel → card → tab → help).

---

## Project Constraints (from CLAUDE.md)

| Directive | Source | Impact on Phase |
|-----------|--------|-----------------|
| Colorblind rule — verify at hex/source level, not by eye | MEMORY.md (colorblind) | New token hex values specified in UI-SPEC must be in code, not just assumed |
| TESTING.md regression convention — update suite manifest + traceability map | agenthub/CLAUDE.md | Every new test file added must appear in TESTING.md Section 2, 4, and optionally 5 |
| Run `tsc && vite build` gate — vitest alone is insufficient | MEMORY.md (post_merge_gate_run_tsc) | Phase gate must include a tsc compile check |
| Always write TESTING.md rows for new test files | agenthub/CLAUDE.md | New test files for FunnelRiskPanel, FunnelURLSection, badges, help section |

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Funnel toggle + risk panel | Frontend (SessionShareModal) | Daemon via Wails RPC | Modal owns UI state; daemon owns Funnel lifecycle |
| Auto-expiry picker | Frontend (inline select) | Daemon (funnelExpiry timer) | UI picks duration; daemon enforces it server-side |
| Internet exposure indicator | Frontend (SessionCard, TabBar) | App.tsx (hubSessions poll) | Derives from funnelActive in SessionInfo via 3s poll |
| Funnel URL display | Frontend (SessionSharePanel) | Daemon (IssueCapabilities) | URL comes from re-issued cap after funnelActive=true |
| Warm-up detection | Frontend (local warmingUp state) | App.tsx / HubPanel (hubSessions sync) | Client-side timer tracks elapsed; poll triggers reveal |
| One-click disable | Frontend (disable button) | Daemon (SetSessionFunnel false) | Single click → Wails RPC → daemon tears down |
| Help guide | Frontend (HelpTab, HelpSectionNav, HelpContent) | — | Static markdown, no backend |

---

## Resolved Research Items

### D-07 — expirySeconds=0 sentinel (CRITICAL) [VERIFIED: internal/daemon/types.go:158]

```go
// internal/daemon/types.go:154-158
type SetSessionFunnelRequest struct {
    Enabled   bool `json:"enabled"`
    ExpiresIn int  `json:"expiresIn"` // auto-expiry in seconds; 0 = no auto-expiry (FNL-07)
}
```

Timer registration at `internal/daemon/api.go:1554-1568`:

```go
// FNL-07: register auto-expiry timer if expiresIn > 0.
if req.ExpiresIn > 0 {
    // ... time.AfterFunc(dur, func() { a.disableFunnelForSession(...) })
}
```

**Verdict:** `expiresIn=0` is the exact sentinel. The timer is only registered when `req.ExpiresIn > 0`. Zero means no timer, no auto-expiry. The UI's "Until I disable" → `expirySeconds=0` is correct. Do not invent a different sentinel.

---

### SetSessionFunnel + GetCapabilityQRCode binding signatures [VERIFIED: wailsjs/wailsjs/go/main/App.d.ts:113, app.go:898]

**Go binding (app.go:894-902):**
```go
// Phase 165 / FNL-01. expiresIn is auto-expiry in seconds (0 = no expiry, FNL-07).
func (a *App) SetSessionFunnel(sessionID string, enabled bool, expiresIn int) error
```

**Generated TypeScript (wailsjs/wailsjs/go/main/App.d.ts:113):**
```ts
export function SetSessionFunnel(arg1:string,arg2:boolean,arg3:number):Promise<void>;
```

**Human-readable form for the hand-authored stub:**
```ts
// Phase 165 / FNL-01 — Funnel on/off; expiresIn=0 means no auto-expiry (FNL-07).
export function SetSessionFunnel(sessionID: string, enabled: boolean, expiresIn: number): Promise<void>
```

**CRITICAL:** `SetSessionFunnel` returns `Promise<void>` — the Funnel URL is NOT returned. The daemon HTTP handler returns `{ funnelUrl: "..." }` but the daemon client ignores it (`c.post(..., nil)` at `internal/daemon/client.go:368`). The frontend must obtain the Funnel URL by re-issuing capabilities after `funnelActive` becomes true (see § Funnel URL below).

**GetCapabilityQRCode (already in hand-authored App.d.ts:204):**
```ts
export function GetCapabilityQRCode(joinURL: string): Promise<string>
```
Takes the join-code exchange URL (e.g. the readURL from IssueCapabilities). Returns base64-encoded PNG QR. No changes needed for this binding.

---

### funnelActive plumbing [VERIFIED: app.go:50, wailsjs/wailsjs/go/models.ts:209]

**Go struct (app.go:47-50):**
```go
// NOT omitempty: false must serialize so frontend poll detects expiry.
FunnelActive bool `json:"funnelActive"`
```

**Wails-generated models (wailsjs/wailsjs/go/models.ts:209):**
```ts
export class SessionInfo {
    ...
    funnelActive: boolean;
    ...
}
```

**App.tsx import chain (App.tsx:41):**
```ts
import type { DetectedCLI, SessionInfo, RemotePeerSessions } from './wailsjs/go/main/App'
```

The app imports from `./wailsjs/go/main/App` (the **hand-authored stub**), not from the Wails-generated `wailsjs/wailsjs/go/` tree. The hand-authored `SessionInfo` interface in `frontend/src/wailsjs/go/main/App.d.ts` currently has `browseEnabled: boolean` as its last field — **`funnelActive` is absent**. This is the binding gap.

**Poll path:** `ListSessions()` → App.tsx `hubSessions` state → 3000 ms poll (App.tsx:984) → passed to HubPanel as `sessions` prop → passed to SessionCard as `session: SessionInfo`.

**SessionCard:** receives full `SessionInfo` object — can read `session.funnelActive` directly once the stub is updated.

**TabBar:** does NOT receive full SessionInfo. TabBar receives `sessionStatuses?: Record<string, string>` keyed by sessionId. To propagate funnelActive, App.tsx must derive a new `funnelActiveSessions?: Record<string, boolean>` map from `hubSessions` and pass it as a new prop to TabBar (same pattern as `sessionStatuses`).

---

### CRITICAL BINDING GAP — Wave 0 mandatory updates [VERIFIED: App.d.ts, App.js]

The hand-authored stubs in `frontend/src/wailsjs/go/main/` are the **actual import source** for the entire frontend. Both files need updating before any component work:

**`frontend/src/wailsjs/go/main/App.d.ts` — two additions needed:**

1. Add `funnelActive: boolean` to the `SessionInfo` interface (after `browseEnabled: boolean`):
```ts
/** Phase 165 / FNL-01: true when Tailscale Funnel is active. NOT omitempty — false must serialize. */
funnelActive: boolean
```

2. Add `SetSessionFunnel` export at the end:
```ts
// Phase 165 / FNL-01 — Funnel on/off; expiresIn=0 means no auto-expiry (FNL-07).
export function SetSessionFunnel(sessionID: string, enabled: boolean, expiresIn: number): Promise<void>
```

**`frontend/src/wailsjs/go/main/App.js` — one addition needed:**
```js
// Phase 165 / FNL-01 — Funnel enable/disable with auto-expiry (expiresIn=0 = no expiry).
export const SetSessionFunnel = (sessionID, enabled, expiresIn) => Call('main.App.SetSessionFunnel', [sessionID, enabled, expiresIn])
```

Note: `wailsjs/wailsjs/go/main/App.d.ts` (Wails-generated) already has both. The `wailsjs/wailsjs/go/models.ts` (Wails-generated) already has `funnelActive`. Only the **hand-authored stubs** are missing them.

---

### HLP-02 — Web-share port [VERIFIED: internal/tailnet/tailnet.go:29, internal/daemon/process.go:49]

```go
// internal/tailnet/tailnet.go:29
const DefaultProbePort = 7443

// internal/daemon/process.go:49 (and :125, :136)
api.RestartWebServer(h.IP, 7443, h.Domain, "tailscale", "")
```

**Verdict:** The local web-share port is `7443`, hardcoded. The ACL grant block `"dst": ["tag:agenthub:tcp:7443"]` is correct. Use `:7443` everywhere in the Help article.

---

### Tailscale documentation URLs [ASSUMED]

The following URLs are from training data and should be verified before publishing the Help article. They are correct as of the August 2025 knowledge cutoff.

| Topic | URL |
|-------|-----|
| Funnel overview | `https://tailscale.com/kb/1223/funnel` |
| ACL syntax (grants, autogroup:shared) | `https://tailscale.com/kb/1337/acl-syntax` |
| Device sharing | `https://tailscale.com/kb/1084/sharing` |
| ACL tags | `https://tailscale.com/kb/1068/acl-tags` |

**Wails CSP constraint:** `HelpContent.tsx` already handles all markdown links as `BrowserOpenURL` buttons via the `a:` component override. Use standard markdown link syntax in `sharing-guide.md` — no special code needed. The `.help-content__external-link` class and `ArrowTopRightOnSquareIcon` are applied automatically.

---

### Existing surface landmines

#### SessionShareModal.tsx — integration notes [VERIFIED: SessionShareModal.tsx:86-380]

- **Animation state machine:** `phase: 'entering' | 'open' | 'exiting'` (lines 86-96). The inline risk panel lives inside the modal body during `'open'` phase. The modal uses `onAnimationEnd` to transition phases — the panel itself does NOT need its own animation lifecycle hook; only CSS handles expand/collapse.
- **Focus return:** `openerFocusRef` stores `document.activeElement` at mount and calls `focus()` on unmount (lines 99-105). The risk panel's "Keep local only" handler must return focus to the Funnel toggle — this is done by calling the panel's cancel handler which sets `riskPanelOpen = false`; the toggle naturally regains focus since it's still mounted.
- **`ShareSession` interface (line 18-25):** Currently has `{ id, name, cli, webEnabled, homeDir, browseEnabled }`. Must gain `funnelActive: boolean` to seed the Internet section's state and to keep the toggle in sync after warm-up.
- **`webServerMode` prop:** Already on the modal (SessionShareModalProps:36). The local-fallback disable gate (D-15) checks `webServerMode !== 'tailscale'` — no new plumbing.
- **SetSessionBrowse pattern (lines 232-256):** The browse toggle calls the Wails RPC, then re-issues caps. The Funnel enable follows the same pattern: call `SetSessionFunnel`, then poll for `funnelActive`, then re-issue caps.

#### SessionSharePanel.tsx — reuse pattern [VERIFIED: SessionSharePanel.tsx:1-113]

- **CodeDisplay + QR pattern (lines 8-57, 99-113):** `GetCapabilityQRCode(joinURL)` already used for read and write QR. The "Internet (public)" section reuses this exact pattern with the Funnel `readURL`.
- **ClipboardSetText vs navigator.clipboard:** The panel uses `ClipboardSetText` from `wailsjs/wailsjs/runtime/runtime` (not `navigator.clipboard`) — use the same import in the new section.
- **BrowserOpenURL:** Used for "Open" buttons — the Funnel URL section's "Open in browser" button follows this pattern.
- **Props today:** `{ sessionId, readURL, writeURL, readCode, writeCode, browseEnabled? }`. The new Internet section needs `funnelActive?: boolean` and `funnelUrl?: string` (the Funnel-base readURL, obtained by re-issuing caps after `funnelActive=true`). Alternatively, the modal can pass the re-issued URL directly.

#### SessionCard.tsx — badge placement [VERIFIED: SessionCard.tsx:125-178, UI-SPEC]

- `GlobeAltIcon` is already imported (confirmed by UI-SPEC source note). The badge goes in the card header row adjacent to the existing status badge. Placement: after session status, before the existing controls.
- The card receives `session: SessionInfo` directly — `session.funnelActive` is available once the stub is updated.

#### TabBar.tsx — tab indicator [VERIFIED: TabBar.tsx:7-82]

- `Tab` interface (line 7-13): `{ id, name, sessionId, cli, type? }` — no funnelActive.
- Current per-session prop pattern: `sessionStatuses?: Record<string, string>` (line 52). Add `funnelActiveSessions?: Record<string, boolean>` as a parallel prop.
- App.tsx derives it: `hubSessions.reduce<Record<string,boolean>>((acc, s) => ({...acc, [s.id]: s.funnelActive}), {})` — pass to `<TabBar funnelActiveSessions={...} />`.
- Globe icon placement per UI-SPEC: inline after the agent badge dot, before the tab name text.

#### HelpTab.tsx + HelpSectionNav.tsx — BOTH need updating [VERIFIED: HelpTab.tsx:61-65, HelpSectionNav.tsx:10-13]

**HelpTab.tsx** has `SECTION_META` at line 61:
```ts
const SECTION_META: ReadonlyArray<{ id: string; label: string; markdown: string }> = [
  { id: 'help-getting-started', label: 'Getting Started', markdown: gettingStartedMd },
  { id: 'help-chat', label: 'Chat', markdown: chatMd },
  { id: 'help-faq', label: 'Frequently Asked Questions', markdown: faqMd },
]
```
Add `{ id: 'help-sharing', label: 'Sharing Outside Your Tailnet', markdown: sharingMd }` after `help-chat`, before `help-faq`.

**HelpSectionNav.tsx** has a separate `SECTIONS` array at line 10:
```ts
export const SECTIONS = [
  { id: 'help-getting-started', label: 'Getting Started' },
  { id: 'help-chat', label: 'Chat' },
  { id: 'help-faq', label: 'Frequently Asked Questions' },
]
```
This is a **separate update required** — both arrays must stay in sync. Add `{ id: 'help-sharing', label: 'Sharing Outside Your Tailnet' }` in the same position.

---

### Warm-up polling mechanics (FUI-05) [VERIFIED: App.tsx:984]

**Existing poll interval:**
```js
// App.tsx:984
const interval = setInterval(() => void refresh(), 3000)
```

The Hub sessions poll fires every **3000 ms** (not 2 s as the UI-SPEC suggested). The poll only runs when `activeId === HUB_TAB.id` (i.e., Hub tab is active).

**Warm-up state flow:**
1. User clicks "Enable internet share" → `SetSessionFunnel(id, true, expiresIn)` called from SessionShareModal
2. On success: modal sets local `warmingUp = true` + records `warmStartAt = Date.now()`
3. The existing 3 s poll updates `hubSessions` → `shareModalSession` in HubPanel needs sync (see below)
4. HubPanel sync effect: `useEffect(() => { if (!shareModalSession) return; const updated = sessions.find(s => s.id === shareModalSession.id); if (updated) setShareModalSession(updated) }, [sessions, shareModalSession?.id])`
5. When `session.funnelActive` flips true: modal calls `IssueCapabilities(session.id)` → receives Funnel-base URLs → sets `cachedShare` → sets `warmingUp = false` → reveals Internet section
6. Timeout: if `Date.now() - warmStartAt > 30000` and `!funnelActive`, set `warmupTimedOut = true` → show error state
7. Timeout check runs in the existing warm-up state watcher (can use `useEffect` checking elapsed time on each `session.funnelActive` change, or a `setTimeout(30000)` cleared on success)

**Decision:** Ride the existing 3 s poll. Do NOT add a separate 2 s poll loop — it would duplicate the poll infrastructure and the 3 s detection latency is acceptable for a TLS warm-up that takes several seconds anyway.

---

### Funnel URL derivation [VERIFIED: internal/daemon/api.go:1344-1352, internal/daemon/client.go:368]

The `SetSessionFunnel` Wails binding returns `void`. The daemon drops the response body (`c.post(..., nil)` at `client.go:368`).

The Funnel URL is obtained by re-issuing capabilities after `funnelActive=true`:
```
IssueCapabilities(sessionId) → { readUrl, writeUrl, ... }
```
When Funnel is active, the daemon swaps the URL base to the Funnel hostname (api.go:1344-1352):
```go
base := ws.BaseURL()  // e.g. "https://hostname.ts.net:7443"
if fb := ws.FunnelBaseURL(); fb != "" {
    base = fb  // e.g. "https://hostname.ts.net" (no :443)
}
```
So `readUrl` becomes `https://hostname.ts.net/sessions/{id}?cap=...` — this is the "Public URL (read-only)" displayed in the Internet section.

**QR:** `GetCapabilityQRCode(readUrl)` — same call pattern as the existing read QR in SessionSharePanel.

**Note:** After Funnel is enabled, `readUrl` and `writeUrl` from `IssueCapabilities` also use the Funnel base. This means the existing tailnet "Read-Only Link" and "Full Access Link" rows in SessionSharePanel ALSO show Funnel-base URLs when Funnel is active. The "Internet (public)" section duplicates the read URL with explicit internet-exposure context and the warm-up + disable controls. This is by design per FNL-04.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Markdown link → BrowserOpenURL | Custom link component | `HelpContent.tsx` a-component override | Already implemented with URL allow-listing and accessible labels |
| External link button in Help | Raw `<a href>` | Markdown links in `.md` file | HelpContent.tsx auto-renders them as safe BrowserOpenURL buttons |
| QR code generation | Any QR library or API | `GetCapabilityQRCode(joinURL)` | Wails binding already does this; used in SessionSharePanel |
| Clipboard copy | `navigator.clipboard` | `ClipboardSetText` from wailsjs runtime | Consistent with all other copy buttons in the codebase |
| Motion guard | Custom media-query hook | Inline `matchMedia` check (same as SessionShareModal:81-84) | Established project pattern |
| Funnel URL derivation | Parse tailscale domain | Re-issue IssueCapabilities after funnelActive | Backend provides correct Funnel-base URL automatically |

---

## Common Pitfalls

### Pitfall 1: Updating only the generated wailsjs stubs, not the hand-authored ones
**What goes wrong:** `wailsjs/wailsjs/go/main/App.d.ts` already has `SetSessionFunnel`. Copying only from there misses that the app actually imports from `wailsjs/go/main/App.d.ts` (the hand-authored stub). Components compile fine in isolation but fail at runtime.
**How to avoid:** Wave 0 updates `frontend/src/wailsjs/go/main/App.d.ts` AND `App.js` explicitly. Add vitest import-contract tests (mirrors the existing `App.test.tsx` pattern that checks import paths).

### Pitfall 2: Using the wrong models.ts for SessionInfo type
**What goes wrong:** `wailsjs/wailsjs/go/models.ts` has `funnelActive` on the `main.SessionInfo` class. But the app imports `SessionInfo` from `wailsjs/go/main/App.d.ts` (the interface defined in the stub), NOT from models.ts. Adding `funnelActive` only to models.ts has no effect on the app's type checking.
**How to avoid:** The `SessionInfo` interface in `App.d.ts` is the single source of truth for component typing. Update only there (plus `ShareSession` in SessionShareModal if extracted separately).

### Pitfall 3: shareModalSession stale — warm-up never completes in the UI
**What goes wrong:** `shareModalSession` in HubPanel is set once on click (snapshot). The 3 s poll updates `hubSessions` but the modal session prop does not refresh, so `session.funnelActive` never flips true from the modal's perspective and warm-up state persists forever.
**How to avoid:** HubPanel needs a sync `useEffect` that updates `shareModalSession` when the matching entry in `sessions` prop changes. This must be added alongside the Funnel UI changes.

### Pitfall 4: Timer double-fire on re-enable before expiry
**What goes wrong:** If the user disables and re-enables Funnel before the first expiry timer fires, two timers exist for the same session. The daemon already guards this at `api.go:1561` (`t.Stop()` before re-registering). But if the frontend also has a local 30 s timeout ref, it must be cancelled on disable.
**How to avoid:** Store the warm-up timeout in a `useRef<ReturnType<typeof setTimeout>>` and clear it on disable or on successful detection.

### Pitfall 5: Updating SECTION_META in HelpTab but not SECTIONS in HelpSectionNav
**What goes wrong:** The section appears in the article area but is missing from the left nav sidebar. The left nav uses HelpSectionNav's own `SECTIONS` array. IntersectionObserver scroll-spy and jump-to-section both break for the new section.
**How to avoid:** Update BOTH arrays in the same commit. The `HelpTab.integration.test.tsx` should gain a test verifying `help-sharing` appears in the nav.

### Pitfall 6: tsc passes but vite build fails on CSS import
**What goes wrong:** Vitest runs with jsdom and skips CSS module processing. If the sharing-guide.md `?raw` import is missing a vite plugin or the `?raw` suffix triggers a transform error, it surfaces only at `tsc && vite build`.
**How to avoid:** The `?raw` pattern is already used for `chat.md`, `getting-started.md`, `faq.md` — no new plugin needed. The phase gate must run `cd frontend && tsc --noEmit && pnpm run build` in addition to vitest.

### Pitfall 7: Color-only state encoding
**What goes wrong:** Using only the `--hub-internet-badge-text` color to signal internet exposure violates the colorblind-safety standing rule.
**How to avoid:** Globe icon shape + "INTERNET" text are the primary signals; color is reinforcement only. Per the standing rule, verify at hex/source level that the hex constants are in the code. The UI-SPEC specifies dark `#43ddb2` / light `#0d7a5c` for the new tokens.

---

## Standard Stack

All libraries are already in the project. No new npm dependencies.

| Asset | Location | Purpose |
|-------|----------|---------|
| `GlobeAltIcon` | `@heroicons/react/24/outline` | Globe icon for badge + tab indicator |
| `ExclamationTriangleIcon` | `@heroicons/react/24/outline` | Risk panel ⚠ icon (or plain text ⚠ glyph) |
| `BrowserOpenURL` | `wailsjs/wailsjs/runtime/runtime` | Open external links in system browser |
| `ClipboardSetText` | `wailsjs/wailsjs/runtime/runtime` | Copy URL/code to clipboard |
| `GetCapabilityQRCode` | `wailsjs/go/main/App` | QR code for Funnel URL |
| `SetSessionFunnel` | `wailsjs/go/main/App` | Enable/disable Funnel (Wave 0 stub update) |
| `IssueCapabilities` | `wailsjs/go/main/App` | Re-issue caps to get Funnel-base URLs |
| `react-markdown` + `remark-gfm` + `rehype-sanitize` | already in frontend deps | Markdown rendering in HelpContent |

**No new packages required.** `react-markdown` is already installed and used by `HelpContent.tsx`.

---

## Architecture Patterns

### Component Responsibility After Phase 166

```
App.tsx
  hubSessions (SessionInfo[] with funnelActive) ← 3s poll
  funnelActiveSessions = derive(hubSessions)     ← new derived Record<id, boolean>
  │
  ├── TabBar
  │     funnelActiveSessions prop (new)
  │     → tab with funnelActive=true shows GlobeAltIcon + aria-label
  │
  └── HubPanel
        sessions prop (SessionInfo[] — includes funnelActive)
        shareModalSession (synced via useEffect when sessions updates) ← new sync
        │
        ├── SessionCard
        │     session.funnelActive → .hub-internet-badge pill when true
        │
        └── SessionShareModal
              session.funnelActive (from synced shareModalSession)
              │
              ├── Funnel toggle → risk panel expand/collapse
              ├── FunnelRiskPanel (inline, new)
              │     expirySelect → SetSessionFunnel(id, true, expiresIn)
              │     warmingUp → polls via session.funnelActive
              │
              └── SessionSharePanel
                    funnelActive + funnelUrl (new props)
                    → .hub-share-internet-section when funnelActive

HelpTab
  SECTION_META: [..., { id: 'help-sharing', ... }, help-faq]
  HelpSectionNav.SECTIONS: same order (separate array, separate update)
  → HelpContent renders sharing-guide.md with auto-BrowserOpenURL links
```

### Inline Risk Panel Expand/Collapse

```css
/* Established motion guard pattern (mirrors existing in project) */
@media (prefers-reduced-motion: no-preference) {
  .hub-funnel-risk-panel {
    transition: max-height 200ms ease-out, opacity 150ms ease;
  }
}
@media (prefers-reduced-motion: reduce) {
  .hub-funnel-risk-panel { transition: none; }
}
```

Expand: `riskPanelOpen = true` → CSS class applied → max-height: 0 → 400px.
Collapse on "Keep local only" or after successful enable.

### Auto-Expiry Selector Values

| Display | `expirySeconds` value |
|---------|----------------------|
| 30 minutes | 1800 |
| 1 hour (default) | 3600 |
| 4 hours | 14400 |
| 8 hours | 28800 |
| Until I disable | **0** (confirmed sentinel) |

### Project File Structure (new files only)

```
frontend/src/
├── content/help/
│   └── sharing-guide.md        # NEW — HLP-01/HLP-02 article
└── components/
    └── Hub/
        └── FunnelRiskPanel.tsx # OPTIONAL — extract risk panel to own component
                                # (or inline in SessionShareModal — planner decides)
```

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 (jsdom) |
| Config file | `frontend/vite.config.ts` (test block at line 14) |
| Setup file | `frontend/src/test-setup.ts` |
| Quick run command | `cd frontend && pnpm run test` |
| Full suite command | `cd frontend && pnpm run test && tsc --noEmit` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | Notes |
|--------|----------|-----------|-------------------|-------|
| FUI-01 | Funnel toggle shows risk panel on every click; no skip option | unit | `pnpm test -- --reporter=verbose SessionShareModal` | Mock SetSessionFunnel |
| FUI-02 | Expiry selector renders presets; selected value passed to SetSessionFunnel | unit | `pnpm test -- --reporter=verbose SessionShareModal` | Check call args |
| FUI-03 | Hub card shows `.hub-internet-badge` when funnelActive=true; aria-label on tab icon | unit | `pnpm test -- SessionCard.share.test.tsx` + TabBar test | Colorblind: hex in source |
| FUI-04 | Disable button calls SetSessionFunnel(id, false, 0) | unit | `pnpm test -- SessionSharePanel` | Mock and assert args |
| FUI-05 | Warm-up state shown immediately; hidden when funnelActive flips; timeout at 30s | unit | `pnpm test -- SessionShareModal` | Use fake timers |
| FUI-06 | Help cross-link visible in risk panel | unit | `pnpm test -- SessionShareModal` | Check button text |
| HLP-01 | Help nav shows "Sharing Outside Your Tailnet" section | unit | `pnpm test -- HelpTab.integration.test.tsx` | Check SECTION_META |
| HLP-02 | sharing-guide.md contains ACL block + :7443 + wildcard gotcha text | unit | `pnpm test -- HelpTab.integration.test.tsx` | String match on markdown |

### Manual Checklist Items (M-NN)

The following behaviors cannot be automated with vitest/jsdom and must be added to TESTING.md Section 5:

| M-ID | Behavior | Why Manual |
|------|----------|-----------|
| M-37 | Live Funnel enable: TLS warm-up completes and public URL opens in browser | Requires real Tailscale daemon + Funnel-capable tailnet |
| M-38 | Live Funnel expiry: Share auto-tears-down after chosen duration (e.g. 30m) | Requires real daemon + timer |
| M-39 | Hub card globe badge + tab globe icon visible while Funnel active; absent after disable | Requires running app + real session |
| M-40 | Local-fallback state: Funnel toggle disabled when Tailscale not running | Requires network state control |

### Wave 0 Gaps (must exist before Wave 1 implementation)

- [ ] `frontend/src/wailsjs/go/main/App.d.ts` — add `funnelActive: boolean` to `SessionInfo` interface
- [ ] `frontend/src/wailsjs/go/main/App.js` — add `SetSessionFunnel` export
- [ ] `frontend/src/wailsjs/go/main/App.d.ts` — add `SetSessionFunnel` export
- [ ] Vitest import-contract test: `SetSessionFunnel` imported from `../../wailsjs/go/main/App` (mirrors existing `App.test.tsx` pattern)
- [ ] Vitest import-contract test: `funnelActive` field accessible on `SessionInfo` type

### Phase Gate

Before `/gsd-verify-work`:
1. `cd frontend && pnpm run test` — full vitest suite green
2. `cd frontend && tsc --noEmit` — no TypeScript errors
3. Manual M-37 through M-40 complete (live UAT against real daemon)

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Funnel access is gated by join code (backend responsibility, Phase 165) |
| V3 Session Management | no | No new session types |
| V4 Access Control | no | Backend enforces Funnel capability gate (FNL-06) |
| V5 Input Validation | yes | expiresIn values are from a fixed preset `<select>` — no free text input; integer only |
| V6 Cryptography | no | TLS handled by Tailscale |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Open redirect via BrowserOpenURL | Spoofing | HelpContent.tsx already validates `SAFE_LINK_SCHEME = /^(https?:|mailto:)/i` before calling BrowserOpenURL |
| Slopsquatted npm package | Tampering | No new packages introduced — not applicable |
| expiresIn injection via UI | Tampering | Values from fixed `<select>` enum; UI cannot submit arbitrary integers |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Tailscale Funnel KB URL is `https://tailscale.com/kb/1223/funnel` | Tailscale doc URLs | Help article links to a 404; low severity (link is in Help text, not a functional dependency) |
| A2 | ACL syntax KB URL is `https://tailscale.com/kb/1337/acl-syntax` | Tailscale doc URLs | Same as A1 |
| A3 | Device sharing KB URL is `https://tailscale.com/kb/1084/sharing` | Tailscale doc URLs | Same as A1 |
| A4 | Tags KB URL is `https://tailscale.com/kb/1068/acl-tags` | Tailscale doc URLs | Same as A1 |

All other claims in this research are VERIFIED from source files.

---

## Open Questions

1. **FunnelRiskPanel: inline or extracted component?**
   - What we know: The risk panel is self-contained with its own state (`expirySeconds`, `isOpen`). It adds ~80-100 lines to `SessionShareModal.tsx` if inlined.
   - What's unclear: Whether the planner wants a separate `FunnelRiskPanel.tsx` file or inline within the modal.
   - Recommendation: Extract to `FunnelRiskPanel.tsx` for testability (vitest can import it independently). Mirrors the `ShellWebShareBanner` pattern (a separate component mounted inside the modal).

2. **SessionSharePanel refactor scope**
   - What we know: SessionSharePanel needs `funnelActive` and `funnelUrl` props for the new section.
   - What's unclear: Whether the Internet section is a nested component or inline JSX in SessionSharePanel.
   - Recommendation: Inline JSX inside SessionSharePanel (no new file needed). The section is simple enough.

3. **`shareModalSession` sync effect scope**
   - What we know: HubPanel owns `shareModalSession` and receives `sessions` prop updated by 3 s poll.
   - What's unclear: Whether the sync effect causes any performance concern (runs on every 3 s poll when modal is open).
   - Recommendation: Guard with `if (!shareModalSession) return` — only runs when modal is open. Safe.

---

## Environment Availability

This phase is frontend-only. No external CLI tools or services are required for the build/test cycle.

| Dependency | Required By | Available | Notes |
|------------|-------------|-----------|-------|
| pnpm | frontend build + test | Yes (existing project) | `cd frontend && pnpm run test` |
| vitest 4.1.0 | test suite | Yes (node_modules) | Already installed |
| tsc | type-checking gate | Yes (TypeScript in frontend) | `tsc --noEmit` |
| Real Tailscale daemon with Funnel | M-37/M-38/M-39/M-40 | Unknown — depends on dev machine | Manual UAT only; not needed for automated tests |

---

## Sources

### Primary (HIGH confidence)
- `internal/daemon/types.go:154-158` — SetSessionFunnelRequest + expiresIn sentinel comment
- `internal/daemon/api.go:1507-1572` — handleSetSessionFunnel + timer conditional (`ExpiresIn > 0`)
- `internal/daemon/client.go:361-368` — SetSessionFunnel client, drops response body (`nil`)
- `app.go:47-50` — SessionInfo.FunnelActive field definition
- `app.go:894-903` — SetSessionFunnel Wails binding signature
- `frontend/src/wailsjs/wailsjs/go/main/App.d.ts:113` — generated SetSessionFunnel TypeScript signature
- `frontend/src/wailsjs/wailsjs/go/models.ts:198-237` — generated SessionInfo class with funnelActive
- `frontend/src/wailsjs/go/main/App.d.ts:1-208` — hand-authored stub (MISSING SetSessionFunnel, funnelActive)
- `frontend/src/wailsjs/go/main/App.js:1-129` — hand-authored JS stub (MISSING SetSessionFunnel)
- `internal/tailnet/tailnet.go:29` — DefaultProbePort = 7443
- `internal/daemon/process.go:49,125,136` — port 7443 passed to AutoStartWebServer/RestartWebServer
- `frontend/src/App.tsx:984` — Hub poll interval 3000ms
- `frontend/src/components/Hub/HubPanel.tsx:277-281` — shareModalSession snapshot pattern
- `frontend/src/components/HelpTab.tsx:61-65` — SECTION_META array
- `frontend/src/components/HelpSectionNav.tsx:10-13` — SECTIONS array (separate from HelpTab)
- `frontend/src/components/HelpContent.tsx:65-85` — auto BrowserOpenURL for markdown links
- `internal/daemon/api.go:1344-1352` — Funnel URL swap in IssueCapabilities

### Secondary (MEDIUM confidence)
- `frontend/src/components/Hub/SessionShareModal.tsx` — animation state machine, ShareSession interface, SetSessionBrowse pattern
- `frontend/src/components/SessionSharePanel.tsx` — CodeDisplay + QR pattern, ClipboardSetText/BrowserOpenURL usage
- `frontend/src/components/TabBar.tsx` — Tab interface, sessionStatuses prop pattern

### Tertiary (LOW confidence / ASSUMED)
- Tailscale documentation URLs — from training data, not verified via web search this session

---

## Metadata

**Confidence breakdown:**
- Binding signatures: HIGH — verified from app.go, generated d.ts, and hand-authored stub
- Expiry sentinel: HIGH — comment + conditional branch in daemon source
- Port 7443: HIGH — two independent source files confirm
- Warm-up poll interval: HIGH — setInterval call in App.tsx
- HelpSectionNav separate SECTIONS: HIGH — file read confirms separate array
- Tailscale doc URLs: LOW — training data only

**Research date:** 2026-06-30
**Valid until:** 2026-07-30 (stable frontend codebase; Go bindings change only when wails dev runs)
