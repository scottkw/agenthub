## Frequently Asked Questions

### DevTools doesn't open in the production app.

Wails disables the browser DevTools in production builds. To inspect the frontend, run the app in development mode with `wails dev`, which opens the webview with DevTools enabled. Alternatively, use the web-share feature to open the session in a regular Chrome browser — Chrome's DevTools work normally there. Logs from the daemon are available at `~/Library/Application Support/agenthub/` on macOS.

### How do I share a session over the network?

Use the **Share** button on a Hub session card. Toggle "Share the session" to get read-only and read/write share links and join codes. Anyone on your Tailscale network can use these links to view or interact with your session. No passwords or manual TLS setup are required — AgentHub handles HTTPS automatically.

### How do I enable remote file browsing?

In the Share modal, toggle **Enable remote file browsing**. The permission level for file access inherits from the link the visitor uses: visitors using the read-only link can only read files, while visitors using the read/write link can also create, edit, and delete files. Web sharing must be enabled first.

### Where are sessions and logs stored?

Sessions are managed by the AgentHub daemon process, which runs in the background. On macOS, logs and application data are stored at `~/Library/Application Support/agenthub/`. The daemon socket is located in the same directory. Sessions persist across GUI restarts — the daemon keeps them alive until you kill them explicitly.

### How do I update AgentHub?

AgentHub checks for updates automatically on startup and once per hour. When a new release is available, an update banner appears at the top of the app with a one-click download link. You can also check manually from **Help → Check for Updates** in the menu bar. Homebrew users can run `brew upgrade --cask agenthub`.

### How do I report a bug?

Open a [GitHub issue](https://github.com/scottkw/agenthub/issues) at the AgentHub repository. Include your AgentHub version number (visible in the Welcome tab or **Help → About**), the steps to reproduce the problem, and any relevant log output from `~/Library/Application Support/agenthub/`. Screenshots or screen recordings are especially helpful for UI issues.
