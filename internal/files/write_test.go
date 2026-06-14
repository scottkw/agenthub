// Package files_test — write-primitive unit tests for Sandbox write methods.
// Covers FSW-01 (atomic write / concurrent-reader safety), FSW-02 (rename
// both-path validation), FSW-03 (mkdir/mkdirall confinement), FSW-04
// (delete confinement), and FSW-06 (shell-RC denylist in a $HOME-rooted
// sandbox).
//
// All tests use files.NewSandbox(t.TempDir()) so no test ever touches the
// real filesystem outside the tmpdir. Denylist tests set the HOME /
// USERPROFILE env vars to a temp dir and root the sandbox there.
package files_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/scottkw/agenthub/internal/files"
)

// --------------------------------------------------------------------------
// TestWriteFileAtomic — FSW-01: write content and verify round-trip.
// --------------------------------------------------------------------------

// TestWriteFileAtomic writes "hello, world" to "a.txt", reads it back via
// os.ReadFile, and asserts the bytes are identical. It also verifies no
// leftover ".agenthub-tmp-*" sibling remains after a successful write.
func TestWriteFileAtomic(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	content := []byte("hello, world")
	if err := sb.WriteFileAtomic("a.txt", content); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, content)
	}

	// No leftover .agenthub-tmp-* sibling must remain.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".agenthub-tmp-") {
			t.Errorf("leftover temp file after success: %q", e.Name())
		}
	}
}

// TestWriteFileAtomic_Overwrite verifies that overwriting an existing file
// replaces the full content and leaves no temp sibling.
func TestWriteFileAtomic_Overwrite(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	first := []byte("first version of the file content")
	second := []byte("second")

	if err := sb.WriteFileAtomic("b.txt", first); err != nil {
		t.Fatalf("WriteFileAtomic (first): %v", err)
	}
	if err := sb.WriteFileAtomic("b.txt", second); err != nil {
		t.Fatalf("WriteFileAtomic (second): %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "b.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, second) {
		t.Errorf("overwrite mismatch: got %q, want %q", got, second)
	}
}

// TestWriteFileAtomic_Subdir verifies writes into nested subdirectories succeed.
func TestWriteFileAtomic_Subdir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub", "deep"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	content := []byte("nested write")
	if err := sb.WriteFileAtomic("sub/deep/c.txt", content); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "sub", "deep", "c.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("subdir write mismatch: got %q, want %q", got, content)
	}
}

// TestWriteFileAtomic_TraversalRejected verifies that traversal payloads are
// rejected before any I/O occurs.
func TestWriteFileAtomic_TraversalRejected(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	cases := []string{
		"../escape.txt",
		"../../escape.txt",
		"/etc/passwd",
	}
	for _, path := range cases {
		if err := sb.WriteFileAtomic(path, []byte("x")); err == nil {
			t.Errorf("WriteFileAtomic(%q) returned nil; want rejection", path)
		}
	}
}

// TestWriteFileAtomic_ConcurrentReadNeverPartial — FSW-01 concurrency
// invariant: a goroutine repeatedly reading the target file must never
// observe an empty or partial file while concurrent writers overwrite it.
func TestWriteFileAtomic_ConcurrentReadNeverPartial(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Seed the file with known initial content.
	initial := bytes.Repeat([]byte("A"), 4096)
	updated := bytes.Repeat([]byte("B"), 4096)

	if err := sb.WriteFileAtomic("concurrent.txt", initial); err != nil {
		t.Fatalf("seed: %v", err)
	}

	targetPath := filepath.Join(root, "concurrent.txt")
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Reader goroutine: read repeatedly and assert full content.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			data, err := os.ReadFile(targetPath)
			if err != nil {
				// File may be transiently absent during rename on some OSes;
				// that is acceptable — we assert non-empty, not "must exist".
				continue
			}
			if len(data) == 0 {
				t.Errorf("observed empty file (O_TRUNC partial write)")
				return
			}
			if len(data) != len(initial) {
				t.Errorf("observed partial file: len=%d, want %d", len(data), len(initial))
				return
			}
			// Must be entirely 'A' or entirely 'B' — never mixed.
			first := data[0]
			for _, b := range data {
				if b != first {
					t.Errorf("observed mixed-content file (non-atomic write)")
					return
				}
			}
		}
	}()

	// Writer goroutine: alternate between initial and updated.
	for i := 0; i < 200; i++ {
		content := initial
		if i%2 == 1 {
			content = updated
		}
		if err := sb.WriteFileAtomic("concurrent.txt", content); err != nil {
			t.Errorf("WriteFileAtomic iteration %d: %v", i, err)
		}
	}

	close(stop)
	wg.Wait()
}

// --------------------------------------------------------------------------
// TestRename — FSW-02
// --------------------------------------------------------------------------

// TestRename_SameDir renames a file within the sandbox root and confirms the
// destination exists with the original content.
func TestRename_SameDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("rename-me"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if err := sb.Rename("old.txt", "new.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	// Destination must exist with the original content.
	got, err := os.ReadFile(filepath.Join(root, "new.txt"))
	if err != nil {
		t.Fatalf("ReadFile new.txt: %v", err)
	}
	if string(got) != "rename-me" {
		t.Errorf("renamed content = %q; want %q", got, "rename-me")
	}

	// Source must no longer exist.
	if _, err := os.Stat(filepath.Join(root, "old.txt")); !os.IsNotExist(err) {
		t.Errorf("source still exists after rename")
	}
}

// TestRename_CrossDirMove moves a file from one subdirectory to another within
// the sandbox root — a move is a rename with a different parent.
func TestRename_CrossDirMove(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dst"), 0o755); err != nil {
		t.Fatalf("mkdir dst: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "f.txt"), []byte("move-me"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if err := sb.Rename("src/f.txt", "dst/f.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "dst", "f.txt"))
	if err != nil {
		t.Fatalf("ReadFile dst/f.txt: %v", err)
	}
	if string(got) != "move-me" {
		t.Errorf("moved content = %q; want %q", got, "move-me")
	}
}

// TestRename_DestinationTraversalRejected — Pitfall 1: validateAndClean must
// be called on the DESTINATION path, not just the source. A traversal payload
// as the destination must be rejected before any rename occurs.
func TestRename_DestinationTraversalRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Destination traversal payloads must be rejected.
	cases := []struct {
		src string
		dst string
	}{
		{"a.txt", "../../.ssh/authorized_keys"},
		{"a.txt", "../outside.txt"},
		{"a.txt", "/etc/crontab"},
	}
	for _, tc := range cases {
		if err := sb.Rename(tc.src, tc.dst); err == nil {
			t.Errorf("Rename(%q, %q) returned nil; want traversal rejection", tc.src, tc.dst)
		}
	}
}

// TestRename_SourceTraversalRejected verifies that a traversal payload in the
// source path is also rejected.
func TestRename_SourceTraversalRejected(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if err := sb.Rename("../../etc/passwd", "safe.txt"); err == nil {
		t.Errorf("Rename with traversal source returned nil; want rejection")
	}
}

// --------------------------------------------------------------------------
// TestMkdir / TestMkdirAll — FSW-03
// --------------------------------------------------------------------------

// TestMkdir creates a single directory within the sandbox root and verifies it
// exists as a directory.
func TestMkdir(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if err := sb.Mkdir("newdir"); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	fi, err := os.Stat(filepath.Join(root, "newdir"))
	if err != nil {
		t.Fatalf("Stat newdir: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("Mkdir result is not a directory")
	}
}

// TestMkdirAll creates a nested directory hierarchy within the sandbox root.
func TestMkdirAll(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if err := sb.MkdirAll("a/b/c"); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	fi, err := os.Stat(filepath.Join(root, "a", "b", "c"))
	if err != nil {
		t.Fatalf("Stat a/b/c: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("MkdirAll result is not a directory")
	}
}

// TestMkdir_TraversalRejected verifies that traversal payloads are rejected.
func TestMkdir_TraversalRejected(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	cases := []string{
		"../outside",
		"../../etc/cron.d",
		"/tmp/escape",
	}
	for _, path := range cases {
		if err := sb.Mkdir(path); err == nil {
			t.Errorf("Mkdir(%q) returned nil; want rejection", path)
		}
		if err := sb.MkdirAll(path); err == nil {
			t.Errorf("MkdirAll(%q) returned nil; want rejection", path)
		}
	}
}

// --------------------------------------------------------------------------
// TestDelete — FSW-04
// --------------------------------------------------------------------------

// TestDelete_File removes a single file within the sandbox root.
func TestDelete_File(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "todelete.txt"), []byte("gone"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if err := sb.Delete("todelete.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "todelete.txt")); !os.IsNotExist(err) {
		t.Errorf("file still exists after Delete")
	}
}

// TestDelete_RecursiveSubtree removes a non-empty directory subtree.
func TestDelete_RecursiveSubtree(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "tree", "leaf")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed x.txt: %v", err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if err := sb.Delete("tree"); err != nil {
		t.Fatalf("Delete tree: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "tree")); !os.IsNotExist(err) {
		t.Errorf("subtree still exists after Delete")
	}
}

// TestDelete_TraversalRejected verifies traversal payloads are rejected.
func TestDelete_TraversalRejected(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	cases := []string{
		"../outside",
		"../../etc/cron.d",
		"/etc/passwd",
	}
	for _, path := range cases {
		if err := sb.Delete(path); err == nil {
			t.Errorf("Delete(%q) returned nil; want rejection", path)
		}
	}
}

// --------------------------------------------------------------------------
// TestDenylist — FSW-06
// --------------------------------------------------------------------------

// setHomeEnv sets HOME (and USERPROFILE on Windows) to dir for the duration
// of the test, restoring the original values via t.Cleanup.
func setHomeEnv(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}

// TestDenylist_HomeRooted verifies that WriteFileAtomic, Rename, Mkdir, and
// Delete all return ErrProtectedSystemFile when targeting a protected
// dotfile/dotdir in a sandbox rooted at the (mock) $HOME directory.
func TestDenylist_HomeRooted(t *testing.T) {
	// Use a temp dir as the fake $HOME so the real $HOME is never touched.
	fakeHome := t.TempDir()
	setHomeEnv(t, fakeHome)

	// Create any prerequisite paths so renames and deletes can proceed past
	// the "file not found" stage and hit the denylist check.
	if err := os.MkdirAll(filepath.Join(fakeHome, ".ssh"), 0o700); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeHome, ".bashrc"), []byte("# shell rc"), 0o644); err != nil {
		t.Fatalf("seed .bashrc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeHome, ".ssh", "authorized_keys"), []byte("ssh-rsa AAAA"), 0o600); err != nil {
		t.Fatalf("seed authorized_keys: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(fakeHome, ".claude"), 0o700); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}

	// Root the sandbox AT $HOME (the dangerous case).
	sb, err := files.NewSandbox(fakeHome)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Targets to test across all four write methods.
	denyTargets := []string{
		".bashrc",
		".zshrc",
		".profile",
		".bash_profile",
		".zprofile",
		".zshenv",
		".bash_login",
		".ssh/authorized_keys",
		".claude/CLAUDE.md",
	}

	for _, target := range denyTargets {
		t.Run("WriteFileAtomic/"+target, func(t *testing.T) {
			err := sb.WriteFileAtomic(target, []byte("pwned"))
			if err == nil {
				t.Errorf("WriteFileAtomic(%q) returned nil; want ErrProtectedSystemFile", target)
				return
			}
			if !isProtected(err) {
				t.Errorf("WriteFileAtomic(%q) error = %v; want ErrProtectedSystemFile", target, err)
			}
		})

		t.Run("Mkdir/"+target, func(t *testing.T) {
			// .ssh, .claude are directories so use a child path.
			dirTarget := target
			if !strings.HasSuffix(target, "/") && !strings.ContainsRune(target, '/') {
				// RC files are flat — try to create a dir with the same name.
				// The denylist must fire before mkdir is attempted.
				dirTarget = target
			}
			err := sb.Mkdir(dirTarget)
			if err == nil {
				t.Errorf("Mkdir(%q) returned nil; want ErrProtectedSystemFile", dirTarget)
				return
			}
			if !isProtected(err) {
				t.Errorf("Mkdir(%q) error = %v; want ErrProtectedSystemFile", dirTarget, err)
			}
		})

		t.Run("Delete/"+target, func(t *testing.T) {
			err := sb.Delete(target)
			if err == nil {
				t.Errorf("Delete(%q) returned nil; want ErrProtectedSystemFile", target)
				return
			}
			if !isProtected(err) {
				t.Errorf("Delete(%q) error = %v; want ErrProtectedSystemFile", target, err)
			}
		})
	}

	// Rename: rename INTO a protected target path must be rejected.
	srcPath := "safe-source.txt"
	if err := os.WriteFile(filepath.Join(fakeHome, srcPath), []byte("src"), 0o644); err != nil {
		t.Fatalf("seed safe-source.txt: %v", err)
	}
	for _, target := range denyTargets {
		t.Run("Rename/into/"+target, func(t *testing.T) {
			err := sb.Rename(srcPath, target)
			if err == nil {
				t.Errorf("Rename(safe, %q) returned nil; want ErrProtectedSystemFile", target)
				return
			}
			if !isProtected(err) {
				t.Errorf("Rename(safe, %q) error = %v; want ErrProtectedSystemFile", target, err)
			}
		})
	}
}

// TestDenylist_NonHomeRootedUnaffected verifies that a sandbox rooted outside
// $HOME is NOT affected by the denylist — a file literally named ".bashrc"
// inside the sandbox must be writable.
func TestDenylist_NonHomeRootedUnaffected(t *testing.T) {
	// The sandbox root is in a tempdir that is NOT under $HOME.
	// t.TempDir() on macOS returns /var/folders/... which is not $HOME.
	// Even if $HOME is somehow set to a broad path, we pick a unique subdir.
	root := t.TempDir()

	// Ensure the sandbox root is definitely not under $HOME.
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(root, home) {
		t.Skipf("tmpdir %q is under $HOME %q; cannot run non-home test", root, home)
	}

	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Writing ".bashrc" in a non-home sandbox must succeed.
	if err := sb.WriteFileAtomic(".bashrc", []byte("# rc in non-home sandbox")); err != nil {
		t.Errorf("WriteFileAtomic(.bashrc) in non-home sandbox: %v (want nil)", err)
	}
}

// isProtected returns true if err is (or wraps) files.ErrProtectedSystemFile.
func isProtected(err error) bool {
	return err != nil && strings.Contains(err.Error(), "protected system file")
}
