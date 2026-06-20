---
phase: 137-share-modal-cap-model
verified: 2026-06-20T00:00:00Z
status: human_needed
score: 9/9 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Open the Hub with a running local session. Click the Share button on the card. Toggle 'Share the session' ON. Verify two link rows (read-only and full-access) appear with copyable URLs and QR codes."
    expected: "Both link rows render; QR codes appear on toggle; URLs are non-empty and distinct."
    why_human: "IssueCapabilities requires a live daemon + web server; cannot be invoked without a running app."
  - test: "With the Share modal open and sharing ON, toggle 'Enable remote file browsing' ON. Open the read-only share URL in a second browser. Verify the file browser is accessible in read mode."
    expected: "Read-only token grants files.read; file list loads; write operations 403."
    why_human: "Requires live daemon, running web server, and second browser session to verify SHARE-03 token inheritance end-to-end."
  - test: "In local-network mode (web server in local/LAN mode), open the Share modal. Verify the Basic Auth password appears."
    expected: "LAN password is shown in the modal under the share links (SHARE-04)."
    why_human: "Requires daemon running in local-network mode with GetLocalNetworkPassword returning a real password."
  - test: "Open the Share modal on a session whose working directory is the user's home directory (homeDir=true). Verify the HomeDirWriteWarning is shown."
    expected: "The warning banner renders before the browse toggle (D-09)."
    why_human: "Requires a live session where the daemon reports homeDir=true."
  - test: "Hover or inspect the Share button on a remote peer card. Verify the button is disabled, shows a lock icon, and the tooltip/aria-label reads 'Only the session owner can share'."
    expected: "Shape + text + tooltip all convey the disabled state; no color-only signal (D-13 colorblind-safe)."
    why_human: "Visual affordance on a remote peer card requires a live multi-peer Hub session for realistic testing; colorblind-safe check must match source (LockClosedIcon confirmed at source level)."
  - test: "Toggle the web server off then on while the Share modal is open. Verify the cached share URLs are cleared and new URLs are issued automatically."
    expected: "After restart, new IssueCapabilities call produces new URLs; old links become invalid (SHARE-05 restart-clear)."
    why_human: "Requires live daemon + controlled server restart; the effect transition from false to true must be observable in the running UI."
---

# Phase 137: Share Modal & Cap Model Verification Report

**Phase Goal:** Each Hub session card has a Share button opening a per-session Share modal; the cap model issues one "share" action minting separate read-only and read/write tokens; file-browse permission inherits from the presented token.
**Verified:** 2026-06-20
**Status:** human_needed (all automated checks pass; 6 live-daemon UAT items require human testing)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | Browse OFF: RO token carries "read", RW token carries "read,write" (D-03) | VERIFIED | `api.go:1116-1120` — `rPerms := "read"`, `wPerms := "read,write"` when `browseEnabledFor(sessionID)` is false; `TestIssueCapabilities_BrowseOff_NoFilesPerms` passes |
| 2  | Browse ON: RO token carries "read,files.read" (never files.write); RW token carries "read,write,files.read,files.write" (D-04) | VERIFIED | `api.go:1118-1121` injects via `browseEnabledFor`; `TestIssueCapabilities_BrowseOn_ROPermsExact` and `TestIssueCapabilities_BrowseOn_RWPermsExact` both pass |
| 3  | Per-session browse toggle is the sole driver; global filesRead and per-session files.write two-gate no longer exist (D-02/D-07) | VERIFIED | No `filesReadEnabled`, `filesWriteEnabledFor`, `SetSessionFilesWrite`, `sessionWrites` in `engine.go` or `api.go`; `sessionBrowse` map is the only driver |
| 4  | Toggling browse off invalidates outstanding caps (ClearGrants), matching the web-serve toggle-off lifecycle (SHARE-05) | VERIFIED | `api.go:1297-1304` — `handleSetSessionBrowse` calls `ws.ClearGrants(id)` on every toggle; comment confirms parity with handleWebServe |
| 5  | SessionInfo.BrowseEnabled surfaces server-truth browse state for modal seeding on open (SHARE-05) | VERIFIED | `types.go:33` — `BrowseEnabled bool json:"browseEnabled"` (no omitempty); `engine.go:474` populates it via `browseEnabledForUnlocked`; `SessionShareModal.tsx:99` seeds `browseEnabled` state from `session.browseEnabled` |
| 6  | KillSession deletes the sessionBrowse entry so a recycled ID cannot inherit stale browse-ON (CR-01 fix) | VERIFIED | `engine.go:506` — `delete(e.sessionBrowse, id)` inside the `e.mu.Lock()` block alongside the other per-session map deletes; `TestKillSession_ClearsStaleBrowseEntry` passes |
| 7  | Each local Hub card has a Share button that opens the per-session Share modal; remote peer cards have the button disabled with a colorblind-safe lock icon + tooltip (SHARE-01/SHARE-06/D-13) | VERIFIED | `SessionCard.tsx:420-426` — `hub-card__share` button, `LockClosedIcon` import at line 15, `aria-label`/`title` "Only the session owner can share" on remote; `SessionCard.share.test.tsx` 4/4 passing |
| 8  | The modal has "Share the session" toggle (reveals RO+RW links) and "Enable remote file browsing" toggle (calls SetSessionBrowse then re-issues caps); browse toggle disabled when sharing is OFF (SHARE-02/SHARE-03) | VERIFIED | `SessionShareModal.tsx:186-227` — `handleShareToggle` calls `ToggleWebServing`; `handleBrowseToggle` calls `SetSessionBrowse` then `IssueCapabilities`; browse toggle has `disabled={!shareEnabled}`; `SessionShareModal.test.tsx` 9/9 passing |
| 9  | The onShare prop chain (App.tsx → HubPanel → SessionCardGrid → both SessionCard render sites) is complete; HubPanel mounts SessionShareModal gated on shareModalSession (SHARE-01 end-to-end) | VERIFIED | `App.tsx:1399` passes `webServerMode`+`webServerRunning` to HubPanel; `HubPanel.tsx:235-238,448,506-511` — `shareModalSession` state, `handleShare` callback, modal mounted; `SessionCardGrid.tsx:150,250,294` — `onShare` prop at both render sites |

**Score: 9/9 truths verified**

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/engine.go` | sessionBrowse map + browseEnabledFor/SetSessionBrowse; retired fields gone | VERIFIED | `sessionBrowse map[string]bool` at line 45; `browseEnabledFor` at line 599; `SetSessionBrowse` at line 616; `filesRead`, `sessionWrites` absent |
| `internal/daemon/api.go` | D-03/D-04 perm injection + POST /sessions/{id}/browse + ClearGrants | VERIFIED | Route registered at line 140; handler at line 1288; perm matrix at lines 1116-1121; ClearGrants at line 1303; audit comment with Reversal 1/3 at lines 1099-1115 |
| `internal/daemon/types.go` | BrowseEnabled bool (no omitempty) + SessionBrowseRequest | VERIFIED | `BrowseEnabled bool json:"browseEnabled"` at line 33; `SessionBrowseRequest` struct present; `FilesWrite` field removed from `SessionInfo` |
| `internal/daemon/client.go` | SetSessionBrowse client method | VERIFIED | `func (c *DaemonClient) SetSessionBrowse` at line 323 |
| `app.go` | SetSessionBrowse Wails binding; SetSessionFilesWrite retired | VERIFIED | `func (a *App) SetSessionBrowse` at line 832; no `SetSessionFilesWrite` anywhere in app.go |
| `frontend/src/components/Hub/SessionShareModal.tsx` | Two-toggle modal: share, browse, LAN password, homeDir warning, server-truth lifecycle (min 80 lines) | VERIFIED | 350 lines; all lifecycle behaviors present; imports SetSessionBrowse, IssueCapabilities, GetLocalNetworkPassword, HomeDirWriteWarning |
| `frontend/src/components/Hub/SessionCard.tsx` | hub-card__share button + LockClosedIcon D-13 disabled state | VERIFIED | `hub-card__share` class at line 420; `LockClosedIcon` at line 426; aria-label/title at lines 423-424 |
| `frontend/src/components/SessionSharePanel.tsx` | CAP-05 two-gate stripped; browseEnabled prop for scope text | VERIFIED | No `ownerWriteEnabled`, `allowFileEditing`, `showWriteConfirm`, `surfaceWriteLink`; `browseEnabled` at line 82; correct scope text at lines 207-209 |
| `frontend/src/components/Hub/HubPanel.tsx` | shareModalSession state; SessionShareModal mounted | VERIFIED | `shareModalSession` state at line 235; modal mounted at lines 506-511 |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | onShare prop to both SessionCard render sites | VERIFIED | `onShare` in props at line 150; passed at lines 250 and 294 |
| `frontend/src/App.tsx` | webServerMode + webServerRunning passed to HubPanel | VERIFIED | Both props at lines 1399-1400 of HubPanel JSX |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `api.go issueCapabilitiesForSession` | `engine.go browseEnabledFor` | perm injection reads browse flag | VERIFIED | `api.go:1118` calls `a.engine.browseEnabledFor(sessionID)` |
| `app.go SetSessionBrowse` | `client.go SetSessionBrowse` | Wails binding delegates to daemon client | VERIFIED | `app.go:836` — `return a.client.SetSessionBrowse(sessionID, enabled)` |
| `SessionCard.tsx` | SessionShareModal via onShare prop | Share button onClick fires onShare with stopPropagation | VERIFIED | `SessionCard.tsx:421` — `onClick={(e) => { e.stopPropagation(); onShare?.(session) }}` |
| `App.tsx` | HubPanel (webServerMode + webServerRunning) | props threaded down | VERIFIED | `App.tsx:1399-1400` |
| `HubPanel.tsx` | SessionShareModal (shareModalSession state) | `onShare={(s) => setShareModalSession(s)}`; modal gated on state | VERIFIED | `HubPanel.tsx:235-238, 448, 506-511` |
| `SessionCardGrid.tsx` | SessionCard (onShare prop) | both render sites receive onShare | VERIFIED | `SessionCardGrid.tsx:250, 294` |
| `SessionShareModal.tsx` | SetSessionBrowse + IssueCapabilities | browse toggle then cap re-issue (Pitfall 1) | VERIFIED | `SessionShareModal.tsx:209-214` |
| `SessionShareModal` | SessionSharePanel | renders simplified panel with RO/RW URLs + browseEnabled | VERIFIED | `SessionShareModal.tsx:337-344` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `SessionShareModal.tsx` | `cachedShare` (readURL/writeURL/readCode/writeCode) | `IssueCapabilities(session.id)` Wails call | Yes — live daemon call mints HMAC-signed tokens | FLOWING |
| `SessionShareModal.tsx` | `browseEnabled` | `session.browseEnabled` (seeds from `SessionInfo.BrowseEnabled`) → `browseEnabledForUnlocked` in engine | Yes — real in-memory per-session map state | FLOWING |
| `SessionShareModal.tsx` | `lanPassword` | `GetLocalNetworkPassword()` Wails call | Yes — live daemon call in local mode | FLOWING |
| `SessionSharePanel.tsx` | `readURL`, `writeURL`, `readCode`, `writeCode` | Props from SessionShareModal's `cachedShare` | Yes — passed from IssueCapabilities response | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Browse-OFF perm matrix: RO="read", RW="read,write" | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOff_NoFilesPerms` | PASS | PASS |
| Browse-ON RO="read,files.read" (no files.write) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_ROPermsExact` | PASS | PASS |
| Browse-ON RW="read,write,files.read,files.write" | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_RWPermsExact` | PASS | PASS |
| CR-01 regression: KillSession clears sessionBrowse | `go test ./internal/daemon/... -run TestKillSession_ClearsStaleBrowseEntry` | PASS | PASS |
| RO cap can browse files (200 on read route) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_BrowseOn_FilesReadRoute200` | PASS | PASS |
| RO cap cannot write files (403 on write route) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_BrowseOn_WriteRoute403` | PASS | PASS |
| RW cap can write files (200 on write route) | `go test ./internal/webserver/... -run TestFilesRoutes_RW_BrowseOn_WriteRoute200` | PASS | PASS |
| No strings.Contains on perm literals in api.go/engine.go | `go test ./internal/webserver/... -run TestHasPerm_NoStringsContains_Browse` | PASS | PASS |
| Frontend: Share button renders, no-bubble, remote-disabled | `vitest run SessionCard.share` | 4/4 PASS | PASS |
| Frontend: Modal toggles, SetSessionBrowse, LAN password, homeDir, seeding, restart-clear | `vitest run SessionShareModal` | 9/9 PASS | PASS |
| Frontend: CAP-05 stripped, write link always shown, browseEnabled scope text | `vitest run SessionSharePanel` | 9/9 PASS | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SHARE-01 | Plans 01, 03 | Each Hub session card has a "Share" button that opens a per-session Share modal | SATISFIED | Share button in SessionCard; SessionShareModal mounted from HubPanel; prop chain complete |
| SHARE-02 | Plans 01, 03 | Share modal has "Share the session" toggle revealing two share links/codes (RO + RW) | SATISFIED | `SessionShareModal.tsx:186-199` toggle; SessionSharePanel renders both link rows |
| SHARE-03 | Plans 01, 02, 03 | "Enable remote file browsing" toggle; browse perm inherits from share code presented | SATISFIED | D-03/D-04 matrix in api.go; requireFilesRead/requireFilesWrite enforcement unchanged; browse toggle calls SetSessionBrowse then IssueCapabilities |
| SHARE-04 | Plans 01, 03 | Links/codes copyable with QR; LAN Basic Auth password in modal (local mode) | SATISFIED | SessionSharePanel has QR + clipboard; `SessionShareModal.tsx:112-117` fetches LAN password in local mode |
| SHARE-05 | Plans 01, 02, 03 | Share modal carries forward all per-session web-share capabilities; no regression in cap/URL/QR lifecycle | SATISFIED | Toggle-off clears ClearGrants (api.go:1303); server-truth seeding effect (SessionShareModal.tsx:161-177); restart-clear effect (lines 130-154); browse toggle clears grants on daemon side |
| SHARE-06 | Plans 01, 03 | Sharing controls unavailable on remote peer cards | SATISFIED | `SessionCard.tsx:420-426` — `disabled={!isLocal}`; LockClosedIcon; aria-label/title colorblind-safe |

All 6 SHARE requirements are satisfied. No orphaned requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| No blocker-level debt markers found in any phase-modified file | — | — | — | — |

No `TBD`, `FIXME`, or `XXX` markers found in phase-modified production files.

**Open review warnings (not yet fixed — assessed as non-blocking to goal):**

| Warning | File | Issue | Severity | Goal Impact |
|---------|------|-------|----------|-------------|
| WR-01 | `SessionShareModal.tsx:130-177` | Restart-clear and seeding effects can both issue IssueCapabilities concurrently on a false→true webServerRunning transition, registering two grant pairs | WARNING | Not a blocker — last-writer-wins on cachedShare; both tokens are valid; no security exposure; doubles grants on daemon until ClearGrants/session end |
| WR-02 | `api.go:1288-1306` | `handleSetSessionBrowse` has no `http.MaxBytesReader` or `DisallowUnknownFields` — deviates from the defense-in-depth pattern of peer handlers | WARNING | Not a blocker — daemon socket is loopback-trusted (127.0.0.1 only); no external attacker can reach this endpoint; a large body would be bounded by OS socket buffer |
| WR-05 | `SessionShareModal.tsx:300-332` | HomeDirWriteWarning is cosmetic-dismiss only; the browse toggle can be enabled on a home-directory session without acknowledging the warning | WARNING | Not a blocker — the phase brief explicitly classifies D-09 as a cosmetic warning, not a gate; backend perms are the authority; the warning is informational |

**Fixed review items (confirmed in codebase):**
- CR-01: FIXED — `delete(e.sessionBrowse, id)` in KillSession + `TestKillSession_ClearsStaleBrowseEntry` passes
- WR-03: FIXED — Scope text at `SessionSharePanel.tsx:208` correctly reads "Watch the live session and browse files read-only — cannot send input." when `browseEnabled=true`

### Human Verification Required

All automated checks pass (9/9 truths VERIFIED, 22/22 frontend tests pass, 8/8 Go security-core tests pass, `go build ./...` clean, `tsc --noEmit` clean). The following items require live-daemon testing and cannot be verified programmatically:

### 1. Share Modal Opens from Hub Card

**Test:** In a running app, click the Share button on a local Hub session card.
**Expected:** The Share modal opens for that specific session, showing the session name and two toggles.
**Why human:** Requires live Wails desktop app with daemon running.

### 2. RO + RW Link Rows with QR (SHARE-02/SHARE-04)

**Test:** Toggle "Share the session" ON. Verify the read-only and full-access link rows appear with copyable URLs, QR toggle buttons, and join codes.
**Expected:** Both link rows render; QR codes appear on demand; URLs are non-empty and distinct (read vs. read/write tokens).
**Why human:** IssueCapabilities requires live daemon + web server running; URLs contain real HMAC-signed tokens.

### 3. File Browse Permission Inheritance (SHARE-03)

**Test:** With sharing ON, toggle "Enable remote file browsing" ON. Open the read-only share URL in a browser. Navigate to the file browser. Try to write/edit a file.
**Expected:** File list loads (files.read granted to RO token); write attempt is 403 forbidden.
**Why human:** Requires live daemon, web server, real HMAC tokens, and second browser to test the enforcement path end-to-end.

### 4. LAN Basic Auth Password (SHARE-04)

**Test:** Run the web server in local mode. Open the Share modal. Check for the LAN password display.
**Expected:** The Basic Auth password appears in the modal under the share links.
**Why human:** Requires daemon in local-network mode with GetLocalNetworkPassword returning a real password.

### 5. HomeDirWriteWarning Shown (D-09)

**Test:** Open a session whose working directory is the user's home directory. Open the Share modal.
**Expected:** The HomeDirWriteWarning banner renders above the browse toggle.
**Why human:** Requires a live session where the daemon reports `homeDir=true`.

### 6. Remote Peer Card Share Button (SHARE-06/D-13)

**Test:** In a Hub with a visible remote peer card, inspect or hover the Share button on that card.
**Expected:** Button is disabled; lock icon visible; tooltip/aria-label reads "Only the session owner can share"; clicking does nothing.
**Why human:** Requires a live multi-peer Hub session. Colorblind-safe affordance verified at source level (LockClosedIcon + aria-label/title confirmed in `SessionCard.tsx:15,423-426` — shape + text, not color alone).

---

_Verified: 2026-06-20_
_Verifier: Claude (gsd-verifier)_
