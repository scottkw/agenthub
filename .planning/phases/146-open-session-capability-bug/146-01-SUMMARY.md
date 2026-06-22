---
phase: 146-open-session-capability-bug
plan: "01"
subsystem: daemon/webserver/tailnet
tags: [capability, join-codes, sessions-meta, fix]
dependency_graph:
  requires: ["146-00"]
  provides: ["mintSessionJoinCodes", "SetJoinCodeIssuer", "ro_join_code in /api/sessions/meta"]
  affects: ["internal/webserver/server.go", "internal/daemon/api.go", "internal/tailnet/sessions.go"]
tech_stack:
  added: []
  patterns: ["set-once callback", "degraded-mode nil-guard", "grant-before-issue ordering"]
key_files:
  created: []
  modified:
    - internal/daemon/api.go
    - internal/webserver/server.go
    - internal/tailnet/sessions.go
    - TESTING.md
decisions:
  - "sessionMetaItem uses no omitempty on ro_join_code/rw_join_code (always serialized) so TestSessionsMeta_NoCapInResponse can assert field presence in the allowed key set"
  - "ShareableSessionMeta uses omitempty (viewer-side struct; only non-empty codes are meaningful to the viewer)"
  - "mintSessionJoinCodes wired at both AutoStartWebServer and handleWebServerStart sites for consistency"
metrics:
  duration: "~20 minutes"
  completed: "2026-06-22"
  tasks_completed: 2
  files_modified: 4
---

# Phase 146 Plan 01: Wave 1 — Owner-Side Go Join-Code Embed Summary

**One-liner:** Extracts `mintSessionJoinCodes` from `issueCapabilitiesForSession`, wires it as a `joinCodeIssuer` callback on `WebServer`, and enriches `GET /api/sessions/meta` to embed fresh per-session `ro_join_code`/`rw_join_code` — making all four Wave 0 RED tests GREEN.

## What Was Built

### Task 1 — mintSessionJoinCodes helper + daemon wiring (`internal/daemon/api.go`)

Added `func (a *API) mintSessionJoinCodes(sessionID string) (roCode, rwCode string, err error)`:
- Extracts the token-mint + grant-register + code-issue body of `issueCapabilitiesForSession`, dropping the URL-construction step (viewer builds the URL)
- Permissions mirror `issueCapabilitiesForSession` exactly: RO = `"read"` (or `"read,files.read"` when browse on), RW = `"read,write"` (or `"read,write,files.read,files.write"`)
- Registers both grants on `WebServer` via `ws.AddGrant` BEFORE calling `joinCodes.Issue` (Pitfall 3 / T-146-04)
- Returns `(roCode, rwCode, nil)`; error on nil signing key, nil web server, or nil join-code manager

Wired `ws.SetJoinCodeIssuer(a.mintSessionJoinCodes)` at both daemon startup sites (`AutoStartWebServer` and `handleWebServerStart`), immediately adjacent to the existing `SetJoinCodes` call.

### Task 2 — WebServer enrichment + ShareableSessionMeta (`internal/webserver/server.go`, `internal/tailnet/sessions.go`)

`server.go`:
- Added `joinCodeIssuer func(sessionID string) (roCode, rwCode string, err error)` field to `WebServer` struct (set-once pattern, mirrors `sessionResolver`)
- Added `SetJoinCodeIssuer(fn func(string) (string, string, error))` setter
- Added `ROJoinCode string \`json:"ro_join_code"\`` and `RWJoinCode string \`json:"rw_join_code"\`` to `sessionMetaItem` — no `omitempty` (see Deviations)
- Updated `handleSessionsMeta` to call `ws.joinCodeIssuer(id)` per session when non-nil; on error leaves codes empty and continues (degraded mode — no 500, no panic per PATTERNS.md)

`sessions.go`:
- Added `ROJoinCode string \`json:"ro_join_code,omitempty"\`` and `RWJoinCode string \`json:"rw_join_code,omitempty"\`` to `ShareableSessionMeta` (stdlib json.Decode maps automatically)

`TESTING.md` (Rule 2 auto-fix — CLAUDE.md standing rule):
- Updated Go test count from 346 to 348 (Wave 0 added 2 new files)
- Added FIX-03 traceability rows for `sessions_meta_embed_test.go` and `mint_join_codes_test.go`

## Test Results

All four target tests GREEN:
- `TestMintSessionJoinCodes` — distinct codes, grants registered, RO perms=`"read"`, RW perms=`"read,write"`
- `TestSessionsMeta_EmbedJoinCodes` — `ro_join_code`/`rw_join_code` populated from wired issuer
- `TestSessionsMeta_NilIssuer` — 200 with empty/absent codes when no issuer wired
- `TestSessionsMeta_NoCapInResponse` — RB-03 preserved, `ro_join_code`/`rw_join_code` in allowed set

Full suite: `go test -race -short ./internal/webserver/... ./internal/daemon/... ./internal/tailnet/...` → all GREEN.

## Commits

| Task | Commit | Files | Description |
|------|--------|-------|-------------|
| Task 1 | 98a2102f | internal/daemon/api.go | mintSessionJoinCodes + SetJoinCodeIssuer wiring |
| Task 2 | f76d7b0c | internal/webserver/server.go, internal/tailnet/sessions.go, TESTING.md | sessionMetaItem enrichment + handleSessionsMeta + ShareableSessionMeta |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] sessionMetaItem fields use no omitempty**

- **Found during:** Task 2, while analyzing Wave 0 test `TestSessionsMeta_NoCapInResponse`
- **Issue:** The plan spec says `json:"ro_join_code,omitempty"` but the Wave 0 test (already committed) includes `ro_join_code` and `rw_join_code` in the `allowed` map AND iterates that map asserting "all required keys are present". With `omitempty`, empty strings are dropped from JSON, causing the assertion to fail when no issuer is wired in that test.
- **Fix:** Removed `omitempty` from `sessionMetaItem.ROJoinCode` and `sessionMetaItem.RWJoinCode`. Empty string serializes as `""` in JSON, which satisfies both `TestSessionsMeta_NilIssuer` (`ok && v != ""` condition) and `TestSessionsMeta_NoCapInResponse` (key present in map).
- **Note:** `ShareableSessionMeta` (viewer-side) retains `omitempty` as specified — viewer only uses non-empty codes, and no test requires the key to be present when empty.
- **Files modified:** internal/webserver/server.go
- **Commit:** f76d7b0c

**2. [Rule 2 - Missing Critical Functionality] TESTING.md not updated for Wave 0 test files**

- **Found during:** Post-task review per CLAUDE.md standing rule
- **Issue:** Wave 0 added `sessions_meta_embed_test.go` and `mint_join_codes_test.go` but did not update TESTING.md (count, traceability rows)
- **Fix:** Updated Go count 346→348; added FIX-03 traceability rows for both test files
- **Files modified:** TESTING.md
- **Commit:** f76d7b0c

## Threat Model Verification

All T-146 mitigations implemented:
- **T-146-01** (RB-03 / info disclosure): `ro_join_code`/`rw_join_code` are short-lived join codes, not cap tokens; `TestSessionsMeta_NoCapInResponse` blacklist unchanged; GREEN
- **T-146-02** (replay): codes minted fresh per meta request via `joinCodes.Issue` (single-use, 5-min TTL); never cached; `mintSessionJoinCodes` called per-request
- **T-146-03** (unshared-session leak): `webEnabledSessions()` remains the sole gate; unshared sessions never appear in meta
- **T-146-04** (elevation): both grants registered before code issuance; RO=`"read"`, RW=`"read,write"` exactly as `issueCapabilitiesForSession`

## Known Stubs

None — all fields wire to real data from the `mintSessionJoinCodes` helper.

## Self-Check: PASSED

- `internal/daemon/api.go` exists with `func (a *API) mintSessionJoinCodes(`
- `internal/webserver/server.go` exists with `SetJoinCodeIssuer` and `ro_join_code`
- `internal/tailnet/sessions.go` exists with `ro_join_code` in `ShareableSessionMeta`
- Commits 98a2102f and f76d7b0c exist in git log
- All tests GREEN (go test -race -short ./internal/webserver/... ./internal/daemon/... ./internal/tailnet/...)
