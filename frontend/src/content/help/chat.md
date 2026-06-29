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
