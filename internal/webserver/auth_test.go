package webserver_test

import (
	"net/http"
	"testing"

	"github.com/agenthub/agenthub/internal/webserver"
)

func TestHashPassword(t *testing.T) {
	hash, err := webserver.HashPassword("test123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("HashPassword returned empty hash")
	}
	// bcrypt hash should start with $2a$
	if hash[0] != '$' {
		t.Errorf("expected bcrypt hash starting with $, got %s", string(hash[:4]))
	}
}

func TestCheckPassword(t *testing.T) {
	hash, err := webserver.HashPassword("test123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !webserver.CheckPassword(hash, "test123") {
		t.Error("CheckPassword returned false for correct password")
	}
	if webserver.CheckPassword(hash, "wrong") {
		t.Error("CheckPassword returned true for wrong password")
	}
}

func TestAuthManagerSetPassword(t *testing.T) {
	am := webserver.NewAuthManager()

	if am.IsPasswordSet() {
		t.Error("IsPasswordSet should be false before SetPassword")
	}

	err := am.SetPassword("mypassword")
	if err != nil {
		t.Fatalf("SetPassword returned error: %v", err)
	}

	if !am.IsPasswordSet() {
		t.Error("IsPasswordSet should be true after SetPassword")
	}
}

func TestAuthManagerLogin(t *testing.T) {
	am := webserver.NewAuthManager()
	err := am.SetPassword("correctpassword")
	if err != nil {
		t.Fatalf("SetPassword returned error: %v", err)
	}

	// Correct password
	cookieVal, err := am.Login("correctpassword")
	if err != nil {
		t.Fatalf("Login with correct password returned error: %v", err)
	}
	if cookieVal == "" {
		t.Error("Login with correct password returned empty cookie value")
	}

	// Wrong password
	cookieVal2, err2 := am.Login("wrongpassword")
	if err2 == nil {
		t.Error("Login with wrong password should return error")
	}
	if cookieVal2 != "" {
		t.Error("Login with wrong password should return empty cookie value")
	}
}

func TestAuthManagerIsAuthenticated(t *testing.T) {
	am := webserver.NewAuthManager()
	err := am.SetPassword("pass")
	if err != nil {
		t.Fatalf("SetPassword returned error: %v", err)
	}

	cookieVal, err := am.Login("pass")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	// Valid cookie
	if !am.IsAuthenticated(cookieVal) {
		t.Error("IsAuthenticated should return true for valid cookie")
	}

	// Random cookie
	if am.IsAuthenticated("randomgarbage") {
		t.Error("IsAuthenticated should return false for random cookie")
	}

	// Empty cookie
	if am.IsAuthenticated("") {
		t.Error("IsAuthenticated should return false for empty cookie")
	}
}

func TestSessionCookieProperties(t *testing.T) {
	am := webserver.NewAuthManager()
	cookie := am.MakeSessionCookie("testvalue")

	if cookie.Name != "agenthub_session" {
		t.Errorf("expected cookie name 'agenthub_session', got %q", cookie.Name)
	}
	if cookie.Value != "testvalue" {
		t.Errorf("expected cookie value 'testvalue', got %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if !cookie.Secure {
		t.Error("cookie should be Secure")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSite=Strict, got %v", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Errorf("expected cookie path '/', got %q", cookie.Path)
	}
}
