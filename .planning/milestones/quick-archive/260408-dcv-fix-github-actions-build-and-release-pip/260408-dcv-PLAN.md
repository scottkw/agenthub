---
phase: quick
plan: 260408-dcv
type: execute
wave: 1
depends_on: []
files_modified:
  - tray_objc_darwin.m  # renamed from tray_objc.m
  - internal/daemon/engine_test.go
  - internal/daemon/path_test.go
  - internal/daemon/process_test.go
  - internal/daemon/socket_test.go
  - internal/pty/detect_test.go
  - internal/relay/server_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - "Windows and Linux CI builds succeed (no Objective-C source file error)"
    - "All Go tests pass on all platforms (darwin, linux, windows)"
    - "macOS builds still work correctly with renamed tray_objc_darwin.m"
  artifacts:
    - path: "tray_objc_darwin.m"
      provides: "Objective-C tray code restricted to Darwin via filename suffix"
    - path: "internal/daemon/engine_test.go"
      provides: "TestEngineResolveCLI skipped on Windows"
    - path: "internal/daemon/socket_test.go"
      provides: "Unix socket tests skipped on Windows"
  key_links:
    - from: "tray_objc_darwin.m"
      to: "tray.go"
      via: "Go build system _GOOS filename convention"
      pattern: "_darwin\\.m$"
---

<objective>
Fix GitHub Actions build and release pipeline failures across three categories: (1) Objective-C file breaking non-Darwin builds, (2) Unix-specific tests failing on Windows, (3) verify wailsassets tag handling.

Purpose: Unblock CI/CD pipeline so builds and releases work on all target platforms.
Output: Renamed tray file, platform-guarded tests, working cross-platform CI.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.github/workflows/build.yml
@.github/workflows/release.yml
@tray_objc.m
@tray.go
@internal/daemon/engine_test.go
@internal/daemon/path_test.go
@internal/daemon/process_test.go
@internal/daemon/socket_test.go
@internal/pty/detect_test.go
@internal/relay/server_test.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix tray_objc.m breaking non-Darwin builds</name>
  <files>tray_objc.m, tray_objc_darwin.m</files>
  <action>
    Rename `tray_objc.m` to `tray_objc_darwin.m` using `git mv`. Go's build system respects `_GOOS` filename suffixes for all source files including `.m` files. This ensures the Objective-C file is only included when GOOS=darwin, where cgo is activated by `tray.go`'s `import "C"`.

    After renaming, verify that `tray.go` (which has `//go:build darwin`) does NOT reference `tray_objc.m` by name anywhere — it uses cgo linkage, so the rename is transparent.

    Do NOT add a build constraint comment to the file — the `_darwin` suffix is sufficient and is the idiomatic Go approach for platform-specific non-Go source files.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && test -f tray_objc_darwin.m && test ! -f tray_objc.m && go build -tags wailsassets ./...</automated>
  </verify>
  <done>tray_objc.m renamed to tray_objc_darwin.m. Local `go build` succeeds. The file no longer exists at old path.</done>
</task>

<task type="auto">
  <name>Task 2: Add Windows skip guards to Unix-specific tests</name>
  <files>
    internal/daemon/engine_test.go,
    internal/daemon/path_test.go,
    internal/daemon/process_test.go,
    internal/daemon/socket_test.go,
    internal/pty/detect_test.go,
    internal/relay/server_test.go
  </files>
  <action>
    Add platform guards to tests that use Unix-specific infrastructure. Use `runtime.GOOS` checks (not build constraints) so the test files still compile on all platforms — they just skip execution on Windows.

    For each test, add at the top of the test function body:

    **internal/daemon/engine_test.go — TestEngineResolveCLI:**
    Uses `/bin/cat` as a mock CLI path. Add: `if runtime.GOOS == "windows" { t.Skip("uses Unix path /bin/cat") }`

    **internal/daemon/path_test.go — TestAugmentServicePath_AddsExistingDirs and TestAugmentServicePath_PrependsNotAppends:**
    Uses `:` PATH separator and Unix paths. Add: `if runtime.GOOS == "windows" { t.Skip("uses Unix PATH separator") }`

    **internal/daemon/process_test.go — TestEnsureDaemon_AlreadyRunning:**
    Uses Unix domain sockets. Add: `if runtime.GOOS == "windows" { t.Skip("uses Unix domain sockets") }`

    **internal/daemon/socket_test.go — TestSocketPathDefault:**
    Expects `.sock` suffix. On Windows the default is a named pipe, not a socket path. Add: `if runtime.GOOS == "windows" { t.Skip("Unix socket path test") }`

    **internal/daemon/socket_test.go — TestSocketPathLength:**
    Tests Unix socket 108-char limit. Add: `if runtime.GOOS == "windows" { t.Skip("Unix socket length limit") }`

    **internal/daemon/socket_test.go — TestCleanupStaleSocket_StaleFile and TestCleanupStaleSocket_ActiveSocket:**
    Use Unix domain sockets directly. Add: `if runtime.GOOS == "windows" { t.Skip("uses Unix domain sockets") }`

    **internal/pty/detect_test.go — TestDetectCLIs_FindsInstalledCLIs:**
    Creates shell script stubs with `#!/bin/sh`. Add: `if runtime.GOOS == "windows" { t.Skip("uses shell script stubs") }`

    **internal/relay/server_test.go — TestHub_SlowClientDisconnected:**
    Has timing sensitivity on Windows. Increase the write deadline timeout if on Windows by using a larger value (e.g., 2x). Alternatively, if it uses a hardcoded timeout, add `if runtime.GOOS == "windows" { t.Skip("timing-sensitive on Windows CI") }` as the simplest reliable fix.

    Ensure `"runtime"` is in the import block of each modified file (add if missing).
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/daemon/... ./internal/pty/... ./internal/relay/... -count=1 -v 2>&1 | tail -30</automated>
  </verify>
  <done>All modified tests pass on macOS. Windows-incompatible tests have runtime.GOOS skip guards. No test logic changed for non-Windows platforms.</done>
</task>

<task type="auto">
  <name>Task 3: Verify wailsassets tag handling in CI workflows</name>
  <files>.github/workflows/build.yml, .github/workflows/release.yml</files>
  <action>
    Investigate whether the `wails-build-action` automatically adds `-tags wailsassets` or if it needs to be explicit.

    Check by:
    1. Read the wails-build-action source (check the action.yml or entrypoint) to see if it sets wailsassets automatically
    2. Check if the Wails CLI `wails build` command adds it by default

    The current build.yml passes `build-flags` only for webkit2_41 on Linux — no wailsassets tag. The release.yml has no build-flags at all for macOS and Windows jobs.

    If the wails-build-action does NOT automatically add wailsassets:
    - In build.yml: update `build-flags` to include `wailsassets` for all platforms. For the Linux webkit2_41 variant, combine: `-tags webkit2_41,wailsassets`. For others: `-tags wailsassets`.
    - In release.yml: add `build-flags: -tags wailsassets` to the macOS, Windows, and Linux build steps (Linux already has `-tags webkit2_41`, change to `-tags webkit2_41,wailsassets`).

    If the wails-build-action DOES handle it automatically (e.g., Wails CLI bundles assets and sets the tag), document this finding but make no changes.

    IMPORTANT: Per project memory, production builds REQUIRE `-tags wailsassets` for correct MIME types via embed.FS. This must be verified.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && grep -n "wailsassets\|build-flags" .github/workflows/build.yml .github/workflows/release.yml</automated>
  </verify>
  <done>Wailsassets tag handling verified. Either workflows updated to include the tag, or confirmed that wails-build-action handles it automatically (with documented evidence).</done>
</task>

</tasks>

<verification>
1. `go build -tags wailsassets ./...` succeeds locally (no .m file error)
2. `go test -race ./...` passes locally on macOS
3. `tray_objc.m` no longer exists, `tray_objc_darwin.m` exists
4. All modified test files compile and pass
5. Workflow files have correct wailsassets handling
</verification>

<success_criteria>
- tray_objc.m renamed to tray_objc_darwin.m (unblocks Windows/Linux builds)
- 10 Unix-specific tests have Windows skip guards (unblocks Windows test step)
- wailsassets tag handling verified/fixed in both build.yml and release.yml
- All existing tests still pass on macOS
</success_criteria>

<output>
After completion, create `.planning/quick/260408-dcv-fix-github-actions-build-and-release-pip/260408-dcv-SUMMARY.md`
</output>
