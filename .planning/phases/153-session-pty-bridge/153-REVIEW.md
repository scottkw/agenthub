---
phase: 153-session-pty-bridge
reviewed: 2026-06-26T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - internal/relay/protocol.go
  - internal/relay/sanitize.go
  - internal/relay/sanitize_test.go
  - internal/relay/hub.go
  - internal/relay/server.go
  - internal/relay/server_inject_test.go
  - internal/daemon/engine.go
  - internal/webserver/server.go
  - internal/webserver/inject_test.go
findings:
  critical: 1
  warning: 3
  info: 4
  total: 8
status: issues_found
---

# Phase 153: Code Review Report

**Reviewed:** 2026-06-26
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Phase 153 adds the one-way `@session`→PTY inject bridge: a dedicated `MsgSessionInject`
frame (0x35), a server-side capability gate in `Hub.HandleInject`, a PTY-text sanitizer
(`SanitizePTYText`), and the read-pump wiring on both the relay (loopback) and web-share
WebSocket paths.

The sanitizer is well-constructed and complete against the bidi/CSI/OSC threat model, the
relay and web read-pump inject handlers are byte-for-byte equivalent, `HandleInject`
correctly releases `hub.mu` before all I/O (no mutex-across-IO, no PTY-write race on the
single Write syscall), and `MsgChatSend`/stray frames correctly never reach the PTY
(MENTION-03 holds).

However, the security gate this phase depends on (SEC-01) is **broken on the web-share
path whenever per-session file browsing is enabled**. The phase relies on a pre-existing
exact-string-match (`claims.Perms == "read"`) that does not hold for the
browse-enabled read-only token (`"read,files.read"`), silently promoting a read-only
viewer to write/inject. The phase's own SEC-01 web-path test mints a bare `"read"` token
and therefore never exercises the broken case — the green test masks the defect.

## Critical Issues

### CR-01: Read-only web viewer can inject into the PTY when file-browsing is enabled (SEC-01 bypass)

**File:** `internal/webserver/server.go:1008`
**Issue:**
The web-share inject gate derives read-only status with an exact-string match:

```go
readonly := claims.Perms == "read"
```

But the capability minter (`internal/daemon/api.go:1263-1267`) emits a richer perms
string for the **read-only** token whenever per-session browse is enabled:

```go
rPerms := "read"
wPerms := "read,write"
if a.engine.browseEnabledFor(sessionID) {
    rPerms = "read," + capability.PermFilesRead              // "read,files.read"
    wPerms = "read,write," + capability.PermFilesRead + "," + capability.PermFilesWrite
}
```

When browse is ON, the RO token carries `Perms == "read,files.read"`. The gate then
evaluates `"read,files.read" == "read"` → **false** → `sub.ReadOnly = false`. The
read-only viewer is treated as a write-capable client and `Hub.HandleInject` performs the
PTY write instead of returning `ErrReadOnly`. This directly violates SEC-01 ("read-only
clients must be NAK'd with zero PTY writes"): a read-only web viewer can inject arbitrary
commands into the session's terminal stdin (and, separately, the same flaw already lets
them send raw `MsgInput` keystrokes via the `!sub.ReadOnly` check at line 1117).

The codebase already ships the correct whole-token helper, written specifically to avoid
this substring/exact-match class of bug (T-124-01/02), and the relay path's equivalent
write authorization is the only thing keeping the relay surface safe (it uses a boolean
query param, not perms). This is gate drift: the web gate is the weaker, broken one.

Note: the `== "read"` predates Phase 153 (it also gates `MsgInput`), but Phase 153 newly
routes PTY-*inject* authorization through this same `sub.ReadOnly`, so satisfying SEC-01
for this phase requires fixing it here.

**Fix:** Gate on the absence of the `write` whole-token capability, not exact equality:

```go
// Read-only unless the token actually carries the "write" capability bit.
readonly := !capability.HasPerm(claims.Perms, "write")
```

`HasPerm("read,files.read", "write")` → false → `readonly = true` (correct);
`HasPerm("read,write,files.read,files.write", "write")` → true → `readonly = false`
(correct). Consider also exporting a `PermWrite = "write"` constant alongside
`PermFilesRead`/`PermFilesWrite` so the literal is not duplicated.

## Warnings

### WR-01: Raw, un-sanitized inject text is persisted and broadcast to all chat clients (SEC-02 bidi mitigation bypassed on the chat surface)

**File:** `internal/relay/hub.go:482-493`
**Issue:**
`HandleInject` sanitizes text before the PTY write, but then persists and broadcasts the
**original, pre-sanitize** string verbatim:

```go
msg, err := fn(ChatMessage{
    ...
    Content: text, // original pre-sanitize text
    ...
})
if err == nil {
    h.BroadcastChat(MakeChatFrame(msg))
}
```

The sanitizer strips bidi-override (Trojan-Source / CVE-2021-42574) and control sequences
to protect the PTY, but the identical dangerous bytes are then stored in `chat.jsonl`,
broadcast to every chat client, and emitted into the Markdown `Export()` (`chat.go:310+`).
This phase is the *first* path that broadcasts user-controlled content, and there is no
content validator equivalent to `ValidateAlias` for `ChatMessage.Content`. The
CVE-2021-42574 mitigation that SEC-02 calls out is therefore defeated on the chat display
and export surfaces, where the same renderer-level spoofing applies.

**Fix:** Keep raw text for PTY fidelity if desired, but sanitize what is persisted/
displayed — strip at minimum the bidi-override set (`isBidiOverride`) and C0/C1 controls
from `Content` before `chatAppendFn`/`BroadcastChat`, or introduce a `ValidateChatContent`
analogous to `ValidateAlias` and apply it on every content-bearing path. Document
explicitly whether stored content is "raw keystrokes" or "display-safe text".

### WR-02: Persist/broadcast failure is silently swallowed after the PTY write already happened

**File:** `internal/relay/hub.go:482-494`
**Issue:**
After a successful PTY write, the chat append result is discarded on error:

```go
if fn != nil {
    msg, err := fn(ChatMessage{...})
    if err == nil {
        h.BroadcastChat(MakeChatFrame(msg))
    }
    // err != nil: no broadcast, no NAK, no log — silently dropped
}
```

`ChatStore.AppendMessage` returns `ErrChatCapReached` and `ErrChatMessageTooLarge`
(`chat.go:252,281`) under normal operation. When that happens the command has *already*
been injected into the live terminal, but no chat record is written and the user receives
no feedback or error bubble — the terminal and the chat thread diverge with no signal.
This is the silent-fallback anti-pattern called out in CLAUDE.md ("let it crash" /
"make beliefs pay rent").

**Fix:** On `fn` error, surface it — at minimum send a `MakeInjectErrorFrame` NAK to the
originating subscriber (the inject partially failed) and/or log it. Decide deliberately
whether the PTY write should be gated behind a successful append (write-after-persist) or
the divergence is acceptable, and document the choice.

### WR-03: SEC-01 web-path test never exercises the broken gate case (masks CR-01)

**File:** `internal/webserver/inject_test.go:116`
**Issue:**
`TestInjectRO_WebPath` mints its read-only capability with a bare perms string:

```go
token := issueCapFor(t, ws, sessionID, "read")  // Perms == "read" exactly
```

This produces `claims.Perms == "read"`, the *one* value for which the buggy
`claims.Perms == "read"` gate happens to behave correctly. The test therefore passes while
the real browse-enabled RO token (`"read,files.read"`) — the value the daemon actually
mints — slips through the gate as writable (CR-01). The green test is not evidence the
gate holds; it encodes the same wrong assumption as the code under test.

**Fix:** Add a case that mints/uses an RO token with `Perms == "read,files.read"` (browse
enabled) and asserts the NAK + zero-PTY-write invariant. This case fails today and will
pass once CR-01 is fixed.

## Info

### IN-01: Inject NAK leaks raw internal error strings to the remote client

**File:** `internal/relay/server.go:376`, `internal/webserver/server.go:1170`
**Issue:** `MakeInjectErrorFrame(err.Error())` forwards the raw error to the client. For
`ErrReadOnly` this is benign, but on a PTY `WriteInput` failure it surfaces internal
plumbing detail (e.g. `io: read/write on closed pipe`) to a remote web viewer.
**Fix:** Map `ErrReadOnly` to a stable user-facing reason and collapse write errors to a
generic "inject failed" string; log the detailed error server-side.

### IN-02: Control-only inject text reduces to a bare newline and still presses Enter at the PTY

**File:** `internal/relay/sanitize.go:108-109`, read-pump empty check at `server.go:370`
**Issue:** The `ip.Text == ""` guard runs before sanitization. Input consisting solely of
control/escape bytes (e.g. `"\x1b[2J"` or `"\x00"`) is non-empty, passes the guard, and
`SanitizePTYText` collapses it to `"\n"` — injecting a bare Enter keystroke into the PTY
and broadcasting the raw sequence to chat (see WR-01). Low impact (RW clients can already
send Enter), but the inject is effectively a no-op command submission.
**Fix:** Treat a post-sanitize result of just `"\n"` (or whitespace-only) as empty and
skip both the PTY write and the chat broadcast.

### IN-03: No explicit size cap on inject text before the PTY write

**File:** `internal/relay/hub.go:465-475`
**Issue:** `HandleInject` writes the full sanitized payload to the PTY before any size
check; the chat-layer cap (`maxChatLineBytes`) only applies afterward and on error is
swallowed (WR-02). The frame is currently bounded only by the `coder/websocket` default
read limit. Relying on a library default for a security-relevant bound is fragile.
**Fix:** Enforce an explicit inject-text length cap in the read pump (reject oversize
frames with a NAK) so the bound is intentional and independent of the WS library default.

### IN-04: Sanitizer doc overstates coverage; DCS/APC/PM/SOS string payloads pass through as plaintext

**File:** `internal/relay/sanitize.go:20-24`
**Issue:** The state machine handles CSI and OSC string termination, but for other
string-introducer escapes — DCS (`ESC P`), APC (`ESC _`), PM (`ESC ^`), SOS (`ESC X`) —
only the two-byte introducer is discarded; the string *body* is then emitted as ordinary
text and the `ESC \` terminator is split apart. No control sequence reaches the PTY (the
introducers are removed), so this is not an injection, but the doc comment claims "All
other ESC-prefixed pairs are silently discarded," which understates that string bodies
leak as text.
**Fix:** Either extend the state machine to consume DCS/APC/PM/SOS bodies up to ST (reuse
the OSC string-termination states), or correct the comment to state that only the
introducer is removed and the body survives as text.

---

_Reviewed: 2026-06-26_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
