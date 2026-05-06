// Phase 94 WR-01 / SC-4 / SRC-04 / SRC-05 — web find-bar animation wiring.
//
// Source-inspection guard: web/assets/terminal.js and web/assets/terminal.css
// must carry the same slide-in / slide-out wiring as the desktop FindBar.tsx
// + style.css. Mirrors the desktop test pair (FindBar.animation.test.tsx +
// TerminalPanel.search.exit.test.tsx) for the plain-DOM web surface.
//
// Same source-inspection style as vendor_drift_test.go.
package webserver

import (
	"os"
	"strings"
	"testing"
)

func TestWebFindBarAnimationWiring(t *testing.T) {
	jsBytes, err := os.ReadFile("../../web/assets/terminal.js")
	if err != nil {
		t.Fatalf("read terminal.js: %v", err)
	}
	js := string(jsBytes)
	for _, want := range []string{
		"classList.add('find-bar--entering')",
		"classList.add('find-bar--exiting')",
		"requestAnimationFrame",
		"findBarExitTimer",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("web/assets/terminal.js missing required animation hook: %q", want)
		}
	}

	cssBytes, err := os.ReadFile("../../web/assets/terminal.css")
	if err != nil {
		t.Fatalf("read terminal.css: %v", err)
	}
	css := string(cssBytes)
	for _, want := range []string{
		"#find-bar.find-bar--entering",
		"#find-bar.find-bar--exiting",
		"translateY(-100%)",
		"translateY(-8px)",
		"transition: transform 200ms ease, opacity 150ms ease",
		"prefers-reduced-motion",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("web/assets/terminal.css missing required rule: %q", want)
		}
	}
}
