package webserver_test

import (
	"testing"

	"github.com/agenthub/agenthub/internal/webserver"
)

func TestGenerateToken(t *testing.T) {
	tok1, err := webserver.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if tok1 == "" {
		t.Fatal("GenerateToken returned empty string")
	}

	tok2, err := webserver.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken second call returned error: %v", err)
	}
	if tok1 == tok2 {
		t.Error("GenerateToken returned the same value twice (not random)")
	}
}

func TestTokenLength(t *testing.T) {
	tok, err := webserver.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	// 32 bytes base64url-encoded without padding = ceil(32*4/3) = 43 chars
	if len(tok) != 43 {
		t.Errorf("expected token length 43, got %d (token: %q)", len(tok), tok)
	}
}

func TestTokenStoreCreateAndLookup(t *testing.T) {
	ts := webserver.NewTokenStore()
	sessionID := "session-abc-123"

	tok, err := ts.Create(sessionID)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if tok == "" {
		t.Fatal("Create returned empty token")
	}

	gotSession, ok := ts.Lookup(tok)
	if !ok {
		t.Fatal("Lookup returned false for created token")
	}
	if gotSession != sessionID {
		t.Errorf("Lookup returned sessionID %q, expected %q", gotSession, sessionID)
	}
}

func TestTokenStoreLookupInvalid(t *testing.T) {
	ts := webserver.NewTokenStore()

	_, ok := ts.Lookup("nonexistent")
	if ok {
		t.Error("Lookup should return false for nonexistent token")
	}
}

func TestTokenStoreRevoke(t *testing.T) {
	ts := webserver.NewTokenStore()
	sessionID := "session-revoke-test"

	tok, err := ts.Create(sessionID)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	ts.Revoke(tok)

	_, ok := ts.Lookup(tok)
	if ok {
		t.Error("Lookup should return false after Revoke")
	}
}

func TestTokenStoreRevokeBySession(t *testing.T) {
	ts := webserver.NewTokenStore()
	sessionID := "session-bulk-revoke"

	tok1, err := ts.Create(sessionID)
	if err != nil {
		t.Fatalf("Create tok1 returned error: %v", err)
	}
	tok2, err := ts.Create(sessionID)
	if err != nil {
		t.Fatalf("Create tok2 returned error: %v", err)
	}

	ts.RevokeBySession(sessionID)

	_, ok1 := ts.Lookup(tok1)
	_, ok2 := ts.Lookup(tok2)

	if ok1 {
		t.Error("tok1 should not be found after RevokeBySession")
	}
	if ok2 {
		t.Error("tok2 should not be found after RevokeBySession")
	}
}

func TestTokenStoreMultiplePerSession(t *testing.T) {
	ts := webserver.NewTokenStore()
	sessionID := "session-multi"

	tok1, err := ts.Create(sessionID)
	if err != nil {
		t.Fatalf("Create tok1 returned error: %v", err)
	}
	tok2, err := ts.Create(sessionID)
	if err != nil {
		t.Fatalf("Create tok2 returned error: %v", err)
	}

	gotSession1, ok1 := ts.Lookup(tok1)
	gotSession2, ok2 := ts.Lookup(tok2)

	if !ok1 || gotSession1 != sessionID {
		t.Errorf("tok1 lookup failed: ok=%v sessionID=%q", ok1, gotSession1)
	}
	if !ok2 || gotSession2 != sessionID {
		t.Errorf("tok2 lookup failed: ok=%v sessionID=%q", ok2, gotSession2)
	}

	tokens := ts.TokensForSession(sessionID)
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens for session, got %d", len(tokens))
	}
}
