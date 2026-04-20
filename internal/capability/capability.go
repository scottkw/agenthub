// Package capability issues and verifies HMAC-SHA256 capability tokens that
// gate web access to individual PTY sessions. Tokens encode a compact claim
// set as base64url(claimsJSON).base64url(sig). See SEC-01..SEC-05.
package capability

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Claims is the capability token payload. Field declaration order is load-bearing:
// the JSON marshaller emits fields in declaration order, and the HMAC signs the
// exact encoded bytes. Changing the order would break verification of previously
// issued tokens. Add new fields only with json:",omitempty" and bump V.
type Claims struct {
	SID     string `json:"sid"`      // session ID the capability is bound to (SEC-03)
	Perms   string `json:"perms"`    // "read" or "read,write" (SEC-04)
	IAT     int64  `json:"iat"`      // issued-at UNIX timestamp
	GrantID string `json:"grant_id"` // 128-bit random ID, enables future revocation
	V       int    `json:"v"`        // claim schema version; always 1 in v3.1
}

// Sign serialises claims as canonical JSON, computes HMAC-SHA256 with key, and
// returns a two-segment base64url token in the form "b64Payload.b64Sig". The
// key must be the 32-byte signing key loaded via KeyStore.
func Sign(c Claims, key []byte) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	sig := mac.Sum(nil)
	b64Payload := base64.RawURLEncoding.EncodeToString(payload)
	b64Sig := base64.RawURLEncoding.EncodeToString(sig)
	return b64Payload + "." + b64Sig, nil
}

// Verify parses and authenticates a token against key, returning the decoded
// Claims on success. Failures map to the sentinel errors:
//   - ErrMalformedToken: wrong segment count or base64url decode failure
//   - ErrInvalidSignature: HMAC mismatch (constant-time comparison via hmac.Equal)
//   - ErrMalformedClaims: signature OK but payload is not a valid Claims JSON object
//
// Signature comparison uses hmac.Equal for constant-time behaviour — never a
// non-constant-time byte comparison — to avoid leaking key material through
// timing side channels.
func Verify(token string, key []byte) (Claims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return Claims{}, fmt.Errorf("%w: segment count", ErrMalformedToken)
	}
	// Reject tokens with more than one dot: SplitN with n=2 returns at most
	// two parts, but an extra "." in the second part would be absorbed into
	// parts[1] and the base64url decode below would fail, which is the
	// desired ErrMalformedToken path.
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return Claims{}, ErrInvalidSignature
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrMalformedClaims, err)
	}
	return c, nil
}
