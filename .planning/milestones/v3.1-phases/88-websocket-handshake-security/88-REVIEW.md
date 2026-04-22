---
phase: 88-websocket-handshake-security
reviewed: 2026-04-22T02:31:35Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - internal/webserver/origin_mw.go
  - internal/webserver/origin_mw_test.go
  - internal/webserver/origin_integration_test.go
  - internal/webserver/security_regression_test.go
  - internal/webserver/server.go
  - internal/webserver/capability_test.go
  - internal/webserver/server_test.go
  - internal/relay/server.go
  - internal/relay/origin_test.go
  - internal/relay/security_regression_test.go
findings:
  critical: 0
  warning: 0
  info: 4
  total: 4
status: clean
---

# Phase 88: Code Review Report

**Reviewed:** 2026-04-22T02:31:35Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** clean

## Summary

Phase 88 implements SEC-06 correctly. The webserver-side `requireAllowedOrigin`
middleware performs a strict byte-for-byte equality check against
`ws.BaseURL()`, rejects missing Origin headers (D-05), fails closed when the
listener is not yet ready, and is composed OUTSIDE `requireCapability` so that
cross-site upgrades short-circuit before any HMAC signature work runs. The
library-layer `OriginPatterns` is wired to the same `ws.allowedOrigins()`
singleton as belt-and-suspenders (D-12). On the relay side, the accept-all
`InsecureSkipVerify: true` has been replaced with a 4-element loopback
allowlist derived from `r.Host`, consistent with the daemon's 127.0.0.1 bind.

Test coverage is thorough across all required sampling points: matching /
mismatched / missing / case-variant / literal-"null" Origins for the webserver,
both loopback forms (127.0.0.1 + localhost) and cross-site rejection for the
relay, an explicit short-circuit-ordering proof, an information-disclosure
body-leak test, a belt-and-suspenders library-layer test, and source-grep
regression guards that fail if the accept-all patterns are reintroduced.

No critical or warning-level issues were found. The items below are low-value
observations and suggestions; the phase is ready to merge as-is.

## Info

### IN-01: Regression guards are literal-string greps and can be evaded

**File:** `internal/webserver/security_regression_test.go:28`, `internal/relay/security_regression_test.go:25`
**Issue:** Both anti-regression tests compare source bytes against exact
literals (`OriginPatterns: []string{"*"}` and `InsecureSkipVerify: true`). A
future maintainer who writes the equivalent through an intermediate variable
or an imported constant — e.g. `allowAll := []string{"*"}; ...
OriginPatterns: allowAll` or `InsecureSkipVerify: *flag` — would bypass the
grep without restoring the vulnerability pattern visually. This is a known
tradeoff of source-grep guards (called out in 88-DISCUSSION-LOG), but worth
recording so future reviewers know the guard is a tripwire, not a proof.
**Fix:** Optionally supplement with a behavioral test that boots the
webserver / relay and asserts that a cross-site `Origin: https://evil.example`
is rejected even when the maintainer has manually rewritten the AcceptOptions
literal. The existing `TestSecurity_WebSocketRejectsCrossSiteOrigin` and
`TestServer_CrossSiteOriginRejected` already provide that behavioral
coverage — no code change required, just acknowledge in a comment that the
grep guard is backstop-only.

### IN-02: Relay `r.Host`-derived allowlist is attacker-writable but safe by loopback bind

**File:** `internal/relay/server.go:74`, `internal/relay/server.go:167-178`
**Issue:** `loopbackOriginPatterns(r.Host)` derives the allowlist port from
the client-supplied Host header, which an HTTP client can forge freely. The
code comment documents the assumption that the daemon binds to 127.0.0.1 (in
`internal/daemon/api.go:137`, verified), so any connecting client is already
loopback — a local attacker who forges Host can also forge Origin, but they
also already have full loopback access. In the landmine case where the
daemon is ever rebound to a non-loopback interface, a browser CSRF attack
from `https://evil.example` still fails because the victim's browser sets
Host to the real listener address and Origin to `https://evil.example`, which
does not match any of the four loopback patterns. The trust model is sound;
this is a note for future reviewers rather than a defect.
**Fix:** None required. Consider adding a one-line comment in
`loopbackOriginPatterns` explicitly stating "r.Host is client-controlled;
this is safe only because (a) the daemon binds loopback-only and (b) the
allowlist patterns are restricted to loopback hostnames regardless of the
port the attacker picks." This compresses the reasoning into the helper
itself rather than spreading it between the comment at line 62-73 and the
helper at line 158-166.

### IN-03: Typo in capability_test.go comment

**File:** `internal/webserver/capability_test.go:179`
**Issue:** The assertion message reads `"reconnect without readonry flag
must still block write"` — "readonry" is a typo for "readonly". The test
itself is correct; only the failure-message string is affected. Pre-existing
from Phase 87, not introduced by Phase 88, but noted here because Phase 88
edits surround this function (lines 163-170 set the new Origin header).
**Fix:** Change `"readonry"` to `"readonly"` in the assertion string.

### IN-04: `allowedOrigins` helper comment could be sharper about nil semantics

**File:** `internal/webserver/origin_mw.go:53-66`
**Issue:** The doc comment explains that returning `nil` prevents the
library from silently accepting via its same-Host fallback, which is
correct. However, a quick reader might miss that the middleware's own
early-reject on `allowed == ""` is what actually guarantees safety —
`allowedOrigins()` returning nil is the second line of defense. The two
layers reinforce each other but the comment presents them in reverse order
of fire (library-fallback concern first, middleware short-circuit
implicitly).
**Fix:** Reorder the sentence or add a cross-reference: "The middleware at
line 42 already rejects when `BaseURL() == ""`, so any request that reaches
the library layer has bypassed it — a bug we want to fail, not paper over."
Non-blocking; code behavior is correct.

---

_Reviewed: 2026-04-22T02:31:35Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
