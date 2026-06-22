# Superseded — Broadcast ("Mechanism B") approach for FIX-03 / #98

**Superseded:** 2026-06-22 by explicit user design decision.

## What this was

The first execution of Phase 146 implemented **Mechanism B** from `146-RESEARCH.md`:
the owner's webserver minted fresh per-session RO **and** RW join codes and embedded
them in the unauthenticated `/api/sessions/meta` discovery payload, so the viewer's
"Open in browser" button could silently exchange a code and open a cap-bearing URL
with zero user interaction (CONTEXT D-02: "must NOT have to paste/enter a join code").

All 4 plans (146-00..03) executed and committed; unit tests were green.

## Why it was abandoned

Two blockers were found by the code-review and verifier gates and independently
confirmed (see `../146-REVIEW.md`, `../146-VERIFICATION.md`):

1. **Dead on arrival** — the broadcast codes never crossed the Go→Wails boundary
   (`app.go` `RemoteSession` struct / `GetRemoteSessionsWithMeta` were never wired),
   so `session.roJoinCode` was always `undefined` and #98 stayed broken. Unit tests
   passed only because none crossed that boundary.
2. **RW broadcast** — both RO and RW codes were placed on an unauthenticated payload
   any tailnet peer could poll, and `/join/exchange` does no caller-identity check.
   The "owner-only RW" rule was client-side (UI hint), not a security boundary.

## The decision

The user rejected automatic broadcast of any codes. Access is now an **explicit,
out-of-band** act: the owner generates a code/link and hands it to a specific person
(or uses it on their own other machine), reusing the existing Phase 122
`RemoteJoinCodeModal` paste flow. Same-machine owner re-attach stays one-click
(local loopback, already works). See the rewritten `../146-CONTEXT.md` decisions.

The git commits from this approach (76d8e41b, 53fe8d71, eb38b5ec, 98a2102f,
f76d7b0c, 8124aa70, e11a9fbf, 46c79854, 960a1f62, 961c88f6, 7e268f24, cf4ce0dd)
remain in history; the broadcast they added is to be **removed** by the re-planned
phase, not left in place. Files here are retained for reference only.
