package release_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSER03_NoAutoSavePatterns enforces the SER-03 invariant: no on-disk
// capture of session state occurs without an explicit user action in v3.2.
// This is a static regex-grep regression test, mirror of Phase 88's
// OriginPatterns: ["*"] reintroduction guard and Phase 93's
// vendor_drift_test.go filepath.Walk pattern.
//
// The test scans both the Go tree and the frontend src/ tree for forbidden
// patterns. It runs GREEN immediately because no auto-save patterns exist
// today; the test exists as forever-defense against regression. Adding a
// setInterval/setTimeout/BeforeQuit/OnShutdown hook that calls serialize()
// — or a PluginSettings field whose JSON name matches autoSave/autoExport/
// autoCapture/saveOnX — will cause this test to FAIL on next CI run.
//
// Phase 97 SER-03. Mirror of internal/webserver/vendor_drift_test.go and
// Phase 88 origin_test.go negative-regression patterns.
func TestSER03_NoAutoSavePatterns(t *testing.T) {
	// Forbidden patterns. Each entry: (regex, description).
	forbidden := []struct {
		re   *regexp.Regexp
		desc string
	}{
		// Long-delay scheduled serialize() invocation
		{regexp.MustCompile(`setInterval\([^)]*[Ss]eriali[zs]e`), "setInterval scheduling serialize()"},
		{regexp.MustCompile(`setTimeout\([^,]*[Ss]eriali[zs]e[^,]*,\s*[0-9]{4,}`), "setTimeout long-delay scheduled serialize()"},
		// Lifecycle-hook serialize() invocation
		{regexp.MustCompile(`(?i)BeforeQuit\b[^}]*[Ss]eriali[zs]e`), "BeforeQuit hook calling serialize()"},
		{regexp.MustCompile(`(?i)OnShutdown\b[^}]*[Ss]eriali[zs]e`), "OnShutdown hook calling serialize()"},
		// Auto-save settings vocabulary
		{regexp.MustCompile(`(?i)\bauto[._-]?save\b`), "auto-save vocabulary"},
		{regexp.MustCompile(`(?i)\bauto[._-]?export\b`), "auto-export vocabulary"},
		{regexp.MustCompile(`(?i)\bauto[._-]?capture\b`), "auto-capture vocabulary"},
		{regexp.MustCompile(`(?i)\bsave[._-]?on[._-]?(quit|exit|close)\b`), "save-on-quit/exit/close vocabulary"},
	}

	// Skip-list directories.
	skipDirs := map[string]bool{
		".git":                   true,
		"node_modules":           true,           // top-level node_modules (if any)
		"frontend/node_modules":  true,           // pnpm workspace node_modules
		"build":                  true,
		"dist":                   true,
		"vendor":                 true,
		"internal/release":       true, // self — regex literals would false-positive
		".planning":              true, // research/plan docs cite the patterns by name
		"frontend/src/wailsjs":   true, // generated bindings
		"screenshots":            true,
		".claude":                true, // agent worktrees + harness state (e.g. .claude/worktrees/agent-*/frontend/node_modules)
		".claire":                true, // alternate harness state dir
	}

	// Repo root is two levels up from internal/release/.
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	var violations []string
	err = filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(repoRoot, path)
		if d.IsDir() {
			for skip := range skipDirs {
				if relPath == skip || strings.HasPrefix(relPath, skip+string(filepath.Separator)) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		// Only scan source files.
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".js" {
			return nil
		}
		// Self-skip (the test file itself).
		if strings.HasSuffix(path, "no_autosave_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // unreadable files — best-effort scan
		}
		for _, pat := range forbidden {
			if loc := pat.re.FindIndex(data); loc != nil {
				line := 1 + strings.Count(string(data[:loc[0]]), "\n")
				violations = append(violations, relPath+":"+itoa(line)+" matches forbidden pattern: "+pat.desc)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.WalkDir: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("SER-03 invariant violated — auto-save patterns detected:\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestSER03_NoAutoSettingsField asserts that PluginSettings (defined in
// internal/daemon/plugin_settings.go) has no JSON field whose name matches
// autoSave/autoExport/autoCapture/saveOnX — locking the SER-03 invariant
// against a future settings-key regression.
func TestSER03_NoAutoSettingsField(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "internal", "daemon", "plugin_settings.go"))
	if err != nil {
		t.Fatalf("ReadFile plugin_settings.go: %v", err)
	}
	forbidden := []*regexp.Regexp{
		regexp.MustCompile("`json:\"auto[Ss]ave\""),
		regexp.MustCompile("`json:\"auto[Ee]xport\""),
		regexp.MustCompile("`json:\"auto[Cc]apture\""),
		regexp.MustCompile("`json:\"saveOn[A-Z][a-zA-Z]+\""),
	}
	for _, re := range forbidden {
		if re.Match(data) {
			t.Errorf("SER-03 invariant violated: PluginSettings contains forbidden auto-save field matching %s", re.String())
		}
	}
}

// TestSER03_OnlySaveTerminalSessionInAppGo asserts that app.go contains
// EXACTLY ONE method matching the regex (?i)func\s+\([^)]*\*App\)\s+\w*[Ss]ave\w*(?:Session|Terminal|Scrollback)\w*
// and that method's name is exactly SaveTerminalSession. This locks the
// SER-03 dialog-call-site invariant: there is one and only one Save*
// method in app.go that handles the terminal/session save gesture. Adding
// a second (e.g., SaveSessionAuto) would cause this test to FAIL.
//
// Per VALIDATION.md row 12 (SER-03: "Only (*App).SaveTerminalSession
// matches (?i)save.*(session|terminal|scrollback) in app.go").
//
// RED scaffold in Wave 0 — Plan 97-01 — because (*App).SaveTerminalSession
// does NOT yet exist (Plan 97-05 lands it). Wave 0 ships this as a t.Skip
// scaffold; Plan 97-05 unskips by removing the t.Skip line — at that point
// len(names) == 1 && names[0] == "SaveTerminalSession" will hold and the
// test flips GREEN automatically.
func TestSER03_OnlySaveTerminalSessionInAppGo(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	src, err := os.ReadFile(filepath.Join(repoRoot, "app.go"))
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}
	re := regexp.MustCompile(`(?i)func\s+\([^)]*\*App\)\s+(\w*[Ss]ave\w*(?:Session|Terminal|Scrollback)\w*)\s*\(`)
	matches := re.FindAllSubmatch(src, -1)
	var names []string
	for _, m := range matches {
		names = append(names, string(m[1]))
	}
	if len(names) != 1 || names[0] != "SaveTerminalSession" {
		t.Errorf("SER-03: expected exactly one Save*(Session|Terminal|Scrollback) method named SaveTerminalSession in app.go, got %v", names)
	}
}

// itoa formats an int as decimal — used to avoid pulling in strconv just
// for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
