package tui

import (
	"context"
	"time"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/files"
)

// FilesClient is the transport-agnostic contract for the TUI Files view.
// Phase 121 originally hard-wired the local *daemon.DaemonClient into the
// loadDir/read/head Cmd factories; Phase 122 introduces this interface so a
// remote-tailnet HTTPS client can stand in for the local Unix-socket client
// without the rest of the TUI knowing the difference.
//
// Phase 126 extends the interface from 4 read methods to 8 (4 read + 4 write)
// so one pipeline drives local (*daemon.DaemonClient) AND remote
// (*tui.RemoteFilesClient) write transport. (TUIW-01)
//
// All eight methods mirror the signatures already on *daemon.DaemonClient so
// the existing local-session path keeps working unchanged — DaemonClient
// satisfies FilesClient via duck typing.
//
// Implementations:
//   - *daemon.DaemonClient — local Unix-socket / named-pipe transport
//   - *RemoteFilesClient   — remote Tailscale HTTPS + cap-token transport
type FilesClient interface {
	// Read methods (Phase 122 — DO NOT CHANGE signatures).
	ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error)
	StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error)
	ReadFile(ctx context.Context, sessionID, relPath string) (data []byte, mime string, err error)
	HeadFile(ctx context.Context, sessionID, relPath string) (size int64, mime string, mtime time.Time, err error)
	// Write methods (Phase 126 / TUIW-01 — return types MUST match DaemonClient
	// signatures exactly: response structs, NOT error-only).
	WriteFile(ctx context.Context, sessionID, relPath string, data []byte) (files.FileWriteResponse, error)
	DeleteFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error)
	RenameFile(ctx context.Context, sessionID, oldRel, newRel string) (files.FileOpResponse, error)
	MkdirFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error)
}

// Compile-time guard: *daemon.DaemonClient satisfies FilesClient (TUIW-01).
// This assertion fails the build if any signature in the interface diverges
// from the live DaemonClient write methods. Cheapest TUIW-01 enforcement.
var _ FilesClient = (*daemon.DaemonClient)(nil)
