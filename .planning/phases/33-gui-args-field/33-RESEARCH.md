# Phase 33: GUI Args Field - Research

**Researched:** 2026-03-26
**Domain:** React frontend — modal form field, localStorage persistence, Wails TypeScript binding update, args string splitting
**Confidence:** HIGH

## Summary

Phase 30 completed all Go backend wiring: `App.CreateSession` already accepts `args []string` (confirmed in `app.go:123`). The Wails TypeScript binding stub (`App.js`) still calls `CreateSession` with only three arguments `(cli, name, workDir)` — the fourth `args` parameter is absent. The frontend `NewSessionModal` component has no args text field, and `App.tsx`'s `createTab` function and `onConfirm` callback do not carry args.

Phase 33 is a pure frontend + TS binding update. There is no Go code to change.

The work decomposes into four additive changes:
1. Update `App.js` and `App.d.ts` Wails stubs to pass `args` as the fourth argument.
2. Add an args text field and clear button to `NewSessionModal` with localStorage persistence keyed by agent name.
3. Thread args from modal `onConfirm` through `App.tsx`'s `createTab` to `CreateSession`.
4. Extend `NewSessionModal.test.tsx` with source-inspection tests for the new field.

The arg string entered by the user is a simple space-delimited string split with `.trim().split(/\s+/).filter(Boolean)` — this covers the common CLI flag pattern. Shell-quoting (ARGS-06) is explicitly deferred and out of scope.

**Primary recommendation:** Work from the call-site inward: update Wails bindings first (so TypeScript types are correct), then update `NewSessionModal` (field + persistence), then thread args through `App.tsx`, then add tests.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ARGS-02 | User can enter extra arguments in the GUI new-session modal text field | `NewSessionModal` needs an `<input>` field added below the folder picker section; CSS pattern is established |
| ARGS-04 | Per-agent argument memory: last-used args pre-filled in GUI modal | localStorage key `agenthub:args:{cliName}` read on mount, written on confirm; same pattern as `agenthub:lastWorkDir` |
| ARGS-05 | User can clear or edit pre-filled args before session creation | Clear button adjacent to args field resets state and removes localStorage key; edit is native (input is editable) |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React (useState, useCallback) | 19.2.4 | Component state for args text field | Already used throughout; `useState` for argsText |
| localStorage | browser built-in | Per-agent args persistence | Same mechanism used for `agenthub:lastWorkDir` in this codebase |
| Vitest | 4.1.0 | Source-inspection test pattern | Established test pattern (`?raw` imports) used in all frontend tests |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| TypeScript | 5.9.3 | Type-safe binding stubs | `App.d.ts` needs `args?: string[]` parameter added |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `localStorage` for per-agent args | In-memory state in `App.tsx` | Memory state is lost on window re-open; localStorage matches the existing LAST_DIR_KEY pattern |
| Simple whitespace split | `shlex` parsing library | ARGS-06 is explicitly deferred; simple split is correct scope |
| React `useEffect` for localStorage read | Lazy initializer in `useState` | Lazy initializer is already used for LAST_DIR_KEY — use the same pattern |

**Installation:** No new packages required.

## Architecture Patterns

### Full Call Chain (current vs. target)

```
Current:
  NewSessionModal.onConfirm(cli, workDir)
    → App.tsx createTab(cliName, workDir)
        → CreateSession(cliName, name, workDir)   // App.js: 3-arg call, drops 4th param

Target:
  NewSessionModal.onConfirm(cli, workDir, args)
    → App.tsx createTab(cliName, workDir, args)
        → CreateSession(cliName, name, workDir, args)  // App.js: 4-arg call
```

### Pattern 1: Per-Agent Args Persistence

localStorage key: `agenthub:args:{cliName}` (cliName is the raw Name string, e.g. "claude", "gemini").

```typescript
// Read on agent selection change (or lazy init for default agent):
const ARGS_KEY = (cli: string) => `agenthub:args:${cli}`

// Initialize state from localStorage when selected agent changes:
const [argsText, setArgsText] = useState(() =>
  localStorage.getItem(ARGS_KEY(clis[0]?.Name ?? '')) ?? ''
)

// When agent changes, update argsText from localStorage:
function handleSelectCLI(name: string) {
  setSelectedCLI(name)
  setArgsText(localStorage.getItem(ARGS_KEY(name)) ?? '')
}

// On confirm, persist:
localStorage.setItem(ARGS_KEY(selectedCLI), argsText)
// But only persist non-empty args; if empty, remove:
if (argsText.trim()) {
  localStorage.setItem(ARGS_KEY(selectedCLI), argsText)
} else {
  localStorage.removeItem(ARGS_KEY(selectedCLI))
}

// Parse to string[]:
const args = argsText.trim().split(/\s+/).filter(Boolean)
// Pass empty array (not null) when no args:
onConfirm(selectedCLI, selectedDir, args)
```

### Pattern 2: Clear Button

The clear button should reset `argsText` to `''` AND remove the localStorage key so the pre-fill does not reappear next time:

```typescript
function handleClearArgs() {
  setArgsText('')
  localStorage.removeItem(ARGS_KEY(selectedCLI))
}
```

### Pattern 3: Wails Binding Stub Update

`App.js` — update CreateSession call:
```javascript
// Before:
export const CreateSession = (cli, name, workDir) =>
  Call('main.App.CreateSession', [cli, name, workDir])

// After:
export const CreateSession = (cli, name, workDir, args) =>
  Call('main.App.CreateSession', [cli, name, workDir, args])
```

`App.d.ts` — update CreateSession type:
```typescript
// Before:
export function CreateSession(cli: string, name: string, workDir: string): Promise<string>

// After:
export function CreateSession(cli: string, name: string, workDir: string, args: string[]): Promise<string>
```

### Pattern 4: App.tsx Threading

```typescript
// createTab: add args parameter
const createTab = useCallback(async (cliName: string, workDir: string, args: string[]) => {
  ...
  const sessionId = await CreateSession(cliName, defaultName, workDir, args)
  ...
}, [tabCounter])

// NewSessionModal onConfirm: add args
onConfirm={(cli, workDir, args) => {
  setShowNewSessionModal(false)
  void createTab(cli, workDir, args)
}}
```

### Pattern 5: NewSessionModal Props Update

```typescript
export interface NewSessionModalProps {
  isOpen: boolean
  clis: DetectedCLI[]
  onConfirm: (cli: string, workDir: string, args: string[]) => void  // added args
  onClose: () => void
}
```

### CSS Pattern: Args Input Field

The new section follows the existing section pattern. Add below the Working Directory section:

```tsx
<div className="new-session-modal__section">
  <label className="new-session-modal__section-label">Extra Arguments</label>
  <div className="new-session-modal__args-row">
    <input
      className="new-session-modal__args-input"
      type="text"
      value={argsText}
      onChange={(e) => setArgsText(e.target.value)}
      placeholder="e.g. --model claude-opus-4-5"
    />
    {argsText && (
      <button
        className="new-session-modal__args-clear"
        onClick={handleClearArgs}
        aria-label="Clear arguments"
      >
        Clear
      </button>
    )}
  </div>
</div>
```

CSS classes to add in `style.css` (following `.new-session-modal__folder-row` pattern):
```css
.new-session-modal__args-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.new-session-modal__args-input {
  flex: 1;
  background: #16161e;
  border: 1px solid #292e42;
  border-radius: 4px;
  color: #c0caf5;
  font-size: 13px;
  padding: 6px 12px;
  outline: none;
  font-family: inherit;
  min-height: 32px;
}
.new-session-modal__args-input:focus {
  border-color: #7aa2f7;
}
.new-session-modal__args-input::placeholder {
  color: #414868;
}
.new-session-modal__args-clear {
  padding: 4px 8px;
  border: 1px solid #292e42;
  border-radius: 4px;
  font-size: 11px;
  background: transparent;
  color: #a9b1d6;
  cursor: pointer;
  white-space: nowrap;
  font-family: inherit;
}
.new-session-modal__args-clear:hover {
  background: #1e2030;
  color: #c0caf5;
}
```

### Anti-Patterns to Avoid

- **Passing `null` instead of `[]` to CreateSession:** `App.CreateSession` in Go accepts `[]string`. The Wails JSON bridge serializes JavaScript `null` as JSON `null` — Go's `encoding/json` decodes that as `nil` slice, which is fine. However, passing `[]` (empty array) is more explicit. Use `.trim().split(/\s+/).filter(Boolean)` which returns `[]` when the input is blank.
- **Storing args in a single global localStorage key:** Per-agent keying (`agenthub:args:{cliName}`) is required by ARGS-04. A single key would overwrite args when switching agents.
- **Splitting in Go vs. TypeScript:** ARGS-06 (shell quoting) is deferred. For Phase 33, split in TypeScript with `split(/\s+/)` — keep it simple, the Go backend receives the already-split `string[]`.
- **Removing the Clear button when argsText is empty:** The button should only render when `argsText` is non-empty — conditionally render with `{argsText && <button...>}`. If always visible, it adds unnecessary UI noise.
- **Not updating argsText when the selected agent changes:** When the user switches from one agent to another in the modal, the args field should reflect the stored value for the new agent. If `handleSelectCLI` only updates `selectedCLI`, the field will show stale args.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Args string parsing | Custom tokenizer | `.trim().split(/\s+/).filter(Boolean)` | Sufficient for Phase 33; ARGS-06 handles edge cases later |
| Per-agent persistence | In-memory Map | `localStorage` with keyed entries | Persists across modal open/close and window hide/show |
| Type definitions for Wails bindings | Auto-generate via CLI | Hand-edit stubs | `wails generate` does not produce TypeScript bindings in Wails v2; the project already uses hand-maintained stubs (confirmed: `App.d.ts` header says "AUTO-GENERATED — DO NOT edit manually" but the project edits them manually; this is the established pattern) |

**Key insight:** Wails v2 (`wails dev`) auto-regenerates `App.js` and `App.d.ts` from the Go source during development. However, the project uses manually-maintained stubs (the `wailsjs` directory is checked into git with hand-edited content). The Wails `generate` subcommand only offers `module` and `template` — not TypeScript bindings. The correct approach for this project is to hand-edit both `App.js` and `App.d.ts`, mirroring the Go method signature. The STATE.md concern ("confirm whether wails dev auto-regenerates TypeScript bindings on Go method signature change") is resolved: the Go signature for `App.CreateSession` already changed in Phase 30; the stubs were NOT auto-updated, confirming they must be updated manually.

## Common Pitfalls

### Pitfall 1: Wails bindings not updated — silently drops args
**What goes wrong:** `App.js` still calls `Call('main.App.CreateSession', [cli, name, workDir])` without the 4th arg. The Go method receives `nil` for args silently — no error, no console warning.
**Why it happens:** The stub file says "DO NOT edit manually" which creates confusion; in practice, this project edits stubs manually.
**How to avoid:** Update `App.js` AND `App.d.ts` as the first task. The TypeScript type error in `App.tsx` (calling 3-arg function with 4 args) will surface during `pnpm run build` once `App.d.ts` is correct.
**Warning signs:** `CreateSession` TypeScript call compiles with 4 args but `App.d.ts` still shows 3-parameter signature.

### Pitfall 2: args persist across agents — wrong localStorage key
**What goes wrong:** Using a single `agenthub:lastArgs` key stores args from one agent (e.g., claude) and pre-fills them when the user switches to a different agent (e.g., gemini), which likely has incompatible flags.
**Why it happens:** Easy to implement as a single key, mirroring the LAST_DIR_KEY pattern directly.
**How to avoid:** Key by agent: `agenthub:args:${selectedCLI}`. Load from this key on agent selection change.
**Warning signs:** Switching from claude to gemini pre-fills claude-specific flags in the gemini args field.

### Pitfall 3: argsText not reset when agent changes in modal
**What goes wrong:** User selects agent A (pre-filled with agent A's args), then selects agent B — argsText still shows agent A's args because `handleSelectCLI` only updates `selectedCLI`, not `argsText`.
**Why it happens:** `selectedDir` does not have this problem (it is not per-agent), so there is no precedent in the existing code to remind the implementer.
**How to avoid:** `handleSelectCLI` must call both `setSelectedCLI(name)` AND `setArgsText(localStorage.getItem(ARGS_KEY(name)) ?? '')`.
**Warning signs:** Switching agents in the modal does not update the args field.

### Pitfall 4: Empty string causes `['']` instead of `[]`
**What goes wrong:** `''.trim().split(/\s+/)` returns `['']` in JavaScript (not `[]`). Passing `['']` to `App.CreateSession` would forward a single empty-string argument to the CLI process.
**Why it happens:** JavaScript's `String.prototype.split` always returns at least one element.
**How to avoid:** Chain `.filter(Boolean)`: `argsText.trim().split(/\s+/).filter(Boolean)`. This converts blank input to `[]`.
**Warning signs:** A session created with empty args field results in the CLI receiving an empty string argument.

### Pitfall 5: Clear button always visible creates UX noise
**What goes wrong:** If the Clear button is always rendered, it appears even when argsText is empty — clicking it does nothing meaningful.
**Why it happens:** Simpler to always render.
**How to avoid:** Conditionally render: `{argsText && <button className="new-session-modal__args-clear" ...>Clear</button>}`.
**Warning signs:** The args row always shows a "Clear" button even when the field is blank.

## Code Examples

Verified patterns from reading source files:

### localStorage lazy initializer pattern (from NewSessionModal.tsx)
```typescript
// Source: frontend/src/components/NewSessionModal.tsx:21
const [selectedDir, setSelectedDir] = useState(() => localStorage.getItem(LAST_DIR_KEY) ?? '')
```

### localStorage persist pattern (from NewSessionModal.tsx)
```typescript
// Source: frontend/src/components/NewSessionModal.tsx:32-33
if (path !== '') {
  setSelectedDir(path)
  localStorage.setItem(LAST_DIR_KEY, path)
}
```

### Wails binding call pattern (from App.js)
```javascript
// Source: frontend/src/wailsjs/go/main/App.js
export const CreateSession  = (cli, name, workDir) => Call('main.App.CreateSession', [cli, name, workDir])
```

### Source-inspection test pattern (from NewSessionModal.test.tsx)
```typescript
// Source: frontend/src/components/__tests__/NewSessionModal.test.tsx:1-2
import { describe, it, expect } from 'vitest'
import raw from '../NewSessionModal.tsx?raw'
// Tests use: expect(raw).toContain('someIdentifier')
```

### onConfirm threading pattern (from App.tsx)
```typescript
// Source: frontend/src/App.tsx:386-389
onConfirm={(cli, workDir) => {
  setShowNewSessionModal(false)
  void createTab(cli, workDir)
}}
```

## Environment Availability

Step 2.6: This phase is purely frontend TypeScript/React changes plus hand-edited binding stubs. No external services or CLIs are required beyond what already runs the dev environment.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | pnpm/vitest | Yes | v20.19.3 | — |
| pnpm | test runner | Yes | (project uses pnpm) | — |
| Vitest | frontend tests | Yes | 4.1.0 | — |
| Wails CLI | build verification | Yes | v2.10.2 | — |

**Missing dependencies with no fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` |
| Quick run command | `cd frontend && pnpm test -- NewSessionModal` |
| Full suite command | `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ARGS-02 | Modal renders args input field | unit (source inspection) | `cd frontend && pnpm test -- NewSessionModal` | Exists — extend |
| ARGS-02 | Args input has placeholder text | unit (source inspection) | `cd frontend && pnpm test -- NewSessionModal` | Exists — extend |
| ARGS-04 | `agenthub:args:` localStorage key used | unit (source inspection) | `cd frontend && pnpm test -- NewSessionModal` | Exists — extend |
| ARGS-04 | localStorage.getItem called for args | unit (source inspection) | `cd frontend && pnpm test -- NewSessionModal` | Exists — extend |
| ARGS-04 | localStorage.setItem called for args | unit (source inspection) | `cd frontend && pnpm test -- NewSessionModal` | Exists — extend |
| ARGS-05 | Clear button present in source | unit (source inspection) | `cd frontend && pnpm test -- NewSessionModal` | Exists — extend |
| ARGS-05 | handleClearArgs or equivalent present | unit (source inspection) | `cd frontend && pnpm test -- NewSessionModal` | Exists — extend |
| ARGS-02 | App.js CreateSession passes 4 args | unit (source inspection) | `cd frontend && pnpm test -- App` | Exists — extend |
| ARGS-02 | App.d.ts CreateSession has args param | unit (source inspection) | `cd frontend && pnpm test -- App` | Exists — extend |
| ARGS-02 | App.tsx createTab passes args to CreateSession | unit (source inspection) | `cd frontend && pnpm test -- App` | Exists — extend |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test -- NewSessionModal`
- **Per wave merge:** `cd frontend && pnpm test`
- **Phase gate:** `cd frontend && pnpm test` green before `/gsd:verify-work`

### Wave 0 Gaps
None — existing test files cover all phase requirements. Add new `it(...)` blocks to:
- `frontend/src/components/__tests__/NewSessionModal.test.tsx` — add ARGS-02/04/05 coverage
- `frontend/src/components/__tests__/App.test.tsx` — add ARGS-02 binding coverage

## Sources

### Primary (HIGH confidence)
- `/Users/ken/dev/agenthub/app.go:123` — `App.CreateSession` already accepts `args []string` (Phase 30 complete)
- `/Users/ken/dev/agenthub/frontend/src/wailsjs/go/main/App.js` — `CreateSession` still 3-arg call, confirmed gap
- `/Users/ken/dev/agenthub/frontend/src/wailsjs/go/main/App.d.ts:17` — TypeScript declaration still 3-param, confirmed gap
- `/Users/ken/dev/agenthub/frontend/src/components/NewSessionModal.tsx` — no args field, confirmed gap; localStorage pattern confirmed
- `/Users/ken/dev/agenthub/frontend/src/App.tsx:144,386-389` — `createTab` and `onConfirm` do not carry args, confirmed gaps
- `/Users/ken/dev/agenthub/frontend/src/style.css` — existing `.new-session-modal__folder-row` CSS pattern
- `/Users/ken/dev/agenthub/.planning/STATE.md` — "confirm whether wails dev auto-regenerates TypeScript bindings" — resolved: Go signature changed in Phase 30 but stubs were not updated, confirming manual edit required

### Secondary (MEDIUM confidence)
- Wails v2.10.2 `wails --help` output — confirmed `wails generate` has no TypeScript binding subcommand
- `/Users/ken/dev/agenthub/wails.json` — `"wailsjsdir": "./frontend/src/wailsjs"` confirms where generated/maintained stubs live

## Metadata

**Confidence breakdown:**
- Gap identification: HIGH — read every file in the call chain directly
- Fix approach: HIGH — additive changes only, no existing logic altered
- localStorage keying strategy: HIGH — mirrors existing `agenthub:lastWorkDir` pattern exactly
- Wails binding approach: HIGH — confirmed by observing Phase 30 did not auto-update stubs
- Test strategy: HIGH — source-inspection pattern is proven in all existing frontend tests

**Research date:** 2026-03-26
**Valid until:** 2026-06-26 (stable — pure React/TypeScript patterns, no third-party library decisions)
