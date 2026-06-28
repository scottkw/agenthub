# Phase 160: v4.1 Chat Closeout — Research

**Researched:** 2026-06-27
**Domain:** React prop-threading, lightweight WebSocket subscription, shell-script hardening, TESTING.md doc fixes
**Confidence:** HIGH

---

## Summary

Phase 160 has two workstreams. The first and only blocker is **NOTIF-01**: the Hub session-card unread badge is dead-wired. The component tree is fully audited at HEAD — `SessionCard` already renders `<ChatBadge>` when `unreadCount`/`hasChatMention` props are supplied, but `SessionCardGrid` never passes them, and `HubInteractiveModal` holds unread state locally without an `onUnreadChange` prop on its interface. There is also a deeper architectural gap: unread accrual lives inside `ChatPanel`, which only mounts while the session modal is open — backgrounded sessions have no WS subscription and cannot accrue unread at all. The fix requires (a) prop threading up the HubInteractiveModal → HubModal → HubPanel → SessionCardGrid chain, and (b) a lightweight per-session WS listener in HubPanel for sessions whose modal is closed.

The second workstream is **tech debt closure** (IN-02, IN-04, NOTIF-02, WR-01, WR-02, WR-03). Source inspection shows that several of these are already partially or fully resolved at HEAD: hub.go already implements the IN-02 guard (`strings.TrimSpace(sanitized) == ""`), the NOTIF-02 traceability row already exists in TESTING.md §4, and the Phase 156 VALIDATION.md was already reconciled on 2026-06-27 by `/gsd-validate-phase 156`. The remaining live work is: (1) add a test for the IN-02 behavior, (2) fix the IN-04 doc comment in `SanitizeChatContent`, (3) add `-F` to install.sh's grep (WR-01), (4) update TESTING.md's build-script Run Command to include `install-sh.test.sh` (WR-02), and (5) add `mkdir -p "$INSTALL_DIR"` to the root branch of install.sh (WR-03).

**Primary recommendation:** Implement NOTIF-01 via prop-threading + a new `useChatUnreadListeners` hook (Option A — lightweight WS per background session using the existing `RelayClient`). Desktop owner TailnetID is always `"local"`, so mention detection in the background hook requires no new API. The background hook and modal threading share a single `unreadMap` state in `HubPanel`; the hub card badge is GUI-only (web-share has no session card grid — out of scope for this phase).

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NOTIF-01 | Unread badge on Hub session card when messages arrive while modal is closed | Prop-threading chain + background WS listener hook identified; exact files and symbols documented below |
| IN-02 | Control-only inject text must not press Enter at PTY | hub.go:608 already has the guard; need a missing test case |
| IN-04 | SanitizeChatContent doc comment overstates DCS/APC/PM/SOS coverage | sanitize.go:134–171 doc comment fix only — no behavioral change |
| NOTIF-02 | Dedicated TESTING.md §4 traceability row for NOTIF-02 | Row already exists at TESTING.md:203; planner should verify and close as no-op |
| WR-01 | install.sh grep without -F: dots are regex wildcards | install.sh:77 — add `-F` flag |
| WR-02 | TESTING.md build-script Run Command omits install-sh.test.sh | TESTING.md:31 — update Run Command cell |
| WR-03 | install.sh root branch sets INSTALL_DIR without mkdir -p | install.sh:102–103 — add mkdir -p before cp |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Hub card unread badge | Browser / Client (HubPanel) | — | Unread state lifted to HubPanel; threaded to SessionCardGrid → SessionCard |
| Background unread accrual | Browser / Client (useChatUnreadListeners hook) | API / Backend (relay WS) | Relay broadcasts MsgChat to all subscribers; frontend hook counts them |
| Modal-open unread accrual | Browser / Client (ChatPanel → onUnreadChange) | — | ChatPanel already accrues unread; prop threading lifts it to HubPanel |
| install.sh fixes | CDN / Static (dist script) | — | Shell script for the Linux distribution path; no daemon changes needed |

---

## NOTIF-01: Root Cause and Fix Architecture

### What is Broken at HEAD

**Component tree (audited via source inspection):**

```
HubPanel
├── SessionCardGrid  ← no unreadBySessionId prop
│   └── SessionCard  ← accepts unreadCount?/hasChatMention? (lines 173-177),
│                       renders <ChatBadge> (line 450) — NO CHANGES NEEDED HERE
└── HubModal         ← no onUnreadChange prop
    └── HubInteractiveModal  ← has local unreadCount/hasMention state (lines 57-58)
                                and handleUnreadChange (lines 60-63),
                                but NO onUnreadChange on HubInteractiveModalProps (line 12-28)
        └── ChatPanel  ← fires onUnreadChange; always-mounted while modal is open (D-09)
```

**Two distinct gaps:**

1. **Prop threading gap (modal-open case):** `HubInteractiveModalProps` has no `onUnreadChange`; `HubModalProps` has no `onUnreadChange`; `SessionCardGridProps` has no unread map. Unread state computed inside `HubInteractiveModal` is never lifted to `HubPanel`. File: `HubInteractiveModal.tsx` lines 12–28 (props) and 57–63 (state).

2. **Closed-modal gap (backgrounded sessions):** `ChatPanel` only mounts while the session modal is open. When the modal closes (`HubModal` unmounts), the ChatPanel WS subscription terminates and unread count is lost. For sessions where no modal is open, there is no subscription source.

### Recommended Fix: Option A — Prop Threading + Background WS Listeners

**Part A: Prop threading chain (modal-open case)**

Files to change, in dependency order:

| File | Change |
|------|--------|
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | Add `onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void` to `HubInteractiveModalProps`. Call `props.onUnreadChange?.(session.id, count, mention)` inside the existing `handleUnreadChange` function. |
| `frontend/src/components/Hub/HubModal.tsx` | Add `onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void` to `HubModalProps`. Thread to `<HubInteractiveModal onUnreadChange={onUnreadChange} ...>`. (HubBriefingModal has no chat — no threading needed there.) |
| `frontend/src/components/Hub/HubPanel.tsx` | Add `unreadMap: Map<string, {count: number, hasMention: boolean}>` state (initialized empty). Add `handleUnreadChange(sessionId, count, hasMention)` callback that updates the map. Pass callback to `<HubModal onUnreadChange={handleUnreadChange} ...>`. On `handleCardClick` (before setModalState), reset the opening session's entry: `setUnreadMap(prev => { const m = new Map(prev); m.delete(session.id); return m })`. Pass `unreadMap` to `<SessionCardGrid unreadBySessionId={unreadMap} ...>`. |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | Add `unreadBySessionId?: Map<string, {count: number, hasMention: boolean}>` to `SessionCardGridProps`. At both SessionCard render sites (named-group path lines 266–282, workDir path lines 316+), pass `unreadCount={unreadBySessionId?.get(s.id)?.count}` and `hasChatMention={unreadBySessionId?.get(s.id)?.hasMention}` to `<SessionCard>`. |

`SessionCard.tsx` — **no changes needed**: already accepts `unreadCount?`/`hasChatMention?` (lines 173–177, 214–215) and already renders `<ChatBadge count={unreadCount ?? 0} hasMention={hasChatMention ?? false} />` (line 450).

**Part B: Background WS listeners (closed-modal case)**

Create a custom hook `useChatUnreadListeners` (can live in `frontend/src/components/Hub/useChatUnreadListeners.ts` or inline in `HubPanel.tsx`):

```typescript
// Pattern outline — source: HubPanel architecture (codebase verified)
function useChatUnreadListeners(
  sessions: SessionInfo[],
  relayPort: number,
  openModalSessionId: string | null,
  onUnreadChange: (sessionId: string, count: number, hasMention: boolean) => void,
): void {
  // For each session where session.id !== openModalSessionId AND relayPort > 0:
  //   Open a RelayClient with only onChat wired (onOutput is no-op).
  //   On MsgChat: call accrueUnread (imported from ChatPanel.tsx — already exported),
  //   then call onUnreadChange(sessionId, newCount, newHasMention).
  //   Keep per-session UnreadState in a useRef Map (no re-renders from the listener).
  //   Cleanup: close RelayClient on session removal or openModalSessionId change.
  // currentUserTailnetID for desktop Hub is always "local" (owner identity).
}
```

Key implementation notes:
- `RelayClient` requires `onOutput` (not optional) — pass `() => {}` as no-op
- `accrueUnread(prev, message, "local")` is already exported from `ChatPanel.tsx` — import and reuse it
- Use `useRef` (not `useState`) for per-session `UnreadState` accumulation to avoid re-renders on every message; only call `onUnreadChange` (which updates HubPanel's state) when the count changes
- When `openModalSessionId` changes (modal opens), the hook should NOT reset the new open session's count — HubPanel's `handleCardClick` already resets the map entry before calling `setModalState`. The hook just stops listening for the newly-opened session.
- The hook should only run for LOCAL sessions (not remote) where the relay is reachable

**Unread reset semantics:**
- Unread resets to 0 when: the user opens the modal for that session (`handleCardClick` in HubPanel)
- Implementation: `handleCardClick` deletes the session's map entry (see Part A above) before calling `setModalState`
- The card badge shows 0 immediately when the modal opens (user is actively engaging with the session)

**Cross-surface scope confirmation:**
- NOTIF-01 (Hub card badge) is **desktop GUI only**
- `SessionCard` / `SessionCardGrid` are not used in `WebShareSessionView` (the web-share surface shows a single-session panel, not a grid)
- The in-modal chat toggle badge (clause a of NOTIF-01) already works via the existing HubInteractiveModal local state; prop threading does not break it

---

## Tech Debt Items: Exact File Locations

### IN-02 — Control-only inject: code already fixed, test missing

**Status at HEAD:** The behavioral fix is ALREADY implemented.

`internal/relay/hub.go` line 601–610:
```go
sanitized := SanitizePTYText(text)
// IN-02: control-only input (e.g. "\x1b[2J" or "\x00") is non-empty and so
// survives the read-pump ip.Text != "" guard, but SanitizePTYText collapses
// it to a bare "\n". Treat a whitespace-only post-sanitize result as empty:
// skip BOTH the PTY write (no spurious Enter keystroke) and the chat persist/broadcast.
if strings.TrimSpace(sanitized) == "" {
    return nil
}
```

`internal/relay/server.go` line 396 (the read-pump guard — runs BEFORE HandleInject):
```go
if json.Unmarshal(payload, &ip) != nil || ip.Text == "" {
    continue // malformed or empty frame: ignore silently
}
```

The `ip.Text == ""` guard runs before sanitization, so a control-only inject like `"\x1b[2J"` passes through (it is non-empty at the raw level) and reaches `HandleInject`, where the `TrimSpace` guard correctly blocks it.

**What Phase 160 must do:** Add a test case to `internal/relay/server_inject_test.go` (or a new `hub.go` test) that proves: sending `"\x1b[2J"` via a MsgSessionInject frame results in `ptyWriteCount == 0`. The code is correct; the coverage is missing.

### IN-04 — SanitizeChatContent doc comment overstates DCS coverage

**File:** `internal/relay/sanitize.go` lines 134–171

**Current doc comment (lines 143–145) — inaccurate:**
```go
//   - C0 control characters (U+0000–U+001F, including ESC, CR, LF, TAB) are
//     stripped. Stripping ESC neutralizes CSI/OSC/DCS introducers so escape
//     sequences cannot be reconstructed by a renderer.
```

**What actually happens:** `SanitizeChatContent` strips ESC (it is in the C0 range, 0x00–0x1F). For a DCS sequence `\x1bPbody\x1b\`, the ESC bytes are stripped but `P`, the body bytes, and `\` all survive (they are printable or above C0). The body leaks as plaintext into `chat.jsonl` and broadcast frames.

**Security impact:** Low. The leaked bytes are rendered in the frontend via `react-markdown` + `rehype-sanitize`, which neutralizes any HTML/script injection. DCS body content appearing as plaintext in a chat thread is cosmetically confusing but not a security bypass.

**What Phase 160 must do:** Correct the doc comment at line 143–145 to say that ESC stripping removes the 2-byte introducer sequence but DCS/APC/PM/SOS bodies (whose bytes are above U+001F) survive as printable plaintext in the output. No code change is needed — this is a documentation correction only.

Note: `SanitizePTYText` correctly handles DCS bodies via the state machine (`stateString` at lines 41, 88–91, 116). The comment at line 22–24 of `SanitizePTYText` ("the body never leaks as text (IN-04)") is accurate. Only `SanitizeChatContent` has the misleading comment.

### NOTIF-02 — TESTING.md §4 traceability row

**Status at HEAD: Row already exists.**

`TESTING.md` line 203:
```
| NOTIF-02 | frontend/src/components/Hub/ChatMessage.test.tsx | vitest | Phase 154-03: ChatMessage renders @mention text highlight — message with Mentions:["local"] and currentUserAlias="local" renders with .chat-msg--mention class and highlighted span in the content (NOTIF-02 @mention visual indicator, carried as traceability gap from Phase 154 VERIFICATION). |
```

The audit flagged this gap at the time it was written; the row has since been added (note "carried as traceability gap from Phase 154 VERIFICATION" in the row text).

**What Phase 160 must do:** Verify the row is present (it is). No TESTING.md edit is needed for NOTIF-02. Close as no-op.

### WR-01 — install.sh grep missing -F flag

**File:** `scripts/install.sh` line 77

**Current (buggy):**
```sh
EXPECTED=$(grep "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
```

`TARBALL` is e.g. `agenthub-v1.2.3-linux-amd64.tar.gz`. Without `-F`, dots are regex wildcards — `agenthub-v1X2X3-linux-amd64Xtar.gz` would also match. This is a false-negative risk only (a corrupt checksum file with a similar-named line could match); the SHA256 integrity guard still fires if the hash doesn't match, so this is not a security bypass.

**Fix:** Add `-F` flag:
```sh
EXPECTED=$(grep -F "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
```

### WR-02 — TESTING.md build-script Run Command omits install-sh.test.sh

**File:** `TESTING.md` line 31 (Suite Manifest table)

**Current:**
```
| build-script | **2** `build-script.test.sh, install-sh.test.sh` | `tests/` | `bash tests/build-script.test.sh` | Go build + Wails asset embedding |
```

The Count column correctly lists both files; the Run Command shows only `bash tests/build-script.test.sh`. `tests/build-script.test.sh` tests `build.sh` only; it does NOT invoke `install-sh.test.sh`. CI runs both scripts separately, but a developer running the documented command would miss the install script tests.

**Fix:** Update Run Command cell to:
```
`bash tests/build-script.test.sh && bash tests/install-sh.test.sh`
```

### WR-03 — install.sh root branch missing mkdir -p

**File:** `scripts/install.sh` lines 102–107

**Current:**
```sh
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi
```

The non-root branch has `mkdir -p "$INSTALL_DIR"`. The root branch sets `/usr/local/bin` without ensuring the directory exists. On standard FHS Linux distros this is benign, but on minimal containers (e.g., `alpine:latest`) `/usr/local/bin` does not always exist.

**Fix:** Add `mkdir -p "$INSTALL_DIR"` to the root branch:
```sh
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
    mkdir -p "$INSTALL_DIR"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi
```

### 156 VALIDATION.md — Already resolved

`/Users/ken/dev/agenthub/.planning/phases/156-install-links-distribution/156-VALIDATION.md` already shows `status: validated` and `nyquist_compliant: true` (reconciled 2026-06-27 via `/gsd-validate-phase 156`). No action needed.

---

## Architecture Patterns

### Hub Unread Map State Flow

```
[RelayClient per background session] ─── onChat ──► useChatUnreadListeners hook
                                                           │
                                          accrueUnread("local") → setUnreadMap(sessionId, state)
                                                           │
[ChatPanel inside open modal] ──── onUnreadChange ────────┤ (prop-threaded through HubModal → HubInteractiveModal)
                                                           │
                               HubPanel: unreadMap (Map<sessionId, UnreadState>)
                                                           │
                                          SessionCardGrid (unreadBySessionId prop)
                                                           │
                                             SessionCard (unreadCount, hasChatMention)
                                                           │
                                             <ChatBadge count={unreadCount ?? 0} ... />
```

Entry points:
- **Background sessions (modal closed):** `useChatUnreadListeners` WS connections → MsgChat frames → `accrueUnread` → `setUnreadMap`
- **Active session (modal open):** `ChatPanel.onUnreadChange` → `HubInteractiveModal.handleUnreadChange` → new `onUnreadChange` prop → `HubModal.onUnreadChange` → `HubPanel.handleUnreadChange` → `setUnreadMap`
- **Modal opens:** `HubPanel.handleCardClick` resets the session's unread map entry to 0 before `setModalState`
- **Result:** `SessionCardGrid` always has the current count per session; `SessionCard` renders badge when count > 0

### RelayClient Usage Pattern for Background Listener

```typescript
// Minimal listener — onOutput is required but unused for background badge tracking
const client = new RelayClient(relayPort, sessionId, {
  onOutput: () => {},        // no-op — background listener never renders PTY output
  onChat: (message) => {
    const next = accrueUnread(prev, message, 'local') // 'local' = desktop owner always
    onUnreadChange(sessionId, next.count, next.hasMention)
  },
})
```

`RelayClient` constructor: `frontend/src/lib/relayClient.ts` lines 213–283. Connects to `ws://127.0.0.1:${port}/sessions/${sessionId}/ws` (local relay path, no remote/wsURL needed for Hub local sessions).

### Prop Threading Callback Shape

`onUnreadChange` signature used at every level of the threading chain must include `sessionId` (unlike ChatPanel's internal callback which is `(count, hasMention)` without the ID, since ChatPanel knows its own session):

```typescript
// HubInteractiveModalProps and HubModalProps:
onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void
```

`ChatPanel.onUnreadChange` is `(count: number, hasMention: boolean) => void` (no sessionId). `HubInteractiveModal.handleUnreadChange` wraps it, adding `session.id`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Unread count per session | Custom polling loop or WebSocket parser | `RelayClient` with `onChat` callback | RelayClient already handles framing, reconnect, ping; `parseServerFrame` handles 0x30 |
| Mention detection in background listener | Separate mention parser | `accrueUnread(prev, msg, 'local')` from `ChatPanel.tsx` | Already exported, pure, unit-tested |
| Map-based state updates | Spread operator on Map | Functional setState with `new Map(prev)` | React won't re-render on mutated Map; must produce new Map reference |

---

## Common Pitfalls

### Pitfall 1: Double-counting when modal is open
**What goes wrong:** If the background `useChatUnreadListeners` hook keeps a WS open for a session whose modal is also open, `ChatPanel` and the background hook both receive the same MsgChat frame, doubling the count.
**Prevention:** `useChatUnreadListeners` must exclude `openModalSessionId` from its subscription set. Pass `modalState?.session.id ?? null` as the exclusion parameter.

### Pitfall 2: Stale relay port (relayPort=0 on mount)
**What goes wrong:** `relayPort` is 0 on initial render; creating a RelayClient with port 0 constructs `ws://127.0.0.1:0/...` which fails immediately.
**Prevention:** Gate hook execution on `relayPort > 0` (same guard used in HubPanel line 525).

### Pitfall 3: Memory leak on session removal
**What goes wrong:** If a session is killed while its background listener is running, the WS connection stays open after the session disappears.
**Prevention:** The hook's dependency array includes session IDs. When a session is removed from the list, the effect cleanup closes its RelayClient.

### Pitfall 4: onOutput required by RelayClient
**What goes wrong:** `RelayClient` constructor's `callbacks` parameter has `onOutput: (data: Uint8Array) => void` — NOT optional (no `?`). Omitting it causes a TypeScript error.
**Prevention:** Pass `onOutput: () => {}` as an explicit no-op.

### Pitfall 5: Map mutation doesn't trigger re-render
**What goes wrong:** `unreadMap.set(key, val)` mutates the existing Map reference; React's `setState` bail-out skips re-render since the reference didn't change.
**Prevention:** Always produce a new Map: `setUnreadMap(prev => { const m = new Map(prev); m.set(key, val); return m })`.

### Pitfall 6: IN-02 code vs test gap
**What goes wrong:** Treating IN-02 as needing a code fix when the code is already correct. hub.go:608 is the fix.
**Prevention:** Phase 160 adds a TEST, not a code change, for IN-02. The test proves `HandleInject("\x1b[2J")` returns nil AND does not call WriteInput.

---

## Code Examples

### Verified: ChatPanel's accrueUnread (exported, reusable)
```typescript
// Source: frontend/src/components/Hub/ChatPanel.tsx lines 185-194
export function accrueUnread(
  prev: UnreadState,
  message: ChatMessage,
  currentUserTailnetID: string,
): UnreadState {
  const hasMention =
    prev.hasMention ||
    !!(message.mentions?.includes(currentUserTailnetID))
  return { count: prev.count + 1, hasMention }
}
```

### Verified: HubInteractiveModal unread state (current, before threading)
```typescript
// Source: frontend/src/components/Hub/HubInteractiveModal.tsx lines 54-63
const [unreadCount, setUnreadCount] = useState(0)
const [hasMention, setHasMention] = useState(false)

function handleUnreadChange(count: number, mention: boolean) {
  setUnreadCount(count)
  setHasMention(mention)
}
```
Phase 160 adds a call to `props.onUnreadChange?.(session.id, count, mention)` inside `handleUnreadChange`.

### Verified: SessionCard badge render (no changes needed)
```typescript
// Source: frontend/src/components/Hub/SessionCard.tsx lines 443-450
{/* NOTIF-01 / D-10: ChatBadge appears right of the session name when unreadCount > 0. */}
<ChatBadge count={unreadCount ?? 0} hasMention={hasChatMention ?? false} />
```

### Verified: install.sh grep line (WR-01 fix site)
```sh
# Source: scripts/install.sh line 77 — current (buggy)
EXPECTED=$(grep "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
# Fix: add -F
EXPECTED=$(grep -F "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
```

### Verified: install.sh root branch (WR-03 fix site)
```sh
# Source: scripts/install.sh lines 102-107 — current (buggy: no mkdir for root)
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi
```

---

## Environment Availability

Step 2.6: SKIPPED — Phase 160 is pure code, test, and doc changes within the existing project. No new external tools, services, CLIs, or runtimes are required beyond what the project already uses (Go, Node/pnpm, shellcheck, bash).

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest (frontend) + Go testing (backend) + shellcheck/bash (scripts) |
| Config file | `frontend/vitest.config.ts` |
| Quick run command | `go test -race -short ./internal/relay/... && cd frontend && pnpm vitest run src/components/Hub/` |
| Full suite command | `go test -race -short ./... && bash tests/install-sh.test.sh && cd frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NOTIF-01 (prop threading) | HubPanel.onUnreadChange updates unreadMap; SessionCardGrid threads to SessionCard | unit/vitest | `cd frontend && pnpm vitest run src/components/Hub/HubPanel` | No — new test needed |
| NOTIF-01 (background WS) | useChatUnreadListeners opens WS per non-modal session; increments on MsgChat | unit/vitest | `cd frontend && pnpm vitest run src/components/Hub/useChatUnreadListeners` | No — new test needed |
| NOTIF-01 (card badge) | SessionCardGrid threads unreadBySessionId to SessionCard | unit/vitest | `cd frontend && pnpm vitest run src/components/Hub/SessionCardGrid` | No — extend SessionCardGrid.test.tsx |
| IN-02 | HandleInject("\x1b[2J") → ptyWriteCount==0 | Go unit | `go test -run TestInject_ControlOnlyInput ./internal/relay/...` | No — add to server_inject_test.go |
| IN-04 | SanitizeChatContent doc comment accuracy | doc-only | n/a (source review) | N/A — doc edit only |
| NOTIF-02 | TESTING.md §4 traceability row | doc-only | `bash tests/check-traceability-paths.sh` | Already present — verify |
| WR-01 | install.sh grep uses -F flag | static | `bash tests/install-sh.test.sh` | Extend install-sh.test.sh |
| WR-02 | TESTING.md Run Command includes install-sh.test.sh | doc-only | n/a | TESTING.md edit |
| WR-03 | install.sh root branch has mkdir -p | static/behavioral | `bash tests/install-sh.test.sh` | Extend install-sh.test.sh |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm vitest run src/components/Hub/ --passWithNoTests` (vitest) or `go test -race -short ./internal/relay/... ./internal/webserver/...` (Go)
- **Per wave merge:** `go test -race -short ./... && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/Hub/useChatUnreadListeners.ts` — new hook file (or inline in HubPanel)
- [ ] Test for `useChatUnreadListeners` — new vitest file
- [ ] Test for HubPanel unread threading — new or extended vitest file
- [ ] `TestInject_ControlOnlyInput` in `internal/relay/server_inject_test.go` or `hub_chatsend_test.go`

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — |
| V3 Session Management | No | — |
| V4 Access Control | No | — |
| V5 Input Validation | Yes (IN-02/IN-04 scope) | `SanitizePTYText` + `SanitizeChatContent` in relay/sanitize.go |
| V6 Cryptography | No | — |

No new attack surface. The background WS listener is read-only (no frames sent to server except the implicit WS connection handshake). The RelayClient's local relay path (`ws://127.0.0.1:{relayPort}`) is loopback-only and already in use by TerminalPanel.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | NOTIF-02 traceability row (TESTING.md:203) was added after the audit and is present at HEAD | Tech Debt: NOTIF-02 | If the row is absent, Phase 160 must add it (one-line TESTING.md edit — low risk) |
| A2 | Phase 156 VALIDATION.md already shows `status: validated` at HEAD | Tech Debt: VALIDATION.md | If still `draft`, Phase 160 must update it — trivial YAML front-matter edit |
| A3 | Desktop Hub is always the local owner; TailnetID = "local" is always correct for mention detection in the background listener | NOTIF-01 architecture | If the Hub ever shows remote-owned sessions where the local machine is not the owner, mention detection for `"local"` would be incorrect for that session. Not relevant for v4.1 scope. |

---

## Open Questions

1. **Should the background listener hook pause when Hub tab is inactive?**
   - What we know: `isActive` is passed to `HubPanel` and gates `usePreviewPoller`
   - What's unclear: Whether idle WS connections when Hub tab is not in focus are a concern
   - Recommendation: Gate `useChatUnreadListeners` on `isActive` for consistency with `usePreviewPoller`

2. **SessionCard unread reset behavior when modal closes without the user reading chat**
   - What we know: Unread resets when modal OPENS (handleCardClick). When modal closes, the ChatPanel's final `onUnreadChange` state is preserved in `unreadMap`.
   - What's unclear: Whether the card badge should persist after the modal closes if the user opened the modal but did NOT open the chat drawer (messages were received while modal was open, drawer closed).
   - Recommendation: Yes, persist. If the drawer was never opened, messages arrived while the modal was open (ChatPanel mounted, drawer closed) → `onUnreadChange` fired → count > 0. When modal closes, that count stays in `unreadMap`. The background listener takes over and will add new messages. The card badge correctly shows "unread since last modal open." Semantics are consistent.

---

## Sources

### Primary (HIGH confidence)
- Source code inspection: `frontend/src/components/Hub/HubInteractiveModal.tsx` (lines 12–104) — verified component props and local state
- Source code inspection: `frontend/src/components/Hub/SessionCardGrid.tsx` (lines 133–341) — verified missing unread props
- Source code inspection: `frontend/src/components/Hub/SessionCard.tsx` (lines 173–177, 214–215, 450) — verified badge rendering ready
- Source code inspection: `frontend/src/components/Hub/HubModal.tsx` (lines 37–252) — verified no onUnreadChange prop
- Source code inspection: `frontend/src/components/Hub/HubPanel.tsx` (lines 142–564) — verified state model and modal flow
- Source code inspection: `frontend/src/components/Hub/ChatPanel.tsx` (lines 64–89, 185–194) — verified accrueUnread export and onUnreadChange callback
- Source code inspection: `frontend/src/lib/relayClient.ts` (lines 192–283) — verified RelayClient callback interface and onOutput requirement
- Source code inspection: `internal/relay/hub.go` (lines 587–646) — verified IN-02 fix present at line 608
- Source code inspection: `internal/relay/sanitize.go` (lines 1–185) — verified IN-04 doc comment issue in SanitizeChatContent
- Source code inspection: `internal/relay/server.go` (lines 390–410) — verified inject read-pump ip.Text guard
- Source code inspection: `scripts/install.sh` (lines 52–124) — verified WR-01 grep at line 77, WR-03 root branch at lines 102–107
- Source code inspection: `TESTING.md` (lines 31, 203) — verified NOTIF-02 row present, WR-02 Run Command gap confirmed
- Source code inspection: `.planning/phases/156-install-links-distribution/156-VALIDATION.md` — verified already validated

### Secondary (MEDIUM confidence)
- `.planning/v4.1-MILESTONE-AUDIT.md` — audit findings cross-referenced against source; tech_debt YAML block used as task list

---

## Metadata

**Confidence breakdown:**
- NOTIF-01 architecture: HIGH — all relevant component source verified at HEAD; exact prop chain, hook pattern, and reset semantics are concrete
- Tech debt items: HIGH — every item location verified in source; most unexpected finding is IN-02 already fixed and NOTIF-02 row already added
- WR-01/WR-02/WR-03: HIGH — install.sh and TESTING.md lines confirmed at HEAD

**Research date:** 2026-06-27
**Valid until:** 2026-07-27 (stable frontend codebase; no fast-moving dependencies)
