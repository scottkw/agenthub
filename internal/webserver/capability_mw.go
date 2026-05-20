package webserver

import (
	"net/http"

	"github.com/scottkw/agenthub/internal/capability"
)

// requireCapability returns a middleware that gates access to a handler on a
// valid HMAC-signed capability token supplied via the ?cap= query parameter.
//
// Enforcement sequence (RESEARCH Pattern 3):
//  1. If no ?cap= is present, respond 401 "capability required".
//  2. If the server has no signing key loaded (SetSigningKey not yet called),
//     respond 401. This defends against a misconfigured daemon that never
//     bootstrapped FileKeyStore — we refuse to authenticate rather than
//     accept any token under a nil key.
//  3. Verify the token. Any failure (malformed, bad sig, bad claims) maps to
//     a single generic 401 body — do NOT distinguish between error classes to
//     the caller (information-disclosure defense, T-87-08).
//  4. If the route carries a path parameter "id", reject with 403 "capability
//     does not match session" when claims.SID != pathID (SEC-03).
//  5. Check that claims.GrantID is still in the session's grant list; if not,
//     respond 403 "capability has been revoked" (SEC-04 via D-15).
//  6. Cross-check that the session is still web-enabled. Toggle-off clears
//     grants AND disables web-serving, so either path blocks; this is a
//     defense-in-depth redundancy (in case grants were cleared without
//     disabling, or vice versa).
//  7. On success, attach the verified Claims to the request context so
//     downstream handlers can read Perms / SID / GrantID without re-parsing.
//
// Ordering (RESEARCH Pitfall 5): this middleware MUST wrap mux.HandleFunc
// registrations so it runs BEFORE the handler body — particularly before
// websocket.Accept in handleWSSRelay. Placing the check after Accept is
// ineffective because the 101 Switching Protocols response has already been
// committed by then.
func (ws *WebServer) requireCapability(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("cap")
		if token == "" {
			http.Error(w, "capability required", http.StatusUnauthorized)
			return
		}
		key := ws.currentSigningKey()
		if key == nil {
			http.Error(w, "capability required", http.StatusUnauthorized)
			return
		}
		claims, err := capability.Verify(token, key)
		if err != nil {
			// Collapse all verify failures to a single 401 body — do not leak
			// whether the token was malformed, had a bad signature, or had
			// unparseable claims (T-87-08).
			http.Error(w, "capability required", http.StatusUnauthorized)
			return
		}
		if pathID := r.PathValue("id"); pathID != "" && claims.SID != pathID {
			http.Error(w, "capability does not match session", http.StatusForbidden)
			return
		}
		if !ws.isGrantActive(claims.SID, claims.GrantID) {
			http.Error(w, "capability has been revoked", http.StatusForbidden)
			return
		}
		if !ws.IsSessionEnabled(claims.SID) {
			// Defense in depth: toggle-off clears grants AND disables web
			// serving. Either alone would block this request; checking both
			// guards against a future code path that touches only one.
			http.Error(w, "capability has been revoked", http.StatusForbidden)
			return
		}
		ctx := capability.WithClaims(r.Context(), claims)
		next(w, r.WithContext(ctx))
	}
}

// requireFilesRead wraps requireCapability and additionally enforces that
// claims.Perms contains the files.read whole-token capability bit
// (FS-11, FS-13).
//
// ORDER MATTERS: requireCapability runs first — HMAC verify, session/grant
// checks, and claims-attach via capability.WithClaims all happen before this
// wrapper's inner handler executes. Only after those succeed does this
// wrapper extract claims via capability.ClaimsFromContext and apply the
// capability.HasPerm(claims.Perms, capability.PermFilesRead) check.
//
// On miss, the response is 403 with body "files.read capability required" so
// the Phase 120 frontend can surface a meaningful permission-denied message
// (PITFALLS.md UX Pitfalls — never a generic "Forbidden" UX). The literal
// substring "files.read" in the body is a load-bearing contract assertion
// (REQUIREMENTS.md FS-13, ROADMAP success criterion 5).
//
// SEPARATION INVARIANT: this is a SEPARATE wrapper from requireCapability —
// adding the files.read check to requireCapability itself would break every
// existing terminal/relay/plugin route that does not carry the bit (Pitfall 4
// anti-pattern, T-118-14).
//
// MOUNT TIMING: defined in Phase 118 but not yet mounted on any route. Phase
// 119 attaches it to /api/files/list, /stat, /read via SetFilesHandlerProvider.
// Phase 118 unit-tests it standalone via httptest (TestRequireFilesRead in
// capability_test.go).
func (ws *WebServer) requireFilesRead(next http.HandlerFunc) http.HandlerFunc {
	return ws.requireCapability(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := capability.ClaimsFromContext(r.Context())
		if !ok {
			// requireCapability always attaches claims on the success path it
			// hands to next, so missing claims here would mean the wrapper
			// chain is mis-composed. Treat as a hard 403 to avoid silently
			// trusting an unverified caller.
			http.Error(w, "files.read capability required", http.StatusForbidden)
			return
		}
		if !capability.HasPerm(claims.Perms, capability.PermFilesRead) {
			http.Error(w, "files.read capability required", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}
