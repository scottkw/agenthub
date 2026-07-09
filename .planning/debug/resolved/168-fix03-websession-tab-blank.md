---
status: resolved
trigger: "FIX-03 (168-03) remote-open via ⋯ 'Open in browser' opens an in-app web-session tab whose terminal is (A) blank (blinking cursor, no PTY output) and (B) only fills the top ~half of the pane with a dead dark band below. The Phase-134 card-body-click modal path streams the SAME remote session fine."
created: 2026-07-02
updated: 2026-07-02
---

## Current Focus

hypothesis: TWO independent root causes — (A) the tab opens a DIRECT cross-origin
  wss to the peer, which the peer rejects with 403 on Origin mismatch; (B) the tab
  mounts WebShareSessionView (modal height classes) directly in block-context
  .terminal-container with no full-height wrapper, so the flex height chain collapses.
test: source trace of the connect path (App.tsx → WebShareSessionView → TerminalPanel
  → RelayClient → peer webserver Origin allowlist) and the CSS/JSX height chain.
expecting: modal path routes via daemon proxy (Origin injected server-side) + full-height
  .hub-modal parent; tab path uses direct wsURL + no height parent.
next_action: DONE — diagnosis only (read-only). Recommend fix to plan-phase.

## Symptoms

expected: The ⋯ → "Open in browser" remote-open tab should stream the remote PTY and
  fill the pane, identical to the Phase-134 card-click modal (which works).
actual:
  (A) Terminal is BLANK — blinking cursor, no remote PTY output ever appears.
  (B) Terminal occupies only the top ~half of the content area; large dead dark band below.
errors: none surfaced in UI (silent). Predicted: 403 on the WS upgrade (DevTools Network).
reproduction: Two-Mac tailnet, production build. Hub SessionCard for a remote peer
  session → ⋯ overflow → "Open in browser" → held-cap reuse OR join-code modal →
  openWebSessionTab → web-session tab renders WebShareSessionView.
started: Introduced by Phase 168-03 (FIX-03), which reversed the Phase-146
  external-browser design (BrowserOpenURL) to an in-app web-session tab.

## Eliminated

- hypothesis: Remote session is unreachable / daemon-level connectivity problem.
  evidence: The Phase-134 card-click modal streams the SAME session correctly via
    TerminalPanel remote → daemon proxy /api/relay/remote/{id}/ws. Reachability is fine;
    the defect is specific to the web-session tab's connect + sizing path.
  timestamp: 2026-07-02

- hypothesis: Symptom A is merely a consequence of Symptom B (collapsed height →
    xterm fit computes ~0 rows → nothing renders).
  evidence: The peer's /sessions/{id}/ws enforces a strict byte-exact Origin allowlist
    (origin_mw.go requireAllowedOrigin + server.go:1316 OriginPatterns) and rejects the
    webview's own origin with 403 BEFORE any bytes flow. A is an independent connect
    failure that would persist even at full height. (Guest mode also does NOT fit/collapse
    rows the way host mode would — it writes output regardless of size and applies a scale
    transform, so B alone would not zero out all output.)
  timestamp: 2026-07-02

## Evidence

- timestamp: 2026-07-02
  checked: frontend/src/components/Hub/WebShareSessionView.tsx:68-77,129-138
  found: For a remote tab (baseURL set), wsURL = `${wsOrigin}/sessions/{id}/ws?cap={token}`
    where wsOrigin = (baseURL).replace(/^http/,'ws') — i.e. a DIRECT cross-origin
    wss://peer.ts.net/... This wsURL is passed to TerminalPanel (not the `remote` prop).
  implication: The desktop Wails webview attempts a direct WebSocket to the REMOTE peer's
    webserver, cross-origin.

- timestamp: 2026-07-02
  checked: frontend/src/lib/relayClient.ts:243-252
  found: When opts.wsURL is set, RelayClient uses it verbatim; when opts.remote is set it
    builds ws://127.0.0.1:{port}/api/relay/remote/{id}/ws (the local daemon proxy). The
    two modes are mutually exclusive.
  implication: WebShareSessionView's remote tab takes the DIRECT path; the working modal
    takes the PROXY path.

- timestamp: 2026-07-02
  checked: internal/webserver/origin_mw.go:41-70 (requireAllowedOrigin) +
    internal/webserver/server.go:932-942,1316-1330 (allowedOrigins / OriginPatterns)
  found: /sessions/{id}/ws rejects any request whose Origin header is not a byte-exact
    match for ws.BaseURL() (peer tailnet URL) or ws.FunnelBaseURL() (peer funnel URL).
    Missing Origin → 403; mismatched Origin → 403 — BEFORE the capability check.
  implication: A direct WS from the desktop webview sends the webview's OWN origin
    (e.g. http://wails.localhost / wails://…), never the peer's https://peer.ts.net.
    Browsers forbid JS from setting the Origin header on a WebSocket, so the tab can
    never satisfy the allowlist → 403 → WS never opens → onOutput never fires → BLANK.
    ROOT CAUSE of Symptom A.

- timestamp: 2026-07-02
  checked: internal/daemon/remote_ws_proxy.go:19-22,84-96
  found: The working modal path (TerminalPanel remote) hits this daemon proxy, which
    explicitly documents Pitfall 1 and injects `hdr.Set("Origin", TrimRight(baseURL,"/"))`
    — "byte-exact match to the peer's BaseURL()" — before dialing the peer. The cap is
    looked up server-side from RemoteCapStore; the webview URL carries no cap.
  implication: The daemon proxy exists PRECISELY to solve the Origin-allowlist problem
    that the direct tab path re-introduces. The webview cannot inject Origin; only the
    Go proxy can. This confirms A is an architectural-invariant violation, not a fluke.
    (Matches project memory: "remote browse uses CORS via local-daemon proxy / D-02 — no
    cross-origin browser fetches.")

- timestamp: 2026-07-02
  checked: frontend/src/App.tsx:1637 (.terminal-container) + 1701-1724 (websession branch)
    + 1899-1902 (normal terminal tabs) + style.css:504-517 (.terminal-container /
    .terminal-wrapper) + 6051-6067 (.hub-modal__body / --interactive) + 5955-5967 (.hub-modal)
  found: The __websession__ branch renders <WebShareSessionView> (root class
    .hub-modal__body--interactive = flex:1; min-height:0; display:flex) as a DIRECT child
    of <div className="terminal-container"> which is display:BLOCK (flex:1; min-height:0;
    overflow:hidden; position:relative — NO display:flex, NO height:100%). Every WORKING
    terminal tab instead wraps its content in <div className="terminal-wrapper">
    (display:flex; flex-direction:column; width:100%; height:100%). The WORKING modal
    parents .hub-modal__body--interactive under .hub-modal (definite height:min(750px,
    100vh-64px); display:flex; flex-direction:column).
  implication: In the tab, .hub-modal__body--interactive's flex:1 is inert (block parent)
    and it has no height:100%, so its height collapses to content height; the inner
    .terminal-session-container (flex:1; min-height:0) has nothing to grow into and the
    terminal renders short, leaving .terminal-container's --hub-bg as the dead band below.
    ROOT CAUSE of Symptom B: missing full-height flex-column link (the .terminal-wrapper
    the normal terminal tabs use, or an equivalent definite-height parent) between
    .terminal-container and WebShareSessionView.

## Resolution

root_cause: |
  TWO INDEPENDENT root causes in the Phase 168-03 (FIX-03) in-app remote web-session tab:

  (A) BLANK TERMINAL — cross-origin WS rejection.
      WebShareSessionView.tsx:77 builds a DIRECT cross-origin socket
      wss://<peer>/sessions/{id}/ws?cap=<token> and hands it to TerminalPanel as `wsURL`
      (relayClient.ts:244-245). The remote peer's /sessions/{id}/ws enforces a strict
      byte-exact Origin allowlist (webserver/origin_mw.go requireAllowedOrigin + the
      OriginPatterns belt-and-suspenders at server.go:1316-1326), accepting ONLY the
      peer's own BaseURL()/FunnelBaseURL(). The desktop webview sends its own origin and
      cannot spoof Origin (browser-forbidden on WebSocket), so the upgrade is 403'd before
      any capability/PTY work → the WS never opens → zero output → blinking cursor only.
      The working modal path avoids this by routing through the local daemon proxy
      (/api/relay/remote/{id}/ws), which injects Origin:<peer baseURL> server-side
      (daemon/remote_ws_proxy.go:92) — the exact seam the direct tab path bypasses.

  (B) HALF-HEIGHT / DEAD BAND — collapsed CSS height chain.
      The __websession__ render branch (App.tsx:1701-1724) mounts WebShareSessionView —
      whose root uses the MODAL classes .hub-modal__body hub-modal__body--interactive
      (flex:1; min-height:0; display:flex) — as a DIRECT child of the block-context
      .terminal-container (style.css:504, no display:flex, no height:100%). Those modal
      classes only fill when parented by a definite-height flex-column ancestor
      (.hub-modal, height:min(750px,100vh-64px)). Normal terminal tabs supply that via a
      <div className="terminal-wrapper"> (height:100%; display:flex; flex-direction:column,
      style.css:512) wrapper; the web-session branch has NO such wrapper, so flex:1 is
      inert and the panel collapses to content height, exposing the dead band.

  A and B are independent: fixing one leaves the other. Both must be addressed.

fix: |
  RECOMMENDED (minimal, consistent with existing patterns):

  (A) Route the remote web-session tab's terminal through the SAME daemon proxy the
      working modal uses, instead of a direct cross-origin wsURL. The cap is already in
      RemoteCapStore for these tabs (handleModalExchange calls RegisterRemoteCap;
      held-cap reuse implies it is already stored), and the desktop relayPort is passed
      to WebShareSessionView, so the proxy at ws://127.0.0.1:{relayPort}/api/relay/
      remote/{id}/ws is reachable. Concretely: give WebShareSessionView a `remote`/proxy
      mode for in-app remote-peer tabs that passes TerminalPanel `remote` (and drops the
      direct wsURL) so RelayClient builds the proxy URL. This reuses daemon/
      remote_ws_proxy.go verbatim — the proven modal transport. (Chat/apiBaseURL for a
      remote tab has the same cross-origin problem and would need an analogous proxy or
      to be disabled for remote tabs; the terminal is the reported priority.)
      Do NOT try to "make the direct wss work" — the webview cannot set Origin, so a
      direct cross-origin socket is architecturally impossible against the peer's
      allowlist. INFERENCE: high confidence from remote_ws_proxy.go's own Pitfall-1 note.

  (B) Wrap the __websession__ render branch in the same full-height flex-column container
      the normal terminal tabs use — e.g. render WebShareSessionView inside
      <div className="terminal-wrapper" style={{display:'flex'}}> (App.tsx pattern at
      1899-1902), OR add a CSS rule giving .terminal-container's web-session child
      height:100% + display:flex. Restores the height chain so .hub-modal__body--interactive
      (flex:1; min-height:0) fills. Prefer the wrapper (parity with terminal tabs) over
      changing the shared .hub-modal__* classes (which must keep working in the modal).

verification: NOT YET APPLIED — read-only diagnosis. Live disambiguation signal: open
  DevTools Network on the web-session tab; the /sessions/{id}/ws request will show 403
  (confirms A) rather than 101 Switching Protocols; the .hub-modal__body--interactive
  computed height will be < .terminal-container height (confirms B).
files_changed: []
</content>
</invoke>
