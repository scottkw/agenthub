---
phase: 124-files-write-capability-webserver-write-routes-web-share-opt-in
reviewed: 2026-06-14T00:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - internal/webserver/capability_mw.go
  - internal/webserver/server.go
  - internal/capability/capability.go
  - internal/daemon/api.go
  - internal/daemon/engine.go
  - internal/daemon/plugin_settings.go
  - internal/daemon/remote_files.go
  - internal/daemon/client.go
  - internal/daemon/types.go
  - internal/tui/files.go
  - app.go
  - frontend/src/components/HomeDirWriteWarning.tsx
  - frontend/src/components/SessionSharePanel.tsx
  - frontend/src/components/DaemonManagerPanel.tsx
findings:
  critical: 0
  warning: 4
  info: 4
  total: 8
status: resolved
resolution: fixed
resolution_note: >
  All 4 warnings + 2 info fixed (WR-01 viewer opt-in now gates write-link disclosure —
  closes the SC#4 consent gap; WR-02 conditional-write headers + last-writer-wins doc;
  WR-03 baseURL validation in RemoteCapStore.Put; WR-04 home-dir banner re-sourced to
  SessionInfo.homeDir for cross-surface parity; IN-01 stale-share clearing; IN-03 slice copy).
  Commits 4548c06, 68bf122, 89c744f, 16d0c37, 260647f. go test -race green (14 pkgs),
  frontend 1136 tests green, gofmt+vet+tsc clean. IN-02/IN-04 no-action (documented safe).
---

# Phase 124: Code Review Report

**Reviewed:** 2026-06-14
**Depth:** standard
**Files Reviewed:** 13 (14 incl. DaemonManagerPanel)
**Status:** issues_found

## Summary

This is the security/authorization phase for AgentHub's write surface. I traced every security-critical path the prompt flagged. The core authorization machinery is sound:

- **Capability gating is correct.** All five webserver write routes (`PUT /write`, `POST /upload`, `DELETE /delete`, `POST /rename`, `POST /mkdir`) are wrapped in `requireFilesWrite`, which composes `requireCapability` → `HasPerm(PermFilesWrite)` → `originAllowedForWrite`. `HasPerm` uses whole-token comma-split semantics — no `strings.Contains` perm checks exist anywhere in the reviewed code.
- **The CSRF Origin inversion is implemented correctly.** `originAllowedForWrite` passes vacuously on absent Origin (desktop Wails), strict-matches a present Origin against `BaseURL()`, and fails closed when `BaseURL()` is empty. It does NOT accidentally allow cross-origin (a present, mismatched Origin returns 403).
- **No viewer privilege escalation.** The read token (`rClaims`) is hardcoded `Perms: "read"` and is NEVER mutated by the write toggle. `files.write` is appended only to the owner/write token (`wClaims`), and only when the per-session `filesWriteEnabledFor(sessionID)` is true.
- **The schemaVersion 4 migration defaults `filesWrite` to false** (plain bool, no `*bool` pre-population), correctly inverting the `filesRead` defaults-merge. Migration tests confirm this.
- **The macOS `$HOME` EvalSymlinks comparison** in `cwdEqualsHome` is correct — both sides resolved through `EvalSymlinks`.
- **`sessionWrites` map** is consistently guarded by `e.mu` on every read (`filesWriteEnabledForUnlocked` callers hold RLock) and write (`SetSessionFilesWrite` holds Lock). No data race found.

The findings below are quality/robustness issues, not authorization bypasses. The most material is a dead viewer opt-in control (WR-01) that creates a false security-UX impression, and a write-concurrency/last-writer-wins gap on the remote proxy (WR-02).

## Narrative Findings (AI reviewer)

## Warnings

### WR-01: "Allow file editing" viewer opt-in toggle is dead UI — never reaches the daemon

**File:** `frontend/src/components/SessionSharePanel.tsx:45-93`
**Issue:** The CAP-05 "Allow file editing" viewer opt-in (`allowFileEditing` state, `handleWriteOptinConfirm`) is purely local React state. Confirming it sets `setAllowFileEditing(true)` and nothing else — there is no daemon call, no effect on the issued tokens, and no consumer of `allowFileEditing` anywhere in the file. The full-access (write) link's `files.write` bit is governed entirely by the owner-side `SetSessionFilesWrite` toggle in `DaemonManagerPanel.tsx` and re-issued capabilities; this viewer-side switch changes nothing.

This is a security-UX defect: the prompt's stated invariant is "viewer cap must only get files.write when explicitly opted-in AND owner enabled it." The owner-enabled half is real, but the "explicitly opted-in" half is a no-op control. An owner who toggles owner-write ON immediately exposes a working `files.write` full-access link regardless of whether this viewer switch is ever touched. The toggle implies a second consent gate that does not exist, which is worse than having no toggle (false sense of a second barrier).

**Fix:** Either (a) wire the opt-in to actually gate which URL/token is surfaced (e.g., only reveal the full-access link / its `files.write` token after confirm), or (b) remove the dead control and rely solely on the owner toggle. If the design intent is that owner-enable alone is sufficient, delete `allowFileEditing`/`showWriteConfirm` and their handlers to avoid implying a non-existent consent step. File a GitHub issue if the gating semantics are still open.

### WR-02: Remote write proxy forwards body but has no idempotency/concurrency guard; last-writer-wins across surfaces

**File:** `internal/daemon/remote_files.go:165-261`
**Issue:** `proxyRemoteFiles` forwards `r.Body` opaquely for PUT/POST/PATCH to the remote peer. There is no coordination between concurrent write proxies for the same session — two clients (GUI + TUI, or two browser viewers) issuing `PUT /write` for the same path race, and the remote `files.Handler.Write` performs last-writer-wins with no version/ETag precondition forwarded. The proxy strips `cap` and force-sets `session`, but it does NOT validate or forward any `If-Match`/`If-Unmodified-Since` precondition header, so the read-side `ETag`/`Last-Modified` the proxy faithfully returns on the response path cannot be used by a caller to make a write conditional. The cross-surface-parity memory note ("GUI/TUI/CLI must stay in sync") makes silent overwrite a real risk.

**Fix:** This is acceptable if the design explicitly accepts last-writer-wins (document it), but the body-forwarding block at lines 207-230 only copies `Content-Type`. Forward conditional-write headers (`If-Match`, `If-Unmodified-Since`) when present so a future optimistic-concurrency check on the peer is reachable:
```go
for _, h := range []string{"If-Match", "If-Unmodified-Since", "If-None-Match"} {
    if v := r.Header.Get(h); v != "" {
        req.Header.Set(h, v)
    }
}
```
At minimum, document the last-writer-wins contract in the function comment so downstream surfaces don't assume safe concurrent edits.

### WR-03: Remote proxy 502 path documents "cap token redacted" but DELETE/build-error paths can still 500 with redaction only on dial

**File:** `internal/daemon/remote_files.go:216-235`
**Issue:** The per-status doc block (lines 158-164) promises `upstream dial / TLS failure → 502 + plain text (cap token redacted)`. The build-request-error path at line 218 returns `http.StatusInternalServerError` (500), not 502, while the dial path at line 235 returns 502. Both apply redaction, so there is no token leak — but the build-error 500 is undocumented and inconsistent with the stated contract. More importantly, `http.NewRequestWithContext` realistically only fails on a malformed `upstreamURL`, which is built from `baseURL` (operator/peer-controlled, stored via `RemoteCapStore.Put`) — if `Put` does not validate `baseURL` as a well-formed `https://` URL, a malformed deposit surfaces here as a 500 rather than a clean 4xx at deposit time.

**Fix:** Validate `baseURL` scheme/host in `RemoteCapStore.Put` (reject non-`https` or unparseable URLs at deposit time) so the proxy build path cannot fail on stored data. Update the doc block to list the 500 build-error case, or map it to 502 for consistency.

### WR-04: `homeDir` warning depends on a freshly-issued capability response, so the TUI/GUI banners can disagree at the same instant

**File:** `frontend/src/components/DaemonManagerPanel.tsx:337` and `internal/tui/files.go:377-387`
**Issue:** The GUI home-dir banner gate is `sessionWrites[s.id] && share?.homeDir && !homeDirDismissed[s.id]`, where `share.homeDir` comes from the last `IssueCapabilities` response. The TUI gate is `m.sessions[i].HomeDir && m.sessions[i].FilesWrite`, sourced from `ListSessions` (`SessionInfo.HomeDir` / `FilesWrite`). These are two different data sources for the same release-blocking cross-surface-parity signal. The GUI's `share.homeDir` is only refreshed when capabilities are re-issued (toggle, reconcile), so if the session's cwd-is-home status were to change, or if `IssueCapabilities` fails (the catch at line 109-111 only `console.warn`s and leaves stale `share`), the GUI banner can show/hide out of step with the TUI's `ListSessions`-derived banner. Per the colorblind-safe + cross-surface-parity memory notes, this is a parity gap.

**Fix:** Source the GUI banner from the same `SessionInfo.homeDir` field the TUI uses (already present on `sessions[]` via `SessionInfo`), rather than from the per-issue `share.homeDir`. That collapses both surfaces to the single server-side source of truth (`ListSessions`), which the engine comment at engine.go:461-467 explicitly calls "single server-side source of truth for ... cross-surface parity."

## Info

### IN-01: `IssueCapabilities` failure after write-toggle silently leaves stale share URLs

**File:** `frontend/src/components/DaemonManagerPanel.tsx:109-111`
**Issue:** When `handleToggleFilesWrite` succeeds in `SetSessionFilesWrite` but the follow-up `IssueCapabilities` throws, the catch only `console.warn`s. The displayed full-access link still reflects the pre-toggle token (which may lack/carry `files.write` incorrectly relative to the new toggle state) until the next reconcile. No user-visible indication that the link is stale.
**Fix:** On the re-issue failure, clear `sessionShares[sessionId]` so the reconcile effect refetches, or surface a "links may be stale — retry" message via the existing `writeError` channel.

### IN-02: `redactCapTokenFromError` ReplaceAll is substring-based; a short/empty-ish token could under-redact adjacent text

**File:** `internal/daemon/remote_files.go:267-276`
**Issue:** `strings.ReplaceAll(msg, capToken, "<redacted>")` is correct for the normal 2-segment HMAC token, but if `capToken` were ever empty the early return handles it; if it were a very short value it could match unintended substrings. Tokens are long base64url in practice, so risk is low, but the redaction is value-dependent rather than structural.
**Fix:** Keep as-is for now (tokens are long), but consider redacting by stripping the known `?cap=...` query segment from the URL in the error rather than blind substring replace. Low priority.

### IN-03: `proxyRemoteFiles` copies `r.URL.Query()` map values by reference

**File:** `internal/daemon/remote_files.go:192-198`
**Issue:** `q[k] = v` aliases the `[]string` slice from `r.URL.Query()` into the new `url.Values`. Harmless today because `q.Encode()` only reads and the request is not reused, but it is a shared-backing-slice footgun if a future edit mutates `q[k]` in place.
**Fix:** Copy the slice: `q[k] = append([]string(nil), v...)`. Cosmetic/defensive.

### IN-04: Migration re-write fires for any `schemaVersion < 4`, including already-v3 files with no behavioral change

**File:** `internal/daemon/engine.go:199-206`
**Issue:** `needsUpgradeWrite := s.SchemaVersion < CurrentSchemaVersion` triggers a settings.json re-write on first load for every pre-v4 file. This is correct and intended (idempotent on second load), and the default for `filesWrite` is false so no silent enable occurs. Noting only that a v3→v4 upgrade re-writes the file purely to bump the version number plus persist `filesWrite:false` (which omitempty drops anyway), so the on-disk diff is just `schemaVersion: 4`. Confirmed safe; no action required beyond awareness.
**Fix:** None required — behavior is correct and test-covered (`TestSettingsMigration_FilesWriteSchemaVersionRewrite`).

---

_Reviewed: 2026-06-14_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
