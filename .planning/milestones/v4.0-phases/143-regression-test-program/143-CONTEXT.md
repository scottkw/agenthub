# Phase 143: Regression Test Program - Context

**Gathered:** 2026-06-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Formalize AgentHub's existing automated tests into a labeled **regression suite**
that runs in CI as a **merge gate**, close the automated coverage gaps introduced
by the v4.0 surface, replace the scattered per-phase UAT logs with a **single
maintained manual checklist**, and document a **standing convention** so every
future phase adds its tests to the right group.

**What already exists (do NOT rebuild):**
- **Go:** 344 `*_test.go` files, run via `go test -race -short ./...` in `.github/workflows/build.yml` (on push + PR)
- **Frontend:** 115 vitest files, run via `pnpm test` (`vitest run`) in `build.yml`
- **Playwright:** 7 e2e specs (`frontend/e2e/*.spec.ts`), run via `pnpm exec playwright test` in `.github/workflows/e2e.yml` (push/PR to `main`, cross-browser chromium/firefox/webkit)
- **Build-script tests:** `tests/build-script.test.sh` (run in build.yml)

So tests already *execute* on PR — this phase adds **labeling, a traceability map,
gate enforcement, gap-closure, the manual checklist, and the convention**. It does
NOT re-author the existing suite.

**In scope:** TEST-01 (label + map), TEST-02 (merge gate), TEST-03 (Hub +
cross-surface coverage gaps), TEST-04 (single manual checklist), TEST-05 (standing
convention).

**Out of scope:** rewriting/relocating existing tests; coverage of archived
pre-v4.0 milestones; full 3-surface UI e2e automation.

</domain>

<decisions>
## Implementation Decisions

### Suite Labeling & Traceability Map (TEST-01)
- **D-01:** The **entire CI suite IS the regression suite** — do NOT add Go build tags, vitest tags, or relocate/rename any of the 459 test files. Delineate the suite via a **manifest** in `TESTING.md` that names each group (Go unit/integration, vitest, Playwright e2e, build-script) and states what each guards.
- **D-02:** The requirement→test traceability map is a **hand-maintained Markdown table** in `TESTING.md` (columns: requirement ID → test file/name → group). No YAML/JSON registry, no inline-annotation generator.
- **D-03:** Add a **lightweight CI check (~10 lines of shell)** that asserts every test *path* referenced in the traceability table still exists on disk — catches the only drift that bites (renamed/deleted tests silently making the map lie) without a structured-schema validator. This check is part of the gate.
- **D-04:** Map **scope = v4.0 release-critical behaviors** (Hub, cross-surface GUI/CLI/web, v4.0 requirement IDs). Do NOT trace all 142 prior phases / full REQUIREMENTS.md history.

### Merge Gate (TEST-02)
- **D-05:** **Reuse existing CI** — mark the existing `build.yml` test job (Go + vitest + build-script + the new path-check) and the `e2e.yml` Playwright job as **required status checks**. Do NOT create a new consolidated `regression.yml` (avoids duplicated compute/config).
- **D-06:** **Enforcement model:** protect `main` to **require the test status checks to pass**, but **do NOT require pull requests**, and **allow admin direct push**. This preserves the GSD direct-to-`main` `.planning/` doc-commit flow. Convention: code changes land via PR with green CI. (Current state: `main` has NO branch protection.)
- **D-07:** **Application:** execute-phase applies branch protection via `gh api` (repo is `scottkw/agenthub`, viewer is ADMIN, `gh` is authed as scottkw), **pausing at a confirmation checkpoint** before mutating repo settings. The exact `gh api` command is also recorded in `TESTING.md` for reproducibility. Branch-protection config is a GitHub repo setting and lives outside the repo files.

### Coverage Gaps (TEST-03)
- **D-08:** Identify gaps via a **gap-analysis pass against the v4.0 traceability map** — enumerate release-critical v4.0 behaviors, mark which already have automated coverage, close only the unmapped ones.
- **D-09:** Bar for "closed" = **≥1 automated test per release-critical flow that currently has none**, recorded in the traceability map. No numeric coverage-% gate in CI (avoids coverage theater + brittle metric-coupled gate).
- **D-10:** **Cross-surface (GUI/CLI/web) parity is tested at the data/contract seam** — assert all three surfaces consume the same daemon endpoints/adapters via tests at that seam (Go daemon API + frontend adapters), plus Playwright for the web surface where it works. Do NOT attempt full 3-surface UI e2e in CI — it's brittle/infeasible given the documented automation limits (no PTY in the wails-dev `:34115` bridge; web-share WS blocks automated input).

### Manual Checklist & Convention Home (TEST-04, TEST-05)
- **D-11:** **Single `TESTING.md` at repo root** is the canonical home, holding: the suite manifest, the traceability table, the manual regression checklist, and the standing convention. Single source of truth, discoverable at root.
- **D-12:** The **manual regression checklist** covers all current release-critical behaviors (the human-intervention items that can't be automated — e.g. native GUI/CLI flows, AirDrop'd signed-build checks, remote-peer UATs). It replaces the scattered per-phase UAT logs as the living doc going forward.
- **D-13:** **Leave existing per-phase UAT logs in place** (`137/138/141-HUMAN-UAT.md` + verification logs) as historical record — do NOT delete or move them (honors the standing "don't delete test artifacts early" rule). Optionally add a one-line note that per-phase logs are historical and `TESTING.md` is now canonical.
- **D-14:** The **standing convention** (every future phase adds its regression tests to the appropriate group: automated vs. human-intervention, and updates the traceability map) is documented as a section in `TESTING.md` **plus a pointer line in a new repo-level `agenthub/CLAUDE.md`** so the rule is surfaced to Claude every session and applied each phase. (Today there is no repo-level CLAUDE.md — only the shared cross-project `/Users/ken/dev/CLAUDE.md`, which is the wrong home for agenthub-specific rules.)

### Claude's Discretion
- Exact `TESTING.md` section ordering and table column layout.
- Exact shell of the path-existence CI check and which `build.yml` step hosts it.
- Which specific release-critical flows make the gap list (subject to the gap-analysis pass) and which test layer (Go/vitest/Playwright) best proves each.
- Wording of the convention and the `agenthub/CLAUDE.md` pointer.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 143 section (goal, success criteria, Depends on Phase 142)
- `.planning/REQUIREMENTS.md` — TEST-01 … TEST-05 definitions and traceability rows

### Existing CI / test execution (the suite being formalized)
- `.github/workflows/build.yml` — Go (`go test -race -short ./...`), vitest (`pnpm test`), build-script tests; runs on push + PR
- `.github/workflows/e2e.yml` — Playwright cross-browser e2e on push/PR to `main`
- `tests/build-script.test.sh` — build-script test harness
- `frontend/package.json` — `test` (`vitest run`) and `test:coverage` scripts
- `frontend/playwright.config.ts` — Playwright config; `frontend/e2e/` specs + fixtures

### Cross-surface seam (for parity tests — D-10)
- `internal/` Go packages exposing the daemon API (capability, tailnet, webserver, statusbar, pty)
- `frontend/src/lib/` adapters (e.g. `remoteAdapter.ts`, `relayClient.ts`, `hubGroups.ts`, `hubStatus.ts`) and their existing `*.test.ts`

### Scattered UAT logs being superseded (TEST-04 — leave in place)
- `.planning/phases/137-share-modal-cap-model/137-HUMAN-UAT.md`
- `.planning/phases/138-hub-first-navigation/138-HUMAN-UAT.md`
- `.planning/phases/141-redesign-implementation/141-HUMAN-UAT.md`
- (plus per-phase `*-VERIFICATION.md` logs across `.planning/phases/`)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 344 Go tests + 115 vitest tests + 7 Playwright specs already green in CI — the suite to label, not rebuild.
- Frontend lib adapters already have unit tests (`hubGroups.test.ts`, `hubStatus.test.ts`, `remoteAdapter.test.ts`, `relayClient.test.ts`) — natural anchors for the contract-seam parity tests (D-10).
- `build.yml` and `e2e.yml` already run on push + PR — only the *required-check* enforcement is missing.

### Established Patterns
- CI runs Go with `-race -short`; frontend via pnpm; Playwright cross-browser (chromium/firefox/webkit).
- `.planning/` docs commit directly to `main` (GSD flow) — the enforcement model (D-06) must not break this.

### Integration Points
- Branch protection (`gh api repos/scottkw/agenthub/branches/main/protection`) — currently absent; gate enforcement attaches here.
- New `TESTING.md` (repo root) — the consolidation point for manifest + map + checklist + convention.
- New repo-level `agenthub/CLAUDE.md` — pointer to the standing convention (D-14).

</code_context>

<specifics>
## Specific Ideas

- Path-existence CI check should fail loudly when a test named in the traceability table no longer exists (D-03) — this is the chosen substitute for a heavier YAML schema validator.
- Branch protection must allow admin direct push and not require PRs, specifically to preserve GSD's direct-to-`main` doc commits (D-06).
- Cross-surface automation walls are real and documented: no PTY in the `:34115` wails-dev bridge; web-share WS blocks automated input. Parity is therefore verified at the daemon/adapter contract seam, not via full UI e2e (D-10).

</specifics>

<deferred>
## Deferred Ideas

- Machine-checkable YAML/JSON traceability registry + full coverage-% CI gate — considered and rejected for now (premature tooling for a solo+agent project); revisit if the project gains contributors.
- Full 3-surface UI e2e automation (native GUI + CLI PTY + web) — infeasible today; would need automation-harness work beyond this phase.
- Strict PR-required branch protection on `main` — deferred in favor of the admin-push-allowed model; revisit if/when team grows.

None of these belong in Phase 143.

</deferred>

---

*Phase: 143-regression-test-program*
*Context gathered: 2026-06-21*
