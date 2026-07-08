---
phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard
verified: 2026-07-07T23:56:00Z
status: passed
score: 20/20 automated must-haves verified
behavior_unverified: 1
overrides_applied: 0
authored_by: orchestrator (gsd-verifier subagent terminated on a stream-idle infra stall mid-run; verification completed manually with recorded evidence — same recovery pattern as Phase 170)
human_verification:

  - test: "M-47 live off-tailnet public-write UAT — the RCE-severity acceptance gate (FNL-09)"
    expected: |
      On a signed production build on a real Tailscale-Funnel-granted machine:

      1. Enable internet (Funnel) sharing for a session, then in the Share modal's Danger
         section complete the ≥3s hold-to-confirm. Confirm the public write URL + single-use
         write code + live countdown appear, and the FULL ACCESS badge/tab icon light up.

      2. From an OFF-TAILNET device, redeem the write code once and confirm terminal INPUT
         reaches the PTY (you can type and run a command). Confirm the SAME write code cannot
         be redeemed a second time (single-use fails closed).

      3. Confirm there is NO other public write path: the raw public URL without the gate /
         without a registered grant is rejected (403/401), i.e. the hard consent gate is the
         ONLY way in.

      4. With a second off-tailnet device holding a READ spectator link open, click
         "Disable public write". Confirm the writer's next input is rejected (401) IMMEDIATELY
         while the read spectator keeps viewing uninterrupted (independent read/write teardown).

      5. Confirm the shorter expiry actually fires: a 15m write grant auto-revokes the writer
         at expiry without touching the read share.
    why_human: |
      This is command execution reachable over the public internet — the consequence of a
      leaked link is RCE, not observation. Proving the closed write perimeter END-TO-END
      requires a real Funnel-granted tailnet plus at least one (ideally two) genuinely
      off-tailnet devices, which CI cannot provide. The automated suite proves the perimeter
      at the unit/integration level (deny-before-gate rejection at the real WS relay upgrade,
      single-use redeem-once, expiry clamp, independent teardown); only a live off-tailnet
      device closes the RCE-severity acceptance gate. TESTING.md M-47 and 171-SECURITY.md both
      designate this as the true acceptance test. Not yet executed live in this session.
---

# Phase 171: Public Full-Access (Read-Write) Sharing — Verification Report

**Phase Goal:** Opt-in public read-write Funnel sharing behind a hard consent gate + single-use
write code + shorter expiry; public write reachable ONLY through the gate; supersedes the accidental
write-cap rebasing; covered by a threat model (FNL-09).

**Verified:** 2026-07-07T23:56:00Z
**Status:** human_needed — all automated must-haves pass; the RCE-severity live off-tailnet UAT (M-47) is the remaining acceptance gate.

> **Authoring note.** The spawned `gsd-verifier` subagent terminated on a stream-idle infra
> stall (no progress for 600s) mid-run, after correctly flagging that the load-bearing WS-gate
> test appeared to fail. The orchestrator investigated that signal to root cause (see
> "Load-bearing test finding" below), fixed it, and completed verification manually with the
> evidence recorded here — the same recovery pattern used at Phase 170 (whose first verifier
> also died before writing a report).

## Goal Achievement

### Observable Truths — automated (all PASS)

**171-01 — enforcement primitives**

- ✅ `IssueSingleUseWithTTL(token, ttl)` redeems exactly once; a second exchange fails closed
  (atomic `delete(m.codes, code)` under the single mutex hold — `internal/capability/joincode.go:117,138`).

- ✅ Per-call TTL honored, not the manager's fixed 5-minute field (`joincode.go` — TTL threaded into the record).
- ✅ **A gate-minted write cap whose grant is NOT registered is rejected at the real
  `GET /sessions/{id}/ws` upgrade (403/401)** — proven through the actual relay-upgrade path by
  `TestHandleWSSRelay_WriteCap_RequiresGate` (`internal/webserver/rwgate_test.go:106`), which passes
  including in isolation for the deny assertion. This is the load-bearing security invariant.

- ✅ `RemoveGrant(sessionID, grantID)` deletes only the named grant, leaving other grants intact.
- ✅ Gate-aware `originAllowedForWrite` rejects a funnel-origin write when `isRWGated==false` and permits it after `SetRWGate(sessionID, true)` (D-02 defense-in-depth).

**171-02 — daemon RW-gate lifecycle**

- ✅ Gate handler `handleSetSessionFunnelWrite` mints `Perms: "read,write"` HARDCODED — never
  `files.write`, regardless of the session's browse setting (`internal/daemon/api.go:1853+`,
  comment "T-171-06/D-05 (Pitfall 5): Perms hardcoded 'read,write'").

- ✅ Unconditional server-side expiry clamp: `ExpiresIn <= 0` or `> funnelWriteExpiryMax` (3600s)
  becomes exactly 3600 (`api.go:1853+`, "R5/D-11 (Pitfall 6): unconditional clamp").

- ✅ **D-04 removed**: `issueCapabilitiesForSession` no longer rebases `writeURL` to the Funnel
  base — `writeURL = writeBase + …` stays on the tailnet base; the public write cap+code are minted
  ONLY by the gate handler (`api.go:1496,1513`).

- ✅ `revokeFunnelWriteLocked` tears down write grant + write code + gate flag + expiry timer without
  breaking the reusable public READ code; wired into all four triggers (owner-disable / funnel-off /
  session-exit / auto-expiry).

- ✅ `SessionInfo.FunnelWriteActive` serializes `false` (no omitempty) so the poll detects true→false.
- ✅ Full RPC surface: `DaemonClient.SetSessionFunnelWrite`, `App.SetSessionFunnelWrite`, hand-authored
  Wails bindings in `App.d.ts` + `App.js` + `models.ts`. `internal/daemon/funnel_test.go` green.

**171-03 — desktop Share-modal Danger section + colorblind-safe indicators**

- ✅ Releasing the hold before 3s issues nothing (zero RPC calls); completing ≥3s fires exactly one
  `SetSessionFunnelWrite` and reveals URL + single-use code + countdown + disable button.

- ✅ **Colorblind-safe**: FULL ACCESS indicator differs from INTERNET by label ("FULL ACCESS" vs
  "INTERNET"), icon (`LockOpenIcon` vs `GlobeAltIcon`), AND shape (`.hub-fullaccess-badge` clip-path
  notch vs pill `border-radius`) — verified at source, never color alone (`SessionCard.tsx`,
  `TabBar.tsx`, `style.css`). Matches the colorblind-owner requirement.

- ✅ Expiry selector offers only 15m/30m/1h, default 15m (`SessionSharePanel.tsx:282,632-634`;
  `useState(900)` — no "Until I disable" option).

- ✅ Hold control is a real `<button>` with Space/Enter keyboard parity, disabled until
  `funnelActive && !warmingUp`.

- ✅ FULL ACCESS badge + tab icon render on `funnelWriteActive`, coexist with the read INTERNET
  indicators, and clear independently when write flips false. `tsc` clean, `pnpm build` succeeds,
  full vitest suite green (2353 tests).

**171-04 — TESTING.md / Sharing Guide / SECURITY.md**

- ✅ TESTING.md Suite Manifest counts reconciled (corrected a real accounting bug: `rwgate_test.go`
  is a genuinely new Go file; Go 375→376, Total 528→529); FNL-09 traceability rows for every new/
  extended Phase-171 test file; `bash tests/check-traceability-paths.sh` exits 0.

- ✅ Section-5 manual item **M-47** captures the live off-tailnet public-write UAT.
- ✅ `171-SECURITY.md` asserts the load-bearing invariant ("no public write path exists except through
  the new hard consent gate"), separates the primary WS-upgrade grant-gating barrier (D-01) from the
  `originAllowedForWrite` defense-in-depth barrier (D-02), and records the loopback endpoint's inherited
  trust boundary.

- ✅ Sharing Guide gains a "Public write access (FULL ACCESS)" section stating the command-execution
  risk plainly, the hold-gate flow, and single-use/short-lived semantics.

### Cross-plan build & regression gate

- ✅ `go build ./...` clean.
- ✅ `go test -short ./...` (full repo) — all pass, no cross-phase regressions.
- ✅ `go test -race ./internal/capability/... ./internal/webserver/... ./internal/daemon/...` green.
- ✅ Full `internal/webserver` package green across 5/5 repeated runs (with `-race`).
- ✅ Frontend `tsc --noEmit` clean; `pnpm build` succeeds; 2353 vitest tests green.

## Load-bearing test finding (root-caused and fixed)

The `gsd-verifier` flagged `TestHandleWSSRelay_WriteCap_RequiresGate` as failing. Investigation:

- The test's **security assertion** (unregistered grant → 403 at the real WS upgrade) passes
  always, including in isolation. The invariant holds.

- The **happy-path assertion** at `rwgate_test.go:160` (write reaches PTY once the grant is
  registered) failed *only when the test was run in isolation*, deterministically. Under
  `-count=5` in one process, iterations 1–2 failed at exactly the 1s deadline, then 3–5 passed
  in 0.03s — a cold-process warmup: the first WS→PTY input round-trip takes ~2s cold
  (io.Pipe + TLS + relay input pump) vs <0.1s warm. The 1s `readPipeWithTimeout` deadline was
  too tight cold, so the test passed only alongside sibling tests that warmed the path. A
  pre-existing non-171 test (`TestRelay_KeystrokesStillForwarded`) shares the exact fragility,
  confirming this is latent package-harness behavior, not a 171 regression and not a product bug.

- **Fix (commit `aa74a058`):** widened the deadline to 8s with an explanatory comment. The
  assertion is that input ARRIVES, so a longer wait never weakens it. The test now passes in
  isolation (3/3 with `-race`) and in-suite (unchanged). Empirically validated: 8s deadline →
  cold single-run passes, input arriving at ~2s.

## Requirement traceability

- **FNL-09** — intentionally left `[ ]` (pending) in REQUIREMENTS.md per this project's documented
  keep-pending-UAT convention: the requirement is not marked validated until the live M-47 off-tailnet
  UAT passes via `/gsd-verify-work`. This is NOT a gap — it is the designed acceptance flow for an
  internet-RCE surface. STATE.md correctly reflects `status: verifying`, `completed_phases: 7`.

## Verdict

All 20 automated must-haves across the 4 plans are verified in the actual codebase, the closed
write perimeter is real and enforced, and no cross-phase regressions were introduced. The one
remaining item — the M-47 live off-tailnet public-write UAT — is the RCE-severity acceptance gate
that only a real Funnel tailnet + off-tailnet device can exercise. **Phase is NOT complete until
M-47 passes.** Run `/gsd-verify-work 171`.
