---
phase: 156-install-links-distribution
plan: "02"
subsystem: scripts
tags: [install-sh, linux-installer, posix-sh, shellcheck, ci-gate, tdd]
status: complete
completed: "2026-06-27"
duration: "~30 minutes"
tasks_completed: 3
files_changed: 4

dependency_graph:
  requires:
    - 156-01 (WelcomeTab.tsx curl URL corrected to raw.githubusercontent.com path)
  provides:
    - scripts/install.sh (POSIX sh Linux installer)
    - tests/install-sh.test.sh (shellcheck + static-pattern gate)
    - .github/workflows/build.yml (install.sh shellcheck gate step)
    - TESTING.md (build-script count 2, Total 503, M-25, INSTALL-01 traceability row)
  affects:
    - INSTALL-01 (Linux curl install command now resolves to a real script)
    - CI ubuntu-latest job (new install.sh shellcheck gate step)

tech_stack:
  added: []
  patterns:
    - POSIX sh installer (set -eu, no bashisms, command -v guards)
    - Build-script suite gate (bash test, pass/fail counter, shellcheck + static-pattern asserts)
    - TDD RED→GREEN: test gate written first (failing), then implementation to make it pass

key_files:
  created:
    - scripts/install.sh
    - tests/install-sh.test.sh
  modified:
    - .github/workflows/build.yml
    - TESTING.md

decisions:
  - "POSIX sh (set -eu, no set -o pipefail, no [[, no local) to survive dash on Debian/Ubuntu defaults"
  - "grep+sed for GitHub API tag_name extraction (no jq dependency)"
  - "sha256sum primary, shasum -a 256 fallback — covers all Linux distros and macOS"
  - "SHA256 verify precedes tar xzf extraction (T-156-01, T-156-03 threat mitigations)"
  - "EXPECTED hash is required-non-empty — missing entry in checksums.txt hard-aborts (T-156-03)"
  - "install to /usr/local/bin (root) or ~/.local/bin (non-root) with PATH warning (locked design)"
  - "CI gate added to ubuntu-latest only (shellcheck pre-installed on those runners)"
---

# Phase 156 Plan 02: Linux Installer (scripts/install.sh) Summary

POSIX-sh Linux installer for agenthub with SHA256 verification gate, shellcheck-clean CI gate, and TESTING.md registration.

## One-liner

POSIX-sh installer that arch-detects, resolves the latest GitHub release tag, downloads + SHA256-verifies the tarball (hard-abort on mismatch or missing entry), extracts and installs the `agenthub` binary, gated in CI via shellcheck.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Add install.sh shellcheck + static-pattern gate | 60e36709 | tests/install-sh.test.sh |
| 2 (GREEN) | Write the POSIX-sh Linux installer | 3df0ddfc | scripts/install.sh |
| 3 | Wire CI gate and register in TESTING.md | ad3b525c | .github/workflows/build.yml, TESTING.md |

## What Was Built

- **tests/install-sh.test.sh** — shellcheck gate with 4 checks: SC-1 (file exists, non-empty), SC-2 (shellcheck -S warning --shell=sh; SKIP when shellcheck absent), SC-3 (bash -n syntax fallback), SC-4 (8 static pattern assertions: uname -m, x86_64, sha256sum, /usr/local/bin, .local/bin, trap, GitHub Releases API URL, mismatch error message). Uses pass/fail counter + `[ $FAIL -eq 0 ] || exit 1` pattern from existing `build-script.test.sh`.

- **scripts/install.sh** — POSIX sh installer (shebang `#!/usr/bin/env sh`, `set -eu`):
  - Preflight: `need_cmd curl` and `need_cmd tar` with clear error messages
  - SHA command selection: `sha256sum` → `shasum -a 256` → error-exit
  - Arch detect: `uname -m` case statement; non-x86_64 hard-fails with releases page link
  - Version resolve: GitHub API `/releases/latest`, grep+sed `tag_name` (no jq)
  - Downloads `agenthub-${VERSION}-linux-amd64.tar.gz` + `checksums.txt` to `mktemp -d` with `trap` cleanup
  - SHA256 verify: `grep + awk '{print $1}'` extracts expected hash; missing entry hard-aborts; mismatch prints Expected/Actual and exits 1
  - `tar xzf` extraction only after verification passes
  - Installs to `/usr/local/bin` (root) or `~/.local/bin` with `mkdir -p` (non-root), `chmod 755`
  - PATH warning when `~/.local/bin` absent from `$PATH`

- **.github/workflows/build.yml**: New step "Run install.sh shellcheck gate" adjacent to "Run build script tests", guarded `if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'`.

- **TESTING.md**: build-script count 1 → 2, Total 502 → 503; INSTALL-01 traceability row (`tests/install-sh.test.sh`, build-script suite); M-25 manual item (Linux install end-to-end with Docker alternative).

## Verification Results

- `bash tests/install-sh.test.sh` — 11/11 PASS (SC-2 shellcheck also passes since shellcheck is installed locally)
- `sh -n scripts/install.sh` — POSIX parse clean (exit 0)
- `bash tests/check-traceability-paths.sh` — exits 0, all traceability paths verified
- SHA256 verification (line 86) precedes `tar xzf` extraction (line 96) in scripts/install.sh — confirmed by grep -n
- CI step confirmed present: `grep -q 'install.sh shellcheck gate' .github/workflows/build.yml`

## Deviations from Plan

None — plan executed exactly as written. The RESEARCH blueprint was followed verbatim for both the test file and the installer script. All acceptance criteria met.

## Known Stubs

None. `scripts/install.sh` is a complete, functional installer. It resolves the live GitHub release tag at runtime, so no per-release maintenance is needed. The only non-automated verification remaining is M-25 (clean Linux box end-to-end install), which requires a live GitHub release asset and a real Linux machine or Docker container.

## Threat Flags

No new threat surface beyond what is documented in the plan's threat model:
- T-156-01 mitigated: SHA256 verify precedes tar xzf; mismatch hard-aborts
- T-156-02 mitigated: curl -fsSL (HTTPS only, -f fails on HTTP errors); SHA256 gate holds even if CDN is compromised
- T-156-03 mitigated: EXPECTED hash is required-non-empty; missing entry aborts before extraction

No new endpoints, auth paths, file access patterns, or schema changes introduced beyond those in the plan's trust boundary table.

## Self-Check

## Self-Check: PASSED

All files found on disk:
- FOUND: scripts/install.sh
- FOUND: tests/install-sh.test.sh
- FOUND: .github/workflows/build.yml
- FOUND: TESTING.md
- FOUND: .planning/phases/156-install-links-distribution/156-02-SUMMARY.md

All commits verified in git log:
- FOUND: 60e36709 (test: RED gate)
- FOUND: 3df0ddfc (feat: installer GREEN)
- FOUND: ad3b525c (chore: CI + TESTING.md)
