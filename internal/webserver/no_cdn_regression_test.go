// Package webserver source-grep regression tests (Phase 89, D-17/D-06).
//
// TestSecurity_NoCDNReferencesInWebAssets walks every .html, .js, .css file
// under web/ (excluding web/vendor/xterm/) and asserts none contain CDN
// references. Phase 89 D-17 requires all script and style bytes to come from
// the vendored /assets/* path, not external CDN URLs.
//
// TestSecurity_NoInlineScriptOrStyleInHTML walks every .html file under web/
// and asserts no inline <script> or <style> blocks exist. Phase 89 D-06
// requires all inline blocks to be extracted to external .js/.css files so
// the strict CSP (script-src 'self'; style-src 'self') can block inline
// execution in the browser.
//
// These tests mirror the anti-regression pattern from security_regression_test.go
// (Phase 88, SC-4). A future maintainer cannot paste a CDN URL or inline block
// back into HTML without immediately failing these tests.
package webserver

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSecurity_NoCDNReferencesInWebAssets walks web/ and asserts no file
// contains a CDN reference. Phase 89 D-17 — prevents future maintainers
// from pasting cdn.jsdelivr.net or similar CDN tags back into HTML or JS.
// The vendor/xterm tree is excluded (per D-01/D-02 xterm is vendored at
// web/vendor/xterm/ on disk) because minified third-party bytes may
// incidentally contain "cdn" substrings.
func TestSecurity_NoCDNReferencesInWebAssets(t *testing.T) {
	forbidden := []struct{ needle, reason string }{
		{"cdn.jsdelivr", "Phase 89 D-17: xterm must be vendored, not fetched from jsDelivr"},
		{"unpkg.com", "Phase 89 D-17: no CDN dependencies in web assets"},
		{"://cdn.", "Phase 89 D-17: no CDN dependencies (catches cdn.* generic hostnames)"},
		{`<script src="http`, "Phase 89 D-17: all script tags must use relative /assets/ paths"},
		{`<link href="http`, "Phase 89 D-17: all link tags must use relative /assets/ paths"},
	}
	err := filepath.WalkDir("../../web", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip vendored third-party tree — minified blobs may
			// incidentally contain "cdn" substrings (verified in Research Q3).
			// Path is vendor/xterm per D-01/D-02 (user-locked decision).
			if strings.HasSuffix(path, "vendor/xterm") || strings.HasSuffix(path, "vendor\\xterm") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".js" && ext != ".css" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(data)
		for _, f := range forbidden {
			if strings.Contains(src, f.needle) {
				t.Errorf("%s contains forbidden string %q — %s (remediation: re-vendor or switch to /assets/ path)", path, f.needle, f.reason)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk web/: %v", err)
	}
}

// TestSecurity_NoInlineScriptOrStyleInHTML walks *.html under web/ and
// asserts NO inline <script> or <style> block (Phase 89 D-06 — all
// inline blocks MUST be extracted to external .js/.css files). External
// <script src="..."> and <link rel="stylesheet" href="..."> tags are
// allowed. This test guards against future maintainers adding a "quick"
// inline block that would be blocked by the strict CSP.
func TestSecurity_NoInlineScriptOrStyleInHTML(t *testing.T) {
	// Match <script> or <script type="..."> etc. WITHOUT a src= attribute.
	inlineScript := regexp.MustCompile(`<script(?:\s+(?:type|nonce|async|defer)="[^"]*")*\s*>`)
	// Match <style> open tags (any attributes).
	inlineStyle := regexp.MustCompile(`<style[^>]*>`)

	err := filepath.WalkDir("../../web", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(data)
		if m := inlineScript.FindString(src); m != "" {
			t.Errorf("%s contains inline <script> tag %q (Phase 89 D-06: extract to web/assets/*.js and reference via <script src=\"/assets/...\">)", path, m)
		}
		if m := inlineStyle.FindString(src); m != "" {
			t.Errorf("%s contains inline <style> tag %q (Phase 89 D-06: extract to web/assets/*.css and reference via <link rel=\"stylesheet\" href=\"/assets/...\">)", path, m)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk web/: %v", err)
	}
}
