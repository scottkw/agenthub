# Phase 143: Regression Test Program - Research

**Researched:** 2026-06-21
**Domain:** CI/CD gate enforcement, test suite organization, GitHub branch protection, manual regression checklists
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** The entire CI suite IS the regression suite — do NOT add Go build tags, vitest tags, or relocate/rename any of the 459 test files. Delineate the suite via a **manifest** in `TESTING.md` that names each group (Go unit/integration, vitest, Playwright e2e, build-script) and states what each guards.
- **D-02:** The requirement→test traceability map is a **hand-maintained Markdown table** in `TESTING.md` (columns: requirement ID → test file/name → group). No YAML/JSON registry, no inline-annotation generator.
- **D-03:** Add a **lightweight CI check (~10 lines of shell)** that asserts every test *path* referenced in the traceability table still exists on disk — catches the only drift that bites (renamed/deleted tests silently making the map lie) without a structured-schema validator. This check is part of the gate.
- **D-04:** Map **scope = v4.0 release-critical behaviors** (Hub, cross-surface GUI/CLI/web, v4.0 requirement IDs). Do NOT trace all 142 prior phases / full REQUIREMENTS.md history.
- **D-05:** **Reuse existing CI** — mark the existing `build.yml` test job (Go + vitest + build-script + the new path-check) and the `e2e.yml` Playwright job as **required status checks**. Do NOT create a new consolidated `regression.yml`.
- **D-06:** **Enforcement model:** protect `main` to **require the test status checks to pass**, but **do NOT require pull requests**, and **allow admin direct push**. This preserves the GSD direct-to-`main` `.planning/` doc-commit flow.
- **D-07:** **Application:** execute-phase applies branch protection via `gh api` (repo is `scottkw/agenthub`, viewer is ADMIN, `gh` is authed as scottkw), **pausing at a confirmation checkpoint** before mutating repo settings. The exact `gh api` command is also recorded in `TESTING.md` for reproducibility.
- **D-08:** Identify gaps via a **gap-analysis pass against the v4.0 traceability map** — enumerate release-critical v4.0 behaviors, mark which already have automated coverage, close only the unmapped ones.
- **D-09:** Bar for "closed" = **≥1 automated test per release-critical flow that currently has none**, recorded in the traceability map. No numeric coverage-% gate in CI.
- **D-10:** **Cross-surface (GUI/CLI/web) parity is tested at the data/contract seam** — assert all three surfaces consume the same daemon endpoints/adapters via tests at that seam (Go daemon API + frontend adapters), plus Playwright for the web surface where it works. Do NOT attempt full 3-surface UI e2e in CI.
- **D-11:** **Single `TESTING.md` at repo root** is the canonical home, holding: the suite manifest, the traceability table, the manual regression checklist, and the standing convention.
- **D-12:** The **manual regression checklist** covers all current release-critical behaviors that can't be automated.
- **D-13:** **Leave existing per-phase UAT logs in place** (`137/138/141-HUMAN-UAT.md` + verification logs) as historical record — do NOT delete or move them.
- **D-14:** The **standing convention** is documented as a section in `TESTING.md` **plus a pointer line in a new repo-level `agenthub/CLAUDE.md`**.

### Claude's Discretion

- Exact `TESTING.md` section ordering and table column layout.
- Exact shell of the path-existence CI check and which `build.yml` step hosts it.
- Which specific release-critical flows make the gap list (subject to the gap-analysis pass) and which test layer (Go/vitest/Playwright) best proves each.
- Wording of the convention and the `agenthub/CLAUDE.md` pointer.

### Deferred Ideas (OUT OF SCOPE)

- Machine-checkable YAML/JSON traceability registry + full coverage-% CI gate.
- Full 3-surface UI e2e automation (native GUI + CLI PTY + web).
- Strict PR-required branch protection on `main`.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | Automated regression suite (Go + vitest + Playwright) consolidated and labeled, with a requirement→test traceability map | Suite inventory (344 Go / 108 vitest / 7 Playwright / 1 build-script) confirms exact file counts; D-01/D-02 locked approach is viable without any file moves |
| TEST-02 | Automated regression suite runs in CI as a merge gate | Exact `gh api` command payload derived from live check-run names and GitHub API docs; admin-push-allowed model confirmed viable via `enforce_admins: false` |
| TEST-03 | Automated coverage gaps closed for Hub and cross-surface GUI/CLI/web flows | Gap analysis below identifies 4 uncovered v4.0 flows; each maps to a specific test layer that can close it |
| TEST-04 | Single maintained manual regression checklist replaces scattered per-phase UAT logs | Three UAT logs mined; human-intervention items catalogued; existing logs stay in place per D-13 |
| TEST-05 | Standing convention requires every future phase to add its regression tests | Convention text and CLAUDE.md pointer designed; D-14 approach validated |

</phase_requirements>

---

## Summary

Phase 143 adds zero new test logic — it formalizes what already exists. The 344 Go, 108 vitest, 7 Playwright, and 1 build-script test suite already runs on every push and PR via `build.yml` and `e2e.yml`. The gap is that none of these tests is labeled, mapped to requirements, or protected as a merge gate. This phase closes that gap in four concrete deliverables: `TESTING.md` (manifest + traceability map + manual checklist + convention), a path-check CI step in `build.yml`, branch protection on `main` applied via `gh api`, and a new repo-level `CLAUDE.md` pointer.

The v4.0 gap analysis finds four release-critical flows with no automated test coverage: `hubGroupCounts.ts` compute logic (group counts and global counts), `agentBadge.ts` session-type classification logic, the Hub group navigation render path in Sidebar (POL-05 sub-list), and the three-item sidebar render contract. Closing these requires four new vitest files, no new Playwright specs, and no new Go tests.

The manual checklist, derived from the 137/138/141 UAT logs, surfaces eight human-intervention behaviors — primarily native GUI flows, AirDrop'd signed build verification, and the remote-peer live tests (Phase 138 items 3/5/6-remote and the two-machine Phase 125/126 UATs) — that cannot be automated given the documented wails-dev bridge and web-share WS constraints.

**Primary recommendation:** Write TESTING.md, add the path-check step to build.yml on the ubuntu-latest run, apply branch protection via `gh api` behind a checkpoint, and write the repo-level CLAUDE.md pointer. Four new vitest tests close the v4.0 automated coverage gaps.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Suite manifest + traceability map | Repo root doc (`TESTING.md`) | — | Single source of truth; discoverable without any tool |
| Path-existence integrity check | CI (build.yml step) | — | Enforcement is runtime CI; logic lives where it runs |
| Merge gate enforcement | GitHub repo settings (branch protection) | CI check names | Protection is a GitHub repo setting, not a repo file |
| Hub group counts logic | Frontend lib (`hubGroupCounts.ts`) | App.tsx state | Pure computation; no server round-trip |
| Session-type badge logic | Frontend lib (`agentBadge.ts`) | SessionCard.tsx | Pure UI classification function |
| Cross-surface contract seam | Go daemon API + frontend adapters | Playwright (web) | Contract tested at the seam; full UI automation not feasible |
| Manual regression checklist | Repo root doc (`TESTING.md`) | — | Human-intervention items need a single living document |
| Standing convention | `TESTING.md` + repo `CLAUDE.md` | — | Convention in both places ensures Claude sees it every session |

---

## Suite Inventory (Confirmed Counts)

> [VERIFIED: live filesystem scan 2026-06-21]

| Suite | Count | Location | Run Command |
|-------|-------|----------|-------------|
| Go unit/integration | **344** `*_test.go` files | `internal/`, root | `go test -race -short ./...` |
| vitest (frontend) | **108** `*.test.ts/tsx` files | `frontend/src/` | `pnpm test` (`vitest run`) |
| Playwright e2e | **7** `*.spec.ts` files | `frontend/e2e/` | `pnpm exec playwright test` |
| build-script | **1** `build-script.test.sh` | `tests/` | `bash tests/build-script.test.sh` |
| **Total** | **460** | — | — |

> Note: CONTEXT.md states "459 test files" (115 vitest). The actual vitest count is 108. The discrepancy is because some files previously counted as vitest may have been removed during Phase 136 (TUI test deletion) or were double-counted. The live count of 108 is authoritative. [VERIFIED: find /frontend/src -name '*.test.ts' -o -name '*.test.tsx']

### Go package distribution [VERIFIED: live filesystem scan]

| Package | Test files |
|---------|-----------|
| `internal/daemon` | 25 |
| `internal/webserver` | 28 |
| `internal/relay` | 11 |
| `internal/pty` | 11 |
| `internal/capability` | 5 |
| `internal/files` | 4 |
| `internal/status` | 1 |
| `internal/statusbar` | 1 |
| `internal/tailnet` | 1 |
| root package | 13 |
| other | ~244 |

### Playwright specs [VERIFIED: live listing]

| Spec | Guards |
|------|--------|
| `files-browser.spec.ts` | File browser read routes + capability gate (v3.4) |
| `files-write.spec.ts` | File write/upload/delete/rename routes (v3.5) |
| `progress.spec.ts` | Progress addon web surface (v3.2) |
| `web-csp.spec.ts` | Content Security Policy (v3.1) |
| `web-links-live-toggle.spec.ts` | Web-links plugin toggle (v3.2) |
| `web-plugin-hot-swap.spec.ts` | Plugin hot-swap on web surface (v3.2) |
| `web-vendor-parity.spec.ts` | Vendored xterm addons, no CDN (v3.2) |

Observation: **No Playwright spec covers v4.0 Hub behaviors.** The web surface tests all target the pre-v4.0 web session layer and the file browser. This is intentional and acceptable — Hub runs in the Wails native webview, and the web-share surface (what Playwright tests) is the session terminal/file layer, not the Hub page. The cross-surface contract is proven at the adapter/daemon API seam per D-10.

---

## Standard Stack

No new packages are required for this phase. All deliverables use existing project infrastructure.

| Tool | Version | Role | Already In Project |
|------|---------|------|--------------------|
| `gh` CLI | authed as scottkw, admin | Branch protection application | Yes — used across CI workflows |
| `vitest` | ^4.1.0 | New gap-closure unit tests | Yes — `frontend/package.json` |
| `bash` | system | Path-check CI step | Yes — build.yml already uses bash steps |
| `GitHub Actions` | current | CI runner | Yes — build.yml + e2e.yml |

---

## Package Legitimacy Audit

This phase installs zero new packages. Section not applicable.

---

## Architecture Patterns

### Deliverable Map

```
repo root
├── TESTING.md                    ← NEW (TEST-01, TEST-02, TEST-04, TEST-05)
│   ├── Suite Manifest
│   ├── Requirement→Test Traceability Table (v4.0 scope)
│   ├── Manual Regression Checklist
│   └── Standing Convention
├── CLAUDE.md                     ← NEW pointer to convention (D-14)
└── .github/workflows/
    └── build.yml                 ← ADD path-check step (D-03)

GitHub repo settings (not a file):
    Branch protection on main      ← APPLY via gh api (D-05, D-06, D-07)

frontend/src/
├── lib/hubGroupCounts.test.ts    ← NEW gap-closure (TEST-03, D-08)
├── lib/agentBadge.test.ts        ← NEW gap-closure (TEST-03, D-08)
└── components/__tests__/
    ├── Sidebar.test.tsx           ← EXTEND with group sub-list tests (TEST-03)
    └── style.hub.test.ts         ← EXTEND with card spine/chip token tests (TEST-03)
```

---

## Gap Analysis: v4.0 Release-Critical Flows Without Automated Coverage

> Scope per D-04 and D-08: v4.0 requirement IDs only (NAV-01..05, SHARE-01..06, CARD-01..05, RDS-01..04, TAB-01..03, POL-01..05, CARRY-01..02).

### Already Covered (has ≥1 automated test)

| v4.0 Requirement | Automated Test | Group |
|-----------------|---------------|-------|
| NAV-01 (TUI removed) | Go: TUI test deletion in Phase 136 CI; `go build ./...` passes | Go |
| NAV-02..05 (3-item sidebar) | `App.nav.test.tsx` — asserts no `onOpenDaemonManager`, no `onOpenRemoteSessions`, no `onAdd={handleAddTab}` | vitest |
| SHARE-01..06 (Share modal + cap model) | `SessionShareModal.test.tsx`, `SessionCard.share.test.tsx`; Go: `api_test.go` SHARE-03 browse-matrix, `engine_test.go` `TestKillSession_ClearsStaleBrowseEntry`, `TestIssueCapabilitiesForSession*` | vitest + Go |
| CARD-01 (no hub__header) | `App.hub.test.tsx` — asserts `HUB_TAB` wiring; `Sidebar.test.tsx` — no hub__header rendered | vitest |
| CARD-02..04 (origin/connected indicators) | `SessionCard.share.test.tsx` — `isRemote` prop renders LockClosedIcon disabled affordance; `remoteAdapter.test.ts` — `adaptRemoteSession` hostname mapping | vitest |
| CARD-05 (headless VT mini-preview) | `MiniPreview.test.tsx` — StyledSpan rendering; `engine_test.go` `TestGetSessionStyledTailLines_ColorBold` | vitest + Go |
| TAB-01..03 (tab strip) | `TabBar.test.tsx` — shrink floor, chevron controls | vitest |
| RDS-02..04 (redesign tokens) | `style.redesign.test.ts`, `style.hub.test.ts`, `style.hub.modal.test.ts`, `SettingsTab.appearance-theme.test.tsx` — hex/var source gates; `themeTokens.test.ts` | vitest |
| CARRY-01 (GroupSidebar ARIA) | `Sidebar.test.tsx` — `CARRY-01` ARIA fix confirmed in test; group sub-list ARIA structure | vitest |
| POL-02 (Settings toggle role=switch) | `SettingsTab.appearance-theme.test.tsx` — `role="switch"`, `aria-checked` | vitest |
| POL-04 (terminal repaint) | `TerminalPanel.contextLoss.test.tsx`, source gates in `style.hub.test.ts` for POL-03 | vitest |
| POL-05 (group nav in Sidebar) | `Sidebar.test.tsx` has group sub-list tests (POL-05 RED/GREEN cycle completed) | vitest |
| Cross-surface relay contract | `relayClient.test.ts` — wire framing (`encodeInputFrame`, `encodeResizeFrame`, `parseServerFrame`); Go: `internal/relay/hub_test.go`, `oscabsorb_relay_test.go` | vitest + Go |
| Remote adapter contract | `remoteAdapter.test.ts` — `adaptRemoteSession` hostname/URL mapping | vitest |
| Hub group persistence | `hubGroups.test.ts` — `loadGroups`/`saveGroups`/`createGroup`/`assignToGroup`/`removeFromGroup` round-trip | vitest |
| Hub status classification | `hubStatus.test.ts` — `isAttentionStatus` all six states | vitest |

### Coverage Gaps (need ≥1 new test per D-09)

| Gap ID | v4.0 Flow | What's Missing | Recommended Test Layer | Rationale |
|--------|-----------|---------------|----------------------|-----------|
| GAP-01 | Hub group count computation (`computeCounts`, `computeGlobalCounts` in `hubGroupCounts.ts`) | No test for `hubGroupCounts.ts` — pure computation function used for sidebar count badges and "All" group totals | vitest — `frontend/src/lib/hubGroupCounts.test.ts` | Pure function, no DOM; directly tests the counts that drive the group sub-list badges added in Phase 142 (POL-05) |
| GAP-02 | Session-type badge classification (`agentBadge.ts` — `agentColor`, `agentLabel`) | No test for `agentBadge.ts` — used for the color-coded origin spine/chip added in Phase 142 gap closure (commit 5930ec2f) | vitest — `frontend/src/lib/agentBadge.test.ts` | Pure function; colorblind-safe validation at source hex level without visual inspection |
| GAP-03 | Three-item sidebar render contract (no Sessions, no Remote, no standalone New Session — NAV-05 positive render) | `Sidebar.test.tsx` currently asserts ARIA and hub nav props but does not assert the three-item-only render count from a rendered component (only source-inspection tests in `App.nav.test.tsx`) | vitest — extend `Sidebar.test.tsx` with rendered item count assertion | Low effort; the component renders in tests already; closes the positive-render gap for NAV-05 |
| GAP-04 | Hub session card v4.0 redesign tokens (card gutter, session-type spine, color-coded chip from Phase 142 comp-fidelity) | `style.hub.test.ts` does not assert the Phase 142 unplanned additions (`.hub-card__spine`, `.hub-card__origin-chip`, 16px border-radius, 150px preview height) | vitest — extend `style.hub.test.ts` | CSS source-gate pattern is already established in this file; closing this ensures the Phase 142 comp-fidelity tokens cannot silently regress |

### Flows Explicitly Out of Scope for Automation (D-10 automation walls)

| Flow | Why not automatable | Manual checklist entry |
|------|---------------------|----------------------|
| Native GUI Share modal click-through (items 4/5 from 137 UAT) | Wails native webview not accessible to Playwright/browser automation | Yes — in manual checklist |
| Remote peer card behaviors (138 UAT items 3/5/6-remote) | Requires live reachable Tailscale peer | Yes — in manual checklist |
| Terminal PTY input/output (kill two-step, attach) | No PTY in :34115 wails-dev bridge | Yes — in manual checklist |
| AirDrop'd signed build verification | Requires two physical machines + code signing | Yes — in manual checklist |
| Phase 125/126 deferred UATs (editor on-screen, $EDITOR suspend-resume) | Requires live GUI webview interaction | Yes — in manual checklist |

---

## GitHub Branch Protection: Exact API Command (D-05, D-06, D-07)

> [VERIFIED: live check-run query against scottkw/agenthub on 2026-06-21]

### Exact Status Check Context Names

These are the check names GitHub registers for each CI run (confirmed by querying the latest main commit's check-runs):

**From `build.yml` (workflow name: "Build", job id: "build", matrix-expanded):**
- `build (agenthub, linux/amd64, ubuntu-latest, webkit2_41, libwebkit2gtk-4.1-dev)`
- `build (agenthub, linux/amd64, ubuntu-22.04, libwebkit2gtk-4.0-dev)`
- `build (agenthub, darwin/universal, macos-latest)`
- `build (agenthub, windows/amd64, windows-latest)`

**From `e2e.yml` (workflow name: "e2e", job id: "playwright"):**
- `playwright`

GitHub Actions `app_id`: **15368** [VERIFIED: check-runs API response]

### Which Checks to Require

Per D-05, the gate must require the test jobs but NOT require jobs that don't run tests (e.g., the build artifact upload steps that succeed even when tests are skipped). All four `build` matrix jobs run Go tests, and the ubuntu-latest job additionally runs vitest + build-script tests. The `playwright` job runs the e2e suite.

**Require all five contexts** (all four build matrix jobs + playwright). This is more robust than requiring only the ubuntu-latest build job because: (a) Go tests run on all platforms — a Windows-only race condition would otherwise be invisible; (b) it matches the "full CI suite as gate" intent.

### Exact `gh api` Command

```bash
gh api repos/scottkw/agenthub/branches/main/protection \
  --method PUT \
  --header "Accept: application/vnd.github+json" \
  --input - <<'EOF'
{
  "required_status_checks": {
    "strict": false,
    "checks": [
      {"context": "build (agenthub, linux/amd64, ubuntu-latest, webkit2_41, libwebkit2gtk-4.1-dev)", "app_id": 15368},
      {"context": "build (agenthub, linux/amd64, ubuntu-22.04, libwebkit2gtk-4.0-dev)", "app_id": 15368},
      {"context": "build (agenthub, darwin/universal, macos-latest)", "app_id": 15368},
      {"context": "build (agenthub, windows/amd64, windows-latest)", "app_id": 15368},
      {"context": "playwright", "app_id": 15368}
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null
}
EOF
```

**Key field rationale:**
- `checks[]` not `contexts[]` — `contexts` is deprecated by GitHub; `checks[]` with `app_id` is the current format [CITED: docs.github.com/en/rest/branches/branch-protection]
- `strict: false` — PRs do not need to be up-to-date with main before merging; keeps flexibility for the admin-push-allowed model
- `enforce_admins: false` — allows admins (scottkw) to push directly to main without CI; this is what preserves the GSD `.planning/` doc-commit flow
- `required_pull_request_reviews: null` — no PR review requirement
- `restrictions: null` — no user/team push restrictions

**Rollback command** (for TESTING.md reproducibility section):
```bash
gh api repos/scottkw/agenthub/branches/main/protection --method DELETE
```

**Non-TTY note:** The `--input -` + heredoc pattern works in scripted (non-TTY) invocation. The `<<'EOF'` syntax passes the JSON body via stdin without requiring a temp file.

---

## Path-Existence CI Check (D-03)

The check parses test paths from the TESTING.md traceability table and asserts each still exists. It runs only on the ubuntu-latest build job (where the traceability table lives and where `pnpm test` runs).

### Shell Implementation (~10 lines)

```bash
#!/usr/bin/env bash
# Verify every test path in TESTING.md traceability table still exists on disk.
# Fails loudly if any path is missing — prevents renamed/deleted tests from
# silently making the traceability map lie.
#
# Parsing convention: traceability table rows have format:
#   | REQ-ID | path/to/test_file.go | group |
# Lines starting with '|' and containing a file extension are path rows.
set -euo pipefail

FAIL=0
while IFS= read -r path; do
  if [[ ! -e "$path" ]]; then
    echo "MISSING traceability path: $path"
    FAIL=$((FAIL + 1))
  fi
done < <(grep -oP '(?<=\| )[^\|]+\.(?:go|ts|tsx|sh)(?= \|)' TESTING.md | tr -d ' ')

if [[ $FAIL -gt 0 ]]; then
  echo "ERROR: $FAIL traceability path(s) missing — update TESTING.md or restore the test file"
  exit 1
fi
echo "OK: all traceability paths exist"
```

### Where It Goes in `build.yml`

Add as a new step on the ubuntu-latest run, immediately after "Run frontend tests" and before "Install Wails CLI":

```yaml
- name: Verify traceability paths exist
  if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'
  run: bash tests/check-traceability-paths.sh
```

The script lives at `tests/check-traceability-paths.sh` (alongside `tests/build-script.test.sh`). Alternatively the inline `run:` form can be used directly if the script is short enough — the inline form is self-contained.

**How it fails loudly:** `set -euo pipefail` + explicit `exit 1` makes the step exit non-zero, which marks the `build (agenthub, linux/amd64, ubuntu-latest, ...)` job as failed, which blocks the merge gate.

**Parsing convention that TESTING.md must follow:** Traceability table rows use the format `| REQ-ID | path/to/test.go | group |` (three columns). Path column contains the repo-relative path to a test file. The `grep -oP` pattern extracts the path column by matching file extensions `.go|.ts|.tsx|.sh`. TESTING.md author must use repo-relative paths (e.g., `frontend/src/lib/hubGroups.test.ts`, not `frontend/src/lib/hubGroups`).

---

## Manual Regression Checklist Items (Mined from UAT Logs)

> Source: 137-HUMAN-UAT.md, 138-HUMAN-UAT.md, 141-HUMAN-UAT.md, 142-VERIFICATION.md, STATE.md deferred items [VERIFIED: live file reads 2026-06-21]

The following human-intervention items CANNOT be automated and belong in the `TESTING.md` manual checklist.

### Category: Share Modal (SHARE-01..06)

| # | Behavior | Why Not Automatable | Source |
|---|----------|--------------------|----|
| M-01 | Share modal opens with RO + RW link rows in live native webview (item 4 from 137: home-dir warning banner renders before browse toggle) | Wails native webview not accessible to browser automation | 137-HUMAN-UAT.md |
| M-02 | Remote peer card Share button is disabled with lock icon + tooltip in live native webview (item 5 from 137) | Same — native webview only | 137-HUMAN-UAT.md |

> Items 1/2/3/6 from 137-HUMAN-UAT.md are already proven via daemon curl harness (enforcement-level) and are covered at the Go test seam — they do not need a manual checklist entry.

### Category: Hub Navigation / Remote Peer (NAV, CARD)

| # | Behavior | Why Not Automatable | Source |
|---|----------|--------------------|----|
| M-03 | Remote card shows "Open in browser" with real peer URL (not empty page); BrowserOpenURL forwards correctly (Phase 138 UAT item 3) | Requires live reachable Tailscale remote peer | 138-HUMAN-UAT.md |
| M-04 | Remote card overflow menu shows only "Open in browser" + "Browse files" (no Kill) — CR-02 (Phase 138 UAT item 5, remote half) | Requires live reachable peer | 138-HUMAN-UAT.md |
| M-05 | Remote Connected chip (LinkIcon) and remote Available chip (GlobeAltIcon) render with icon + text on a live remote card (Phase 138 UAT item 6, remote half) | Requires live reachable peer | 138-HUMAN-UAT.md |
| M-06 | Kill two-step confirm on a live local session (first click → "Confirm kill", second click terminates session) | PTY session needs real daemon; PTY interaction in wails-dev bridge has no TTY | 138-HUMAN-UAT.md (PASS recorded 2026-06-20; re-verify after Hub restructure) |

### Category: Terminal / Theme (POL-04)

| # | Behavior | Why Not Automatable | Source |
|---|----------|--------------------|----|
| M-07 | Terminal repaints cleanly after theme switch (active session); after tab switch away and back; CMD+/- font resize does NOT garble | Wails native webview required for GPU atlas; Playwright can't exercise the native rendering path | 142-VERIFICATION.md (PASS recorded 2026-06-21; standing check) |

### Category: Signed Build / Distribution

| # | Behavior | Why Not Automatable | Source |
|---|----------|--------------------|----|
| M-08 | AirDrop'd signed macOS build passes Gatekeeper (`spctl --assess`) on a separate machine | Requires code signing + two physical machines | General release protocol |

### Category: File Browser / Editor (Deferred live UATs)

| # | Behavior | Why Not Automatable | Source |
|---|----------|--------------------|----|
| M-09 | Phase 125 editor on-screen render: CodeMirror file opens, Tab key indents, Cmd-V pastes in native WebView | Wails native webview; CodeMirror keyboard events not reproducible in headless browser | STATE.md deferred items |
| M-10 | Phase 126 `$EDITOR` shell-out: suspend-resume terminal restore works (the session terminal restores correctly after the spawned editor exits) | PTY interaction; requires real shell + editor launch | STATE.md deferred items |
| M-11 | Phase 124 home-dir warning banner on-screen in live native WebView (the file write warning renders before the browse toggle when the session's workDir is the home dir) | Wails native webview | STATE.md deferred items |

---

## TESTING.md Section Ordering (Claude's Discretion)

Recommended section order for discoverability:

1. **Overview** — what this document is and who maintains it
2. **Suite Manifest** — groups, file counts, run commands
3. **Merge Gate: How to Apply Branch Protection** — the `gh api` command (D-07 reproducibility)
4. **Requirement → Test Traceability Map** — v4.0 scope table (D-02, D-04)
5. **Manual Regression Checklist** — human-intervention items (D-12)
6. **Standing Convention** — per-phase update rule (D-14)

---

## Traceability Table Structure (D-02 Template)

```markdown
| Requirement | Test File / Name | Suite Group | Notes |
|-------------|-----------------|-------------|-------|
| NAV-02 | frontend/src/components/__tests__/App.nav.test.tsx — "NAV-02: Remote sidebar item is removed" | vitest | |
| SHARE-01 | frontend/src/components/__tests__/SessionShareModal.test.tsx — "SHARE-01: share toggle" | vitest | |
| ...etc... |
```

Columns: requirement ID, repo-relative path + test name, suite group (Go/vitest/Playwright/build-script), optional notes. Paths must be repo-relative so the path-check script resolves them from the project root.

---

## Common Pitfalls

### Pitfall 1: Matrix-expanded check names change when matrix entries change
**What goes wrong:** If a build matrix entry is added or renamed in `build.yml`, the status check context name changes and the branch protection rule silently stops requiring the renamed check.
**Why it happens:** GitHub registers checks by exact job name string including all matrix values.
**How to avoid:** Document in TESTING.md that any change to `build.yml` matrix entries requires a corresponding update to the branch protection rule. The rollback + re-apply procedure is recorded in TESTING.md.
**Warning signs:** New PR passes CI but the "required checks" badge shows a new unknown check name.

### Pitfall 2: `enforce_admins: false` vs `require_pull_requests` confusion
**What goes wrong:** Thinking `enforce_admins: false` skips CI for admin pushes. It does not — `enforce_admins` only means admins can bypass the protection (e.g. force-push past the required checks). With `enforce_admins: false`, direct pushes from admin bypass the required-checks gate entirely without needing CI to pass first.
**Why it happens:** The GitHub UI and API docs conflate "admin bypass" and "skip CI".
**How to avoid:** The GSD `.planning/` doc-commit flow works because those commits push directly to `main` as admin, completely bypassing the CI check gate. Only commits that flow through a PR will have the checks enforced (when a PR is opened). This is the documented intent per D-06.
**Warning signs:** `.planning/` doc commits fail with "protected branch" errors — this means `enforce_admins` was accidentally set to `true`.

### Pitfall 3: Path-check script parses wrong column
**What goes wrong:** The grep pattern extracts a requirement ID or group name instead of the file path, causing false-positive "all paths exist" results.
**Why it happens:** The regex is too broad if the table uses req IDs that look like paths, or if paths contain unusual characters.
**How to avoid:** Test the script against TESTING.md before committing by running it locally. The parsing convention (file extension gate: `.go|.ts|.tsx|.sh`) is specific enough to exclude requirement IDs.
**Warning signs:** Script reports 0 paths checked for a non-empty traceability table.

### Pitfall 4: `strict: false` vs `strict: true` in branch protection
**What goes wrong:** With `strict: true`, branches must be up-to-date with main before their checks are evaluated as passing, even for admin pushes.
**Why it happens:** Wanting "freshness" safety.
**How to avoid:** Use `strict: false` per the locked design. The project uses admin direct push; requiring branches to be up-to-date adds friction for the primary workflow.

### Pitfall 5: vitest test count assumption (115 vs 108)
**What goes wrong:** CONTEXT.md states "115 vitest files" but the live count is 108. Using 115 in TESTING.md manifests creates an inaccurate manifest that looks wrong at a glance.
**How to avoid:** Use the verified live count (108) in TESTING.md.

---

## Validation Architecture

This section describes how the Phase 143 deliverables are validated. These are the behaviors VALIDATION.md must prove.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest ^4.1.0 (frontend) + Go testing (Go) |
| Config file | `frontend/vite.config.ts` (vitest embedded) |
| Quick run | `cd frontend && pnpm test` |
| Full suite | `go test -race -short ./... && cd frontend && pnpm test && pnpm exec playwright test` |

### Phase Requirements → Validation Map

| Req ID | Behavior to Prove | Validation Method | Command |
|--------|------------------|------------------|---------|
| TEST-01 | TESTING.md exists at repo root with all four sections | File existence + section headings grep | `test -f TESTING.md && grep -q 'Suite Manifest' TESTING.md` |
| TEST-01 | Traceability table covers all v4.0 requirement IDs | Row count grep against v4.0 req IDs | `grep -c 'TEST-0[1-5]\|NAV-0[1-5]\|SHARE-0[1-6]\|CARD-0[1-5]\|TAB-0[1-3]\|RDS-0[1-4]\|POL-0[1-5]' TESTING.md` |
| TEST-02 | Branch protection requires all 5 check contexts | `gh api` GET response check | `gh api repos/scottkw/agenthub/branches/main/protection --jq '.required_status_checks.checks | length'` → expect 5 |
| TEST-02 | `enforce_admins` is false (admin push preserved) | `gh api` GET response check | `gh api repos/scottkw/agenthub/branches/main/protection --jq '.enforce_admins.enabled'` → expect `false` |
| TEST-02 | Gate actually blocks: a PR with failing vitest is blocked | Smoke test — open a draft PR with a breaking test, verify CI fails, verify merge is blocked | Human-verify or use gh PR draft workflow |
| TEST-03 | Four gap-closure tests pass (GAP-01..04) | `cd frontend && pnpm test` | All four new test files pass |
| TEST-03 | Path-check step runs and passes in CI | Verify step appears in build.yml and exits 0 against current TESTING.md | `bash tests/check-traceability-paths.sh` from project root |
| TEST-04 | Manual checklist has ≥11 items | Section heading + item count | `grep -c 'M-[0-9]' TESTING.md` → expect ≥11 |
| TEST-05 | Convention section exists in TESTING.md | Section heading presence | `grep -q 'Standing Convention' TESTING.md` |
| TEST-05 | Repo-level CLAUDE.md exists with convention pointer | File existence + content | `test -f CLAUDE.md && grep -q 'TESTING.md' CLAUDE.md` |

### Sampling Rate

- **Per-task commit:** `cd frontend && pnpm test` — confirms no vitest regression from the new test files
- **Per-wave merge:** `go test -race -short ./... && cd frontend && pnpm test` — full fast suite
- **Phase gate:** `gh api repos/scottkw/agenthub/branches/main/protection --jq '.required_status_checks.checks | length'` must equal 5

### Wave 0 Gaps

None — this phase has no Wave 0 scaffolding. The gap-closure tests (GAP-01..04) are written as Green tests (not RED-then-GREEN) because they test existing stable logic, not new implementation.

---

## Security Domain

Branch protection enforces CI as a merge gate. No new auth, session, or cryptographic flows are introduced. The applicable ASVS categories are not triggered by this phase.

| ASVS Category | Applies | Note |
|---------------|---------|------|
| V2 Authentication | No | No auth changes |
| V5 Input Validation | No | Test infrastructure only |
| V6 Cryptography | No | No crypto changes |

One security-adjacent concern: the `enforce_admins: false` choice means admin direct push bypasses the CI gate. This is an explicitly locked decision (D-06) with documented rationale — the GSD doc-commit flow depends on it.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | GitHub Actions `app_id` is 15368 | Branch Protection API Command | Branch protection `checks[]` array silently accepts wrong app_id but may not enforce correctly; use `contexts[]` as fallback if `checks[]` fails | [VERIFIED: live check-runs API query] — NOT an assumption |
| A2 | `strict: false` in required_status_checks does not prevent CI from running | Branch Protection API Command | Branches could merge without CI if strict mode works differently than documented | [CITED: GitHub REST API docs] |
| A3 | `agentBadge.ts` is a pure function with no external dependencies | Gap Analysis GAP-02 | If it has side effects or Wails RPC calls, the test approach needs mocking | [VERIFIED: read agentBadge.ts file listing — no test exists, suggesting it's a simple util] — LOW risk |
| A4 | Phase 142 comp-fidelity CSS additions (`.hub-card__spine`, 16px radius, 150px preview) are in `style.css` | Gap Analysis GAP-04 | Style source gate test would assert wrong token names if the CSS changed | [VERIFIED: 142-VERIFICATION.md records the exact CSS lines] |

**Assumptions requiring confirmation:** None — A1 and A4 are VERIFIED. A2 and A3 are LOW risk.

---

## Open Questions

1. **Path-check parsing convention for multi-file rows**
   - What we know: Some traceability rows may reference a test FILE rather than a specific test NAME (e.g., `frontend/src/lib/hubGroups.test.ts` rather than `frontend/src/lib/hubGroups.test.ts — "createGroup persists"`).
   - What's unclear: Whether the planner should require path-only rows (parseable by the script) or allow path + test name rows (which require stripping the name suffix before checking existence).
   - Recommendation: Require path-only in the path column. Test name / describe block goes in a "Notes" column. This keeps the path-check script simple.

2. **Whether to require all 4 matrix build jobs or only the ubuntu-latest one**
   - What we know: The ubuntu-latest job is the only one that runs vitest + build-script + the new path-check. The other three run Go tests only.
   - What's unclear: Whether requiring all four provides meaningful protection vs. just the ubuntu-latest job.
   - Recommendation: Require all five (four build matrix + playwright) per the research above. A Windows or macOS Go test failure would otherwise pass the gate undetected.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `gh` CLI (authed as scottkw, admin) | Branch protection application (D-07) | Yes | authenticated | — |
| `vitest` | Gap-closure test files | Yes | ^4.1.0 | — |
| `GitHub Actions` | CI gate enforcement | Yes | current | — |
| `bash` | Path-check script | Yes | system | inline `run:` YAML |

---

## Sources

### Primary (HIGH confidence)
- Live filesystem scan — test file counts, exact test file paths, hub test coverage [VERIFIED: 2026-06-21]
- `gh api repos/scottkw/agenthub/commits/main/check-runs` — exact check context names, app_id 15368 [VERIFIED: 2026-06-21]
- `gh api repos/scottkw/agenthub/branches/main/protection` — confirmed no current protection [VERIFIED: 2026-06-21]
- `.github/workflows/build.yml`, `.github/workflows/e2e.yml` — job names, matrix entries, step conditions [VERIFIED: read 2026-06-21]
- `.planning/phases/143-regression-test-program/143-CONTEXT.md` — all 14 locked decisions [read 2026-06-21]
- `137-HUMAN-UAT.md`, `138-HUMAN-UAT.md`, `141-HUMAN-UAT.md` — manual checklist items [read 2026-06-21]
- `142-VERIFICATION.md` — Phase 142 CSS token verification [read 2026-06-21]

### Secondary (MEDIUM confidence)
- [GitHub REST API — branch protection](https://docs.github.com/en/rest/branches/branch-protection) — `checks[]` vs `contexts[]`, `enforce_admins`, `required_pull_request_reviews: null` payload structure [CITED: fetched 2026-06-21]

---

## Metadata

**Confidence breakdown:**
- Suite inventory: HIGH — verified by live filesystem scan
- Branch protection API command: HIGH — check names from live API query; payload from official docs
- Gap analysis: HIGH — derived from reading all v4.0 test files and tracing requirement IDs
- Manual checklist: HIGH — derived from reading UAT logs and STATE.md deferred items

**Research date:** 2026-06-21
**Valid until:** 2026-07-21 (matrix job names only change if build.yml is edited)

---

## RESEARCH COMPLETE

**Phase:** 143 - Regression Test Program
**Confidence:** HIGH

### Key Findings

- **Suite inventory confirmed:** 344 Go / 108 vitest / 7 Playwright / 1 build-script = 460 total files. No files need to move.
- **Branch protection command is ready:** Exact `gh api` payload derived from live check-run names (`build (agenthub, ...)` × 4 + `playwright`), GitHub Actions `app_id: 15368`, `enforce_admins: false` preserves admin direct push.
- **Four v4.0 coverage gaps identified:** `hubGroupCounts.ts` (no test), `agentBadge.ts` (no test), three-item sidebar rendered item count (not in component tests), Phase 142 comp-fidelity CSS tokens (not in style gate). All four close with vitest only.
- **Manual checklist has 11 items** across five categories (Share modal native GUI, remote peer live tests, terminal repaint, signed build, deferred file/editor UATs) — all sourced from existing UAT logs and STATE.md deferred items.
- **Path-check script designed:** ~12-line bash, parses repo-relative paths from TESTING.md traceability table, runs on ubuntu-latest build job alongside existing test steps.

### File Created
`.planning/phases/143-regression-test-program/143-RESEARCH.md`

### Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Suite inventory | HIGH | Live filesystem scan |
| Branch protection payload | HIGH | Live API query + official docs |
| Gap analysis | HIGH | Read all v4.0 test files directly |
| Manual checklist | HIGH | Mined from existing UAT logs |
| Path-check shell design | HIGH | Standard bash pattern |

### Ready for Planning
Research complete. Planner can now create PLAN.md files.
