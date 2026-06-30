---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: 02
type: tdd
wave: 2
depends_on: [160-01]
files_modified:
  - frontend/src/components/Hub/HubInteractiveModal.tsx
  - frontend/src/components/Hub/HubModal.tsx
  - frontend/src/components/Hub/SessionCardGrid.tsx
  - frontend/src/components/Hub/HubPanel.tsx
  - frontend/src/components/Hub/HubInteractiveModal.test.tsx
  - frontend/src/components/Hub/SessionCardGrid.test.tsx
  - frontend/src/components/Hub/HubPanel.test.tsx
autonomous: true
requirements: [NOTIF-01]
must_haves:
  truths:
    - "When a backgrounded session accrues unread, its Hub session card shows a ChatBadge with count > 0."
    - "When the user opens a session's modal, that session's card badge resets to 0."
    - "Unread state from the open modal's ChatPanel is lifted out to HubPanel (not trapped locally)."
    - "An @mention while backgrounded sets hasMention true on the card badge."
  artifacts:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/HubModal.tsx
    - frontend/src/components/Hub/HubInteractiveModal.tsx
  key_links:
    - "ChatPanel onUnreadChange (count,hasMention) -> HubInteractiveModal.handleUnreadChange wraps with session.id -> new onUnreadChange prop -> HubModal -> HubPanel.handleUnreadChange -> unreadMap."
    - "HubPanel passes unreadBySessionId={unreadMap} to SessionCardGrid; SessionCardGrid threads unreadCount/hasChatMention to BOTH SessionCard render sites."
    - "HubPanel calls useChatUnreadListeners (from 160-01) with modalState?.session.id as the exclusion id."
  prohibitions:
    - "MUST NOT modify SessionCard.tsx — it already accepts unreadCount?/hasChatMention? and renders ChatBadge (no changes needed)."
    - "MUST NOT thread onUnreadChange into HubBriefingModal (briefing has no chat)."
    - "MUST NOT mutate the unreadMap in place — always setUnreadMap(prev => new Map(prev)) (Pitfall 5)."
    - "MUST NOT scope-creep into web-share — SessionCardGrid is desktop-GUI only; WebShareSessionView has no card grid."
---

<objective>
Part A of the NOTIF-01 fix: thread the per-session unread signal out of the open modal AND into the Hub session cards, and wire the background listener from 160-01. This closes the v4.1 milestone audit BLOCKER — the Hub-card unread badge that is currently dead-wired (SessionCardGrid never passes unreadCount/hasChatMention; HubInteractiveModal exposes no onUnreadChange).

Purpose: Lift unread state to HubPanel's single `unreadMap`, fed by two sources (open-modal ChatPanel via prop threading + backgrounded sessions via the 160-01 hook), and render it on the session cards.

Output: onUnreadChange prop threaded HubInteractiveModal -> HubModal -> HubPanel; unreadBySessionId prop on SessionCardGrid threaded to both SessionCard sites; unreadMap state + handleUnreadChange + reset-on-open in HubPanel; useChatUnreadListeners wired.
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-RESEARCH.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-PATTERNS.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-01-SUMMARY.md
@frontend/src/components/Hub/HubInteractiveModal.tsx
@frontend/src/components/Hub/HubModal.tsx
@frontend/src/components/Hub/SessionCardGrid.tsx
@frontend/src/components/Hub/SessionCard.tsx
@frontend/src/components/Hub/HubPanel.tsx
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Lift unread out of the modal (HubInteractiveModal + HubModal)</name>
  <files>frontend/src/components/Hub/HubInteractiveModal.tsx, frontend/src/components/Hub/HubModal.tsx, frontend/src/components/Hub/HubInteractiveModal.test.tsx</files>
  <read_first>
    - 160-RESEARCH.md lines 76-83 (prop-threading table) and 293-302 (callback shape — sessionId injection)
    - 160-PATTERNS.md lines 113-167 (HubInteractiveModal + HubModal exact edits)
    - frontend/src/components/Hub/HubInteractiveModal.tsx lines 12-28 (props) and 54-63 (handleUnreadChange)
    - frontend/src/components/Hub/HubModal.tsx lines 37-50 (HubModalProps) and the HubInteractiveModal render site
  </read_first>
  <behavior>
    - HubInteractiveModalProps gains optional onUnreadChange(sessionId, count, hasMention).
    - handleUnreadChange, in addition to its existing setUnreadCount/setHasMention, calls props.onUnreadChange?.(session.id, count, mention) — injecting the session id that ChatPanel's (count,hasMention) callback lacks.
    - HubModalProps gains optional onUnreadChange and threads it to <HubInteractiveModal>; HubBriefingModal branch is untouched.
  </behavior>
  <action>
    Add `onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void` to HubInteractiveModalProps and to HubModalProps (both optional — briefing-modal sessions must not break). Inside HubInteractiveModal's existing handleUnreadChange, add a call forwarding `(session.id, count, mention)` to the new prop. In HubModal, pass `onUnreadChange={onUnreadChange}` to the HubInteractiveModal render site only (not the briefing branch). Extend HubInteractiveModal.test.tsx to assert that when ChatPanel's onUnreadChange fires (count, hasMention), the modal's onUnreadChange prop is invoked with the session's id plus the same count/hasMention.
  </action>
  <verify>
    <automated>cd frontend && pnpm vitest run src/components/Hub/HubInteractiveModal src/components/Hub/HubModal</automated>
  </verify>
  <acceptance_criteria>
    Both props interfaces expose optional onUnreadChange; handleUnreadChange forwards with session.id; HubModal threads it to HubInteractiveModal; the new test asserting sessionId injection is GREEN.
  </acceptance_criteria>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Thread unreadBySessionId through SessionCardGrid to both card sites</name>
  <files>frontend/src/components/Hub/SessionCardGrid.tsx, frontend/src/components/Hub/SessionCardGrid.test.tsx</files>
  <read_first>
    - 160-RESEARCH.md line 81 (both SessionCard render sites) and lines 374-379 (SessionCard badge render — no changes needed there)
    - 160-PATTERNS.md lines 225-247 (SessionCardGrid edits + two render sites)
    - frontend/src/components/Hub/SessionCardGrid.tsx lines 133-170 (props) and the two SessionCard render sites (named-group ~266-282, workDir ~316+)
    - frontend/src/components/Hub/SessionCard.tsx lines 173-177, 214-215, 450 (already renders ChatBadge from unreadCount/hasChatMention)
  </read_first>
  <behavior>
    - SessionCardGridProps gains optional unreadBySessionId: Map<string, {count, hasMention}>.
    - BOTH SessionCard render sites pass unreadCount={unreadBySessionId?.get(s.id)?.count} and hasChatMention={unreadBySessionId?.get(s.id)?.hasMention}.
    - A session with map entry {count:3,hasMention:true} renders a ChatBadge showing 3 with the mention style; a session absent from the map renders no badge (count 0).
  </behavior>
  <action>
    Add `unreadBySessionId?: Map<string, { count: number; hasMention: boolean }>` to SessionCardGridProps (following the existing optional-prop pattern). At BOTH SessionCard render sites, pass `unreadCount={unreadBySessionId?.get(s.id)?.count}` and `hasChatMention={unreadBySessionId?.get(s.id)?.hasMention}`. Do NOT touch SessionCard.tsx — it already renders ChatBadge from those props. Extend SessionCardGrid.test.tsx to render the grid with an unreadBySessionId map and assert the badge count/mention propagates to the rendered SessionCard for a mapped session, and that an unmapped session shows count 0.
  </action>
  <verify>
    <automated>cd frontend && pnpm vitest run src/components/Hub/SessionCardGrid</automated>
  </verify>
  <acceptance_criteria>
    SessionCardGridProps exposes unreadBySessionId; both render sites thread it; SessionCard.tsx is unchanged; the grid test proving badge propagation (mapped count>0, unmapped 0) is GREEN.
  </acceptance_criteria>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Wire HubPanel unreadMap, hook call, reset-on-open, and prop pass-down</name>
  <files>frontend/src/components/Hub/HubPanel.tsx, frontend/src/components/Hub/HubPanel.test.tsx</files>
  <read_first>
    - 160-RESEARCH.md lines 80 (HubPanel changes), 114-118 (reset semantics), 254-276 (state-flow diagram), 472-481 (isActive gate + persist-after-close decision)
    - 160-PATTERNS.md lines 171-221 (HubPanel exact edits: state, handleUnreadChange, reset in handleCardClick, hook call, pass-down)
    - frontend/src/components/Hub/HubPanel.tsx lines 59-119 (usePreviewPoller call site ~334), the handleCardClick / setModalState path, and the <HubModal> + <SessionCardGrid> render sites
  </read_first>
  <behavior>
    - HubPanel holds unreadMap state (Map<sessionId,{count,hasMention}>), initialized empty.
    - handleUnreadChange(sessionId,count,hasMention) updates the map via functional setState with a new Map (never mutated in place).
    - handleCardClick deletes the opening session's entry (reset to 0) BEFORE setModalState.
    - useChatUnreadListeners is called with (sessions, relayPort ?? 0, modalState?.session.id ?? null, isActive ?? false, handleUnreadChange).
    - <HubModal onUnreadChange={handleUnreadChange}> and <SessionCardGrid unreadBySessionId={unreadMap}> receive the new props.
  </behavior>
  <action>
    Add `unreadMap` state and `handleUnreadChange` to HubPanel per 160-PATTERNS.md lines 176-197 (functional setState + new Map(prev); never mutate). In handleCardClick, before setModalState, delete the clicked session's map entry to reset its badge to 0. Call `useChatUnreadListeners(sessions, relayPort ?? 0, modalState?.session.id ?? null, isActive ?? false, handleUnreadChange)` adjacent to the existing usePreviewPoller call (import from `./useChatUnreadListeners`, built in 160-01) — passing modalState's session id as the exclusion id prevents the double-count pitfall. Pass `onUnreadChange={handleUnreadChange}` to the <HubModal> render site and `unreadBySessionId={unreadMap}` to the <SessionCardGrid> render site. Extend HubPanel.test.tsx to assert: (a) handleUnreadChange updates unreadMap so the corresponding card badge appears, and (b) clicking a card with a nonzero entry resets that entry to 0 (badge cleared) when the modal opens.
  </action>
  <verify>
    <automated>cd frontend && pnpm vitest run src/components/Hub/HubPanel && pnpm exec tsc --noEmit</automated>
  </verify>
  <acceptance_criteria>
    HubPanel owns unreadMap fed by both sources; reset-on-open works; the hook is wired with the correct exclusion id; props reach SessionCardGrid and HubModal; HubPanel.test.tsx assertions GREEN; full tsc passes (wails-build parity gate).
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| (none new) | Pure client-side prop threading + state; no data crosses a new trust boundary |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-160-03 | Information Disclosure | unread badge content | low | accept | Badge exposes only a count + mention boolean for sessions the local owner already controls; no message content rendered on the card. No new attack surface (RESEARCH §Security Domain). |
</threat_model>

<verification>
- `cd frontend && pnpm vitest run src/components/Hub/` is GREEN across HubInteractiveModal, HubModal, SessionCardGrid, HubPanel.
- `cd frontend && pnpm exec tsc --noEmit` passes (vitest tolerates TS errors the wails build rejects — run tsc).
- SessionCard.tsx is unchanged in this plan's diff.
</verification>

<success_criteria>
NOTIF-01 BLOCKER closed: the Hub session-card unread badge lights for both open-modal (prop-threaded ChatPanel) and backgrounded (160-01 hook) sources, and resets to 0 when the user opens the session modal.
</success_criteria>

<output>
Create `.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-02-SUMMARY.md` when done.
</output>
