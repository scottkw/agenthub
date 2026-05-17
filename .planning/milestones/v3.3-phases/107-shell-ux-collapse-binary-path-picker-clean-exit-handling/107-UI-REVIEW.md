---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
type: ui-review
audited: 2026-05-13
baseline: 107-UI-SPEC.md (locked)
screenshots: not captured (no dev server running — code-only audit)
advisory: true
---

# Phase 107 — UI Review

**Audited:** 2026-05-13
**Baseline:** 107-UI-SPEC.md (locked design contract)
**Screenshots:** Not captured — no dev server detected on ports 3000, 5173, or 8080. Audit is code-only.
**Scope:** Three targeted deltas — NewSessionModal single Shell row (SHELL-10), SettingsTab Shell binary path field (SHELL-11), App.tsx session:exit clean-exit suppression (SHELL-12). Minimal reuse phase; no new tokens or components introduced per spec.

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Visual Hierarchy & Spacing | 4/4 | New Shell row and Settings row use identical class names and markup as their peer elements; no spacing deviations |
| 2. Typography & Copywriting | 4/4 | All label strings match UI-SPEC verbatim; no em-dash on Shell label (correct per spec); no generic or off-spec strings |
| 3. Color & Contrast | 4/4 | Zero hardcoded colors introduced in phase 107 code; existing tokens `#89ddff` / `#f7768e` used via CSS classes only |
| 4. Accessibility | 3/4 | ARIA chain correct for error-present state; IN-03 (dangling aria-describedby when no error) is a known deferred WCAG advisory nit; Browse button in NewSessionModal lacks accessible name |
| 5. Interaction & Feedback | 4/4 | CR-01 (autoCloseRef guard) and WR-02 (false Saved!) fixes confirmed in source; save/browse/create flows all have correct in-progress states |
| 6. Responsive & Robustness | 3/4 | Empty-path and error-inline cases correctly handled; ExitCountdownBanner / exitCountdowns paths now permanently unreachable dead UI (IN-02, deferred) |

**Overall: 22/24**

---

## Top 3 Priority Fixes

1. **Browse button in NewSessionModal has no accessible name** (Pillar 4, WARNING) — The working-directory Browse button at `NewSessionModal.tsx:167-172` carries no `aria-label` or `title`. When the label reads "Browse…" it is technically self-describing, but the button is identified only by its text content. Screen readers will announce "Browse, button" with no context about what is being browsed. Fix: add `title="Browse for working directory"` to match the `settings-panel__browse-btn` pattern used throughout SettingsTab.

2. **IN-03: aria-describedby dangles when no error is showing** (Pillar 4, WARNING, deferred to v3.4) — `SettingsTab.tsx:723` unconditionally sets `aria-describedby="settings-shell-path-desc"` on the shell-path input; the element with that id only exists in the DOM when `shellPathError` is truthy (`SettingsTab.tsx:733`). The attribute points nowhere the rest of the time. Fix already specified in 107-REVIEW.md IN-03: apply `aria-describedby={shellPathError ? 'settings-shell-path-desc' : undefined}`. Confirmed deferred; flagged here for v3.4 tracking.

3. **IN-02: ExitCountdownBanner and exitCountdowns TabBar prop are dead UI** (Pillar 6, WARNING, deferred to v3.4) — `App.tsx:1064` and `App.tsx:1175` filter `sessionExits` for entries with `exitCode === 0 && countdown > 0`. After the SHELL-12 redesign, no such entry can ever exist: exit-code-0 sessions either close immediately (autoClose ON) or fall through to `setSessionExits` with `countdown: -1` (autoClose OFF). Both render paths are permanently unreachable. The code compiles and causes no behavioral harm, but it is misleading and a maintenance liability. Fix: remove `ExitCountdownBanner`, the `exitCountdowns` TabBar prop, `countdownTimers` ref, and `handleKeepOpen` — or document explicitly that countdown UI is retired.

---

## Detailed Findings

### Pillar 1: Visual Hierarchy & Spacing (4/4)

**Shell row in NewSessionModal (SHELL-10):**
The implementation at `NewSessionModal.tsx:146-157` applies exactly the three classes specified in UI-SPEC §2:
- `new-session-modal__agent-btn` (base button)
- `new-session-modal__agent-btn--shell` (2-line tall variant, `min-height: 56px`, flex column layout)
- `new-session-modal__agent-btn--selected-shell` (conditional, `border-color: #89ddff`)

The `--shell` class grows the row to `min-height: 56px` via `style.css:827-836`, distinguishing it from the 36px AI CLI rows. This matches the spec requirement that the shell row "reuse the existing shell-row markup verbatim." The hover state at `style.css:811-814` applies uniformly to all `agent-btn` elements including the shell row — no deviation.

The detail span at `NewSessionModal.tsx:156` uses class `new-session-modal__agent-btn__detail` (`style.css:842-848`): monospace, 11px, color `#565f89`. All correct.

**Shell binary row in SettingsTab (SHELL-11):**
`SettingsTab.tsx:711-739` is structurally identical to the peer AI CLI rows at lines 684-708 and the tailscale row at lines 740-764. The same `settings-panel__cli-name`, `settings-panel__path-row`, `settings-panel__path-input`, and `settings-panel__browse-btn` classes are used. Position is correct: after AI CLI rows (line 708), before tailscale row (line 740). No new CSS classes were introduced.

No arbitrary spacing values found in either component (`[.*px]` / `[.*rem]` grep returned zero hits).

---

### Pillar 2: Typography & Copywriting (4/4)

**Label strings — exact match to UI-SPEC:**

| UI-SPEC requirement | Implemented string | File:line | Status |
|--------------------|--------------------|-----------|--------|
| Shell row label: `Shell` (no em-dash) | `<span>Shell</span>` | `NewSessionModal.tsx:155` | PASS |
| Shell row detail: resolved path | `{resolvedShellPath}` | `NewSessionModal.tsx:156` | PASS |
| Settings cell name: `shell` (lowercase) | `shell` | `SettingsTab.tsx:712` | PASS |
| Settings placeholder: `e.g. /bin/zsh` | `placeholder="e.g. /bin/zsh"` | `SettingsTab.tsx:721` | PASS |
| Browse button text: `Browse` | `Browse` | `SettingsTab.tsx:730` | PASS |

The UI-SPEC §2 explicitly states "Label: `Shell` (no em-dash suffix — there is no per-binary subtype to name)." The implementation correctly omits the em-dash. The em-dash in `NewSessionModal.tsx:67` appears only in a comment string.

The `SHELL_ARGS_PLACEHOLDER` constant (`"Arguments are not passed to shell sessions"`) is used on `NewSessionModal.tsx:8,183`. This string does not appear in UI-SPEC §2, but it covers the args-field-disabled state when Shell is selected. It is clear, accurate, and not generic. No concern.

No other new strings were introduced outside the spec contract.

Font size and weight: No Tailwind typography classes are used in either component (this project uses custom CSS classes). Inline `fontSize` overrides appear only in the pre-existing tailscale diagnostics panel at `SettingsTab.tsx:485-486` — not phase 107 code.

---

### Pillar 3: Color & Contrast (4/4)

**No new tokens introduced.** UI-SPEC §1 states "New tokens: None — all colors already in `style.css`." This holds.

**Phase 107 new code — hardcoded color audit:**

- `NewSessionModal.tsx`: Zero hardcoded hex values in JSX or style props. The comment at line 145 references `#89ddff` as documentation only.
- `SettingsTab.tsx` lines 709-739 (new Shell row): Zero hardcoded hex values or inline style props. All styling delegated to existing CSS classes.
- `App.tsx:547-575` (SHELL-12 handler): No styling.

**Pre-existing hardcoded colors (not phase 107, not in scope):**
`SettingsTab.tsx:485-513` contains inline `style={{ color: '#7aa2f7' }}` etc. in the tailscale diagnostics section. These predate Phase 107 and are not in scope.

**Token correctness:**
- `#89ddff` via `.new-session-modal__agent-btn--selected-shell` (`style.css:838`) — correct shell cyan token for selected-state border.
- `#f7768e` via `.settings-panel__error` — existing destructive token for error paragraph color. Used correctly in the error state at `SettingsTab.tsx:734`.
- AI CLI selected border `#7aa2f7` via `.new-session-modal__agent-btn--selected` (`style.css:816`) — unchanged.

60/30/10 distribution: This is a small-surface phase with no new layout. The existing color distribution is unchanged.

---

### Pillar 4: Accessibility (3/4)

**What passes:**

SHELL-10 — Single shell row carries `aria-pressed={selectedAgent === 'shell'}` at `NewSessionModal.tsx:152`. Screen readers will announce "Shell, toggle button, pressed/not pressed." Consistent with the AI CLI rows above it.

SHELL-11 — ARIA chain when error is present:
- Input: `id="settings-shell-path"`, `aria-label="Shell binary path"`, `aria-describedby="settings-shell-path-desc"` (`SettingsTab.tsx:716-723`)
- Error paragraph: `id="settings-shell-path-desc"`, `role="alert"` (`SettingsTab.tsx:734`)
- When an error exists, the chain is intact: the `role="alert"` live region fires on save failure; `aria-describedby` links input to the error text.

Browse button in SettingsTab: `title="Browse for shell executable"` at `SettingsTab.tsx:728`, consistent with peer Browse buttons (`title="Browse for executable"`).

SHELL-12 — ExitToast unchanged. Existing `role="alert"` / `aria-live="polite"` preserved for non-zero exits. For exit-code 0 with auto-close ON, no toast renders, which is appropriate (completed task closes silently).

**Defects:**

WARNING — Browse button in NewSessionModal (`NewSessionModal.tsx:167-172`) has no `aria-label` or `title`. The button text is "Browse…" or "Browsing…" (loading state). While "Browse…" provides some context, the accessible name gives no indication of what is being browsed (working directory). The SettingsTab Browse buttons all carry `title` attributes for this reason. This is a minor inconsistency, not introduced by Phase 107, but the modal was touched in this phase and the gap is easy to close.

WARNING (deferred, IN-03) — `aria-describedby="settings-shell-path-desc"` is unconditional on the shell-path input (`SettingsTab.tsx:723`). The referenced element only exists when `shellPathError` is truthy (`SettingsTab.tsx:733`). When no error is present, the attribute points to a non-existent id. Assistive technologies tolerate dangling `aria-describedby` silently (WCAG 2.1 SC 1.3.1 advisory), but axe-core will flag it. The fix is specified in 107-REVIEW.md IN-03. Deferred to v3.4 per scope decision — not downgrading harshly per audit brief.

Score rationale: The two aria issues are both real but neither blocks user task completion. The SHELL-11 error chain — the one ARIA feature that is new in this phase — works correctly in the error-present state where it matters most.

---

### Pillar 5: Interaction & Feedback (4/4)

**CR-01 (autoCloseRef guard) — CONFIRMED IN SOURCE:**
`App.tsx:554-561` — the `exitCode === 0` branch now checks `autoCloseRef.current` before closing. When ON: immediate close, no toast. When OFF: falls through to `setSessionExits` with `countdown: -1`, ExitToast appears. The fix matches the CR-01 spec in 107-REVIEW.md exactly. `autoCloseRef` is populated from `GetAutoCloseSession()` at `App.tsx:397`.

**WR-02 (false Saved! indicator) — CONFIRMED IN SOURCE:**
`SettingsTab.tsx:259-271` — `shellPathOk = true` flag is set; the inner catch sets it to false on `SetShellPath` rejection; `setSaved(true)` is inside `if (shellPathOk)`. A validation failure renders the error paragraph and does not trigger the success state. The `setSaved` timeout (1500ms) also only fires on `shellPathOk`, avoiding the scenario where "Saved!" and the error paragraph were visible simultaneously.

**Shell row — GetShellPath on each open:**
`NewSessionModal.tsx:60-65` — `useEffect` on `[isOpen]` calls `GetShellPath().then(setResolvedShellPath)`. Modal opens always refresh the displayed path. Path changes in Settings are reflected immediately on next modal open without a page reload.

**SettingsTab Browse flow:**
`SettingsTab.tsx:289-295` — `handleShellBrowse` passes the current directory (derived from shellPath) to `OpenFileDialog`, consistent with peer Browse handlers. Result updates shellPath state (controlled input).

**Create Session button:**
`NewSessionModal.tsx:200-206` — Disabled when `!selectedAgent || creating`. Shows "Creating…" after click. The shell confirm path at lines 103-108 correctly sends bare `'shell'` and does not persist args.

**Save Paths button:**
`SettingsTab.tsx:773-776` — Disabled during `saving || saved`. Shows "Saving…" / "Saved!" / "Save Paths" per state. Now correctly suppresses "Saved!" on shell-path validation failure (WR-02).

**ExitToast — no-toast on clean exit:**
`ExitToast.tsx:21-23` — returns null when `exits` has no entries. SHELL-12 never adds an entry for `exitCode === 0` with auto-close ON. Component unchanged, behavior change is entirely in App.tsx handler.

---

### Pillar 6: Responsive & Robustness (3/4)

**Robustness — passes:**

Empty shell path (SHELL-10): `NewSessionModal.tsx:58,64` — `resolvedShellPath` initializes to `''`; the catch sets it to `''` on RPC failure. An empty detail span renders (`{resolvedShellPath}` — empty string). The row remains selectable. No skeleton, no disabled state — correct per UI-SPEC §3 edge case.

Settings field on empty path (SHELL-11): `SettingsTab.tsx:113,146` — `shellPath` initializes to `''`; `GetShellPath()` on mount populates it with the daemon-resolved default (which the daemon guarantees is non-empty). If the RPC fails, the catch sets `shellPath` to `''`, leaving the input empty but functional. The `placeholder="e.g. /bin/zsh"` is shown.

Invalid path error (SHELL-11): Daemon 400 response is caught at `SettingsTab.tsx:263-266`, sets `shellPathError` to the daemon's error message, renders inline below the field at line 733-737. Layout shift: the `<p>` appears below the `<div class="settings-panel__path-row">` — same DOM position as error paragraphs on peer rows. No layout shift to sibling rows (table cells contain the error).

**Dead UI (WARNING, deferred IN-02):**

`App.tsx:1064` filters `sessionExits` for `exitCode === 0 && !cancelled && countdown > 0` to build `exitCountdowns` for TabBar. `App.tsx:1175` renders `ExitCountdownBanner` under the same condition. After SHELL-12, no `sessionExits` entry will ever have `countdown > 0` — the fallthrough path (auto-close OFF) sets `countdown: -1`. Both render paths compile and produce no runtime error, but they are permanently unreachable.

This is a code quality issue rather than a user-visible regression — users see no broken UI. However, the dead paths will mislead future maintainers into thinking a countdown feature is still active. The IN-02 decision to defer cleanup to v3.4 is reasonable given it is informational-only.

Score rationale: The dead-UI issue reduces confidence in the codebase state but has no user-visible impact. The robustness of the core error and empty-state flows is solid.

---

## Registry Safety

No `components.json` found — shadcn not initialized. Registry audit skipped.

---

## Files Audited

| File | Scope |
|------|-------|
| `frontend/src/components/NewSessionModal.tsx` | Full file (212 lines) — SHELL-10 |
| `frontend/src/components/SettingsTab.tsx` | Lines 1-300 (state + handlers) + 684-769 (table section) — SHELL-11 |
| `frontend/src/App.tsx` | Lines 540-576 (session:exit handler, CR-01 fix) |
| `frontend/src/components/ExitToast.tsx` | Full file (70 lines) — SHELL-12 no-change verification |
| `frontend/src/style.css` | Lines 797-860 (agent-btn + shell-btn CSS) |
| `107-UI-SPEC.md` | Design contract baseline |
| `107-REVIEW.md` | Post-fix state reference |
| `107-VERIFICATION.md` | Verification evidence reference |
