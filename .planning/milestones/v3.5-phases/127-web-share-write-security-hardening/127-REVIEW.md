---
phase: 127-web-share-write-security-hardening
reviewed: 2026-06-15
depth: focused (orchestrator direct inspection — security audit phase)
files_reviewed: 1
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: clean
---

# Phase 127: Code Review

This phase is ~70% audit/test/doc. The only net-new production code is the `denylistCheck` hardening in `internal/files/sandbox.go` (all other plans are tests, the `127-SECURITY.md` audit artifact, and an e2e cell). The orchestrator directly inspected the `denylistCheck` diff (b08c14a..HEAD).

## Assessment — `denylistCheck` (sandbox.go) — CLEAN

- **Case-fold (SEC-02):** `strings.ToLower` applied to both the base-name switch and the dir-prefix `relSlash` comparison. Correct and safe — all protected names are ASCII, so ToLower cannot mangle multibyte sequences. Catches `.BASHRC`/`.Bashrc`/`.SSH` on case-insensitive macOS/Windows volumes.
- **macOS daemon config dir (SEC-02):** derived from `os.UserConfigDir()` (returns `~/Library/Application Support` on macOS, `~/.config` on Linux, `%AppData%` on Windows) + `agenthub` subdir. Correctly rebased onto the EvalSymlinks-resolved home (handles macOS `/var` → `/private/var`) before `filepath.Rel`, avoiding a spurious `..` escape. Cycle-free — derived locally, does NOT import `internal/daemon`.
- **Belt-and-suspenders:** static `.ssh/ .claude/ .config/agenthub/` prefixes retained (covers cross-platform copied trees) and the OS-correct config dir appended.
- **Isolation:** change is confined to `denylistCheck`; all four write methods (WriteFileAtomic/Rename/Mkdir/Delete) funnel through it, so one fix protects all. No write-method bodies touched (grep-verified in 127-01).
- **Coverage:** `TestDenylist_CaseVariation`, `TestDenylist_DaemonConfigDir`, `TestSandbox_WritePathSymlinkEscapeBlocked`, the SEC-05 race/interrupted-write tests, and the 60s `FuzzSandboxWrite` gate (0 crashes, 82k execs) all pass.

## IN-01 (info, accepted): case-fold over-protection on case-sensitive volumes
On a case-sensitive Linux volume, a file literally named `.BASHRC` (distinct from `.bashrc`) would now also be blocked. This is an intentional, negligible false-positive — erring toward over-protection on a security denylist is the correct tradeoff. No action.

**Verdict:** No critical or warning findings. The hardening is correct, well-tested, and properly isolated. The capability-escalation audit (127-SECURITY.md) documents all enforcement surfaces and 3 accepted residual risks.
