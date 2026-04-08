# Phase 56: Navigation Wiring & Tab Bar Cleanup - Research

**Researched:** 2026-04-08
**Domain:** React component wiring, callback props, CSS cleanup
**Confidence:** HIGH

## Summary

Phase 56 is primarily a wiring and cleanup phase — the heavy lifting was done in Phase 55.
The Sidebar component already exists with all five navigation buttons and their callback props
(`onHome`, `onOpenRemoteSessions`, `onOpenDaemonManager`, `onAdd`, `onSettings`). App.tsx already
wires all five handlers into the Sidebar. The navigation logic (`handleHome`,
`handleOpenRemoteSessions`, `handleOpenDaemonManager`, `handleAddTab`) is fully implemented in
App.tsx and was verified working in Phase 55.

The only remaining work is: (1) verify the wiring is complete and correct against the requirements,
(2) remove the now-dead CSS for `.tab-bar__controls` and `.tab-bar__btn` from style.css (those
buttons no longer exist in TabBar.tsx), and (3) write RED tests for the NAV-01..05 and TAB-01
requirements before verifying.

The test suite is currently GREEN at 256 tests, 0 failures. No pre-existing failures to fix.

**Primary recommendation:** Write RED tests for NAV-01..05 and TAB-01 using the `App.tsx?raw`
source-inspection pattern established by `App.wiring.test.tsx`. Then verify App.tsx already satisfies
them (it does). Remove dead CSS. Tests go green. Done.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NAV-01 | User can click Home icon to open the Welcome tab | `handleHome` in App.tsx: finds existing `type === 'welcome'` tab or adds WELCOME_TAB; Sidebar `onHome={handleHome}` already wired |
| NAV-02 | User can click Remote icon to open the Remote Sessions panel | `handleOpenRemoteSessions` in App.tsx: finds existing `type === 'remote-sessions'` tab or adds REMOTE_SESSIONS_TAB; Sidebar `onOpenRemoteSessions={handleOpenRemoteSessions}` already wired |
| NAV-03 | User can click Sessions icon to open the Daemon Manager panel | `handleOpenDaemonManager` in App.tsx: finds existing `type === 'daemon-manager'` tab or adds DAEMON_MANAGER_TAB; Sidebar `onOpenDaemonManager={handleOpenDaemonManager}` already wired |
| NAV-04 | User can click New Tab icon to create a new terminal session | `handleAddTab` in App.tsx: opens NewSessionModal or Settings if no CLIs; Sidebar `onAdd={handleAddTab}` already wired |
| NAV-05 | User can click Settings icon (pinned to sidebar bottom) to open the Settings panel | `setShowSettings(true)` lambda; Sidebar `onSettings={() => setShowSettings(true)}` already wired; Settings button is in `sidebar__bottom` div |
| TAB-01 | Tab bar retains session tabs but no longer has action buttons on the right | TabBar.tsx already has no action buttons (removed in Phase 55); dead CSS `.tab-bar__controls`/`.tab-bar__btn` remain in style.css |
</phase_requirements>

## Current State Assessment (Critical for Planning)

**What Phase 55 completed:**
- Sidebar.tsx created with all 5 nav buttons + hamburger toggle
- App.tsx wired: `onHome`, `onOpenRemoteSessions`, `onOpenDaemonManager`, `onAdd`, `onSettings`
- TabBar.tsx has NO action buttons — only session tabs with close/rename
- App layout restructured to flex-row (sidebar + app__content)

**What Phase 56 must do:**
1. Write RED tests for NAV-01..05 and TAB-01
2. Verify tests go green (they already should — wiring is in place)
3. Remove dead CSS: `.tab-bar__controls` and `.tab-bar__btn` blocks from style.css
4. Update/remove the UILAY-01 test describe block in TabBar.test.tsx (it tests `.tab-bar__btn` CSS which will be deleted)

**UILAY-01 test conflict:** `TabBar.test.tsx` lines 82-110 have a `describe('UILAY-01 toolbar button dimensions (style.css)')` that asserts `.tab-bar__btn` CSS properties exist. When we delete `.tab-bar__btn` from style.css (per TAB-01), these 4 tests will fail. The plan MUST address this: remove the UILAY-01 describe block since the buttons no longer exist.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2.4 (installed) | Component framework | Project baseline |
| @heroicons/react | 2.2.0 (installed) | Sidebar icons | Installed in Phase 55 |

No new dependencies required for this phase.

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| vitest | 4.1.0 (installed) | Testing | RED/GREEN test cycle |

**Installation:** None required.

## Architecture Patterns

### Pattern 1: Source-Inspection Tests for App.tsx Wiring

The existing `App.wiring.test.tsx` uses `import raw from '../../App.tsx?raw'` to inspect App.tsx
source code without mounting the full component tree (which requires Wails runtime mocks). This is
the established pattern for verifying wiring in this codebase.

**When to use:** Any test that verifies App.tsx wiring (imports, prop passing, handler definitions,
tab type constants) without needing DOM rendering.

**Example:**
```typescript
// Source: App.wiring.test.tsx (established pattern)
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App.tsx sidebar navigation wiring (NAV-01)', () => {
  it('defines handleHome callback', () => {
    expect(raw).toContain('handleHome')
  })

  it('passes onHome to Sidebar', () => {
    expect(raw).toContain('onHome={handleHome}')
  })

  it('handleHome finds existing welcome tab by type', () => {
    expect(raw).toContain("t.type === 'welcome'")
  })
})
```

### Pattern 2: Tab Type Routing Pattern (already implemented)

All navigation targets use the "find-or-create" pattern:
1. Search `tabs` for an existing tab with the target `type`
2. If found, call `setActiveId(existing.id)`
3. If not found, append the typed tab constant and set it active

This prevents duplicate Welcome/Sessions/Remote tabs.

```typescript
// Source: App.tsx — handleOpenDaemonManager (reference implementation)
const handleOpenDaemonManager = useCallback(() => {
  const existing = tabs.find((t) => t.type === 'daemon-manager')
  if (existing) {
    setActiveId(existing.id)
    return
  }
  setTabs((prev) => [...prev, DAEMON_MANAGER_TAB])
  setActiveId(DAEMON_MANAGER_TAB.id)
}, [tabs])
```

All three navigation panel handlers (`handleHome`, `handleOpenRemoteSessions`,
`handleOpenDaemonManager`) follow this exact pattern. `handleAddTab` and `setShowSettings` are
different in nature (modal triggers, not tab navigation).

### Pattern 3: CSS Dead Code Removal

The `.tab-bar__controls` and `.tab-bar__btn` CSS classes exist in style.css but are no longer
rendered anywhere in the component tree. Removing them is safe:

```bash
# Verify no usage before deletion
grep -rn "tab-bar__btn\|tab-bar__controls" frontend/src/ --include="*.tsx" --include="*.ts"
# Returns: no matches (confirmed)
```

### Anti-Patterns to Avoid

- **Don't mount App in tests for this phase:** App.tsx has Wails runtime imports that fail in
  jsdom without extensive mocking. Use the `?raw` import pattern from `App.wiring.test.tsx`.
- **Don't leave UILAY-01 tests intact when removing `.tab-bar__btn` CSS:** The 4 tests in that
  describe block check for CSS properties of `.tab-bar__btn`. Deleting the CSS without deleting
  the tests will cause 4 test failures. The correct action is to delete those test assertions
  (the feature they tested — toolbar action buttons — no longer exists).
- **Don't add `onHome` as a new feature:** It was added in Phase 55. Phase 56 just needs to
  test and verify it.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tab deduplication | Custom uniqueness logic | Find-or-create pattern (already in handleOpenDaemonManager) | Already proven, consistent |
| Modal trigger for New Tab | Custom event system | `setShowNewSessionModal(true)` directly in handleAddTab | Already implemented |

## Common Pitfalls

### Pitfall 1: UILAY-01 test breakage when removing tab-bar__btn CSS
**What goes wrong:** Removing `.tab-bar__btn` CSS (required for TAB-01 cleanup) will break the
4 assertions in `describe('UILAY-01 toolbar button dimensions (style.css)')` in TabBar.test.tsx.
**Why it happens:** The UILAY-01 tests check that the old toolbar buttons met 38x38px click-target
specs. Those buttons no longer exist.
**How to avoid:** Delete the UILAY-01 describe block before or in the same commit as the CSS removal.
**Warning signs:** `pnpm test` shows 4 new failures after CSS cleanup without test update.

### Pitfall 2: Leaving dead CSS in style.css
**What goes wrong:** Not removing `.tab-bar__controls` / `.tab-bar__btn` CSS leaves technical debt
and the UILAY-01 tests continue to assert on stale specs.
**How to avoid:** Remove both CSS blocks in the same plan wave as the UILAY-01 test cleanup.

### Pitfall 3: Testing the wrong thing for NAV-04 (New Tab)
**What goes wrong:** NAV-04 says "create a new terminal session (opens new-session modal)". The
test should verify that `handleAddTab` opens the `NewSessionModal` (sets `showNewSessionModal = true`)
when CLIs are detected, NOT that it directly creates a session.
**How to avoid:** Test that App.tsx contains `setShowNewSessionModal(true)` inside the `handleAddTab`
function, not a direct `CreateSession` call.

### Pitfall 4: Testing NAV-05 Settings as a tab (it's a modal, not a tab)
**What goes wrong:** Settings opens as a `SettingsPanel` (a modal/overlay), not as a tab. Tests
should verify `setShowSettings(true)` is called from the Sidebar `onSettings` prop, not that a
Settings tab is added to the `tabs` array.
**How to avoid:** Verify `onSettings={() => setShowSettings(true)}` in App.tsx source, and that
`<SettingsPanel isOpen={showSettings}` is present.

## Code Examples

### NAV wiring verification (source-inspection pattern)

```typescript
// Proposed test file: App.nav.test.tsx
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('NAV-01: Home sidebar button opens Welcome tab', () => {
  it('defines handleHome callback', () => {
    expect(raw).toContain('handleHome')
  })
  it('passes onHome={handleHome} to Sidebar', () => {
    expect(raw).toContain('onHome={handleHome}')
  })
  it('handleHome finds existing welcome tab by type', () => {
    expect(raw).toContain("t.type === 'welcome'")
  })
  it('handleHome adds WELCOME_TAB when none exists', () => {
    expect(raw).toContain('WELCOME_TAB')
  })
})

describe('NAV-02: Remote sidebar button opens Remote Sessions panel', () => {
  it('passes onOpenRemoteSessions={handleOpenRemoteSessions} to Sidebar', () => {
    expect(raw).toContain('onOpenRemoteSessions={handleOpenRemoteSessions}')
  })
  it('handleOpenRemoteSessions finds existing remote-sessions tab', () => {
    expect(raw).toContain("t.type === 'remote-sessions'")
  })
})

describe('NAV-03: Sessions sidebar button opens Daemon Manager panel', () => {
  it('passes onOpenDaemonManager={handleOpenDaemonManager} to Sidebar', () => {
    expect(raw).toContain('onOpenDaemonManager={handleOpenDaemonManager}')
  })
  it('handleOpenDaemonManager finds existing daemon-manager tab', () => {
    expect(raw).toContain("t.type === 'daemon-manager'")
  })
})

describe('NAV-04: New Tab sidebar button opens new-session modal', () => {
  it('passes onAdd={handleAddTab} to Sidebar', () => {
    expect(raw).toContain('onAdd={handleAddTab}')
  })
  it('handleAddTab opens NewSessionModal when CLIs detected', () => {
    expect(raw).toContain('setShowNewSessionModal(true)')
  })
})

describe('NAV-05: Settings sidebar button opens Settings panel', () => {
  it('passes onSettings to Sidebar', () => {
    expect(raw).toContain('onSettings={')
  })
  it('Settings callback calls setShowSettings(true)', () => {
    expect(raw).toContain('setShowSettings(true)')
  })
  it('SettingsPanel receives isOpen={showSettings}', () => {
    expect(raw).toContain('isOpen={showSettings}')
  })
})

describe('TAB-01: Tab bar has no action buttons', () => {
  it('TabBar does not receive onAdd prop', () => {
    // The TabBar JSX in App.tsx should not pass onAdd
    const tabBarBlock = raw.slice(raw.indexOf('<TabBar'), raw.indexOf('</TabBar>') + 9)
    expect(tabBarBlock).not.toContain('onAdd')
  })
  it('TabBar does not receive onSettings prop', () => {
    const tabBarBlock = raw.slice(raw.indexOf('<TabBar'), raw.indexOf('</TabBar>') + 9)
    expect(tabBarBlock).not.toContain('onSettings')
  })
  it('TabBar does not receive onOpenDaemonManager prop', () => {
    const tabBarBlock = raw.slice(raw.indexOf('<TabBar'), raw.indexOf('</TabBar>') + 9)
    expect(tabBarBlock).not.toContain('onOpenDaemonManager')
  })
})
```

### CSS blocks to remove from style.css

```css
/* DELETE THIS ENTIRE BLOCK — buttons moved to Sidebar in Phase 55 */
/* ─── Tab bar controls (+ and gear) ─────────────────────────────── */
.tab-bar__controls {
  display: flex;
  align-items: center;
  padding: 0 4px;
  gap: 2px;
  flex-shrink: 0;
}

.tab-bar__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: none;
  background: transparent;
  color: #9aa5ce;
  cursor: pointer;
  font-size: 20px;
  border-radius: 4px;
  transition: background-color 0.1s, color 0.1s;
}
.tab-bar__btn:hover {
  background-color: #1e2030;
  color: #c0caf5;
}
.tab-bar__btn--remote {
  font-size: 17px;
}
```

### UILAY-01 test block to remove from TabBar.test.tsx

```typescript
// DELETE THIS ENTIRE DESCRIBE BLOCK — .tab-bar__btn CSS is being removed
describe('UILAY-01 toolbar button dimensions (style.css)', () => {
  // 4 tests that assert .tab-bar__btn CSS properties
  // These tests are obsolete since the tab bar no longer has action buttons (TAB-01)
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Navigation via TabBar action buttons (+ gear, globe) | Navigation via Sidebar icons | Phase 55 (completed) | All nav triggers now in sidebar |
| Unicode/emoji icons in TabBar | Heroicons SVGs in Sidebar | Phase 55 (completed) | Crisp rendering, accessible |

**Deprecated/outdated (to clean up in Phase 56):**
- `.tab-bar__controls` CSS block — no element uses this class
- `.tab-bar__btn` CSS block — no element uses this class
- `.tab-bar__btn--remote` CSS block — no element uses this class
- UILAY-01 test describe block in TabBar.test.tsx — tests CSS that will be deleted

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — this phase is pure code/CSS changes with no new packages)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | frontend/vite.config.ts |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Current Test Suite Baseline

**256 tests, 0 failures** as of Phase 55 completion. All tests green.

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NAV-01 | handleHome defined, wired to Sidebar onHome, uses find-or-create welcome tab | unit (source inspection) | `cd .../frontend && pnpm test -- App.nav` | No — Wave 0 gap |
| NAV-02 | handleOpenRemoteSessions wired to Sidebar onOpenRemoteSessions | unit (source inspection) | `cd .../frontend && pnpm test -- App.nav` | No — Wave 0 gap |
| NAV-03 | handleOpenDaemonManager wired to Sidebar onOpenDaemonManager | unit (source inspection) | `cd .../frontend && pnpm test -- App.nav` | No — Wave 0 gap |
| NAV-04 | handleAddTab wired to Sidebar onAdd, opens NewSessionModal | unit (source inspection) | `cd .../frontend && pnpm test -- App.nav` | No — Wave 0 gap |
| NAV-05 | Settings callback wired to Sidebar onSettings, opens SettingsPanel modal | unit (source inspection) | `cd .../frontend && pnpm test -- App.nav` | No — Wave 0 gap |
| TAB-01 | TabBar receives no action button props; dead CSS removed | unit (source inspection + CSS check) | `cd .../frontend && pnpm test -- App.nav` | No — Wave 0 gap |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/App.nav.test.tsx` — covers NAV-01, NAV-02, NAV-03, NAV-04, NAV-05, TAB-01
- [ ] Remove UILAY-01 describe block from `frontend/src/components/__tests__/TabBar.test.tsx` — 4 tests that assert `.tab-bar__btn` CSS (will break when CSS is deleted for TAB-01)

## Open Questions

1. **TAB-01 scope: remove just the CSS, or also the CSS-reading tests?**
   - What we know: `TabBar.tsx` has no action buttons. `style.css` still has `.tab-bar__btn` CSS.
     `TabBar.test.tsx` UILAY-01 tests assert on `.tab-bar__btn` CSS. Removing the CSS will break
     those tests.
   - What's unclear: None — this is clear. Both must be removed together.
   - Recommendation: In a single wave, delete the CSS blocks AND the UILAY-01 describe block.

2. **Should App.nav.test.tsx be a new file or append to App.wiring.test.tsx?**
   - What we know: `App.wiring.test.tsx` is already focused on remote-sessions wiring (52-03-02).
   - Recommendation: Create a new `App.nav.test.tsx` file. Keeps concerns separated by phase and
     is consistent with the naming convention established by `App.wiring.test.tsx`.

## Sources

### Primary (HIGH confidence)
- `/Users/ken/dev/agenthub/frontend/src/App.tsx` — direct inspection; all 5 handlers confirmed wired
- `/Users/ken/dev/agenthub/frontend/src/components/Sidebar.tsx` — direct inspection; all 5 props confirmed
- `/Users/ken/dev/agenthub/frontend/src/components/TabBar.tsx` — direct inspection; no action buttons
- `/Users/ken/dev/agenthub/frontend/src/style.css` — direct inspection; `.tab-bar__btn` CSS dead code confirmed
- `/Users/ken/dev/agenthub/frontend/src/components/__tests__/TabBar.test.tsx` — direct inspection; UILAY-01 block confirmed needs removal
- `cd frontend && pnpm test` — confirmed 256 tests, 0 failures baseline

### Secondary (MEDIUM confidence)
- None required — all findings come from direct source inspection.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; everything installed
- Architecture: HIGH — wiring already exists; verified by reading source
- Pitfalls: HIGH — UILAY-01 breakage confirmed by reading TabBar.test.tsx lines 82-110

**Research date:** 2026-04-08
**Valid until:** N/A — this is a final cleanup phase for v1.10

## Project Constraints (from CLAUDE.md)

| Directive | Applies To |
|-----------|------------|
| Use `pnpm` for Node package management | No new packages this phase |
| TypeScript strict mode, `camelCase`/`PascalCase` | New test file must use proper naming |
| ESLint + Prettier | Format test file consistently |
| Testing: vitest, 80%+ coverage in critical components | App.nav.test.tsx required before implementation |
| `noUnusedLocals`, `noUnusedParameters` | N/A — no new component code |
