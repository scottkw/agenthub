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

      // Phase 93 PLUG-04 / WEB-03: fetch plugin config to gate addon loading.
      // Defaults to all-on (matches Phase 92 daemon defaults for v3.2 returning
      // users) if fetch fails. Failure paths are silent per UI-SPEC — CSP /
      // capability errors are handled at the network layer.
      var pluginConfig = {
        webgl: true, unicode11: true, clipboard: true,
        search: true, webLinks: true, image: true,
        serialize: true, progress: false
      };
      if (cap && sessionID) {
        try {
          var pcResp = await fetch(withCap('/api/plugin-config'));
          if (pcResp.ok) {
            var pc = await pcResp.json();
            // Defensive merge — daemon may add new keys before web is updated.
            for (var k in pc) {
              if (Object.prototype.hasOwnProperty.call(pc, k)) pluginConfig[k] = pc[k];
            }
          }
        } catch (e) {
          // Silent fall-through with defaults.
        }
      }

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

      // Phase 93 WGL-03: software-rasterizer probe (web parity with desktop webglProbe.ts)
      function isSoftwareWebGL() {
        try {
          var c = document.createElement('canvas');
          var gl = c.getContext('webgl') || c.getContext('experimental-webgl');
          if (!gl) return false;
          var renderer = gl.getParameter(gl.RENDERER);
          if (!renderer) return false;
          return /SwiftShader|llvmpipe|ANGLE.*Software|ANGLE.*SwiftShader/i.test(renderer);
        } catch (e) { return false; }
      }

      // Phase 93 WGL-02: context-loss / software-preemption banner.
      // One-shot per session via sessionStorage (best-effort — may be blocked
      // in some browser modes, in which case the banner can show again).
      function showWebGLBanner(reason) {
        try {
          if (sessionStorage.getItem('webgl-banner-shown') === '1') return;
          sessionStorage.setItem('webgl-banner-shown', '1');
        } catch (e) { /* sessionStorage may be blocked — best-effort */ }
        var el = document.getElementById('webgl-recovery-banner');
        if (!el) return;
        var msg = reason === 'software-rasterized'
          ? 'Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.'
          : 'Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact.';
        el.innerHTML = '';
        var span = document.createElement('span');
        span.className = 'webgl-recovery-banner__message';
        span.textContent = msg;
        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'webgl-recovery-banner__dismiss';
        btn.setAttribute('aria-label', 'Dismiss notification');
        btn.textContent = '×'; // × Unicode multiplication sign
        btn.addEventListener('click', function() { el.hidden = true; });
        el.appendChild(span);
        el.appendChild(btn);
        el.hidden = false;
        // Auto-dismiss after 8s only for context-loss (not software-rasterized).
        if (reason !== 'software-rasterized') {
          setTimeout(function() { el.hidden = true; }, 8000);
        }
      }

      // Phase 93 U11-02: server-shared Unicode 11. Applied at construction
      // ONLY — server-shared semantics mean a running session must NOT switch
      // its width tables mid-buffer (would corrupt scrollback on existing
      // characters). The next page load picks up any change automatically.
      if (pluginConfig.unicode11) {
        try {
          var u11 = new Unicode11Addon.Unicode11Addon();
          term.loadAddon(u11);
          term.unicode.activeVersion = '11';
        } catch (e) { /* addon UMD may not be present — silent */ }
      }

      // Phase 93 hot-swap-capable addon handles. Declared at IIFE scope so the
      // SSE EventSource handler (Task 5 below) can dispose / reconstruct them
      // without a page reload.
      var webglAddonHandle = null;
      var clipboardAddonHandle = null;

      // applyPluginConfig diff-applies a new pluginConfig against the current
      // state. Used both at initial load (vs. an "everything-off" prev) and on
      // each SSE plugin-config push frame. Phase 93 PLUG-04 hot-swap path.
      var lastApplied = '';
      function applyPluginConfig(newConfig) {
        var prev = pluginConfig;
        pluginConfig = newConfig;

        // WebGL hot-swap.
        if (!newConfig.webgl && webglAddonHandle) {
          try { webglAddonHandle.dispose(); } catch (e) {}
          webglAddonHandle = null;
        } else if (newConfig.webgl && !webglAddonHandle && !isSoftwareWebGL()) {
          try {
            webglAddonHandle = new WebglAddon.WebglAddon();
            webglAddonHandle.onContextLoss(function() {
              try { webglAddonHandle.dispose(); } catch (e) {}
              webglAddonHandle = null;
              showWebGLBanner('context-loss');
            });
            term.loadAddon(webglAddonHandle);
          } catch (e) {
            // WebGL construction failed — silent (user enabled but env refused).
          }
        } else if (newConfig.webgl && !webglAddonHandle && isSoftwareWebGL() && !prev.webgl) {
          // Newly toggled on, but software-rasterized — show preemption banner.
          showWebGLBanner('software-rasterized');
        }

        // Clipboard hot-swap. CLIP-02: read-only viewers never get clipboard.
        if (!newConfig.clipboard && clipboardAddonHandle) {
          try { clipboardAddonHandle.dispose(); } catch (e) {}
          clipboardAddonHandle = null;
        } else if (newConfig.clipboard && !clipboardAddonHandle && window.__perms !== 'read') {
          try {
            clipboardAddonHandle = new ClipboardAddon.ClipboardAddon();
            term.loadAddon(clipboardAddonHandle);
          } catch (e) { /* silent */ }
        }

        // Unicode 11 deliberately NOT hot-swapped here — see comment at the
        // initial-load block above.

        lastApplied = JSON.stringify(newConfig);
      }

      // Phase 93 WGL-04: initial WebGL/Clipboard application via diff-apply
      // against an everything-off seed. Also handles the at-startup software-
      // rasterizer preemption banner.
      if (pluginConfig.webgl && isSoftwareWebGL()) {
        showWebGLBanner('software-rasterized');
      }
      var initialConfig = pluginConfig;
      pluginConfig = {
        webgl: false, unicode11: false, clipboard: false,
        search: false, webLinks: false, image: false,
        serialize: false, progress: false
      };
      applyPluginConfig(initialConfig);

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

      // Phase 93 PLUG-04 push channel — subscribe to live plugin-config
      // updates so toggle changes apply WITHOUT a page reload (closes
      // ROADMAP SC#4). Per UI-SPEC §"Web plugin-config live update":
      // silent on programmatic changes (no toast); reload-free apply path.
      var pluginConfigStream = null;
      if (cap && sessionID && typeof EventSource !== 'undefined') {
        try {
          pluginConfigStream = new EventSource(withCap('/api/plugin-config/stream'));
          pluginConfigStream.addEventListener('plugin-config', function(ev) {
            try {
              var pushed = JSON.parse(ev.data);
              // Defensive merge over current pluginConfig — additive merge
              // prevents a partial frame from disabling plugins the user has
              // on (T-93-WEB-03 mitigation).
              var merged = {};
              for (var k0 in pluginConfig) {
                if (Object.prototype.hasOwnProperty.call(pluginConfig, k0)) merged[k0] = pluginConfig[k0];
              }
              for (var k1 in pushed) {
                if (Object.prototype.hasOwnProperty.call(pushed, k1)) merged[k1] = pushed[k1];
              }
              if (JSON.stringify(merged) === lastApplied) return; // idempotent
              applyPluginConfig(merged);
            } catch (e) {
              // Malformed frame — silent. Browser will keep reading the next.
            }
          });
          pluginConfigStream.addEventListener('error', function() {
            // Browser auto-retries on transient network drops. On 401 (cap
            // expiry) the readyState transitions to CLOSED — stop retrying.
            if (pluginConfigStream && pluginConfigStream.readyState === EventSource.CLOSED) {
              pluginConfigStream = null;
            }
          });
          window.addEventListener('beforeunload', function() {
            if (pluginConfigStream) {
              pluginConfigStream.close();
              pluginConfigStream = null;
            }
          });
        } catch (e) {
          // EventSource construction failed — degrade to fetch-on-load only.
        }
      }

      connect();
    })();
