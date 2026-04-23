---
phase: 90-release-pipeline-hardening
plan: 02
type: execute
wave: 1
depends_on: []
files_modified:
  - tools.go
  - go.mod
  - go.sum
  - .github/dependabot.yml
autonomous: true
requirements: [SEC-09, SEC-10]
tags: [ci, hardening, go-tools, dependabot, wave-1]

must_haves:
  truths:
    - "The project declares its CI build tools (wails, nfpm) in go.mod via a tools.go blank-import pattern, so Dependabot's gomod ecosystem tracks them"
    - "Both wails (v2.12.0) and nfpm (v2.46.3+) appear in go.mod and go.sum"
    - "go build -tags tools ./... compiles cleanly, proving tools.go is well-formed Go source"
    - "Dependabot is configured to open weekly PRs for both github-actions and gomod ecosystems, with no auto-merge enabled"
  artifacts:
    - path: "tools.go"
      provides: "Build-tool dependency manifest (tools build tag, blank imports)"
      contains: "//go:build tools"
    - path: "go.mod"
      provides: "Pinned versions of wails CLI and nfpm via require block"
      contains: "github.com/goreleaser/nfpm/v2"
    - path: "go.sum"
      provides: "Cryptographic pinning of the above modules"
      must_exist: true
    - path: ".github/dependabot.yml"
      provides: "Weekly dependency PRs for github-actions + gomod ecosystems"
      contains: "package-ecosystem: \"github-actions\""
  key_links:
    - from: "tools.go"
      to: "go.mod"
      via: "blank imports trigger go mod tidy to add require entries"
      pattern: "_ \"github.com/goreleaser/nfpm/v2/cmd/nfpm\""
    - from: ".github/dependabot.yml"
      to: "go.mod + .github/workflows/*"
      via: "package-ecosystem gomod + github-actions scan on weekly schedule"
      pattern: "package-ecosystem: \"gomod\""
---

<objective>
Bootstrap SEC-10's source of truth for build tools and SEC-09's long-term dependency update pipeline.

Purpose:
- SEC-10 requires wails and nfpm pinned to exact versions, no `@latest`. The `tools.go` + `go.mod` approach (D-10) makes `go.mod` the single source of truth; Wave 2 (Plan 03) and Wave 3 (Plan 04) consume it via `go install ...@$(go list -m -f '{{.Version}}' ...)`.
- SEC-09 requires SHA-pinning. Dependabot's github-actions ecosystem keeps those pins fresh without silent drift (D-07). Manual merge enforced by repo setting — no auto-merge workflow.

Output: `tools.go` at repo root, updated `go.mod`/`go.sum` with nfpm added and (optionally) wails bumped to v2.12.0, and `.github/dependabot.yml`. Zero workflow changes — those are downstream plans.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md

@.planning/phases/90-release-pipeline-hardening/90-CONTEXT.md
@.planning/phases/90-release-pipeline-hardening/90-RESEARCH.md
@.planning/phases/90-release-pipeline-hardening/90-PATTERNS.md

@./CLAUDE.md

<interfaces>
<!-- Existing go.mod structure at read-time — Phase 90 baseline -->

From go.mod (lines 1-25):
```
module github.com/scottkw/agenthub

go 1.26.1

require (
  // ... runtime deps ...
  github.com/godbus/dbus/v5 v5.2.0
  github.com/kardianos/service v1.2.4
  // ... more ...
  github.com/wailsapp/wails/v2 v2.10.2   // EXISTING — candidate for bump to v2.12.0
  // ... rest ...
)
```

Target after this plan (new entry alphabetized):
```
require (
  // ...
  github.com/godbus/dbus/v5 v5.2.0
  github.com/goreleaser/nfpm/v2 v2.46.3       // NEW
  github.com/kardianos/service v1.2.4
  // ...
  github.com/wailsapp/wails/v2 v2.12.0        // BUMPED (RESEARCH recommends option (a))
  // ...
)
```

`tools.go` target content (verbatim from 90-PATTERNS.md lines 372-391 / 90-RESEARCH.md Example 1):
```go
//go:build tools
// +build tools

// Package tools documents build-tool dependencies. It is excluded from normal
// builds by the `tools` build tag. The blank imports cause `go mod tidy` to
// keep these modules in go.mod alongside runtime dependencies, making Dependabot
// aware of them via the gomod ecosystem.
//
// CI and build.sh install these tools using:
//   go install <path>@$(go list -m -f '{{.Version}}' <module>)
//
// See .planning/phases/90-release-pipeline-hardening/ for rationale.
package tools

import (
  _ "github.com/goreleaser/nfpm/v2/cmd/nfpm"
  _ "github.com/wailsapp/wails/v2/cmd/wails"
)
```

`.github/dependabot.yml` target content (verbatim from 90-PATTERNS.md lines 402-428 / 90-RESEARCH.md Example 4):
```yaml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
      timezone: "America/Los_Angeles"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "ci(actions)"
    labels:
      - "dependencies"
      - "github-actions"
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "deps"
    labels:
      - "dependencies"
      - "go"
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create tools.go + add nfpm / bump wails in go.mod + run go mod tidy</name>
  <read_first>
    - .planning/phases/90-release-pipeline-hardening/90-CONTEXT.md (D-10, D-11 — tools.go rationale and location)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Example 1 lines 554-591 — verbatim tools.go content + Go 1.24 tool directive caveat + wails v2.12.0 bump rationale)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 368-392 for tools.go pattern; lines 532-549 for go.mod edit pattern)
    - go.mod (full file — need the current wails pin v2.10.2 at line 20 and the alphabetized require block structure)
  </read_first>
  <files>tools.go, go.mod, go.sum</files>
  <action>
    **Step 1 — Create `tools.go` at repo root** with the exact content from the `<interfaces>` block above. Two lines are critical:
    - Line 1: `//go:build tools` (Go 1.17+ build constraint form)
    - Line 2: `// +build tools` (legacy form for tooling compatibility; harmless in Go 1.26)

    The file MUST be at repo root, NOT in `internal/tools/` — per 90-PATTERNS.md line 393 and 90-RESEARCH.md line 575 (grpc-go, kubernetes, cockroachdb convention).

    Use `package tools` (matches filename convention, isolated from main package).

    Blank imports:
    ```go
    import (
      _ "github.com/goreleaser/nfpm/v2/cmd/nfpm"
      _ "github.com/wailsapp/wails/v2/cmd/wails"
    )
    ```

    The `//go:build tools` tag means this file is excluded from normal builds — `go build ./...` and the Wails build ignore it. Only `go build -tags tools ./...` or `go mod tidy` considers it.

    **Step 2 — Bump wails to v2.12.0** in `go.mod` (line 20) per 90-RESEARCH.md "option (a)" recommendation (Standard Stack section, lines 127-130). The rationale: keeping runtime dep and CLI pinned to the same version prevents CLI/library skew (a documented wails issue class).

    Edit: `github.com/wailsapp/wails/v2 v2.10.2` → `github.com/wailsapp/wails/v2 v2.12.0`.

    **Step 3 — Run `go mod tidy`** to:
    - Discover the nfpm blank import in `tools.go` and add it to the `require` block.
    - Refresh `go.sum` with checksums for the bumped wails + newly added nfpm transitive deps.

    Do NOT hand-author `go.sum` lines. `go mod tidy` is authoritative.

    If `go mod tidy` complains about conflicting versions or missing deps, investigate:
    - **Expected:** One or more `// indirect` lines moved around; new nfpm require block added alphabetically (between `godbus` and `kardianos`).
    - **Surprising:** Unrelated version bumps. If seen, that means an existing indirect dep was quietly needing update for the wails v2.12.0 bump. Accept those bumps (they're part of the coherent v2.10.2 → v2.12.0 change).

    **Step 4 — Verify `tools.go` compiles** with the tools build tag:
    ```bash
    go build -tags tools ./...
    ```

    This proves the file is valid Go source and that wails + nfpm packages resolve correctly.

    **NOT IN SCOPE for this task:**
    - Updating `build.sh` — that's Plan 03 Task 3.
    - Updating any workflow YAML — Plans 03/04/05.
    - Actually installing wails or nfpm binaries — the `@$(go list -m ...)` install pattern is consumed by later plans.
  </action>
  <verify>
    <automated>test -f tools.go && grep -c '//go:build tools' tools.go && grep -F 'github.com/goreleaser/nfpm/v2' go.mod && grep -F 'github.com/wailsapp/wails/v2 v2.12.0' go.mod && go build -tags tools ./... && echo "PASS: tools.go valid + go.mod + go.sum reflect nfpm/wails"</automated>
  </verify>
  <acceptance_criteria>
    - `test -f tools.go` exits 0 (file exists at repo root, not in a subdirectory)
    - `grep -c '//go:build tools' tools.go` returns 1 (modern build constraint present)
    - `grep -c '// +build tools' tools.go` returns 1 (legacy form also present)
    - `grep -F 'package tools' tools.go` matches (package declaration correct)
    - `grep -F '_ "github.com/goreleaser/nfpm/v2/cmd/nfpm"' tools.go` matches
    - `grep -F '_ "github.com/wailsapp/wails/v2/cmd/wails"' tools.go` matches
    - `grep -E 'github.com/goreleaser/nfpm/v2 v2\.' go.mod` matches at least once (nfpm now tracked — any v2.x.y version from tidy is acceptable; v2.46.3 expected per research)
    - `grep -F 'github.com/wailsapp/wails/v2 v2.12.0' go.mod` matches exactly once (bumped from v2.10.2)
    - `grep -c 'github.com/wailsapp/wails/v2 v2.10.2' go.mod` returns 0 (old pin removed)
    - `grep -c 'github.com/goreleaser/nfpm/v2' go.sum` returns >=1 (sum updated — at least a `h1:` or `/go.mod h1:` line)
    - `go build -tags tools ./...` exits 0 (tools.go compiles cleanly)
    - `go build ./...` still exits 0 (default build IGNORES tools.go due to build tag — no regression)
    - `go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2` outputs `v2.12.0`
    - `go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2` outputs a non-empty `v2.*.*` string (version selected by tidy; v2.46.3 expected)
  </acceptance_criteria>
  <done>
    `tools.go` exists with correct build tag and blank imports. `go.mod`/`go.sum` reflect nfpm addition + wails v2.12.0 bump. Both `go build ./...` (default) and `go build -tags tools ./...` (tools) compile. No workflow YAMLs touched. Commit message: `deps(90): add tools.go + pin wails v2.12.0 / nfpm in go.mod (SEC-10)`.
  </done>
</task>

<task type="auto">
  <name>Task 2: Create .github/dependabot.yml with github-actions + gomod ecosystems</name>
  <read_first>
    - .planning/phases/90-release-pipeline-hardening/90-CONTEXT.md (D-07 — manual merge, weekly schedule)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Example 4 lines 651-678 — verbatim config; Pitfall 6 lines 528-539 — why no auto-merge field; also recommend ungrouped per line 680-682)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 397-430 — full verbatim template)
  </read_first>
  <files>.github/dependabot.yml</files>
  <action>
    Create `.github/dependabot.yml` with the exact content from the `<interfaces>` block above (verbatim from 90-PATTERNS.md lines 402-428).

    Two `updates` entries, both with `directory: "/"` and `interval: "weekly"`:

    1. `package-ecosystem: "github-actions"` — day `monday`, time `09:00`, timezone `America/Los_Angeles`, `open-pull-requests-limit: 5`, commit prefix `ci(actions)`, labels `["dependencies", "github-actions"]`.

    2. `package-ecosystem: "gomod"` — day `monday`, `open-pull-requests-limit: 5`, commit prefix `deps`, labels `["dependencies", "go"]`.

    **Do NOT add an `auto-merge` field.** Per 90-RESEARCH.md Pitfall 6 (line 532): `dependabot.yml` has no such schema field. Auto-merge is a separate repo setting (Settings → Pull requests → Allow auto-merge). D-07 requires manual merge — that's enforced by NOT enabling the repo setting AND by NOT adding any `gh pr merge --auto` workflow.

    **Do NOT add a `groups:` section.** 90-RESEARCH.md line 682 explicitly recommends ungrouped updates for this project: "the audit discipline is the whole point of SEC-09, and 4 workflows × ~6 actions = ~24 actions — bump frequency is measured in PRs-per-month, not per-day."

    The first Dependabot PR wave after this lands will be substantial (every `actions/checkout@v4` → pinned SHA, etc.) — this is expected and desired; each PR provides a reviewable unit.

    **No additional workflow file needed for Dependabot to function** — GitHub's Dependabot service reads `.github/dependabot.yml` directly. The file presence alone activates it after merge to default branch.

    **Placement note:** `.github/dependabot.yml` sits alongside `.github/workflows/` at the same level. It is NOT inside `.github/workflows/`.
  </action>
  <verify>
    <automated>test -f .github/dependabot.yml && grep -F 'package-ecosystem: "github-actions"' .github/dependabot.yml && grep -F 'package-ecosystem: "gomod"' .github/dependabot.yml && grep -F 'interval: "weekly"' .github/dependabot.yml && (grep -q 'auto-merge' .github/dependabot.yml && (echo "FAIL: auto-merge field present (D-07 violation)"; exit 1) || echo "PASS: no auto-merge field") && echo "PASS: dependabot.yml has both ecosystems"</automated>
  </verify>
  <acceptance_criteria>
    - `test -f .github/dependabot.yml` exits 0
    - `grep -c 'version: 2' .github/dependabot.yml` returns 1
    - `grep -c 'package-ecosystem: "github-actions"' .github/dependabot.yml` returns 1 exactly
    - `grep -c 'package-ecosystem: "gomod"' .github/dependabot.yml` returns 1 exactly
    - `grep -c 'directory: "/"' .github/dependabot.yml` returns 2 (once per ecosystem)
    - `grep -c 'interval: "weekly"' .github/dependabot.yml` returns 2
    - `grep -c 'open-pull-requests-limit: 5' .github/dependabot.yml` returns 2
    - `grep -F 'ci(actions)' .github/dependabot.yml` matches (github-actions commit prefix)
    - `grep -F "prefix: \"deps\"" .github/dependabot.yml` matches (gomod commit prefix)
    - `grep -c 'auto-merge' .github/dependabot.yml` returns 0 (D-07: no auto-merge field; exit 1 on violation)
    - `grep -c 'groups:' .github/dependabot.yml` returns 0 (ungrouped per RESEARCH line 682)
    - File is valid YAML: `python3 -c 'import sys, yaml; yaml.safe_load(open(".github/dependabot.yml"))'` exits 0 (or use `yq` if available: `yq . .github/dependabot.yml >/dev/null`)
  </acceptance_criteria>
  <done>
    `.github/dependabot.yml` exists with both ecosystems, weekly schedule, Monday 09:00 America/Los_Angeles for github-actions, manual-merge-only (no auto-merge field), ungrouped. Commit message: `ci(90): add dependabot config for github-actions + gomod (SEC-09 D-07)`.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Developer → go.mod | Any change to `require` block updates the CI tool versions used by subsequent release jobs. |
| Dependabot → PR queue | Dependabot opens PRs to bump SHAs; without auto-merge, human review is the policy gate. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-90-05 | Tampering | Malicious nfpm version silently pulled via `go mod tidy` | mitigate | `go.sum` pins the cryptographic hash. A substituted module with the same version but different content fails the sum check. `GOPROXY=proxy.golang.org` (Go default) adds a second layer (proxy hosts only verified module versions). |
| T-90-06 | Tampering | Attacker compromises wails v2.12.0 tag at source repo | mitigate | `go mod` does not re-fetch a module whose `go.sum` entry matches locally. A post-pin upstream tampering would not affect existing clones; only new clones see the change, and `go mod verify` would flag mismatch. Dependabot `gomod` weekly scan surfaces a version change for review before it lands. |
| T-90-07 | Elevation of Privilege | Auto-merge workflow silently lands compromised Dependabot PR | mitigate | D-07 forbids auto-merge. `.github/dependabot.yml` has no such field (enforced in acceptance criteria). Repo setting must ALSO have auto-merge disabled — documented externally as a manual check (below). |
| T-90-08 | Tampering | `tools.go` modified to pull an additional, unvetted tool | mitigate | Any addition to `tools.go` goes through PR review. `go mod tidy` then updates `go.mod`, which Dependabot also monitors — drift between `tools.go` and `go.mod` surfaces in the PR diff. |
| T-90-09 | Information Disclosure | Dependabot PR body reveals repo internal paths | accept | PRs are public (public repo). No sensitive info is encoded in commit messages or dependency diffs beyond what's already public in `go.mod`. |
| T-90-10 | Denial of Service | 24+ Dependabot PRs at once exhaust reviewer attention | accept | `open-pull-requests-limit: 5` per ecosystem caps it. RESEARCH recommends ungrouped for audit clarity. |

**Residual risk:** The `go mod tidy` step pulls nfpm and its transitive dependencies from the Go proxy. First-time trust-on-first-use for nfpm is the main new supply-chain surface. Mitigated by: (a) nfpm is a well-known tool (goreleaser project), (b) `go.sum` locks the hash immediately, (c) Dependabot `gomod` ecosystem will flag future version changes.

**External dependency (documented, not enforced by this plan):** Repository setting "Settings → Pull requests → Allow auto-merge" must be OFF or the manual-merge guarantee of D-07 is undermined. This is called out in 90-RESEARCH.md Pitfall 6 line 536.
</threat_model>

<verification>
After both tasks land:
1. `test -f tools.go && test -f .github/dependabot.yml` — both artifacts present
2. `go build ./... && go build -tags tools ./...` — both builds succeed
3. `go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2` returns `v2.12.0`
4. `go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2` returns a non-empty v2.x.y string
5. `grep -F 'auto-merge' .github/dependabot.yml` returns no matches (no auto-merge configuration)
6. The grep-gate from Plan 01 still fails — this plan does not touch workflow pins. That's expected.
</verification>

<success_criteria>
- `tools.go` exists at repo root with `//go:build tools` tag and blank imports for wails + nfpm
- `go.mod` contains `github.com/goreleaser/nfpm/v2` in the require block
- `go.mod` has `github.com/wailsapp/wails/v2 v2.12.0` (bumped from v2.10.2)
- `go.sum` has checksum entries for nfpm and updated wails
- `go build -tags tools ./...` succeeds
- `.github/dependabot.yml` has both github-actions and gomod ecosystems, weekly schedule, no auto-merge field, no groups
- Plan 03 can now run `go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2` in CI and local and get `v2.12.0`
</success_criteria>

<output>
After completion, create `.planning/phases/90-release-pipeline-hardening/90-02-SUMMARY.md` documenting:
- Files created: `tools.go`, `.github/dependabot.yml`
- Files modified: `go.mod` (add nfpm, bump wails), `go.sum` (auto-updated by tidy)
- Versions pinned: wails `v2.12.0`, nfpm `vX.Y.Z` (actual tidy result)
- Dependabot first-PR expectation: github-actions ecosystem will open ~10+ PRs the following Monday pinning every `@v4`/`@v5`/etc. to SHAs — these will LAND in Plans 03/04/05 before Dependabot can open them; Dependabot becomes steady-state after.
- Handoff to Plan 03: the `go list -m -f '{{.Version}}'` command now returns a concrete version string for both wails and nfpm; Plan 03 consumes this pattern in `build.sh`.
</output>
