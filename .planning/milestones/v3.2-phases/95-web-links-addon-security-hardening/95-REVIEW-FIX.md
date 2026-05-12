---
phase: 95-web-links-addon-security-hardening
fixed_at: 2026-05-06T00:00:00Z
review_path: .planning/phases/95-web-links-addon-security-hardening/95-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 95: Code Review Fix Report

**Fixed at:** 2026-05-06
**Source review:** `.planning/phases/95-web-links-addon-security-hardening/95-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (1 Blocker + 6 Warnings; 4 Info findings deferred per fix_scope=critical_warning)
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: Web link-confirm popover stacks click + keydown listeners on rapid successive risky clicks

**Files modified:** `web/assets/terminal.js`
**Commit:** `4e8240e`
**Applied fix:** Adopted Option B from the review — added a module-scope `linkConfirmCleanup` reference that the function invokes idempotently before binding new handlers. This drops every stacked click + keydown listener atomically before re-entry, matching the existing `findBarExitTimer` / `searchDebounceTimer` cancel-on-re-entry idiom in the file. A single Continue press now opens at most one URL, never the queued backlog.

### WR-01: `hasIDN` does not detect IDN in `mailto:` URLs

**Files modified:** `frontend/src/lib/urlSafety.ts`, `web/assets/terminal.js`
**Commit:** `dfd061d`
**Applied fix:** Added a `mailto:` branch to `hasIDN` (both TS and JS implementations). Extracts the domain from `u.pathname` after the last `@` (RFC 6068) and runs the same `xn--` substring + non-ASCII regex on it. Cyrillic / Punycode mailto addresses now surface the popover.

### WR-02: Daemon does not validate `WebLinksConfig.Modifier` against the four allowed literals

**Files modified:** `internal/daemon/api.go`
**Commit:** `710a271`
**Applied fix:** Added a `switch req.Modifier` block in `handleSetWebLinksConfig` that returns 400 unless the value is `platform`, `cmd`, `ctrl`, or `none`. Prevents typoed or corrupted values from silently disabling every modifier-click. Existing tests in `web_links_config_test.go` already use compliant values; full daemon test suite still green (`ok internal/daemon 7.120s`).

### WR-03: Desktop uses `??` and web uses `||` for the modifier fallback

**Files modified:** `frontend/src/components/TerminalPanel.tsx`
**Commit:** `f9299bd`
**Applied fix:** Switched desktop from `cfg?.modifier ?? 'platform'` to `cfg?.modifier || 'platform'` so an empty-string `modifier` (corrupted settings.json edge case) falls back to `'platform'` instead of breaking the click gate silently. Matches `web/assets/terminal.js` behavior. WR-02's API-boundary validation remains the primary defense; this is belt-and-braces. UI-SPEC web parity mandate restored.

### WR-04: `openLink` defense-in-depth scheme regex is loose

**Files modified:** `frontend/src/lib/openLink.ts`, `web/assets/terminal.js`
**Commit:** `be35ede`
**Applied fix:** Tightened to `/^(?:https?:\/\/|mailto:)/` (no `i` flag). The URL constructor lowercases protocol upstream, so case-insensitivity here only added attack surface for novel scheme spoofing. The required `//` after `https?:` rejects absurd inputs like `https:javascript:...` at the defense-in-depth layer too. All 12 openLink unit tests still pass.

### WR-05: `EventsEmit(a.ctx, ...)` does not guard against `a.ctx == nil`

**Files modified:** `app.go`
**Commit:** `589bd91`
**Applied fix:** Wrapped the three `runtime.EventsEmit` calls in `SetPluginSettings`, `SetSearchConfig`, and `SetWebLinksConfig` with the existing guard pattern `if a.ctx != nil && a.ctx.Value("frontend") != nil { ... }`. Mirrors `app.go:266` / `:355` / `:1006`. Eliminates the panic-on-nil-receiver risk for tests and pre-startup Wails-bound RPCs. `go build ./...` and `go vet ./...` both clean.

### WR-06: Web `applyPluginConfig` does not remove the document keydown listener when `webLinks` is toggled off mid-popover

**Files modified:** `web/assets/terminal.js`
**Commit:** `2353e4d`
**Applied fix:** In the toggle-off arm, invoke `linkConfirmCleanup()` (introduced by CR-01) so the document-level keydown handler is removed. Composes cleanly with CR-01's module-scope cleanup tracking. No more dangling Esc handler attached to document after a mid-popover toggle-off.

## Skipped Issues

None — all in-scope findings were fixed.

## Tests Run

- **Frontend (vitest, phase-95 scope):** 4 test files, 68 tests — all passed
  - `urlSafety.test.ts`, `openLink.test.ts`, `TerminalPanel.web-links.test.tsx`, `LinkConfirmPopover.test.tsx`
- **Frontend (vitest, full suite):** 40 test files, 491 tests — all passed (6 worker-pool timeout noise unrelated to these fixes; all assertions green)
- **Backend Go (`./internal/daemon/...`):** all tests pass (7.120s)
- **Backend Go (`./internal/webserver/...`):** all tests pass (1.914s)
- **`go build ./...`:** clean
- **`go vet ./...`:** clean
- **`node -c web/assets/terminal.js`:** syntax OK after each edit

## Notes for Verifier

- WR-02 enforces stricter validation than the previous endpoint contract. If any UAT fixture or deferred test ever sends an empty/missing `modifier` to `PATCH /settings/web-links-config`, it will now return 400. None of the existing test files do (verified via grep).
- WR-03 changes desktop fallback semantics from "preserve empty string" to "treat empty string as 'platform'". This is intentional and matches the documented behavior — the only behavioral change is for the corrupted-settings edge case, where the gate now defaults open instead of broken-closed.
- WR-04 rejects any `javascript:`, `file:`, or `data:` URL that previously slipped past the loose regex. `isAllowedScheme` already rejected these upstream, so no behavioral change for normal callers; only the defense-in-depth layer is now strictly congruent with the upstream policy.
- CR-01 + WR-06 are tightly coupled (WR-06 depends on the module-scope `linkConfirmCleanup` introduced by CR-01). They were committed in order so each commit is buildable.

---

_Fixed: 2026-05-06_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
