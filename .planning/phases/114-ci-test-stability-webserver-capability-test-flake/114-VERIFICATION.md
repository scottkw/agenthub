---
phase: 114-ci-test-stability-webserver-capability-test-flake
plan: 01
requirements:
  - TEST-01
  - TEST-02
human_needed: true
human_needed_reason: "macOS executor cannot directly drive a Linux CI runner; the gold-standard 100/100 Linux CI gate (TEST-01 final acceptance) requires a human or CI workflow to confirm post-push."
executor_platform: darwin/arm64
go_version: go1.26.3
date: 2026-05-18
---

# 114-01 Verification Record

Variant A fix for Issue #58 — `TestPluginConfigStream_ExpiredCap_Returns401` deflake.

## §1 Local macOS evidence (TEST-01 partial)

**STRONG evidence; not final acceptance.** Variant A removes all known sources of non-determinism (no base64 byte-flip, no wall-clock-second dependency, no shuffle dependency). Linux CI 100/100 is the gold-standard gate per 114-CONTEXT.md.

Commands run from `/Users/ken/dev/agenthub` on darwin/arm64 (Go 1.26.3):

1. **Targeted stress run, 100 iterations:**
   ```
   go test -race -shuffle=on -count=100 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/
   ```
   Result: `ok  github.com/scottkw/agenthub/internal/webserver  2.023s` → **100/100 PASS**.

2. **Full-package smoke, 10 iterations, shuffled:**
   ```
   go test -race -shuffle=on -count=10 ./internal/webserver/
   ```
   Result: `ok  github.com/scottkw/agenthub/internal/webserver  31.980s` → **PASS, no sibling regression**.

3. **Sanity check (targeted, count=1):**
   ```
   go test -race -count=1 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/
   ```
   Result: PASS.

## §2 Mathematical determinism argument

Restated from 114-RESEARCH.md §7. Variant A correctness reduces to **HMAC-SHA256 second-preimage resistance**:

- The helper signs claims with an all-0xFF 32-byte key (`wrongKey`).
- The server holds `capTestKey` (a different 32-byte key).
- `capability.Verify` computes HMAC-SHA256(payload, capTestKey) and compares it (via `hmac.Equal`, constant-time) to the token's signature segment.
- For the test to spuriously pass (i.e., for the wrong-key signature to also verify under the real key), an adversary-equivalent collision in HMAC-SHA256 would be required — collision probability ≈ 2^-256 ≈ 0.
- **No wall-clock dependency**: the IAT is `time.Now().Unix()`, but capability.Verify ignores IAT (no expiry path), and the signature path is the verification target.
- **No test-ordering dependency**: each test uses a fresh `*WebServer` via `testServer + t.Cleanup`; the wrong-key sign does not depend on any global or shared state.
- **No race dependency**: `-race` is on in all runs; passing under `-race` over 100 iterations confirms no data race in the helper.

## §3 Linux CI gold-standard step (TEST-01 final acceptance — HUMAN-GATED)

**Marker:** `human_needed: true`.
**Reason:** A macOS arm64 executor cannot directly drive a Linux CI runner; confirming Linux CI 100/100 requires a human (or merged CI workflow) to observe the post-push run.

**Command (run on Linux runner):**
```
go test -race -shuffle=on -count=100 ./internal/webserver/
```

**Expected result:** 100/100 PASS for `TestPluginConfigStream_ExpiredCap_Returns401` and no regressions in sibling tests in `./internal/webserver/`.

**Procedure:**
1. Push the Task 3 fix commit on `main` to origin (timing is a release-workflow decision — push directly, batch with other commits, or open a PR; the gate is the result, not the mechanism).
2. Observe the GitHub Actions CI run (or trigger an ad-hoc Linux run via `gh workflow run`, or check out the commit on a Linux box / Docker container).
3. Confirm: 100/100 PASS, no sibling regression.

**Interpretation:**
- **100/100 PASS** → TEST-01 final acceptance met. Mark TEST-01 Complete in REQUIREMENTS.md.
- **Anything < 100/100** → DO NOT approve. Variant A is provably deterministic by HMAC-SHA256 second-preimage resistance, so a Linux CI failure here would mean a SECOND root cause exists that 114-RESEARCH.md did not capture (per Assumption A1 below). File a follow-up phase; do NOT revert Variant A.

## §4 TEST-02 — root cause stated in writing

The base64-padding-bit root cause is stated in writing in two authoritative locations:

1. **114-RESEARCH.md §5** — full math derivation:
   - HMAC-SHA256 outputs 32 bytes → encoded with `base64.RawURLEncoding` to 43 chars.
   - The final base64 char carries 4 data bits + 2 padding bits; the 2 padding bits are discarded by the decoder.
   - Alphabet chars A, B, C, D (indices 0, 1, 2, 3) share the same top 4 data bits (`0000`); their differences live in the discarded low 2 padding bits.
   - When the signature's tail char landed in {A, B, C, D} (4/64 = 6.25% of cases), the historical helper's `A↔B` flip was a no-op: the decoder produced the same final signature byte, `capability.Verify` accepted the token, the middleware advanced past the HMAC check, found no grant (helper deliberately omitted `AddGrant`), and returned 403 "capability has been revoked" instead of the expected 401 "capability required".
   - The IAT was `time.Now().Add(-365d).Unix()` and advances per UTC second; a single `go test -count=N` process samples one IAT, so the run either all-passes or all-fails. Linux CI runs hit a single-second sample; macOS author runs spread over many seconds, masking the bug locally.

2. **Fix commit message (Task 3)** — restates the same root cause in the commit log so future reviewers and maintainers learn the lesson without having to fetch the research doc.

The fix (Variant A) replaces the source of non-determinism with a wrong-key sign that exercises the production `ErrInvalidSignature → 401` path used elsewhere by the non-flaky `TestCapability_InvalidSignatureReturns401` in `internal/webserver/capability_test.go`.

## §5 What the fix did NOT do (TEST-02 acceptance)

The fix is bounded to **root-cause repair**, not workaround:

- **No `t.Skip`** introduced.
- **No retry loop** added (no "try up to N times until pass" pattern).
- **No `-shuffle=off`** invoked anywhere (in the test, in the CI config, in any tooling).
- **No test rename**: `TestPluginConfigStream_ExpiredCap_Returns401` body and name are unchanged.
- **No fixture reset / global state mutation** added.
- **Diff bounded** to one helper function (`issueExpiredCapFor`) + its comment block in `internal/webserver/plugin_config_stream_test.go`, plus this VERIFICATION.md file.

This is confirmed by reading `git show HEAD` after Task 3 commit lands.

## §6 Assumption A1 carry-forward (from 114-RESEARCH.md)

**Assumption A1:** Linux CI's ~6% failure rate is fully explained by the wall-clock-second base64-padding-bit no-op model. Equivalently: there is only ONE root cause for this flake.

**If Linux CI continues to flake after this fix lands**, that is a signal of a SECOND cause that Phase 114 did not capture. The correct response is to file a follow-up phase with the new evidence — do NOT retroactively claim Variant A was wrong (it is provably deterministic by HMAC-SHA256 second-preimage resistance). Variant A is necessary in any case; the second cause would be additive.

## Requirement traceability

| Requirement | Acceptance | Status |
|-------------|------------|--------|
| TEST-01 (deterministic 401 path) | Local macOS 100/100 + Linux CI 100/100 | Local PASS (this doc §1); Linux CI gate PENDING (§3, human-gated) |
| TEST-02 (root cause stated in writing) | Stated in 114-RESEARCH.md §5 AND in fix commit message | RESEARCH.md complete (pre-existing); commit message will be produced by Task 3 |
