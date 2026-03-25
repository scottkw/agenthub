# Phase 28: CLI Package Removal - Research

**Researched:** 2026-03-25
**Domain:** Go package deletion, reference cleanup, repository hygiene
**Confidence:** HIGH

## Summary

Phase 28 is a surgical deletion of `cmd/agenthub-cli/` — a Go package that was the old standalone CLI binary. Phase 27 (completed) moved all CLI logic into the root package (unified binary). The old package is now dead weight: it has its own `main()`, duplicates the command implementations that already live in `cmd_cli.go` at the root, and its tests run against internal packages that the root package also tests directly.

The scope is tightly bounded. An exhaustive grep of all active source files (Go, shell, YAML, JSON, Markdown) outside of `.planning/` and `.claude/` worktrees found exactly **two live references** to `agenthub-cli`: one in `README.md` (a package table row) and zero in CI workflows or build scripts. The `go.mod` has no dependency on `cmd/agenthub-cli` because Go workspace packages are not declared as module dependencies — they are just directories under the module root. Deleting the directory is sufficient to remove it from `go build ./...`.

The only risk worth naming is leaving a dangling reference in `README.md`. Everything else is fully contained.

**Primary recommendation:** Delete `cmd/agenthub-cli/` (8 files), update the one README table row, then run `go build ./...` and `go test ./...` to confirm clean state.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CLEAN-01 | `cmd/agenthub-cli/` directory fully removed | Directory contains 8 files (main.go, cmd_daemon.go, cmd_attach.go, cmd_attach_unix.go, cmd_attach_windows.go, cmd_daemon_test.go, cmd_attach_test.go, main_test.go). All logic is already duplicated in root package. Safe to delete. |
| CLEAN-02 | No references to `agenthub-cli` remain in docs, CI, or build scripts | Only active reference found: README.md line 74 — one table row `cmd/agenthub-cli | CLI command implementations`. CI workflow (build.yml) has no references. build.sh has no references. |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| `go build ./...` | Go 1.26.1 (from go.mod) | Verify no import errors after deletion | Standard Go verification command |
| `go test ./...` | Go 1.26.1 | Run all tests post-deletion | Validates removed package tests are gone cleanly |

No new libraries are introduced. This phase is pure deletion.

**Pre-deletion build state (verified):**
```
ok  github.com/agenthub/agenthub                    6.284s
ok  github.com/agenthub/agenthub/cmd/agenthub-cli   2.282s
ok  github.com/agenthub/agenthub/internal/daemon    1.390s
...
```

**Post-deletion expected state:** `cmd/agenthub-cli` row disappears; all other packages remain green.

## Architecture Patterns

### What `cmd/agenthub-cli/` Contains (8 files)

```
cmd/agenthub-cli/
├── main.go              # Old CLI entrypoint — duplicate of root dispatch logic
├── cmd_daemon.go        # daemon subcommands — already in root cmd_daemon.go
├── cmd_attach.go        # attach command — already in root cmd_attach.go
├── cmd_attach_unix.go   # Unix attach impl — already in root cmd_attach_unix.go
├── cmd_attach_windows.go# Windows attach impl — already in root cmd_attach_windows.go
├── cmd_daemon_test.go   # Tests against internal/daemon — covered by root tests
├── cmd_attach_test.go   # Tests against internal/relay — covered by root tests
└── main_test.go         # Integration tests for CLI commands — superseded by root cmd_cli_test.go
```

### Why Deletion Is Safe

1. **No imports from this package.** `grep -r "cmd/agenthub-cli" --include="*.go"` returns zero results. Nothing imports from this package.
2. **All logic duplicated in root.** Phase 27 created `cmd_cli.go`, `cmd_attach.go`, `cmd_daemon.go`, `dispatch_test.go`, `cmd_cli_test.go`, and `cmd_attach_test.go` at the root level.
3. **`go.mod` has no entry.** Go does not declare sub-packages of the same module as dependencies. Deleting a directory is sufficient.
4. **CI does not reference the package.** `.github/workflows/build.yml` runs `go test ./...` (picks up all packages automatically) and builds with Wails. No hard-coded reference to `cmd/agenthub-cli`.
5. **`build.sh` has no reference.** Confirmed by grep.

### Reference Inventory (exhaustive, active files only)

| File | Reference | Type | Action |
|------|-----------|------|--------|
| `README.md:74` | `\| \`cmd/agenthub-cli\` \| CLI command implementations \|` | Doc table row | Delete the row |
| `.github/workflows/build.yml` | None | — | No action |
| `build.sh` | None | — | No action |
| All `*.go` files (root + internal) | None | — | No action |

**`.planning/`, `.claude/worktrees/`:** These contain historical docs referencing `agenthub-cli`. They are planning artifacts and worktree snapshots — not active source, not scanned by `go build`, not part of the success criteria. Leave untouched.

### Anti-Patterns to Avoid

- **Editing files inside `.planning/` to remove historical references:** The success criteria says "no file in the repo (docs, CI workflows, build scripts, Go source)" — planning artifacts and worktree snapshots are not in scope for CLEAN-02.
- **Running `go mod tidy` after deletion:** Not needed. The `cmd/agenthub-cli` package imports the same dependencies that the root already uses. Module dependencies will not change.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Verifying no dangling imports | Custom grep script | `go build ./...` | The compiler will fail with a clear error if any import of the deleted package remains |
| Finding all references | Manual search | `grep -r "agenthub-cli"` on active files | One command, authoritative |

**Key insight:** `go build ./...` is the authoritative reference check for Go imports. The only non-Go references (README) must be checked separately since the Go compiler ignores `.md` files.

## Runtime State Inventory

> This phase is a file deletion and doc update with no external service dependencies.

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — verified by reviewing all internal packages; no database stores the string `agenthub-cli` as a key or record | None |
| Live service config | None — no n8n workflows, no external services reference this binary name | None |
| OS-registered state | None — service manager integration (Phase 23) uses the name `agenthub`, not `agenthub-cli` | None |
| Secrets/env vars | None — no .env files, CI secrets, or SOPS keys reference `agenthub-cli` | None |
| Build artifacts | `agenthub-cli` binary exists at repo root (ls output confirmed) — this is a previously built artifact, not a source file | Delete or ignore; it is not tracked by git |

**Nothing found in categories 1-4.** The `agenthub-cli` binary at the repo root is a stale build artifact. It is not tracked by git (confirmed by `go.mod` — Wails builds go to `build/bin/`). The planner may optionally include deleting it, but it is not required for the success criteria.

## Common Pitfalls

### Pitfall 1: Scope Creep into Planning Artifacts
**What goes wrong:** Attempting to update all `.planning/` and `.claude/worktrees/` files that mention `agenthub-cli` (80+ files). This is unnecessary and time-consuming.
**Why it happens:** A broad `grep -r` sweep includes worktree snapshots and historical phase summaries.
**How to avoid:** Scope the cleanup to active source files only: `*.go`, `*.sh`, `*.yml`, `*.yaml`, `*.md` at the repo root level (not inside `.planning/` or `.claude/`).
**Warning signs:** Task list has 80+ file edits — stop and re-scope.

### Pitfall 2: Forgetting `go test ./...` Has the Package Listed
**What goes wrong:** After deletion, `go test ./...` output changes (the `cmd/agenthub-cli` line disappears). If a CI check compares exact test output, it would fail.
**Why it happens:** CI currently runs `go test ./...` and passes. The package just disappears from the list.
**How to avoid:** Verify CI does not pin expected package names. Review confirmed: `build.yml` just runs `go test ./...` with no output assertion. Safe.

### Pitfall 3: `go.sum` / `go.mod` Drift
**What goes wrong:** Assuming `go mod tidy` is needed after deletion.
**Why it happens:** Developers associate package removal with module changes.
**How to avoid:** Since `cmd/agenthub-cli` only imported packages already used by the root module (`internal/daemon`, `internal/webserver`, `github.com/skip2/go-qrcode`), removing it does not orphan any `go.mod` dependencies. Run `go mod tidy` only if `go build ./...` produces an "unused dependency" warning — it won't here.

## Code Examples

### Verification Commands (post-deletion)
```bash
# Source: standard Go toolchain
go build ./...               # Must exit 0 with no errors
go test ./...                # cmd/agenthub-cli row must be absent; all others green

# Confirm no source references remain
grep -r "agenthub-cli" . \
  --include="*.go" \
  --include="*.sh" \
  --include="*.yml" \
  --include="*.yaml" \
  --include="*.md" \
  --exclude-dir=".planning" \
  --exclude-dir=".claude"
# Must return zero results
```

### README Edit (line 74, one row deletion)
Before:
```markdown
| `cmd/agenthub-cli` | CLI command implementations |
```
After: row deleted entirely. The table still makes sense — the unified binary's CLI commands now live in the root package, which is not a separate `cmd/` entry.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Separate `cmd/agenthub-cli/` binary for CLI | CLI commands in root package, dispatched by `main.go` | Phase 27 (2026-03-25) | `cmd/agenthub-cli/` is now dead code |

**Deprecated:**
- `cmd/agenthub-cli/main.go`: The standalone CLI entrypoint. Replaced by `dispatch_test.go` + `main.go` dispatch logic in root package.

## Open Questions

None. The scope is fully determined:
- 8 files to delete (all in `cmd/agenthub-cli/`)
- 1 line to remove from `README.md`
- 2 verification commands to run

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — pure file deletion and doc update).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package |
| Config file | none (go test uses go.mod) |
| Quick run command | `go build ./...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLEAN-01 | `cmd/agenthub-cli/` directory does not exist | smoke | `ls cmd/agenthub-cli 2>/dev/null && exit 1 \|\| exit 0` | N/A — shell check |
| CLEAN-01 | `go build ./...` succeeds after deletion | build | `go build ./...` | N/A — toolchain |
| CLEAN-02 | No `agenthub-cli` references in active source files | smoke | `grep -r "agenthub-cli" . --include="*.go" --include="*.sh" --include="*.yml" --include="*.md" --exclude-dir=.planning --exclude-dir=.claude` | N/A — grep check |

### Sampling Rate
- **Per task commit:** `go build ./...`
- **Phase gate:** `go test ./...` full suite green before `/gsd:verify-work`

### Wave 0 Gaps
None — existing test infrastructure covers all phase requirements. This phase deletes tests, it does not add them.

## Sources

### Primary (HIGH confidence)
- Direct file inspection — `cmd/agenthub-cli/` directory listing and file reads
- `grep -r "agenthub-cli"` across all active source files — confirmed single README reference
- `go test ./...` output — confirmed package currently compiles and tests pass
- `.github/workflows/build.yml` — confirmed no `agenthub-cli` references
- `build.sh` — confirmed no `agenthub-cli` references
- `go.mod` — confirmed no module-level dependency on the package

### Secondary (MEDIUM confidence)
- Go module documentation: sub-packages of the same module are not declared in `go.mod`; deleting them requires no `go.mod` edit

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — toolchain is standard Go; no new libraries
- Architecture: HIGH — exhaustive grep confirms all references; file list is complete
- Pitfalls: HIGH — pitfalls derived from direct code inspection, not speculation

**Research date:** 2026-03-25
**Valid until:** N/A — this is a point-in-time snapshot of the repo state; valid for the duration of Phase 28 execution
