//go:build !windows

package files_test

import "os"

// readFilePlatformSafe reads the file. On non-Windows, os.ReadFile is sufficient
// because rename(2) is atomic and does not require FILE_SHARE_DELETE semantics.
// The rename target can be held open by readers without blocking the rename.
func readFilePlatformSafe(path string) ([]byte, error) {
	return os.ReadFile(path)
}
