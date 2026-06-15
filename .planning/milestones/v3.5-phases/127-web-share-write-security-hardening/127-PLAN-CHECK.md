# 127-PLAN-CHECK.md

**Phase:** 127 — Web-Share Write Security Hardening
**Plans checked:** 127-01, 127-02, 127-03, 127-04
**Date:** 2026-06-15
**Checker:** gsd-plan-checker (Revision Gate)

---

## PLAN CHECK PASSED

No blockers. One warning. All 5 success criteria and all 7 SEC requirement IDs are covered.

---

## Coverage Summary

| Requirement | Covering Plans | Status |
|-------------|---------------|--------|
| SEC-01 (write-path symlink escape → 403) | 127-02 Task 1 | COVERED |
| SEC-02 (denylist: macOS config dir + case variation) | 127-01 Tasks 1+2 | COVERED |
| SEC-03 (upload abuse: ../filename, over-cap, no zip-slip) | 127-02 Task 2 (confirm existing) | COVERED |
| SEC-04 (capability-escalation audit + committed SECURITY artifact) | 127-03 Task 1 | COVERED |
| SEC-05 (concurrent-write race + interrupted-write no partial file) | 127-03 Task 2 | COVERED |
| SEC-06 (FuzzSandboxWrite corpus finalized, 0 crashes) | 127-01 Task 2 + 127-02 Task 2 | COVERED |
| SEC-07 (Playwright e2e: write-OK / 403-no-cap / Origin-mismatch 403) | 127-04 Task 1 | COVERED |

---

## Plan Summary

| Plan | Wave | Tasks | Files Modified | Depends On | Status |
|------|------|-------|---------------|------------|--------|
| 127-01 | 1 | 2 | sandbox.go, write_test.go, sandbox_test.go | [] | VALID |
| 127-02 | 2 | 2 | sandbox_test.go | ["127-01"] | VALID |
| 127-03 | 2 | 2 | write_test.go, 127-SECURITY.md | ["127-01"] | VALID |
| 127-04 | 1 | 1 | files-write.spec.ts | [] | VALID |

---

## Verification Dimensions

### Dimension 1: Requirement Coverage — PASS

All 7 SEC requirement IDs from REQUIREMENTS.md appear in at least one plan's `requirements` frontmatter field:

- SEC-01 → 127-02
- SEC-02 → 127-01
- SEC-03 → 127-02
- SEC-04 → 127-03
- SEC-05 → 127-03
- SEC-06 → 127-01 (seeds), 127-02 (merge gate confirm)
- SEC-07 → 127-04

### Dimension 2: Task Completeness — PASS

All tasks have `<read_first>`, `<action>`, `<verify>`, `<done>`, and `<acceptance_criteria>`. Actions are specific (cite exact file paths and line numbers). Verify blocks are runnable commands. Done states are measurable.

| Plan | Task | Has read_first | Has action | Has verify | Has done | Has acceptance_criteria |
|------|------|---------------|-----------|-----------|---------|------------------------|
| 127-01 | 1 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 127-01 | 2 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 127-02 | 1 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 127-02 | 2 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 127-03 | 1 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 127-03 | 2 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 127-04 | 1 | ✅ | ✅ | ✅ | ✅ | ✅ |

### Dimension 3: Dependency Correctness — PASS

- 127-01: depends_on [] → Wave 1 ✅
- 127-02: depends_on ["127-01"] → Wave 2 ✅ (correct: adds seeds from 01 before running fuzz gate)
- 127-03: depends_on ["127-01"] → Wave 2 ✅ (parallel to 02; touches disjoint files)
- 127-04: depends_on [] → Wave 1 ✅ (disjoint from 01; only touches files-write.spec.ts)

No cycles. No missing references. Wave assignments consistent with dependency graph. 127-02 and 127-03 correctly run in parallel in Wave 2, touching disjoint files (sandbox_test.go vs write_test.go + SECURITY.md). 127-04 is genuinely independent of the Go test plans.

### Dimension 4: Key Links Planned — PASS

All plans wire their artifacts:

- 127-01: denylistCheck → os.UserConfigDir (via local derivation); WriteFileAtomic/Rename/Mkdir/Delete → denylistCheck (single chokepoint; no write-method-body edits required)
- 127-02: TestSandbox_WritePathSymlinkEscapeBlocked → os.OpenRoot terminal boundary in WriteFileAtomic/Rename/Mkdir (test exercises existing production boundary)
- 127-03: 127-SECURITY.md per-surface matrix → requireFilesWrite / HasPerm / originAllowedForWrite (source anchors cited); TestWrite_TwoWritersIfMatchRace → WriteFileAtomic validator re-check at sandbox.go:256-276
- 127-04: Origin-mismatch e2e cell → originAllowedForWrite (capability_mw.go); test uses env.writeCap (valid cap) to prove CSRF rejection, not cap failure

### Dimension 5: Scope Sanity — PASS

All plans are within budget. Maximum 2 tasks per plan. Maximum 3 files per plan.

| Plan | Tasks | Files | Verdict |
|------|-------|-------|---------|
| 127-01 | 2 | 3 | OK |
| 127-02 | 2 | 1 | OK |
| 127-03 | 2 | 2 | OK |
| 127-04 | 1 | 1 | OK |

### Dimension 6: Verification Derivation — PASS

All must_haves truths are user-/operator-observable security outcomes, not implementation details:

- "Writing/renaming/deleting the daemon's own config file ... returns ErrProtectedSystemFile" ✅ observable
- "Case-varied denylist targets (.BASHRC, .Bashrc ...) return ErrProtectedSystemFile" ✅ observable
- "A file literally named .bashrc inside a NON-home-rooted sandbox stays writable" ✅ observable negative control
- "FuzzSandboxWrite reports zero crashes" ✅ observable merge-gate outcome
- "WriteFileAtomic through a symlink escaping the sandbox root returns a non-nil error and creates nothing outside root" ✅ observable
- "Two concurrent writers ... produce exactly one success and one ErrPreconditionFailed" ✅ observable
- "127-SECURITY.md exists and is committed" ✅ observable
- "A web-share PUT ... with a mismatched Origin header ... is rejected with HTTP 403" ✅ observable

Artifacts have concrete paths and `contains` anchors. Key links specify the wiring mechanism.

### Dimension 7: Context Compliance — PASS

CONTEXT.md locked decisions (SC1–SC5) vs plan coverage:

| SC | Locked Decision | Implementing Task |
|----|----------------|------------------|
| SC1 | symlink-escape on write/rename → 403 | 127-02 Task 1 (TestSandbox_WritePathSymlinkEscapeBlocked) |
| SC2 | denylist: case/Unicode/encoding bypass closed + daemon config dir | 127-01 Tasks 1+2 (case-fold + os.UserConfigDir) |
| SC3 | FuzzSandboxWrite 0 crashes; over-cap before parse | 127-01 Task 2 (seeds) + 127-02 Tasks 1+2 (gate + SEC-03 confirm) |
| SC4 | capability-escalation audit + committed SECURITY artifact | 127-03 Task 1 |
| SC5 | Playwright e2e: write-OK / 403-no-cap / CSRF Origin-mismatch 403 | 127-04 Task 1 |

No locked decision is contradicted. No tasks implement anything from Deferred Ideas (there are none). Discretion areas (SECURITY artifact structure, auditor-agent choice, NFC scope) are handled within planner's latitude.

**PATTERNS.md resolved assumption A1:** go.mod:140 confirms `golang.org/x/text v0.37.0` is in-tree. Plan 01 correctly opts for ASCII case-fold only and documents NFC as LOW residual, which PATTERNS.md explicitly validates as an acceptable planner decision. No blocker.

### Dimension 7b: Scope Reduction Detection — PASS

No scope reduction language found in any plan. The plans do not use v1/v2 versioning, "static for now", placeholders, or deferral language for any locked decision. SEC-03 is correctly handled as an "already-covered confirm" (the research established this as ALREADY-COVERED — confirmed by 127-02 Task 2's action, which explicitly cites the existing tests and treats a red result as a RULE 0 stop, not a fix-in-place).

### Dimension 7c: Architectural Tier Compliance — PASS

RESEARCH.md has an Architectural Responsibility Map. Checking task-to-tier assignments:

| Capability | Correct Tier | Plan Assignment |
|------------|-------------|-----------------|
| Sandbox confinement (denylist) | internal/files (Sandbox) | 127-01: sandbox.go denylistCheck only ✅ |
| CSRF Origin check | internal/webserver | Not modified (already implemented; 127-04 only adds e2e test) ✅ |
| Capability gate (files.write HasPerm) | internal/webserver | Not modified (already implemented) ✅ |
| e2e contract verification | frontend/e2e | 127-04: files-write.spec.ts ✅ |
| SECURITY artifact | .planning/ | 127-03: 127-SECURITY.md ✅ |

No tier mismatches. Crucially, 127-01's action explicitly prohibits touching write.go handlers or requireFilesWrite/originAllowedForWrite (middleware tier), keeping the security fix correctly in the sandbox tier.

### Dimension 8: Nyquist Compliance — PASS

VALIDATION.md exists at `127-VALIDATION.md`. Checking checks 8a–8d:

**8a — Automated Verify Presence:**
All 7 tasks across all 4 plans have `<automated>` commands in their `<verify>` blocks:
- 127-01 T1: `gofmt -l ... && go build ... && go vet ...` ✅
- 127-01 T2: `go test ./internal/files/ -run 'TestDenylist' -count=1 && go test -fuzz=FuzzSandboxWrite -fuzztime=60s` ✅
- 127-02 T1: `go test ./internal/files/ -run 'TestSandbox_WritePathSymlinkEscapeBlocked' -count=1 -v` ✅
- 127-02 T2: `go test -race ./internal/files/... -count=1 && go test -fuzz=FuzzSandboxWrite -fuzztime=60s` ✅
- 127-03 T1: `test -f .../127-SECURITY.md && grep -qi "capability-escalation..." && ...` ✅
- 127-03 T2: `go test -race ./internal/files/ -run 'TestWrite_TwoWritersIfMatchRace|TestWrite_InterruptedWritePreservesOriginal' -count=1 -v` ✅
- 127-04 T1: `pnpm exec playwright test files-write --reporter=line` ✅

**8b — Feedback Latency:**
127-01 T2 and 127-02 T2 include `go test -fuzz=FuzzSandboxWrite -fuzztime=60s` in their per-task verify commands. This is a 60-second command, exceeding the 30-second latency threshold.

**WARNING** (severity: warning, not blocker): The 60-second fuzz run is embedded in the per-task automated verify for 127-01 Task 2 and 127-02 Task 2. Per VALIDATION.md's own sampling contract ("full suite" = post-wave, not per-task), this is misaligned. The per-task verify for 127-01 T2 should be the unit test only (`go test ./internal/files/ -run 'TestDenylist' -count=1`); the fuzz gate belongs at wave-level. The current plan works correctly — the fuzz will complete and pass — but it embeds a 60s wait into mid-task CI feedback. Since this is an audit phase with small task count and the planner explicitly documented the latency in VALIDATION.md ("~30-90s + 60s fuzz"), this is an acceptable tradeoff.

**8c — Sampling Continuity:**
Wave 1: Tasks 127-01/T1, 127-01/T2, 127-04/T1 — all have automated verify ✅ (3/3)
Wave 2: Tasks 127-02/T1, 127-02/T2, 127-03/T1, 127-03/T2 — all have automated verify ✅ (4/4)
No window of 3 consecutive tasks without automated verify.

**8d — Wave 0 Completeness:**
VALIDATION.md Wave 0 requirements list the new tests as missing (❌ W0). The plans themselves ARE the Wave 0 delivery — they create the tests before production code runs against them. 127-01 Task 1 (production code) correctly comes after 127-01 Task 2 would establish test expectations via TDD mode. However, Task 1 is marked `type="auto" tdd="true"` and Task 2 is `type="auto"` — the task ordering within 127-01 places the production code change (Task 1) BEFORE the tests (Task 2). This is a minor sequencing note, not a blocker, because the test task is in the same plan and the executor handles intra-plan sequencing, and the `tdd="true"` flag on Task 1 signals the TDD intent.

### Dimension 9: Cross-Plan Data Contracts — PASS

Shared data: FuzzSandboxWrite corpus seeds (added in 127-01, consumed by 127-02). 127-01 adds f.Add() seeds; 127-02 runs the fuzz merge gate. 127-02's depends_on ["127-01"] correctly gates the fuzz run on the seed additions. No incompatible transforms. No shared stream modification conflict.

### Dimension 10: CLAUDE.md Compliance — PASS

CLAUDE.md relevant rules checked:

- **Go fmt/vet:** 127-01 T1 verify explicitly checks `gofmt -l` and `go vet` ✅
- **golangci-lint:** 127-01 T1 action says "Run gofmt; ensure golangci-lint clean" ✅
- **Virtual environment / Python:** Not applicable (Go-only changes) ✅
- **No global installs:** Not applicable ✅
- **Testing: pytest / vitest / jest:** Go tests use `go test` (correct); Playwright is already established ✅
- **No internal/daemon import from internal/files:** 127-01 explicitly prohibits this and includes a grep acceptance criterion ✅

### Dimension 11: Research Resolution — PASS

RESEARCH.md has an `## Open Questions` section (lines 246-257) WITHOUT a `(RESOLVED)` suffix, which would normally trigger a blocker. However, inspecting the three open questions:

1. SECURITY artifact filename/location — answered inline with a recommendation ("`.planning/phases/127-web-share-write-security-hardening/127-SECURITY.md`"). Plans adopt this recommendation (127-03 creates exactly that path).
2. gsd-security-auditor pattern — answered inline ("structured manual audit is sufficient").
3. Unicode NFC/NFD scope — answered inline ("ship ASCII case-fold; document NFC/NFD as LOW residual").

PATTERNS.md explicitly states "Unicode NFC — RESOLVED ASSUMPTION (overrides RESEARCH A1)" (PATTERNS.md line 85). All three questions have inline resolutions and the plans fully implement the recommendations. The missing `(RESOLVED)` suffix on the section heading is a documentation hygiene gap, not a substantive unresolved question.

**WARNING** (severity: warning): `## Open Questions` section in RESEARCH.md lacks the `(RESOLVED)` suffix. All three questions are materially answered inline and in PATTERNS.md, so execution is safe to proceed. The heading should be updated to `## Open Questions (RESOLVED)` for hygiene.

### Dimension 12: Pattern Compliance — PASS

PATTERNS.md has a File Classification table. Checking each file against plan references:

| File | PATTERNS.md Analog | Plan References Analog |
|------|-------------------|----------------------|
| internal/files/sandbox.go (denylistCheck) | itself (sandbox.go:114-146) | 127-01 T1 read_first cites sandbox.go:96-148 and PATTERNS.md "GAP 1/2 fix" sections ✅ |
| internal/files/write_test.go | TestDenylist_HomeRooted (write_test.go:478) | 127-01 T2 read_first cites write_test.go:464-607 ✅ |
| internal/files/sandbox_test.go (symlink test) | TestSandbox_SymlinkEscapeBlocked (sandbox_test.go:217) | 127-02 T1 read_first cites sandbox_test.go:217-248 ✅ |
| internal/files/sandbox_test.go (fuzz seeds) | FuzzSandboxWrite (sandbox_test.go:367) | 127-02 T1 and 127-01 T2 read_first cite sandbox_test.go:367-441 ✅ |
| frontend/e2e/files-write.spec.ts | standalone APIRequestContext smoke (spec.ts:570) | 127-04 T1 read_first cites files-write.spec.ts:570-588 ✅ |
| .planning/.../127-SECURITY.md | 83-SECURITY.md, 74-SECURITY.md, 60-SECURITY.md | 127-03 T1 read_first cites 83-SECURITY.md ✅ |

Shared patterns (capability enforcement, denylist chokepoint) are documented in PATTERNS.md Shared Patterns section and referenced in 127-03's context. All file classification entries have analog references in the corresponding plan.

---

## Net-New Code Isolation Check (Special Requirement)

The plan-check prompt requires explicit verification that net-new production code changes are ISOLATED to `denylistCheck` in `sandbox.go` — no plan may touch `WriteFileAtomic`, `Rename`, `Mkdir`, `MkdirAll`, or `Delete` method bodies.

- **127-01 Task 1 action:** "Modify ONLY `denylistCheck`. Do NOT touch WriteFileAtomic, Rename, Mkdir, MkdirAll, Delete, validateRelativePath, or any write.go handler" — explicitly stated ✅
- **127-01 Task 1 acceptance_criteria:** "WriteFileAtomic/Rename/Mkdir/Delete bodies unchanged (diff touches only denylistCheck)" — explicitly asserted ✅
- **127-02 acceptance_criteria:** "No production source file (sandbox.go, write.go) modified by this plan" ✅
- **127-03 acceptance_criteria:** "No production source modified" ✅
- **127-04 acceptance_criteria:** "No Go source file modified by this plan" ✅

ISOLATION CONFIRMED.

---

## STRIDE Coverage Check

The phase carries a STRIDE threat register across its plans. All 11 threat IDs (T-127-01 through T-127-11) are assigned, with dispositions (mitigate / accept) and mitigation plans. Key coverage:

| STRIDE Category | Threats | All Addressed? |
|-----------------|---------|---------------|
| Spoofing | T-127-11 (CSRF cross-origin) | ✅ 127-04 |
| Tampering | T-127-02 (case bypass), T-127-05 (symlink write), T-127-06 (fuzz), T-127-08 (concurrent write), T-127-10 (TOCTOU residual), T-127-12 (upload abuse) | ✅ 127-01/02/03 |
| Repudiation | Not applicable to this surface | N/A |
| Information Disclosure | T-127-09 (daemon socket) | ✅ accepted/documented in 127-03 |
| Denial of Service | T-127-12 (upload size cap, already covered) | ✅ 127-02 confirm |
| Elevation of Privilege | T-127-01 (macOS config dir), T-127-07 (capability escalation) | ✅ 127-01/03 |

STRIDE coverage is thorough for this surface's threat profile.

---

## Issues

```yaml
issues:
  - plan: null
    dimension: nyquist_compliance
    severity: warning
    description: "127-01 Task 2 and 127-02 Task 2 embed 'go test -fuzz=FuzzSandboxWrite -fuzztime=60s' in per-task <verify> commands, exceeding the 30s latency threshold. The fuzz gate is wave-level work but is embedded in per-task verification."
    fix_hint: "Split per-task verify to unit-test-only (go test ./internal/files/ -run 'TestDenylist|FuzzSandboxWrite' without -fuzz flag) and keep the -fuzz=FuzzSandboxWrite run in the plan-level <verification> block only. VALIDATION.md already documents this correctly at the wave-level — align the task verify commands to match."

  - plan: null
    dimension: research_resolution
    severity: warning
    description: "RESEARCH.md '## Open Questions' section header lacks the '(RESOLVED)' suffix required by Dimension 11. All three questions are answered inline and in PATTERNS.md, so execution is unblocked, but the heading is misleading."
    fix_hint: "Change '## Open Questions' to '## Open Questions (RESOLVED)' in 127-RESEARCH.md."
```

---

## Recommendation

**0 blockers. 2 warnings (both cosmetic/latency hygiene — neither affects correctness or security outcome).**

Plans are ready to execute. The 2 warnings may be addressed before execution or deferred; neither affects security coverage or test correctness.

Run `/gsd:execute-phase 127` to proceed.
