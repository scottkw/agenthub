---
phase: 90-release-pipeline-hardening
plan: 01
type: execute
wave: 0
depends_on: []
files_modified:
  - scripts/grep-gate.sh
  - tests/build-script.test.sh
  - .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md
autonomous: true
requirements: [SEC-09, SEC-10]
tags: [ci, hardening, scaffolding, wave-0]

must_haves:
  truths:
    - "A grep-gate script exists that fails CI if any workflow contains @main, @master, @latest, or non-40-char-SHA action refs"
    - "The build.sh test suite has assertions that fail if @latest is present in build.sh or if the new WAILS_PINNED_VER pattern is absent"
    - "A documented manual step exists for pre-creating the release-90-test branch in scottkw/homebrew-agenthub before the E2E verification runs"
  artifacts:
    - path: "scripts/grep-gate.sh"
      provides: "SHA-pin regression guard (SEC-09 + SEC-10)"
      must_exist: true
    - path: "tests/build-script.test.sh"
      provides: "Section 12 — SEC-10 compliance assertions"
      contains: "@latest"
    - path: ".planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md"
      provides: "Documented manual prerequisite for E2E verification (D-16)"
      must_exist: true
  key_links:
    - from: "scripts/grep-gate.sh"
      to: ".github/workflows/*.yml + build.sh"
      via: "grep -rE pattern matching against workflow files and build.sh"
      pattern: "uses:\\s*[^#]*@(main|master|v[0-9]+|[a-z]+)$"
    - from: "tests/build-script.test.sh Section 12"
      to: "build.sh"
      via: "grep -c '@latest' assertion and grep -F WAILS_PINNED_VER assertion"
      pattern: "@latest|WAILS_PINNED_VER"
---

<objective>
Wave 0 scaffolding: create the test infrastructure that subsequent waves will satisfy. Produces the grep-gate script (SEC-09 + SEC-10 regression guard), extends the build.sh test suite to assert the new install pattern, and documents the Homebrew tap branch prerequisite for the Wave 5 E2E verification (D-16).

Purpose: The grep gate is the automated acceptance check for SEC-09 and SEC-10 across every subsequent wave. Without it landing first, no task in Waves 1-4 has an automated verify for SHA-pinning or @latest absence. The test suite extension ensures Wave 2 build.sh changes are guarded by an automated test. The tap-branch doc unblocks Wave 5 (rc verification cannot run until that branch exists).

Output: One new script, one file with new test sections, one setup-instruction doc. Zero changes to workflow YAMLs, `build.sh`, or `go.mod` — those are owned by downstream plans.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md

@.planning/phases/90-release-pipeline-hardening/90-CONTEXT.md
@.planning/phases/90-release-pipeline-hardening/90-RESEARCH.md
@.planning/phases/90-release-pipeline-hardening/90-PATTERNS.md
@.planning/phases/90-release-pipeline-hardening/90-VALIDATION.md

@./CLAUDE.md

<interfaces>
<!-- Existing test helpers in tests/build-script.test.sh (lines 63-84) — reuse verbatim -->

```bash
# From tests/build-script.test.sh lines 63-73:
assert_file_contains_literal() {
  local name="$1"
  local pattern="$2"
  local file="$3"
  if grep -qF -- "$pattern" "$file"; then
    pass "$name"
  else
    fail "$name" "literal pattern not found in $file: $pattern"
  fi
}

# From tests/build-script.test.sh lines 75-84:
assert_file_contains_regex() {
  local name="$1"
  local pattern="$2"
  local file="$3"
  if grep -qE "$pattern" "$file"; then
    pass "$name"
  else
    fail "$name" "regex pattern not found in $file: $pattern"
  fi
}

# PASS=0; FAIL=0 counters at top of file (lines 12-13)
# Summary footer at lines 294-303
```

Grep-gate regex set (verbatim from 90-RESEARCH.md Example 5, lines 684-719):
```bash
# Any uses: line with @main, @master, @vMajor, or plain-word ref — reject
BAD=$(grep -rEn 'uses:\s*[^#]*@(main|master|v[0-9]+|[a-z]+)$' .github/workflows/ || true)

# Any @latest in workflows, build.sh, tests/ — reject
LATEST=$(grep -rEn '@latest' .github/workflows/ build.sh tests/ || true)

# Negation check — every uses: line must match 40-char SHA
NON_SHA=$(grep -rE 'uses:\s*[^ ]+@' .github/workflows/ \
  | grep -Ev 'uses:\s*[^ ]+@[a-f0-9]{40}(\s|$)' || true)
```

Tap branch creation sequence (from 90-RESEARCH.md Environment Availability, line 787):
```bash
gh repo clone scottkw/homebrew-agenthub
cd homebrew-agenthub
git checkout -b release-90-test
git push -u origin release-90-test
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create scripts/grep-gate.sh — SEC-09/SEC-10 regression guard</name>
  <read_first>
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (lines 684-719 — Example 5 grep-gate logic)
    - .planning/phases/90-release-pipeline-hardening/90-CONTEXT.md (D-09 + Claude's Discretion note on location)
    - .github/workflows/ (all four workflow files — need to understand the current baseline that the gate will flag as BAD until Wave 2-4 lands)
  </read_first>
  <files>scripts/grep-gate.sh</files>
  <action>
    Create `scripts/grep-gate.sh` with `#!/usr/bin/env bash` shebang and `set -euo pipefail`.

    Embed the three grep checks verbatim from 90-RESEARCH.md Example 5 (this file, `<interfaces>` block):

    1. **Floating-ref check** — fails if any `uses:` line ends with `@main`, `@master`, `@v<num>`, or `@<word>`:
       ```bash
       BAD=$(grep -rEn 'uses:\s*[^#]*@(main|master|v[0-9]+|[a-z]+)$' .github/workflows/ || true)
       if [[ -n "$BAD" ]]; then
         echo "FAIL: unpinned action refs found:"; echo "$BAD"; exit 1
       fi
       ```

    2. **`@latest` check** — fails if `@latest` appears anywhere in workflows, `build.sh`, or `tests/`:
       ```bash
       LATEST=$(grep -rEn '@latest' .github/workflows/ build.sh tests/ || true)
       if [[ -n "$LATEST" ]]; then
         echo "FAIL: @latest references found:"; echo "$LATEST"; exit 1
       fi
       ```

    3. **Non-SHA action ref check** — every `uses: owner/repo@X` line must match 40-char hex:
       ```bash
       NON_SHA=$(grep -rE 'uses:\s*[^ ]+@' .github/workflows/ \
         | grep -Ev 'uses:\s*[^ ]+@[a-f0-9]{40}(\s|$)' || true)
       if [[ -n "$NON_SHA" ]]; then
         echo "FAIL: non-SHA action refs (likely @v4 tags):"; echo "$NON_SHA"; exit 1
       fi
       ```

    End with `echo "PASS: all action refs are SHA-pinned"` on success.

    Make the script executable: `chmod +x scripts/grep-gate.sh`.

    **Path choice rationale (honors Claude's Discretion per D-09):** `scripts/grep-gate.sh` keeps the script out of `.github/workflows/` (the thing it's auditing), aligns with the path 90-VALIDATION.md line 22 already names (`scripts/grep-gate.sh`), and makes local invocation obvious. The wiring into a CI workflow is a separate task in Plan 03 — this plan only delivers the script.

    **Expected behavior TODAY (before Waves 2-4 land):** The script MUST fail on the current repo — every workflow file still has `@v4`/`@v5`/`@main` refs. Do NOT silence this; the script's job is to fail until pinning lands. It will start passing as Plan 03/04/05 commits land.

    **Document this expected-failure behavior** inline at the top of the script as a comment:
    ```bash
    # grep-gate.sh — Phase 90 SEC-09 + SEC-10 regression guard
    #
    # Asserts zero floating-ref, @latest, or non-SHA action references in
    # .github/workflows/, build.sh, and tests/.
    #
    # EXPECTED to FAIL during Phase 90 Waves 1-4; passes once all SHA-pin
    # work lands (end of Wave 4). Becomes part of CI after the hardening-check
    # workflow step is added (Plan 03 or Plan 04).
    ```
  </action>
  <verify>
    <automated>test -x scripts/grep-gate.sh && bash -n scripts/grep-gate.sh && grep -c 'grep -rE' scripts/grep-gate.sh | (read n; [ "$n" -ge 2 ] && echo "PASS: script has >=2 grep-rE checks" || (echo "FAIL: expected >=2 grep-rE checks, found $n"; exit 1))</automated>
  </verify>
  <acceptance_criteria>
    - `test -x scripts/grep-gate.sh` exits 0 (file exists and is executable)
    - `bash -n scripts/grep-gate.sh` exits 0 (syntax valid)
    - `grep -c '#!/usr/bin/env bash' scripts/grep-gate.sh` returns 1
    - `grep -c 'set -euo pipefail' scripts/grep-gate.sh` returns 1
    - `grep -F 'uses:\s*[^#]*@(main|master|v[0-9]+|[a-z]+)$' scripts/grep-gate.sh` exits 0 (floating-ref regex present)
    - `grep -F '@latest' scripts/grep-gate.sh` matches at least once (the `@latest` literal used by the check)
    - `grep -F 'uses:\s*[^ ]+@[a-f0-9]{40}' scripts/grep-gate.sh` exits 0 (40-char SHA negative regex present)
    - `bash scripts/grep-gate.sh` exits NON-ZERO against the current repo (every workflow still floating refs — this is the expected pre-Wave-4 state)
  </acceptance_criteria>
  <done>
    Script exists, is executable, has valid bash syntax, contains all three grep checks, and correctly fails against the current unpinned repo state. Commit message: `ci(90): add scripts/grep-gate.sh — SEC-09/SEC-10 regression guard (Phase 90 Wave 0)`.
  </done>
</task>

<task type="auto">
  <name>Task 2: Extend tests/build-script.test.sh with Section 12 — SEC-10 compliance + tap-branch setup doc</name>
  <read_first>
    - tests/build-script.test.sh (FULL FILE — need the existing helper signatures, section numbering convention, and PASS/FAIL accounting footer)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Example 3, lines 619-647 — the target patterns that Plan 03 will make build.sh contain)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 504-526 — the exact Section 12 assertion block this task must produce)
    - build.sh (lines 61-67 — the CURRENT @latest pattern that Plan 03 will replace)
  </read_first>
  <files>tests/build-script.test.sh, .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md</files>
  <action>
    **Part A — Extend `tests/build-script.test.sh`:** Append a new "Section 12: SEC-10 compliance — no @latest refs" block AFTER Section 11 (line 290) and BEFORE the `# Summary` block (line 292). Use the verbatim pattern from 90-PATTERNS.md lines 504-526:

    ```bash
    # ---------------------------------------------------------------------------
    # Section 12: SEC-10 compliance — build.sh install pattern
    # ---------------------------------------------------------------------------
    echo ""
    echo "=== Section 12: SEC-10 compliance — no @latest refs ==="

    # Negation: @latest must be ABSENT from build.sh
    output=$(grep -c '@latest' "$BUILD_SH" || true)
    if [[ "$output" -eq 0 ]]; then
      pass "build.sh contains no @latest references (SEC-10)"
    else
      fail "build.sh contains @latest — SEC-10 violation" "matches: $output"
    fi

    # Positive: new go list -m pin pattern present
    assert_file_contains_literal "build.sh contains go list -m pin pattern" \
      "go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2" "$BUILD_SH"

    # Positive: WAILS_PINNED_VER sanity gate present (Pitfall 5 defense)
    assert_file_contains_literal "build.sh gates on WAILS_PINNED_VER" \
      "WAILS_PINNED_VER" "$BUILD_SH"
    ```

    **CRITICAL sequencing note** — ALL THREE new assertions in Section 12 will FAIL today, because `build.sh:65` still reads `go install github.com/wailsapp/wails/v2/cmd/wails@latest`. Plan 03 Task 3 is the one that makes them pass. This is the intended Wave 0 contract: Wave 0 creates the red tests; later waves turn them green.

    Preserve the existing Summary footer (lines 292-303) and PASS/FAIL counter usage — new assertions use the same `pass`/`fail` functions and thus roll into the footer naturally. Do NOT renumber existing sections.

    **Part B — Create `.planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md`:** Document the D-16 prerequisite. The Wave 5 E2E verification requires `scottkw/homebrew-agenthub` to have a `release-90-test` branch. Write a short doc with:

    ```markdown
    # Phase 90 — Homebrew Tap Test Branch Setup (D-16)

    **Prerequisite for:** Plan 06 (E2E rc verification)
    **Required before:** Pushing `v3.1.0-rc1` tag

    ## Why

    D-16 (90-CONTEXT.md): the `distribute.yml` tap step runs against a `release-90-test` branch
    of `scottkw/homebrew-agenthub` when the tag matches `v*-rc*`. This prevents test artifacts
    from appearing to tap users during rc verification.

    ## One-time setup

    Run these commands locally (or from any machine with push access to
    `scottkw/homebrew-agenthub`):

    ```bash
    # Clone tap repo (skip if already cloned)
    gh repo clone scottkw/homebrew-agenthub /tmp/homebrew-agenthub
    cd /tmp/homebrew-agenthub

    # Create branch from current main
    git fetch origin main
    git checkout -b release-90-test origin/main

    # Push so distribute.yml can check it out during rc flow
    git push -u origin release-90-test
    ```

    ## Verification

    ```bash
    gh api /repos/scottkw/homebrew-agenthub/branches/release-90-test --jq .name
    # Expected output: release-90-test
    ```

    ## Teardown (after v3.1 ships)

    Once the real (non-rc) `v3.1.0` tag ships and the tap updates cleanly against `main`,
    the test branch can be deleted:

    ```bash
    git push origin --delete release-90-test
    ```

    But NOT before the real release — it's the target of the rc distribute.yml run.
    ```

    **This doc is prose only** — no code changes triggered. It's a human-facing runbook
    entry that Plan 06's E2E task will reference before pushing the rc tag.
  </action>
  <verify>
    <automated>bash tests/build-script.test.sh 2>&1 | grep -F "Section 12:" && test -f .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md && grep -F "release-90-test" .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md && echo "PASS: Section 12 wired + tap-branch doc present"</automated>
  </verify>
  <acceptance_criteria>
    - `grep -c '=== Section 12:' tests/build-script.test.sh` returns 1 (exactly one new section header)
    - `grep -F 'go list -m -f' tests/build-script.test.sh` matches at least once (the positive assertion)
    - `grep -F 'WAILS_PINNED_VER' tests/build-script.test.sh` matches at least once
    - `grep -F 'no @latest references' tests/build-script.test.sh` matches at least once
    - `bash tests/build-script.test.sh` runs to completion (no bash syntax error) — it IS expected to report 3+ FAILs at this Wave-0 point (those failures are the contract for Plan 03 to satisfy)
    - The pre-existing `PASS=0; FAIL=0` counters at line 12-13 are unmodified
    - The pre-existing `Results: $PASS passed, $FAIL failed` footer at line 297 is unmodified
    - `.planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md` exists
    - `grep -F 'release-90-test' .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md` matches at least once
    - `grep -F 'gh repo clone scottkw/homebrew-agenthub' .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md` matches at least once
  </acceptance_criteria>
  <done>
    Section 12 block appended to test file; new assertions reference the exact patterns (`go list -m -f '{{.Version}}'`, `WAILS_PINNED_VER`, `@latest`-absent check) that Plan 03 must satisfy. Tap-branch setup doc exists and documents the exact command sequence. Commit message: `test(90): scaffold SEC-10 assertions + D-16 tap branch runbook (Phase 90 Wave 0)`.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| CI check → main branch | The grep-gate script is the policy boundary between "arbitrary commit" and "mergeable to main." Any commit that fails the gate must be rejected before merge. |
| Phase 90 wave boundary | Wave 0 deliverables (this plan) define the acceptance contract (red tests) that Waves 1-4 must turn green. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-90-01 | Tampering | `scripts/grep-gate.sh` itself | mitigate | Script is committed to main; any modification goes through PR review. The grep-gate workflow step (Plan 03 Task 1) runs the script from the repo checkout — a hypothetical attacker who modifies the script must also bypass PR review. |
| T-90-02 | Spoofing | Regex false-negative (script says PASS but bad ref slipped) | mitigate | Three independent checks (floating-ref, `@latest`, non-SHA) give defense in depth. Assumption A4 (90-RESEARCH.md) acknowledges edge cases exist; the `grep -Ev '[a-f0-9]{40}'` negative check is the authoritative one — if someone writes `uses: foo/bar@deadbeef` (only 8 hex chars), the non-SHA check still catches it because 8 ≠ 40. |
| T-90-03 | Elevation of Privilege | Test-file modification grants merge-to-main | accept | Tests are not security-critical — they're acceptance contracts. A developer who modifies `tests/build-script.test.sh` to delete Section 12 bypasses Phase 90's automated assertion, but the grep-gate script (external to the test file) still runs independently as the authoritative policy check. Redundancy by design. |
| T-90-04 | Information Disclosure | Tap-branch doc reveals scottkw/homebrew-agenthub structure | accept | The tap repo is already public; its branch list is `gh api` queryable by anyone. No new information disclosed. |

**Residual risk:** Low. Wave 0 is a pure-test deliverable — no production code, no secrets, no runtime behavior changes. The only "failure mode" is false confidence if the grep-gate regexes are incomplete — mitigated by running the gate against the current red state (which it correctly detects).
</threat_model>

<verification>
After both tasks land:
1. `test -x scripts/grep-gate.sh && bash -n scripts/grep-gate.sh` — script exists, syntax valid
2. `bash scripts/grep-gate.sh; echo exit=$?` — MUST exit non-zero (current repo is unpinned)
3. `bash tests/build-script.test.sh; echo exit=$?` — MUST exit non-zero (Section 12 red) but MUST print "Section 12" header (no syntax error)
4. `ls .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md` — file exists
</verification>

<success_criteria>
- Wave 0 delivers three red tests (grep-gate exits non-zero; 3 Section 12 assertions fail) that subsequent waves must turn green
- Zero changes to `.github/workflows/*.yml`, `build.sh`, `go.mod`, `go.sum`
- Zero new dependencies added
- 90-VALIDATION.md `## Wave 0 Requirements` checkboxes 1 and 2 can be checked (item 3 — `release-90-test` branch creation — is deferred to the user as a manual step per the new runbook doc; the doc itself IS delivered)
</success_criteria>

<output>
After completion, create `.planning/phases/90-release-pipeline-hardening/90-01-SUMMARY.md` documenting:
- Files created: `scripts/grep-gate.sh`, `.planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md`
- Files modified: `tests/build-script.test.sh` (+Section 12)
- Current gate state: fails (expected until Wave 4 completes)
- Handoff to next plan: Plan 02 can proceed immediately (files_modified has no overlap)
</output>
