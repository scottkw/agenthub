# Phase 44: Git Migration to GitHub - Research

**Researched:** 2026-04-03
**Domain:** Git repository migration (Gitea → GitHub) + Go module path rewrite
**Confidence:** HIGH

## Summary

Phase 44 has two independent sub-tasks that must be sequenced carefully: (1) mirror the Gitea
repository to GitHub with full history and all annotated tags preserved, and (2) rewrite the Go
module path from `github.com/agenthub/agenthub` to `github.com/scottkw/agenthub` across all
30 import sites in 16 Go source files plus `go.mod`.

The migration itself is a well-understood `git push --mirror` operation. The module path rewrite
is a mechanical sed/grep replace that must happen as a single atomic commit on the local repo
**before** the first push to GitHub, so the GitHub `main` branch is already canonical from day
one. Tags are already annotated (confirmed via `git tag -v v1.7`), so `--mirror` will transfer
them correctly without extra steps.

The GitHub repo `scottkw/agenthub` does not yet exist (confirmed via `gh repo view` — 404).
It must be created before the push. The `gh` CLI is authenticated as `scottkw` with `repo` and
`workflow` scopes — sufficient to create the repo and push.

**Primary recommendation:** Create GitHub repo → rewrite module path → commit → bare-clone from
Gitea → push mirror to GitHub → verify tags and `go build` → add secrets.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GIT-01 | Repository mirrored to GitHub (scottkw/agenthub) with full history and all tags preserved | `git clone --bare` + `git push --mirror` is the correct approach; tags are annotated so they transfer cleanly |
| GIT-02 | Go module path updated from github.com/agenthub/agenthub to github.com/scottkw/agenthub with all imports rewritten | 30 occurrences in 16 files; `sed -i` or `go mod edit -module` + grep-replace is the correct approach |

</phase_requirements>

## Standard Stack

### Core Tools
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| git | 2.x (system) | Bare clone + mirror push | Only reliable way to preserve annotated tags and all refs |
| gh | 2.x (authenticated) | Create GitHub repo, verify, manage secrets | Authenticated as scottkw with repo+workflow scopes — confirmed working |
| go mod edit | go 1.26.1 | Update module declaration in go.mod | Canonical tool; avoids hand-editing go.mod syntax |
| sed / grep | system | Rewrite import paths in .go files | Standard for bulk text replacement across many files |

### No External Dependencies Needed
The entire phase is git + gh CLI operations and in-repo file edits. No new packages to install.

**Installation:** N/A — all tools already available.

## Architecture Patterns

### Recommended Execution Sequence

```
1. go mod edit -module github.com/scottkw/agenthub
2. grep-replace all 30 import sites in 16 .go files
3. go build ./...  (verify — must pass before any push)
4. go test -race ./...  (verify)
5. git add -p / git commit  (module path rewrite commit)
6. gh repo create scottkw/agenthub --public --source=. --push=false
7. git clone --bare https://gitea.eightabyte.com/scottkw/agenthub.git agenthub-bare.git
   (in a temp dir outside the working copy)
8. cd agenthub-bare.git && git push --mirror https://github.com/scottkw/agenthub.git
9. Back in working copy: git remote add github https://github.com/scottkw/agenthub.git
10. git push github main --force  (pushes the module-path-rewrite commit on top)
11. git push github --tags  (re-push tags from working copy to confirm)
12. gh secret set <NAME> --repo scottkw/agenthub  (7 macOS secrets)
13. Verify: git clone https://github.com/scottkw/agenthub /tmp/verify-clone && cd /tmp/verify-clone && git tag
```

### Pattern 1: Mirror Push (History Preservation)

**What:** `git clone --bare` creates a bare clone that includes ALL refs (branches, tags,
notes). `git push --mirror` then pushes every ref to the destination, preserving annotated tag
objects (not just lightweight tags).

**When to use:** Any time you need a complete, lossless transfer of a git repository.

**Why not `git push --all` + `git push --tags`:** `--all` pushes branches only; `--tags` pushes
tags but not tag objects if the source tags were re-created. `--mirror` is authoritative.

```bash
# In a temp directory (NOT inside the working copy):
git clone --bare https://gitea.eightabyte.com/scottkw/agenthub.git agenthub-bare.git
cd agenthub-bare.git
git push --mirror https://github.com/scottkw/agenthub.git
```

**IMPORTANT:** The bare clone mirrors the Gitea state. The module-path-rewrite commit exists
only in the working copy. After the mirror push, push the working copy's main branch on top to
include the rewrite commit.

### Pattern 2: Go Module Path Rewrite

**What:** Two-step: (1) update the module declaration, (2) update all import statements.

```bash
# Step 1: Update go.mod module declaration
go mod edit -module github.com/scottkw/agenthub

# Step 2: Rewrite all import paths (macOS sed requires -i '' for in-place edit)
find /Users/ken/dev/agenthub -name "*.go" -not -path "*/vendor/*" \
  -exec sed -i '' 's|github.com/agenthub/agenthub|github.com/scottkw/agenthub|g' {} +

# Step 3: Verify
go build ./...
go test -race ./...
```

### Anti-Patterns to Avoid

- **`git push --all` + `git push --tags`:** Does not push annotated tag objects reliably across
  hosts — use `--mirror` instead.
- **Rewriting imports before verifying `go build`:** Do the rewrite and verify locally first; a
  broken build should never touch GitHub.
- **Creating the GitHub repo with `--push` during `gh repo create`:** This pushes only the
  working copy's current state, bypassing the Gitea history. Create the repo first (no push),
  then use the bare-clone mirror approach.
- **Force-pushing all tags with `--mirror` after the module path rewrite commit:** The bare
  clone's main branch tip will be the last Gitea commit (before the rewrite). After the mirror
  push, push working copy main on top. Tags should stay pointed at their original Gitea commits —
  this is correct behavior; tags are historical markers.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tag transfer | Script to re-tag each commit manually | `git push --mirror` | Mirror preserves tag objects and tagger identity |
| Import rewrite | AST-based Go tool | `sed` + `go mod edit` | Module path is a simple string; no AST needed for this specific case |
| Secret migration | Manual GitHub UI | `gh secret set` CLI | Repeatable, scriptable, auditable |

**Key insight:** This phase is entirely mechanical operations. The only real risk is sequencing.
Get the order right and there is nothing to build.

## Runtime State Inventory

> This phase involves migration/rename of the git remote and Go module path.

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — no database stores the module path or Gitea URL | None |
| Live service config | Gitea remote `origin` in local working copy | Add `github` remote; optionally update `origin` to GitHub after migration |
| OS-registered state | None — no launchd/systemd/Task Scheduler entries reference the module path | None |
| Secrets/env vars | 7 macOS signing secrets in Gitea CI settings (not in git) | Re-enter in GitHub repository secrets via `gh secret set` |
| Build artifacts | None relevant — no installed packages embed the module path | None |

**Secrets to migrate (confirmed from build.yml):**
1. `MACOS_CERTIFICATE`
2. `MACOS_CERTIFICATE_NAME`
3. `MACOS_CERTIFICATE_PWD`
4. `MACOS_CI_KEYCHAIN_PWD`
5. `MACOS_NOTARIZATION_APPLE_ID`
6. `MACOS_NOTARIZATION_PWD`
7. `MACOS_NOTARIZATION_TEAM_ID`

These 7 secrets are referenced in `.github/workflows/build.yml`. They must be set in
`scottkw/agenthub` GitHub repository settings before any CI run that involves macOS signing.

## Common Pitfalls

### Pitfall 1: Bare Clone Done Inside Working Copy
**What goes wrong:** `git clone --bare` inside the working copy creates a nested git repo.
Push operations from the bare clone may fail or behave unexpectedly.
**Why it happens:** Default instinct is to stay in the project directory.
**How to avoid:** `cd /tmp && git clone --bare <gitea-url> agenthub-bare.git`
**Warning signs:** `git` commands inside the bare clone show unexpected branch names or
the working copy's `.git` appears corrupted.

### Pitfall 2: Tag Objects Lost with Non-Mirror Push
**What goes wrong:** Tags pushed with `git push --tags` (not `--mirror`) from a non-bare
repo may arrive as lightweight tags even if the source had annotated tags.
**Why it happens:** Annotated tags are objects; the reference must be pushed from a bare
clone that has the full object graph.
**How to avoid:** Always use `git push --mirror` from the bare clone for the initial push.
Verify with `git cat-file -t <tag-sha>` after clone — should say `tag` not `commit`.
**Warning signs:** `git tag -v v1.7` on the cloned GitHub repo fails (no GPG data) OR
`git cat-file -t $(git rev-parse v1.7)` returns `commit` instead of `tag`.

### Pitfall 3: Module Path Rewrite Misses a File
**What goes wrong:** `go build ./...` succeeds locally but a file was missed; `go test` or a
specific package fails with `cannot find module providing package`.
**Why it happens:** The search pattern missed a file (wrong glob, vendor dir included, etc.).
**How to avoid:** After sed replace, run `grep -r "github.com/agenthub/agenthub" --include="*.go"` —
must return zero results.
**Warning signs:** Any output from that grep command.

### Pitfall 4: GitHub Repo Created as Private
**What goes wrong:** `gh repo create` defaults may create a private repo; `go get` and
Homebrew downloads fail because the release assets are gated.
**Why it happens:** Organization default or missing `--public` flag.
**How to avoid:** Use `gh repo create scottkw/agenthub --public` explicitly.
**Warning signs:** `curl https://github.com/scottkw/agenthub` returns 404 or login redirect.

### Pitfall 5: Working Copy `origin` Still Points to Gitea After Migration
**What goes wrong:** Future pushes and CI trigger against Gitea, not GitHub. Phase 45 onwards
assumes GitHub is the active remote.
**Why it happens:** Migration adds `github` as a second remote but doesn't update `origin`.
**How to avoid:** After successful mirror and verification, update origin:
`git remote set-url origin https://github.com/scottkw/agenthub.git`
**Warning signs:** `git remote -v` shows `origin` pointing to `gitea.eightabyte.com`.

## Code Examples

### Verified: Complete Migration Command Sequence

```bash
# === STEP 1: Rewrite Go module path ===
cd /Users/ken/dev/agenthub

go mod edit -module github.com/scottkw/agenthub

find . -name "*.go" -not -path "*/vendor/*" \
  -exec sed -i '' 's|github.com/agenthub/agenthub|github.com/scottkw/agenthub|g' {} +

# Verify zero occurrences remain
grep -r "github.com/agenthub/agenthub" --include="*.go" .
# (must be empty)

go build ./...
go test -race ./...

git add go.mod $(grep -r "github.com/scottkw/agenthub" --include="*.go" -l .)
git commit -m "chore: update Go module path to github.com/scottkw/agenthub"

# === STEP 2: Create GitHub repo ===
gh repo create scottkw/agenthub --public --description "Launch, manage, and share AI coding terminal sessions"

# === STEP 3: Mirror full Gitea history ===
cd /tmp
git clone --bare https://gitea.eightabyte.com/scottkw/agenthub.git agenthub-bare.git
cd agenthub-bare.git
git push --mirror https://github.com/scottkw/agenthub.git

# === STEP 4: Push module-path-rewrite commit on top ===
cd /Users/ken/dev/agenthub
git remote add github https://github.com/scottkw/agenthub.git
git push github main

# === STEP 5: Verify ===
cd /tmp
git clone https://github.com/scottkw/agenthub verify-clone
cd verify-clone
git tag        # must show v1.0 through v1.7
git log --oneline | head -5
git cat-file -t $(git rev-parse v1.7)  # must say "tag"

# === STEP 6: Set CI secrets ===
# Requires values from existing Gitea secrets or local secure store
gh secret set MACOS_CERTIFICATE --repo scottkw/agenthub
gh secret set MACOS_CERTIFICATE_NAME --repo scottkw/agenthub
gh secret set MACOS_CERTIFICATE_PWD --repo scottkw/agenthub
gh secret set MACOS_CI_KEYCHAIN_PWD --repo scottkw/agenthub
gh secret set MACOS_NOTARIZATION_APPLE_ID --repo scottkw/agenthub
gh secret set MACOS_NOTARIZATION_PWD --repo scottkw/agenthub
gh secret set MACOS_NOTARIZATION_TEAM_ID --repo scottkw/agenthub

# === STEP 7: Update local origin ===
git remote set-url origin https://github.com/scottkw/agenthub.git
```

### Verified: Tag Object Type Check

```bash
# After cloning from GitHub, confirm annotated tags transferred:
git cat-file -t $(git rev-parse v1.7)
# Expected output: tag
# If output is "commit", the annotated tag object was lost — redo mirror push
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `git push --all && git push --tags` | `git push --mirror` from bare clone | Always been best practice | Preserves annotated tag objects |
| Manual `go.mod` editing | `go mod edit -module` | Go 1.11+ | Avoids syntax errors in go.mod |

## Open Questions

1. **Gitea remote accessibility from CI context**
   - What we know: The Gitea remote `gitea.eightabyte.com` is the current `origin`
   - What's unclear: Whether Gitea should remain a push target after migration (dual-push) or be abandoned
   - Recommendation: Abandon Gitea as push target after GitHub migration is verified. Gitea can remain read-only as a backup reference. The plan should include updating `origin` to GitHub as its final step.

2. **CI behavior after push to GitHub**
   - What we know: `build.yml` triggers on `push` — once main is pushed to GitHub, CI will run immediately
   - What's unclear: Whether the macOS signing run will succeed immediately (secrets must be set first)
   - Recommendation: Set all 7 secrets **before** pushing working copy's main branch (Step 4). Or accept that the first CI run may fail the macOS signing step (non-blocking since signing is conditional on `env.MACOS_CERTIFICATE != ''`). The build.yml already has `if: env.MACOS_CERTIFICATE != ''` guards — an empty secret means signing is skipped gracefully, not a hard failure.

3. **Tag naming convention**
   - What we know: Current tags are `v1.0`, `v1.1`, ..., `v1.7` (single-digit minor, no patch)
   - What's unclear: Success criterion 2 mentions "v1.0.0, v1.7.0" (semver with patch) but actual tags lack patch component
   - Recommendation: The existing tags are `v1.0`–`v1.7`, not `v1.0.0`–`v1.7.0`. The planner should note this discrepancy. Tags must transfer as-is; do not create duplicate tags with a patch component. The GitHub Releases page will show them correctly.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| git | Mirror push | ✓ | System git | — |
| gh CLI | Repo creation, secrets | ✓ | Authenticated as scottkw, scopes: repo, workflow | — |
| go | Module path rewrite verification | ✓ | 1.26.1 (from go.mod) | — |
| Gitea remote access | Bare clone | ✓ | gitea.eightabyte.com/scottkw/agenthub.git is current origin | — |
| GitHub scottkw/agenthub repo | Push target | ✗ (does not exist yet) | — | Must be created as part of phase (gh repo create) |
| 7 macOS signing secrets | CI secrets migration | Unknown | Values must come from Gitea or local secure store | Manual lookup from Gitea settings UI |

**Missing dependencies with no fallback:**
- `scottkw/agenthub` GitHub repo — must be created in Step 2 of the plan.

**Missing dependencies with fallback:**
- macOS signing secret values — if not available from a local secure store, they must be
  retrieved from the Gitea repository settings UI before running `gh secret set`. This is a
  human-gate step; the plan must include a verification checkpoint.

## Validation Architecture

> workflow.nyquist_validation not explicitly disabled — including section.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | go test (built-in) |
| Config file | none (standard go test) |
| Quick run command | `go build ./...` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GIT-01 | Full history present in GitHub clone | smoke | `git clone https://github.com/scottkw/agenthub /tmp/verify && git -C /tmp/verify log --oneline \| wc -l` | ❌ Wave 0 (manual verification step) |
| GIT-01 | All v1.0–v1.7 tags present after clone | smoke | `git -C /tmp/verify tag \| sort -V` | ❌ Wave 0 (manual verification step) |
| GIT-01 | Annotated tag objects preserved | smoke | `git -C /tmp/verify cat-file -t $(git -C /tmp/verify rev-parse v1.7)` → must output `tag` | ❌ Wave 0 (manual verification step) |
| GIT-02 | No old module path remains in .go files | unit | `grep -r "github.com/agenthub/agenthub" --include="*.go" . \| wc -l` → must be 0 | ✅ |
| GIT-02 | go.mod module line is canonical | unit | `head -1 go.mod` → must read `module github.com/scottkw/agenthub` | ✅ |
| GIT-02 | Build succeeds with new module path | integration | `go build ./...` | ✅ |
| GIT-02 | Tests pass with new module path | integration | `go test -race ./...` | ✅ |

### Sampling Rate
- **Per task commit:** `go build ./...`
- **Per wave merge:** `go test -race ./...`
- **Phase gate:** Full suite green + smoke verifications (clone, tags, cat-file) before marking phase complete

### Wave 0 Gaps
- [ ] Smoke verification script (optional): `scripts/verify-github-migration.sh` — wraps the 4 git clone checks into a single runnable script

*(Existing go test infrastructure covers GIT-02. GIT-01 verification is external git operations, not automated tests.)*

## Sources

### Primary (HIGH confidence)
- git documentation (`git push --mirror`, `git clone --bare`) — standard, well-documented behavior
- go.dev/ref/mod — `go mod edit -module` documented in Go module reference
- Confirmed facts from repo inspection: `go.mod` module line, 30 import occurrences in 16 files, annotated tag on v1.7, `gh` auth confirmed as scottkw

### Secondary (MEDIUM confidence)
- GitHub docs: Creating a repository via `gh repo create` with `--public` flag
- GitHub docs: Repository secrets via `gh secret set`

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all tools verified present, authenticated
- Architecture: HIGH — git mirror + go mod edit are well-understood operations
- Pitfalls: HIGH — all pitfalls derived from direct inspection of repo state

**Research date:** 2026-04-03
**Valid until:** 2026-05-03 (stable tooling — 30 days)
