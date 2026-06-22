//go:build windows

package files_test

import (
	"io"
	"os"
	"syscall"
)

// readFilePlatformSafe reads path with FILE_SHARE_DELETE so WriteFileAtomic's
// POSIX-semantics rename (via windows.Renameat + FILE_RENAME_POSIX_SEMANTICS)
// can replace the file while this read handle is open.
//
// On Windows, os.ReadFile opens with FILE_SHARE_READ|FILE_SHARE_WRITE but NOT
// FILE_SHARE_DELETE (syscall/syscall_windows.go:395). This means
// NtSetInformationFile with FILE_RENAME_POSIX_SEMANTICS fails when the
// destination file is held open by the reader, causing writeAtomicRename to
// exhaust its retry loop and return an error. By opening with
// FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE, we allow the atomic
// POSIX-semantics rename to succeed while a concurrent read handle is open.
func readFilePlatformSafe(path string) ([]byte, error) {
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(
		pathp,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(h), path)
	defer f.Close()
	return io.ReadAll(f)
}
