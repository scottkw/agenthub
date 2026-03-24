//go:build !windows

package daemon

// cleanupStaleWindowsPipe is never called on non-Windows platforms because
// isWindowsNamedPipe returns false for all Unix socket paths. This stub
// satisfies the compiler.
func cleanupStaleWindowsPipe(path string) error {
	panic("cleanupStaleWindowsPipe called on non-Windows platform")
}
