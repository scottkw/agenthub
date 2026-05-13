---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
type: ui-spec
status: locked
mode: minimal-reuse (existing patterns; no new tokens or components)
gathered: 2026-05-13
---

# Phase 107 UI-SPEC: Shell UX Collapse + Binary Path Picker + Clean-Exit Handling

All design decisions pre-locked in `107-CONTEXT.md §Decisions`. This document
specifies the minimal visual contract needed to implement the three deltas.

---

## 1. Locked Design Decisions

| Decision | Value | Source |
|----------|-------|--------|
| Design system | TokyoNight dark, existing tokens only | CONTEXT.md |
| New tokens | None — all colors already in `style.css` | CONTEXT.md |
| New components | None | CONTEXT.md |
| Shell selected-state border | `#89ddff` via `.new-session-modal__agent-btn--selected-shell` | `style.css:838` |
| AI CLI selected-state border | `#7aa2f7` via `.new-session-modal__agent-btn--selected` | `style.css:816` |
| Settings path-field pattern | `settings-panel__path-row` table row: text input + Browse button | `SettingsTab.tsx:654-671` |
| Em-dash in row label | U+2014 (`—`) | `NewSessionModal.tsx:169` |

---

## 2. Element Specs

### SHELL-10 — Single Shell row in NewSessionModal

**Requirement:** Collapse the `sortedShells.map(...)` loop (L154-172) to one
static `<button>` element. Remove `shellsLoading` skeleton.

**Row structure** — reuse the existing shell-row markup verbatim:

```tsx
<button
  className={[
    'new-session-modal__agent-btn',
    'new-session-modal__agent-btn--shell',
    selectedAgent === 'shell'
      ? 'new-session-modal__agent-btn--selected-shell'
      : '',
  ].filter(Boolean).join(' ')}
  aria-pressed={selectedAgent === 'shell'}
  onClick={() => handleSelectShell('shell')}
>
  <span>Shell</span>
  <span className="new-session-modal__agent-btn__detail">{resolvedShellPath}</span>
</button>
```

- **Label:** `Shell` (no em-dash suffix — there is no per-binary subtype to name).
- **Detail line:** resolved path from `paths.shell` settings, e.g. `/bin/zsh`.
  Shown in `.new-session-modal__agent-btn__detail` (mono, muted, existing class).
- **Agent id:** `'shell'` (bare string, no `shell:` prefix). `handleSelectShell('shell')`
  already strips the prefix before calling `onConfirm`; the bare name is what the
  daemon expects.
- **Selected border:** `#89ddff` (class `--selected-shell`, unchanged from Phase 101).
- **Hover state:** `.new-session-modal__agent-btn:hover` — unchanged.
- **Props to remove:** `shells`, `shellsLoading` can be dropped from
  `NewSessionModalProps` once SHELL-11 ships (the `shells` list moves to SettingsTab
  for the dropdown suggestions). Until then, keep props but stop rendering the loop.
- **`resolvedShellPath` source:** call new Wails RPC `GetShellPath()` on modal open
  (returns the daemon-resolved default). Cache in local state; no loading skeleton —
  if the RPC is slow, show an empty detail line rather than a skeleton.

**What is removed:**
- `sortedShells` memo
- `sortedShells.map(...)` JSX block
- `shellsLoading && sortedShells.length === 0` skeleton block
- `SHELL_PREFIX` parsing in `handleConfirm` (agent id is always bare `'shell'`)

---

### SHELL-11 — Settings → Paths "Shell binary" field

**Requirement:** Add one new `<tr>` to the existing `settings-panel__table` under
`<h3 id="settings-paths">`. Pattern is identical to every other CLI row in that table.

**Row markup** — copy the `tailscale` row pattern at `SettingsTab.tsx:675-699`:

```tsx
<tr key="shell">
  <td className="settings-panel__cli-name">shell</td>
  <td>
    <div className="settings-panel__path-row">
      <input
        id="settings-shell-path"
        className="settings-panel__path-input"
        type="text"
        value={shellPath}
        onChange={(e) => setShellPath(e.target.value)}
        placeholder="e.g. /bin/zsh"
        aria-label="Shell binary path"
        aria-describedby="settings-shell-path-desc"
      />
      <button
        className="settings-panel__browse-btn"
        onClick={() => void handleShellBrowse()}
        title="Browse for shell executable"
      >
        Browse
      </button>
    </div>
    {shellPathError && (
      <p id="settings-shell-path-desc" className="settings-panel__error" role="alert">
        {shellPathError}
      </p>
    )}
  </td>
</tr>
```

- **Position in table:** after the last AI CLI row, before the tailscale row.
- **Browse behavior:** `OpenFileDialog(shellPath)` — same Wails call as other CLI
  rows; no "pick from discovered" dropdown in this iteration. Discovered-shells list
  (from `GET /shells`) is used only by the `NewSessionModal` CLI flag path. The field
  is free-text with Browse; users who want `bash` type `/bin/bash`.
- **Save:** participates in the existing `handleSaveCLIPaths` button flow. On save,
  call `SetShellPath(shellPath)` (new Wails RPC). On daemon 400 response, surface
  the error message in `settings-panel__error` below the field (id `settings-shell-path-desc`).
- **Load on mount:** call `GetShellPath()` to populate initial value alongside
  `GetCLIPaths()`. If unset, daemon returns resolved default (`$SHELL` / `/bin/zsh`
  etc.); field shows that resolved path as a non-empty value, not a placeholder.
- **Validation display:** inline `<p className="settings-panel__error">` below the
  `<div class="settings-panel__path-row">`. Same error color (`#f7768e`) as existing
  CLI path errors. Error text from daemon: pass through verbatim (daemon returns
  human-friendly message on 400).
- **No new CSS classes.** All classes already exist in `style.css`.

**ARIA:**
- Input `id="settings-shell-path"` + `aria-label="Shell binary path"`
- Error paragraph `id="settings-shell-path-desc"` + `role="alert"` (live region —
  announced on save failure)
- Browse button `title="Browse for shell executable"` (tooltip, consistent with
  existing browse buttons which already use `title`)

---

### SHELL-12 — Auto-close tab on exit-code 0

**Requirement:** Shell tab that exits cleanly closes immediately — no ExitToast, no
countdown. Non-zero exit still shows toast with existing behavior.

**App.tsx `session:exit` handler — new branch logic:**

```
if (data.exitCode === 0) {
  // Close tab immediately — no toast, no countdown
  void handleCloseTabRef.current?.(data.sessionId)
  // Do NOT call setSessionExits for this session
} else {
  // Existing behavior: record exit state, show ExitToast
  setSessionExits(prev => ({ ...prev, [data.sessionId]: exitState }))
  // (no countdown for non-zero exits — existing behavior unchanged)
}
```

- **No flash:** the tab must close before any render cycle shows the toast. The
  early-return pattern above achieves this — `setSessionExits` is never called for
  exit-code 0.
- **Focus shift on active-tab close:** existing `handleCloseTab` logic at
  `App.tsx:703` already shifts focus to the left tab or Welcome if the closed tab was
  active. No change needed — just invoke `handleCloseTabRef.current` as above.
- **ExitToast component:** no code change to `ExitToast.tsx`. The component renders
  only what is in `exits` prop; SHELL-12 simply never adds exit-code-0 entries to
  that map.
- **Session state `stopped`:** the daemon (not frontend) sets state to `stopped` on
  exit-code 0 (CONTEXT.md §SHELL-12 backend fix). The frontend already reads session
  state from the daemon's session list; no frontend state-machine change needed.

**State transitions:**

| Event | exitCode | Tab closes? | Toast shown? | Focus shift? |
|-------|----------|------------|-------------|-------------|
| Shell exits normally | 0 | Yes (immediate) | No | Yes — left tab or Welcome |
| Shell exits with error | ≠ 0 | No | Yes (existing) | No |
| PTY EOF normalized from -1 | 0 (after normalization) | Yes (immediate) | No | Yes |

---

## 3. Edge Cases

### SHELL-10 — Empty / unresolved shell path

If `GetShellPath()` returns empty (daemon error or race), the detail line renders
as an empty string — acceptable; the row is still present and selectable. Do not
show a loading skeleton or disable the row. The user can still create a shell
session; the daemon will use its platform default.

### SHELL-11 — Invalid path on PATCH

- Daemon returns HTTP 400 with a plain-text error body (e.g., `"path /foo/bar does
  not exist or is not executable"`).
- Frontend catches the rejection, sets `shellPathError` state to that message,
  renders it in the `settings-panel__error` paragraph below the field.
- The Save Paths button re-enables (not permanently locked).
- On next successful save, clear `shellPathError`.

### SHELL-12 — Active tab auto-close

- If the auto-closing shell tab is the active tab, `handleCloseTab` selects the tab
  immediately to its left. If it was the only non-Welcome tab, focus returns to
  the Welcome tab. This is the existing `handleCloseTab` adjacency logic — no
  special case needed.
- If the user has multiple exit-code-0 shell tabs close in rapid succession,
  each `handleCloseTab` call operates on the tab list at that moment; the last one
  standing falls back to Welcome.

---

## 4. Test Contract

Tests must assert behavior from source inspection and DOM render. No new test
infrastructure — run in existing `pnpm test` (Vitest) harness.

### SHELL-10 — `NewSessionModal.shellRow.test.tsx` (new)

| # | Assert |
|---|--------|
| 1 | Renders exactly **one** button with class `new-session-modal__agent-btn--shell` regardless of `shells` prop length |
| 2 | That button has `aria-pressed={true}` when `selectedAgent === 'shell'` |
| 3 | Button contains a `<span>` with text `Shell` and a `.agent-btn__detail` `<span>` showing the resolved path |
| 4 | No `Loading shells…` skeleton renders when `shellsLoading={true}` |
| 5 | Clicking the row calls `onConfirm` with bare `'shell'` (no prefix) |
| 6 | Existing AI CLI rows are unchanged |

### SHELL-11 — `SettingsTab.shellPath.test.tsx` (new)

| # | Assert |
|---|--------|
| 1 | Renders a `<tr>` with `settings-panel__cli-name` cell containing text `shell` |
| 2 | Input `id="settings-shell-path"` is present with `aria-label="Shell binary path"` |
| 3 | Input value initializes from `GetShellPath()` mock return |
| 4 | Typing in the input updates local state (controlled input) |
| 5 | Browse button calls `OpenFileDialog` and updates input value |
| 6 | Save button calls `SetShellPath` with current input value |
| 7 | On `SetShellPath` rejection, error paragraph with `role="alert"` renders with the error message |
| 8 | Error paragraph id matches input's `aria-describedby` |

### SHELL-12 — `App.shellExit.test.tsx` (new)

| # | Assert |
|---|--------|
| 1 | `session:exit` event with `exitCode: 0` does NOT add entry to `sessionExits` state |
| 2 | `session:exit` event with `exitCode: 0` calls `handleCloseTab` with the session id |
| 3 | `session:exit` event with `exitCode: 1` adds entry to `sessionExits` state and does NOT call `handleCloseTab` |
| 4 | `ExitToast` receives no entries for exit-code-0 sessions |
| 5 | When the active tab closes, `activeId` shifts to the adjacent tab id (or `'welcome'`) |

---

## 5. Accessibility

### SHELL-10

No new ARIA attributes beyond what the existing shell button row already uses
(`aria-pressed`). The single static row is structurally identical to Phase 101's
multi-row output; screen readers announce "Shell, toggle button, pressed/not pressed".

### SHELL-11

| Element | ARIA attribute | Value |
|---------|---------------|-------|
| Path input | `id` | `settings-shell-path` |
| Path input | `aria-label` | `"Shell binary path"` |
| Path input | `aria-describedby` | `"settings-shell-path-desc"` (points at error paragraph) |
| Error paragraph | `id` | `settings-shell-path-desc"` |
| Error paragraph | `role` | `"alert"` (live region — announced on save failure) |
| Browse button | `title` | `"Browse for shell executable"` |

No changes to Browse button beyond the title — consistent with existing CLI Browse
buttons which are icon-free and title-described.

### SHELL-12

No ARIA changes. ExitToast's existing `role="alert"` / `aria-live="polite"` is
preserved for non-zero exits. For exit-code 0, no toast renders, so no ARIA
announcement is made — the tab simply disappears, which is standard expected behavior
for a completed task.
