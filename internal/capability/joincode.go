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
type joinEntry struct {
	token  string
	expiry time.Time
}

// JoinCodeManager maps short-lived join codes to capability tokens. A code is
// single-use (D-11) — the first successful Exchange deletes the entry before
// returning the token. Expired entries are removed lazily on the next
// Exchange attempt rather than by a background sweeper; the map stays small
// because the TTL (5 minutes) is short and codes are transient artefacts.
//
// mu is a plain sync.Mutex because Exchange must read and delete atomically
// to prevent the TOCTOU race described in RESEARCH Pitfall 4. A reader-writer
// mutex is the wrong primitive here: its read lock would allow two goroutines
// to observe the entry before either deletes it, producing two successful
// exchanges from one code.
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

// Exchange atomically looks up, deletes, and returns the token for a join
// code. Returns ErrCodeNotFound when the code was never issued, was already
// exchanged, or was garbage-collected after expiry. Returns ErrCodeExpired
// when the code is known but past its TTL (and deletes the entry as a side
// effect so the next Exchange gets ErrCodeNotFound).
//
// The lookup+delete is performed under a single mutex hold to prevent the
// TOCTOU race where two goroutines both observe the entry as valid and both
// perform the delete — only one of them would report a successful exchange
// in a race-free design, and this method is that design.
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
	delete(m.codes, code)
	return entry.token, nil
}
