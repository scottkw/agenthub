# Settings Modal 3-Tab Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the 2-tab settings modal into 3 tabs (CLI Paths, Web Server, Security) by extracting password and certificate content from the former "Web Serving" tab.

**Architecture:** Pure JSX refactoring — no new state, no new handlers, no new CSS classes. The existing `activeTab` union type gains a third value, the tab bar gains a third button, and the old `web-serving` conditional block is split into `web-server` and `security` blocks. Tests updated to match.

**Tech Stack:** React, TypeScript, Vitest

**Spec:** `docs/superpowers/specs/2026-03-19-settings-modal-3-tab-design.md`

---

### Task 1: Update tests for 3-tab structure (RED phase)

**Files:**
- Modify: `frontend/src/components/__tests__/SettingsPanel.test.tsx`

- [ ] **Step 1: Read the existing test file**

Read `frontend/src/components/__tests__/SettingsPanel.test.tsx` to confirm current state matches plan expectations (8 tests, `renderSettingsPanel` helper, Wails mocks).

- [ ] **Step 2: Rewrite the test file with updated + new tests**

Replace the entire contents of `frontend/src/components/__tests__/SettingsPanel.test.tsx` with:

```typescript
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { SettingsPanel } from '../SettingsPanel'

vi.mock('../../wailsjs/go/main/App', () => ({
  UpdateCLIPath: vi.fn(),
  SetWebPassword: vi.fn(),
  IsWebPasswordSet: vi.fn().mockResolvedValue(false),
  GetNetworkInterfaces: vi.fn().mockResolvedValue([]),
  StartWebServer: vi.fn(),
  StopWebServer: vi.fn(),
  GetWebServerURL: vi.fn().mockResolvedValue(''),
  GetCACertPath: vi.fn().mockResolvedValue('/fake/ca.crt'),
  IsWebServerRunning: vi.fn().mockResolvedValue(false),
}))

interface SettingsPanelProps {
  isOpen: boolean
  onClose: () => void
  clis: Array<{ Name: string; Path: string }>
}

function renderSettingsPanel(props: Partial<SettingsPanelProps> = {}) {
  const defaults: SettingsPanelProps = {
    isOpen: true,
    onClose: vi.fn(),
    clis: [{ Name: 'claude', Path: '/usr/bin/claude' }],
  }
  const merged = { ...defaults, ...props }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(SettingsPanel, merged as any))
  })
  return { container, root }
}

function clickTabByText(container: HTMLElement, text: string) {
  const buttons = container.querySelectorAll('.settings-panel__tab-btn')
  const btn = Array.from(buttons).find((b) => b.textContent?.trim() === text)
  expect(btn).not.toBeUndefined()
  flushSync(() => {
    btn!.click()
  })
}

describe('SettingsPanel', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders three tab buttons: "CLI Paths", "Web Server", "Security"', () => {
    ;({ container, root } = renderSettingsPanel())
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    expect(tabs.length).toBe(3)
    const tabTexts = Array.from(tabs).map((t) => t.textContent?.trim())
    expect(tabTexts).toEqual(['CLI Paths', 'Web Server', 'Security'])
  })

  it('CLI Paths tab button has active class on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    const cliTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'CLI Paths')
    expect(cliTab?.classList.contains('settings-panel__tab-btn--active')).toBe(true)
    expect(cliTab?.getAttribute('aria-selected')).toBe('true')
  })

  it('CLI Paths content is visible on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const table = container.querySelector('.settings-panel__table')
    expect(table).not.toBeNull()
  })

  it('Web Server content is NOT in the DOM on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const selects = container.querySelectorAll('.settings-panel__select')
    expect(selects.length).toBe(0)
  })

  it('Security content is NOT in the DOM on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(0)
  })

  it('clicking Web Server tab shows network interface and port, hides CLI table', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Web Server')
    // Web Server content present
    const description = container.querySelector('.settings-panel__description')
    expect(description?.textContent).toContain('HTTPS access')
    // CLI Paths content gone
    const table = container.querySelector('.settings-panel__table')
    expect(table).toBeNull()
    // Password not present (that's Security tab)
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(0)
  })

  it('clicking Security tab shows password field and hides Web Server and CLI content', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Security')
    // Password field present
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(1)
    // CLI table gone
    const table = container.querySelector('.settings-panel__table')
    expect(table).toBeNull()
    // Network select gone
    const selects = container.querySelectorAll('.settings-panel__select')
    expect(selects.length).toBe(0)
  })

  it('Security tab button has aria-selected="true" when active', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Security')
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    const securityTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'Security')
    expect(securityTab?.getAttribute('aria-selected')).toBe('true')
    // Other tabs should not be selected
    const cliTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'CLI Paths')
    expect(cliTab?.getAttribute('aria-selected')).toBe('false')
  })

  it('footer contains exactly one button with text "Close"', () => {
    ;({ container, root } = renderSettingsPanel())
    const footer = container.querySelector('.settings-panel__footer')
    expect(footer).not.toBeNull()
    const footerButtons = footer!.querySelectorAll('button')
    expect(footerButtons.length).toBe(1)
    expect(footerButtons[0].textContent?.trim()).toBe('Close')
  })

  it('Close button has class settings-panel__btn--cancel (secondary style)', () => {
    ;({ container, root } = renderSettingsPanel())
    const footer = container.querySelector('.settings-panel__footer')
    const closeBtn = footer?.querySelector('button')
    expect(closeBtn?.classList.contains('settings-panel__btn--cancel')).toBe(true)
  })

  it('CLI Paths tab contains a "Save Paths" button inline, not in footer', () => {
    ;({ container, root } = renderSettingsPanel())
    const footer = container.querySelector('.settings-panel__footer')
    const body = container.querySelector('.settings-panel__body')
    const footerBtnTexts = Array.from(footer!.querySelectorAll('button')).map((b) => b.textContent?.trim())
    expect(footerBtnTexts).not.toContain('Save Paths')
    const savePathsBtn = Array.from(body!.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Save Paths')
    expect(savePathsBtn).not.toBeUndefined()
  })

  it('Security tab shows CA certificate path', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Security')
    const codeBlocks = container.querySelectorAll('.settings-panel__code')
    const certCode = Array.from(codeBlocks).find((c) => c.textContent?.includes('ca.crt'))
    expect(certCode).not.toBeUndefined()
  })
})
```

- [ ] **Step 3: Run tests to verify they fail (RED)**

Run: `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsPanel 2>&1 | tail -30`

Expected: Multiple failures — component still has 2 tabs with "Web Serving" text, h3 headings present, no "Security" tab.

- [ ] **Step 4: Commit RED tests**

```bash
git add frontend/src/components/__tests__/SettingsPanel.test.tsx
git commit -m "test: update SettingsPanel tests for 3-tab structure (RED)"
```

---

### Task 2: Refactor SettingsPanel to 3-tab layout (GREEN phase)

**Files:**
- Modify: `frontend/src/components/SettingsPanel.tsx`

- [ ] **Step 1: Read the current component**

Read `frontend/src/components/SettingsPanel.tsx` to confirm current state.

- [ ] **Step 2: Update tab state union type**

Change line 27 from:
```typescript
const [activeTab, setActiveTab] = useState<'cli-paths' | 'web-serving'>('cli-paths')
```
to:
```typescript
const [activeTab, setActiveTab] = useState<'cli-paths' | 'web-server' | 'security'>('cli-paths')
```

- [ ] **Step 3: Update tab bar — rename Web Serving and add Security tab**

Replace the entire `settings-panel__tabs` div (lines 168-185) with:

```tsx
<div className="settings-panel__tabs" role="tablist">
  <button
    className={`settings-panel__tab-btn ${activeTab === 'cli-paths' ? 'settings-panel__tab-btn--active' : ''}`}
    onClick={() => setActiveTab('cli-paths')}
    role="tab"
    aria-selected={activeTab === 'cli-paths'}
  >
    CLI Paths
  </button>
  <button
    className={`settings-panel__tab-btn ${activeTab === 'web-server' ? 'settings-panel__tab-btn--active' : ''}`}
    onClick={() => setActiveTab('web-server')}
    role="tab"
    aria-selected={activeTab === 'web-server'}
  >
    Web Server
  </button>
  <button
    className={`settings-panel__tab-btn ${activeTab === 'security' ? 'settings-panel__tab-btn--active' : ''}`}
    onClick={() => setActiveTab('security')}
    role="tab"
    aria-selected={activeTab === 'security'}
  >
    Security
  </button>
</div>
```

- [ ] **Step 4: Remove h3 from CLI Paths block**

In the `activeTab === 'cli-paths'` block, remove the line:
```tsx
<h3>CLI Paths</h3>
```

- [ ] **Step 5: Replace the web-serving block with web-server and security blocks**

Replace the entire `{activeTab === 'web-serving' && ( ... )}` block (lines 237-349) with two new blocks:

**Web Server block:**
```tsx
{activeTab === 'web-server' && (
  <>
    <p className="settings-panel__description">
      Enable HTTPS access to terminal sessions from remote browsers.
    </p>

    {/* Network Interface Selector */}
    <div className="settings-panel__field-group">
      <label className="settings-panel__label">Network Interface</label>
      {networkInterfaces.length === 0 ? (
        <p className="settings-panel__empty">No non-loopback interfaces found.</p>
      ) : (
        <select
          className="settings-panel__select"
          value={selectedInterface}
          onChange={(e) => setSelectedInterface(e.target.value)}
          disabled={isServerRunning}
        >
          {networkInterfaces.map((iface) => (
            <option key={`${iface.Name}-${iface.IP}`} value={iface.IP}>
              {iface.Name} ({iface.IP}){iface.IsTailscale ? ' — Tailscale' : ''}
            </option>
          ))}
        </select>
      )}
    </div>

    {/* Port */}
    <div className="settings-panel__field-group">
      <label className="settings-panel__label">Port</label>
      <input
        className="settings-panel__path-input settings-panel__port-input"
        type="number"
        value={selectedPort}
        onChange={(e) => setSelectedPort(Number(e.target.value))}
        disabled={isServerRunning}
        min={1}
        max={65535}
      />
    </div>

    {/* Start/Stop Server */}
    <div className="settings-panel__field-group">
      <button
        className={`settings-panel__btn ${isServerRunning ? 'settings-panel__btn--cancel' : 'settings-panel__btn--save'}`}
        onClick={handleToggleServer}
        disabled={serverLoading || (!isServerRunning && !isPasswordSet)}
        title={!isPasswordSet && !isServerRunning ? 'Set a password in the Security tab first' : undefined}
      >
        {serverLoading
          ? (isServerRunning ? 'Stopping…' : 'Starting…')
          : (isServerRunning ? 'Stop Web Server' : 'Start Web Server')}
      </button>
      {serverError && <p className="settings-panel__error">{serverError}</p>}
      {isServerRunning && serverURL && (
        <p className="settings-panel__url">
          Server running at: <a href={serverURL} target="_blank" rel="noreferrer">{serverURL}</a>
        </p>
      )}
    </div>
  </>
)}
```

**Security block:**
```tsx
{activeTab === 'security' && (
  <>
    {/* Password Setup */}
    <div className="settings-panel__field-group">
      <label className="settings-panel__label">
        Dashboard Password
        {isPasswordSet && (
          <span className="settings-panel__check" title="Password is set"> ✓</span>
        )}
      </label>
      <div className="settings-panel__row">
        <input
          className="settings-panel__path-input"
          type="password"
          value={webPassword}
          onChange={(e) => setWebPassword(e.target.value)}
          placeholder={isPasswordSet ? 'Change password…' : 'Set a password to enable web serving'}
          onKeyDown={(e) => { if (e.key === 'Enter') void handleSetPassword() }}
        />
        <button
          className="settings-panel__btn settings-panel__btn--save"
          onClick={handleSetPassword}
          disabled={passwordSaving || !webPassword.trim()}
        >
          {passwordSaving ? 'Saving…' : 'Set Password'}
        </button>
      </div>
      {passwordError && <p className="settings-panel__error">{passwordError}</p>}
    </div>

    <hr style={{ border: 'none', borderTop: '1px solid #292e42', margin: '20px 0' }} />

    {/* CA Certificate Guidance */}
    {caCertPath && (
      <div className="settings-panel__field-group">
        <label className="settings-panel__label">CA Certificate</label>
        <p className="settings-panel__description">
          To avoid browser security warnings, install the local CA cert:
        </p>
        <code className="settings-panel__code">{caCertPath}</code>
        <details className="settings-panel__details">
          <summary>Installation instructions</summary>
          <pre className="settings-panel__code settings-panel__code--block">
            {getCACertInstructions()}
          </pre>
          <p className="settings-panel__description">
            After installation, restart your browser and refresh the page.
            The CA cert can also be downloaded directly from the server at{' '}
            <code>/ca.crt</code>.
          </p>
        </details>
      </div>
    )}
  </>
)}
```

- [ ] **Step 6: Run tests to verify they pass (GREEN)**

Run: `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsPanel 2>&1 | tail -30`

Expected: All 12 tests pass.

- [ ] **Step 7: Run full test suite for regressions**

Run: `cd /Users/ken/dev/agenthub/frontend && pnpm test 2>&1 | tail -10`

Expected: All tests pass (existing + updated).

- [ ] **Step 8: Commit GREEN implementation**

```bash
git add frontend/src/components/SettingsPanel.tsx
git commit -m "feat: split settings modal into 3 tabs (CLI Paths, Web Server, Security)"
```
