---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: 04
type: execute
wave: 1
depends_on: []
files_modified:
  - scripts/install.sh
  - tests/install-sh.test.sh
autonomous: true
requirements: [WR-01, WR-03]
must_haves:
  truths:
    - "install.sh matches the tarball name against checksums.txt as a fixed string (dots are literal, not regex wildcards)."
    - "install.sh creates INSTALL_DIR before copying in BOTH the root and non-root branches."
  artifacts:
    - scripts/install.sh
    - tests/install-sh.test.sh
  key_links:
    - "tests/install-sh.test.sh statically asserts the grep -F flag (WR-01) and the root-branch mkdir -p (WR-03)."
  prohibitions:
    - "MUST NOT change the SHA256 integrity verification logic itself — only make the tarball-name match exact and ensure the install dir exists."
    - "MUST NOT alter the non-root branch behavior (it already has mkdir -p)."
---

<objective>
Close the Phase 156 install-script tech debt (WR-01, WR-03) from the v4.1 milestone audit: the checksum-line lookup uses grep without -F (dots in the tarball name act as regex wildcards — a false-negative risk), and the root branch sets INSTALL_DIR=/usr/local/bin without mkdir -p (fails on minimal containers). Both are surgical edits to scripts/install.sh plus static regression assertions in the existing tests/install-sh.test.sh.

Purpose: Harden the Linux distribution path so checksum matching is exact and installation succeeds on minimal/container filesystems.

Output: install.sh WR-01 + WR-03 fixes; install-sh.test.sh extended with two new assertions.

Note: WR-02 (the TESTING.md build-script Run Command cell) is a TESTING.md edit handled in plan 160-05 (sole TESTING.md owner). install-sh.test.sh is an existing registered file (count unchanged), so no §2 manifest edit is needed here.
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-RESEARCH.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-PATTERNS.md
@scripts/install.sh
@tests/install-sh.test.sh
</context>

<tasks>

<task type="auto">
  <name>Task 1: Apply WR-01 (grep -F) and WR-03 (root mkdir -p) to install.sh</name>
  <files>scripts/install.sh</files>
  <read_first>
    - 160-RESEARCH.md lines 187-244 (WR-01 at line 77; WR-03 at lines 102-107 — exact before/after)
    - 160-PATTERNS.md lines 341-370 (both surgical edits)
    - scripts/install.sh lines 52-124 (the checksum-extract line and the id -u install-dir branch)
  </read_first>
  <action>
    WR-01: at the checksum-extract line (~77), add the `-F` flag to grep so the tarball filename is matched as a fixed string (its dots become literal). WR-03: in the root branch of the `id -u` conditional (the branch setting INSTALL_DIR=/usr/local/bin), add `mkdir -p "$INSTALL_DIR"` mirroring the non-root branch; leave the non-root branch unchanged. Keep the script shellcheck-clean and `sh -n`-valid.
  </action>
  <verify>
    <automated>grep -nE 'grep -F .*checksums' scripts/install.sh && sh -n scripts/install.sh && shellcheck scripts/install.sh && echo OK</automated>
  </verify>
  <acceptance_criteria>
    grep at the checksum line uses -F; the root branch has mkdir -p; `sh -n` and shellcheck both pass; non-root branch unchanged.
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: Extend install-sh.test.sh with WR-01 and WR-03 assertions</name>
  <files>tests/install-sh.test.sh</files>
  <read_first>
    - 160-VALIDATION.md lines 44, 57 (WR-01/WR-03 assertions to add)
    - 160-RESEARCH.md lines 428-430 (WR-01/WR-03 covered by install-sh.test.sh)
    - tests/install-sh.test.sh (existing assertion style — match its harness/helpers)
  </read_first>
  <action>
    Add two static assertions following the file's existing assertion style: (1) WR-01 — the checksum-extract grep in scripts/install.sh uses the -F flag (assert the pattern is present); (2) WR-03 — the root branch of the install-dir conditional contains mkdir -p for INSTALL_DIR (assert both branches create the directory). Keep assertions exact and shellcheck-clean. Do not introduce watch/loop modes.
  </action>
  <verify>
    <automated>bash tests/install-sh.test.sh</automated>
  </verify>
  <acceptance_criteria>
    `bash tests/install-sh.test.sh` passes with the two new assertions covering WR-01 (grep -F) and WR-03 (root mkdir -p).
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| remote tarball + checksums.txt -> local FS | Downloaded release artifacts verified before install |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-160-WR-01 | Tampering | checksum line match (grep) | low | mitigate | `-F` makes the tarball-name match exact (dots literal), removing the false-negative window; the SHA256 hash compare remains the integrity gate. V5 Input Validation. |
| T-160-WR-03 | Denial of Service | root install dir | low | mitigate | mkdir -p ensures /usr/local/bin exists on minimal containers before copy; no privilege change. |
</threat_model>

<verification>
- `sh -n scripts/install.sh` and `shellcheck scripts/install.sh` pass.
- `bash tests/install-sh.test.sh` passes including the two new assertions.
- Non-root branch of install.sh is unchanged in the diff.
</verification>

<success_criteria>
WR-01 and WR-03 closed: exact checksum-name matching and guaranteed install-dir creation in both branches, pinned by static regression assertions.
</success_criteria>

<output>
Create `.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-04-SUMMARY.md` when done.
</output>
