---
status: resolved
phase: 05-qr-codes-status-indicators
source: [05-01-SUMMARY.md, 05-02-SUMMARY.md, 05-03-SUMMARY.md]
started: 2026-03-18T20:30:00Z
updated: 2026-03-18T22:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Status dots visible on tabs
expected: Each tab shows a small colored dot before the tab name. New sessions show a blue dot (running).
result: pass

### 2. Status dot transitions
expected: When a CLI session returns to its prompt (e.g., Claude Code shows ❯), the dot changes from blue (running) to green (idle). When you type or a command runs, it changes back to blue.
result: issue
reported: "the dot stays blue the whole time"
severity: major

### 3. Terminal fills window
expected: The active terminal fills the full available height below the tab bar. No wasted whitespace at the bottom.
result: issue
reported: "No the terminal output is not filling the window. Large blank area below terminal content."
severity: major

### 4. No web-serving bar when server is off
expected: When the web server is NOT started (default), there are no "Web Off" buttons or web-serving controls visible anywhere. The terminal area is clean.
result: pass

### 5. Web server start and web serving toggle
expected: Open Settings (gear icon). Set a password. Start Web Server. Close settings. A "Web Off" button now appears above the active terminal. Click it to enable web serving — it changes to "Web On" and shows the session URL.
result: pass

### 6. QR button and modal
expected: With web serving enabled on a session, a "QR" button appears next to "Copy Token Link". Clicking it opens a modal overlay with a 256x256 QR code image and the session URL below it. Modal closes on clicking outside or pressing Escape.
result: pass

### 7. Web dashboard accessible
expected: With the web server running, the server URL (shown in Settings) opens a dashboard in a browser. The dashboard shows a list of web-enabled sessions.
result: issue
reported: "When I open the link, all I get is Unauthorized."
severity: major

### 8. Dashboard QR thumbnails
expected: On the web dashboard, each session row shows a small (64x64) QR code thumbnail. Clicking it opens an enlarged 256x256 QR overlay with the session URL.
result: skipped
reason: Blocked by test 7 — dashboard returns Unauthorized

## Summary

total: 8
passed: 4
issues: 3
pending: 0
skipped: 1

## Gaps

- truth: "Status dot changes from blue (running) to green (idle) when CLI returns to prompt"
  status: resolved
  reason: "User reported: the dot stays blue the whole time"
  severity: major
  test: 2
  root_cause: "Watch() feeds raw relay-framed messages (0x01 prefix byte + payload) to Detector.Feed() without stripping the protocol byte. The rolling tail contains binary noise, so regex ❯\\s*$ never matches the prompt. Additionally, reANSI misses OSC sequences (\\x1b]...\\x07) emitted by Claude Code before the prompt."
  artifacts:
    - path: "internal/status/detector.go"
      issue: "Watch() feeds raw framed messages to Feed() without calling relay.ParseFrame() to extract MsgOutput payload"
    - path: "internal/status/detector.go"
      issue: "reANSI regex missing OSC sequence pattern for window title updates"
  missing:
    - "Strip relay protocol framing in Watch() — call relay.ParseFrame(), skip non-MsgOutput, feed only payload"
    - "Decode scrollback snapshot frame-by-frame before feeding to detector"
    - "Extend reANSI to cover OSC sequences: \\x1b\\][^\\x07\\x1b]*(?:\\x07|\\x1b\\\\)"
    - "Add TestWatch_IdleTransition that feeds framed output through mock hub"
  debug_session: ".planning/debug/status-dot-stays-blue.md"
- truth: "Terminal fills full available height below tab bar with no wasted whitespace"
  status: resolved
  reason: "User reported: terminal output is not filling the window, large blank area below terminal content"
  severity: major
  test: 3
  root_cause: "FitAddon.fit() called synchronously after term.open() and on isActive change — browser has not completed layout pass yet, so containerRef.clientHeight is 0. xterm canvas gets 0-height dimensions and stays that size. Also missing ResizeObserver for dynamic layout changes."
  artifacts:
    - path: "frontend/src/components/TerminalPanel.tsx"
      issue: "Line 57: fitAddon.fit() called synchronously after term.open() — layout not computed yet"
    - path: "frontend/src/components/TerminalPanel.tsx"
      issue: "Line 100: fit() called synchronously in isActive effect — stale dimensions"
    - path: "frontend/src/style.css"
      issue: ".terminal-wrapper missing display:flex in CSS (relies on inline style only)"
    - path: "frontend/src/style.css"
      issue: "No .web-serving-bar rule — no flex-shrink:0"
  missing:
    - "Defer fit() calls with requestAnimationFrame"
    - "Replace window resize listener with ResizeObserver on container div"
    - "Add display:flex to .terminal-wrapper CSS rule"
    - "Add .web-serving-bar { flex-shrink: 0 } CSS rule"
  debug_session: ".planning/debug/xterm-height-not-filling.md"
- truth: "Web dashboard loads in browser when navigating to server URL"
  status: resolved
  reason: "User reported: When I open the link, all I get is Unauthorized."
  severity: major
  test: 7
  root_cause: "dashboardAuth middleware unconditionally returns 401 for unauthenticated requests. The login form exists inside dashboard.html (#login-section) but is only reachable after passing auth — circular dependency. POST /login endpoint works correctly but the HTML page that contains the form is blocked."
  artifacts:
    - path: "internal/webserver/server.go"
      issue: "Lines 259-268: dashboardAuth returns 401 for all unauthenticated requests including GET /dashboard, blocking login page delivery"
  missing:
    - "Serve GET /dashboard without auth — the HTML's JavaScript already handles auth state (probes /api/sessions, shows login form on 401, shows dashboard on 200)"
  debug_session: ""
