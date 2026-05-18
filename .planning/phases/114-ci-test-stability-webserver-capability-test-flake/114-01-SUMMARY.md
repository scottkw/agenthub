---
phase: 114-ci-test-stability-webserver-capability-test-flake
plan: 01
subsystem: webserver/test-fixture
tags:
  - go
  - hmac
  - base64
  - test-stability
  - webserver
  - capability
requirements:
  - TEST-01
  - TEST-02
dependency_graph:
  requires: []
  provides:
    - deterministic-401-helper-issueExpiredCapFor
  affects:
    - internal/webserver/plugin_config_stream_test.go
tech_stack:
  added: []
  patterns:
    - "Variant A: sign-with-wrong-key (32 bytes 0xFF) to exercise production ErrInvalidSignature → 401 path"
key_files:
  created:
    - .planning/phases/114-ci-test-stability-webserver-capability-test-flake/114-VERIFICATION.md
  modified:
    - internal/webserver/plugin_config_stream_test.go
decisions:
  - "Adopt Variant A (wrong-key sign) — root-cause repair, not workaround; deterministic by HMAC-SHA256 second-preimage resistance"
  - "Keep helper name issueExpiredCapFor and test name TestPluginConfigStream_ExpiredCap_Returns401 unchanged to preserve Issue #58 / CI log continuity"
  - "Linux CI 100/100 gate deferred to operator (human_needed) — macOS executor cannot drive Linux runner"
metrics:
  duration_minutes: 3
  completed_date: 2026-05-18
  tasks_completed: 3
  tasks_total: 4
  task_4_status: human-gated (deferred to operator per orchestrator instruction)
commits:
  - 904cd14: "fix(webserver): deflake TestPluginConfigStream_ExpiredCap_Returns401 (Issue #58)"
---

# Phase 114 Plan 01: CI Test Stability — Webserver Capability Test Flake Summary

Variant A fix (sign with deliberately wrong 32-byte 0xFF key) applied to `issueExpiredCapFor` in `internal/webserver/plugin_config_stream_test.go`; macOS stress run 100/100 PASS; Linux CI 100/100 gate deferred to operator.

## What Changed

- **`internal/webserver/plugin_config_stream_test.go`** — `issueExpiredCapFor` rewritten per Variant A:
  - Signs claims with `wrongKey := make([]byte, 32)` + `wrongKey[i] = 0xFF` for all i, instead of signing with `capTestKey` and then attempting to corrupt the result.
  - Removed all byte-flip / `strings.LastIndex` / `corrupt[len(corrupt)-1]` logic.
  - Comment block rewritten to reflect that the helper produces an **invalid-signature** token (not an "expired" one — `capability.Verify` has no expiry path); the historical "expired" framing in the helper / test name is preserved for Issue #58 / CI log continuity.
  - Test function `TestPluginConfigStream_ExpiredCap_Returns401` body and name **unchanged** per plan constraint.
  - `strings` import preserved (used elsewhere in the file).
- **`.planning/phases/114-ci-test-stability-webserver-capability-test-flake/114-VERIFICATION.md`** — created. Six sections: local evidence, mathematical determinism argument, Linux CI human-gated step, TEST-02 root-cause pointers, "what the fix did NOT do" guarantee, Assumption A1 carry-forward.

## Local Verification Results (macOS arm64, Go 1.26.3)

| Command | Result |
|---------|--------|
| `go test -race -count=1 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/` | PASS (sanity, 1.06s) |
| `go test -race -shuffle=on -count=100 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/` | **100/100 PASS** (2.02s) |
| `go test -race -shuffle=on -count=10 ./internal/webserver/` | PASS, **no sibling regression** (31.98s) |
| `go vet ./internal/webserver/` | clean |
| `gofmt -w internal/webserver/plugin_config_stream_test.go` | no diff after format |

## Linux CI Status (Task 4 — HUMAN-GATED)

**Status:** PENDING external check.

Task 4 (`checkpoint:human-verify`) defers Linux CI 100/100 confirmation to the operator. The macOS executor cannot directly drive a Linux runner. The scaffold lives in `114-VERIFICATION.md §3` with the exact command, expected result, and interpretation rules.

Per the orchestrator instruction ("Task 4 is human_needed — scaffold in VERIFICATION.md, defer to operator"), this plan is recorded as **approved-local-only**:
- TEST-01 is held PENDING in REQUIREMENTS.md (Linux CI gate not yet observed).
- TEST-02 is COMPLETE (root cause stated in writing in 114-RESEARCH.md §5 AND in commit 904cd14's body).

When the operator confirms Linux CI 100/100 PASS post-push, TEST-01 can be marked Complete.

## Issue #58 Closure Status

- Commit 904cd14 includes `Closes #58` in the message body.
- Pre-closure gate: Linux CI 100/100 confirmation (Task 4, operator-driven). Until then, #58 remains technically open even though the fix has landed locally on `main` — GitHub will auto-close on push if a PR/push triggers the keyword.

## Fix Commit

`904cd14 fix(webserver): deflake TestPluginConfigStream_ExpiredCap_Returns401 (Issue #58)`

The commit message body states the base64-padding-bit root cause in writing (TEST-02 acceptance):

> The 32-byte HMAC encodes (via base64.RawURLEncoding) to 43 chars, where the FINAL char carries only 4 data bits + 2 padding bits. Base64 alphabet chars A, B, C, D (indices 0-3) all share the same top 4 data bits and decode to the same final signature byte; the low 2 bits are padding that the decoder discards. When the signature's tail char landed in {A, B, C, D} (4/64 = 6.25% of cases), the A<->B flip was a no-op…

Diff bounded to exactly 2 files (helper rewrite + VERIFICATION.md). No `t.Skip`, no retry loop, no `-shuffle=off`, no test rename.

## Deviations from Plan

None — plan executed exactly as written. Task 4 (Linux CI gate) was correctly identified upstream as `human_needed`; the orchestrator instruction explicitly directed deferral to operator with a VERIFICATION.md scaffold, which has been done.

## Pointers

- Full verification record: `.planning/phases/114-ci-test-stability-webserver-capability-test-flake/114-VERIFICATION.md`
- Root cause math: `.planning/phases/114-ci-test-stability-webserver-capability-test-flake/114-RESEARCH.md §5`
- Reference non-flaky pattern: `internal/webserver/capability_test.go::TestCapability_InvalidSignatureReturns401` (~lines 205–230)
- Production verification path: `internal/webserver/capability_mw.go::requireCapability` (step 3, capability.Verify → 401)

## Self-Check: PASSED

Verified post-write:
- `internal/webserver/plugin_config_stream_test.go` — modified, contains `wrongKey`, no `corrupt[len(corrupt)-1]` in non-comment lines.
- `.planning/phases/114-ci-test-stability-webserver-capability-test-flake/114-VERIFICATION.md` — created, contains Linux CI / human_needed / TEST-01 / TEST-02 / base64 / padding / 6.25% keywords.
- Commit 904cd14 — exists on `main`; body contains base64 / padding / 6.25 / A,B,C,D / wrongKey / TEST-01 / TEST-02 / Closes #58; diff bounded to 2 files.
- Local stress: 100/100 PASS targeted; PASS package-wide x10 shuffled.
