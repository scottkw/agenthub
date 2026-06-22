# Phase 143: Regression Test Program - Pattern Map

**Mapped:** 2026-06-21
**Files analyzed:** 6 new/modified artifacts
**Analogs found:** 5 / 6 (1 file is documentation with no code analog)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `frontend/src/lib/hubGroupCounts.test.ts` | test | transform | `frontend/src/lib/hubStatus.test.ts` | exact |
| `frontend/src/lib/agentBadge.test.ts` | test | transform | `frontend/src/lib/hubStatus.test.ts` | exact |
| `frontend/src/components/__tests__/Sidebar.test.tsx` (extend) | test | request-response | `frontend/src/components/__tests__/Sidebar.test.tsx` (self) | self |
| `frontend/src/components/__tests__/style.hub.test.ts` (extend) | test | transform | `frontend/src/components/__tests__/style.hub.test.ts` (self) | self |
| `tests/check-traceability-paths.sh` | utility | batch | `tests/build-script.test.sh` | role-match |
| `.github/workflows/build.yml` (add step) | config | request-response | `.github/workflows/build.yml` (self — existing step pattern) | self |

---

## Pattern Assignments

### `frontend/src/lib/hubGroupCounts.test.ts` (test, transform)

**Analog:** `frontend/src/lib/hubStatus.test.ts`
**Secondary analog for multi-case enumeration:** `frontend/src/lib/hubGroups.test.ts`

**Imports pattern** (`hubStatus.test.ts` lines 1-4):
```typescript
import { describe, it, expect } from 'vitest'
import { isAttentionStatus } from './hubStatus'
import type { HubStatus } from './hubStatus'
```

The pure-function lib pattern uses only `describe`, `it`, `expect` — no mocks, no DOM, no `beforeEach` unless localStorage is involved. Import the function(s) and types directly from `./hubGroupCounts`.

**Core pattern — exhaustive single-function coverage** (`hubStatus.test.ts` lines 5-35):
```typescript
describe('isAttentionStatus', () => {
  it('returns true for waiting', () => {
    const status: HubStatus = 'waiting'
    expect(isAttentionStatus(status)).toBe(true)
  })

  it('returns false for running', () => {
    const status: HubStatus = 'running'
    expect(isAttentionStatus(status)).toBe(false)
  })
  // ... one it() per distinct input class
})
```

**Core pattern — multi-input fixture style** (`hubGroups.test.ts` lines 46-61):
```typescript
describe('createGroup', () => {
  it('appends a new group with a non-empty id, given name, and empty memberKeys', () => {
    const start: HubGroupDef[] = []
    const updated = createGroup(start, 'My Group')
    expect(updated).toHaveLength(1)
    expect(updated[0].name).toBe('My Group')
    expect(updated[0].id).toBeTruthy()
    expect(updated[0].memberKeys).toEqual([])
  })
```

**Pattern to copy for `hubGroupCounts.test.ts`:**
- One `describe` block per exported function (`computeCounts`, `computeGlobalCounts`)
- One `it()` per distinct input class (empty sessions array, all-running sessions, mixed running/idle/attention, global "All" totals)
- Assert concrete numeric values: `expect(result.running).toBe(2)` etc.
- No `beforeEach`, no localStorage, no mocks — pure function, pass data in and assert out

---

### `frontend/src/lib/agentBadge.test.ts` (test, transform)

**Analog:** `frontend/src/lib/hubStatus.test.ts` (exact — same role/data-flow: pure classification function, no DOM, no mocks)

**Imports pattern** (copy from `hubStatus.test.ts` lines 1-3, adapting names):
```typescript
import { describe, it, expect } from 'vitest'
import { agentColor, agentLabel } from './agentBadge'
```

**Core pattern — one it() per discrete input** (`hubStatus.test.ts` lines 6-35):
```typescript
describe('agentColor', () => {
  it('returns the correct hex for "claude"', () => {
    expect(agentColor('claude')).toBe('#<hex>')
  })
  // ... one it() per known CLI type
})

describe('agentLabel', () => {
  it('returns the display label for "claude"', () => {
    expect(agentLabel('claude')).toBe('Claude')
  })
})
```

**Colorblind constraint (from MEMORY.md):** GAP-02 rationale is colorblind-safe source-level verification. Assert concrete hex values (`expect(agentColor('claude')).toBe('#7aa2f7')`), mirroring how `style.hub.test.ts` pins hex constants (lines 65-66):
```typescript
it('dark theme --hub-bg has correct hex #14151b (comp dark surface)', () => {
  expect(cssRaw).toContain('--hub-bg: #14151b')
})
```

---

### `frontend/src/components/__tests__/Sidebar.test.tsx` — extend with GAP-03 (test, request-response)

**This is an extension of an existing file.** The file already contains the `renderSidebar` helper and all test infrastructure. New `it()` blocks are added inside an existing or new `describe` block.

**Existing render helper to reuse** (lines 40-61):
```typescript
function renderSidebar(overrides: Partial<Parameters<typeof Sidebar>[0]> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps = {
    onSettings: vi.fn(),
    onHome: vi.fn(),
    onOpenHub: vi.fn(),
    groupDefs: [] as HubGroupDef[],
    activeGroupId: null as string | null,
    onGroupSelect: vi.fn(),
    onCreateGroup: vi.fn(),
    onDropOnGroup: vi.fn(),
    groupCounts: {} as Record<string, { running: number; total: number; attention: number; waiting: number }>,
    globalGroupCounts: { running: 0, total: 0, attention: 0, waiting: 0 },
  }
  act(() => {
    root.render(<Sidebar {...defaultProps} {...overrides} />)
  })
  return { container, root, ...defaultProps, ...(overrides as typeof defaultProps) }
}
```

**Existing analog assertion to copy for GAP-03 rendered item count** (lines 91-96):
```typescript
// Phase 138 / NAV-02..05: exactly 3 items (Home, Hub, Settings)
it('renders exactly 3 sidebar__item buttons (Home, Hub, Settings)', () => {
  ;({ container, root } = renderSidebar())
  const items = container.querySelectorAll('button.sidebar__item')
  expect(items.length).toBe(3)
})
```

GAP-03 adds a rendered-item assertion in a `describe` block labeled `NAV-05 positive render contract` (or appends to the existing `Sidebar component (SIDE-01)` describe). The new test asserts:
- `querySelectorAll('button.sidebar__item').length` equals 3 with no group defs supplied
- No element with text matching `Sessions` or `Remote` is present

**afterEach cleanup pattern** (lines 64-69):
```typescript
afterEach(() => {
  root.unmount()
  container.remove()
})
```

**Imports already in file** (lines 1-8) — no new imports needed for GAP-03.

---

### `frontend/src/components/__tests__/style.hub.test.ts` — extend with GAP-04 (test, transform)

**This is an extension of an existing file.** The file reads `style.css` via `readFileSync` at line 9. All new `describe` blocks append to the file and reuse `cssRaw`.

**CSS source-gate pattern to copy** (lines 8-9, 13-14, 66-67):
```typescript
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

// Basic presence check:
it('defines .hub-card__row5 (actions row)', () => {
  expect(cssRaw).toContain('.hub-card__row5')
})

// Hex pin check (colorblind constraint):
it('dark theme --hub-bg has correct hex #14151b (comp dark surface)', () => {
  expect(cssRaw).toContain('--hub-bg: #14151b')
})
```

**Block-scoped extraction pattern for multi-property rules** (lines 119-125):
```typescript
it('.hub__card-row declares display: grid', () => {
  const hubCardRowIdx = cssRaw.indexOf('.hub__card-row')
  expect(hubCardRowIdx).toBeGreaterThan(-1)
  const blockEnd = cssRaw.indexOf('}', hubCardRowIdx)
  const block = cssRaw.slice(hubCardRowIdx, blockEnd)
  expect(block).toContain('display: grid')
})
```

GAP-04 adds a new `describe` block at the bottom of the file:
```typescript
describe('Phase 142 comp-fidelity CSS tokens (GAP-04 anti-regression)', () => {
  it('defines .hub-card__spine class', () => { ... })
  it('defines .hub-card__origin-chip class', () => { ... })
  it('hub-card border-radius is 16px', () => { expect(cssRaw).toContain('border-radius: 16px') })
  it('.hub-card__preview has min-height: 150px', () => { ... })
})
```

No new imports — `cssRaw` is already declared at file scope.

---

### `tests/check-traceability-paths.sh` (utility, batch)

**Analog:** `tests/build-script.test.sh`

**Shebang and preamble pattern** (lines 1-8):
```bash
#!/usr/bin/env bash
# tests/build-script.test.sh
# Behavioral tests for build.sh argument parsing, error paths, and static pattern checks.
# Does NOT require Wails, Docker, or mingw-w64.
#
# Run: bash tests/build-script.test.sh    (from project root)

set -uo pipefail
```

Note: `build-script.test.sh` uses `set -uo pipefail` (without `-e` to allow controlled FAIL increments). The path-check script per the RESEARCH.md design uses `set -euo pipefail` because it exits on first failure — that is intentional and correct for the path-check use case.

**Pass/fail counter pattern** (lines 12-14, 319-325):
```bash
PASS=0
FAIL=0
# ...
if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
exit 0
```

**File existence check pattern** (lines 92-96):
```bash
if [[ -f "$BUILD_SH" ]]; then
  pass "build.sh exists at project root"
else
  fail "build.sh exists at project root" "file not found: $BUILD_SH"
fi
```

**`if [[ ! -e "$path" ]]` pattern** (adapted from analog for the loop body in `check-traceability-paths.sh`):
```bash
if [[ ! -e "$path" ]]; then
  echo "MISSING traceability path: $path"
  FAIL=$((FAIL + 1))
fi
```

The RESEARCH.md section "Path-Existence CI Check" provides the complete ~12-line implementation. The script's structure (shebang + `set -euo pipefail` + loop + loud `exit 1`) is consistent with the `build-script.test.sh` analog's conventions.

---

### `.github/workflows/build.yml` — add path-check step (config, request-response)

**Analog:** the existing "Run build script tests" step (lines 60-62):
```yaml
- name: Run build script tests
  if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'
  run: bash tests/build-script.test.sh
```

**Pattern to copy exactly** — the new step uses the identical `if:` condition (ubuntu-latest Linux only, same as frontend tests) and the same `run: bash tests/...` form:
```yaml
- name: Verify traceability paths exist
  if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'
  run: bash tests/check-traceability-paths.sh
```

**Placement:** Insert immediately after the "Run frontend tests" step (line 76-78) and before "Install Wails CLI" (line 80). The "Run frontend tests" step is:
```yaml
- name: Run frontend tests
  if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'
  run: cd frontend && pnpm install && pnpm test
```

---

## Shared Patterns

### Pure-function vitest lib test pattern
**Source:** `frontend/src/lib/hubStatus.test.ts` (36 lines total)
**Apply to:** `hubGroupCounts.test.ts`, `agentBadge.test.ts`

Key rules extracted from analogs:
1. Import only `{ describe, it, expect }` — no `vi`, no `beforeEach` unless state needs reset
2. One `describe` block per exported function
3. Assertion style: `expect(fn(input)).toBe(value)` — direct, no intermediate variables unless they clarify the type
4. Label test names with the requirement ID as a suffix where applicable (e.g., `'GROUP-03 persistence'` in `hubGroups.test.ts` line 56)

### CSS source-gate pattern
**Source:** `frontend/src/components/__tests__/style.hub.test.ts` (lines 8-9, 66-67, 119-125)
**Apply to:** `style.hub.test.ts` extension (GAP-04)

Key rules:
- `readFileSync` at file scope, reused by all `describe` blocks — never re-read inside `it()`
- Hex values pinned as string literals: `expect(cssRaw).toContain('#14151b')` not computed
- Block-scoped extraction (`indexOf` + `slice`) for multi-property assertions within a single CSS rule
- All new `describe` blocks append to the bottom of the file (preserve existing test order)

### Component render + cleanup pattern
**Source:** `frontend/src/components/__tests__/Sidebar.test.tsx` (lines 40-69)
**Apply to:** `Sidebar.test.tsx` extension (GAP-03)

Key rules:
- `renderSidebar(overrides?)` helper abstracts `createRoot` + `act()` + default props
- `afterEach` always calls `root.unmount()` then `container.remove()` — no leaking DOM nodes
- `container` and `root` declared in `describe` scope, assigned inside `it()` via destructuring

### bash script CI gate pattern
**Source:** `tests/build-script.test.sh` (lines 1-8, 319-325)
**Apply to:** `tests/check-traceability-paths.sh`

Key rules:
- `#!/usr/bin/env bash` shebang
- `set -...pipefail` near the top
- Non-zero exit on failure (`exit 1`)
- Run via `bash tests/<script>.sh` from project root (no install required, no PATH assumptions)

### `build.yml` conditional step pattern
**Source:** `.github/workflows/build.yml` lines 60-62
**Apply to:** new "Verify traceability paths exist" step

Key rules:
- `if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'` — single-runner gate
- `run: bash tests/<script>.sh` — no `shell:` override needed (GitHub Actions defaults to bash on Linux)
- Step name follows sentence case: "Verb noun phrase"

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `TESTING.md` (repo root) | documentation | — | No existing suite-manifest/checklist doc in repo; content is authored from RESEARCH.md, not copied from code |
| `CLAUDE.md` (repo root) | documentation | — | No existing repo-level CLAUDE.md; content is a one-line pointer (D-14); pattern is prose, not code |

---

## Metadata

**Analog search scope:** `frontend/src/lib/`, `frontend/src/components/__tests__/`, `tests/`, `.github/workflows/`
**Files read:** 8 (hubGroups.test.ts, hubStatus.test.ts, remoteAdapter.test.ts, relayClient.test.ts, style.hub.test.ts, Sidebar.test.tsx [first 100 lines], build-script.test.sh, build.yml [first 110 lines])
**Pattern extraction date:** 2026-06-21

---

## PATTERN MAPPING COMPLETE

**Phase:** 143 - Regression Test Program
**Files classified:** 6
**Analogs found:** 5 / 6

### Coverage
- Files with exact analog: 2 (`hubGroupCounts.test.ts` → `hubStatus.test.ts`; `agentBadge.test.ts` → `hubStatus.test.ts`)
- Files with self-extension pattern: 2 (`Sidebar.test.tsx`, `style.hub.test.ts` — existing files being extended)
- Files with role-match analog: 1 (`check-traceability-paths.sh` → `build-script.test.sh`; `build.yml` step → existing step)
- Files with no analog: 2 (`TESTING.md`, `CLAUDE.md` — documentation, no code to copy)

### Key Patterns Identified
- All new lib tests copy the `hubStatus.test.ts` pattern: `{ describe, it, expect }` only, one describe per function, one it per input class, direct `toBe`/`toEqual` assertions with concrete values
- CSS source-gate tests use `readFileSync` at file scope and `cssRaw.toContain(literal)` — hex values pinned as string constants for colorblind-safe source verification (no visual inspection)
- The `build.yml` step pattern is `if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'` + `run: bash tests/<name>.sh` — identical to the existing build-script step at line 61
- Shell scripts follow `#!/usr/bin/env bash` + `set -...pipefail` + explicit `exit 1` on failure

### Files Created
`/Users/ken/dev/agenthub/.planning/phases/143-regression-test-program/143-PATTERNS.md`

### Ready for Planning
Pattern mapping complete. Planner can now reference analog patterns in PLAN.md files.
