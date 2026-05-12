---
phase: 95-web-links-addon-security-hardening
reviewed: 2026-05-06T00:00:00Z
depth: standard
files_reviewed: 28
files_reviewed_list:
  - app.go
  - frontend/e2e/web-links-live-toggle.spec.ts
  - frontend/src/__tests__/App.plugin-event.test.tsx
  - frontend/src/components/__tests__/LinkConfirmPopover.test.tsx
  - frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx
  - frontend/src/components/LinkConfirmPopover.tsx
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/lib/__tests__/openLink.test.ts
  - frontend/src/lib/__tests__/urlSafety.test.ts
  - frontend/src/lib/openLink.ts
  - frontend/src/lib/urlSafety.ts
  - frontend/src/style.css
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - frontend/src/wailsjs/go/models.ts
  - internal/daemon/api.go
  - internal/daemon/client.go
  - internal/daemon/engine.go
  - internal/daemon/plugin_settings_test.go
  - internal/daemon/plugin_settings.go
  - internal/daemon/web_links_config_test.go
  - internal/webserver/vendor_drift_test.go
  - internal/webserver/web_links_test.go
  - web/assets/terminal.css
  - web/assets/terminal.js
  - web/embed.go
  - web/terminal.html
findings:
  blocker: 1
  warning: 6
  info: 4
  total: 11
status: issues_found
---

# Phase 95: Code Review Report

**Reviewed:** 2026-05-06
**Depth:** standard
**Files Reviewed:** 28 (filtered to 27 source files; 1 docs-style PLAN/RESEARCH context wasn't in scope)
**Status:** issues_found

## Summary

Phase 95 hardens the xterm web-links click pipeline with a scheme allowlist, modifier-click gates, IDN/typosquat detection, a confirmation popover, and full web parity. The core security model is sound: `isAllowedScheme` re-validates inside `openLink` (defense-in-depth), `window.open` is invoked with the literal `'_blank', 'noopener,noreferrer'`, and the popover renders untrusted URL text via React text content / DOM `textContent` (never `innerHTML`). The vendor-drift test correctly extends to seven `@xterm/*` packages and the source-inspection guards (banned navigation patterns, BrowserOpenURL absence in `terminal.js`) are tight.

Issues fall in three buckets:

1. **Web parity popover has a real event-listener-stacking bug** that opens unintended URLs on rapid successive risky clicks (BLOCKER). The desktop React path is unaffected because state-driven re-mount cleans up `useEffect` listeners; the plain-DOM web mirror leaks them.
2. **Several silent-failure surfaces** where invalid daemon config (unknown `modifier` value, empty string) makes link clicks behave inconsistently between desktop (`??`) and web (`||`) and degrades gracefully but quietly.
3. **Defense-in-depth gaps** in `hasIDN` (mailto IDN bypass, `xn--` substring false-positives) and `openLink`'s loose scheme regex that don't break security today but invite regressions.

## Blocker Issues

### CR-01: Web link-confirm popover stacks click + keydown listeners on rapid successive risky clicks → opens multiple URLs from one Continue press

**File:** `web/assets/terminal.js:344-379`
**Issue:** `showLinkConfirmPopover` is a singleton DOM popover. Each invocation calls `addEventListener('click', handleContinue)` / `('click', handleCancel)` / `(document, 'keydown', handleKey)` and registers a `cleanup()` closure that only removes its own three references. If the user (or a synthesized event sequence — see the Cyrillic + typosquat UAT walks in `frontend/e2e/web-links-live-toggle.spec.ts`) modifier-clicks a second risky URL while the first popover is still showing, invocation 2 stacks new handlers on top of the still-registered invocation 1. Pressing **Continue** then fires BOTH `handleContinue1` (closes over `url1`) AND `handleContinue2` (closes over `url2`), invoking `openLink(url1)` AND `openLink(url2)` from a single click. Same compounding for Esc on `document.keydown`.

Trace:

```
click link A → showLinkConfirmPopover(urlA, ...)
   continueBtn handlers: [hC_A]
   cancelBtn handlers:   [hX_A]
   document handlers:    [hK_A]
click link B (popover already visible — popover.hidden was already false)
   showLinkConfirmPopover(urlB, ...) — does NOT call cleanup() on the old invocation
   continueBtn handlers: [hC_A, hC_B]
   cancelBtn handlers:   [hX_A, hX_B]
   document handlers:    [hK_A, hK_B]
user presses Continue
   hC_A runs → cleanup_A → openLink(urlA)
   hC_B runs → cleanup_B → openLink(urlB)   ← UNINTENDED
```

`openLink` is bare `window.open(url, '_blank', 'noopener,noreferrer')` so the user gets two popup tabs — one of which they did NOT confirm. Worse, if the second risky URL was the malicious one and the user was about to click Cancel for it but they had not yet seen the second popover content (because the popover only re-binds reason/url textContent — doesn't queue), they may unwittingly Continue on a URL they thought they were dismissing.

The desktop counterpart in `frontend/src/components/TerminalPanel.tsx:655-667` uses React state (`linkConfirmState`) so the popover is destroyed and re-mounted; the `useEffect` cleanup in `LinkConfirmPopover.tsx:69-78` removes the old `keydown` listener. Web parity diverges here.

**Fix:** Run cleanup before re-binding, idempotent style. Two equivalent options:

Option A — clone-and-replace the buttons to drop ALL existing listeners atomically:

```javascript
function showLinkConfirmPopover(url, risk, x, y) {
  var pop = document.getElementById('link-confirm-popover');
  if (!pop) return;
  // ... reasonEl / urlEl / position code unchanged ...

  // Clear ALL stacked listeners from previous invocation by replacing the
  // button nodes with clones (DOM-spec way to drop every handler at once).
  var oldContinue = document.getElementById('link-confirm-continue');
  var oldCancel = document.getElementById('link-confirm-cancel');
  var continueBtn = oldContinue.cloneNode(true);
  var cancelBtn = oldCancel.cloneNode(true);
  oldContinue.replaceWith(continueBtn);
  oldCancel.replaceWith(cancelBtn);

  // Track the document keydown handler at module scope so we can remove the
  // previous one before adding a new one.
  if (linkConfirmKeyHandler) {
    document.removeEventListener('keydown', linkConfirmKeyHandler);
    linkConfirmKeyHandler = null;
  }
  // ... rest of function with handlers attached to fresh buttons ...
}
```

Option B — gate at top: if popover is already shown, dismiss the previous one first:

```javascript
var linkConfirmCleanup = null; // module scope

function showLinkConfirmPopover(url, risk, x, y) {
  if (linkConfirmCleanup) linkConfirmCleanup(); // idempotent dismiss of previous
  // ... rest of function ...
  linkConfirmCleanup = cleanup;
}
```

Option B is the smaller diff and matches the existing `findBarExitTimer`/`searchDebounceTimer` cancel-on-re-entry idiom already established in this file.

## Warnings

### WR-01: `hasIDN` does not detect IDN in `mailto:` URLs (allowed scheme)

**File:** `frontend/src/lib/urlSafety.ts:56-71` and `web/assets/terminal.js:273-289`
**Issue:** `mailto:` is in `ALLOWED_SCHEMES`. A `mailto:user@домен.рф` URL has `new URL(...).hostname === ''` (mailto URLs have no host component per WHATWG URL spec) so the primary `hasIDN` path returns false. The fallback regex `/^[a-z][a-z0-9+.-]*:\/\/([^/?#]+)/i` requires `://` — also doesn't match `mailto:`. So an IDN-spoofed mailto URL bypasses the IDN risk path entirely; only the typosquat list (which only stores web hostnames) and OSC8 mismatch (dormant for plain-text URLs) might catch it. Net effect: a Cyrillic-spoofed mailto URL opens without confirmation.

This is a hardening gap, not a current exploit — `mailto:` URLs are consumed by the system mail client, not a browser, so the homograph risk is reduced but not zero (a mail client autocomplete could route to the spoofed domain).

**Fix:** Extract the local-part-after-`@` from mailto URLs and run the IDN check on it:

```typescript
export function hasIDN(href: string): boolean {
  try {
    const u = new URL(href);
    if (u.protocol === 'mailto:') {
      // mailto has no .hostname; pull domain from pathname (RFC 6068).
      const at = u.pathname.lastIndexOf('@');
      if (at < 0) return false;
      const domain = u.pathname.slice(at + 1);
      return domain.includes('xn--') || /[^\x00-\x7F]/.test(domain);
    }
    if (u.hostname.includes('xn--')) return true;
    if (/[^\x00-\x7F]/.test(u.hostname)) return true;
    return false;
  } catch {
    // ... existing fallback ...
  }
}
```

Mirror the change in `web/assets/terminal.js`.

### WR-02: Daemon does not validate `WebLinksConfig.Modifier` against the four allowed literals

**File:** `internal/daemon/api.go:590-603`, `internal/daemon/plugin_settings.go:33-45`
**Issue:** `handleSetWebLinksConfig` decodes a `WebLinksConfig` struct with `DisallowUnknownFields` (good) and 8 KiB cap (good), but `Modifier` is `string` with no value-set validation. A request body `{"modifier":"hax","confirmOSC8":true,...}` is accepted and persisted. On the next click, `isModifierPressed(event, 'hax')` falls through every `if` and returns `false` — every modifier-click is silently gated off, so the user's web-links feature appears broken with no feedback. There is also no symmetric validation in the engine `SetWebLinksConfig` writer (the public Wails-bound API in `app.go:541`) — same risk via the desktop path.

This is not a security bypass (the gate fails closed), but it is a UX cliff: a typoed value or a corrupted settings.json silently disables the entire feature. The struct comment at `plugin_settings.go:34` documents the four valid values; the code doesn't enforce them.

**Fix:** Validate at the API boundary before persistence.

```go
func (a *API) handleSetWebLinksConfig(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 8192)
    var req WebLinksConfig
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    switch req.Modifier {
    case "platform", "cmd", "ctrl", "none":
        // ok
    default:
        http.Error(w, "modifier must be one of: platform, cmd, ctrl, none", http.StatusBadRequest)
        return
    }
    a.engine.SetWebLinksConfig(req)
    w.WriteHeader(http.StatusNoContent)
}
```

Optionally, also coerce on `loadSettingsFromDisk` (settings.json edited by hand): if `Modifier` is empty or unknown, replace with `"platform"`.

### WR-03: Desktop uses `??` and web uses `||` for the modifier fallback — inconsistent behavior on empty-string `modifier`

**File:** `frontend/src/components/TerminalPanel.tsx:397` vs `web/assets/terminal.js:504`
**Issue:**

- Desktop: `const modifier = (cfg?.modifier ?? 'platform') as ModifierMode` — nullish coalescing. Empty string `""` flows through and `isModifierPressed(event, '')` returns false → click gated off.
- Web: `var modifier = cfg.modifier || 'platform';` — truthy fallback. Empty string falls back to `'platform'` → click works.

If a corrupted/edge-case settings.json has `"modifier": ""`, desktop links break silently while web links keep working. Either choice is defensible; **inconsistency between platforms is the bug**. UI-SPEC mandates web parity verbatim.

**Fix:** Make both use the same fallback policy. The safer / more forgiving choice is the web behavior (treat empty string as "use default"):

```typescript
// frontend/src/components/TerminalPanel.tsx
const modifier = (cfg?.modifier || 'platform') as ModifierMode
```

Or — and this composes with WR-02's daemon-side validation — keep `??` and prevent empty strings from reaching this layer at the source. Either way, pick one and apply consistently.

### WR-04: `openLink` defense-in-depth scheme regex is loose and accepts `Https://`/`MAILTO:` upper-case forms while `isAllowedScheme` would reject them on some platforms

**File:** `frontend/src/lib/openLink.ts:49`, `web/assets/terminal.js:334`
**Issue:** The regex `/^(https?:|mailto:)/i` is case-insensitive. `URL` constructor lowercases `protocol`, so `isAllowedScheme('HTTPS://example.com')` returns true (lowercased to `https:`), and `openLink` accepts `HTTPS://example.com` and forwards to `window.open` / `BrowserOpenURL`. Both downstream sinks tolerate mixed case. So no current bug — but the case-insensitive flag is unnecessary and obscures intent. More importantly:

- The regex does NOT anchor with `$` after the colon, which is correct for `https://example.com` but means the regex would also accept absurd inputs like `https:foo` (no `//`) or `https:javascript:alert(1)` — both of which `URL` would still parse with `protocol === 'https:'` (the rest becomes pathname). `window.open('https:javascript:alert(1)')` is harmless because the browser navigates to `https:javascript:alert(1)` as a relative URL, NOT a `javascript:` URL. So no security bypass.

The looseness is fine **only because the upstream `isAllowedScheme` already validated via `new URL().protocol`**. If a future refactor removed the upstream check, the defense-in-depth in `openLink` would be weaker than commented.

**Fix:** Tighten the regex to match only `https://`, `http://`, and `mailto:` exactly, and remove the `i` flag (URL constructor normalizes to lowercase upstream — case-sensitivity here only adds attack surface for novel scheme spoofing):

```typescript
if (!/^(?:https?:\/\/|mailto:)/.test(url)) return;
```

Mirror in `web/assets/terminal.js`.

### WR-05: `EventsEmit(a.ctx, ...)` in `SetWebLinksConfig` (and `SetSearchConfig` / `SetPluginSettings`) does not guard against `a.ctx == nil` like other emit sites do

**File:** `app.go:557` (SetWebLinksConfig), `app.go:521` (SetSearchConfig), `app.go:487` (SetPluginSettings)
**Issue:** The pattern `if a.ctx != nil && a.ctx.Value("frontend") != nil { runtime.EventsEmit(...) }` is used at `app.go:266`, `app.go:355`, `app.go:1006` etc., but the three plugin-settings setters call `runtime.EventsEmit(a.ctx, ...)` unconditionally. If `a.ctx` is nil (test harness, or an early Wails-bound RPC before `startup`), `runtime.EventsEmit` will panic on the nil receiver. Phase 95 inherits this from Phase 92 / Phase 94 — it is a pre-existing pattern that Phase 95 did not introduce, but it now applies to one more surface (`SetWebLinksConfig`).

**Fix:** Apply the same guard:

```go
if a.ctx != nil && a.ctx.Value("frontend") != nil {
    runtime.EventsEmit(a.ctx, "settings:plugins", full)
}
```

Apply consistently to `SetPluginSettings`, `SetSearchConfig`, and `SetWebLinksConfig`. Without the guard a Wails-bound RPC fired before `startup` (or in a test that doesn't construct a frontend context) panics.

### WR-06: Web `applyPluginConfig` does not remove the document keydown listener when `webLinks` is toggled off mid-popover

**File:** `web/assets/terminal.js:491-498`
**Issue:** The toggle-off arm sets `popOff.hidden = true` but does NOT remove the `document.addEventListener('keydown', handleKey)` that `showLinkConfirmPopover` registered. Until the user explicitly Cancels/Continues (which they cannot do because the popover is now hidden), that handler stays attached. Pressing Esc anywhere else in the page now hides the (already-hidden) popover and silently invokes the closure — minor leak, not user-visible breakage.

This compounds with CR-01: rapidly toggling webLinks off and back on while the popover is shown can leave multiple stacked keydown listeners.

**Fix:** Track the cleanup closure at module scope and invoke it from the toggle-off arm (the same fix proposed for CR-01 Option B addresses both):

```javascript
var linkConfirmCleanup = null;

// ... in showLinkConfirmPopover:
function cleanup() {
  pop.hidden = true;
  continueBtn.removeEventListener('click', handleContinue);
  cancelBtn.removeEventListener('click', handleCancel);
  document.removeEventListener('keydown', handleKey);
  linkConfirmCleanup = null;
}
linkConfirmCleanup = cleanup;

// ... in applyPluginConfig toggle-off arm (line 491):
if (!newConfig.webLinks && webLinksAddonHandle) {
  if (linkConfirmCleanup) linkConfirmCleanup();
  // ... existing dispose code ...
}
```

## Info

### IN-01: `hasIDN` `xn--` substring check is over-broad — flags hosts that contain `xn--` in a non-label position

**File:** `frontend/src/lib/urlSafety.ts:59` and `web/assets/terminal.js:276`
**Issue:** `u.hostname.includes('xn--')` matches any substring, not the IDN-label prefix. A pathological host like `fooxn--bar.example.com` would be flagged as IDN. Punycode prefixes are always at label boundaries (`label.xn--ascii.tld` or `xn--ascii`), so a stricter check would be:

```typescript
// Match xn-- only at the start of a DNS label (after . or at hostname start).
if (/(^|\.)xn--/.test(u.hostname)) return true;
```

Net effect today: false positives surface the popover on harmless hostnames. Risk is one-way (over-flagging), so this is informational, not a defect.

### IN-02: `osc8Mismatch` falls back to `mismatch=true` when `displayText` does not parse as a URL — but plain ASCII text that happens to contain a slash and be parseable can yield false negatives

**File:** `frontend/src/lib/urlSafety.ts:35-45` and `web/assets/terminal.js:290-297`
**Issue:** Currently dormant per Plan B (the handler always passes `displayText === uri`). When v3.3 wires real OSC 8 display strings, edge cases like `displayText = "github.com/foo"` (no scheme) will throw in `new URL(displayText.trim())` and fall through to `return true` — correct. But `displayText = "http://x"` and `href = "http://attacker.example/y"` parses both successfully; same protocol, different host → returns true. Good. The risk is `displayText = "//github.com"` which is a protocol-relative URL — `new URL` requires a base, so it throws → mismatch. OK.

This is a Plan-B-dormant code path; flagged here as a Wave-3 follow-up to validate before wiring goes live.

### IN-03: `LinkConfirmPopover.useLayoutEffect` lint-disable comment masks a real concern: stale closure on `position`

**File:** `frontend/src/components/LinkConfirmPopover.tsx:96-101`
**Issue:** The `// eslint-disable-next-line react-hooks/exhaustive-deps` comment intentionally omits `position` from the dep array. The condition `if (nextLeft !== position.left || nextTop !== position.top)` reads `position` from the closure created at the deps' last refresh — but since `position` is omitted, this read uses the stale value from the previous `[x, y]` change. Trace shows that `nextLeft` is computed from props `x` and rect, so on the second invocation (after setPosition triggered re-render with the SAME `[x, y]` deps) the effect should not actually re-run because deps didn't change. So the stale-closure read is only a concern if `[x, y]` change in lockstep with a parent re-render — possible but the condition still compares correctly because `nextLeft === position.left` for a no-op tick. Confirmed safe; left as info because the lint disable is the kind of comment that gets copy-pasted into a future bug.

### IN-04: Test `frontend/src/lib/__tests__/openLink.test.ts` mutates `navigator.platform` globally without restoration

**File:** `frontend/src/lib/__tests__/openLink.test.ts:104-137`
**Issue:** `setPlatform('Linux x86_64')` sets `navigator.platform` via `Object.defineProperty`. Subsequent test files that run in the same vitest worker and read `navigator.platform` will see the last-set value — `MacIntel` if the file ends on `'ctrl'`, or whatever the previous `it` block set. This is currently survivable because the only consumer in this worker is `urlSafety.test.ts` (does not read platform) and the helpers are deterministic w.r.t. the property at call time. Add an `afterEach` that restores the original:

```typescript
const originalPlatform = navigator.platform;
afterEach(() => setPlatform(originalPlatform));
```

This is a test-hygiene nit, not a defect.

---

_Reviewed: 2026-05-06_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
