---
phase: 156-install-links-distribution
verified: 2026-06-27T07:00:00Z
status: human_needed
score: 2/3 must-haves verified
behavior_unverified: 1
overrides_applied: 0
behavior_unverified_items:
  - truth: "Following the Linux curl command executes install.sh, detects correct architecture, verifies SHA256 from checksums.txt, and installs the agenthub binary to a standard path (TESTING.md M-25)"
    test: "On a clean amd64 Linux machine or docker run --rm ubuntu:22.04 bash: run `curl -fsSL https://raw.githubusercontent.com/scottkw/agenthub/main/scripts/install.sh | sh`, then confirm (a) the 'SHA256 verified.' line appears, (b) the binary is installed to /usr/local/bin/agenthub or ~/.local/bin/agenthub, (c) `agenthub --help` exits 0, (d) re-run is idempotent, (e) non-root PATH warning printed when ~/.local/bin absent from PATH."
    expected: "Installation completes without error; binary is runnable; SHA256 check passes against the release checksums.txt."
    why_human: "The shellcheck/static-pattern CI gate (SC-4) and POSIX syntax check (sh -n) prove code correctness but do not exercise the actual network fetch → SHA256 compare → binary copy state transition. Checking this requires a real GitHub release asset and a clean Linux runtime."
human_verification:
  - test: "M-25 — Linux install end-to-end on a clean amd64 box (or docker run --rm ubuntu:22.04)"
    expected: "curl -fsSL https://raw.githubusercontent.com/scottkw/agenthub/main/scripts/install.sh | sh completes; 'SHA256 verified.' appears; binary installed to standard path; agenthub --help exits 0; idempotent re-run succeeds; non-root PATH warning present when applicable. See TESTING.md M-25 for the full checklist."
    why_human: "Requires a live GitHub release tarball + checksums.txt and a real Linux runtime. Automated checks (shellcheck, sh -n, static patterns) confirm code structure but cannot confirm the SHA256 download-verify-install state transition succeeds end-to-end."
  - test: "M-26 step 3 — Confirm WINGET_TOKEN secret is provisioned in the repo"
    expected: "`gh secret list | grep WINGET_TOKEN` returns a row for WINGET_TOKEN (classic PAT, public_repo scope only). The live submission in Steps 4–7 of the runbook is NOT a phase blocker; the WINGET_TOKEN check alone is the phase gate."
    why_human: "TESTING.md M-26 labels WINGET_TOKEN provisioning as a phase gate. `gh secret list` returned empty output during automated verification — could not confirm presence or absence of the secret. Requires operator to run `gh secret list | grep WINGET_TOKEN` in the scottkw/agenthub repo context."
---

# Phase 156: Install Links & Distribution Verification Report

**Phase Goal:** The Welcome screen install instructions are accurate and the Linux curl + Windows winget distribution paths work end-to-end.
**Verified:** 2026-06-27T07:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC-1 | Linux curl command executes install.sh, detects correct arch, verifies SHA256, installs binary — verified and documented in TESTING.md manual checklist | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Code correct (shellcheck 11/11 PASS, sh -n exit 0, all static patterns present). M-25 documented. End-to-end clean-box install not exercised by any automated test — routes to M-25 human verification. |
| SC-2 | Welcome screen shows `winget install scottkw.agenthub` and `github.com/scottkw/agenthub`; no placeholder URLs or wrong IDs | ✓ VERIFIED | WelcomeTab.tsx line 42 (raw.githubusercontent.com URL), line 46 (winget install scottkw.agenthub), line 54 (github.com/scottkw/agenthub). Source-gate test 6/6 green. Negative assertions confirm no agenthub.dev, no bare winget id, no agenthub-dev org. |
| SC-3 | distribute.yml first-submission code path correct when WINGET_FIRST_SUBMISSION=true; repo-side automation complete and verifiable by dry-run; phase ships while Microsoft review pending | ✓ VERIFIED | distribute.yml lines 108–131: WINGET_FIRST_SUBMISSION flag controls new vs update path; scottkw.agenthub id correct. Dry-run produced packaging/winget/output/4.0/ with 3 valid YAML manifests (python3 yaml.safe_load PASS). PackageIdentifier: scottkw.agenthub, windows-amd64-installer.exe URL present. WINGET_TOKEN provision → human check (M-26 step 3). |

**Score:** 2/3 truths verified (1 present, behavior-unverified)

### Derived Plan Truths

All PLAN frontmatter truths below support the three ROADMAP SCs above.

**Plan 01 (INSTALL-01/02 — Welcome screen strings):**

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| P1-T1 | Linux URL is raw.githubusercontent.com install.sh | ✓ VERIFIED | WelcomeTab.tsx line 42 confirmed; WelcomeTab.install.test.tsx assertion 1/6 PASS |
| P1-T2 | Windows command is `winget install scottkw.agenthub` | ✓ VERIFIED | WelcomeTab.tsx line 46 confirmed; test 3/6 PASS |
| P1-T3 | Repo link is `github.com/scottkw/agenthub` | ✓ VERIFIED | WelcomeTab.tsx line 54 span (not anchor); test 5/6 PASS |
| P1-T4 | No placeholder URL or wrong package id | ✓ VERIFIED | Negative assertions (agenthub.dev, bare winget id, agenthub-dev org) all PASS; WelcomeTab.test.tsx regression fixed (commit 2a036b98) |

**Plan 02 (INSTALL-01 — installer script):**

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| P2-T1 | install.sh detects arch, resolves tag, downloads + SHA256-verifies, installs binary | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Code present at scripts/install.sh (lines 39–96); shellcheck PASS; all 8 static patterns PASS. State transition (actual install on Linux) not exercised — M-25 required |
| P2-T2 | install.sh hard-aborts on SHA256 mismatch or missing checksum entry | ✓ VERIFIED | Lines 77–81: empty EXPECTED exits 1; lines 86–91: mismatch prints Expected/Actual and exits 1; tar xzf (line 96) only reached after both guards pass |
| P2-T3 | install.sh is POSIX sh: set -eu, no bashisms, command -v guards | ✓ VERIFIED | `sh -n scripts/install.sh` exits 0; `shellcheck --shell=sh` PASS; set -eu on line 5; need_cmd uses command -v |
| P2-T4 | Installs to /usr/local/bin (root) or ~/.local/bin (non-root) with PATH warning | ✓ VERIFIED | Lines 102–124: id -u check, mkdir -p non-root branch, PATH case guard with warning text |
| P2-T5 | Shellcheck + static-pattern gate runs in CI against install.sh | ✓ VERIFIED | build.yml lines 64–66: "Run install.sh shellcheck gate" step with ubuntu-latest guard; runs bash tests/install-sh.test.sh |

**Plan 03 (INSTALL-03 — winget first-submission):**

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| P3-T1 | Winget manifest-generation path exercised end-to-end with real checksums producing valid YAML | ✓ VERIFIED | packaging/winget/output/4.0/ exists with 3 files; python3 yaml.safe_load parses all 3 without error |
| P3-T2 | Generated manifests carry PackageIdentifier scottkw.agenthub, real version, windows installer URL | ✓ VERIFIED | scottkw.agenthub.installer.yaml: PackageIdentifier: scottkw.agenthub, PackageVersion: "4.0", InstallerUrl contains windows-amd64-installer.exe |
| P3-T3 | Operator runbook documents 7 live submission steps including token scope, PR shepherd, post-acceptance reset | ✓ VERIFIED | FIRST-SUBMISSION-RUNBOOK.md: all 7 steps present; Step 2 mandates public_repo scope only (T-156-07); Step 6 requires removing continue-on-error (T-156-08); explicit "not a phase blocker" header at top |
| P3-T4 | Live microsoft/winget-pkgs submission is NOT a phase-completion blocker | ✓ VERIFIED | Runbook header: "steps after the dry-run are operator follow-ups, NOT phase-completion blockers"; M-26 steps 4–6 marked non-blocker; SC-3 states "phase ships even while Microsoft's catalog PR review is externally pending" |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|---------|--------|---------|
| `frontend/src/components/WelcomeTab.tsx` | Corrected curl URL, winget id, repo link | ✓ VERIFIED | Lines 42/46/54 corrected; line 54 is `<span>` not `<a>` |
| `frontend/src/components/__tests__/WelcomeTab.install.test.tsx` | 6-assertion source-gate | ✓ VERIFIED | 6 assertions, 6/6 PASS; readFileSync pattern (no render) |
| `scripts/install.sh` | POSIX sh installer, executable | ✓ VERIFIED | 126 lines; shebang #!/usr/bin/env sh; set -eu; chmod 755 (-rwxr-xr-x) |
| `tests/install-sh.test.sh` | Shellcheck gate with SC-1–4, executable | ✓ VERIFIED | 11/11 PASS including SC-2 (shellcheck installed locally); -rwxr-xr-x |
| `.github/workflows/build.yml` | New CI step guarded ubuntu-latest | ✓ VERIFIED | Lines 64–66: step present, if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest' |
| `TESTING.md` | vitest 127, build-script 2, Total 503; INSTALL-01/02 rows; M-25, M-26 | ✓ VERIFIED | vitest: 127, build-script: 2, Total: 503; Section 4 rows for INSTALL-01 and INSTALL-02; M-25 in Category N; M-26 in Category O |
| `packaging/winget/dry-run-first-submission.sh` | Dry-run helper, executable, 5 steps | ✓ VERIFIED | 151 lines; set -euo pipefail; v-strip (${TAG#v}); python3 yaml.safe_load; PackageIdentifier assert; -rwxr-xr-x |
| `packaging/winget/FIRST-SUBMISSION-RUNBOOK.md` | 7-step runbook with non-blocker statement | ✓ VERIFIED | All 7 steps present; least-privilege token scope documented; continue-on-error removal in Step 6b; non-blocker header at top |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| WelcomeTab.tsx line 42 | scripts/install.sh on main | raw.githubusercontent.com URL | ✓ WIRED | URL in component matches script path; script exists at repo root |
| WelcomeTab.install.test.tsx | WelcomeTab.tsx | readFileSync('../WelcomeTab.tsx') | ✓ WIRED | Test reads source file at relative path ../../components/WelcomeTab.tsx; 6/6 assertions exercised |
| dry-run-first-submission.sh | populate-manifests.sh | bash "${SCRIPT_DIR}/populate-manifests.sh" | ✓ WIRED | SCRIPT_DIR resolved at runtime; VERSION stripped of v (${TAG#v}); output/4.0/ produced |
| distribute.yml submit-winget | scottkw.agenthub catalog | WINGET_FIRST_SUBMISSION → new vs update branch | ✓ WIRED | Lines 108–131: flag controls wingetcreate new / wingetcreate update; scottkw.agenthub in --package-identifier |
| install.sh | checksums.txt + tarball | SHA256 verify before tar xzf | ✓ WIRED | Lines 77–91 (verify) precede line 96 (extract); ordering confirmed |

### Data-Flow Trace (Level 4)

WelcomeTab.tsx install strings are static literals (not dynamic data from API). No data-flow trace required for the new strings. The `version` state variable (GetVersion() → Wails call) is pre-existing, out-of-scope functionality unchanged by this phase.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| WelcomeTab source-gate (6 assertions) | `cd frontend && pnpm vitest run src/components/__tests__/WelcomeTab.install.test.tsx` | 6/6 Tests passed | ✓ PASS |
| install.sh shellcheck gate (11 checks) | `bash tests/install-sh.test.sh` | 11/11 passed | ✓ PASS |
| install.sh POSIX syntax | `sh -n scripts/install.sh` | exit 0 | ✓ PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | "OK: all traceability paths exist" exit 0 | ✓ PASS |
| Winget manifest YAML validity | `python3 -c "import yaml,glob; [yaml.safe_load(open(f)) for f in glob.glob(...)]"` | 3/3 manifests parsed; all valid | ✓ PASS |
| Installer manifest assertions | grep PackageIdentifier + windows-amd64-installer.exe in output/4.0/scottkw.agenthub.installer.yaml | Both present | ✓ PASS |
| End-to-end Linux install on clean box | M-25 (requires real Linux box or Docker) | — | ? SKIP — routes to human verification |

### Probe Execution

No probe-*.sh files declared or present for this phase. Step 7c: SKIPPED (no probes).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| INSTALL-01 | 156-01, 156-02 | Linux install command works end-to-end — scripts/install.sh with arch detect, SHA256, install; TESTING.md M-25 | ✓ SATISFIED (code) / ⚠️ M-25 pending | WelcomeTab.tsx URL correct; install.sh present and shellcheck-clean; CI gate wired; M-25 documented. Clean-box execution = human item. |
| INSTALL-02 | 156-01 | Welcome screen shows correct winget id and repo link | ✓ SATISFIED | WelcomeTab.tsx lines 46/54 correct; 6/6 source-gate assertions green |
| INSTALL-03 | 156-03 | Winget first submission via distribute.yml; dry-run proven; live PR gated on Microsoft review | ✓ SATISFIED (code + dry-run) | distribute.yml submit-winget job correct; dry-run output/4.0/ 3 valid YAMLs; runbook present; TESTING.md M-26 documented. WINGET_TOKEN provision = human check per M-26 step 3. |

No orphaned requirements: INSTALL-01, INSTALL-02, INSTALL-03 all claimed by plans and covered.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| scripts/install.sh | 77 | `grep "${TARBALL}"` without `-F` — dots in tarball name are regex wildcards | ⚠️ Warning | Blast radius: false negative (wrong hash → mismatch → abort), NOT a security bypass. SHA256 guard still fires. Code review WR-01. No BLOCKER. |
| TESTING.md | 31 | build-script `Run Command` cell still shows only `bash tests/build-script.test.sh` (not the new install-sh gate) | ⚠️ Warning | Documentation gap; CI correctly runs both gates. Operator running by hand would miss install-sh.test.sh. Code review WR-02. |
| scripts/install.sh | 102–104 | Root branch sets `INSTALL_DIR="/usr/local/bin"` without `mkdir -p` (non-root branch does mkdir -p) | ⚠️ Warning | Fails on minimal containers without /usr/local/bin. Standard FHS distros always have this dir — low-probability edge case. Code review WR-03. |

No TBD/FIXME/XXX debt markers found in any phase-modified file. No stub/placeholder content in new implementations.

### Human Verification Required

#### 1. M-25 — Linux Install End-to-End (SC-1 behavioral verification)

**Test:** On a clean amd64 Linux machine or `docker run --rm -it ubuntu:22.04 bash`:
```bash
apt-get update && apt-get install -y curl  # if Docker
curl -fsSL https://raw.githubusercontent.com/scottkw/agenthub/main/scripts/install.sh | sh
```
Then verify:
- "Installing agenthub …" and "SHA256 verified." appear in output
- Binary installed to `/usr/local/bin/agenthub` (root) or `~/.local/bin/agenthub` (non-root)
- `agenthub --help` exits 0 (or shows expected usage)
- Idempotent re-run completes without error
- Non-root: PATH warning printed when `~/.local/bin` absent from `$PATH`

**Expected:** All 5 checklist items pass. Full procedure in TESTING.md M-25.

**Why human:** The shellcheck gate and static-pattern checks (SC-4) confirm code structure but do not exercise the actual network fetch → SHA256 compare → binary copy state transition. Requires a live release tarball and a real Linux runtime.

#### 2. M-26 Step 3 — WINGET_TOKEN Secret Provisioned

**Test:** In the scottkw/agenthub repo context:
```bash
gh secret list | grep WINGET_TOKEN
```
Confirm a row is returned showing WINGET_TOKEN (classic PAT, `public_repo` scope only per FIRST-SUBMISSION-RUNBOOK.md Step 2).

**Expected:** `WINGET_TOKEN` appears in `gh secret list` output.

**Why human:** `gh secret list` returned no output during automated verification — could not confirm secret presence or absence. This is labeled a phase gate in TESTING.md M-26 step 3. The live submission (M-26 steps 4–6) is explicitly a non-blocker; only the token provisioning is the phase gate item.

**Note:** Steps 4–7 of M-26 (trigger workflow, shepherd PR, post-acceptance reset, Windows verify) are documented in FIRST-SUBMISSION-RUNBOOK.md and explicitly NOT phase-completion blockers. Phase 156 ships while Microsoft's catalog PR review is externally pending.

### Gaps Summary

No gaps. All code-verifiable must-haves are VERIFIED. Status is `human_needed` because:

1. **SC-1 behavior-unverified:** The Linux installer code is correct and CI-gated, but the end-to-end clean-box installation (state transition) cannot be confirmed without a real Linux runtime. M-25 is documented and ready to execute.

2. **M-26 step 3 UNCERTAIN:** WINGET_TOKEN secret provision could not be confirmed via automated check (`gh secret list` returned empty). Requires operator to confirm in repo context.

Three code review warnings (WR-01 regex grep, WR-02 TESTING.md run-command cell, WR-03 root mkdir) are non-blocking documentation/robustness issues. The orchestrator-noted regression (stale WelcomeTab.test.tsx assertions) was fixed and committed (2a036b98) before this verification ran.

---

_Verified: 2026-06-27T07:00:00Z_
_Verifier: Claude (gsd-verifier)_
