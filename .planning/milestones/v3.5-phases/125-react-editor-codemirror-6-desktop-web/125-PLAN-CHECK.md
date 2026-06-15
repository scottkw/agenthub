# Phase 125 — Plan Check

**Checked:** 2026-06-14
**Plans verified:** 125-01 through 125-06 (6 plans, 13 tasks total, 2 human-verify checkpoints)
**Checker:** gsd-plan-checker

---

## PLAN CHECK PASSED

All 13 EDIT requirements are covered. All blockers listed in earlier iterations are resolved in the plans. No blocker-severity issues remain. Two warnings are noted below.

---

## Coverage Summary

| Requirement | Plans | Status |
|-------------|-------|--------|
| EDIT-01 (CM6 install + vendor_drift_test) | 125-01, 125-06 | Covered |
| EDIT-02 (Editor.tsx, Compartment toggle) | 125-02 | Covered |
| EDIT-03 (Edit button gated on canWrite + !binary) | 125-02 | Covered |
| EDIT-04 (syntax highlighting by extension, Bash via legacy-modes) | 125-02 | Covered |
| EDIT-05 (Cmd/Ctrl+S → PUT with If-Match) | 125-01 (server), 125-03 (client) | Covered |
| EDIT-06 (dirty indicator + 3-state save) | 125-03 | Covered |
| EDIT-07 (React-level unsaved guard, no beforeunload) | 125-03 | Covered |
| EDIT-08 (412 conflict modal, buffer preserved) | 125-01 (server), 125-03 (client) | Covered |
| EDIT-09 (create/mkdir/delete/rename/move + 409 modal) | 125-04 | Covered |
| EDIT-10 (upload single+multi, XHR progress, per-file 409) | 125-05 | Covered |
| EDIT-11 (large-file guard >500KB / near-5MB plain-text) | 125-02 | Covered |
| EDIT-12 (useFilesWrite hook + canWrite gating everywhere) | 125-02, 125-04, 125-05 | Covered |
| EDIT-13 (14-scenario cross-browser Playwright gate) | 125-06 | Covered |

---

## Plan Summary

| Plan | Wave | Tasks | Type | Depends On | Status |
|------|------|-------|------|------------|--------|
| 125-01 | 1 | 3 auto | execute (autonomous) | — | Valid |
| 125-02 | 2 | 1 checkpoint + 2 auto | execute (non-autonomous) | 125-01 | Valid |
| 125-03 | 3 | 2 auto | execute (autonomous) | 125-02 | Valid |
| 125-04 | 4 | 2 auto | execute (autonomous) | 125-03 | Valid |
| 125-05 | 5 | 2 auto | execute (autonomous) | 125-04 | Valid |
| 125-06 | 6 | 1 auto + 1 checkpoint | execute (non-autonomous) | 125-05 | Valid |

---

## Dimension-by-Dimension Findings

### Dimension 1: Requirement Coverage — PASS

All 13 EDIT-XX IDs from REQUIREMENTS.md are claimed in at least one plan's `requirements` frontmatter field and have concrete covering tasks. No requirement is left to a vague umbrella task.

**Notes on coverage completeness:**
- EDIT-05/08 coverage is split correctly across plans: 125-01 owns the server-side 412 emission (the enabling condition), and 125-03 owns the client-side If-Match send + conflict modal. This is a valid split — the server task is Wave 1, the client task Wave 3 that depends on it.
- EDIT-01 appears in both 125-01 (vendor_drift_test scaffolding) and 125-06 (gate runs in the final Playwright verify). Both are required: the test exists from Wave 1 and becomes load-bearing when CM6 is installed in Wave 2.
- EDIT-12 (canWrite gating) is appropriately distributed: hook created in 125-02, filled in 125-04 and 125-05. Every affordance plan asserts `canWrite` gating in its acceptance criteria.
- EDIT-13 is covered by 125-06 Task 1, which explicitly names all 14 scenarios from RESEARCH §Validation.

### Dimension 2: Task Completeness — PASS

All `type="auto"` tasks contain `<read_first>`, `<action>`, `<verify>` (with `<automated>` command), `<acceptance_criteria>`, and `<done>` fields. Actions are specific (exact file paths, line references, concrete code patterns). Verify commands are runnable.

**Checkpoint tasks (125-02 pnpm legitimacy, 125-06 desktop parity):**
The structure validator notes these differently because checkpoint tasks have no `<files>`, `<verify>`, or `<done>` fields — this is the correct schema for `type="checkpoint:human-verify"`. Both checkpoints are genuinely non-automatable:
- 125-02 checkpoint: slopcheck is unavailable (verified in RESEARCH); org-scoped provenance is corroborated but a human eyes-on npmjs.com check before an `pnpm add` of 17 packages is legitimate security hygiene, not a masking of missing automation.
- 125-06 checkpoint: VALIDATION.md §Manual-Only explicitly identifies Wails desktop Tab/Cmd-V interaction and visual desktop render as automation-impossible; the checkpoint is the correct gate for both. Cross-surface parity is release-blocking (MEMORY).

Both checkpoints are legitimately scoped and not masking missing automation.

### Dimension 3: Dependency Correctness — PASS

Dependency chain: 125-01 → 125-02 → 125-03 → 125-04 → 125-05 → 125-06. Strictly serial, no cycles. Wave numbers match the depends_on chain (Wave N depends on Wave N-1). No forward references.

**Shared-file safety:** The checker prompt notes concern about parallel modification of shared files (filesApi.ts, useFilesWrite.ts, FileBrowserTab.tsx, FileRow.tsx). Since all plans are strictly serialized (each depends on the previous), there is no parallel execution risk. The shared files are modified sequentially: filesApi.ts in 125-02 → extended in 125-03 → extended again in 125-04 → extended again in 125-05. Each extension is additive (adding methods, not rewriting existing ones), which is safe in a serial chain.

### Dimension 4: Key Links Planned — PASS

All critical wiring is explicitly in at least one plan's `key_links` or task action:
- Server ETag → client If-Match echo: 125-01 key_links + 125-02 filesApi etag extension + 125-03 writeFile If-Match header.
- Editor Cmd-S → useFilesWrite.write: 125-03 key_links and Task 1 action.
- 412 response → ConflictModal: 125-03 key_links (isConflict → open modal) and Task 2 action.
- FileRowActions → useFilesWrite del/rename/mkdir: 125-04 key_links and Task 1 action.
- XHR upload → onprogress: 125-05 key_links and Task 1 action.
- Playwright WRITE_CAP fixture → 403 test scenario: 125-06 key_links and Task 1 action.
- vendor_drift_test → package.json↔pnpm-lock parity: 125-01 Task 3 action (NOT a web/vendor/codemirror/ file copy — correctly aligned with RESEARCH Open Q1 resolution).

### Dimension 5: Scope Sanity — PASS

All plans are within the 2-3 task target (none exceed 3 auto tasks). Plan-01 has 3 tasks (backend, tests, fixture+drift). Plans 02-05 have 2 tasks each. Plan 06 has 1 auto + 1 checkpoint. Files modified per plan are within the 10-file warning threshold (125-04 has 10 files — borderline but within the warning range, not the blocker threshold of 15+).

### Dimension 6: Verification Derivation — PASS

All `must_haves.truths` across all six plans are user-observable behaviors, not implementation details:
- "PUT with stale If-Match returns 412" — observable HTTP behavior
- "Clicking the pencil Edit button mounts CM6 editor" — observable UI behavior
- "Navigating away while dirty opens UnsavedChangesModal" — observable interaction
- "Playwright cross-browser passes all 14 scenarios" — verifiable test outcome

`must_haves.artifacts` have concrete paths with `provides` and `contains` assertions. `must_haves.key_links` cover the critical wiring points with pattern strings.

### Dimension 7: Context Compliance — PASS

All locked decisions from CONTEXT.md are implemented:

| Locked Decision | Implementing Plan/Task |
|-----------------|------------------------|
| CodeMirror 6, syntax highlighting by extension | 125-02 |
| Edit absent for binary + !canWrite | 125-02 Task 2 |
| >500KB warn; near-5MB plain-text | 125-02 Task 2 |
| Cmd/Ctrl+S → atomic If-Match PUT | 125-01 (server) + 125-03 (client) |
| 3-state save indicator (idle/saving/saved ~1.5s) | 125-03 Task 2 |
| React-level unsaved guard (NO beforeunload) | 125-03 Task 2 |
| 412 → [Force/Save-as-new/Discard], buffer never discarded | 125-03 |
| canWrite gating on all write affordances | 125-02, 125-04, 125-05 |
| 409 collision → Cancel-default modal | 125-04 |
| Recursive dir delete → count confirmation | 125-04 Task 2 |
| Playwright cross-browser (Chromium+Firefox+WebKit), zero CSP violations | 125-06 |
| vendor_drift_test.go, package.json↔pnpm-lock parity | 125-01 |

No deferred ideas (there are none in CONTEXT.md) are included. No scope reduction language detected.

### Dimension 7b: Scope Reduction Detection — PASS

No scope reduction language found in any plan action. No "v1/static for now/future enhancement" patterns. The ETag Open Q6 (A3 vs server-emit) is resolved to "server-emits, client echoes verbatim" in 125-02's `<interfaces>` block and 125-01's implementation, which is the full delivery of the locked decision. The recursive-dir count (RESEARCH Open Q3) is resolved to client-side listFiles walk — implemented in 125-04 Task 2 action. All six RESEARCH open questions are resolved in the plans.

### Dimension 7c: Architectural Tier Compliance — PASS

The Architectural Responsibility Map from RESEARCH.md is correctly honored:

| Capability | Expected Tier | Plan Assignment | Verdict |
|------------|---------------|-----------------|---------|
| If-Match / 412 conflict detection | API/Backend | 125-01 Task 1 (write.go) | Correct |
| ETag emission | API/Backend | 125-01 Task 1 (handler.go) | Correct |
| Text editing UI / syntax highlighting | Browser/Client | 125-02 Task 2 (Editor.tsx) | Correct |
| Edit/affordance gating (canWrite UI) | Browser/Client (UX) | 125-02, 125-04, 125-05 | Correct |
| Capability enforcement (server authority) | API/Backend | Not modified in 125 (shipped 124) | Correct |
| Multipart upload parse | API/Backend | Not modified in 125 (shipped 123) | Correct |

No security-sensitive capability is placed in a lower-trust tier than the map specifies. The plans explicitly note that canWrite UI gating is advisory and the server (requireFilesWrite + HasPerm) is the real authority.

### Dimension 8: Nyquist Compliance

VALIDATION.md exists. `nyquist_compliant: false` and `wave_0_complete: false` reflect the pre-execution state (correct — these flip to true during execution as Wave 0 tasks complete).

| Task | Plan | Wave | Automated Command | Status |
|------|------|------|-------------------|--------|
| Task 1 (If-Match/412 + ETag) | 125-01 | 1 | `go test ./internal/files/... -run TestWrite_IfMatch -count=1` | TDD |
| Task 2 (TestWrite_IfMatch*) | 125-01 | 1 | `go test ./internal/files/... -run TestWrite_IfMatch -count=1 -race` | TDD |
| Task 3 (vendor_drift + WRITE_CAP) | 125-01 | 1 | `go test ./internal/webserver/... -run CodeMirror -count=1 && tsc --noEmit` | Auto |
| Task 1 (install CM6 + languageFor + canWrite) | 125-02 | 2 | `cd frontend && pnpm test -- --run useFilesCapability && go test ./internal/webserver/... -run CodeMirror -count=1` | Auto |
| Task 2 (Editor.tsx + Compartment + Edit button) | 125-02 | 2 | `cd frontend && pnpm test -- --run Editor` | TDD |
| Task 1 (useFilesWrite hook + Cmd-S) | 125-03 | 3 | `cd frontend && pnpm test -- --run Editor` | TDD |
| Task 2 (EditorHeader + Modals + ConflictModal) | 125-03 | 3 | `cd frontend && pnpm test -- --run Editor && pnpm exec tsc --noEmit` | TDD |
| Task 1 (del/rename/mkdir + InlineNameInput + FileRowActions) | 125-04 | 4 | `cd frontend && pnpm test -- --run FileRowActions && pnpm exec tsc --noEmit` | TDD |
| Task 2 (Delete/Collision/MoveTo modals + wiring) | 125-04 | 4 | `cd frontend && pnpm test -- --run FileRowActions && pnpm exec tsc --noEmit` | TDD |
| Task 1 (XHR upload + filesApi) | 125-05 | 5 | `cd frontend && pnpm test -- --run Upload` | TDD |
| Task 2 (UploadQueuePanel + UploadDropOverlay + tab wiring) | 125-05 | 5 | `cd frontend && pnpm test -- --run Upload && pnpm exec tsc --noEmit` | TDD |
| Task 1 (14-scenario e2e + CSP) | 125-06 | 6 | `cd frontend && pnpm exec playwright test files-write.spec.ts web-csp.spec.ts && go test ./internal/webserver/... -run CodeMirror -count=1` | Auto |
| Desktop parity checkpoint | 125-06 | 6 | (manual — documented Manual-Only in VALIDATION.md) | Checkpoint |

**Sampling:** Every wave has ≥2 tasks with automated verify. No 3 consecutive tasks without automation. No watch-mode flags (`--watchAll` absent from all commands). All commands estimated <30s for component tests; Playwright is explicitly the pre-verify gate (acceptable per VALIDATION.md §Sampling Rate). No MISSING markers — all Wave 0 tests exist in the same wave as the tasks that require them.

Overall: PASS

### Dimension 9: Cross-Plan Data Contracts — PASS

The primary shared data entity is the ETag string (`"<UnixNano>-<size>"`). The format contract is established once in 125-01 (server emits in handler.go) and echoed verbatim by the client via `res.headers.get('etag')` in 125-02/03. No plan transforms the ETag — it is passed through unchanged. The shared filesApi.ts is extended additively across 125-02 → 125-03 → 125-04 → 125-05; each plan adds new methods and does not rewrite existing ones. No incompatible transforms on shared data entities detected.

### Dimension 10: CLAUDE.md Compliance — PASS

| CLAUDE.md Rule | Plan Compliance |
|----------------|-----------------|
| pnpm preferred (not npm/yarn) | 125-02 explicitly uses `pnpm add`; all frontend verify commands use `pnpm test` / `pnpm exec playwright` |
| vitest for JS/TS unit tests | All frontend unit tests use `pnpm test` (vitest); no Jest |
| go fmt / golangci-lint conventions | Plans don't add linting steps, but they're adding to existing files where linting runs in CI |
| TypeScript types; ESLint + Prettier | All new `.tsx` files are TypeScript; tsc --noEmit runs in verify steps |
| Python venv rule | Not applicable (no Python in this phase) |
| Wails prod build requires -tags wailsassets | RESEARCH/RUNTIME INVENTORY explicitly notes this; plans modify frontend/dist indirectly via pnpm add; no plan incorrectly omits this for production |

No forbidden patterns introduced.

**Note:** Plans do not explicitly add `eslint` or `prettier` run steps to their verify blocks. This is consistent with the existing pattern in the codebase (CLAUDE.md says "ESLint + Prettier" as conventions, not as a mandatory per-task hook). This is a WARNING, not a blocker — the project may run linting in CI separately.

### Dimension 11: Research Resolution — PASS

RESEARCH.md has an `## Open Questions` section. All six open questions are resolved in the plans:

| Open Question | Resolution in Plans |
|---------------|---------------------|
| OQ1: vendor-drift mechanism (package.json↔pnpm-lock, NOT web/vendor/) | 125-01 Task 3 action explicitly names this and avoids web/vendor/codemirror/ |
| OQ2: Bash via @codemirror/legacy-modes | 125-02 Task 1 action uses StreamLanguage.define(shell) from legacy-modes |
| OQ3: recursive-dir delete count source | 125-04 Task 2 action: client-side listFiles walk |
| OQ4: theme (one-dark vs TokyoNight) | 125-02 interfaces block: hand-rolled EditorView.theme with existing TokyoNight hexes; theme-one-dark explicitly dropped |
| OQ5: canWrite source per surface | 125-02 interfaces block: desktop=SessionInfo.FilesWrite, web-share=write-route probe |
| OQ6: etag timestamp format alignment | 125-01/02: server emits ETag, client echoes verbatim (eliminating format-mismatch risk) |

The `## Open Questions` heading does not carry a `(RESOLVED)` suffix, but all questions have inline resolutions in the plans. This is a minor process gap but does not affect execution.

### Dimension 12: Pattern Compliance — PASS

All 16 files in PATTERNS.md `## File Classification` table have plans that reference the correct analog. Key verifications:

- `internal/files/write.go` → analog Handler.Upload/Handler.Read in same pkg; 125-01 Task 1 action explicitly reads both
- `frontend/src/components/FileBrowser/modals/*.tsx` → analog QuitConfirmModal.tsx; 125-03/04 explicitly copy the overlay/Escape/safe-default-focus/acting-guard pattern
- `vendor_drift_test.go` → analog TestXtermVendorVersionsMatchPnpmLock; 125-01 Task 3 reads that file and reuses the lock-parse loop
- `languageFor.ts` (NO ANALOG) → RESEARCH §Code Examples is the documented source; 125-02 Task 1 action references RESEARCH explicitly
- `cmd/playwright-fixture/main.go` → analog existing owner/viewer cap mint lines 178-202; 125-01 Task 3 reads those lines

Shared patterns (Heroicons, BEM classes, modal overlay pattern, API client query/error mapping) are present in the relevant plan actions and enforced via acceptance criteria grep checks.

---

## Specific Verifications on Requested Topics

### Backend task (125-01): ETag contract
**Contract = SERVER EMITS ETag, CLIENT ECHOES VERBATIM** — confirmed. 125-01 Task 1 adds `ETag` header to `Handler.Read` (both zero-byte and ServeContent branches). 125-02 filesApi `readFileText` returns `etag: res.headers.get('etag')`. 125-03 `useFilesWrite.write` threads it as `If-Match`. The validator format `"<ModTime().UnixNano()>-<Size()>"` is set once in 125-01 and echoed unchanged. Unit test asserts 412 on mismatch, 200 on match, 200 on new-file (three TestWrite_IfMatch* cases in 125-01 Task 2).

### Vendor-drift gate definition
Confirmed package.json↔pnpm-lock CodeMirror version parity (NOT a web/vendor/codemirror/ file manifest). 125-01 Task 3 action explicitly: "Do NOT create a web/vendor/codemirror/ directory or VERSION manifest — CodeMirror is Vite-bundled." UI-SPEC's incorrect `web/vendor/codemirror/` wording is noted as superseded.

### lang-shell NOT used (legacy-modes for Bash)
Confirmed. 125-02 Task 1 action: "StreamLanguage.define(shell) from @codemirror/legacy-modes/mode/shell for sh/bash/zsh (RESEARCH Open Q2 — @codemirror/lang-shell does not exist)." Acceptance criteria greps for "legacy-modes" in languageFor.ts.

### Colorblind contract
All save-indicator states (idle/saving/saved/error) carry icon+text in every relevant plan's behavior and acceptance criteria. 125-03 Task 2 action explicitly names icons per state (ArrowPathIcon+"Saving…", CheckCircleIcon+"Saved", ExclamationTriangleIcon for error). Conflict modal carries ExclamationTriangleIcon + heading text. Delete modal carries TrashIcon + verb. Acceptance criteria for 125-03 Task 2 include `grep -c "aria-live" SaveIndicator.tsx >= 1`. The plans' verify blocks include source-level grep checks (never "verify by eye"). Colorblind table in UI-SPEC §Color is the authoritative contract and is referenced in all relevant plans.

### Cross-surface parity: desktop + web-share
125-02 canWrite implementation: desktop derives from SessionInfo.FilesWrite; web-share probes the write route (403 map). All affordances gated on canWrite. The 125-06 desktop parity checkpoint explicitly covers Wails-specific Tab/Cmd-V and visual render. Cross-surface parity is release-blocking per MEMORY and is enforced in 125-06 as a blocking-human checkpoint.

### Playwright 14 scenarios + CSP + WRITE_CAP
125-06 Task 1 action enumerates all 14 EDIT-13 scenarios verbatim (matching the REQUIREMENTS.md list and RESEARCH §Validation). WRITE_CAP fixture (minted in 125-01 Task 3) is used for web-share write + 412 scenarios; the existing viewer read-only cap covers the 403 scenario. CSP extension: `web-csp.spec.ts` is extended to drive the editor + write flow. Verify command runs both specs + vendor_drift gate.

### Wave serialization for shared files
All plans depend strictly on the previous plan (125-01 → 125-02 → ... → 125-06). No parallel execution is planned. filesApi.ts is extended additively in plans 02/03/04/05; each extension adds new exports without rewriting existing ones. No race conditions.

### Checkpoint task legitimacy
- 125-02 checkpoint (pnpm legitimacy): Justified — slopcheck unavailable, 17 packages being installed, human spot-check on npmjs.com is appropriate security hygiene before adding a large dependency bundle. The drift gate (automated, Wave 1) handles ongoing version parity; this checkpoint gates the initial install.
- 125-06 checkpoint (desktop parity): Justified — VALIDATION.md §Manual-Only explicitly documents these as non-automatable (Wails desktop keyboard/clipboard interaction, visual render). The checkpoint covers exactly the two residues: Tab/Cmd-V conflict and visual parity verification.

### Threat-model completeness (ASVS L1)
- **Stored XSS via CM6**: T-125-04 in 125-02 explicitly identifies this and mitigates via "CM6 renders content as text via its own DOM layer, never innerHTML; no rehype-raw; verify no dangerouslySetInnerHTML in Editor.tsx." The acceptance criteria includes a grep to confirm `theme-one-dark` is absent but does not explicitly grep for `dangerouslySetInnerHTML`. The task action says "verify no dangerouslySetInnerHTML" but the `<acceptance_criteria>` block does not contain that grep. This is a **WARNING** (see below).
- **412 force-clobber**: T-125-07 mitigated — force-overwrite reachable only via explicit ConflictModal user choice; default focus is "Discard my changes" (safe path).
- **Upload injection (Phase 123 caps reused)**: T-125-13 — server sanitizes via filepath.Base + validateAndClean (shipped 123). Plans correctly do not re-implement server-side upload filename sanitization.
- **Capability bypass on web-share**: T-125-05/16 — canWrite UI gate is explicitly advisory; server requireFilesWrite is the authority. e2e 403-without-cap scenario asserts this.

---

## Issues

```yaml
issues:
  - plan: "125-02"
    dimension: "task_completeness"
    severity: "warning"
    description: >
      Task 2 action says 'verify no dangerouslySetInnerHTML in Editor.tsx' but the
      <acceptance_criteria> block does not include a grep assertion for it. The XSS threat
      (T-125-04) is correctly identified in the threat model but the automated acceptance
      criterion that would confirm it is absent.
    task: 2
    fix_hint: >
      Add to Task 2 acceptance_criteria: 
      `grep -c "dangerouslySetInnerHTML" frontend/src/components/Editor.tsx` == 0
      This makes the stored-XSS mitigation verifiable at source rather than relying on the
      executor remembering the prose instruction.

  - plan: "125-06"
    dimension: "task_completeness"
    severity: "warning"
    description: >
      Task 1 acceptance_criteria verifies that "the spec file references all 14 scenario
      names" but does not enumerate the expected scenario strings. This means the acceptance
      criterion is met by any 14 describe/it blocks regardless of content. The behavioral
      content of each scenario is described only in the action prose and the RESEARCH
      reference, not in a verifiable acceptance criterion.
    task: 1
    fix_hint: >
      Add at least a spot-check grep for the two hardest-to-forget scenarios:
      `grep -c "412" frontend/e2e/files-write.spec.ts` >= 1 (conflict flow)
      `grep -c "binary" frontend/e2e/files-write.spec.ts` >= 1 (binary no-edit)
      This doesn't replace reading the file but makes partial scenario coverage visible.
```

---

## Recommendation

Both issues are warnings. Neither prevents execution. The plans are ready to execute in wave order (125-01 → 125-06). The two warnings are minor hardening improvements that can be addressed either before execution starts or as the executor naturally implements the missing grep checks.

**Execute:** `gsd:execute-phase 125`
