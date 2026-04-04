---
phase: 44-git-migration-to-github
plan: 02
subsystem: infra
tags: [github, git, migration, secrets, ci]

# Dependency graph
requires:
  - "go.mod declares module github.com/scottkw/agenthub"
provides:
  - "GitHub repository scottkw/agenthub with full history"
  - "All v1.0-v1.7 annotated tags on GitHub"
  - "Local origin remote points to GitHub"
  - "7 macOS signing secrets in release environment"

key-files:
  created: []
  modified: []
---

## What was done

1. Created public GitHub repository `scottkw/agenthub`
2. Bare-cloned from Gitea and mirror-pushed full history to GitHub (all branches, all tags)
3. Pushed module-path-rewrite commit from Plan 01 as tip of main
4. Verified: tags v1.0-v1.7 present, annotated tag objects preserved (type=tag), build passes from fresh clone
5. Updated local origin remote from Gitea to GitHub
6. Created `release` environment on GitHub repo
7. Set 7 macOS CI signing secrets (MACOS_CERTIFICATE, MACOS_CERTIFICATE_NAME, MACOS_CERTIFICATE_PWD, MACOS_CI_KEYCHAIN_PWD, MACOS_NOTARIZATION_APPLE_ID, MACOS_NOTARIZATION_PWD, MACOS_NOTARIZATION_TEAM_ID) in release environment

## Deviations

- Secrets were set as **environment-level** secrets (in `release` environment) rather than repo-level, matching the storcat repo pattern. The existing `build.yml` references `secrets.MACOS_*` which works with both scopes, but the workflow will need `environment: release` to access them (Phase 45 concern).
- MACOS_CI_KEYCHAIN_PWD was auto-generated (random hex) since it's only used for ephemeral CI keychains.

## Self-Check: PASSED

- [x] `gh repo view scottkw/agenthub --json name` returns valid JSON
- [x] `git ls-remote` shows v1.0-v1.7 tags
- [x] Annotated tag objects preserved (cat-file shows `tag`)
- [x] Module-path-rewrite commit is tip of main on GitHub
- [x] Local origin points to github.com/scottkw/agenthub
- [x] 7 secrets configured in release environment
- [x] Fresh clone builds successfully
