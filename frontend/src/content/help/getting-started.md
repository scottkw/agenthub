## Getting Started

AgentHub launches AI coding CLIs — Claude Code, OpenCode, Codex, Gemini CLI, and others — in tabbed terminal sessions. Sessions persist across app restarts via a background daemon, so you can close and reopen the window without losing your work.

### Creating a Session

Click the **+ New Session** button in the Hub toolbar to open the session creation modal. Choose an agent (Claude Code, OpenCode, Codex, Gemini CLI, or a raw shell), select a working directory, and optionally pass extra arguments to the agent. The session opens in a new terminal tab.

### Running Shell Sessions

In addition to AI coding CLIs, AgentHub supports plain shell sessions (bash, zsh, or your configured default). Choose **Shell** from the agent picker. The shell binary can be changed in **Settings → Paths**. Shell sessions auto-close when the shell exits naturally.

### Switching Between Sessions

Open sessions appear as tabs along the top of the window. Click any tab to switch to that session. The **Hub** view (Home icon in the sidebar) shows all sessions as cards grouped by working directory — click a card to open or expand that session.

### Using the File Browser

Click the file-browser icon on a session card or use the Share modal to browse the session's working directory. You can view, edit, create, and delete files via the built-in CodeMirror editor. Remote sessions require web sharing to be enabled first.

### Sharing Sessions Over the Network

AgentHub serves sessions over your Tailscale network with automatic HTTPS. In the Hub, click the **Share** button on any session card. Toggle "Share the session" to generate read-only and read/write links. Toggle "Enable remote file browsing" to grant file-access permissions alongside the share.

### Settings Overview

Open **Settings** from the sidebar (or press the gear icon). Settings is divided into sections: Plugins (feature toggles), Web Server (URL, QR code, copy link), Paths (CLI and shell binary locations), and Appearance (terminal theme, font size). Changes take effect immediately or after saving, depending on the setting.
