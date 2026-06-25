# Phase 143: Regression Test Program - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-21
**Phase:** 143-regression-test-program
**Areas discussed:** Labeling & traceability map, Merge-gate mechanism, Coverage-gap scope & bar, Checklist & convention home

---

## Labeling & traceability map

| Option | Description | Selected |
|--------|-------------|----------|
| Whole suite + manifest | Treat the whole CI suite as the regression suite; name groups via a manifest doc; no re-tagging 459 files | ✓ |
| Build tags + vitest tags | Add `//go:build regression` + vitest tags to make the suite machine-selectable | |
| Directory/naming convention | Relocate/rename regression tests into a known path | |

| Option | Description | Selected |
|--------|-------------|----------|
| Hand-maintained Markdown table | Req→test table in the testing doc; zero tooling | ✓ |
| Machine-checkable YAML/JSON + validator | Structured registry validated in CI | |
| Inline annotations, grep-generated | Req IDs in test names/comments, map generated | |

| Option | Description | Selected |
|--------|-------------|----------|
| v4.0 release-critical behaviors | Map Hub, cross-surface, v4.0 reqs | ✓ |
| All REQUIREMENTS.md IDs (full history) | Trace all 142 prior phases | |

**User's choice:** Whole suite + manifest; hand-maintained Markdown table; v4.0 release-critical scope.
**Notes:** User asked for recommendations on format/scope. Agreed to add a ~10-line CI check asserting every test path referenced in the table still exists — chosen as a cheap substitute for a YAML schema validator (catches renamed/deleted-test drift).

---

## Merge-gate mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Required checks on existing build.yml + e2e.yml | Mark existing jobs as required status checks; add path-check step | ✓ |
| New consolidated regression.yml | One workflow running Go+vitest+Playwright as a single check | |

| Option | Description | Selected |
|--------|-------------|----------|
| Required status checks, no PR requirement, admin push allowed | Protect main for checks; preserve direct-to-main doc commits | ✓ |
| PR-required + status checks (strict) | Code AND docs land via PR; no direct push | |
| CI-only convention, no branch protection | Honor-system; enforce nothing in GitHub settings | |

| Option | Description | Selected |
|--------|-------------|----------|
| execute-phase runs gh api, with confirmation checkpoint | Apply branch protection during execute, with a go/no-go pause | ✓ |
| Runbook only — user applies manually | Document command; user runs it | |

**User's choice:** Required checks on existing workflows; status-checks-required with admin push allowed; applied via `gh api` at execute with a confirmation checkpoint.
**Notes:** Grounded on live state — `main` currently has NO branch protection; repo `scottkw/agenthub`, viewer ADMIN, `gh` authed. Enforcement model chosen specifically to preserve GSD direct-to-main `.planning/` doc commits.

---

## Coverage-gap scope & bar

| Option | Description | Selected |
|--------|-------------|----------|
| Gap-analysis pass against the v4.0 traceability map | Enumerate release-critical behaviors, close unmapped ones | ✓ |
| Numeric coverage threshold (e.g. 70%) | Write tests to hit a coverage % | |
| Opportunistic / Claude's discretion | Add tests where gaps appear | |

| Option | Description | Selected |
|--------|-------------|----------|
| ≥1 automated test per release-critical flow with no coverage | Behavior-focused bar | ✓ |
| Numeric coverage threshold enforced in CI | Hard coverage % gate | |

| Option | Description | Selected |
|--------|-------------|----------|
| Test parity at the data/contract seam | Daemon API + adapters; Playwright for web | ✓ |
| Full 3-surface UI e2e in CI | Drive GUI+CLI+web end-to-end | |
| Web surface e2e only | Automate web only; leave GUI/CLI to manual | |

**User's choice:** Gap-analysis-driven; ≥1 test per uncovered release-critical flow; cross-surface parity tested at the contract seam.
**Notes:** Cross-surface decision factored in documented automation limits (no PTY in `:34115` wails-dev bridge; web-share WS blocks input) and the standing "cross-surface parity is release-blocking" rule.

---

## Checklist & convention home

| Option | Description | Selected |
|--------|-------------|----------|
| Single TESTING.md at repo root | Manifest + map + checklist + convention in one file | ✓ |
| Split: TESTING.md + REGRESSION-CHECKLIST.md | Reference vs living checklist separated | |
| Under docs/ | docs/TESTING.md etc. | |

| Option | Description | Selected |
|--------|-------------|----------|
| Leave old UAT logs in place; new checklist supersedes | Don't delete historical artifacts | ✓ |
| Archive into .planning/archive/ | Move old logs to archive | |
| Delete them | Remove superseded logs | |

| Option | Description | Selected |
|--------|-------------|----------|
| TESTING.md section + pointer in new repo-level CLAUDE.md | Canonical in TESTING.md, surfaced to Claude each session | ✓ |
| TESTING.md section only | Convention in TESTING.md alone | |
| CONTRIBUTING.md | Convention in a new CONTRIBUTING.md | |

**User's choice:** Single repo-root TESTING.md; leave old per-phase UAT logs in place; convention in TESTING.md + pointer in a new repo-level CLAUDE.md.
**Notes:** No TESTING.md/CONTRIBUTING.md/docs/ exist today; no repo-level CLAUDE.md (only the shared parent one). Honors the standing "don't delete test artifacts early" rule.

## Claude's Discretion

- TESTING.md section ordering and table column layout
- Exact shell of the path-existence CI check and its host step in build.yml
- Which specific flows make the gap list and which test layer proves each
- Wording of the convention and the agenthub/CLAUDE.md pointer

## Deferred Ideas

- Machine-checkable YAML/JSON traceability registry + coverage-% CI gate (premature tooling for now)
- Full 3-surface UI e2e automation (infeasible today)
- Strict PR-required branch protection on main (revisit if team grows)
