# Pitfalls Research — v3.2 xterm.js Plugin Suite (Issue #36)

**Domain:** Bundling six `@xterm/addon-*` packages (webgl, search, image, web-links, unicode11, serialize) into AgentHub's existing Wails GUI + multi-client web-served xterm.js, with vendored-only assets, strict CSP, and Tailscale + local-network browsing audiences.
**Researched:** 2026-05-03
**Confidence:** HIGH for addon-specific behavior (verified against upstream addon repos and known issues); HIGH for AgentHub-specific architecture (read directly from `.planning/PROJECT.md` and v1.6/v2.0/v3.1 milestones); MEDIUM for Safari/Tailscale-iOS WebGL claims (single-source verification only).

> **Scope discipline:** Every pitfall below is something AgentHub-specific that the team can actually trip over in v3.2 — not generic xterm.js advice. The CLI raw-PTY attach (`agenthub attach`) and Bubble Tea TUI do **not** use xterm.js; the addons apply only to the Wails GUI tab renderer and the Tailscale-/local-network-served `web/vendor/xterm/` frontend. Wherever a pitfall would fan out to those surfaces, it's called out explicitly.

---

## Critical Pitfalls

### Pitfall 1: WebGL Context Loss Drops Scrollback Permanently

**What goes wrong:**
On GPU driver crashes, OS sleep/wake, multi-monitor relocation, browser tab backgrounding under memory pressure, or on long-running Tailscale-served sessions on iPad/Mac Safari, the WebGL context is lost. The xterm.js README is explicit: *"the browser may drop WebGL contexts for various reasons like OOM or after the system has been suspended."* If we follow the "easy" path the README hints at (dispose `WebglAddon` on `onContextLoss`), we lose the renderer — but the **buffer** survives because xterm.js's buffer is renderer-agnostic. The trap is doing this *after* swapping to a new `Terminal` instance, or disposing/re-creating the terminal — at which point scrollback is gone.

**Why it happens:**
Developers conflate "renderer crashed" with "terminal crashed." The xterm.js addon-webgl `onContextLoss(cb)` API is intentionally minimal — it tells you the GPU is gone, but the recovery is up to the embedder. The naive recovery is "create a new terminal," which obliterates the buffer. The correct recovery is "dispose the WebGL addon, optionally load `@xterm/addon-canvas`, leave the `Terminal` instance untouched."

**How to avoid:**
- Implement `onContextLoss` to: (a) dispose only the `WebglAddon`, (b) load `@xterm/addon-canvas` as a fallback (vendor it alongside webgl in `web/vendor/xterm/`), (c) emit a one-time toast/banner ("GPU acceleration unavailable, switched to canvas"), (d) re-attempt webgl after a configurable cooldown only on user action — never automatically retry, because a flapping driver will ping-pong.
- Vendor `@xterm/addon-canvas` even though it's not in the v3.2 feature list — it is the documented fallback path and adding it later means re-pulling vendored assets.
- Never re-create the `Terminal` on context loss. The buffer is the source of truth for scrollback fan-out and any rebuild loses content.

**Warning signs:**
- User reports "my terminal went blank after my laptop woke up"
- Console: `WebGL: CONTEXT_LOST_WEBGL: loseContext: context lost`
- Multi-monitor users seeing a black canvas after dragging the window between displays
- iPad Safari users on Tailscale reporting blank terminal after backgrounding for 30+ seconds

**Phase to address:**
WebGL renderer phase — must ship with canvas fallback addon vendored and `onContextLoss` handler. Do not ship WebGL addon without the fallback wired.

---

### Pitfall 2: Initial-Paint Timing Regression (v1.6 Phase 35 Re-Break)

**What goes wrong:**
v1.6 Phase 35 fixed initial terminal fill via a bounded rAF retry loop polling `FitAddon.proposeDimensions()` until cell dimensions are non-zero (~333ms cap, 20 attempts). The retry loop was sized for the *DOM renderer's* glyph-measurement path. The WebGL renderer measures glyphs differently — it builds a texture atlas on first activation, and `proposeDimensions()` may report dimensions before the atlas is ready, or report dimensions that are subtly off because the WebGL canvas uses an integer pixel grid that the DOM-renderer baseline doesn't. Result: terminals fill but with a 1-cell off-by-one, OR the retry loop fires too early when WebGL is enabled and cell dims come back as zero on a different code path.

**Why it happens:**
The Phase 35 fix was empirically tuned against the DOM renderer for Claude/Codex/Gemini/OpenCode. WebGL has a separate measurement pipeline (`CharSizeService` integrates with the texture atlas). Whoever toggles WebGL in Settings doesn't know about the rAF tuning, and the rAF loop doesn't know which renderer is active.

**How to avoid:**
- Order of operations: load `WebglAddon` **before** the first `fit()` call, not after. The texture atlas needs to be initialized so `CharSizeService` reports stable dimensions.
- Add a regression test: open all four CLIs (Claude, Codex, Gemini, OpenCode) with WebGL enabled and assert no off-by-one in `cols × rows` after first paint. The existing fit tests use the DOM renderer — they will not catch this.
- If WebGL is the default, re-tune the rAF retry cap if needed; do not silently increase it. Document the new cap and why.
- Settings toggle for WebGL: make it apply to **new sessions only** (see Pitfall 13). Hot-swapping renderers mid-session is a known xterm.js footgun.

**Warning signs:**
- One column or row missing on first paint that wasn't missing before
- "Terminal looks slightly cropped at startup" reports
- Differential bug only on Wails (because WebGL is enabled) but not on web-served (where it might be canvas-fallback'd)

**Phase to address:**
WebGL phase — initial-paint regression test must run against all four CLIs before the toggle ships.

---

### Pitfall 3: Sixel Storage Bomb Crashes the Tab

**What goes wrong:**
`@xterm/addon-image` defaults `storageLimit` to **100 MB** of decoded RGBA pixel data per terminal instance (4-channel unpacked). A user runs `cat huge.six`, `imgcat *.png` over a watch loop, or hits a misbehaving CLI that emits sixel as a debug spam. The browser tab hits ~600 MB resident, then OOMs — taking down the whole Wails WebView with it. Worse on web-served Safari/iOS where the OS is far more aggressive about killing memory-hungry tabs.

**Why it happens:**
100 MB sounds small but it's *decoded* image data. A 1920x1080 RGBA frame is ~8 MB; ten of them and you're at 80 MB. Sixel from `imgcat` is uncompressed pixel data; the addon's FIFO eviction kicks in only at the limit, which means brief spikes can already exhaust the tab.

**How to avoid:**
- Override the default: set `storageLimit: 16` (16 MB) for AgentHub. CLI agents rarely emit images; we can be aggressive.
- Expose `storageLimit` in the per-plugin config panel (advanced section, MB units).
- Across multiple sessions/tabs, the limit is **per-terminal**, not global. With 8 sessions open, total budget is 8× the per-instance limit. Document this.
- Consider exposing a "drop oldest sixel" / "purge images" Settings action for users who hit the wall.
- On web-served clients, drop the limit even further (8 MB) — Safari is stricter than Chromium and we don't control the user's device memory.

**Warning signs:**
- Browser tab memory growth in DevTools profiler trending up while images render
- "AgentHub froze when I cat'd a sixel file"
- Wails WebView crash logs mentioning OOM
- iPad Safari Tailscale browsing reports of "terminal disappeared after image"

**Phase to address:**
Image addon phase — ship with conservative `storageLimit` default and config exposure. Test with a known-large sixel file (≥50 MB encoded) before merge.

---

### Pitfall 4: Sixel Decoder Adds 200KB+ to First-Paint Bundle on Tailscale

**What goes wrong:**
`@xterm/addon-image` includes a sixel decoder, IIP parser, and image worker. The published bundle is roughly 80–150 KB minified (verify in vendoring step) — but the worker file is loaded from a separate URL, and CSP `worker-src` is not currently set in v3.1's policy. A user opens a session on iPad Safari over Tailscale on a coffee-shop hotspot with 500 KB/s and 200ms RTT — the addon now adds palpable lag to first paint, *and* the worker fails to load if CSP isn't updated.

**Why it happens:**
Vendoring discipline says "bundle the addon," but bundlers don't always inline workers. CSP from v3.1 Phase 89 (D-09) is `script-src 'self'; connect-src 'self' wss://<host>; style-src 'self' 'unsafe-inline'` — no `worker-src`. Browsers default `worker-src` to `child-src`, then `default-src` which is implicitly `'self'` in our policy if we set `default-src 'self'`. But if we forgot `default-src`, workers from `blob:` URLs (which bundlers often emit) are rejected.

**How to avoid:**
- Audit `@xterm/addon-image`'s build output during vendoring. If it ships a separate worker file, vendor it alongside and update CSP to `worker-src 'self'`. If it uses `blob:` workers, add `worker-src 'self' blob:` (preferred to keep blob workers contained).
- Make image addon **lazy-loaded**: only fetch the chunk when the user enables it in Settings. Default it to **off** initially. This sidesteps the first-paint cost for the common case (most users don't need sixel).
- Measure cold-cache first-paint over a throttled network (3G profile in DevTools) before and after enabling. Set a hard budget (e.g., +50ms p95).
- Re-run the v3.1 CSP zero-violation Chromium e2e suite after addon integration. Add Safari and Firefox to that suite for v3.2 since iPad/Mac users will hit this.

**Warning signs:**
- v3.1 CSP e2e suite flips from green to red after addon integration
- DevTools Network tab shows a separate `image.worker.js` 404 or CSP block
- "AgentHub web is slower" on Tailscale iPad after v3.2 ships
- Console: `Refused to create a worker from 'blob:...' because it violates the following Content Security Policy directive`

**Phase to address:**
Image addon phase + CSP audit phase. The CSP review must happen for **every** addon shipped, not just image.

---

### Pitfall 5: Image Replay Skipped for Late-Joining Web Clients (Multi-Client Fan-Out Hole)

**What goes wrong:**
v2.0's relay Hub does scrollback replay via a binary framing protocol (MsgOutput frames). When client B joins a session that already had client A render a sixel, client B sees only the text but **not the image** — sixel data is part of the byte stream, but if the relay's scrollback buffer is line-oriented or lossy on long binary sequences, the sixel escape may be truncated, garbled, or dropped at the byte boundary. Worse: if AgentHub stripped escape sequences from scrollback for any reason (it doesn't, but historically many terminal apps do), images vanish entirely.

This is also relevant for the `serialize` addon used in any future "session restore" feature: if we serialize on the GUI side and the web client is mid-session-restore, images may not survive the round-trip — see [jerch/xterm-addon-image#47](https://github.com/jerch/xterm-addon-image/issues/47): *"images aren't serialized."* This is an upstream limitation, not a bug we can fix.

**Why it happens:**
Sixel is multi-kilobyte and crosses many writes from the PTY. The relay's framing must preserve the byte stream **exactly**, including all DCS escapes. Any logic that splits on newlines, strips control characters, or buffers per-line will corrupt sixel.

**How to avoid:**
- Audit `internal/relay/` for any byte-level filtering or line-based buffering on the scrollback path. Confirm scrollback replay is byte-for-byte identical to the live stream.
- Add a regression test: client A renders a sixel, client B joins, client B's screen is byte-for-byte identical to client A's at the join point.
- For serialize addon: document that "image state is not preserved" in the Settings UI tooltip. Do **not** advertise serialize as a "full restore" — it's a visual-text restore only.
- Consider whether to scope serialize to "off by default" until the upstream image serialization issue is resolved. AgentHub's primary use case is AI coding CLIs which are largely text — image serialization is not load-bearing.

**Warning signs:**
- Multi-client UAT: client B sees garbled output where client A saw an image
- Issue reports: "I shared my session with my coworker and the chart didn't show up for them"
- No console error — silent corruption is the failure mode

**Phase to address:**
Image addon phase must include a multi-client replay regression test. Serialize phase must include explicit Settings-UI documentation of the image-loss limitation.

---

### Pitfall 6: Search Addon Locks the Renderer at AgentHub's Default Scrollback

**What goes wrong:**
Upstream Issue [#5176](https://github.com/xtermjs/xterm.js/issues/5176) and [#4902](https://github.com/xtermjs/xterm.js/issues/4902) document that search is non-incremental and that "search is very slow when there is a lot of content and many wrapped lines" — slowdowns reproduced at 10,000 lines. Default highlight cap is 1000 matches. AgentHub doesn't currently set a fixed scrollback cap (default xterm.js scrollback is 1000 lines, which we may or may not have raised). For long Claude/Codex sessions where the user has been working for an hour, scrollback can easily exceed 10k visual lines after wrapping. A Cmd+F + regex `.*` would block the renderer for seconds — during which time PTY output is **still streaming in** (search-while-streaming).

**Why it happens:**
Search runs synchronously on the main thread. Streaming output appends to the buffer concurrently, so the search target is moving. Highlight rendering is also main-thread.

**How to avoid:**
- Verify and document AgentHub's current `scrollback` setting (in the Terminal constructor — likely default 1000). If we want to support large scrollback for AI coding sessions, do that explicitly with a cap (say, 10000) and accept the search performance cost as a documented trade.
- Cap regex searches: if the user enters a "broad" regex (`.*`, `^`, `$`, no anchors), warn or reject. This is a UX safeguard, not a performance fix.
- Show a "Searching…" indicator while search is running, even if it blocks. Silent freezes cause force-kill.
- Clamp `decorations.matchOverviewRuler` count to keep highlight rendering bounded (the upstream 1000-cap exists for a reason — don't override unless you've measured).
- Web-served clients on iPad: regex search on Safari is markedly slower than Chrome. UAT this specifically.

**Warning signs:**
- Beach ball / "page unresponsive" on Cmd+F with regex enabled
- User feedback: "search froze the terminal for 5 seconds"
- DevTools Performance tab shows search call stack dominating frame time

**Phase to address:**
Search addon phase — measure with realistic AI coding scrollback (e.g., a 30-min Claude session captured to a fixture) before merge. Document scrollback policy.

---

### Pitfall 7: Cmd+F Conflicts with Browser Find on Web-Served Sessions

**What goes wrong:**
On the desktop Wails GUI, Cmd+F can be bound to xterm-search via a global key handler — easy. On the web-served Tailscale URL, Cmd+F triggers the **browser's** find bar, not xterm's. We override it with `event.preventDefault()` in the keydown handler — and now the user has *no* way to find anything on the page itself (e.g., the AgentHub session list, header, status bar). Worse, blind preventDefault breaks accessibility for screen-reader users who rely on browser find for navigation.

**Why it happens:**
Cmd+F is a browser-level keybinding. Capturing it inside xterm requires the terminal to have keyboard focus, which it usually does — but if the user clicks a header button and presses Cmd+F, they expect browser find. Symmetric problem on Ctrl+F (Linux/Windows browsers).

**How to avoid:**
- **Only** preventDefault when the xterm DOM node has focus (`document.activeElement` is inside `.xterm`).
- Provide a visible search affordance (a small magnifying-glass icon in the per-tab status bar, or a slash-key hotkey like vim) so users can search without ever hitting Cmd+F.
- On web-served, document explicitly: "Cmd+F searches the terminal when focused, otherwise opens browser find."
- Consider Ctrl+/ or `/` as the primary key (vim convention) and keep Cmd+F as a secondary, focus-conditioned bind.
- For accessibility: make the search input keyboard-reachable via Tab from the terminal, with proper aria labels.

**Warning signs:**
- User feedback: "browser find doesn't work in AgentHub"
- Screen-reader UAT failures (if we add it)
- Bug report: "I can't search the page header anymore"

**Phase to address:**
Search addon phase — keybinding scope must be focus-conditioned. Add UAT step: test browser-find on web-served session header.

---

### Pitfall 8: Web-Links Addon Creates an OAuth-Phish / Typosquat Vector on Web-Served

**What goes wrong:**
The terminal renders text from arbitrary processes. A malicious or compromised tool (or just a typo in a command's URL output) emits `http://github-com.evil.example/oauth/callback?token=...` or `https://gооgle.com` (Cyrillic 'о'). The user sees a styled, clickable hyperlink in their web-served terminal session and clicks it. On the web-served path, that's a tab-open in their browser with full credentials. AgentHub becomes a click-jacking surface.

The risk is *higher* on web-served than desktop because:
1. The browser has the user's full cookie jar, OAuth sessions, password manager
2. The user trusts the AgentHub URL (it's their own Tailscale FQDN with Let's Encrypt) and may not extend skepticism to the content
3. OSC 8 explicit hyperlinks let any escape sequence label a URL with arbitrary display text — making the link look like `https://github.com` while pointing to `https://evil.example`

**Why it happens:**
The default web-links handler is `window.open(uri, '_blank')`. There's no allowlist, no click-confirmation, and OSC 8's display-text/URL split is a classic phishing primitive (see the [Hyperlinks in Terminal Emulators gist](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda) — even iTerm2 and modern terminals have grappled with this).

**How to avoid:**
- **Web-served** path: show a confirmation popover on first click of any link in a session, with the **full resolved URL** (especially for OSC 8 — show the actual href, not the display text). "AgentHub wants to open https://...  [Open] [Cancel] [Don't ask again for github.com]"
- **Desktop** path: same protection by default, but lower-risk since there's no shared browser session — still, do it for consistency.
- For OSC 8 specifically: always show the URL on hover (we'd need a tooltip, not just CSS title since web-served Safari mobile won't show title). Strict: if the display text *looks like* a URL but doesn't match the OSC 8 href, mark it visually (red underline, warning icon) — this is the classic spoofing case.
- IDN check: if the hostname contains non-ASCII characters that look like ASCII (Punycode mismatch), warn loudly. Modern browsers do this; we should too.
- Trailing-punctuation parser test: `https://example.com.` (with trailing period) and `(https://example.com)` (parenthesized) should NOT include the punctuation in the linked URL. The xterm-addon-web-links default regex is decent but worth verifying against a fixture set.
- Document an opt-in: per-session "disable auto-link" setting for paranoid users running untrusted CLIs.

**Warning signs:**
- Security review flags it (this should be auto-flagged in PR review since v3.1 was security-focused)
- Issue from a security researcher
- An actual incident — too late

**Phase to address:**
Web-links phase — must ship with click-confirmation and OSC 8 URL display. This is the **most important** pitfall in this list given AgentHub's web-served audience. Treat it like v3.1 did the WS Origin allowlist.

---

### Pitfall 9: Click vs Cmd-Click vs Right-Click Platform Conventions Diverge

**What goes wrong:**
Default web-links addon binds plain click to "open." On macOS, users expect Cmd-click for "open in new tab" and right-click for context menu. On the Wails desktop, plain click should open in default browser via Wails `BrowserOpenURL` (we already use this in `RemoteSessionsPanel`) — *not* `window.open` (which doesn't make sense in a non-browser context). On web-served Linux, middle-click expectation differs.

**Why it happens:**
The default handler is `window.open` which doesn't work the same way across Wails vs. browser, vs. across platforms within the browser.

**How to avoid:**
- Custom click handler:
  - Wails desktop: route to `BrowserOpenURL` (already in use elsewhere).
  - Web-served: `window.open(url, '_blank', 'noopener,noreferrer')` — `noopener` prevents reverse-tabnabbing.
- Add Cmd-click / Ctrl-click handling explicitly even though the web-served default is "new tab" anyway — make it consistent.
- Right-click: bring up an AgentHub context menu with "Open Link", "Copy Link Address", "Open in New Tab". This is what every browser does and users expect it.
- On touch (iPad Safari Tailscale): long-press should show the URL in a tooltip before opening. Test this.

**Warning signs:**
- Wails desktop: clicking a link does nothing (because `window.open` doesn't work in WebView reliably, or opens *inside* the WebView replacing the AgentHub UI)
- iPad users: tapping a link replaces the AgentHub session with the linked page in the same Safari tab — terrible UX
- "Open in new tab" doesn't work on macOS Cmd-click

**Phase to address:**
Web-links phase — platform-aware click handlers in the same milestone where the addon is enabled.

---

### Pitfall 10: WebGL Unavailable on Tailscale-Browsing Audience We Don't Profile

**What goes wrong:**
We assume "WebGL works." On the Tailscale-served web path, our audience includes:
- iPad Safari (WebGL2 supported but historically buggy with text rendering)
- Older Linux laptops with software-rasterized WebGL (extremely slow — slower than DOM)
- Corporate environments where WebGL is GPO-disabled
- Headless/test browsers
- Browsers with GPU blacklisted (`chrome://gpu` shows hardware acceleration disabled)

We enable WebGL by default. iPad users see worse performance, not better. Software-WebGL Linux users see *much* worse performance.

**Why it happens:**
WebGL is "supported" per `WebGLRenderingContext` existing, but performance characteristics vary by 10–100x between hardware-accelerated and software paths. The xterm-addon-canvas exists explicitly because "WebGL2 isn't supported or performant for some reason" — upstream knows this is a problem.

**How to avoid:**
- Detect WebGL2 hardware acceleration: check `gl.getParameter(gl.RENDERER)` for "SwiftShader," "llvmpipe," "ANGLE (...software...)", etc. If software, fall back to canvas.
- Default WebGL toggle to **on for Wails desktop, opt-in for web-served** — until we have telemetry on actual device profiles. Or: "auto" mode that picks based on detection.
- Document the renderer in a hidden Settings diagnostic line ("Renderer: WebGL2 hardware / WebGL2 software / Canvas / DOM").
- iPad UAT: explicit test on iOS Safari over Tailscale before merging.

**Warning signs:**
- iPad/iOS users report sluggish typing latency
- Linux user reports "AgentHub is slower since the upgrade"
- `chrome://gpu` shows software rasterization

**Phase to address:**
WebGL phase — software-renderer detection must be in scope.

---

### Pitfall 11: Unicode 11 Activation Reflows Existing Scrollback Visually

**What goes wrong:**
A user has 5000 lines of scrollback in a Claude session. They go to Settings, enable Unicode 11, and the display shifts: emoji that were 1-cell wide are now 2-cell wide. xterm.js does **not** retroactively re-wrap historical scrollback when the unicode version changes — line widths are computed at write time. So old lines may now have leftover trailing space or appear visually "wrong" while new lines look correct. Users perceive this as "the upgrade broke my history."

**Why it happens:**
Buffer storage is character-cell oriented. Width is computed when the cell is written, not when it's rendered. Changing the unicode version affects `wcwidth()` for new writes only. Re-flowing 5000 lines retroactively would require a full buffer rewrite — xterm.js does not do this, by design (perf).

**How to avoid:**
- Apply Unicode 11 toggle to **new sessions only**. Show "(applies to new sessions)" indicator in the Settings UI — same pattern we'll use for WebGL.
- If the user enables it for the current session anyway (via Settings UI per-session), accept that scrollback above the current line may look misaligned; document it.
- Default Unicode 11 to **on** for new sessions in v3.2 (it's strictly better for emoji and modern CJK width). Don't surface the toggle prominently — surface it under "Advanced" since most users want it on and don't need to think about it.
- Width-table size cost (~12 KB after gzip per upstream bundle inspection) — negligible, not a bundle pitfall.

**Warning signs:**
- Bug report: "my scrollback looks weird after enabling Unicode 11"
- Mixed emoji widths in the same scrollback view
- Lines visually overlapping or under-filling

**Phase to address:**
Unicode 11 phase + Settings UI phase — "(applies to new sessions)" affordance must be designed in the Settings UI work, not retrofitted.

---

### Pitfall 12: Serialize Captures Secrets Verbatim — PII/Token Exfil Risk

**What goes wrong:**
Scrollback in AI coding sessions routinely contains:
- API keys printed by `env` or accidental `echo $ANTHROPIC_API_KEY`
- OAuth tokens from `gh auth status -t`
- Database connection strings with passwords
- Customer PII the agent was asked to process
- AWS credentials from `aws configure list` or stack-trace dumps

If we ship `serialize` and a "Save terminal state" button (or worse, automatic crash-recovery serialization), we are creating a high-fidelity dump of secrets. If that dump is written to disk under the daemon settings dir, and the user emails their `~/.config/agenthub/` to support, or it gets backed up to iCloud/Dropbox, we have a credential leak.

**Why it happens:**
`serialize` is doing exactly what its name says — it dumps the buffer verbatim, including ANSI sequences, into a string. The trap is the *use case* we wire it to.

**How to avoid:**
- v3.2 scope: ship the addon as a **library capability**, not as a user-visible "Save my terminal" feature. No automatic serialization. No on-disk persistence.
- If future scope adds session-restore, gate it behind:
  - Explicit per-session opt-in ("Allow this session to be restored")
  - Encrypted-at-rest storage (use the OS keychain, not plaintext JSON)
  - Auto-expiry (e.g., 1 hour)
  - Redaction pass — but acknowledge that regex-based secret redaction has 50%+ false-negative rates against modern token formats. Don't claim it's "safe."
- Document explicitly in the Settings UI: "Serialize captures all visible terminal text including any secrets, tokens, or sensitive data printed in this session."
- Default the toggle to **off**.

**Warning signs:**
- Anyone asking us to add "auto-save scrollback" or "restore my session"
- Support tickets with attached `~/.config/agenthub/` dumps containing tokens
- A security researcher noticing serialize is loaded by default

**Phase to address:**
Serialize phase — scope hygiene more important than implementation. Define what serialize is *for* in v3.2 (probably: nothing user-facing; available as a hook for future scope).

---

### Pitfall 13: "Applies to New Sessions Only" Miscommunication Causes Bug Reports

**What goes wrong:**
Several addons cannot be hot-swapped on a live xterm instance without re-creating the terminal (and losing scrollback): WebGL renderer, Unicode 11 width table, scrollback size. User goes to Settings, toggles "Use WebGL renderer," nothing visibly changes in their existing tab, files a bug "WebGL toggle is broken." They never realize it took effect on their *next* session.

**Why it happens:**
Other AgentHub Settings (theme, font size) **do** apply live. Users build a mental model of "Settings = live." Plugin toggles violate it silently.

**How to avoid:**
- Categorize each plugin toggle in Settings UI:
  - **Live**: applies immediately (search, web-links, image, serialize-as-library)
  - **Next session only**: webgl, unicode11, scrollback-size
- Show an inline indicator next to "next session only" toggles: a small "(new sessions)" caption *and* a gray-styled "Will apply to new sessions" badge after toggling.
- Optional: "Apply now (will reload terminal)" button — explicit, requires confirmation, says exactly what happens.
- Per-tab config? Out of scope — one global plugin config; per-tab adds combinatorial complexity (already noted in PROJECT.md "out of scope: Per-tab theme overrides — global theme sufficient; per-tab adds UI complexity").

**Warning signs:**
- Issue stream: "WebGL toggle does nothing"
- Confused users on Discord/Slack: "I enabled X, it's not working"
- Multiple users hitting the same misunderstanding

**Phase to address:**
Settings UI phase — design the affordance up front, not after the bug reports come in.

---

### Pitfall 14: Settings.json Migration Breaks Existing Users

**What goes wrong:**
v3.1 `settings.json` has no plugin keys. v3.2 introduces (say) `plugins.webgl.enabled: true`, `plugins.image.storageLimit: 16`, etc. We naively read the file with `json.Unmarshal` into a struct; missing fields default to Go's zero value, which is `false` / `0`. So a returning user lands with **all addons disabled and storage limit at 0** — terminal looks worse than before.

**Why it happens:**
JSON deserialization defaults vs. "what we want when the key is absent" mismatch. The codebase already uses `daemonSettings` struct + `saveSettingsToDisk()` (per Key Decisions); this pattern works fine when adding fields *if* defaults are set explicitly.

**How to avoid:**
- Add a `defaultSettings()` constructor that returns the desired defaults. Load: read file, then **merge over defaults** (i.e., parse on top of an already-populated default struct, or post-process zero values with explicit defaults).
- Write a one-time migration: on first v3.2 start with an existing `settings.json`, populate plugin defaults and save. Idempotent.
- Test: copy a v3.1 `settings.json` fixture into a temp config dir, boot daemon, assert plugin defaults are populated correctly.
- Schema version field: add `"schemaVersion": 2` to settings.json. Future-proofs migration.
- **Never** use `omitempty` on plugin keys — we want them written explicitly so users can see/edit them.

**Warning signs:**
- v3.2 launch: existing users report "terminal looks worse / colors weird / no images"
- New install works, upgrade install doesn't
- Bug reports specifically about "after upgrading from v3.1"

**Phase to address:**
Settings persistence phase — migration test is non-negotiable. Add `tests/fixtures/settings_v3.1.json` that the test loads and asserts upgrade behavior.

---

### Pitfall 15: Multi-Client Plugin State Drift (A Enables, B Doesn't)

**What goes wrong:**
v2.0 fan-out: client A and client B both attach to session `claude-1`. Client A is on a Mac with WebGL on. Client B is on iPad with WebGL forced off (software renderer detected). What happens with image addon? With Unicode 11? If plugin state is a *server* property, B can't opt out. If it's a *client* property, A and B see the same data rendered differently — which is **fine** for renderer differences (DOM vs WebGL vs canvas) but **breaks** if the addon's behavior changes the byte stream interpretation (Unicode 11 width affects line wrap).

The genuinely tricky case: if A enables image addon and B doesn't, and the PTY emits sixel — A sees an image, B sees garbled escape codes scrolling by. This is correct behavior (B's terminal can't render sixel), but users will report it as a bug.

**Why it happens:**
PROJECT.md explicitly says "Synchronized scrollback across clients — violates universal terminal-sharing contract; independent scrollback is expected." Independent client state is the design. But "rendering capability" is a new axis users haven't reasoned about.

**How to avoid:**
- **Per-client local override** for renderer choice (WebGL / canvas / DOM) and image-addon. Each client's web frontend reads their localStorage and applies independently. This is the existing pattern (theme, font-size).
- **Server-broadcast** for choices that affect line-wrap or buffer interpretation (Unicode 11). All clients *must* see the same width tables, otherwise scrollback diverges. Push this from the daemon to all attached clients.
- Settings UI: distinguish "rendering preference (local)" from "session preference (shared)". Use different visual treatment.
- Document: "If your collaborator's terminal looks different from yours, check their Plugins settings."

**Warning signs:**
- Multi-client UAT: pair-programming session, one sees images and the other doesn't
- "My session looks different on my laptop vs my iPad"

**Phase to address:**
Multi-client integration phase (or whichever phase touches the relay). Decision must be made before Settings UI is designed — the UI surface differs between local and shared toggles.

---

### Pitfall 16: Vendoring Drift — Addon Versions Decouple from xterm.js Core

**What goes wrong:**
v3.1 vendored `xterm.js` core. We add 6 addons. Six months later, `@xterm/addon-search` ships v0.16.0 with a fix we want; we vendor-update *just that addon*. But the new search addon's `loadAddon(terminal)` contract requires xterm.js core ≥ 5.6.0 and we have 5.5.0 vendored. Result: silent breakage — `loadAddon` succeeds but search returns no results, or throws a misleading error.

**Why it happens:**
xterm core and addons are versioned independently in the `@xterm/*` namespace post-2024 rename. Each addon's `peerDependencies` declares the core version range. If you don't enforce, you can install incompatible combinations. CDN consumers don't care because they pull whatever's latest; vendored consumers freeze in time and skew accumulates.

**How to avoid:**
- Pin a known-good combination in `package.json` and document in `web/vendor/xterm/VERSIONS.md`:
  ```
  @xterm/xterm: 5.5.0
  @xterm/addon-fit: 0.10.0
  @xterm/addon-webgl: 0.18.0
  @xterm/addon-canvas: 0.7.0
  @xterm/addon-search: 0.15.0
  @xterm/addon-image: 0.8.0
  @xterm/addon-web-links: 0.11.0
  @xterm/addon-unicode11: 0.8.0
  @xterm/addon-serialize: 0.13.0
  ```
  (Verify exact versions during the vendoring step — the above are reasonable mid-2026 estimates, not exact.)
- Vendoring script: read versions from a single manifest, fail if any addon's `peerDependencies.@xterm/xterm` doesn't satisfy the core's version.
- CI gate: re-run e2e test suite on every vendor update.
- Update strategy: bump core + all addons together as a unit, never individually. Document this in the vendoring runbook.

**Warning signs:**
- "Search doesn't work after I updated the addon"
- Console: `loadAddon is not a function` or `terminal._core is undefined`
- Subtle: addon loads but features are subtly broken (highlights wrong, decorations missing)

**Phase to address:**
Vendoring phase — version manifest + verification script. This is a v3.1-style discipline that v3.2 must continue.

---

### Pitfall 17: FitAddon Interaction with WebGL Geometry

**What goes wrong:**
FitAddon computes `cols × rows` from the container size and the per-cell pixel dimensions reported by `CharSizeService`. With the DOM renderer, char dimensions come from a hidden measurement span. With WebGL, char dimensions come from the texture atlas. The two should agree, but rounding differs (WebGL rounds to integer pixels for atlas alignment). On a Retina display where CSS pixels are subpixel, this can lead to a half-cell discrepancy — terminal reports 80 cols, but the atlas only renders 79 columns of glyphs, leaving a one-cell black strip on the right.

**Why it happens:**
DOM uses fractional pixel measurements; WebGL uses integer pixels in its rendering grid. `proposeDimensions()` doesn't always round the same way as the atlas does.

**How to avoid:**
- Test fit on a Retina display with WebGL enabled at multiple zoom levels (90%, 100%, 110%, 125%, 150%). Compare cols×rows vs. visible glyph columns.
- If a discrepancy exists, file/follow the upstream issue and apply a workaround (snap container to integer-cell width).
- Multi-monitor: dragging window between Retina and non-Retina displays triggers a `devicePixelRatio` change — confirm `IRenderService.handleDevicePixelRatioChange` runs and the atlas rebuilds cleanly. Existing FitAddon retry loop should handle this, but verify.

**Warning signs:**
- Black strip at the right edge of the terminal after enabling WebGL
- Scrollback shows wrapped lines that didn't wrap before
- Differential between DOM render and WebGL render at the same window size

**Phase to address:**
WebGL phase — Retina + multi-monitor regression test.

---

### Pitfall 18: Theme Switching With WebGL Texture Atlas (Already Solved, Don't Regress)

**What goes wrong:**
WebGL caches glyph textures per (char, fg-color, bg-color, attrs). Live theme switching changes fg/bg colors. If we don't clear the atlas, glyphs keep their old color until the cell is rewritten. **PROJECT.md Key Decisions** already lists this:
> `clearTextureAtlas + refresh for WebGL theme updates` — WebGL renderer caches glyph textures; must clear atlas when theme colors change. ✓ Good — reliable live theme switching

This is already wired. The pitfall is **regressing it**: someone refactors theme application, drops the `clearTextureAtlas` call, and live theme switching subtly breaks (works on first apply, broken on second).

**Why it happens:**
The fix is non-obvious from reading the apply code. If you don't know about the atlas, you'd remove the `clearTextureAtlas` thinking it's redundant.

**How to avoid:**
- Add a comment block above the `clearTextureAtlas` call explicitly explaining why and linking to this pitfall.
- Add a regression test: with WebGL on, apply theme A, capture a glyph color, apply theme B, capture again, assert different. Without `clearTextureAtlas`, second capture would equal first.
- Code review checklist: any change touching theme application requires verifying atlas clear is preserved.

**Warning signs:**
- Live theme switch shows old colors mixed with new (most visible on color-rich content)
- Theme works on first apply but reverts/glitches after a few

**Phase to address:**
Already fixed in v1.12 — flag for regression vigilance during WebGL phase work.

---

### Pitfall 19: Bundle Size Bloat From "All 6 Enabled by Default"

**What goes wrong:**
We enable all six addons by default to give users the best UX. Cold-cache web-served first paint loads xterm core (~200 KB) + 6 addons (image is the biggest at ~150 KB; webgl ~80 KB; canvas ~50 KB if vendored as fallback; search ~30 KB; web-links ~10 KB; unicode11 ~20 KB; serialize ~30 KB). Total ~570 KB minified. Over Tailscale on iPad cellular, that's noticeable.

**Why it happens:**
"More features = better UX" misses that web-served users on slow links care about first paint more than feature breadth.

**How to avoid:**
- **Lazy-load**: ship an `index.js` that loads xterm core + the lightweight addons (web-links, unicode11) on first paint. Defer webgl, image, search, serialize until actually needed (or until idle-time prefetch).
- Default-on for: webgl (or canvas fallback), web-links, unicode11. These are "always-on quality" — load them eagerly.
- Default-off for: image (rare use case), serialize (no UI surface in v3.2). Load on enable.
- Default-off, lazy-load: search (loaded on first Cmd+F).
- Measure: cold-cache first paint over 3G profile, before/after, against a hard budget. Don't merge if regression > X ms.

**Warning signs:**
- Lighthouse score drops on web-served URL
- p50 first-paint increases after deploy
- iPad/cellular users report slowness

**Phase to address:**
Final integration phase — bundle audit before shipping the feature flag default.

---

### Pitfall 20: Forgetting to Update CSP for Image Worker / Blob URLs

**What goes wrong:**
v3.1 Phase 89 set CSP to `script-src 'self'; connect-src 'self' wss://<host>; style-src 'self' 'unsafe-inline'`. The image addon uses Web Workers and (likely) blob URLs for decoded image data. If we don't update CSP, the addon either fails silently (workers blocked) or with a cryptic console error users won't read. The v3.1 D-09 amendment added `'unsafe-inline'` to `style-src`; v3.2 may need similar amendments. Each one is a security review event, not a casual change.

**Why it happens:**
Each addon has different runtime behaviors. Image needs workers + maybe blobs. WebGL needs `data:` for canvas readback in some paths. Search may use no extra resources. Web-links is pure JS. The author of each integration phase needs to *check* — not assume.

**How to avoid:**
- For each addon, audit at integration time:
  - Does it spawn workers? → `worker-src` directive
  - Does it create blob URLs? → `worker-src 'self' blob:` or `img-src blob:`
  - Does it inline styles? → already covered by `'unsafe-inline'` (but document the new uses)
  - Does it use `eval` or `new Function`? → would require `'unsafe-eval'`, which we should refuse
- Re-run the v3.1 zero-violation Chromium e2e suite after every addon integration.
- Extend the e2e suite to cover Safari (WebKit) and Firefox in v3.2 — Tailscale audience includes both.
- Document each CSP amendment in `.planning/research/` with the same rigor as v3.1 D-09.
- If an addon requires `'unsafe-eval'` or `unsafe-inline` script, **do not ship that addon**. The bar v3.1 set should not be lowered.

**Warning signs:**
- Console: `Refused to ... because it violates the following Content Security Policy directive`
- Image addon doesn't render but no obvious error
- Worker spawn fails silently
- e2e suite goes red

**Phase to address:**
Each addon integration phase + a final CSP audit phase. Treat CSP as a release gate.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Pull addons from npm CDN instead of vendoring | Faster integration in v3.2 | Violates v3.1 CSP discipline; breaks offline; regresses security posture | **Never** — vendor or don't ship |
| Default all 6 addons enabled, optimize later | Demos well, looks feature-rich | First-paint regression on web-served; CSP audit explosion; harder to opt-out for power users | Only if measured budget is met |
| Skip canvas fallback addon ("just use DOM") | One less vendored package | Loses graceful degradation; DOM renderer is markedly slower for active CLIs; users on software-WebGL get the worst of both | Never if WebGL ships |
| Single global "plugins enabled" toggle | Simple UI | Coarse-grained; users want per-feature control especially for image | MVP only — must split before v3.2 GA |
| Apply WebGL hot-swap on existing tabs | Faster UX feedback | Loses scrollback (terminal recreate required); buffer-rebuild edge cases; regresses Phase 35 fix | Never — always "new sessions only" |
| Inline xterm-addon docs in our Settings UI rather than linking | No external dependency | Stale docs; maintenance burden every addon update | Only for security-sensitive labels (image storage, serialize warning) |
| Skip Safari/Firefox e2e CSP coverage | CI runs faster | iPad Safari is a real user surface; bugs slip through | Never — Tailscale audience makes this load-bearing |
| Roll our own settings-migration | Avoids schema-version dep | Bug-prone, every new field is a migration task | When schema is stable; v3.2 is *not* stable yet |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `@xterm/addon-webgl` | Loaded after `fit()` | Load **before** first `fit()` so atlas is ready; `terminal.loadAddon(webgl)` then `fitAddon.fit()` |
| `@xterm/addon-canvas` (fallback) | Not vendored because "we're using webgl" | Vendor as primary fallback path; `onContextLoss` swap-in |
| `@xterm/addon-search` | Used with default 1000-line scrollback assumption | Test with realistic AI coding scrollback (5–20k lines); cap regex; show progress |
| `@xterm/addon-image` | Default 100 MB storage limit kept | Override to 16 MB; expose in Settings; clamp lower on web-served |
| `@xterm/addon-web-links` | Default `window.open` handler | Custom handler: BrowserOpenURL on Wails, `window.open(_,'_blank','noopener,noreferrer')` on web; click-confirm + OSC 8 URL display |
| `@xterm/addon-unicode11` | Forget `terminal.unicode.activeVersion = '11'` after load | The two-step API requires explicit version-set; just `loadAddon` doesn't activate |
| `@xterm/addon-serialize` | Wired to "auto-save scrollback" feature | Don't expose user-facing surface in v3.2; library capability only |
| Vendoring script | Pulls latest addon, doesn't check core peer-dep | Manifest-driven version pin; verify peer-dep ranges; CI gate |
| `internal/relay/` scrollback replay | Byte-level filtering breaks sixel | Verify byte-for-byte fidelity; multi-client image regression test |
| Wails `BrowserOpenURL` vs web `window.open` | Same code path on both | Detect environment; route through Wails binding on desktop |
| CSP | One-time set in v3.1 | Re-audit per addon; treat changes as release gates |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Software WebGL silently used | Sluggish typing, high CPU during render | Detect `RENDERER` string for SwiftShader/llvmpipe; fall back to canvas | Linux laptops, GPU-blacklisted browsers, headless test envs |
| Search on large scrollback | Beach ball / "page unresponsive" on Cmd+F | Cap regex permissiveness; show progress UI; consider scrollback cap | >10k visual lines, regex with no anchors |
| Sixel storage exhaustion | Tab memory growing, eventual OOM | `storageLimit: 16` (16 MB); per-tab not per-app | Watching image-emitting commands in a loop |
| Bundle bloat first paint | Slow on cellular/iPad over Tailscale | Lazy-load image/search/serialize | Cold-cache new device, slow link |
| Render-while-streaming with WebGL atlas miss | Frame stutter when many new chars/colors arrive | Pre-warm atlas on common ASCII set; tune atlas size | High-throughput logs, build outputs, color-heavy CLIs |
| Buffer fan-out for many web clients | Memory growth proportional to client count | Independent scrollback per client (current design) — ensure no shared mutable buffer state grew with addons | >5 clients per session, long sessions |
| Theme switch with full reflow | Brief freeze on theme change | Already mitigated via `clearTextureAtlas`; don't regress | Live theme switching during heavy output |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Default web-links handler with no click-confirm on web-served | Phishing / OAuth-callback exfil / typosquat | Click-confirmation popover with full URL; OSC 8 href display; IDN warning |
| OSC 8 link with mismatched display text vs href | Spoofing — user sees `github.com`, opens `evil.example` | Show full href on hover *and* in click-confirm; visual warning when display text looks URL-like but doesn't match href |
| Serialize wired to disk persistence | Token / API key / PII leak via support attachments, backups | No on-disk serialize in v3.2; if added later: encrypted, ephemeral, opt-in, audit-logged |
| CSP relaxation for an addon | Reverses v3.1 hardening | Treat each `unsafe-*` request as a release-blocking review; refuse `unsafe-eval` |
| Image addon `blob:` worker not in CSP | Worker spawn fails OR succeeds and runs un-vetted | Audit CSP; explicit `worker-src 'self' blob:` if needed |
| Trusting WebGL `extension` strings without sanitization | Information disclosure (driver/GPU details to remote relay) — low severity | Don't transmit `getParameter(RENDERER)` over the wire; keep it client-local |
| Auto-link `file://` or `javascript:` URIs | XSS / local-file disclosure | Allowlist link schemes: `https`, `http`, `mailto`. Reject everything else. |
| Trusting OSC 8 URLs without scheme check | Same as above | Same allowlist applies to OSC 8 hrefs, not just regex-detected URLs |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Toggle with no live effect (renderer / unicode11) | "Settings is broken" reports | "(applies to new sessions)" caption + post-toggle badge |
| Search keybind hijacks browser-find globally on web | Lost browser-find on whole page; a11y break | Focus-conditioned binding; visible search affordance; `/` as alternative |
| Image rendering inconsistent across multi-client | "Why does my coworker see a chart and I don't?" | Per-client capability surfaced in Settings; clear documentation |
| Cmd-click doesn't open in new tab | macOS users frustrated | Platform-aware click handler |
| Plain click in Wails opens link inside WebView | Replaces AgentHub UI with the link target | Route through `BrowserOpenURL` |
| iPad long-press on link does nothing useful | Touch users have no "preview URL" affordance | Implement long-press → URL tooltip |
| Plugin Settings panel grows unbounded | Cluttered Settings with rarely-used knobs | Collapse "advanced" config under disclosure; keep main surface to enable/disable + 0–1 most-relevant option |
| WebGL fallback to canvas with no notice | "Feels slower today" with no explanation | One-time toast: "GPU acceleration unavailable, switched to canvas renderer" with [Try Again] |
| Serialize feature exists but unclear what it does | Users misunderstand "what's saved" | Don't expose user-facing serialize in v3.2; if exposed later, name carefully (not "Save terminal") |
| Cmd+F when terminal not focused does nothing | Power users lost | Always allow browser-default Cmd+F when terminal not focused |

---

## "Looks Done But Isn't" Checklist

- [ ] **WebGL toggle:** Often missing context-loss handler — verify `onContextLoss` is wired and falls back to canvas, not "broken terminal"
- [ ] **WebGL toggle:** Often missing software-renderer detection — verify `RENDERER` string is checked
- [ ] **WebGL toggle:** Often missing Retina/multi-monitor regression test — verify glyph width on Retina + 100/125/150% zoom
- [ ] **Search addon:** Often missing keybind focus-conditioning — verify Cmd+F still works in browser when terminal not focused
- [ ] **Search addon:** Often missing large-scrollback test — verify search on 10k-line fixture doesn't lock for >1s
- [ ] **Image addon:** Often missing storage limit override — verify `storageLimit: 16` (or chosen value) is in the loaded options
- [ ] **Image addon:** Often missing CSP `worker-src` update — verify e2e CSP suite still green after addon enabled
- [ ] **Image addon:** Often missing multi-client replay test — verify client B sees image when joining mid-render
- [ ] **Web-links addon:** Often missing OSC 8 URL display — verify hover and click-confirm show actual href, not display text
- [ ] **Web-links addon:** Often missing scheme allowlist — verify `javascript:` and `file://` URLs are not auto-linked
- [ ] **Web-links addon:** Often missing platform-aware handler — verify Wails uses `BrowserOpenURL`, web uses `window.open` with `noopener`
- [ ] **Unicode 11 addon:** Often missing `activeVersion='11'` step — verify emoji are rendered double-width after enable
- [ ] **Unicode 11 addon:** Often missing "new sessions only" affordance — verify toggle doesn't claim to apply live
- [ ] **Serialize addon:** Often missing scope discipline — verify no user-facing "save terminal" UI in v3.2
- [ ] **Settings UI:** Often missing migration test — verify v3.1 settings.json upgrades cleanly to v3.2 with sensible defaults
- [ ] **Settings UI:** Often missing schema version field — verify `"schemaVersion": 2` is written
- [ ] **CSP audit:** Often missing per-addon review — verify each addon was checked for new directive needs
- [ ] **CSP audit:** Often missing Safari/Firefox coverage — verify e2e suite covers more than Chromium
- [ ] **Vendoring:** Often missing peer-dep verification — verify a script checks core ≥ each addon's `peerDependencies`
- [ ] **Vendoring:** Often missing canvas fallback — verify `@xterm/addon-canvas` is vendored even though feature list says webgl
- [ ] **First paint:** Often missing bundle budget — verify cold-cache 3G first paint regression < N ms
- [ ] **Cross-platform:** Often missing iPad/iOS Safari Tailscale UAT — verify on real device, not just emulator

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| WebGL crashes with no fallback shipped | HIGH | Hotfix release with addon-canvas vendored + onContextLoss handler; document workaround as "Settings → Plugins → disable WebGL" until patched |
| Sixel storage bomb shipped with default 100 MB | MEDIUM | Hotfix `storageLimit: 16`; deploy via auto-update banner; users clear cache or restart |
| Web-links phishing surface shipped without click-confirm | HIGH | Hotfix click-confirm; security advisory in release notes; review whether any incidents reported |
| CSP regression breaks image addon silently | LOW–MEDIUM | Hotfix CSP with `worker-src 'self' blob:`; verify e2e green; update v3.1-style amendment doc |
| Settings.json migration zeroes existing prefs | MEDIUM | Hotfix `defaultSettings()` merge logic; re-run on next start; user prefs partially reconstructable from localStorage if frontend has them |
| Vendor drift breaks search after addon-only update | LOW | Roll back vendored search to last-known-good; pin manifest correctly; re-run vendoring script |
| Initial-paint regression with WebGL on | LOW–MEDIUM | Re-tune rAF cap; add WebGL-specific path; ensure load order: webgl-before-fit |
| Multi-client image fan-out broken | MEDIUM | Audit relay byte-fidelity; add regression test; if upstream serialize limitation is the issue, document it as known limitation |
| Bundle bloat regression detected post-launch | LOW | Lazy-load post-launch; doesn't require server change, just frontend rebuild |
| Unicode 11 visual scrollback drift complaints | LOW | Document expected behavior; add "(applies to new sessions)" if missing; offer "clear scrollback" workaround |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1. WebGL context loss | WebGL phase | Manual test: trigger context loss via `WEBGL_lose_context` extension; confirm canvas fallback engaged, scrollback intact |
| 2. Initial-paint timing regression | WebGL phase | Automated: 4-CLI fit test with WebGL on; assert no off-by-one |
| 3. Sixel storage bomb | Image phase | Test with 50 MB sixel fixture; assert no OOM; assert eviction at limit |
| 4. Image bundle / CSP worker-src | Image phase + CSP audit phase | e2e CSP suite green on Chromium + Safari + Firefox; bundle budget met |
| 5. Image replay for late clients | Image phase + multi-client integration | Multi-client UAT: B joins after A renders image, confirm visible |
| 6. Search lock on large scrollback | Search phase | Performance test: 10k-line fixture, regex search, assert <1s |
| 7. Cmd+F browser-find conflict | Search phase | Manual test: focus header, Cmd+F, browser find appears; focus terminal, Cmd+F, xterm search appears |
| 8. Web-links phishing | Web-links phase | Security review checklist; OSC 8 spoof fixture test; click-confirm UI present |
| 9. Click platform conventions | Web-links phase | Manual test on Wails macOS/Linux/Windows + web Chrome/Safari/Firefox + iPad |
| 10. WebGL on Tailscale audience | WebGL phase | iPad Safari Tailscale UAT; software-renderer detection unit test |
| 11. Unicode 11 scrollback reflow | Unicode 11 phase + Settings UI phase | Manual test: existing scrollback before/after enable; "(new sessions)" affordance present |
| 12. Serialize PII/secrets | Serialize phase | Scope review: no on-disk serialize, no user-facing surface in v3.2 |
| 13. "Applies to new sessions" miscom | Settings UI phase | UI review: every relevant toggle has affordance; usability test with 3+ users |
| 14. Settings.json migration | Settings persistence phase | Fixture test: load v3.1 settings.json, assert v3.2 defaults populated, assert no zeroing |
| 15. Multi-client plugin state drift | Multi-client integration phase | Decision recorded in PROJECT.md Key Decisions; UI distinguishes local vs shared toggles |
| 16. Vendoring drift | Vendoring phase | Manifest verify script in CI; peer-dep check fails if mismatched |
| 17. FitAddon + WebGL geometry | WebGL phase | Retina + multi-zoom regression test |
| 18. Theme switch atlas regression | Cross-cutting / WebGL phase | Existing test preserved; comment block linking to this pitfall |
| 19. Bundle bloat first paint | Final integration phase | 3G cold-cache benchmark vs. budget; lazy-load wired |
| 20. Forgetting CSP per addon | Each addon phase + final CSP audit phase | Per-addon CSP checklist in plan; final e2e suite green on 3 browsers |

---

## Sources

- [@xterm/addon-webgl README — context loss handling](https://github.com/xtermjs/xterm.js/blob/master/addons/addon-webgl/README.md)
- [@xterm/addon-canvas npm — fallback renderer](https://www.npmjs.com/package/@xterm/addon-canvas)
- [WEBGL_lose_context: loseContext() — MDN](https://developer.mozilla.org/en-US/docs/Web/API/WEBGL_lose_context/loseContext)
- [WebGL HandlingContextLost — Khronos wiki](https://www.khronos.org/webgl/wiki/HandlingContextLost)
- [@xterm/addon-image — sixel + IIP, storageLimit default 100 MB](https://www.npmjs.com/package/@xterm/addon-image)
- [jerch/xterm-addon-image — upstream addon source](https://github.com/jerch/xterm-addon-image)
- [jerch/xterm-addon-image issue #47 — "images aren't serialized"](https://github.com/jerch/xterm-addon-image/issues/47)
- [Are We Sixel Yet? — terminal sixel support landscape](https://www.arewesixelyet.com/)
- [@xterm/addon-search — npm](https://www.npmjs.com/package/@xterm/addon-search)
- [xterm.js issue #5176 — search is too slow](https://github.com/xtermjs/xterm.js/issues/5176)
- [xterm.js issue #4902 — search slow on large wrapped scrollback](https://github.com/xtermjs/xterm.js/issues/4902)
- [@xterm/addon-web-links — npm](https://www.npmjs.com/package/@xterm/addon-web-links)
- [xterm.js Link Handling guide](https://xtermjs.org/docs/guides/link-handling/)
- [Hyperlinks in Terminal Emulators — egmontkob gist (OSC 8 reference)](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda)
- [xterm-addon-web-links issue #3557 — Electron URL handling](https://github.com/xtermjs/xterm.js/issues/3557)
- [@xterm/addon-unicode11 — npm](https://www.npmjs.com/package/@xterm/addon-unicode11)
- [xterm.js issue #1709 — Unicode handling discussion](https://github.com/xtermjs/xterm.js/issues/1709)
- [@xterm/addon-serialize source](https://github.com/xtermjs/xterm.js/tree/master/addons/addon-serialize)
- [xterm.js using-addons guide](https://github.com/xtermjs/xtermjs.org/blob/master/_docs/guides/using-addons.md)
- AgentHub `.planning/PROJECT.md` (v3.1 vendoring + CSP, v2.0 multi-client fan-out, v1.6 Phase 35 fit fix, v1.12 clearTextureAtlas decision)
- AgentHub `.planning/MILESTONES.md` (v1.6, v2.0, v3.0, v3.1 milestone scope and known tech debt)

---
*Pitfalls research for: v3.2 xterm.js Plugin Suite (Issue #36) — addon integration into established Wails + multi-client web-served terminal app*
*Researched: 2026-05-03*
