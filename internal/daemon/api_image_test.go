package daemon

import "testing"

// TestHandleSetImageConfig_RangeRejected — Plan 96-02 implements.
// PATCH /settings/image-config with StorageLimit <= 0 OR > 1000
// must return 400 Bad Request; sibling fields untouched.
// Per 96-PATTERNS.md §`internal/daemon/api.go` Adapt block: validate
// req.StorageLimit > 0 && req.StorageLimit <= 1000.
func TestHandleSetImageConfig_RangeRejected(t *testing.T) {
	t.Skip("Pending until Plan 96-02 implements api.handleSetImageConfig with [1, 1000] range gate (96-VALIDATION row IMG-02 PATCH validates [1, 1000]).")
}

// TestHandleSetImageConfig_ValidAccepted — Plan 96-02 implements.
// PATCH /settings/image-config with StorageLimit=16 returns 204
// No Content; persisted struct reflects the new value; sibling
// settings unchanged.
func TestHandleSetImageConfig_ValidAccepted(t *testing.T) {
	t.Skip("Pending until Plan 96-02 implements api.handleSetImageConfig success path (96-VALIDATION row IMG-02 PATCH 204 success).")
}
