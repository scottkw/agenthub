---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - frontend/src/App.tsx
  - frontend/src/components/SettingsTab.tsx
autonomous: true
must_haves:
  truths:
    - "No HealthModal flashes on app startup regardless of Tailscale state"
    - "Clicking Settings gear in sidebar opens Settings as an inline tab, not a modal overlay"
    - "Settings tab shows LAN password section when webServerMode is 'local' and server is running"
    - "Settings tab follows singleton pattern (clicking gear when already open focuses existing tab)"
  artifacts:
    - path: "frontend/src/App.tsx"
      provides: "Root component without HealthModal, with Settings-as-tab"
    - path: "frontend/src/components/SettingsTab.tsx"
      provides: "Inline settings tab with webServerMode and LAN password support"
  key_links:
    - from: "Sidebar onSettings"
      to: "handleOpenSettings"
      via: "callback prop"
      pattern: "onSettings.*handleOpenSettings"
    - from: "App.tsx terminal-container"
      to: "SettingsTab"
      via: "conditional render when settings tab active"
      pattern: "SETTINGS_TAB\\.id.*SettingsTab"
---

<objective>
Fix two UI bugs: (1) Remove HealthModal that flashes on startup due to race condition between webServerRunning and tailscaleHealth states -- it is obsolete since Phase 60 added LocalNetworkBanner as the Tailscale nudge mechanism. (2) Switch Settings from a modal overlay (SettingsPanel) to an inline tab (SettingsTab), merging the LAN password feature from SettingsPanel into SettingsTab.

Purpose: Eliminate jarring startup flash and complete the Phase 58 intent of Settings-as-tab.
Output: Clean App.tsx with no HealthModal references, SettingsTab rendered inline with LAN password support.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@frontend/src/App.tsx
@frontend/src/components/SettingsTab.tsx
@frontend/src/components/SettingsPanel.tsx
@frontend/src/components/HealthModal.tsx

<interfaces>
<!-- From SettingsTab.tsx - current interface (will be extended): -->
```typescript
interface SettingsTabProps {
  clis: DetectedCLI[]
  tailscaleHealth: {
    installed: boolean
    connected: boolean
    hasCerts: boolean
    ip: string
    domain: string
  } | null
  onWebServerStateChange: () => Promise<void>
}
```

<!-- From TabBar.tsx line 8 - Tab type already supports 'settings': -->
```typescript
type?: 'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions' | 'settings'
```

<!-- From Sidebar.tsx - onSettings is a simple callback: -->
```typescript
onSettings: () => void
```

<!-- From App.tsx - singleton tab pattern used by DaemonManagerPanel (line 414-424): -->
```typescript
const DAEMON_MANAGER_TAB: Tab = { id: '__daemon_manager__', name: 'Sessions', sessionId: '', cli: '', type: 'daemon-manager' }

const handleOpenDaemonManager = useCallback(() => {
  const existing = tabs.find((t) => t.type === 'daemon-manager')
  if (existing) { setActiveId(existing.id); return }
  setTabs((prev) => [...prev, DAEMON_MANAGER_TAB])
  setActiveId(DAEMON_MANAGER_TAB.id)
}, [tabs])
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add webServerMode and LAN password to SettingsTab</name>
  <files>frontend/src/components/SettingsTab.tsx</files>
  <action>
Extend SettingsTab to support the LAN password feature currently only in SettingsPanel:

1. Add `webServerMode` to the props interface:
   ```
   webServerMode?: 'tailscale' | 'local' | null
   ```

2. Add `GetLocalNetworkPassword` to the imports from `../wailsjs/go/main/App`.

3. Add state for LAN password display (same as SettingsPanel lines 55-56):
   ```
   const [localPassword, setLocalPassword] = useState('')
   const [copied, setCopied] = useState(false)
   ```

4. Add the useEffect to fetch LAN password when in local mode (same as SettingsPanel lines 82-88):
   ```
   useEffect(() => {
     if (webServerMode === 'local' && isServerRunning) {
       GetLocalNetworkPassword().then(setLocalPassword).catch(() => setLocalPassword(''))
     } else {
       setLocalPassword('')
     }
   }, [webServerMode, isServerRunning])
   ```

5. Add the copy handler (same as SettingsPanel lines 90-95):
   ```
   async function handleCopyPassword() {
     if (!localPassword) return
     await navigator.clipboard.writeText(localPassword)
     setCopied(true)
     setTimeout(() => setCopied(false), 1500)
   }
   ```

6. Add the LAN password JSX block inside the `activeTab === 'web-server'` section, after the Start/Stop Server field-group (before the closing fragment). Copy verbatim from SettingsPanel lines 321-346:
   - Mode indicator div
   - LAN Access Credentials label
   - Username hint
   - Password label
   - Clickable password field with copy functionality

7. Destructure `webServerMode` from props in the function signature.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && npx --prefix frontend tsc --noEmit --project frontend/tsconfig.json 2>&1 | head -30</automated>
  </verify>
  <done>SettingsTab accepts webServerMode prop and displays LAN password section when mode is 'local' and server is running, matching SettingsPanel's feature set.</done>
</task>

<task type="auto">
  <name>Task 2: Remove HealthModal, switch Settings from modal to tab in App.tsx</name>
  <files>frontend/src/App.tsx</files>
  <action>
Two changes in App.tsx: remove HealthModal entirely, and replace SettingsPanel modal with SettingsTab inline tab.

**Part A — Remove HealthModal:**

1. Remove import of `HealthModal` (line 29).
2. Remove import of `AutoInstallTailscale` from wailsjs bindings (line 21) — only used by HealthModal.
3. Remove state declarations (lines 77-79):
   - `installProgress`
   - `installStatus`
   - `installError`
4. Remove event listeners in the mount useEffect (lines 202-212):
   - `cancelInstallProgress` (EventsOn 'tailscale:install:progress')
   - `cancelInstallDone` (EventsOn 'tailscale:install:done')
   - Remove both from the cleanup return (lines 219-220: `cancelInstallProgress()` and `cancelInstallDone()`)
5. Remove callback `handleCheckHealthAgain` (lines 347-354).
6. Remove callback `handleAutoInstallTailscale` (lines 356-366).
7. Remove the HealthModal JSX (lines 643-653).

Do NOT remove: `tailscaleHealth` state, `GetTailscaleStatus` import/call, `tailscale:health` event listener — these are still consumed by SettingsTab.

**Part B — Replace SettingsPanel with SettingsTab:**

1. Change import: replace `import { SettingsPanel } from './components/SettingsPanel'` (line 5) with `import { SettingsTab } from './components/SettingsTab'`.

2. Add the SETTINGS_TAB constant alongside the other tab constants (near line 44):
   ```
   const SETTINGS_TAB: Tab = { id: '__settings__', name: 'Settings', sessionId: '', cli: '', type: 'settings' }
   ```

3. Remove the `showSettings` state (line 48): `const [showSettings, setShowSettings] = useState(false)`.

4. Remove the `handleSettingsClose` callback (lines 325-337).

5. Add `handleOpenSettings` callback using the singleton tab pattern (matching handleOpenDaemonManager exactly):
   ```
   const handleOpenSettings = useCallback(() => {
     const existing = tabs.find((t) => t.type === 'settings')
     if (existing) {
       setActiveId(existing.id)
       return
     }
     setTabs((prev) => [...prev, SETTINGS_TAB])
     setActiveId(SETTINGS_TAB.id)
   }, [tabs])
   ```

6. Update the Sidebar `onSettings` prop (line 513): change `onSettings={() => setShowSettings(true)}` to `onSettings={handleOpenSettings}`.

7. In the `handleAddTab` callback (line 256), change `setShowSettings(true)` to `handleOpenSettings()` (no-CLIs fallback opens settings).

8. Add SettingsTab render block in the terminal-container, alongside the other tab panels (after RemoteSessionsPanel, before the daemonError block):
   ```
   {activeId === SETTINGS_TAB.id && (
     <SettingsTab
       clis={detectedCLIs}
       tailscaleHealth={tailscaleHealth}
       webServerMode={webServerMode}
       onWebServerStateChange={async () => {
         try {
           const running = await IsWebServerRunning()
           setWebServerRunning(running)
           if (!running) {
             setWebServerMode(null)
           } else {
             const mode = await GetWebServerMode()
             setWebServerMode(mode === 'tailscale' || mode === 'local' ? mode : null)
           }
         } catch (_) { /* ignore */ }
       }}
     />
   )}
   ```

9. Remove the SettingsPanel JSX block (lines 635-641).

10. Update the terminal tab filter to also skip 'settings' type. The `tabs.map` in the terminal rendering area (line 585) already filters `welcome`, `daemon-manager`, `remote-sessions` — add `settings`:
    ```
    if (tab.type === 'welcome' || tab.type === 'daemon-manager' || tab.type === 'remote-sessions' || tab.type === 'settings') return null
    ```
    Also update the daemonError guard (line 546) to include `'settings'` in the filter.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && npx --prefix frontend tsc --noEmit --project frontend/tsconfig.json 2>&1 | head -30</automated>
  </verify>
  <done>
- HealthModal import, state, events, callbacks, and JSX are fully removed from App.tsx
- SettingsPanel import and JSX are removed; SettingsTab renders inline as a tab
- Settings gear opens a singleton tab (not a modal)
- No TypeScript compilation errors
- AutoInstallTailscale import removed (no longer referenced)
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

No new trust boundaries introduced — this is a removal/refactor of existing UI components.

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-quick-01 | I (Info Disclosure) | LAN password display in SettingsTab | accept | Password already displayed in SettingsPanel (modal); moving to tab does not change exposure. Password is only shown when webServerMode=local and server running. Clipboard copy is user-initiated. |
</threat_model>

<verification>
1. `npx --prefix frontend tsc --noEmit` passes with no errors
2. App starts without any flash/modal on startup
3. Clicking Settings gear opens an inline tab (not modal overlay)
4. Settings tab shows LAN password when in local network mode
5. Clicking Settings gear when tab already open focuses existing tab
6. Closing Settings tab works via tab close button
</verification>

<success_criteria>
- Zero HealthModal references in App.tsx (grep returns nothing)
- Zero SettingsPanel references in App.tsx (grep returns nothing)
- SettingsTab renders inline in terminal-container
- LAN password section visible in SettingsTab when webServerMode='local'
- TypeScript compiles cleanly
</success_criteria>

<output>
After completion, create `.planning/quick/260409-vop-remove-flashing-tailscale-check-modal-an/260409-vop-SUMMARY.md`
</output>
