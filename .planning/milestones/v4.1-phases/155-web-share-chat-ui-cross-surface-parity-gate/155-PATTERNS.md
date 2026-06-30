# Phase 155: Web-Share Chat UI + Cross-Surface Parity Gate - Pattern Map

**Mapped:** 2026-06-26
**Files analyzed:** 9 (3 new, 6 modified) + TESTING.md (docs-only)
**Analogs found:** 8 / 8 code files (every new/modified code file has an in-repo analog)

> Phase 155 is a wiring phase, not a greenfield build. The chat UI (ChatPanel,
> ChatBadge, ChatMessage, MentionPopover, etc.) shipped in Phase 154 and is reused
> as-is. Every new file has a direct sibling to copy from; every modified file is an
> additive prop/field extension. Prefer the in-repo analogs below over RESEARCH.md
> sketches — the sketches are directionally correct but the real signatures (below)
> are authoritative.

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/Hub/WebShareSessionView.tsx` (NEW) | component (wrapper) | event-driven (WS) | `frontend/src/components/Hub/HubInteractiveModal.tsx` | exact (clone + reconfigure) |
| `frontend/src/components/Hub/WebShareSessionView.test.tsx` (NEW) | test (vitest) | — | `frontend/src/components/Hub/ChatPanel.test.tsx` (sibling vitest) | role-match |
| `frontend/e2e/chat-parity.spec.ts` (NEW) | test (Playwright e2e) | request-response + WS | `frontend/e2e/web-csp.spec.ts` + `fixture-env.ts` helpers | role-match |
| `frontend/src/components/Hub/ChatPanel.tsx` (MOD) | component | event-driven (WS) + CRUD fetch | self (Phase 154 GREEN) | self-extend |
| `frontend/src/lib/relayClient.ts` (MOD) | utility (WS client) | streaming/event-driven | self (`remote?` opt precedent) | self-extend |
| `frontend/src/components/TerminalPanel.tsx` (MOD) | component | streaming (PTY WS) | self (`remote?` prop precedent) | self-extend |
| `frontend/src/App.tsx` (MOD) | component (root/router) | request-response | self (web-mode bootstrap effect at :1087) | self-extend |
| `internal/daemon/chat.go` (MOD) | model/store | transform (serialize) | self (`ChatStore.Export()` at :321) | self-extend |
| `cmd/playwright-fixture/main.go` (MOD) | config/fixture | request-response | self (`SetFilesHandler`/`SetPluginSettingsProvider` wiring at :122-167) | self-extend |
| `TESTING.md` (MOD) | docs | — | self (Section 2/4/5 conventions) | self-extend |

---

## Pattern Assignments

### `frontend/src/components/Hub/WebShareSessionView.tsx` (NEW — component, event-driven)

**Analog:** `frontend/src/components/Hub/HubInteractiveModal.tsx` (104 lines — clone the whole structure).

This is a near-verbatim clone of HubInteractiveModal with three changes: (1) take
`sessionId: string` + `capToken: string` props instead of a `SessionInfo`, (2) compute
`wsURL` + `apiBaseURL` and thread them down, (3) the chat toggle / overlay JSX is
identical (same class names — required by the Playwright selector contract).

**Imports + overlay structure to copy** (HubInteractiveModal.tsx:1-8, 65-104):
```tsx
import React, { useState } from 'react'
import { ChatBubbleLeftRightIcon } from '@heroicons/react/24/outline'
import { TerminalPanel } from '../TerminalPanel'
import { ChatPanel } from './ChatPanel'
import { ChatBadge } from './ChatBadge'
// ...
return (
  <div className="hub-modal__body hub-modal__body--interactive">
    <TerminalPanel sessionId={session.id} isActive={open} relayPort={relayPort} ... />
    <ChatPanel sessionId={session.id} relayPort={relayPort} open={chatOpen} onUnreadChange={handleUnreadChange} />
    <button
      type="button"
      className="hub-modal__chat-toggle"
      aria-label={chatOpen ? 'Close chat' : 'Open chat'}
      onClick={() => setChatOpen((prev) => !prev)}
    >
      <ChatBubbleLeftRightIcon className="hub-modal__chat-toggle-icon" aria-hidden="true" />
      <ChatBadge count={unreadCount} hasMention={hasMention} />
    </button>
  </div>
)
```

**Unread state hooks to copy verbatim** (HubInteractiveModal.tsx:55-63):
```tsx
const [chatOpen, setChatOpen] = useState(false)
const [unreadCount, setUnreadCount] = useState(0)
const [hasMention, setHasMention] = useState(false)
function handleUnreadChange(count: number, mention: boolean) {
  setUnreadCount(count)
  setHasMention(mention)
}
```

**Props interface** (per UI-SPEC §3 — authoritative):
```tsx
interface WebShareSessionViewProps {
  sessionId: string
  capToken: string          // from readWebModeParams()
  relayPort: number         // 0 on web-share — safe sentinel, ignored when wsURL present
  theme?: TerminalTheme      // NOTE: HubInteractiveModal uses ITheme from '@xterm/xterm'
  pluginConfig?: PluginConfig
}
```

**WS + API URL construction (the ONLY behavioral difference from the analog):**
```tsx
const wsURL = `wss://${window.location.host}/sessions/${encodeURIComponent(sessionId)}/ws?cap=${encodeURIComponent(capToken)}`
const apiBaseURL = window.location.origin
```
Then pass `wsURL` to BOTH `<TerminalPanel wsURL={wsURL} ...>` and
`<ChatPanel wsURL={wsURL} apiBaseURL={apiBaseURL} capToken={capToken} ...>`.
**Pitfall 6 (RESEARCH):** forgetting `wsURL` on TerminalPanel = terminal connects to
`ws://127.0.0.1:0/...` while chat works — both must receive it.

**Note on `isActive` timing:** HubInteractiveModal binds `isActive={open}` to the
modal-open animation phase (Pitfall 1: 0-column fit during animation). WebShareSessionView
has no grow animation, so `isActive={true}` is correct once mounted.

---

### `frontend/src/components/Hub/ChatPanel.tsx` (MODIFIED — component)

**Analog:** itself (Phase 154 GREEN). All changes are additive.

**1. Props interface** — current shape (ChatPanel.tsx:63-78), add three optional props:
```tsx
export interface ChatPanelProps {
  sessionId: string
  relayPort: number
  open: boolean
  currentUserTailnetID?: string
  onUnreadChange?: (count: number, hasMention: boolean) => void
  // ── Phase 155 additions (all optional → non-breaking for desktop) ──
  wsURL?: string       // overrides ws://127.0.0.1:relayPort in RelayClient
  apiBaseURL?: string  // base for history+export fetch; default http://127.0.0.1:relayPort
  capToken?: string    // appended as ?cap= to web-surface API calls
}
```

**2. RelayClient construction** — current call (ChatPanel.tsx:345), thread `wsURL`:
```tsx
// current
const client = new RelayClient(relayPort, sessionId, { ... })
// Phase 155: pass opts.wsURL through
const client = new RelayClient(relayPort, sessionId, { ...callbacks }, { wsURL })
```
The `useEffect` dep array at :385 is `[sessionId, relayPort]` — add `wsURL` (and
`apiBaseURL`, `capToken` if read inside) so a web-mode remount reconnects correctly.

**3. `loadChatHistory` signature** — current (ChatPanel.tsx:219-232) is hardcoded to
loopback. Extend with an opts param exactly as RESEARCH §Pattern 2 shows:
```tsx
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
  } catch { return [] }
}
```
Update the call site in the `onOpen` callback (ChatPanel.tsx:364) to pass
`{ apiBaseURL, capToken }`. **Pitfall 2 (RESEARCH):** missing `?cap=` → webserver 401 →
`setPhase('error')` fires immediately after WS connects.

**4. Export button** — add to the header (`chat-panel__header`, ChatPanel.tsx:586-611).
Per UI-SPEC §1: `ArrowDownTrayIcon` from `@heroicons/react/24/outline`, `data-chat-export`
attribute (Playwright selector), `aria-label="Export chat as Markdown"`. Use the
hidden-anchor download helper (UI-SPEC §1, RESEARCH:360-368):
```tsx
function triggerExport(url: string): void {
  const a = document.createElement('a')
  a.href = url; a.download = ''
  document.body.appendChild(a); a.click(); document.body.removeChild(a)
}
function buildExportURL(): string {
  const base = apiBaseURL ?? `http://127.0.0.1:${relayPort}`
  const capParam = capToken ? `?cap=${encodeURIComponent(capToken)}` : ''
  return `${base}/api/chat/${sessionId}/export${capParam}`
}
```

**5. RO-cap suppression** — the existing Send button (ChatPanel.tsx:759-770) currently
has NO `data-chat-send` attribute. Phase 155 must:
- Add `data-chat-send` to the Send button (present in BOTH enabled and disabled states —
  UI-SPEC §5 selector contract).
- Derive `isReadOnly` from the presence roster (the `PresenceEntry` for self with
  `ReadOnly: true`). Set `disabled` + `aria-disabled="true"` when RO.
- Short-circuit `handleInjectPointerDown` (ChatPanel.tsx:506) on RO.
- Render the `"Read only"` label below the composer (UI-SPEC §2).

Current Send button to extend (ChatPanel.tsx:760-769):
```tsx
<button
  type="button"
  className="chat-composer__send-btn"
  aria-label="Send message"
  onClick={handleSend}
  disabled={!draft.trim()}          // Phase 155: || isReadOnly
>                                    // Phase 155: add data-chat-send
  <PaperAirplaneIcon className="chat-composer__send-icon" aria-hidden="true" />
  <span>Send</span>
</button>
```

---

### `frontend/src/lib/relayClient.ts` (MODIFIED — utility, WS client)

**Analog:** itself. The `opts?.remote` branch (relayClient.ts:215-220) is the exact
precedent for adding an opt-in URL override.

**Current constructor** (relayClient.ts:211-221):
```ts
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

**Phase 155 change** — add `wsURL?: string` to opts, short-circuit before port use:
```ts
opts?: { remote?: boolean; wsURL?: string },
) {
  let url: string
  if (opts?.wsURL) {
    url = opts.wsURL                  // web-share override (wss://host/...?cap=)
  } else {
    const path = opts?.remote ? `/api/relay/remote/${sessionId}/ws` : `/sessions/${sessionId}/ws`
    url = `ws://127.0.0.1:${port}${path}`
  }
  this.ws = new WebSocket(url)
```
Everything below (binaryType, ping interval, onmessage frame switch at :234-254) is
unchanged — the override only changes the URL string. **Pitfall 1 (RESEARCH):** the
override must short-circuit BEFORE `127.0.0.1:${port}` is built, since `port` is `0` in
web mode.

---

### `frontend/src/components/TerminalPanel.tsx` (MODIFIED — component, streaming)

**Analog:** itself. `TerminalPanelProps.remote?` (TerminalPanel.tsx:62-66) is the exact
precedent — add `wsURL?: string` alongside it and thread to the RelayClient constructor.

**Current props block** (TerminalPanel.tsx:55-66):
```ts
interface TerminalPanelProps {
  sessionId: string
  isActive: boolean
  relayPort: number
  fontSize: number
  onFontSizeChange: (delta: number) => void
  theme: ITheme
  remote?: boolean   // ← add `wsURL?: string` here
  ...
}
```
Find the `new RelayClient(...)` call inside TerminalPanel (it passes `{ remote }` today)
and add `wsURL` to the opts object: `{ remote, wsURL }`.

---

### `frontend/src/App.tsx` (MODIFIED — root/router)

**Analog:** itself. The existing web-mode bootstrap effect (App.tsx:1087-1091) is the
exact insertion point.

**Current** (App.tsx:1087-1091):
```tsx
useEffect(() => {
  if (mode === 'web' && webParams.sessionId) {
    handleOpenFileBrowser(webParams.sessionId, webParams.sessionId)
  }
}, [])
```

**Phase 155** (UI-SPEC §4): open `WebShareSessionView` as the PRIMARY (active) tab; keep
`handleOpenFileBrowser` as the secondary background tab. `webParams` comes from
`readWebModeParams()` which returns `{ sessionId, capToken }` (verified
`frontend/src/lib/webMode.ts:30-71`). The cap cannot be decoded client-side, so always
open the file tab and let `PermissionDeniedTakeover` handle a missing `files.read`
(RESEARCH Pattern 5, option a — backward compatible). The planner must locate the
tab-open helper used for new tabs (the same mechanism `handleOpenFileBrowser` uses) and
add an `openWebSessionTab(sessionId, capToken)` analog that mounts `WebShareSessionView`.

---

### `internal/daemon/chat.go` (MODIFIED — model/store, transform)

**Analog:** itself. `ChatStore.Export()` (chat.go:321-341) — replace the header block and
message-header format with the YAML-frontmatter version.

**Current Export()** (chat.go:321-341):
```go
func (s *ChatStore) Export() (string, error) {
	s.mu.Lock()
	msgs := make([]relay.ChatMessage, len(s.messages))
	copy(msgs, s.messages)
	sessionID := s.sessionID
	s.mu.Unlock()

	var b strings.Builder
	fmt.Fprintf(&b, "# Chat Thread: %s\n\n", sessionID)
	for _, msg := range msgs {
		ts := time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339)
		fmt.Fprintf(&b, "## %s (%s)\n\n", msg.AuthorAlias, ts)
		fmt.Fprintf(&b, "**Author ID:** %s\n\n", msg.AuthorID)
		fmt.Fprintf(&b, "%s\n\n", msg.Content)
		if msg.SessionInject {
			fmt.Fprintf(&b, "_injected into terminal_\n\n")
		}
		fmt.Fprintf(&b, "---\n\n")
	}
	return b.String(), nil
}
```

**Phase 155** — same lock/copy preamble (keep verbatim), then emit frontmatter +
per-message blocks per the Export File Format Contract (UI-SPEC §"Export File Format
Contract"). Field facts confirmed from source:
- `relay.ChatMessage` fields used: `.AuthorAlias`, `.AuthorID`, `.TimestampMs`,
  `.Content`, `.SessionInject` (all already referenced in current Export).
- `exported_at`: `time.Now().UTC().Format(time.RFC3339)`.
- `participants`: dedup by `AuthorID`, order of first appearance, ALWAYS double-quoted:
  `"alias (authorID)"`, with embedded `"` escaped via `strings.ReplaceAll(alias, "\"", "\\\"")`
  (Pitfall 4 — aliases may contain `:` / `#`). Message header changes to
  `## %s (%s) — %s` (alias, authorID, ts). See RESEARCH:695-738 for the full Go body.
- `session` value is bare (session IDs are `[A-Za-z0-9_-]` only — no quoting needed).

**Test analog:** unit tests live in `internal/daemon/chat_test.go` — add
`TestChatStore_Export` asserting the `---` fence, `session:`, `exported_at:`,
`participants:` lines, and quoting of an alias containing `:`.

---

### `cmd/playwright-fixture/main.go` (MODIFIED — fixture)

**Analog:** itself. The provider-wiring pattern is established by `SetFilesHandler`
(main.go:161-167) and `SetPluginSettingsProvider` (main.go:135-141). Add ChatStore
wiring in the same style, after the `manager.Create(...)` at :99 and before/after the
`ws` setup at :110-167.

**Confirmed APIs:**
- Constructor: `daemon.NewChatStore(baseDir, sessionID string) (*ChatStore, error)`
  (chat.go:99). Use a tempdir as `baseDir`.
- Seed: `store.AppendMessage(msg relay.ChatMessage)` (chat.go:244).
- Webserver providers (internal/webserver/chat.go:35,45):
  ```go
  ws.SetChatHistoryProvider(func(sessionID string) (history []byte, found bool, err error) { ... })
  ws.SetChatExportProvider(func(sessionID string) (markdown string, found bool, err error) { ... })
  ```
  History provider returns `json.Marshal(store.Messages())` (chat.go:207 returns
  `[]relay.ChatMessage`); export provider returns `store.Export()`.
- The fixture already imports `internal/webserver`, `internal/relay`,
  `internal/capability` and constructs `cfg`/`ws` (main.go:62, 95, 103-122). Add
  `internal/daemon` import (no cycle — fixture is `package main`, per RESEARCH:520).

**RO cap for the parity gate:** the fixture already mints OWNER
(`Perms: "read,write,files.read"`, main.go:178-189) and VIEWER (`Perms: "read"`,
main.go:191-202). The VIEWER cap (read-only, no write) is the RO chat cap for the
SC-3 test. If the RO test needs chat-history read but no write, confirm `viewerCap`
grants history GET (it has `read`); otherwise mint a dedicated chat-RO cap following
the exact `capability.Claims{...}` + `capability.Sign(...)` + `ws.AddGrant(...)` block
at main.go:191-202.

---

### `frontend/e2e/chat-parity.spec.ts` (NEW — Playwright e2e)

**Analogs:** `frontend/e2e/web-csp.spec.ts` (spec structure, `test.describe`, page event
capture) + `frontend/e2e/fixture-env.ts` (URL + cap helpers).

**Imports + env pattern to copy** (web-csp.spec.ts:19-20, fixture-env.ts:88-95):
```ts
import { test, expect } from '@playwright/test'
import { appUrl, viewerAppUrl, loadFixtureEnv } from './fixture-env'
```
`appUrl(env)` already builds `${baseURL}/app/?session=playwright-test-session&cap=${env.cap}`
(fixture-env.ts:88-90) and `viewerAppUrl(env)` does the same with the RO `viewerCap`
(fixture-env.ts:92-95) — use these instead of hand-building URLs (RESEARCH:423 sketches
a manual URL; prefer the helper).

**Two-client harness:** `browser.newContext({ ignoreHTTPSErrors: true })` × 2 (TLS
self-signed fixture). Selectors are FROZEN by UI-SPEC §5 — use exactly:
`.hub-modal__chat-toggle`, `.chat-panel__composer textarea`, `.chat-msg`,
`.chat-msg--mention`, `[data-chat-send]`, `[data-chat-export]`, `.chat-presence`,
`.chat-msg--inject`, `.chat-badge`, `.chat-typing`. See RESEARCH:415-506 for the full
test bodies (broadcast, RO-gate, export-download).

**Registration:** add the file to TESTING.md Section 2 (Playwright suite) and Section 4
(traceability rows for EXPORT-01 + PARITY-01). Run
`bash tests/check-traceability-paths.sh` before commit (repo CLAUDE.md standing rule).

---

### `frontend/src/components/Hub/WebShareSessionView.test.tsx` (NEW — vitest)

**Analog:** the sibling vitest file `frontend/src/components/Hub/ChatPanel.test.tsx`
(same directory, same `@testing-library/react` + `vitest` conventions). Asserts: correct
`wsURL` constructed (`wss://...?cap=...`), RO prop threading, renders without error.
Register in TESTING.md Section 2 (vitest suite).

---

## Shared Patterns

### Pattern: opt-in URL override (non-breaking)
**Source:** `frontend/src/lib/relayClient.ts:215` (`opts?.remote` branch) and
`frontend/src/components/TerminalPanel.tsx:62-66` (`remote?` prop).
**Apply to:** relayClient.ts (`wsURL?`), TerminalPanel.tsx (`wsURL?`), ChatPanel.tsx
(`wsURL?`/`apiBaseURL?`/`capToken?`). Every new prop is OPTIONAL so desktop callers
(HubInteractiveModal) compile and behave identically with zero changes.

### Pattern: cap token is opaque on the client
**Source:** `frontend/src/lib/webMode.ts:65` (`readWebModeParams` returns raw `capToken`).
**Apply to:** all web-surface API calls — read `?cap=` from the URL, forward verbatim as
`?cap=${encodeURIComponent(capToken)}`. NEVER JWT-decode client-side (signing key is
server-only — RESEARCH "Don't Hand-Roll").

### Pattern: Content-Disposition download via hidden anchor
**Source:** server already sets `Content-Disposition: attachment; filename="chat-{id}.md"`
(`internal/daemon/chat_routes.go:82`, `internal/webserver/chat.go:101`).
**Apply to:** the Export button on ChatPanel (both surfaces). Client just `a.click()` a
hidden `<a href={exportURL} download="">` — no Blob/File API.

### Pattern: frozen CSS class / data-attr selectors (cross-surface parity)
**Source:** UI-SPEC §5 + Phase 154 class names already in ChatPanel/HubInteractiveModal.
**Apply to:** WebShareSessionView (must reuse `hub-modal__body--interactive`,
`hub-modal__chat-toggle`) and the ChatPanel Send/Export buttons (`data-chat-send`,
`data-chat-export`). Renaming any of these breaks the parity gate spec.

### Pattern: webserver provider injection
**Source:** `cmd/playwright-fixture/main.go:135-167` (`SetPluginSettingsProvider`,
`SetFilesHandler`) + `internal/webserver/chat.go:35,45`
(`SetChatHistoryProvider`/`SetChatExportProvider`).
**Apply to:** fixture ChatStore wiring — same closure-provider shape.

---

## No Analog Found

None. Every new code file has a direct in-repo analog (HubInteractiveModal for the
component, ChatPanel.test/web-csp.spec for tests). Every modified file is a self-extension.

---

## Metadata

**Analog search scope:** `frontend/src/components/Hub/`, `frontend/src/lib/`,
`frontend/src/components/`, `frontend/e2e/`, `internal/daemon/`, `internal/webserver/`,
`cmd/playwright-fixture/`.
**Files scanned:** 11 (HubInteractiveModal.tsx, ChatPanel.tsx, relayClient.ts,
TerminalPanel.tsx, App.tsx, chat.go, chat_routes.go ref, webserver/chat.go,
playwright-fixture/main.go, fixture-env.ts, web-csp.spec.ts).
**Pattern extraction date:** 2026-06-26
