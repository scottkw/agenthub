---
phase: 62-tech-debt-cleanup
reviewed: 2026-04-10T12:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/App.test.tsx
  - frontend/src/components/__tests__/App.nav.test.tsx
  - frontend/src/style.css
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 62: Code Review Report

**Reviewed:** 2026-04-10T12:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

This phase performs tech-debt cleanup: removing the obsolete HealthModal component and its CSS (~200 lines), removing the `settings-overlay` CSS (settings moved to an in-tab panel in a prior phase), updating tests to match the current SettingsTab/LocalNetworkBanner wiring, and adding SERVE-02 web-enabled state seeding for both `init()` and `retryInit()` code paths.

The CSS cleanup is clean -- no stale references to `health-modal` or `settings-overlay` remain anywhere in the codebase. The test updates correctly reflect the current component structure. The SERVE-02 seeding logic is correct but introduces significant code duplication.

Two warnings flag bugs that could affect correctness. No critical (security) issues found.

## Warnings

### WR-01: Stale closure on `remotePeers` in remote-sessions polling useEffect

**File:** `frontend/src/App.tsx:380`
**Issue:** The `useEffect` at line 376 captures `remotePeers` in its closure but only lists `[activeId]` as a dependency. The `refresh()` function reads `remotePeers.length` at line 380 to decide whether to show a loading spinner. On the first call this works correctly (length is 0, spinner shows). But when the 30-second interval fires subsequent calls, `remotePeers` in the closure is still the value from when the effect was created (empty array), so `setRemoteLoading(true)` fires on every poll cycle, causing the loading spinner to briefly flash every 30 seconds even though data is already loaded.
**Fix:** Use a ref to track whether initial data has loaded, or add `remotePeers` to the dependency array (though the latter will restart the interval on every fetch). The cleanest fix:
```tsx
// Replace line 380 with a ref-based check:
const remotePeersLoadedRef = useRef(false)

// In the useEffect:
useEffect(() => {
  if (activeId !== REMOTE_SESSIONS_TAB.id) return
  remotePeersLoadedRef.current = false
  let cancelled = false
  async function refresh() {
    if (!remotePeersLoadedRef.current) setRemoteLoading(true)
    try {
      const peers = await GetRemoteSessions()
      if (!cancelled) {
        setRemotePeers(peers ?? [])
        remotePeersLoadedRef.current = true
        setRemoteLoading(false)
      }
    } catch {
      if (!cancelled) setRemoteLoading(false)
    }
  }
  void refresh()
  const interval = setInterval(() => void refresh(), 30_000)
  return () => { cancelled = true; clearInterval(interval) }
}, [activeId])
```

### WR-02: Duplicated init/retryInit session-restore logic (divergence risk)

**File:** `frontend/src/App.tsx:84-181` and `frontend/src/App.tsx:435-503`
**Issue:** The `init()` function (lines 84-181) and `retryInit()` callback (lines 435-503) contain nearly identical session-restoration and web-enabled-seeding logic, duplicated across ~60 lines each. The SERVE-02 seeding block was copy-pasted into both paths in this phase. The `init()` path has additional logic that `retryInit()` lacks (the web-server polling loop at lines 113-131), creating a subtle behavioral difference: after `retryInit()`, if the web server is still starting up, the mode/running state may not be detected. Any future change to the init sequence must be applied in two places, and forgetting one creates a silent regression.
**Fix:** Extract the shared session-restoration logic into a helper function:
```tsx
async function restoreSessions(sessions: SessionInfo[], running: boolean) {
  if (sessions.length === 0) return
  const restoredTabs: Tab[] = sessions.map((s) => ({
    id: s.id, name: s.name || s.cli, sessionId: s.id, cli: s.cli,
  }))
  setTabs(restoredTabs)
  setActiveId(restoredTabs[0].id)
  sessions.forEach((s) => {
    GetSessionStatus(s.id)
      .then((st) => setSessionStatuses((prev) => ({ ...prev, [s.id]: st })))
      .catch(() => {})
  })
  if (running) {
    // ... shared SERVE-02 seeding logic
  }
}
```

## Info

### IN-01: Tab constant objects recreated on every render

**File:** `frontend/src/App.tsx:40-43`
**Issue:** `WELCOME_TAB`, `DAEMON_MANAGER_TAB`, `REMOTE_SESSIONS_TAB`, and `SETTINGS_TAB` are declared inside the component body, creating new object references on every render. While this does not cause bugs (comparisons use `.id` strings), it means closures in `useCallback` hooks like `handleHome` (line 425) and `handleOpenSettings` (line 264) capture a fresh object each render. If these constants were moved outside the component or wrapped in `useMemo`, the callbacks that reference them could have stable dependencies.
**Fix:** Move the four `const` declarations above the `function App()` line, or to a module-level constant.

### IN-02: `console.error` and `console.warn` calls retained in production code

**File:** `frontend/src/App.tsx:178,260,293,317,341,364,500`
**Issue:** Seven `console.error`/`console.warn` calls exist across the component. These are tagged with `[App]` prefixes which is good for debugging, but they will appear in production builds. This is a minor concern since Wails apps run in a WebView and these logs are typically not user-visible, but it is worth noting for consistency.
**Fix:** No action required for now. Consider a logging utility with configurable levels in a future phase.

### IN-03: `settings-panel` CSS classes retained despite SettingsPanel removal

**File:** `frontend/src/style.css:304-561`
**Issue:** The `.settings-panel` CSS block (~250 lines) remains in `style.css`. The old `SettingsPanel` modal component was replaced by `SettingsTab`, but `SettingsTab` reuses these same CSS class names, so these styles are still actively used. This is not dead code -- just a naming inconsistency where the CSS says "panel" but the component is now "tab". No action needed unless a naming cleanup is desired.
**Fix:** No action required. The class names match the styled elements and renaming would be a large, low-value change.

---

_Reviewed: 2026-04-10T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
