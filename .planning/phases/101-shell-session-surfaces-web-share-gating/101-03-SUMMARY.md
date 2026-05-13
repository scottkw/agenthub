---
phase: 101-shell-session-surfaces-web-share-gating
plan: 03
subsystem: gui-react
tags: [shell-sessions, web-share-gating, banner, security, tdd-green]
provides:
  - "ShellWebShareBanner role=alert / aria-live=assertive component with locked UI-SPEC copy"
  - "ShellWebShareBanner focuses Cancel on mount, Esc dismisses, Enabling… transient state"
  - "App.tsx handleToggleWeb interception gates shell-session web-toggle ON via SHELL_CLIS set"
  - "App.tsx shellWebShareWarned state hydrated from daemon on mount via GetShellWebShareWarned"
  - "App.tsx handleShellWebShareConfirm fires SetShellWebShareWarned(true) + ToggleWebServing in parallel"
  - "Race mitigation (RESEARCH §8): setShellWebShareWarned(true) runs SYNCHRONOUSLY before Promise.all await"
  - "Banner renders at TOP of .banner-stack (priority slot #1) above all 5 existing banner types"
  - "CSS modifier .webgl-recovery-banner--shell-warning (3px destructive-red left border, no new hex)"
requires:
  - "Plan 101-01 daemon ShellWebShareWarned settings + GetShellWebShareWarned/SetShellWebShareWarned Wails RPCs (already on main, commits dbd95a7 + 8901178)"
  - "Plan 101-02 NewSessionModal shell rows + TabBar agent badge + test-setup.ts (already on main, commit 12f424f)"
affects:
  - "No downstream phase consumers — this plan closes SHELL-07 (frontend) and SHELL-08 outright. CLI/TUI (Plan 04) explicitly skip this banner per SHELL-08 'GUI-only' scope."
tech-stack:
  added: []
  patterns:
    - "Source-inspection App-level tests (?raw import) matching App.exit.test.tsx precedent — no full App mount required"
    - "Synchronous-before-await race mitigation pattern for double-write Wails RPC pairs"
    - "Banner-stack priority via slot-based render ordering (not array sort) — JSX position is the priority signal"
    - "Module-level SHELL_CLIS constant (acceptable v3.3 duplication with TabBar.agentBadgeModifier and daemon isShellSession)"
key-files:
  created:
    - frontend/src/components/ShellWebShareBanner.tsx
    - frontend/src/components/__tests__/ShellWebShareBanner.test.tsx
    - frontend/src/components/__tests__/App.shellWebShare.test.tsx
  modified:
    - frontend/src/App.tsx
    - frontend/src/style.css
decisions:
  - "Banner-stack ordering uses slot-based JSX position (not sort-on-push). The new ShellWebShareBanner is the first child inside the .banner-stack <div>, before LocalNetworkBanner/UpdateBanner/WebGLRecoveryBanner/saveBanner/PluginToggleBanner. This is the simpler of the two options RESEARCH §8 §Open-decision §4 left to the planner, and avoids introducing a priority-sort step on every banner-state mutation."
  - "aria-busy plumbing on per-session Web On buttons (DaemonManagerPanel/StatusBar) is deferred per Task 2 §7 fallback. The banner already visually blocks the UI (top of stack, action-blocking role=alert), and the interception itself prevents ToggleWebServing from being called twice. Re-evaluate in a future polish phase if double-click-during-banner ever surfaces as a real UX bug."
  - "App-level tests use the source-inspection (?raw import) pattern matching App.exit.test.tsx / App.test.tsx / App.dead-modal.test.tsx precedent. Full-mount tests would require mocking ~20 Wails bindings + EventsOn subscriptions that the existing suite intentionally avoids. The component-level tests (ShellWebShareBanner.test.tsx) cover full DOM behavior."
  - "Race mitigation per RESEARCH §8 'First-toggle persistence race': setShellWebShareWarned(true) and setPendingShellWebToggle(null) run SYNCHRONOUSLY before the Promise.all await. A rapid second shell-session toggle sees the updated shellWebShareWarned flag and falls through without re-showing the banner. The assertion order (sync-set BEFORE Promise.all index) is pinned by test #8 in App.shellWebShare.test.tsx."
  - "SHELL_CLIS is a new module-level constant in App.tsx. The same 5-cli set already exists in TabBar.tsx's agentBadgeModifier (collapsed to 'shell' modifier) and in the daemon's isShellSession check (internal/daemon/engine.go). This is the third call-site of the same membership test; per RESEARCH §8 Pitfall 'isShellSession constant drift' this is acceptable v3.3 duplication. If a fourth call-site emerges in v3.4, extract to a shared module."
  - "Plan referred to frontend/src/styles.css (plural); actual filename is frontend/src/style.css (singular) — same finding as Plan 02 SUMMARY decision row. Tracked as deviation 2."
metrics:
  duration_minutes: 8
  completed: 2026-05-12
  tasks_completed: 3
  test_cases_added: 26
  test_cases_passing: 26
  full_suite: 848/848 GREEN
---

# Phase 101 Plan 03: ShellWebShareBanner + App.tsx Toggle Interception Summary

Add the one-time security-confirmation banner (SHELL-08) and the App.tsx toggle interception that enforces it (SHELL-07 frontend defense-in-depth). When the user toggles "Web On" for a shell session for the first time on this machine, the toggle is intercepted, a locked-copy `role="alert"` banner appears at the top of the banner stack, and only on explicit Confirm does the daemon receive both `SetShellWebShareWarned(true)` and `ToggleWebServing(true)`. Cancel/Esc/× dismiss without side effects.

## What Was Built

### ShellWebShareBanner (frontend/src/components/ShellWebShareBanner.tsx)
New React component with the following contract:
- **role="alert" + aria-live="assertive"** (NOT polite — differs from PluginToggleBanner/WebGLRecoveryBanner because this banner blocks an action).
- **Verbatim locked copy** from UI-SPEC §Web-share security banner copy:
  - Heading: `Web sharing this shell will expose arbitrary command execution.`
  - Body line 1 (with sessionName interpolated): `You are about to share '<sessionName>'. Anyone on your tailnet who can reach the daemon will be able to type commands as your user account.`
  - Body line 2: `Read-only viewers cannot type, but commands you run remain visible to them.`
  - Primary CTA: `Enable web sharing`
  - Transient label: `Enabling…` (U+2026 single-character ellipsis)
  - Secondary CTA: `Cancel`
  - Dismiss `aria-label`: `Dismiss security warning`
- **Props:** `{ sessionName: string; onConfirm: () => void; onCancel: () => void }`.
- **Focus on mount:** the Cancel button receives focus via a `useRef` + `useEffect` pair, mirroring `QuitConfirmModal` (Phase 85, `killFocusYes=false`).
- **Esc key:** a `document.addEventListener('keydown', ...)` handler treats Escape as Cancel.
- **Enabling transient state:** internal `useState(false)` boolean. Clicking "Enable web sharing" flips it to true, which:
  - Re-renders the primary button label as `Enabling…`.
  - Sets `disabled` on Cancel + primary + × dismiss buttons.
  - Sets `aria-busy="true"` on the root element.
  The parent (`App.tsx`) unmounts the banner after `handleShellWebShareConfirm` resolves; this transient state is purely for the click-to-unmount window.
- **CSS:** reuses `.webgl-recovery-banner` BEM base class with new `--shell-warning` modifier (3px left border `#f7768e` — existing destructive-red, no new hex). Internal layout uses `__shell-body / __shell-heading / __shell-text / __shell-actions / __shell-btn` namespaced helpers to avoid colliding with the existing `__message / __dismiss` classes from the info-style variant.

### App.tsx interception (frontend/src/App.tsx)
- **Module-level constant** `SHELL_CLIS = new Set(['shell', 'bash', 'zsh', 'pwsh', 'powershell'])` — the 5 shell cli identifiers that route through the banner gate.
- **Two new state hooks:**
  - `shellWebShareWarned: boolean` (init `false`) — hydrated from `GetShellWebShareWarned()` in the existing mount `useEffect` (placed next to `GetAutoCloseSession`). On failure, defaults to `false` so the banner shows on next toggle attempt (safe-degrade).
  - `pendingShellWebToggle: { sessionId, sessionName } | null` (init `null`) — carries the in-flight toggle metadata while the banner is shown.
- **Modified `handleToggleWeb`:** before the existing `ToggleWebServing` call, when `nowEnabled === true`, look up the originating tab via `tabs.find(t => t.sessionId === sessionId)`. If the tab's `cli` is in `SHELL_CLIS` and `!shellWebShareWarned`, set `pendingShellWebToggle` and return early (skip the daemon RPC entirely). Otherwise fall through to the existing parallel optimistic-update path.
- **New `handleShellWebShareConfirm`:**
  - Reads `pendingShellWebToggle`.
  - **Synchronously** (before any await) calls `setShellWebShareWarned(true)` and `setPendingShellWebToggle(null)` — this is the RESEARCH §8 race mitigation.
  - Then `await Promise.all([SetShellWebShareWarned(true), ToggleWebServing(sessionId, true)])`.
  - On success: `setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))`.
  - On failure: rolls back `setShellWebShareWarned(false)` so the banner re-shows on the next toggle attempt.
- **New `handleShellWebShareCancel`:** simply clears `pendingShellWebToggle`. No Wails RPC fires. No daemon state mutates.
- **Banner-stack render:** `<ShellWebShareBanner>` is the **first child** inside `<div className="banner-stack">`, gated on `pendingShellWebToggle`. The outer `&&`-block wrapping the banner-stack was extended to include `pendingShellWebToggle ||` as the first disjunct so the stack mounts even when no other banners are active.

### CSS (frontend/src/style.css)
Appended a new BEM block under the existing `.webgl-recovery-banner` rules:
- `.webgl-recovery-banner--shell-warning` — overrides left-border accent to `3px solid #f7768e`, keeps everything else.
- `.webgl-recovery-banner__shell-body / __shell-heading / __shell-text` — 2-line layout helpers (flex-column, 13px+12px, TokyoNight `#c0caf5` heading + `#a9b1d6` body).
- `.webgl-recovery-banner__shell-actions` — inline-flex action row.
- `.webgl-recovery-banner__shell-btn` + `--secondary` + `--primary-destructive` — button modifiers (secondary = ghost with TokyoNight `#292e42` border; primary-destructive = `#f7768e` fill / `#1a1b26` text / weight 600). Disabled state opacity 0.6 + `cursor: not-allowed`. Focus-visible outline `2px solid #7aa2f7` (accent blue, matches the rest of the project).

No new colors introduced; all hex values are existing TokyoNight palette members.

## Locked-Copy / Locked-Color Verification

| Token | Location | Count | Verbatim |
|-------|----------|-------|----------|
| `Web sharing this shell will expose arbitrary command execution.` | `ShellWebShareBanner.tsx` | 2 (component + doc-comment) | yes |
| `You are about to share '` | `ShellWebShareBanner.tsx` | 2 | yes |
| `Read-only viewers cannot type, but commands you run remain visible to them.` | `ShellWebShareBanner.tsx` | 2 | yes |
| `Enable web sharing` | `ShellWebShareBanner.tsx` | 2 | yes |
| `Enabling…` (U+2026) | `ShellWebShareBanner.tsx` | 2 | yes |
| `Cancel` | `ShellWebShareBanner.tsx` | 1 | yes (only button label, not a doc-comment string) |
| `Dismiss security warning` | `ShellWebShareBanner.tsx` | 2 | yes |
| `webgl-recovery-banner--shell-warning` modifier | `style.css` + component | 1 + 1 | yes |
| `role="alert"` | `ShellWebShareBanner.tsx` | 1 | yes |
| `aria-live="assertive"` | `ShellWebShareBanner.tsx` | 1 | yes |
| `#f7768e` (destructive-red) | `style.css` (reused from existing palette) | unchanged | yes |
| `SHELL_CLIS` constant | `App.tsx` | 2 (decl + use) | yes |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — Tooling] Frontend `node_modules` missing on worktree start**
- **Found during:** first `pnpm test -- --run ShellWebShareBanner` invocation
- **Issue:** the worktree had no installed Node packages; vitest CLI was not found.
- **Fix:** ran `pnpm install` (which reused the lockfile, no version drift) before continuing.
- **Files modified:** none in repo (only `node_modules/`).

**2. [Rule 1 — Filename mismatch] Plan referred to `frontend/src/styles.css`; actual file is `frontend/src/style.css`**
- **Found during:** Task 1 CSS edit
- **Fix:** edited the actual file `frontend/src/style.css`. Same finding as Plan 02's SUMMARY decision row — appears to be a recurring plan-doc typo.

**3. [Rule 1 — Test regex] First-mention of `GetShellWebShareWarned` lands in a doc-comment, not at the call-site**
- **Found during:** Task 2 GREEN test run (1 of 13 tests red after implementation)
- **Issue:** the test scanned the first 200 chars after the first `GetShellWebShareWarned()` occurrence for `setShellWebShareWarned`. My JSDoc comment in App.tsx mentions `GetShellWebShareWarned()` ~10 lines before the actual call-site, so the 200-char window captured comment text instead of the `.then()` chain.
- **Fix:** tightened the test to use a regex pattern that specifically matches `GetShellWebShareWarned()` followed within 250 chars by `.then(...setShellWebShareWarned`. This is a more precise contract assertion (the doc-comment doesn't satisfy the regex; only the real call-site does).
- **Commit:** folded into Task 2 GREEN commit (`ab568fe`).

### Scope Boundary Notes (deferred per plan permission)

- **aria-busy on per-session Web On buttons** (DaemonManagerPanel + StatusBar): NOT implemented per Task 2 §7 fallback. The banner already visually blocks the UI (top of stack, action-blocking role=alert), and the interception itself prevents ToggleWebServing from being called twice via `handleToggleWeb`'s early-return. If double-click-during-banner ever surfaces as a real UX bug, add `webToggleBusyFor: string | null` prop in a future polish phase.

### Genuine deviations from plan acceptance criteria

- **Test count:** plan acceptance §Task 2 says "12 named test cases" — actual file has **13 cases** (added one extra: explicit `clicking Cancel fires onCancel and not onConfirm` separated from `clicking dismiss (×) fires onCancel` for finer-grained coverage). Total new test cases: **26** (13 banner + 13 App), vs plan's expected **25** (13 + 12). All 26 pass; no negative impact.

## Test Surface

26 new Vitest cases added (13 banner + 13 App-level). All pass. Full repo suite is **848/848 GREEN** (prior baseline: 822/822 after Plan 02; +26 = 848 — exact match).

### ShellWebShareBanner.test.tsx (13 cases, fully-rendered DOM)
1. renders heading verbatim
2. renders body with sessionName interpolated and both body sentences
3. primary CTA reads 'Enable web sharing' verbatim
4. secondary CTA reads 'Cancel' verbatim
5. dismiss button aria-label is 'Dismiss security warning'
6. root has role="alert" and aria-live="assertive" (NOT polite — differs from PluginToggleBanner)
7. focuses Cancel button on mount (safe action per QuitConfirmModal precedent)
8. Esc keydown fires onCancel and not onConfirm
9. clicking Cancel fires onCancel and not onConfirm
10. clicking dismiss (×) fires onCancel
11. clicking 'Enable web sharing' fires onConfirm
12. after confirm click, primary shows 'Enabling…' and aria-busy=true; both buttons disabled
13. BEM class includes both webgl-recovery-banner and webgl-recovery-banner--shell-warning

### App.shellWebShare.test.tsx (13 cases, source-inspection ?raw)
1. imports ShellWebShareBanner from components/ShellWebShareBanner
2. imports GetShellWebShareWarned and SetShellWebShareWarned Wails bindings
3. declares SHELL_CLIS set with shell/bash/zsh/pwsh/powershell membership
4. declares shellWebShareWarned React state
5. declares pendingShellWebToggle React state for pending banner data
6. calls GetShellWebShareWarned in a mount useEffect to seed shellWebShareWarned
7. intercepts handleToggleWeb: short-circuits when shell session + enabling + !shellWebShareWarned
8. on confirm: calls SetShellWebShareWarned(true) and ToggleWebServing in parallel
9. on confirm: sets shellWebShareWarned synchronously BEFORE await (race mitigation per RESEARCH §8)
10. on cancel: clears pendingShellWebToggle without invoking any Wails RPC
11. renders ShellWebShareBanner at the TOP of the banner-stack (priority slot #1)
12. ShellWebShareBanner receives sessionName, onConfirm, onCancel props
13. handleToggleWeb continues to work for AI CLIs (no banner short-circuit when cli !∈ SHELL_CLIS)

## Threat-Register Coverage

| Threat ID | Disposition | How satisfied |
|-----------|-------------|---------------|
| T-101-03-01 (Tampering — locked copy drift) | mitigate | Vitest cases #1 / #3 / #4 / #5 in ShellWebShareBanner.test.tsx pin all locked strings verbatim. Any rewording fails CI. |
| T-101-03-02 (EoP — banner bypass) | mitigate | App.tsx interception is the only `ToggleWebServing(id, true)` path for shell sessions when `!shellWebShareWarned`. Test #7 + #13 lock the gate structure. |
| T-101-03-03 (Info disclosure — XSS via sessionName) | mitigate | sessionName rendered as a React JSX expression (`{sessionName}`), auto-escaped by React. No `dangerouslySetInnerHTML` anywhere in the component. |
| T-101-03-04 (DoS — stuck in Enabling…) | accept | `setPendingShellWebToggle(null)` runs synchronously; the banner unmounts regardless of RPC outcome. On RPC failure `setShellWebShareWarned(false)` rolls back so the user retains the gate. |
| T-101-03-05 (Race — rapid second shell toggle) | mitigate | RESEARCH §8 pattern: `setShellWebShareWarned(true)` runs SYNCHRONOUSLY before the Promise.all await. Test #9 pins the sync-set BEFORE Promise.all ordering. |

## Closes / Opens

**Closes:**
- **SHELL-07** (frontend defense-in-depth) — `handleToggleWeb` intercepts shell ON-toggles when `!shellWebShareWarned`; AI CLIs, OFF-toggles, and confirmed shell toggles fall through unchanged. The daemon-side SHELL-07 (no auto-enable on session create) was already satisfied by Phase 87 SEC-01 (`api.go:407`).
- **SHELL-08** (one-time confirmation banner) — `ShellWebShareBanner` renders at TOP of banner-stack with verbatim locked copy, role=alert, focus-on-mount, Esc dismissal, Enabling transient state. Confirm flow fires both Wails RPCs in parallel after synchronous race-mitigation state update; Cancel flow is side-effect-free.

**Opens:** None. Plan 101-04 (CLI + TUI shells) is independent — it lives in `cmd_cli.go` and `internal/tui/*` only.

## Commits

| Hash | Type | Summary |
|------|------|---------|
| `a999cfd` | test | Add failing tests for ShellWebShareBanner (RED) — 13 cases |
| `5f7aa97` | feat | Implement ShellWebShareBanner component (GREEN) — incl. CSS modifier |
| `383b7fc` | test | Add failing tests for App.tsx shell web-toggle interception (RED) — 13 cases |
| `ab568fe` | feat | App.tsx shell web-toggle interception (GREEN) — closes SHELL-07/SHELL-08 |

## Threat Flags

None — all new security-relevant surface (the role=alert banner + the toggle gate) is explicitly catalogued in the plan's `<threat_model>` block (T-101-03-01 through T-101-03-05) and covered by mitigations above. No new threat surface introduced beyond what the plan anticipated.

## Self-Check: PASSED

- Files created/modified exist on disk:
  - `frontend/src/components/ShellWebShareBanner.tsx` — FOUND
  - `frontend/src/components/__tests__/ShellWebShareBanner.test.tsx` — FOUND
  - `frontend/src/components/__tests__/App.shellWebShare.test.tsx` — FOUND
  - `frontend/src/App.tsx` — MODIFIED
  - `frontend/src/style.css` — MODIFIED
- Commit hashes present in `git log`: `a999cfd`, `5f7aa97`, `383b7fc`, `ab568fe` — all FOUND.
- Locked copy strings appear verbatim in `ShellWebShareBanner.tsx`: yes (grep counts above all ≥1).
- `SHELL_CLIS` constant + `pendingShellWebToggle` state in `App.tsx`: yes.
- TypeScript typecheck clean: `pnpm tsc --noEmit` exits 0.
- Full Vitest suite passes: **848/848** green.
- Go build clean: `go build ./...` exits 0.
- No files outside the declared scope touched: confirmed via `git diff --name-only main..HEAD` returning exactly the 5 expected files.
