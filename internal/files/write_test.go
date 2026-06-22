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
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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
	// Redirect $HOME to a fake directory distinct from the sandbox root so that
	// denylistCheck reliably treats the sandbox as non-home-rooted on every OS.
	// On Windows, t.TempDir() lands under %USERPROFILE% (i.e. $HOME), so the
	// old case-sensitive strings.HasPrefix skip guard was fragile and would either
	// skip or fail depending on path capitalisation. By pointing $HOME at fakeHome
	// (a separate t.TempDir()) we guarantee the sandbox is outside $HOME without
	// relying on OS-specific tmpdir placement or a case-sensitive prefix check.
	fakeHome := t.TempDir()
	sandboxRoot := t.TempDir()
	setHomeEnv(t, fakeHome)

	sb, err := files.NewSandbox(sandboxRoot)
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

// TestDenylist_CaseVariation verifies that case-varied denylist targets are
// blocked in a home-rooted sandbox. This covers the macOS/Windows
// case-insensitive volume bypass (e.g. .BASHRC resolves to the same inode as
// .bashrc on those filesystems). SEC-02 / FSW-127-02.
func TestDenylist_CaseVariation(t *testing.T) {
	fakeHome := t.TempDir()
	setHomeEnv(t, fakeHome)

	// Seed prerequisite dirs/files so rename and delete reach the denylist.
	if err := os.MkdirAll(filepath.Join(fakeHome, ".SSH"), 0o700); err != nil {
		t.Fatalf("mkdir .SSH: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeHome, ".BASHRC"), []byte("# rc"), 0o644); err != nil {
		t.Fatalf("seed .BASHRC: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeHome, ".SSH", "authorized_keys"), []byte("ssh-rsa"), 0o600); err != nil {
		t.Fatalf("seed .SSH/authorized_keys: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(fakeHome, ".Claude"), 0o700); err != nil {
		t.Fatalf("mkdir .Claude: %v", err)
	}

	sb, err := files.NewSandbox(fakeHome)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	caseDenyTargets := []string{
		".BASHRC",
		".Bashrc",
		".SSH/authorized_keys",
		".Claude/CLAUDE.md",
	}

	for _, target := range caseDenyTargets {
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
	}
}

// TestDenylist_DaemonConfigDir verifies that the daemon config dir (derived at
// runtime from os.UserConfigDir()+"/agenthub") is protected in a home-rooted
// sandbox. On macOS this is ~/Library/Application Support/agenthub, on Linux
// ~/.config/agenthub. SEC-02 / FSW-127-01.
func TestDenylist_DaemonConfigDir(t *testing.T) {
	fakeHome := t.TempDir()
	setHomeEnv(t, fakeHome)

	// Set the platform-specific config-dir env so that os.UserConfigDir()
	// returns a path under fakeHome.
	switch runtime.GOOS {
	case "linux":
		// XDG_CONFIG_HOME overrides ~/.config on Linux.
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(fakeHome, ".config"))
	default:
		// On macOS and Windows os.UserConfigDir() is not controlled by a
		// per-process env var. Derive what it actually returns and skip if it
		// is not under fakeHome (i.e. the OS config dir cannot be faked here).
		cfgBase, cfgErr := os.UserConfigDir()
		if cfgErr != nil {
			t.Skipf("os.UserConfigDir() error: %v", cfgErr)
		}
		cfgRel, relErr := filepath.Rel(fakeHome, cfgBase)
		if relErr != nil || strings.HasPrefix(cfgRel, "..") {
			t.Skipf("os.UserConfigDir() %q is not under fakeHome %q; skipping daemon-config-dir test on this OS", cfgBase, fakeHome)
		}
	}

	// Derive the expected config dir the same way denylistCheck does.
	cfgBase, cfgErr := os.UserConfigDir()
	if cfgErr != nil {
		t.Skipf("os.UserConfigDir() error after env setup: %v", cfgErr)
	}
	cfgDir := filepath.Join(cfgBase, "agenthub")
	home, _ := os.UserHomeDir()
	if home == "" {
		t.Skip("UserHomeDir empty after setHomeEnv")
	}
	cfgRel, relErr := filepath.Rel(home, cfgDir)
	if relErr != nil || strings.HasPrefix(cfgRel, "..") {
		t.Skipf("config dir %q is not under fakeHome %q after env setup; cannot exercise this guard", cfgDir, home)
	}

	// Seed the config dir so rename and delete can reach the denylist.
	settingsPath := filepath.Join(cfgDir, "settings.json")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir cfgDir: %v", err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"key":"val"}`), 0o600); err != nil {
		t.Fatalf("seed settings.json: %v", err)
	}

	// Root the sandbox at fakeHome (the dangerous case — full home access).
	sb, err := files.NewSandbox(home)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Relative path inside the sandbox that resolves to the config file.
	relSettings := filepath.ToSlash(cfgRel) + "/settings.json"

	t.Run("WriteFileAtomic", func(t *testing.T) {
		err := sb.WriteFileAtomic(relSettings, []byte("pwned"))
		if err == nil {
			t.Errorf("WriteFileAtomic(%q) returned nil; want ErrProtectedSystemFile", relSettings)
			return
		}
		if !isProtected(err) {
			t.Errorf("WriteFileAtomic(%q) error = %v; want ErrProtectedSystemFile", relSettings, err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := sb.Delete(relSettings)
		if err == nil {
			t.Errorf("Delete(%q) returned nil; want ErrProtectedSystemFile", relSettings)
			return
		}
		if !isProtected(err) {
			t.Errorf("Delete(%q) error = %v; want ErrProtectedSystemFile", relSettings, err)
		}
	})

	t.Run("Rename/into", func(t *testing.T) {
		// Seed a safe source file to rename from.
		srcPath := "safe-src-for-rename.txt"
		if err := os.WriteFile(filepath.Join(home, srcPath), []byte("src"), 0o644); err != nil {
			t.Fatalf("seed safe-src: %v", err)
		}
		err := sb.Rename(srcPath, relSettings)
		if err == nil {
			t.Errorf("Rename(safe, %q) returned nil; want ErrProtectedSystemFile", relSettings)
			return
		}
		if !isProtected(err) {
			t.Errorf("Rename(safe, %q) error = %v; want ErrProtectedSystemFile", relSettings, err)
		}
	})
}

// --------------------------------------------------------------------------
// TestWrite_IfMatch* — EDIT-05/08 optimistic-concurrency precondition tests.
// These exercise Handler.Write (the HTTP handler layer), not the Sandbox
// primitive, so they drive the handler via httptest.NewRequest + NewRecorder,
// reusing the newHandler helper defined in handler_test.go.
// --------------------------------------------------------------------------

// invokeWrite is a helper that calls Handler.Write with the supplied body and
// optional If-Match header via httptest machinery.
func invokeWrite(t *testing.T, h *files.Handler, path string, body []byte, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	u := "/?session=good&path=" + path
	req := httptest.NewRequest(http.MethodPut, u, bytes.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rr := httptest.NewRecorder()
	h.Write(rr, req)
	return rr
}

// validatorFor computes the expected ETag / If-Match validator for the file at
// the given absolute path on disk, using the same format as Handler.Write:
// "<UnixNano>-<size>" (quoted via fmt.Sprintf("%q")).
func validatorFor(t *testing.T, absPath string) string {
	t.Helper()
	fi, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("validatorFor os.Stat: %v", err)
	}
	return fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))
}

// TestWrite_IfMatch_Match verifies that a PUT with the correct on-disk
// validator succeeds (200) and the new content is persisted.
func TestWrite_IfMatch_Match(t *testing.T) {
	h, root := newHandler(t)

	// Seed an existing file.
	initial := []byte("original content")
	absPath := filepath.Join(root, "match.txt")
	if err := os.WriteFile(absPath, initial, 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	// Build the correct validator from on-disk state.
	etag := validatorFor(t, absPath)

	// PUT with the matching If-Match header → expect 200.
	updated := []byte("updated content")
	rr := invokeWrite(t, h, "match.txt", updated, etag)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}

	// Verify new content is on disk.
	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, updated) {
		t.Errorf("disk content = %q; want %q", got, updated)
	}
}

// TestWrite_IfMatch_Mismatch verifies that a PUT with a stale validator
// returns 412 and leaves the original file content unchanged.
func TestWrite_IfMatch_Mismatch(t *testing.T) {
	h, root := newHandler(t)

	// Seed an existing file.
	initial := []byte("do not overwrite me")
	absPath := filepath.Join(root, "mismatch.txt")
	if err := os.WriteFile(absPath, initial, 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	// Use a stale validator (fabricated, won't match the real on-disk value).
	staleETag := `"0-0"`

	// PUT with stale If-Match → expect 412.
	rr := invokeWrite(t, h, "mismatch.txt", []byte("should not land"), staleETag)
	if rr.Code != http.StatusPreconditionFailed {
		t.Fatalf("status = %d; want 412; body=%s", rr.Code, rr.Body.String())
	}

	// Verify original content is still on disk.
	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, initial) {
		t.Errorf("disk content changed after 412: got %q; want %q", got, initial)
	}
}

// TestWrite_IfMatch_NewFile verifies that a PUT with no If-Match header to a
// non-existent path creates the file (200) without requiring a validator.
func TestWrite_IfMatch_NewFile(t *testing.T) {
	h, root := newHandler(t)

	absPath := filepath.Join(root, "newfile.txt")
	// Pre-condition: the file must not exist yet.
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		t.Fatalf("expected newfile.txt to be absent before test; err=%v", err)
	}

	content := []byte("brand new content")
	// PUT with no If-Match → new-file path, expect 200.
	rr := invokeWrite(t, h, "newfile.txt", content, "" /* no If-Match */)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}

	// Verify file was created on disk.
	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("disk content = %q; want %q", got, content)
	}
}

// ---------------------------------------------------------------------------
// WR-06: cross-surface upload cap parity assertion
// ---------------------------------------------------------------------------

// TestMaxUploadBytes_Is50MiB asserts that the server-side MaxUploadBytes
// constant is exactly 50 MiB — the milestone-locked value. This is the Go
// half of the cross-surface parity check; the frontend MAX_UPLOAD_BYTES
// constant (filesApi.ts) must equal the same value. Failing this test means
// the cap drifted and the client pre-check would diverge from the server cap.
func TestMaxUploadBytes_Is50MiB(t *testing.T) {
	const want = 50 * 1024 * 1024
	if files.MaxUploadBytes != want {
		t.Errorf("MaxUploadBytes = %d; want %d (50 MiB — milestone-locked cap)", files.MaxUploadBytes, want)
	}
}

// ---------------------------------------------------------------------------
// CR-01: TOCTOU validator re-check inside WriteFileAtomic
// ---------------------------------------------------------------------------

// TestWriteFileAtomic_ValidatorRecheck verifies that WriteFileAtomic re-checks
// the validator immediately before rename and returns ErrPreconditionFailed
// when the file was modified between the caller's pre-write Stat and the
// WriteFileAtomic call (simulating a concurrent writer landing between the two).
func TestWriteFileAtomic_ValidatorRecheck(t *testing.T) {
	tmp := t.TempDir()
	sb, err := files.NewSandbox(tmp)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Seed the file.
	initial := []byte("original")
	absPath := filepath.Join(tmp, "recheck.txt")
	if err := os.WriteFile(absPath, initial, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Capture the validator BEFORE a competing write changes the file.
	fi, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	validator := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))

	// Simulate a concurrent write that lands AFTER we captured the validator.
	if err := os.WriteFile(absPath, []byte("concurrent update"), 0o644); err != nil {
		t.Fatalf("concurrent write: %v", err)
	}

	// Now WriteFileAtomic should detect the mismatch and return ErrPreconditionFailed.
	writeErr := sb.WriteFileAtomic("recheck.txt", []byte("should not land"), validator)
	if writeErr == nil {
		t.Fatal("WriteFileAtomic returned nil; want ErrPreconditionFailed")
	}
	if !errors.Is(writeErr, files.ErrPreconditionFailed) {
		t.Errorf("WriteFileAtomic error = %v; want ErrPreconditionFailed", writeErr)
	}

	// Verify the concurrent update is still on disk (our write did not land).
	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, []byte("concurrent update")) {
		t.Errorf("disk content = %q; want concurrent update content", got)
	}
}

// TestWriteFileAtomic_ValidatorMatch verifies that WriteFileAtomic proceeds
// when the validator matches the current on-disk state.
func TestWriteFileAtomic_ValidatorMatch(t *testing.T) {
	tmp := t.TempDir()
	sb, err := files.NewSandbox(tmp)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	initial := []byte("before")
	absPath := filepath.Join(tmp, "match2.txt")
	if err := os.WriteFile(absPath, initial, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	validator := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))

	// No concurrent modification — validator should still match.
	if err := sb.WriteFileAtomic("match2.txt", []byte("after"), validator); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, []byte("after")) {
		t.Errorf("disk content = %q; want \"after\"", got)
	}
}

// TestWrite_IfMatch_RecheckViaHTTP verifies the HTTP handler maps
// ErrPreconditionFailed (from the in-WriteFileAtomic re-check) to 412.
// It simulates a mid-write file change by seeding a file, capturing its
// validator, externally updating it, then issuing PUT with the stale validator.
func TestWrite_IfMatch_RecheckViaHTTP(t *testing.T) {
	h, root := newHandler(t)

	initial := []byte("version-1")
	absPath := filepath.Join(root, "recheck-http.txt")
	if err := os.WriteFile(absPath, initial, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	etag := validatorFor(t, absPath)

	// Concurrent write changes the file after we captured the etag.
	if err := os.WriteFile(absPath, []byte("version-2-by-concurrent-writer"), 0o644); err != nil {
		t.Fatalf("concurrent: %v", err)
	}

	// PUT with the now-stale validator: both the pre-write Stat check AND the
	// in-WriteFileAtomic re-check should catch this; either way we get 412.
	rr := invokeWrite(t, h, "recheck-http.txt", []byte("version-3-never-lands"), etag)
	if rr.Code != http.StatusPreconditionFailed {
		t.Fatalf("status = %d; want 412; body=%s", rr.Code, rr.Body.String())
	}

	// Verify only version-2 is on disk.
	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, []byte("version-2-by-concurrent-writer")) {
		t.Errorf("disk = %q; want version-2", got)
	}
}

// ---------------------------------------------------------------------------
// WR-07 / IN-05: Upload collision (409) + overwrite=1 path
// ---------------------------------------------------------------------------

// invokeUpload is a helper that calls Handler.Upload with a multipart form.
// Pass overwrite="1" to test the WR-07 overwrite path, "" for the normal path.
func invokeUpload(t *testing.T, h *files.Handler, dir, filename, content, overwrite string) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("dir", dir)
	if overwrite != "" {
		_ = mw.WriteField("overwrite", overwrite)
	}
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("write part: %v", err)
	}
	mw.Close()

	u := "/?session=good"
	req := httptest.NewRequest(http.MethodPost, u, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	h.Upload(rr, req)
	return rr
}

// TestUpload_NewFile verifies that uploading a new file (no prior content)
// returns 200 and the file is placed on disk.
func TestUpload_NewFile(t *testing.T) {
	h, root := newHandler(t)

	rr := invokeUpload(t, h, ".", "upload-new.txt", "hello upload", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}
	got, err := os.ReadFile(filepath.Join(root, "upload-new.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello upload" {
		t.Errorf("disk = %q; want \"hello upload\"", got)
	}
}

// TestUpload_Collision_Returns409 verifies the WR-07 contract: uploading a file
// whose name already exists (without overwrite=1) returns 409 Conflict.
func TestUpload_Collision_Returns409(t *testing.T) {
	h, root := newHandler(t)

	// Seed the file.
	if err := os.WriteFile(filepath.Join(root, "existing.txt"), []byte("original"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rr := invokeUpload(t, h, ".", "existing.txt", "new content", "" /* no overwrite */)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d; want 409; body=%s", rr.Code, rr.Body.String())
	}

	// Original content must be unchanged.
	got, err := os.ReadFile(filepath.Join(root, "existing.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "original" {
		t.Errorf("disk = %q; want \"original\" after rejected collision", got)
	}
}

// TestUpload_Overwrite_Returns200 verifies the WR-07 contract: uploading with
// overwrite=1 skips the collision check and overwrites the existing file (200).
func TestUpload_Overwrite_Returns200(t *testing.T) {
	h, root := newHandler(t)

	// Seed the file.
	if err := os.WriteFile(filepath.Join(root, "overwrite-me.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rr := invokeUpload(t, h, ".", "overwrite-me.txt", "v2", "1" /* overwrite=1 */)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", rr.Code, rr.Body.String())
	}

	got, err := os.ReadFile(filepath.Join(root, "overwrite-me.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("disk = %q; want \"v2\" after overwrite", got)
	}
}

// TestUpload_IN05_MalformedMultipart_Returns400 verifies IN-05: a malformed
// multipart body (not an over-cap error) returns 400 rather than 413.
func TestUpload_IN05_MalformedMultipart_Returns400(t *testing.T) {
	h, _ := newHandler(t)

	// Send a non-multipart body to trigger ParseMultipartForm failure.
	req := httptest.NewRequest(http.MethodPost, "/?session=good", strings.NewReader("not-multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=nonexistent")
	rr := httptest.NewRecorder()
	h.Upload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400 for malformed multipart; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "malformed") {
		t.Errorf("body = %q; want 'malformed' in error message", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// SEC-05: Data-integrity tests — two-writer If-Match race + interrupted write
// ---------------------------------------------------------------------------

// TestWrite_TwoWritersIfMatchRace verifies that when two goroutines race to
// write the same file using the same If-Match validator, exactly one succeeds
// (returns nil) and the other gets ErrPreconditionFailed.  The resulting file
// content must equal exactly one writer's complete payload — never a
// byte-interleaved mix — and no .agenthub-tmp-* sibling may remain.
//
// This exercises the CR-01 TOCTOU mitigation: WriteFileAtomic re-checks the
// validator immediately before root.Rename, so the second writer whose
// validator has become stale returns ErrPreconditionFailed and removes its
// temp file without committing.
func TestWrite_TwoWritersIfMatchRace(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Seed the target file with known content.
	initial := []byte("seed content for two-writer race test")
	absPath := filepath.Join(root, "race-target.txt")
	if err := os.WriteFile(absPath, initial, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Capture the current validator (mtime-UnixNano + size, quoted) — same
	// format as validatorFor used by the existing IfMatch tests.
	fi, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("stat seed: %v", err)
	}
	sharedValidator := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))

	contentA := bytes.Repeat([]byte("A"), 512)
	contentB := bytes.Repeat([]byte("B"), 512)

	type result struct{ err error }
	results := make(chan result, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	// Both goroutines use the SAME validator captured before either write
	// commits.  One will win the rename; the other will find the validator
	// stale in the re-check and return ErrPreconditionFailed.
	go func() {
		defer wg.Done()
		results <- result{sb.WriteFileAtomic("race-target.txt", contentA, sharedValidator)}
	}()
	go func() {
		defer wg.Done()
		results <- result{sb.WriteFileAtomic("race-target.txt", contentB, sharedValidator)}
	}()

	wg.Wait()
	close(results)

	var nilCount, precondFailCount int
	for r := range results {
		if r.err == nil {
			nilCount++
		} else if errors.Is(r.err, files.ErrPreconditionFailed) {
			precondFailCount++
		} else {
			t.Errorf("unexpected error: %v", r.err)
		}
	}

	if nilCount != 1 {
		t.Errorf("nilCount = %d; want exactly 1 successful writer", nilCount)
	}
	if precondFailCount != 1 {
		t.Errorf("precondFailCount = %d; want exactly 1 ErrPreconditionFailed", precondFailCount)
	}

	// Final file content must be one writer's complete payload — never mixed.
	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile after race: %v", err)
	}
	isAllA := len(got) == len(contentA) && bytes.Count(got, []byte("A")) == len(contentA)
	isAllB := len(got) == len(contentB) && bytes.Count(got, []byte("B")) == len(contentB)
	if !isAllA && !isAllB {
		t.Errorf("file content is neither all-A nor all-B (interleaved write?): len=%d first byte=%q",
			len(got), got[:min(8, len(got))])
	}

	// No .agenthub-tmp-* sibling may remain after both writes complete.
	tmpPattern := filepath.Join(root, "race-target.txt.agenthub-tmp-*")
	leftover, err := filepath.Glob(tmpPattern)
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(leftover) != 0 {
		t.Errorf("leftover temp files after race: %v", leftover)
	}
}

// TestWrite_InterruptedWritePreservesOriginal verifies that a failed
// WriteFileAtomic call leaves the original file content intact and no
// .agenthub-tmp-* sibling behind.
//
// Implementation note: the most deterministic way to trigger the "temp
// created, rename skipped, temp cleaned up, original preserved" path through
// the public API is to pass a validator that will mismatch the on-disk value
// at the point of the re-check inside WriteFileAtomic.  This is exactly the
// same code path as the two-writer race failure branch: the temp file is
// created and written, the validator re-check returns ErrPreconditionFailed,
// the temp is removed via root.Remove(tmp), and the original is untouched.
// We document this choice here so a future reader understands why the test
// uses a stale validator rather than injecting a write-level fault.
func TestWrite_InterruptedWritePreservesOriginal(t *testing.T) {
	root := t.TempDir()
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	// Seed the target file with known original content.
	original := []byte("original content — must survive the failed write")
	absPath := filepath.Join(root, "preserve-me.txt")
	if err := os.WriteFile(absPath, original, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Modify the file after capturing the validator so it will be stale at
	// the time WriteFileAtomic performs its re-check.
	fi, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	staleValidator := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))

	// Simulate an interleaving write that changes the file before our call.
	interleaved := []byte("interleaved update — should be what remains on disk")
	if err := os.WriteFile(absPath, interleaved, 0o644); err != nil {
		t.Fatalf("interleaved write: %v", err)
	}

	// Now call WriteFileAtomic with the stale validator.  It must fail and
	// leave the interleaved content intact (the file as it was before our
	// call, which now represents "the original" from our perspective).
	writeErr := sb.WriteFileAtomic("preserve-me.txt", []byte("should not land"), staleValidator)
	if writeErr == nil {
		t.Fatal("WriteFileAtomic returned nil; want ErrPreconditionFailed")
	}
	if !errors.Is(writeErr, files.ErrPreconditionFailed) {
		t.Errorf("WriteFileAtomic error = %v; want ErrPreconditionFailed", writeErr)
	}

	// Original (interleaved) content must be intact — our failed write did not
	// clobber it.
	got, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, interleaved) {
		t.Errorf("disk content after failed write = %q; want %q", got, interleaved)
	}

	// No .agenthub-tmp-* sibling may remain — WriteFileAtomic must clean up.
	tmpPattern := filepath.Join(root, "preserve-me.txt.agenthub-tmp-*")
	leftover, err := filepath.Glob(tmpPattern)
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(leftover) != 0 {
		t.Errorf("leftover temp file after failed write: %v", leftover)
	}
}

// min is a small helper used by TestWrite_TwoWritersIfMatchRace to safely
// slice a byte slice for error messages without panicking on short content.
// (Redeclared here to avoid a dependency on the built-in min added in Go 1.21.)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
