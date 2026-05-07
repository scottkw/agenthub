package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// TestSaveTerminalSession exercises the four behavioral paths of the
// Phase 97 SER-01 Wails RPC. Uses the saveFileDialogFunc injection
// (function-injection pattern per PROJECT.md "Key Decisions") to mock
// runtime.SaveFileDialog. Mirror of TestUpdateCLIPath table-driven
// pattern at app_test.go lines 234-259.
func TestSaveTerminalSession(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "test.txt")
	invalidPath := filepath.Join(tmpDir, "no-such-dir", "test.txt") // parent does not exist
	sentinelDialogErr := errors.New("dialog setup boom")

	cases := []struct {
		name        string
		mockDialog  func(ctx context.Context, opts wailsruntime.SaveDialogOptions) (string, error)
		content     string
		wantErrSub  string // "" means expect nil error
		wantWritten bool
	}{
		{
			name: "cancel — empty path is silent success",
			mockDialog: func(_ context.Context, _ wailsruntime.SaveDialogOptions) (string, error) {
				return "", nil // user cancelled
			},
			content:     "hello",
			wantErrSub:  "",
			wantWritten: false,
		},
		{
			name: "normal write — file written with content",
			mockDialog: func(_ context.Context, _ wailsruntime.SaveDialogOptions) (string, error) {
				return validPath, nil
			},
			content:     "scrollback contents\nline2\n",
			wantErrSub:  "",
			wantWritten: true,
		},
		{
			name: "write IO error — wrapped error",
			mockDialog: func(_ context.Context, _ wailsruntime.SaveDialogOptions) (string, error) {
				return invalidPath, nil // parent dir doesn't exist; WriteFile fails
			},
			content:    "ignored",
			wantErrSub: "SaveTerminalSession: write:",
		},
		{
			name: "dialog setup error — wrapped error",
			mockDialog: func(_ context.Context, _ wailsruntime.SaveDialogOptions) (string, error) {
				return "", sentinelDialogErr
			},
			content:    "ignored",
			wantErrSub: "SaveTerminalSession: dialog:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up any prior file from the same fixture path so
			// wantWritten=false assertions are reliable.
			_ = os.Remove(validPath)

			a := &App{
				ctx:                context.Background(),
				saveFileDialogFunc: tc.mockDialog,
			}
			err := a.SaveTerminalSession("", "default.txt", tc.content)

			if tc.wantErrSub == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrSub)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErrSub, err.Error())
				}
			}

			if tc.wantWritten {
				data, readErr := os.ReadFile(validPath)
				if readErr != nil {
					t.Fatalf("expected file at %s, got read error: %v", validPath, readErr)
				}
				if string(data) != tc.content {
					t.Errorf("file content mismatch: want %q, got %q", tc.content, string(data))
				}
			} else {
				if _, statErr := os.Stat(validPath); !os.IsNotExist(statErr) {
					t.Errorf("expected no file at %s for case %q, but stat returned %v", validPath, tc.name, statErr)
				}
			}
		})
	}
}
