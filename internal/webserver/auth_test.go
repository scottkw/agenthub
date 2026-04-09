package webserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scottkw/agenthub/internal/webserver"
)

// TestBasicAuthMiddleware_Unauthorized verifies that a request with no
// Authorization header returns 401 with WWW-Authenticate header.
func TestBasicAuthMiddleware_Unauthorized(t *testing.T) {
	handler := webserver.BasicAuthMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}

	wwwAuth := rr.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("expected WWW-Authenticate header to be set")
	}
	if want := `Basic realm="AgentHub"`; wwwAuth != want {
		t.Errorf("WWW-Authenticate = %q, want %q", wwwAuth, want)
	}
}

// TestBasicAuthMiddleware_WrongPassword verifies that a request with the wrong
// password returns 401.
func TestBasicAuthMiddleware_WrongPassword(t *testing.T) {
	handler := webserver.BasicAuthMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("user", "wrong")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestBasicAuthMiddleware_Authorized verifies that a request with the correct
// password is allowed through regardless of username.
func TestBasicAuthMiddleware_Authorized(t *testing.T) {
	const password = "testpass"
	handler := webserver.BasicAuthMiddleware(password)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Any username should work — only password matters.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("anyuser", password)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// TestBasicAuthMiddleware_EmptyUsername verifies that an empty username is
// accepted as long as the password matches.
func TestBasicAuthMiddleware_EmptyUsername(t *testing.T) {
	const password = "testpass"
	handler := webserver.BasicAuthMiddleware(password)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("", password)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with empty username, got %d", rr.Code)
	}
}
