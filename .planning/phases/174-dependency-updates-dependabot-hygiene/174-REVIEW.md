---
phase: 174-dependency-updates-dependabot-hygiene
reviewed: 2026-07-08T00:00:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - .github/dependabot.yml
  - .github/workflows/build.yml
  - .github/workflows/e2e.yml
  - .github/workflows/release.yml
  - go.mod
  - go.sum
findings:
  critical: 0
  warning: 0
  info: 2
  total: 2
status: clean
---

# Phase 174: Code Review Report

**Reviewed:** 2026-07-08
**Depth:** standard
**Files Reviewed:** 6
**Status:** clean

## Summary

Phase 174 is a dependency-hygiene phase: four SHA-pinned GitHub Actions bumps, three Go module bumps, and three surgical Dependabot `ignore` deferrals. Every load-bearing invariant was verified with tooling, not by eye:

- **SHA/version-comment integrity (SEC-critical):** All four bumped `uses:` pins were resolved against the upstream GitHub tag API and match their `# vX.Y.Z` comments exactly:
  - `actions/setup-go@924ae3a1…` = v6.5.0 (OK)
  - `pnpm/action-setup@0ebf4713…` = v6.0.9 (OK)
  - `actions/attest-build-provenance@0f67c3f4…` = v4.1.1 (OK)
  - `softprops/action-gh-release@718ea10b…` = v3.0.1 (OK)
  - Unchanged pins spot-checked and still valid: `actions/checkout@df4cb1c0` = v6.0.3, `actions/setup-node@48b55a01` = v6.4.0.
- **go.sum integrity:** `go mod verify` → "all modules verified"; `go mod tidy` produces **zero** diff (go.mod/go.sum are internally consistent and tidy). New sums for coder/websocket v1.8.15, golang.org/x/term v0.44.0, and goreleaser/nfpm/v2 v2.47.0 are present and consistent.
- **go-webview2 pin (Windows build constraint):** Untouched — remains `github.com/wailsapp/go-webview2 v1.0.19 // indirect`. Not present in the go.sum diff.
- **dependabot.yml:** Parses as valid YAML; both ecosystems (`github-actions`, `gomod`) present; `ignore` entries use correct syntax (`update-types` for the checkout semver-major defer, `versions: [">=X"]` for the wails/tailscale range defers).

No BLOCKER or WARNING findings. Two INFO-level scope-hygiene notes below are advisory only and do not block shipping.

## Info

### IN-01: go.mod/go.sum change is broader than the stated "three module bumps"

**File:** `go.sum` (and `go.mod`)
**Issue:** The phase scope described three direct bumps (coder/websocket, x/term, nfpm/v2), but `go mod tidy` also pulled several transitive updates into go.sum that were not individually called out in the commits or plan:
- `github.com/go-git/go-git/v5` 5.18.0 → 5.19.1
- `github.com/go-git/go-billy/v5` 5.8.0 → 5.9.0
- `github.com/pjbgf/sha1cd` 0.3.2 → 0.6.0 (notable multi-minor jump)
- `github.com/klauspost/compress` 1.18.5 → 1.18.6
- `github.com/klauspost/cpuid/v2` 2.3.0 (new indirect entry)
- `golang.org/x/crypto` 0.51.0 → 0.52.0, `golang.org/x/net` 0.53.0 → 0.55.0, `golang.org/x/exp` bump

These are legitimate downstream consequences of bumping nfpm (goreleaser → go-git chain) and are internally consistent (`go mod verify` passes, tidy is a no-op), so there is no correctness risk. The note exists only so a reviewer confirms these transitive bumps — especially the sha1cd and go-git jumps in the git-object/crypto path — were intended rather than an accidental `go get -u`.
**Fix:** No code change required. If desired, add a one-line note in the phase summary listing the transitive bumps carried in by the nfpm/x-family updates so the delta is traceable.

### IN-02: Undocumented `go` directive bump 1.26.3 → 1.26.4

**File:** `go.mod:3`
**Issue:** The `go` directive was raised from `1.26.3` to `1.26.4` — an out-of-scope change not mentioned in the phase's bump list (likely auto-written by `go mod tidy` under a newer local toolchain, go1.26.5). CI is unaffected because every `setup-go` step uses `go-version-file: go.mod` and will install the pinned patch. The only practical effect is that local contributors/builders on exactly 1.26.3 will now hit `go.mod requires go >= 1.26.4` until they update.
**Fix:** No action required if a 1.26.x floor is acceptable. If the bump was unintentional, revert the directive to `1.26.3` (there is no `toolchain` line, so this is a one-line edit) since none of the three module bumps require 1.26.4.

---

_Reviewed: 2026-07-08_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
