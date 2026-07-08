---
phase: 174-dependency-updates-dependabot-hygiene
plan: 02
subsystem: infra
tags: [go-modules, dependabot, coder-websocket, x-term, nfpm, deb-packaging, dependency-hygiene]

# Dependency graph
requires:
  - phase: 174-01
    provides: 4 low-risk CI-action Dependabot bumps already applied on v4.2-funnel-sharing
provides:
  - coder/websocket bumped 1.8.14 -> 1.8.15, gated on webserver+relay race tests
  - golang.org/x/term bumped 0.43.0 -> 0.44.0, gated on full go build/vet/test -short ./...
  - goreleaser/nfpm/v2 bumped 2.46.3 -> 2.47.0, gated on a live nfpm-produced .deb
  - go-webview2 confirmed still pinned at v1.0.19 (Windows-build guard intact)
affects: [174-03, release-pipeline, dependabot-hygiene]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Committed Task 1 (coder/websocket) and Task 2 (x/term + nfpm) as two separate atomic commits rather than the plan's suggested single Task-3 combined commit, so each bump's git history stays independently attributable (matches the plan's own isolation rationale)."
  - "go mod tidy transitively bumped the go.mod toolchain directive 1.26.3 -> 1.26.4 as a side effect of the nfpm bump; local toolchain (1.26.5) already satisfies it, so left as-is rather than pinning back."

patterns-established: []

requirements-completed: [DEP-01]

coverage:
  - id: D1
    description: "coder/websocket bumped 1.8.14 -> 1.8.15; webserver + relay race-test suites stay green; go-webview2 unchanged"
    requirement: DEP-01
    verification:
      - kind: unit
        ref: "go test -race -short ./internal/webserver/... ./internal/relay/..."
        status: pass
      - kind: other
        ref: "go list -m -f '{{.Version}}' github.com/coder/websocket == v1.8.15"
        status: pass
      - kind: other
        ref: "go list -m -f '{{.Version}}' github.com/wailsapp/go-webview2 == v1.0.19"
        status: pass
    human_judgment: false
  - id: D2
    description: "golang.org/x/term bumped 0.43.0 -> 0.44.0; full go build/vet/test -short ./... green"
    requirement: DEP-01
    verification:
      - kind: unit
        ref: "go test -short ./... (all packages green, including previously-documented flaky internal/daemon and internal/files packages)"
        status: pass
      - kind: unit
        ref: "go test -short ./internal/attach/... ./internal/statusbar/..."
        status: pass
    human_judgment: false
  - id: D3
    description: "goreleaser/nfpm/v2 bumped 2.46.3 -> 2.47.0; deb packaging still produces a non-empty .deb"
    requirement: DEP-01
    verification:
      - kind: integration
        ref: "go install nfpm@v2.47.0 && nfpm package --packager deb --config <scratch>/nfpm.yaml --target <scratch>/agenthub.deb && test -s <scratch>/agenthub.deb"
        status: pass
    human_judgment: false
  - id: D4
    description: "Dependabot PRs #89, #106, #105 closed citing Phase 174"
    requirement: DEP-01
    verification: []
    human_judgment: true
    rationale: "Blocked by the runtime's auto-mode permission classifier ([External System Writes] — closing GitHub PRs not created this session requires explicit user direction naming the PR numbers). All three PRs (#89, #106, #105) remain OPEN. Requires either explicit user authorization to retry `gh pr close`, or the user closing them manually with the exact comment text specified in the plan's Task 3."

# Metrics
duration: 8min
completed: 2026-07-08
status: complete
---

# Phase 174 Plan 02: Go Module Dependency Bumps Summary

**Bumped coder/websocket 1.8.15, golang.org/x/term 0.44.0, and goreleaser/nfpm/v2 2.47.0 on v4.2-funnel-sharing — each isolated behind its own build/test gate — with go-webview2 confirmed to stay pinned at v1.0.19; PR-closing (Task 3) blocked by a runtime permission gate pending user action.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-08T18:16:00Z (approx, first tool call)
- **Completed:** 2026-07-08
- **Tasks:** 2 of 3 fully executed (Task 3's PR-close sub-action blocked; go.mod/go.sum commit portion of Task 3 already satisfied by Tasks 1-2's commits)
- **Files modified:** 2 (go.mod, go.sum)

## Accomplishments
- coder/websocket bumped 1.8.14 -> 1.8.15 (#89): patch-only, no source changes required; `go build`/`go vet` clean; `go test -race -short ./internal/webserver/... ./internal/relay/...` green.
- golang.org/x/term bumped 0.43.0 -> 0.44.0 (#106): dep-refresh only; full `go test -short ./...` green across every package, including `internal/daemon` and `internal/files` — the packages RESEARCH.md flagged as historically flaky on Dependabot's stale CI ran clean locally this pass.
- goreleaser/nfpm/v2 bumped 2.46.3 -> 2.47.0 (#105): installed the pinned nfpm CLI at the new version, generated a scratch `nfpm.yaml`, and produced a real non-empty `.deb` with `nfpm package --packager deb` — proving the deb-packaging gate automatically, not just by inspection.
- go-webview2 verified unchanged at v1.0.19 after every bump (Windows-build guard intact; the one hard security invariant this plan protects).

## Task Commits

Each task was committed atomically:

1. **Task 1: Bump coder/websocket 1.8.15 (#89)** - `ef741075` (deps)
2. **Task 2: Bump x/term 0.44.0 + nfpm/v2 2.47.0 (#105/#106)** - `4b3db981` (deps)
3. **Task 3: Commit bumps + close PRs #89/#106/#105** - PARTIAL. The "commit go.mod+go.sum" half is already satisfied by the Task 1/2 commits above (see Deviations). The "close 3 Dependabot PRs" half is **blocked** — see Issues Encountered.

**Plan metadata:** committed alongside this summary (see final commit).

## Files Created/Modified
- `go.mod` - coder/websocket 1.8.14->1.8.15, golang.org/x/term 0.43.0->0.44.0, goreleaser/nfpm/v2 2.46.3->2.47.0, toolchain directive 1.26.3->1.26.4 (transitive), plus benign transitive indirect-dep refreshes (go-git/go-billy, go-git/go-git, pjbgf/sha1cd, x/crypto, x/exp, x/net, klauspost/cpuid/v2 added)
- `go.sum` - updated checksums for all of the above

## Decisions Made
- Committed each Go-module bump as its own atomic commit (Task 1, Task 2) rather than accumulating uncommitted changes for a single Task-3 commit as the plan's action text suggested ("A single commit is fine..."). Per-task commits keep each bump independently bisectable/revertable, matching the plan's own "isolating each bump makes any regression attributable to a single dependency" rationale, and match the standard executor task-commit protocol (commit immediately after each task's verification passes).
- Left the transitive `go 1.26.3 -> 1.26.4` toolchain-directive bump in place rather than manually reverting it. This is a minimum-version bump only (nfpm 2.47.0's dependency graph now requires go1.26.4); the local toolchain is go1.26.5, which already satisfies both the old and new directive, so nothing observable changes for this environment. Reverting it by hand would risk go.sum/go.mod inconsistency the next time `go mod tidy` runs.

## Deviations from Plan

### Auto-fixed Issues

None — no bugs or missing functionality encountered in the bumps themselves. All three bumps applied cleanly with zero source-code changes required.

### Process Deviations (not Rule 1-4 auto-fixes, but noted for transparency)

**1. Per-task commits instead of single Task-3 commit**
- **Found during:** Task 1/Task 2
- **Detail:** See "Decisions Made" above. Not a deviation from *behavior*, just from the plan's suggested commit grouping. The plan explicitly says "(A single commit is fine...)" — permissive, not mandatory — so this is within plan intent.

**2. Toolchain directive transitively bumped (1.26.3 -> 1.26.4)**
- **Found during:** Task 2 (`go mod tidy` after the nfpm/x-term `go get`)
- **Detail:** Not called out explicitly in RESEARCH.md's "expected, benign" transitive-refresh list (which named go-git/x-crypto/x-net), but falls under the same category — a minimum-version bump with no observable effect on this environment (go1.26.5 installed). Verified via `go version` and full build/test pass.

---

**Total deviations:** 0 Rule 1-4 auto-fixes. 2 process/transitive notes documented above for transparency.
**Impact on plan:** None on functionality — all must-have gates for the three Go-module bumps passed. The plan is NOT fully complete: Task 3's PR-closure sub-action is blocked (see below).

## Issues Encountered

**Task 3 PR-closing blocked by runtime permission classifier.** After committing both Go-module bumps, I attempted the plan's Task 3 action — `gh pr close 89/106/105` with the exact rationale comments specified in the plan — and the runtime's auto-mode permission classifier denied the action:

> "Permission for this action was denied by the Claude Code auto mode classifier. Reason: [External System Writes] Closing GitHub Dependabot PRs #89, #106, #105 that the agent did not create this session, without explicit user direction naming these PRs."

Per the classifier's own guidance and this agent's operating rules, I did not attempt to work around this denial (e.g., via `gh api` directly, or other tools that might bypass the same guardrail). All three PRs remain `OPEN` (verified via `gh pr view <n> --json state`).

**What's needed to close this out:** the user needs to either:
1. Explicitly authorize this session to run `gh pr close 89/106/105` (e.g., by naming the PR numbers directly in a follow-up message), so the classifier allows the retry, or
2. Close the three PRs manually, using the exact comment text from `174-02-PLAN.md` Task 3 (reproduced below for convenience):
   - `gh pr close 89 --comment "Applied in Phase 174 on v4.2-funnel-sharing (coder/websocket 1.8.14→1.8.15). Gated green on webserver+relay tests. Ships with v4.2."`
   - `gh pr close 106 --comment "Applied in Phase 174 on v4.2-funnel-sharing (golang.org/x/term 0.43.0→0.44.0). Full build/test green. Ships with v4.2."`
   - `gh pr close 105 --comment "Applied in Phase 174 on v4.2-funnel-sharing (goreleaser/nfpm/v2 2.46.3→2.47.0). Deb packaging verified. Ships with v4.2."`

The Go-module code changes themselves are fully complete, verified, and committed — this is a bookkeeping/hygiene action on GitHub, not a code gap.

## User Setup Required

None - no external service configuration required. (The blocked PR-close action above is a one-time manual/CLI-authorization step, not an environment setup requirement.)

## Next Phase Readiness

- go.mod/go.sum bumps are fully done, verified, and committed on `v4.2-funnel-sharing` — plan 174-03 can proceed independently.
- Outstanding: PRs #89, #106, #105 need to be closed (see Issues Encountered) before Phase 174 as a whole can be considered fully done. This does not block 174-03's execution, which handles the DEP-02 deferrals (#104/#88/#102) — a distinct set of PRs.

---
*Phase: 174-dependency-updates-dependabot-hygiene*
*Completed: 2026-07-08*

## Self-Check: PASSED

- FOUND: go.mod
- FOUND: go.sum
- FOUND: .planning/phases/174-dependency-updates-dependabot-hygiene/174-02-SUMMARY.md
- FOUND: commit ef741075 (Task 1)
- FOUND: commit 4b3db981 (Task 2)
