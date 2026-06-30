---
phase: 156-install-links-distribution
reviewed: 2026-06-27T11:46:06Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - .github/workflows/build.yml
  - TESTING.md
  - frontend/src/components/WelcomeTab.tsx
  - frontend/src/components/__tests__/WelcomeTab.install.test.tsx
  - packaging/winget/FIRST-SUBMISSION-RUNBOOK.md
  - packaging/winget/dry-run-first-submission.sh
  - scripts/install.sh
  - tests/install-sh.test.sh
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 156: Code Review Report

**Reviewed:** 2026-06-27T11:46:06Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Phase 156 fixes the broken install strings in `WelcomeTab.tsx` (replacing the
non-existent `agenthub.dev` domain and bare `winget install agenthub` id),
adds a real POSIX Linux installer (`scripts/install.sh`), a shellcheck gate
(`tests/install-sh.test.sh`) wired into CI, a winget first-submission dry-run
script + operator runbook, and updates `TESTING.md`.

The work is solid and well-verified. I cross-checked every cross-file
assumption against the release pipeline: the install string changes match
the test gates; `install.sh`'s expected tarball name
(`agenthub-${VERSION}-linux-amd64.tar.gz`) and extracted member name
(`agenthub`) match what `release.yml` produces (lines 247-253, including the
`mv build/bin/AgentHub build/bin/agenthub` lowercase rename); the winget
`PackageIdentifier scottkw.agenthub` and `windows-amd64-installer.exe` URL
match `distribute.yml` and the manifest templates; the runbook's hardcoded
`distribute.yml` line numbers (76, 77, 79, 80, 108) are all currently
accurate; `M-25`/`M-26` and `Category N`/`O` extend the existing sequence
without collision; and the test-count delta (501→503) is internally
consistent. I ran both shell test suites and `shellcheck` against all three
new/changed bash scripts — all pass clean.

No BLOCKER-class defects found. The findings below are robustness and
documentation-correctness issues.

## Warnings

### WR-01: Checksum/asset lookups use regex `grep` instead of fixed-string matching

**File:** `scripts/install.sh:77` (also `packaging/winget/dry-run-first-submission.sh:52,129` and `packaging/winget/populate-manifests.sh:39`)
**Issue:** The checksum entry is selected with
`EXPECTED=$(grep "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')`.
`$TARBALL` is `agenthub-vX.Y.Z-linux-amd64.tar.gz` — every `.` is a regex
"any character" metacharacter, so the pattern can match unintended lines in
`checksums.txt`. The blast radius here is a false *negative* (a mismatched
hash would then fail the `[ "$ACTUAL" = "$EXPECTED" ]` check and refuse a
valid install), not a security bypass — but it is still a correctness defect,
and notably the project's own test for this file (`tests/install-sh.test.sh:38`)
deliberately uses `grep -qF`. The same unescaped-`.` pattern appears in the
dry-run's `grep -q "\"${VERSION}\""` (line 129) and populate's version grep.
**Fix:**
```sh
# install.sh:77
EXPECTED=$(grep -F "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
```
Apply `-F` to the checksum/asset/version greps in all three scripts.

### WR-02: TESTING.md Suite Manifest omits the new install gate from its Run Command

**File:** `TESTING.md:32` (Section 2 "Suite Manifest", build-script row)
**Issue:** The build-script group's `Count`/`Location` cells were updated to
list two files (`build-script.test.sh, install-sh.test.sh`), but the
`Run Command` cell still shows only `bash tests/build-script.test.sh`, and the
`Guards` cell still reads "Go build + Wails asset embedding" with no mention of
the install.sh shellcheck gate. CLAUDE.md's standing rule makes `TESTING.md`
the canonical regression-suite home; an operator running the documented suite
by hand would silently skip the new INSTALL-01 gate. CI runs it correctly
(`build.yml:64-66`), so this is a documentation-correctness gap, not a CI gap.
**Fix:** Update the `Run Command` cell to include
`bash tests/install-sh.test.sh` (or note "see CI"), and extend the `Guards`
cell to mention the install.sh shellcheck/static-pattern gate.

### WR-03: Root-install branch does not create `/usr/local/bin` before copying

**File:** `scripts/install.sh:102-110`
**Issue:** The non-root branch does `mkdir -p "$INSTALL_DIR"` (line 106), but
the root branch sets `INSTALL_DIR="/usr/local/bin"` with no `mkdir -p`. Under
`set -eu`, if `/usr/local/bin` is absent (minimal/scratch container images),
the `cp` on line 109 fails *after* the download + SHA256 verification have
already succeeded, producing a confusing late-stage error. Standard FHS distros
always have this directory, so likelihood is low, but the asymmetry with the
user branch is a real unhandled edge case.
**Fix:**
```sh
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
    mkdir -p "$INSTALL_DIR"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi
```

## Info

### IN-01: install.sh verifies integrity but not authenticity of the release

**File:** `scripts/install.sh:74-91`
**Issue:** `checksums.txt` is fetched from the same GitHub Releases origin as
the tarball, so the SHA256 check guarantees the download is not *corrupt* but
provides no independent *authenticity* guarantee if that origin were
compromised (the attacker controlling the tarball could also rewrite
`checksums.txt`). HTTPS to github.com mitigates this in practice, and this is
the standard `curl | sh` posture. Worth noting: `release.yml` already publishes
a build-provenance attestation bundle (`attestation-linux.json`) that the
installer does not consume. Not actionable for v1, but a candidate hardening
step if supply-chain assurance becomes a requirement.
**Fix:** (Optional) Document the integrity-vs-authenticity scope, or add
`gh attestation verify` as an opt-in step.

### IN-02: Runbook instructs editing distribute.yml by line number

**File:** `packaging/winget/FIRST-SUBMISSION-RUNBOOK.md:164` (Step 6b)
**Issue:** The runbook tells the operator to "remove line 76
(`continue-on-error: true`...)". All cited line numbers (76, 77, 79, 80-82, 108)
are accurate against the current `distribute.yml` (verified), but instructing a
by-number edit is fragile: any future insertion above the `submit-winget` job
shifts these references, and "remove line 76" could then delete the wrong line.
The accompanying before/after YAML block mitigates this.
**Fix:** Prefer content-based instruction ("delete the
`continue-on-error: true` line from the `submit-winget:` job") and treat the
line numbers as advisory.

### IN-03: TESTING.md Section 2 running narrative not appended for Phase 156

**File:** `TESTING.md:29` (Section 2 narrative note block)
**Issue:** The per-phase running commentary in Section 2 ends at the Phase
155-05 entry ("...366 Go / 501 total") and was not extended with a Phase 156
note, even though the manifest counts in the table were updated (501→503) and
the traceability rows + manual M-25/M-26 items were added. This is a minor
deviation from the documentation pattern the standing convention establishes.
**Fix:** Append a Phase 156 note recording vitest +1
(`WelcomeTab.install.test.tsx`) and build-script +1 (`install-sh.test.sh`),
reaching 503 total.

---

_Reviewed: 2026-06-27T11:46:06Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
