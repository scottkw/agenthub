# Phase 115 Plan 01 — Summary

**Status:** Complete (2026-05-19)
**Closes:** GitHub Issue #60 + Phase 111 RESEARCH Open Question 1
**Requirements satisfied:** WEB-04, WEB-05, WEB-06

## What shipped

Lifted the Phase 111 `InputAbsorber` from `internal/webserver/handleWSSRelay` into `internal/relay/handleSession`. Same proven 117-line state machine (now moved to `internal/relay/oscabsorb.go`), now wired into the daemon-direct relay path that the Wails desktop GUI and CLI `agenthub attach` use.

## Files touched

| File | Change | Why |
|------|--------|-----|
| `internal/webserver/oscabsorb.go` → `internal/relay/oscabsorb.go` | `git mv` + package rename | Absorber is a relay-protocol filter; webserver already depends on relay, so import direction is correct. |
| `internal/webserver/oscabsorb_test.go` → `internal/relay/oscabsorb_test.go` | `git mv` + package rename | 26-subtest unit suite stays with the source. |
| `internal/webserver/server.go` | `&InputAbsorber{}` → `&relay.InputAbsorber{}` (1 line) | Type qualification after the move. |
| `internal/relay/server.go` | +13 lines net in `handleSession` read pump | Per-subscriber absorber, applied to MsgInput payload before `hub.WriteInput`. |
| `internal/relay/oscabsorb_relay_test.go` | +195 LoC (new) | 6 integration tests against `handleSession` mirroring the Phase 111 webserver-layer suite. |

## Commits

- `f95e61c` test(115): RED — handleSession leaks OSC 10/11 + DA1 to PTY
- `1b34dcb` fix(115): absorb OSC 10/11 + DA1 on daemon-direct relay path (closes #60)

## Test results

- All 6 new `TestRelay_handleSession_*` tests: GREEN
- `internal/relay` full suite under `-race -count=3 -shuffle=on`: PASS (7.94 s)
- `internal/webserver` full suite under `-race -count=1`: PASS (4.22 s) — existing 26 unit + 6 integration tests on the moved absorber still green
- `go build ./...`: clean

## Manual UAT

Rebuilt Wails app (`wails build -tags wailsassets`); ran the OSC 11 + DA1 sensitive probe in a fresh Shell session. Pre-fix produced `11;rgb:1d1d/1f1f/212162;4;9;22c` typed at the next prompt (see `uat-evidence/desktop-osc-probe-pre-fix.png`). Post-fix produces `ZZZ_MARKER` followed by a clean `>` prompt (see `uat-evidence/desktop-osc-probe-fixed.png`).

Web ↔ desktop full parity now empirically holds. Phase 111 `approved with desktop follow-up: #60` resume signal is retired.

## Open questions

None — Phase 111 RESEARCH Open Question 1 (`does desktop also leak?`) is now answered empirically (yes, it leaked) and addressed (no longer leaks).

## Surprises

The webserver-layer absorber's `oscabsorb_relay_test.go` integration suite continued to pass without any test code changes — confirms the move was a pure refactor at the absorber level. The state machine is byte-shape-driven, not package-coupled.
