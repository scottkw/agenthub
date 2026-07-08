# Phase 174: Dependency Updates & Dependabot Hygiene - Research

**Researched:** 2026-07-08
**Domain:** Dependency management / Dependabot hygiene / CI supply-chain (GitHub Actions + Go modules)
**Confidence:** HIGH (grounded in live `gh` PR state, changelogs, and repo file inspection)

## Summary

This phase merges 7 low-risk Dependabot bumps and formally defers 3 high-risk ones. All 10 enumerated PRs are **still open** on the repo, all target `base=main`, and their version numbers **exactly match the ROADMAP spec — no drift**. No new/unlisted Dependabot PRs exist.

The single most important finding: **the "FAILURE" CI status on nearly every Dependabot PR is NOT caused by the bump.** The failing legs are pre-existing flaky tests (`internal/daemon` chat-store path tests, `TestExitEvent_ListSessions_ExitCodePopulatedForStopped`, `TestAliasStoreFilePerms` Windows perms, and `internal/files` `TestWrite_TwoWritersIfMatchRace` race precondition) plus a flaky `playwright [e2e]` job. These flakes appear even on trivial CI-YAML-only bumps (#85 pnpm, #103 action-gh-release) that cannot possibly affect Go tests. **The planner must NOT gate merge decisions on the stale red CI badges — it must verify locally.** (See Common Pitfalls.)

The second critical finding, **verified**: `wailsapp/wails/v2 2.12.0` (#104) requires `go-webview2 v1.0.22`, but the project pins `go-webview2 v1.0.19` because `>=1.0.20` breaks the Windows build (Phase 145). The existing `dependabot.yml` ignore on `go-webview2` does **not** protect against this — an ignore only stops Dependabot opening a PR; it does not stop a transitive minimum-version requirement bump when `wails` itself is upgraded. This is the concrete, proven reason #104 must be deferred.

**Primary recommendation:** Apply the 7 low-risk bumps directly onto `v4.2-funnel-sharing` (the active integration branch, 376 commits ahead of main), verify each locally with the real gate, then close all 10 Dependabot PRs with rationale — the low-risk ones as "applied in Phase 174 on v4.2-funnel-sharing," the 3 high-risk ones as "deferred, see dependabot.yml ignore." Do not attempt to merge the Dependabot PRs into `main` directly (their base lacks the 376 v4.2 commits and their CI is stale/flaky).

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DEP-01 | Merge 7 low-risk bumps, each verified green (go build/vet/test + `tsc && vite build` + deb packaging) | Exact SHAs/versions captured below; local gate commands confirmed; changelogs reviewed (all patch/minor, no breaking API); flaky-CI caveat documented |
| DEP-02 | Defer 3 high-risk bumps: dependabot.yml ignore entries + close PRs citing this phase | Exact ignore syntax per ecosystem drafted below; wails↔go-webview2 coupling verified; tailscale/Funnel + checkout-major rationale grounded |

## Standard Stack

No new packages are introduced. This phase only changes versions of incumbent, long-established dependencies already in `go.mod` / workflow files. See Package Legitimacy Audit.

### Exact bump targets (verified live, 2026-07-08)

**Low-risk — MERGE (DEP-01):**

| PR | Ecosystem | Dependency | From → To | Where it lives | Touches compiled code? |
|----|-----------|------------|-----------|----------------|------------------------|
| #114 | github-actions | actions/attest-build-provenance | 4.1.0 → 4.1.1 | `release.yml` only (×4) | No — release pipeline only |
| #113 | github-actions | actions/setup-go | 6.4.0 → 6.5.0 | `build.yml`, `e2e.yml`, `release.yml` | No — CI runner setup |
| #103 | github-actions | softprops/action-gh-release | 3.0.0 → 3.0.1 | `release.yml` only | No — release pipeline only |
| #85 | github-actions | pnpm/action-setup | 6.0.8 → 6.0.9 | `build.yml`, `e2e.yml`, `release.yml` | No — CI runner setup |
| #89 | gomod | github.com/coder/websocket | 1.8.14 → 1.8.15 | `go.mod` (webserver/relay/attach) | **YES** — gate on webserver+relay tests |
| #106 | gomod | golang.org/x/term | 0.43.0 → 0.44.0 | `go.mod` (cmd_attach, statusbar, attach_unix) | **YES** — but only x/ dep refresh (see changelog) |
| #105 | gomod | github.com/goreleaser/nfpm/v2 | 2.46.3 → 2.47.0 | `go.mod` (tools.go pin; used as `nfpm` CLI at release) | **YES (tool)** — gate = deb packaging still builds |

**High-risk — DEFER (DEP-02):**

| PR | Ecosystem | Dependency | From → To | Defer reason |
|----|-----------|------------|-----------|--------------|
| #104 | gomod | github.com/wailsapp/wails/v2 | 2.10.2 → 2.12.0 | **VERIFIED:** requires go-webview2 **v1.0.22** (project pins **v1.0.19**; ≥1.0.20 breaks Windows build) |
| #88 | gomod | tailscale.com | 1.98.3 → 1.100.0 | Entire Funnel feature built + live-UAT'd on 1.98.3; needs full off-tailnet re-UAT post-ship |
| #102 | github-actions | actions/checkout | 6.0.3 → 7.0.0 | Major bump; evaluate in a branch (fork-PR blocking + ESM/runtime change) |

### Exact SHA changes for the 4 low-risk action bumps

Actions are **SHA-pinned with a version comment**. Bumping means replacing the SHA (the comment alone is cosmetic). Copy the new SHA verbatim from the Dependabot diff — do not hand-type:

```
# #114 attest-build-provenance  (release.yml — 4 occurrences)
- actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
+ actions/attest-build-provenance@0f67c3f4856b2e3261c31976d6725780e5e4c373 # v4.1.1

# #113 setup-go  (build.yml, e2e.yml, release.yml — every occurrence)
- actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0
+ actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0

# #103 action-gh-release  (release.yml — 1 occurrence)
- softprops/action-gh-release@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0
+ softprops/action-gh-release@718ea10b132b3b2eba29c1007bb80653f286566b # v3.0.1

# #85 pnpm/action-setup  (build.yml, e2e.yml, release.yml — every occurrence)
- pnpm/action-setup@0e279bb959325dab635dd2c09392533439d90093 # v6.0.8
+ pnpm/action-setup@0ebf47130e4866e96fce0953f49152a61190b271 # v6.0.9
```
[VERIFIED: `gh pr diff` on #114/#113/#103/#85, 2026-07-08]

> **Landmine:** setup-go and pnpm/action-setup each appear in **three** workflow files (`build.yml`, `e2e.yml`, `release.yml`). A manual edit must hit **every** occurrence or CI drifts inconsistently. `grep -rn '<action>' .github/workflows/*.yml` enumerates them. Dependabot's own PR updates all occurrences at once — an alternative is to `gh pr checkout` each and copy the resulting file(s).

### Go module bumps — how to apply on the branch

```bash
go get github.com/coder/websocket@v1.8.15
go get golang.org/x/term@v0.44.0
go get github.com/goreleaser/nfpm/v2@v2.47.0
go mod tidy
```
Each edits `go.mod` + `go.sum`. `go mod tidy` may pull transitive updates — review the `go.sum` diff.

## Package Legitimacy Audit

> All bumps target **incumbent, already-vetted dependencies** — no new package is introduced. Legitimacy is established (these are the project's existing direct deps and pinned CI actions). Verdicts below reflect maintenance health / breaking-change risk, not slopsquatting risk.

| Package | Ecosystem | Provenance | Verdict | Disposition |
|---------|-----------|------------|---------|-------------|
| actions/attest-build-provenance | github-actions | official GitHub action, SHA-pinned | OK | Approve 4.1.1 |
| actions/setup-go | github-actions | official GitHub action, SHA-pinned | OK | Approve 6.5.0 |
| softprops/action-gh-release | github-actions | established (SHA-pinned) | OK | Approve 3.0.1 |
| pnpm/action-setup | github-actions | official pnpm action, SHA-pinned | OK | Approve 6.0.9 |
| github.com/coder/websocket | gomod | incumbent direct dep | OK | Approve 1.8.15 (patch) |
| golang.org/x/term | gomod | golang.org/x official | OK | Approve 0.44.0 (dep refresh only) |
| github.com/goreleaser/nfpm/v2 | gomod | incumbent build tool | OK | Approve 2.47.0 (minor) |
| github.com/wailsapp/wails/v2 | gomod | incumbent framework | DEFER | go-webview2 coupling — see DEP-02 |
| tailscale.com | gomod | incumbent core dep | DEFER | Funnel re-UAT — see DEP-02 |
| actions/checkout | github-actions | official, SHA-pinned | DEFER | major bump — see DEP-02 |

**Packages removed due to [SLOP]:** none. **Packages flagged [SUS]:** none.

## Architecture Patterns

### The verify/merge gate (maps to DEP-01 "each verified green")

The authoritative local gate, matching what CI (`build.yml`) enforces:

```bash
# Go — compiled-code bumps (#89, #106; #105 is tool-only)
go build ./...
go vet ./...
go test -short ./...                       # full suite
go test -race -short ./internal/webserver/... ./internal/relay/...   # coder/websocket gate (#89)
go test -race -short ./internal/attach/... ./internal/statusbar/...  # x/term gate (#106)

# Frontend — required by DEP-01 wording (unaffected by these bumps, but part of the gate)
cd frontend && pnpm install && pnpm test
# The wails-equivalent build check (CI runs `wails build ... -tags wailsassets`):
cd frontend && pnpm exec tsc && pnpm exec vite build      # note: tsc catches TS errors vitest tolerates

# deb packaging (#105 nfpm gate) — nfpm is a Go-module-pinned CLI, installed then run:
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@$(go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2)
# then `nfpm package --packager deb` against a generated nfpm.yaml (release.yml:255-277 shows the recipe)
```

**Key gate facts:**
- CI required checks (branch protection on `main`): the 4 `build (...)` legs + `playwright`. [VERIFIED: `gh api .../branches/main/protection`]
- `-tags wailsassets` is mandatory for the production/embed build (prior-phase lesson; `build.yml:102`).
- The wails desktop build is **not** part of `go build ./...` output verification for these bumps — only `go build`/`vet`/`test` + frontend build. Full `wails build` matters only for #104 (deferred).

### Packaging reality check
DEP-01 says "deb/rpm packaging still builds," but the pipeline **only produces `.deb`** — `release.yml:276` runs `nfpm package --packager deb` and there is **no `--packager rpm` anywhere**. nfpm 2.47.0 *can* produce rpm, but this repo doesn't. Treat the #105 gate as **deb-only**. [VERIFIED: grep of release.yml]

### Merge / branch strategy (answers planner Q4 & Q5)

- All 10 Dependabot PRs `base=main`. `v4.2-funnel-sharing` is **376 commits ahead of main, 0 behind**. [VERIFIED: `git rev-list --left-right --count`]
- Active work + the eventual milestone ship happen on `v4.2-funnel-sharing`. These bumps must be verified against the **actual v4.2 code** (Funnel feature), not stale `main`.
- **Recommended:** apply each low-risk bump **onto `v4.2-funnel-sharing`** (SHA edits for actions; `go get` for modules), verify locally, commit; then **close** the corresponding Dependabot PR with a comment ("applied in Phase 174 on v4.2-funnel-sharing"). This satisfies DEP-01's intent (the bump lands + is verified) without the divergence/conflict cost of merging into `main` and re-merging 376 commits.
- **Do NOT** `gh pr merge` the Dependabot PRs into `main`: their CI is stale+flaky, their base lacks the v4.2 commits, and it would force a `main → v4.2` back-merge.
- Confirm the intended merge target with the user during planning if there's any chance v4.2 is meant to ship via squash (the Dependabot commits would be absorbed anyway).

### Wave strategy (answers planner Q5)

- **Wave A — CI-action YAML edits (independent, batch together):** #114, #113, #103, #85. Pure workflow-file SHA swaps. No Go/frontend impact. Verify by `git grep` consistency + a lint/`actionlint` pass; real proof comes when CI runs on the branch. #114 and #103 touch `release.yml` only (no PR-CI exercise until a release).
- **Wave B — Go module bumps (isolate each build+test run):** #89 (websocket — webserver/relay tests), #106 (x/term — attach/statusbar), #105 (nfpm — deb packaging). Apply and verify **one at a time** so a failure is attributable. #89 and #106 are independent; #105 is orthogonal (release-tool only).
- **Wave C — Deferrals (DEP-02):** dependabot.yml ignore entries (2 ecosystems) + close #104/#88/#102 with rationale. Independent of Waves A/B.

## dependabot.yml ignore syntax (answers planner Q2)

Current file has one `ignore:` block, under the **gomod** ecosystem only, ignoring `go-webview2` entirely (Phase 145). Structure recap:

```yaml
updates:
  - package-ecosystem: "github-actions"   # ← add checkout ignore HERE
    ...
  - package-ecosystem: "gomod"            # ← wails + tailscale ignores go HERE
    ...
    ignore:
      - dependency-name: "github.com/wailsapp/go-webview2"   # existing
```

### Precise ignore entries to add (stop the risky bump WITHOUT freezing security patches)

**#104 wails/v2 → gomod block.** Deferring the specific 2.12.0 jump. To keep future patch/minor security fixes flowing while blocking the go-webview2-coupled bump, ignore only the versions at/after the break. Two valid forms:

```yaml
# Option A — block the specific problematic range (preferred: still allows a future 2.11.x patch)
- dependency-name: "github.com/wailsapp/wails/v2"
  versions: [">=2.11.0"]
# Option B — block minor+major, allow patch security bumps of the current line
- dependency-name: "github.com/wailsapp/wails/v2"
  update-types: ["version-update:semver-minor", "version-update:semver-major"]
```
> Note: the *root* cause is `go-webview2`, already fully ignored. wails is ignored here so Dependabot stops re-opening the 2.12.0 PR that would transitively force go-webview2 ≥1.0.22. When the coordinated webview2 upgrade happens, remove/relax this entry. `versions` form (Option A) is more surgical; `update-types` (Option B) is coarser. [CITED: docs.github.com/en/code-security/dependabot/dependabot-options-reference — `ignore`, `update-types`, `versions`]

**#88 tailscale.com → gomod block.** Deferring 1.100.0 pending Funnel re-UAT:
```yaml
- dependency-name: "tailscale.com"
  versions: [">=1.100.0"]
```
(Allows a 1.98.x/1.99.x patch if one appears; blocks the 1.100 jump. After post-ship re-UAT, remove.)

**#102 actions/checkout → github-actions block.** Deferring the v7 major only, keep v6 patch/minor security updates:
```yaml
ignore:
  - dependency-name: "actions/checkout"
    update-types: ["version-update:semver-major"]
```
This is the canonical "pin a major, allow patches" pattern — the github-actions ecosystem currently has **no** `ignore:` block, so this adds one.

**Ecosystem placement (critical):** wails + tailscale go under the **`gomod`** update; checkout goes under the **`github-actions`** update. Putting an action ignore under gomod (or vice-versa) silently does nothing.

## Changelog risk review (answers planner Q6)

| Bump | Verdict | Notes |
|------|---------|-------|
| coder/websocket 1.8.14→1.8.15 | **Low risk, patch** | Bug fixes + perf only, no API change. Notable behavior: *"transmit in single frame when compression enabled"* (#552) and read-path alloc reduction (#565). AgentHub relays raw PTY frames; if permessage-deflate is off (default), effectively no-op. Still gate on webserver/relay tests per DEP-01. [CITED: github.com/coder/websocket/releases/tag/v1.8.15] |
| golang.org/x/term 0.43.0→0.44.0 | **Very low risk** | Sole change is *"go.mod: update golang.org/x dependencies"* — no term API change. [CITED: github.com/golang/term compare v0.43.0...v0.44.0] |
| nfpm/v2 2.46.3→2.47.0 | **Low risk, minor** | New: RiscV64, RPM `Requires(Post)`, fang styled CLI output; one bug fix (tolerate empty overrides); x/crypto+x/net security dep updates. No breaking change to `nfpm package --packager deb`. `fang` may restyle stdout but not behavior. [CITED: github.com/goreleaser/nfpm/releases/tag/v2.47.0] |
| wails 2.10.2→2.12.0 | **DEFER — breaks Windows** | Requires go-webview2 **v1.0.22** vs pinned **v1.0.19**; ≥1.0.20 changes `Chromium.MessageCallback` signature and breaks `wails` own `desktop/windows/frontend.go` (Phase 145 comment in dependabot.yml). [VERIFIED: wails v2.12.0 go.mod requires go-webview2 v1.0.22; v2.10.2 requires v1.0.19] |
| checkout 6.0.3→7.0.0 | **DEFER — major** | Blocks fork-PR checkout for `pull_request_target`/`workflow_run`; ESM/runtime upgrade. build.yml uses plain `pull_request` so likely fine, but a major bump warrants a throwaway branch test. [CITED: github.com/actions/checkout/releases/tag/v7.0.0] |
| tailscale 1.98.3→1.100.0 | **DEFER — feature risk** | 2 minor versions; entire Funnel feature was built + live-UAT'd on 1.98.3. Bumping now would require full off-tailnet Funnel re-UAT before ship — out of scope for a pre-ship hygiene pass. |

## Live Dependabot PR State (answers planner Q1)

All 10 enumerated PRs are **OPEN** as of 2026-07-08. No drift from ROADMAP versions. No new/unlisted Dependabot PRs. [VERIFIED: `gh pr list --state open`]

| PR | Head SHA | mergeable | CI rollup | Real failure cause |
|----|----------|-----------|-----------|--------------------|
| #114 | 67c0aa87 | MERGEABLE | 9× SUCCESS | — (green) |
| #113 | b5f00aa2 | MERGEABLE | 8 pass / 1 fail | flaky `internal/daemon` chat-store test (NOT the bump) |
| #106 | edca47cd | UNKNOWN | 8 / 1 fail | flaky `playwright [e2e]` |
| #105 | 79b2233a | UNKNOWN | 8 / 1 fail | flaky `playwright [e2e]` |
| #104 | 21ddcdb2 | UNKNOWN | 7 / 2 fail | flaky daemon test (job aborted before wails build ran) + playwright |
| #103 | 9aecc2dc | UNKNOWN | 8 / 1 fail | flaky `playwright [e2e]` |
| #102 | 735a070f | UNKNOWN | 9× SUCCESS | — (green) |
| #89 | 74f13667 | UNKNOWN | 5 / 4 fail | flaky `internal/files` race test + windows daemon flake + playwright |
| #88 | 89e53926 | UNKNOWN | 6 / 3 fail | windows daemon flake + playwright |
| #85 | eece06c9 | UNKNOWN | 6 / 3 fail | windows daemon flake + playwright |

`mergeable: UNKNOWN` = stale (PR base `main` unchanged but GitHub hasn't recomputed; these PRs are weeks old). Re-checkable via `gh pr checkout`.

## Common Pitfalls

### Pitfall 1: Trusting the red CI badge
**What goes wrong:** Treating a Dependabot PR's FAILURE status as proof the bump breaks something, and either force-merging the "green" ones blindly or skipping "red" safe ones.
**Why it happens:** The failing legs are **pre-existing flaky tests unrelated to any bump** — proven by the same failures on CI-YAML-only PRs (#85, #103) that cannot touch Go code.
**Known flaky tests to expect (and discount):**
- `playwright [e2e]` — flaky across nearly all PRs.
- `internal/daemon`: `TestEngineChatStoreFor_FailedNewChatStore` (path mkdir), `TestExitEvent_ListSessions_ExitCodePopulatedForStopped` (documented Windows/`cat` flake), `TestAliasStoreFilePerms` (Windows 0600 mode — documented v4.1).
- `internal/files`: `TestWrite_TwoWritersIfMatchRace` — `precondFailCount = 0; want exactly 1` race precondition flake.
**How to avoid:** Verify **locally** on `v4.2-funnel-sharing` with the gate commands. A local green `go test -short ./...` + targeted race tests is the real signal. Re-running CI after applying on the up-to-date branch will likely also clear most flakes.
**Warning signs:** A "failure" whose failing test lives in a package the bump doesn't touch (e.g. pnpm-action bump "failing" a Go daemon test).

### Pitfall 2: Assuming the go-webview2 ignore protects against the wails bump
**What goes wrong:** Believing #104 wails is safe to merge because `go-webview2` is already ignored.
**Why it happens:** Dependabot `ignore` only suppresses *Dependabot-opened PRs* for that package; it does **not** cap a transitive **minimum-version requirement**. wails 2.12.0's `go.mod` requires go-webview2 ≥1.0.22, so `go get wails@2.12.0 && go mod tidy` would bump go-webview2 to 1.0.22 regardless of the ignore — breaking the Windows build.
**How to avoid:** Defer #104 (add the wails ignore entry); do not run `go get wails@v2.12.0`.

### Pitfall 3: Missing occurrences of a bumped action
**What goes wrong:** Editing `setup-go`/`pnpm` in `build.yml` but not `e2e.yml`/`release.yml`, leaving inconsistent pins.
**How to avoid:** `grep -rn '<action>' .github/workflows/*.yml` and update every hit, or `gh pr checkout <pr>` to take Dependabot's complete file(s).

### Pitfall 4: Ignore entry under the wrong ecosystem
**What goes wrong:** Putting `actions/checkout` ignore under `gomod`, or `wails` under `github-actions` — silently ineffective; the PR re-opens next Monday.
**How to avoid:** actions → `github-actions` block; Go modules → `gomod` block.

## Runtime State Inventory

Not applicable — this is a dependency/CI-config phase. No stored data, live-service config, OS-registered state, secrets, or build artifacts embed a renamed string. The only durable state touched is `go.mod`/`go.sum` and `.github/workflows/*.yml` + `.github/dependabot.yml` (all in git). **None — verified by scope inspection.**

## Validation Architecture

> nyquist_validation not set in config → treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (+ `-race -short`); frontend `vitest`; `tsc` type-gate; `nfpm` deb packaging |
| Config file | `go.mod` (Go); `frontend/package.json` + vitest config; `build.yml` = CI source of truth |
| Quick run command | `go build ./... && go vet ./... && go test -short ./internal/webserver/... ./internal/relay/...` |
| Full suite command | `go test -race -short ./...` (non-Windows) + `cd frontend && pnpm install && pnpm test && pnpm exec tsc && pnpm exec vite build` |

### Phase Requirements → Test Map
| Req | Behavior | Test Type | Automated Command | Exists? |
|-----|----------|-----------|-------------------|---------|
| DEP-01 #89 | websocket bump keeps relay/webserver green | unit/integration | `go test -race -short ./internal/webserver/... ./internal/relay/...` | ✅ |
| DEP-01 #106 | x/term bump keeps attach/statusbar green | unit | `go test -short ./internal/attach/... ./internal/statusbar/...` | ✅ |
| DEP-01 #105 | nfpm bump still builds a `.deb` | packaging | `go install .../nfpm@<ver> && nfpm package --packager deb` (per release.yml:255-277) | ✅ (CI release.yml) |
| DEP-01 actions | workflow YAML still valid after SHA swaps | lint | `actionlint .github/workflows/*.yml` (if available) + branch CI run | ⚠️ actionlint optional — Wave 0 |
| DEP-02 | ignore entries present + PRs closed | manual/CLI | inspect `dependabot.yml`; `gh pr view <n> --json state` | ✅ |

### Sampling Rate
- **Per bump (Wave B):** `go build ./... && go vet ./... && go test -short <affected pkgs>` before commit.
- **Per wave merge:** full `go test -race -short ./...` + frontend `tsc && vite build`.
- **Phase gate:** full suite green locally on `v4.2-funnel-sharing` + (optionally) push branch and confirm CI legs green (discount the known flakes) before closing PRs.

### Wave 0 Gaps
- [ ] `actionlint` not confirmed installed — either install for local YAML validation or rely on branch CI to validate workflow edits.
- Otherwise: None — existing Go + frontend test infrastructure covers the bumps.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | Go module bumps, build/test | ✓ | go1.26.5 (go.mod requires 1.26.3) | — |
| node | frontend build/test | ✓ | v24.14.1 | — |
| pnpm | frontend install/test | ✓ | 9.15.9 | — |
| gh CLI | inspect/close Dependabot PRs | ✓ | (authenticated to scottkw/agenthub) | — |
| nfpm | #105 deb-packaging gate | ✗ (installable) | — | `go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.47.0` |
| actionlint | workflow YAML lint | ? (unconfirmed) | — | rely on branch CI to validate |
| Windows runner | wails Windows build (#104) | ✗ (CI-only) | — | N/A — #104 deferred, no local Windows build needed |

**Missing with no fallback:** none blocking. **Missing with fallback:** nfpm (go install), actionlint (branch CI).

## Security Domain

Supply-chain / dependency hygiene phase. Standard ASVS app categories (auth/session/access/input/crypto) are **not applicable** — no application code paths change. Relevant supply-chain controls:

| Concern | Control (already in place / to preserve) |
|---------|------------------------------------------|
| Action pinning | All actions are **SHA-pinned** with version comments — preserve this; copy Dependabot's exact SHA (don't switch to floating tags). |
| Provenance | `attest-build-provenance` bump (#114) is itself a supply-chain integrity tool — verify its new SHA `0f67c3f4...` matches upstream v4.1.1 before applying. |
| Deferred security patches | Use surgical `versions`/`update-types` ignores (above) so deferring wails/tailscale/checkout does **not** freeze future security patches on those lines. |
| Transitive pins | `go mod tidy` after each `go get` — review `go.sum` diff for unexpected transitive bumps (esp. anything pulling go-webview2 ≥1.0.20). |

No STRIDE threats introduced; the main risk is a poisoned action SHA — mitigated by copying SHAs from Dependabot's own diff (which resolves them against the upstream tag).

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| Merge Dependabot PRs on CI-green alone | Verify locally when CI is known-flaky | Avoids both false-block (safe PR shows red) and false-pass |
| Ignore whole package to stop a bump | Surgical `versions`/`update-types` ignore | Keeps security patches flowing on deferred deps |

**Deprecated/outdated:** none relevant.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Applying bumps on `v4.2-funnel-sharing` + closing the PRs satisfies DEP-01 "merge" intent | Merge strategy | If the user requires the literal Dependabot commits on `main`, plan must instead merge to main + back-merge — confirm in planning |
| A2 | coder/websocket single-frame-compression change is a no-op for AgentHub (permessage-deflate off) | Changelog review | If compression is enabled somewhere, relay frame behavior could shift — webserver/relay tests are the guard |
| A3 | The `playwright [e2e]` + daemon/files failures are flaky, not the bumps | Pitfall 1 | If a specific bump genuinely regresses, local per-bump testing (Wave B isolation) will catch it |
| A4 | `actionlint` may not be installed | Environment | Only affects optional local YAML lint; branch CI validates regardless |

## Open Questions

1. **Merge target for the low-risk bumps.**
   - What we know: all Dependabot PRs base=main; active work is on v4.2-funnel-sharing (376 ahead).
   - What's unclear: whether the milestone ships v4.2 by merging to main (bumps ride along) or the user wants Dependabot commits landed on main independently.
   - Recommendation: apply on v4.2-funnel-sharing + close PRs with rationale; confirm with user during planning (skip_discuss is on, so surface this in the plan).

2. **Should the 3 deferred PRs be closed now, or left open with a defer label?**
   - What we know: DEP-02 says "close the PRs citing this phase."
   - Recommendation: follow the requirement — add ignore entries first (so they don't re-open), then close #104/#88/#102 with a comment linking Phase 174 rationale.

## Sources

### Primary (HIGH confidence)
- `gh pr list/view/diff/checks` on scottkw/agenthub — live PR state, SHAs, CI conclusions (2026-07-08)
- `gh api .../branches/main/protection` — required status checks
- `gh run view --job <id> --log(-failed)` — actual failure causes (flaky tests identified)
- wails v2.10.2 vs v2.12.0 `go.mod` (raw.githubusercontent) — go-webview2 v1.0.19 vs v1.0.22 requirement
- Repo files: `.github/dependabot.yml`, `.github/workflows/{build,e2e,release}.yml`, `go.mod`, `go.sum`, `.planning/ROADMAP.md`

### Secondary (MEDIUM confidence)
- github.com/coder/websocket/releases/tag/v1.8.15 — changelog
- github.com/golang/term compare v0.43.0...v0.44.0 — changelog
- github.com/goreleaser/nfpm/releases/tag/v2.47.0 — changelog
- github.com/actions/checkout/releases/tag/v7.0.0 — changelog
- docs.github.com Dependabot options reference — ignore/update-types/versions syntax

### Tertiary (LOW confidence)
- none

## Metadata

**Confidence breakdown:**
- Live PR state / SHAs / versions: HIGH — directly from `gh`.
- wails↔go-webview2 coupling: HIGH — verified against both wails go.mod files.
- Flaky-CI diagnosis: HIGH — confirmed by failure logs + cross-PR pattern.
- Changelog risk assessment: MEDIUM — release notes reviewed, not line-by-line diffed.
- Merge-target recommendation: MEDIUM — sound but depends on user's ship intent.

**Research date:** 2026-07-08
**Valid until:** 2026-07-15 (PR state is live and can change if Dependabot re-runs or PRs are touched; re-verify `gh pr list` at plan time)
