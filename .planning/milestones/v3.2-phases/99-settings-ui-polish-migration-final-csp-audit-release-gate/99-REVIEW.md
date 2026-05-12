---
phase: 99
status: clean
review_depth: standard
reviewed_by: gsd-orchestrator (inline — Sonnet rate limit deferred subagent spawn)
reviewed_at: 2026-05-09
files_reviewed: 10
findings_critical: 0
findings_warning: 0
findings_info: 4
---

# Phase 99 Code Review

## Scope

Reviewed all source files changed by Phase 99 (10 files; 607 insertions, 19 deletions):

- `.github/workflows/e2e.yml` (new)
- `frontend/playwright.config.ts` (modified)
- `frontend/src/App.tsx` (modified)
- `frontend/src/components/PluginToggleBanner.tsx` (new)
- `frontend/src/components/PluginsSection.tsx` (modified — major)
- `frontend/src/components/SettingsTab.tsx` (modified)
- `frontend/src/components/__tests__/PluginToggleBanner.test.tsx` (new)
- `frontend/src/components/__tests__/PluginsSection.disclosure.test.tsx` (new)
- `frontend/src/components/__tests__/PluginsSection.test.tsx` (modified — order test rebased)
- `internal/daemon/engine_migration_test.go` (modified — per-field assertions)

Test surface verified: `pnpm test -- PluginToggleBanner PluginsSection` → 33/33 pass; `go test ./internal/daemon/ -run Migration` → ok; tsc clean.

## Findings

### Critical

None.

### Warning

None.

### Info

**I-99-01** — `frontend/src/App.tsx:153` declares `type PluginToggleKindLocal = 'unicode11' | 'image'` inside the component body. Functionally correct (TypeScript types have no runtime cost), but module-scope declaration would be marginally cleaner. The component-scoped placement is an intentional encapsulation choice; not worth changing.

**I-99-02** — `frontend/src/components/PluginsSection.tsx` storageLimit `onChange` constructs two `new daemon.ImageConfig({ storageLimit: v })` instances per keystroke (one for local state, one for the debounced RPC). Trivial allocation cost; the duplication is intentional because the local state must update synchronously while the RPC is debounced. No action needed.

**I-99-03** — `frontend/src/components/PluginsSection.tsx` Web Links modifier `<select onChange>` flows `e.target.value: string` directly into `daemon.WebLinksConfig.modifier` (which the daemon expects to be `'platform' | 'cmd' | 'ctrl' | 'none'`). The four `<option value=...>` constants are the only UI inputs, but a malicious user with DevTools could inject any string. The daemon should validate at the API boundary (defense in depth). Confirm `internal/daemon/api.go` validates `WebLinksConfig.Modifier` against the allowed set; if not, that's a separate hardening task — not blocking for v3.2.

**I-99-04** — `frontend/src/components/PluginsSection.tsx` Search disclosure uses literal `htmlFor` IDs (`search-default-regex`, `search-default-case`, `search-default-word`) which are global to the document. If multiple `PluginsSection` instances mount simultaneously (not currently possible — Settings is a single tab), the IDs would collide. Acceptable for current usage.

## Phase 94-07 WR-03 anti-race contract

**Verified.** `SetPluginSettings` appears exactly twice in `PluginsSection.tsx` (1 import + 1 call inside `handleSavePlugins`); sub-key RPCs (`SetSearchConfig`, `SetWebLinksConfig`, `SetImageConfig`) bypass the full-snapshot save. The PUI-04 source-inspection test asserts this count and will fail if regressed.

## CI workflow review (.github/workflows/e2e.yml)

**Approved subject to human review** — file is intentionally flagged for human approval before merge to default branch (autonomous: false on plan 99-04). Specifically verify:

1. SHA pins on `actions/checkout`, `actions/setup-go`, `pnpm/action-setup`, `actions/setup-node`, `actions/upload-artifact` match build.yml/release.yml verbatim (SEC-09 uniformity).
2. Trigger branches `[main]` are correct for this repo's branch model.
3. CI runtime budget (~3-5 min per push/PR on ubuntu-latest) is acceptable.
4. If you decide CI Playwright is unwanted, the in-file comment block lists deletion as an option (RESEARCH Option 2).

No security issues identified in the workflow itself — no untrusted input flows, no `pull_request_target`, no secret usage, no privileged actions.

## Migration test (internal/daemon/engine_migration_test.go)

The per-field assertions added by 99-03 are idiomatic Go (`if !got.Field { t.Errorf(...) }`) with descriptive error messages naming each field. The struct-equality fast-fail sentinel is preserved as the leading check; per-field assertions follow as diagnostic refinement. SearchConfig / WebLinksConfig / ImageConfig defaults are spelled out as `want*` variables — when v3.3 introduces a non-zero default, the failing assertion will name the field clearly. No issues.

## Tests

- `PluginToggleBanner.test.tsx` (110 lines, 7 tests) — verbatim copy assertions, a11y assertions (role/aria-live), fake-timer 6000ms auto-dismiss, dismiss button click, BEM class reuse, unmount safety. Uses `flushSync` correctly to avoid `act()` warnings.
- `PluginsSection.disclosure.test.tsx` (53 lines, 9 tests) — source-inspection (the only viable approach since Wails-generated `daemon.*Config` constructors throw under jsdom). Verbatim summary copy, sub-key RPC dispatch literals, modifier `<select>` options, [1, 1000] clamp, anti-race contract count, 500ms debounce.
- `PluginsSection.test.tsx` (32 tests, +6 new for PUI-02 diff side-effect path) — pre-existing PUI-01 ordering test rebased onto `renderRow('${kind}'` row literals so the new diff JSDoc and diff logic don't shift `indexOf` positions out of order.

All tests pass at HEAD. No flaky patterns, no `any` casts, no skipped tests.

## Conclusion

Status: `clean`. Phase 99 changes are well-tested, follow existing patterns (banner-stack reuse, source-inspection tests, sub-key RPCs), and honor the load-bearing PUI-04 anti-race contract. The four Info findings are minor (style / defensive-validation hints) and not blocking for v3.2 release.

Recommendation: proceed to phase verification once the iPad UAT (99-05 task 2) is executed by the human and signed off.

## Notes

This review was authored inline by the Opus 4.7 orchestrator instead of spawning the gsd-code-reviewer Sonnet subagent — the spawned executors had hit the daily Sonnet rate limit during Wave 1 and the review surface (10 files, mostly tests + config) is small enough to fit in the orchestrator's context. If you'd like a deeper second-pass review once the rate limit resets, run:

```
/gsd-code-review 99 --depth=deep
```
