//go:build phase87_wave1

// Package capability_test contains RED skeletons for the capability package
// (Phase 87, Plan 02). Every test body calls t.Skip("implemented in plan 02")
// so these files compile once the production package exists but do not yet
// execute any behavior. The build tag gates the entire file from the default
// go test command until Plan 02 removes the tag.
package capability_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/capability"
)

// testKey returns a deterministic 32-byte key for signing in the skeleton
// references. Plan 02 un-skips these tests and implements the production
// symbols referenced here (Sign, Verify, Claims, sentinel errors,
// WithClaims/ClaimsFromContext). The helper is intentionally unexported.
func testKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

// TestSign_RoundTrip asserts that a signed token round-trips back to the
// original claims via Verify when the same key is used.
func TestSign_RoundTrip(t *testing.T) {
	t.Skip("implemented in plan 02")
	claims := capability.Claims{SID: "s1", Perms: "read", IAT: 1, GrantID: "g1", V: 1}
	tok, err := capability.Sign(claims, testKey())
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	got, err := capability.Verify(tok, testKey())
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != claims {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, claims)
	}
}

// TestVerify_RejectsTamperedPayload asserts that flipping a single byte in
// the base64url(claims) segment causes Verify to return ErrInvalidSignature.
func TestVerify_RejectsTamperedPayload(t *testing.T) {
	t.Skip("implemented in plan 02")
	claims := capability.Claims{SID: "s1", Perms: "read", IAT: 1, GrantID: "g1", V: 1}
	tok, err := capability.Sign(claims, testKey())
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	parts := strings.SplitN(tok, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("expected two-segment token, got %q", tok)
	}
	// Flip a byte in the payload segment.
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	raw[0] ^= 0x01
	tampered := base64.RawURLEncoding.EncodeToString(raw) + "." + parts[1]
	if _, err := capability.Verify(tampered, testKey()); !errors.Is(err, capability.ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

// TestVerify_RejectsTamperedSignature asserts that flipping a single byte in
// the base64url(sig) segment causes Verify to return ErrInvalidSignature.
func TestVerify_RejectsTamperedSignature(t *testing.T) {
	t.Skip("implemented in plan 02")
	claims := capability.Claims{SID: "s1", Perms: "read", IAT: 1, GrantID: "g1", V: 1}
	tok, err := capability.Sign(claims, testKey())
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	parts := strings.SplitN(tok, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("expected two-segment token, got %q", tok)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	sig[0] ^= 0x01
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString(sig)
	if _, err := capability.Verify(tampered, testKey()); !errors.Is(err, capability.ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

// TestVerify_RejectsWrongKey asserts that a token signed with one key fails
// Verify under a different key.
func TestVerify_RejectsWrongKey(t *testing.T) {
	t.Skip("implemented in plan 02")
	claims := capability.Claims{SID: "s1", Perms: "read", IAT: 1, GrantID: "g1", V: 1}
	tok, err := capability.Sign(claims, testKey())
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	other := make([]byte, 32)
	if _, err := rand.Read(other); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if bytes.Equal(other, testKey()) {
		t.Fatal("test key collision — regenerate")
	}
	if _, err := capability.Verify(tok, other); !errors.Is(err, capability.ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature for wrong key, got %v", err)
	}
}

// TestVerify_RejectsMalformedTokenSegmentCount asserts that a token missing
// the "." separator (zero segments) or containing extra separators returns
// ErrMalformedToken.
func TestVerify_RejectsMalformedTokenSegmentCount(t *testing.T) {
	t.Skip("implemented in plan 02")
	for _, tok := range []string{"", "no-dot-here", "a.b.c"} {
		if _, err := capability.Verify(tok, testKey()); !errors.Is(err, capability.ErrMalformedToken) {
			t.Errorf("Verify(%q): expected ErrMalformedToken, got %v", tok, err)
		}
	}
}

// TestVerify_RejectsMalformedBase64 asserts that a token with invalid base64url
// in either segment returns ErrMalformedToken.
func TestVerify_RejectsMalformedBase64(t *testing.T) {
	t.Skip("implemented in plan 02")
	// "!!!" is not valid base64url.
	for _, tok := range []string{"!!!.aGVsbG8", "aGVsbG8.!!!"} {
		if _, err := capability.Verify(tok, testKey()); !errors.Is(err, capability.ErrMalformedToken) {
			t.Errorf("Verify(%q): expected ErrMalformedToken, got %v", tok, err)
		}
	}
}

// TestVerify_RejectsMalformedClaimsJSON asserts that a token whose payload
// decodes to non-JSON bytes returns ErrMalformedClaims (after the signature
// verifies successfully against that payload).
func TestVerify_RejectsMalformedClaimsJSON(t *testing.T) {
	t.Skip("implemented in plan 02")
	// The test harness in Plan 02 will construct a token with non-JSON payload
	// signed correctly, then assert Verify returns ErrMalformedClaims. This
	// exercises the path where HMAC matches but JSON unmarshal fails.
	_ = capability.ErrMalformedClaims
}

// TestVerify_ConstantTimeComparison asserts (via source grep or reflect) that
// Verify uses hmac.Equal for signature comparison rather than bytes.Equal or
// ==. Plan 02 will satisfy this by using hmac.Equal verbatim per RESEARCH
// Pattern 1 and Don't-Hand-Roll table.
func TestVerify_ConstantTimeComparison(t *testing.T) {
	t.Skip("implemented in plan 02")
}

// TestClaims_Context_RoundTrip asserts that WithClaims attaches Claims to a
// context.Context and ClaimsFromContext retrieves them. Zero-value claims and
// the "not present" path are also covered.
func TestClaims_Context_RoundTrip(t *testing.T) {
	t.Skip("implemented in plan 02")
	ctx := context.Background()
	if _, ok := capability.ClaimsFromContext(ctx); ok {
		t.Error("expected ok=false on empty context")
	}
	want := capability.Claims{SID: "s1", Perms: "read,write", IAT: 42, GrantID: "g1", V: 1}
	ctx = capability.WithClaims(ctx, want)
	got, ok := capability.ClaimsFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true after WithClaims")
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
