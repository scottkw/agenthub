# Phase 124 Plan Check

**Checked:** 2026-06-14
**Plans reviewed:** 124-01, 124-02, 124-03, 124-04, 124-05
**Method:** Goal-backward verification against phase goal + 5 success criteria + 12 verification dimensions

---

## PLAN CHECK FAILED

**1 BLOCKER, 3 WARNINGS**

The plans are architecturally sound and cover all 10 CAP requirements. The three critical inversions are correctly encoded. Cross-surface parity is planned in both GUI (124-04) and TUI (124-05) plans. The BLOCKER is a documentation formality (RESEARCH.md Open Questions not formally marked resolved), not a design flaw — fix is 2 lines.

---

## Requirement Coverage

| Requirement | Covered By | Status |
|-------------|-----------|--------|
| CAP-01 | 124-01 | COVERED |
| CAP-02 | 124-01 | COVERED |
| CAP-03 | 124-01 | COVERED |
| CAP-04 | 124-02, 124-04 | COVERED |
| CAP-05 | 124-04 | COVERED |
| CAP-06 | 124-02, 124-04, 124-05 | COVERED |
| CAP-07 | 124-01 | COVERED |
| CAP-08 | 124-02 | COVERED |
| CAP-09 | 124-01 | COVERED |
| CAP-10 | 124-03 | COVERED |

All 10 CAP requirements have covering tasks.

---

## Success Criteria Trace

| Success Criterion | Covering Plan(s) | Verdict |
|-------------------|-----------------|---------|
| SC#1: 403 without files.write / 2xx with on all 5 routes | 124-01 Task 0+2, TestRequireFilesWrite | COVERED |
| SC#2: mismatched Origin → 403; absent Origin → pass | 124-01 Task 1 originAllowedForWrite (Critical Inversion 1) | COVERED |
| SC#3: TestHasPerm_NoStringsContains_Write static-grep gate | 124-01 Task 0 | COVERED |
| SC#4: owner toggle + viewer opt-in + home-dir warning GUI AND TUI | 124-02 (server), 124-04 (GUI), 124-05 (TUI) | COVERED |
| SC#5: TestSettingsMigration_FilesWriteDefaultsFalse; persists across restart | 124-02 Task 0+1 | COVERED |

---

## Dimension Results

### Dimension 1: Requirement Coverage — PASS
All 10 CAP IDs appear in plan frontmatter `requirements` fields. No orphaned requirements.

### Dimension 2: Task Completeness — PASS
All plans have 3 tasks or fewer (max 3/plan). Every `<task type="auto">` has `<files>`, `<action>`, `<verify>/<automated>`, `<acceptance_criteria>`, and `<done>` fields. TDD tasks include `<behavior>` blocks. `<read_first>` is present throughout. Task actions are concrete and line-referenced.

### Dimension 3: Dependency Correctness — PASS WITH WARNING
- 124-01 (wave 1, depends_on: []) — correct.
- 124-02 (wave 1, depends_on: []) — see WARNING-1.
- 124-03 (wave 2, depends_on: ["124-02"]) — correct.
- 124-04 (wave 3, depends_on: ["124-01","124-02","124-03"]) — correct. Both plans modifying api.go (124-03 wave 2, 124-04 wave 3) are serialized.
- 124-05 (wave 2, depends_on: ["124-02"]) — correct.
- No cycles. No missing references.

### Dimension 4: Key Links Planned — PASS WITH WARNING
All plans have well-formed `key_links` frontmatter with `from/to/via/pattern`. Critical wires are traced:
- server.go → requireFilesWrite → filesDispatch (124-01)
- api.go issueCapabilitiesForSession → filesWriteEnabledFor (124-02)
- proxyRemoteFiles → r.Body for write verbs (124-03)
- DaemonManagerPanel.tsx → SetSessionFilesWrite (124-04)
- renderFilesTab → SessionInfo.HomeDir && FilesWrite (124-05)

See WARNING-3 for Wails TS binding gap.

### Dimension 5: Scope Sanity — PASS
All plans: 2–3 tasks each. Total files across all plans: 16 unique files (split across 5 focused plans). No plan exceeds the warning threshold.

### Dimension 6: Verification Derivation — PASS
All `must_haves.truths` are user-observable:
- "403 on all five webserver write routes" (observable from HTTP response)
- "schemaVersion 3 migrates to 4 with FilesWrite default false" (observable from settings.json)
- "warning visible in GUI and TUI" (observable at source level per colorblind rule)
All `artifacts` have `contains` assertions that verify wiring, not just file existence.

### Dimension 7: Context Compliance — PASS
All four locked decisions are implemented:
- D-1: files.write OPT-IN for all tokens (no default-on): 124-02 FilesWrite plain bool zero-value, 124-04 default OFF toggle.
- D-2: Phase 88 CSRF pattern (absent → pass vacuously): 124-01 originAllowedForWrite with Critical Inversion 1.
- D-3: HasPerm (not strings.Contains): 124-01 Task 1 + static-grep gate (Task 0).
- D-4: schemaVersion 3→4, FilesWrite default false: 124-02 Task 0+1.
No deferred ideas present. No scope reductions detected.

### Dimension 7b: Scope Reduction — PASS
No "v1/v2", "static for now", "hardcoded", "future enhancement", "not wired to", or "stub" language found in task actions. All three inversions are fully implemented (not deferred). The per-session persistence model (Q1) is fully specified in 124-02 Task 1 action.

### Dimension 7c: Architectural Tier Compliance — PASS
The Architectural Responsibility Map from RESEARCH.md is honored:
- CSRF + requireFilesWrite: webserver tier. Daemon socket gets NO Origin check (loopback-trust). Plans 124-01 and 124-03 both specify this explicitly.
- Per-session write state: API (daemon engine). No global flag mirroring FilesRead.
- GUI toggles: browser/client (React). Server computes homeDir, client renders it.
- Remote proxy: API (daemon proxy). Cap anti-smuggling preserved (124-03 T-124-10).
No tier mismatches found.

### Dimension 8: Nyquist Compliance — PASS WITH WARNING

Check 8e: VALIDATION.md EXISTS. Proceed.

| Task | Plan | Wave | Automated Command | Status |
|------|------|------|-------------------|--------|
| Task 0 (Wave 0 failing tests) | 124-01 | 1 | `go test -run 'TestRequireFilesWrite\|...' ... grep RED-OK` | Present |
| Task 1 (middleware) | 124-01 | 1 | `go build ./... && go test -run '...' . tail -3` | Present |
| Task 2 (route mounts) | 124-01 | 1 | `go test -run 'TestRequireFilesWrite' . tail -3` | Present |
| Task 0 (Wave 0 migration test) | 124-02 | 1 | `go test -run 'TestSettingsMigration_FilesWrite' ... grep RED-OK` | Present |
| Task 1 (schema+engine) | 124-02 | 1 | `go build ./... && go test -run '...' . tail -3` | Present |
| Task 2 (cap mint) | 124-02 | 1 | `go build ./... && go test -run '...' . tail -3` | Present |
| Task 0 (Wave 0 proxy test) | 124-03 | 2 | `go test -run 'TestRemoteFilesWrite' ... grep RED-OK` | Present |
| Task 1 (body fix) | 124-03 | 2 | `go build ./... && go test -run '...' . tail -4` | Present |
| Task 2 (route registration) | 124-03 | 2 | `go build ./... && go test -race . tail -3` | Present |
| Task 1 (binding chain) | 124-04 | 3 | `go build ./... && go test -race ./internal/daemon/ tail -3` | Present |
| Task 2 (warning component) | 124-04 | 3 | `pnpm test -- --run HomeDirWriteWarning tail -5` | Present |
| Task 3 (toggles) | 124-04 | 3 | `pnpm test -- --run SessionSharePanel DaemonManagerPanel tail -6` | Present |
| Task 0 (Wave 0 TUI test) | 124-05 | 2 | `go test -run 'HomeDirWarning\|...' ... grep RED-OK` | Present |
| Task 1 (TUI warning line) | 124-05 | 2 | `go test -run 'HomeDirWarning\|...' . tail -4` | Present |

Check 8a: All 14 tasks have `<automated>` commands. No MISSING references.
Check 8b: No watch-mode flags. Max latency ~90s.
Check 8c: Every consecutive window of 3 tasks per wave has all 3 automated. Continuity satisfied.
Check 8d: No Wave 0 MISSING references.

See WARNING-2: VALIDATION.md per-task table is a placeholder ("planner fills"), not populated. The `nyquist_compliant: false` flag in VALIDATION.md frontmatter is self-declared. Functional test commands are present in the plan files, so execution will verify correctly — but the VALIDATION.md artifact is incomplete.

### Dimension 9: Cross-Plan Data Contracts — PASS
The only shared data path is `capability.Claims.Perms` string mutation. Plan 124-01 adds `PermFilesWrite` to the constant vocab; plan 124-02 appends it to ownerPerms via string concatenation. Both plans explicitly note Claims struct field order is HMAC-load-bearing and must not be reordered. No incompatible transforms on shared data.

### Dimension 10: CLAUDE.md Compliance — PASS
- Go: `go fmt`/`golangci-lint`/context-aware verified in plan `<verification>` blocks.
- No new external packages (CLAUDE.md pnpm preferred; plans add zero packages).
- No `strings.Contains` for capability check (static-grep gate enforces).
- Let-it-crash: originAllowedForWrite fails closed on empty BaseURL + present Origin.
- TypeScript/React: BEM CSS reuse, no new CSS files, PascalCase components.
- Wails build note preserved in RESEARCH.md runtime state section.

### Dimension 11: Research Resolution — BLOCKER

`## Open Questions` section in RESEARCH.md lacks the `(RESOLVED)` suffix. Neither listed question has an inline `RESOLVED` marker:

1. Per-session write opt-in persistence model — UNRESOLVED in RESEARCH.md (resolved implicitly in 124-02 plan actions).
2. homeDir endpoint choice — UNRESOLVED in RESEARCH.md (resolved implicitly in 124-02 plan actions).

Both questions ARE answered in the plan task actions. This is a documentation formality gap only.

**Fix:** Add `(RESOLVED)` to the section heading and inline resolution markers to both questions.

### Dimension 12: Pattern Compliance — PASS
Every file in the PATTERNS.md File Classification table has a covering plan that references its analog. All three Critical Inversions are encoded correctly (not verbatim-copied). Shared patterns (HasPerm, claims-append-only, EvalSymlinks, settings defaults-merge, toggle-row BEM CSS) are referenced in all applicable plans.

---

## BLOCKER

### BLOCKER-1: [research_resolution] RESEARCH.md Open Questions not formally marked resolved

- **Plan:** 124-RESEARCH.md (phase-level)
- **Dimension:** research_resolution
- **Severity:** BLOCKER
- **Description:** `## Open Questions` section in RESEARCH.md has no `(RESOLVED)` suffix, and neither question has an inline `RESOLVED` marker. Per Dimension 11, this is a hard fail regardless of whether the plan actions implicitly resolve the questions.
- **Questions unresolved in RESEARCH.md:**
  1. Per-session write opt-in persistence model (resolved in 124-02 Task 1 action: per-session in-memory map + FilesWrite bool settings default).
  2. homeDir endpoint choice (resolved in 124-02 Task 2 action: IssueCapabilitiesResponse).
- **Fix:** In RESEARCH.md, rename `## Open Questions` to `## Open Questions (RESOLVED)` and add inline resolution text to each question, e.g.:
  - Q1: "RESOLVED: Model `FilesWrite bool` as persisted settings default (false) + per-session in-memory map. 124-02 Task 1."
  - Q2: "RESOLVED: homeDir rides `IssueCapabilitiesResponse` (existing endpoint, least new surface). 124-02 Task 2."

---

## WARNINGS

### WARNING-1: [dependency_correctness] 124-02 references `capability.PermFilesWrite` (added by 124-01) but declares `depends_on: []`

- **Plans:** 124-01, 124-02
- **Dimension:** dependency_correctness
- **Severity:** WARNING
- **Description:** Plan 124-02 Task 2 action includes `ownerPerms += "," + capability.PermFilesWrite`, a hard Go reference to the constant that 124-01 adds to `internal/capability/capability.go`. Both plans are wave 1 with `depends_on: []`. True parallel execution would cause 124-02's `go build ./...` verify in Task 2 to fail with "undefined: capability.PermFilesWrite" before 124-01 has run. Sequential single-threaded execution (the typical GSD executor behavior) works correctly.
- **Fix:** Either add `depends_on: ["124-01"]` to 124-02 (serializes the wave), or acknowledge in 124-02 Task 2's `<action>` that the constant must already exist (ensure 124-01 runs first). The `depends_on` change is cleaner.

### WARNING-2: [verification_derivation] VALIDATION.md per-task verification table is unfilled

- **Plan:** 124-VALIDATION.md (phase-level)
- **Dimension:** verification_derivation (Dimension 8)
- **Severity:** WARNING
- **Description:** `VALIDATION.md` frontmatter declares `nyquist_compliant: false` and `wave_0_complete: false`. The per-task verification map contains only a placeholder row `| (planner fills) | ...`. The actual test commands ARE present in the `<automated>` blocks of each plan task — so execution will proceed correctly — but the VALIDATION.md artifact is incomplete. The `nyquist_compliant: false` self-declaration means the planner has not signed off on Nyquist compliance.
- **Fix:** Populate the per-task verification map in VALIDATION.md with the 14 tasks across the 5 plans, their automated commands, and set `nyquist_compliant: true` once the table is verified against Dimension 8 checks 8a–8d.

### WARNING-3: [key_links_planned] Wails TS binding regeneration for `SetSessionFilesWrite` and `IssueCapabilitiesResponse.homeDir` not explicitly tasked

- **Plan:** 124-02, 124-04
- **Dimension:** key_links_planned
- **Severity:** WARNING
- **Description:** Plan 124-02 Task 2 adds `HomeDir bool json:"homeDir"` to `IssueCapabilitiesResponse` in Go, and plan 124-04 Task 1 adds `func (a *App) SetSessionFilesWrite(...)` to `app.go`. Both additions require the Wails TS binding (`frontend/src/wailsjs/go/main/App.d.ts` and `App.js`) to be regenerated so `DaemonManagerPanel.tsx` can use `response.homeDir` and `SetSessionFilesWrite(...)` with type safety. Plan 124-02 Task 2 says "Regenerate the Wails TS binding if the project's build does so automatically; otherwise note..." — this is conditional and not a guaranteed step. If the binding is not regenerated before 124-04 Task 3 runs, the frontend tests may pass (they mock Wails calls) but `wails dev` or a production build will fail with missing TS types.
- **Fix:** In 124-04 Task 1 `<acceptance_criteria>`, add: "`grep -c 'SetSessionFilesWrite' frontend/src/wailsjs/go/main/App.d.ts` is at least 1 (binding regenerated)." Alternatively, add an explicit `wails generate` step or call `wails dev` as part of the verify command. If Wails auto-regenerates on `go build`, document that assumption in the task.

---

## Structured Issues (YAML)

```yaml
issues:
  - plan: "124-RESEARCH.md"
    dimension: "research_resolution"
    severity: "blocker"
    description: "## Open Questions section in RESEARCH.md has no (RESOLVED) suffix and no inline RESOLVED markers for either listed question. Plans resolve both questions implicitly in task actions but RESEARCH.md is not formally marked."
    fix_hint: "Rename section to '## Open Questions (RESOLVED)' and add inline RESOLVED markers to Q1 and Q2 with the chosen approach."

  - plan: "124-02"
    dimension: "dependency_correctness"
    severity: "warning"
    description: "124-02 Task 2 references capability.PermFilesWrite (added by 124-01 Task 1) but 124-02 declares depends_on: []. True parallel wave-1 execution would cause 124-02 Task 2 go build to fail."
    fix_hint: "Add depends_on: ['124-01'] to 124-02 frontmatter, or add a note to 124-02 Task 2 that PermFilesWrite must exist before running."

  - plan: "124-VALIDATION.md"
    dimension: "verification_derivation"
    severity: "warning"
    description: "VALIDATION.md per-task verification table is a placeholder. nyquist_compliant: false in frontmatter. Functional test commands ARE in plan tasks, so execution works, but the artifact is incomplete."
    fix_hint: "Populate the per-task verification table from the 14 <automated> commands across the 5 plans; set nyquist_compliant: true."

  - plan: "124-04"
    dimension: "key_links_planned"
    severity: "warning"
    task: 1
    description: "Wails TS binding regeneration for SetSessionFilesWrite and IssueCapabilitiesResponse.homeDir is not explicitly tasked. 124-02 Task 2 makes it conditional ('if automatic'). If binding not regenerated, frontend imports from wailsjs will lack type definitions."
    fix_hint: "Add acceptance criterion in 124-04 Task 1: grep SetSessionFilesWrite in App.d.ts must return >= 1. Or add explicit wails generate step."
```

---

## Additional Verification Notes

**Three inversions: CORRECTLY ENCODED**
- Inversion 1 (Origin vacuous-pass): 124-01 Task 1 action explicitly says `return true when r.Header.Get("Origin") == ""` and documents the `requireAllowedOrigin` anti-pattern to avoid. Test in Task 0 asserts absent Origin passes.
- Inversion 2 (per-session not global): 124-02 Task 1 action explicitly says `DO NOT mirror this shape for write (Inversion 2)` and specifies `map[string]bool` under `e.mu`. Acceptance criteria explicitly forbid `e.filesWrite *bool` global field.
- Inversion 3 (r.Body not nil): 124-03 Task 1 action fixes `nil` → `r.Body` for PUT/POST/PATCH and forwards Content-Type. Test in Task 0 asserts body byte-equality.

**Phase 123 frozen engine: RESPECTED**
No plan's `files_modified` list includes any file under `internal/files/`. The anti-pattern "Touching internal/tui/files_client.go" is not violated (no plan lists that file).

**Cross-surface parity: PLANNED IN BOTH SURFACES**
- GUI: 124-04 Task 2 (HomeDirWriteWarning) + Task 3 (mounts in DaemonManagerPanel when homeDir true && filesWrite).
- TUI: 124-05 Task 1 (renderFilesTab, `⚠ Warning:` line under HomeDir && FilesWrite).
- Both use the same server-computed signal (`SessionInfo.HomeDir && SessionInfo.FilesWrite` from ListSessions populated by `sessionCwdIsHome` in 124-02 Task 1).
- Test in 124-05 Task 0 asserts positive and negative cases.

**Colorblind safety: VERIFIED AT SOURCE LEVEL**
- GUI: 124-04 `<verification>` includes `grep -F '⚠' frontend/src/components/HomeDirWriteWarning.tsx` and `grep -F 'Warning:' ...` both required to match.
- TUI: 124-05 Task 1 `<acceptance_criteria>` includes `grep -F 'Warning: cwd is $HOME' internal/tui/files.go` and `grep -F '⚠' internal/tui/files.go`.
- Color (#f59e0b amber) is decoration only; glyph+text carry the meaning.

**macOS EvalSymlinks: PLANNED**
124-02 Task 1 explicitly invokes `filepath.EvalSymlinks(home)` before comparison, mirroring sandbox.go:96-100. The Pitfall 4 macOS /var vs /private/var bug is acknowledged and mitigated.

**ASVS coverage: COMPLETE**
- V2 Authentication: HMAC cap tokens unchanged. Old tokens fail-safe to 403 (no files.write bit).
- V3 Session Management: per-session write map (not global). Grant-list revocation unchanged.
- V4 Access Control: requireFilesWrite + HasPerm + 403 ordering (requireCapability→HasPerm→Origin).
- V5 Input Validation: Phase 123 sandbox/denylist untouched; proxyRemoteFiles only fixes body forwarding.
- V6 Cryptography: Claims struct field order preserved; no hand-rolled signing.
- CSRF: originAllowedForWrite correctly inverts requireAllowedOrigin's absent-Origin behavior.

---

## Recommendation

**Fix the BLOCKER before execution**: Add `(RESOLVED)` to the `## Open Questions` section in `124-RESEARCH.md` and add inline resolution text to both questions. This is a ~2-line edit.

Address WARNING-1 (dependency) by adding `depends_on: ["124-01"]` to 124-02 frontmatter, or document that the GSD executor runs wave-1 plans sequentially.

Address WARNING-2 (VALIDATION.md table) and WARNING-3 (Wails binding) during plan revision.

Once BLOCKER-1 is resolved, the plans are ready for execution.
