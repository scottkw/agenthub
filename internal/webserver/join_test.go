// Package webserver_test covers the public POST /join/exchange handler
// (handleJoinExchange, server.go). This file proves the FNL-08 reusable
// join-code contract at the HTTP boundary that an anonymous public-share
// recipient actually hits — not just at the JoinCodeManager unit layer
// (internal/capability/joincode_test.go).
package webserver_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/webserver"
)

// joinExchangeTestServer wires a WebServer with a signing key, a fresh
// JoinCodeManager, and sid enabled — the minimum wiring handleJoinExchange
// needs (SetSigningKey for capability.Verify, SetJoinCodes for jc.Exchange,
// EnableSession so the post-verify IsSessionEnabled gate passes). Returns
// the signing key too, since the caller must sign tokens with the exact
// same key installed on ws (SetSigningKey has no getter, by design).
func joinExchangeTestServer(t *testing.T, sid string) (*webserver.WebServer, *http.Client, *capability.JoinCodeManager, []byte) {
	t.Helper()
	manager := relay.NewHubManager()
	tlsCfg, client := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })

	key, err := capability.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	ws.SetSigningKey(key)

	jc := capability.NewJoinCodeManager(5 * time.Minute)
	ws.SetJoinCodes(jc)

	ws.EnableSession(sid)

	// noRedirectClient reuses the CA-trusting transport from
	// selfSignedTLSForTest but stops at 3xx without following, so the test
	// can observe the Location header emitted by handleJoinExchange.
	noRedirectClient := &http.Client{
		Transport: client.Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return ws, noRedirectClient, jc, key
}

// signReadOnly mints a read-only capability token bound to sid. T-170-01:
// this plan's fixtures must NEVER bind a write token to a reusable code —
// read-only-scope enforcement at the mint site is 170-02's job, but this
// plan's own tests must not accidentally exercise a write-scoped reusable
// code.
func signReadOnly(t *testing.T, sid string, key []byte) string {
	t.Helper()
	claims := capability.Claims{
		SID:     sid,
		Perms:   "read",
		IAT:     time.Now().Unix(),
		GrantID: "join-test-grant-" + sid,
		V:       1,
	}
	tok, err := capability.Sign(claims, key)
	if err != nil {
		t.Fatalf("capability.Sign: %v", err)
	}
	return tok
}

// postJoinCode POSTs form field code=<code> to /join/exchange and returns
// the response (redirects not followed by the caller's client).
func postJoinCode(t *testing.T, client *http.Client, baseURL, code string) *http.Response {
	t.Helper()
	resp, err := client.PostForm(baseURL+"/join/exchange", url.Values{"code": {code}})
	if err != nil {
		t.Fatalf("POST /join/exchange: %v", err)
	}
	return resp
}

// TestJoinExchange_ReusableCodeSurvivesTwoExchanges proves the core FNL-08
// property at the public HTTP boundary: a reusable read-only join code
// resolves on a second POST /join/exchange, not just the first. This is the
// exact UAT dead-end FNL-08 closes — a public share recipient must be able
// to hit the join link more than once (new tab, reload, second viewer)
// without landing on /join?error=.
func TestJoinExchange_ReusableCodeSurvivesTwoExchanges(t *testing.T) {
	const sid = "pub-sess-1"
	ws, client, jc, key := joinExchangeTestServer(t, sid)
	baseURL := ws.BaseURL()

	rTok := signReadOnly(t, sid, key)
	code, err := jc.IssueReusable(rTok, time.Hour)
	if err != nil {
		t.Fatalf("IssueReusable: %v", err)
	}

	wantLocation := "/sessions/" + sid + "?cap=" + rTok

	// First exchange.
	resp1 := postJoinCode(t, client, baseURL, code)
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusSeeOther {
		t.Fatalf("first exchange: expected 303, got %d", resp1.StatusCode)
	}
	loc1 := resp1.Header.Get("Location")
	if loc1 != wantLocation {
		t.Errorf("first exchange: Location = %q, want %q", loc1, wantLocation)
	}
	if strings.HasPrefix(loc1, "/join?error=") {
		t.Fatalf("first exchange: unexpected error redirect %q", loc1)
	}

	// Second exchange — the exact FNL-08 assertion. A single-use code would
	// 303 to /join?error=invalid here; a reusable code must succeed again.
	resp2 := postJoinCode(t, client, baseURL, code)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusSeeOther {
		t.Fatalf("second exchange: expected 303, got %d", resp2.StatusCode)
	}
	loc2 := resp2.Header.Get("Location")
	if loc2 != wantLocation {
		t.Errorf("second exchange: Location = %q, want %q", loc2, wantLocation)
	}
	if strings.HasPrefix(loc2, "/join?error=") {
		t.Errorf("second exchange: unexpected error redirect %q (reusable code should survive a second exchange)", loc2)
	}
}

// TestJoinExchange_SingleUseCodeStillFailsOnSecondExchange is the negative
// guard: a code minted with the existing single-use jc.Issue path must still
// succeed once then fail on a second POST, proving the reusable branch did
// not accidentally make ALL codes reusable (T-170-04).
func TestJoinExchange_SingleUseCodeStillFailsOnSecondExchange(t *testing.T) {
	const sid = "pub-sess-2"
	ws, client, jc, key := joinExchangeTestServer(t, sid)
	baseURL := ws.BaseURL()

	rTok := signReadOnly(t, sid, key)
	code, err := jc.Issue(rTok)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	wantLocation := "/sessions/" + sid + "?cap=" + rTok

	// First exchange succeeds.
	resp1 := postJoinCode(t, client, baseURL, code)
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusSeeOther {
		t.Fatalf("first exchange: expected 303, got %d", resp1.StatusCode)
	}
	if loc1 := resp1.Header.Get("Location"); loc1 != wantLocation {
		t.Errorf("first exchange: Location = %q, want %q", loc1, wantLocation)
	}

	// Second exchange must fail — single-use code was consumed.
	resp2 := postJoinCode(t, client, baseURL, code)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusSeeOther {
		t.Fatalf("second exchange: expected 303, got %d", resp2.StatusCode)
	}
	loc2 := resp2.Header.Get("Location")
	if loc2 != "/join?error=invalid" {
		t.Errorf("second exchange: Location = %q, want %q", loc2, "/join?error=invalid")
	}
}
