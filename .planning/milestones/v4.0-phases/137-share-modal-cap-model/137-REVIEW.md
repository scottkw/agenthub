---
phase: 137-share-modal-cap-model
reviewed: 2026-06-20T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - app.go
  - frontend/src/App.tsx
  - frontend/src/components/Hub/SessionCard.tsx
  - frontend/src/components/Hub/SessionShareModal.tsx
  - frontend/src/components/SessionSharePanel.tsx
  - internal/daemon/api.go
  - internal/daemon/client.go
  - internal/daemon/engine.go
  - internal/daemon/types.go
findings:
  critical: 1
  warning: 6
  info: 4
  total: 11
status: issues_found
---

# Phase 137: Code Review Report

**Reviewed:** 2026-06-20
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Phase 137 collapses the global `filesRead` kill-switch + per-session `files.write` two-gate into a single per-session browse toggle that drives the D-03/D-04 capability/permission matrix, adds a `POST /sessions/{id}/browse` daemon endpoint + Wails binding, and ships a per-card Share modal.

The core security-perm injection logic in `issueCapabilitiesForSession` (api.go:1116-1121) is correct: it uses whole-token perm strings, never leaks `files.write` into the RO token, and the RO-never-writes invariant is pinned by `files_routes_test.go`. The `ClearGrants`-on-toggle stale-cap mitigation is wired in both `handleWebServe` and the new `handleSetSessionBrowse`.

However, there is one **BLOCKER**: the per-session `sessionBrowse` map is never cleaned up on `KillSession`, which both leaks memory and — combined with the daemon's session-ID reuse path — can resurrect a browse-ON permission for a different session that happens to reuse the ID, directly undermining the SHARE-05 stale-cap threat model the phase set out to close. Several WARNINGs concern React effect double-issuance, missing request-body size limits on the new endpoint, and an inverted scope-text copy bug in the share panel.

## Critical Issues

### CR-01: `sessionBrowse` map entry is never deleted on session kill — stale browse-grant resurrection + memory leak

**File:** `internal/daemon/engine.go:495-513` (`KillSession`); map declared at `engine.go:45`, set at `engine.go:615-622`
**Issue:**
`KillSession` deletes every other per-session map entry — `tabNames`, `sessionCLIs`, `sessionWorkDirs` (under `e.mu`) and `sessionStatuses` (under `e.statusMu`) — but does **not** delete `e.sessionBrowse[id]`:

```go
e.mu.Lock()
delete(e.tabNames, id)
delete(e.sessionCLIs, id)
delete(e.sessionWorkDirs, id) // Phase 118 / FS-02
// sessionBrowse[id] is NOT deleted  ← bug
e.mu.Unlock()
```

Two consequences:

1. **Unbounded memory leak.** Every session that ever had browse toggled (`SetSessionBrowse` writes the key for both ON *and* OFF — `e.sessionBrowse[sessionID] = enabled`) leaves a permanent map entry. A long-running daemon churning sessions accumulates these forever.

2. **Stale browse-ON resurrection — the exact SHARE-05 threat this phase exists to close.** `browseEnabledFor` returns `e.sessionBrowse[sessionID]`, and the phase's stale-cap mitigation is built on "absent from map = OFF (D-06 default)". If a session with browse ON is killed and a new session is later created that reuses the same ID (the `onExit`/`runSessionExitCleanup` comment at `api.go:524-526` explicitly worries about "a recycled session ID" inheriting stale grants), the new session silently inherits `browseEnabled = true`. The next `issueCapabilitiesForSession` then mints `files.read`/`files.write` perms for a session whose owner never enabled browsing — a privilege escalation across the session boundary. `ListSessions` would also report `BrowseEnabled: true`, so the GUI modal would seed the toggle ON for a session the user never shared.

This is the same silent-corruption class the file documents elsewhere; the cleanup was simply omitted.

**Fix:** delete the browse entry in `KillSession` alongside the other per-session maps:
```go
e.mu.Lock()
delete(e.tabNames, id)
delete(e.sessionCLIs, id)
delete(e.sessionWorkDirs, id)
delete(e.sessionBrowse, id) // Phase 137 / SHARE-03: clear browse flag so a recycled ID never inherits stale browse-ON
e.mu.Unlock()
```
Add a regression test asserting `browseEnabledFor(id) == false` after kill+recreate with the same ID. Consider also clearing the entry in `runSessionExitCleanup` (api.go:554) for natural exits, mirroring the `ClearGrants` call there.

## Warnings

### WR-01: Browse toggle and server-restart can issue capabilities twice for the same session

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:130-181`
**Issue:** Two effects can each call `IssueCapabilities` for the same render cycle. The restart-clear effect (line 130) sets `setCachedShare(null)` then immediately re-issues when `shareEnabledRef.current` is true. The seeding effect (line 161) re-runs whenever `shareEnabled`/`session.id` change and fires when `cachedShareRef.current === null`. Because the restart effect sets `cachedShare` to `null` synchronously, on a false→true restart transition the seeding effect can observe `cachedShareRef.current === null` on the same commit and launch a *second* concurrent `IssueCapabilities`. Both resolve and call `setCachedShare`, so two grant pairs are registered on the daemon for one visible link — wasted grants that linger until `ClearGrants`/session end, and a last-writer-wins race on which URL the user sees.
**Fix:** Guard the seeding effect against an in-flight issue, or have the restart effect set a sentinel (e.g. set `cachedShareRef.current` to a placeholder before awaiting) so the seeding effect's `!== null` check short-circuits. Alternatively gate the seeding effect on `prevWebServerRunning.current` not being mid-restart.

### WR-02: `handleSetSessionBrowse` has no request-body size limit (inconsistent with peer handlers)

**File:** `internal/daemon/api.go:1288-1306`
**Issue:** Every other body-decoding settings handler caps the body via `http.MaxBytesReader` (e.g. `handleSetPluginSettings` api.go:781, `handleUpdateShellPath` api.go:734, search/web-links/image configs). The new `handleSetSessionBrowse` decodes `r.Body` directly with no cap and no `DisallowUnknownFields`. While the daemon socket is loopback-trust, an oversized or malformed body is no longer bounded, breaking the established defense-in-depth pattern for this surface.
**Fix:**
```go
func (a *API) handleSetSessionBrowse(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    r.Body = http.MaxBytesReader(w, r.Body, 8192)
    var req SessionBrowseRequest
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&req); err != nil { /* 400 */ }
    ...
```

### WR-03: Inverted scope-text copy in read-only link — browse-ON says "cannot browse files"

**File:** `frontend/src/components/SessionSharePanel.tsx:206-210`
**Issue:** The read-only scope text is logically inverted. When `browseEnabled` is **true**, the RO cap actually carries `files.read` (D-04: RO = `read,files.read`), so the read-only viewer *can* browse files. But the copy reads:
```tsx
{browseEnabled
  ? 'Watch the live session — cannot send input or browse files.'   // WRONG: browse IS allowed
  : 'Watch the live session only — cannot send input or browse files.'}
```
Both branches claim "cannot browse files", and the `browseEnabled === true` branch is exactly the case where the RO link grants `files.read`. This misrepresents the actual access the shared link confers — a security-relevant copy bug because the owner is told the RO link is watch-only when it is not.
**Fix:** Swap the copy so the `browseEnabled` branch states the RO link grants read-only file browsing, e.g. `'Watch the live session and browse files (read-only) — cannot send input.'`, and keep the watch-only string for the `!browseEnabled` branch.

### WR-04: Browse toggle re-issue swallows failures and can leave the cap mismatched with the daemon perm matrix

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:205-229`
**Issue:** In `handleBrowseToggle`, after `SetSessionBrowse` succeeds the daemon has already flipped the browse flag *and* cleared all grants (api.go:1295-1304). If the subsequent `IssueCapabilities` throws, the catch sets `setCachedShare(null)` but `browseEnabled` state has already been updated. The SessionSharePanel is then unmounted (because `cachedShare` is null), and the seeding effect will re-issue — but only if `shareEnabled` is still true and the effect re-runs. There is no user-visible error and no retry affordance; the modal can sit with sharing ON, browse toggled, daemon grants cleared, and no live link, until an unrelated state change re-triggers seeding. The `SetSessionBrowse` failure path (outer catch) reverts `browseEnabled` locally but the daemon write may have partially landed.
**Fix:** Surface a user-visible error on re-issue failure and trigger an explicit retry (or force the seeding effect by toggling a retry counter in deps). At minimum, document that a transient failure leaves the modal link-less until the next render.

### WR-05: `homeDir` warning only blocks on dismissal state, not on browse-enable — warning can be bypassed

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:300-332`
**Issue:** The D-09 home-dir write warning renders only when `session.homeDir && !homeDirDismissed`. The "Enable remote file browsing" toggle immediately below is fully interactive regardless of whether the warning has been acknowledged — a user can flip browse ON (granting `files.write` on a home-directory session via the RW link) without ever interacting with the warning, since `homeDirDismissed` is purely cosmetic and dismissing it *hides* the warning rather than gating the toggle. The phase brief calls this a home-dir *write* warning, implying it should gate the dangerous action.
**Fix:** If the warning is meant to gate browse-enable on a home-dir session, disable the browse toggle until the warning is explicitly acknowledged (acknowledge ≠ dismiss-and-hide). If it is purely informational, the naming and placement should be clarified — but confirm with the security-phase intent given the RW link grants `files.write` at `$HOME`.

### WR-06: `prevWebServerRunning` ref defaults to `undefined`, so a modal opened while the server is already down never detects the subsequent restart cleanly

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:123-154`
**Issue:** `prevWebServerRunning` starts `undefined`. `isRestart` is `prevWebServerRunning.current === false && webServerRunning === true`. If the modal mounts with `webServerRunning === false` (server down at open time), the first effect run sets `prev = false` but does not treat mount as a restart (correct). However if the modal mounts with `webServerRunning === undefined` (the prop is optional, `webServerRunning?: boolean`), the comparison `undefined === false` is false on the first transition to `true`, so the very first false-ish→true transition after an undefined initial value is missed. Callers that pass `undefined` (the prop is optional and `App.tsx` does not always thread a defined value) will not get stale-URL clearing on the first restart.
**Fix:** Normalize the prop to a boolean at the component boundary (`const running = webServerRunning ?? false`) and use that consistently for both the ref seed and the `isRestart` computation.

## Info

### IN-01: Dead/retained field `filesWriteDefault` is persisted and round-tripped but never drives behavior

**File:** `internal/daemon/engine.go:44, 109, 194, 221`
**Issue:** `filesWriteDefault` / `daemonSettings.FilesWrite` is loaded, stored, and re-serialized but explicitly "NOT wired to perm injection" — retained solely for `TestSettingsMigration_FilesWriteDefaultsFalse`. Keeping a settings key alive only to satisfy a migration test invites future confusion (a maintainer may wire it back in, reintroducing the global-flag model D-07 removed). Consider replacing the round-trip with a migration-only assertion or a clearly quarantined legacy block.
**Fix:** Add a one-line `// LEGACY — do not wire to perm injection` marker at the field's read site, or migrate the test to assert on raw JSON so the engine field can be dropped.

### IN-02: `SessionShareModal` browse toggle does not reset when sharing is turned OFF

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:186-199`
**Issue:** `handleShareToggle` clears `cachedShare` on OFF but leaves `browseEnabled` untouched. The daemon's browse flag also persists (toggling share off does not call `SetSessionBrowse`). On re-enabling share, the browse toggle visually reflects the prior ON state and the next cap issuance re-grants `files.read`/`files.write` — which may be intended, but is undocumented and surprising relative to the "fresh share" mental model.
**Fix:** Document the intended persistence, or reset `browseEnabled` (and call `SetSessionBrowse(id, false)`) on share-OFF if a clean slate is desired.

### IN-03: Duplicated `SessionInfo` struct between `main` and `daemon` packages drifts silently

**File:** `app.go:29-54` vs `internal/daemon/types.go:20-35`
**Issue:** `main.SessionInfo` and `daemon.SessionInfo` are hand-maintained parallel structs copied field-by-field in `App.ListSessions` (app.go:362-383). The phase added `BrowseEnabled` to both, but nothing enforces parity — a future field added to one and not the other silently drops to zero across the Wails boundary (the very UAT bug class the comments warn about). This is structural, not a phase-137 regression, but the surface grew this phase.
**Fix:** Consider a single shared wire type or a compile-time copy helper. At minimum, a test asserting field-name parity between the two structs.

### IN-04: Magic numbers in `SessionShareModal` inline styles

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:292-296`, `SessionSharePanel.tsx:28, 211-217`
**Issue:** Hardcoded hex colors (`#a9b1d6`, `#16161e`, `#9aa5ce`) and pixel literals in inline `style` objects duplicate values that live in `style.css` design tokens elsewhere. Minor maintainability issue; the user is colorblind and verifies colors at source, so scattering hex literals across inline styles increases the audit surface.
**Fix:** Move to CSS classes / design tokens consistent with the rest of the Hub components.

---

_Reviewed: 2026-06-20_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
