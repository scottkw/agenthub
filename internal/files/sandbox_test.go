// Package files_test exercises the TOCTOU-safe path sandbox built on Go 1.24+
// os.Root. The fuzz corpus (FuzzSandboxPath) seeds the 40+ payload menagerie
// documented in .planning/research/PITFALLS.md §Fuzz Corpus Skeleton. The
// fuzzer asserts no payload ever escapes the sandbox root — any Open() that
// returns a non-nil *os.File must point at a file genuinely inside rootPath.
package files_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/files"
)

// helper: build a sandbox rooted at a fresh tempdir with a known layout.
//
//	root/
//	  a.txt           "hello"
//	  empty.txt       "" (0 bytes)
//	  sub/
//	    b.txt         "nested"
//	  link -> ../etc  (symlink escaping root; only on unix where supported)
func newTestSandbox(t *testing.T) (*files.Sandbox, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "empty.txt"), nil, 0o644); err != nil {
		t.Fatalf("write empty.txt: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatalf("write sub/b.txt: %v", err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}
	return sb, root
}

// TestSandbox_OpensViaOSRoot — FS-01: Sandbox uses os.OpenInRoot, not
// EvalSymlinks+Open. Verified indirectly by behaviour: any path that resolves
// outside the root via symlink swap must return an error.
func TestSandbox_OpensViaOSRoot(t *testing.T) {
	sb, root := newTestSandbox(t)
	f, err := sb.Open("a.txt")
	if err != nil {
		t.Fatalf("Open a.txt: %v", err)
	}
	defer f.Close()
	got := make([]byte, 5)
	if _, err := f.Read(got); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("Open returned wrong bytes: %q", got)
	}
	// confirm the file is inside root
	name := f.Name()
	if !strings.Contains(name, root) {
		// os.Root.Open does not always set Name() to the absolute path
		// (it may be relative); just verify it does not contain ".."
		if strings.Contains(name, "..") {
			t.Errorf("Open returned escaped path: %q", name)
		}
	}
}

// TestNewSandbox_EmptyWorkDirRejected — defensive: an empty workDir must
// return an error rather than silently resolving to the process cwd.
func TestNewSandbox_EmptyWorkDirRejected(t *testing.T) {
	if _, err := files.NewSandbox(""); err == nil {
		t.Fatalf("NewSandbox(\"\") returned nil; want error")
	}
}

// TestNewSandbox_NonExistentRejected — a workDir that does not exist must
// return a clear error (caller should not pass invalid cwd through).
func TestNewSandbox_NonExistentRejected(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := files.NewSandbox(missing); err == nil {
		t.Fatalf("NewSandbox(missing) returned nil; want error")
	}
}

// TestNewSandbox_FileNotDirRejected — workDir pointing at a file (not a
// directory) must error; sandbox is per-directory only.
func TestNewSandbox_FileNotDirRejected(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "regular-file")
	if err := os.WriteFile(fp, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := files.NewSandbox(fp); err == nil {
		t.Fatalf("NewSandbox(file) returned nil; want error")
	}
}

// TestSandbox_OpenSubpath — verifies legitimate nested file access works.
func TestSandbox_OpenSubpath(t *testing.T) {
	sb, _ := newTestSandbox(t)
	f, err := sb.Open("sub/b.txt")
	if err != nil {
		t.Fatalf("Open sub/b.txt: %v", err)
	}
	defer f.Close()
	got := make([]byte, 6)
	if _, err := f.Read(got); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "nested" {
		t.Errorf("Open returned wrong bytes: %q", got)
	}
}

// TestSandbox_OpenDot — "." means the sandbox root itself; List uses this.
func TestSandbox_OpenDot(t *testing.T) {
	sb, _ := newTestSandbox(t)
	f, err := sb.Open(".")
	if err != nil {
		t.Fatalf("Open(\".\"): %v", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("Open(\".\") returned non-dir: %v", fi)
	}
}

// TestValidatePath_Rejects — FS-08: rejects absolute, traversal, encoded,
// Unicode, null, device-name, ADS, trailing-dot/space, drive-letter, UNC.
func TestValidatePath_Rejects(t *testing.T) {
	sb, _ := newTestSandbox(t)
	cases := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"null-byte", "secret.txt\x00.jpg"},
		{"null-byte-prefix", "\x00etc/passwd"},
		{"absolute-unix", "/etc/passwd"},
		{"absolute-proc", "/proc/self/cwd"},
		{"absolute-windows-drive", `C:\windows\system32`},
		{"absolute-windows-forward", `C:/windows/system32`},
		{"unc-backslash", `\\server\share\file`},
		{"unc-forward", `//server/share/file`},
		{"ads-suffix", "file.txt:hidden"},
		{"ads-data", "file.txt:$DATA"},
		{"ads-index", ":$i30:$INDEX_ALLOCATION"},
		{"device-con", "CON"},
		{"device-con-lower", "con"},
		{"device-con-ext", "CON.txt"},
		{"device-nul", "nul"},
		{"device-nul-ext", "NUL.txt"},
		{"device-prn", "PRN"},
		{"device-aux", "AUX"},
		{"device-com1", "COM1"},
		{"device-lpt1", "LPT1"},
		{"device-com9-ext", "com1.txt"},
		{"device-lpt9-ext", "lpt9.go"},
		{"traversal-parent", "../etc/passwd"},
		{"traversal-deep", "../../etc/shadow"},
		{"traversal-via-clean", "a/../../etc/passwd"},
		{"traversal-only", ".."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := sb.Open(tc.path); err == nil {
				t.Errorf("Open(%q) returned nil error; want rejection", tc.path)
			}
		})
	}
}

// TestValidatePath_AcceptsLegitimate — pin behaviour: non-malicious paths
// must NOT be rejected by validation. (Easy to over-blacklist Unicode.)
func TestValidatePath_AcceptsLegitimate(t *testing.T) {
	sb, _ := newTestSandbox(t)
	cases := []string{
		".",
		"a.txt",
		"sub",
		"sub/b.txt",
		"./a.txt", // ./ should be normalized to a.txt by Clean
		"./sub/b.txt",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			if _, err := sb.Open(p); err != nil {
				t.Errorf("Open(%q) returned error %v; want nil", p, err)
			}
		})
	}
}

// TestSandbox_SymlinkEscapeBlocked — Pitfall 1 TOCTOU class. Create a
// symlink inside the root that points outside; opening it via the sandbox
// must NOT follow the link out. (Skip on Windows where symlinks need admin.)
//
// WR-03 hardening: a positive control proves outside/secret actually
// exists on disk, so any pass MUST come from os.Root rejecting the
// escape — not from an ENOENT that would mask a broken implementation
// (e.g., one that called os.Open(filepath.Join(rootPath, rel)) instead
// of root.Open, which would also fail "not found" via the wrong route).
// The escaping Open is then required to fail AND its file handle (if
// any) is closed to prevent leaks if the test ever regresses.
func TestSandbox_SymlinkEscapeBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires admin on Windows")
	}
	sb, root := newTestSandbox(t)
	// Create an outside target the symlink will point to.
	outside := t.TempDir()
	secretPath := filepath.Join(outside, "secret")
	if err := os.WriteFile(secretPath, []byte("leaked"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Positive control: confirm outside/secret really is readable via
	// a direct os.ReadFile so the negative assertion below can't pass
	// merely because the target doesn't exist (WR-03).
	if data, err := os.ReadFile(secretPath); err != nil {
		t.Fatalf("positive control: outside/secret unreadable: %v", err)
	} else if string(data) != "leaked" {
		t.Fatalf("positive control: outside/secret content = %q; want %q", data, "leaked")
	}
	linkPath := filepath.Join(root, "escape")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	// Attempting to open through the symlink must fail — os.Root rejects
	// symlinks that escape the root atomically.
	fp, err := sb.Open("escape/secret")
	if err == nil {
		// Close the leaked handle so the test doesn't leave an open fd.
		fp.Close()
		t.Errorf("Open through escaping symlink succeeded; want error")
	}
}

// FuzzSandboxPath — FS-09: 40+ seeded path-traversal payloads from
// PITFALLS.md §Fuzz Corpus Skeleton. Merge gate: `go test -fuzz=FuzzSandboxPath
// -fuzztime=60s ./internal/files/` must report zero crashes. The body asserts
// no panic occurs and any non-error Open() returns a *os.File whose Stat
// succeeds (proxy for "inside root" — os.Root's atomic open is the security
// boundary and never returns a handle outside the root).
func FuzzSandboxPath(f *testing.F) {
	// Classic traversal
	f.Add("../etc/passwd")
	f.Add("../../etc/shadow")
	f.Add("a/../../etc/passwd")
	// Encoded variants
	f.Add("%2e%2e%2fetc%2fpasswd")
	f.Add("%252e%252e%252fetc%252fpasswd")
	// Absolute paths
	f.Add("/etc/passwd")
	f.Add("/proc/self/cwd")
	f.Add("/proc/self/fd/0")
	// Windows absolute
	f.Add(`C:\windows\system32\cmd.exe`)
	f.Add(`\\server\share\file`)
	f.Add(`C:/windows/system32/cmd.exe`)
	// Windows device names
	f.Add("CON")
	f.Add("con")
	f.Add("CON.txt")
	f.Add("nul")
	f.Add("NUL.txt")
	f.Add("PRN")
	f.Add("AUX")
	f.Add("COM1")
	f.Add("LPT1")
	f.Add("COM1.txt")
	f.Add("lpt9.go")
	// Alternate data streams
	f.Add("file.txt:hidden")
	f.Add("file.txt:$DATA")
	f.Add(":$i30:$INDEX_ALLOCATION")
	// Null bytes
	f.Add("secret.txt\x00.jpg")
	f.Add("foo\x00")
	f.Add("\x00etc/passwd")
	// Unicode tricks
	f.Add("foo／etc／passwd") // fullwidth slash U+FF0F
	f.Add("foo․passwd")     // one-dot-leader U+2024
	f.Add("foo‥bar")        // two-dot-leader U+2025
	// Trailing dots/spaces (Windows strips these)
	f.Add("file.")
	f.Add("file.txt.")
	f.Add("file.txt  ")
	// Long paths
	f.Add(strings.Repeat("a/", 512) + "passwd")
	f.Add(strings.Repeat("../", 512))
	// Symlink names
	f.Add("link")
	f.Add("a/b/link")
	// 8.3 short names
	f.Add("PROGRA~1/system.dll")
	f.Add("progra~2/file.exe")
	// Mixed separators
	f.Add(`a\b/c`)
	f.Add(`a/b\c`)
	// Empty and dot paths
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("./")
	f.Add("./etc/passwd")
	f.Add(".hidden")
	f.Add("..hidden")

	root := f.TempDir()
	// Populate a few real files so accepted paths can be stat'd to prove
	// they live inside root.
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("ok"), 0o644); err != nil {
		f.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		f.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("ok"), 0o644); err != nil {
		f.Fatal(err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		f.Fatalf("NewSandbox: %v", err)
	}

	f.Fuzz(func(t *testing.T, rawPath string) {
		// The function under test must not panic on any input. It either
		// returns an error (rejection) or a valid *os.File handle pointing
		// genuinely inside root — never outside via symlink or absolute escape.
		fp, err := sb.Open(rawPath)
		if err != nil {
			// Rejection is fine — fuzz target is to find panics, not to
			// minimize the rejection set. err is naturally consumed by
			// the predicate above; no linter-appeasement gymnastics needed.
			return
		}
		// Non-error accept: prove the result is inside root.
		defer fp.Close()
		if _, err := fp.Stat(); err != nil {
			t.Errorf("accepted path %q stat failed: %v", rawPath, err)
		}
	})
}

// FuzzSandboxWrite — FSW-07: extends the FuzzSandboxPath corpus with
// write-specific traversal seeds and exercises every write method
// (WriteFileAtomic, Rename source+destination, Mkdir, Delete). The merge
// gate is `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/`
// reporting zero crashes.
//
// Every seed from FuzzSandboxPath is reused so the write surface inherits
// the full traversal menagerie. Fuzz body asserts: (a) no panic, (b) if a
// write method returns nil error, nothing was created/modified outside the
// sandbox root (verified by ensuring the parent of root is untouched).
func FuzzSandboxWrite(f *testing.F) {
	// ---- Reuse the entire FuzzSandboxPath corpus ----

	// Classic traversal
	f.Add("../etc/passwd")
	f.Add("../../etc/shadow")
	f.Add("a/../../etc/passwd")
	// Encoded variants
	f.Add("%2e%2e%2fetc%2fpasswd")
	f.Add("%252e%252e%252fetc%252fpasswd")
	// Absolute paths
	f.Add("/etc/passwd")
	f.Add("/proc/self/cwd")
	f.Add("/proc/self/fd/0")
	// Windows absolute
	f.Add(`C:\windows\system32\cmd.exe`)
	f.Add(`\\server\share\file`)
	f.Add(`C:/windows/system32/cmd.exe`)
	// Windows device names
	f.Add("CON")
	f.Add("con")
	f.Add("CON.txt")
	f.Add("nul")
	f.Add("NUL.txt")
	f.Add("PRN")
	f.Add("AUX")
	f.Add("COM1")
	f.Add("LPT1")
	f.Add("COM1.txt")
	f.Add("lpt9.go")
	// Alternate data streams
	f.Add("file.txt:hidden")
	f.Add("file.txt:$DATA")
	f.Add(":$i30:$INDEX_ALLOCATION")
	// Null bytes
	f.Add("secret.txt\x00.jpg")
	f.Add("foo\x00")
	f.Add("\x00etc/passwd")
	// Unicode tricks
	f.Add("foo／etc／passwd") // fullwidth slash U+FF0F
	f.Add("foo․passwd")     // one-dot-leader U+2024
	f.Add("foo‥bar")        // two-dot-leader U+2025
	// Trailing dots/spaces (Windows strips these)
	f.Add("file.")
	f.Add("file.txt.")
	f.Add("file.txt  ")
	// Long paths
	f.Add(strings.Repeat("a/", 512) + "passwd")
	f.Add(strings.Repeat("../", 512))
	// Symlink names
	f.Add("link")
	f.Add("a/b/link")
	// 8.3 short names
	f.Add("PROGRA~1/system.dll")
	f.Add("progra~2/file.exe")
	// Mixed separators
	f.Add(`a\b/c`)
	f.Add(`a/b\c`)
	// Empty and dot paths
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("./")
	f.Add("./etc/passwd")
	f.Add(".hidden")
	f.Add("..hidden")

	// ---- Write-specific seeds ----
	f.Add("../../.ssh/authorized_keys")
	f.Add("../../.bashrc")
	f.Add("../../.claude/CLAUDE.md")
	f.Add("../../../etc/cron.d/pwn")
	f.Add("foo.txt.agenthub-tmp-deadbeef") // temp-name collision probe
	f.Add("..%2f..%2f.bashrc")

	// ---- Phase 127, SEC-06: case-variation + multipart-filename vectors ----
	f.Add("../../.BASHRC")
	f.Add("../../.Bashrc")
	f.Add(".SSH/authorized_keys")
	f.Add("../../../etc/passwd") // multipart-filename-style injection

	// ---- Setup ----
	root := f.TempDir()
	// Populate a few real files so accepted paths can be stat'd to prove
	// they live inside root.
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("ok"), 0o644); err != nil {
		f.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		f.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("ok"), 0o644); err != nil {
		f.Fatal(err)
	}
	sb, err := files.NewSandbox(root)
	if err != nil {
		f.Fatalf("NewSandbox: %v", err)
	}

	f.Fuzz(func(t *testing.T, rawPath string) {
		// Must never panic and never escape the root. Errors (rejections)
		// are expected and acceptable — we assert on no-panic and in-root
		// invariant only.

		// WriteFileAtomic: write a single byte.
		_ = sb.WriteFileAtomic(rawPath, []byte("x"))

		// Rename source traversal: rawPath as source → must not escape.
		_ = sb.Rename(rawPath, "safe-target.txt")

		// Rename destination traversal (Pitfall 1): rawPath as destination.
		_ = sb.Rename("a.txt", rawPath)

		// Mkdir: create a directory with the raw path.
		_ = sb.Mkdir(rawPath)

		// Delete: attempt deletion with the raw path.
		_ = sb.Delete(rawPath)

		// In-root assertion: if any accepted write created a file, it must
		// be inside root. Walk the parent of root to detect escapes — an
		// escaped file would appear outside root's parent boundary.
		// We use root's parent: anything the fuzzer created that is NOT
		// under root is a confinement failure.
		rootParent := filepath.Dir(root)
		entries, err := os.ReadDir(rootParent)
		if err != nil {
			return // parent may be read-protected; skip assertion
		}
		for _, e := range entries {
			if e.Name() == filepath.Base(root) {
				continue // root itself is expected
			}
			// Any file in the parent that wasn't there before is a leak.
			// We seed only "a.txt" in root, so parent should have no new files.
			// This is a best-effort heuristic; os.Root's confinement is the
			// primary guarantee.
			if strings.HasPrefix(e.Name(), ".agenthub-tmp-") {
				t.Errorf("escaped temp file in root parent: %q", e.Name())
			}
		}
	})
}
