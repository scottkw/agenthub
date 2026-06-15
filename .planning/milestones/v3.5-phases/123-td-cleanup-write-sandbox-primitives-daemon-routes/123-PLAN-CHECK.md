# Phase 123 Plan Check

**Checked:** 2026-06-14
**Plans:** 4 (123-01, 123-02, 123-03, 123-04)
**Phase goal:** internal/files/ sandbox has all write primitives (atomic write, rename, delete, mkdir, upload), shell-RC denylist enforced on all write paths, TD-4 and TD-5 closed, daemon local-socket write routes live — a correct, trusted, fuzz-proven write API for all subsequent phases.

---

## Dimension 1: Requirement Coverage

| Req ID | Plans | Tasks | Status |
|--------|-------|-------|--------|
| FSW-01 | 123-01 | T1 (test), T3 (impl) | COVERED |
| FSW-02 | 123-01 | T1 (test), T3 (impl) | COVERED |
| FSW-03 | 123-01 | T1 (test), T3 (impl) | COVERED |
| FSW-04 | 123-01 | T1 (test), T3 (impl) | COVERED |
| FSW-05 | 123-03 | T1 (upload handler + test) | COVERED |
| FSW-06 | 123-01 | T2 (denylist), T3 (enforcement) | COVERED |
| FSW-07 | 123-01 | T1 (harness), T4 (fuzz gate) | COVERED |
| FSW-08 | 123-03 | T1 (handlers + routes), T2 (race gate) | COVERED |
| FSW-09 | 123-04 | T1 (client methods), T2 (race gate) | COVERED |
| FSW-10 | 123-02 | T1 (ExchangeJoinCodeAtURL fix) | COVERED |
| FSW-11 | 123-02 | T2 (WR-01..05 hardening) | COVERED |
| FSW-12 | 123-03 | T1 (MaxBytesReader 50 MiB) | COVERED |

Result: All 12 FSW requirements covered. PASS.

---

## Dimension 2: Task Completeness

All tasks have required fields (read_first, files, action, verify with automated command, done, acceptance_criteria). TDD tasks have behavior blocks. Automated verify commands are specific go test / pnpm test commands with named test patterns — not vague no-op commands. Done criteria are measurable acceptance conditions.

One structural note: Plan 01 has 4 tasks. This is at the warning threshold (see Dimension 5).

Result: PASS with one WARNING (scope_sanity, Plan 01 T-count).

---

## Dimension 3: Dependency Correctness

| Plan | Wave | depends_on | Valid? |
|------|------|-----------|--------|
| 123-01 | 1 | [] | PASS |
| 123-02 | 1 | [] | PASS — parallel to 01 (TD-4/TD-5 are independent of sandbox primitives) |
| 123-03 | 2 | ["123-01"] | PASS — needs sandbox write methods from Plan 01 |
| 123-04 | 3 | ["123-03"] | PASS — needs daemon write routes from Plan 03 |

No cycles. Wave numbering is consistent with max(deps)+1. Plan 03 context block references 123-01-SUMMARY.md (appropriate — generated when Plan 01 completes before Plan 03 starts). Plan 04 references 123-03-SUMMARY.md (same pattern). No forward references.

Result: PASS.

---

## Dimension 4: Key Links Planned

- 123-01: sandbox write methods → validateAndClean + denylistCheck + os.OpenRoot — the security chain is explicit in the key_links frontmatter and in the task actions (via the per-op pattern).
- 123-02: ExchangeJoinCodeAtURL → Location ?cap= → dedicated client with ErrUseLastResponse — wiring fully described.
- 123-03: api.go mux → a.filesHandler.Write/Upload/Delete/Rename/Mkdir — both the registration snippet and the handler-to-sandbox wiring (MaxBytesReader → WriteFileAtomic) are explicit.
- 123-04: DaemonClient methods → filesURL("write"/"upload"/etc.) → daemon write routes — the data flow mirrors the existing read methods exactly.

Result: PASS.

---

## Dimension 5: Scope Sanity

| Plan | Tasks | Files | Context Estimate | Status |
|------|-------|-------|-----------------|--------|
| 123-01 | 4 | 4 | moderate | WARNING — 4 tasks at threshold |
| 123-02 | 2 | 5 | light | PASS |
| 123-03 | 2 | 4 | moderate | PASS |
| 123-04 | 2 | 2 | light | PASS |

Plan 01's 4 tasks are structured as a deliberate TDD wave (Wave 0 RED → GREEN types → GREEN primitives → fuzz gate). Each task is narrowly scoped to one responsibility. This is the most defensible 4-task structure possible, and splitting further would break the RED→GREEN progression. The WARNING stands per the threshold but should not block execution.

Result: WARNING (plan 01 task count).

---

## Dimension 6: Verification Derivation

All must_haves.truths are user-observable or test-executable statements (no implementation-focused truths like "bcrypt installed"). Examples: "WriteFileAtomic writes via sibling temp + Sync + Rename; a concurrent reader never sees a partial/empty file" — observable via TestWriteFileAtomic_ConcurrentReadNeverPartial. "ExchangeJoinCodeAtURL parses a 303 Location ?cap=<token>" — observable via TestExchangeJoinCode_303Success.

Artifacts have specific `contains` fields (concrete function names / symbols). Key links specify the wiring method (per-op fresh os.OpenRoot, ErrUseLastResponse, MaxBytesReader pattern).

Result: PASS.

---

## Dimension 7: Context Compliance

CONTEXT.md declares no locked decisions (discuss phase skipped; all implementation at Claude's discretion). No deferred ideas are listed. Plans do not introduce any out-of-scope Phase 124+ items (PermFilesWrite, requireFilesWrite, CSRF, Origin checks, schemaVersion migration, FilesClient interface extension). The scope guard is explicitly called out in Plan 04's interfaces block and acceptance criteria.

Result: PASS.

---

## Dimension 7b: Scope Reduction Detection

No scope-reduction language found in any plan action. Plans do not use "v1", "static for now", "stub", "future enhancement", "simplified", "hardcoded", or similar deferral markers. Where REQUIREMENTS.md text is stale (FSW-02 claims os.Root.Rename does not exist), the plan correctly overrides it with the RESEARCH-verified finding and explains the correction explicitly in the task context. This is a scope CORRECTION, not a scope REDUCTION.

Result: PASS.

---

## Dimension 7c: Architectural Tier Compliance

RESEARCH.md Architectural Responsibility Map verified against all plan task targets:

| Capability | Map Owner | Plan Assignment | Match |
|-----------|-----------|----------------|-------|
| Path validation / sandbox confinement | internal/files (Sandbox) | Plan 01 sandbox.go | PASS |
| Shell-RC denylist | internal/files (Sandbox write methods) | Plan 01 sandbox.go denylistCheck | PASS |
| HTTP verb→operation mapping | internal/files (Handler write.go) | Plan 03 write.go | PASS |
| Route registration (loopback trust) | internal/daemon (api.go) | Plan 03 api.go | PASS |
| Multipart parse + size cap | internal/files (Handler.Upload) | Plan 03 write.go | PASS |
| Client write methods | internal/daemon (DaemonClient) | Plan 04 client.go | PASS |
| Join-code 303 parsing (TD-5) | internal/daemon (client_remote_files.go) | Plan 02 Task 1 | PASS |
| File-browser hardening (TD-4) | internal/webserver + frontend/src | Plan 02 Task 2 | PASS |

No security-sensitive capability is assigned to a less-trusted tier. Denylist remains in the Sandbox (server-side, all callers inherit) per RESEARCH anti-patterns.

Result: PASS.

---

## Dimension 8: Nyquist Compliance

VALIDATION.md exists. Check 8e: PASS.

| Task | Plan | Wave | Automated Command | Status |
|------|------|------|-------------------|--------|
| T1 (write tests RED) | 123-01 | 1 | `go vet ./internal/files/` | PASS |
| T2 (types + denylist) | 123-01 | 1 | `go build ./internal/files/ && go test ./internal/files/ -run TestDenylist` | PASS |
| T3 (write primitives) | 123-01 | 1 | `go test -race ./internal/files/ -run TestWriteFileAtomic|TestRename|TestMkdir|TestDelete|TestDenylist` | PASS |
| T4 (fuzz gate) | 123-01 | 1 | `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` | PASS |
| T1 (TD-5) | 123-02 | 1 | `go test -race ./internal/daemon/ -run TestExchangeJoinCode` | PASS |
| T2 (TD-4) | 123-02 | 1 | `pnpm test -- --run humanSize && pnpm exec tsc --noEmit` | PASS |
| T1 (write handlers + routes) | 123-03 | 2 | `go test -race ./internal/files/ ./internal/daemon/ -run ...` | PASS |
| T2 (race gate) | 123-03 | 2 | `go test -race ./internal/files/... ./internal/daemon/...` | PASS |
| T1 (client write methods) | 123-04 | 3 | `go test -race ./internal/daemon/ -run TestDaemonClientWrite|...` | PASS |
| T2 (full race gate) | 123-04 | 3 | `go test -race ./internal/files/... ./internal/daemon/...` | PASS |

Check 8b: No watch-mode flags. All commands are targeted test runs or bounded fuzz (60s fixed). PASS.
Check 8c: No window of 3+ consecutive tasks without automated verify. PASS.
Check 8d: No MISSING automated references. PASS.

VALIDATION.md file-name discrepancy (Wave 0 references `sandbox_write_test.go` and `fuzz_write_test.go`) is noted in Issues below.

Result: PASS.

---

## Dimension 9: Cross-Plan Data Contracts

Plan 01 produces: `Sandbox.WriteFileAtomic/Rename/Mkdir/MkdirAll/Delete`, `ErrProtectedSystemFile`, `FileWriteResponse`, `FileOpResponse`.
Plan 03 consumes: all of the above via the Handler, per the interfaces block.
Plan 04 consumes: daemon write routes from Plan 03 via DaemonClient.

No conflicting transforms. Plan 03's interfaces block precisely documents the Plan 01 contract (function signatures, wire types). Plan 04's interfaces block documents the Plan 03 route contract. Data flows downstream cleanly.

Result: PASS.

---

## Dimension 10: CLAUDE.md Compliance

| Rule | Plans | Status |
|------|-------|--------|
| Go context-aware functions (ctx context.Context) | Plan 04 acceptance criteria explicitly asserts ctx first param | PASS |
| pnpm preferred | Plan 02 Task 2 uses pnpm test / pnpm exec tsc | PASS |
| golangci-lint | Plans 01, 03, 04 verification blocks | PASS |
| gofmt | Plans 01, 02, 03, 04 verification blocks | PASS |
| No new external packages | Plans explicitly confirm stdlib-only; go.mod unchanged | PASS |
| No global Python/Node package installs | N/A — Go-only phase | PASS |

Result: PASS.

---

## Dimension 11: Research Resolution

RESEARCH.md has `## Open Questions` section without a `(RESOLVED)` suffix.

The three open questions are:
1. daemonSettings.FilesWrite — plans defer it (correct; no FSW req covers it)
2. Owner files.write default — flagged as Phase 124 concern; plans don't touch it (correct)
3. Windows rename retry — Plan 01 Task 3 explicitly implements the bounded 3-attempt retry

All three are answered by the plans, but the section heading does not carry the `(RESOLVED)` marker required by Dimension 11. This is an administrative compliance gap.

Result: WARNING — `## Open Questions` section heading should be `## Open Questions (RESOLVED)` to pass Dimension 11 formally.

---

## Dimension 12: Pattern Compliance

| File | PATTERNS.md Analog | Plan References Analog | Status |
|------|--------------------|----------------------|--------|
| sandbox.go (write methods) | sandbox.go:68-112 Open/Stat | Plan 01 T2/T3 read_first | PASS |
| write.go | handler.go:110-303 List/Stat/Read | Plan 03 T1 read_first | PASS |
| types.go | types.go:26-46 FileEntry/FileListResponse | Plan 01 T2 read_first | PASS |
| sandbox_test.go (FuzzSandboxWrite) | sandbox_test.go:256-339 FuzzSandboxPath | Plan 01 T1 read_first | PASS |
| write_test.go | sandbox_test.go conventions | Plan 01 T1 read_first | PASS |
| api.go (routes) | api.go:144-147 read-route block | Plan 03 T1 read_first | PASS |
| client.go (write methods) | client.go:381-449 ListFiles + doJSON:488 | Plan 04 T1 read_first | PASS |
| client_remote_files.go (TD-5) | joincode_prompt.go:153-208 | Plan 02 T1 read_first | PASS |
| TD-4 frontend | self-referential (PATTERNS.md TD-4 table) | Plan 02 T2 read_first | PASS |

All shared patterns applied: path validation (validateAndClean on ALL write methods, BOTH paths for Rename), per-op os.OpenRoot (never cached), HTTP status convention (404/403/400/200), client request→typed error, method-prefixed routes, loopback trust.

Result: PASS.

---

## Issues

```yaml
issues:

  - plan: null
    dimension: research_resolution
    severity: warning
    description: >
      RESEARCH.md '## Open Questions' section lacks the '(RESOLVED)' suffix required by
      Dimension 11. All three questions are answered by the plans (Q1 deferred to Phase 124,
      Q2 deferred to Phase 124, Q3 addressed in Plan 01 Task 3 Windows retry), but the
      formal marker is absent.
    fix_hint: >
      Change '## Open Questions' to '## Open Questions (RESOLVED)' in 123-RESEARCH.md and
      add an inline RESOLVED annotation to each question. No plan changes required.

  - plan: "123-01"
    dimension: scope_sanity
    severity: warning
    description: >
      Plan 01 has 4 tasks — at the scope_sanity warning threshold. Each task has a distinct
      responsibility (Wave 0 RED, types + denylist GREEN, primitives GREEN, fuzz gate), and
      splitting further would break the TDD wave progression. The 4-task structure is
      intentional and well-bounded.
    task: null
    metrics:
      tasks: 4
      files: 4
    fix_hint: >
      No structural change required. The TDD wave decomposition justifies 4 tasks. If the
      reviewer wants to reduce count, T1 (RED tests) + T2 (types + denylist) could be merged
      since both touch test infrastructure, but this is optional.

  - plan: "123-02"
    dimension: verification_derivation
    severity: warning
    description: >
      Success criterion #3 requires the shell-RC denylist to be enforced on all FIVE write
      methods including upload. Plan 01 Task 3 enforces denylistCheck inside WriteFileAtomic
      (which Upload routes through), so the denylist IS enforced. However, Plan 03 Task 1
      has no explicit TestHandlerUpload_DenylistForbidden test — the behavior list covers
      filename sanitization (TestHandlerUpload_FilenameSanitized) and size cap
      (TestHandlerUpload_OverCap413) but not a dedicated upload-to-denylist-path test at
      the HTTP handler layer.
    task: 1
    fix_hint: >
      Add TestHandlerUpload_DenylistForbidden to Plan 03 Task 1 behavior: multipart upload
      targeting a denylist path (e.g. dir=".ssh", file="authorized_keys") in a home-rooted
      sandbox should return 403 "Protected system file". This closes the success-criterion #3
      evidence gap for the upload route specifically.

  - plan: null
    dimension: task_completeness
    severity: warning
    description: >
      VALIDATION.md Wave 0 Requirements section names test files ('sandbox_write_test.go',
      'fuzz_write_test.go') that do not match the actual test files planned in 123-01-PLAN.md
      frontmatter ('write_test.go' and an extension of 'sandbox_test.go'). This could cause
      confusion during execution if an executor checks VALIDATION.md for Wave 0 targets.
    fix_hint: >
      Update VALIDATION.md Wave 0 Requirements section to match Plan 01's actual file names:
      'internal/files/write_test.go' (unit tests) and 'internal/files/sandbox_test.go'
      (FuzzSandboxWrite extension). No plan changes required.
```

---

## Threat-Model Completeness

ASVS L1 categories evaluated:

| ASVS | Category | Coverage | Status |
|------|----------|----------|--------|
| V4 Access Control | Shell-RC denylist | Plan 01 T2/T3; Plan 03 T1 TestHandlerWrite_DenylistForbidden; STRIDE T-123-03/14 | COVERED |
| V5 Input Validation | Path traversal + rename-dest traversal + multipart filename | Plan 01 T1 TestRename_DestinationTraversalRejected; Plan 03 T1 TestHandlerUpload_FilenameSanitized; FuzzSandboxWrite; STRIDE T-123-01 | COVERED |
| V5 Input Validation | Method-verb enforcement 405 | Plan 03 T1 TestFilesWriteRoutes_WrongVerb405; STRIDE T-123-15 | COVERED |
| V6 Cryptography | crypto/rand temp suffix; cap-token redaction | Plan 01 T3 action (O_EXCL + rand); Plan 02 T1 (redactCapTokenFromError); STRIDE T-123-07 | COVERED |
| V12 File & Resources | 50 MiB MaxBytesReader before parse; atomicity | Plan 03 T1 TestHandlerUpload_OverCap413 + MaxBytesReader assertion; Plan 01 T3 WriteFileAtomic; STRIDE T-123-04/05/13 | COVERED |

No HIGH-severity ASVS gap found. The deferred auth/CSRF surface (T-123-16) is explicitly accepted with documentation (Phase 124 scope). The upload denylist test gap (noted in issues) is a WARNING, not a HIGH-severity ASVS blocker — the kernel protection exists at the sandbox layer.

---

## Success Criteria Traceability

| Success Criterion | Plans | Evidence |
|------------------|-------|---------|
| #1 FuzzSandboxWrite 60s zero crashes | 123-01 T4 | fuzz gate in Task 4 verify block; TestRename_DestinationTraversalRejected + write seeds in FuzzSandboxWrite |
| #2 PUT write → 200, atomic temp+sync+rename, GET read-back identical | 123-03 T1 | TestHandlerWrite_RoundTrip; curl smoke in Plan 03 verification |
| #3 denylist 403 on all five write methods | 123-01 T2/T3, 123-03 T1 | TestDenylist_HomeRooted (sandbox), TestHandlerWrite_DenylistForbidden (HTTP); upload 403 not explicitly tested at HTTP layer (WARNING) |
| #4 ExchangeJoinCodeAtURL parses 303 cap | 123-02 T1 | TestExchangeJoinCode_303Success + TestExchangeJoinCode_NoAutoFollow + TestExchangeJoinCode_SharedClientUnchanged |
| #5 go test -race green; five routes on Unix socket, no auth | 123-01..04 (all Task 2 race gates) | Full suite race gate in Plan 04 T2; TestFilesWriteRoutes_Registered in Plan 03 T1 |

All five success criteria have direct plan coverage. Success criterion #3 has a completeness WARNING (upload HTTP layer).

---

## Plan Summary

| Plan | Wave | Tasks | Files | Status |
|------|------|-------|-------|--------|
| 123-01 | 1 | 4 | 4 | Valid (WARNING: 4 tasks) |
| 123-02 | 1 | 2 | 5 | Valid |
| 123-03 | 2 | 2 | 4 | Valid |
| 123-04 | 3 | 2 | 2 | Valid |

Total issues: 0 blockers, 3 warnings.

---

## PLAN CHECK PASSED

All 12 FSW requirements are covered. All 5 phase success criteria have direct plan coverage. No BLOCKER issues found. Dependency graph is valid and acyclic. Threat model is complete with no unmitigated HIGH-severity ASVS gaps. All tasks have specific automated verify commands.

The 3 WARNING items are:

1. **RESEARCH.md open questions need `(RESOLVED)` suffix** — administrative, no plan change required
2. **Plan 01 has 4 tasks** — intentional TDD wave structure, no split required
3. **No TestHandlerUpload_DenylistForbidden test** — the sandbox layer protects correctly; adding an explicit HTTP-layer test for this case strengthens success criterion #3 evidence but is not a blocker

**Recommendation:** Address WARNING #3 (add TestHandlerUpload_DenylistForbidden to Plan 03 Task 1 behavior list) before execution. WARNINGs #1 and #2 can be addressed at execution time. Execution may proceed.
