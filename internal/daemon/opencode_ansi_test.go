package daemon

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
)

// TestOpenCodeANSICapture validates assumption A1 from RESEARCH.md:
// OpenCode's "system" theme emits ANSI palette indices (e.g. \033[31m)
// rather than 24-bit RGB escape sequences (e.g. \033[38;2;R;G;Bm).
//
// This is an integration test that spawns the real opencode binary in a PTY.
// It is skipped when:
//   - opencode is not installed (exec.LookPath fails)
//   - running in CI (CI env var set)
//   - running with -short flag
func TestOpenCodeANSICapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping integration test in CI")
	}

	_, err := exec.LookPath("opencode")
	if err != nil {
		t.Skip("opencode not installed, skipping integration test")
	}

	// Setup managed tui.json with system theme in a temp directory.
	tuiConfigPath := ensureOpenCodeTUIConfig(t.TempDir())
	t.Logf("tui.json path: %s", tuiConfigPath)

	// Spawn opencode in a real PTY so the TUI renders escape sequences.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := gopty.New()
	if err != nil {
		t.Fatalf("open pty: %v", err)
	}
	defer p.Close()

	cmd := p.CommandContext(ctx, "opencode")
	cmd.Env = append(os.Environ(),
		"OPENCODE_TUI_CONFIG="+tuiConfigPath,
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("start opencode: %v", err)
	}

	// Read PTY output for up to 5 seconds to capture TUI startup rendering.
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, p)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
	cancel() // kill the process
	_ = cmd.Wait()

	output := buf.Bytes()
	t.Logf("Output length: %d bytes", len(output))

	// If insufficient output was captured, skip gracefully.
	if len(output) < 50 {
		t.Skip("insufficient output captured from opencode — may need longer startup time or TTY allocation")
	}

	outputStr := string(output)

	// Count 24-bit RGB sequences: \033[38;2;R;G;Bm or \033[48;2;R;G;Bm
	rgb24Regex := regexp.MustCompile(`\x1b\[(?:38|48);2;\d+;\d+;\d+m`)
	rgbMatches := rgb24Regex.FindAllString(outputStr, -1)

	// Count ANSI palette sequences: \033[30-37m, \033[90-97m (foreground)
	// and \033[40-47m, \033[100-107m (background)
	ansiPaletteRegex := regexp.MustCompile(`\x1b\[(?:3[0-7]|4[0-7]|9[0-7]|10[0-7])m`)
	ansiMatches := ansiPaletteRegex.FindAllString(outputStr, -1)

	t.Logf("24-bit RGB sequences found: %d", len(rgbMatches))
	t.Logf("ANSI palette sequences found: %d", len(ansiMatches))
	if len(rgbMatches) > 0 {
		limit := len(rgbMatches)
		if limit > 5 {
			limit = 5
		}
		t.Logf("First %d RGB sequences: %v", limit, rgbMatches[:limit])
	}
	if len(ansiMatches) > 0 {
		limit := len(ansiMatches)
		if limit > 5 {
			limit = 5
		}
		t.Logf("First %d ANSI sequences: %v", limit, ansiMatches[:limit])
	}

	// Analyze results: the system theme may or may not use ANSI palette indices.
	// This test is diagnostic — it validates assumption A1 from RESEARCH.md
	// and logs the empirical result. The finding informs whether the current
	// env-injection approach is sufficient or needs adjustment.
	total := len(ansiMatches) + len(rgbMatches)
	if total > 0 {
		ratio := float64(len(ansiMatches)) / float64(total) * 100
		t.Logf("ANSI palette ratio: %.1f%% (%d palette / %d total color sequences)",
			ratio, len(ansiMatches), total)
	}

	if len(rgbMatches) > 0 && len(ansiMatches) == 0 {
		t.Logf("WARNING: assumption A1 violated — system theme produced %d 24-bit RGB sequences but 0 ANSI palette sequences", len(rgbMatches))
		t.Logf("The system theme does NOT emit ANSI palette indices as expected.")
		t.Logf("The env-injection approach alone may be insufficient; xterm.js cannot remap 24-bit RGB colors.")
	} else if len(ansiMatches) > 0 && len(rgbMatches) == 0 {
		t.Logf("assumption A1 confirmed — system theme uses only ANSI palette indices")
	} else if len(ansiMatches) > 0 && len(rgbMatches) > 0 {
		t.Logf("mixed output — system theme uses both ANSI palette (%d) and 24-bit RGB (%d) sequences", len(ansiMatches), len(rgbMatches))
	} else {
		t.Logf("no color sequences detected in output")
	}
}
