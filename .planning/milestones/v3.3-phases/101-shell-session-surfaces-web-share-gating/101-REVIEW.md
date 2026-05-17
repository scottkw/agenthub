---
phase: 101-shell-session-surfaces-web-share-gating
reviewed: 2026-05-12T00:00:00Z
depth: standard
files_reviewed: 22
files_reviewed_list:
  - app.go
  - internal/daemon/engine.go
  - internal/daemon/api.go
  - internal/daemon/client.go
  - internal/daemon/engine_test.go
  - internal/daemon/api_test.go
  - frontend/src/App.tsx
  - frontend/src/components/NewSessionModal.tsx
  - frontend/src/components/TabBar.tsx
  - frontend/src/components/ShellWebShareBanner.tsx
  - frontend/src/components/__tests__/NewSessionModal.test.tsx
  - frontend/src/components/__tests__/TabBar.test.tsx
  - frontend/src/components/__tests__/ShellWebShareBanner.test.tsx
  - frontend/src/components/__tests__/App.shellWebShare.test.tsx
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - frontend/src/wailsjs/go/models.ts
  - frontend/src/test-setup.ts
  - frontend/vite.config.ts
  - cmd_cli.go
  - main.go
  - internal/tui/modal.go
  - internal/tui/tui.go
  - internal/tui/model.go
  - internal/tui/styles.go
  - internal/tui/update.go
findings:
  critical: 2
  warning: 9
  info: 6
  total: 17
status: issues_found
---

# Phase 101: Code Review Report

**Reviewed:** 2026-05-12
**Depth:** standard
**Files Reviewed:** 22 source files (plus 4 test files spot-checked, plus locked UI-SPEC/RESEARCH for copy/race contract verification)
**Status:** issues_found

## Summary

Phase 101 introduces shell sessions as a first-class agent type across all three surfaces (GUI/CLI/TUI) plus a one-time security-banner gate on web sharing. The implementation is generally careful: the daemon-side persistence is small and well-tested, the locked UI-SPEC copy is rendered verbatim, and the WCAG-locked colors are applied consistently. Security posture for the shell argv path is sound (no caller args forwarded; daemon-side `isShellSession` allowlist gates the spawn branch; CLI flag values are strictly allowlisted).

However the **banner confirmation flow has two notable race/UX bugs**: the `pendingShellWebToggle` state is cleared synchronously *before* `Promise.all([SetShellWebShareWarned, ToggleWebServing])` is awaited (CR-01), which (a) renders the banner's `enabling` / `Enabling…` transient state dead code and (b) makes the rollback path on partial failure inconsistent with what the daemon actually accepted. The Esc-key listener in the banner is attached at `document` level with no focus/active-modal gating (CR-02), creating a real risk of dismissing the banner when the user meant to dismiss a different modal that may layer on top (kill confirm, rename, find bar, etc.).

A cluster of WARNING-level issues centers on (a) the canonical "is this a shell" set being duplicated across six call-sites with no shared source of truth, (b) silent disk-write failure in `SetShellWebShareWarned`, and (c) inconsistent error handling between `handleShellWebShareConfirm`'s partial failure path and `webEnabled`/`shellWebShareWarned` local state.

The verification report claims "26/26 vitest tests pass" and the locked-copy assertions are well covered, but **no test exercises the banner's `Enabling…` transient state** because the production code unmounts the banner before the state can be observed — surfacing CR-01.

## Critical Issues

### CR-01: Banner unmounts synchronously before confirm RPCs resolve — `Enabling…` state and rollback unreachable

**File:** `frontend/src/App.tsx:766-784`
**Issue:** `handleShellWebShareConfirm` calls `setPendingShellWebToggle(null)` *synchronously* before `await Promise.all([SetShellWebShareWarned(true), ToggleWebServing(sessionId, true)])`. Because `pendingShellWebToggle === null` makes the banner JSX (`App.tsx:1002-1008`) unmount immediately, the banner's internal `enabling` state and the "Enabling…" transient label (`ShellWebShareBanner.tsx:104`) are dead code — they can never appear in production. This contradicts the locked UI-SPEC copy contract (line 305: *"Confirming (button clicked) | Primary button shows `Enabling…`, both buttons disabled | aria-busy=\"true\""*) and the human-verification expectation (101-VERIFICATION.md line 29: *"Click 'Enable web sharing' — button changes to 'Enabling…', both buttons disable, banner unmounts when daemon confirms"*).

Worse, on RPC failure the rollback path (`App.tsx:782 setShellWebShareWarned(false)`) executes after the banner is already gone, so the user gets no visible error feedback — they see the toggle silently fail and the security flag silently revert. If `ToggleWebServing` succeeded but `SetShellWebShareWarned` failed (or vice versa via `Promise.all` short-circuit on first rejection), local state is now inconsistent with the daemon: `webEnabled[sessionId]` was never set (it's only updated after `await Promise.all` returns successfully), while the daemon may have actually enabled web serving. The user sees "Web On" still ghosted off in the UI but the session IS being served.

**Fix:** Defer the banner dismissal until after the RPCs complete (or fail), and gate the buttons via an `aria-busy` prop instead of unmount. Two viable patterns:

```typescript
const handleShellWebShareConfirm = useCallback(async () => {
  if (!pendingShellWebToggle) return
  const { sessionId } = pendingShellWebToggle
  // Race mitigation per RESEARCH §8: flip in-memory flag synchronously
  // BEFORE awaiting so a concurrent second-shell toggle observes warned=true.
  setShellWebShareWarned(true)
  // Mark the banner as "enabling" — but DO NOT unmount it yet; the banner's
  // own `aria-busy` state is meaningful UI affordance per UI-SPEC.
  setPendingShellWebToggle((prev) => prev ? { ...prev, enabling: true } : null)
  try {
    await Promise.all([
      SetShellWebShareWarned(true),
      ToggleWebServing(sessionId, true),
    ])
    setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
    setPendingShellWebToggle(null) // unmount on success
  } catch (err) {
    console.warn('[App] shell web-share confirm failed:', err)
    // Surface the error to the user — silently rolling back without
    // feedback leaves them confused about whether the toggle worked.
    setShellWebShareWarned(false)
    setPendingShellWebToggle(null)
    // Optionally: push an error banner via saveBanner pattern.
  }
}, [pendingShellWebToggle])
```

And update `ShellWebShareBanner` to take an `enabling` prop instead of owning the state locally, OR keep the local state but ensure the parent only unmounts after RPC completion. Then add a vitest that asserts "Enabling…" actually renders during the confirm flow (currently absent — the existing tests only verify the static string is in the file).

---

### CR-02: Banner Esc keydown listener has no focus or active-modal gate — can dismiss banner when user meant to dismiss something else

**File:** `frontend/src/components/ShellWebShareBanner.tsx:55-65`
**Issue:** The Esc handler is attached to `document` and fires unconditionally on every keydown anywhere in the app. AgentHub already has multiple Esc-consumers layered above the banner stack:

- `TabBar.tsx:122-129` — context menu Esc dismiss
- `QuitConfirmModal` — Esc dismiss
- `NewSessionModal` — Esc dismiss (`onClose`)
- TerminalPanel find-bar — Esc dismiss
- Other modals registered for Esc

If any of these modals are open *above* the banner-stack (the banner is rendered persistently at the top of the layout, the modals overlay), pressing Esc fires both handlers. The banner's `onCancel` clears `pendingShellWebToggle`, silently dismissing a one-time security gate that the user did not intend to cancel — they meant to dismiss the focused modal.

Reproduction: open the security banner by clicking Web On on a shell session, then open the new-session modal (Sidebar +). Press Esc. Both the modal closes AND the banner is cancelled. The user now thinks they've dismissed only the modal but the banner is gone too, and the next toggle attempt will surface it again — making the toggle appear "stuck off" until they realize what happened.

Additional problem: the listener is registered at component mount and removed at unmount. Since the banner mounts on intercept and unmounts on confirm/cancel, the registration window is bounded — but during that window the global listener fires for any Esc on the page.

**Fix:** Gate the Esc handler. Either (1) check `event.target` against a focus container before firing, (2) only register the listener when no other modal is open (requires a context or App-level coordinator), or (3) simplest — move focus into the banner on mount (the code does this via `cancelRef.current?.focus()`) and only handle Esc when the activeElement is inside the banner:

```typescript
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent) {
    if (e.key !== 'Escape') return
    // Only dismiss if focus is inside the banner — otherwise the user is
    // interacting with a layered modal/popover and meant to dismiss that.
    const bannerEl = cancelRef.current?.closest('.webgl-recovery-banner')
    if (!bannerEl || !bannerEl.contains(document.activeElement)) return
    e.preventDefault()
    onCancel()
  }
  document.addEventListener('keydown', handleKeyDown)
  return () => document.removeEventListener('keydown', handleKeyDown)
}, [onCancel])
```

Add a vitest that opens the banner, programmatically blurs the cancel button (or focuses an external element), dispatches Esc, and asserts `onCancel` is NOT called.

---

## Warnings

### WR-01: Canonical "is shell" set duplicated across six call-sites with no shared source of truth

**File:** multiple
**Issue:** The set `{shell, bash, zsh, pwsh, powershell}` (or with PowerShell-like extensions) appears in at least six places:

1. `internal/daemon/engine.go:115-121` — `isShellSession(cli string) bool` switch
2. `internal/daemon/engine.go:105-110` — `knownShells` map (different set — includes sh/fish/csh/tcsh/dash/ksh + .exe forms)
3. `cmd_cli.go:111-117` — `allowed` map in `cmdNewShell` (no "shell" — it's the empty-string sentinel)
4. `frontend/src/App.tsx:71` — `SHELL_CLIS` Set
5. `frontend/src/components/TabBar.tsx:27-32` — `agentBadgeModifier` switch
6. `internal/tui/styles.go:111` — `agentBadgeColor` switch
7. `internal/tui/update.go:692-698` — `isShellCLI(cli string) bool` switch
8. `frontend/src/components/NewSessionModal.tsx:8` — `SHELL_PREFIX` namespace for arg memory

The App.tsx code comment (lines 65-70) acknowledges this as "acceptable v3.3 duplication" and defers extraction. Two of these sets are subtly different (`knownShells` is a superset for stale-override detection; `allowed` lacks `"shell"`). Drift is already visible: if a future shell variant is added, six files must be updated atomically and CI has no guard. The Phase 100 RESEARCH and Phase 101 RESEARCH both highlight this duplication as a maintenance risk.

**Fix:** Introduce a single canonical declaration. Two viable approaches:

- **Go side:** export `daemon.IsShellSession(cli string) bool` and `daemon.KnownShellNames() []string` from `internal/daemon`, have `cmd_cli.go` and `internal/tui` import them.
- **Frontend side:** export `SHELL_CLIS` from a single module (`frontend/src/lib/shellCLIs.ts`) and import from App, TabBar, and any future call-site.
- **Cross-language:** mark the Go side as canonical and run a CI grep that asserts the frontend constants match. Cheap and catches drift.

At minimum, add a comment in each of the 6+ duplicate locations referencing the others so a future code reviewer notices the dependency.

---

### WR-02: `SetShellWebShareWarned` engine method swallows disk-write failure

**File:** `internal/daemon/engine.go:593-599`, `internal/daemon/engine.go:186-200`
**Issue:** `SetShellWebShareWarned` returns `error` but `saveSettingsToDisk` (the only thing that could fail) does:

```go
_ = os.WriteFile(settingsPath(e.configDir), data, 0600)
```

The error is intentionally swallowed (a separate WARNING since the pattern is repeated for all settings). `SetShellWebShareWarned` is documented as "always returns nil today." Net effect: a failed disk write (disk full, permissions, FS read-only) is invisible to the user. They confirm the banner, the in-memory flag flips to true, but on next daemon restart the banner re-appears — but the user has no idea why.

Worse for SHELL-08 specifically: this is the only persistence path for a security acknowledgement. If it silently fails, the user has no way to debug why their "I accept the risk" gesture isn't sticking.

**Fix:** At minimum, log the failure with enough context to debug:

```go
func (e *SessionEngine) saveSettingsToDisk() {
    s := daemonSettings{ /* ... */ }
    data, err := json.Marshal(s)
    if err != nil {
        log.Printf("daemon: settings marshal failed: %v", err)
        return
    }
    if err := os.WriteFile(settingsPath(e.configDir), data, 0600); err != nil {
        log.Printf("daemon: settings write to %s failed: %v", settingsPath(e.configDir), err)
    }
}
```

And consider plumbing the error back through `SetShellWebShareWarned`'s declared return type rather than always returning nil — the existing return signature already accommodates it.

---

### WR-03: `handleShellWebShareConfirm` partial-failure path leaves `webEnabled` and `shellWebShareWarned` out of sync with daemon

**File:** `frontend/src/App.tsx:766-784`
**Issue:** Related to CR-01 but distinct. `Promise.all` rejects on the first rejection but does not cancel the other call. If `SetShellWebShareWarned` succeeds (warned=true persisted on disk) but `ToggleWebServing` fails (or the inverse), the catch block runs `setShellWebShareWarned(false)` — which now contradicts the daemon's persisted state. The next time the user toggles, the banner re-appears (correct from local state's POV), but the daemon still has warned=true so the daemon-side state doesn't reset.

Conversely, if `ToggleWebServing` succeeded and `SetShellWebShareWarned` failed, the session IS web-served on the daemon, but `setWebEnabled` was never called (it's after the `await`), so the UI shows it OFF. The user sees a discrepancy.

**Fix:** Drive both calls separately and reconcile per-result:

```typescript
try {
  await SetShellWebShareWarned(true)
} catch (err) {
  console.warn('[App] persist shellWebShareWarned failed:', err)
  setShellWebShareWarned(false) // re-show banner next time
}
try {
  await ToggleWebServing(sessionId, true)
  setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
} catch (err) {
  console.warn('[App] toggle web on shell failed:', err)
  // Show error banner — the toggle didn't take.
}
setPendingShellWebToggle(null)
```

Or, more aggressively, refuse to clear the banner until both calls confirm; show an error state in the banner when one fails. The current "best-effort rollback" comment in code (line 780) understates the inconsistency risk.

---

### WR-04: NewSessionModal `creating` state is never reset → "Create Session" button stuck on error

**File:** `frontend/src/components/NewSessionModal.tsx:51, 105-121, 224-230`
**Issue:** `const [creating, setCreating] = useState(false)` is set to true in `handleConfirm` but **never reset to false anywhere**. The button label flips to "Creating…" and `disabled={!selectedAgent || creating}` locks it. The modal does not own its own dismissal — the parent (`App.tsx`) closes it via `setShowNewSessionModal(false)` after `CreateSession` resolves. If `CreateSession` rejects (line `App.tsx:663 console.error('[App] CreateSession failed:', err)`), the modal stays open and the button is permanently stuck "Creating…" until the user clicks "Close" and re-opens.

This is pre-existing — the bug pre-dates Phase 101 — but Phase 101 propagates it: the shell branch also calls `setCreating(true)` and returns immediately (line 112), and the daemon-unreachable error path for shells is just as common as for AI CLIs.

**Fix:** Reset `creating` in a `useEffect` keyed on `isOpen`, OR have the parent pass a callback that fires onError so the child can reset:

```typescript
useEffect(() => {
  if (!isOpen) setCreating(false)
}, [isOpen])
```

Or, more robustly, accept `creating` as a prop driven by the parent so it reflects actual in-flight RPC state.

---

### WR-05: `shellWebShareWarned` hydration race window — banner may show despite daemon already having warned=true

**File:** `frontend/src/App.tsx:100, 403-408, 745-751`
**Issue:** The local `shellWebShareWarned` state defaults to `false` (line 100). Hydration via `GetShellWebShareWarned()` is awaited *outside* the initial render path (it's a non-blocking `.then()` in the mount `useEffect`). Between mount and resolution, any shell Web On toggle the user clicks will see `shellWebShareWarned === false` and show the banner — even if the daemon already has it persisted as true from a previous session.

The behavior is documented in RESEARCH §8 as "fail-safe" (over-prompt > under-prompt), and the on-error path also sets false. But the spec explicitly calls out *"race between shellWebShareWarned hydration and first toggle"* as a focus area, and the current implementation accepts the over-prompt. That's a UX defect, not a security defect, but worth flagging because the verification report claims this is mitigated and it is not — it is accepted.

**Fix:** Two options:

1. Block the shell-toggle handler until hydration completes — add a `shellWebShareLoaded: boolean` state, gate the interception on `!shellWebShareLoaded || !shellWebShareWarned`. When `shellWebShareLoaded` is false, instead of pushing the banner, defer the toggle (or show a brief "Loading…" state). Simpler implementation: hydrate `shellWebShareWarned` synchronously in the initial `Promise.all` block (line 345-351) so by the time the user can interact, it's settled.

2. Document the over-prompt as accepted in code (currently only in RESEARCH.md). Add an inline comment at App.tsx:100 noting "defaults to false; hydration is racy but over-prompting is the safe direction."

Option 1 is strictly better.

---

### WR-06: `ShellWebShareBanner` body uses straight ASCII apostrophes around `sessionName` — XSS-style breakage if `sessionName` contains `'`

**File:** `frontend/src/components/ShellWebShareBanner.tsx:85`
**Issue:** The body renders `"You are about to share '{sessionName}'."` — `sessionName` is passed directly as a React child (good — auto-escaped — no XSS), but the visual rendering breaks awkwardly when the user has named a session with an apostrophe (e.g., a tab renamed to `Ken's Bash`). The output reads `share 'Ken's Bash'` which is ambiguous and ugly. AgentHub already supports rename to arbitrary strings via the tab context menu.

Not a security issue (React escapes the text node), but a UX one. The locked UI-SPEC copy specifies the single-quote wrapping, so this needs a UI-SPEC amendment to fix properly. But at minimum the implementation should be aware that user-controlled `sessionName` is going into a quoted slot.

**Fix:** Either:

1. UI-SPEC amendment: change quote style to typographic quotes (U+2018/U+2019) or replace `'...'` with `"..."` or omit quotes entirely.
2. At the App.tsx level, sanitize `sessionName` to strip single quotes before passing as banner prop.

Option 1 is the right call for the locked-copy contract; flag for a planner amendment.

---

### WR-07: `cmdNewShell` does not pass `extraArgs` allow/error-out — fails open with stderr-only warning

**File:** `cmd_cli.go:132-134`
**Issue:** Per UI-SPEC and Phase 100 RESEARCH A6, args are **not** forwarded to shells. The implementation correctly drops them and emits a stderr warning:

```go
if len(extraArgs) > 0 {
    fmt.Fprintf(os.Stderr, "agenthub new shell: extra arguments are not forwarded to shell sessions; ignoring %v\n", extraArgs)
}
```

This is "fail-open" — if a user scripts `agenthub new shell ~/dir -- --some-flag`, the session is created and the args are silently ignored. This is the documented design (UI-SPEC line 329 *"emit a warning to stderr"*). But the warning is on stderr and easy to miss in scripts that redirect stderr. A user scripting `agenthub new shell ... -- --rm-rf-something` might assume the destructive flag is being honored. Since the shell session is interactive, the user will quickly see no command runs — but the failure mode is not loud enough for a security-adjacent surface.

**Fix:** Consider exit-on-extra-args instead of warn-and-continue. Per UI-SPEC this is a design call ("forgiving for users who muscle-memory the AI CLI pattern"), so leave the warning but make it more emphatic:

```go
fmt.Fprintf(os.Stderr, "agenthub new shell: WARNING: extra arguments are not forwarded to shell sessions. The following are being IGNORED: %v\n", extraArgs)
fmt.Fprintf(os.Stderr, "agenthub new shell: If you intended to pass arguments, this session will NOT receive them.\n")
```

Or — preferred — add a `--force` flag that suppresses the warning, and absent that flag exit-1 with the warning. The current implementation accepts the foot-gun.

---

### WR-08: `internal/daemon/engine.go` defensive copy of `spec.Argv` inside loop creates per-call allocation

**File:** `internal/daemon/engine.go:505, 522-523, 533-534`
**Issue:** Three places in `resolveShellSpawn` do `argv := append([]string(nil), spec.Argv...)` to defensively copy the slice. Correct for safety (caller could mutate, races with concurrent CreateSession calls). But each call allocates a fresh slice on the heap. Not a bug per se, but the comment chain doesn't acknowledge this is a defense against `pty.knownShellSpecs` being a package-level variable — a quick test could mutate that variable in one test and pollute another. Currently the package-level variable is unexported and stable; the defensive copy is belt-and-suspenders.

**Fix:** Leave as-is (correctness wins over micro-allocation). Add a one-line comment explaining *why* the copy is necessary so a future reader doesn't strip it as redundant:

```go
// Defensive copy: spec.Argv is shared with pty.knownShellSpecs and
// callers downstream (backend.Create) may mutate the slice.
argv := append([]string(nil), spec.Argv...)
```

(This is INFO-leaning but flagged here because the threat model comment in `handleListShells` mentions T-100-09 about the same pattern — consistency in documenting the rationale would help.)

---

### WR-09: TUI `submitNewSession` requires non-empty workDir, diverging from CLI/GUI shell semantics

**File:** `internal/tui/update.go:662-668`
**Issue:** TUI rejects empty workDir with toast `"Directory is required"`. But the CLI `cmdNewShell` accepts empty workDir (line 127-131) and the daemon resolves it to `$HOME` (engine.go:245-259). The GUI's `NewSessionModal` accepts empty `selectedDir` (it's stored as `""` in state) and passes it through to `CreateSession`. So all three surfaces have different semantics:

- CLI shell: empty → `$HOME`
- GUI shell: empty → `$HOME`
- TUI shell: empty → rejected with toast

This is a UX inconsistency at the contract boundary. The TUI used to require workDir for AI CLI sessions (where $HOME doesn't make sense — you want a project dir), and the Phase 101 work added shell support without relaxing this for shells specifically.

**Fix:** Either:

1. Relax `submitNewSession` to allow empty workDir for shell entries:
   ```go
   if workDir == "" && !isShellCLI(cli) {
       m.toast = "Directory is required"
       // ...
       return m, nil
   }
   ```
2. Pre-fill the TUI dirInput with `$HOME` when the agent picker lands on a shell entry (and the dir is empty) — keeps the rejection logic simple but ensures the field is never empty by the time the user submits.

Option 1 is the minimum fix and aligns the three surfaces.

---

## Info

### IN-01: `handleClearArgs` in NewSessionModal silently no-ops for shell sessions

**File:** `frontend/src/components/NewSessionModal.tsx:98-103`
**Issue:** When a shell is selected, the "Clear Args" button is hidden anyway (line 210 gates `argsText && !isShellSelected`), but the handler defensively no-ops on `isShellSelected`. The handler is unreachable for shells in production. Dead branch.

**Fix:** Either remove the dead branch or document why the defensive check exists (e.g., "future-proof against argsText being prefilled from a localStorage entry that bypasses the disable").

---

### IN-02: `handleShellWebShareCancel` doesn't include the `enabling` aria-busy reset path

**File:** `frontend/src/App.tsx:786-788`
**Issue:** Related to CR-01. If CR-01 is fixed by deferring banner unmount, the cancel path also needs to clear any `enabling` state. Today the cancel handler just clears `pendingShellWebToggle` — if `enabling=true` were ever observable, the cancel path would race the in-flight RPC. Currently this is a moot point because the RPC isn't initiated until confirm is clicked, but flagging for the CR-01 fix to consider.

**Fix:** When CR-01 is addressed, ensure cancel can only fire when `enabling === false`. The banner already disables both Cancel and × when `enabling`, so this is structurally enforced — just confirm the state machine remains consistent.

---

### IN-03: Empty interceptBlock window in App.shellWebShare.test.tsx (600 chars) is fragile

**File:** `frontend/src/components/__tests__/App.shellWebShare.test.tsx:69-71`
**Issue:** The test slices 600 characters after `SHELL_CLIS.has` to find `setPendingShellWebToggle`. If a future refactor moves the handler body or adds comments, the 600-char window may exceed the function body. A drift detector that fails for the right reason (semantics) but a fragile measurement (string slicing).

**Fix:** Switch to a proper AST-based assertion or, at minimum, slice on a sentinel like `const handleToggleWeb = useCallback` and stop at the next `useCallback`.

---

### IN-04: `cmdNewShell` session name fallback to `cli` is misleading when daemon resolves to $HOME

**File:** `cmd_cli.go:137-140`
**Issue:** When `workDir == ""`, the CLI sets `name = cli` (e.g., `"shell"`, `"bash"`). The session then appears in `agenthub list` with NAME=`bash`, but the actual working directory is `$HOME`. A user might expect the session name to reflect the directory it's working in (e.g., `Users` for `/Users/ken`). The CLI doesn't know `$HOME` at this layer (per the inline comment), but the daemon does. The result: TWO sources of truth for "session name" — the CLI's bare cli string and the daemon's $HOME basename.

**Fix:** Have the daemon return a useful default session name when the caller passes an empty name AND empty workDir for shells. Or have the CLI call `os.UserHomeDir()` to compute the basename locally. Minor UX polish.

---

### IN-05: Duplicate banner-priority predicate in App.tsx render guard

**File:** `frontend/src/App.tsx:992-997`
**Issue:** The "should we render the banner-stack div?" predicate ORs together 6 conditions: `pendingShellWebToggle || (webServerMode === 'local' && !localBannerDismissed) || update || ((webglContextLost || webglSoftwareDetected) && !webglBannerDismissed) || saveBanner !== null || pluginToggleBanners.length > 0`. Each banner inside the stack has its OWN render condition that re-evaluates the same predicate. If a new banner is added in v3.4, the outer guard must be updated in two places.

**Fix:** Extract the predicate into a named variable:

```typescript
const showBannerStack =
  pendingShellWebToggle !== null ||
  (webServerMode === 'local' && !localBannerDismissed) ||
  update !== null ||
  ((webglContextLost || webglSoftwareDetected) && !webglBannerDismissed) ||
  saveBanner !== null ||
  pluginToggleBanners.length > 0

// ...

{showBannerStack && (
  <div className="banner-stack">
    {/* ... */}
  </div>
)}
```

Cosmetic but prevents drift when adding the 7th banner.

---

### IN-06: `frontend/src/test-setup.ts` `localStorage` polyfill registers via `Object.defineProperty` with `configurable: true` but no removal in cleanup

**File:** `frontend/src/test-setup.ts:36-39`
**Issue:** The polyfill defines `globalThis.localStorage` once at test-setup time. If a future test wants to mock `localStorage` with a per-test spy via `vi.spyOn(window.localStorage, 'setItem')`, the `configurable: true` flag is correct. But the polyfill never resets state between test files — the `store` Map is shared across the entire vitest run. Tests that pollute localStorage in one file may leak to the next.

**Fix:** Add a `beforeEach` in `test-setup.ts` or document that tests must clear their own `localStorage` keys in their `afterEach`. Low-priority because the existing test files don't share keys, but a future test author may not know the contract.

```typescript
beforeEach(() => {
  if (target === fallback) store.clear()
})
```

---

_Reviewed: 2026-05-12_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
