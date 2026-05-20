// Package capability_test covers the capability package's Sign/Verify
// round-trip, tamper detection, wrong-key rejection, malformed-input handling,
// constant-time comparison sentinel, and Claims context round-trip.
package capability_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/capability"
)

// testKey returns a deterministic 32-byte key for signing across tests.
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
// ErrMalformedToken. The three-segment case ("a.b.c") is rejected because
// SplitN(..., 2) yields parts[1]="b.c" which fails base64url decode — still
// an ErrMalformedToken path.
func TestVerify_RejectsMalformedTokenSegmentCount(t *testing.T) {
	for _, tok := range []string{"", "no-dot-here", "a.b.c"} {
		if _, err := capability.Verify(tok, testKey()); !errors.Is(err, capability.ErrMalformedToken) {
			t.Errorf("Verify(%q): expected ErrMalformedToken, got %v", tok, err)
		}
	}
}

// TestVerify_RejectsMalformedBase64 asserts that a token with invalid base64url
// in either segment returns ErrMalformedToken.
func TestVerify_RejectsMalformedBase64(t *testing.T) {
	// "!!!" is not valid base64url.
	for _, tok := range []string{"!!!.aGVsbG8", "aGVsbG8.!!!"} {
		if _, err := capability.Verify(tok, testKey()); !errors.Is(err, capability.ErrMalformedToken) {
			t.Errorf("Verify(%q): expected ErrMalformedToken, got %v", tok, err)
		}
	}
}

// TestVerify_RejectsMalformedClaimsJSON asserts that a token whose payload
// decodes to non-JSON bytes returns ErrMalformedClaims (after the signature
// verifies successfully against that payload). We build the token by hand:
// a non-JSON payload is HMAC-signed with the same key, base64url-encoded,
// and concatenated. Verify's HMAC check passes; the JSON unmarshal fails.
func TestVerify_RejectsMalformedClaimsJSON(t *testing.T) {
	key := testKey()
	payload := []byte("not valid json {{{")
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	sig := mac.Sum(nil)
	tok := base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)

	_, err := capability.Verify(tok, key)
	if !errors.Is(err, capability.ErrMalformedClaims) {
		t.Errorf("expected ErrMalformedClaims, got %v", err)
	}
}

// TestVerify_ConstantTimeComparison asserts via source inspection that Verify
// uses hmac.Equal for signature comparison rather than bytes.Equal or ==.
// This guards against a subtle regression where timing-side-channel resistance
// could be removed without any behavioural test failing.
func TestVerify_ConstantTimeComparison(t *testing.T) {
	data, err := os.ReadFile("capability.go")
	if err != nil {
		t.Fatalf("ReadFile capability.go: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "hmac.Equal") {
		t.Error("capability.go must call hmac.Equal for signature comparison")
	}
	if strings.Contains(src, "bytes.Equal") {
		t.Error("capability.go must not use bytes.Equal on signature bytes (timing side channel)")
	}
}

// TestHasPerm asserts that HasPerm uses whole-token comma-split semantics,
// never strings.Contains. The "no-files.read" false-positive case is the
// critical Pitfall 4 guard from RESEARCH.md: a substring-based check would
// match "files.read" inside "no-files.read" and silently grant access.
//
// Source: 118-03-PLAN.md Task 1 behavior table; RESEARCH.md Pattern 4 and
// Pitfall 4; PITFALLS.md Pitfall 4 (Capability Token files.read Bit) and
// Pitfall 3 (HasPerm false-positive via substring match).
func TestHasPerm(t *testing.T) {
	cases := []struct {
		name  string
		perms string
		perm  string
		want  bool
	}{
		{"empty perms", "", "files.read", false},
		{"empty target perm", "read,files.read", "", false},
		{"read,write does not grant files.read", "read,write", "files.read", false},
		{"read,files.read grants files.read", "read,files.read", "files.read", true},
		{"position-independent — files.read first", "files.read,read", "files.read", true},
		{"no-files.read false-positive guard", "no-files.read", "files.read", false},
		{"no-files.read with siblings false-positive guard", "no-files.read,read", "files.read", false},
		{"suffix attack", "files.read.extra", "files.read", false},
		{"plain read whole-token", "read", "read", true},
		{"write present alongside files.read", "read,write,files.read", "write", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := capability.HasPerm(tc.perms, tc.perm); got != tc.want {
				t.Fatalf("HasPerm(%q, %q) = %v, want %v", tc.perms, tc.perm, got, tc.want)
			}
		})
	}
}

// TestPermFilesRead_Constant pins the wire value of PermFilesRead so a future
// refactor cannot silently change the token string and de-couple it from the
// owner-token issuance in internal/daemon/api.go (Plan 05).
func TestPermFilesRead_Constant(t *testing.T) {
	if capability.PermFilesRead != "files.read" {
		t.Fatalf("PermFilesRead = %q, want %q", capability.PermFilesRead, "files.read")
	}
}

// TestHasPerm_NoStringsContains is a source-inspection guard: the HasPerm
// implementation must not use strings.Contains for the whole-token check
// (Pitfall 4). This pairs with the "no-files.read false-positive guard"
// behavioural subtest above — a substring-based implementation would pass the
// guard subtest if the check string were re-shaped, but the source inspection
// catches the regression at the implementation level rather than just the
// observed behaviour.
func TestHasPerm_NoStringsContains(t *testing.T) {
	data, err := os.ReadFile("capability.go")
	if err != nil {
		t.Fatalf("ReadFile capability.go: %v", err)
	}
	src := string(data)
	// Extract just the HasPerm function body — the package may use
	// strings.Contains elsewhere legitimately.
	idx := strings.Index(src, "func HasPerm")
	if idx < 0 {
		t.Fatal("capability.go must declare func HasPerm")
	}
	// Bound the search at the next top-level "\nfunc " or end of file.
	rest := src[idx:]
	end := strings.Index(rest[1:], "\nfunc ")
	if end < 0 {
		end = len(rest)
	} else {
		end = end + 1
	}
	body := rest[:end]
	if !strings.Contains(body, "strings.Split") {
		t.Error("HasPerm body must use strings.Split for whole-token comma-split semantics")
	}
	if strings.Contains(body, "strings.Contains") {
		t.Error("HasPerm body must NOT use strings.Contains (Pitfall 4 — substring false-positive)")
	}
}

// TestClaims_Context_RoundTrip asserts that WithClaims attaches Claims to a
// context.Context and ClaimsFromContext retrieves them. Zero-value claims and
// the "not present" path are also covered.
func TestClaims_Context_RoundTrip(t *testing.T) {
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
