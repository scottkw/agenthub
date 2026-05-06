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
        serialize: true, progress: false,
        // Phase 94 SRC-05: SearchConfig defaults (web is read-only consumer
        // per UI-SPEC line 335 — canonical state arrives via /api/plugin-config
        // + SSE settings:plugins push; web does NOT write back to daemon).
        searchConfig: { regex: false, caseSensitive: false, wholeWord: false }
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
      // allowProposedApi: required for (a) unicode11 width-table addon and
      // (b) SearchAddon's registerDecoration calls when passing decorations: {}
      // (Phase 94 SRC-02 onDidChangeResults gating). Desktop TerminalPanel.tsx
      // has had this since Phase 93 — web parity restored here in Plan 94-05.
      var term = new Terminal({
        cursorBlink: !isReadOnly,
        disableStdin: isReadOnly,
        allowProposedApi: true,
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
      // Phase 89 / 94-04 / 94-05 e2e harness — expose Terminal for chromedp tests.
      // Production-safe: this is a write-only assignment to the window object,
      // mirrors the pattern already used by the perf harness for `window.term`.
      window.term = term;

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

      // Phase 94 SRC-05 — Find bar state + handles. Module-scope inside the
      // IIFE so wireFindBarHandlers() / showFindBar() / hideFindBar() / runSearch()
      // close over them.
      var searchAddonHandle = null;
      var searchResultsDisposable = null;
      var searchDebounceTimer = null;
      // Phase 94 WR-01 / SC-4 — pending exit-animation unmount timer. setTimeout
      // ID returned by hideFindBar; canceled by showFindBar on mid-exit re-open.
      var findBarExitTimer = null;
      var findBarOpen = false;
      var searchOptions = { regex: false, caseSensitive: false, wholeWord: false };
      var matchInfo = { index: -1, count: 0 };

      // ─── Phase 95 — web-links security helpers (mirror of frontend/src/lib/{urlSafety,openLink}.ts) ───
      // Plain ES5 inline copy because the embed has no module bundler. Behavior
      // and security invariants mirror the desktop helpers verbatim:
      //   - isAllowedScheme: defense-in-depth scheme allowlist (https/http/mailto only)
      //   - hasIDN: Punycode prefix OR non-ASCII codepoint on hostname (Cyrillic spoof gate)
      //   - osc8Mismatch: display-vs-href divergence (Plan B keeps the helper for parity;
      //     the click handler currently calls getRisk(uri, uri) so this never fires for
      //     plain-text URLs the addon emits — wired live in v3.3 per 95-RESEARCH spike outcome).
      //   - isTypoSquat: best-effort heuristic; NOT a security boundary (popover always surfaces URL).
      //   - getRisk: priority osc8 → idn → typosquat (mirrors urlSafety.ts JSDoc).
      //   - isModifierPressed: 'none' / 'platform' (mac=meta / non-mac=ctrl) / 'cmd' / 'ctrl'.
      //   - openLink: NEVER assigns to location.href; ALWAYS window.open with literal
      //     '_blank', 'noopener,noreferrer'; web context never has the Wails runtime
      //     opener — that is desktop-only (see frontend/src/lib/openLink.ts).
      var ALLOWED_SCHEMES = ['https:', 'http:', 'mailto:'];
      function isAllowedScheme(href) {
        try { return ALLOWED_SCHEMES.indexOf(new URL(href).protocol) !== -1; } catch (e) { return false; }
      }
      function hasIDN(href) {
        try {
          var u = new URL(href);
          if (u.hostname.indexOf('xn--') !== -1) return true;
          if (/[^\x00-\x7F]/.test(u.hostname)) return true;
          return false;
        } catch (e) {
          // URL constructor rejects some legacy Punycode labels. Fall back to a
          // raw-href regex on the host portion so we still surface IDN risk.
          var m = /^[a-z][a-z0-9+.-]*:\/\/([^/?#]+)/i.exec(String(href || ''));
          if (!m) return false;
          var host = m[1];
          if (host.indexOf('xn--') !== -1) return true;
          if (/[^\x00-\x7F]/.test(host)) return true;
          return false;
        }
      }
      function osc8Mismatch(displayText, href) {
        if (displayText === href) return false;
        try {
          var d = new URL(String(displayText).trim());
          var h = new URL(href);
          return d.host !== h.host || d.protocol !== h.protocol;
        } catch (e) { return true; }
      }
      var TYPOSQUAT_LIST = (function() {
        var s = {};
        var entries = [
          'paypa1.com','goog1e.com','arnazon.com','amaz0n.com','microsft.com',
          'micros0ft.com','app1e.com','git-hub.com','tw1tter.com','twltter.com',
          'face-book.com','faceb00k.com','linked1n.com','linkedln.com','g00gle.com',
          'goggle.com','youtub3.com','reddlt.com','instagrarn.com','wlkipedia.org',
          'netfllx.com','spot1fy.com','dropb0x.com','aple.com','rnicrosoft.com',
          'arnzon.com','gocgle.com','githab.com','gitlub.com','app1eid.com'
        ];
        for (var i = 0; i < entries.length; i++) s[entries[i]] = true;
        return s;
      })();
      function isTypoSquat(href) {
        try {
          var u = new URL(href);
          var host = u.hostname.toLowerCase().replace(/^www\./, '');
          return TYPOSQUAT_LIST[host] === true;
        } catch (e) { return false; }
      }
      function getRisk(displayText, href) {
        if (osc8Mismatch(displayText, href)) return 'osc8';
        if (hasIDN(href)) return 'idn';
        if (isTypoSquat(href)) return 'typosquat';
        return null;
      }
      function isModifierPressed(event, mode) {
        if (mode === 'none') return true;
        var isMac = navigator.platform.toUpperCase().indexOf('MAC') !== -1;
        if (mode === 'platform') return isMac ? !!event.metaKey : !!event.ctrlKey;
        if (mode === 'cmd') return !!event.metaKey;
        if (mode === 'ctrl') return !!event.ctrlKey;
        return false;
      }
      function openLink(url) {
        // Defense-in-depth scheme re-validation — never trust upstream callers.
        if (!/^(https?:|mailto:)/i.test(url)) return;
        // Web context: Wails runtime never present; window.open with the literal
        // '_blank', 'noopener,noreferrer' options string is the ONLY navigation
        // path. NEVER assign to location.href. NEVER set window.location.
        window.open(url, '_blank', 'noopener,noreferrer');
      }

      // Web-side link confirmation popover (plain DOM mirror of desktop
      // LinkConfirmPopover, Plan 95-03). textContent only — never innerHTML.
      // Edge-clipping mitigation mirrors Pitfall #4. Escape dismisses.
      //
      // CR-01 fix: track the cleanup closure at module scope and invoke it
      // on re-entry so rapid successive risky clicks do NOT stack click /
      // keydown listeners (which previously caused a single Continue press
      // to open EVERY queued URL — including ones the user had not yet
      // visually confirmed). Mirrors the existing findBarExitTimer /
      // searchDebounceTimer cancel-on-re-entry idiom in this file.
      var linkConfirmCleanup = null;
      function showLinkConfirmPopover(url, risk, x, y) {
        // Idempotent dismiss of any prior popover invocation — drops stacked
        // click + keydown handlers before binding fresh ones (CR-01).
        if (linkConfirmCleanup) {
          try { linkConfirmCleanup(); } catch (e) {}
        }
        var pop = document.getElementById('link-confirm-popover');
        if (!pop) return;
        var reasonEl = document.getElementById('link-confirm-reason');
        var urlEl = document.getElementById('link-confirm-url');
        var continueBtn = document.getElementById('link-confirm-continue');
        var cancelBtn = document.getElementById('link-confirm-cancel');
        if (!reasonEl || !urlEl || !continueBtn || !cancelBtn) return;
        reasonEl.textContent =
          risk === 'osc8' ? 'This link displays one address but points to another. Verify the destination before continuing.'
          : risk === 'idn' ? 'This link contains internationalized characters that can spoof familiar domains.'
          : risk === 'typosquat' ? 'This domain matches a known impersonation pattern. Verify the spelling carefully.'
          : '';
        urlEl.textContent = url; // textContent — never innerHTML (T-95-06-05)
        pop.style.left = x + 'px';
        pop.style.top = y + 'px';
        pop.hidden = false;
        // Edge-clipping mitigation (mirrors desktop Pitfall #4).
        var rect = pop.getBoundingClientRect();
        if (x + rect.width + 8 > window.innerWidth) pop.style.left = Math.max(8, x - rect.width) + 'px';
        if (y + rect.height + 8 > window.innerHeight) pop.style.top = Math.max(8, y - rect.height) + 'px';
        try { cancelBtn.focus(); } catch (e) {}

        function handleContinue() { cleanup(); openLink(url); }
        function handleCancel() { cleanup(); }
        function handleKey(e) { if (e.key === 'Escape') handleCancel(); }
        function cleanup() {
          pop.hidden = true;
          continueBtn.removeEventListener('click', handleContinue);
          cancelBtn.removeEventListener('click', handleCancel);
          document.removeEventListener('keydown', handleKey);
          linkConfirmCleanup = null;
        }
        continueBtn.addEventListener('click', handleContinue);
        cancelBtn.addEventListener('click', handleCancel);
        document.addEventListener('keydown', handleKey);
        linkConfirmCleanup = cleanup;
      }

      // Phase 95 hot-swap-capable web-links handle + sub-config sink. Read at
      // click time so toggling modifier/confirm* sub-keys does NOT re-attach
      // the addon (mirror of desktop webLinksConfigRef pattern, Pitfall #8).
      var webLinksAddonHandle = null;
      var currentWebLinksConfig = null;
      // ────────────────────────────────────────────────────────────────────────

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

        // Phase 94 SRC-05: SearchAddon hot-swap (mirrors WebGL/Clipboard arms above).
        // UMD global name verified via:
        //   grep -E "root\\[['\"](SearchAddon|.*Addon)['\"]\\]" web/vendor/xterm/addons/addon-search.js
        // Result: e.SearchAddon=t() (matches Phase 93 precedent for WebglAddon /
        // Unicode11Addon / ClipboardAddon). Constructor: new SearchAddon.SearchAddon()
        // — RESEARCH Pitfall #7 mitigation. Plan 94-04 SUMMARY confirmed
        // window.SearchAddon is a NAMESPACE OBJECT { SearchAddon: <ctor> } at runtime.
        if (!newConfig.search && searchAddonHandle) {
          if (searchResultsDisposable) {
            try { searchResultsDisposable.dispose(); } catch (e) {}
            searchResultsDisposable = null;
          }
          try { searchAddonHandle.dispose(); } catch (e) {}
          searchAddonHandle = null;
          window.searchAddonHandle = null;
          if (findBarOpen) hideFindBar();
        } else if (newConfig.search && !searchAddonHandle) {
          try {
            searchAddonHandle = new SearchAddon.SearchAddon(); // Pitfall #7
            term.loadAddon(searchAddonHandle);
            searchResultsDisposable = searchAddonHandle.onDidChangeResults(function(e) {
              matchInfo = { index: e.resultIndex, count: e.resultCount };
              updateMatchCountUI();
            });
            window.searchAddonHandle = searchAddonHandle;
          } catch (e) {
            // SearchAddon UMD missing or construction failed — silent. Find bar
            // will still appear if Cmd-F is pressed but findNext will be a no-op
            // (null-guards at the call sites handle this). See Pitfall #7.
            searchAddonHandle = null;
            window.searchAddonHandle = null;
          }
        }

        // Phase 94 SRC-05 / SSE sync: if the canonical SearchConfig changed via
        // SSE settings:plugins push (e.g. desktop user toggled regex elsewhere),
        // re-sync the local searchOptions and update toggle UI. Per UI-SPEC line
        // 335 web is in-memory only; the daemon push is the only one-way write.
        if (newConfig.searchConfig) {
          var canonical = newConfig.searchConfig;
          var differs =
            canonical.regex !== searchOptions.regex ||
            canonical.caseSensitive !== searchOptions.caseSensitive ||
            canonical.wholeWord !== searchOptions.wholeWord;
          if (differs) {
            searchOptions = {
              regex: !!canonical.regex,
              caseSensitive: !!canonical.caseSensitive,
              wholeWord: !!canonical.wholeWord
            };
            syncToggleUI();
          }
        }

        // Phase 95 — web-links arm (LNK-01..06). Mirrors the desktop hot-swap
        // pattern from TerminalPanel.tsx Plan 95-04: addon load on toggle-on,
        // dispose on toggle-off; sub-config (modifier/confirm*) flows via
        // currentWebLinksConfig and is read at click time so toggling those
        // sub-keys does NOT re-attach the addon (Pitfall #8).
        currentWebLinksConfig = newConfig.webLinksConfig || null;
        if (!newConfig.webLinks && webLinksAddonHandle) {
          try { webLinksAddonHandle.dispose(); } catch (e) {}
          webLinksAddonHandle = null;
          // Toggle-off also clears any in-flight popover so a user disabling
          // the feature mid-confirmation doesn't get stuck looking at a dialog
          // whose Continue path no longer makes sense.
          //
          // WR-06 fix: invoke the popover's cleanup closure (if any) so the
          // document-level keydown listener is removed too — otherwise a
          // mid-popover toggle-off leaves a dangling Esc handler attached
          // to document that would silently invoke a closure on the now-
          // hidden popover. Composes with CR-01's linkConfirmCleanup.
          if (linkConfirmCleanup) {
            try { linkConfirmCleanup(); } catch (e) {}
          }
          var popOff = document.getElementById('link-confirm-popover');
          if (popOff) popOff.hidden = true;
        } else if (newConfig.webLinks && !webLinksAddonHandle) {
          try {
            var handler = function(event, uri) {
              if (!isAllowedScheme(uri)) return;                              // LNK-01
              var cfg = currentWebLinksConfig || {};
              var modifier = cfg.modifier || 'platform';
              if (!isModifierPressed(event, modifier)) return;                 // LNK-02
              var risk = getRisk(uri, uri);                                    // LNK-03 (Plan B: osc8 dormant for plain-text URLs)
              var shouldConfirm =
                (risk === 'osc8' && (cfg.confirmOSC8 !== false)) ||
                (risk === 'idn' && (cfg.confirmIDN !== false)) ||
                (risk === 'typosquat' && (cfg.confirmTyposquat !== false));
              if (risk && shouldConfirm) {
                showLinkConfirmPopover(uri, risk, event.clientX, event.clientY);
                return;
              }
              openLink(uri);                                                   // LNK-04
            };
            // Pitfall #7 (Phase 93): WebLinksAddon UMD global is a NAMESPACE
            // OBJECT { WebLinksAddon: <ctor> }. Constructor is therefore
            // new WebLinksAddon.WebLinksAddon(...), NOT new WebLinksAddon(...).
            // Verified via grep on web/vendor/xterm/addons/addon-web-links.js
            // for `e.WebLinksAddon=t()` (UMD root assignment).
            webLinksAddonHandle = new WebLinksAddon.WebLinksAddon(handler, {
              hover: function(event, uri) {
                if (event.target && event.target.setAttribute) event.target.setAttribute('title', uri);
              },
              leave: function(event) {
                if (event.target && event.target.removeAttribute) event.target.removeAttribute('title');
              }
            });
            term.loadAddon(webLinksAddonHandle);
          } catch (e) {
            // WebLinksAddon construction failed — silent; the user enabled it
            // but the env refused. Hover/click do nothing for URLs.
            console.warn('[web-links] addon load failed:', e);
            webLinksAddonHandle = null;
          }
        }

        lastApplied = JSON.stringify(newConfig);
      }

      // ── Phase 94 SRC-05 — Find bar wiring ─────────────────────────────────
      // Mirrors desktop FindBar.tsx + TerminalPanel handlers (Plan 94-03).
      // Plain DOM only; no React. UI-SPEC §"Interaction Contract"; RESEARCH
      // §"Pattern 4" (debounce); Pitfall #1 (focus gate), #3 (Esc on container),
      // #7 (UMD global name), #10 (close clears debounce + decorations).
      // All function declarations — hoisted so applyPluginConfig (defined above)
      // can reference syncToggleUI / hideFindBar / updateMatchCountUI.
      function findBarEl()      { return document.getElementById('find-bar'); }
      function findBarInputEl() { return document.getElementById('find-bar-input'); }
      function findBarCountEl() { return document.getElementById('find-bar-count'); }

      // Phase 94 SRC-02 + SRC-04 reconciliation. SearchAddon._fireResults gates
      // the onDidChangeResults event on !!opts.decorations — without this, the
      // match-count callback never fires (SRC-02 broken). Passing decorations: {}
      // (an empty object) is truthy, so the event fires; with no per-theme
      // color overrides set, the registered decoration overlays are invisible
      // and the active match is highlighted via xterm core's selection
      // (theme.selectionBackground), preserving the 138-theme invariant
      // (SRC-04). The forbidden color keys are listed and source-inspected by
      // TestTerminalJS_SearchAddon and FindBar.themeMatrix.test.tsx.
      // Discovered during Plan 94-05 chromedp e2e — see SUMMARY.
      function searchOpts() {
        return {
          regex: searchOptions.regex,
          caseSensitive: searchOptions.caseSensitive,
          wholeWord: searchOptions.wholeWord,
          decorations: {}
        };
      }

      function syncToggleUI() {
        var pairs = [
          ['find-bar-toggle-case',  'caseSensitive'],
          ['find-bar-toggle-regex', 'regex'],
          ['find-bar-toggle-word',  'wholeWord']
        ];
        for (var i = 0; i < pairs.length; i++) {
          var btn = document.getElementById(pairs[i][0]);
          if (!btn) continue;
          var on = !!searchOptions[pairs[i][1]];
          btn.setAttribute('aria-pressed', on ? 'true' : 'false');
          if (on) btn.classList.add('find-bar__toggle--active');
          else    btn.classList.remove('find-bar__toggle--active');
        }
      }

      function updateMatchCountUI() {
        var el = findBarCountEl();
        if (!el) return;
        var input = findBarInputEl();
        var query = input ? input.value : '';
        if (query === '') {
          el.classList.add('find-bar__count--hidden');
          el.classList.remove('find-bar__count--no-results');
          el.textContent = '';
          return;
        }
        el.classList.remove('find-bar__count--hidden');
        if (matchInfo.count === 0) {
          el.classList.add('find-bar__count--no-results');
          el.textContent = '0 of 0';
        } else {
          el.classList.remove('find-bar__count--no-results');
          el.textContent = (matchInfo.index + 1) + ' of ' + matchInfo.count;
        }
      }

      function runSearch(query) {
        if (searchDebounceTimer !== null) {
          clearTimeout(searchDebounceTimer);
          searchDebounceTimer = null;
        }
        searchDebounceTimer = setTimeout(function() {
          searchDebounceTimer = null;
          if (!searchAddonHandle) return;
          if (query === '') {
            try { searchAddonHandle.clearDecorations(); } catch (e) {}
            matchInfo = { index: -1, count: 0 };
            updateMatchCountUI();
            return;
          }
          // SRC-04 invariant honored via searchOpts(): no per-theme color
          // overrides, so the active match is highlighted via xterm core's
          // selection (theme.selectionBackground) across all 138 themes.
          try { searchAddonHandle.findNext(query, searchOpts()); } catch (e) {}
        }, 100); // T-94-04 mitigation: 100ms debounce per RESEARCH Pitfall #5
      }

      function showFindBar() {
        findBarOpen = true;
        syncToggleUI();
        var el = findBarEl();
        if (el) {
          // Phase 94 WR-01 / SC-4 — apply the entering modifier BEFORE removing
          // [hidden] so the browser sees the class flip and runs the 200ms
          // transform+opacity transition (terminal.css §"Phase 94 — Find bar").
          el.classList.add('find-bar--entering');
          // Cancel any pending exit-unmount from a rapid re-open (no zombie state).
          if (findBarExitTimer !== null) {
            clearTimeout(findBarExitTimer);
            findBarExitTimer = null;
          }
          el.classList.remove('find-bar--exiting');
          el.hidden = false;
          // Drop --entering on the next animation frame so the transition fires.
          requestAnimationFrame(function() {
            el.classList.remove('find-bar--entering');
          });
        }
        var input = findBarInputEl();
        if (input) { input.value = ''; input.focus(); }
        matchInfo = { index: -1, count: 0 };
        updateMatchCountUI();
      }

      function hideFindBar() {
        // Pitfall #10: clear pending debounce BEFORE clearing decorations.
        if (searchDebounceTimer !== null) {
          clearTimeout(searchDebounceTimer);
          searchDebounceTimer = null;
        }
        if (searchAddonHandle) {
          try { searchAddonHandle.clearDecorations(); } catch (e) {}
        }
        matchInfo = { index: -1, count: 0 };
        var input = findBarInputEl();
        if (input) input.value = '';
        updateMatchCountUI();
        var el = findBarEl();
        if (el) {
          // Phase 94 WR-01 / SC-4 — play the exit transition before el.hidden=true.
          // Symmetric with desktop TerminalPanel.handleSearchClose (200ms unmount
          // delay matching the longer transform-200ms / opacity-150ms transition —
          // UI-SPEC §"Animation" line 200).
          el.classList.remove('find-bar--entering');
          el.classList.add('find-bar--exiting');
          if (findBarExitTimer !== null) {
            clearTimeout(findBarExitTimer);
          }
          findBarExitTimer = setTimeout(function() {
            findBarExitTimer = null;
            el.hidden = true;
            el.classList.remove('find-bar--exiting');
          }, 200);
        }
        findBarOpen = false;
        // Restore terminal focus per UI-SPEC §"Closing the Find Bar".
        try { term.focus(); } catch (e) {
          var t = document.querySelector('#terminal .xterm-helper-textarea');
          if (t) t.focus();
        }
      }

      function wireFindBarHandlers() {
        var el = findBarEl(); if (!el) return;
        var input = findBarInputEl();
        var prevBtn = document.getElementById('find-bar-prev');
        var nextBtn = document.getElementById('find-bar-next');
        var closeBtn = document.getElementById('find-bar-close');

        // Pitfall #3 — Esc handler at CONTAINER level so it works regardless
        // of which child has focus (input, toggle, or nav button).
        el.addEventListener('keydown', function(e) {
          if (e.key === 'Escape') { e.preventDefault(); hideFindBar(); }
        });

        if (input) {
          input.addEventListener('input', function(e) {
            runSearch(e.target.value);
          });
          input.addEventListener('keydown', function(e) {
            var isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
            var modifier = isMac ? e.metaKey : e.ctrlKey;
            if (e.key === 'Enter') {
              e.preventDefault();
              if (!searchAddonHandle) return;
              if (e.shiftKey) { try { searchAddonHandle.findPrevious(input.value, searchOpts()); } catch (err) {} }
              else            { try { searchAddonHandle.findNext(input.value, searchOpts()); } catch (err) {} }
            } else if (modifier && e.key.toLowerCase() === 'g') {
              e.preventDefault();
              if (!searchAddonHandle) return;
              if (e.shiftKey) { try { searchAddonHandle.findPrevious(input.value, searchOpts()); } catch (err) {} }
              else            { try { searchAddonHandle.findNext(input.value, searchOpts()); } catch (err) {} }
            }
          });
        }

        if (prevBtn) prevBtn.addEventListener('click', function() {
          if (!searchAddonHandle || !input) return;
          try { searchAddonHandle.findPrevious(input.value, searchOpts()); } catch (e) {}
        });
        if (nextBtn) nextBtn.addEventListener('click', function() {
          if (!searchAddonHandle || !input) return;
          try { searchAddonHandle.findNext(input.value, searchOpts()); } catch (e) {}
        });
        if (closeBtn) closeBtn.addEventListener('click', function() { hideFindBar(); });

        // Toggle handlers — flip option, sync UI, re-run search instantly. Web
        // persistence is desktop-only per UI-SPEC line 335 (in-memory for the
        // web session; canonical state arrives via /api/plugin-config + SSE).
        var toggleSpecs = [
          ['find-bar-toggle-case',  'caseSensitive'],
          ['find-bar-toggle-regex', 'regex'],
          ['find-bar-toggle-word',  'wholeWord']
        ];
        for (var i = 0; i < toggleSpecs.length; i++) {
          (function(id, key) {
            var btn = document.getElementById(id);
            if (!btn) return;
            btn.addEventListener('click', function() {
              searchOptions[key] = !searchOptions[key];
              syncToggleUI();
              if (searchAddonHandle && input && input.value) {
                try { searchAddonHandle.findNext(input.value, searchOpts()); } catch (e) {}
              }
            });
          })(toggleSpecs[i][0], toggleSpecs[i][1]);
        }
      }

      // Phase 94 SRC-01 / SRC-05 — focus-conditioned Cmd-F (mac) / Ctrl-F (lin/win)
      // listener. Pitfall #1: gate by termEl.contains(document.activeElement) so
      // non-terminal page text (status bar, banner) falls through to browser-native
      // find. preventDefault() is ONLY called when the focus gate passes.
      window.addEventListener('keydown', function(e) {
        if (!pluginConfig.search) return;
        var isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
        var modifier = isMac ? e.metaKey : e.ctrlKey;
        if (!modifier || e.key.toLowerCase() !== 'f') return;
        var termEl = document.getElementById('terminal');
        if (!termEl || !termEl.contains(document.activeElement)) return;
        e.preventDefault();
        if (findBarOpen) {
          var input = findBarInputEl();
          if (input) input.focus();
        } else {
          showFindBar();
        }
      });
      // ────────────────────────────────────────────────────────────────────────

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
        serialize: false, progress: false,
        searchConfig: { regex: false, caseSensitive: false, wholeWord: false }
      };
      applyPluginConfig(initialConfig);

      // Phase 94 SRC-05 — initialize searchOptions from canonical SearchConfig
      // delivered by /api/plugin-config. The hot-swap arm in applyPluginConfig
      // also handles this on subsequent SSE pushes (see SSE sync block).
      if (initialConfig.searchConfig) {
        searchOptions = {
          regex:         !!initialConfig.searchConfig.regex,
          caseSensitive: !!initialConfig.searchConfig.caseSensitive,
          wholeWord:     !!initialConfig.searchConfig.wholeWord
        };
        syncToggleUI();
      }
      wireFindBarHandlers();

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
