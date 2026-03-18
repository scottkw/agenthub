package pty

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSpawnPTY(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess, err := b.Create(ctx, CreateRequest{CLI: "cat", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer b.Kill(sess.ID) //nolint:errcheck

	if sess.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if sess.State != StateRunning {
		t.Errorf("expected StateRunning, got %v", sess.State)
	}
}

func TestSpawnPTY_RealPTY(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess, err := b.Create(ctx, CreateRequest{CLI: "cat", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer b.Kill(sess.ID) //nolint:errcheck

	// cat echoes input — write "hello\n" and read it back
	if _, err := sess.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read back with timeout via goroutine
	type result struct {
		data string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		buf := make([]byte, 256)
		var sb strings.Builder
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			n, err := sess.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if strings.Contains(sb.String(), "hello") {
				ch <- result{data: sb.String()}
				return
			}
			if err != nil {
				ch <- result{data: sb.String(), err: err}
				return
			}
		}
		ch <- result{data: sb.String()}
	}()

	select {
	case r := <-ch:
		if !strings.Contains(r.data, "hello") {
			t.Errorf("expected 'hello' in PTY output, got: %q (err: %v)", r.data, r.err)
		}
	case <-time.After(4 * time.Second):
		t.Error("timed out waiting for PTY output")
	}
}

func TestSpawnPTY_EnvSet(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess, err := b.Create(ctx, CreateRequest{CLI: "env", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer b.Kill(sess.ID) //nolint:errcheck

	// env prints all environment variables and exits — collect all output until
	// EOF or we find what we need.
	type result struct {
		data string
	}
	ch := make(chan result, 1)
	go func() {
		buf := make([]byte, 4096)
		var sb strings.Builder
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			n, err := sess.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
				// Found it — no need to read more.
				if strings.Contains(sb.String(), "TERM=xterm-256color") {
					ch <- result{data: sb.String()}
					return
				}
			}
			if err != nil {
				// EOF or error — stop reading.
				ch <- result{data: sb.String()}
				return
			}
		}
		ch <- result{data: sb.String()}
	}()

	select {
	case r := <-ch:
		if !strings.Contains(r.data, "TERM=xterm-256color") {
			t.Errorf("expected TERM=xterm-256color in env output, got: %q", r.data)
		}
	case <-time.After(4 * time.Second):
		t.Error("timed out waiting for env output")
	}
}

func TestResize(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess, err := b.Create(ctx, CreateRequest{CLI: "cat", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer b.Kill(sess.ID) //nolint:errcheck

	if err := b.Resize(sess.ID, 120, 40); err != nil {
		t.Errorf("Resize failed: %v", err)
	}
}

func TestResize_NotFound(t *testing.T) {
	b := NewNativePTYBackend()
	err := b.Resize("nonexistent", 80, 24)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess1, err := b.Create(ctx, CreateRequest{CLI: "cat", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create 1 failed: %v", err)
	}
	defer b.Kill(sess1.ID) //nolint:errcheck

	sess2, err := b.Create(ctx, CreateRequest{CLI: "cat", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create 2 failed: %v", err)
	}
	defer b.Kill(sess2.ID) //nolint:errcheck

	list := b.List()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestKillClean(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess, err := b.Create(ctx, CreateRequest{CLI: "cat", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := b.Kill(sess.ID); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// After Kill the session state must be StateStopped.
	if sess.State != StateStopped {
		t.Errorf("expected StateStopped after Kill, got %v", sess.State)
	}

	// After Kill the session must not appear in List.
	list := b.List()
	for _, s := range list {
		if s.ID == sess.ID {
			t.Errorf("killed session %s still appears in List", sess.ID)
		}
	}
}

func TestKillClean_NoOrphans(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Spawn a shell that starts a background sleep — killing the session must
	// also terminate the sleep (no orphan process).
	// We use a short sleep so even if cleanup fails the test doesn't hang long.
	sess, err := b.Create(ctx, CreateRequest{
		CLI:  "sh",
		Args: []string{"-c", "sleep 300 & cat"},
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Let the shell start the sleep process.
	time.Sleep(200 * time.Millisecond)

	if err := b.Kill(sess.ID); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// Give the OS a moment to reap.
	time.Sleep(300 * time.Millisecond)

	// The spawned process group should be gone. We verify by checking that
	// sending signal 0 to the (negated) pgid returns an error — meaning the
	// process group no longer exists.
	// pgid == pid because go-pty uses Setsid.
	if sess.cmd != nil && sess.cmd.Process != nil {
		pgid := sess.cmd.Process.Pid
		err := killSignal0(-pgid)
		if err == nil {
			t.Errorf("process group %d still exists after Kill — orphan processes", pgid)
		}
	}
}

func TestKillClean_NotFound(t *testing.T) {
	b := NewNativePTYBackend()
	err := b.Kill("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}
