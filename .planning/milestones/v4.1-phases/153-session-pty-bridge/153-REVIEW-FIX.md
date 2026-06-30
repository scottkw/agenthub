---
phase: 153-session-pty-bridge
fixed_at: 2026-06-26T00:00:00Z
review_path: .planning/phases/153-session-pty-bridge/153-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 153: Code Review Fix Report

**Fixed at:** 2026-06-26
**Source review:** .planning/phases/153-session-pty-bridge/153-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (this run): 4 (IN-01, IN-02, IN-03, IN-04)
- Fixed: 4
- Skipped: 0

The four Info findings were the only OPEN items remaining for this `--fix` run.
The Critical (CR-01) and all three Warnings (WR-01, WR-02, WR-03) were already
resolved in prior commits before this run started; they are recorded under
"Out-of-this-run context" below for traceability and were intentionally NOT
touched.

## Fixed Issues

### IN-01: Inject NAK leaks raw internal error strings to the remote client

**Files modified:** `internal/relay/hub.go`, `internal/relay/server.go`, `internal/webserver/server.go`
**Commit:** 2a64d943
**Applied fix:** Added the exported `InjectErrorReason(err error) string` helper
in `internal/relay/hub.go` that maps each `HandleInject` error to a stable,
client-safe reason: `ErrReadOnly` → "inject rejected: read-only access",
`ErrInjectTooLarge` → "inject rejected: text too large", `ErrInjectNotRecorded`
→ "inject delivered to terminal but not recorded in chat", and any other
(write) error → generic "inject failed". Both read pumps now NAK with
`MakeInjectErrorFrame(InjectErrorReason(err))` instead of `err.Error()`, and log
the detailed error server-side (`log.Printf` on the relay path, `slog.Warn` on
the web path). Internal plumbing detail such as `io: read/write on closed pipe`
can no longer reach a remote viewer.

### IN-02: Control-only inject text reduces to a bare newline and still presses Enter at the PTY

**Files modified:** `internal/relay/hub.go`
**Commit:** 978de6b6
**Applied fix:** In `HandleInject`, after `SanitizePTYText`, added a
`strings.TrimSpace(sanitized) == ""` guard that returns `nil` (no-op, no NAK)
before the PTY write. Control-only input (e.g. `"\x1b[2J"`, `"\x00"`) passes the
read-pump `ip.Text != ""` guard but collapses to a bare `"\n"`; this is now
treated as empty, skipping both the spurious Enter keystroke at the PTY and the
chat persist/broadcast. Applied in the shared `HandleInject` so both relay and
web read pumps are covered. Added the `strings` import.

### IN-03: No explicit size cap on inject text before the PTY write

**Files modified:** `internal/relay/hub.go`
**Commit:** dbb84afd
**Applied fix:** Added the `MaxInjectTextBytes = 64 * 1024` constant and the
`ErrInjectTooLarge` sentinel. `HandleInject` now rejects raw inject text larger
than the cap (`len(text) > MaxInjectTextBytes`) before any PTY write, returning
`ErrInjectTooLarge` which the read pump turns into a NAK (and which IN-01's
`InjectErrorReason` maps to "inject rejected: text too large"). The bound is now
intentional and independent of the `coder/websocket` default read limit. 64 KiB
is generous for a pasted command line yet well under the chat-layer
`maxChatLineBytes` (1 MiB).

### IN-04: Sanitizer doc overstates coverage; DCS/APC/PM/SOS string payloads pass through as plaintext

**Files modified:** `internal/relay/sanitize.go`
**Commit:** 981c0e1f
**Applied fix:** Chose the "fix code" option (not just the comment). Extended the
`SanitizePTYText` state machine with `stateString` and `stateStringEsc` states
and added `'P'` (DCS), `'_'` (APC), `'^'` (PM), `'X'` (SOS) cases to the escape
state. These string-introducer bodies are now consumed up to the ST terminator
(`ESC \`) — terminated only by ST, not BEL — so the body never leaks as
plaintext. Updated the doc comment to accurately describe the new coverage.

## Out-of-this-run context (already resolved before this run)

These findings were resolved in prior commits and were not modified by this run:

- **CR-01** (SEC-01 bypass: read-only web viewer inject when browse enabled) —
  RESOLVED in commit `fcdfd03a`. The web-share inject gate now uses
  `!capability.HasPerm(claims.Perms, "write")`.
- **WR-01** (raw un-sanitized inject text persisted/broadcast) — RESOLVED in
  commit `0efcb772`. `SanitizeChatContent` added in
  `internal/relay/sanitize.go`; `HandleInject` persists the sanitized content.
- **WR-02** (persist/broadcast failure silently swallowed after PTY write) —
  RESOLVED in commit `41005dfd`. `ErrInjectNotRecorded` added; `HandleInject`
  returns the wrapped append error so the read pump emits a NAK.
- **WR-03** (SEC-01 web-path test never exercised the broken gate) — RESOLVED in
  commit `fcdfd03a`. `TestInjectRO_WebPath` is now table-driven over `"read"`
  and `"read,files.read"`.

## Verification

Each fix was verified before commit:
- `go build ./internal/relay/ ./internal/webserver/` — passed after every fix
- `gofmt -l` on every modified file — clean (no formatting drift)
- `go test ./internal/relay/ ./internal/webserver/` — passed after every fix

---

_Fixed: 2026-06-26_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
