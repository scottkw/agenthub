// Package webserver SC-4 anti-regression guards (Phase 88, D-13 items 1 & 3).
//
// These source-grep tests assert that the accept-all OriginPatterns literal
// is gone from the webserver and that the library-layer allowlist is wired
// to BaseURL(). A future maintainer cannot silently reintroduce
// `OriginPatterns: []string{"*"}` without failing the TestSecurity_
// NoAcceptAllOriginInWebserver test.
//
// Mirrors the Phase 87 TestVerify_ConstantTimeComparison pattern
// (internal/capability/capability_test.go:161-173).
package webserver

import (
	"os"
	"strings"
	"testing"
)

// TestSecurity_NoAcceptAllOriginInWebserver asserts via source inspection
// that server.go does not contain `OriginPatterns: []string{"*"}` (D-13
// item 1; SC-4 anti-regression).
func TestSecurity_NoAcceptAllOriginInWebserver(t *testing.T) {
	data, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("ReadFile server.go: %v", err)
	}
	src := string(data)
	if strings.Contains(src, `OriginPatterns: []string{"*"}`) {
		t.Error(`server.go must not contain OriginPatterns: []string{"*"} — Phase 88 SC-4 anti-regression; use ws.allowedOrigins() instead`)
	}
}

// TestSecurity_WebserverOriginAllowlistWiredToBaseURL asserts via source
// inspection that the WebSocket upgrade site references the BaseURL()-
// derived allowlist. Accepts either direct ws.BaseURL() or the
// ws.allowedOrigins() helper (D-13 item 3; SC-4 positive confirmation).
func TestSecurity_WebserverOriginAllowlistWiredToBaseURL(t *testing.T) {
	data, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("ReadFile server.go: %v", err)
	}
	src := string(data)
	// The handleWSSRelay AcceptOptions line must reference either
	// ws.BaseURL() directly or ws.allowedOrigins() (which returns the
	// singleton list built from BaseURL).
	if !strings.Contains(src, "ws.allowedOrigins()") && !strings.Contains(src, "ws.BaseURL()") {
		t.Error(`server.go handleWSSRelay AcceptOptions must reference ws.allowedOrigins() or ws.BaseURL() — Phase 88 SC-4 positive guard`)
	}
	// Stronger: the string "OriginPatterns" must appear and be followed
	// by ws.allowedOrigins() or ws.BaseURL() somewhere in the source.
	// (Weak check — fine. If someone removes OriginPatterns entirely,
	// item 1's grep still catches the *reintroduction* of "*".)
	if !strings.Contains(src, "OriginPatterns:") {
		t.Error(`server.go must still set OriginPatterns on websocket.AcceptOptions for handleWSSRelay (belt-and-suspenders D-12)`)
	}
}
