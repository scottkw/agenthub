// Package relay SC-4 anti-regression guard (Phase 88, D-13 item 2).
//
// This source-grep test asserts that the InsecureSkipVerify: true literal
// is gone from server.go. A future maintainer cannot silently reintroduce
// it without failing this test.
//
// Mirrors the Phase 87 TestVerify_ConstantTimeComparison pattern
// (internal/capability/capability_test.go:161-173).
package relay

import (
	"os"
	"strings"
	"testing"
)

// TestSecurity_NoInsecureSkipVerifyInRelay asserts via source inspection
// that server.go does not contain `InsecureSkipVerify: true` (D-13 item
// 2; SC-4 anti-regression).
func TestSecurity_NoInsecureSkipVerifyInRelay(t *testing.T) {
	data, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("ReadFile server.go: %v", err)
	}
	if strings.Contains(string(data), "InsecureSkipVerify: true") {
		t.Error(`relay/server.go must not contain InsecureSkipVerify: true — Phase 88 SC-4 anti-regression; use loopbackOriginPatterns(r.Host) instead`)
	}
}
