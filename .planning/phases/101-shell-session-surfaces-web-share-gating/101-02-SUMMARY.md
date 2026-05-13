---
phase: 101-shell-session-surfaces-web-share-gating
plan: 02
subsystem: gui-react
tags: [shell-sessions, new-session-modal, tab-bar, css-bem, tdd-green]
provides:
  - "NewSessionModal renders shell rows (one per DetectedShell) with locked 'Shell — DISPLAYNAME' copy"
  - "NewSessionModal has shells + shellsLoading optional props (default [] / false)"
  - "Args namespace 'agenthub:args:shell:NAME' isolated from AI CLI 'agenthub:args:CLI'"
  - "App.tsx threads ListShells() result + loading state into NewSessionModal"
  - "TabBar renders .tab__agent-badge between status dot and tab name (8px decorative)"
  - "TabBar tooltip suffix carries agent type ('Shell — bash', 'Claude Code', etc.)"
  - "CSS palette for 7 agent badges (6 AI CLIs + shell) + selected-shell cyan border"
  - "ListShells Wails binding + daemon.DetectedShell TS model (101-01 prerequisite delivered as deviation)"
requires:
  - "Phase 100 daemon /shells HTTP route + DetectedShell wire type (already on main)"
  - "daemon.DaemonClient.ListShells() over Unix socket (already on main, internal/daemon/client.go:109)"
affects:
  - "Plan 101-03 (shell web-share banner): consumes selectedAgent prefix scheme + tab agent badge for visual continuity"
  - "Plan 101-04 (CLI + TUI shells): independent wave; no shared code path with 101-02"
tech-stack:
  added: []
  patterns:
    - "BEM modifier-per-agent CSS palette (no inline styles)"
    - "selectedAgent prefix scheme ('shell:NAME' for shells, plain name for AI CLIs)"
    - "Silent absence on empty discovery (no empty-state placeholder)"
    - "Loading skeleton shown only when shellsLoading && shells.length === 0"
key-files:
  created:
    - frontend/src/test-setup.ts
  modified:
    - app.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/models.ts
    - frontend/src/components/NewSessionModal.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/__tests__/NewSessionModal.test.tsx
    - frontend/src/components/__tests__/TabBar.test.tsx
    - frontend/src/App.tsx
    - frontend/src/style.css
    - frontend/vite.config.ts
decisions:
  - "Renamed selectedCLI → selectedAgent in NewSessionModal state; updated 2 source-inspection tests that pinned the old name. Rationale: selectedAgent must hold either an AI-CLI name (e.g., 'claude') or a 'shell:NAME' prefix; the old name lied about its content type."
  - "Added 101-01 prerequisite work (ListShells Wails binding + DetectedShell TS model) as a Rule 3 deviation. Plan 101-01 PLAN.md exists but no implementation was merged to main despite the orchestrator's claim, blocking 101-02. Only the bindings strictly needed for 101-02 were added; the daemon ShellWebShareWarned settings half of 101-01 is intentionally NOT included here (it's required only by 101-03)."
  - "Added frontend/src/test-setup.ts to polyfill globalThis.localStorage. jsdom 29 + Vitest 4 expose window.localStorage but not the bare global, and NewSessionModal touches localStorage in useState initializers. This is the first plan in the repo with render-based tests against a component using localStorage; existing tests were source-inspection (?raw) only."
  - "Plan referred to frontend/src/styles.css; actual filename is frontend/src/style.css (singular). Tracked in deviation log."
metrics:
  duration_minutes: 7
  completed: 2026-05-13
  tasks_completed: 3
  test_cases_added: 20
  test_cases_passing: 20
  full_suite: 822/822 GREEN
---

# Phase 101 Plan 02: NewSessionModal Shell Rows + TabBar Agent Badge Summary

Surface discovered shells in the GUI: render one row per `DetectedShell` in the new-session modal (with locked "Shell — DISPLAYNAME" copy and a mono detail line for the resolved path), and tag every tab with an 8px agent-color badge dot between the status dot and the tab name. Wire `App.tsx` to call `ListShells()` on mount and thread the result through. Closes the GUI half of **SHELL-01** (new-shell-session creation) and **SHELL-06** (visual badge for shell sessions); the CLI/TUI halves of SHELL-01/06 land in plan 04, and the web-share confirmation banner lands in plan 03.

## What Was Built

### NewSessionModal (frontend/src/components/NewSessionModal.tsx)
- **New optional props:** `shells?: daemon.DetectedShell[]` (default `[]`), `shellsLoading?: boolean` (default `false`).
- **Selection prefix scheme:** state field renamed `selectedCLI → selectedAgent`. AI CLIs stored as plain name (`"claude"`); shells stored as `"shell:NAME"` (`"shell:bash"`). The `"shell:"` prefix is stripped before invoking `onConfirm`, so the daemon still receives bare `"bash"`/`"zsh"`/`"shell"`.
- **Shell rows render AFTER the AI CLI list** (no divider per UI-SPEC D-04). Each row has the locked primary label `"Shell — DISPLAYNAME"` (em-dash U+2014, NOT a colon) and a mono `.new-session-modal__agent-btn__detail` span showing the resolved path. Client-side sort puts `name === "shell"` (system default) first.
- **Selected shell row** gains `.new-session-modal__agent-btn--selected-shell` (border `#89ddff` cyan). AI CLI rows keep `.new-session-modal__agent-btn--selected` (border `#7aa2f7` blue). The two modifiers are mutually exclusive.
- **Args field** is `disabled` when a shell is selected; placeholder swaps to the locked copy `"Arguments are not passed to shell sessions"`. The Clear-Args button is hidden for shells (nothing to clear).
- **Args memory namespace:** `agenthub:args:CLI` for AI CLIs (unchanged); `agenthub:args:shell:NAME` for shells. The namespaces never cross-contaminate — selecting `shell:bash` does NOT read `agenthub:args:bash`, and vice versa.
- **Loading skeleton:** rendered ONLY when `shellsLoading === true` AND `shells.length === 0` (locked text `"Loading shells…"` U+2026). When discovery returns 0 shells, the modal renders nothing extra — silent absence per UI-SPEC §Edge Cases.

### TabBar (frontend/src/components/TabBar.tsx)
- **`agentBadgeModifier(cli)`** module-level helper returns the BEM modifier suffix (without `--`) or `null` for unknown CLIs. 6 AI CLIs return their own name; 5 shell variants (`shell`/`bash`/`zsh`/`pwsh`/`powershell`) collapse to `"shell"` (the badge communicates "this is a shell session", not which shell).
- **`agentDisplayName(cli)`** helper returns the tooltip-friendly label. Shells get `"Shell — DISPLAYNAME"` (em-dash); AI CLIs get product names from the modal palette; unknown CLIs return the raw string.
- **New element:** `<span class="tab__agent-badge tab__agent-badge--MODIFIER" aria-hidden="true" />` inserted between `.tab__status` and the tab-name. DOM order: `status → badge → name → countdown → close → progress`.
- **Tab tooltip extended:** the `title` attribute on `.tab__name` becomes `"Double-click or right-click to rename · Shell — bash"` for shell sessions, `"… · Claude Code"` for AI CLI sessions, and stays as the bare rename hint for synthetic tabs (welcome/daemon/settings, where `tab.cli === ""`). The right-click pre-existing test still passes because the suffix is appended, not replaced.

### App.tsx (frontend/src/App.tsx)
- Imported `ListShells` from `./wailsjs/go/main/App`.
- New state: `detectedShells: daemon.DetectedShell[]` (init `[]`) and `shellsLoading: boolean` (init `true`).
- `ListShells()` called in the main `init()` path AND the `retryInit` path so a recovered daemon repopulates the modal. Failures fall through to `[]` with `shellsLoading=false` — silent absence per UI-SPEC.
- `<NewSessionModal>` receives both new props at its instantiation site.

### CSS (frontend/src/style.css)
Added BEM classes (no inline styles, no new colors beyond UI-SPEC's locked `#89ddff` shell cyan):
- `.new-session-modal__agent-btn--shell` — 56px 2-line layout, flex-column, gap 4px.
- `.new-session-modal__agent-btn--selected-shell` — border `#89ddff`, background `#1e2030`.
- `.new-session-modal__agent-btn__detail` — mono 11px `#565f89` for the resolved path.
- `.new-session-modal__agent-btn--loading` — muted 11px skeleton row.
- `.tab__agent-badge` — 8px circle, muted `#9aa5ce` default fallback.
- `.tab__agent-badge--claude / --opencode / --codex / --gemini / --cursor / --aider / --shell` — the 7-color palette matching the TUI dark-mode `BadgeXxx` set.

### Wails Bindings (Rule 3 deviation — see Deviations)
- `app.go ListShells() []daemon.DetectedShell` — thin delegation to `a.client.ListShells()` with graceful-degrade-to-empty.
- `frontend/src/wailsjs/go/main/App.d.ts + .js` — TypeScript binding stubs (hand-edited per Phase 99 PUI-04 precedent).
- `frontend/src/wailsjs/go/models.ts` — `daemon.DetectedShell` class with `name / displayName / path / argv` fields mirroring `internal/daemon/types.go`.

### Test Infrastructure
- `frontend/src/test-setup.ts` (NEW) — polyfills `globalThis.localStorage` for Vitest 4 + jsdom 29 (which expose `window.localStorage` but not the bare global).
- `frontend/vite.config.ts` — registered the new setup file via `test.setupFiles`.

## Locked-Copy / Locked-Color Verification

All locked tokens from UI-SPEC §Locked Visual Constants appear verbatim:

| Token | Location | Verbatim |
|-------|----------|----------|
| `Shell — ` prefix (em-dash U+2014) | `NewSessionModal.tsx:158` | yes — JSX text node `Shell — {s.displayName}` |
| `agenthub:args:shell:` namespace | `NewSessionModal.tsx:7` | yes — `const SHELL_ARGS_KEY` |
| `Arguments are not passed to shell sessions` placeholder | `NewSessionModal.tsx:9` | yes — `const SHELL_ARGS_PLACEHOLDER` |
| `Loading shells…` (U+2026) | `NewSessionModal.tsx:172` | yes — literal JSX text |
| `#89ddff` (shell cyan, dark) | `style.css` ×4 (selected-shell + tab badge + 2 other surfaces) | yes |
| `#7aa2f7` (AI-CLI accent blue) | `style.css` ×49 (unchanged) | yes — no regression |
| `tab__agent-badge` BEM class | `TabBar.tsx:181, 184` | yes |
| `aria-hidden="true"` on badge | `TabBar.tsx:209` | yes |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — Blocker] Plan 101-01 prerequisites not on main**
- **Found during:** plan startup (worktree-branch verification + Wails binding grep)
- **Issue:** Plan 101-02's interfaces block declares "Plan 101-01 Wails bindings (in place after Wave 1)" with `ListShells()` and `daemon.DetectedShell`. Inspection showed:
  - `app.go` had no `ListShells` method.
  - `frontend/src/wailsjs/go/main/App.{d.ts,js}` had no `ListShells` export.
  - `frontend/src/wailsjs/go/models.ts` had no `DetectedShell` class.
  - `git log --all --grep="101-01"` returned only the planning-doc commits, no implementation commits in any branch.
  - The orchestrator's prompt asserted "Plan 101-01 already merged to main with daemon ShellWebShareWarned settings + Wails ListShells binding" — this proved inaccurate.
- **Fix:** Added the minimum 101-01 prerequisites strictly needed by 101-02:
  - `app.go ListShells() []daemon.DetectedShell` delegating to the existing `internal/daemon/client.go ListShells()` (already on main from Phase 100).
  - `App.d.ts / App.js` typed binding stubs.
  - `models.ts daemon.DetectedShell` class.
- **NOT included:** the `ShellWebShareWarned` daemon settings + Wails RPC half of 101-01. Plan 101-03 depends on those, not 101-02, so adding them now would broaden scope without unblocking the current plan.
- **Commit:** `172c6c2`
- **Files modified:** `app.go`, `frontend/src/wailsjs/go/main/App.d.ts`, `frontend/src/wailsjs/go/main/App.js`, `frontend/src/wailsjs/go/models.ts`

**2. [Rule 3 — Blocker] Test infrastructure missing `localStorage` global**
- **Found during:** Task 1 RED test run
- **Issue:** `NewSessionModal` touches `localStorage` in its `useState` initializers. Vitest 4 + jsdom 29 provide `window.localStorage` but do NOT expose the bare `localStorage` identifier on the global scope. All render-based tests for NewSessionModal crashed with `TypeError: Cannot read properties of undefined (reading 'getItem')` before any assertion ran. Existing tests were all `?raw` source-inspection, so this had never surfaced.
- **Fix:** Created `frontend/src/test-setup.ts` to polyfill `globalThis.localStorage` (using the jsdom-provided `window.localStorage` when available, falling back to a Map-backed Storage shim). Registered via `vite.config.ts test.setupFiles`.
- **Commit:** `c1bed18` (folded into Task 1 RED commit because the infrastructure fix is what turned the localStorage crash into a clean RED assertion failure).

**3. [Rule 1 — Filename mismatch] Plan referred to `frontend/src/styles.css` (plural); actual file is `frontend/src/style.css` (singular)**
- **Found during:** Task 2 CSS edit
- **Fix:** Used the actual filename `frontend/src/style.css`. All other references in the plan body and verification block also updated to match.

### Scope Boundary Notes (NOT auto-fixed — deferred)

- **Plan 101-03 / 101-04 dependencies:** the `ShellWebShareWarned` daemon settings field, `GetShellWebShareWarned`/`SetShellWebShareWarned` Wails RPCs, and HTTP route plumbing are intentionally NOT added here. They are scoped to plan 101-03 (banner). Adding them now would have broadened the diff beyond the SHELL-01 + SHELL-06 GUI half this plan owns.
- **Locked-copy tooltip text:** plan acceptance criterion 19 says the tooltip "includes 'Shell — bash' (or whatever the agent display form is — match UI-SPEC)". The chosen format is `"Double-click or right-click to rename · Shell — bash"` (interpunct separator). The right-click hint is preserved because the existing tooltip test asserts `title.includes("right-click")`.

## Renamed State (Behavior Contract)

- `selectedCLI` (old) → `selectedAgent` (new). The field now holds either an AI-CLI name (`"claude"`) or a shell prefix (`"shell:bash"`). The 2 source-inspection tests that pinned the old name were updated:
  - `frontend/src/components/__tests__/NewSessionModal.test.tsx:33` — `tracks selectedCLI state` → `tracks selected agent state` (asserts `selectedAgent`).
  - `frontend/src/components/__tests__/NewSessionModal.test.tsx:89` — `expect(raw).toContain('ARGS_KEY(selectedCLI)')` → `… 'ARGS_KEY(selectedAgent)'`.

## Test Surface

20 new Vitest cases added (10 modal + 10 tab bar). All 20 pass. Full repo suite is **822/822 GREEN**.

NewSessionModal cases:
1. renders one row per shell with Shell em-dash prefix
2. system default shell row appears first
3. shell row shows resolved path as mono secondary line
4. selecting shell row applies shell cyan border modifier
5. selecting AI CLI row applies accent blue border modifier not shell cyan
6. loading skeleton renders when shellsLoading is true and shells is empty
7. no shell rows when shells prop is empty and not loading
8. args field disabled when shell selected
9. args field enabled when AI CLI selected
10. args namespace key uses shell prefix when shell selected

TabBar cases:
11. renders agent badge for claude session
12. renders shell agent badge for cli=shell
13. renders shell agent badge for cli=bash
14. renders shell agent badge for cli=zsh
15. renders shell agent badge for cli=pwsh
16. renders shell agent badge for cli=powershell
17. falls back to muted badge for unknown cli
18. agent badge is aria-hidden
19. tab tooltip includes agent type for shell session
20. tab agent badge appears between status dot and tab name

## Closes / Opens

**Closes:**
- SHELL-01 (GUI half) — user can select a shell from the new-session modal.
- SHELL-06 (GUI half) — every tab carries a distinct agent-color badge; shell sessions render the locked `#89ddff` cyan badge.

**Opens (for downstream plans):**
- Plan 101-03 (`ShellWebShareBanner` + App.tsx toggle interception) — can now consume the `selectedAgent.startsWith("shell:")` discriminator and the `.tab__agent-badge--shell` selector for its banner-stack visual continuity.
- Plan 101-04 (CLI `agenthub new shell` + TUI agent picker + TUI `BadgeShell`) — independent wave; no shared code path with 101-02.

## Commits

| Hash | Type | Summary |
|------|------|---------|
| `172c6c2` | feat | Add `ListShells` Wails binding + `DetectedShell` model (101-01 prerequisite) |
| `c1bed18` | test | Add failing RED tests for shell rows + agent badge |
| `c2363da` | feat | `NewSessionModal` shell rows + `App.tsx` `ListShells` wiring (GREEN) |
| `aa591fa` | feat | `TabBar` agent badge + tooltip + CSS palette (GREEN) |

## Threat Flags

None — the threat model in 101-02-PLAN.md anticipated `DetectedShell.path` rendered as a text node (React-escaped, T-101-02-01 mitigated) and the unknown-CLI badge color fallback (T-101-02-02 accepted). No new threat surface introduced beyond what the plan's threat register catalogued.

## Self-Check: PASSED

- Files created/modified exist on disk: yes (12 files).
- Commit hashes present in `git log`: yes (172c6c2, c1bed18, c2363da, aa591fa).
- Locked copy strings appear verbatim in the production component files: yes (grep counts above).
- TypeScript typecheck clean: `tsc --noEmit` exits 0.
- Full Vitest suite passes: 822/822.
- Go build clean: `go build ./...` exits 0.
