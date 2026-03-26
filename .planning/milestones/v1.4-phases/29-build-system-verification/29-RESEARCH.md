# Phase 29: Build System & Verification — Research

**Researched:** 2026-03-25
**Domain:** Go build toolchain, Wails v2 CI, GitHub Actions, Go race detector
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BUILD-01 | `build.sh` produces single binary that handles GUI + CLI + daemon | build.sh already builds only `agenthub`; no structural changes needed — verified by inspection and `tests/build-script.test.sh` (35/35 pass) |
| BUILD-02 | GitHub Actions CI builds and tests unified binary | CI exists at `.github/workflows/build.yml`; currently runs `go test ./...` without `-race`; build-script.test.sh is not run in CI — both gaps need closing |
| BUILD-03 | All existing tests pass against unified binary (daemon tests, CLI tests, attach tests) | 194 tests pass with `go test -race -count=1 ./...` locally; no failures |
</phase_requirements>

---

## Summary

Phase 29 is primarily a CI hardening phase, not a build restructuring phase. The build pipeline already produces a single unified binary (`agenthub`) since Phase 27 merged CLI into the root package and Phase 28 deleted the old `cmd/agenthub-cli/` package entirely. The `wails.json` output filename is `agenthub`, `build.sh` has no references to the deleted CLI binary, and all 194 tests pass locally with `-race`.

The two concrete gaps are: (1) the CI workflow runs `go test ./...` without the `-race` flag, violating BUILD-03's requirement that tests pass under race detection; and (2) `tests/build-script.test.sh` exists locally (35 behavioral tests for build.sh) but is not executed in CI. BUILD-02 requires CI to validate the unified binary — adding `-race` to the test step and invoking the build-script tests on Linux covers that requirement without adding new tests.

BUILD-01 is already satisfied. The phase requires verifying it (the verification is already in `tests/build-script.test.sh`) and ensuring CI confirms it.

**Primary recommendation:** Two CI workflow edits — add `-race` to `go test`, add a `bash tests/build-script.test.sh` step on Linux — plus verification that the exact test counts mentioned in the success criteria are met locally before CI is updated.

---

## Standard Stack

### Core (already in project — no new dependencies)

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| Go | 1.26.1 (from go.mod) | Build and test language | Project language |
| Wails v2 | v2.10.2 (from go.mod) | GUI framework / build tool | Established in project |
| `go test -race` | stdlib | Race condition detection | Go standard tooling |
| GitHub Actions | — | CI pipeline | Already configured |
| `dAppServer/wails-build-action@main` | main | Cross-platform Wails CI builds | Already used in workflow |

### Supporting (already in project)

| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| `actions/setup-go@v5` | v5 | Go toolchain setup in CI | Already in workflow |
| `actions/upload-artifact@v4` | v4 | Artifact upload | Already in workflow |
| bash test script | — | build.sh behavioral verification | Run on Linux CI matrix leg |

**Installation:** No new packages needed.

---

## Architecture Patterns

### Current Project Structure (relevant to phase)

```
/
├── main.go                        # Unified dispatch — GUI / CLI / daemon
├── build.sh                       # Cross-platform build script
├── wails.json                     # Wails config: outputfilename = "agenthub"
├── go.mod                         # go 1.26.1
├── tests/
│   └── build-script.test.sh       # 35-test behavioral suite for build.sh
├── cmd_cli_test.go                # 25 CLI command tests (TestCmd*)
├── cmd_daemon_test.go             # 7 daemon command tests (TestCmdDaemon*)
├── cmd_attach_test.go             # 7 attach/relay tests
├── dispatch_test.go               # 5 dispatch routing tests
├── tray_test.go                   # 2 tray tests
├── internal/
│   ├── daemon/                    # 44 tests (api, client, engine, process, service, socket)
│   ├── relay/                     # 28 tests
│   ├── pty/                       # tests
│   ├── status/                    # tests
│   └── webserver/                 # tests
└── .github/workflows/
    └── build.yml                  # CI: 4-platform matrix build + go test ./...
```

### Pattern: Minimal CI Diff

The CI file is an established workflow. The change is surgical — modify the existing `go test` step name/command, add one bash step on Linux only. Do not restructure the matrix or add new jobs.

### Pattern: build-script tests on Linux only

`tests/build-script.test.sh` checks bash syntax, argument parsing, static patterns in `build.sh`. It does not require Wails, Docker, or mingw-w64 (by design). It runs on any Linux/macOS with bash. In CI, run it on the `ubuntu-latest` leg (the first Linux leg in the matrix) to avoid running it 4x.

```yaml
- name: Run build script tests
  if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'
  run: bash tests/build-script.test.sh
```

### Pattern: Go race detector in CI

```yaml
- name: Run Go tests (all platforms, race detector)
  run: go test -race ./...
```

The `-race` flag works on all three OS families (macOS, Linux, Windows) with CGO enabled. The GitHub Actions matrix uses native runners (not containers), so race detection is available everywhere.

### Anti-Patterns to Avoid

- **Adding new tests in Phase 29:** This phase verifies the existing suite passes — do not add tests to hit a count target.
- **Running build-script tests on all 4 matrix legs:** They're pure bash static checks — running once on Linux is sufficient.
- **Hardcoding Go version in CI:** The workflow already uses `go-version-file: go.mod` — do not change this to a hardcoded version string.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform Wails builds | Custom Docker/CMake steps | `dAppServer/wails-build-action@main` | Already handles Node, pnpm, webkit deps, NSIS per platform |
| Race detector | Custom mutex auditing | `go test -race` | Built into Go runtime, zero configuration |
| Build script verification | Inline CI shell script | `tests/build-script.test.sh` | Already exists with 35 tests; simply invoke it |

---

## Common Pitfalls

### Pitfall 1: `-race` on Windows requires CGO
**What goes wrong:** `go test -race` requires CGO on all platforms. On the Windows matrix leg, if CGO is not enabled or the toolchain is not set up, the race detector fails to compile.
**Why it happens:** The race detector uses a C runtime component.
**How to avoid:** The `dAppServer/wails-build-action` already sets up the environment for CGO on Windows (MinGW). The `go test -race ./...` step runs before the wails build step, so the Go toolchain is already present via `actions/setup-go`. Verify the Windows test step passes by checking that `go test -race ./...` succeeds in the existing CI run after the change.
**Warning signs:** CI failure on `windows-latest` with `CGO_ENABLED` or `-race` linker errors.

### Pitfall 2: `-race` increases test runtime significantly
**What goes wrong:** Tests that previously ran in 10s now take 60+s under the race detector, causing CI timeouts.
**Why it happens:** Race instrumentation is 5-10x slower.
**How to avoid:** Locally confirmed: `go test -race -count=1 ./...` completes in ~18s on the developer machine. With CI overhead, budget ~90s. Default GitHub Actions job timeout is 6h — no risk.
**Warning signs:** N/A for this project size.

### Pitfall 3: build-script test path is hardcoded
**What goes wrong:** `tests/build-script.test.sh` contains the line `BUILD_SH="/Users/ken/dev/agenthub/build.sh"` — a hardcoded absolute path that will fail in CI (checked out to a different path).
**Why it happens:** The script was written for local development.
**How to avoid:** The script must be updated to use a path relative to the script itself or the project root. Options: `BUILD_SH="$(dirname "$0")/../build.sh"` or pass as an argument, or use `${GITHUB_WORKSPACE}/build.sh`. The simplest fix is to use a relative path from the project root: `BUILD_SH="$(cd "$(dirname "$0")/.." && pwd)/build.sh"`.
**Warning signs:** CI failure on the build-script test step with "file not found: /Users/ken/dev/agenthub/build.sh".

### Pitfall 4: Success criteria counts are aspirational, not actual
**What goes wrong:** The phase description says "28+ daemon tests, 16 CLI tests, 7 attach tests" — these are approximate. Actual counts are: 44 daemon tests (across `internal/daemon/`), 33 `TestCmd*` tests in the root package, 7 attach tests.
**Why it happens:** Counts were set when the plan was written.
**How to avoid:** The actual test run `go test -race ./...` passing green is the authoritative success criterion. Do not add padding tests to hit specific numbers.

---

## Code Examples

### Corrected build-script.test.sh path resolution

```bash
# Source: tests/build-script.test.sh (line 10)
# Current (broken in CI):
BUILD_SH="/Users/ken/dev/agenthub/build.sh"

# Fix: resolve relative to script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SH="$SCRIPT_DIR/../build.sh"
```

### Updated CI test step (adding -race)

```yaml
# Source: .github/workflows/build.yml (line 59-60)
# Current:
- name: Run Go tests (all platforms)
  run: go test ./...

# Updated:
- name: Run Go tests (all platforms, race detector)
  run: go test -race ./...
```

### New CI step for build-script tests

```yaml
# Add after the Go test step, before the Wails build step
- name: Run build script tests
  if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'
  run: bash tests/build-script.test.sh
```

---

## Runtime State Inventory

> Not applicable — this is not a rename/refactor phase. Omitted.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | `go test -race` | Yes | 1.26.1 | — |
| bash | `tests/build-script.test.sh` | Yes | 3.2+ | — |
| GitHub Actions runners | CI | Yes (existing workflow) | macos-latest, ubuntu-latest, ubuntu-22.04, windows-latest | — |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

---

## Validation Architecture

> `workflow.nyquist_validation` is not set to `false` in `.planning/config.json` — section included.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` stdlib + bash (for build-script tests) |
| Config file | None (standard `go test`) |
| Quick run command | `go test -race ./...` |
| Full suite command | `go test -race -count=1 ./... && bash tests/build-script.test.sh` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BUILD-01 | build.sh produces `agenthub` single binary, handles GUI/CLI/daemon dispatch | smoke (static) | `bash tests/build-script.test.sh` | Yes (tests/build-script.test.sh) |
| BUILD-02 | CI builds and tests unified binary | integration (CI) | Push to GitHub, observe workflow | N/A — CI workflow file |
| BUILD-03 | All tests pass with `-race` | automated | `go test -race -count=1 ./...` | Yes (all 194 tests exist) |

### Sampling Rate

- **Per task commit:** `go test -race ./...`
- **Per wave merge:** `go test -race -count=1 ./... && bash tests/build-script.test.sh`
- **Phase gate:** Full suite green + CI green before `/gsd:verify-work`

### Wave 0 Gaps

None — existing test infrastructure covers all phase requirements. The only gap is that `tests/build-script.test.sh` has a hardcoded absolute path (pitfall 3) that must be fixed before CI can run it.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Two binaries: `agenthub` + `agenthub-cli` | One binary: `agenthub` | Phase 27+28 | Build produces one artifact; CI validates one binary |
| `go test ./...` without race detector | `go test -race ./...` | Phase 29 (pending) | Race conditions caught in CI, not just locally |
| build-script tests run locally only | build-script tests run in CI | Phase 29 (pending) | Build pipeline is verified by CI on every push |

---

## Open Questions

1. **Does `go test -race` pass on Windows in CI?**
   - What we know: Passes locally on macOS arm64. The CI Windows matrix uses `windows-latest` with native Go toolchain. The wails-build-action sets up MinGW for CGO.
   - What's unclear: Whether `setup-go@v5` alone (without the wails build action completing first) provides a CGO-capable environment for the test step.
   - Recommendation: The test step runs after `setup-go` but before `wails-build-action`. If Windows tests fail due to CGO not being available at that point, add `CGO_ENABLED=0` exclusion only for Windows, or reorder steps. Check the first CI run after the change.

2. **Should the build-script test step fail the entire CI if it fails?**
   - What we know: `set -euo pipefail` is NOT set in the test invocation in CI (it would be set inside the script). The script exits with `exit 1` on failures.
   - What's unclear: Nothing — `bash tests/build-script.test.sh` returning non-zero will fail the CI step as expected.
   - Recommendation: No special configuration needed.

---

## Sources

### Primary (HIGH confidence)
- Direct code inspection: `build.sh` — confirmed no agenthub-cli references, produces `agenthub` only
- Direct code inspection: `.github/workflows/build.yml` — confirmed `go test ./...` (no `-race`), no build-script invocation
- `go test -race -count=1 ./...` — confirmed all 194 tests pass locally (run during research)
- `bash tests/build-script.test.sh` — confirmed 35/35 pass locally (run during research)
- `wails.json` — confirmed `outputfilename: "agenthub"`, single binary name

### Secondary (MEDIUM confidence)
- Go documentation on race detector: https://go.dev/doc/articles/race_detector — `-race` supported on linux/amd64, darwin/amd64, darwin/arm64, windows/amd64

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all tools already in project
- Architecture: HIGH — inspected actual code and CI files directly
- Pitfalls: HIGH — pitfall 3 (hardcoded path) confirmed by reading line 10 of tests/build-script.test.sh
- Open question on Windows -race: MEDIUM — untested in CI, known to work locally

**Research date:** 2026-03-25
**Valid until:** 2026-04-25 (stable toolchain, unlikely to change)
