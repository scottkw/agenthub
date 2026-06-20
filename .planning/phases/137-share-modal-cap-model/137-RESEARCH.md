# Phase 137: Share Modal & Cap Model — Research

**Researched:** 2026-06-20
**Domain:** Go capability model + React modal UI (security-sensitive)
**Confidence:** HIGH — all claims sourced from direct codebase inspection

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01:** When "Enable remote file browsing" is ON, file permission inherits fully from the share code presented — no separate owner write opt-in, no viewer confirm.

**D-02:** Removes the CAP-05 viewer "Allow file editing" two-gate and the separate per-session `files.write` opt-in (`SetSessionFilesWrite` is subsumed by the browse toggle). Justification: the RW code already grants full terminal read+write; granting `files.write` to the same code is not a privilege escalation beyond what the holder already has.

**D-03:** Browse OFF (default): RO code = `"read"`; RW code = `"read,write"`. Neither grants filesystem access.

**D-04:** Browse ON: RO code = `"read,files.read"` (list/stat/read within sandbox, no writes); RW code = `"read,write,files.read,files.write"`. Symmetric inherit-from-code model.

**D-05:** RO-code holders gaining read-only filesystem access when browse is ON is intended and accepted.

**D-06:** Browse toggle is per-session, default OFF.

**D-07:** Remove the global `filesReadEnabled` setting from engine.go and the global file-browsing control (if any) from Settings UI. Per-session browse toggle becomes the sole driver for injecting `files.read` (both codes) and `files.write` (RW code only).

**D-08:** Browse-enabled state is ephemeral — in-memory alongside the existing web-serve toggle. A daemon restart resets to OFF. Modal seeds from server truth on open (SHARE-05).

**D-09:** CAP-06 home-directory write warning is retained. When session cwd is `$HOME`, the modal shows the home-dir warning before the owner enables browsing.

**D-10:** Two toggles in modal: ① "Share the session" (reveals RO + RW links/codes, copyable, each with QR code, and LAN Basic Auth password when local mode); ② "Enable remote file browsing" (disabled/no-op when sharing is OFF).

**D-11:** Reuse + simplify existing `SessionSharePanel` inside the new per-card modal. Strip the dead CAP-05 two-gate write UI. Wire the single browse toggle. Do NOT rebuild from scratch.

**D-12:** A dedicated "Share" button on Hub `SessionCard` opens the modal (SHARE-01).

**D-13:** On remote peer cards, Share button is visible but disabled with a tooltip ("Only the session owner can share"). Disabled state must be colorblind-safe (greyed + lock icon + tooltip, not color alone). Satisfies SHARE-06.

### Claude's Discretion

- Exact mechanism for collapsing `SetSessionFilesWrite` / `filesReadEnabled` into the new per-session browse state in `SessionEngine` (rename, repurpose `sessionWrites`, or new field).
- Cap-reissue-on-toggle plumbing.
- Migration/cleanup of the removed global Settings control.
- Final modal copy/labels and exact disabled-state iconography (subject to Phase 140 UI-spec).

### Deferred Ideas (OUT OF SCOPE)

- Persisting share/browse state across daemon restarts (chose ephemeral, D-08).
- Card visual redesign, local/remote + connected/available indicators, mini-preview/tail VT render fix — CARD-01..05 / RDS belong to Phases 138-140.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SHARE-01 | Each Hub session card has a "Share" button opening a per-session Share modal | `SessionCard.tsx` row5 pattern; add dedicated Share button alongside existing Open button |
| SHARE-02 | Share modal has "Share the session" toggle; when ON reveals RO + RW links/codes | Reuse `SessionSharePanel` with new modal wrapper; strip CAP-05 gate |
| SHARE-03 | "Enable remote file browsing" toggle; file-browse perm inherits from presented share code | Per-session `sessionBrowse` flag in engine.go; perm injection edit site api.go:1095-1116 |
| SHARE-04 | Links/codes copyable, each has a QR code, LAN Basic Auth password surfaces in local mode | All already in `SessionSharePanel`; LAN password flow in `DaemonManagerPanel` to migrate |
| SHARE-05 | Carries forward all per-session web-share capabilities from Sessions page with no regression | Reconcile effect + server-truth seeding pattern from `DaemonManagerPanel.tsx` lines 176-233 |
| SHARE-06 | Sharing controls unavailable on remote peer cards (cannot re-share unowned session) | `isLocal` check on `SessionCard`; disabled + lock icon + tooltip pattern (D-13) |
</phase_requirements>

---

## Summary

Phase 137 delivers a per-session Share modal on each Hub card, replacing/migrating all web-share functionality previously in `DaemonManagerPanel` (the Sessions page). The core security change is the "file-browse inherits from cap" model: instead of a separate two-gate system (owner enables write, viewer confirms), the single browse toggle injects file perms directly into the cap at issuance time — RO code gets `files.read`, RW code gets `files.read,files.write`.

The work splits cleanly into three areas:

1. **Go backend (security-sensitive):** Replace `filesReadEnabled()` (global) + `filesWriteEnabledFor()` / `SetSessionFilesWrite()` (per-session two-gate) with a single per-session `browseEnabled` state. Edit `issueCapabilitiesForSession()` at api.go:1095-1116 to inject perms from this flag instead of the old global+write-gate logic. Add a new daemon API endpoint to toggle browse (or repurpose the existing files-write endpoint with new semantics). Update `SessionInfo` types if browse state needs surfacing.

2. **Frontend modal (UI):** Create `SessionShareModal` — a new modal wrapping a simplified `SessionSharePanel`. The simplification strips the CAP-05 "Allow file editing" opt-in row entirely, adds a single "Enable remote file browsing" toggle, and wires the LAN password block (migrated from `DaemonManagerPanel`). Add a "Share" button to `SessionCard` (row5 pattern). Disable it for remote peer cards with a lock icon and tooltip.

3. **Settings cleanup:** Remove the global `filesRead` setting from settings persistence and serialization in `engine.go`. The Settings UI doesn't currently expose a global file-browsing toggle (confirmed by codebase search — the `filesRead` field was internal to `daemonSettings` and engine.go only; no frontend PATCH endpoint exists for it).

**Primary recommendation:** The perm-injection rewrite at api.go:1095-1116 is the security core. Write it as a clean isolated delta with explicit test coverage for all four D-03/D-04 matrix cells before any frontend work.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Browse toggle state | Backend daemon (engine.go in-memory) | — | Ephemeral per-session state, same lifecycle as `sessionWrites`; must survive frontend reload |
| Perm injection into RO/RW caps | Backend daemon (api.go) | — | HMAC signing happens server-side; clients receive opaque tokens |
| Cap enforcement (files.read/files.write) | Backend webserver (capability_mw.go) | — | Already implemented; `requireFilesRead` and `requireFilesWrite` wrappers are the enforcement point |
| Share modal UI | Frontend (React) | — | Per-card modal opened by Share button on SessionCard |
| Share button on card | Frontend (SessionCard.tsx) | — | New dedicated button in row5 of the Hub card |
| Remote-peer disabled state | Frontend (SessionCard.tsx) | — | `isLocal` boolean already derived in SessionCard from `hostname` |
| LAN password display | Frontend (Share modal) | — | Migrated from DaemonManagerPanel; already fetched via `GetLocalNetworkPassword()` Wails binding |
| Home-dir warning | Frontend (Share modal) | — | `HomeDirWriteWarning` component already exists; source from `session.homeDir` field (D-09) |

---

## Standard Stack

This phase introduces **no new external packages**. All components are internal. [VERIFIED: direct codebase inspection]

### Core Components Being Modified

| Component | File | Change Type |
|-----------|------|-------------|
| `SessionEngine` | `internal/daemon/engine.go:44-633` | Remove `filesRead *bool` + `filesWriteDefault` + `sessionWrites`; add per-session `sessionBrowse map[string]bool` |
| `issueCapabilitiesForSession` | `internal/daemon/api.go:1060-1146` | Rewrite perm injection at lines 1095-1116 per D-03/D-04 matrix |
| `SessionInfo` | `internal/daemon/types.go:20-35` | No new fields needed (browse state is ephemeral, not surfaced in ListSessions response) |
| `DaemonClient` | `internal/daemon/client.go` | Add `SetSessionBrowse(sessionID string, enabled bool)` method; remove `SetSessionFilesWrite` or repurpose it |
| Wails `App` | `app.go` | Add `SetSessionBrowse` binding; retire `SetSessionFilesWrite` binding |
| `SessionSharePanel` | `frontend/src/components/SessionSharePanel.tsx` | Strip CAP-05 two-gate opt-in (`ownerWriteEnabled` prop + `allowFileEditing` state + confirm flow); add `browseEnabled` prop to control scope label |
| `SessionCard` | `frontend/src/components/Hub/SessionCard.tsx` | Add Share button; add `onShare` prop; remote-peer disabled state |
| New: `SessionShareModal` | `frontend/src/components/Hub/SessionShareModal.tsx` | Modal wrapper (reuse HubModal overlay pattern); hosts simplified `SessionSharePanel` + "Share" toggle + browse toggle + LAN password block |

### Package Legitimacy Audit

No new external packages are installed in this phase. All dependencies are already present in the project. [VERIFIED: direct codebase inspection]

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
Owner action in GUI
        │
        ▼
 [SessionCard] ─── Share button click ──▶ [SessionShareModal]
                                                  │
                          ┌───────────────────────┤
                          │                       │
                 "Share the session"     "Enable remote file browsing"
                     toggle                      toggle
                          │                       │
                          ▼                       ▼
                   ToggleWebServing()      SetSessionBrowse()
                   [daemon API]            [daemon API — new endpoint]
                          │                       │
                          └──────────┬────────────┘
                                     │
                              IssueCapabilities()
                              [daemon API — on toggle-on]
                                     │
                                     ▼
                       issueCapabilitiesForSession()
                       [api.go:1060 — EDIT SITE]
                                     │
                       ┌─────────────┴──────────────┐
                    Browse OFF                  Browse ON
                       │                           │
              RO="read"  RW="read,write"   RO="read,files.read"
                                           RW="read,write,files.read,files.write"
                                     │
                                     ▼
                          [SessionSharePanel] renders:
                          - RO link + join code + QR
                          - RW link + join code + QR
                          - LAN password (if local mode)
                                     │
                        Web visitor presents cap token
                                     ▼
                         [capability_mw.go]
                         requireCapability → requireFilesRead / requireFilesWrite
                         (enforces what's IN the token — no server-side lookup)
```

### Recommended Project Structure

The new modal lives alongside existing Hub modals:

```
frontend/src/components/
├── Hub/
│   ├── SessionCard.tsx          # Add Share button + onShare prop
│   ├── SessionShareModal.tsx    # NEW: modal wrapper (thin shell over SessionSharePanel)
│   └── ...
├── SessionSharePanel.tsx        # SIMPLIFIED: strip CAP-05 opt-in row
└── HomeDirWriteWarning.tsx      # REUSED as-is (D-09)

internal/daemon/
├── api.go                       # Edit issueCapabilitiesForSession (lines 1095-1116)
├── engine.go                    # Collapse filesRead + sessionWrites → sessionBrowse
├── client.go                    # Add SetSessionBrowse; retire SetSessionFilesWrite
└── types.go                     # Add SetSessionBrowseRequest (mirrors SessionFilesWriteRequest)

app.go                           # Add SetSessionBrowse binding; retire SetSessionFilesWrite
```

### Pattern 1: Per-Session Browse State in SessionEngine

**What:** New in-memory `sessionBrowse map[string]bool` field, replacing `filesRead *bool` (global) and `sessionWrites map[string]bool` (two-gate write).

**When to use:** On every `issueCapabilitiesForSession` call; on `SetSessionBrowse` toggle.

**Example (proposed implementation):**
```go
// Source: internal/daemon/engine.go — Claude's discretion area (D-01)
// Replaces: filesReadEnabled() + filesWriteEnabledFor() + SetSessionFilesWrite()

func (e *SessionEngine) browsEnabledFor(sessionID string) bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.sessionBrowse[sessionID] // false if absent (default OFF per D-06)
}

func (e *SessionEngine) SetSessionBrowse(sessionID string, enabled bool) {
    e.mu.Lock()
    defer e.mu.Unlock()
    if e.sessionBrowse == nil {
        e.sessionBrowse = make(map[string]bool)
    }
    e.sessionBrowse[sessionID] = enabled
}
```

[VERIFIED: engine.go:588-633 — existing `filesWriteEnabledFor` / `SetSessionFilesWrite` are the direct predecessors; same mutex, same map pattern]

### Pattern 2: Perm Injection in issueCapabilitiesForSession (D-03/D-04 Matrix)

**What:** Replace the current conditional perm-building logic at api.go:1095-1116 with the browse-flag-driven four-cell matrix.

**CURRENT logic (to be replaced):**
```go
// api.go:1103-1114 — CURRENT (to be removed)
ownerPerms := "read,write"
if a.engine.filesReadEnabled() {            // global gate — D-07: REMOVE
    ownerPerms = "read,write," + capability.PermFilesRead
}
if a.engine.filesWriteEnabledFor(sessionID) { // two-gate write — D-02: REMOVE
    ownerPerms += "," + capability.PermFilesWrite
}
rClaims := capability.Claims{SID: sessionID, Perms: "read", ...}
wClaims := capability.Claims{SID: sessionID, Perms: ownerPerms, ...}
```

**NEW logic (D-03/D-04 implementation):**
```go
// api.go:1095-1116 — PROPOSED REPLACEMENT
// D-03: Browse OFF: RO="read", RW="read,write"
// D-04: Browse ON:  RO="read,files.read", RW="read,write,files.read,files.write"
rPerms := "read"
wPerms := "read,write"
if a.engine.browseEnabledFor(sessionID) {
    rPerms = "read," + capability.PermFilesRead
    wPerms = "read,write," + capability.PermFilesRead + "," + capability.PermFilesWrite
}
rClaims := capability.Claims{SID: sessionID, Perms: rPerms, IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
wClaims := capability.Claims{SID: sessionID, Perms: wPerms, IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}
```

[VERIFIED: api.go:1095-1116 — exact edit site confirmed by codebase inspection; `capability.PermFilesRead` / `capability.PermFilesWrite` constants verified at capability.go:30,37]

### Pattern 3: Browse Toggle → Cap Reissue Flow

**What:** Toggle-on or toggle-off of browse must trigger a cap reissuance (same as the existing files-write toggle pattern in `DaemonManagerPanel.tsx:90-143`).

**When:** Whenever `SetSessionBrowse` is called AND web-share is currently ON.

**Frontend pattern (from DaemonManagerPanel.tsx lines 90-143):** [VERIFIED: direct codebase read]
```typescript
// Toggle browse: call SetSessionBrowse → then IssueCapabilities to refresh URLs
async function handleToggleBrowse(sessionId: string, enabled: boolean): Promise<void> {
    await SetSessionBrowse(sessionId, enabled)
    if (webEnabled[sessionId]) {
        const resp = await IssueCapabilities(sessionId)
        // update share state with new RO+RW URLs
    }
}
```

### Pattern 4: Share Modal Structure (SessionShareModal)

**What:** A dedicated modal component following the HubModal overlay pattern (see `HubModal.tsx`).

**Architecture:** Not a full HubModal (which hosts a TerminalPanel). Instead: a simpler overlay + dialog that hosts:
- "Share the session" toggle (wires to `ToggleWebServing`)
- `SessionSharePanel` (when sharing ON) — simplified version
- "Enable remote file browsing" toggle (wires to `SetSessionBrowse`, disabled when sharing OFF)
- `HomeDirWriteWarning` (when `session.homeDir` is true and browse is being enabled)
- LAN Basic Auth password block (when `webServerMode === 'local'`)

**Modal dismissal:** Click-outside + Escape key, same as HubModal. Focus trap not strictly required (no terminal inside), but still good practice. Focus returns to the Share button on close.

### Pattern 5: Remote Peer Disabled State (D-13)

**What:** Share button on remote peer cards is visible but disabled with lock icon + tooltip. Colorblind-safe: not color alone.

**Detection:** `SessionCard` already derives `isLocal` from `!hostname || hostname === ''`. Remote peer cards have `hostname` set. [VERIFIED: SessionCard.tsx:157-158]

```tsx
// Source: SessionCard.tsx — new Share button, colorblind-safe disabled state
<button
  type="button"
  className="hub-card__share"
  onClick={(e) => { e.stopPropagation(); onShare?.(session) }}
  disabled={!isLocal}
  aria-label={isLocal ? `Share ${name}` : 'Only the session owner can share'}
  title={isLocal ? 'Share session' : 'Only the session owner can share'}
>
  {/* COLORBLIND-SAFE: LockClosedIcon (shape) + text label carry state; color is reinforcement only */}
  {!isLocal && <LockClosedIcon aria-hidden="true" />}
  Share
</button>
```

### Pattern 6: Server-Truth Seeding (SHARE-05)

**What:** When the Share modal opens, it must seed its state from the server, not local assumptions. This matches the SHARE-05 requirement ("server-truth seeding").

**Mechanism:** The existing reconcile effect in `DaemonManagerPanel.tsx:176-233` handles this by calling `IssueCapabilities` for any session that is `webEnabled` but missing a share entry. The Share modal should follow the same pattern: on open, if `session.webEnabled` is true and no share entry is cached, call `IssueCapabilities` to fetch current state.

**Browse state seeding:** `session.webEnabled` is already surfaced via `SessionInfo.WebEnabled`. Browse state is currently NOT surfaced in `SessionInfo` (it's in-memory engine state). Two options (Claude's discretion):
1. Add `browseEnabled bool` to `SessionInfo` so the modal can read it on open
2. The modal reads browse state from a new Wails binding / daemon API endpoint on open

Option 1 is simpler and consistent with how `FilesWrite bool` was added to `SessionInfo` in Phase 124 (types.go:33). [VERIFIED: types.go:20-35]

### Anti-Patterns to Avoid

- **Substring-match on perms:** Never `strings.Contains(claims.Perms, "files.read")` — always `capability.HasPerm(claims.Perms, capability.PermFilesRead)`. The static-grep gate in `TestHasPerm_NoStringsContains_Write` at capability_test.go:780-796 will catch violations. [VERIFIED: capability_test.go:774-796]
- **Rebuilding SessionSharePanel from scratch:** D-11 explicitly requires reuse. The existing component has proven cap/URL/QR/password lifecycle; rebuilding risks regressing SHARE-05 behaviors.
- **Putting browse state in the cap token beyond the injected perms:** The cap already carries the perms; the browse flag is only needed at issuance time. Don't embed it redundantly.
- **Applying the CSRF Origin check to the browse-toggle endpoint:** The browse toggle is a daemon-socket API call (not a webserver route), so no CSRF applies. Only the five webserver write routes need CSRF.
- **Adding `files.write` to `requireFilesRead`:** The SEPARATION INVARIANT at capability_mw.go:96-98 must be preserved — `requireFilesRead` and `requireFilesWrite` are separate wrappers, not combined. The new coupling is in the perm INJECTION (at issuance time), not in the enforcement layer.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cap token generation | New token structure | Existing `capability.Sign` + `Claims{Perms: ...}` | HMAC-SHA256, field-order-load-bearing JSON; format is audited |
| Join code issuance | New short-code scheme | Existing `JoinCodeManager.Issue()` | Single-use 5-min TTL, 40-bit entropy, TOCTOU-safe Exchange |
| QR code generation | QR library | Existing `GetCapabilityQRCode` Wails binding | Already integrated |
| Clipboard copy | `navigator.clipboard` | Existing `ClipboardSetText` Wails runtime | Cross-platform; works in desktop webview |
| Modal overlay + animation | Custom CSS animation state machine | Follow `HubModal.tsx` overlay + `hub-modal-overlay--entering/open/exiting` phase machine | Already A11Y-compliant (inert trap, focus return, Escape handler) — though for a simpler Share modal without a TerminalPanel, a leaner version without the grow animation is acceptable |
| perm check in enforcement layer | `strings.Contains` | `capability.HasPerm` | Pitfall 4 — `"no-files.read"` would false-positive |
| Home-dir detection | Manual path compare | Existing `sessionCwdIsHome()` (engine.go) + `session.homeDir` field in `SessionInfo` | EvalSymlinks already handles macOS `/var→/private/var` trap |

---

## Runtime State Inventory

This phase is not a rename/refactor/migration. The only runtime state to consider:

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | Settings files (`~/.config/agenthub/settings.json` or platform equivalent) may have `"filesRead": false` set by users | Code change: remove `filesRead` field from `daemonSettings` struct; existing settings files with the field will silently ignore it (Go's JSON Unmarshal ignores unknown fields in the struct, NOT in the JSON — but since we're removing the field FROM the struct, the JSON key will just be ignored on next read). No migration needed. |
| Live service config | None — browse state is in-memory only (D-08) | None |
| OS-registered state | None | None |
| Secrets/env vars | None | None |
| Build artifacts | None | None |

**Settings file note:** The `filesRead` field in `daemonSettings` (engine.go:108) is serialized via `json:"filesRead,omitempty"`. Removing the field from the struct means it's no longer written or read. Users who had `"filesRead": false` in their settings file (disabling the global file-browser cap) will find that setting silently no longer applies — their first IssueCapabilities call after the upgrade will mint caps WITHOUT `files.read` unless they enable the per-session browse toggle. This is the intended behavior (D-07). No migration needed, but a code comment noting the deliberate removal should be included.

---

## Common Pitfalls

### Pitfall 1: Stale Cap URLs After Browse Toggle

**What goes wrong:** If browse is toggled ON after IssueCapabilities was already called for this session (with browse OFF), the displayed RO URL still carries `"read"` perms (no files.read). The web file-browse surface will 403 with "files.read capability required".

**Why it happens:** `issueCapabilitiesForSession` snapshots perm state at call time. The existing reconcile effect in DaemonManagerPanel handles this by calling IssueCapabilities AFTER write-toggle (lines 103-116). The new modal must do the same: toggle browse → if web-share is ON → re-issue caps.

**How to avoid:** Always follow `SetSessionBrowse(true)` with `IssueCapabilities` when `webEnabled[sessionId]` is true. This is the same pattern as the existing `handleToggleFilesWrite` in DaemonManagerPanel.tsx:90-143.

**Warning signs:** Web file browse returns 403 "files.read capability required" even though the owner enabled browse.

### Pitfall 2: files.write Appearing in the RO Token

**What goes wrong:** A coding error that appends both `files.read` AND `files.write` to the RO token when browse is ON would give read-only viewers write access to the filesystem.

**Why it happens:** Browse ON logic writes both perms into `wPerms`; if the wrong variable is used for `rClaims`, the RO token gets write perms.

**How to avoid:** D-04 is explicit: Browse ON → `rPerms = "read,files.read"` (no files.write). The unit test `TestIssueCapabilities_BrowseOn_RO_GetsFilesReadOnly` (must be written) pins this. The `requireFilesWrite` middleware in capability_mw.go:147-170 enforces `files.write` at access time regardless — but defense in depth means the token should not carry it in the first place.

**Warning signs:** Automated test `TestIssueCapabilities_BrowseOn_RO_GetsFilesReadOnly` fails; or web file-write routes respond 200 to an RO-code web visitor.

### Pitfall 3: CAP-05 Opt-In State Leaking Into the New Modal

**What goes wrong:** `SessionSharePanel` currently has `ownerWriteEnabled` prop + `allowFileEditing` state + the two-gate confirm flow. If the simplification removes `ownerWriteEnabled` but leaves the inner `allowFileEditing` state, the write link row will be permanently locked.

**Why it happens:** The existing tests for `SessionSharePanel` at `SessionSharePanel.test.tsx` test the CAP-05 behavior. After stripping, those tests must be updated — if they're left as-is they'll fail on the new simpler panel.

**How to avoid:** D-11 says "strip the now-dead CAP-05 two-gate write UI". The `ownerWriteEnabled` prop, `allowFileEditing` state, `showWriteConfirm` state, `surfaceWriteLink` derived value, and the opt-in row JSX block all need removal. The write link should always be surfaced (when web-share is ON and caps are available) with no viewer gate. The existing SessionSharePanel tests for CAP-05 must be updated to match the new simplified API.

**Warning signs:** Full Access Link row shows "Enable `Allow file editing` above" even with working browse ON; or `SessionSharePanel.test.tsx` tests fail after stripping.

### Pitfall 4: strings.Contains on Perms (Whole-Token Semantics)

**What goes wrong:** Using `strings.Contains(claims.Perms, "files.read")` allows `"no-files.read"` to match.

**Why it happens:** Copy-paste from non-permissions code; incorrect intuition about comma-separated strings.

**How to avoid:** Always use `capability.HasPerm(claims.Perms, capability.PermFilesRead)`. The static-grep gate `TestHasPerm_NoStringsContains_Write` catches violations at the webserver level. [VERIFIED: capability_test.go:774-796]

**Warning signs:** Static-grep test failure; security audit finding.

### Pitfall 5: Global filesRead Removal Breaking Existing Owner Cap Tests

**What goes wrong:** The four existing tests `TestIssueCapabilities_*` at api_test.go:1982-2115 test the old `filesReadEnabled()` global logic. After the global is removed and replaced with `browseEnabledFor()`, these tests will fail.

**Why it happens:** Tests were written for Phase 118's global logic. The new per-session browse flag changes the semantics entirely.

**How to avoid:** Retire the four old tests and replace with new tests for the four D-03/D-04 matrix cells. Old test names: `TestIssueCapabilities_OwnerHasFilesRead_WhenSettingNil`, `TestIssueCapabilities_ViewerNoFilesRead`, `TestIssueCapabilities_OwnerNoFilesReadWhenDisabled`, `TestIssueCapabilities_OwnerHasFilesReadWhenExplicitTrue`. New tests replace them with the browse-on/off × RO/RW matrix.

**Warning signs:** Test suite fails on removed global logic after engine.go cleanup.

### Pitfall 6: Remote Peer Share Button Not Stopping Click Propagation

**What goes wrong:** If the Share button click propagates to the card's `onClick` handler, it triggers `onCardClick` which opens the Hub session modal.

**Why it happens:** `SessionCard` has an `onClick` on the `<article>` with guards for specific child classes. The Share button must either be in the guarded set or use `e.stopPropagation()`.

**How to avoid:** Follow the same `e.stopPropagation()` pattern as the existing Open button (SessionCard.tsx:392-402). Add the Share button's class to the `onClick` guard (e.g., `target.closest('.hub-card__share')`).

**Warning signs:** Clicking Share on a card opens the Hub interactive modal instead of the Share modal.

---

## Code Examples

### Existing Perm Injection (to be replaced)

```go
// Source: internal/daemon/api.go:1097-1116 — CURRENT implementation
ownerPerms := "read,write"
if a.engine.filesReadEnabled() {
    ownerPerms = "read,write," + capability.PermFilesRead
}
if a.engine.filesWriteEnabledFor(sessionID) {
    ownerPerms += "," + capability.PermFilesWrite
}
rClaims := capability.Claims{SID: sessionID, Perms: "read", IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
wClaims := capability.Claims{SID: sessionID, Perms: ownerPerms, IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}
```

### Existing RequireFilesRead Enforcement (unchanged — not an edit site)

```go
// Source: internal/webserver/capability_mw.go:102-119 — NOT modified this phase
func (ws *WebServer) requireFilesRead(next http.HandlerFunc) http.HandlerFunc {
    return ws.requireCapability(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := capability.ClaimsFromContext(r.Context())
        if !ok {
            http.Error(w, "files.read capability required", http.StatusForbidden)
            return
        }
        if !capability.HasPerm(claims.Perms, capability.PermFilesRead) {
            http.Error(w, "files.read capability required", http.StatusForbidden)
            return
        }
        next(w, r)
    })
}
```

### Existing LAN Password Flow in DaemonManagerPanel (to be migrated)

```typescript
// Source: frontend/src/components/DaemonManagerPanel.tsx:70-88
// Migrate this pattern into the new SessionShareModal
const [lanPassword, setLanPassword] = useState('')
useEffect(() => {
  if (webServerMode === 'local' && webServerRunning) {
    GetLocalNetworkPassword().then(setLanPassword).catch(() => setLanPassword(''))
  } else {
    setLanPassword('')
  }
}, [webServerMode, webServerRunning])
```

### Existing Reconcile Effect (to be migrated/adapted)

```typescript
// Source: frontend/src/components/DaemonManagerPanel.tsx:176-233
// The Share modal's cap-fetch on open should follow this pattern:
// - If session is webEnabled but no share entry cached → call IssueCapabilities
// - If share entry present but session no longer webEnabled → clear it
// - On webServerRunning toggle (restart) → clear all cached share entries
useEffect(() => {
  // for every webEnabled session missing a share entry, fetch capabilities
  ...
}, [sessions, webEnabled, webServerRunning])
```

### HasPerm Usage (required pattern throughout)

```go
// Source: internal/capability/capability.go:51-61 — VERIFIED
// Always use HasPerm, never strings.Contains
capability.HasPerm(claims.Perms, capability.PermFilesRead)  // correct
capability.HasPerm(claims.Perms, capability.PermFilesWrite) // correct
strings.Contains(claims.Perms, "files.read")                // WRONG — pitfall 4
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Global `filesReadEnabled()` — all owner tokens get files.read if global ON | Per-session `browseEnabledFor()` — only if owner explicitly enables browse for this session | Phase 137 (this phase) | Browse is now OFF by default per session; no global kill-switch needed |
| `filesWriteEnabledFor()` / `SetSessionFilesWrite()` — per-session two-gate write | Browse toggle subsumes write toggle — RW code inherits files.write automatically when browse ON | Phase 137 (this phase) | Removes viewer "Allow file editing" confirm step; owner act of enabling browse + handing out RW code IS the consent model |
| CAP-05 viewer opt-in — second gate on top of owner enable | Removed — no viewer-side confirm for file access | Phase 137 (this phase) | Simpler mental model: code type determines access level; browse toggle is the sole owner-side gate |
| DaemonManagerPanel (Sessions page) as the web-share control surface | Per-card Share modal on Hub | Phase 137 (this phase) | Sessions page is being removed in Phase 138; Share modal must exist first |

**Deprecated/outdated in this phase:**
- `filesReadEnabled()` function — removed from engine.go
- `filesWriteEnabledFor()` / `SetSessionFilesWrite()` — subsumed by browse toggle
- `filesRead *bool` / `filesWriteDefault bool` / `sessionWrites map[string]bool` fields — removed from SessionEngine
- `SetSessionFilesWrite` Wails binding — retired from app.go
- `ownerWriteEnabled` prop + `allowFileEditing` state + two-gate confirm flow in `SessionSharePanel` — stripped
- CAP-05 behavior in `SessionSharePanel.test.tsx` tests — must be replaced with new browse-oriented tests

---

## Deliberate Security Reversals — Documented for Secure-Phase Audit

The following are intentional changes to the security model. The `secure-phase` auditor must review each against the threat model.

### Reversal 1: T-124-07 (write no longer separately gated)

**Old model:** Per-session `SetSessionFilesWrite` was required before `files.write` could be injected. Justified by T-124-07 (never a global flag for write).

**New model:** When browse is ON and the RW code is presented, `files.write` is injected automatically. The owner's act of enabling browse IS the write-enable for RW-code holders.

**Justification (D-02):** The RW code already grants complete terminal control (send arbitrary input to a running shell). `files.write` does not escalate beyond that. The owner generates and distributes the RW code knowingly.

**Audit question:** Is granting `files.write` implicitly (without a separate per-share confirmation per the old T-124-07) acceptable given that the RW code holder already has full PTY control?

### Reversal 2: CAP-05 viewer confirm removed

**Old model:** Even with owner write enabled, the viewer had to check "Allow file editing" — a second gate preventing accidental write access.

**New model:** No viewer-side gate. The code type (RO vs RW) fully determines access.

**Justification (D-02):** Code type is already the consent model for terminal access. Adding a separate file-write gate was inconsistent with the fact that the RW code already granted full terminal input (which can write any file accessible from the shell).

**Audit question:** Are there threat scenarios where a viewer would receive a RW code without realizing it grants file write access?

### Reversal 3: CAP-08 / global files.read removed (D-07)

**Old model:** `filesReadEnabled()` was a global admin kill-switch. Setting `filesRead: false` in settings prevented files.read from being injected into ANY owner token across ALL sessions.

**New model:** No global kill-switch. Per-session browse toggle is the sole gate.

**Justification (D-07):** The global setting was a blunt instrument not visible in the per-session share UI. Users who needed to disable file browsing for all sessions would have had to know about and edit `settings.json` directly (no UI for this was found in the codebase). The per-session default OFF (D-06) provides equivalent protection.

**Audit question:** Were there deployments using `"filesRead": false` in settings as an administrative control? If so, that control is lost.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Global `filesReadEnabled` has no Settings UI toggle (no PATCH endpoint in api.go, no frontend control found) | Architecture Patterns, Common Pitfalls | If there IS a frontend toggle for it, it must also be removed from the Settings UI (D-07). Codebase search found no such UI — only the engine.go field and api.go consumption. |
| A2 | `SessionInfo.browseEnabled` field (if added) doesn't require daemon protocol versioning | Code Examples | If existing clients (CLI, web surface) break on an unexpected field, proto-version bump may be needed. Go's `json:",omitempty"` handles gracefully. |
| A3 | Phase 138 removes DaemonManagerPanel entirely; Phase 137 only needs to add the Share modal | Architecture | If Phase 138 is delayed, DaemonManagerPanel will have the old web-share UI running alongside the new Share modal. Coordination needed if both phases run in parallel. |

**All other claims were verified by direct codebase inspection in this session.**

---

## Open Questions (RESOLVED)

1. **SetSessionBrowse endpoint design**
   - What we know: `SetSessionFilesWrite` uses `POST /sessions/{id}/files-write` with `SessionFilesWriteRequest{Enabled bool}` (types.go:130-134; client.go:326-328).
   - What's unclear: Should `SetSessionBrowse` be a new endpoint (`POST /sessions/{id}/browse`) or should `SetSessionFilesWrite` be renamed/repurposed? The CONTEXT says "subsume" — repurposing is cleaner but the client.go method rename breaks the CLI if it calls `SetSessionFilesWrite` directly.
   - Recommendation: Add new `POST /sessions/{id}/browse` endpoint + new client method `SetSessionBrowse`. Deprecate `POST /sessions/{id}/files-write` by removing the handler (it's not needed any more) OR leave a redirect stub. Planner decides based on CLI surface audit.
   - **RESOLVED:** New `POST /sessions/{id}/browse` endpoint + `SetSessionBrowse` client method — see Plan 137-02 (`internal/daemon/api.go`, `client.go`, `types.go`).

2. **browseEnabled in SessionInfo wire type**
   - What we know: Browse state is ephemeral (D-08). `FilesWrite bool` was added to `SessionInfo` so the frontend could seed local toggle state (DaemonManagerPanel.tsx:158). Browse needs the same seeding for the Share modal.
   - What's unclear: Add `BrowseEnabled bool` to `SessionInfo` (server truth on ListSessions poll), or have the modal call a separate API on open?
   - Recommendation: Add `BrowseEnabled bool json:"browseEnabled"` to `SessionInfo` (types.go). The modal seeds from `session.browseEnabled` on open, matching the existing `filesWrite` seeding pattern. Mark `json:",omitempty"` is NOT appropriate here (false would be omitted); use plain `bool`.
   - **RESOLVED:** `BrowseEnabled bool` added to `SessionInfo` (server truth on ListSessions poll) — see Plan 137-02 (`internal/daemon/types.go`); the Share modal seeds `browseEnabled` from it in Plan 137-03.

3. **SessionSharePanel prop cleanup scope**
   - What we know: The panel currently takes `ownerWriteEnabled bool` which drives the CAP-05 gate. This prop and all downstream state must be removed (D-11).
   - What's unclear: Should `SessionSharePanel` gain a `browseEnabled` prop to display different link scope text (e.g., "Watch only — no file access" vs "Watch + browse files"), or should that logic live in the modal wrapper?
   - Recommendation: Add a `browseEnabled bool` prop to `SessionSharePanel` for scope text only. The panel itself doesn't call any APIs; it just renders URLs and codes it receives. The modal drives the browse toggle and passes `browseEnabled` as a display hint.
   - **RESOLVED:** `browseEnabled?` prop added to `SessionSharePanel` (scope text only) — see Plan 137-03 Task 1; the SessionShareModal owns the toggle and passes it as a display hint.

---

## Environment Availability

No external dependencies beyond the existing project stack.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Backend changes | Yes | (project's existing) | — |
| pnpm | Frontend build/test | Yes | (project's existing) | — |
| vitest | Frontend unit tests | Yes | 4.1.0 | — |
| jsdom | Frontend test env | Yes | (vite.config.ts confirmed) | — |

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Go tests | `go test ./internal/daemon/... -run TestIssueCapabilities` |
| Go webserver tests | `go test ./internal/webserver/... -run TestRequireFiles` |
| Frontend unit tests | `cd frontend && pnpm test` (vitest run) |
| Quick Go run | `go test ./internal/daemon/... -run TestIssueCapabilities -v` |
| Full suite | `go test ./... && cd frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SHARE-01 | Share button on Hub card opens modal | Frontend component | `pnpm test SessionCard` | ❌ Wave 0: `SessionCard.share.test.tsx` |
| SHARE-02 | "Share the session" toggle reveals RO + RW links/codes | Frontend component | `pnpm test SessionShareModal` | ❌ Wave 0: `SessionShareModal.test.tsx` |
| SHARE-03 / D-03 | Browse OFF: RO token = `"read"`, RW token = `"read,write"` | Go unit (api_test.go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOff` | ❌ Wave 0 |
| SHARE-03 / D-04 RO | Browse ON, RO token = `"read,files.read"` only | Go unit (api_test.go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_RO` | ❌ Wave 0 |
| SHARE-03 / D-04 RW | Browse ON, RW token = `"read,write,files.read,files.write"` | Go unit (api_test.go) | `go test ./internal/daemon/... -run TestIssueCapabilities_BrowseOn_RW` | ❌ Wave 0 |
| SHARE-03 cross-surface | Web file browse 403s on RO cap without files.read (browse OFF) | Go unit (webserver) | `go test ./internal/webserver/... -run TestRequireFilesRead` | ✅ Exists |
| SHARE-03 cross-surface | Web file browse 200 on RO cap with files.read (browse ON) | Go unit (webserver) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_BrowseOn` | ❌ Wave 0 |
| SHARE-03 cross-surface | Web file write 403 on RO cap even with browse ON (no files.write) | Go unit (webserver) | `go test ./internal/webserver/... -run TestFilesRoutes_RO_NoWrite` | ❌ Wave 0 |
| SHARE-04 | LAN password visible in modal in local mode | Frontend component | `pnpm test SessionShareModal` (local mode fixture) | ❌ Wave 0 |
| SHARE-04 | QR codes copyable per link row | Frontend component | `pnpm test SessionSharePanel` (existing, keep passing) | ✅ Exists |
| SHARE-05 regression | Server-truth seeding on modal open (webEnabled + caps fetched) | Frontend component | `pnpm test SessionShareModal` (webEnabled=true fixture) | ❌ Wave 0 |
| SHARE-05 regression | stale-URL cleared on web-server restart (P-2 from DaemonManagerPanel) | Frontend component | `pnpm test SessionShareModal` (server-restart fixture) | ❌ Wave 0 |
| SHARE-06 | Remote peer card Share button is disabled | Frontend component | `pnpm test SessionCard` (remote fixture) | ❌ Wave 0 |
| D-02 removal | `ownerWriteEnabled` prop + CAP-05 state stripped from SessionSharePanel | Frontend component | `pnpm test SessionSharePanel` (must NOT find AllowFileEditing toggle) | ⚠️ Existing tests test the OLD behavior — must be updated |
| D-07 removal | `filesReadEnabled()` no longer called in perm injection | Go unit + static grep | `go test ./internal/daemon/... -run TestIssueCapabilities` | ❌ Wave 0 (existing tests retired) |
| D-09 | Home-dir warning appears when session.homeDir=true + browse being enabled | Frontend component | `pnpm test SessionShareModal` (homeDir fixture) | ❌ Wave 0 |
| D-13 colorblind | Remote card Share button: lock icon + tooltip (not color alone) | Source inspection | `pnpm test SessionCard` — verify COLORBLIND-SAFE comment in source | ❌ Wave 0 |

### Security-Delta Test Map (Audit Target)

The following tests are specifically for the security changes and are the primary audit targets for the later `/gsd:secure-phase` pass:

| Security Delta | What to Test | Test Name | Layer |
|----------------|-------------|-----------|-------|
| RO token gets files.read when browse ON (never files.write) | `rPerms = "read,files.read"` exactly | `TestIssueCapabilities_BrowseOn_ROPermsExact` | Go unit |
| RW token gets files.read + files.write when browse ON | `wPerms = "read,write,files.read,files.write"` exactly | `TestIssueCapabilities_BrowseOn_RWPermsExact` | Go unit |
| Browse OFF means neither token gets any file perm | `rPerms = "read"`, `wPerms = "read,write"` exactly | `TestIssueCapabilities_BrowseOff_NoFilesPerms` | Go unit |
| RO cap with files.read cannot reach files.write routes | requireFilesWrite returns 403 | `TestRequireFilesWrite_RO_BrowseOn_Still403` | Go unit (webserver) |
| RW cap with files.write CAN reach files.write routes | requireFilesWrite returns 200 | `TestRequireFilesWrite_RW_BrowseOn_200` | Go unit (webserver) |
| No substring-match on perms in new code | Static grep | `TestHasPerm_NoStringsContains_Browse` | Go static grep |
| CSRF check still applies to webserver write routes after this change | requireFilesWrite still calls originAllowedForWrite | Existing `TestRequireFilesWrite` (keep passing) | Go unit (webserver) |

### Sampling Rate

- **Per task commit:** `go test ./internal/daemon/... -run TestIssueCapabilities && go test ./internal/webserver/... -run TestRequireFiles && cd frontend && pnpm test SessionSharePanel SessionShareModal`
- **Per wave merge:** `go test ./... && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/daemon/api_test.go` — retire 4 old `TestIssueCapabilities_*` tests; add 3 new browse-matrix tests
- [ ] `internal/webserver/files_routes_test.go` — add `TestFilesRoutes_RO_BrowseOn` (RO cap + files.read → 200) and `TestFilesRoutes_RO_NoWrite` (RO cap + files.read but no files.write → 403 on write route)
- [ ] `frontend/src/components/__tests__/SessionCard.share.test.tsx` — Share button renders; click fires onShare; disabled on remote peer; click does not bubble to onCardClick
- [ ] `frontend/src/components/__tests__/SessionShareModal.test.tsx` — Share toggle; browse toggle; LAN password; homeDir warning; server-truth seeding on open; stale URL cleared on server restart
- [ ] `frontend/src/components/__tests__/SessionSharePanel.test.tsx` — update existing tests: remove CAP-05 opt-in tests; add simplified panel tests (write link always shown when sharing ON)

---

## Security Domain

`security_enforcement` is not set to false; included per default.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Yes (capability token) | HMAC-SHA256 via `capability.Sign`/`Verify` — existing, unchanged |
| V3 Session Management | Yes (per-session caps) | Grant list + `ClearGrants` on toggle-off — existing, unchanged |
| V4 Access Control | Yes (RO vs RW file browse) | `requireFilesRead`/`requireFilesWrite` middleware — existing, enforcement unchanged; PERM INJECTION is the change |
| V5 Input Validation | Partial | `browseEnabled` is a boolean toggle from owner UI only; no external string input in perm path |
| V6 Cryptography | Yes | HMAC-SHA256 key via `KeyStore`; constant-time comparison via `hmac.Equal` — existing, unchanged |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation | Status after Phase 137 |
|---------|--------|---------------------|------------------------|
| Token privilege escalation (RO→RW) | Elevation of Privilege | `capability.HasPerm` whole-token semantics; grant-list revocation | Unchanged — enforced at webserver layer regardless of what browser presents |
| File browse without explicit owner consent | Elevation of Privilege | Per-session browse toggle default OFF (D-06) | NEW: browse toggle is the sole gate. Owner must explicitly enable. |
| RO code holder writing files | Elevation of Privilege | RO code never gets `files.write` in D-03/D-04 matrix | NEW test required: `TestIssueCapabilities_BrowseOn_RO_NoFilesWrite` |
| CSRF on webserver write routes | Tampering | `originAllowedForWrite` check in `requireFilesWrite` | Unchanged — not affected by this phase |
| Stale cap after share toggle-off | Repudiation | `ClearGrants` on toggle-off + `isGrantActive` check | Unchanged — browse toggle-off should also clear grants (follow `handleWebServe` pattern) |
| Viewer confirms "Allow file editing" for a session they don't own | Spoofing | N/A — CAP-05 viewer gate is being removed | Not applicable; removal eliminates this attack surface (which was a UI-only gate, not a backend one) |

**Critical security invariant for secure-phase audit:**
The enforcement in `capability_mw.go` (`requireFilesRead`, `requireFilesWrite`) is token-content-driven and NOT changed by this phase. The change is ONLY in what perms are injected at issuance time (`issueCapabilitiesForSession`). Any browser that has an old RO token issued before browse was enabled cannot gain file access — the token carries `"read"` only, and `requireFilesRead` will 403. This is the correct failure mode. [VERIFIED: capability_mw.go:102-119]

---

## Sources

### Primary (HIGH confidence — direct codebase inspection)
- `internal/capability/capability.go:15-61` — Claims struct, PermFilesRead/Write constants, HasPerm semantics
- `internal/capability/joincode.go:33-93` — JoinCodeManager, Issue/Exchange atomicity
- `internal/daemon/api.go:1033-1146` — handleWebServe + issueCapabilitiesForSession (primary edit site)
- `internal/daemon/engine.go:44-633` — SessionEngine fields, filesReadEnabled, filesWriteEnabledFor, SetSessionFilesWrite, sessionCwdIsHome
- `internal/daemon/types.go:20-152` — SessionInfo struct, IssueCapabilitiesResponse, SessionFilesWriteRequest
- `internal/daemon/client.go:275-356` — DaemonClient methods: ToggleWebServing, SetSessionFilesWrite, IssueCapabilities, GetLocalNetworkPassword
- `internal/webserver/capability_mw.go:37-198` — requireCapability, requireFilesRead, requireFilesWrite, originAllowedForWrite
- `internal/webserver/files_routes_test.go` — existing RO/RW files.read enforcement test patterns
- `internal/daemon/api_test.go:1982-2115` — existing issueCapabilitiesForSession tests (to be retired/replaced)
- `internal/webserver/capability_test.go:434-796` — requireFilesRead, requireFilesWrite, static-grep HasPerm gate
- `frontend/src/components/SessionSharePanel.tsx` — full component read; CAP-05 two-gate to strip
- `frontend/src/components/Hub/SessionCard.tsx` — full component read; Share button insertion point
- `frontend/src/components/DaemonManagerPanel.tsx` — full component read; source of web-serve/cap/LAN-password flow to migrate
- `frontend/src/components/Hub/HubModal.tsx` — overlay + animation pattern for new modal
- `frontend/src/components/__tests__/SessionSharePanel.test.tsx` — existing tests (must be updated)
- `app.go:784-915` — Wails bindings for ToggleWebServing, SetSessionFilesWrite, IssueCapabilities, GetLocalNetworkPassword
- `.planning/REQUIREMENTS.md` — authoritative SHARE-01..06 text
- `.planning/phases/137-share-modal-cap-model/137-CONTEXT.md` — locked decisions D-01..D-13

### Secondary (MEDIUM confidence)
- `.planning/milestones/v3.5-phases/127-web-share-write-security-hardening/127-CONTEXT.md` — capability-escalation audit baseline; T-124-07, CAP-05, CAP-08 background

---

## Metadata

**Confidence breakdown:**
- Cap model edit site (api.go:1095-1116): HIGH — exact lines read and verified
- Engine.go collapse (filesRead/sessionWrites → sessionBrowse): HIGH — existing fields and methods verified
- Frontend simplification (SessionSharePanel): HIGH — full component read; CAP-05 gate fully understood
- Modal architecture (new SessionShareModal): MEDIUM — pattern is established from HubModal but new file
- Settings global removal: HIGH — confirmed no frontend UI exists for global filesRead toggle

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable Go/React/Wails stack)
