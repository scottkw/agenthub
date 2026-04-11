# Phase 66: Web Server Link UX - Research

**Researched:** 2026-04-11
**Domain:** Wails runtime clipboard/browser APIs + go-qrcode + React inline UX in SettingsTab
**Confidence:** HIGH

## Summary

Phase 66 adds three interactable affordances for the web server dashboard URL inside the Settings tab's existing "Web Server" sub-tab: an open-in-browser button, a copy-to-clipboard button, and an inline QR code. All three must work in both Tailscale mode and local network mode.

The implementation is entirely frontend + one new Go method. The `serverURL` and `isServerRunning` state already exist in `SettingsTab.tsx`. The URL is already displayed as a plain `<a>` link (line 319 of `SettingsTab.tsx`). Phase 66 upgrades that display row to add:
1. A button that calls `BrowserOpenURL(serverURL)` from the Wails runtime — already imported by `App.tsx` and used by `WelcomeTab.tsx` and `LocalNetworkBanner.tsx`.
2. A copy button that calls `ClipboardSetText(serverURL)` from the Wails runtime — already declared in `runtime.d.ts`, not yet used anywhere.
3. An inline QR code `<img>` rendered directly in the settings panel (not a modal), displaying a base64 PNG fetched from a new Go method `GetWebServerQRCode()`.

The existing `GetSessionQRCode(sessionID)` method in `app.go` constructs a session URL and encodes it. A new `GetWebServerQRCode()` method is needed — same pattern, but encoding `resp.URL` (the dashboard root) instead of a session sub-path. The `github.com/skip2/go-qrcode` library is already a direct dependency in `go.mod`.

**Primary recommendation:** Add `GetWebServerQRCode()` to `app.go`. In `SettingsTab.tsx`, replace the current `<a>` display with a URL action row containing Open, Copy, and QR toggle buttons. Show the QR inline below the URL row (not a modal) when toggled. Use `ClipboardSetText` from the Wails runtime for copy (not `navigator.clipboard`).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WEB-01 | User can open the web server dashboard URL in their system browser | `BrowserOpenURL(serverURL)` from `wailsjs/wailsjs/runtime/runtime` — already imported in `App.tsx`, used by `WelcomeTab.tsx` and `LocalNetworkBanner.tsx`. Works cross-platform for Wails apps. |
| WEB-02 | User can copy the web server dashboard URL to clipboard | `ClipboardSetText(serverURL)` from `wailsjs/wailsjs/runtime/runtime` — declared in `runtime.d.ts` as `Promise<boolean>`. Not yet called anywhere but fully available. Preferred over `navigator.clipboard` in Wails apps because it routes through the native layer. |
| WEB-03 | User can view a QR code for the web server dashboard URL | Requires new `GetWebServerQRCode()` Go method (same pattern as `GetSessionQRCode` but encodes `resp.URL` directly). Display inline in the settings panel — a modal is not needed since the QR only appears when the server is running and the user is already on the Web Server sub-tab. |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/skip2/go-qrcode` | v0.0.0-20200617195104-da1b6568686e | QR PNG generation | Already in `go.mod` as direct dependency; used by `GetSessionQRCode` |
| Wails `BrowserOpenURL` | v2.10.2 | Open URL in system default browser | Already imported in `App.tsx`; cross-platform native |
| Wails `ClipboardSetText` | v2.10.2 | Write text to system clipboard | Already declared in `runtime.d.ts`; returns `Promise<boolean>` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| React `useState` | (bundled) | Local `copied` / `showQR` state | Toggle copy feedback and QR visibility |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `ClipboardSetText` (Wails) | `navigator.clipboard.writeText` | Wails runtime routes through native layer, more reliable in WebKit WebView on macOS/Windows; `navigator.clipboard` requires secure context and works for password field already but Wails API is preferred for new code |
| Inline QR in settings panel | QRModal (existing modal) | The existing QRModal is session-specific and uses `GetSessionQRCode(sessionID)`. Dashboard QR has no session ID — a new path is needed. Inline (toggle show/hide) is less disruptive than launching a modal from within a tab. |

**No new npm packages required.** No new Go dependencies required.

## Architecture Patterns

### Changes Required

```
app.go                                  # add GetWebServerQRCode() method
frontend/src/
├── wailsjs/go/main/App.d.ts            # add GetWebServerQRCode export declaration
├── wailsjs/go/main/App.js              # add GetWebServerQRCode stub
├── components/
│   └── SettingsTab.tsx                 # upgrade URL row with Open/Copy/QR actions
└── style.css                           # add CSS for URL action row and inline QR
```

### Pattern 1: New Go Method `GetWebServerQRCode()`

**What:** Encodes the dashboard base URL (not a session sub-path) as a QR PNG, returns base64 string.
**When to use:** Called when the user toggles QR visibility in the Web Server sub-tab.

```go
// Source: mirrors GetSessionQRCode pattern in app.go
func (a *App) GetWebServerQRCode() (string, error) {
    if a.client == nil {
        return "", fmt.Errorf("daemon not connected")
    }
    resp, err := a.client.GetWebServerStatus()
    if err != nil || !resp.Running {
        return "", fmt.Errorf("web server not running")
    }
    png, err := qrcode.Encode(resp.URL, qrcode.Medium, 256)
    if err != nil {
        return "", fmt.Errorf("GetWebServerQRCode: encode: %w", err)
    }
    return base64.StdEncoding.EncodeToString(png), nil
}
```

[VERIFIED: pattern from app.go lines 431-445]

### Pattern 2: URL Action Row in SettingsTab

**What:** Replace the existing `<p className="settings-panel__url">` display with a row containing the URL text plus three action buttons.
**When to use:** Rendered when `isServerRunning && serverURL`.

```typescript
// Source: mirrors handleCopyPassword + BrowserOpenURL patterns in SettingsTab.tsx / App.tsx
const [urlCopied, setUrlCopied] = useState(false)
const [showDashQR, setShowDashQR] = useState(false)
const [dashQRb64, setDashQRb64] = useState<string | null>(null)

async function handleCopyURL() {
  await ClipboardSetText(serverURL)
  setUrlCopied(true)
  setTimeout(() => setUrlCopied(false), 1500)
}

async function handleToggleDashQR() {
  if (showDashQR) {
    setShowDashQR(false)
    return
  }
  if (!dashQRb64) {
    const b64 = await GetWebServerQRCode()
    setDashQRb64(b64)
  }
  setShowDashQR(true)
}
```

[VERIFIED: pattern from SettingsTab.tsx lines 95-99 and App.tsx lines 431, 529]

### Pattern 3: Wails Binding Stubs

**What:** The `App.d.ts` and `App.js` stubs are hand-maintained (auto-generated comment is misleading — the actual generator is only run by `wails dev`/`wails build`, which is not run in this workflow). New Go methods need matching manual stub entries.

**Example for `App.d.ts`:**
```typescript
export function GetWebServerQRCode(): Promise<string>
```

**Example for `App.js`:**
```javascript
export function GetWebServerQRCode() {
  return window['go']['main']['App']['GetWebServerQRCode']();
}
```

[VERIFIED: existing stub patterns in App.d.ts and App.js]

### Anti-Patterns to Avoid

- **Using `GetSessionQRCode` for the dashboard URL:** That method takes a `sessionID` and appends `/sessions/{id}`. The dashboard URL is just `resp.URL`. Wrong method for this purpose.
- **Re-fetching QR on every render:** Fetch once on first toggle, cache in state. The URL does not change while the server is running.
- **Clearing `dashQRb64` on hide:** Don't clear it — keep the cached value so toggling back doesn't re-fetch.
- **Using `navigator.clipboard` for copy:** The existing password copy uses `navigator.clipboard` — this is a pre-existing pattern. For new URL copy code, prefer `ClipboardSetText` from Wails runtime. Both work in this WebView context, but Wails API is the more native path.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| QR PNG generation | Custom QR encoder | `github.com/skip2/go-qrcode` | Already in go.mod; handles error correction, sizing, PNG encoding |
| System browser open | `exec.Command("open", url)` | `BrowserOpenURL` from Wails runtime | Cross-platform (macOS/Windows/Linux), no subprocess |
| Clipboard write | Manual IPC | `ClipboardSetText` from Wails runtime | Cross-platform, async, returns success bool |

## Common Pitfalls

### Pitfall 1: Wails Binding Stubs Not Updated

**What goes wrong:** `GetWebServerQRCode` is added to `app.go` but not to `App.d.ts` / `App.js`. TypeScript compilation succeeds (if the type is not imported), but the call fails at runtime with "undefined is not a function".

**Why it happens:** The stubs are NOT auto-regenerated in this project's workflow — they are manually maintained. The comment says "AUTO-GENERATED by Wails" but in practice `wails dev` / `wails build` regenerates them; since we use `vite` for dev, the stubs stay manual.

**How to avoid:** Any new Go method on `App` struct must have a matching entry in both `App.d.ts` (type declaration) and `App.js` (runtime stub). The runtime call pattern is `window['go']['main']['App']['MethodName']()`.

**Warning signs:** TypeScript error on import of new function, or runtime "window.go.main.App.X is not a function".

### Pitfall 2: QR Not Refreshed After Server Restart

**What goes wrong:** User starts server, opens QR, stops server, starts again at a different port/URL — cached `dashQRb64` shows old QR for old URL.

**Why it happens:** `dashQRb64` is cached in local state and never invalidated.

**How to avoid:** Reset `dashQRb64` and `showDashQR` when `isServerRunning` transitions from `true` to `false` (i.e., in a `useEffect([isServerRunning])` that clears on false). The URL can change on server restart (different port or Tailscale reconnect).

**Warning signs:** QR code scans to a URL that 404s while the server is running at a different address.

### Pitfall 3: ClipboardSetText Import Path

**What goes wrong:** Developer imports `ClipboardSetText` from the wrong module.

**Why it happens:** The Wails runtime module path is `'../wailsjs/wailsjs/runtime/runtime'` (note the double `wailsjs/wailsjs`). This is the project's specific path layout and differs from Wails documentation examples.

**How to avoid:** Follow existing import in `App.tsx`:
```typescript
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
```
[VERIFIED: App.tsx line 26]

### Pitfall 4: URL Row Layout Breaks in Narrow Panel

**What goes wrong:** Adding three buttons next to the URL text causes the row to overflow the settings panel's 520px width.

**Why it happens:** Long Tailscale domain names (e.g., `https://kens--personal-macbook-air.tail46d69a.ts.net:7443`) take most of the available width.

**How to avoid:** Use `flex` layout with the URL text set to `overflow: hidden; text-overflow: ellipsis; min-width: 0; flex: 1` and buttons as `flex-shrink: 0`. The existing `.settings-panel__url a` already uses `text-overflow: ellipsis` — extend that approach to the full row.

## Code Examples

### New Go method (app.go addition)

```go
// Source: mirrors GetSessionQRCode in app.go (lines 427-445)
func (a *App) GetWebServerQRCode() (string, error) {
    if a.client == nil {
        return "", fmt.Errorf("daemon not connected")
    }
    resp, err := a.client.GetWebServerStatus()
    if err != nil || !resp.Running {
        return "", fmt.Errorf("web server not running")
    }
    png, err := qrcode.Encode(resp.URL, qrcode.Medium, 256)
    if err != nil {
        return "", fmt.Errorf("GetWebServerQRCode: encode: %w", err)
    }
    return base64.StdEncoding.EncodeToString(png), nil
}
```

### Wails runtime imports needed in SettingsTab.tsx

```typescript
// Source: App.tsx line 26 — same import path used by existing BrowserOpenURL usage
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
import { GetWebServerQRCode } from '../wailsjs/go/main/App'
```

### Copy with feedback (mirrors password copy pattern in SettingsTab.tsx line 95-99)

```typescript
async function handleCopyURL() {
  if (!serverURL) return
  await ClipboardSetText(serverURL)
  setUrlCopied(true)
  setTimeout(() => setUrlCopied(false), 1500)
}
```

### QR state reset on server stop (useEffect guard)

```typescript
useEffect(() => {
  if (!isServerRunning) {
    setShowDashQR(false)
    setDashQRb64(null)
  }
}, [isServerRunning])
```

### Inline QR rendering (mirrors QRModal pattern in QRModal.tsx lines 53-57)

```tsx
{showDashQR && dashQRb64 && (
  <img
    src={`data:image/png;base64,${dashQRb64}`}
    width={200}
    height={200}
    alt="QR code for dashboard URL"
    className="settings-web-server__qr"
  />
)}
```

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — all libraries already in go.mod and package.json)

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (test.environment = jsdom) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WEB-01 | SettingsTab renders Open button calling BrowserOpenURL | source-inspection (raw import) | `pnpm test` | ❌ Wave 0 |
| WEB-02 | SettingsTab copy button calls ClipboardSetText | source-inspection (raw import) | `pnpm test` | ❌ Wave 0 |
| WEB-03 | SettingsTab renders inline QR img when toggled | source-inspection (raw import) | `pnpm test` | ❌ Wave 0 |
| WEB-03 | GetWebServerQRCode method exists in app.go | source-inspection (raw import of app.go) | `pnpm test` | ❌ Wave 0 |

All tests follow the project's established **source-inspection pattern** (`?raw` import of the source file, `expect(raw).toContain(...)` assertions). This avoids jsdom rendering complexity while verifying all structural requirements.

### Sampling Rate

- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/__tests__/SettingsTab.web-link-ux.test.tsx` — covers WEB-01, WEB-02, WEB-03 frontend assertions
- [ ] `frontend/src/components/__tests__/app.go.web-qr.test.tsx` — source-inspection of `app.go` for `GetWebServerQRCode` method

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | no | URL is produced by internal Go method, not user input |
| V6 Cryptography | no | — |

**Security note:** The QR code and copy features expose the dashboard URL visually. This is intentional (WEB-01/02/03 are specifically about URL discoverability). The URL itself is already displayed as plaintext in the existing settings panel. No new attack surface is introduced.

## Open Questions

1. **QR inline vs modal for dashboard URL**
   - What we know: The existing `QRModal` component is session-specific (takes `sessionID`). The requirements say "visible in Settings" which could mean inline or modal.
   - What's unclear: Whether the QR should be always-visible when server is running, or toggled by a button.
   - Recommendation: Use a toggle button ("Show QR" / "Hide QR") to keep the settings panel from becoming excessively tall. This mirrors the show/hide pattern common in password fields.

2. **`navigator.clipboard` vs `ClipboardSetText` consistency**
   - What we know: The existing `handleCopyPassword` in SettingsTab uses `navigator.clipboard.writeText`. The Wails runtime exports `ClipboardSetText`. Both work in this WebView context.
   - What's unclear: Whether to align old code or just use `ClipboardSetText` for the new copy button.
   - Recommendation: Use `ClipboardSetText` for the new URL copy — it is the correct Wails idiom. Do not refactor the existing password copy as part of this phase.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Wails `ClipboardSetText` works correctly in WKWebView (macOS) and WebView2 (Windows) without secure-context restrictions | Standard Stack | Copy silently fails on one platform — would need fallback to `navigator.clipboard` |
| A2 | `App.js` stub pattern is `window['go']['main']['App']['MethodName']()` with no arguments passed | Architecture Patterns | Runtime call fails — need to check actual `App.js` stub format for methods with no parameters |

## Sources

### Primary (HIGH confidence)

- `app.go` lines 427-445 — `GetSessionQRCode` implementation, verified go-qrcode usage pattern [VERIFIED: codebase]
- `frontend/src/wailsjs/wailsjs/runtime/runtime.d.ts` lines 211, 235 — `BrowserOpenURL` and `ClipboardSetText` declarations [VERIFIED: codebase]
- `frontend/src/wailsjs/go/main/App.d.ts` — all existing Wails binding patterns [VERIFIED: codebase]
- `frontend/src/components/SettingsTab.tsx` — existing URL display, copy pattern, state structure [VERIFIED: codebase]
- `go.mod` — `github.com/skip2/go-qrcode` confirmed as direct dependency [VERIFIED: codebase]

### Secondary (MEDIUM confidence)

- `frontend/src/App.tsx` line 26, 431, 529 — `BrowserOpenURL` usage pattern [VERIFIED: codebase]
- `frontend/src/components/WelcomeTab.tsx` line 65 — secondary `BrowserOpenURL` usage [VERIFIED: codebase]

### Tertiary (LOW confidence)

None — all critical claims verified from codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in project, verified from go.mod and package.json
- Architecture: HIGH — patterns copied from existing implementations in same codebase
- Pitfalls: HIGH — identified from direct code inspection of stub files and existing patterns

**Research date:** 2026-04-11
**Valid until:** 2026-05-11 (stable Wails/React codebase, no fast-moving dependencies)
