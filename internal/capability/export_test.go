package capability

import "time"

// SetClockForTest replaces the JoinCodeManager's internal clock with the
// supplied closure. Only compiled during `go test` (file suffix _test.go) so
// production code cannot import or call this helper.
func (m *JoinCodeManager) SetClockForTest(now func() time.Time) {
	m.mu.Lock()
	m.now = now
	m.mu.Unlock()
}
