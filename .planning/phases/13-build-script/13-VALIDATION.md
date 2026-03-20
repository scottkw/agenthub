---
phase: 13
slug: build-script
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-20
updated: 2026-03-20
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) + go test (backend) + bash (build script) |
| **Config file** | `frontend/vite.config.ts` (test section) |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test && cd .. && go test ./...` |
| **Build script tests** | `bash tests/build-script.test.sh` |
| **Estimated runtime** | ~15 seconds (unit); BUILD integration tests ~5-10 min each |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test` (regression guard)
- **After every plan wave:** Run `cd frontend && pnpm test && cd .. && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green + BUILD integration checks
- **Max feedback latency:** 15 seconds (unit tests)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File | Status |
|---------|------|------|-------------|-----------|-------------------|------|--------|
| 13-01-01 | 01 | 1 | BUILD-01 | integration | `bash build.sh --platform macos && test -d build/bin/agenthub.app` | manual-only | manual-only |
| 13-01-02 | 01 | 1 | BUILD-03 | integration | `bash build.sh --platform windows && test -f build/bin/agenthub.exe` | manual-only | manual-only |
| 13-01-03 | 01 | 1 | BUILD-02 | integration | `bash build.sh --platform linux && file build/bin/agenthub \| grep ELF` | manual-only | manual-only |
| 13-01-04 | 01 | 1 | BUILD-04 | integration | `bash build.sh --all && test -d build/bin/agenthub.app && test -f build/bin/agenthub.exe` | manual-only | manual-only |
| 13-02-01 | 02 | 2 | BUILD-05 | manual | N/A — requires Apple Developer credentials | N/A | manual-only |
| arg-parse-01 | 01 | 1 | BUILD-01 | unit/smoke | `bash tests/build-script.test.sh` | `tests/build-script.test.sh` | green |
| arg-parse-02 | 01 | 1 | BUILD-03 | unit/smoke | `bash tests/build-script.test.sh` | `tests/build-script.test.sh` | green |
| arg-parse-03 | 02 | 2 | BUILD-05 | unit/smoke | `bash tests/build-script.test.sh` | `tests/build-script.test.sh` | green |

*Status: manual-only · green · red · flaky*

---

## Automated Test Coverage (tests/build-script.test.sh)

35 tests, 0 failures. Covers:

| Category | Tests | Requirement |
|----------|-------|-------------|
| File properties (exists, executable) | 2 | BUILD-01..04 |
| Syntax check (bash -n) | 1 | All |
| No-args exits 1 + prints Usage: | 2 | All |
| --platform bogus exits 1 + error | 3 | All |
| --platform without value exits 1 + error | 2 | All |
| Unknown flag exits 1 + error | 2 | All |
| --sign without env vars: "Missing required environment variables" | 1 | BUILD-05 |
| --sign without env vars: lists all 4 var names | 4 | BUILD-05 |
| Function names present (build_macos/windows/linux/sign_and_notarize) | 4 | BUILD-01..05 |
| darwin/universal + -clean in macOS function | 2 | BUILD-01 |
| x86_64-w64-mingw32-gcc in Windows function | 1 | BUILD-03 |
| docker run in Linux function | 1 | BUILD-02 |
| notarytool submit + --wait present | 2 | BUILD-05 |
| ditto -c -k --keepParent (not zip -r) | 1 | BUILD-05 |
| spctl --assess present | 1 | BUILD-05 |
| stapler staple present | 1 | BUILD-05 |
| docker info prerequisite check | 1 | BUILD-02 |
| command -v "$MINGW_CC" prerequisite check | 1 | BUILD-03 |
| WAILS path via go env GOPATH | 1 | All |
| Shebang + set -euo pipefail | 2 | All |

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Status |
|----------|-------------|------------|--------|
| `build.sh --platform macos` produces `build/bin/agenthub.app` | BUILD-01 | Requires Wails + Go toolchain | Verified during phase execution (13-01-SUMMARY.md) |
| `build.sh --platform windows` produces `build/bin/agenthub.exe` | BUILD-03 | Requires mingw-w64 | Manual-only |
| `build.sh --platform linux` produces ELF binary | BUILD-02 | Requires Docker | Manual-only |
| `build.sh --all` builds all three platforms | BUILD-04 | Requires all toolchains | Manual-only |
| `build.sh --platform macos --sign` with valid credentials produces signed+notarized build passing `spctl --assess` | BUILD-05 | Requires Apple Developer credentials | Verified manually (missing-credentials path) |

---

## Validation Sign-Off

- [x] All tasks have automated or manual-only classification
- [x] Argument parsing, error paths, and pattern checks fully automated
- [x] BUILD-01 (macOS) verified during phase execution; no persistent test file gap — tests/build-script.test.sh covers static+behavioral checks
- [x] BUILD-02/03/04 correctly classified as manual-only (external toolchains required)
- [x] BUILD-05 missing-credentials path covered by automated test; full signing pipeline manual-only
- [x] No watch-mode flags
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete
