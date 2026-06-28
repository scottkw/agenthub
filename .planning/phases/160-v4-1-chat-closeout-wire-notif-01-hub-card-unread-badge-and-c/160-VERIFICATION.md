---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
verified: 2026-06-28T05:00:00Z
status: passed
score: 15/15 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification: false
---

# Phase 160: v4.1 Chat Closeout Verification Report

**Phase Goal:** Close the remaining v4.1 chat gaps so the milestone can ship. (1) NOTIF-01 — thread the Hub session-card unread badge prop chain AND provide a session-scoped unread signal for backgrounded sessions via a new hook. (2) Clear v4.1 milestone-audit tech debt: 153 IN-02 / IN-04, 154 NOTIF-02, 156 WR-01 / WR-02 / WR-03.
**Verified:** 2026-06-28
**Status:** PASSED
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A backgrounded session (modal closed) accrues an unread count when a MsgChat frame arrives over its relay WS | VERIFIED | `useChatUnreadListeners.ts`: `onChat` callback calls `accrueUnread(prev, message, 'local')` then `onUnreadChange(session.id, next.count, next.hasMention)`; 6/6 vitest tests pass including `calls onUnreadChange with correct (sessionId, count, hasMention) when onChat fires` |
| 2 | The session whose modal is open is excluded from background subscription (no double-count) | VERIFIED | `useChatUnreadListeners.ts` line 49: `if (session.id === openModalSessionId) continue`; test `excludes openModalSessionId from the subscription set` verifies only 2 clients opened for 3 sessions when one is excluded |
| 3 | When relayPort is 0 or isActive is false, the hook opens zero RelayClient connections | VERIFIED | `if (!isActive \|\| sessions.length === 0) return` (isActive gate) + `if (relayPort <= 0) continue` (port gate); two dedicated tests `constructs zero RelayClients when relayPort === 0` and `constructs zero RelayClients when isActive is false` pass |
| 4 | Closing/unmounting the hook (or removing a session) closes that session's RelayClient (no leak) | VERIFIED | Cleanup function: `return () => { for (const client of clients) { client.close() } }`; test `closes every RelayClient on unmount (no leak)` asserts all `close()` spies called once |
| 5 | When a backgrounded session accrues unread, its Hub session card shows a ChatBadge with count > 0 | VERIFIED | Full prop chain end-to-end: hook `onUnreadChange` → `HubPanel.handleUnreadChange` updates `unreadMap` → `<SessionCardGrid unreadBySessionId={unreadMap}>` → both SessionCard render sites pass `unreadCount={unreadBySessionId?.get(s.id)?.count}`; `SessionCardGrid.test.tsx` asserts badge propagation; `ChatBadge.test.tsx` asserts count > 0 renders badge |
| 6 | When the user opens a session's modal, that session's card badge resets to 0 | VERIFIED | `handleCardClick` in `HubPanel.tsx` lines 443-448: `setUnreadMap(prev => { const m = new Map(prev); m.delete(session.id); return m })` before `setModalState`; `HubPanel.test.tsx` asserts clicking card with nonzero entry resets badge |
| 7 | Unread state from the open modal's ChatPanel is lifted out to HubPanel (not trapped locally) | VERIFIED | Chain: `ChatPanel.onUnreadChange(count, hasMention)` → `HubInteractiveModal.handleUnreadChange` calls `onUnreadChange?.(session.id, count, mention)` (session.id injection) → `HubModal` threads `onUnreadChange={onUnreadChange}` to HubInteractiveModal only → `HubPanel.handleUnreadChange` updates unreadMap; `HubInteractiveModal.test.tsx` +2 tests assert sessionId injection |
| 8 | An @mention while backgrounded sets hasMention true on the card badge | VERIFIED | `accrueUnread` (imported from ChatPanel, not re-implemented) sets hasMention; threaded through unreadMap → SessionCardGrid → `hasChatMention={unreadBySessionId?.get(s.id)?.hasMention}` to SessionCard; ChatBadge renders `@` glyph when `hasMention=true` |
| 9 | A MsgSessionInject frame carrying control-only text results in zero PTY writes (no spurious Enter) | VERIFIED | `TestInject_ControlOnlyInput` in `internal/relay/server_inject_test.go` lines 231-251 asserts `ptyWriteCount.Load() == 0` after sending `"\x1b[2J"`; `go test -race -run TestInject_ControlOnly ./internal/relay/...` PASS (1.276s confirmed live) |
| 10 | SanitizeChatContent doc comment accurately states ESC stripping removes only the 2-byte introducer; DCS/APC/PM/SOS bodies survive as plaintext | VERIFIED | `internal/relay/sanitize.go` lines 143-148: "Stripping ESC removes the 2-byte introducer of CSI/OSC/DCS/APC/PM/SOS sequences, but body bytes (which are above U+001F) survive as printable plaintext in the output. DCS body content in chat is cosmetically confusing but is neutralized by react-markdown + rehype-sanitize before rendering." No behavioral change; `SanitizePTYText` comment untouched |
| 11 | install.sh matches the tarball name against checksums.txt as a fixed string | VERIFIED | `scripts/install.sh` line 77: `grep -F "${TARBALL}" "${TMPDIR}/checksums.txt"`; `tests/install-sh.test.sh` line 57 WR-01 assertion passes; `bash tests/install-sh.test.sh` 13/13 PASS confirmed live |
| 12 | install.sh creates INSTALL_DIR before copying in BOTH the root and non-root branches | VERIFIED | `scripts/install.sh` lines 103-107: root branch has `mkdir -p "$INSTALL_DIR"`, non-root branch already had it; WR-03 assertion `grep -cF 'mkdir -p "$INSTALL_DIR"'` count ≥ 2 passes |
| 13 | TESTING.md build-script Run Command runs both build-script.test.sh AND install-sh.test.sh | VERIFIED | `TESTING.md` line 31: `bash tests/build-script.test.sh && bash tests/install-sh.test.sh` |
| 14 | TESTING.md §4 has traceability rows for NOTIF-02 (verified present) and the new NOTIF-01 hub-card test | VERIFIED | Line 203: NOTIF-02 → `ChatMessage.test.tsx`; Line 226: NOTIF-01 → `useChatUnreadListeners.test.tsx` (Phase 160-01); Line 227: IN-02 → `server_inject_test.go` (Phase 160-03); no duplicate NOTIF-02 row |
| 15 | tests/check-traceability-paths.sh passes | VERIFIED | `bash tests/check-traceability-paths.sh` outputs "OK: all traceability paths exist" (confirmed live; grep -P flag warning is macOS-only and does not affect exit status) |

**Score:** 15/15 truths verified (0 present-behavior-unverified)

---

### Carried-Forward Scope (Non-Blocking)

The ROADMAP Phase 160 description includes a parenthetical appended on 2026-06-27: **"terminal bottom empty-space (xterm row-quantization, pre-existing/global — deferred from 159 UAT)"**. This item is:

- NOT covered by any of the 5 Phase 160 plans
- NOT in the phase requirement IDs
- NOT addressed by any later phase (161 alias UI, 162 settings polish, 163 RO-can-chat)
- Explicitly labeled **"pre-existing/global"** in the ROADMAP (not introduced by Phase 160)
- Explicitly labeled **"deferred"** in the ROADMAP description

Assessment: This is an acknowledged pre-existing xterm.js row-quantization quirk that causes a gap at the terminal bottom when the container height is not an exact multiple of the character cell height. It was deferred from 159 UAT scope without a specific phase assignment. It does NOT block Phase 160 completion — it does not relate to any of the 5 plans' deliverables and is noted in the ROADMAP purely as a known open item. A future phase or standalone fix should be planned to address this.

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/Hub/useChatUnreadListeners.ts` | Background unread WS hook | VERIFIED | 79 lines; reuses `accrueUnread`/`UnreadState` from ChatPanel; `useRef<Map>` for accumulation; `isActive` gate; cleanup closes all RelayClients |
| `frontend/src/components/Hub/useChatUnreadListeners.test.tsx` | Hook test suite | VERIFIED | 285 lines; 6 tests covering all 5 plan behavior cases + onChat→onUnreadChange threading |
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | onUnreadChange prop | VERIFIED | `HubInteractiveModalProps.onUnreadChange?` added; `handleUnreadChange` calls `onUnreadChange?.(session.id, count, mention)` |
| `frontend/src/components/Hub/HubModal.tsx` | onUnreadChange threading | VERIFIED | `HubModalProps.onUnreadChange?` added; threaded to `<HubInteractiveModal>` only (not HubBriefingModal) |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | unreadBySessionId prop | VERIFIED | `unreadBySessionId?: Map<string, {count, hasMention}>` added; both SessionCard render sites (named-group ~287-288, workDir-fallback ~339-340) pass `unreadCount` and `hasChatMention` |
| `frontend/src/components/Hub/HubPanel.tsx` | unreadMap wiring | VERIFIED | `unreadMap` state; `handleUnreadChange` (functional setState + new Map); `handleCardClick` deletes entry before setModalState; `useChatUnreadListeners` call with `modalState?.session.id ?? null` exclusion; `<HubModal onUnreadChange={handleUnreadChange}>`; `<SessionCardGrid unreadBySessionId={unreadMap}>` |
| `internal/relay/server_inject_test.go` | IN-02 regression test | VERIFIED | `TestInject_ControlOnlyInput` added at line 231; reuses `setupInjectTestServer`/`dialInjectWS`; no new helpers; no production code changed |
| `scripts/install.sh` | WR-01 + WR-03 hardening | VERIFIED | Line 77: `grep -F`; lines 103-104: root branch has `mkdir -p "$INSTALL_DIR"` |
| `tests/install-sh.test.sh` | WR-01 + WR-03 assertions | VERIFIED | Lines 57/61-63: assert_literal for `grep -F "${TARBALL}"` (WR-01) + `grep -cF 'mkdir -p "$INSTALL_DIR"'` count ≥ 2 (WR-03); 13/13 PASS |
| `internal/relay/sanitize.go` | IN-04 doc comment correction | VERIFIED | `SanitizeChatContent` comment corrected at lines 143-148; no behavioral change; `SanitizePTYText` untouched |
| `TESTING.md` | WR-02 + counts + §4 rows | VERIFIED | Line 31 Run Command includes both test scripts; vitest=132; Total=509; §4 NOTIF-01/IN-02/NOTIF-02 rows present; check-traceability-paths.sh PASS |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `ChatPanel.onUnreadChange(count, hasMention)` | `HubPanel.handleUnreadChange(sessionId, count, hasMention)` | `HubInteractiveModal.handleUnreadChange` wraps with `session.id` → `HubModal.onUnreadChange` prop | WIRED | Confirmed in HubInteractiveModal.tsx line 72: `onUnreadChange?.(session.id, count, mention)` |
| `HubPanel.unreadMap` | `SessionCardGrid (both render sites)` | `unreadBySessionId={unreadMap}` prop | WIRED | HubPanel.tsx line 522; SessionCardGrid.tsx lines 287-288 (named-group) and 339-340 (workDir) |
| `HubPanel` → `useChatUnreadListeners` | backgrounded sessions | `modalState?.session.id ?? null` as exclusion id | WIRED | HubPanel.tsx lines 356-362; exclusion id prevents double-count |
| `useChatUnreadListeners` | `accrueUnread` in ChatPanel | `import { accrueUnread } from './ChatPanel'` | WIRED | Confirmed in useChatUnreadListeners.ts line 23; `export function accrueUnread` confirmed at ChatPanel.tsx line 185 |
| `tests/install-sh.test.sh` WR-01 | `scripts/install.sh` grep -F line | Static `grep -F "${TARBALL}"` string match | WIRED | `assert_literal` at line 57 of test file |
| `TESTING.md` §4 rows | Actual test files | Path-column repo-relative paths | WIRED | `bash tests/check-traceability-paths.sh` PASS |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| IN-02: control-only inject writes zero PTY bytes | `go test -race -run TestInject_ControlOnly ./internal/relay/...` | `ok internal/relay 1.276s` | PASS |
| WR-01/WR-03: install-sh.test.sh assertions | `bash tests/install-sh.test.sh` | `Results: 13 passed, 0 failed` | PASS |
| tsc: no TypeScript errors in modified files | `cd frontend && pnpm exec tsc --noEmit 2>&1 \| grep -c "error TS"` | `0` (zero errors) | PASS |
| Vitest count matches TESTING.md | `find frontend/src -name "*.test.ts" -o -name "*.test.tsx" \| wc -l` | `132` — matches TESTING.md line 29 | PASS |
| traceability-paths pass | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | PASS |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| NOTIF-01 | 160-01, 160-02 | Hub session-card unread badge wired end-to-end (hub-card badge BLOCKER closed) | SATISFIED | Full prop chain verified; `useChatUnreadListeners` hook for backgrounded sessions; `unreadMap` in HubPanel fed by both sources |
| NOTIF-02 | 160-05 | @mention traceability row verified present in TESTING.md §4 | SATISFIED | TESTING.md line 203 confirmed; no implementation gap (feature shipped in Phase 154) |
| IN-02 | 160-03 | Regression test for control-only inject → zero PTY writes | SATISFIED | `TestInject_ControlOnlyInput` added and passes |
| IN-04 | 160-05 | SanitizeChatContent doc comment corrected | SATISFIED | Comment at sanitize.go lines 143-148 accurately describes introducer-only stripping + surviving body bytes |
| WR-01 | 160-04 | `grep -F` for exact tarball checksum match | SATISFIED | `scripts/install.sh` line 77; test assertion passes |
| WR-02 | 160-05 | TESTING.md build-script Run Command includes both test scripts | SATISFIED | TESTING.md line 31 |
| WR-03 | 160-04 | Root branch of install.sh creates INSTALL_DIR with `mkdir -p` | SATISFIED | `scripts/install.sh` lines 103-104; test assertion passes |

Note: IN-02, IN-04, WR-01, WR-02, WR-03 are milestone-audit tech-debt identifiers, not top-level REQUIREMENTS.md entries. They appear in `.planning/v4.1-MILESTONE-AUDIT.md`. NOTIF-01 and NOTIF-02 are formal REQUIREMENTS.md requirements (lines 48-49), both SATISFIED.

---

### Anti-Patterns Found

No blockers. Scanning Phase 160 modified files:

- `useChatUnreadListeners.ts`: No TBD/FIXME/XXX/TODO/PLACEHOLDER. No empty implementations. `useRef` for state (correct — not useState). No hardcoded data.
- `useChatUnreadListeners.test.tsx`: Test-only mock patterns (`accrueUnread` returns `count + 1`); appropriate for threading tests.
- `HubInteractiveModal.tsx`, `HubModal.tsx`, `SessionCardGrid.tsx`, `HubPanel.tsx`: No debt markers. Functional setState patterns correct (`new Map(prev)` throughout). `m.delete(session.id)` correctly prevents badge flicker on modal open.
- `server_inject_test.go`: Test-only file. No production code changed (confirmed `git log -- internal/relay/hub.go` last touch was Phase 157, not Phase 160).
- `scripts/install.sh`: `sh -n` passes; shellcheck clean.
- `internal/relay/sanitize.go`: Comment-only change. `SanitizePTYText` untouched. `go build ./internal/relay/... && go vet ./internal/relay/...` passes.
- `TESTING.md`: No orphaned or duplicate rows.

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| (none) | — | — | — |

**Prohibition compliance:**
- `SessionCard.tsx` was NOT modified (last touch: Phase 154 commit `63e46d3f`)
- `hub.go` was NOT modified (last touch: Phase 157 commit `3c23414c`)
- `onUnreadChange` is NOT threaded into `HubBriefingModal` (verified in HubModal.tsx lines 233-252)
- `useChatUnreadListeners` uses `useRef` not `useState` for per-session accumulation
- `unreadMap` always mutated via `new Map(prev)` (never in-place)
- NOTIF-02 row not duplicated (confirmed single row at line 203)

---

### Human Verification Required

None. All must-haves are verified programmatically.

---

### Gaps Summary

No gaps. All 15 must-haves are VERIFIED.

**Open item (non-blocking):** The "terminal bottom empty-space (xterm row-quantization, pre-existing/global)" issue appended to the Phase 160 ROADMAP description is not addressed by any Phase 160 plan and is not covered by phases 161, 162, or 163. It is a pre-existing xterm.js behavior noted as deferred. It should be tracked as an open item and planned separately — it is NOT a Phase 160 deliverable gap.

---

_Verified: 2026-06-28_
_Verifier: Claude (gsd-verifier)_
