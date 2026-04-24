---
phase: "90"
plan: "02"
subsystem: ci-hardening
tags: [ci, hardening, go-tools, dependabot, wave-1, sec-09, sec-10]
dependency_graph:
  requires:
    - Plan 01: scaffolding (grep-gate.sh + Section 12 red tests)
  provides:
    - tools.go (build-tool dependency manifest, //go:build tools)
    - go.mod updated (nfpm v2.33.1, wails v2.12.0)
    - go.sum updated (cryptographic pins for nfpm + wails bump)
    - .github/dependabot.yml (weekly PRs for github-actions + gomod)
  affects:
    - "Plan 03: build.sh — can now use go list -m -f '{{.Version}}' wails/nfpm"
    - "Plan 04: release.yml — same go list pattern available in CI"
    - "Plan 05: distribute.yml — dependabot config active after merge"
tech_stack:
  added:
    - "github.com/goreleaser/nfpm/v2 v2.33.1 (go.mod direct dep)"
  patterns:
    - "tools.go blank-import pattern for build tool pinning"
    - "dependabot.yml dual-ecosystem config (github-actions + gomod)"
key_files:
  created:
    - tools.go
    - .github/dependabot.yml
  modified:
    - go.mod (wails bumped v2.10.2 → v2.12.0; nfpm v2.33.1 added)
    - go.sum (checksums refreshed by go mod tidy)
decisions:
  - "tools.go imports library roots (github.com/goreleaser/nfpm/v2 and github.com/wailsapp/wails/v2) instead of cmd/nfpm and cmd/wails — both cmd packages are package main and cannot be blank-imported; library root blank imports achieve identical module pinning in go.mod"
  - "nfpm resolved to v2.33.1 by go mod tidy (not v2.46.3 as RESEARCH estimated); tidy selects the minimum version satisfying the dependency graph — v2.33.1 is current as of tidy run"
  - "go build -tags tools ./... fails pre-existingly due to security-review/ package conflict; go build -tags tools . (root package) passes — verified pre-existing by git stash test"
  - "No auto-merge field in dependabot.yml — D-07 enforced; manual merge required"
  - "Ungrouped dependabot updates — per RESEARCH line 682, audit discipline is the goal"
metrics:
  duration: "~7 minutes"
  completed: "2026-04-24T13:06:05Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 2
  lines_added: 455
---

# Phase 90 Plan 02: tools.go + Dependabot Summary

**One-liner:** Build-tool version pinning via tools.go blank-import pattern + weekly Dependabot PRs for github-actions and gomod ecosystems (SEC-09/SEC-10).

## What Was Built

Two artifacts that establish the dependency-version source of truth for the rest of Phase 90:

1. **`tools.go`** — Build-tool dependency manifest at repo root with `//go:build tools` + `// +build tools` constraints. Blank-imports `github.com/goreleaser/nfpm/v2` and `github.com/wailsapp/wails/v2` (library roots, not cmd sub-packages). Excluded from normal `go build` by the build tag; included in `go mod tidy` graph so go.mod tracks these modules.

2. **`.github/dependabot.yml`** — Dual-ecosystem Dependabot config:
   - `github-actions` ecosystem: weekly Monday 09:00 America/Los_Angeles, 5 PRs max, prefix `ci(actions)`, labels `[dependencies, github-actions]`
   - `gomod` ecosystem: weekly Monday, 5 PRs max, prefix `deps`, labels `[dependencies, go]`
   - No `auto-merge` field (D-07 compliance), no `groups:` (ungrouped for audit clarity)

3. **`go.mod` / `go.sum`** — Updated by `go mod tidy`:
   - `github.com/goreleaser/nfpm/v2 v2.33.1` added as direct dependency
   - `github.com/wailsapp/wails/v2` bumped from `v2.10.2` to `v2.12.0`
   - `go.sum` refreshed with cryptographic hashes for all new/changed modules

## Versions Pinned

| Module | Version |
|--------|---------|
| `github.com/wailsapp/wails/v2` | `v2.12.0` (bumped from v2.10.2) |
| `github.com/goreleaser/nfpm/v2` | `v2.33.1` (added; tidy resolved; RESEARCH estimated v2.46.3) |

## Commits

| Task | Commit | Message |
|------|--------|---------|
| Task 1: tools.go + go.mod/go.sum | 5d5251f | deps(90): add tools.go + pin wails v2.12.0 / nfpm in go.mod (SEC-10) |
| Task 2: dependabot.yml | 0d07d15 | ci(90): add dependabot config for github-actions + gomod (SEC-09 D-07) |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Import paths adjusted from cmd sub-packages to library roots**
- **Found during:** Task 1 — `go build -tags tools ./...`
- **Issue:** `github.com/goreleaser/nfpm/v2/cmd/nfpm` and `github.com/wailsapp/wails/v2/cmd/wails` are both `package main` programs; Go does not permit blank imports of `main` packages
- **Fix:** Changed imports to library roots `github.com/goreleaser/nfpm/v2` and `github.com/wailsapp/wails/v2`. Both are importable library packages. The module pinning effect in `go.mod` is identical — go.mod tracks module paths, not package paths, and `go mod tidy` correctly adds the `require` entries for both modules regardless of which package is imported
- **Files modified:** tools.go
- **Commit:** 5d5251f

**2. [Pre-existing] `security-review/` package conflict in `go build ./...`**
- **Found during:** Task 1 verification
- **Status:** Pre-existing issue — confirmed by `git stash` test showing identical failure before this plan's changes
- **Impact:** `go build ./...` and `go build -tags tools ./...` both fail with "found packages relay ... and webserver ..." in security-review/. Root package builds (`go build .` and `go build -tags tools .`) pass.
- **Action:** Logged to deferred-items; not introduced by this plan

**3. [Informational] nfpm version is v2.33.1 not v2.46.3**
- **Found during:** Task 1 — `go mod tidy` output
- **Issue:** RESEARCH estimated v2.46.3 based on latest release knowledge; the Go module proxy resolved v2.33.1 as the minimum version satisfying the dependency graph
- **Impact:** None — any v2.x.y is acceptable per acceptance criteria; Dependabot gomod will propose upgrades on the weekly schedule
- **Action:** Documented; no fix needed

## Dependabot First-PR Expectation

After this plan merges to main, Dependabot will activate on the following Monday and open PRs for:
- **github-actions ecosystem:** ~10-24 PRs pinning every `@v4`/`@v5`/`@v3` action ref to a 40-char SHA. Plans 03/04/05 are landing these pins before Dependabot runs — Dependabot becomes steady-state maintenance after.
- **gomod ecosystem:** PRs for any dependency with a newer version available, including nfpm and wails if newer releases exist

## Handoff to Plan 03

Plan 03 can now use the `go list` pattern with guaranteed non-empty results:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
# Returns: v2.12.0

go install github.com/goreleaser/nfpm/v2/cmd/nfpm@$(go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2)
# Returns: v2.33.1
```

These queries now work correctly both locally and in CI (D-11, D-12).

## Known Stubs

None — no UI data flows involved in this plan.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes. `tools.go` is excluded from production builds by build tag.

## Self-Check: PASSED

| Item | Status |
|------|--------|
| tools.go exists at repo root | FOUND |
| //go:build tools in tools.go | FOUND (grep -c returns 1) |
| // +build tools in tools.go | FOUND (grep -c returns 1) |
| github.com/goreleaser/nfpm/v2 in go.mod | FOUND (v2.33.1) |
| github.com/wailsapp/wails/v2 v2.12.0 in go.mod | FOUND |
| go build -tags tools . exits 0 | PASS |
| go build . exits 0 | PASS |
| .github/dependabot.yml exists | FOUND |
| github-actions ecosystem in dependabot.yml | FOUND |
| gomod ecosystem in dependabot.yml | FOUND |
| no auto-merge field | CONFIRMED (grep -c returns 0) |
| Commit 5d5251f exists | FOUND |
| Commit 0d07d15 exists | FOUND |
