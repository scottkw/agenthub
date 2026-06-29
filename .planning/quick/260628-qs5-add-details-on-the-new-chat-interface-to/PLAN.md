---
type: quick
quick_id: 260628-qs5
slug: add-details-on-the-new-chat-interface-to
date: 2026-06-28
files_modified:
  - frontend/src/content/help/chat.md
  - frontend/src/components/HelpTab.tsx
  - frontend/src/components/HelpSectionNav.tsx
  - frontend/src/components/__tests__/HelpTab.integration.test.tsx
  - TESTING.md
---

# Quick Task: Add a "Chat" section to the in-app Help screen

## Problem

The v4.1 "Session Chat" feature (per-session human-to-human chat with aliases, @session prompt
injection, resizable drawer, web-share parity, read-only-can-chat) shipped with NO documentation in
the in-app Help screen. The Help tab (`HelpTab.tsx`) currently has only two sections — Getting Started
and FAQ — neither of which mentions chat.

## Approach

Add a new dedicated **Chat** help section, matching the existing section architecture exactly:
- Help content lives as a bundled markdown file under `frontend/src/content/help/`.
- Each section is registered in TWO synchronized lists that MUST stay in lockstep:
  1. `SECTION_META` in `HelpTab.tsx` (id + label + imported markdown) — drives rendering + search index.
  2. `SECTIONS` in `HelpSectionNav.tsx` (id + label) — drives the left nav + scroll-spy.
- Section ids follow the `help-<slug>` convention (`help-getting-started`, `help-faq` → add `help-chat`).
- Position the new section **between Getting Started and FAQ** (feature topic before the FAQ).

## Tasks

### Task 1: Create the chat help content + register the section
- **Create `frontend/src/content/help/chat.md`** with EXACTLY this content (tone/structure mirrors
  `getting-started.md`: a `##` section heading, `###` subsections, user-facing present tense, no
  internal jargon like frame numbers or JSONL):

```markdown
## Chat

Every session has a built-in **chat** — a side conversation attached to the session, separate from the agent's terminal. Chat is for people: you and anyone you share the session with can talk in real time while watching the same terminal. It works the same way on the desktop app, in the Hub session view, and for remote guests who open a shared link, so everyone sees one conversation.

### Opening Chat

Click the **chat button** in the lower-right corner of a session to slide the chat panel in from the right. The panel overlays the session — it never resizes or disturbs the terminal. Click the button again to slide it closed. Drag the panel's left edge to make it wider or narrower; your chosen width is remembered the next time you open it.

### Setting Your Display Name

When you join a chat you are identified by your Tailscale identity plus a display name (alias). Set or change your display name from the **"chatting as"** control at the top of the chat panel — the new name appears to everyone in the conversation and is remembered for future sessions. The same control is available on every surface, including remote shared links.

### Mentions and Prompting the Agent

Type **@** to mention another participant. One mention is special: **@session** sends your message straight into the agent's terminal as a prompt — a one-way bridge from chat to the running CLI. The agent's reply appears in the terminal, not in chat, so you can hand the agent an instruction without leaving the conversation.

### Read-Only and Read/Write Guests

When you share a session you choose read-only or read/write links. **Both kinds of guest can post chat messages** and take part in the conversation. Only read/write guests can use **@session** to inject a prompt into the agent or type into the terminal — read-only guests are limited to chatting and watching.

### Presence, History, and Markdown

The chat panel shows who is currently in the session and a typing indicator when someone is composing a message. Messages are kept for the life of the session by the AgentHub daemon, so a late joiner sees the full backlog when they open chat, and the conversation survives an app restart. Sessions with unread chat activity are flagged in the Hub so you can tell where the discussion is happening. Messages support **Markdown** — code blocks, lists, links, and emphasis all render — and you can export a thread as a Markdown document to save or share the discussion.
```

- **`HelpTab.tsx`**: add `import chatMd from '../content/help/chat.md?raw'` alongside the existing
  `gettingStartedMd`/`faqMd` imports, and insert `{ id: 'help-chat', label: 'Chat', markdown: chatMd }`
  into `SECTION_META` BETWEEN the getting-started and faq entries. Do not change the `useMemo([])`
  search-index dependency rationale (chatMd is also a module-level constant — the empty deps array
  stays correct).
- **`HelpSectionNav.tsx`**: insert `{ id: 'help-chat', label: 'Chat' }` into `SECTIONS` in the SAME
  position (between getting-started and faq) so the nav order matches the render order. Keep the
  `as const`.
- **verify**: `cd frontend && npx vitest run src/components/__tests__/HelpTab.test.tsx src/components/__tests__/HelpTab.integration.test.tsx src/components/__tests__/HelpSectionNav.test.tsx src/components/__tests__/HelpContent.test.tsx src/components/__tests__/HelpSearch.test.tsx` and `npx tsc --noEmit`.

### Task 2: Update the help integration test for the new section count
- **`HelpTab.integration.test.tsx`**: the test "renders … sections" asserts
  `expect(sections.length).toBe(2)` (~line 79) — update it to `toBe(3)` now that Chat is a third
  section. Add a sibling assertion that `document.getElementById('help-chat')` resolves and its
  `textContent` contains `'Chat'` (mirror the existing `#help-getting-started` / `#help-faq` checks).
  Leave the nav-button tests (which use "at least two") unchanged — they already tolerate more sections.
- **verify**: the five help test files above all pass; `npx tsc --noEmit` clean.

### Task 3: Register the change in TESTING.md (standing regression convention)
- Add a Suite Manifest `> Note:` (Section 2) for this quick task: no new test files; NEW help content
  file `frontend/src/content/help/chat.md`; EXTENDED `HelpTab.integration.test.tsx` (section count
  2→3 + `#help-chat` presence). No new traceability row (in-app help content, not a mapped requirement).
- **verify**: `bash tests/check-traceability-paths.sh` exits 0.

## Done when
- The Help tab shows a third **Chat** section (nav button + `#help-chat` content) between Getting
  Started and FAQ, documenting opening/resizing the panel, aliases, @session, RO-vs-RW guests, and
  presence/history/markdown.
- `SECTION_META` (HelpTab.tsx) and `SECTIONS` (HelpSectionNav.tsx) both include `help-chat` in the
  same position.
- All five Help-* test files pass (integration count updated to 3); tsc clean; traceability green.
