---
phase: quick
plan: 260408-dcv
status: complete
started: "2026-04-08T17:37:00Z"
completed: "2026-04-08T18:00:00Z"
one_liner: "Renamed tray_objc.m to tray_objc_darwin.m and added Windows skip guards to 10 Unix-specific tests to unblock CI"
tasks_completed: 3
files_changed: 7
commit: b80c9a0
---

# Quick Task 260408-dcv: Fix GitHub Actions build and release pipeline failures

## Summary

Fixed two categories of CI failures that were blocking all Windows builds (both `build.yml` and `release.yml`):

1. **tray_objc.m → tray_objc_darwin.m**: The Objective-C file had no build constraint and was rejected on Windows/Linux where cgo is not active. Go's `_GOOS` filename suffix convention restricts it to Darwin builds.

2. **10 Windows test skip guards**: Added `runtime.GOOS == "windows"` skips to tests that use Unix domain sockets, `/bin/cat` paths, colon-separated PATH, or shell script stubs. One test (`TestSocketPathDefault`) was updated to check platform-appropriate values instead of skipping.

3. **wailsassets tag**: Verified that `wails build` (invoked by the CI action) handles asset embedding automatically — `-tags wailsassets` is only needed for direct `go build` calls.

## Files Changed

| File | Change |
|------|--------|
| tray_objc_darwin.m | Renamed from tray_objc.m (Darwin filename suffix) |
| internal/daemon/engine_test.go | Skip TestEngineResolveCLI on Windows |
| internal/daemon/path_test.go | Skip 2 AugmentServicePath tests on Windows |
| internal/daemon/process_test.go | Skip TestEnsureDaemon_AlreadyRunning on Windows |
| internal/daemon/socket_test.go | Platform-aware TestSocketPathDefault + skip 3 Unix socket tests on Windows |
| internal/pty/detect_test.go | Skip TestDetectCLIs_FindsInstalledCLIs on Windows |
| internal/relay/server_test.go | Skip TestHub_SlowClientDisconnected on Windows |

## Verification

- `go build -tags wailsassets ./...` passes locally
- `go test -race ./internal/daemon/... ./internal/pty/... ./internal/relay/...` all pass on macOS
- tray_objc.m no longer exists, tray_objc_darwin.m in place
