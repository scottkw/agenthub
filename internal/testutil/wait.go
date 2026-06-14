// Package testutil provides small helpers shared across the project's tests.
package testutil

import (
	"testing"
	"time"
)

// WaitFor polls cond until it returns true or timeout elapses, failing the test
// if the condition never holds. It replaces fixed time.Sleep calls that race
// asynchronous events (hub subscriptions, goroutine side effects) and flake
// under load on CI. See issue #80.
//
// cond is evaluated immediately and then every 10ms. msg/args are passed to
// t.Fatalf when the deadline is reached.
func WaitFor(t testing.TB, timeout time.Duration, cond func() bool, msg string, args ...any) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return
		}
		if !time.Now().Before(deadline) {
			t.Fatalf(msg, args...)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
