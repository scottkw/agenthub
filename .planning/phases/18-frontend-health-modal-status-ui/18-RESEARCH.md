# Phase 18: Frontend Health Modal + Status UI - Research

**Researched:** 2026-03-22
**Domain:** React/TypeScript — Wails frontend, modal UI, event-driven state, platform-specific instructional content
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HEALTH-04 | User sees a modal with clear, actionable instructions when any health check fails | New `HealthModal` React component consumes `GetTailscaleStatus()` return value and `tailscale:health` events already emitted by Phase 14 backend; renders one of three instruction panels based on which flag is false |
| HEALTH-05 | Modal instructions are platform-specific (macOS, Linux, Windows) | Wails `Environment()` runtime call returns `{ platform: "darwin" | "linux" | "windows" }` — call once on app init, store in state, pass to modal; no new Go backend needed |
</phase_requirements>

---

## Summary

Phase 18 is a pure frontend change. The entire health check backend (TailscaleHealth struct, CheckHealth, startHealthPoller, GetTailscaleStatus Wails binding, `tailscale:health` event emission) was completed in Phase 14. The frontend currently never reads `GetTailscaleStatus()` or listens to `tailscale:health` events — they are wired but unused.

This phase adds two new React components and wires them into the existing `App.tsx`:

1. **`HealthModal`** — A modal overlay that appears when `TailscaleHealth` indicates a problem. Shows one of three distinct instruction panels: "Not Installed", "Not Connected", or "Certs Not Enabled". Each panel contains platform-specific instructions determined by the Wails `Environment()` call. The "Certs Not Enabled" panel includes a "Check Again" button that re-polls `GetTailscaleStatus()`. The CT disclosure text from Phase 15 must be visible within this modal before any cert provisioning attempt (TLS-04 cross-reference).

2. **Tailscale status indicator in SettingsPanel's Web Server tab** — Replaces the removed VPN interface picker (CLEAN-01/03 removed it in Phase 17). Shows a read-only colored status pill: green dot + "Connected" when healthy, amber dot + "Not Connected", or red dot + "Not Installed".

`App.tsx` gains health state (`TailscaleHealth | null`), calls `GetTailscaleStatus()` on init, subscribes to `tailscale:health` events, derives a boolean `healthOk` (all three flags true), and passes health state down to both the modal and the settings panel.

**Primary recommendation:** Add `HealthModal.tsx` as a standalone component. Add a `TailscaleStatusIndicator` inline sub-component inside `SettingsPanel.tsx`. Wire both through `App.tsx` health state. Use the existing `?raw` source-inspection test pattern for all new tests — no jsdom behavioral tests needed.

---

## Standard Stack

### Core (already present — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2.4 | UI components, state, effects | Existing project stack |
| TypeScript | 5.9.3 | Types | Existing project stack |
| Wails runtime `Environment()` | v2 (existing) | Returns `{ platform: "darwin"|"linux"|"windows" }` — one async call on app init | Already stubbed in `wailsjs/wailsjs/runtime/runtime.d.ts`; no new binding needed |
| `GetTailscaleStatus` Wails binding | Phase 14 | Returns `TailscaleHealth` struct | Already in `App.d.ts` and `App.js`; just needs frontend callsite |
| `tailscale:health` Wails event | Phase 14 | Push updates when health state changes | Already emitted by `startHealthPoller` in `app.go` |

### Supporting (already present)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `EventsOn` from Wails runtime | v2 | Subscribe to `tailscale:health` push events | Subscribe in `App.tsx` `useEffect` alongside existing `session:status` subscription |
| vitest | 4.1.0 | Test framework | Existing test suite; use `?raw` source-inspection pattern |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Wails `Environment()` for platform | Go backend `GetPlatform()` new binding | `Environment()` is already available with no new Go code needed — use it |
| `?raw` source-inspection tests | jsdom DOM tests | jsdom lacks Canvas/WebGL (breaks xterm.js tests); project decision is to use `?raw` pattern — consistent |
| Separate `TailscaleStatusIndicator` component file | Inline JSX in SettingsPanel | The indicator is small (~20 lines JSX) and only used in one place; inline keeps file count low |

**Installation:** No new packages needed. Everything is already in `package.json`.

---

## Architecture Patterns

### Recommended Project Structure

```
frontend/src/
├── components/
│   ├── HealthModal.tsx         ← NEW: health failure modal with platform instructions
│   ├── SettingsPanel.tsx       ← MODIFY: add TailscaleStatusIndicator in Web Server tab
│   └── __tests__/
│       ├── HealthModal.test.tsx     ← NEW: ?raw source-inspection tests
│       └── SettingsPanel.test.tsx   ← MODIFY: update for new status indicator
├── App.tsx                     ← MODIFY: add health state, Environment() call, event subscription
└── style.css                   ← MODIFY: add .health-modal and .ts-status classes
```

### Pattern 1: Health State in App.tsx

**What:** `App.tsx` owns `tailscaleHealth: TailscaleHealth | null` state. Initialized from `GetTailscaleStatus()` on mount (alongside existing `GetRelayPort`, `DetectCLIs` etc.). Updated by `tailscale:health` event subscription. Platform string fetched once via `Environment()` and stored as `platform: string`.

**When to use:** Centralize health state in the root component so both `HealthModal` and `SettingsPanel` receive it as props — consistent with how `webServerRunning` is already managed.

**Example:**
```typescript
// Source: App.tsx additions — mirrors existing session:status EventsOn pattern
import { Environment } from './wailsjs/wailsjs/runtime/runtime'
import { GetTailscaleStatus } from './wailsjs/go/main/App'
import type { TailscaleHealth } from './wailsjs/go/main/App'
import { HealthModal } from './components/HealthModal'

// State additions
const [tailscaleHealth, setTailscaleHealth] = useState<TailscaleHealth | null>(null)
const [platform, setPlatform] = useState<string>('linux')  // safe default

// In init useEffect — extend existing Promise.all
const [port, clis, sessions, running, health, env] = await Promise.all([
  GetRelayPort(),
  DetectCLIs(),
  ListSessions(),
  IsWebServerRunning(),
  GetTailscaleStatus(),
  Environment(),
])
setTailscaleHealth(health)
setPlatform(env.platform)

// Alongside existing session:status EventsOn subscription
const offHealth = EventsOn('tailscale:health', (h: TailscaleHealth) => {
  setTailscaleHealth(h)
})
return () => {
  offStatus()
  offHealth()
}
```

**Derived boolean for modal visibility:**
```typescript
const healthOk = tailscaleHealth !== null
  && tailscaleHealth.installed
  && tailscaleHealth.connected
  && tailscaleHealth.hasCerts
```

### Pattern 2: HealthModal Component

**What:** A modal overlay (same visual style as `SettingsPanel` and `QRModal`) that renders when `!healthOk`. Receives `health: TailscaleHealth | null` and `platform: string` props. Shows one of three panels depending on which flag is false, in priority order: not installed > not connected > no certs.

**When to use:** Displayed unconditionally in `App.tsx` JSX when `!healthOk`, identical to how `QRModal` is shown when `qrSessionId !== null`.

**Example structure:**
```typescript
// Source: frontend/src/components/HealthModal.tsx (new file)
interface HealthModalProps {
  health: TailscaleHealth | null
  platform: string  // 'darwin' | 'linux' | 'windows'
  onCheckAgain: () => void  // re-polls GetTailscaleStatus()
}

export function HealthModal({ health, platform, onCheckAgain }: HealthModalProps) {
  if (health === null) return null  // loading — no modal yet

  const isInstalled = health.installed
  const isConnected = health.connected
  const hasCerts    = health.hasCerts

  // All good — no modal
  if (isInstalled && isConnected && hasCerts) return null

  return (
    <div className="health-modal-overlay">
      <div className="health-modal">
        <div className="health-modal__header">
          <h2>Tailscale Setup Required</h2>
        </div>
        <div className="health-modal__body">
          {!isInstalled && <NotInstalledPanel platform={platform} />}
          {isInstalled && !isConnected && <NotConnectedPanel platform={platform} />}
          {isInstalled && isConnected && !hasCerts && (
            <NoCertsPanel platform={platform} onCheckAgain={onCheckAgain} />
          )}
        </div>
      </div>
    </div>
  )
}
```

**Panel determination priority:** Not installed takes precedence over not connected takes precedence over no certs. Only one panel is visible at a time.

### Pattern 3: Platform-Specific Instruction Panels

**What:** Three inline sub-components (or functions returning JSX) within `HealthModal.tsx`. Each receives `platform: string` and renders a different instruction set based on `platform === 'darwin'`, `'linux'`, or `'windows'`.

**Platform string values from Wails `Environment()`:** `"darwin"` (macOS), `"linux"`, `"windows"`. Verified from `runtime.d.ts` `EnvironmentInfo.platform` type.

**Example — NotInstalledPanel:**
```typescript
// Instructions sourced from Tailscale official docs
function NotInstalledPanel({ platform }: { platform: string }) {
  return (
    <div className="health-modal__panel">
      <p className="health-modal__title">Tailscale is not installed or not running.</p>
      {platform === 'darwin' && (
        <>
          <p>Install Tailscale from the Mac App Store or tailscale.com/download.</p>
          <p>Once installed, look for the Tailscale icon in your menu bar and sign in.</p>
        </>
      )}
      {platform === 'linux' && (
        <>
          <p>Install Tailscale with your package manager:</p>
          <code className="health-modal__code">curl -fsSL https://tailscale.com/install.sh | sh</code>
          <p>Then run: <code>sudo tailscale up</code></p>
        </>
      )}
      {platform === 'windows' && (
        <>
          <p>Download and install Tailscale from tailscale.com/download.</p>
          <p>Once installed, find Tailscale in the system tray and sign in.</p>
        </>
      )}
    </div>
  )
}
```

**NotConnectedPanel:** Platform-specific — macOS "click menu bar icon > Connect", Linux `sudo tailscale up`, Windows "click tray icon > Connect". Must be distinct from NotInstalledPanel text (HEALTH-05).

**NoCertsPanel:** Cert instructions are less platform-specific (all use the Tailscale admin console URL). This panel includes:
1. The CT disclosure text (TLS-04 cross-reference: "Your hostname will be visible in Certificate Transparency logs")
2. Steps to enable HTTPS: tailscale.com/admin → DNS → Enable HTTPS
3. A "Check Again" button that calls `onCheckAgain` prop

### Pattern 4: TailscaleStatusIndicator in SettingsPanel

**What:** Read-only status display in the Web Server tab replacing the removed VPN interface picker. Shows a colored dot + label based on `TailscaleHealth` state.

**When to use:** Always visible in Web Server tab when panel is open. Not interactive — information only.

**Props change to SettingsPanel:**
```typescript
interface SettingsPanelProps {
  isOpen: boolean
  onClose: () => void
  clis: DetectedCLI[]
  tailscaleHealth: TailscaleHealth | null   // NEW
}
```

**Example JSX in Web Server tab:**
```typescript
{/* Tailscale Status Indicator */}
<div className="settings-panel__field-group">
  <label className="settings-panel__label">Tailscale Status</label>
  <div className="ts-status">
    <span className={`ts-status__dot ts-status__dot--${tailscaleStatusClass(tailscaleHealth)}`} />
    <span className="ts-status__text">{tailscaleStatusText(tailscaleHealth)}</span>
  </div>
</div>
```

Where `tailscaleStatusClass` returns `'ok' | 'warn' | 'error'` and `tailscaleStatusText` returns `'Connected' | 'Not Connected' | 'Not Installed'`.

### Pattern 5: `?raw` Source-Inspection Tests

**What:** The existing project test pattern uses `import raw from '../Component.tsx?raw'` and inspects source text with `expect(raw).toContain(...)`. This avoids jsdom Canvas/WebGL constraints and is stable against internal implementation changes.

**When to use:** All new `.test.tsx` files in this phase. Tests verify component structure, required props, CSS class names, and JSX conditionals — not runtime DOM behavior.

**Example for HealthModal.test.tsx:**
```typescript
// Source: existing pattern from App.test.tsx, SettingsPanel.test.tsx
import { describe, it, expect } from 'vitest'
import raw from '../HealthModal.tsx?raw'

describe('HealthModal', () => {
  it('imports GetTailscaleStatus from wailsjs', () => {
    // HealthModal receives health as prop; App.tsx imports GetTailscaleStatus
    // Test App.tsx instead for wiring
  })

  it('renders NotInstalledPanel when !health.installed', () => {
    expect(raw).toContain('!isInstalled')
    expect(raw).toContain('NotInstalledPanel')
  })

  it('renders NotConnectedPanel when installed but !connected', () => {
    expect(raw).toContain('isInstalled && !isConnected')
    expect(raw).toContain('NotConnectedPanel')
  })

  it('renders NoCertsPanel when connected but !hasCerts', () => {
    expect(raw).toContain('isInstalled && isConnected && !hasCerts')
    expect(raw).toContain('NoCertsPanel')
  })

  it('NoCertsPanel includes onCheckAgain button', () => {
    expect(raw).toContain('onCheckAgain')
  })

  it('platform prop drives platform-specific content', () => {
    expect(raw).toContain("platform === 'darwin'")
    expect(raw).toContain("platform === 'linux'")
    expect(raw).toContain("platform === 'windows'")
  })

  it('includes CT disclosure text in NoCertsPanel', () => {
    expect(raw).toContain('Certificate Transparency')
  })
})
```

For SettingsPanel updates, add tests to the existing `SettingsPanel.test.tsx`:
```typescript
// Source-inspection addition
import rawSettings from '../SettingsPanel.tsx?raw'

it('Web Server tab contains Tailscale Status label', () => {
  expect(rawSettings).toContain('Tailscale Status')
})

it('SettingsPanel accepts tailscaleHealth prop', () => {
  expect(rawSettings).toContain('tailscaleHealth')
})
```

### Anti-Patterns to Avoid

- **Calling `GetTailscaleStatus()` from HealthModal directly:** The modal receives health as props from App.tsx. Components don't call Wails bindings — App.tsx owns all backend calls.
- **Blocking app init on `GetTailscaleStatus()`:** Include it in the existing `Promise.all` in the init `useEffect`. Health state `null` = loading — modal does not render while null.
- **Using `Environment()` inside HealthModal:** Call `Environment()` once in App.tsx on init. Platform string is stable for the app's lifetime — no need to re-fetch.
- **Showing the modal while health is null (loading):** The modal must return `null` when `health === null` to avoid a flash during app initialization.
- **Combining all three platform instruction strings in one JSX block without conditionals:** Each platform must have clearly distinct JSX branches — not a single string with embedded platform names. Tests verify the conditional structure.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Platform detection | Custom `navigator.userAgent` parsing or Go `runtime.GOOS` binding | Wails `Environment().platform` | Already available in `runtime.d.ts`; returns `"darwin"/"linux"/"windows"` — no new Go code |
| Live health updates | JS-side `setInterval` polling `GetTailscaleStatus()` | `EventsOn('tailscale:health', ...)` | Backend already pushes changes via `startHealthPoller` every 10s; event-driven avoids duplicate polling |
| Modal overlay CSS | New overlay system | Reuse `.settings-overlay` / `.qr-modal-overlay` pattern | Three existing modals already have the same overlay pattern; copy it |
| Status dot colors | Custom color management | Extend existing `.tab__status--*` dot pattern in CSS | Dots with colored border-radius already styled; add `ts-status__dot--ok/warn/error` variants |

**Key insight:** The entire Phase 14 backend is already live — `GetTailscaleStatus()` works, events fire, the struct is fully typed. Phase 18 is "connect the wires on the frontend side."

---

## Common Pitfalls

### Pitfall 1: Health Modal Appears Instantly on App Start (Flicker)

**What goes wrong:** Health state is `null` until `GetTailscaleStatus()` resolves. If the modal renders before the Promise resolves, users see a flash of the "not installed" panel even when Tailscale is healthy.

**Why it happens:** `GetTailscaleStatus()` has a 5-second timeout (set in Go) but usually resolves in <100ms when tailscaled is running. If the component renders before state is populated, `health === null` or all flags default to false.

**How to avoid:** Modal renders `null` when `health === null`. Use `null` as the initial state (not `{ installed: false, ... }`). The `Promise.all` in init ensures health is set alongside other initial state.

**Warning signs:** In testing, mock `GetTailscaleStatus` to return a resolved promise with healthy status; verify no modal appears.

### Pitfall 2: SettingsPanel Test Breakage — Props Interface Changed

**What goes wrong:** `SettingsPanel.test.tsx` creates `SettingsPanelProps` inline (it re-declares the interface). Adding `tailscaleHealth` prop to the real component without updating the test's inline interface causes TypeScript errors.

**Why it happens:** The test file has its own `interface SettingsPanelProps` declaration (lines 17-21) separate from the component. Both must stay in sync.

**How to avoid:** When adding `tailscaleHealth: TailscaleHealth | null` to `SettingsPanel.tsx`'s props, also add it to the test's inline interface. Use `vi.fn()` mock return value `null` as the default for tests that don't exercise the new UI.

**Warning signs:** `vitest run` fails with TypeScript type error on the `renderSettingsPanel` helper.

### Pitfall 3: `Environment()` is Async — Platform Available After Mount

**What goes wrong:** Rendering platform-specific instructions synchronously before `Environment()` resolves shows the wrong platform's content briefly.

**Why it happens:** `Environment()` returns a Promise. Platform defaults to `'linux'` (safe fallback) until it resolves.

**How to avoid:** Default `platform` state to `'linux'` (most common non-interactive CLI platform). `Environment()` resolves quickly (< 1ms — Wails synchronous bridge call on the injected `window.runtime`). In `Promise.all` alongside other init calls, it resolves before first meaningful render. No special handling needed.

**Warning signs:** Tests that assert platform-specific text should mock `Environment` to return a specific platform.

### Pitfall 4: CT Disclosure Text Must Be in NoCertsPanel (TLS-04 Cross-Reference)

**What goes wrong:** Cert-enablement instructions in the health modal don't show the CT disclosure before the user enables certs, violating TLS-04.

**Why it happens:** Phase 15 put CT disclosure in the SettingsPanel Web Server tab (before "Start Web Server"). But a user following the NoCertsPanel steps (enabling HTTPS in admin console → certs become available → server starts automatically) bypasses the SettingsPanel flow.

**How to avoid:** NoCertsPanel must include the CT disclosure statement: "When you enable HTTPS, Tailscale will provision a Let's Encrypt certificate. Your device hostname will be permanently visible in public Certificate Transparency logs." This is informational text — no checkbox needed. The existing CT acknowledgement persisted to disk (via `HasCTDisclosure`/`AcknowledgeCTDisclosure`) covers the SettingsPanel flow; the health modal text is a disclosure, not a gate.

**Warning signs:** Success criterion 4 explicitly requires: "The Certificate Transparency disclosure is visible in the modal before any cert provisioning attempt."

### Pitfall 5: `tailscale:health` Event Listener Not Cleaned Up

**What goes wrong:** Memory leak / stale closure updates state after component unmounts.

**Why it happens:** `EventsOn` returns an unsubscribe function. If the returned `offHealth` is not called in the `useEffect` cleanup return, the listener accumulates.

**How to avoid:** Same pattern as the existing `offStatus` cleanup in `App.tsx`:
```typescript
return () => {
  offStatus()
  offHealth()
}
```

**Warning signs:** React strict mode double-mount warnings; multiple state updates from a single event.

### Pitfall 6: SettingsPanel Existing Tests Fail Due to Mock Scope

**What goes wrong:** `SettingsPanel.test.tsx` uses `vi.mock('../../wailsjs/go/main/App', ...)`. Adding `GetTailscaleStatus` to the mock scope is only needed if SettingsPanel calls it directly. It does not — health is passed as a prop. No mock change needed.

**Why it happens:** Confusion about whether the indicator calls backend directly.

**How to avoid:** `TailscaleStatusIndicator` is purely presentational — receives `tailscaleHealth` as prop, makes no Wails calls. No new mocks needed in SettingsPanel tests.

---

## Code Examples

Verified patterns from codebase inspection:

### Wails `Environment()` Call (platform detection)
```typescript
// Source: frontend/src/wailsjs/wailsjs/runtime/runtime.d.ts:30-32
// Environment(): Promise<EnvironmentInfo>  where EnvironmentInfo.platform: string
// Values: "darwin" | "linux" | "windows"
import { Environment } from './wailsjs/wailsjs/runtime/runtime'
const env = await Environment()
// env.platform === 'darwin' on macOS, 'linux' on Linux, 'windows' on Windows
```

### GetTailscaleStatus Binding (already exists)
```typescript
// Source: frontend/src/wailsjs/go/main/App.d.ts:44-50
export function GetTailscaleStatus(): Promise<{
  installed: boolean
  connected: boolean
  hasCerts: boolean
  ip: string
  domain: string
}>
```

### tailscale:health Event Subscription
```typescript
// Source: mirrors session:status pattern in App.tsx:91-96
// Event payload shape matches TailscaleHealth struct JSON tags:
// { installed: bool, connected: bool, hasCerts: bool, ip: string, domain: string }
const offHealth = EventsOn('tailscale:health', (h: TailscaleHealth) => {
  setTailscaleHealth(h)
})
// cleanup: return () => { offStatus(); offHealth() }
```

### Extending Promise.all in App.tsx init
```typescript
// Source: App.tsx:54-58 (existing pattern to extend)
const [port, clis, sessions, running, health, env] = await Promise.all([
  GetRelayPort(),
  DetectCLIs(),
  ListSessions(),
  IsWebServerRunning(),
  GetTailscaleStatus(),   // ADD
  Environment(),          // ADD
])
setTailscaleHealth(health)
setPlatform(env.platform)
```

### CSS Status Dot Pattern (extend existing)
```css
/* Source: style.css existing tab status dots pattern */
.ts-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}
.ts-status__dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.ts-status__dot--ok    { background: #9ece6a; }  /* green — matches tab--idle */
.ts-status__dot--warn  { background: #f59e0b; }  /* amber — matches tab--waiting */
.ts-status__dot--error { background: #f7768e; }  /* red — matches tab--errored */
```

### Modal Overlay CSS Pattern (reuse)
```css
/* Source: style.css .settings-overlay and .qr-modal-overlay patterns */
.health-modal-overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.health-modal {
  background-color: #1e2030;
  border: 1px solid #292e42;
  border-radius: 8px;
  width: 520px;
  max-width: 95vw;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| VPN interface picker in Settings → Web Server tab | Tailscale status indicator (read-only) | Phase 17 removed picker, Phase 18 adds indicator | Settings tab now shows health rather than a removed control |
| No health modal — user had to figure out Tailscale issues | `HealthModal` with platform-specific instructions | Phase 18 | Clear user guidance before they attempt to start the web server |
| CT disclosure only gated in SettingsPanel | CT disclosure also present in HealthModal NoCertsPanel | Phase 18 | Ensures disclosure appears in both flows where cert provisioning is triggered |

**Already in place from prior phases:**
- `GetTailscaleStatus()` Wails binding: Phase 14
- `tailscale:health` event emission (10s poll): Phase 14
- `TailscaleHealth` TypeScript type in `App.d.ts`: Phase 14
- CT disclosure persistence (`HasCTDisclosure` / `AcknowledgeCTDisclosure`): Phase 15

---

## Open Questions

1. **Should `HealthModal` be dismissable?**
   - What we know: Success criteria do not mention a dismiss/close button; the modal describes a hard dependency (Tailscale certs required for web serving)
   - What's unclear: Whether users might find a non-dismissable modal frustrating when they have started Tailscale and are waiting for the 10s poll to fire
   - Recommendation: Make it non-dismissable (no X / Close button) — the user resolves the problem and the modal disappears automatically via the `tailscale:health` event. The "Check Again" button in NoCertsPanel provides the only manual action. This is the pattern for critical configuration requirements.

2. **When does HealthModal first appear — on startup or only when user tries to use web serving?**
   - What we know: HEALTH-04 says "when any health check fails" without specifying trigger. Phase 14's poller starts immediately on `startup()`.
   - What's unclear: Whether showing the modal on cold start (before user tries anything) is desired UX
   - Recommendation: Show modal on startup if health is already failed — consistent with how app communicates errors elsewhere. `GetTailscaleStatus()` runs in the init `Promise.all`, so health state is known before first render with real data. The modal simply renders whenever `!healthOk` is true.

3. **Should `GetTailscaleStatus()` be called on HealthModal "Check Again" or wait for next poll?**
   - What we know: The "Check Again" button is called out explicitly in success criterion 3
   - What's unclear: Whether to call `GetTailscaleStatus()` imperatively or trigger via poll timeout
   - Recommendation: "Check Again" calls `GetTailscaleStatus()` directly and updates state immediately via `setTailscaleHealth()` — no need to wait 10s for next poll tick. The App-level `handleCheckHealthAgain` callback does this call.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 (existing) |
| Config file | `frontend/vite.config.ts` — `test.environment: 'jsdom'` |
| Quick run command | `cd frontend && pnpm test` (runs `vitest run`) |
| Full suite command | `cd frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HEALTH-04 | HealthModal renders when health is unhealthy | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | Wave 0 |
| HEALTH-04 | Three distinct panels: NotInstalled, NotConnected, NoCerts | source-inspection | same | Wave 0 |
| HEALTH-04 | NoCertsPanel contains "Check Again" button (`onCheckAgain`) | source-inspection | same | Wave 0 |
| HEALTH-04 | CT disclosure text present in NoCertsPanel | source-inspection | same | Wave 0 |
| HEALTH-04 | App.tsx subscribes to `tailscale:health` event | source-inspection | same | Wave 0 |
| HEALTH-04 | App.tsx calls `GetTailscaleStatus()` on init | source-inspection | same | Wave 0 |
| HEALTH-04 | SettingsPanel receives and renders `tailscaleHealth` prop | source-inspection | same | Wave 0 |
| HEALTH-05 | HealthModal uses `platform` prop with darwin/linux/windows branches | source-inspection | same | Wave 0 |
| HEALTH-05 | App.tsx calls `Environment()` and stores platform | source-inspection | same | Wave 0 |

### Sampling Rate

- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/HealthModal.tsx` — new component file (covered by tests once created)
- [ ] `frontend/src/components/__tests__/HealthModal.test.tsx` — `?raw` source-inspection tests for HEALTH-04 and HEALTH-05
- [ ] Updates to `frontend/src/components/__tests__/SettingsPanel.test.tsx` — add tests for `tailscaleHealth` prop and "Tailscale Status" label in Web Server tab
- [ ] Updates to `frontend/src/components/__tests__/App.test.tsx` — add tests verifying `GetTailscaleStatus()` call, `Environment()` call, `tailscale:health` event subscription, and `HealthModal` rendered in JSX

*(No new test framework needed — existing vitest infrastructure covers all requirements.)*

---

## Sources

### Primary (HIGH confidence — direct codebase inspection)

- `/Users/ken/dev/agenthub/frontend/src/wailsjs/go/main/App.d.ts` — `GetTailscaleStatus` binding, `TailscaleHealth` shape
- `/Users/ken/dev/agenthub/frontend/src/wailsjs/go/main/App.js` — `GetTailscaleStatus` Wails Call binding
- `/Users/ken/dev/agenthub/frontend/src/wailsjs/wailsjs/runtime/runtime.d.ts` — `Environment()` signature, `EnvironmentInfo.platform` type
- `/Users/ken/dev/agenthub/frontend/src/App.tsx` — existing `EventsOn` pattern, `Promise.all` init pattern, modal rendering pattern
- `/Users/ken/dev/agenthub/frontend/src/components/SettingsPanel.tsx` — existing Web Server tab structure, props interface, CT disclosure banner
- `/Users/ken/dev/agenthub/frontend/src/components/QRModal.tsx` — modal overlay pattern to replicate
- `/Users/ken/dev/agenthub/frontend/src/style.css` — existing CSS patterns: dots (`.tab__status--*`), overlays (`.settings-overlay`, `.qr-modal-overlay`), disclosure (`.ct-disclosure`)
- `/Users/ken/dev/agenthub/frontend/src/components/__tests__/SettingsPanel.test.tsx` — `?raw` + `createRoot`/`flushSync` test pattern; inline `SettingsPanelProps` that needs updating
- `/Users/ken/dev/agenthub/frontend/src/components/__tests__/App.test.tsx` — `?raw` source-inspection test pattern
- `/Users/ken/dev/agenthub/app.go` — `startHealthPoller` implementation; event emission as `tailscale:health`
- `/Users/ken/dev/agenthub/.planning/STATE.md` — Phase 14/15 decisions; CT disclosure implementation

### Secondary (MEDIUM confidence)

- `/Users/ken/dev/agenthub/.planning/REQUIREMENTS.md` — HEALTH-04, HEALTH-05 requirement text; TLS-04 CT disclosure requirement
- `/Users/ken/dev/agenthub/.planning/phases/14-tailscale-health-check-infrastructure/14-RESEARCH.md` — TailscaleHealth field semantics; three-state decision table

---

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — all required libraries already present and in use; `Environment()` verified in runtime.d.ts
- Architecture: HIGH — all backend contracts are live (Phase 14 complete); frontend patterns copied from existing modals
- Pitfalls: HIGH — derived from direct inspection of code that will be modified (SettingsPanel.test.tsx inline interface, EventsOn cleanup pattern, CT disclosure requirement)

**Research date:** 2026-03-22
**Valid until:** 2026-04-22 (no new dependencies; valid until next Wails or React major version change)
