---
phase: 87
plan: 02
type: execute
wave: 1
depends_on: [1]
files_modified:
  - internal/capability/capability.go
  - internal/capability/keystore.go
  - internal/capability/joincode.go
  - internal/capability/errors.go
  - internal/capability/context.go
  - internal/capability/capability_test.go
  - internal/capability/keystore_test.go
  - internal/capability/joincode_test.go
  - internal/capability/capability_fuzz_test.go
autonomous: true
requirements:
  - SEC-01
  - SEC-02
  - SEC-03
  - SEC-04
  - SEC-05
tags:
  - security
  - capability
  - crypto

must_haves:
  truths:
    - "capability.Sign produces a two-segment base64url token that capability.Verify accepts with the same key"
    - "capability.Verify rejects any single-bit tamper of payload or signature"
    - "capability.Verify uses hmac.Equal for constant-time comparison"
    - "FileKeyStore persists a 32-byte signing key to capability.key with mode 0600"
    - "LoadOrGenerate returns the same key on subsequent calls across process lifetimes"
    - "JoinCodeManager.Exchange is single-use (second Exchange returns ErrCodeNotFound) and expires after TTL"
    - "FuzzVerify runs 30s without a panic on random input"
  artifacts:
    - path: internal/capability/capability.go
      provides: "Sign, Verify, Claims type, and package doc"
      contains: "func Sign"
    - path: internal/capability/keystore.go
      provides: "KeyStore interface, FileKeyStore, GenerateKey, LoadOrGenerate"
      contains: "type KeyStore interface"
    - path: internal/capability/joincode.go
      provides: "JoinCodeManager with Issue and Exchange, 5-minute TTL"
      contains: "type JoinCodeManager struct"
    - path: internal/capability/errors.go
      provides: "Sentinel errors: ErrMalformedToken, ErrInvalidSignature, ErrMalformedClaims, ErrCodeNotFound, ErrCodeExpired"
    - path: internal/capability/context.go
      provides: "WithClaims, ClaimsFromContext (context-plumbing helpers)"
  key_links:
    - from: internal/capability/capability.go
      to: "crypto/hmac.Equal"
      via: "constant-time signature comparison"
      pattern: "hmac\\.Equal\\("
    - from: internal/capability/keystore.go
      to: "os.WriteFile with 0600"
      via: "atomic 0600 key persistence mirroring engine.go:132"
      pattern: "os\\.WriteFile"
    - from: internal/capability/joincode.go
      to: "sync.Mutex (NOT RWMutex)"
      via: "TOCTOU-safe single-use exchange"
      pattern: "sync\\.Mutex"
---

<objective>
Author the `internal/capability` package — the pure-Go, stdlib-only token signing/verification, key persistence, and join-code management subsystem that gates every web-facing session route. This plan turns the Wave 0 RED tests from Plan 01 into GREEN tests.

Purpose: Deliver SEC-01..SEC-05 primitive — the capability token format (D-01..D-03), the file-backed KeyStore (D-04..D-05), and the short-code flow (D-09..D-11). No HTTP wiring happens in this plan; that is Plan 03.

Output: Five production Go files totaling ~500 lines; 21 unit tests passing; FuzzVerify running 30 seconds without panic.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/ROADMAP.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md
@internal/daemon/engine.go
@internal/daemon/engine_settings_test.go
@internal/webserver/auth.go
@internal/relay/protocol.go

<interfaces>
Exact signatures this plan produces. Downstream plans (03, 04) import from this package.

```go
package capability

type Claims struct {
    SID     string `json:"sid"`
    Perms   string `json:"perms"`
    IAT     int64  `json:"iat"`
    GrantID string `json:"grant_id"`
    V       int    `json:"v"`
}

func Sign(c Claims, key []byte) (string, error)
func Verify(token string, key []byte) (Claims, error)

type KeyStore interface {
    Load() ([]byte, error)
    Save(key []byte) error
    Location() string
}

type FileKeyStore struct{ path string }
func NewFileKeyStore(dir string) *FileKeyStore
func (s *FileKeyStore) Load() ([]byte, error)
func (s *FileKeyStore) Save(key []byte) error
func (s *FileKeyStore) Location() string

func GenerateKey() ([]byte, error)
func LoadOrGenerate(store KeyStore) ([]byte, error)

type JoinCodeManager struct {
    mu    sync.Mutex
    codes map[string]joinEntry
    ttl   time.Duration
    now   func() time.Time
}
func NewJoinCodeManager(ttl time.Duration) *JoinCodeManager
func (m *JoinCodeManager) Issue(token string) (string, error)
func (m *JoinCodeManager) Exchange(code string) (string, error)

var (
    ErrMalformedToken   = errors.New("capability: malformed token")
    ErrInvalidSignature = errors.New("capability: invalid signature")
    ErrMalformedClaims  = errors.New("capability: malformed claims")
    ErrCodeNotFound     = errors.New("capability: join code not found")
    ErrCodeExpired      = errors.New("capability: join code expired")
)

func WithClaims(ctx context.Context, c Claims) context.Context
func ClaimsFromContext(ctx context.Context) (Claims, bool)
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <id>87-02-01</id>
  <name>Task 1: Implement capability.go, errors.go, context.go (token sign/verify + ctx helpers)</name>
  <files>internal/capability/capability.go, internal/capability/errors.go, internal/capability/context.go, internal/capability/capability_test.go, internal/capability/capability_fuzz_test.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 214-281 Pattern 1 Sign/Verify; lines 605-615 Pitfall 6)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 36-93 capability.go style)
    - /Users/ken/dev/agenthub/internal/webserver/auth.go
    - /Users/ken/dev/agenthub/internal/relay/protocol.go
    - /Users/ken/dev/agenthub/internal/capability/capability_test.go
    - /Users/ken/dev/agenthub/internal/capability/capability_fuzz_test.go
  </read_first>
  <behavior>
    Turn these RED tests GREEN:
    - TestSign_RoundTrip: Sign then Verify returns the same Claims
    - TestVerify_RejectsTamperedPayload: single-bit flip in payload segment yields errors.Is(err, ErrInvalidSignature)
    - TestVerify_RejectsTamperedSignature: bit flip in sig segment yields errors.Is(err, ErrInvalidSignature)
    - TestVerify_RejectsWrongKey: sign with key A verify with key B yields ErrInvalidSignature
    - TestVerify_RejectsMalformedTokenSegmentCount: "no-dot-here" yields ErrMalformedToken
    - TestVerify_RejectsMalformedBase64: "!!!.!!!" yields ErrMalformedToken
    - TestVerify_RejectsMalformedClaimsJSON: valid sig over non-JSON payload yields ErrMalformedClaims
    - TestVerify_ConstantTimeComparison: source-grep assertion that capability.go uses hmac.Equal and does not use bytes.Equal or == on sig bytes
    - TestClaims_Context_RoundTrip: WithClaims/ClaimsFromContext round-trip; empty ctx returns (zero, false)
    - FuzzVerify: 30-second fuzz run with one seed corpus entry; must not panic for any input
  </behavior>
  <action>
    **Ordering invariant (WARNING #5):** Create production code FIRST, compile-gate it, THEN remove the build tags from test files, THEN un-skip tests. Removing build tags before production code exists would break `go test ./internal/capability/...` between steps — the build tags are what kept Wave 0 green.

    1. Create `internal/capability/errors.go` with package capability and sentinel errors ErrMalformedToken, ErrInvalidSignature, ErrMalformedClaims using errors.New with "capability: ..." prefixes per PATTERNS lines 68-75.

    2. Create `internal/capability/context.go` with package capability; imports context; declares unexported type ctxKey struct{}; exports WithClaims(ctx, c) and ClaimsFromContext(ctx) per PATTERNS lines 77-93.

    3. Create `internal/capability/capability.go`. Package doc (mirror auth.go terse style): "Package capability issues and verifies HMAC-SHA256 capability tokens that gate web access to individual PTY sessions. Tokens encode a compact claim set as base64url(claimsJSON).base64url(sig). See SEC-01..SEC-05."
       Imports: crypto/hmac, crypto/sha256, encoding/base64, encoding/json, fmt, strings.
       Declare Claims struct with exact JSON tags in declaration order: SID, Perms, IAT, GrantID, V.
       Sign: marshal claims, compute HMAC-SHA256 with key, base64url-encode payload and sig, return "b64Payload" + "." + "b64Sig".
       Verify: strings.SplitN(token, ".", 2); on wrong count return fmt.Errorf("%w: segment count", ErrMalformedToken). base64.RawURLEncoding.DecodeString each segment; on error return fmt.Errorf("%w: %v", ErrMalformedToken, err). Compute expected HMAC over decoded payload. hmac.Equal(sig, expected) — on mismatch return ErrInvalidSignature. json.Unmarshal(payload, &c); on error return fmt.Errorf("%w: %v", ErrMalformedClaims, err).

    4. **Compile gate:** Run `go build ./internal/capability/...`. Must succeed before proceeding. This proves the production code compiles independent of the (still tag-gated) test files.

    5. Remove the line `//go:build phase87_wave1` from `internal/capability/capability_test.go` and `internal/capability/capability_fuzz_test.go` (note: keystore_test.go and joincode_test.go still carry the tag; they get un-tagged in task 87-02-02).

    6. Un-skip the 9 test bodies in capability_test.go one by one and iterate. For TestVerify_ConstantTimeComparison the body reads internal/capability/capability.go via os.ReadFile and asserts strings.Contains(content, "hmac.Equal") and !strings.Contains(content, "bytes.Equal").

    7. Run `go test ./internal/capability/ -run 'TestSign|TestVerify|TestClaims' -count=1 -v` — all 9 tests PASS (confirms GREEN).

    8. Run `go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=30s` — no panics.

    Anti-patterns to avoid: NEVER use bytes.Equal or == on sig; NEVER add alg field to Claims (D-01); NEVER add exp claim (D-12); NEVER use math/rand.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/capability/ -run 'TestSign|TestVerify|TestClaims' -count=1 -v 2>&1 | tee /tmp/cap-unit.log ; grep -q PASS /tmp/cap-unit.log && ! grep -q FAIL /tmp/cap-unit.log && grep -q 'hmac.Equal' internal/capability/capability.go && ! grep -q 'bytes.Equal' internal/capability/capability.go && go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=30s 2>&1 | tee /tmp/cap-fuzz.log ; ! grep -q 'fuzz: failed' /tmp/cap-fuzz.log</automated>
  </verify>
  <acceptance_criteria>
    - internal/capability/capability.go, errors.go, context.go all compile
    - `grep -q "hmac.Equal" internal/capability/capability.go` succeeds
    - `grep -q "base64.RawURLEncoding" internal/capability/capability.go` succeeds
    - `grep -q "bytes.Equal" internal/capability/capability.go` fails (no matches)
    - `go test ./internal/capability/ -run 'TestSign|TestVerify|TestClaims' -count=1` exits 0 with all tests PASS
    - `go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=30s` exits 0 with no panics
    - Zero remaining t.Skip calls in capability_test.go
  </acceptance_criteria>
  <done>Capability token subsystem green: Sign/Verify round-trip works; tamper/wrong-key/malformed inputs rejected with typed errors; fuzz runs 30s cleanly; context plumbing works.</done>
</task>

<task type="auto" tdd="true">
  <id>87-02-02</id>
  <name>Task 2: Implement keystore.go and joincode.go (file persistence + short-code manager)</name>
  <files>internal/capability/keystore.go, internal/capability/joincode.go, internal/capability/errors.go, internal/capability/keystore_test.go, internal/capability/joincode_test.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 285-368 Pattern 2 FileKeyStore; lines 460-485 Pattern 5 JoinCodeManager; lines 589-594 Pitfall 4 TOCTOU)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 97-152 keystore idiom; lines 156-213 joincode idiom)
    - /Users/ken/dev/agenthub/internal/daemon/engine.go
    - /Users/ken/dev/agenthub/internal/daemon/engine_settings_test.go
    - /Users/ken/dev/agenthub/internal/capability/keystore_test.go
    - /Users/ken/dev/agenthub/internal/capability/joincode_test.go
  </read_first>
  <behavior>
    Turn these RED tests GREEN:
    - TestFileKeyStore_RoundTrip: Save 32 bytes, os.Stat mode == 0600, Load returns same bytes
    - TestFileKeyStore_MissingFileReturnsNilNil: Load on empty TempDir returns (nil, nil)
    - TestFileKeyStore_CorruptLengthReturnsError: write 16 bytes manually, Load returns non-nil error mentioning "corrupt"
    - TestGenerateKey_Length32: returns 32-byte slice
    - TestLoadOrGenerate_GeneratesOnFirstRun: new store, returns 32 bytes and file exists on disk
    - TestLoadOrGenerate_ReloadsOnSecondRun: two calls return identical bytes
    - TestJoinCodeManager_IssueFormat: regex `^[A-Z2-7]{4}-[A-Z2-7]{4}$`
    - TestJoinCodeManager_ExchangeSucceedsOnce: Issue(tok) -> code; Exchange(code) -> (tok, nil)
    - TestJoinCodeManager_ExchangeRejectsDoubleUse: second Exchange returns ErrCodeNotFound
    - TestJoinCodeManager_ExchangeExpiresAfterTTL: inject now, advance past TTL, Exchange returns ErrCodeExpired
    - TestJoinCodeManager_ConcurrentExchangeIsAtomic: 100 goroutines, exactly 1 succeeds (atomic.Int64 counter)
    - TestJoinCodeManager_ExchangeRejectsUnknownCode: Exchange("never-issued") returns ErrCodeNotFound
  </behavior>
  <action>
    1. Remove `//go:build phase87_wave1` from keystore_test.go and joincode_test.go.

    2. Extend errors.go: append ErrCodeNotFound = errors.New("capability: join code not found") and ErrCodeExpired = errors.New("capability: join code expired").

    3. Create internal/capability/keystore.go per RESEARCH Pattern 2 verbatim:
       - Imports: crypto/rand, fmt, os, path/filepath.
       - KeyStore interface with Load/Save/Location.
       - FileKeyStore struct with path field.
       - NewFileKeyStore(dir) returns &FileKeyStore{path: filepath.Join(dir, "capability.key")}.
       - Location returns s.path.
       - Load: os.ReadFile(s.path); if os.IsNotExist return (nil, nil); if other err return wrapped; if len(data) != 32 return fmt.Errorf("capability: key file corrupt (got %d bytes, want 32)", len(data)); else return data.
       - Save: return os.WriteFile(s.path, key, 0600) — mirrors engine.go:132 atomic 0600 pattern.
       - GenerateKey: make([]byte, 32); rand.Read(key); return key, err.
       - LoadOrGenerate: store.Load(); if nil key, GenerateKey; store.Save; return.

    4. Create internal/capability/joincode.go:
       - Imports: crypto/rand, encoding/base32, sync, time.
       - Package-level: var joinCodeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding).
       - type joinEntry struct { token string; expiry time.Time }.
       - type JoinCodeManager struct { mu sync.Mutex; codes map[string]joinEntry; ttl time.Duration; now func() time.Time }. Use sync.Mutex NOT RWMutex (Pitfall 4).
       - NewJoinCodeManager(ttl): returns &JoinCodeManager with codes make, ttl, now: time.Now.
       - Issue(token): var raw [5]byte; rand.Read(raw[:]); encoded := joinCodeEncoding.EncodeToString(raw[:]); code := encoded[:4] + "-" + encoded[4:8]; lock; codes[code] = joinEntry{token, m.now().Add(m.ttl)}; unlock; return code, nil.
       - Exchange(code): lock; defer unlock; entry, ok := m.codes[code]; if !ok return "", ErrCodeNotFound; if m.now().After(entry.expiry) { delete(m.codes, code); return "", ErrCodeExpired }; delete(m.codes, code); return entry.token, nil.

    5. Un-skip tests in keystore_test.go and joincode_test.go. For the TOCTOU test use sync.WaitGroup + atomic.Int64; assert final count == 1. For the TTL expiry test, build manager and set m.now to a closure that returns start.Add(6*time.Minute).

    6. Run `go test ./internal/capability/ -count=1 -v` — all 21 tests (9+6+6) PASS.

    Anti-patterns: NEVER use sync.RWMutex in JoinCodeManager; NEVER use math/rand; NEVER use time.Now() directly inside methods — always through m.now() for testability.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/capability/ -count=1 -v 2>&1 | tee /tmp/cap-all.log ; ! grep -q FAIL /tmp/cap-all.log && grep -q 'sync.Mutex' internal/capability/joincode.go && ! grep -q 'sync.RWMutex' internal/capability/joincode.go && grep -q 'os.WriteFile' internal/capability/keystore.go && grep -q '0600' internal/capability/keystore.go && grep -q 'crypto/rand' internal/capability/keystore.go && grep -q 'crypto/rand' internal/capability/joincode.go && ! grep -rq '"math/rand"' internal/capability/</automated>
  </verify>
  <acceptance_criteria>
    - `go test ./internal/capability/ -count=1` exits 0 with all 21 unit tests PASS
    - `grep -q "sync.Mutex" internal/capability/joincode.go` succeeds
    - `grep -q "sync.RWMutex" internal/capability/joincode.go` fails (RWMutex forbidden per Pitfall 4)
    - `grep -q "os.WriteFile" internal/capability/keystore.go` succeeds
    - `grep -q "0600" internal/capability/keystore.go` succeeds
    - `grep -rq '"math/rand"' internal/capability/` fails (no insecure randomness)
    - `grep -q "crypto/rand" internal/capability/joincode.go` succeeds
    - Zero remaining t.Skip calls in keystore_test.go and joincode_test.go
  </acceptance_criteria>
  <done>Keystore and join-code subsystems green: 32-byte keys persist with mode 0600; missing file is not an error; join codes are single-use XXXX-XXXX base32 strings with 5-minute TTL and atomic TOCTOU-safe exchange.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| disk→memory | Signing key read from capability.key file (mode 0600) into daemon memory |
| user→daemon | Join code issued to user is consumed once by the daemon |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-87-03 | Spoofing | Token forgery via HMAC bypass or algorithm confusion | mitigate | capability.Verify uses crypto/hmac + hmac.Equal (constant-time); fixed HMAC-SHA256 algo (no alg field); TestVerify_ConstantTimeComparison + FuzzVerify (30s) in task 87-02-01 |
| T-87-06 | Availability | Signing key lost on daemon restart invalidates all shared URLs | mitigate | FileKeyStore.Save with mode 0600 + LoadOrGenerate regenerates only when absent; TestLoadOrGenerate_ReloadsOnSecondRun in task 87-02-02 |
| T-87-07 | Elevation of Privilege (replay) | Join code reused after exchange grants second party access | mitigate | JoinCodeManager.Exchange is single-use (deletes entry inside mutex); TestJoinCodeManager_ConcurrentExchangeIsAtomic proves 100-way race yields exactly one success |
| T-87-08 | Information Disclosure (timing) | Timing attack on HMAC comparison reveals key bits | mitigate | hmac.Equal constant-time comparison (D-01); source-grep test TestVerify_ConstantTimeComparison asserts no bytes.Equal or == on sig |
</threat_model>

<verification>
Phase-level gate after this plan:
- `go test ./internal/capability/ -count=1` passes with 21 unit tests green
- `go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=30s` completes without fuzz failures
- `go build ./...` passes (no downstream consumer yet — pure leaf package)
- `ls internal/capability/*.go | grep -v _test.go | wc -l` returns 5 (capability.go, errors.go, context.go, keystore.go, joincode.go)
</verification>

<success_criteria>
- All 21 capability unit tests PASS
- FuzzVerify runs 30s with zero panics
- No `math/rand`, `bytes.Equal` on sig bytes, `sync.RWMutex` in JoinCodeManager, `exp` claim, or `alg` field anywhere in the package
- No existing test in the repo regresses (`go test ./... -count=1` passes)
- File `internal/capability/capability.key` does not exist (production runtime creates it; tests use t.TempDir)
</success_criteria>

<output>
After completion, create `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-02-SUMMARY.md` documenting package layout, exported symbols, and the behavior guarantees each test asserts. Include the fuzz corpus count (should be 1 seed + N discovered).
</output>
