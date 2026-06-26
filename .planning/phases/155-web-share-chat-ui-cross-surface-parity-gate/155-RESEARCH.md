# Phase 155: Web-Share Chat UI + Cross-Surface Parity Gate — Research

**Researched:** 2026-06-26
**Domain:** React SPA web-mode routing, WebSocket URL parameterization, Markdown export, Playwright multi-client e2e
**Confidence:** HIGH (all key claims verified against source)

---

## Summary

Phase 155 surfaces the Session Chat experience on the web-share browser — delivering the same thread, presence indicators, typing indicators, unread badge, @mention highlights, and @session injection available on the desktop GUI. It also gates the v4.1 release by verifying cross-surface parity via Playwright e2e.

The desktop chat UI (ChatPanel.tsx, MentionPopover, ChatBadge, etc.) was fully built in Phase 154 and is REUSED as-is. Phase 155 does NOT rebuild the chat UI; it wires the web-mode connection path and adds three missing capabilities: (1) a web-mode WS URL that routes through the webserver instead of the local relay loopback, (2) a web-mode HTTP base URL for history and export fetches, and (3) YAML-frontmatter export on both surfaces. The chat components themselves are unchanged.

The largest technical risk is the RelayClient WebSocket URL, which is hardcoded to `ws://127.0.0.1:${port}`. Every surface that is NOT the local relay loopback needs a different URL. This is a precision surgery on relayClient.ts and ChatPanel.tsx — the changes are small but must propagate consistently or chat will silently fail to connect on the web-share surface.

**Primary recommendation:** Add a `wsURL?: string` override prop to RelayClient (opt-in, non-breaking), then construct the correct `wss://{host}/sessions/{id}/ws?cap={token}` URL in the new `WebShareSessionView` component. Mirror the same override approach for the HTTP history/export fetch: add `apiBaseURL?: string` and `capToken?: string` props to ChatPanel so each surface passes its own base.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EXPORT-01 | A user can download a chat thread as a Markdown file, from both the desktop GUI and the web-share surface. | Export route exists on relay (`/api/chat/{id}/export` via `wrapRelayWithChat`) and webserver (`GET /api/chat/{id}/export`, cap-gated). `ChatStore.Export()` MUST be updated to emit YAML frontmatter. UI: add Export button to ChatPanel toolbar on both surfaces; desktop uses `http://127.0.0.1:${relayPort}/api/chat/{id}/export`, web uses `${origin}/api/chat/{id}/export?cap={token}`. Browser triggers download via `Content-Disposition: attachment` already set server-side. |
| PARITY-01 | Every Session Chat feature behaves identically on the desktop GUI and the web-share browser surface (release-blocking). | Requires mounting ChatPanel on web-share with correct WS URL and HTTP base URL. All chat behaviors (thread, presence, typing, unread badge, @mention, @session inject, RO/RW cap gate) already work on the relay/webserver read-pump. Phase 155 proves parity via a Playwright two-client e2e test (see Parity Gate Design section). |

</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Chat WS connection (web-share) | Frontend (React SPA) | webserver (routes the WS upgrade) | Web-share SPA constructs `wss://{host}/sessions/{id}/ws?cap=TOKEN` and owns the RelayClient instance |
| Chat history fetch (web-share) | Frontend fetch → webserver REST | — | `GET /api/chat/{id}/history?cap=TOKEN` already cap-gated on webserver |
| Chat export fetch (web-share) | Frontend fetch → webserver REST | — | `GET /api/chat/{id}/export?cap=TOKEN` already cap-gated on webserver; `Content-Disposition: attachment` server-side triggers browser download |
| Chat WS connection (desktop) | Frontend (React SPA) | relay loopback (127.0.0.1:relayPort) | TerminalPanel and ChatPanel both open their own RelayClient to the local relay; NO WS change needed for desktop |
| Chat history fetch (desktop) | Frontend fetch → relay loopback | — | `GET /api/chat/{id}/history` on relay via `wrapRelayWithChat`; no cap needed (loopback trust boundary) |
| Chat export fetch (desktop) | Frontend fetch → relay loopback | — | Same relay wrap; no cap needed on loopback |
| RO/RW cap enforcement | Daemon relay hub (HandleChatSend / HandleInject) | webserver read-pump | Server-side gate; already correct on both paths per Phases 153–154 |
| YAML frontmatter in export | daemon (ChatStore.Export) | — | `internal/daemon/chat.go:321` must be updated; both relay and webserver call `store.Export()` |
| Parity gate (e2e) | Playwright fixture (cmd/playwright-fixture/main.go) | frontend/e2e/chat-parity.spec.ts | Fixture needs ChatStore + provider wiring; tests drive two browser pages against the webserver |
| Web-share session view mount | Frontend (App.tsx web-mode branch) | WebShareSessionView component (new) | App.tsx:1087-1091 must be extended to mount terminal+chat when in web mode |

---

## Surface Detection and Rendering Gap (Verified)

### What Exists

`frontend/src/lib/webMode.ts:51` — `detectMode()` returns `'web'` when `window.location.pathname.startsWith('/app/')`. `readWebModeParams()` at :65 reads `?session=` and `?cap=` from the URL. [VERIFIED: file:frontend/src/lib/webMode.ts:51,65]

`frontend/src/App.tsx:108` — `const mode = detectMode()`. Web-mode bootstrap at :1087-1091:
```ts
useEffect(() => {
  if (mode === 'web' && webParams.sessionId) {
    handleOpenFileBrowser(webParams.sessionId, webParams.sessionId)
  }
}, [])
```
[VERIFIED: file:frontend/src/App.tsx:1087-1091]

Web mode currently opens a **file browser tab only**. `HubPanel` is never mounted in web mode (gated at `activeId === HUB_TAB.id`, which is never set in web mode). `HubInteractiveModal` and `ChatPanel` are therefore never mounted on the web-share surface today. [VERIFIED: file:frontend/src/App.tsx:1440-1481]

### What Phase 155 Must Add

A new `WebShareSessionView` component (suggested path: `frontend/src/components/Hub/WebShareSessionView.tsx`) that mounts `TerminalPanel` + `ChatPanel` with:
- WS URL: `wss://${window.location.host}/sessions/${sessionId}/ws?cap=${capToken}`
- History/export base URL: `window.location.origin`
- Cap token forwarded from `webParams.capToken`

The App.tsx web-mode bootstrap effect must be extended to mount `WebShareSessionView` in addition to (or instead of) the file browser tab, depending on what the cap permits.

**Design decision for planner:** When the user lands on `/app/?session={id}&cap={token}`, show the session view (terminal + chat drawer) as the primary tab, with a separate file-browser tab if `cap` includes `files.read`. This mirrors how the desktop Hub shows the session modal as the primary interaction surface.

---

## Standard Stack

All packages already installed. No new npm packages required.

### Core (all already in frontend/package.json)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@tanstack/react-virtual` | 3.14.3 | Virtualized message list in ChatPanel | Already installed Phase 154 |
| `react-textarea-autosize` | 8.5.9 | Auto-growing composer | Already installed Phase 154 |
| `react-markdown` | 10.1.0 | Safe Markdown rendering | Already installed Phase 154 |
| `remark-gfm` | ^4.0.1 | GitHub-flavored Markdown | Already installed Phase 154 |
| `rehype-sanitize` | ^6.0.0 | XSS sanitization | Already installed Phase 154 |

### New npm packages: NONE
All required functionality is achieved by extending existing components and adding new props.

## Package Legitimacy Audit

No new external packages are introduced in this phase. Section is not applicable.

---

## Architecture Patterns

### Pattern 1: RelayClient WS URL Override (New)

**What:** Add an optional `wsURL?: string` parameter to `RelayClient` constructor to override the hardcoded `ws://127.0.0.1:${port}${path}`.

**Current code** (`frontend/src/lib/relayClient.ts:211-221`):
```typescript
constructor(
  port: number,
  sessionId: string,
  private callbacks: RelayClientCallbacks,
  opts?: { remote?: boolean },
) {
  const path = opts?.remote
    ? `/api/relay/remote/${sessionId}/ws`
    : `/sessions/${sessionId}/ws`
  const url = `ws://127.0.0.1:${port}${path}`
  this.ws = new WebSocket(url)
```
[VERIFIED: file:frontend/src/lib/relayClient.ts:211-221]

**Proposed change:**
```typescript
constructor(
  port: number,
  sessionId: string,
  private callbacks: RelayClientCallbacks,
  opts?: { remote?: boolean; wsURL?: string },
) {
  let url: string
  if (opts?.wsURL) {
    url = opts.wsURL  // caller-provided override (web-share path)
  } else {
    const path = opts?.remote
      ? `/api/relay/remote/${sessionId}/ws`
      : `/sessions/${sessionId}/ws`
    url = `ws://127.0.0.1:${port}${path}`
  }
  this.ws = new WebSocket(url)
```

**Web-share WS URL construction** (modeled on `web/assets/terminal.js:44-45`):
```typescript
const wsURL = `wss://${window.location.host}/sessions/${sessionId}/ws?cap=${encodeURIComponent(capToken)}`
```
[VERIFIED: file:web/assets/terminal.js:41-45]

The webserver route `GET /sessions/{id}/ws` is gated by `requireCapability` which reads `?cap=` from the query string. [VERIFIED: file:internal/webserver/server.go:669-670, capability_mw.go:39]

**When to use:** `WebShareSessionView` (new web-mode component) passes the constructed `wsURL` to both ChatPanel and TerminalPanel.

### Pattern 2: ChatPanel Web-Mode Props (New)

**What:** Add `apiBaseURL?: string` and `capToken?: string` to `ChatPanelProps` to control where history and export fetches point.

**Current ChatPanelProps** (`frontend/src/components/Hub/ChatPanel.tsx:63-78`):
```typescript
export interface ChatPanelProps {
  sessionId: string
  relayPort: number
  open: boolean
  currentUserTailnetID?: string
  onUnreadChange?: (count: number, hasMention: boolean) => void
}
```
[VERIFIED: file:frontend/src/components/Hub/ChatPanel.tsx:63-78]

**Proposed additions:**
```typescript
export interface ChatPanelProps {
  sessionId: string
  relayPort: number
  open: boolean
  currentUserTailnetID?: string
  onUnreadChange?: (count: number, hasMention: boolean) => void
  /** Override WS URL — used by web-share surface. When absent, uses ws://127.0.0.1:relayPort */
  wsURL?: string
  /** Base URL for history and export fetches. Defaults to http://127.0.0.1:relayPort */
  apiBaseURL?: string
  /** Capability token appended as ?cap= to web-surface API calls. */
  capToken?: string
}
```

**`loadChatHistory` update** (`frontend/src/components/Hub/ChatPanel.tsx:219-232`):
```typescript
export async function loadChatHistory(
  relayPort: number,
  sessionId: string,
  opts?: { apiBaseURL?: string; capToken?: string },
): Promise<ChatMessage[]> {
  try {
    const base = opts?.apiBaseURL ?? `http://127.0.0.1:${relayPort}`
    const capParam = opts?.capToken ? `?cap=${encodeURIComponent(opts.capToken)}` : ''
    const resp = await fetch(`${base}/api/chat/${sessionId}/history${capParam}`)
    if (!resp.ok) return []
    return resp.json() as Promise<ChatMessage[]>
  } catch {
    return []
  }
}
```

**Export URL construction:**
```typescript
function buildExportURL(
  relayPort: number,
  sessionId: string,
  opts?: { apiBaseURL?: string; capToken?: string },
): string {
  const base = opts?.apiBaseURL ?? `http://127.0.0.1:${relayPort}`
  const capParam = opts?.capToken ? `?cap=${encodeURIComponent(opts.capToken)}` : ''
  return `${base}/api/chat/${sessionId}/export${capParam}`
}
```

### Pattern 3: WebShareSessionView (New Component)

**What:** A thin wrapper that composes `TerminalPanel` + `ChatPanel` + toggle for the web-share surface, equivalent to `HubInteractiveModal` but configured for web connections.

**Reuse:** Same `HubInteractiveModal` JSX structure — the D-02 overlay design (ChatPanel `position: absolute` over terminal) is surface-agnostic. The only difference is the WS URL and the API base URL.

**Sketch:**
```typescript
// frontend/src/components/Hub/WebShareSessionView.tsx
export function WebShareSessionView({ sessionId, capToken, relayPort, theme, pluginConfig }: Props) {
  const [chatOpen, setChatOpen] = useState(false)
  const [unreadCount, setUnreadCount] = useState(0)
  const [hasMention, setHasMention] = useState(false)

  const wsURL = `wss://${window.location.host}/sessions/${encodeURIComponent(sessionId)}/ws?cap=${encodeURIComponent(capToken)}`
  const apiBaseURL = window.location.origin

  return (
    <div className="hub-modal__body hub-modal__body--interactive">
      <TerminalPanel
        sessionId={sessionId}
        isActive={true}
        relayPort={relayPort}   // still needed as fallback; wsURL overrides
        wsURL={wsURL}           // NEW prop, overrides the loopback URL
        ...
      />
      <ChatPanel
        sessionId={sessionId}
        relayPort={relayPort}   // fallback
        open={chatOpen}
        wsURL={wsURL}           // NEW prop
        apiBaseURL={apiBaseURL} // NEW prop
        capToken={capToken}     // NEW prop
        onUnreadChange={(c, m) => { setUnreadCount(c); setHasMention(m) }}
      />
      <button className="hub-modal__chat-toggle" onClick={() => setChatOpen(p => !p)}>
        <ChatBubbleLeftRightIcon />
        <ChatBadge count={unreadCount} hasMention={hasMention} />
      </button>
    </div>
  )
}
```

**Note:** `TerminalPanel` also uses `RelayClient` internally. It must receive the same `wsURL` override so it connects via the webserver WS rather than the loopback. Add `wsURL?: string` to `TerminalPanelProps` and thread it to the `RelayClient` constructor.

### Pattern 4: HubInteractiveModal — No Change Needed

The desktop `HubInteractiveModal` (`frontend/src/components/Hub/HubInteractiveModal.tsx`) uses `relayPort` exclusively and has no web-mode coupling. It does NOT need changes for PARITY-01. The web-share surface gets its own `WebShareSessionView` component. [VERIFIED: file:frontend/src/components/Hub/HubInteractiveModal.tsx]

### Pattern 5: App.tsx Web-Mode Bootstrap Extension

Replace the current single file-browser auto-open:
```typescript
// current (App.tsx:1087-1091)
useEffect(() => {
  if (mode === 'web' && webParams.sessionId) {
    handleOpenFileBrowser(webParams.sessionId, webParams.sessionId)
  }
}, [])
```

With a dual tab auto-open:
```typescript
useEffect(() => {
  if (mode === 'web' && webParams.sessionId) {
    // Primary: open the session view (terminal + chat)
    openWebSessionTab(webParams.sessionId, webParams.capToken)
    // Secondary: also open file browser if cap grants files.read
    // (the FileBrowserTab's own requireCapability logic handles the 403 if not granted)
    handleOpenFileBrowser(webParams.sessionId, webParams.sessionId)
  }
}, [])
```

**Design note for planner:** Since we cannot decode the cap on the client without the signing key, we cannot pre-check for `files.read`. Options: (a) always open the file-browser tab and let it show `PermissionDeniedTakeover` if the cap lacks `files.read`, or (b) open only the session view and omit the file-browser tab. Option (a) preserves backward compatibility.

---

## Export Design (EXPORT-01)

### YAML Frontmatter Spec

`ChatStore.Export()` at `internal/daemon/chat.go:321` currently outputs:
```markdown
# Chat Thread: <sessionID>

## alias (2026-06-26T12:00:00Z)

**Author ID:** local

message content

---
```
[VERIFIED: file:internal/daemon/chat.go:321-341]

**Required output** (success criterion 2):
```markdown
---
session: <sessionID>
exported_at: <ISO-8601 UTC>
participants:
  - "alias1 (tailnetID1)"
  - "alias2 (tailnetID2)"
---

# Chat: <sessionID>

## alias (tailnetID) — 2026-06-26T12:00:00Z

message content

_injected into terminal_

---
```

**Frontmatter fields:**
- `session`: the session ID (already available as `s.sessionID`)
- `exported_at`: `time.Now().UTC().Format(time.RFC3339)`
- `participants`: deduplicated list of `"AuthorAlias (AuthorID)"` pairs from the message list (order of first appearance)

**Implementation location:** `internal/daemon/chat.go:321` — update `Export()` to prepend the YAML block using `strings.Builder`. The participants list is derived from the already-copied `msgs` slice: iterate once, collect `(AuthorAlias, AuthorID)` pairs, dedup by AuthorID.

**GitHub-compatible:** YAML frontmatter is recognized by GitHub's Markdown renderer. Plain `---` fences with YAML keys are the GitHub standard. No third-party library needed — use `fmt.Fprintf` to emit valid YAML (no special chars in session IDs or typical aliases that would require quoting; aliases are already validated by `ValidateAlias` to printable-ASCII).

### Download Trigger Per Surface

**Both surfaces use Content-Disposition: attachment** — already set server-side:
- Relay: `internal/daemon/chat_routes.go:82`: `Content-Disposition: attachment; filename="chat-{id}.md"`
- Webserver: `internal/webserver/chat.go:101`: same header

A browser navigating to or fetching the export URL will receive an `attachment` disposition and trigger a native download prompt. The cleanest client-side trigger is a hidden `<a>` element:

**Desktop:** `<a href={`http://127.0.0.1:${relayPort}/api/chat/${sessionId}/export`} download>`
**Web-share:** `<a href={`${window.location.origin}/api/chat/${sessionId}/export?cap=${encodeURIComponent(capToken)}`} download>`

Both can be implemented as a programmatic click on a hidden anchor:
```typescript
function triggerExport(url: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = ''  // let Content-Disposition filename win
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}
```

**Export button placement:** Add an "Export" button (or icon button) to the ChatPanel header/toolbar, visible when the panel is open. Clicking triggers `triggerExport(buildExportURL(...))`. This is the same button on both surfaces; it just receives different URL props.

### Does Desktop Need a New Route?

**No.** The relay already exposes `GET /api/chat/{id}/export` via `wrapRelayWithChat` in `internal/daemon/chat_routes.go:94-95`. CORS is already applied (`relay.FilesCORS`). The desktop ChatPanel can fetch from `http://127.0.0.1:${relayPort}/api/chat/${sessionId}/export` without changes to the Go layer. [VERIFIED: file:internal/daemon/chat_routes.go:94-95]

---

## RO/RW Cap Gate Verification (Success Criteria 3 & 4)

**Server-side gating is already correct on both paths.** Verified:

| Path | Where ReadOnly is Set | Effect |
|------|-----------------------|--------|
| Relay loopback (desktop) | `relay/server.go` — owner always RW; `sub.ReadOnly = false` | MsgChatSend and MsgSessionInject both go through `hub.HandleChatSend` / `hub.HandleInject` which check `ErrChatReadOnly` / `!sub.ReadOnly` |
| Webserver (web-share) | `webserver/server.go:1015-1016`: `readonly := !capability.HasPerm(claims.Perms, "write")` | `Subscriber.ReadOnly` bound to signed JWT claim, NOT to query string |

[VERIFIED: file:internal/webserver/server.go:1015-1016]
[VERIFIED: file:internal/webserver/capability_mw.go:39]

The signed JWT is verified by `requireCapability` which reads `?cap=` from the query string and verifies HMAC. A RO client (`Perms: "read"`) cannot promote itself to RW by omitting `?readonly=1` or constructing a custom frame. The gate is in `Hub.HandleChatSend` and `Hub.HandleInject`, which are shared across both WS paths. [VERIFIED: Phase 154 VERIFICATION.md lines 107-108]

**What Phase 155 adds for SC-3:**
- Client-side: suppress the Send button and Inject button in `ChatPanel` when in RO mode (defense-in-depth). Read RO status from the presence payload (each subscriber's identity is already broadcast with `Origin: "web"` or `"local"`); or derive from whether the WS connection receives an inject-error frame on send attempt.
- e2e: Send a chat message from a RO cap via the React UI → verify no message appears in the thread AND the server sends an inject-error or silently drops (already proven in Phase 153 Go tests; the e2e closes the browser test gap).

**What Phase 155 does NOT need to change:**
- `HandleChatSend` RO gate (Phase 154)
- `HandleInject` RO gate (Phase 153)
- `requireCapability` middleware (Phase 87)

---

## Parity Gate Design (PARITY-01)

### Fundamental Constraint

The "known automated-input limitation" from the MEMORY note (`"web-share WS blocks automated input"`) refers specifically to the xterm-based PTY terminal (`web/assets/terminal.js`): Playwright cannot type into the xterm canvas to produce PTY input. This does NOT apply to the React ChatPanel compositor textarea, which is a standard `<textarea>` fully interactable via Playwright's `page.fill()` and `page.keyboard.press()`.

Chat message sending in the React SPA is 100% automatable via Playwright.

### Two-Client Harness Shape

Use `browser.newContext()` to open two browser contexts against the same fixture server. Both contexts connect to the same WebSocket hub via the webserver. A message sent by Context A is broadcast to Context B via `hub.BroadcastChat`.

```typescript
// frontend/e2e/chat-parity.spec.ts (new file)
import { test, expect } from '@playwright/test'
import { loadFixtureEnv } from './fixture-env'

test.describe('Phase 155 PARITY-01 — chat parity gate', () => {
  test('RW sender, RW receiver — message broadcast between two web-share clients', async ({ browser }) => {
    const env = loadFixtureEnv()
    const rwURL = `${env.baseURL}/app/?session=playwright-test-session&cap=${encodeURIComponent(env.cap)}`

    const ctx1 = await browser.newContext({ ignoreHTTPSErrors: true })
    const ctx2 = await browser.newContext({ ignoreHTTPSErrors: true })
    try {
      const page1 = await ctx1.newPage()
      const page2 = await ctx2.newPage()

      // Both viewers navigate to the web-share app URL
      await page1.goto(rwURL)
      await page2.goto(rwURL)

      // Open chat on both pages (click the chat toggle)
      await page1.locator('.hub-modal__chat-toggle').click()
      await page2.locator('.hub-modal__chat-toggle').click()

      // Page 1 sends a message
      const testMessage = `parity-${Date.now()}`
      await page1.locator('.chat-panel__composer textarea').fill(testMessage)
      await page1.keyboard.press('Enter')

      // Page 2 sees the same message (broadcast via WS)
      await expect(page2.locator('.chat-msg').filter({ hasText: testMessage }))
        .toBeVisible({ timeout: 5_000 })

      // Both pages show the same unread badge was cleared (not accumulating on sender's own message)
      // ... (detailed assertions per SC-1)
    } finally {
      await ctx1.close()
      await ctx2.close()
    }
  })

  test('RO viewer cannot send — server gate holds', async ({ browser }) => {
    const env = loadFixtureEnv()
    const roURL = `${env.baseURL}/app/?session=playwright-test-session&cap=${encodeURIComponent(env.viewerCap)}`
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true })
    try {
      const page = await ctx.newPage()
      await page.goto(roURL)
      await page.locator('.hub-modal__chat-toggle').click()

      // Send button should be disabled or absent for RO cap
      const sendButton = page.locator('[data-chat-send]')
      await expect(sendButton).toBeDisabled()

      // Direct WS frame injection attempt (black-box adversarial test):
      // Use page.evaluate to send a MsgChatSend frame directly via the WebSocket
      // and assert no message appears in the thread
      const messagesBefore = await page.locator('.chat-msg').count()
      await page.evaluate(() => {
        // Access the WS if exposed; otherwise use fetch to hit the API directly
        // This mirrors the Phase 153 adversarial Go tests (TestInjectRO_WebPath)
      })
      await page.waitForTimeout(500)
      const messagesAfter = await page.locator('.chat-msg').count()
      expect(messagesAfter).toBe(messagesBefore)  // no new messages from RO client
    } finally {
      await ctx.close()
    }
  })

  test('EXPORT-01 — export button downloads .md file with YAML frontmatter', async ({ page }) => {
    const env = loadFixtureEnv()
    const rwURL = `${env.baseURL}/app/?session=playwright-test-session&cap=${encodeURIComponent(env.cap)}`
    await page.goto(rwURL)
    await page.locator('.hub-modal__chat-toggle').click()

    // Intercept the download
    const downloadPromise = page.waitForEvent('download')
    await page.locator('[data-chat-export]').click()
    const download = await downloadPromise

    // Filename matches Content-Disposition
    expect(download.suggestedFilename()).toMatch(/^chat-.*\.md$/)

    // Read content (stream to string)
    const content = await download.path()
      .then(p => p ? require('fs').readFileSync(p, 'utf8') : '')
    expect(content).toContain('---')       // YAML frontmatter fence
    expect(content).toContain('session:')  // YAML session field
    expect(content).toContain('exported_at:')
  })
})
```

### Fixture Gaps to Close

The playwright fixture (`cmd/playwright-fixture/main.go`) does NOT currently wire `SetChatHistoryProvider` or `SetChatExportProvider`. Phase 155 must add:

1. A `ChatStore` instance seeded with a few test messages at fixture startup
2. `ws.SetChatHistoryProvider(...)` wired to the store
3. `ws.SetChatExportProvider(...)` wired to the store
4. The fixture ChatStore must use a `baseDir` (e.g., `t.TempDir()` equivalent at startup) so it has a real file to read/write

The fixture already has a `HubManager` with a session. The ChatStore can be constructed with `daemon.NewChatStore(tmpDir, sessionID)` and pre-populated.

**Note on fixture imports:** `cmd/playwright-fixture/main.go` is in the `main` package; it can import `internal/daemon` directly (no cycle concern). The `ChatStore` is already used via the daemon package in production; importing it in the fixture is safe.

### What Each Assertion Proves

| Assertion | Success Criterion | How |
|-----------|-----------------|-----|
| Message sent by RW Client A appears on RW Client B | SC-1 thread parity | Two-page broadcast test |
| Presence roster shows both clients | SC-1 presence parity | `page.locator('.chat-presence')` |
| Typing indicator appears on Page 2 when Page 1 types | SC-1 typing parity | `page1.keyboard.type('abc')` → assert typing indicator on page2 |
| Unread badge appears on Page 2 when Page 1 sends while Page 2 has chat closed | SC-1 unread badge parity | close chat on page2, page1 sends → assert badge count on page2 |
| @mention highlight on mentioned user's page | SC-1 mention highlight parity | page1 sends "@alias …" → assert `.chat-msg--mention` on page2 |
| RO cap viewer cannot send (Send disabled) | SC-3 | `expect(sendButton).toBeDisabled()` |
| RO cap server-side gate (direct WS frame) | SC-3 | `page.evaluate` WS injection |
| Export downloads .md with YAML frontmatter | SC-2 | `page.waitForEvent('download')` |
| @session inject indicator appears on both pages after RW inject | SC-4 | two-page inject + assert `.chat-msg--inject` |

### What Must Remain Manual UAT

The following CANNOT be automated in the Playwright e2e suite and must remain in TESTING.md Section 5:

1. **@session inject visual indicator in native Wails WebView** — Phase 153/154 deferred UAT that now runs live on the desktop GUI; the Playwright fixture is web-only and has no real PTY.
2. **Typing indicator visual timing** — the 500ms debounce and 5s TTL auto-clear require precise timing that is brittle in CI.
3. **Overlay no-resize proof on live PTY (M-21)** — already registered; cannot be proven via the web fixture.

---

## Common Pitfalls

### Pitfall 1: RelayClient Port=0 in Web Mode
**What goes wrong:** `WebShareSessionView` passes `relayPort={0}` (web mode never sets a relay port) to `ChatPanel`. The `ChatPanel` effect `new RelayClient(relayPort, sessionId, ...)` builds `ws://127.0.0.1:0/sessions/...` and immediately fails.
**Why it happens:** The `relayPort` prop is currently required. In web mode, `relayPort` is always `0` or `null` (App.tsx never calls `GetRelayPort()` in web mode).
**How to avoid:** The `wsURL` override in RelayClient short-circuits before using the port. The `WebShareSessionView` must always provide `wsURL` and should not rely on `relayPort`. Keep `relayPort` as a required prop but document that it is ignored when `wsURL` is provided; pass `relayPort={0}` (a safe sentinel).
**Warning signs:** Console error `WebSocket connection to 'ws://127.0.0.1:0/...' failed.`

### Pitfall 2: Cap Token Not Forwarded to History Fetch
**What goes wrong:** `loadChatHistory` on web-share fetches `${window.location.origin}/api/chat/${sessionId}/history` without `?cap=TOKEN` → webserver returns 401.
**Why it happens:** The `requireCapability` middleware reads `?cap=` from the query string; without it, ALL chat API calls fail.
**How to avoid:** `WebShareSessionView` reads `capToken` from `readWebModeParams()` and passes it down to `ChatPanel` via the `capToken` prop; `loadChatHistory` appends `?cap=${encodeURIComponent(capToken)}`.
**Warning signs:** Chat thread shows "error" loading state immediately after WS connects (history fetch returns 401 and the `setPhase('error')` branch fires).

### Pitfall 3: WS Origin Mismatch (requireAllowedOrigin)
**What goes wrong:** The webserver WS upgrade is wrapped in `requireAllowedOrigin` which allows only `ws.allowedOrigins()` — i.e., `ws.BaseURL()`. If the browser sends an `Origin` header that doesn't match, the upgrade is rejected with 403.
**Why it happens:** `ws.BaseURL()` in the fixture is `https://127.0.0.1:<PORT>` (TLS). The browser's `Origin` header for a page loaded from `https://127.0.0.1:<PORT>/app/...` is `https://127.0.0.1:<PORT>` — this matches. But if the fixture is configured differently, the match fails silently.
**How to avoid:** Always load the SPA from the same origin as the webserver; the WS `wss://...` URL must use the same host. Confirmed: in both fixture and production, the SPA is served by the same webserver process; same origin is guaranteed.
[VERIFIED: file:internal/webserver/server.go:1041-1043]

### Pitfall 4: Export YAML Quoting
**What goes wrong:** If `AuthorAlias` contains `:`, `#`, `"`, `\n`, or other YAML special characters, the frontmatter becomes invalid YAML.
**Why it happens:** `ValidateAlias` restricts aliases to printable ASCII and 1-50 chars, but `:` and `#` are both valid printable ASCII and valid alias characters. They are YAML special chars outside quoted strings.
**How to avoid:** Always quote participant strings in the YAML output: `  - "alias (tailnetID)"`. Go's `strings.Replace(alias, `"`, `\\"`, -1)` to escape embedded quotes. Session IDs are strictly `[A-Za-z0-9_-]` (`validChatSessionID`) so they never need quoting.
[VERIFIED: file:internal/relay/protocol.go — ValidateAlias accepts printable ASCII except controls]

### Pitfall 5: Two-Page Playwright Test State Leak
**What goes wrong:** Test A sends a message; Test B sees it in history (persisted in the ChatStore) — causing false parity assertions or phantom message counts.
**Why it happens:** The fixture ChatStore persists to disk and survives between tests.
**How to avoid:** Use unique test messages with `Date.now()` suffix so each test identifies its own messages. Alternatively, the fixture should start with a fresh ChatStore (tempdir) for each test run — this is already the case since `globalSetup` starts a fresh fixture process. Within a single test file, tests run sequentially and share the same fixture (so messages accumulate). Use `.filter({ hasText: specificMessage })` assertions, not `.count()` comparisons alone.

### Pitfall 6: TerminalPanel Missing wsURL Prop
**What goes wrong:** `WebShareSessionView` passes the correct `wsURL` to `ChatPanel` but NOT to `TerminalPanel`. The terminal connects to `ws://127.0.0.1:0/sessions/{id}/ws` (wrong) while chat connects correctly.
**Why it happens:** Two separate RelayClient instances: `TerminalPanel` owns one, `ChatPanel` owns another. Both must be updated.
**How to avoid:** Verify that `TerminalPanel` also receives and uses the `wsURL` override. The existing `remote?: boolean` option in `TerminalPanelProps` is a precedent; add `wsURL?: string` alongside it.
[VERIFIED: file:frontend/src/components/TerminalPanel.tsx:55-80; file:frontend/src/lib/relayClient.ts:211-221]

### Pitfall 7: Fixture ChatStore Not Wired = 404 on History
**What goes wrong:** E2e tests load the web-share app page; `ChatPanel` connects WS fine but `loadChatHistory` returns 404 (provider not set) → chat shows empty thread even though messages exist.
**Why it happens:** `cmd/playwright-fixture/main.go` does not call `ws.SetChatHistoryProvider` or `ws.SetChatExportProvider`.
**How to avoid:** Wire both providers in the fixture using a pre-seeded ChatStore (see Fixture Gaps section above).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML serialization in Export | Custom YAML encoder | `fmt.Fprintf` with quoted strings | Only 3 scalar fields needed; a full YAML library is overkill and adds a dependency |
| WebSocket client for web-share | New WS class | Extend existing `RelayClient` with `wsURL` override | RelayClient already handles binary framing, ping-keepalive, all frame types |
| Cap token parsing client-side | JWT decode | None — don't decode on client | Signing key is server-side; client never needs to inspect the JWT. Read `?cap=` from URL params and pass it opaquely to `?cap=` query on API calls |
| Download trigger | File API / Blob download | Native `<a download>` + `Content-Disposition: attachment` (already set server-side) | Simpler, zero JS overhead, cross-browser |
| Multi-client e2e orchestration | WebSocket mock | `browser.newContext()` two real browser contexts | Playwright's real browser WS support is exactly the right tool |

---

## Runtime State Inventory

This is a greenfield UI wiring phase, NOT a rename/refactor. No runtime state migration is required. Section omitted per phase type.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.22+ | `http.NewServeMux` pattern matching (`{id}`) | Yes | darwin/arm64 | None needed |
| Playwright | e2e chat-parity.spec.ts | Yes | (installed in frontend/) | None — required for PARITY-01 gate |
| pnpm | frontend builds | Yes | (installed) | None needed |

**Missing dependencies:** None.

---

## Validation Architecture

`workflow.nyquist_validation` is not set in `.planning/config.json` → treat as enabled.

### Test Framework

| Property | Value |
|----------|-------|
| Go test framework | `testing` + `go test ./...` |
| Frontend unit framework | vitest (`pnpm test run`) |
| e2e framework | Playwright (`pnpm exec playwright test`) |
| Quick run command (Go) | `go test ./internal/daemon/... -run Chat -count=1` |
| Quick run command (TS) | `pnpm test run src/components/Hub/ChatPanel.test.tsx` |
| Full suite command | `go test ./... && pnpm test run && pnpm exec playwright test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXPORT-01 | `ChatStore.Export()` emits YAML frontmatter | Go unit | `go test ./internal/daemon/... -run TestChatStore_Export` | No — add to `internal/daemon/chat_test.go` |
| EXPORT-01 | Export URL returns .md with YAML frontmatter on webserver | Go | `go test ./internal/webserver/... -run TestChatExport` | No — add to `internal/webserver/chat_test.go` |
| EXPORT-01 | Export URL returns .md on relay loopback | Go | `go test ./internal/daemon/... -run TestChatRoutes_Export` | No — add to `internal/daemon/chat_routes_test.go` |
| EXPORT-01 | Export button triggers browser download | e2e | `pnpm exec playwright test --grep "EXPORT-01"` | No — new `frontend/e2e/chat-parity.spec.ts` |
| PARITY-01 | Two web-share clients exchange messages | e2e | `pnpm exec playwright test --grep "PARITY-01"` | No — new `frontend/e2e/chat-parity.spec.ts` |
| PARITY-01 | RO cap viewer: Send disabled + server rejects | e2e | `pnpm exec playwright test --grep "RO viewer"` | No — new `frontend/e2e/chat-parity.spec.ts` |
| PARITY-01 | ChatPanel renders on web-share (smoke) | vitest | `pnpm test run src/components/Hub/WebShareSessionView.test.tsx` | No — new file |
| PARITY-01 | WebShareSessionView: correct wsURL constructed | vitest | same file | No — new file |
| PARITY-01 | @session inject path from web-share: same frame, same handler | Go unit | `go test ./internal/webserver/... -run TestInject` (existing Phase 153 tests) | Yes — `internal/webserver/inject_test.go` |

**TESTING.md gaps to resolve in Phase 155:**
- Add `frontend/e2e/chat-parity.spec.ts` to Section 2 (Playwright suite)
- Add `frontend/src/components/Hub/WebShareSessionView.test.tsx` to Section 2 (vitest)
- Add rows for EXPORT-01 and PARITY-01 to Section 4 (traceability)
- Add NOTIF-02 dedicated traceability row (minor gap from Phase 154 VERIFICATION.md)
- Add Phase 155 UAT items to Section 5

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/... -run Chat -count=1 && pnpm test run src/components/Hub/`
- **Per wave merge:** `go test ./... && pnpm test run`
- **Phase gate:** Full suite green + Playwright e2e green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/e2e/chat-parity.spec.ts` — covers EXPORT-01 + PARITY-01
- [ ] `frontend/src/components/Hub/WebShareSessionView.test.tsx` — covers PARITY-01 component render
- [ ] `cmd/playwright-fixture/main.go` — wire ChatStore + providers before tests can run
- [ ] YAML frontmatter unit test in `internal/daemon/chat_test.go`

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Phase 155 adds no new auth surface |
| V3 Session Management | No | Cap tokens already managed (Phase 87) |
| V4 Access Control | Yes | RO/RW gate on chat send + inject; already implemented in relay hub + webserver read-pump; Phase 155 adds client-side defense-in-depth (disable Send button for RO cap) |
| V5 Input Validation | Yes | YAML frontmatter: quote participant strings; alias validated by `ValidateAlias` upstream |
| V6 Cryptography | No | Phase 155 does not add or change crypto |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| RO client promotes to RW by constructing raw WS frame | Elevation of Privilege | Server-side `Subscriber.ReadOnly` bound to signed JWT claim (already implemented Phase 153); Phase 155 adds UI suppression as depth |
| YAML injection via AuthorAlias in frontmatter | Tampering | Always quote participant values in `fmt.Fprintf`; `ValidateAlias` limits to printable ASCII |
| Cap token in WS URL leaks in browser history | Info Disclosure | Cap tokens in `?cap=` query param appear in history; this is the existing production pattern and is acceptable (tokens are time-bounded JWTs) |
| File download via `<a>` spoofed | Spoofing | `Content-Disposition: attachment; filename="chat-{id}.md"` — session ID is `[A-Za-z0-9_-]` only; no path traversal in filename |

---

## Code Examples

### Export YAML Frontmatter Pattern (Go)
```go
// internal/daemon/chat.go — updated Export()
func (s *ChatStore) Export() (string, error) {
    s.mu.Lock()
    msgs := make([]relay.ChatMessage, len(s.messages))
    copy(msgs, s.messages)
    sessionID := s.sessionID
    s.mu.Unlock()

    // Deduplicate participants by AuthorID (order of first appearance)
    seen := make(map[string]bool)
    var participants []string
    for _, msg := range msgs {
        if !seen[msg.AuthorID] {
            seen[msg.AuthorID] = true
            // Quote to handle YAML special chars in alias
            safeAlias := strings.ReplaceAll(msg.AuthorAlias, `"`, `\"`)
            participants = append(participants, fmt.Sprintf(`"%s (%s)"`, safeAlias, msg.AuthorID))
        }
    }

    var b strings.Builder
    exportedAt := time.Now().UTC().Format(time.RFC3339)
    fmt.Fprintf(&b, "---\n")
    fmt.Fprintf(&b, "session: %s\n", sessionID)
    fmt.Fprintf(&b, "exported_at: %s\n", exportedAt)
    fmt.Fprintf(&b, "participants:\n")
    for _, p := range participants {
        fmt.Fprintf(&b, "  - %s\n", p)
    }
    fmt.Fprintf(&b, "---\n\n")
    fmt.Fprintf(&b, "# Chat: %s\n\n", sessionID)

    for _, msg := range msgs {
        ts := time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339)
        fmt.Fprintf(&b, "## %s (%s) — %s\n\n", msg.AuthorAlias, msg.AuthorID, ts)
        fmt.Fprintf(&b, "%s\n\n", msg.Content)
        if msg.SessionInject {
            fmt.Fprintf(&b, "_injected into terminal_\n\n")
        }
        fmt.Fprintf(&b, "---\n\n")
    }
    return b.String(), nil
}
```

Note: `relay.ChatMessage` uses field `AuthorAlias` (Go struct field) with json tag `"alias"`. Confirm struct field name before editing. [VERIFIED: file:internal/relay/protocol.go — ChatMessage struct]

### Web-Share WS URL Construction (TypeScript)
```typescript
// In WebShareSessionView.tsx
const { sessionId, capToken } = readWebModeParams()
const wsURL = `wss://${window.location.host}/sessions/${encodeURIComponent(sessionId!)}/ws?cap=${encodeURIComponent(capToken!)}`
```

Pattern matches `web/assets/terminal.js:41-45`. [VERIFIED: file:web/assets/terminal.js:41-45]

---

## Open Questions

1. **Should WebShareSessionView replace or supplement the FileBrowserTab?**
   - What we know: Current web mode opens only FileBrowserTab; PARITY-01 requires chat on web-share
   - What's unclear: Whether existing web-share users rely on the file-browser-first UX
   - Recommendation: Open BOTH tabs by default; session view as the active (primary) tab, file browser as a secondary tab. This is backward-compatible and adds chat without removing files.

2. **Who is `currentUserTailnetID` on the web-share surface?**
   - What we know: `ChatPanel` defaults `currentUserTailnetID` to `"local"`. On web-share, the real TailnetID is the Tailscale node pubkey (resolved by `lc.WhoIs` server-side at WS upgrade) and broadcast in the first `MsgPresence` frame.
   - What's unclear: How the client knows its OWN TailnetID before the first `MsgPresence` arrives (which includes all participants, not "self").
   - Recommendation: The presence frame includes `personKey = tailnetID + ":web"`. The client's own `personKey` can be inferred from the first presence roster entry that matches the `origin: "web"` entry with a connection count > 0 AND that just joined. Or: the server could send a dedicated "your identity" frame on WS connect. For Phase 155, derive self-identity from the presence roster's first entry with `origin: "web"` and `connCount >= 1` that wasn't present before this client joined. This is the same approach Phase 154 used for the desktop (defaulting to `"local"`).

3. **Does the Playwright fixture need a real PTY for the @session inject e2e test?**
   - What we know: The fixture uses `io.Pipe` stubs (not a real PTY). `Hub.WriteInput` calls `inputCaptureW.Write` which goes to the pipe.
   - What's unclear: Whether the chat broadcast and inject indicator appear without a real terminal process responding.
   - Recommendation: The inject indicator is a chat broadcast (`SessionInject: true` flag in the `ChatMessage`) — it does NOT require PTY output. The e2e test can assert the inject indicator renders in the chat thread without any PTY being active. The PTY write itself goes to `inputCaptureW` (the pipe) silently.

---

## Sources

### Primary (HIGH confidence — verified against source code)
- `frontend/src/lib/webMode.ts` — Surface detection (detectMode, readWebModeParams)
- `frontend/src/App.tsx` — Web-mode rendering branch; FileBrowserTab auto-open; HubPanel not mounted in web mode
- `frontend/src/components/Hub/HubInteractiveModal.tsx` — TerminalPanel + ChatPanel composition pattern
- `frontend/src/components/Hub/ChatPanel.tsx` — Props, loadChatHistory, RelayClient usage, web-mode gaps
- `frontend/src/lib/relayClient.ts` — RelayClient constructor; hardcoded `ws://127.0.0.1:{port}` URL
- `internal/daemon/chat.go` — ChatStore.Export() current output (no YAML frontmatter)
- `internal/daemon/chat_routes.go` — relay loopback chat history + export routes via wrapRelayWithChat
- `internal/daemon/api.go` — RelayHandler + wrapRelayWithChat wiring; setChatProviders
- `internal/webserver/chat.go` — webserver chat history + export handlers
- `internal/webserver/server.go` — Route registrations; handleWSSRelay; requireCapability; requireAllowedOrigin
- `internal/webserver/capability_mw.go` — Cap token extraction from `?cap=` query param
- `web/assets/terminal.js` — Web-share WS URL pattern (wss:// + withCap)
- `frontend/playwright.config.ts` + `frontend/e2e/global-setup.ts` — Playwright fixture infrastructure
- `frontend/e2e/fixture-env.ts` — fixture-env.json schema; appUrl, viewerAppUrl helpers
- `cmd/playwright-fixture/main.go` — Fixture binary; ChatStore not wired (gap)
- `.planning/phases/154-desktop-chat-ui/154-CONTEXT.md` — Locked design decisions D-01..D-10
- `.planning/phases/154-desktop-chat-ui/154-VERIFICATION.md` — What shipped; RO/RW gate confirmed
- `.planning/REQUIREMENTS.md` — EXPORT-01, PARITY-01 requirement text
- `TESTING.md` — Section 4 traceability map; Section 5 manual checklist; existing Phase 154 entries

### Metadata

**Confidence breakdown:**
- Architecture gap analysis: HIGH — verified line-by-line against source
- Export YAML frontmatter spec: HIGH — ChatStore.Export confirmed; YAML structure is a straightforward spec derivation
- Parity gate harness: MEDIUM — the two-context Playwright approach is correct in theory; exact locator selectors depend on the CSS class names that will be used in the new component (not yet created)
- Fixture wiring approach: HIGH — `daemon.NewChatStore` import is safe; SetChatHistoryProvider API confirmed

**Research date:** 2026-06-26
**Valid until:** 2026-07-26 (Go package APIs and React component shapes are stable; React SPA behavior is highly stable once Phase 154 is verified)
