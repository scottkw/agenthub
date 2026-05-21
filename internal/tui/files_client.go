package tui

import (
	"context"
	"time"

	"github.com/scottkw/agenthub/internal/files"
)

// FilesClient is the transport-agnostic contract for the TUI Files view.
// Phase 121 originally hard-wired the local *daemon.DaemonClient into the
// loadDir/read/head Cmd factories; Phase 122 introduces this interface so a
// remote-tailnet HTTPS client can stand in for the local Unix-socket client
// without the rest of the TUI knowing the difference.
//
// All four methods mirror the signatures already on *daemon.DaemonClient so
// the existing local-session path keeps working unchanged — DaemonClient
// satisfies FilesClient via duck typing.
//
// Implementations:
//   - *daemon.DaemonClient — local Unix-socket / named-pipe transport
//   - *RemoteFilesClient   — remote Tailscale HTTPS + cap-token transport
type FilesClient interface {
	ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error)
	StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error)
	ReadFile(ctx context.Context, sessionID, relPath string) (data []byte, mime string, err error)
	HeadFile(ctx context.Context, sessionID, relPath string) (size int64, mime string, mtime time.Time, err error)
}
