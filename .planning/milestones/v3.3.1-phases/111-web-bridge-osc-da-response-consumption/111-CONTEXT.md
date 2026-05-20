# Phase 111: Web bridge OSC/DA response consumption - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning
**Mode:** Pre-authored from `.planning/milestones/v3.3.1-ROADMAP.md` + `.planning/REQUIREMENTS.md`

<domain>
## Phase Boundary

On the web-served terminal, OSC color-query (OSC 10/11) and Device Attributes (CSI c) responses are **consumed by the web bridge** and do NOT leak into shell stdin — so `chafa --format=sixel <png>` produces a clean prompt on the web surface, matching desktop. Web ↔ desktop parity is the release gate. Closes GitHub Issue #54.

</domain>

<decisions>
## Implementation Decisions

### Root cause (per v3.3.1-ROADMAP.md Phase 111 + Issue #54)
- **Web-only bug** — desktop (Wails webview) goes through xterm.js directly and consumes those responses cleanly; the dedicated web relay path in `internal/webserver` does NOT.
- The web bridge currently does not strip / consume OSC 10/11 / DA1 responses that xterm.js (on the browser) emits in reply to terminal queries from the running program — so chafa's color queries get echoed back to shell stdin instead of being absorbed.
- Pre-existing (not a v3.3 regression).

### Fix shape (LOCKED)
- **Server-side consumption in the web bridge** (in `internal/webserver/relay.go` or equivalent). The bridge proxies WebSocket frames between browser and PTY. It must parse the browser → PTY direction for terminal-query responses (OSC 10, OSC 11, CSI c / DA1) and consume them — NOT forward them to the PTY's stdin.
  - **Why server-side, not client-side:** parity with how the Wails webview path already handles this — xterm.js on the desktop side absorbs the responses before they hit any forwarding path. The web bridge needs to mirror that absorption. A client-side fix in the browser would split the contract between web and desktop and add complexity.
- **Three response shapes to consume (no false positives):**
  - `\x1b]10;rgb:RRRR/GGGG/BBBB\x1b\\` — OSC 10 FG color reply (BEL `\x07` terminator also valid).
  - `\x1b]11;rgb:RRRR/GGGG/BBBB\x1b\\` — OSC 11 BG color reply (BEL terminator also valid).
  - `\x1b[?<params>c` — CSI c / DA1 reply (numeric params like `62;4;9;22`).
- **Streaming state machine across WebSocket frames** — these responses can be split across frames. A naive byte-by-byte regex won't work. Use a small state machine that tracks "in OSC body" / "in CSI body" / "outside" and buffers the response across frames if needed.
- **Forward everything else verbatim** — user keystrokes, control sequences not in the absorption set, etc. Must NOT regress normal terminal interaction.

### Test surface (WEB-03)
- **Go-level regression test** in `internal/webserver/` that feeds the relay synthetic OSC 10/11 / DA1 responses (across frame boundaries) and asserts the relay does NOT forward them downstream. Easier to maintain than an e2e browser test; sufficient to lock the path.

### Cross-surface verification (release gate)
- **Web:** open browser, web-share a shell session, run `chafa --format=sixel /tmp/test.png`. Clean prompt afterward; no leaked `10;rgb:…`, `11;rgb:…`, `62;4;9;22c` strings.
- **Desktop:** same `chafa` command on a Wails-attached session — confirm unchanged (still produces clean prompt, same output as web after fix).
- **Test on macOS** using local web-share (start agenthub locally, web-share a session, open `http://localhost:<port>` in Chrome) — possible without external hardware.

### Out of scope
- xterm.js client-side changes — server-side fix only.
- Other terminal-query absorptions (e.g., OSC 4 palette query) — only the three documented in WEB-01 (OSC 10, OSC 11, DA1).

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/webserver/` — web bridge package. Look for `relay.go` or similar WebSocket proxy code.
- `internal/relay/` — separate package for the desktop relay (different surface, separate code path).
- Existing terminal escape parsing helpers, if any (researcher should grep).

### Established Patterns
- WebSocket frame handling in `internal/webserver/` — researcher will identify the exact handler. Likely uses `gorilla/websocket` or similar (verify in `go.mod`).
- Logging patterns in `internal/webserver/` for diagnostics — emit a debug log on every absorbed response so we can audit during web UAT.

### Integration Points
- WS frame data flow: browser xterm.js → WS frame → web bridge → PTY stdin (current). Fix injects an absorption layer between WS frame parse and PTY write.
- Existing tests in `internal/webserver/` for guidance on test scaffolding.

</code_context>

<specifics>
## Specific Ideas

- Issue #54 reproduction (UAT-05 in `.planning/phases/101-shell-session-surfaces-and-web-share-gating/101-UAT.md` from v3.3): `chafa --format=sixel /tmp/<png>` in a web-shared shell session → currently leaks `10;rgb:c4c4d4d4d4d4` etc. into next prompt line.
- Three response shapes documented; researcher should verify byte-level sequences against xterm.js source if needed.
- macOS executor CAN do web UAT — local web-share + Chrome.

</specifics>

<deferred>
## Deferred Ideas

None — Phase 111 scope is tight.

</deferred>
