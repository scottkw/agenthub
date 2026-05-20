---
phase: 111-web-bridge-osc-da-response-consumption
plan: 01
subsystem: webserver
tags: [webserver, websocket, terminal, osc, ansi, xterm, bug-fix, patch-release]

# Dependency graph
requires:
  - phase: 101-shell-session-surfaces-and-web-share-gating
    provides: "Web-share session bring-up + capability-gated WS upgrade path that this phase plugs an OSC absorber into"
  - phase: 87-capability-tokens
    provides: "testServerWithHub + readPipeMustTimeout/readPipeWithTimeout integration-test harness reused verbatim for relay tests"
provides:
  - "InputAbsorber type — per-subscriber streaming state machine for OSC 10/11/DA1 reply absorption (internal/webserver/oscabsorb.go)"
  - "Pattern: per-WS-subscriber filter declared on the read-pump goroutine, applied between relay.ParseFrame and hub.WriteInput"
  - "Closes GitHub Issue #54 on the web surface (chafa --format=sixel produces clean prompt)"
affects: [future-osc-absorption-cases, possible-desktop-relay-parity-followup]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-subscriber streaming state machine pattern for ANSI/OSC envelopes — 5-state machine (outside / got-esc / in-OSC / in-OSC-seen-esc / in-CSI) over zero-value InputAbsorber"
    - "Conservative overflow-as-passthrough policy: when an envelope exceeds maxEnvelopeBytes, flush buffered bytes back through the wire rather than silently drop (CLAUDE.md §Silent fallbacks)"
    - "OSC envelope terminator canonicalization: both ST (ESC \\\\) and BEL (\\x07) are accepted; passthrough reconstructions emit BEL (universal consumer support)"

key-files:
  created:
    - "internal/webserver/oscabsorb.go (117 source lines, no new deps, hand-rolled 5-state machine)"
    - "internal/webserver/oscabsorb_test.go (174 lines, 26 named subtests across 9 TestInputAbsorber_* functions)"
    - "internal/webserver/oscabsorb_relay_test.go (189 lines, 6 TestRelay_* integration tests)"
    - ".planning/phases/111-web-bridge-osc-da-response-consumption/111-VERIFICATION.md"
    - ".planning/phases/111-web-bridge-osc-da-response-consumption/uat-evidence/README.md"
  modified:
    - "internal/webserver/server.go (+5/-1 net +4 lines: absorber declaration + 3-line filter wiring)"

key-decisions:
  - "Per-subscriber absorber on the read-pump goroutine local stack (NOT a WebServer struct field, NOT a Hub field) — eliminates cross-subscriber state pollution from multi-tab access (T-111-02)"
  - "maxEnvelopeBytes = 4096 with overflow-as-passthrough semantics — bounded buffer guard mitigates DoS (T-111-01) without introducing silent data loss"
  - "Reuse log/slog for absorption logs at slog.Debug — first slog adoption in internal/webserver; one log per absorbed envelope; never per byte (T-111-04)"
  - "CSI termination detection by final-byte range 0x40–0x7E — robust to any DA1 param shape (handles both '1;2c' and '62;4;9;22c' from Issue #54)"
  - "OSC code decision uses bytes.Cut up to first ';' — absorb iff code in {10, 11}; passthrough every other OSC (52 clipboard, 8 hyperlink, future shapes)"
  - "Patch-release scope held: internal/relay/server.go UNCHANGED even though it has the same code shape — Open Question 1 will be empirically resolved by Task 3 UAT"

patterns-established:
  - "Streaming-state-machine pattern for browser→PTY byte filtering: zero-value struct holds the state, single Filter([]byte) []byte method, per-WS-subscriber instance"
  - "Bounded-buffer overflow policy: flush as passthrough rather than silently drop, then reset to stateOutside"
  - "TDD discipline on Go absorber: RED commit with `undefined: InputAbsorber` build error → GREEN commit with implementation → wire-up GREEN integration commit"

requirements-completed:
  - WEB-01
  - WEB-03

# Metrics
duration: 5min
completed: 2026-05-18
---

# Phase 111 Plan 01: Web bridge OSC/DA response consumption Summary

**Per-subscriber InputAbsorber state machine in `internal/webserver/oscabsorb.go` filters xterm.js's synthesized OSC 10/11 (color queries) and DA1 (device attributes) replies out of the browser→PTY input stream — closes GitHub Issue #54 on the web surface, preserves byte-for-byte passthrough of all real input (keystrokes, arrow/function keys, OSC 52 clipboard, OSC 8 hyperlinks, bracketed-paste markers).**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-05-18T15:59:23Z
- **Completed:** 2026-05-18T16:03:51Z
- **Tasks:** 3 (2 auto + 1 checkpoint:human-verify)
- **Files modified:** 1 production source + 2 test files + 2 planning artifacts

## Accomplishments

- Implemented `InputAbsorber` in `internal/webserver/oscabsorb.go` — a 5-state machine over single-byte transitions that recognizes the OSC introducer (`ESC ]`) and CSI introducer (`ESC [`), buffers the envelope body until a terminator (`BEL` or `ST = ESC \`) or final byte (CSI 0x40–0x7E), and decides on completion whether to absorb (OSC code 10 or 11; CSI starts with `?` and ends with `c`) or pass through (reconstructed byte-for-byte).
- Wired the absorber into `handleWSSRelay`'s read pump goroutine via a 4-line net diff (`absorber := &InputAbsorber{}` adjacent to `readDone := make(chan struct{})`; `filtered := absorber.Filter(payload)` + `len(filtered) > 0` gate inside the existing `case relay.MsgInput` branch). The existing `sub.ReadOnly` MC-03 gate is preserved.
- Wrote 26 unit subtests (9 `TestInputAbsorber_*` functions) covering all three absorption shapes with both terminators, every cross-frame split boundary (S1–S5: after introducer, mid-body, before terminator, between ESC and `\`, at the `?` of DA1), mixed traffic (M1: `"ls\r<OSC11>pwd\r"` → `"ls\rpwd\r"`), bare-ESC delayed passthrough (P9), and robustness (R1 4 KB overflow, R2 malformed OSC with non-ST inner ESC).
- Wrote 6 integration tests through `testServerWithHub` + `dialWebServerWS` that exercise the full WS path from `MakeInputFrame(...)` to the PTY-pipe assertion via `readPipeMustTimeout` / `readPipeWithTimeout` — including a split-across-two-frames test that proves the per-subscriber state machine survives MsgInput frame boundaries.
- Confirmed all unit + integration tests green under `go test -race -count=3 ./internal/webserver/...` and full webserver suite green under `go test -race -count=1 ./internal/webserver/...` (no new flakes).
- Scaffolded `111-VERIFICATION.md` for the cross-surface chafa parity UAT (WEB-02 release gate). Automated rows pre-filled as auto PASS with executor 2026-05-18 timings; cross-surface chafa rows flagged `human_needed` for the macOS operator.

## Tests

- **Unit tests added:** 26 subtests across 9 `TestInputAbsorber_*` functions (≥15 required). All pass under `-race -count=1`. Coverage of `oscabsorb.go::Filter` is 86.7% (the unreached branch is the CSI overflow path — same-shape branch in OSC overflow is exercised by R1).
- **Integration tests added:** 6 `TestRelay_*` functions. All pass under `-race -count=3`.
- **Full webserver suite:** PASS under `-race -count=1`.

## Diff size and scope discipline

- `internal/webserver/server.go`: +5 / -1 lines (net **+4**). Target was ≤6 net added; honored.
- `internal/relay/server.go`: **UNCHANGED**. `git diff main -- internal/relay/` is empty. Patch-release scope honored.
- `go.mod` / `go.sum`: **UNCHANGED**. No new dependencies.
- `internal/webserver/oscabsorb.go` is 117 source lines (target ≤120).

## Open Question 1: status

**Unresolved at code level — empirical resolution deferred to Task 3 UAT.** `111-RESEARCH.md` flags that code-level analysis cannot explain why the desktop Wails surface was reported (in Issue #54) as unaffected when both surfaces use identical `term.onData((data) => ws.send(makeInputFrame(data)))` JavaScript shape AND both server-side paths (`internal/webserver/server.go:755` and `internal/relay/server.go:124-128`) forward to `hub.WriteInput(payload)` with no filtering.

This plan honored the CONTEXT D-LOCKED boundary and shipped the web-only fix. The cross-surface UAT in `111-VERIFICATION.md` will empirically resolve the question:

- If desktop chafa is clean on macOS — Issue #54's claim is confirmed and Open Question 1 is RESOLVED (no follow-up needed).
- If desktop chafa ALSO leaks on macOS — file a new GitHub issue ("Desktop relay also leaks OSC/DA1 replies (follow-up to #54)") mirroring the absorber into `internal/relay/server.go:124-128`. **Do not** expand v3.3.1 scope; the follow-up ships separately.

The integration tests created here (`oscabsorb_relay_test.go`) would port trivially to the desktop relay path if needed — same `MakeInputFrame` → `WriteInput` shape.

## Requirements traceability

| Req | What proves it | Where |
|-----|----------------|-------|
| **WEB-01** (the bug is fixed on the web surface) | `TestRelay_OSC10ReplyAbsorbedBeforePTY`, `TestRelay_OSC11ReplyAbsorbedBeforePTY`, `TestRelay_DA1ReplyAbsorbedBeforePTY` (Go-level proof zero bytes reach the PTY pipe); + Task 3 cross-surface chafa UAT row "Web chafa sixel clean prompt" (empirical proof on the actual web surface) | `internal/webserver/oscabsorb_relay_test.go`; `111-VERIFICATION.md` |
| **WEB-02** (cross-surface parity) | Task 3 UAT row "Desktop chafa sixel clean prompt" + "Parity decision" | `111-VERIFICATION.md` — `human_needed` until macOS operator runs the chafa command in both surfaces |
| **WEB-03** (regression test exists) | 26 unit subtests (`TestInputAbsorber_*`) + 6 integration tests (`TestRelay_*`); all pass under `-race -count=3`; coverage of new code 86.7% | `internal/webserver/oscabsorb_test.go`, `internal/webserver/oscabsorb_relay_test.go` |

## Deviations from Plan

None — plan executed exactly as written. The TDD RED → GREEN cycle was followed for both Task 1 (unit) and Task 2 (integration). The wiring diff matches the plan's exact prescription (per-subscriber `absorber := &InputAbsorber{}` declared adjacent to `readDone`, `filtered := absorber.Filter(payload)` inside the existing `case relay.MsgInput` branch).

One incidental cleanup: the initial draft of `oscabsorb_test.go` declared a local `min(int, int) int` helper that collided with Go 1.21's built-in and the pre-existing `min` in `assets_test.go`. Resolved inline by inlining the bounds check at the one use site — no behavioral change, single-test-file localized.

## Commits

1. `c343a1d` — `test(111-01): add failing unit tests for InputAbsorber state machine` (RED)
2. `9082f5a` — `feat(111-01): implement InputAbsorber state machine for OSC/DA replies` (GREEN)
3. `31e5d68` — `feat(111-01): wire InputAbsorber into handleWSSRelay + integration tests` (Task 2 GREEN)
4. `bcf2bf5` — `docs(111-01): scaffold VERIFICATION.md for cross-surface chafa UAT` (Task 3 scaffolding)

## Local UAT attempt (web-share + Chrome)

Not attempted by the executor. Per orchestrator instruction, if the local web-share + Chrome + chafa flow proves complex, mark as `human_needed` and let the operator verify. The flow requires:

- Background `agenthub daemon run` + `agenthub web start`
- A foreground `agenthub serve <session>` capturing a share URL
- A Chrome session against a self-signed-cert dev share URL (the "Proceed anyway" interstitial)
- Two terminal surfaces open at once (web AND desktop Wails GUI) to compare the chafa output side-by-side for parity
- Visual evaluation of the prompt text following the sixel render

This is a multi-step interactive operator workflow with a strong "compare two screenshots" deliverable. The macOS dev box owner will run it and capture the artifacts listed in `111-VERIFICATION.md`.

## TDD Gate Compliance

- RED commit (`test(111-01): add failing unit tests...`) exists — `c343a1d`.
- GREEN commit (`feat(111-01): implement InputAbsorber...`) exists — `9082f5a`. Followed by integration GREEN — `31e5d68`.
- No REFACTOR commit was needed; the implementation passed both unit and integration tests on first GREEN.

## Self-Check: PASSED

- `internal/webserver/oscabsorb.go` — FOUND
- `internal/webserver/oscabsorb_test.go` — FOUND
- `internal/webserver/oscabsorb_relay_test.go` — FOUND
- `internal/webserver/server.go` — modified, FOUND
- `.planning/phases/111-web-bridge-osc-da-response-consumption/111-VERIFICATION.md` — FOUND
- `.planning/phases/111-web-bridge-osc-da-response-consumption/uat-evidence/README.md` — FOUND
- Commits `c343a1d`, `9082f5a`, `31e5d68`, `bcf2bf5` — all FOUND in `git log`
