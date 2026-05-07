---
status: findings
phase: 96
phase_name: image-addon-csp-audit
depth: standard
files_reviewed: 16
critical: 0
warning: 1
info: 4
created: 2026-05-07
files_reviewed_list:
  - internal/daemon/plugin_settings.go
  - internal/daemon/engine.go
  - internal/daemon/api.go
  - internal/daemon/client.go
  - internal/webserver/csp_mw.go
  - internal/webserver/csp_mw_test.go
  - internal/webserver/vendor_drift_test.go
  - internal/webserver/browser_csp_image_e2e_test.go
  - internal/daemon/api_image_test.go
  - internal/daemon/image_config_test.go
  - internal/relay/image_byte_fidelity_test.go
  - web/embed.go
  - web/terminal.html
  - web/assets/terminal.js
  - web/vendor/xterm/VERSION
  - app.go
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/components/PluginsSection.tsx
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - frontend/src/wailsjs/go/models.ts
  - frontend/package.json
---

# Phase 96 Code Review

## Summary

Phase 96 ships a structurally clean mirror of Phase 95: `ImageConfig` sub-key
plumbing (Go struct + defaults + sub-key writer + PATCH route + Wails RPC +
TS class), CSP `'wasm-unsafe-eval'` amendment with a token-aware regression
guard, addon-image vendoring (web/embed.go + VERSION + script tag + drift-test
bump), TerminalPanel MOUNT-useEffect ImageAddon construction with
`enableSizeReports: false` and `try/catch` around dispose, and a multi-client
relay byte-fidelity test. The CSP middleware change is one line plus a
substantial Amendment 2 documentation block; the new `extractDirective` helper
in csp_mw_test.go is correct under spot-checked edge cases (trailing semicolon
or not, last clause, prefix collisions like `script-src-elem`). Sub-key writer
concurrency contract (lock → mutate → save → capture listener → unlock →
invoke) is preserved verbatim from `SetWebLinksConfig`. Defense-in-depth on
the PATCH handler (`MaxBytesReader(8192)` + `DisallowUnknownFields()` + range
check) is present and tested. Wails hand-edited bindings (App.d.ts, App.js,
models.ts) match the Go signatures and follow the sibling
`SetWebLinksConfig` pattern.

One WARNING: the implemented numeric range for `StorageLimit` is `[1, 1000]`,
which deviates from the design intent `[1, 128]` documented in 96-PATTERNS.md
and 96-RESEARCH.md (where 128 MB is the upstream addon-image ceiling and
16 MB is the AgentHub-chosen default). The handler's own comment block
acknowledges the deviation as "hypothetical future power-user advanced
disclosure" — the deviation is internally consistent (handler doc, daemon
client doc, and api_image_test.go all use 1000) but unjustified by Phase 96
requirements (no UI ships in 96; the cap is theoretical until Phase 99
PUI-03). The remaining four findings are INFO-level (cosmetic / documentation
alignment / minor style inconsistency).

No critical issues found. No security vulnerabilities. No correctness bugs.

---

## Findings

### WR-01 — StorageLimit upper bound deviates from documented plan: `[1, 1000]` shipped vs `[1, 128]` planned

**Severity:** WARNING
**File:** `internal/daemon/api.go:628-647`

The PATCH handler validates `req.StorageLimit < 1 || req.StorageLimit > 1000`,
rejecting only values outside `[1, 1000]`. The plan-of-record (96-PATTERNS.md
§`internal/daemon/api.go` "Adapt — value validation"; 96-RESEARCH.md
"Pitfall #6: Storage Cap Too High → Tab OOM") prescribed an upper bound of
**128 MB** — the upstream addon-image default (and the largest reasonable
value that aligns with the "tab-OOM mitigation per ROADMAP Phase 96 SC-3"
rationale the handler comment cites). The handler's own docstring
acknowledges the gap: "the upstream addon-image default is 128 MB but
AgentHub locks 16 MB by default, allowing user override up to 1000 MB hard
cap for hypothetical future power-user advanced disclosure."

Concrete consequences:

1. A directly-edited `settings.json` (or a future RPC caller) can persist
   `imageConfig.storageLimit = 1000`, allocating up to ~1 GB of decoded RGBA
   per terminal tab — well past the SC-3 tab-OOM ceiling Phase 96 set out to
   enforce.
2. The DaemonClient docstring (`internal/daemon/client.go:183`), the handler
   docstring, and `api_image_test.go` all encode `[1, 1000]` consistently —
   so this isn't an off-by-one or copy-paste typo, it's a deliberate
   widening that escaped plan review.
3. There is currently no UI to set this value (Phase 99 PUI-03 deferral), so
   this is latent and reachable only via direct settings.json edit or a
   handcrafted PATCH. Risk in 96 is theoretical; the WARNING is for the
   inheritable design surface that ships into Phase 99.

**Fix (one of):**

(a) Tighten the bound to match the plan:

```go
if req.StorageLimit < 1 || req.StorageLimit > 128 {
    http.Error(w, "storageLimit must be in range [1, 128]", http.StatusBadRequest)
    return
}
```

…and update the matching test cases in `api_image_test.go`
(`StorageLimit=1001` → `StorageLimit=129`; `wantBodyHasOne` `"1000"` →
`"128"`), the DaemonClient docstring at `client.go:183-184`
("StorageLimit is in [1, 1000] MB" → "[1, 128] MB"), and the handler's own
docstring.

(b) If the wider range is intentional (and 1000 was chosen knowingly),
update 96-PATTERNS.md and ROADMAP `Phase 96 SC-3` so the tab-OOM ceiling
prose matches reality, and add an explicit "Decisions" entry in STATE.md
documenting the deviation rationale. The deviation should not be
rediscovered as folklore in Phase 99.

---

### IN-01 — Web parity: storageLimit fallback uses `||` (rejects 0 as falsy) while desktop uses `??` (respects 0)

**Severity:** INFO
**File:** `web/assets/terminal.js:250` (compared to `frontend/src/components/TerminalPanel.tsx:205`)

Web:
```javascript
var storageLimit = (pluginConfig.imageConfig && pluginConfig.imageConfig.storageLimit) || 16;
```

Desktop:
```typescript
const storageLimit = pluginConfig?.imageConfig?.storageLimit ?? 16
```

If a future change ever lets `storageLimit: 0` reach the client (it cannot
today — the daemon validates `[1, 1000]` server-side and rejects 0 with
HTTP 400), the two clients would disagree: the web would silently fall back
to 16, the desktop would honor 0. Today this divergence is harmless because
the server gate makes 0 unreachable. Worth aligning if/when the web client
gains direct settings entry, or pre-emptively to keep the parity invariant
crisp.

**Fix:** Either (a) document the divergence in a one-line comment at
terminal.js:250 explaining that `||` is intentional and safe because the
daemon rejects 0, or (b) tighten the web check to mirror the desktop:

```javascript
var hasLimit = pluginConfig.imageConfig && typeof pluginConfig.imageConfig.storageLimit === 'number';
var storageLimit = hasLimit ? pluginConfig.imageConfig.storageLimit : 16;
```

---

### IN-02 — `(*DaemonClient).SetImageConfig` docstring says `[1, 1000]` but doesn't explain the 1000-vs-128 discrepancy with planned cap

**Severity:** INFO
**File:** `internal/daemon/client.go:183-184`

```go
// The daemon validates StorageLimit is in [1, 1000] MB; out-of-range values
// surface as a non-nil error from this call (HTTP 400 from the handler).
```

This docstring is internally consistent with the handler, but readers
referencing 96-PATTERNS.md or 96-RESEARCH.md will see "128 MB" and wonder
which is authoritative. If WR-01 is resolved by tightening to 128, this
goes away automatically. If the wider range is preserved, add one sentence
naming the rationale (and pointing to STATE.md §Decisions where the
deviation should be logged).

**Fix:** Bound the docstring to the rationale:

```go
// The daemon validates StorageLimit is in [1, 1000] MB; out-of-range values
// surface as a non-nil error from this call (HTTP 400 from the handler).
// Note: 1000 is a hard ceiling reserved for Phase 99 power-user disclosure;
// the v3.2 default is 16 MB and there is no UI to set this in 96 (see
// STATE.md §Decisions).
```

---

### IN-03 — `image_config_test.go` constructs `SessionEngine` literal directly, bypassing `NewSessionEngine` — depends on undocumented zero-value contract

**Severity:** INFO
**File:** `internal/daemon/image_config_test.go:48-52, 110-113`

```go
e := &SessionEngine{
    configDir:      dir,
    cliPaths:       make(map[string]string),
    pluginSettings: baseline,
}
```

The test sidesteps any constructor and assumes `e.mu` (zero-value sync.Mutex
is valid), `e.pluginSettingsListener` (nil at start, then set by
`SetPluginSettingsListener`), and `e.saveSettingsToDisk()` are all
self-sufficient given just `configDir`, `cliPaths`, and `pluginSettings`.
The test is correct today, but it pins an internal-state contract that is
not enforced by any constructor signature. If `SessionEngine` ever gains a
field that requires non-zero initialization (e.g. a buffered channel, a
context), this test will compile and run with subtle wrong behavior rather
than fail loudly. The sibling `web_links_config_test.go` and
`search_config_test.go` use the same pattern, so this is a pre-existing
convention — INFO, not WARNING — but worth flagging while in the area.

**Fix:** Either (a) factor a private `newTestSessionEngine(t, dir,
baseline)` helper in a `_test.go` file that documents the minimum-viable
field set, or (b) leave as-is and accept the pre-existing convention.
Recommend (a) only if a fourth `*_config_test.go` joins this family in
Phase 99 — three is the rule-of-three threshold for abstraction.

---

### IN-04 — CSP Amendment 2 doc block in `csp_mw.go` references the test by name, creating a doc-test rename coupling

**Severity:** INFO
**File:** `internal/webserver/csp_mw.go:43-44`

```go
// which this codebase explicitly forbids via the token-aware
// TestCSPHeaders_NoUnsafeEvalToken_TokenAware regression guard
// (csp_mw_test.go).
```

Naming the test in the source-file documentation is great for readers but
creates a hidden coupling: if a future refactor renames the test (e.g.
shortens it to `TestCSPHeaders_BareUnsafeEvalForbidden`), this comment
silently goes stale. There is no compile-time link. The Phase 95 CSP
amendment block (lines 7-19) does NOT reference test names by exact
identifier — it cites the e2e test class only ("Chromium e2e test
TestBrowserCSP_TerminalNoViolations") which is more stable.

**Fix:** Loosen the citation to the test FILE rather than the test
identifier:

```go
// which this codebase explicitly forbids via the token-aware regression
// guard in csp_mw_test.go (TestCSPHeaders_NoUnsafeEvalToken_TokenAware as
// of 2026-05-07).
```

…or accept the coupling and add a comment-anchor regex check in
csp_mw_test.go that asserts the named test exists (mirror the
"comment-anchor" pattern PATTERNS.md mentions for the TerminalPanel mount
useEffect range check).

---

_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
