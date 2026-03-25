---
status: diagnosed
trigger: "Terminal tabs show no output after process separation refactor"
created: 2026-03-23T00:00:00Z
updated: 2026-03-23T00:00:00Z
---

## Current Focus

hypothesis: relayPort=0 passed to frontend due to GetRelayPort() swallowing errors, OR stale daemon from old binary
test: Verified daemon API returns correct relay port; traced full data flow
expecting: Root cause identified - see Resolution
next_action: Report findings

## Symptoms

expected: Terminal tabs show PTY output after session creation
actual: Sessions created (tabs appear with names), but terminal area is completely blank - no PTY output
errors: Unknown - need to investigate
reproduction: Create a session, observe blank terminal
started: After phase 20 process separation refactor

## Eliminated

## Evidence

- timestamp: 2026-03-23T00:01:00Z
  checked: Daemon process status
  found: PID 8640 running from deleted binary at build/bin/agenthub.app/Contents/MacOS/agenthub (binary no longer exists on disk)
  implication: Daemon survives binary deletion; will be reused by EnsureDaemon health check

- timestamp: 2026-03-23T00:02:00Z
  checked: Daemon health and relay port via Unix socket
  found: Health OK, relay port 56115, sessions exist ("claude 1", "gemini 2")
  implication: Daemon backend is functional

- timestamp: 2026-03-23T00:03:00Z
  checked: Relay HTTP endpoint
  found: GET /sessions returns both session IDs; WebSocket upgrade returns 101
  implication: Relay server is working correctly

- timestamp: 2026-03-23T00:04:00Z
  checked: Frontend code for relay port guard
  found: App.tsx line 334 guards with `relayPort !== null` but NOT `relayPort > 0`. Go App.GetRelayPort() returns 0 on error (not an error to Wails)
  implication: If GetRelayPort() returns 0, TerminalPanels render but WebSocket connects to port 0, failing silently

- timestamp: 2026-03-23T00:05:00Z
  checked: RelayClient error handling
  found: ws.onerror is a no-op (comment: "Error is always followed by close"), ws.onclose just logs debug. No retry, no error surfacing.
  implication: WebSocket connection failures are completely silent

- timestamp: 2026-03-23T00:06:00Z
  checked: Full data flow from frontend to daemon
  found: Structurally correct. SessionEngine, HubManager, relay.Server all properly wired in daemon process. Protocol unchanged.
  implication: No code-level bug in the relay data flow itself

- timestamp: 2026-03-23T00:07:00Z
  checked: React StrictMode
  found: Enabled in main.tsx. Effects double-fire in dev mode (mount, unmount, mount). Terminal and RelayClient are disposed and recreated.
  implication: Should not cause blank output but adds complexity. Second mount should work correctly.

## Resolution

root_cause: |
  Two contributing factors, likely working together:

  1. SILENT FAILURE ON PORT 0: App.GetRelayPort() returns 0 when the daemon client
     call errors, but this is NOT surfaced as an error to Wails. The frontend guards
     with `relayPort !== null` but 0 passes this check, so TerminalPanels render
     and attempt WebSocket to ws://127.0.0.1:0 which fails silently (RelayClient
     has no error reporting -- ws.onerror is a no-op, ws.onclose just logs debug).

  2. STALE DAEMON RISK: The daemon binary was deleted (by wails rebuild) but daemon
     process PID 8640 keeps running from the old binary. EnsureDaemon's health check
     passes, so the old daemon is reused. If the old daemon was from a build that
     predates StartRelay() (or has different relay behavior), it could return port 0.

  The most likely scenario: During development iteration, a stale daemon from an
  intermediate build (where StartRelay wasn't called, or the relay-port route
  wasn't registered) responds to /health but returns {"port":0} for /relay-port.
  The frontend receives 0, passes the null check, renders terminals, and WebSocket
  to port 0 fails completely silently.

fix: |
  THREE fixes needed:

  1. FRONTEND GUARD (immediate fix): Change `relayPort !== null` to
     `relayPort !== null && relayPort > 0` in App.tsx line 334. Also add
     an error state when relayPort is 0.

  2. BACKEND ERROR (defensive fix): Change App.GetRelayPort() to return
     an error when port is 0 instead of silently returning 0:
     ```go
     func (a *App) GetRelayPort() (int, error) {
         port, err := a.client.GetRelayPort()
         if err != nil {
             return 0, fmt.Errorf("relay port unavailable: %w", err)
         }
         if port == 0 {
             return 0, fmt.Errorf("relay port is 0 (relay not started)")
         }
         return port, nil
     }
     ```

  3. STALE DAEMON (robustness fix): Add version checking to EnsureDaemon.
     The daemon should expose its build version in /health response.
     EnsureDaemon should compare versions and restart the daemon if mismatched.
     Alternatively, add a /relay-port check to EnsureDaemon to verify the relay
     is actually running before considering the daemon healthy.

verification:
files_changed: []
