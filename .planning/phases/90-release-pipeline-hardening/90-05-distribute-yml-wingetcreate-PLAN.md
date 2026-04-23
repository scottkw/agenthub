---
phase: 90-release-pipeline-hardening
plan: 05
type: execute
wave: 4
depends_on: [04]
files_modified:
  - .github/workflows/distribute.yml
autonomous: true
requirements: [SEC-09, SEC-11]
tags: [ci, hardening, distribute-yml, wingetcreate, rc-guards, wave-4]

must_haves:
  truths:
    - ".github/workflows/distribute.yml has zero @main, @master, or unpinned refs — every uses: is SHA-pinned with a # vX.Y.Z comment"
    - "The floating third-party ref vedantmgoyal9/winget-releaser@main is GONE; replaced by an inline wingetcreate.exe invocation on windows-latest with SHA-256 verification against 8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D"
    - "submit-winget runs on windows-latest (not ubuntu-latest — wingetcreate is .NET+Windows only)"
    - "submit-winget skips on rc tags: if: !contains(github.ref, '-rc') — hyphen-anchored"
    - "update-homebrew-tap checks out release-90-test branch when the tag is rc; main branch otherwise — both the checkout ref: and the git push target match the rc-condition"
  artifacts:
    - path: ".github/workflows/distribute.yml"
      provides: "First-party WinGet submission via wingetcreate + rc-aware tap branch routing"
      contains: "wingetcreate.exe"
  key_links:
    - from: "submit-winget job"
      to: "microsoft/winget-create v1.12.8.0 release asset"
      via: "Invoke-WebRequest + Get-FileHash SHA-256 verification"
      pattern: "8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D"
    - from: "update-homebrew-tap checkout + push steps"
      to: "scottkw/homebrew-agenthub branch selection"
      via: "contains(github.ref, '-rc') ternary — release-90-test if rc, main if not"
      pattern: "release-90-test"
---

<objective>
Finish SEC-09 by eliminating the last floating third-party ref in the repo (`vedantmgoyal9/winget-releaser@main`) and implementing D-08 (first-party wingetcreate swap), D-16 (rc tap branch routing), and D-17 (rc WinGet skip). Also SHA-pin the remaining `actions/checkout` and `nick-fields/retry` refs in distribute.yml.

Purpose: After Plan 03 pinned build.yml + release-please.yml, and Plan 04 pinned release.yml, distribute.yml is the last workflow file with floating refs. This plan closes the SEC-09 acceptance gap for the full workflow surface. The wingetcreate swap is also a SEC-11-adjacent control — the first-party Microsoft tool removes one third-party actor from the distribute trust graph.

Output: Updated `.github/workflows/distribute.yml`. Grep-gate now passes end-to-end. Wave 0 grep-gate script becomes CI-green for the first time.
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

@.planning/phases/90-release-pipeline-hardening/90-01-SUMMARY.md
@.planning/phases/90-release-pipeline-hardening/90-04-SUMMARY.md

@./CLAUDE.md

<interfaces>
<!-- Current state of distribute.yml — from live read -->

Current layout:
- `update-homebrew-tap` job (ubuntu-latest) with checkout + retry + checksum fetch + tap update + git push
- `submit-winget` job (ubuntu-latest) with the floating `vedantmgoyal9/winget-releaser@main`

Lines to edit:
- Line 27: `nick-fields/retry@v3` → SHA-pinned v4.0.0
- Line 49: `actions/checkout@v4` → SHA-pinned v6.0.2
- Line 52: `token:` stays (TAP_DEPLOY_TOKEN is correct here — its legitimate home per D-02)
- Lines 48-53: add `ref:` with D-16 branch-selection ternary
- Lines 62-70: modify `git push` to target the rc-aware branch
- Lines 72-81: REPLACE the entire submit-winget job body

<!-- wingetcreate recipe — verbatim from 90-PATTERNS.md lines 266-295 / 90-RESEARCH.md Pattern 4 lines 393-422 -->

```yaml
submit-winget:
  runs-on: windows-latest           # D-08: wingetcreate is .NET+Windows only
  continue-on-error: true           # preserve — WinGet first submission is known-flaky
  if: ${{ !contains(github.ref, '-rc') }}   # D-17: skip rc tags
  env:
    WINGET_CREATE_GITHUB_TOKEN: ${{ secrets.WINGET_TOKEN }}
    WINGETCREATE_VERSION: v1.12.8.0
    WINGETCREATE_SHA256: 8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D
  steps:
    - name: Download and verify wingetcreate.exe
      shell: pwsh
      run: |
        $url = "https://github.com/microsoft/winget-create/releases/download/$env:WINGETCREATE_VERSION/wingetcreate.exe"
        Invoke-WebRequest -Uri $url -OutFile wingetcreate.exe
        $actual = (Get-FileHash wingetcreate.exe -Algorithm SHA256).Hash
        if ($actual -ne $env:WINGETCREATE_SHA256) {
          Write-Error "SHA256 mismatch: expected $env:WINGETCREATE_SHA256 got $actual"
          exit 1
        }
    - name: Submit to WinGet
      shell: pwsh
      run: |
        $version = "${{ github.event.release.tag_name }}".Trim('v')
        $installerUrl = "https://github.com/scottkw/agenthub/releases/download/${{ github.event.release.tag_name }}/agenthub-${{ github.event.release.tag_name }}-windows-amd64-installer.exe"
        .\wingetcreate.exe update scottkw.agenthub `
          --version $version `
          --urls $installerUrl `
          --submit
```

<!-- D-16 tap branch ternary pattern -->

```yaml
ref: ${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}
```

For the `git push` step, derive the branch name the same way:
```bash
BRANCH="${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}"
git push origin HEAD:"$BRANCH"
```

<!-- SHA-pin table (subset used here) -->

| Action | Pinned form |
|--------|-------------|
| `actions/checkout` | `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2` |
| `nick-fields/retry` | `nick-fields/retry@ad984534de44a9489a53aefd81eb77f87c70dc60 # v4.0.0` |
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Update update-homebrew-tap — SHA-pin actions + add rc-aware branch selection (D-16)</name>
  <read_first>
    - .github/workflows/distribute.yml (FULL FILE — 82 lines)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 222-247 — tap rc-branch guard pattern; lines 679-693 — rc-tag guards)
    - .planning/phases/90-release-pipeline-hardening/90-CONTEXT.md (D-16 — tap branch routing; D-02 clarification — TAP_DEPLOY_TOKEN belongs ONLY to distribute.yml)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Anti-pattern line 436 — hyphen anchor)
  </read_first>
  <files>.github/workflows/distribute.yml</files>
  <action>
    Modify the `update-homebrew-tap:` job in place. Four changes:

    **Change 1 — SHA-pin `nick-fields/retry` (line 27):**
    ```yaml
    # BEFORE
    uses: nick-fields/retry@v3

    # AFTER
    uses: nick-fields/retry@ad984534de44a9489a53aefd81eb77f87c70dc60 # v4.0.0
    ```

    **Change 2 — SHA-pin `actions/checkout` (line 49):**
    ```yaml
    # BEFORE
    uses: actions/checkout@v4

    # AFTER
    uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
    ```

    **Change 3 — Add rc-aware branch selection to the checkout step (D-16):**
    ```yaml
    - name: Checkout tap repo
      uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
      with:
        repository: scottkw/homebrew-agenthub
        ref: ${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}
        token: ${{ secrets.TAP_DEPLOY_TOKEN }}
        path: tap
    ```

    Note the added `ref:` line. Without it, checkout defaults to the tap repo's default branch (main), which means rc flows would poison the public-visible tap. With it, rc tags check out `release-90-test`; non-rc tags check out `main`.

    **TAP_DEPLOY_TOKEN stays** (line 52) — per D-02, TAP_DEPLOY_TOKEN's legitimate home is exactly this checkout/push flow. It does NOT belong in release.yml (which Plan 04 already fixed).

    **Change 4 — rc-aware git push (lines 62-70):**

    Current:
    ```yaml
    - name: Commit and push formula update
      run: |
        cd tap
        git config user.name "github-actions[bot]"
        git config user.email "github-actions[bot]@users.noreply.github.com"
        git add Casks/agenthub.rb
        git diff --cached --quiet && echo "No changes to commit" && exit 0
        git commit -m "agenthub ${{ steps.version.outputs.version }}"
        git push
    ```

    Replace with:
    ```yaml
    - name: Commit and push formula update
      run: |
        cd tap
        BRANCH="${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}"
        git config user.name "github-actions[bot]"
        git config user.email "github-actions[bot]@users.noreply.github.com"
        git add Casks/agenthub.rb
        git diff --cached --quiet && echo "No changes to commit" && exit 0
        git commit -m "agenthub ${{ steps.version.outputs.version }}"
        git push origin HEAD:"$BRANCH"
    ```

    Two changes: `BRANCH=` derivation at top of script; `git push origin HEAD:"$BRANCH"` at bottom. The `HEAD:"$BRANCH"` refspec is important — it pushes the current commit TO the target branch even if they don't share a history pattern locally (checkout pulled release-90-test, we commit locally, push HEAD to the same target name remotely).

    **Hyphen-anchoring emphasized:** both ternaries use `contains(github.ref, '-rc')` NOT `'rc'` — the anti-pattern defense repeated from Plan 04. A tag like `v3.1.0-archive` would false-match `'rc'`; only `'-rc'` avoids that.

    **PRESERVE everything else in this job:**
    - `runs-on: ubuntu-latest` — tap update is bash+sed+git, no Windows needed
    - The `Extract version` step (lines 19-24)
    - The retry logic around checksums download (lines 26-35) — retry action is SHA-pinned by Change 1 but structure unchanged
    - The `Extract DMG SHA256` step (lines 37-46)
    - The `Update cask formula` step (lines 55-60) — sed-based cask update logic is fine
    - `env: RELEASE_TAG: ${{ inputs.tag || github.ref_name }}` (lines 12-13) at workflow level — preserved
  </action>
  <verify>
    <automated>grep -F 'nick-fields/retry@ad984534de44a9489a53aefd81eb77f87c70dc60 # v4.0.0' .github/workflows/distribute.yml && grep -F 'actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2' .github/workflows/distribute.yml && grep -F "contains(github.ref, '-rc') && 'release-90-test' || 'main'" .github/workflows/distribute.yml | (read line; [ -n "$line" ] && echo "PASS: branch ternary present" || (echo "FAIL: ternary missing"; exit 1)) && grep -F 'git push origin HEAD:"$BRANCH"' .github/workflows/distribute.yml</antl-automated></automated>
  </verify>
  <acceptance_criteria>
    - `grep -c 'nick-fields/retry@ad984534de44a9489a53aefd81eb77f87c70dc60 # v4.0.0' .github/workflows/distribute.yml` returns 1
    - `grep -c 'nick-fields/retry@v3' .github/workflows/distribute.yml` returns 0
    - `grep -c 'actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2' .github/workflows/distribute.yml` returns 1
    - `grep -F "ref: \${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}" .github/workflows/distribute.yml` matches (the checkout ref:)
    - `grep -c 'release-90-test' .github/workflows/distribute.yml` returns >= 2 (one in checkout ref, one in git push BRANCH)
    - `grep -F 'git push origin HEAD:' .github/workflows/distribute.yml` matches (the explicit refspec)
    - `grep -c "contains(github.ref, 'rc')" .github/workflows/distribute.yml` returns 0 (no non-hyphen-anchored variant)
    - `grep -F 'TAP_DEPLOY_TOKEN' .github/workflows/distribute.yml` matches 1 (preserved — its legitimate home per D-02)
    - `grep -c 'git push' .github/workflows/distribute.yml` returns 1 (exactly one push invocation, not two)
    - YAML valid: `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/distribute.yml"))'` exits 0
  </acceptance_criteria>
  <done>
    update-homebrew-tap SHA-pinned, rc-aware with release-90-test branch routing on both checkout and push, TAP_DEPLOY_TOKEN preserved as the legitimate deploy token for this job. Commit: `ci(90): rc-aware tap branch routing + SHA-pin (SEC-09 D-16)`.
  </done>
</task>

<task type="auto">
  <name>Task 2: Replace submit-winget job — first-party wingetcreate on windows-latest with rc skip (D-08 + D-17)</name>
  <read_first>
    - .github/workflows/distribute.yml (lines 72-81 — the whole current submit-winget job to be replaced)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 249-296 — the verbatim replacement block and rationale)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Pattern 4 lines 387-428 — including SHA-256 verification recipe and token env-var alias rationale; Assumption A3 line 745 — standalone .exe is self-contained; surprise #1 line 78 — Windows-only requirement)
    - .planning/phases/90-release-pipeline-hardening/90-CONTEXT.md (D-08 wingetcreate swap; D-17 rc skip)
  </read_first>
  <files>.github/workflows/distribute.yml</files>
  <action>
    **Replace the entire `submit-winget:` job** (current lines 72-81) with the wingetcreate recipe from the `<interfaces>` block above (verbatim from 90-PATTERNS.md lines 266-295).

    The replacement encodes five changes from the current state:
    1. **Runner swap** (D-08 / surprise #1): `ubuntu-latest` → `windows-latest`. `wingetcreate.exe` is a .NET binary; it does not run on Linux. RESEARCH line 78 verified against Microsoft's own workflow examples (microsoft/edit, microsoft/terminal, github/copilot-cli all use `runs-on: windows-latest`).
    2. **Third-party action removal** (SEC-09): the floating `vedantmgoyal9/winget-releaser@main` is the last floating ref in the repo. Replaced by inline PowerShell that downloads + SHA-verifies + invokes Microsoft's own `wingetcreate.exe`.
    3. **Integrity check**: SHA-256 hash verification of the downloaded `wingetcreate.exe` against `8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D` (authoritative hash from `curl https://github.com/microsoft/winget-create/releases/download/v1.12.8.0/wingetcreate.exe.txt | iconv -f UTF-16LE -t UTF-8` on 2026-04-23 per RESEARCH line 136). A hash mismatch causes the step to exit 1 — tamper-evident download.
    4. **Token aliasing**: the project's existing `WINGET_TOKEN` secret is re-exported as `WINGET_CREATE_GITHUB_TOKEN` — the env var name Microsoft's `wingetcreate` expects. The secret itself is unchanged; only the exposed name differs.
    5. **rc skip** (D-17): `if: ${{ !contains(github.ref, '-rc') }}`. rc tags like `v3.1.0-rc1` should NOT submit to the public WinGet manifest repo (avoids polluting their review queue). Hyphen-anchored consistent with all other guards in this plan.

    **continue-on-error: true PRESERVED** — current distribute.yml line 74 has a comment: `# WinGet first submission pending — remove after accepted`. This is load-bearing. RESEARCH line 437 flags that WinGet's first submission is known-flaky; continue-on-error prevents a tap-update success from being hidden by a WinGet-submission flake. Keep the flag AND the comment. Future phase work removes it after first successful submission.

    **Installer URL pattern:** The `wingetcreate update` step hardcodes the URL pattern:
    ```
    https://github.com/scottkw/agenthub/releases/download/${{ github.event.release.tag_name }}/agenthub-${{ github.event.release.tag_name }}-windows-amd64-installer.exe
    ```
    This matches the filename produced by release.yml's `build-windows` job (release.yml line 199 in the pre-Plan-04 state: `agenthub-${VERSION}-windows-amd64-installer.exe`). Plan 04's changes to release.yml PRESERVE this filename pattern (verified: the rename step in release.yml build-windows kept `mv ... agenthub-${VERSION}-windows-amd64-installer.exe`). No coordination bug.

    **Verbatim block to insert** (replace current lines 72-81 — the entire submit-winget job — with):

    ```yaml
      submit-winget:
        runs-on: windows-latest
        continue-on-error: true  # WinGet first submission pending — remove after accepted
        if: ${{ !contains(github.ref, '-rc') }}
        env:
          WINGET_CREATE_GITHUB_TOKEN: ${{ secrets.WINGET_TOKEN }}
          WINGETCREATE_VERSION: v1.12.8.0
          WINGETCREATE_SHA256: 8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D
        steps:
          - name: Download and verify wingetcreate.exe
            shell: pwsh
            run: |
              $url = "https://github.com/microsoft/winget-create/releases/download/$env:WINGETCREATE_VERSION/wingetcreate.exe"
              Invoke-WebRequest -Uri $url -OutFile wingetcreate.exe
              $actual = (Get-FileHash wingetcreate.exe -Algorithm SHA256).Hash
              if ($actual -ne $env:WINGETCREATE_SHA256) {
                Write-Error "SHA256 mismatch: expected $env:WINGETCREATE_SHA256 got $actual"
                exit 1
              }
          - name: Submit to WinGet
            shell: pwsh
            run: |
              $version = "${{ github.event.release.tag_name }}".Trim('v')
              $installerUrl = "https://github.com/scottkw/agenthub/releases/download/${{ github.event.release.tag_name }}/agenthub-${{ github.event.release.tag_name }}-windows-amd64-installer.exe"
              .\wingetcreate.exe update scottkw.agenthub `
                --version $version `
                --urls $installerUrl `
                --submit
    ```

    Note PowerShell backtick-continuation (`` ` ``) on the `wingetcreate` invocation — this is the Windows PowerShell line-continuation syntax and is correct for `shell: pwsh`. Do NOT substitute with `\` (bash-style).

    **Final verification against grep-gate:** after this task, `bash scripts/grep-gate.sh` MUST exit 0 for the first time in Phase 90. Every workflow file is now SHA-pinned; zero `@main`/`@master`/`@latest` across `.github/workflows/`, `build.sh`, `tests/`.
  </action>
  <verify>
    <automated>grep -F 'runs-on: windows-latest' .github/workflows/distribute.yml && grep -F 'vedantmgoyal9/winget-releaser' .github/workflows/distribute.yml | head -1 | (read line; [ -z "$line" ] && echo "PASS: vedantmgoyal9 ref gone" || (echo "FAIL: floating ref still present: $line"; exit 1)) && grep -F '8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D' .github/workflows/distribute.yml && grep -F "if: \${{ !contains(github.ref, '-rc') }}" .github/workflows/distribute.yml && grep -F 'WINGET_CREATE_GITHUB_TOKEN: ${{ secrets.WINGET_TOKEN }}' .github/workflows/distribute.yml && bash scripts/grep-gate.sh && echo "FINAL PASS: grep-gate is GREEN — Phase 90 SHA-pin complete"</automated>
  </verify>
  <acceptance_criteria>
    - `grep -c 'vedantmgoyal9/winget-releaser' .github/workflows/distribute.yml` returns 0 (last floating ref eliminated — SEC-09 closeout)
    - `grep -c '^  submit-winget:' .github/workflows/distribute.yml` returns 1 (job still exists — not deleted, replaced)
    - `grep -F 'runs-on: windows-latest' .github/workflows/distribute.yml` matches (D-08)
    - `grep -F 'continue-on-error: true' .github/workflows/distribute.yml` matches 1 (preserved — WinGet flakiness defense)
    - `grep -F 'WinGet first submission pending' .github/workflows/distribute.yml` matches (comment preserved — Chesterton's Fence)
    - `grep -F "if: \${{ !contains(github.ref, '-rc') }}" .github/workflows/distribute.yml` matches (D-17 rc skip)
    - `grep -c "contains(github.ref, 'rc')" .github/workflows/distribute.yml` returns 0 (no non-hyphen-anchored `'rc'` slipped in)
    - `grep -F 'WINGET_CREATE_GITHUB_TOKEN: ${{ secrets.WINGET_TOKEN }}' .github/workflows/distribute.yml` matches (token aliasing)
    - `grep -F 'WINGETCREATE_VERSION: v1.12.8.0' .github/workflows/distribute.yml` matches
    - `grep -F '8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D' .github/workflows/distribute.yml` matches (authoritative SHA-256 from RESEARCH line 136)
    - `grep -F 'SHA256 mismatch' .github/workflows/distribute.yml` matches (the tamper-evident check)
    - `grep -F 'Invoke-WebRequest -Uri $url -OutFile wingetcreate.exe' .github/workflows/distribute.yml` matches (PowerShell download)
    - `grep -F 'wingetcreate.exe update scottkw.agenthub' .github/workflows/distribute.yml` matches (submit invocation)
    - `grep -F 'agenthub-${{ github.event.release.tag_name }}-windows-amd64-installer.exe' .github/workflows/distribute.yml` matches (installer URL pattern aligns with release.yml filename output)
    - `grep -c '@latest' .github/workflows/distribute.yml` returns 0
    - `grep -cE 'uses:\s*[^ ]+@' .github/workflows/distribute.yml | head -1` — count equals the number of `uses:` lines, and every one is a 40-char SHA: `grep -E 'uses:\s*[^ ]+@' .github/workflows/distribute.yml | grep -Ev '@[a-f0-9]{40}(\s|$)'` returns 0 matches
    - YAML valid: `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/distribute.yml"))'` exits 0
    - **Grep-gate cross-repo PASS:** `bash scripts/grep-gate.sh` exits 0 (the Wave 0 gate is now GREEN — Phase 90's SEC-09 + SEC-10 objectives for static analysis are complete)
  </acceptance_criteria>
  <done>
    submit-winget runs on windows-latest, uses first-party wingetcreate with SHA-256-verified download, skips rc tags, aliases token env var. The last floating ref in the repo (`vedantmgoyal9/winget-releaser@main`) is gone. The Wave 0 grep-gate passes end-to-end. Commit: `ci(90): swap winget-releaser for wingetcreate on windows-latest + rc skip (SEC-09 + SEC-11 D-08 + D-17)`.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| distribute.yml → scottkw/homebrew-agenthub | `TAP_DEPLOY_TOKEN` is the trust-bearing credential. rc-branch routing prevents test artifacts from polluting the public tap. |
| submit-winget → microsoft/winget-pkgs | `WINGET_TOKEN` permits PR creation on Microsoft's review queue. rc skip prevents test submissions from entering the queue. |
| wingetcreate.exe download → execution | SHA-256 verification is the tamper-evident check. Without it, a compromised GitHub releases asset would run with the WINGET_TOKEN scope. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-90-26 | Tampering | `vedantmgoyal9/winget-releaser@main` upstream compromise (the floating-ref attack class) | mitigate (eliminated) | D-08: the action is removed entirely. Microsoft's first-party `wingetcreate.exe` replaces it. Threat surface drops by one third-party action. |
| T-90-27 | Tampering | Attacker compromises microsoft/winget-create v1.12.8.0 release asset | mitigate | SHA-256 verification against `8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D`. Authoritative hash sourced from Microsoft's own `.txt` sidecar (verified via `curl ... \| iconv -f UTF-16LE -t UTF-8` per RESEARCH line 136). Asset substitution detected before execution. |
| T-90-28 | Information Disclosure | Test artifacts (rc tags) leak to public Homebrew tap users | mitigate | D-16: rc tags check out + push to `release-90-test` branch; non-rc tags target `main`. Tap users on the default tap branch see only real releases. |
| T-90-29 | Information Disclosure | Test artifacts (rc tags) pollute Microsoft's WinGet review queue | mitigate | D-17: `if: !contains(github.ref, '-rc')` gates the entire submit-winget job. rc tags skip cleanly (the tap update job still runs with its own rc-branch routing). |
| T-90-30 | Spoofing | False-positive rc match on non-rc tags (e.g., `v3.1.0-archive`) | mitigate | Hyphen-anchored `'-rc'` used in all three rc conditions (tap checkout ref, tap git push BRANCH, submit-winget if). Grep-gate in acceptance criteria rejects any `contains(github.ref, 'rc')` (non-hyphen) slip. |
| T-90-31 | Elevation of Privilege | WINGET_TOKEN scope creep — could the token submit PRs with write access to agenthub repo? | accept | Scope audit is out-of-scope for this plan (token is managed in repo secrets UI). The env-var renaming from `WINGET_TOKEN` → `WINGET_CREATE_GITHUB_TOKEN` is aliasing only — same secret. 90-RESEARCH.md Environment Availability line 780 confirms token scope is `public_repo` (submission PR scope), which is minimum-necessary for wingetcreate's workflow. |
| T-90-32 | Denial of Service | Microsoft releases server outage during wingetcreate download | accept | `continue-on-error: true` is preserved on the job — a transient outage doesn't fail the overall distribute workflow. The rc-skip already narrows exposure to real-tag runs. |
| T-90-33 | Repudiation | Disputed WinGet submission — was it actually AgentHub's? | accept | WINGET_CREATE_GITHUB_TOKEN's `public_repo` scope means the forked PR is authored by the `scottkw` account. PR history on github/winget-pkgs is the authoritative record. No cryptographic attestation needed for this legacy submission path. |

**Residual risk:**
- **wingetcreate v1.12.8.0 is a specific pinned version.** Microsoft may release v1.13+ with bug fixes or required for newer WinGet schema. Dependabot does not (currently) update pinned-in-env-var versions embedded in workflow YAMLs — this is a manual review item. Mitigate by documenting this as a quarterly re-check item.
- **Installer URL pattern is hardcoded** in PowerShell. If release.yml's Windows artifact naming ever changes, the wingetcreate submit step breaks silently (URL 404 causes wingetcreate to fail). Mitigate: the URL pattern is explicit and testable; add a comment pointing back to release.yml's line producing the filename.
- **rc-branch hygiene on the tap:** D-16 requires `release-90-test` branch to exist on `scottkw/homebrew-agenthub`. Plan 01's 90-TAP-BRANCH-SETUP.md documents the manual precreation step. If the branch does not exist at rc time, the checkout step fails (not a silent-success; visible in Actions log). This is the intended fail-closed behavior.
</threat_model>

<verification>
After both tasks land:
1. **YAML validity:** `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/distribute.yml"))'` exits 0
2. **Floating refs eliminated:** `grep -c 'vedantmgoyal9' .github/workflows/distribute.yml` returns 0
3. **All uses: SHA-pinned:** `grep -E 'uses:\s*[^ ]+@' .github/workflows/distribute.yml | grep -Ev '@[a-f0-9]{40}(\s|$)'` returns 0 matches
4. **rc guards hyphen-anchored:** `grep -c "contains(github.ref, 'rc')" .github/workflows/distribute.yml` returns 0 (only `'-rc'` variants)
5. **wingetcreate hash present:** `grep -F '8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D' .github/workflows/distribute.yml` matches
6. **Cross-repo grep-gate PASS:** `bash scripts/grep-gate.sh` exits 0 — first time in the phase
7. **Static SHA-pin verification bundle:** All four workflow files are now pin-clean. Print summary:
   ```bash
   for f in .github/workflows/*.yml; do
     echo "$f: $(grep -c '@' "$f" | awk '{print $1}') uses lines, all SHA-pinned (see grep-gate)"
   done
   ```
</verification>

<success_criteria>
- distribute.yml SHA-pinned end-to-end
- Floating ref `vedantmgoyal9/winget-releaser@main` eliminated
- submit-winget runs on windows-latest with SHA-256-verified wingetcreate.exe
- rc tags: tap targets release-90-test branch, winget submission skipped
- Grep-gate passes for the entire repo — SEC-09 static-analysis objective complete
- Plan 06 (Wave 5 E2E) can proceed — all code changes are now in place
</success_criteria>

<output>
After completion, create `.planning/phases/90-release-pipeline-hardening/90-05-SUMMARY.md` documenting:
- File modified: `.github/workflows/distribute.yml` — update-homebrew-tap rc-aware + SHA-pinned; submit-winget swapped to first-party wingetcreate on windows-latest
- Final grep-gate state: GREEN across all workflows + build.sh + tests
- Completed decisions: D-02 (confirmed — TAP_DEPLOY_TOKEN properly scoped here), D-08 (wingetcreate), D-16 (tap branch), D-17 (rc winget skip)
- Handoff to Plan 06: all code ready for end-to-end rc verification. Pre-flight items: (1) manually create `release-90-test` branch per 90-TAP-BRANCH-SETUP.md, (2) `gh api /repos/scottkw/agenthub/environments/release` to audit protection rules (RESEARCH Open Q4).
</output>
