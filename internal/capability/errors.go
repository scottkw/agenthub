package capability

import "errors"

// Sentinel errors returned by Sign and Verify. Callers should compare using
// errors.Is to decide how to respond (401 vs 500, etc.).
var (
	// ErrMalformedToken indicates the token string is not a valid two-segment
	// base64url(payload).base64url(sig) shape or one of the segments failed to
	// decode. Returned before signature verification is attempted.
	ErrMalformedToken = errors.New("capability: malformed token")

	// ErrInvalidSignature indicates the HMAC signature did not match the
	// expected value computed from the decoded payload and key. Constant-time
	// comparison is used via hmac.Equal.
	ErrInvalidSignature = errors.New("capability: invalid signature")

	// ErrMalformedClaims indicates the signature verified successfully but the
	// payload bytes could not be decoded as a Claims JSON object. This is the
	// post-signature-verification parse failure path.
	ErrMalformedClaims = errors.New("capability: malformed claims")

	// ErrCodeNotFound indicates JoinCodeManager.Exchange was called with a
	// code that was never issued, was already exchanged (single-use), or has
	// been garbage-collected. This is the 404 path for join-code exchange.
	ErrCodeNotFound = errors.New("capability: join code not found")

	// ErrCodeExpired indicates the join code was issued and never exchanged,
	// but the TTL (D-11, 5 minutes) has elapsed. The code is deleted as part
	// of Exchange's cleanup so subsequent calls return ErrCodeNotFound.
	ErrCodeExpired = errors.New("capability: join code expired")
)
