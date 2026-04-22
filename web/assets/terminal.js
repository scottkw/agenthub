    // Binary framing protocol constants (must match relay/protocol.go)
    const MsgOutput  = 0x01;
    const MsgInput   = 0x10;
    const MsgResize2 = 0x11;
    const MsgPing    = 0x12;

    function makeFrame(type, payload) {
      const frame = new Uint8Array(1 + payload.length);
      frame[0] = type;
      frame.set(payload, 1);
      return frame;
    }

    function makeInputFrame(text) {
      const enc = new TextEncoder().encode(text);
      return makeFrame(MsgInput, enc);
    }

    function makeResizeFrame(cols, rows) {
      const payload = new Uint8Array(4);
      payload[0] = (cols >> 8) & 0xff;
      payload[1] = cols & 0xff;
      payload[2] = (rows >> 8) & 0xff;
      payload[3] = rows & 0xff;
      return makeFrame(MsgResize2, payload);
    }

    // Extract session ID from URL path: /sessions/{id}
    const pathParts = location.pathname.split('/');
    const sessionID = pathParts[pathParts.indexOf('sessions') + 1] || '';

    // Extract capability token from query params. The cap is the ONLY
    // authorization credential the server accepts — there is no ?readonly
    // hint anymore (D-23 / Pitfall 7). Read-only state is sourced from the
    // server-verified perms claim via /api/sessions/{id}/info below.
    const params = new URLSearchParams(location.search);
    const cap = params.get('cap') || '';

    function withCap(path) {
      if (!cap) return path;
      return path + (path.indexOf('?') === -1 ? '?' : '&') + 'cap=' + encodeURIComponent(cap);
    }

    // Build WSS URL (includes ?cap= via withCap).
    const wsURL = 'wss://' + location.host + withCap('/sessions/' + encodeURIComponent(sessionID) + '/ws');

    // Session metadata state
    let sessionMeta = null;
    let connected = false;
    const statusDot = document.getElementById('status-dot');
    const sessionInfoEl = document.getElementById('session-info');
    const statusBar = document.getElementById('web-status-bar');

    function updateStatusBar(meta, wsConnected) {
      if (meta) {
        var parts = [meta.name || sessionID];
        if (meta.cli_type) parts.push(meta.cli_type);
        if (meta.hostname) parts.push(meta.hostname);
        sessionInfoEl.textContent = parts.join(' \u2502 ');
      }

      statusDot.className = 'status-dot';
      if (wsConnected) {
        statusDot.classList.add('status-dot--connected');
      } else if (meta === null) {
        statusDot.classList.add('status-dot--connecting');
      } else {
        statusDot.classList.add('status-dot--disconnected');
      }
    }

    function fetchSessionInfo() {
      fetch(withCap('/api/sessions/' + encodeURIComponent(sessionID) + '/info'))
        .then(function(r) { return r.ok ? r.json() : null; })
        .then(function(data) {
          if (data) {
            sessionMeta = data;
            updateStatusBar(sessionMeta, connected);
          }
        })
        .catch(function() {
          // Fetch failed — server unreachable, mark disconnected
          updateStatusBar(sessionMeta, false);
        });
    }

    // Kick off the terminal wiring. The initial perms fetch is awaited before
    // xterm is constructed so the input caret is never momentarily enabled for
    // a read-only capability (UI flicker would contradict the SEC-04 guarantee
    // visually, even though the server blocks writes regardless).
    //
    // Fail-safe default: if the perms fetch errors or the cap is missing, we
    // assume 'read' and disable stdin. This is the most restrictive default
    // per SEC-04 and D-23.
    (async function initTerminal() {
      var perms = 'read';
      if (cap && sessionID) {
        try {
          var resp = await fetch(withCap('/api/sessions/' + encodeURIComponent(sessionID) + '/info'));
          if (resp.ok) {
            var info = await resp.json();
            sessionMeta = info;
            if (typeof info.perms === 'string' && info.perms.length > 0) {
              perms = info.perms;
            }
          }
        } catch (e) {
          // Fall through with perms = 'read' (fail-safe).
        }
      }
      window.__perms = perms;
      var isReadOnly = perms === 'read';

      updateStatusBar(sessionMeta, connected);
      // Continue polling session info for status-bar updates.
      setInterval(fetchSessionInfo, 3000);

      // Initialize xterm.js with perms-bound disableStdin.
      var term = new Terminal({
        cursorBlink: !isReadOnly,
        disableStdin: isReadOnly,
        theme: {
          background: '#1a1b26',
          foreground: '#cccccc',
          cursor: '#cccccc',
        },
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        fontSize: 14,
      });

      var fitAddon = new FitAddon.FitAddon();
      term.loadAddon(fitAddon);
      term.open(document.getElementById('terminal'));
      fitAddon.fit();

      if (isReadOnly) {
        var badge = document.createElement('span');
        badge.className = 'readonly-badge';
        badge.textContent = 'READ ONLY';
        statusBar.appendChild(badge);
      }

      // Open WebSocket connection
      var ws = null;

      function connect() {
        ws = new WebSocket(wsURL);
        ws.binaryType = 'arraybuffer';

        ws.onopen = function() {
          connected = true;
          updateStatusBar(sessionMeta, true);
          if (term.cols && term.rows) {
            ws.send(makeResizeFrame(term.cols, term.rows));
          }
        };

        ws.onmessage = function(evt) {
          var data = new Uint8Array(evt.data);
          if (data.length === 0) return;

          var msgType = data[0];
          var payload = data.slice(1);

          if (msgType === MsgOutput) {
            var text = new TextDecoder().decode(payload);
            term.write(text);
          }
          // Other message types from server are ignored in browser client
        };

        ws.onclose = function() {
          connected = false;
          updateStatusBar(sessionMeta, false);
        };

        ws.onerror = function() {
          connected = false;
          updateStatusBar(sessionMeta, false);
        };
      }

      // Terminal input -> send MsgInput frame. We STILL install the handler on
      // a read-only terminal so that any keyboard events that leak through the
      // disableStdin gate (e.g. paste via middle-click in some browsers) are
      // gated here too. The server drops MsgInput on read-only caps anyway
      // (SEC-04), so this is defense-in-depth at the UI layer.
      term.onData(function(data) {
        if (isReadOnly) return;
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(makeInputFrame(data));
        }
      });

      // Terminal resize -> send MsgResize2 frame
      term.onResize(function(size) {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(makeResizeFrame(size.cols, size.rows));
        }
      });

      // Window resize -> fit terminal
      window.addEventListener('resize', function() {
        fitAddon.fit();
      });

      connect();
    })();
