---
phase: 74-multi-client-fan-out
reviewed: 2026-04-14T12:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - cmd_attach.go
  - internal/daemon/api_test.go
  - internal/daemon/engine.go
  - internal/daemon/types.go
  - internal/relay/hub.go
  - internal/relay/hub_test.go
  - internal/relay/server.go
  - internal/relay/server_test.go
  - internal/webserver/server.go
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 74: Code Review Report

**Reviewed:** 2026-04-14T12:00:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Phase 74 adds multi-client fan-out capabilities to the relay system: read-only subscribers (MC-03), viewer count reporting (MC-04), client identity tracking (MC-05), and a max-wins resize arbiter (MC-06). The new features are applied consistently across both the local relay server (`relay/server.go`) and the remote web server (`webserver/server.go`).

The implementation is solid overall. The hub's subscribe-before-snapshot anti-race pattern is correctly maintained, the max-wins resize policy is well-tested, and the new `Subscriber` fields are threaded through both code paths. Three warnings were identified: a silently discarded fetch error that produces misleading user messages, duplicated WebSocket relay logic across two server files that could drift, and the relay server's origin-check bypass lacking a tracking mechanism. Two informational items note a minor inconsistency and missing input validation.

## Warnings

### WR-01: Silently discarded fetch error produces misleading remote-attach error message

**File:** `cmd_attach.go:180`
**Issue:** `fetchErr` from `fetchPeerSessions` / `fetchPeerSessionsWithClient` is explicitly discarded with `_ = fetchErr`. When the fetch fails due to a network error, TLS failure, or timeout, the user receives `"session X not found on remote host Y"` instead of a meaningful error describing the connection failure. This violates the project's "Silent Fallbacks" principle from CLAUDE.md: silent conversion of hard failures into incorrect-but-plausible messages makes debugging expensive.
**Fix:**
```go
if fetchErr != nil {
    return fmt.Errorf("attach: failed to fetch sessions from remote host %q: %w", hostname, fetchErr)
}
```

### WR-02: Duplicated WebSocket relay logic in relay/server.go and webserver/server.go

**File:** `internal/relay/server.go:96-138` and `internal/webserver/server.go:433-477`
**Issue:** The read pump (client -> PTY) and write pump (hub -> client) logic, including MC-03 read-only enforcement, MC-05 client name parsing, and MC-06 ResizeClient calls, is copy-pasted across both handlers. Any future protocol change (new message types, security checks, error handling) must be applied in both places or they will silently drift. The Phase 74 diff itself demonstrates this: identical 14-line blocks were added to both files.
**Fix:** Extract the shared relay pump logic into a reusable function in the `relay` package, e.g.:
```go
// relay/pump.go
func RunRelayPumps(ctx context.Context, conn *websocket.Conn, hub *Hub, sub *Subscriber) {
    // read pump + write pump in one composable function
}
```
Both `server.go` and `webserver/server.go` would call this single implementation, eliminating the duplication.

### WR-03: Relay server WebSocket accepts all origins without tracking or mitigation

**File:** `internal/relay/server.go:60-61`
**Issue:** The `InsecureSkipVerify: true` option in `websocket.AcceptOptions` disables WebSocket origin checking. The comment references "Phase 4" for adding a proper CORS/origin policy, but there is no issue tracker reference, no TODO with a tracking ID, and no runtime guard. While the relay server binds to `127.0.0.1` (limiting exposure to localhost), any page open in the user's browser can initiate cross-origin WebSocket connections to `ws://127.0.0.1:{port}/sessions/{id}/ws` and interact with PTY sessions (DNS rebinding / localhost CSRF). This is a known risk on localhost-bound services.
**Fix:** Add a tracking comment with a concrete identifier and consider restricting the allowed origin to the expected client:
```go
conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
    // TODO(MC-SECURITY): restrict to expected origins. Tracked in Phase X.
    // Risk: localhost CSRF via DNS rebinding. Mitigated by localhost-only binding.
    InsecureSkipVerify: true,
})
```
Or, better, check origin against `http://127.0.0.1:{port}` and `http://localhost:{port}`.

## Info

### IN-01: clientName truncation to 64 bytes is not validated on the CLI side

**File:** `cmd_attach.go:37-38`
**Issue:** The `--client=` flag value is sent as-is in the URL query parameter but the server truncates it to 64 characters (`server.go:49`, `webserver/server.go:389`). There is no client-side validation or warning, so a user passing a long client name would have it silently truncated. Minor usability inconsistency.
**Fix:** Add client-side validation in `cmdAttach`:
```go
} else if len(arg) > 9 && arg[:9] == "--client=" {
    clientName = arg[9:]
    if len(clientName) > 64 {
        return fmt.Errorf("attach: --client name must be 64 characters or fewer")
    }
```

### IN-02: ResizeClient max-wins computes cols and rows independently across subscribers

**File:** `internal/relay/hub.go:103-112`
**Issue:** The max-wins policy computes `maxCols` and `maxRows` independently. If subscriber A reports 220x24 and subscriber B reports 80x60, the PTY is sized to 220x60 -- a combination that matches neither client's actual terminal. This is a design choice (documented via tests) rather than a bug, but it could cause layout artifacts in applications that query terminal size (e.g., TUI frameworks). An alternative policy would be to use the dimensions of the single largest-area subscriber.
**Fix:** No code change needed if the current behavior is intentional. Document the tradeoff in the function's doc comment:
```go
// Note: cols and rows are maximized independently. The resulting PTY size
// may not match any individual subscriber's terminal dimensions.
```

---

_Reviewed: 2026-04-14T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
