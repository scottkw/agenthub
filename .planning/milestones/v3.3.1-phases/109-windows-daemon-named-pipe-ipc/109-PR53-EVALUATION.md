# Phase 109 — PR #53 Evaluation Note

**Date:** 2026-05-18
**Phase:** 109 — Windows daemon named-pipe IPC
**Branch:** `phase-109-windows-named-pipe-ipc` (off local `main` at `9cc1087`)
**PR:** [#53](https://github.com/scottkw/agenthub/pull/53) by `@im-alexandre` (Alexandre Castro `<im.alexandre07@gmail.com>`)
**Closes:** Issue [#52](https://github.com/scottkw/agenthub/issues/52) — Windows daemon unable to bind socket
**Requirement closed by this document:** IPC-06 (author attribution documentation)

## Decision

**Cherry-pick** the three PR #53 commits onto the phase branch in their original order, preserving the `Author:` field on each. No `Co-Authored-By:` trailer is added on top of cherry-pick — the preserved `Author:` line carries IPC-06's attribution canonically.

Per CONTEXT.md D-01 ("PR #53 evaluation MANDATORY first task") and D-02 ("IPC abstraction: platform split via `ipc_windows.go` + `ipc_nonwindows.go` build tags"). Per RESEARCH.md "Recommendation: cherry-pick the three PR commits onto a phase-109 branch".

## Rationale

1. **Linear history.** Cherry-pick replays the three commits onto the phase branch with current `main` parentage. No merge-commit ceremony; no reintroduction of the 140-commit pre-base history that lived under PR #53's base (`032a6e9` / v3.2).
2. **Automatic IPC-06 attribution via preserved `Author:` field.** `git cherry-pick` preserves the original `Author:` (`Alexandre Castro <im.alexandre07@gmail.com>`) and updates only the `Committer:` to the project committer. This is the canonical GitHub attribution shape — `Co-Authored-By:` trailers are for *additional* contributors when the primary author is someone else.
3. **Per-commit granularity for separating the kernel32 tray fix from the IPC fix.** PR #53 commit 3 (`d1f0cdfb`) is an unrelated real Windows bug (`GetModuleHandleW` loaded from `user32.dll` instead of `kernel32.dll`). Cherry-pick lets us land it as its own logical commit so the audit trail is honest. A merge commit would fuse them.
4. **No conflicts.** Empirically verified — see Empirical Evidence section below.

## Empirical Evidence (`git merge-tree` simulation)

Re-verified on 2026-05-18, on the phase branch, against current `main`:

```
$ git fetch origin pull/53/head:pr-53-temp
From https://github.com/scottkw/agenthub
 * [new ref]         refs/pull/53/head -> pr-53-temp

$ git log -1 --format='%H %s' pr-53-temp
d1f0cdfb23651b748ce92d9fb034ed1e489eed97 fix: load Windows module handle from kernel32

$ git rev-parse main
9cc108796022fe3d19ed7140dda891c5af63c0c3

$ git merge-tree --write-tree --messages main pr-53-temp
2b5c3f259549aa4ee2ae95666af9f5cdfb85a102

Auto-merging internal/daemon/api.go
Auto-merging internal/daemon/client.go
```

**Result:** Exit 0, tree id `2b5c3f259549aa4ee2ae95666af9f5cdfb85a102`. Zero conflict markers. The two files git auto-merged (`internal/daemon/api.go` and `internal/daemon/client.go`) are the ones research predicted as the only conflict-candidate surfaces; the five v3.3 commits since the PR base added handlers at distinct line ranges that do not overlap with PR #53's edits to `API.Start`, `API.Stop`, and `NewDaemonClient`. Two new IPC files (`ipc_windows.go`, `ipc_nonwindows.go`) drop in clean because nothing on `main` references those paths.

This makes the re-apply-from-scratch path unnecessary. Cherry-pick is the lower-risk option.

## Three Commits to Cherry-Pick

| # | SHA       | Author                                          | Summary                                          | Files Touched                                                                                                                                  |
| - | --------- | ----------------------------------------------- | ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `6f312e1` | Alexandre Castro `<im.alexandre07@gmail.com>` | docs: design Windows daemon named pipe IPC fix   | `docs/superpowers/specs/2026-05-17-windows-daemon-named-pipe-ipc-design.md` (+104)                                                             |
| 2 | `410586d` | Alexandre Castro `<im.alexandre07@gmail.com>` | fix: use Windows named pipes for daemon IPC      | `internal/daemon/api.go`, `internal/daemon/client.go`, `internal/daemon/ipc_nonwindows.go` (new), `internal/daemon/ipc_windows.go` (new), `internal/daemon/socket_windows_test.go` |
| 3 | `d1f0cdf` | Alexandre Castro `<im.alexandre07@gmail.com>` | fix: load Windows module handle from kernel32    | `tray_windows.go`                                                                                                                              |

All three commits enumerated by `git log --reverse --format='%H %an <%ae> %s' main..pr-53-temp` on 2026-05-18. Authorship is uniform across the three commits.

## Attribution Mechanic (IPC-06)

IPC-06 acceptance criterion verbatim from `.planning/REQUIREMENTS.md`:

> **IPC-06** — PR #53 author (`im-alexandre`) credited via `Co-Authored-By` trailer on the merged/cherry-picked commits, or via dedicated commit message attribution if re-applied from scratch.

The criterion is an **OR**: trailer-on-merged-or-cherry-picked-commits, **or** dedicated-commit-message-attribution-if-re-applied. Cherry-pick takes the first arm of the OR via the `Author:` line, which is the canonical attribution medium in Git. A `Co-Authored-By:` trailer would be redundant — and arguably misleading, because trailers signal "additional contributor" semantics when the primary author would otherwise be the committer.

**Mechanic on this phase:**
- `git cherry-pick <sha>` preserves `Author:` verbatim (`Alexandre Castro <im.alexandre07@gmail.com>`).
- `Committer:` becomes the project committer (Ken Scott / `kscott@iprosystems.com`).
- Verification command after cherry-picks land:
  ```
  git log --format='%an <%ae>' main..HEAD | grep -c "Alexandre Castro"
  ```
  must return `3` (the three PR commits) on the phase branch.

The planner-authored docs in this phase (this evaluation note + `109-VERIFICATION.md`) are committed under the project committer's identity — Alexandre did not author them, and adding a `Co-Authored-By: Alexandre Castro` trailer on them would muddle the audit trail (he did not contribute to the eval note or the UAT runbook; he contributed the code that the eval note describes).

## Why Not Re-Apply From Scratch

Re-apply would mean: read PR #53's diff, hand-author equivalent commits on the phase branch under the project committer's identity, and add a `Re-applies PR #53 by @im-alexandre` line in the commit message to satisfy IPC-06's second arm.

**Costs of re-apply:**
- Loses author attribution unless a trailer is manually added on every commit (extra ceremony, easy to forget).
- More work to keep the design doc commit (#1) faithful — would require copying the doc text verbatim and committing under the project committer's identity (still attributable via prose, but less precise than `Author:`).
- No upside: the merge-tree result above (exit 0, no conflict markers) proves the PR applies cleanly, so the re-apply rationale "avoid messy merge" doesn't apply.

**Conclusion:** Re-apply is strictly worse than cherry-pick for this PR. Cherry-pick chosen.

## Windows UAT Blocker (IPC-05)

This executor is running on macOS (Darwin 25.5.0). The Windows-side cross-surface verification (IPC-05 — GUI, CLI, TUI all working on Windows 11) cannot be performed from this thread.

`109-VERIFICATION.md` (created in Task 4 of this plan) lists three discrete `human_needed` items:

- `WIN-GUI-01` — Tray icon registers + GUI new-session round-trip on Windows 11
- `WIN-CLI-01` — `agenthub.exe daemon status / new / list` round-trip on Windows 11
- `WIN-TUI-01` — TUI session list + attach + detach on Windows 11

The Windows-only unit tests (`TestAPIStart_WindowsNamedPipeHealth`, `TestAPIStop_WindowsNamedPipe`, hardened `TestCleanupStaleSocket_WindowsPipe_*`) added by PR #53 commit 2 are build-tagged `//go:build windows`; they compile-check on macOS via `GOOS=windows GOARCH=amd64 go build -tags wailsassets ./...` (gating cross-compile in Tasks 2 and 3) but they only run on Windows hardware. Their execution belongs to Task 6 Step 1, performed by a human operator.

## Closing-Requirement Index

| Requirement | Closed By                                                                              | Status After This Plan                |
| ----------- | -------------------------------------------------------------------------------------- | ------------------------------------- |
| IPC-01      | Cherry-pick of `410586d` — `listenDaemonSocket` swap in `api.go::API.Start`            | code-complete; pending WIN-CLI-01 UAT |
| IPC-02      | Cherry-pick of `410586d` — `dialDaemonSocket` swap in `client.go::NewDaemonClient`     | code-complete; pending WIN-CLI/TUI UAT |
| IPC-03      | Cherry-pick of `410586d` — `removeDaemonSocket` no-op-on-pipe in `api.go::API.Stop`    | code-complete; pending UAT            |
| IPC-04      | Cherry-pick of `410586d` — `TestAPIStart_WindowsNamedPipeHealth`, `TestAPIStop_WindowsNamedPipe`, `uniqueWindowsPipePath` | tests compile via cross-compile; pending Windows test run |
| IPC-05      | Human UAT on Windows 11 (Task 6 of this plan)                                          | pending Windows hardware              |
| IPC-06      | This document + cherry-pick author preservation                                         | **CLOSED** (this document) + verified after cherry-picks |
