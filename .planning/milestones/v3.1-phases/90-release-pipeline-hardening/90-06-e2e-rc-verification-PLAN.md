---
phase: 90-release-pipeline-hardening
plan: 06
type: execute
wave: 5
depends_on: [01, 02, 03, 04, 05]
files_modified: []
autonomous: false
requirements: [SEC-09, SEC-10, SEC-11]
tags: [ci, hardening, verification, rc, e2e, wave-5, human-checkpoint]

must_haves:
  truths:
    - "A real v3.1.0-rc1 git tag has been pushed and the Release workflow ran to completion with sign-macos verifying the internal attestation successfully, publish generating release attestations, and the release created as a DRAFT (not public)"
    - "The draft release contains: signed+notarized macOS DMG, Windows installer + exe, Linux tar.gz, Linux .deb, checksums.txt — all files present with non-zero size"
    - "gh attestation verify <file> --owner scottkw succeeds for every file in the draft release, proving the release-provenance attestation is externally verifiable"
    - "The distribute.yml run (triggered by release:published after draft was un-drafted for tap test OR manually via workflow_dispatch) updated ONLY the release-90-test branch of scottkw/homebrew-agenthub and SKIPPED the submit-winget job (rc-guarded)"
    - "After human UAT confirmation: the draft release has been deleted, the release-90-test branch changes have been reverted or archived, and no poisoning of public distribution channels occurred"
  artifacts:
    - path: ".planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md"
      provides: "Signed record of the v3.1.0-rc1 E2E verification outcome"
      must_exist: true
  key_links:
    - from: "local push of v3.1.0-rc1 tag"
      to: "release.yml workflow run — validate → build → sign-macos → publish"
      via: "tag push triggers release.yml on:push:tags:'v*'"
      pattern: "v3.1.0-rc1"
    - from: "publish job attestation"
      to: "external gh attestation verify"
      via: "Sigstore public-good — externally verifiable from any clean client"
      pattern: "gh attestation verify.*--owner scottkw"
---

<objective>
Execute the D-14 end-to-end verification — cut a real `v3.1.0-rc1` pre-release tag and prove the Phase 90 restructured pipeline produces signed, notarized, attested artifacts without poisoning public distribution channels. This is the SC-4 acceptance for the phase.

Purpose: Static analysis (grep-gate, source-grep tests) proves the workflow files are SHA-pinned and secret-scoped as specified. But only a real tag cut proves:
1. The attestation trust boundary actually works (sign-macos verifies build-macos's bundle cryptographically).
2. The tar/untar handoff preserves the .app's symlinks and +x bits through actions/upload-artifact.
3. `draft: contains(github.ref, '-rc')` actually creates a draft (not public) release.
4. `sign-macos` has the right permissions for offline `--bundle` verify.
5. `contains(github.ref, '-rc') && 'release-90-test' || 'main'` evaluates correctly in distribute.yml.
6. The `wingetcreate` skip on rc tags works.
7. Every published artifact is `gh attestation verify`-passable from a clean client.

Output: a draft GitHub release for `v3.1.0-rc1`, an external-verifier report (`90-UAT-REPORT.md`), and a human sign-off to either promote (by re-cutting `v3.1.0` non-rc) or iterate.

**This plan has checkpoints.** It requires human action (pre-flight tap branch creation, environment rules audit, tag push, draft cleanup) and human observation (reading the Actions logs, inspecting the draft release in the UI, running `gh attestation verify` from a clean shell).
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
@.planning/phases/90-release-pipeline-hardening/90-VALIDATION.md
@.planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md

@.planning/phases/90-release-pipeline-hardening/90-01-SUMMARY.md
@.planning/phases/90-release-pipeline-hardening/90-02-SUMMARY.md
@.planning/phases/90-release-pipeline-hardening/90-03-SUMMARY.md
@.planning/phases/90-release-pipeline-hardening/90-04-SUMMARY.md
@.planning/phases/90-release-pipeline-hardening/90-05-SUMMARY.md

@./CLAUDE.md

<interfaces>
<!-- Expected tag format (D-14): v3.1.0-rc1 -->
<!-- Why v3.1.0-rc1 specifically: matches v*-rc* glob for draft + tap rc-branch + winget skip -->
<!-- If v3.1.0 was already used: bump to v3.1.1-rc1 -->

Expected release.yml job flow on `v3.1.0-rc1` tag push:
```
validate → build-macos ─┐
         → build-windows ─┤
         → build-linux ─┬─┘
                       sign-macos → publish
```

Expected distribute.yml job flow on release:published (only fires after draft is manually published):
```
update-homebrew-tap  → pushes to release-90-test branch
submit-winget        → SKIPPED via if: !contains(github.ref, '-rc')
```

**Key verification commands** (run locally from a clean shell):
```bash
# (A) Internal attestation verification proof — look at workflow logs
# Navigate to sign-macos job logs; find the "Verify internal attestation" step; confirm:
#   ✅ Sigstore signature verified
#   ✅ Attestation matched subject: artifacts/AgentHub.app.tar.gz

# (B) External attestation verification — from any machine with gh CLI
gh attestation verify <downloaded-file.dmg> --owner scottkw
gh attestation verify <downloaded-file-installer.exe> --owner scottkw
gh attestation verify <downloaded-file.tar.gz> --owner scottkw
gh attestation verify <downloaded-file.deb> --owner scottkw
gh attestation verify <downloaded-checksums.txt> --owner scottkw
# All five must output "Loaded ... attestation(s) ... verification succeeded"

# (C) Draft release confirmation
gh release view v3.1.0-rc1 --json isDraft --jq .isDraft
# Expected: true

# (D) Tap branch verification (post release:published trigger, if we publish the draft or use workflow_dispatch)
git fetch origin release-90-test  # in a local clone of scottkw/homebrew-agenthub
git log origin/main..origin/release-90-test --oneline
# Expected: 1 new commit on release-90-test; main untouched

# (E) WinGet skip verification
gh run list --workflow distribute.yml --limit 5 --json event,headBranch,conclusion,name
# Find the rc-triggered run; submit-winget should show "conclusion: skipped"
```

**Reference UAT template** (skeleton the task fills in):
```markdown
# Phase 90 — v3.1.0-rc1 UAT Report

**Tag cut:** 2026-04-??
**Workflow run:** https://github.com/scottkw/agenthub/actions/runs/<id>
**Draft release:** https://github.com/scottkw/agenthub/releases/tag/v3.1.0-rc1

## Pipeline job outcomes

| Job | Status | Duration | Notes |
|-----|--------|----------|-------|
| validate | ✅ | 2m | |
| build-macos | ✅ | ?m | env: empty verified |
| build-windows | ✅ | ?m | |
| build-linux | ✅ | ?m | nfpm .deb produced |
| sign-macos | ✅ | ?m | gh attestation verify --bundle PASSED |
| publish | ✅ | ?m | Draft release created |

## External attestation verification

| File | gh attestation verify | Notes |
|------|----------------------|-------|
| agenthub-v3.1.0-rc1-darwin-universal.dmg | ✅ | |
| agenthub-v3.1.0-rc1-windows-amd64-installer.exe | ✅ | |
| agenthub-v3.1.0-rc1-windows-amd64.exe | ✅ | |
| agenthub-v3.1.0-rc1-linux-amd64.tar.gz | ✅ | |
| agenthub-v3.1.0-rc1-linux-amd64.deb | ✅ | |
| checksums.txt | ✅ | |

## distribute.yml run

| Job | Condition | Outcome |
|-----|-----------|---------|
| update-homebrew-tap | always | Pushed to release-90-test branch ✅ |
| submit-winget | if: !contains(github.ref, '-rc') | Skipped ✅ |

## Cleanup performed

- [ ] Draft release deleted
- [ ] release-90-test branch rewound (or preserved for v3.1.0 real cut)
- [ ] No secrets leaked in logs

## Sign-off

Phase 90 SC-4 acceptance: PASS / FAIL
Human reviewer: <name>
Date: <date>
```
</interfaces>
</context>

<tasks>

<task type="checkpoint:human-action" gate="blocking">
  <name>Task 1: Pre-flight — create release-90-test tap branch, audit release environment, verify Wave 1-4 landed</name>
  <read_first>
    - .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md (the runbook from Plan 01)
    - .planning/phases/90-release-pipeline-hardening/90-05-SUMMARY.md (handoff notes)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Open Question 4 line 768 — environment protection rule audit)
  </read_first>
  <what-built>Plans 01-05 have landed:
- scripts/grep-gate.sh passes
- tests/build-script.test.sh passes
- tools.go + go.mod + go.sum + .github/dependabot.yml committed
- build.yml + release-please.yml + build.sh SHA-pinned
- release.yml split (build-*, sign-macos, publish) with attestations
- distribute.yml swapped to wingetcreate
Before cutting the v3.1.0-rc1 tag, the human must (a) create the tap test branch, (b) audit the release environment for protection rules, (c) confirm CI is green on the phase branch.</what-built>
  <how-to-verify>
    **Step 1 — Run the static verification bundle locally:**
    ```bash
    # All five must pass
    bash scripts/grep-gate.sh
    bash tests/build-script.test.sh
    go build -tags tools ./...
    go test -race -short ./...
    python3 -c 'import yaml; [yaml.safe_load(open(f)) for f in ["/Users/ken/dev/agenthub/.github/workflows/build.yml", "/Users/ken/dev/agenthub/.github/workflows/release.yml", "/Users/ken/dev/agenthub/.github/workflows/distribute.yml", "/Users/ken/dev/agenthub/.github/workflows/release-please.yml", "/Users/ken/dev/agenthub/.github/dependabot.yml"]]'
    ```
    If any fails, fix before proceeding — do NOT push an rc tag against a red tree.

    **Step 2 — Create the `release-90-test` branch on the tap repo** (D-16 prerequisite):
    ```bash
    gh repo clone scottkw/homebrew-agenthub /tmp/homebrew-agenthub
    cd /tmp/homebrew-agenthub
    git fetch origin main
    git checkout -b release-90-test origin/main
    git push -u origin release-90-test

    # Verify
    gh api /repos/scottkw/homebrew-agenthub/branches/release-90-test --jq .name
    # Expected: release-90-test
    ```

    **Step 3 — Audit the `release` environment on scottkw/agenthub** (90-RESEARCH.md Open Question 4):
    ```bash
    gh api /repos/scottkw/agenthub/environments/release
    ```
    Inspect the output for:
    - Protection rules (required reviewers, wait timer)
    - Deployment branch policies
    - Secrets list (should include all MACOS_* — confirm the secrets are still bound to this environment)

    Plan 04 re-scoped `environment: release` from build-macos to sign-macos. If the environment has a required-reviewer rule, that rule now gates the sign-macos job (intended — the human reviewer sees only the signing action, which is the most sensitive). If the environment has a wait-timer, a delay inserts between validate-completion and sign-macos-start — acceptable but the human should know to expect it.

    **Document findings** inline in the UAT report template (Task 2 creates it) so Plan 06 Task 3 can reconcile observed behavior against expectations.

    **Step 4 — Confirm all phase commits are on the active branch:**
    ```bash
    git log --oneline -20 | head -10
    # Expected: 6+ commits with "ci(90):" / "deps(90):" / "test(90):" / "build(90):" prefixes
    # corresponding to Plans 01-05
    ```

    **Step 5 — Confirm build.yml is green on current HEAD:**
    ```bash
    gh run list --workflow build.yml --branch "$(git branch --show-current)" --limit 5 --json conclusion,status,headSha --jq '.[] | select(.headSha == "'"$(git rev-parse HEAD)"'")' | jq -s '.[0]'
    # Expected: conclusion: "success", status: "completed"
    ```
    If the build workflow hasn't run on the current HEAD yet, push a no-op commit or trigger via `gh workflow run build.yml` and wait.

    **Step 6 — Check that the `release` Dependabot hasn't opened PRs that conflict with Phase 90's pinned SHAs:**
    ```bash
    gh pr list --label dependencies --state open
    # Review any open PRs; merge-or-close them BEFORE cutting the rc tag if they touch workflow YAMLs
    ```
  </how-to-verify>
  <resume-signal>
    Type "pre-flight complete" once:
    - Static verification bundle green
    - release-90-test branch exists on scottkw/homebrew-agenthub
    - release environment rules audited (note any surprises inline in the UAT report you'll start in Task 2)
    - build.yml green on current HEAD
    - No conflicting Dependabot PRs open

    If any of the above fails, type "blocker: <description>" and fix BEFORE proceeding.
  </resume-signal>
</task>

<task type="checkpoint:human-action" gate="blocking">
  <name>Task 2: Cut v3.1.0-rc1 tag + observe release.yml workflow end-to-end + start UAT report</name>
  <read_first>
    - .planning/phases/90-release-pipeline-hardening/90-04-SUMMARY.md (expected release.yml job flow)
    - .planning/phases/90-release-pipeline-hardening/90-CONTEXT.md (D-14 — real tag cut; D-15 — draft behavior)
  </read_first>
  <what-built>The phase branch is fully committed and pre-flight is complete. Time to cut the rc tag and watch the split pipeline run end-to-end.</what-built>
  <how-to-verify>
    **Step 1 — Decide which tag to use:** Use `v3.1.0-rc1` unless that tag already exists on the repo. Check:
    ```bash
    git tag -l 'v3.1.*'
    ```
    If `v3.1.0-rc1` is already used, increment: `v3.1.0-rc2`, `v3.1.1-rc1`, etc. Record the actual tag in the UAT report.

    **Step 2 — Cut and push the tag:**
    ```bash
    # From the phase branch
    git tag -a v3.1.0-rc1 -m "Phase 90 rc verification: SHA-pin + tool-pin + split pipeline + attestations"
    git push origin v3.1.0-rc1
    ```

    **Step 3 — Watch release.yml run live:**
    ```bash
    gh run watch --workflow release.yml --exit-status
    ```
    Or open the Actions tab in the browser. Expect:
    1. `validate` — 2-3 minutes, green
    2. `build-macos`, `build-windows`, `build-linux` — 5-10 minutes each, running in parallel; all three end with an "Attest unsigned artifact (internal)" step + an "Upload unsigned ... + attestation bundle" step
    3. `sign-macos` — 3-5 minutes AFTER build-macos completes; it downloads the tarball + bundle, runs "Verify internal attestation" (watch this step closely — it's the D-04 proof), untars, signs, notarizes, creates DMG, uploads signed DMG
    4. `publish` — 1-2 minutes AFTER all build jobs + sign-macos; it downloads everything, generates checksums, attests via `subject-checksums`, creates the DRAFT release

    **Step 4 — Create `.planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md`** using the template from the `<interfaces>` block. Fill in:
    - Workflow run URL (from `gh run list --workflow release.yml --limit 1 --json url --jq .[0].url`)
    - Per-job status + duration (from `gh run view <run-id> --json jobs`)
    - Screenshot/quote the "Verify internal attestation" step log (look for "Sigstore" / "verification succeeded" text)

    **Step 5 — Confirm the draft status:**
    ```bash
    gh release view v3.1.0-rc1 --json isDraft,tagName --jq '"tag=\(.tagName) draft=\(.isDraft)"'
    # Expected: tag=v3.1.0-rc1 draft=true
    ```
    If `draft=false`, D-15 failed — the `contains(github.ref, '-rc')` guard did not evaluate to true. STOP. Capture the release.yml "Upload to GitHub Release" step logs. The guard may have an escaping issue (common: GitHub Actions expression `${{ contains(...) }}` vs Bash).

    **Step 6 — Confirm asset presence:**
    ```bash
    gh release view v3.1.0-rc1 --json assets --jq '.assets[].name'
    # Expected (non-empty names):
    #   agenthub-v3.1.0-rc1-darwin-universal.dmg
    #   agenthub-v3.1.0-rc1-windows-amd64-installer.exe
    #   agenthub-v3.1.0-rc1-windows-amd64.exe
    #   agenthub-v3.1.0-rc1-linux-amd64.tar.gz
    #   agenthub-v3.1.0-rc1-linux-amd64.deb
    #   checksums.txt
    ```
    Six files expected. If any are missing, consult the `publish` job's "Upload to GitHub Release" step logs — likely a glob mismatch.

    **If release.yml fails at any step, STOP the plan**, diagnose via `gh run view --log` (paste the failure into the UAT report under a "Findings" section), decide whether to revert and patch (return to Plan 03/04/05 as appropriate) or proceed with a hotfix commit on the same phase branch. A planner-revision cycle may be warranted if failures are systemic.
  </how-to-verify>
  <resume-signal>
    Type "release.yml green + draft confirmed + 6 assets present" once all of the above passes. UAT report Section 1 filled in.

    If release.yml fails, type "release.yml failed at <job>" — the plan pauses for diagnostic work; the orchestrator may trigger a planner-revision or route to plan-gaps.
  </resume-signal>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: External attestation verification + distribute.yml run + cleanup + sign-off</name>
  <read_first>
    - .planning/phases/90-release-pipeline-hardening/90-05-SUMMARY.md (distribute.yml expected behavior on rc)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Pattern 3 lines 369-385 — external `gh attestation verify` is the documented downstream-consumer command)
  </read_first>
  <what-built>The draft v3.1.0-rc1 release exists with 6 assets, all created by the new split pipeline with attestations. Now prove externally that the attestations verify, trigger distribute.yml, and confirm rc guards worked on the tap and winget paths.</what-built>
  <how-to-verify>
    **Step 1 — Download all assets from the draft to a clean directory:**
    ```bash
    mkdir -p /tmp/rc-verify && cd /tmp/rc-verify
    gh release download v3.1.0-rc1 --repo scottkw/agenthub
    ls -la
    # Expected: 6 files
    ```

    **Step 2 — Externally verify every attestation:**
    ```bash
    for f in agenthub-*.dmg agenthub-*.exe agenthub-*.tar.gz agenthub-*.deb checksums.txt; do
      [[ -f "$f" ]] || { echo "MISSING: $f"; continue; }
      echo "=== $f ==="
      gh attestation verify "$f" --owner scottkw
      echo
    done
    ```
    Each run must output `Loaded N attestation(s) ... verification succeeded`. Record PASS/FAIL per file in UAT report Section 2.

    **Step 3 — Trigger distribute.yml for the rc:**

    The `distribute.yml` workflow fires on `release:published`. Since the rc release is DRAFT, it has not been published — distribute.yml has NOT yet fired. Two options:

    **Option A (recommended for rc verification):** Use `workflow_dispatch` with the `tag` input:
    ```bash
    gh workflow run distribute.yml --field tag=v3.1.0-rc1
    gh run watch --workflow distribute.yml --exit-status
    ```

    Expect:
    - `update-homebrew-tap` job runs — checks out `release-90-test` branch (NOT `main`), updates the formula, pushes to `release-90-test`. Inspect logs: the "Checkout tap repo" step should show `ref: release-90-test`; the "Commit and push" step should show `git push origin HEAD:release-90-test`.
    - `submit-winget` job — **SKIPPED** per D-17 (`if: !contains(github.ref, '-rc')` evaluates to false on the rc tag).

    **Option B:** Temporarily publish the draft (un-draft it) to fire the natural release:published trigger. Only do this if workflow_dispatch is blocked for some reason — publishing even briefly makes the rc assets publicly visible until you re-draft.

    **Step 4 — Confirm tap branch received the update and main did NOT:**
    ```bash
    cd /tmp/homebrew-agenthub  # the clone from pre-flight
    git fetch origin
    git log origin/main..origin/release-90-test --oneline
    # Expected: 1 new commit on release-90-test (the agenthub version bump)

    git log --oneline -5 origin/main
    # Expected: last commit should predate the rc push (main untouched)
    ```

    If `main` received a commit, D-16 failed. Diagnose distribute.yml logs for the `git push` step — the refspec must be `HEAD:"$BRANCH"` with `$BRANCH=release-90-test`.

    **Step 5 — Confirm submit-winget was skipped:**
    ```bash
    gh run view <distribute-run-id> --json jobs --jq '.jobs[] | {name, conclusion}'
    # Expected: update-homebrew-tap: success, submit-winget: skipped
    ```

    **Step 6 — Fill out UAT report Section 3 (distribute.yml run) and Section 4 (cleanup actions).**

    **Step 7 — Cleanup (D-14 + D-16 hygiene):**
    Decide, in consultation with the user, ONE of:

    a. **Phase 90 accepted — promote to real release:** Do NOT delete the draft. Do NOT rewind release-90-test. Instead: cut a new real tag `v3.1.0` after this plan completes, which triggers the same pipeline with `draft: false` (main tap path). This is the happy path.

    b. **Phase 90 accepted but iteration planned — preserve rc artifacts:** Leave the draft release in place for further UAT (Gatekeeper test, install testing), delete later. Leave release-90-test branch for now.

    c. **Phase 90 rejected — full cleanup:** Delete the draft release and rewind release-90-test:
    ```bash
    gh release delete v3.1.0-rc1 --yes --cleanup-tag
    # (cleanup-tag removes the git tag too)

    cd /tmp/homebrew-agenthub
    git push origin --delete release-90-test
    # Or rewind:
    # git push origin +main:release-90-test  # force-move release-90-test to current main
    ```

    **User must explicitly choose (a), (b), or (c).** Record choice in UAT report Section 4.

    **Step 8 — Sign off UAT report Section 5.** The human reviewer fills in name + date + PASS/FAIL.
  </how-to-verify>
  <resume-signal>
    Type "UAT complete: PASS — chose option <a|b|c>" or "UAT complete: FAIL — <reason>".

    On PASS: Phase 90's SC-4 is satisfied. On FAIL: the orchestrator may route back to plan-gaps for remediation.
  </resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Developer tag push → release.yml | The rc tag authorizes the full pipeline execution including signing material access. |
| Draft release → public visibility | `draft: true` is the seal between internal rc artifacts and public distribution. Publishing the draft makes them public. |
| rc tap branch → main tap branch | D-16 prevents rc formula updates from reaching Homebrew users. A mistake here would serve an unsigned or pre-release binary to tap users. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-90-34 | Tampering | rc artifacts shipped publicly by mistake | mitigate | `draft: contains(github.ref, '-rc')` gate in publish step + hyphen-anchor; human verification in Task 2 Step 5; workflow_dispatch option for distribute.yml avoids triggering the release:published event prematurely. |
| T-90-35 | Information Disclosure | Draft release visible to non-collaborators | accept | GitHub draft releases are visible only to repo collaborators via the API; public-facing release listing hides them. Not end-user-visible. |
| T-90-36 | Tampering | Attacker intercepts tag push and substitutes commit | mitigate | Tags are local git operations pushed over authenticated HTTPS/SSH. Force-push auditability via `git log --reflog`. Branch protection on main includes tag protection where applicable. |
| T-90-37 | Repudiation | UAT report lost or unsigned | accept | UAT report is committed to `.planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md`; git history is the authoritative record. Human sign-off is a prose field; not a crypto attestation — acceptable for this process step. |
| T-90-38 | Elevation of Privilege | Environment protection rules bypassed due to environment move (pre-flight Step 3 catches) | mitigate | Pre-flight audit of `release` environment rules in Task 1 Step 3. If a required-reviewer rule existed for `build-macos` and was not replicated on `sign-macos`, that's an auditable deviation surfaced before tag cut. |
| T-90-39 | Denial of Service | Sigstore public-good instance unavailable during attestation | accept | Sigstore is HA but has had outages. If attestation step fails, release.yml fails and the rc doesn't publish — fail-closed. Retry by re-running the workflow after Sigstore recovers. |
| T-90-40 | Tampering | WinGet review queue sees rc submission (if D-17 guard broken) | mitigate (multiple layers) | D-17 is a workflow-level `if:` gate + hyphen-anchored pattern. Task 3 Step 5 explicitly checks `gh run view` conclusion for submit-winget; any non-skipped conclusion on an rc run is a deviation requiring remediation. |

**Residual risk:**
- **Attestation verification depends on Sigstore public-good instance.** In the unlikely event that Sigstore is offline at UAT time, external `gh attestation verify` fails. Retry later. This is not a Phase 90 regression; it's a dependency of the SLSA L2 posture we're establishing.
- **Draft release cleanup is manual.** If the reviewer forgets to delete a failed rc draft, it remains visible to collaborators (but not end users). Not a security issue; hygiene only.
- **release-90-test branch pollution over time.** Multiple rc cuts create multiple commits on the test branch. Before the real `v3.1.0` release, the test branch should be rewound or deleted per Task 3 Step 7. Documented in 90-TAP-BRANCH-SETUP.md.
</threat_model>

<verification>
Phase 90 SC-4 is satisfied when:
1. `gh release view v3.1.0-rc1 --json isDraft --jq .isDraft` returns `true`
2. `gh attestation verify <file> --owner scottkw` returns "verification succeeded" for all 6 release assets
3. `git log origin/main..origin/release-90-test --oneline` on scottkw/homebrew-agenthub shows the rc formula update on release-90-test only
4. `gh run view <distribute-run-id> --json jobs` shows submit-winget: skipped
5. `.planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md` exists with human sign-off
</verification>

<success_criteria>
- v3.1.0-rc1 draft release exists with 6 assets (DMG, installer.exe, .exe, tar.gz, .deb, checksums.txt)
- Every asset passes external `gh attestation verify` (SLSA L2 end-to-end)
- sign-macos log shows internal attestation verification succeeded BEFORE codesign ran (D-04 proof)
- Tap `release-90-test` branch received the formula update; `main` did not (D-16 proof)
- submit-winget job concluded "skipped" on the rc run (D-17 proof)
- UAT report committed with sign-off
- Phase 90 SC-1 (SHA-pin), SC-2 (tool-pin), SC-3 (secret scope split), SC-4 (E2E verification) all satisfied
</success_criteria>

<output>
After completion, create `.planning/phases/90-release-pipeline-hardening/90-06-SUMMARY.md` documenting:
- Tag cut: v3.1.0-rc1 (or actual tag used)
- Workflow run URL + per-job outcomes
- All 6 attestations verified externally
- distribute.yml run behavior (rc-branch targeting + winget skip both confirmed)
- Cleanup option chosen (a/b/c) and actions taken
- UAT report path: `.planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md`
- Phase 90 accepted — ready to mark SEC-09, SEC-10, SEC-11 complete in REQUIREMENTS.md
- Next step: human cuts real v3.1.0 tag (or issues iteration plan via plan-gaps)
</output>
