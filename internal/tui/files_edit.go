package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
)

// resolveEditor returns the LookPath-resolved path for the first non-empty
// candidate in the locked order: $EDITOR → $VISUAL → nano → vim → vi.
// Returns "" when none of the candidates resolves on PATH.
//
// Locked order per TUIW-03 / CONTEXT.md: $EDITOR wins, then $VISUAL, then
// fallbacks. This order is intentional even though many tools check $VISUAL
// first — honor the project decision.
func resolveEditor() string {
	for _, cand := range []string{os.Getenv("EDITOR"), os.Getenv("VISUAL"), "nano", "vim", "vi"} {
		if cand == "" {
			continue
		}
		if p, err := exec.LookPath(cand); err == nil {
			return p
		}
	}
	return ""
}

// filesEditReadyMsg is returned by editFetchCmd once the file bytes have been
// read from the daemon and written to a host-local temp file. The editor
// binary path is carried through so the Update handler can spawn it without
// calling resolveEditor() a second time.
type filesEditReadyMsg struct {
	sessionID  string
	generation uint64
	tmpPath    string // host-local temp file path (os.CreateTemp result)
	relPath    string // sandbox-relative path (for write-back)
	editor     string // resolved editor binary path
	err        error
}

// editorExitMsg is the callback payload produced by tea.ExecProcess when the
// editor process exits. The Update handler uses this to trigger the write-back
// and listing refresh.
type editorExitMsg struct {
	sessionID  string
	generation uint64
	tmpPath    string // same temp file written to the editor
	relPath    string // sandbox-relative path (for write-back to the correct location)
	exitErr    error  // non-nil if the editor process exited non-zero
}

// filesOpMsg is the result envelope for write operations: edit write-back,
// delete, rename, mkdir. The op field names the operation for toast text.
// Stale messages (generation < current) are discarded in the Update handler.
type filesOpMsg struct {
	sessionID  string
	generation uint64
	op         string // "edit", "delete", "rename", "mkdir"
	err        error
}

// editFetchCmd returns a tea.Cmd that reads the target file from the daemon
// and writes the bytes to a host-local temp file, then returns a
// filesEditReadyMsg. All filesystem I/O (CreateTemp, Write, Close) is inside
// the closure so it never runs synchronously in Update (T-126-07 / TUIW-07).
//
// Nil-client guard mirrors loadDirCmd/readFileCmd (files_cmds.go).
// Temp file extension matches relPath for editor syntax highlighting.
func editFetchCmd(client FilesClient, sid, relPath, editor string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		if isNilFilesClient(client) {
			return filesEditReadyMsg{sessionID: sid, generation: gen, relPath: relPath, editor: editor, err: errNilClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		data, _, err := client.ReadFile(ctx, sid, relPath)
		if err != nil {
			return filesEditReadyMsg{sessionID: sid, generation: gen, relPath: relPath, editor: editor, err: err}
		}

		ext := filepath.Ext(relPath)
		tmp, err := os.CreateTemp("", "agenthub-edit-*"+ext)
		if err != nil {
			return filesEditReadyMsg{sessionID: sid, generation: gen, relPath: relPath, editor: editor, err: err}
		}
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return filesEditReadyMsg{sessionID: sid, generation: gen, relPath: relPath, editor: editor, err: err}
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmp.Name())
			return filesEditReadyMsg{sessionID: sid, generation: gen, relPath: relPath, editor: editor, err: err}
		}
		return filesEditReadyMsg{
			sessionID:  sid,
			generation: gen,
			tmpPath:    tmp.Name(),
			relPath:    relPath,
			editor:     editor,
		}
	}
}

// editWriteBackCmd returns a tea.Cmd that reads the (potentially edited)
// temp file and writes the bytes back to the daemon via client.WriteFile.
// The temp file is always removed (defer os.Remove) regardless of outcome,
// satisfying T-126-05 (no lingering sensitive bytes in /tmp).
//
// The write-back is UNCONDITIONAL — it is dispatched regardless of the editor's
// exit code (the caller, editorExitMsg handler, batches this cmd always).
// Stale-guard via generation on the result msg.
func editWriteBackCmd(client FilesClient, sid, relPath, tmpPath string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		defer os.Remove(tmpPath) // T-126-05: always clean up the temp file

		data, rerr := os.ReadFile(tmpPath)
		if rerr != nil {
			return filesOpMsg{sessionID: sid, generation: gen, op: "edit", err: rerr}
		}
		if isNilFilesClient(client) {
			return filesOpMsg{sessionID: sid, generation: gen, op: "edit", err: errNilClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, werr := client.WriteFile(ctx, sid, relPath, data)
		return filesOpMsg{sessionID: sid, generation: gen, op: "edit", err: werr}
	}
}
