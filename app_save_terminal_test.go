package main

import "testing"

// TestSaveTerminalSession — Plan 97-05 implements.
//
// Table-driven tests for (*App).SaveTerminalSession covering:
//
//  1. Cancel path — saveFileDialogFunc returns ("", nil); assert no file
//     written and method returns nil (silent success per
//     OpenFileDialog precedent at app.go lines 815-829).
//
//  2. Normal write — saveFileDialogFunc returns (tempdir/path, nil);
//     assert file written with content matching the supplied bytes,
//     file mode 0o644.
//
//  3. Write IO error — saveFileDialogFunc returns a path under a
//     non-existent parent dir; assert returned error wraps with
//     "SaveTerminalSession: write:" prefix.
//
//  4. Dialog setup error — saveFileDialogFunc returns ("", err);
//     assert returned error wraps with "SaveTerminalSession: dialog:"
//     prefix.
//
// Plan 97-05 introduces a saveFileDialogFunc field on *App (function-
// injection pattern matching serviceControlFunc / statusFunc precedent
// per PROJECT.md "Key Decisions") so the test can mock the Wails runtime
// free function. See 97-PATTERNS.md §"Function-injection for testability".
func TestSaveTerminalSession(t *testing.T) {
	t.Skip("Pending until Plan 97-05 implements (*App).SaveTerminalSession + saveFileDialogFunc injection (97-VALIDATION row SER-01 Wails RPC writes file via SaveFileDialog).")
}
