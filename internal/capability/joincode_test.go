//go:build phase87_wave1

// Package capability_test join-code tests (RED skeletons for Plan 02).
// Covers JoinCodeManager.Issue format (D-10 base32 A-Z2-7 dashed),
// single-use Exchange, double-use rejection, 5-minute TTL (D-11), and
// TOCTOU atomicity under concurrent Exchange (RESEARCH Pitfall 4).
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
// matches the D-10 format regex. Plan 02 will implement Issue to encode 5
// random bytes via base32.StdEncoding.WithPadding(NoPadding) split as 4-4.
func TestJoinCodeManager_IssueFormat(t *testing.T) {
	t.Skip("implemented in plan 02")
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
	t.Skip("implemented in plan 02")
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
	t.Skip("implemented in plan 02")
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
// ErrCodeExpired once the 5-minute TTL has elapsed. Plan 02 will expose an
// injectable clock seam (e.g. a now func field) so this test can jump time
// without real sleeps.
func TestJoinCodeManager_ExchangeExpiresAfterTTL(t *testing.T) {
	t.Skip("implemented in plan 02")
	// Plan 02 will set mgr.now = func() time.Time { return fixedTime }
	// and advance it past the TTL before calling Exchange.
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	code, err := mgr.Issue("tok")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	// Simulated time advance will be wired by Plan 02.
	if _, err := mgr.Exchange(code); !errors.Is(err, capability.ErrCodeExpired) {
		t.Errorf("Exchange after TTL: expected ErrCodeExpired, got %v", err)
	}
}

// TestJoinCodeManager_ConcurrentExchangeIsAtomic verifies the TOCTOU property
// from RESEARCH Pitfall 4: 100 goroutines call Exchange on the same code;
// exactly ONE must succeed. A naive lookup-then-delete implementation would
// allow multiple successes.
func TestJoinCodeManager_ConcurrentExchangeIsAtomic(t *testing.T) {
	t.Skip("implemented in plan 02")
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
	t.Skip("implemented in plan 02")
	mgr := capability.NewJoinCodeManager(5 * time.Minute)
	if _, err := mgr.Exchange("AAAA-BBBB"); !errors.Is(err, capability.ErrCodeNotFound) {
		t.Errorf("Exchange(unknown): expected ErrCodeNotFound, got %v", err)
	}
}
