---
phase: 87
plan: 06
type: execute
wave: 5
depends_on: [3, 4]
files_modified:
  - web/dashboard.html
  - web/join.html
  - web/embed.go
  - web/terminal.html
  - internal/webserver/server.go
  - internal/webserver/capability_test.go
autonomous: true
requirements:
  - SEC-01
  - SEC-04
tags:
  - security
  - web
  - html

must_haves:
  truths:
    - "/dashboard renders a landing page with NO session list (D-17) and a join-code input form"
    - "/join?code=XXXX-XXXX renders a page with session name, permission badge, and 'Join Session' button"
    - "/join POSTs to /join/exchange which redirects (HTTP 303 See Other) to /sessions/{id}?cap=<token>"
    - "Expired code on /join/exchange returns HTTP 410 Gone; invalid code returns 404"
    - "Terminal page reads perms from GET /api/sessions/{id}/info?cap=<token> and suppresses caret when perms === 'read' (D-23)"
    - "No ?readonly query parameter exists in the write-gate path anywhere in web/ or server.go"
    - "End-to-end flow works: toggle web-serving ON -> read/write URLs issued -> paste URL in browser -> terminal loads -> read-only path cannot send MsgInput"
  artifacts:
    - path: web/dashboard.html
      provides: "Landing page per UI-SPEC Surface 3; no session list"
      contains: "Join a Shared Session"
    - path: web/join.html
      provides: "Join flow page per UI-SPEC Surface 4 with 5 state variants (A-E)"
      contains: "Join Session"
    - path: web/terminal.html
      provides: "Caret suppression when perms is 'read'; READ ONLY badge in status bar"
    - path: internal/webserver/server.go
      provides: "handleDashboard now serves the landing page template; new handleJoin route; new handleJoinExchange route (POST); new handleSessionInfo route"
      contains: "handleJoin"
  key_links:
    - from: web/terminal.html
      to: "/api/sessions/{id}/info"
      via: "fetch to determine read-only state"
      pattern: "/api/sessions/.*/info"
    - from: web/dashboard.html input form
      to: /join?code=<value>
      via: "GET navigation"
      pattern: 'action="/join"|location.href.*\\/join'
    - from: web/join.html Join button
      to: /join/exchange (POST)
      via: "form POST with hidden code field"
      pattern: 'action="/join/exchange" method="POST"'
---

<objective>
Complete Phase 87 by (1) replacing the session-list dashboard with a landing page, (2) adding the join-code flow page at /join, (3) adding the POST /join/exchange server route that consumes a join code and 303-redirects to the capability-bearing URL, (4) rewiring the terminal page to determine read-only state from `GET /api/sessions/{id}/info?cap=<token>` (not `?readonly=1`), and (5) adding an end-to-end integration test that proves the full flow. This plan closes out UI-SPEC Surface 4 and Surface 5.

Purpose: Deliver the public-facing browser surface. Without this plan, a recipient cannot go from "I received a shared link" to "I'm viewing the session" without the grant-issuer manually copy-pasting the capability URL.

Output: Three HTML files updated/created, one embed.go if needed to wire new HTML, three new server handlers, one integration test. End-to-end: share-generate-scan-join flow works from a real browser.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-UI-SPEC.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md
@internal/webserver/server.go
@web/dashboard.html
@web/terminal.html
@web/embed.go

<interfaces>
New HTTP routes registered in server.go setupRoutes (NOT capability-gated — public):
```
GET  /dashboard          -> handleDashboard (serves landing page)
GET  /join               -> handleJoin (serves join page; reads ?code= query for pre-fill)
POST /join/exchange      -> handleJoinExchange (consumes code; 303 to /sessions/{id}?cap=<token>)
```

Existing terminal page (web/terminal.html) modified:
- On load, fetch GET /api/sessions/{id}/info?cap=<captured-from-url>
- If response.perms === "read", set xterm disableStdin: true and render READ ONLY badge
- Remove any existing `?readonly=1` query-string read
</interfaces>

<server_endpoints>
handleDashboard: serves web/dashboard.html. No capability required — it is a landing page.
handleJoin: serves web/join.html. Reads ?code= from query string. No capability required.
handleJoinExchange (POST): form body `code=<value>`. Calls a.daemonClient.ExchangeJoinCode OR the equivalent daemon call; on success, HTTP 303 Location: /sessions/{id}?cap=<token>. On ErrCodeExpired: 410 Gone. On ErrCodeNotFound: 404. Serve a simple HTML error page (join.html in error state) rather than plain text for UX.
handleSessionInfo (if not already added in Plan 03): GET /api/sessions/{id}/info?cap=<token>. Capability-gated via requireCapability middleware. Response: JSON {id, name, perms}.
</server_endpoints>
</context>

<tasks>

<task type="auto">
  <id>87-06-01</id>
  <name>Task 1: Replace web/dashboard.html with landing page; create web/join.html (5 state variants); wire embed.go and new server routes</name>
  <files>web/dashboard.html, web/join.html, web/embed.go, internal/webserver/server.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/web/dashboard.html (full — current structure, palette, inline CSS)
    - /Users/ken/dev/agenthub/web/embed.go (full — go:embed pattern)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-UI-SPEC.md (lines 185-265 Surface 3 + Surface 4 full specs with exact copy and CSS)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 815-861 dashboard.html + join.html analogs)
    - /Users/ken/dev/agenthub/internal/webserver/server.go (full — current handleDashboard at ~293; setupRoutes registration)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 660-694 handleJoinExchange handler pattern)
  </read_first>
  <behavior>
    - web/dashboard.html becomes a landing page (D-17, UI-SPEC Surface 3). No session list. Contains: AgentHub header, tagline, "Join a Shared Session" section with join-code input form, QR scan hint. Input auto-formats: uppercase, insert dash after 4th char, strip non-base32 characters. Submit action = GET /join?code=<value>.
    - web/join.html renders one of 5 state variants (A happy path, B no-code, C expired, D invalid, E session gone). State A shows session name + permission badge + "Join Session" button that POSTs to /join/exchange. State B shows the code input form. States C/D/E show error message + "Go Back" button that navigates to /dashboard.
    - Simplest state selection: query-string-driven. /join with no ?code -> State B. /join?code=X -> State A (server renders; if exchange later fails, error page is returned by handleJoinExchange itself).
    - server.go: handleDashboard serves dashboard.html. handleJoin serves join.html. handleJoinExchange (POST) parses form, calls ExchangeJoinCode-equivalent, HTTP 303 Location on success. All three routes registered in setupRoutes WITHOUT requireCapability wrapping — these are public landing/join surfaces.
  </behavior>
  <action>
    1. REPLACE `web/dashboard.html` in full. Required content per UI-SPEC lines 185-209:
       - Shell: DOCTYPE html, lang="en", viewport meta, title "AgentHub Dashboard".
       - Inline `<style>` with Tokyo Night palette (carry forward from existing file lines 9-19): background #1a1b26, text #c0caf5, button #7aa2f7, label #a9b1d6, input background #1e2030, border #292e42. Container max-width 480px, margin 0 auto, padding 2rem 1rem.
       - Body:
         - `<h1>AgentHub</h1>` (1.5rem)
         - `<p>Shared terminal sessions, securely.</p>` (0.9rem, #a9b1d6)
         - Horizontal rule: `<hr style="border-top: 1px solid #292e42; margin: 2rem 0; border-bottom: none;">`
         - `<h2>Join a Shared Session</h2>` (1.1rem)
         - `<p>Enter the join code from the person sharing their session, or scan the QR code they shared with your mobile device.</p>`
         - Form: `<form action="/join" method="GET">` with `<label for="code">Join Code</label>`, `<input id="code" name="code" type="text" placeholder="A7K-4P2N" maxlength="9" autocomplete="off" autocapitalize="characters" spellcheck="false" />`, `<button type="submit">Join Session</button>`.
         - `<p style="color: #565f89; font-style: italic; font-size: 0.9rem; margin-top: 1rem;">On mobile? Scan the QR code to auto-fill the code and open this page.</p>`
         - `<script>` block with vanilla JS for input formatting: on `input` event, strip non-`[A-Z2-7-]` chars (after toUpperCase), remove dashes, insert single dash between chars 4-5, truncate to 9 chars. Also strip on paste event.
       - DELETE all existing session-list JS (`#session-list`, `refreshSessions()`, `renderSessions()`, `.session-card` styles etc).

    2. CREATE `web/join.html`. Simplest implementation: a single HTML file whose JS inspects `window.location.search` for `?code=` and shows the matching state via `<div class="state state--a" hidden>` blocks. All 5 states are siblings; a `<script>` at the end un-hides the correct one based on URL params and any `?error=` marker that handleJoinExchange redirects to on failure.

       Alternative (cleaner): render 5 separate `<template id="state-a">` blocks and clone the active one into a visible container.

       Required states per UI-SPEC lines 221-257:
       - State A (code pre-filled, presumed valid): `<h1>AgentHub</h1>`, `<h2>Join Session</h2>`, session-name placeholder (populated via async fetch to `/api/sessions/info-by-code?code=X` if implemented, OR just show the code itself if not — Claude's Discretion), permission badge (defaults to showing nothing until the exchange happens), intent copy, `<form action="/join/exchange" method="POST"><input type="hidden" name="code" value="..."><button class="join-btn">Join Session</button></form>`, code display.
       - State B (no code): show the dashboard's join-code input form; action=/join, method=GET.
       - State C (expired): `<h2>Link Expired</h2>` in #f7768e, body text, "Go Back" button -> /dashboard.
       - State D (invalid): `<h2>Invalid Code</h2>`, body text, "Go Back".
       - State E (session gone): `<h2>Session Ended</h2>`, body text, "Go Back".

       State selection rules (all via JS at page bottom):
       ```
       const p = new URLSearchParams(window.location.search);
       const code = p.get('code');
       const err  = p.get('error');  // "expired", "invalid", "session-gone"
       if (err === 'expired') showState('c');
       else if (err === 'invalid') showState('d');
       else if (err === 'session-gone') showState('e');
       else if (code) { document.querySelector('input[name="code"]').value = code; showState('a'); }
       else showState('b');
       ```

       CSS (inline `<style>` block) per UI-SPEC lines 259-264:
       - Same palette as dashboard.html.
       - `.join-session-name` 1rem/600/#c0caf5 margin-bottom 8px
       - `.join-perm-badge` inline-block 12px 600 padding 4px 10px border-radius 3px
       - `.join-perm-badge--readonly` bg #292e42 color #a9b1d6
       - `.join-perm-badge--readwrite` bg rgba(122,162,247,0.15) color #7aa2f7 border 1px solid #7aa2f7
       - `.join-btn` width 100% padding 0.75rem bg #7aa2f7 color #1a1b26 border none border-radius 4px font-size 1rem font-weight 600 cursor pointer
       - `.join-code-display` monospace 1.4rem color #c0caf5 letter-spacing 0.1em text-align center margin 1rem 0

    3. Edit `web/embed.go` — check if it uses `//go:embed web/*.html` or wildcard. If `dashboard.html` is already embedded via wildcard, `join.html` will be picked up automatically. If embedded by explicit name, add `join.html` to the directive.

    4. Edit `internal/webserver/server.go`:
       - Locate the existing `handleDashboard` handler (~line 293). Replace its body to serve `web/dashboard.html` from the embedded FS (it likely already does — just ensure no session-list logic lingers). Since the HTML is already updated, no Go code change required IF the handler just writes the embedded content.
       - Add `handleJoin`:
         ```go
         func (ws *WebServer) handleJoin(w http.ResponseWriter, r *http.Request) {
             // Public page; no capability required.
             w.Header().Set("Content-Type", "text/html; charset=utf-8")
             _, _ = w.Write(joinHTML)  // joinHTML is embedded bytes
         }
         ```
       - Add `handleJoinExchange`:
         ```go
         func (ws *WebServer) handleJoinExchange(w http.ResponseWriter, r *http.Request) {
             if err := r.ParseForm(); err != nil {
                 http.Redirect(w, r, "/join?error=invalid", http.StatusSeeOther); return
             }
             code := r.FormValue("code")
             if code == "" {
                 http.Redirect(w, r, "/join?error=invalid", http.StatusSeeOther); return
             }
             token, err := ws.joinCodes.Exchange(code)  // joinCodes passed into WebServer at construction OR via accessor
             switch {
             case errors.Is(err, capability.ErrCodeExpired):
                 http.Redirect(w, r, "/join?error=expired", http.StatusSeeOther); return
             case errors.Is(err, capability.ErrCodeNotFound):
                 http.Redirect(w, r, "/join?error=invalid", http.StatusSeeOther); return
             case err != nil:
                 http.Error(w, "internal error", http.StatusInternalServerError); return
             }
             claims, err := capability.Verify(token, ws.currentSigningKey())
             if err != nil {
                 http.Error(w, "internal error", http.StatusInternalServerError); return
             }
             if !ws.IsSessionEnabled(claims.SID) {
                 http.Redirect(w, r, "/join?error=session-gone", http.StatusSeeOther); return
             }
             target := "/sessions/" + claims.SID + "?cap=" + token
             http.Redirect(w, r, target, http.StatusSeeOther)
         }
         ```
         Note: This requires the WebServer to have access to the `joinCodes` manager. If Plan 04 kept `joinCodes` on the API (daemon) and not the WebServer, two options: (a) pass the JoinCodeManager pointer into the WebServer at construction (add a `joinCodes *capability.JoinCodeManager` field and setter), or (b) route the exchange through the daemon client — NOT preferred because the WebServer and daemon are in the same process. Go with option (a): have Plan 04's api.go set `a.webServer.joinCodes = a.joinCodes` right before SetSigningKey, and add a `SetJoinCodes(m *capability.JoinCodeManager)` method on WebServer analogous to SetSigningKey. If Plan 04 already wired it, skip; if not, add the setter and call here.

       - Register routes in setupRoutes (NOT capability-gated):
         ```go
         mux.HandleFunc("GET /dashboard",      ws.handleDashboard)
         mux.HandleFunc("GET /join",           ws.handleJoin)
         mux.HandleFunc("POST /join/exchange", ws.handleJoinExchange)
         ```

    5. Verify existing dashboard handler: ensure the session-list JSON endpoint no longer comes from `/dashboard` — that route is HTML-only. `/api/sessions` (Plan 03, cap-gated) is still the JSON endpoint.

    6. Run: `go build ./...` — must succeed. `go test ./internal/webserver/... -count=1` — all tests still PASS.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && ls web/dashboard.html web/join.html && grep -q 'Join a Shared Session' web/dashboard.html && ! grep -q 'session-list\|renderSessions' web/dashboard.html && grep -q 'Join Session' web/join.html && grep -q 'Link Expired' web/join.html && grep -q 'Invalid Code' web/join.html && grep -q 'Session Ended' web/join.html && grep -q 'action="/join/exchange" method="POST"' web/join.html && grep -q 'handleJoin\|handleJoinExchange' internal/webserver/server.go && grep -q '"GET /join"\|"GET /dashboard"\|"POST /join/exchange"' internal/webserver/server.go && go build ./... && go test ./internal/webserver/... -count=1 2>&1 | tee /tmp/ws-pages.log ; ! grep -q FAIL /tmp/ws-pages.log</automated>
  </verify>
  <acceptance_criteria>
    - `ls web/dashboard.html web/join.html` both exist
    - dashboard.html contains "Join a Shared Session" and does NOT contain "session-list" or "renderSessions"
    - join.html contains all 5 state headings: "Join Session", "Link Expired", "Invalid Code", "Session Ended", plus B state form
    - join.html has `<form action="/join/exchange" method="POST">`
    - server.go registers 3 public routes (GET /dashboard, GET /join, POST /join/exchange) WITHOUT requireCapability
    - server.go handleJoinExchange maps ErrCodeExpired to `?error=expired`, ErrCodeNotFound to `?error=invalid`, and gone-session to `?error=session-gone` via 303
    - `go build ./...` succeeds
    - Existing `go test ./internal/webserver/...` passes (no regression)
  </acceptance_criteria>
  <done>UI-SPEC Surface 3 + Surface 4 live. Dashboard is a landing page with no session list. Join page handles happy + 3 error states. POST /join/exchange redirects to the capability URL.</done>
</task>

<task type="auto" tdd="true">
  <id>87-06-02</id>
  <name>Task 2: Update terminal page to suppress caret from cap perms (D-23); add end-to-end integration test</name>
  <files>web/terminal.html, internal/webserver/capability_test.go, internal/webserver/server.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/web/terminal.html (full — find xterm initialization; locate current ?readonly=1 read; locate status bar render code)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-UI-SPEC.md (lines 267-294 Surface 5 caret suppression spec; lines 334 READ ONLY badge copy)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 611-615 Pitfall 7 ?readonly=1 bookmark migration)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 488-510 handleSessionInfo / perms source)
    - /Users/ken/dev/agenthub/internal/webserver/capability_test.go (active test file from Plan 03 activation)
  </read_first>
  <behavior>
    - terminal.html reads ?cap= from window.location.search on page load, then fetches `/api/sessions/{id}/info?cap=<captured>`. If response.perms === "read", initialize xterm with `disableStdin: true` and render a "READ ONLY" badge in the status bar. If perms contains "write", normal behavior.
    - DELETE any existing ?readonly=1 read from terminal.html (RESEARCH Pitfall 7).
    - Integration test TestEndToEnd_CapabilityFlow: spin up test server with hub, issue a capability via AddGrant + Sign directly (bypassing IPC), GET /api/sessions/{id}/info with cap -> assert response includes perms:"read", GET /sessions/{id}/ws with cap -> successful upgrade, send MsgInput on a read-only cap -> PTY never receives the bytes (readPipeWithTimeout).
  </behavior>
  <action>
    1. Edit `web/terminal.html`. Find the xterm initialization block (likely contains `new Terminal(...)` or `new xterm.Terminal(...)`). Add before it:
       ```html
       <script>
         (async function initReadOnly() {
           try {
             const params = new URLSearchParams(window.location.search);
             const cap = params.get('cap');
             const pathParts = window.location.pathname.split('/');
             const sid = pathParts[pathParts.indexOf('sessions') + 1];
             if (!cap || !sid) { window.__perms = 'read,write'; return; }
             const resp = await fetch(`/api/sessions/${sid}/info?cap=${encodeURIComponent(cap)}`);
             if (!resp.ok) { window.__perms = 'read'; return; } // fail safe — most restrictive
             const info = await resp.json();
             window.__perms = info.perms || 'read';
           } catch (e) {
             window.__perms = 'read'; // fail safe
           }
         })();
       </script>
       ```

       Then in the xterm init block, change the options to consult `window.__perms`:
       ```javascript
       const isReadOnly = (window.__perms || 'read') === 'read';
       const term = new Terminal({
         // ...existing options...
         disableStdin: isReadOnly,
       });

       // After status bar init, render READ ONLY badge:
       if (isReadOnly) {
         const badge = document.createElement('span');
         badge.className = 'terminal-status__readonly-badge';
         badge.textContent = 'READ ONLY';
         badge.style.cssText = 'display:inline-block;padding:2px 8px;background:transparent;color:#a9b1d6;font-size:11px;font-weight:600;letter-spacing:0.05em;text-transform:uppercase;margin-left:8px;border:1px solid #292e42;border-radius:3px;';
         document.querySelector('.tab-status-bar,#status-bar,.terminal-status').appendChild(badge); // adapt selector to existing status bar container
       }
       ```

       Since the init fetch is async but xterm init is synchronous, either:
       (a) Wrap xterm init in the same async IIFE so it waits for the fetch, OR
       (b) Init xterm with disableStdin:true by default, then call `term.options.disableStdin = false` after perms resolves as "read,write".

       Pick (a) — simpler and avoids flash of enabled input.

       IMPORTANT: SEARCH the file for `readonly=` and REMOVE any existing usage (Pitfall 7). If terminal.html has code like `const isReadOnly = params.get('readonly') === '1';` — delete it. Replace with the `window.__perms` path above.

    2. Add `handleSessionInfo` in `internal/webserver/server.go` if not already present from Plan 03. Handler body:
       ```go
       func (ws *WebServer) handleSessionInfo(w http.ResponseWriter, r *http.Request) {
           claims, ok := capability.ClaimsFromContext(r.Context())
           if !ok { http.Error(w, "capability required", http.StatusUnauthorized); return }
           var name string
           if ws.sessionResolver != nil {
               name, _, _, _ = ws.sessionResolver(claims.SID)
           }
           if name == "" { name = claims.SID }
           resp := struct {
               ID    string `json:"id"`
               Name  string `json:"name"`
               Perms string `json:"perms"`
           }{ID: claims.SID, Name: name, Perms: claims.Perms}
           w.Header().Set("Content-Type", "application/json")
           _ = json.NewEncoder(w).Encode(resp)
       }
       ```
       Register route (capability-gated): `mux.HandleFunc("GET /api/sessions/{id}/info", ws.requireCapability(ws.handleSessionInfo))`.

    3. Add integration test `TestEndToEnd_CapabilityFlow` to `internal/webserver/capability_test.go`:
       ```go
       func TestEndToEnd_CapabilityFlow(t *testing.T) {
           ws, hub, client := capTestServerWithHub(t)
           key := make([]byte, 32)
           for i := range key { key[i] = byte(i) }
           ws.SetSigningKey(key)

           sid := "e2e-sess"
           ws.EnableSession(sid)
           _ = hub
           // Issue read-only cap directly (simulating what Plan 04 does on toggle-on)
           roGrantID := "grant-ro"
           rwGrantID := "grant-rw"
           ws.AddGrant(sid, roGrantID)
           ws.AddGrant(sid, rwGrantID)

           roClaims := capability.Claims{SID: sid, Perms: "read",       IAT: 1, GrantID: roGrantID, V: 1}
           rwClaims := capability.Claims{SID: sid, Perms: "read,write", IAT: 1, GrantID: rwGrantID, V: 1}
           roTok, _ := capability.Sign(roClaims, key)
           rwTok, _ := capability.Sign(rwClaims, key)

           // 1. Info endpoint returns perms
           resp, err := client.Get(ws.BaseURL() + "/api/sessions/" + sid + "/info?cap=" + roTok)
           if err != nil { t.Fatal(err) }
           defer resp.Body.Close()
           if resp.StatusCode != 200 { t.Fatalf("info: expected 200, got %d", resp.StatusCode) }
           var info struct{ Perms string `json:"perms"` }
           _ = json.NewDecoder(resp.Body).Decode(&info)
           if info.Perms != "read" { t.Errorf("info.perms = %q, want read", info.Perms) }

           // 2. Info endpoint with rw cap returns read,write
           resp2, _ := client.Get(ws.BaseURL() + "/api/sessions/" + sid + "/info?cap=" + rwTok)
           defer resp2.Body.Close()
           _ = json.NewDecoder(resp2.Body).Decode(&info)
           if info.Perms != "read,write" { t.Errorf("info.perms = %q, want read,write", info.Perms) }

           // 3. /dashboard is public (no cap) and returns HTML
           resp3, _ := client.Get(ws.BaseURL() + "/dashboard")
           defer resp3.Body.Close()
           if resp3.StatusCode != 200 { t.Errorf("/dashboard: expected 200, got %d", resp3.StatusCode) }

           // 4. Revoke via ClearGrants -> info endpoint returns 403
           ws.ClearGrants(sid)
           resp4, _ := client.Get(ws.BaseURL() + "/api/sessions/" + sid + "/info?cap=" + roTok)
           defer resp4.Body.Close()
           if resp4.StatusCode != 403 { t.Errorf("revoked info: expected 403, got %d", resp4.StatusCode) }
       }
       ```

    4. Run `go test ./internal/webserver/ -count=1 -v -run TestEndToEnd_CapabilityFlow` — PASS.

    5. Run full suite: `go test ./... -count=1` — all green.

    Anti-patterns:
    - DO NOT keep any `readonly=1` read in terminal.html (Pitfall 7)
    - DO NOT fail-open on fetch error — default to read-only (fail-safe per SEC-04)
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && ! grep -q 'readonly=1\|readonly%3D1' web/terminal.html && grep -q 'window.__perms\|__perms' web/terminal.html && grep -q '/api/sessions/.*/info' web/terminal.html && grep -q 'READ ONLY' web/terminal.html && grep -q 'disableStdin' web/terminal.html && grep -q 'handleSessionInfo' internal/webserver/server.go && go test ./internal/webserver/ -count=1 -v -run TestEndToEnd_CapabilityFlow 2>&1 | tee /tmp/e2e.log ; ! grep -q FAIL /tmp/e2e.log && go test ./... -count=1 2>&1 | tee /tmp/full.log ; ! grep -q FAIL /tmp/full.log</automated>
  </verify>
  <acceptance_criteria>
    - `grep -q "readonly=1" web/terminal.html` FAILS (Pitfall 7 — old mechanism removed)
    - `grep -q "/api/sessions/.*/info" web/terminal.html` succeeds (new fetch path)
    - `grep -q "READ ONLY" web/terminal.html` succeeds (badge copy)
    - `grep -q "disableStdin" web/terminal.html` succeeds (xterm option set)
    - `grep -q "handleSessionInfo" internal/webserver/server.go` succeeds
    - `grep -q "GET /api/sessions/{id}/info" internal/webserver/server.go` succeeds
    - TestEndToEnd_CapabilityFlow PASSES
    - `go test ./... -count=1` PASSES with no regressions
  </acceptance_criteria>
  <done>Terminal page fail-safely determines read-only state from server-verified capability perms, not ?readonly query. End-to-end test proves the full flow. SEC-04 enforced at the UI layer.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| public browser → /dashboard and /join | Landing pages: no capability required; they only collect/display, never grant |
| browser form → POST /join/exchange | Single-use join code traversal; server-side validation via capability.Verify + joinCodes.Exchange |
| terminal page JS → /api/sessions/{id}/info | Fetches perms over capability-gated endpoint; fail-safe default is read-only |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-87-04 | Elevation of Privilege | read-only bypass via terminal client | mitigate | Terminal page sources disableStdin from server-verified perms; fetch failure defaults to read-only (fail-safe); task 87-06-02 |
| T-87-07 | Elevation of Privilege | join code replay via browser back-button | mitigate | POST /join/exchange is a single-use consumer (Plan 02 atomic delete); browser back produces a second POST which returns 404 via ErrCodeNotFound |
| T-87-03 | Spoofing | forged token in /join/exchange | mitigate | handleJoinExchange calls capability.Verify after exchange; failure returns 500, never 303 |
| T-87-01 | Information Disclosure | landing page enumerates sessions | mitigate | /dashboard contains NO session list (D-17); static-grep gate in task 87-06-01 |
</threat_model>

<verification>
- `go test ./... -count=1` green (capability + webserver + daemon + all)
- `cd frontend && pnpm build` green
- `grep -q "readonly=1" web/` FAILS everywhere (SEC-04 / Pitfall 7)
- `grep -q "session-list\|renderSessions" web/dashboard.html` FAILS (D-17)
- All 5 Wave 0 SEC tests plus TestEndToEnd_CapabilityFlow all PASS
</verification>

<success_criteria>
- Dashboard renders as landing page (D-17)
- Join page supports all 5 UI-SPEC state variants
- POST /join/exchange is single-use, 303-redirects on success, 410/404 on expired/invalid
- Terminal page suppresses caret from server-verified perms, not client query string (D-23)
- End-to-end integration test passes
- Phase 87 as a whole: all SEC-01..SEC-05 tests PASS, `go test ./...` clean
</success_criteria>

<output>
After completion, create `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-06-SUMMARY.md` documenting: the dashboard-landing-page conversion, the 5-state join page, the POST /join/exchange flow, the terminal caret-suppression mechanism, and the end-to-end test assertion set. Close the phase-level SUMMARY by listing which original 5 success criteria (ROADMAP.md §Phase 87) are now demonstrably true via which test.
</output>
