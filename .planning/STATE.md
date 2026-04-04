---
gsd_state_version: 1.0
milestone: v1.8
milestone_name: GitHub Distribution & CI/CD
status: executing
stopped_at: Completed 45-release-please-ci-signing-removal-45-01-PLAN.md
last_updated: "2026-04-04T18:28:06.668Z"
last_activity: 2026-04-04
progress:
  total_phases: 5
  completed_phases: 2
  total_plans: 4
  completed_plans: 4
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-03)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 45 — release-please-ci-signing-removal

## Current Position

Phase: 46
Plan: Not started
Status: Ready to execute
Last activity: 2026-04-04

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 0
- Total phases: 5
- Timeline: Started 2026-04-03

## Accumulated Context

### Decisions

- Use `git clone --bare` + `git push --mirror` for Gitea migration — prevents history/tag loss (Pitfall 1 from research)
- Do NOT rely on `version-file` config in release-please — use `extra-files` with `// x-release-please-version` annotation due to Go type bug (issue #2541)
- macOS signing moves from `build.yml` to `release.yml` only — saves notarization quota on every PR
- Both `TAP_DEPLOY_TOKEN` and `WINGET_TOKEN` must be classic PATs (not fine-grained) — fine-grained incompatible with cross-repo dispatch and winget-releaser
- WinGet first submission is manual — `winget-releaser` only works after package identity exists in microsoft/winget-pkgs
- [Phase 44-git-migration-to-github]: Capture trayCallbackApp pointer in caller goroutine before onTrayQuit goroutine to eliminate cgo callback data race
- [Phase 45-release-please-ci-signing-removal]: Use release-type: simple (not go) to sidestep Go version-file bug (issue #2541); wails.json updated via JSON-path extra-file
- [Phase 45-release-please-ci-signing-removal]: macOS signing removed from build.yml; moves exclusively to release.yml in Phase 46 to save notarization quota on PR builds

### Pending Todos

None.

### Blockers/Concerns

- Phase 48 (WinGet) has an external timeline dependency: Microsoft PR review for first submission takes hours to ~1 day. Start early; does not block milestone completion since submission is async.

## Session Continuity

Last session: 2026-04-04T18:03:06.138Z
Stopped at: Completed 45-release-please-ci-signing-removal-45-01-PLAN.md
Resume file: None
Next action: `/gsd:plan-phase 44`
