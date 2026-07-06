// Package capability_test join-code tests. Covers JoinCodeManager.Issue
// format (D-10 base32 A-Z2-7 dashed), single-use Exchange, double-use
// rejection, 5-minute TTL (D-11), and TOCTOU atomicity under concurrent
// Exchange (RESEARCH Pitfall 4).
package capability_test

import (
	"errors"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
)

// joinCodeRegex matches the D-10 wire format: two groups of 4 base32 chars
// (A-Z and 2-7) separated by a single dash. Matches "A7K4-P2N3" etc.
var joinCodeRegex = regexp.MustCompile(`^[A-Z2-7]{4}-[A-Z2-7]{4}$`)

// TestJoinCodeManager_IssueFormat asserts the join code returned by Issue
// matches the D-10 format regex.
func TestJoinCodeManager_IssueFormat(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	code, err := mgr.Issue("tok")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if !joinCodeRegex.MatchString(code) {
		t.Errorf("code %q does not match %s", code, joinCodeRegex)
	}
}

// TestJoinCodeManager_ExchangeSucceedsOnce asserts the first Exchange call
// with a valid, unexpired code returns the originally-issued token.
func TestJoinCodeManager_ExchangeSucceedsOnce(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	code, err := mgr.Issue("tok-xyz")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	got, err := mgr.Exchange(code)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if got != "tok-xyz" {
		t.Errorf("Exchange returned %q, want %q", got, "tok-xyz")
	}
}

// TestJoinCodeManager_ExchangeRejectsDoubleUse asserts that the second
// Exchange call with the same code returns ErrCodeNotFound (D-11 single-use).
// This is a core property: the code is consumed on first success.
func TestJoinCodeManager_ExchangeRejectsDoubleUse(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	code, err := mgr.Issue("tok")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := mgr.Exchange(code); err != nil {
		t.Fatalf("first Exchange: %v", err)
	}
	if _, err := mgr.Exchange(code); !errors.Is(err, capability.ErrCodeNotFound) {
		t.Errorf("second Exchange: expected ErrCodeNotFound, got %v", err)
	}
}

// TestJoinCodeManager_ExchangeExpiresAfterTTL asserts that Exchange returns
// ErrCodeExpired once the 5-minute TTL has elapsed. Uses the SetClockForTest
// seam (defined in export_test.go) so the test does not rely on real sleeps.
func TestJoinCodeManager_ExchangeExpiresAfterTTL(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := start
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	mgr.SetClockForTest(func() time.Time { return clock })

	code, err := mgr.Issue("tok")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	// Advance the clock past the TTL.
	clock = start.Add(6 * time.Minute)

	if _, err := mgr.Exchange(code); !errors.Is(err, capability.ErrCodeExpired) {
		t.Errorf("Exchange after TTL: expected ErrCodeExpired, got %v", err)
	}
	// Subsequent Exchange must return ErrCodeNotFound (entry was deleted).
	if _, err := mgr.Exchange(code); !errors.Is(err, capability.ErrCodeNotFound) {
		t.Errorf("Exchange after expiry cleanup: expected ErrCodeNotFound, got %v", err)
	}
}

// TestJoinCodeManager_ConcurrentExchangeIsAtomic verifies the TOCTOU property
// from RESEARCH Pitfall 4: 100 goroutines call Exchange on the same code;
// exactly ONE must succeed. A naive lookup-then-delete implementation would
// allow multiple successes.
func TestJoinCodeManager_ConcurrentExchangeIsAtomic(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	code, err := mgr.Issue("tok")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	const N = 100
	var successes atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := mgr.Exchange(code); err == nil {
				successes.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 successful Exchange, got %d", got)
	}
}

// TestJoinCodeManager_ExchangeRejectsUnknownCode asserts that a code that
// was never issued returns ErrCodeNotFound (distinct from ErrCodeExpired).
func TestJoinCodeManager_ExchangeRejectsUnknownCode(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	if _, err := mgr.Exchange("AAAA-BBBB"); !errors.Is(err, capability.ErrCodeNotFound) {
		t.Errorf("Exchange(unknown): expected ErrCodeNotFound, got %v", err)
	}
}

// TestJoinCodeManager_IssueReusable_MultiExchange asserts that a code minted
// via IssueReusable survives repeated Exchange calls — the core FNL-08
// property that distinguishes it from the single-use Issue path.
func TestJoinCodeManager_IssueReusable_MultiExchange(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	code, err := mgr.IssueReusable("tok-reusable", time.Hour)
	if err != nil {
		t.Fatalf("IssueReusable: %v", err)
	}
	if !joinCodeRegex.MatchString(code) {
		t.Errorf("code %q does not match %s", code, joinCodeRegex)
	}

	got1, err := mgr.Exchange(code)
	if err != nil {
		t.Fatalf("first Exchange: %v", err)
	}
	if got1 != "tok-reusable" {
		t.Errorf("first Exchange returned %q, want %q", got1, "tok-reusable")
	}

	got2, err := mgr.Exchange(code)
	if err != nil {
		t.Fatalf("second Exchange: expected success (reusable), got error: %v", err)
	}
	if got2 != "tok-reusable" {
		t.Errorf("second Exchange returned %q, want %q", got2, "tok-reusable")
	}
}

// TestJoinCodeManager_Revoke asserts that Revoke immediately invalidates a
// reusable code — the next Exchange returns ErrCodeNotFound.
func TestJoinCodeManager_Revoke(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	code, err := mgr.IssueReusable("tok-revoke", time.Hour)
	if err != nil {
		t.Fatalf("IssueReusable: %v", err)
	}
	// Sanity: the code resolves before Revoke.
	if _, err := mgr.Exchange(code); err != nil {
		t.Fatalf("Exchange before Revoke: %v", err)
	}

	mgr.Revoke(code)

	if _, err := mgr.Exchange(code); !errors.Is(err, capability.ErrCodeNotFound) {
		t.Errorf("Exchange after Revoke: expected ErrCodeNotFound, got %v", err)
	}
}

// TestJoinCodeManager_Revoke_UnknownCodeIsNoOp asserts that revoking a code
// that was never issued does not panic and leaves the manager usable.
func TestJoinCodeManager_Revoke_UnknownCodeIsNoOp(t *testing.T) {
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	mgr.Revoke("ZZZZ-ZZZZ") // must not panic

	// Manager remains fully functional afterward.
	code, err := mgr.Issue("tok")
	if err != nil {
		t.Fatalf("Issue after no-op Revoke: %v", err)
	}
	if _, err := mgr.Exchange(code); err != nil {
		t.Fatalf("Exchange after no-op Revoke: %v", err)
	}
}

// TestJoinCodeManager_ReusableExpiresAfterTTL asserts that a reusable code
// whose per-call TTL has elapsed returns ErrCodeExpired (and is deleted),
// same as the single-use expiry contract — reusable does not mean immortal.
// Uses SetClockForTest so no real sleeps are involved.
func TestJoinCodeManager_ReusableExpiresAfterTTL(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := start
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	mgr.SetClockForTest(func() time.Time { return clock })

	code, err := mgr.IssueReusable("tok-ttl", time.Hour)
	if err != nil {
		t.Fatalf("IssueReusable: %v", err)
	}
	// Advance the clock past the per-code TTL (1 hour).
	clock = start.Add(2 * time.Hour)

	if _, err := mgr.Exchange(code); !errors.Is(err, capability.ErrCodeExpired) {
		t.Errorf("Exchange after TTL: expected ErrCodeExpired, got %v", err)
	}
	// Subsequent Exchange must return ErrCodeNotFound (entry was deleted).
	if _, err := mgr.Exchange(code); !errors.Is(err, capability.ErrCodeNotFound) {
		t.Errorf("Exchange after expiry cleanup: expected ErrCodeNotFound, got %v", err)
	}
}
