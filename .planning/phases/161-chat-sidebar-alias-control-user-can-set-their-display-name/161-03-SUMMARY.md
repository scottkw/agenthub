---
phase: 161-chat-sidebar-alias-control-user-can-set-their-display-name
plan: "03"
subsystem: frontend
tags: [alias, chat, ui, tdd, cross-surface-parity]
requires: [161-01, 161-02]
provides: [ALIAS-UI-01, ALIAS-UI-02]
affects: [ChatPanel, style.css, TESTING.md]
tech_stack:
  added: []
  patterns: [TDD RED/GREEN, module-scope pure helper, onSelf callback, controlled input, CSS token reuse]
key_files:
  created: []
  modified:
    - frontend/src/components/Hub/ChatPanel.tsx
    - frontend/src/components/Hub/ChatPanel.test.tsx
    - frontend/src/style.css
    - TESTING.md
decisions:
  - "validateAlias uses Array.from (code points) not String.length — mirrors Go []rune cast exactly"
  - "handleAliasCommit has NO isReadOnly early-return — D-06: alias-set is the RO exception"
  - "currentAlias priority: onSelf.alias > local:local roster entry > empty (no parallel client alias store)"
  - "aliasError shown when validateAlias returns null (server sends no NAK — RESEARCH Pitfall 4)"
  - ".chat-panel__title flex overridden to 0 0 auto; .chat-panel__alias takes flex:1 for remaining space"
metrics:
  duration: "~6 minutes"
  completed: "2026-06-28"
  tasks_completed: 2
  files_changed: 4
status: complete
---

# Phase 161 Plan 03: Alias Control in Shared ChatPanel Header Summary

validateAlias client mirror of Go ValidateAlias + "chatting as" header control wired to sendAliasSet via onSelf pre-fill, enabled for read-only guests (D-06).

## What Was Built

### Task 1: validateAlias client mirror (TDD RED → GREEN)

Added `export function validateAlias(raw: string): string | null` to `ChatPanel.tsx` as a module-scope pure helper alongside `accrueUnread` and `getRowStyle`. Mirrors Go `ValidateAlias` exactly:

- Trims whitespace; returns null for empty result
- Counts code points via `Array.from(trimmed)` — NOT `String.length` (mirrors Go `[]rune` cast)
- Rejects (not truncates) if code-point count exceeds 32
- Rejects any code point in C0 (`< 0x20`) or C1 (`0x7F..0x9F`) ranges

12 TDD tests cover: trim, empty/whitespace null, 32 vs 33 code-point boundary, C0 tab/newline, C1 chars (0x80, 0x9F), and the Array.from-vs-.length astral guard (32 emoji accepted / 33 rejected).

### Task 2: Alias header control (TDD RED → GREEN)

Added to the shared `ChatPanel` component (all three surfaces — GUI tab, Hub modal, web-share guest — mount the same `<ChatPanel>`, so zero host-component changes needed):

**State added:**
- `selfIdentity`: `{ personKey, alias } | null` — set from `onSelf` MsgSelf (0x37) callback
- `aliasEditing`: bool — toggles between label-button and edit-input views
- `aliasDraft`: string — controlled input value while editing
- `aliasError`: string — client-side validation error message

**`onSelf` registration:** Added alongside `onPresence` in the RelayClient subscription effect. Stores self-identity for both web pre-fill and (as a bonus) potential web mention-of-me comparison.

**`currentAlias` computed value:**
```
selfIdentity?.alias ?? participants.find(p => p.personKey === 'local:local')?.alias ?? ''
```
Desktop: `local:local` roster entry (owner constant from relay/server.go:265).
Web-share: `onSelf` identity delivered on WS connect via MsgSelf (0x37, Plan 161-01/02).

**`handleAliasCommit`:** Validates via `validateAlias`, shows `aliasError` on null, calls `clientRef.current?.sendAliasSet(validated)` on success. **Critically, no `isReadOnly` early-return** — the alias control is the D-06 exception (only handleSend/handleInjectPointerDown remain RO-gated).

**Header JSX:** `.chat-panel__alias` div with `title="Your global display name — shown to all chat participants across all sessions"`. Two render states:
- Label: button `.chat-panel__alias-label` showing `«name» ✏️`; click opens edit
- Edit: `.chat-panel__alias-input` (controlled) + `.chat-panel__alias-save` button; Enter or click commits; Escape cancels
- Error: `.chat-panel__alias-error` (role=alert) shown when validateAlias returns null

**CSS in `style.css`:**
- `.chat-panel__title { flex: 0 0 auto }` — override so alias control absorbs remaining space
- `.chat-panel__alias { flex: 1; min-width: 0; display: flex; flex-direction: column }`
- `.chat-panel__alias-label`, `.chat-panel__alias-name`, `.chat-panel__alias-edit`,
  `.chat-panel__alias-input`, `.chat-panel__alias-save`, `.chat-panel__alias-error`
- All using existing `--hub-*` tokens (`--hub-border`, `--hub-surface`, `--hub-text-primary`, `--hub-text-muted`, `--hub-text-dim`, `--hub-radius-sm`)

**TESTING.md:** Added ALIAS-UI-01 and ALIAS-UI-02 traceability rows; updated the Phase 161-03 delta note.

8 TDD alias-control tests cover: render, RO-not-disabled, valid-commit→sendAliasSet once, invalid-commit (33 cp / C0 char) blocks send + shows error, desktop pre-fill from local:local, web pre-fill from onSelf, global scope in title.

## Acceptance Criteria Verification

- [x] `npx vitest run src/components/Hub/ChatPanel.test.tsx` passes (59/59)
- [x] `npx tsc --noEmit` clean
- [x] `grep -c 'Array.from' ChatPanel.tsx` = 2 (in validateAlias implementation)
- [x] `grep -c 'chat-panel__alias' style.css` = 10
- [x] `handleAliasCommit` has NO `isReadOnly` early-return (only handleSend/handleInjectPointerDown remain RO-gated)
- [x] Alias control is NOT disabled on any surface regardless of isReadOnly
- [x] ChatPanel remains un-forked (all three hosts consume the same component unchanged)

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 (TDD) | d2c35eab | test(161-03): add failing validateAlias tests (RED) + implement client mirror (GREEN) |
| Task 2 (TDD) | 7dd4c076 | feat(161-03): add alias control to shared ChatPanel header (ALIAS-UI-01/02) |

## Deviations from Plan

None — plan executed exactly as written. No architectural changes. No Rule 1/2/3 auto-fixes needed.

## Known Stubs

None. The alias control is fully wired: pre-fill reads from live roster/onSelf state, commit calls the live sendAliasSet on the existing RelayClient, server persists via AliasStore and rebroadcasts presence which updates the roster for all participants.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. The alias control calls the existing `sendAliasSet` WS path (Phase 152 wire). Alias text is rendered via React text nodes (no `dangerouslySetInnerHTML`). T-161-04 mitigation (client validateAlias mirror) is in place. No new threat surface beyond what the plan's threat model already covers.

## Self-Check: PASSED

- `frontend/src/components/Hub/ChatPanel.tsx`: modified (validateAlias + alias state + control)
- `frontend/src/components/Hub/ChatPanel.test.tsx`: modified (20 new tests: 12 validateAlias + 8 alias control)
- `frontend/src/style.css`: modified (.chat-panel__alias* CSS rules)
- `TESTING.md`: modified (ALIAS-UI-01/02 traceability rows + delta note)
- Task 1 commit d2c35eab: confirmed in git log
- Task 2 commit 7dd4c076: confirmed in git log
- 59/59 vitest green; tsc clean
