package release_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPRG_OffPath_NoProgressLogic enforces the PRG OFF-path invariant: no
// polling pattern for progress state exists in the codebase. The Progress addon
// is event-driven (addon.onChange); any setInterval/setTimeout pattern that
// references [Pp]rogress is a forbidden anti-pattern (Pitfall #6 in
// 98-RESEARCH.md).
//
// Runs GREEN immediately at Wave 0 — no progress code exists yet. Exists as
// forever-defense: adding a setInterval/setTimeout polling loop that references
// progress will cause this test to FAIL on next CI run.
//
// Phase 98 PRG-01/PRG-02. Mirror of internal/release/no_autosave_test.go
// (Phase 97 SER-03 pattern) and internal/webserver/vendor_drift_test.go.
func TestPRG_OffPath_NoProgressLogic(t *testing.T) {
	forbidden := []struct {
		re   *regexp.Regexp
		desc string
	}{
		{regexp.MustCompile(`setInterval\([^)]*[Pp]rogress`), "setInterval polling progress (Pitfall #6 — addon is event-driven)"},
		{regexp.MustCompile(`setTimeout\([^,]*[Pp]rogress[^,]*,\s*[0-9]{4,}`), "setTimeout long-delay progress polling"},
	}

	skipDirs := map[string]bool{
		".git":                  true,
		"node_modules":          true,
		"frontend/node_modules": true,
		"build":                 true,
		"dist":                  true,
		"vendor":                true,
		"internal/release":      true, // self — regex literals would false-positive
		".planning":             true, // research/plan docs cite patterns by name
		"frontend/src/wailsjs":  true, // generated bindings
		"screenshots":           true,
		".claude":               true, // agent worktrees + harness state
		".claire":               true, // alternate harness state dir
	}

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
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".js" {
			return nil
		}
		if strings.HasSuffix(path, "no_progress_when_off_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // unreadable files — best-effort scan
		}
		for _, pat := range forbidden {
			if loc := pat.re.FindIndex(data); loc != nil {
				line := 1 + strings.Count(string(data[:loc[0]]), "\n")
				violations = append(violations, relPath+":"+itoa2(line)+" matches forbidden pattern: "+pat.desc)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.WalkDir: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("PRG OFF-path invariant violated — progress polling patterns detected:\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestPRG_NewProgressAddonIsGated asserts that any file in frontend/src/**
// containing `new ProgressAddon(` MUST also contain `pluginConfig?.progress`
// OR `pluginConfig.progress` in the same file. This is a coarse-grained
// file-level check — the real same-useEffect gating is enforced by Wave 2's
// hot-swap arm structure; the file-level invariant catches accidental top-level
// construction.
//
// Wave 0 result: GREEN (no `new ProgressAddon` exists yet). Wave 2 adds the
// construction inside the gated hot-swap arm — the same file will also contain
// `pluginConfig?.progress`, so the test stays GREEN.
//
// Skip: test files (__tests__/) and the aggregateProgress.ts stub.
func TestPRG_NewProgressAddonIsGated(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	frontendSrc := filepath.Join(repoRoot, "frontend", "src")
	constructionRe := regexp.MustCompile(`new\s+ProgressAddon\s*\(`)
	gatePattern1 := "pluginConfig?.progress"
	gatePattern2 := "pluginConfig.progress"

	var violations []string
	err = filepath.WalkDir(frontendSrc, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == "__tests__" || base == "node_modules" || base == "wailsjs" {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip test files and the stub
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") {
			return nil
		}
		if base == "aggregateProgress.ts" {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".tsx" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !constructionRe.Match(data) {
			return nil // no ProgressAddon construction — OK
		}
		// File contains `new ProgressAddon(` — must also contain the gate
		content := string(data)
		if !strings.Contains(content, gatePattern1) && !strings.Contains(content, gatePattern2) {
			relPath, _ := filepath.Rel(repoRoot, path)
			violations = append(violations, relPath+": contains 'new ProgressAddon(' without a 'pluginConfig?.progress' gate in the same file")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.WalkDir: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("PRG gating invariant violated — ungated ProgressAddon construction detected:\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestPRG_SetTrayProgressUsage asserts there is exactly ONE definition of
// `func (a *App) SetTrayProgress` in the Go source tree. This is a RED
// scaffold at Wave 0 — the method does not exist yet. Wave 1 (Plan 02 Task 2)
// adds the method; at that point len(names)==1 && names[0]=="SetTrayProgress"
// and the test flips GREEN automatically.
func TestPRG_SetTrayProgressUsage(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	src, err := os.ReadFile(filepath.Join(repoRoot, "app.go"))
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}
	re := regexp.MustCompile(`func\s+\([^)]*\*App\)\s+(SetTrayProgress)\s*\(`)
	matches := re.FindAllSubmatch(src, -1)
	var names []string
	for _, m := range matches {
		names = append(names, string(m[1]))
	}
	if len(names) != 1 || names[0] != "SetTrayProgress" {
		t.Errorf("PRG-03: expected exactly one (*App).SetTrayProgress method in app.go, got %v (Wave 0 RED — Wave 1 Plan 02 Task 2 adds the method)", names)
	}
}

// itoa2 formats an int as decimal. Separate from no_autosave_test.go's itoa
// to avoid duplicate declaration in the same package.
func itoa2(n int) string {
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
