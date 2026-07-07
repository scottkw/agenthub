package capability

import (
	"crypto/rand"
	"encoding/base32"
	"sync"
	"time"
)

// joinCodeEncoding uses the RFC 4648 standard base32 alphabet (A-Z 2-7), no
// padding. This matches D-10: avoids 0/O and 1/I/l ambiguity without any
// custom character-substitution table.
var joinCodeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// joinEntry is an in-memory record of an issued join code. The token field
// holds the already-signed capability that Exchange returns on success.
// reusable distinguishes the two code classes: false (default, zero value)
// is the original single-use code minted by Issue; true is a code minted by
// IssueReusable that survives repeated Exchange calls until its own expiry
// or an explicit Revoke.
type joinEntry struct {
	token    string
	expiry   time.Time
	reusable bool
}

// JoinCodeManager maps short-lived join codes to capability tokens. There are
// two code classes:
//   - Single-use (default, minted by Issue, D-11): the first successful
//     Exchange deletes the entry before returning the token.
//   - Reusable (minted by IssueReusable, FNL-08): the entry survives repeated
//     successful Exchange calls until its own per-code TTL elapses or Revoke
//     is called explicitly. This supports a public share link that must serve
//     an unbounded number of anonymous viewers for the lifetime of the share.
//
// Expired entries (of either class) are removed lazily on the next Exchange
// attempt rather than by a background sweeper; the map stays small because
// TTLs are short-to-moderate and codes are transient artefacts.
//
// mu is a plain sync.Mutex because Exchange must read and (conditionally)
// delete atomically to prevent the TOCTOU race described in RESEARCH Pitfall
// 4. A reader-writer mutex is the wrong primitive here: its read lock would
// allow two goroutines to observe the entry before either deletes it,
// producing two successful exchanges from one single-use code.
type JoinCodeManager struct {
	mu    sync.Mutex
	codes map[string]joinEntry
	ttl   time.Duration
	now   func() time.Time // injection seam for tests; defaults to time.Now
}

// NewJoinCodeManager constructs an empty manager with the supplied TTL. The
// now clock defaults to time.Now; tests may replace it after construction to
// simulate expiry without real sleeps.
func NewJoinCodeManager(ttl time.Duration) *JoinCodeManager {
	return &JoinCodeManager{
		codes: make(map[string]joinEntry),
		ttl:   ttl,
		now:   time.Now,
	}
}

// Issue generates a new 8-character base32 join code in XXXX-XXXX form and
// associates it with the supplied token. The code carries ~40 bits of
// entropy — sufficient for a 5-minute single-use window per D-10/D-11.
func (m *JoinCodeManager) Issue(token string) (string, error) {
	var raw [5]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	// 5 bytes = 40 bits, which encodes to exactly 8 base32 characters.
	encoded := joinCodeEncoding.EncodeToString(raw[:])
	code := encoded[:4] + "-" + encoded[4:8]

	m.mu.Lock()
	m.codes[code] = joinEntry{token: token, expiry: m.now().Add(m.ttl)}
	m.mu.Unlock()
	return code, nil
}

// IssueReusable generates a new 8-character base32 join code in XXXX-XXXX
// form, exactly like Issue (same crypto/rand + joinCodeEncoding path — no
// second RNG or alphabet is introduced, preserving the ~40 bits of entropy),
// but marks the entry reusable so it survives repeated Exchange calls for
// the supplied ttl instead of being deleted on first use. This is the
// primitive behind a public share link (FNL-08): the code must resolve for
// every anonymous viewer for the lifetime of the share, not just the first.
//
// The code is still bounded by ttl — once it elapses, Exchange returns
// ErrCodeExpired and deletes the entry, same as any single-use code. Revoke
// provides the explicit early-termination path (e.g. the user disables the
// share before ttl elapses).
func (m *JoinCodeManager) IssueReusable(token string, ttl time.Duration) (string, error) {
	var raw [5]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	encoded := joinCodeEncoding.EncodeToString(raw[:])
	code := encoded[:4] + "-" + encoded[4:8]

	m.mu.Lock()
	m.codes[code] = joinEntry{token: token, expiry: m.now().Add(ttl), reusable: true}
	m.mu.Unlock()
	return code, nil
}

// IssueSingleUseWithTTL generates a new 8-character base32 join code in
// XXXX-XXXX form, exactly like Issue (same crypto/rand + joinCodeEncoding
// path — no second RNG or alphabet is introduced), but accepts a
// caller-supplied ttl instead of the manager's fixed field. reusable is left
// at its zero value (false), so Exchange keeps its atomic delete-on-first-
// redeem single-use semantics unchanged (R2 concurrency guarantee) — only
// the expiry window is customizable here. This is the primitive behind a
// public write-share code (FNL-09): a 15m/30m/1h write grant must not be
// silently truncated to the manager's fixed 5-minute Issue window (RESEARCH
// Pitfall 4).
func (m *JoinCodeManager) IssueSingleUseWithTTL(token string, ttl time.Duration) (string, error) {
	var raw [5]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	encoded := joinCodeEncoding.EncodeToString(raw[:])
	code := encoded[:4] + "-" + encoded[4:8]

	m.mu.Lock()
	m.codes[code] = joinEntry{token: token, expiry: m.now().Add(ttl)}
	m.mu.Unlock()
	return code, nil
}

// Revoke removes code from the manager immediately, regardless of class or
// remaining TTL. The next Exchange(code) will return ErrCodeNotFound.
// Revoking an unknown or already-removed code is a no-op — deleting an
// absent key from a Go map does not panic or error, so no existence check
// or return value is needed.
func (m *JoinCodeManager) Revoke(code string) {
	m.mu.Lock()
	delete(m.codes, code)
	m.mu.Unlock()
}

// Rebind swaps the token stored for an existing code WITHOUT changing the code
// string, its expiry, or its reusable class. Returns ErrCodeNotFound when the
// code is absent (never issued, expired-and-swept, or revoked) so the caller
// can decide whether to mint a fresh code instead.
//
// This is the FNL-08 CR-01 primitive: a reusable public code already handed to
// viewers must keep resolving after the owner clears+re-issues the session's
// grant set (the "Enable remote file browsing" toggle calls ClearGrants and
// mints a NEW underlying token). Rebind points the stable code string at that
// new token so the code the viewer holds never silently 403s. Expiry is
// preserved so the ~40-bit brute-force window (T-170-03) is not extended.
func (m *JoinCodeManager) Rebind(code, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.codes[code]
	if !ok {
		return ErrCodeNotFound
	}
	entry.token = token
	m.codes[code] = entry
	return nil
}

// Exchange atomically looks up, conditionally deletes, and returns the token
// for a join code. Returns ErrCodeNotFound when the code was never issued,
// was already exchanged (single-use), was revoked, or was garbage-collected
// after expiry. Returns ErrCodeExpired when the code is known but past its
// TTL (and deletes the entry — for BOTH single-use and reusable codes — as a
// side effect so the next Exchange gets ErrCodeNotFound).
//
// On a successful (unexpired) exchange, the entry is deleted UNLESS it is
// marked reusable — that conditional is the entire single-use vs reusable
// distinction; a reusable entry is left in place so a later Exchange call
// can resolve the same code again.
//
// The lookup+expiry-check+delete is performed under a single mutex hold to
// prevent the TOCTOU race where two goroutines both observe the entry as
// valid and both perform the delete — only one of them would report a
// successful exchange of a single-use code in a race-free design, and this
// method is that design. Reusable codes are intentionally exempt from that
// single-success guarantee (that is the whole point of reusable), but the
// lookup+expiry-check still happens under the same lock for both classes so
// expiry is evaluated consistently.
func (m *JoinCodeManager) Exchange(code string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.codes[code]
	if !ok {
		return "", ErrCodeNotFound
	}
	if m.now().After(entry.expiry) {
		delete(m.codes, code)
		return "", ErrCodeExpired
	}
	if !entry.reusable {
		delete(m.codes, code)
	}
	return entry.token, nil
}
