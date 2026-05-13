---
phase: 107
plan: "107-03"
type: execute
status: pending
wave: 1
depends_on: ["107-01"]
requirements: [SHELL-10, SHELL-11]
files_modified:
  - frontend/src/components/NewSessionModal.tsx
  - frontend/src/components/SettingsTab.tsx
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/NewSessionModal.shellRow.test.tsx
  - frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx
autonomous: true
must_haves:
  truths:
    - "NewSessionModal renders exactly ONE shell row labeled 'Shell' regardless of how many shells DiscoverShells returned."
    - "The shell row's detail line shows the resolved `paths.shell` value from the daemon (e.g., `/bin/zsh`)."
    - "Clicking the shell row and confirming sends bare `'shell'` (no `shell:` prefix) to the daemon CreateSession call."
    - "The `Loading shells…` skeleton is GONE from the modal."
    - "Settings → Paths section contains a new row labeled 'shell' with a text input, a Browse button, and an inline error paragraph that renders only on PATCH failure."
    - "The shell-path input is pre-populated from `GetShellPath()` on mount (showing the daemon-resolved default when unset)."
    - "Clicking Save Paths calls `SetShellPath(currentInputValue)`; daemon 400 surfaces a `role=\"alert\"` error paragraph with the daemon's verbatim message."
  artifacts:
    - path: frontend/src/components/NewSessionModal.tsx
      provides: "Single-row Shell entry reading paths.shell via Wails RPC; removed sortedShells loop + shellsLoading + SHELL_PREFIX"
      contains: "handleSelectShell"
    - path: frontend/src/components/SettingsTab.tsx
      provides: "Shell binary path row in Paths section + GetShellPath/SetShellPath wiring"
      contains: "settings-shell-path"
    - path: frontend/src/components/__tests__/NewSessionModal.shellRow.test.tsx
      provides: "Vitest suite locking the 6 SHELL-10 assertions from UI-SPEC §4"
    - path: frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx
      provides: "Vitest suite locking the 8 SHELL-11 assertions from UI-SPEC §4"
  key_links:
    - from: NewSessionModal.tsx
      to: GetShellPath Wails binding
      via: "useEffect on modal open → setResolvedShellPath"
    - from: SettingsTab.tsx
      to: GetShellPath / SetShellPath Wails bindings
      via: "useEffect on mount load + handleSaveCLIPaths write"
    - from: handleSaveCLIPaths
      to: shellPathError state → role='alert' paragraph
      via: "try/catch around SetShellPath"
---

<objective>
SHELL-10 + SHELL-11 frontend: collapse the new-session modal's per-shell row loop into ONE static `Shell` row that reads `paths.shell` via the new GetShellPath Wails RPC, AND add a "Shell binary" path-row to Settings → Paths that participates in the existing Save Paths flow with daemon-side validation. Wave 1 (consumes 107-01's GetShellPath/SetShellPath bindings).

Purpose: First-user feedback was "too many shells, just give me one" — multi-binary picker confused users who don't care about shell brand. SHELL-10 reverts to one row; SHELL-11 moves the (still-needed) binary choice to Settings where power users can find it without polluting the modal.

Output: Modal that always shows one Shell row. Settings field that round-trips through 107-01's daemon plumbing. Two new Vitest suites locking the UI-SPEC §4 test contract. No backend changes — pure frontend consumer of 107-01.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-CONTEXT.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-UI-SPEC.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-01-SUMMARY.md

@frontend/src/components/NewSessionModal.tsx
@frontend/src/components/SettingsTab.tsx
@frontend/src/components/__tests__/SettingsTab.persistence.test.tsx

<interfaces>
New Wails bindings shipped by 107-01 (consume these — do not redefine):

```typescript
// From frontend/src/wailsjs/go/main/App.d.ts after 107-01:
export function GetShellPath(): Promise<string>
export function SetShellPath(v: string): Promise<void>
```

Existing analogue we are mirroring for Settings field — `tailscale` row at SettingsTab.tsx:675-699:

```tsx
<tr key="tailscale">
  <td className="settings-panel__cli-name">tailscale</td>
  <td>
    <div className="settings-panel__path-row">
      <input
        className="settings-panel__path-input"
        type="text"
        value={customPaths['tailscale'] ?? ''}
        onChange={(e) =>
          setCustomPaths((prev) => ({ ...prev, tailscale: e.target.value }))
        }
        placeholder="Path to tailscale (leave blank to auto-detect)"
      />
      <button
        className="settings-panel__browse-btn"
        onClick={() => void handleBrowse('tailscale')}
        title="Browse for executable"
      >
        Browse
      </button>
    </div>
  </td>
</tr>
```

UI-SPEC §2 SHELL-10 locked markup (NewSessionModal single-row):
```tsx
<button
  className={[
    'new-session-modal__agent-btn',
    'new-session-modal__agent-btn--shell',
    selectedAgent === 'shell' ? 'new-session-modal__agent-btn--selected-shell' : '',
  ].filter(Boolean).join(' ')}
  aria-pressed={selectedAgent === 'shell'}
  onClick={() => handleSelectShell('shell')}
>
  <span>Shell</span>
  <span className="new-session-modal__agent-btn__detail">{resolvedShellPath}</span>
</button>
```

UI-SPEC §2 SHELL-11 locked markup (Settings field):
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
</interfaces>

</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Collapse NewSessionModal to a single Shell row + RED-first vitest suite</name>
  <files>frontend/src/components/NewSessionModal.tsx, frontend/src/components/__tests__/NewSessionModal.shellRow.test.tsx</files>
  <behavior>
    Per UI-SPEC §4 SHELL-10 test contract — six assertions in the new test file:
    1. Renders exactly ONE button with class `new-session-modal__agent-btn--shell` regardless of `shells` prop length (test with shells=[], shells=[zsh,bash], shells=[zsh,bash,pwsh] — all yield count 1).
    2. That button has `aria-pressed=true` when selectedAgent === 'shell'.
    3. Button contains a `<span>` with text `Shell` and a `.new-session-modal__agent-btn__detail` `<span>` showing the resolved path (mocked GetShellPath returns "/bin/zsh"; assert that string is in the document).
    4. No `Loading shells…` skeleton renders even when `shellsLoading=true`.
    5. Clicking the row, then confirming, calls onConfirm with bare `'shell'` (NOT `'shell:bash'` or any prefix).
    6. Existing AI CLI rows are unchanged — render clis=[{Name:'claude',Path:'/usr/local/bin/claude'}] and assert the claude row still appears and is clickable.

    Implementation behavior:
    - Remove `SHELL_PREFIX` constant (line 8).
    - Remove `sortedShells` useMemo (lines 58-66).
    - Remove the `sortedShells.map(...)` block (lines 154-173).
    - Remove the `shellsLoading && sortedShells.length === 0` skeleton (lines 174-181).
    - Replace with the locked single-row markup from UI-SPEC §2.
    - Add `const [resolvedShellPath, setResolvedShellPath] = useState('')` state.
    - Add useEffect on mount (or when isOpen becomes true) that calls `GetShellPath()` and setResolvedShellPath on the result. No loading skeleton; empty string is acceptable until the RPC resolves.
    - Remove `isShellSelected = selectedAgent.startsWith(SHELL_PREFIX)` and replace with `isShellSelected = selectedAgent === 'shell'`.
    - In handleSelectShell, set selectedAgent to the bare `'shell'` (no prefix) and clear argsText (shell args are still placeholdered as informational-only).
    - In handleConfirm's shell branch, just call `onConfirm('shell', selectedDir, [])` — no prefix-strip needed.
    - Keep `shells` and `shellsLoading` props in the interface for now (still consumed by App.tsx call site at line 1217), but mark them with a TSDoc comment indicating they are unused by the modal and pending removal in a future cleanup. Do NOT change the App.tsx call site — that keeps the prop wiring intact while we ship.
  </behavior>
  <action>
    (1) Create frontend/src/components/__tests__/NewSessionModal.shellRow.test.tsx FIRST (TDD-style, even though TDD mode is off — this is a UI-SPEC test-contract that's clearer to write before the markup edit). Pattern from existing NewSessionModal.test.tsx in the same directory; mock `'../../wailsjs/go/main/App'` with `vi.mock(...)` returning `GetShellPath: vi.fn().mockResolvedValue('/bin/zsh')`. Use `@testing-library/react` `render`, `screen`, `fireEvent`, `act`/`waitFor` for async-state. Write all six assertions per the behavior list. Run the file once and confirm it FAILS against current NewSessionModal (proves the test is meaningful).

    (2) Edit frontend/src/components/NewSessionModal.tsx:
        - Delete SHELL_PREFIX (line 8) and any references to it.
        - Delete the sortedShells useMemo (lines 58-66).
        - Delete the sortedShells.map block (lines 154-173).
        - Delete the shellsLoading skeleton block (lines 174-181).
        - Import GetShellPath from '../wailsjs/go/main/App' — add to the existing import line at top.
        - Add `const [resolvedShellPath, setResolvedShellPath] = useState('')` near the other useState declarations.
        - Add a useEffect (with empty dep array OR dep on `isOpen`) that calls `GetShellPath().then(setResolvedShellPath).catch(() => setResolvedShellPath(''))`. Use the `isOpen` dep so the RPC is re-issued whenever the modal opens (user may have changed the Settings field between modal opens) — this matches the UI-SPEC §3 edge case where a slow RPC degrades to an empty detail line without a skeleton.
        - Change `isShellSelected` derivation to `selectedAgent === 'shell'`.
        - In handleSelectShell, accept the single bare name `'shell'`: `setSelectedAgent('shell'); setArgsText(localStorage.getItem(SHELL_ARGS_KEY('shell')) ?? '')`.
        - In handleConfirm shell branch: `onConfirm('shell', selectedDir, []); return`.
        - In the JSX inside the agent-list `<div>`, after the clis.map closing, insert the new single-row markup (from UI-SPEC §2 verbatim). Drop the comment block at lines 147-153 (no longer accurate).

    (3) Re-run the test file — all six assertions should now pass.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub/frontend && pnpm test --run src/components/__tests__/NewSessionModal.shellRow.test.tsx</automated>
  </verify>
  <done>
    Six new test assertions pass. Existing NewSessionModal.test.tsx still passes (no regression). `grep -c "SHELL_PREFIX\|sortedShells" frontend/src/components/NewSessionModal.tsx` returns 0. `grep -c "Loading shells" frontend/src/components/NewSessionModal.tsx` returns 0. `grep -c "new-session-modal__agent-btn--shell" frontend/src/components/NewSessionModal.tsx` returns exactly 1 (the single static row class).
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Add "Shell binary" row to Settings → Paths + RED-first vitest suite</name>
  <files>frontend/src/components/SettingsTab.tsx, frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx</files>
  <behavior>
    Per UI-SPEC §4 SHELL-11 test contract — eight assertions in the new test file:
    1. Renders a `<tr>` with `settings-panel__cli-name` cell containing text `shell`.
    2. Input `id="settings-shell-path"` is present with `aria-label="Shell binary path"`.
    3. Input value initializes from `GetShellPath()` mock return (e.g., `/bin/zsh`).
    4. Typing in the input updates local state (controlled input behavior).
    5. Browse button calls `OpenFileDialog` and updates input value with the dialog's return value.
    6. Save button calls `SetShellPath` with the current input value.
    7. On `SetShellPath` rejection (mock rejects with `new Error("path /foo does not exist or is not executable")`), an error paragraph with `role="alert"` renders containing that error message.
    8. The error paragraph's id matches the input's aria-describedby (both `settings-shell-path-desc`).

    Implementation behavior:
    - On mount, alongside existing GetCLIPaths fetch, call GetShellPath and set local `shellPath` state.
    - Render the new `<tr key="shell">` immediately AFTER the `clis.map` rows and BEFORE the `tailscale` row (per UI-SPEC §2 "after the last AI CLI row, before the tailscale row").
    - Browse button calls `OpenFileDialog(shellPath)` (the existing Wails RPC, already imported at SettingsTab.tsx top). On non-empty return, setShellPath to the chosen path.
    - On Save Paths click (existing `handleSaveCLIPaths`), after the existing CLI-paths saves, call `SetShellPath(shellPath.trim())` inside a try/catch. On rejection, setShellPathError to the error message (use `err instanceof Error ? err.message : String(err)`). On success, clear shellPathError. The existing `saved`/`saving` state should still cycle correctly (no separate "saving shell path" indicator — it's part of the same save).
    - shellPathError state defaults to '' and clears on successful save.
  </behavior>
  <action>
    (1) Create frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx using SettingsTab.persistence.test.tsx as the template (same vi.mock structure for `'../../wailsjs/go/main/App'`). Mock GetShellPath/SetShellPath/GetCLIPaths/OpenFileDialog. Write all eight assertions. Confirm the test file FAILS against current SettingsTab.

    (2) Edit frontend/src/components/SettingsTab.tsx:
        - Add imports: extend the existing `import { GetCLIPaths, ... } from '../wailsjs/go/main/App'` to include `GetShellPath` and `SetShellPath`. (OpenFileDialog should already be imported per existing Browse pattern; if not, add it.)
        - Add state: `const [shellPath, setShellPath] = useState('')` and `const [shellPathError, setShellPathError] = useState('')` near other useState calls.
        - In the existing mount useEffect that calls `GetCLIPaths` (~L133), add a parallel `GetShellPath().then(setShellPath).catch(() => setShellPath(''))` call.
        - Add a `handleShellBrowse` async function near the existing `handleBrowse(cliName)` (~L255). It calls `OpenFileDialog(shellPath)` and setShellPath on non-empty return.
        - In `handleSaveCLIPaths` (~L226), after the existing CLI paths save loop and the tailscale save, add:
          ```ts
          try {
            await SetShellPath(shellPath.trim())
            setShellPathError('')
          } catch (err) {
            setShellPathError(err instanceof Error ? err.message : String(err))
            // Continue — partial save (other paths saved) is acceptable per existing tailscale pattern.
          }
          ```
        - In JSX, immediately after the closing `))}` of the `clis.map` and BEFORE the `{!clis.find(c => c.Name === 'tailscale') && (` block, insert the locked SHELL-11 row markup from UI-SPEC §2 verbatim.
        - Do NOT alter the `tailscale` row, the Save Paths button, or any other existing structure.

    (3) Re-run the test file — all eight assertions should pass. Re-run the existing SettingsTab.persistence.test.tsx and SettingsTab.hyperlinked-index.test.tsx to confirm no regression.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub/frontend && pnpm test --run src/components/__tests__/SettingsTab.shellPath.test.tsx src/components/__tests__/SettingsTab.persistence.test.tsx src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx</automated>
  </verify>
  <done>
    Eight new tests pass; existing SettingsTab.persistence + .hyperlinked-index tests still pass. `grep -c "settings-shell-path" frontend/src/components/SettingsTab.tsx` returns 2 (input id + error paragraph id). The new row is rendered between the clis.map output and the tailscale row (DOM order verified by the new test).
  </done>
</task>

</tasks>

<verification>
- `cd frontend && pnpm test --run` — full Vitest suite green (existing tests unaffected; 14 new assertions added across two files).
- `cd frontend && pnpm typecheck` — no TypeScript errors. (The new GetShellPath/SetShellPath signatures from 107-01 are imported and typed.)
- Manual UAT (dev-browser optional): `pnpm dev` → open new-session modal → confirm exactly one Shell row appears with `/bin/zsh` detail line. Open Settings → Paths, type `/bin/bash`, click Save Paths, confirm green Saved state. Re-open new-session modal → detail line now reads `/bin/bash`. Type `/no/such/path` in Settings, click Save Paths, confirm inline error appears below the field.
</verification>

<success_criteria>
- UI-SPEC §4 test contracts SHELL-10 (6 assertions) and SHELL-11 (8 assertions) are wholly satisfied by the two new test files.
- "Critical invariant: No regression of SHELL-01..09" preserved: NewSessionModal.test.tsx still passes; the daemon `pty.DiscoverShells()` API is untouched; the CLI `agenthub new shell --shell=bash` flag still works (still uses cliPaths["bash"] path).
- Settings persistence parity: GetShellPath returns the resolved default on first load (never empty); SetShellPath writes through to settings.json via 107-01's daemon plumbing.
- ARIA contract: input has `aria-label`, error paragraph has `role="alert"` + `id` matching the input's `aria-describedby` — screen readers announce save failures.
</success_criteria>

<output>
After completion, create `.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-03-SUMMARY.md` covering: modal collapse diff (lines removed vs added), Settings field placement, new test counts (6 + 8 = 14 assertions), confirmation that the unused `shells`/`shellsLoading` props are flagged for future cleanup but not removed in this plan. Note: the Wails GetShellPath RPC fires on every modal open per UI-SPEC §2 — this is intentional so changes to the Settings field reflect in the modal without a full reload.
</output>
