---
phase: 90-release-pipeline-hardening
plan: 03
type: execute
wave: 2
depends_on: [02]
files_modified:
  - .github/workflows/build.yml
  - .github/workflows/release-please.yml
  - build.sh
autonomous: true
requirements: [SEC-09, SEC-10]
tags: [ci, hardening, sha-pin, build-yml, release-please, build-sh, wave-2]

must_haves:
  truths:
    - ".github/workflows/build.yml has zero @v<N>, @main, @master, or @latest references — every uses: line is pinned to a 40-char commit SHA with a trailing # vX.Y.Z comment"
    - ".github/workflows/release-please.yml uses googleapis/release-please-action pinned to SHA 45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0"
    - "build.sh no longer references 'go install github.com/wailsapp/wails/v2/cmd/wails@latest'; the install hint uses the go list -m -f '{{.Version}}' pattern with a WAILS_PINNED_VER sanity gate (Pitfall 5 defense)"
    - "build.yml installs wails via the same go list -m pattern (not @latest)"
    - "tests/build-script.test.sh Section 12 (from Plan 01) now passes all three assertions"
  artifacts:
    - path: ".github/workflows/build.yml"
      provides: "SHA-pinned CI build matrix with tools.go-derived wails install"
      contains: "@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a"
    - path: ".github/workflows/release-please.yml"
      provides: "SHA-pinned release-please-action"
      contains: "45996ed1f6d02564a971a2fa1b5860e934307cf7"
    - path: "build.sh"
      provides: "Local build script with go-list-derived wails install hint + WAILS_PINNED_VER gate"
      contains: "WAILS_PINNED_VER"
  key_links:
    - from: "build.yml wails install step"
      to: "go.mod via go list -m"
      via: "shell substitution go install ...@$(go list -m -f '{{.Version}}' ...)"
      pattern: "go list -m -f '\\{\\{\\.Version\\}\\}' github.com/wailsapp/wails/v2"
    - from: "build.sh line 62-70 (new block)"
      to: "go.mod"
      via: "WAILS_PINNED_VER sanity gate — aborts if wails not in go.mod"
      pattern: "WAILS_PINNED_VER.*wails not pinned in go.mod"
---

<objective>
Plan 03 applies SEC-09 SHA-pinning to the two simpler workflows (build.yml, release-please.yml) and implements SEC-10's `@latest` removal in both `build.yml` and `build.sh`. Release.yml (large) and distribute.yml (wingetcreate swap) are owned by Plans 04 and 05 respectively — no file overlap with this plan.

Purpose:
- SEC-09: every `uses:` in build.yml + release-please.yml must be a 40-char SHA. 11 pin edits total.
- SEC-10: the `go install wails@latest` at `build.yml:80-81` and the install hint at `build.sh:65` must both use the go-list-derived pattern.
- Turns the Section 12 assertions (Plan 01 red tests) green.

Output: three files updated in place, no new files created. Grep-gate progresses from failing on all four workflows to failing only on release.yml (still) and distribute.yml (still).
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md

@.planning/phases/90-release-pipeline-hardening/90-CONTEXT.md
@.planning/phases/90-release-pipeline-hardening/90-RESEARCH.md
@.planning/phases/90-release-pipeline-hardening/90-PATTERNS.md

@.planning/phases/90-release-pipeline-hardening/90-02-SUMMARY.md

@./CLAUDE.md

<interfaces>
<!-- SHA-pin reference table — verbatim from 90-PATTERNS.md Shared Patterns (lines 568-580) -->
<!-- All SHAs live-verified on 2026-04-23 via gh api; DO NOT substitute other values -->

| Action | Pinned form (use exactly this string in every `uses:`) |
|--------|--------------------------------------------------------|
| `actions/checkout` | `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2` |
| `actions/setup-go` | `actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0` |
| `actions/setup-node` | `actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0` |
| `actions/upload-artifact` | `actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1` |
| `actions/download-artifact` | `actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1` |
| `pnpm/action-setup` | `pnpm/action-setup@903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3` |
| `googleapis/release-please-action` | `googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0` |

<!-- Wails install pattern — verbatim from 90-PATTERNS.md Shared Patterns lines 597-602 -->

```yaml
- name: Install Wails CLI (version from go.mod)
  run: |
    WAILS_VER=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
    [ -n "$WAILS_VER" ] || { echo "wails not pinned in go.mod"; exit 1; }
    go install github.com/wailsapp/wails/v2/cmd/wails@"$WAILS_VER"
```

<!-- build.sh replacement block — verbatim from 90-PATTERNS.md lines 452-468 / 90-RESEARCH.md Example 3 -->

```bash
# REPLACE build.sh:61-67 with:
WAILS="$(go env GOPATH)/bin/wails"
WAILS_PINNED_VER="$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2 2>/dev/null)"
if [[ -z "$WAILS_PINNED_VER" ]]; then
  echo "ERROR: wails not pinned in go.mod — Phase 90 tools.go setup missing"
  exit 1
fi
if [[ ! -x "$WAILS" ]]; then
  echo "ERROR: wails not found at $WAILS"
  echo "Install: go install github.com/wailsapp/wails/v2/cmd/wails@$WAILS_PINNED_VER"
  exit 1
fi
# Optional: verify installed version matches pinned version
if ! "$WAILS" version 2>/dev/null | grep -qF "$WAILS_PINNED_VER"; then
  echo "WARN: installed wails does not match pinned version ($WAILS_PINNED_VER)"
  echo "Reinstall: go install github.com/wailsapp/wails/v2/cmd/wails@$WAILS_PINNED_VER"
fi
```

<!-- Current build.yml uses: lines requiring SHA-pin (from live grep) -->

Lines to edit in build.yml:
- Line 34: `uses: actions/checkout@v4` → SHA-pinned v6.0.2
- Line 45: `uses: actions/setup-go@v5` → SHA-pinned v6.4.0
- Line 65: `uses: pnpm/action-setup@v4` → SHA-pinned v6.0.3
- Line 70: `uses: actions/setup-node@v4` → SHA-pinned v6.4.0
- Line 81 (content): `go install github.com/wailsapp/wails/v2/cmd/wails@latest` → go-list pattern
- Lines 106, 114, 122, 130: `uses: actions/upload-artifact@v4` → SHA-pinned v7.0.1 (four occurrences)

Total build.yml edits: 4 simple SHA swaps (checkout/setup-go/pnpm/setup-node) + 1 install-step replacement + 4 upload-artifact SHA swaps = 9 line/block changes.

Lines to edit in release-please.yml:
- Line 16: `uses: googleapis/release-please-action@v4` → SHA-pinned v5.0.0
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: SHA-pin all uses: lines in .github/workflows/build.yml + swap wails install to go-list pattern</name>
  <read_first>
    - .github/workflows/build.yml (FULL FILE — 137 lines — need all 8 uses: line contexts + the wails install step at lines 80-81)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 30-53 for build.yml edit list; lines 557-611 for shared SHA-pin + wails install patterns)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Core third-party GitHub Actions table lines 101-119 — authoritative SHA source; Pitfall 5 lines 514-526 — why the `[ -n "$WAILS_VER" ]` gate is mandatory)
  </read_first>
  <files>.github/workflows/build.yml</files>
  <action>
    Edit `.github/workflows/build.yml` in place:

    **SHA-pin edits (8 total):** Replace each `uses:` line per the authoritative SHA table above. DO NOT substitute different SHAs — those are the exact 40-char hashes verified via `gh api` on 2026-04-23. The trailing `# vX.Y.Z` comment is mandatory for Dependabot comment-sync (per D-06).

    Specific edits (with line numbers at read time — adjust if shifted by edits above them):
    - Line 34: `uses: actions/checkout@v4` → `uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2`
    - Line 45: `uses: actions/setup-go@v5` → `uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0`
    - Line 65: `uses: pnpm/action-setup@v4` → `uses: pnpm/action-setup@903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3`
    - Line 70: `uses: actions/setup-node@v4` → `uses: actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0`
    - Lines 106, 114, 122, 130 (four `actions/upload-artifact@v4` occurrences): each → `uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1`

    **Wails install replacement (lines 80-81):** Replace:
    ```yaml
    - name: Install Wails CLI
      run: go install github.com/wailsapp/wails/v2/cmd/wails@latest
    ```
    with (verbatim from `<interfaces>` pattern, D-11):
    ```yaml
    - name: Install Wails CLI (version from go.mod)
      run: |
        WAILS_VER=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
        [ -n "$WAILS_VER" ] || { echo "wails not pinned in go.mod"; exit 1; }
        go install github.com/wailsapp/wails/v2/cmd/wails@"$WAILS_VER"
    ```

    The `[ -n "$WAILS_VER" ]` check is mandatory per 90-RESEARCH.md Pitfall 5 (line 520): `go list -m` on a non-required module may print empty string under `set -e`, silently expanding to `go install wails@` which defaults to `@latest`. The explicit empty-string check converts that silent failure into an exit-1.

    **Upload-artifact v4 → v7 consideration (A1 acknowledged):** 90-RESEARCH.md Assumption A1 (line 743) flags that v4→v7 is reported backward-compatible for simple file uploads (.exe, .tar.gz) — which is exactly what build.yml uploads. The macOS upload at line 106 uploads `build/bin/agenthub.app` (directory) — under v4 and v7 both, this will zip-flatten per the well-known limitation, but build.yml is NOT the release pipeline (release.yml is) so this is not a correctness regression for build.yml's purpose. Sticking with v7 for the full hardening pass.

    **Matrix structure PRESERVED (lines 6-29):** Do not touch the matrix block. Only SHA-pin edits and the one install-step replacement.

    **`if:` conditionals PRESERVED (lines 39, 52, 61, 77, 91, 105, 113, 121, 129):** All conditional guards stay exactly as-is.

    **Do not add an inline grep-gate step here.** Plan 04 has the choice to add it either in release.yml validate job or a standalone hardening-check.yml. Decision deferred to Plan 04's executor.
  </action>
  <verify>
    <automated>grep -c '@latest' .github/workflows/build.yml | (read n; [ "$n" -eq 0 ] && echo "PASS: no @latest in build.yml" || (echo "FAIL: @latest still present ($n matches)"; exit 1)) && grep -c 'de0fac2e4500dabe0009e67214ff5f5447ce83dd' .github/workflows/build.yml | (read n; [ "$n" -eq 1 ] && echo "PASS: checkout v6.0.2 SHA pinned" || (echo "FAIL: checkout SHA count = $n (expected 1)"; exit 1)) && grep -c '043fb46d1a93c77aae656e7c1c64a875d1fc6a0a' .github/workflows/build.yml | (read n; [ "$n" -eq 4 ] && echo "PASS: upload-artifact v7 SHA pinned 4x" || (echo "FAIL: upload-artifact SHA count = $n (expected 4)"; exit 1)) && grep -F "go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2" .github/workflows/build.yml</automated>
  </verify>
  <acceptance_criteria>
    - `grep -cE '@(main|master|v[0-9]+)$' .github/workflows/build.yml` returns 0 (zero floating refs)
    - `grep -c '@latest' .github/workflows/build.yml` returns 0
    - `grep -c 'actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2' .github/workflows/build.yml` returns 1
    - `grep -c 'actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0' .github/workflows/build.yml` returns 1
    - `grep -c 'pnpm/action-setup@903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3' .github/workflows/build.yml` returns 1
    - `grep -c 'actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0' .github/workflows/build.yml` returns 1
    - `grep -c 'actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1' .github/workflows/build.yml` returns 4
    - `grep -F "go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2" .github/workflows/build.yml` returns >= 1 match
    - `grep -F '[ -n "$WAILS_VER" ] || { echo "wails not pinned in go.mod"; exit 1; }' .github/workflows/build.yml` returns >= 1 match (Pitfall 5 defense)
    - YAML still validates: `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/build.yml"))'` exits 0
    - Every `uses:` line is a 40-char hex SHA: `grep -E 'uses:\s*[^ ]+@' .github/workflows/build.yml | grep -Ev '@[a-f0-9]{40}(\s|$)'` returns no matches (exit code 1)
  </acceptance_criteria>
  <done>
    build.yml has zero floating refs, zero `@latest`, all uses: lines SHA-pinned with version comments, wails install uses go-list pattern with Pitfall-5 gate. Commit message: `ci(90): SHA-pin build.yml actions + tools.go-derived wails install (SEC-09 + SEC-10)`.
  </done>
</task>

<task type="auto">
  <name>Task 2: SHA-pin release-please.yml</name>
  <read_first>
    - .github/workflows/release-please.yml (FULL FILE — only 22 lines)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 307-319)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Core Actions table line 116)
  </read_first>
  <files>.github/workflows/release-please.yml</files>
  <action>
    Single-line edit at line 16:

    ```yaml
    # BEFORE
    - uses: googleapis/release-please-action@v4

    # AFTER
    - uses: googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0
    ```

    **SHA source:** `gh api repos/googleapis/release-please-action/tags` on 2026-04-23 returned `45996ed1f6d02564a971a2fa1b5860e934307cf7` for tag `v5.0.0`. This is the latest stable; current file uses floating `@v4`.

    **Version-bump consideration (Chesterton's Fence):** The current `@v4` pin is floating. A v4→v5 bump alongside the SHA-pin is a coherent change — the alternative (SHA-pin to the latest v4.x.x) means a second upgrade PR immediately afterward. 90-RESEARCH.md Core Actions table picks v5.0.0. If the executor has concerns about v5.0.0 compatibility with the existing `config-file: release-please-config.json` + `manifest-file: .release-please-manifest.json` args, those args are preserved (v5 did not rename those inputs; verified via releases notes; recommended course — honor v5.0.0 per research).

    **PRESERVE:** `token: ${{ secrets.RELEASE_PLEASE_TOKEN }}`, `config-file: release-please-config.json`, `manifest-file: .release-please-manifest.json`, and the `permissions:` + `secrets:` usage (lines 8-10, 18-21).

    No other changes needed — release-please.yml is a single-job single-step workflow.
  </action>
  <verify>
    <automated>grep -c 'googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0' .github/workflows/release-please.yml | (read n; [ "$n" -eq 1 ] && echo "PASS" || (echo "FAIL: expected 1 match, got $n"; exit 1)) && grep -cE '@(main|master|v[0-9]+)$' .github/workflows/release-please.yml | (read n; [ "$n" -eq 0 ] && echo "PASS: no floating refs" || (echo "FAIL: floating refs still present"; exit 1))</automated>
  </verify>
  <acceptance_criteria>
    - `grep -c 'googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0' .github/workflows/release-please.yml` returns 1
    - `grep -c 'googleapis/release-please-action@v4' .github/workflows/release-please.yml` returns 0
    - `grep -cE '@(main|master|v[0-9]+)$' .github/workflows/release-please.yml` returns 0
    - `grep -F 'secrets.RELEASE_PLEASE_TOKEN' .github/workflows/release-please.yml` matches (token ref preserved)
    - `grep -F 'config-file: release-please-config.json' .github/workflows/release-please.yml` matches (config preserved)
    - YAML valid: `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/release-please.yml"))'` exits 0
  </acceptance_criteria>
  <done>
    release-please.yml has a single SHA-pinned `uses:` line, all other config preserved. Commit message: `ci(90): SHA-pin release-please-action to v5.0.0 (SEC-09)`.
  </done>
</task>

<task type="auto">
  <name>Task 3: Update build.sh — @latest hint → go-list pattern + WAILS_PINNED_VER gate</name>
  <read_first>
    - build.sh (FULL FILE — 218 lines; focus lines 61-67 which are being replaced)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 437-472 — exact replacement block)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Example 3 lines 605-636 — full before/after, and Pitfall 5 lines 514-526 for the why)
    - tests/build-script.test.sh (lines 288-290 — current end of Section 11 — to confirm Section 12 assertions from Plan 01 are aligned)
  </read_first>
  <files>build.sh</files>
  <action>
    Replace lines 61-67 of `build.sh` (the "Wails binary check" block) with the verbatim block from the `<interfaces>` section.

    The new block:
    1. Reads the pinned wails version from `go.mod` into `WAILS_PINNED_VER` using the Pitfall-5-safe pattern (`2>/dev/null` captures any stderr warning; empty-string check below).
    2. Fails fast with an explanatory error if `WAILS_PINNED_VER` is empty — this surfaces a broken tools.go setup rather than silently falling back to `@latest`.
    3. Checks if `wails` binary exists at `$GOPATH/bin/wails`; if not, prints the CORRECT install command (using `$WAILS_PINNED_VER`, not `@latest`).
    4. As a best-effort sanity check, grep-matches the installed wails version against `$WAILS_PINNED_VER`; warns (not errors) on mismatch — this is informational, not blocking, because `wails version` output format may not include a literal `v` prefix on all versions.

    **PRESERVE (do not touch):**
    - Shebang `#!/usr/bin/env bash` (line 1)
    - `set -euo pipefail` (line 2)
    - Script header comment (lines 4-5)
    - Variable defaults `PLATFORM=""` + `SIGN="false"` (lines 7-8)
    - `usage()` function (lines 10-19)
    - Argument parsing loop (lines 21-45)
    - Empty-platform check (lines 47-49)
    - Valid-platform case statement (lines 51-59)
    - All `build_macos` / `build_windows` / `build_linux` / `sign_and_notarize` functions (lines 69-207)
    - Final dispatch case (lines 209-217)

    **Only lines 61-67 change.** The surrounding comment `# Wails binary check` can stay — the block below it changes.

    **After editing, verify the test suite passes all Section 12 assertions:**
    ```bash
    bash tests/build-script.test.sh 2>&1 | tail -20
    ```
    should show `FAIL: 0` in the footer. Plan 01's three red Section 12 assertions now turn green because:
    - `grep -c '@latest' build.sh` returns 0 (removed from install hint on old line 65)
    - `go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2` literal is present in the new block
    - `WAILS_PINNED_VER` literal is present in the new block

    **If Section 12 still shows failures:** verify the `@latest` literal isn't hiding elsewhere in `build.sh` (it shouldn't be — line 65 was the sole occurrence per research).
  </action>
  <verify>
    <automated>grep -c '@latest' build.sh | (read n; [ "$n" -eq 0 ] && echo "PASS: no @latest in build.sh" || (echo "FAIL: @latest present"; exit 1)) && grep -F "go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2" build.sh && grep -F 'WAILS_PINNED_VER' build.sh && bash -n build.sh && bash tests/build-script.test.sh 2>&1 | tail -3 | grep -E 'Results:.*0 failed'</automated>
  </verify>
  <acceptance_criteria>
    - `grep -c '@latest' build.sh` returns 0 (SEC-10 compliance)
    - `grep -F "go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2" build.sh` matches at least once
    - `grep -c 'WAILS_PINNED_VER' build.sh` returns >= 3 (variable assignment + empty check + error-message reference)
    - `grep -F 'ERROR: wails not pinned in go.mod — Phase 90 tools.go setup missing' build.sh` matches (the error message content is part of the Pitfall-5 defense)
    - `bash -n build.sh` exits 0 (syntax valid)
    - `bash build.sh` with no args still prints `Usage:` and exits non-zero (pre-existing Section 3 test must still pass)
    - `bash tests/build-script.test.sh` exits 0 (all assertions pass, including Plan 01's Section 12 red tests now green)
    - Pre-existing test footer `Results: $PASS passed, 0 failed` visible at end of test output
    - Lines preserved: the `usage()` function, `sign_and_notarize()` function, and `build_macos`/`build_windows`/`build_linux` functions are UNCHANGED. Verify: `grep -c '^sign_and_notarize()' build.sh` returns 1; `grep -c '^build_macos()' build.sh` returns 1.
  </acceptance_criteria>
  <done>
    build.sh has the new go-list-derived wails check block with WAILS_PINNED_VER sanity gate. The existing test suite (with Plan 01's Section 12 additions) runs green end-to-end. Commit message: `build(90): replace wails@latest with go list -m pattern in build.sh (SEC-10 + Pitfall 5)`.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Workflow `uses:` → action code | Every `uses:` is a remote code execution. SHA-pinning is the authoritative defense: the pinned SHA is immutable even if the upstream tag is moved. |
| build.sh → local wails binary | The script runs wails as a subprocess during `--platform macos/linux/windows`. The WAILS_PINNED_VER gate ensures the developer knows which version will be installed; it does not verify the binary's identity. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-90-11 | Tampering | Upstream `actions/checkout@v6` tag moved to attacker commit | mitigate | SHA-pin `@de0fac2e4500dabe0009e67214ff5f5447ce83dd` is immutable. The tag-move attack (tj-actions changed-files 2025-03) is neutralized because the SHA does not depend on the tag. Future reviews via Dependabot PRs, which bump both SHA and comment atomically. |
| T-90-12 | Tampering | Attacker opens PR that sets `uses: actions/checkout@<old-SHA>` (downgrade) | accept | PR review catches version regressions; the trailing `# vX.Y.Z` comment makes downgrades visually obvious. Grep-gate does not detect this specific attack, but the comment-SHA pair is a standard review artifact. |
| T-90-13 | Spoofing | Typo in SHA (e.g., one-char different) silently points at a different commit | mitigate | Grep-gate's third check (`grep -Ev '@[a-f0-9]{40}(\s|$)'`) catches short/long SHAs. Typos within the 40 hex chars still produce valid 40-char strings — not detected by gate. Mitigated by: the Dependabot PR pattern (atomic SHA + comment bumps from authoritative upstream), and the fact that a typo'd SHA typically points at a non-existent commit and `uses:` resolution fails at runtime. |
| T-90-14 | Information Disclosure | `build.sh` error message leaks go.mod path | accept | Error is dev-facing only; go.mod is public in the public repo. No new info exposed. |
| T-90-15 | Denial of Service | `go list -m` hangs or network-unavailable during `build.sh` execution | accept | `go list -m` on an already-required module reads local `go.mod` — no network call in normal operation. Edge case (first run with unfetched deps) falls back to `GOPROXY`; `set -euo pipefail` surfaces any error. |
| T-90-16 | Elevation of Privilege | Compromised `go list` subshell returns attacker-chosen string | mitigate | The `[ -n "$WAILS_VER" ]` check prevents empty-string fallback to `@latest`. A non-empty attacker string still passes the check but would have to be a valid-shaped version string; downstream `go install` with a non-existent version fails fast. Defense in depth via the `wails version` sanity check that follows. |

**Residual risk:** `upload-artifact@v7` is a new pin (bumped from `v4` of same source). A1 acknowledges v4→v7 is backward-compatible for plain files but not exhaustively tested. If v7 breaks build.yml upload steps, roll back to pinned v4 SHA `11bd71901bbe5b1630ceea73d27597364c9af683` (verified in 90-RESEARCH.md footnote). Grep-gate passes either way.
</threat_model>

<verification>
After all three tasks land:
1. `grep -c '@latest' build.sh .github/workflows/build.yml .github/workflows/release-please.yml` returns 0 total (SEC-10)
2. `grep -rcE '@(main|master|v[0-9]+)$' .github/workflows/build.yml .github/workflows/release-please.yml` returns 0 (SEC-09 for these two files)
3. `bash scripts/grep-gate.sh` STILL FAILS — release.yml and distribute.yml still unpinned (owned by Plans 04 and 05). This is expected partial progress.
4. `bash tests/build-script.test.sh` exits 0 with 0 failures — all Plan 01 Section 12 assertions green.
5. `python3 -c 'import yaml; [yaml.safe_load(open(f)) for f in [".github/workflows/build.yml", ".github/workflows/release-please.yml"]]'` exits 0 — both YAMLs still valid.
</verification>

<success_criteria>
- build.yml has zero floating refs, all 8 `uses:` lines SHA-pinned with comments
- build.yml wails install uses `go list -m -f '{{.Version}}'` with the `[ -n "$WAILS_VER" ]` Pitfall-5 gate
- release-please.yml SHA-pinned to v5.0.0
- build.sh has zero `@latest`, the new WAILS_PINNED_VER block, and all existing functions intact
- tests/build-script.test.sh runs to completion with 0 failures
- Plans 04 and 05 can proceed (files_modified: release.yml, distribute.yml — no overlap with this plan)
</success_criteria>

<output>
After completion, create `.planning/phases/90-release-pipeline-hardening/90-03-SUMMARY.md` documenting:
- Files modified: `.github/workflows/build.yml` (8 SHA pins + 1 install swap), `.github/workflows/release-please.yml` (1 SHA pin + minor version bump v4 → v5.0.0), `build.sh` (7-line block replacement lines 61-67)
- Grep-gate state: now fails on release.yml + distribute.yml only (down from all four workflows)
- Section 12 of `tests/build-script.test.sh`: all three assertions green
- Handoff to Plan 04: can start immediately; files_modified = release.yml (no overlap with this plan)
</output>
