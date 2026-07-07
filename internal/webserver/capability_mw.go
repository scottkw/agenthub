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
// 119 attaches it to /api/files/list, /stat, /read via SetFilesHandler.
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

// requireFilesWrite wraps requireCapability and additionally enforces that
// claims.Perms contains the files.write whole-token capability bit (CAP-02,
// CAP-09) AND that the request passes the CSRF Origin check (CAP-03).
//
// ORDER MATTERS (Pitfall 6): requireCapability runs first — HMAC verify,
// session/grant checks, and claims-attach via capability.WithClaims all
// happen before this wrapper's inner handler executes. Only after those
// succeed does this wrapper extract claims via capability.ClaimsFromContext
// and apply the capability.HasPerm(claims.Perms, capability.PermFilesWrite)
// check, then the CSRF Origin check.
//
// On perm miss the response is 403 with body "files.write capability required"
// (load-bearing contract assertion, SC#1 / CAP-09). The literal substring
// "files.write" in the body allows the frontend to surface a meaningful
// permission-denied message.
//
// The CSRF Origin check (originAllowedForWrite) is the INVERSE of
// requireAllowedOrigin: an absent Origin passes vacuously (desktop Wails
// fetch sends none), while a present-and-mismatched Origin returns 403
// "forbidden". See Pitfall 1 in RESEARCH.md and Critical Inversion 1 in
// PATTERNS.md.
//
// SEPARATION INVARIANT (CAP-02): this is a THIRD separate wrapper — it does
// NOT touch requireCapability or requireFilesRead. Adding files.write to
// requireCapability would break every non-file route (T-124-04). The
// TestRequireCapability_UnchangedByPhase118 static-grep gate pins this.
func (ws *WebServer) requireFilesWrite(next http.HandlerFunc) http.HandlerFunc {
	return ws.requireCapability(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := capability.ClaimsFromContext(r.Context())
		if !ok {
			// requireCapability always attaches claims on success; missing
			// claims here indicates a mis-composed wrapper chain.
			http.Error(w, "files.write capability required", http.StatusForbidden)
			return
		}
		if !capability.HasPerm(claims.Perms, capability.PermFilesWrite) {
			http.Error(w, "files.write capability required", http.StatusForbidden)
			return
		}
		// CSRF Origin check — must run AFTER the cap check so a missing-perm
		// request 403s with the informative body rather than the generic
		// "forbidden" origin error. Ordering: requireCapability (401) →
		// HasPerm (403) → Origin (403). (T-124-03, T-124-05)
		if !ws.originAllowedForWrite(r, claims.SID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

// originAllowedForWrite reports whether the request's Origin header permits a
// write operation for sessionID.
//
// CRITICAL INVERSION (Critical Inversion 1 in PATTERNS.md, Pitfall 1 in
// RESEARCH.md): this is the OPPOSITE of requireAllowedOrigin.
//
// requireAllowedOrigin (WS-upgrade-only, origin_mw.go:31) REJECTS an absent
// Origin header because browsers always send Origin on WebSocket upgrade.
// Write routes are reached by both browser (web-share) and desktop Wails
// fetch(); desktop Wails fetch() sends NO Origin header. Therefore:
//
//   - Absent Origin → return true (pass vacuously; trusted desktop caller).
//   - Present Origin, tailnet BaseURL match → return true (D-03; unaffected
//     by the RW gate — the gate is a Funnel-origin-only defense, D-02).
//   - Present Origin, Funnel URL match → return true ONLY when
//     isRWGated(sessionID) is also true (FNL-09 D-02 defense-in-depth); a
//     non-gated session gets a Funnel-origin write rejected even with a
//     structurally valid files.write capability.
//   - Present Origin with empty BaseURL/FunnelBaseURL (listener not ready,
//     or Funnel inactive) → return false (fail closed; CLAUDE.md "Silent
//     Fallbacks Forbidden" — unchanged Phase 165 FNL-04 posture).
//
// originAllowedForWrite reaches ONLY the files.write HTTP routes via
// requireFilesWrite — it does NOT gate MsgInput/MsgSessionInject at
// handleWSSRelay. The terminal-write gate is grant validity at WS upgrade
// (isGrantActive, sub.ReadOnly derivation — see RemoveGrant/SetRWGate in
// server.go and TestHandleWSSRelay_WriteCap_RequiresGate). Treating this
// function as covering terminal writes is the RESEARCH Pitfall 1
// anti-pattern; it is defense-in-depth for files.write only, since the
// gate-minted write cap never carries files.write perms (D-05).
func (ws *WebServer) originAllowedForWrite(r *http.Request, sessionID string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Desktop Wails fetch() sends no Origin — pass vacuously (CAP-03).
		return true
	}
	// Primary: tailnet URL — exact byte-for-byte match (D-03). Unaffected by
	// the RW gate; the gate is Funnel-origin-only defense-in-depth.
	// Fail closed when BaseURL() is empty (listener not ready with a present
	// Origin — never silently allow).
	allowed := ws.BaseURL()
	if allowed != "" && origin == allowed {
		return true
	}
	// Secondary: Funnel origin — exact match; fail-closed when FunnelBaseURL()==""
	// (T-165-01 / T-165-07 no prefix/substring widen). Additionally requires
	// the session to have passed the public-write consent gate (FNL-09 D-02) —
	// a Funnel-origin write for a non-gated session is rejected even with a
	// structurally valid capability.
	if funnelURL := ws.FunnelBaseURL(); funnelURL != "" && origin == funnelURL {
		return ws.isRWGated(sessionID)
	}
	return false
}
