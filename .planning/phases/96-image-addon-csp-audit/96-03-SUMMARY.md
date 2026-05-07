---
phase: 96
plan: 03
subsystem: webserver/csp
tags: [phase-96, csp, security, wave-1, img-03, amendment-2]
requires: [96-01]
provides:
  - "script-src 'self' 'wasm-unsafe-eval' (CSP3 §6.3) on all HTML-serving routes"
  - "Token-aware 'unsafe-eval' regression guard (extractDirective + strings.Fields)"
  - "v3.1 D-09-rigor Amendment 2 documentation block in csp_mw.go package comment"
affects:
  - "All HTML-serving routes (terminal.html, dashboard.html, join.html) — CSP response header now permits WebAssembly.compile/instantiate"
tech-stack:
  added: []
  patterns:
    - "Token-aware CSP directive assertion (extract clause → strings.Fields → per-token equality)"
    - "Amendment block documentation: policy spec line + rationale + RESEARCH citation + narrow-vs-broad contrast + browser support floors"
key-files:
  created: []
  modified:
    - internal/webserver/csp_mw.go
    - internal/webserver/csp_mw_test.go
decisions:
  - "Approach A (whitespace tokenization via strings.Fields on the script-src clause) chosen over regex word-boundary — simpler, more idiomatic Go, easier to read."
  - "extractDirective helper added once at file bottom; reused by both new tests."
  - "'unsafe-eval' entry REMOVED from globallyForbidden substring slice in TestCSPHeaders_NoUnsafeTokens with a redirect comment pointing at TestCSPHeaders_NoUnsafeEvalToken_TokenAware (the token-aware test owns the responsibility)."
metrics:
  tasks_completed: 1
  files_modified: 2
  files_created: 0
  duration_minutes: ~15
  commits: 1
  tests_added: 0  # both test bodies already existed as t.Skip scaffolds in Plan 96-01
  tests_flipped_red_to_green: 2
completed: 2026-05-07
---

# Phase 96 Plan 03: CSP `'wasm-unsafe-eval'` Amendment 2 Summary

**One-liner:** Added `'wasm-unsafe-eval'` to the `script-src` CSP directive (single-token append) and shipped the v3.1 D-09-rigor Amendment 2 documentation block + token-aware regression guard so the substring `'unsafe-eval'` ⊂ `'wasm-unsafe-eval'` overlap doesn't silently break the existing defense check.

## What Shipped

### Source line change (`internal/webserver/csp_mw.go`)

The literal one-token amendment in the string builder:

```go
// Phase 96 IMG-03 Amendment 2: 'wasm-unsafe-eval' permits the
// @xterm/addon-image SIXEL/IIP WASM decoder to instantiate
// (CSP3 §6.3 governs WebAssembly.compile/instantiate/Module/Instance).
// 'wasm-unsafe-eval' is NARROW — it does NOT enable JS eval() or
// new Function() (those still require the broader 'unsafe-eval', which
// remains forbidden by csp_mw_test.go). See package comment Amendment 2.
b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")
```

Result CSP (script-src clause): `script-src 'self' 'wasm-unsafe-eval'`. All other directives unchanged.

### Amendment 2 documentation block (`csp_mw.go` package comment)

The package comment now carries TWO amendment blocks (Amendment 1 = Phase 89 D-09 `'unsafe-inline'` for style-src; Amendment 2 = NEW). Amendment 2 contains all five plan-required elements:

1. **Updated policy spec line:** `script-src 'self' 'wasm-unsafe-eval'; ...`
2. **Amendment rationale:** `@xterm/addon-image` embeds SIXEL + IIP WebAssembly bytecode (~10 KB raw) instantiated synchronously via `WebAssembly.instantiate / new WebAssembly.Module / new WebAssembly.Instance` on the main thread.
3. **Discovery vector citation:** `RESEARCH §"Mandatory Pre-Phase CSP Audit Finding 2"` (single-line literal so grep gates pass) — references 0 Workers / 0 eval calls / 6 WASM bootstraps surfaced by the audit.
4. **Defense-in-depth contrast:** `'wasm-unsafe-eval'` is **NARROW** (only `WebAssembly.compile/instantiate/Module/Instance`); `'unsafe-eval'` is **BROAD** (also enables JS `eval()`, `new Function()`, `setTimeout(string)`). The two source expressions share a substring but are distinct CSP3 tokens — hence the token-aware regression guard.
5. **Browser support floors:** Chrome 102+ (May 2022), Firefox 102+ (June 2022), Safari 16.0+ (Sept 2022), iPad Safari 16.0+ (Sept 2022). All four are within the Phase 99 supported matrix with multi-year headroom.

### Token-tightening fix (`internal/webserver/csp_mw_test.go`)

Two scaffolds from Plan 96-01 (`t.Skip("Pending until Plan 96-03 ...")`) flipped to GREEN:

- **`TestCSPHeaders_HasWasmUnsafeEval`** — asserts `'wasm-unsafe-eval'` is present in the CSP header AND lives inside the `script-src` clause (defensive directive-isolation check via `extractDirective`).
- **`TestCSPHeaders_NoUnsafeEvalToken_TokenAware`** — extracts the `script-src` clause, tokenizes via `strings.Fields`, asserts no token equals exactly `"'unsafe-eval'"` (the bare-token form). Permits `'wasm-unsafe-eval'` as a distinct token.

`extractDirective` helper added at the bottom of `csp_mw_test.go`:

```go
// extractDirective returns the value portion of a single CSP directive
// (e.g. extractDirective("default-src 'none'; script-src 'self'", "script-src")
// returns "'self'"). Returns empty string if the directive is not present.
func extractDirective(csp, name string) string { ... }
```

Existing `TestCSPHeaders_NoUnsafeTokens` tightened: `"'unsafe-eval'"` REMOVED from the `globallyForbidden` substring slice (would falsely match `'wasm-unsafe-eval'` after Amendment 2). A redirect comment above the slice points readers to `TestCSPHeaders_NoUnsafeEvalToken_TokenAware`, which carries the responsibility correctly. `'unsafe-hashes'` retained (no overlap risk; substring-safe).

### Test results

```
=== RUN   TestCSPHeaders_HasWasmUnsafeEval
--- PASS: TestCSPHeaders_HasWasmUnsafeEval (0.00s)
=== RUN   TestCSPHeaders_NoUnsafeEvalToken_TokenAware
--- PASS: TestCSPHeaders_NoUnsafeEvalToken_TokenAware (0.00s)
```

All 15 CSP tests in the suite (including the legacy 8 + 5 strict + 2 newly-flipped) pass. `go test ./internal/webserver/... -count=1` exits 0. `go vet` clean. `gofmt -l` reports no diffs.

## Deviations from Plan

None — plan executed exactly as written. One micro-adjustment: the RESEARCH citation `§"Mandatory Pre-Phase CSP Audit Finding 2"` was kept on a single Go comment line (rather than the wrapped two-line form the planner prototyped) so the acceptance-criteria grep gate matches the literal phrase. This is faithful to the plan's intent (the gate explicitly checks for the literal phrase) — call it a typographic refinement, not a deviation.

## Wave Coordination

This plan is **parallel-safe with Plan 96-02** (daemon sub-key RPC):

- **Disjoint files:** 96-02 touches the daemon RPC layer; 96-03 touches `internal/webserver/csp_mw{,_test}.go`. No overlap.
- **No shared interfaces:** the CSP middleware does not consume daemon RPC; the daemon does not consume CSP headers.

## Plan-Checker Compliance

Frontmatter `must_haves` (the v3.1 D-09 rigor contract) all satisfied:

| Truth | Evidence |
|-------|----------|
| `'wasm-unsafe-eval'` lives in script-src (and only script-src) | `TestCSPHeaders_HasWasmUnsafeEval` (defensive `extractDirective` check) |
| No bare `'unsafe-eval'` token anywhere | `TestCSPHeaders_NoUnsafeEvalToken_TokenAware` (token-aware) |
| Amendment 2 block with 5 required elements | `grep` gates AC1-AC7 all green |
| Both Wave-0 scaffolds flipped SKIP→PASS | `go test -run TestCSPHeaders_HasWasmUnsafeEval -v` shows `--- PASS:` |
| Existing `TestCSPHeaders_RequiredTokens` + `TestCSPHeaders_NoUnsafeTokens` still pass (after tightening) | Full suite green |

## Next Wave Implications

- **Plan 96-04** (chromedp e2e) will now find `'wasm-unsafe-eval'` in the live CSP header and the addon-image WASM decoder will instantiate without browser console violations.
- **Plan 96-05/06** (Wails parity / docs) inherit the amended policy unchanged — no additional CSP work needed.
- **Future addons** that need WASM bootstrap continue to ride the existing `'wasm-unsafe-eval'` allowance; no further `script-src` changes anticipated.

## Self-Check: PASSED

- File `internal/webserver/csp_mw.go` exists and contains amended directive + Amendment 2 block.
- File `internal/webserver/csp_mw_test.go` exists with two flipped tests + `extractDirective` helper + tightened slice.
- Commit `9bccee2` exists in git log.
- All acceptance-criteria grep gates green (AC1=2, AC2=3, AC3=2, AC4=1, AC5=4, AC6=4, AC7=4, AC8=1, AC9=5, AC10=0, AC11=NONE-OK).
- All 15 CSP tests pass; `go vet` exits 0; `gofmt -l` produces no output.
