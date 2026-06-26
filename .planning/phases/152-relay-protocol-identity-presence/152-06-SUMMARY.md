---
phase: 152-relay-protocol-identity-presence
plan: "06"
subsystem: webserver-identity-presence
tags: [identity, presence, web-share, WhoIs, alias, typing, parity, testing]
dependency_graph:
  requires: ["152-05"]
  provides: [IDENT-01-web, IDENT-02-web, PRESENCE-01-web, PRESENCE-02-web]
  affects: [internal/webserver/server.go, internal/daemon/api.go, TESTING.md]
tech_stack:
  added: []
  patterns:
    - "var lc local.Client zero-value WhoIs after websocket.Accept"
    - "non-'local' sentinel TailnetID on WhoIs failure (D-04 / criterion 5)"
    - "SetAliasProviders closure pattern mirrors setChatProviders (T-151-09)"
    - "Unsubscribe(bool) + conditional NotifyPresence on leave"
    - "MsgAliasSet + MsgTyping outside ReadOnly gate (D-06)"
key_files:
  created:
    - internal/webserver/identity_test.go
  modified:
    - internal/webserver/server.go
    - internal/daemon/api.go
    - TESTING.md
decisions:
  - "SetAliasProviders setter added alongside SetChatHistoryProvider/SetChatExportProvider — same closure-over-import-cycle pattern (T-151-09)"
  - "setChatProviders extended to also call SetAliasProviders — one call site, both wiring sites (AutoStartWebServer + handleWebServerStart) updated atomically"
  - "WhoIs called after websocket.Accept (Pitfall: WhoIs on wrong pre-upgrade address)"
  - "WhoIs failure → tailnetID='unknown' (non-'local') so web client never collides with owner's 'local:local'"
metrics:
  duration: "~8 minutes"
  completed: "2026-06-26"
  tasks_completed: 2
  files_changed: 4
status: complete
---

# Phase 152 Plan 06: Web-Share Identity + Presence Parity Summary

Completed the web-share WS path (handleWSSRelay) to reach identity/presence/typing parity with the relay path. Added lc.WhoIs identity resolution, default-alias derivation, Origin='web' stamping, alias-provider wiring, and MsgAliasSet/MsgTyping dispatch. Updated TESTING.md per the standing regression-suite convention.

## Tasks Completed

### Task 1: WhoIs identity stamping + alias/typing dispatch in handleWSSRelay

**Commit:** 76bc4e88

Added to `internal/webserver/server.go`:
- `aliasGet` / `aliasSet` fields on `WebServer` struct
- `SetAliasProviders(getAlias, setAlias func)` setter — mirrors `SetFilesHandler` pattern; avoids webserver→daemon import cycle (T-151-09)
- `strings` import added for `strings.SplitN` (LoginName prefix derivation)
- In `handleWSSRelay` after `websocket.Accept`: `var lc local.Client; lc.WhoIs(ctx, r.RemoteAddr)` — populates `tailnetID` from `who.Node.Key.String()` and `defaultAlias` from `ComputedName` or LoginName prefix; failure leaves `tailnetID="unknown"` (non-"local" sentinel, D-04)
- `personKey = tailnetID + ":web"`; alias resolved via `ws.aliasGet(personKey, defaultAlias)`
- `sub.TailnetID`, `sub.Origin="web"`, `sub.PersonKey`, `sub.Alias`, `sub.AliasSetFn` set before `hub.Subscribe`
- `relay.NotifyPresence(hub)` after subscribe; `hub.Unsubscribe(sub)` captures bool; conditional `relay.NotifyPresence(hub)` in defer
- `case relay.MsgTyping` + `case relay.MsgAliasSet` added to read pump outside the `ReadOnly` gate (D-06)

Added to `internal/daemon/api.go`:
- `setChatProviders` extended to also call `ws.SetAliasProviders(...)` — two closure callbacks over `a.engine.Aliases()`, both guarded for nil `aliasStore`; error from `Set` intentionally discarded (alias persistence failure must not close connection); one change site keeps both `AutoStartWebServer` and `handleWebServerStart` in sync

**Verification:** `go build ./...` and `go vet ./internal/webserver/... ./internal/daemon/...` — clean.

### Task 2: Web-path identity/parity tests + TESTING.md regression registration

**Commit:** d58b1db1

Created `internal/webserver/identity_test.go` (package webserver internal — uses existing helpers):

| Test | Invariant | Result |
|------|-----------|--------|
| `TestWebIdentity_WhoIsFailureFallback` | WhoIs failure → `":web"` suffix, NOT `"local:local"`, Origin="web", TailnetID≠"local" (criterion 5 / D-04) | PASS |
| `TestWebAliasPropagation` | `MsgAliasSet` → next `MsgPresence` roster shows new alias; alias persisted to in-memory provider | PASS |
| `TestWebReadOnlyCanChat` | RO-cap client (`perms="read"`) can set alias via `MsgAliasSet`; next `MsgPresence` shows alias (D-06) | PASS |

Helper infrastructure:
- `inMemAliasMap` — thread-safe in-memory alias map for tests (no disk dependency)
- `drainUntilPresence` — reads `websocket.Conn` until `MsgPresence` frame arrives, skipping `MsgMeta`/scrollback
- `dialIdentityWS` — composes `testServerWithHub` + `SetAliasProviders` + `issueCapFor` + `dialWebServerWS` with correct `Origin` header

TESTING.md changes per standing convention:
- **Section 2 manifest:** Go count updated 355→359; total 483→487; added notes for Phase 152-01/03/05/06
- **Section 4 traceability:** Added rows for IDENT-01 (relay + web), IDENT-02 (web alias propagation), PRESENCE-01 (hub + relay server), PRESENCE-02 (hub typing TTL)
- **Section 5 manual checklist:** Added M-18 (live two-client presence distinctness over live tailnet) and M-19 (typing ≤500ms / clears-after-5s timing in real browser)
- All traceability paths verified on disk; `bash tests/check-traceability-paths.sh` exits 0 (Linux CI)

## Success Criteria Verification

| Criterion | Status |
|-----------|--------|
| Web clients stamped with WhoIs TailnetID + web origin (D-01, IDENT-01) | PASS — `tailnetID + ":web"` personKey; Origin="web" in subscriber |
| Owner vs same-machine browser are distinct composite keys (criterion 5) | PASS — WhoIs failure → "unknown:web" ≠ "local:local"; proven by `TestWebIdentity_WhoIsFailureFallback` |
| Alias-set + typing dispatch at parity with relay (IDENT-02, PRESENCE-01/02) | PASS — `TestWebAliasPropagation` proves MsgAliasSet; `TestWebReadOnlyCanChat` proves D-06 parity |
| RO clients full chat participants (D-06) | PASS — `TestWebReadOnlyCanChat` confirms alias-set works with `perms="read"` |
| TESTING.md updated per standing regression convention | PASS — counts, traceability rows, manual items all updated |
| Traceability check green | PASS — all 5 new Phase 152 test file paths exist on disk |
| `go build ./...` clean | PASS |
| `go test ./...` green (full module) | PASS — all 15 packages pass including new webserver/identity tests |

## Deviations from Plan

None — plan executed exactly as written.

The `strings` import was already needed for `strings.SplitN` (LoginName prefix) and was not present in the import block; added as part of Task 1 (Rule 3 — blocking import, not a separate deviation).

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes introduced beyond what the plan's threat model covers. T-152-01 (alias tampering) is mitigated by `relay.ValidateAlias` in the `case relay.MsgAliasSet` pump case, identical to the relay path.

## Self-Check: PASSED

- `internal/webserver/identity_test.go` — FOUND on disk
- `internal/webserver/server.go` — FOUND and modified (76bc4e88)
- `internal/daemon/api.go` — FOUND and modified (76bc4e88)
- `TESTING.md` — FOUND and modified (d58b1db1)
- `152-06-SUMMARY.md` — FOUND on disk
- Commits `76bc4e88` and `d58b1db1` — verified in git log
- `go build ./...` — clean
- `go test ./...` — all 15 packages pass
