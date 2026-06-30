---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: "04"
subsystem: install
tags: [install, hardening, checksum, posix-sh]
requirements: [WR-01, WR-03]
dependency_graph:
  requires: []
  provides: [WR-01, WR-03]
  affects: [scripts/install.sh, tests/install-sh.test.sh]
tech_stack:
  added: []
  patterns: [POSIX sh, shellcheck, static regression assertions]
key_files:
  created: []
  modified:
    - scripts/install.sh
    - tests/install-sh.test.sh
decisions:
  - "Used grep -cF to count mkdir -p occurrences in WR-03 test rather than positional line checks — more resilient to minor reformatting."
metrics:
  duration: "5m"
  completed: "2026-06-28T03:16:39Z"
  tasks_completed: 2
  tasks_total: 2
status: complete
---

# Phase 160 Plan 04: install.sh Hardening Summary

Hardened the Linux installer against two Phase 156 tech-debt items: exact checksum-name matching (WR-01) and guaranteed install-dir creation in both privilege branches (WR-03), pinned by static regression assertions.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 | Apply WR-01 (grep -F) and WR-03 (root mkdir -p) to install.sh | 8c91485e | scripts/install.sh |
| 2 | Extend install-sh.test.sh with WR-01 and WR-03 assertions | 0fff2dc3 | tests/install-sh.test.sh |

## Changes Made

### scripts/install.sh

**WR-01 (line 77):** Added `-F` flag to the `grep` that extracts the expected checksum line. Before: `grep "${TARBALL}" …`; after: `grep -F "${TARBALL}" …`. Without `-F`, dots in the tarball filename (`agenthub-v1.2.3-linux-amd64.tar.gz`) are regex wildcards and a similarly-named corrupt entry could produce a false match. The SHA256 guard would still catch a bad download, but the grep fix eliminates the ambiguity entirely.

**WR-03 (lines 102-108):** Added `mkdir -p "$INSTALL_DIR"` to the root (`id -u == 0`) branch. The non-root branch already had this guard. On minimal containers (`alpine:latest`, scratch-based images) `/usr/local/bin` is not guaranteed to exist. Both branches are now symmetric.

### tests/install-sh.test.sh

Added two assertions after the existing eight pattern checks:

- **WR-01:** `assert_literal` for the literal string `grep -F "${TARBALL}" "${TMPDIR}/checksums.txt"` — fails if the flag is removed.
- **WR-03:** `grep -cF 'mkdir -p "$INSTALL_DIR"'` count must be ≥ 2 — fails if either branch loses its mkdir.

Test run: **13/13 passed, 0 failed.**

## Verification

- `sh -n scripts/install.sh` — PASS
- `shellcheck -S warning --shell=sh scripts/install.sh` — PASS (clean)
- `bash tests/install-sh.test.sh` — 13/13 PASS
- Non-root branch of install.sh diff: unchanged (only root branch received the `mkdir -p` addition)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — changes are confined to the install script hardening path already modelled in the plan's threat register (T-160-WR-01, T-160-WR-03).

## Self-Check: PASSED

- `scripts/install.sh` exists and contains both fixes
- `tests/install-sh.test.sh` exists and contains WR-01 + WR-03 assertions
- Commit 8c91485e verified in git log
- Commit 0fff2dc3 verified in git log
