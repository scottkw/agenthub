//go:build phase87_wave1

// Package capability_test fuzz harness (RED skeleton for Plan 02).
// Seeded with a single known-good token produced by capability.Sign against a
// deterministic 32-byte key. The fuzz body calls capability.Verify against
// arbitrary input and asserts only that no panic occurs — Verify must return
// an error for malformed/tampered input, never panic.
//
// Fuzz entry points cannot be t.Skip'd (the harness discards skipped runs
// silently, masking regressions). Instead the entire file is guarded by the
// phase87_wave1 build tag so it is excluded from `go test ./...` until Plan
// 02 removes the tag and implements Sign/Verify.
package capability_test

import (
	"testing"

	"github.com/scottkw/agenthub/internal/capability"
)

// FuzzVerify exercises capability.Verify against arbitrary token strings. The
// corpus is seeded with exactly one known-good token (per the plan spec) so
// the fuzzer starts from a realistic shape before mutating. The body asserts
// no panic on any input — tampered, random, or valid. A bad signature must
// produce an error, not a crash.
func FuzzVerify(f *testing.F) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	// One known-good seed token — exercise the happy path in the fuzzer's
	// starting corpus so mutations have a realistic structure to perturb.
	good, err := capability.Sign(capability.Claims{SID: "s1", Perms: "read", V: 1}, key)
	if err != nil {
		// Plan 02 implements Sign; if this fails in Wave 0 the build tag
		// excludes the file so we never reach here. Left as a Fatalf to
		// surface any future Plan 02 regression.
		f.Fatalf("Sign seed: %v", err)
	}
	f.Add(good)

	f.Fuzz(func(t *testing.T, token string) {
		// Must never panic; must return an error for malformed/tampered input.
		_, _ = capability.Verify(token, key)
	})
}
