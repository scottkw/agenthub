---
phase: 153-session-pty-bridge
fixed_at: 2026-06-26T00:00:00Z
review_path: .planning/phases/153-session-pty-bridge/153-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 153: Code Review Fix Report

**Fixed at:** 2026-06-26
**Source review:** .planning/phases/153-session-pty-bridge/153-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (WR-01, WR-02)
- Fixed: 2
- Skipped: 0

> CR-01 and WR-03 were already resolved in commit `fcdfd03a` (per REVIEW.md
> frontmatter `resolution` block) and were intentionally not re-touched. The four
> Info findings (IN-01..IN-04) are out of scope for this run (no `--all` flag).

## Fixed Issues

### WR-01: Raw, un-sanitized inject text is persisted and broadcast to all chat clients

**Files modified:** `internal/relay/sanitize.go`, `internal/relay/hub.go`
**Commit:** 0efcb772
**Applied fix:** Added a `SanitizeChatContent` helper (the content-surface
analogue of `SanitizePTYText`) that strips bidi-override characters
(CVE-2021-42574), C0 controls (including ESC, so CSI/OSC introducers cannot be
reconstructed by a renderer), DEL, and C1 controls — while otherwise preserving
the text the user typed (no newline collapse, no appended terminator).
`HandleInject` now stores `SanitizeChatContent(text)` in `ChatMessage.Content`
instead of the raw pre-sanitize string, so the dangerous bytes never reach
`chat.jsonl`, `BroadcastChat`, or `Export()`. Updated the `HandleInject` doc
comment to explicitly document that stored content is now "display-safe text",
not "raw keystrokes" (reversing the prior A1/A3 raw-store note, as the review
required). PTY fidelity is unchanged — the PTY still receives the full
`SanitizePTYText` output.

### WR-02: Persist/broadcast failure is silently swallowed after the PTY write already happened

**Files modified:** `internal/relay/hub.go`
**Commit:** 41005dfd
**Applied fix:** Added an exported `ErrInjectNotRecorded` sentinel and changed
`HandleInject` to return `fmt.Errorf("%w: %v", ErrInjectNotRecorded, err)` when
the chat append fails, instead of discarding the error. Both read pumps
(`relay/server.go:373`, `webserver/server.go:1176`) already convert a non-nil
`HandleInject` return into a `MakeInjectErrorFrame` NAK to the originating
subscriber, so the failure is now surfaced rather than silently swallowed.
Deliberate design choice (documented inline): the PTY write is NOT rolled back —
the inject's primary job (reach the live terminal) succeeded; only the chat
mirror failed, and the client is informed of the divergence via the NAK. This
follows the CLAUDE.md "let it crash / no silent fallback" principle.

**Note — requires human verification:** WR-02 encodes a deliberate semantic
decision (NAK-after-successful-PTY-write rather than write-after-persist
gating). Syntax checks and the existing relay test suite pass, but a developer
should confirm this divergence-signalling behavior is the intended contract
before the phase proceeds to verification — particularly whether a client
should treat an inject NAK as "rejected" vs "delivered-but-not-recorded".

## Verification

- `gofmt -l` clean on all modified files.
- `go build ./internal/relay/` succeeds.
- `go test ./internal/relay/` passes (includes `sanitize_test.go` and
  `server_inject_test.go`).

---

_Fixed: 2026-06-26_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
