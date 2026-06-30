---
phase: 153-session-pty-bridge
plan: "01"
subsystem: relay-protocol-sanitization
tags: [protocol, security, sanitization, tdd]
dependency_graph:
  requires: [152-06]
  provides: [MsgSessionInject, MsgInjectError, SanitizePTYText, MakeChatFrame, MakeInjectErrorFrame]
  affects: [internal/relay/protocol.go, internal/relay/sanitize.go]
tech_stack:
  added: []
  patterns: [rune-scan-state-machine, make-frame-builder, tdd-red-green]
key_files:
  created:
    - internal/relay/sanitize.go
    - internal/relay/sanitize_test.go
  modified:
    - internal/relay/protocol.go
decisions:
  - "0x35/0x36 verified free before allocation — grep returned empty on pre-change protocol.go"
  - "SanitizePTYText uses dedup-space guard for CRLF to prevent double-space output (Pitfall 1)"
  - "isBidiOverride uses explicit switch over 11 codepoints — no regex, matches ValidateAlias style"
metrics:
  duration: "8 minutes"
  completed: "2026-06-26"
  tasks_completed: 2
  files_changed: 3
status: complete
---

# Phase 153 Plan 01: Protocol Constants + SanitizePTYText Summary

Inject protocol constants (D-02) and the SEC-02 sanitizer (D-03) for the `@session` PTY bridge. Foundation for all downstream inject wiring.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add inject protocol constants, payloads, and frame builders | 2cc3b94f | internal/relay/protocol.go |
| 2 (RED) | Add failing SEC-02 corpus test | 0c4363de | internal/relay/sanitize_test.go |
| 2 (GREEN) | Implement SanitizePTYText state machine | 201d3ef6 | internal/relay/sanitize.go, internal/relay/sanitize_test.go |

## What Was Built

**Task 1 — Protocol constants and frame builders** (`internal/relay/protocol.go`):

- `MsgSessionInject = 0x35` (client → server: inject text into session PTY, RW only)
- `MsgInjectError = 0x36` (server → client: inject rejected)
- `InjectPayload{Text string}` — JSON body for MsgSessionInject frames
- `InjectErrorPayload{Reason string}` — JSON body for MsgInjectError frames
- `MakeChatFrame(msg ChatMessage) []byte` — encodes ChatMessage as MsgChat frame
- `MakeInjectErrorFrame(reason string) []byte` — encodes rejection reason as MsgInjectError frame

**Task 2 — SanitizePTYText state machine** (`internal/relay/sanitize.go`):

Five-state sanitizer (`stateNormal` / `stateEscape` / `stateCSI` / `stateOSC` / `stateOSCEscape`) that strips:
- C0 control characters (0x00–0x1F, excluding ESC which drives state transitions)
- DEL (0x7F)
- C1 controls (U+0080–U+009F)
- CSI sequences (ESC `[` ... final-byte in 0x40–0x7E)
- OSC sequences (ESC `]` ... BEL or ESC `\`)
- Unicode bidi-override codepoints via `isBidiOverride` (11 codepoints: U+061C, U+200E, U+200F, U+202A–U+202E, U+2066–U+2069)
- Collapses LF/CR/CRLF to a single space using a dedup guard
- TrimRight trailing spaces, then appends exactly one `\n`

**Output invariant (SEC-02):** only printable text + exactly one trailing `\n` ever exits `SanitizePTYText`.

**Test corpus** (`internal/relay/sanitize_test.go`): 19 table-driven cases covering every SEC-02 category — plain passthrough, trailing spaces, LF/CR/CRLF collapse, null byte, C0 BEL, DEL, C1 NEL + U+0080, CSI clear-screen + color, OSC BEL-terminated + ST-terminated, bidi RLO + LRM, empty input, only-newlines, mixed attack vector.

## Verification

```
go build ./internal/relay/...                                  PASS
go test -race -short -run TestSanitizePTYText ./internal/relay/...  PASS (19 cases)
go vet ./internal/relay/...                                    PASS
gofmt -l sanitize.go protocol.go                               PASS (no output)
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Mixed attack vector expected value incorrect in plan**
- **Found during:** Task 2 GREEN phase
- **Issue:** The plan's behavior block specified `"cmd\x1b[A\x00\r\n;evil" → "cmd evil\n"`, dropping the semicolon ';' (0x3B). The ';' character is printable ASCII — it is not a C0/C1 control, DEL, CSI sequence, OSC sequence, or bidi override — so the sanitizer must pass it through unchanged. Stripping it would require treating shell metacharacters as unsafe, which is outside the stated D-03 scope and would break legitimate text input.
- **Fix:** Updated the `mixed attack vector` test case `want` value from `"cmd evil\n"` to `"cmd ;evil\n"`. The implementation is correct; the plan's expected value had a typo.
- **Files modified:** `internal/relay/sanitize_test.go`
- **Commit:** 201d3ef6

## TDD Gate Compliance

- RED gate commit: `0c4363de` — `test(153-01): add failing SEC-02 corpus test for SanitizePTYText` (build failed with "undefined: SanitizePTYText")
- GREEN gate commit: `201d3ef6` — `feat(153-01): implement SanitizePTYText state machine with SEC-02 corpus test`
- REFACTOR: not required (implementation clean on first pass)

## Known Stubs

None. This plan contains no UI and no placeholder data — it is a pure backend protocol + utility slice.

## Threat Flags

No new threat surface beyond what is already in the plan's threat model. `SanitizePTYText` mitigates T-153-01 through T-153-05 as specified. No new network endpoints, auth paths, or trust boundary crossings introduced.

## Self-Check: PASSED
