package main

// Phase 175 Plan 02 Task 1 — Wave 0 test scaffolding for BUG-03 (#126).
//
// shouldContinuePolling is the pure, injectable-clock helper extracted from
// pollSessionStatus's inline `time.Now().Before(deadline)` loop condition
// (app.go:386 pre-extraction). This test pins the pre-existing 300s deadline
// semantics so 175-05's behavioral fix (re-arming the deadline / removing the
// fixed window) has a regression guard proving no accidental behavior change
// slips through the refactor itself (T-175-02-01).

import (
	"testing"
	"time"
)

func TestShouldContinuePolling(t *testing.T) {
	pollStart := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	const maxWindow = 300 * time.Second

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "within window returns true",
			now:  pollStart.Add(150 * time.Second),
			want: true,
		},
		{
			name: "just before deadline returns true",
			now:  pollStart.Add(maxWindow - time.Millisecond),
			want: true,
		},
		{
			name: "at deadline returns false",
			now:  pollStart.Add(maxWindow),
			want: false,
		},
		{
			name: "after deadline returns false",
			now:  pollStart.Add(maxWindow + time.Second),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldContinuePolling(pollStart, tc.now, maxWindow); got != tc.want {
				t.Errorf("shouldContinuePolling(%v, %v, %v) = %v, want %v",
					pollStart, tc.now, maxWindow, got, tc.want)
			}
		})
	}
}

func TestShouldContinuePolling_ZeroWindow(t *testing.T) {
	pollStart := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	// A zero maxWindow means the deadline equals pollStart itself, so any
	// "now" at or after pollStart must return false — there is no window to
	// poll within.
	if got := shouldContinuePolling(pollStart, pollStart, 0); got != false {
		t.Errorf("shouldContinuePolling(pollStart, pollStart, 0) = %v, want false", got)
	}
	if got := shouldContinuePolling(pollStart, pollStart.Add(time.Second), 0); got != false {
		t.Errorf("shouldContinuePolling(pollStart, pollStart+1s, 0) = %v, want false", got)
	}
}
