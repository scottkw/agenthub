//go:build !windows

package daemon

// platformExtraBins returns nil on non-Windows platforms. All non-Windows
// paths are already included directly in the candidates list in path.go.
func platformExtraBins() []string {
	return nil
}
