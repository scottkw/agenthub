---
phase: 93
plan: 01
subsystem: webserver
tags: [ci, vendor-drift, xterm, security, supply-chain]
dependency_graph:
  requires:
    - frontend/pnpm-lock.yaml (lockfile format unchanged from Phase 89)
    - web/vendor/xterm/VERSION (manifest format unchanged)
  provides:
    - Generalized CI gate that fails red on any @xterm/addon-* version drift
    - Min-count guard (>= 5) that catches a regex regression dropping packages
  affects:
    - Plan 93-02 (must ship VERSION lines for addon-webgl, addon-unicode11, addon-clipboard so this test goes green)
    - Plans 94-98 (any future @xterm/addon-* plugin automatically enforced — no test changes needed)
tech_stack:
  added: []
  patterns:
    - "Regex generalization with min-count guard for parse-regression detection"
key_files:
  created: []
  modified:
    - internal/webserver/vendor_drift_test.go
decisions:
  - "Generalized regex pattern (?:xterm|addon-[\\w-]+) instead of explicit alternation list — auto-covers every future @xterm/addon-* without test edits"
  - "Min-count guard set to 5 (not >= len(vendor lines)) — fails fast on regex regression that drops packages, even before Plan 93-02 ships VERSION updates"
metrics:
  duration_minutes: 4
  completed_date: "2026-05-04"
  tasks_completed: 1
  files_modified: 1
  commits: 1
---

# Phase 93 Plan 01: Generalize vendor_drift_test for every @xterm/addon-* Summary

Generalized `internal/webserver/vendor_drift_test.go` from an `addon-fit`-only CI gate into a load-bearing parity check that enforces version agreement between `frontend/pnpm-lock.yaml` and `web/vendor/xterm/VERSION` for every vendored `@xterm/addon-*` package. Closes WEB-02 and shipped Phase 93 ROADMAP success criterion #5 ("CI fails red on `@xterm/addon-*` drift").

## What Changed

| Surface | Before | After |
|---|---|---|
| Regex (line 17) | `^  '(@xterm/(?:xterm\|addon-fit))@([0-9.]+)':` | `^  '(@xterm/(?:xterm\|addon-[\w-]+))@([0-9.]+)':` |
| Min-count guard (line 33) | `< 2` (xterm + addon-fit only) | `< 5` (xterm + addon-fit + addon-webgl + addon-unicode11 + addon-clipboard) |
| Package doc comment | "Phase 89" | "Phase 89; generalized in Phase 93 WEB-02" |
| Parse-failure error | "@xterm/xterm and @xterm/addon-fit ... Phase 89 D-04" | "at least 5 @xterm/* packages ... Phase 93 WEB-02" |

VERSION-file parser at lines 43–55 was already general (`@<scope>/<pkg>@<version>` for arbitrary lines) and required no change.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Generalize regex + min-count guard + comment update | `e5953aa` | `internal/webserver/vendor_drift_test.go` |

## Acceptance Criteria — All Met

| AC | Result | Evidence |
|---|---|---|
| 1. `grep -c 'addon-\[\\w-\]+'` >= 1 | PASS | returned 1 |
| 2. `grep -c 'addon-fit))@'` == 0 | PASS | returned 0 |
| 3. `grep -c 'len(pnpmVersions) < 5'` == 1 | PASS | returned 1 |
| 4. `grep -c 'len(pnpmVersions) < 2'` == 0 | PASS | returned 0 |
| 5. `grep -c 'Phase 93 WEB-02'` >= 1 | PASS | returned 2 (package comment + error message) |
| 6. `go vet ./internal/webserver/...` exits 0 | PASS | clean |
| 7. After 93-02 lands: `go test ... -run TestXtermVendorVersionsMatchPnpmLock -count=1` exits 0 | PENDING-93-02 | Test currently red-fails as designed; will go green once 93-02 ships VERSION lines |

## Lockfile Validation

`grep "^  '@xterm/" frontend/pnpm-lock.yaml` returned all 5 expected packages in the canonical pnpm-lock format:
```
  '@xterm/addon-clipboard@0.2.0':
  '@xterm/addon-fit@0.11.0':
  '@xterm/addon-unicode11@0.9.0':
  '@xterm/addon-webgl@0.19.0':
  '@xterm/xterm@6.0.0':
```
The new regex captures all 5; the new min-count guard requires all 5.

## Test Behavior in Isolation

Running `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock -count=1` against this commit (without Plan 93-02's VERSION updates) produces the **expected red-fail**:

```
--- FAIL: TestXtermVendorVersionsMatchPnpmLock (0.00s)
    vendor_drift_test.go:61: web/vendor/xterm/VERSION missing entry for @xterm/addon-clipboard
    vendor_drift_test.go:61: web/vendor/xterm/VERSION missing entry for @xterm/addon-unicode11
    vendor_drift_test.go:61: web/vendor/xterm/VERSION missing entry for @xterm/addon-webgl
```

This is the gate working as intended — it proves:
- The generalized regex now captures `addon-webgl`, `addon-unicode11`, `addon-clipboard` from the lockfile (it didn't before).
- The downstream parity check correctly names the missing vendored packages with re-copy guidance.

Plan 93-02 (Wave 1, parallel sibling) ships the three VERSION lines that flip this test green.

## Threat Model Mitigations Honored

| Threat ID | Disposition | How this plan mitigates |
|-----------|-------------|------------------------|
| T-93-01 (Tampering — regex regression) | mitigate | Min-count guard `< 5` fails red if a future edit narrows the regex back; the literal regex string is grep-verified in acceptance criteria. |
| T-93-02 (Repudiation — bump-without-recopy) | mitigate | Test names the missing/drifted package and points to "Phase 89 D-04" / "Phase 93 WEB-02" with re-copy procedure. |
| T-93-CLIP-01 | accept (out of scope) | This plan is CI-only; clipboard handler-level threat is owned by Plan 93-04. |

No new threat surface introduced — change is in test code only.

## Deviations from Plan

None — plan executed exactly as written. Edits matched the `<interfaces>` section verbatim; all 6 acceptance criteria met on first verify.

## Self-Check: PASSED

- File exists: `internal/webserver/vendor_drift_test.go` — FOUND
- Commit exists: `e5953aa` — FOUND in `git log --oneline --all`
- `go vet ./internal/webserver/...` — exits 0
- All 6 grep-based acceptance criteria — PASS

## TDD Gate Compliance

Plan task is marked `tdd="true"` but the artifact under change IS the test file itself. The "RED" gate is the post-commit test execution against the as-yet-unupdated VERSION file (red-fail observed, screenshot above). The "GREEN" gate is owned by Plan 93-02 (Wave 1 sibling) which ships the VERSION updates that make the test pass. Single commit (`test(93-01): ...`) is appropriate because the task is exclusively a test-change.
