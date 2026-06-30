# Phase 160: v4.1 Chat Closeout — Pattern Map

**Mapped:** 2026-06-27
**Files analyzed:** 9 (1 new, 8 modified)
**Analogs found:** 9 / 9

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `frontend/src/components/Hub/useChatUnreadListeners.ts` (NEW) | hook | event-driven (WS subscription per session) | `HubPanel.tsx` — `usePreviewPoller` (inline hook, lines 59–119) | role-match |
| `frontend/src/components/Hub/useChatUnreadListeners.test.ts` (NEW) | test | — | `HubPanel.test.tsx` (pattern) + `server_inject_test.go` (mock pattern) | role-match |
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | component | request-response | itself — current file is the analog (add `onUnreadChange` prop + callback) | self |
| `frontend/src/components/Hub/HubModal.tsx` | component | request-response | itself — current file is the analog (thread prop one level up) | self |
| `frontend/src/components/Hub/HubPanel.tsx` | component | event-driven + CRUD | itself — current file is the analog (add `unreadMap` state + `handleUnreadChange`) | self |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | component | CRUD | itself — current file is the analog (add `unreadBySessionId` prop) | self |
| `internal/relay/server_inject_test.go` | test | request-response | itself — add one new test function following existing pattern | self |
| `internal/relay/sanitize.go` | utility | transform | itself — doc comment edit only | self |
| `scripts/install.sh` | config/script | file-I/O | itself — surgical line edits (WR-01, WR-03) | self |

---

## Pattern Assignments

### `frontend/src/components/Hub/useChatUnreadListeners.ts` (NEW — hook, event-driven)

**Closest analog:** `frontend/src/components/Hub/HubPanel.tsx`, `usePreviewPoller` function (lines 59–119)

`usePreviewPoller` is an inline custom hook with the same lifecycle shape required by `useChatUnreadListeners`: it takes a sessions array + an `isActive` gate, uses `useEffect` keyed on a stable `sessionIdKey` derived from session IDs, runs one side-effect per session in the list, and returns cleanup via the effect return. This is the precise lifecycle pattern to copy.

**Hook signature pattern** (analog: `HubPanel.tsx` lines 64–67):
```typescript
function usePreviewPoller(
  sessions: SessionInfo[],
  isActive: boolean,
): Map<string, daemon.StyledSpan[][]> {
```

New hook signature:
```typescript
// useChatUnreadListeners.ts
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { useEffect, useRef } from 'react'
import { RelayClient } from '../../lib/relayClient'
import { accrueUnread } from './ChatPanel'
import type { UnreadState } from './ChatPanel'

export function useChatUnreadListeners(
  sessions: SessionInfo[],
  relayPort: number,
  openModalSessionId: string | null,
  isActive: boolean,
  onUnreadChange: (sessionId: string, count: number, hasMention: boolean) => void,
): void
```

**Stable dep key pattern** (analog: `HubPanel.tsx` lines 73):
```typescript
const sessionIdKey = sessions.map((s) => s.id).join(',')
```
Copy this exactly. Used in `useEffect` dependency array to avoid effect re-runs on array reference churn.

**Effect with isActive gate + cleanup pattern** (analog: `HubPanel.tsx` lines 75–116):
```typescript
useEffect(() => {
  if (!isActive || sessions.length === 0) return
  let cancelled = false

  // ... per-session setup ...

  return () => { cancelled = true; /* teardown each client */ }
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [sessionIdKey, isActive])
```

**Per-session RelayClient construction** (source: RESEARCH.md architecture notes):
```typescript
// For each session where session.id !== openModalSessionId AND relayPort > 0:
const client = new RelayClient(relayPort, session.id, {
  onOutput: () => {},    // REQUIRED by RelayClient — pass no-op (Pitfall 4)
  onChat: (message) => {
    const prev = unreadRefs.current.get(session.id) ?? { count: 0, hasMention: false }
    const next = accrueUnread(prev, message, 'local')  // 'local' = desktop owner always
    unreadRefs.current.set(session.id, next)
    onUnreadChange(session.id, next.count, next.hasMention)
  },
})
```

**useRef for per-session state accumulation** (avoid re-renders on every message):
```typescript
// Use useRef map (not useState) so MsgChat callbacks don't trigger re-renders
const unreadRefs = useRef<Map<string, UnreadState>>(new Map())
```

**Map state update pattern in HubPanel** (analog: `HubPanel.tsx` line 99–108 — how usePreviewPoller produces new Map):
```typescript
setTails((prev) => {
  const next = new Map(prev)
  localSessions.forEach((s, i) => {
    // ...
    next.set(s.id, lines)
  })
  return next
})
```
Copy the functional-setState + `new Map(prev)` pattern for `setUnreadMap` in HubPanel (Pitfall 5 — mutating a Map reference does not trigger re-render).

---

### `frontend/src/components/Hub/HubInteractiveModal.tsx` (modified — add `onUnreadChange` prop)

**Analog:** itself. The current file structure is the pattern.

**Current props interface** (lines 12–28 — add one optional prop):
```typescript
export interface HubInteractiveModalProps {
  session: SessionInfo
  isOpen: boolean
  relayPort: number
  fontSize: number
  theme: ITheme
  pluginConfig?: PluginSettings | null
  onFontSizeChange?: (delta: number) => void
  remote?: boolean
  // ADD:
  onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void
}
```

**Current `handleUnreadChange` function** (lines 60–63 — add one call):
```typescript
function handleUnreadChange(count: number, mention: boolean) {
  setUnreadCount(count)
  setHasMention(mention)
  // ADD: props.onUnreadChange?.(session.id, count, mention)
}
```

Note the signature difference: `ChatPanel.onUnreadChange` is `(count, hasMention)` without sessionId. `HubInteractiveModal.handleUnreadChange` wraps it and adds `session.id` when calling the new prop. This is the sessionId injection point.

---

### `frontend/src/components/Hub/HubModal.tsx` (modified — thread `onUnreadChange` prop)

**Analog:** itself. `HubModalProps` interface (lines 37–50) shows the prop threading pattern.

**Current `HubModalProps`** (lines 37–50 — add one optional prop):
```typescript
export interface HubModalProps {
  session: SessionInfo
  sourceRect: DOMRect
  relayPort: number
  fontSize: number
  theme: ITheme
  pluginConfig?: PluginSettings | null
  onFontSizeChange?: (delta: number) => void
  remote?: boolean
  onClose: () => void
  // ADD:
  onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void
}
```

Thread to `<HubInteractiveModal onUnreadChange={onUnreadChange} ...>` at the HubInteractiveModal render site. `HubBriefingModal` has no chat — no threading needed for the briefing branch.

---

### `frontend/src/components/Hub/HubPanel.tsx` (modified — add unreadMap state + hook call)

**Analog:** itself. State + prop threading pattern already established by `usePreviewPoller` (lines 59–119) and `localPreviewTails` (line 334).

**State additions in HubPanel component body** (parallel to existing state declarations around lines 300–335):
```typescript
// NOTIF-01: unread counts per session — lifted from HubInteractiveModal
const [unreadMap, setUnreadMap] = useState<Map<string, { count: number; hasMention: boolean }>>(new Map())

function handleUnreadChange(sessionId: string, count: number, hasMention: boolean) {
  setUnreadMap((prev) => {
    const m = new Map(prev)
    m.set(sessionId, { count, hasMention })
    return m
  })
}
```

**Reset on modal open** (inside `handleCardClick`, before `setModalState`):
```typescript
// Reset unread badge when user opens the session modal
setUnreadMap((prev) => {
  const m = new Map(prev)
  m.delete(session.id)
  return m
})
```

**Hook call** (parallel to `usePreviewPoller` call at line 334):
```typescript
// Existing:
const localPreviewTails = usePreviewPoller(sessions, isActive ?? false)
// Add:
useChatUnreadListeners(sessions, relayPort ?? 0, modalState?.session.id ?? null, isActive ?? false, handleUnreadChange)
```

**Pass unreadMap to SessionCardGrid** (existing `<SessionCardGrid>` render site — add one prop):
```typescript
<SessionCardGrid
  // ... existing props ...
  unreadBySessionId={unreadMap}
/>
```

**Pass onUnreadChange to HubModal** (existing `<HubModal>` render site — add one prop):
```typescript
<HubModal
  // ... existing props ...
  onUnreadChange={handleUnreadChange}
/>
```

---

### `frontend/src/components/Hub/SessionCardGrid.tsx` (modified — add `unreadBySessionId` prop)

**Analog:** itself. `SessionCardGridProps` interface (lines 133–170) shows the prop addition pattern — each phase adds optional props to this interface and passes them to `<SessionCard>` at two render sites.

**Props addition** (after line 170, following existing pattern):
```typescript
export interface SessionCardGridProps {
  // ... existing props ...
  // NOTIF-01: unread badge data keyed by session ID
  unreadBySessionId?: Map<string, { count: number; hasMention: boolean }>
}
```

**Two SessionCard render sites** — both must receive the unread props (named-group path ~lines 266–282, workDir path ~lines 316+):
```typescript
<SessionCard
  // ... existing props ...
  unreadCount={unreadBySessionId?.get(s.id)?.count}
  hasChatMention={unreadBySessionId?.get(s.id)?.hasMention}
/>
```

`SessionCard.tsx` — NO CHANGES NEEDED. Already accepts `unreadCount?`/`hasChatMention?` (lines 173–177) and renders `<ChatBadge count={unreadCount ?? 0} hasMention={hasChatMention ?? false} />` (line 450).

---

### `frontend/src/components/Hub/useChatUnreadListeners.test.ts` (NEW — vitest unit test)

**Analog:** `HubPanel.test.tsx` — same mock setup pattern for Wails RPCs + jsdom environment.

**Mock setup pattern** (from `HubPanel.test.tsx` lines 1–32):
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'

// Mock RelayClient before import — same technique used for GetSessionStyledTailLines mock
vi.mock('../../lib/relayClient', () => ({
  RelayClient: vi.fn().mockImplementation((_port, _id, callbacks) => ({
    callbacks,
    close: vi.fn(),
  })),
}))

vi.mock('./ChatPanel', () => ({
  accrueUnread: vi.fn((prev, _msg, _id) => ({ count: prev.count + 1, hasMention: false })),
}))
```

**Test structure to cover:**
- Hook opens a RelayClient per non-modal session when `relayPort > 0` and `isActive === true`
- Hook excludes `openModalSessionId` from subscription set
- MsgChat callback calls `onUnreadChange` with correct `(sessionId, count, hasMention)`
- Hook closes RelayClient on unmount (cleanup test)
- Hook skips all clients when `relayPort === 0` (Pitfall 2)

---

### `internal/relay/server_inject_test.go` (modified — add `TestInject_ControlOnlyInput`)

**Analog:** itself. The existing test functions (`TestInject_RWCap_WritesToPTY`, `TestInject_OnlyDedicatedFrame`, `TestInject_ROCap_RelayPath`) define the exact pattern to copy.

**Test function structure to add** (copy from `TestInject_OnlyDedicatedFrame`, lines 164–191):
```go
// TestInject_ControlOnlyInput verifies IN-02: a MsgSessionInject frame carrying
// control-only text (e.g. "\x1b[2J") results in zero PTY writes.
// SanitizePTYText collapses the escape sequence to "\n"; strings.TrimSpace
// then returns "" triggering the early-return guard at hub.go:608.
func TestInject_ControlOnlyInput(t *testing.T) {
    ts, _, sessionID, ptyWriteCount := setupInjectTestServer(t)

    conn := dialInjectWS(t, ts, sessionID, "")
    time.Sleep(50 * time.Millisecond)

    // Control-only text: ESC + CSI clear-screen — SanitizePTYText collapses to "\n",
    // TrimSpace produces "", triggering the IN-02 guard in hub.go:608.
    injectPayload, _ := json.Marshal(InjectPayload{Text: "\x1b[2J"})
    frame := append([]byte{MsgSessionInject}, injectPayload...)
    ctx := context.Background()
    if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
        t.Fatalf("write control-only inject frame: %v", err)
    }

    time.Sleep(100 * time.Millisecond)

    if count := ptyWriteCount.Load(); count != 0 {
        t.Errorf("PTY write count = %d after control-only inject, want 0 (IN-02 guard failed)", count)
    }
}
```

Helper reuse: `setupInjectTestServer`, `dialInjectWS` — already defined in this file. No new helpers needed.

---

### `internal/relay/sanitize.go` (modified — doc comment correction, IN-04)

**Analog:** itself. Doc comment edit only — no behavioral change.

**Current inaccurate doc comment** (lines 143–145):
```go
//   - C0 control characters (U+0000–U+001F, including ESC, CR, LF, TAB) are
//     stripped. Stripping ESC neutralizes CSI/OSC/DCS introducers so escape
//     sequences cannot be reconstructed by a renderer.
```

**Replace with accurate statement:**
```go
//   - C0 control characters (U+0000–U+001F, including ESC, CR, LF, TAB) are
//     stripped. Stripping ESC removes the 2-byte introducer of CSI/OSC/DCS/APC/PM/SOS
//     sequences, but body bytes (which are above U+001F) survive as printable
//     plaintext in the output. DCS body content in chat is cosmetically confusing
//     but is neutralized by react-markdown + rehype-sanitize before rendering.
```

---

### `scripts/install.sh` (modified — WR-01, WR-03)

**Analog:** itself. Two surgical line edits.

**WR-01 fix** (line 77 — add `-F` flag to grep):
```sh
# Before:
EXPECTED=$(grep "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
# After:
EXPECTED=$(grep -F "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
```

**WR-03 fix** (lines 102–107 — add `mkdir -p` to root branch):
```sh
# Before:
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi
# After:
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
    mkdir -p "$INSTALL_DIR"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi
```

---

## Shared Patterns

### Functional setState with new Map reference
**Source:** `HubPanel.tsx` lines 99–108 (`usePreviewPoller` setTails call)
**Apply to:** All `setUnreadMap` calls in HubPanel and `handleUnreadChange` — never mutate the existing Map in place.
```typescript
setUnreadMap((prev) => {
  const m = new Map(prev)
  m.set(sessionId, { count, hasMention })
  return m
})
```

### Optional prop threading (add `?` + pass through)
**Source:** `HubPanel.tsx` lines 152–206 (props interface pattern — each phase adds optional props with `?`)
**Apply to:** `onUnreadChange?` on both `HubInteractiveModalProps` and `HubModalProps`. Optional because briefing-modal sessions (no chat) must not be broken.

### isActive gate in hooks
**Source:** `HubPanel.tsx` line 76 (`if (!isActive || sessions.length === 0) return`)
**Apply to:** `useChatUnreadListeners` — gate the entire hook on `isActive` for consistency with `usePreviewPoller` and to avoid idle WS connections when Hub tab is not in focus.

### sessionIdKey stable dep
**Source:** `HubPanel.tsx` line 73 (`const sessionIdKey = sessions.map((s) => s.id).join(',')`)
**Apply to:** `useChatUnreadListeners` useEffect dependency array — prevents polling storm on array reference churn.

---

## No Analog Found

None. All files have clear analogs in the codebase.

---

## Metadata

**Analog search scope:** `frontend/src/components/Hub/`, `internal/relay/`, `scripts/`
**Files scanned:** 8 analog files read (HubPanel.tsx, HubInteractiveModal.tsx, HubModal.tsx, SessionCardGrid.tsx, HubPanel.test.tsx, server_inject_test.go, sanitize.go, install.sh)
**Pattern extraction date:** 2026-06-27
