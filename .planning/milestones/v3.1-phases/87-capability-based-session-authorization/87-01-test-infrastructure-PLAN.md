---
phase: 87
plan: 01
type: execute
wave: 0
depends_on: []
files_modified:
  - internal/capability/capability_test.go
  - internal/capability/keystore_test.go
  - internal/capability/joincode_test.go
  - internal/capability/capability_fuzz_test.go
  - internal/webserver/capability_test_helpers.go
  - internal/webserver/capability_test.go
autonomous: true
requirements:
  - SEC-01
  - SEC-02
  - SEC-03
  - SEC-04
  - SEC-05
tags:
  - security
  - testing
  - capability

must_haves:
  truths:
    - "Test scaffolds exist for every SEC-01..SEC-05 behavior before implementation begins"
    - "Fuzz harness exists for token verification forgery resistance"
    - "TLS + HTTP helpers from the security-review scaffold are relocated into internal/webserver/ test support"
  artifacts:
    - path: internal/capability/capability_test.go
      provides: "RED test skeletons for Sign/Verify round-trip, tamper detection, wrong-key rejection, malformed input"
    - path: internal/capability/keystore_test.go
      provides: "RED test skeletons for FileKeyStore Load/Save round-trip + missing-file-is-not-error"
    - path: internal/capability/joincode_test.go
      provides: "RED test skeletons for single-use join-code exchange + TTL expiry + TOCTOU race"
    - path: internal/capability/capability_fuzz_test.go
      provides: "FuzzVerify harness seeded with one known-good token"
    - path: internal/webserver/capability_test_helpers.go
      provides: "selfSignedTLSForTest, testServer, testServerWithHub, dialWebServerWS, readPipeWithTimeout helpers relocated from security-review scaffold"
    - path: internal/webserver/capability_test.go
      provides: "RED test skeletons for SEC-01..SEC-05 HTTP + WS behaviors; all tests compile with t.Skip() markers noting the Wave/Plan that will implement the behavior"
  key_links:
    - from: internal/webserver/capability_test.go
      to: internal/webserver/capability_test_helpers.go
      via: "shared helpers"
      pattern: "testServer\\(t\\)|testServerWithHub\\("
    - from: internal/capability/capability_fuzz_test.go
      to: internal/capability/capability_test.go
      via: "seed corpus from known-good Sign output"
      pattern: "capability\\.Sign\\("
---

<objective>
Create the complete test infrastructure (Wave 0) for Phase 87 before any implementation runs. Every downstream plan (02–06) must be able to run its `<automated>` verify command against an already-existing test file. This plan authors those test files as RED skeletons: they compile, they reference the (yet-to-be-written) production symbols via `//go:build` guards or `t.Skip("implemented in plan 02-XX")` markers, and they relocate the security-review test helpers into the canonical `internal/webserver/` location.

Purpose: Satisfy the Nyquist validation rule (`87-VALIDATION.md`) that every subsequent task has an `<automated>` command pointing at an already-existing test file — no "MISSING — Wave 0 must create" placeholders survive past this plan.

Output: Six new Go test files on disk, compiling (with skips for unimplemented symbols), runnable via `go test ./internal/capability/... ./internal/webserver/... -count=1 -run TestWave0Sentinel`.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/ROADMAP.md
@.planning/REQUIREMENTS.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-VALIDATION.md
@security-review/internal_webserver_server_test.go
@security-review/internal_relay_protocol_fuzz_test.go
@internal/daemon/engine_settings_test.go
@internal/webserver/auth_test.go

<interfaces>
<!-- Symbols that will be implemented by Plan 02 (capability core) -->
<!-- Wave 0 test skeletons compile by declaring expected signatures as references only -->

Package `internal/capability` (to be authored in Plan 02):
```go
type Claims struct {
    SID     string `json:"sid"`
    Perms   string `json:"perms"`    // "read" or "read,write"
    IAT     int64  `json:"iat"`
    GrantID string `json:"grant_id"`
    V       int    `json:"v"`
}

func Sign(claims Claims, key []byte) (string, error)
func Verify(token string, key []byte) (Claims, error)

type KeyStore interface {
    Load() ([]byte, error)
    Save(key []byte) error
    Location() string
}
type FileKeyStore struct { /* unexported */ }
func NewFileKeyStore(dir string) *FileKeyStore
func GenerateKey() ([]byte, error)
func LoadOrGenerate(store KeyStore) ([]byte, error)

type JoinCodeManager struct { /* unexported */ }
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

Helpers relocated from `security-review/internal_webserver_server_test.go` into `internal/webserver/capability_test_helpers.go`:
```go
func selfSignedTLSForTest(t *testing.T) *tls.Config
func testServer(t *testing.T) (*WebServer, *http.Client)
func testServerWithHub(t *testing.T) (*WebServer, *relay.Hub, *http.Client)
func dialWebServerWS(t *testing.T, baseURL, path string, headers http.Header) *websocket.Conn
func readPipeWithTimeout(t *testing.T, r io.Reader, want string, timeout time.Duration)
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <id>87-01-01</id>
  <name>Task 1: Create capability package RED test skeletons</name>
  <files>internal/capability/capability_test.go, internal/capability/keystore_test.go, internal/capability/joincode_test.go, internal/capability/capability_fuzz_test.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 214-281 Sign/Verify pattern; lines 285-368 FileKeyStore pattern; lines 460-485 JoinCodeManager pattern; lines 700-723 test pattern; lines 455-467 FuzzVerify)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 36-93 capability.go style; lines 97-152 keystore.go mirroring engine_settings_test.go; lines 156-213 joincode.go; lines 217-263 keystore_test.go template; lines 438-467 fuzz harness)
    - /Users/ken/dev/agenthub/internal/daemon/engine_settings_test.go (full — template for round-trip tests)
    - /Users/ken/dev/agenthub/security-review/internal_relay_protocol_fuzz_test.go (full — fuzz harness reference)
  </read_first>
  <behavior>
    Every test must compile today even though `internal/capability/*.go` production files do not yet exist. Accomplish this by writing tests against a TO-BE-CREATED package — the tests will fail to compile until Plan 02 creates `internal/capability/capability.go`, `keystore.go`, `joincode.go`. This is intentional and matches the RED phase of TDD. The `go test` command run at the end of this task must fail with a compilation error like "no Go files in internal/capability" — that is the expected Wave 0 signal that the test files exist and are waiting for Plan 02.

    - capability_test.go: tests for TestSign_RoundTrip, TestVerify_RejectsTamperedPayload, TestVerify_RejectsTamperedSignature, TestVerify_RejectsWrongKey, TestVerify_RejectsMalformedTokenSegmentCount, TestVerify_RejectsMalformedBase64, TestVerify_RejectsMalformedClaimsJSON, TestVerify_ConstantTimeComparison (asserts via hmac.Equal usage not leaking), TestClaims_Context_RoundTrip
    - keystore_test.go: TestFileKeyStore_RoundTrip (with 0600 mode assertion per engine_settings_test.go pattern), TestFileKeyStore_MissingFileReturnsNilNil, TestFileKeyStore_CorruptLengthReturnsError, TestGenerateKey_Length32, TestLoadOrGenerate_GeneratesOnFirstRun, TestLoadOrGenerate_ReloadsOnSecondRun
    - joincode_test.go: TestJoinCodeManager_IssueFormat (asserts regex `^[A-Z2-7]{4}-[A-Z2-7]{4}$`), TestJoinCodeManager_ExchangeSucceedsOnce, TestJoinCodeManager_ExchangeRejectsDoubleUse (returns ErrCodeNotFound on second call), TestJoinCodeManager_ExchangeExpiresAfterTTL (uses injectable `now` clock), TestJoinCodeManager_ConcurrentExchangeIsAtomic (TOCTOU test: 100 goroutines, exactly one succeeds), TestJoinCodeManager_ExchangeRejectsUnknownCode
    - capability_fuzz_test.go: FuzzVerify seeded with one known-good `Sign(Claims{SID:"s1", Perms:"read", V:1}, key)` output (32-byte key of byte index i). Fuzz body calls `capability.Verify(token, key)` and asserts no panic on ANY input.
  </behavior>
  <action>
    Create FOUR files under `/Users/ken/dev/agenthub/internal/capability/`:

    1. `capability_test.go` — package `capability_test`. Imports: `bytes`, `context`, `crypto/rand`, `encoding/base64`, `errors`, `strings`, `testing`, `github.com/kenscott/agenthub/internal/capability` (verify correct module path via `go.mod` read). Test functions EXACTLY as named in behavior block. Each test body stubbed with `t.Skip("implemented in plan 02")` at top — this lets the file compile as soon as `internal/capability/capability.go` exists (Plan 02), and skip execution until the test body is un-skipped by Plan 02. Declare expected signatures via the imported package.

    2. `keystore_test.go` — package `capability_test`. Imports: `bytes`, `os`, `path/filepath`, `testing`, capability package. Use `t.TempDir()` per the engine_settings_test.go:11-99 template. Test functions EXACTLY as named. Each body stubbed with `t.Skip("implemented in plan 02")`.

    3. `joincode_test.go` — package `capability_test`. Imports: `regexp`, `sync`, `sync/atomic`, `testing`, `time`, capability package. Test functions EXACTLY as named. Each body stubbed with `t.Skip("implemented in plan 02")`. For concurrent exchange test, seed with exactly 100 goroutines calling Exchange on the same code; use `atomic.Int64` counter for successes; assert count == 1.

    4. `capability_fuzz_test.go` — package `capability_test`. Imports: `testing`, capability package. `func FuzzVerify(f *testing.F)` with `f.Add(goodToken)` seed and body `_, _ = capability.Verify(token, key)`. Fuzz body must NOT be skipped — fuzz entry points cannot be skipped. Instead, guard with build tag `//go:build phase87_wave1` at the top of the file so the file is excluded from `go test` until Plan 02 removes the tag.

    CRITICAL: The production files (`capability.go`, `keystore.go`, `joincode.go`) do NOT exist yet. This is the RED phase. The acceptance criterion for THIS task is that the test files compile independently when the production package is empty — which they WILL NOT, because Go needs the import to resolve. To work around: add `//go:build phase87_wave1` to ALL FOUR files so `go test ./internal/capability/...` passes with "no matching test files" (exit code 0). Plan 02's first task will REMOVE these build tags after creating the production files.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && mkdir -p internal/capability && ls internal/capability/capability_test.go internal/capability/keystore_test.go internal/capability/joincode_test.go internal/capability/capability_fuzz_test.go && grep -c '^//go:build phase87_wave1$' internal/capability/capability_test.go internal/capability/keystore_test.go internal/capability/joincode_test.go internal/capability/capability_fuzz_test.go && go test ./internal/capability/... -count=1 2>&1 | tee /tmp/wave0-cap.log && ! grep -q "FAIL" /tmp/wave0-cap.log</automated>
  </verify>
  <acceptance_criteria>
    - `ls internal/capability/capability_test.go` exits 0
    - `ls internal/capability/keystore_test.go` exits 0
    - `ls internal/capability/joincode_test.go` exits 0
    - `ls internal/capability/capability_fuzz_test.go` exits 0
    - Every file has `//go:build phase87_wave1` on line 1
    - `grep -c "^func Test" internal/capability/capability_test.go` returns `9` (9 test functions)
    - `grep -c "^func Test" internal/capability/keystore_test.go` returns `6`
    - `grep -c "^func Test" internal/capability/joincode_test.go` returns `6`
    - `grep -q "^func FuzzVerify" internal/capability/capability_fuzz_test.go` succeeds
    - `grep -q 't.Skip("implemented in plan 02")' internal/capability/capability_test.go` succeeds
    - `go test ./internal/capability/... -count=1` exits 0 with "no matching test files"
  </acceptance_criteria>
  <done>Four RED test files exist, build-tag-gated, totaling 21 test functions + 1 fuzz harness, matching research and pattern-map references.</done>
</task>

<task type="auto" tdd="true">
  <id>87-01-02</id>
  <name>Task 2: Relocate security-review test helpers and author webserver capability test skeletons</name>
  <files>internal/webserver/capability_test_helpers.go, internal/webserver/capability_test.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/security-review/internal_webserver_server_test.go (full — source of helpers and inverted tests)
    - /Users/ken/dev/agenthub/internal/webserver/server_test.go (full — existing helper style and naming)
    - /Users/ken/dev/agenthub/internal/webserver/auth_test.go (full — short-test pattern for webserver package)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 699-724 test inversion guidance; lines 803-813 per-requirement test map)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 429-467 test file analog guidance)
  </read_first>
  <behavior>
    - capability_test_helpers.go: relocates `selfSignedTLSForTest`, `testServer`, `testServerWithHub`, `dialWebServerWS`, `readPipeWithTimeout` from the security-review scaffold into the canonical webserver test-support location. Build-tag-gated with `//go:build phase87_wave2` so it does NOT compile until Plan 03 wires the referenced `capability` package into `WebServer`.
    - capability_test.go: skeletons for TestSecurity_UnauthenticatedClientCannotEnumerateSessions (SEC-02), TestSecurity_WrongSessionCapRejected (SEC-03), TestSecurity_ReadOnlyParamCannotGrantWrite (SEC-04), TestSecurity_ReadOnlyCapabilityBlocksMsgInput (SEC-05), TestSecurity_ReconnectWithoutReadonlyStillBlocked (SEC-05 bypass regression), TestCapability_MissingCapReturns401, TestCapability_InvalidSignatureReturns401, TestCapability_RevokedGrantReturns403, TestCapability_ValidCapReturnsSession. Also build-tag-gated for Wave 2.
  </behavior>
  <action>
    Create TWO files under `/Users/ken/dev/agenthub/internal/webserver/`:

    1. `capability_test_helpers.go`:
       - First line: `//go:build phase87_wave2`
       - Blank line, then `package webserver`
       - COPY verbatim the helper functions from `security-review/internal_webserver_server_test.go:29-198` (selfSignedTLSForTest, testServer, testServerWithHub, dialWebServerWS, readPipeWithTimeout). Adjust imports to match this package's existing test imports (see `server_test.go` imports for reference).
       - If a helper name collides with an existing helper in `server_test.go`, rename the relocated helper to `capTestServer`, `capTestServerWithHub`, etc. Grep `server_test.go` first to check collisions.
       - Add a `// Package webserver test helpers relocated from security-review/ for Phase 87.` doc comment at the top.

    2. `capability_test.go`:
       - First line: `//go:build phase87_wave2`
       - Blank line, then `package webserver`
       - Nine test functions EXACTLY as named in behavior. Every body stubbed with `t.Skip("implemented in plan 03")`.
       - Use the relocated helpers (`testServer(t)` etc.).
       - Each test includes a doc comment stating the REQ-ID it verifies (e.g. `// SEC-02: GET /api/sessions without a valid cap returns 401.`).
       - For TestSecurity_ReconnectWithoutReadonlyStillBlocked: comment documents the flow — (1) dial WS with read-only cap (no readonly= param); (2) send MsgInput frame; (3) assert PTY pipe receives ZERO bytes via `readPipeWithTimeout`.

    Acceptance: both files compile independently (or are build-tag excluded from non-phase87_wave2 builds), referencing no symbols that do not exist in the current codebase — the only symbols they reference are the relocated helpers in the same build tag.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && ls internal/webserver/capability_test_helpers.go internal/webserver/capability_test.go && head -1 internal/webserver/capability_test_helpers.go | grep -q "^//go:build phase87_wave2$" && head -1 internal/webserver/capability_test.go | grep -q "^//go:build phase87_wave2$" && grep -c "^func Test" internal/webserver/capability_test.go | grep -q "^9$" && go test ./internal/webserver/... -count=1 2>&1 | tee /tmp/wave0-ws.log && ! grep -q "FAIL" /tmp/wave0-ws.log</automated>
  </verify>
  <acceptance_criteria>
    - `ls internal/webserver/capability_test_helpers.go` exits 0
    - `ls internal/webserver/capability_test.go` exits 0
    - Both files have `//go:build phase87_wave2` on line 1
    - `grep -c "^func Test" internal/webserver/capability_test.go` returns `9`
    - `grep -q "TestSecurity_UnauthenticatedClientCannotEnumerateSessions" internal/webserver/capability_test.go` succeeds
    - `grep -q "TestSecurity_ReconnectWithoutReadonlyStillBlocked" internal/webserver/capability_test.go` succeeds
    - `grep -q "selfSignedTLSForTest\|capTestServer" internal/webserver/capability_test_helpers.go` succeeds
    - `go test ./internal/webserver/... -count=1` exits 0 (existing tests pass; new gated files excluded)
  </acceptance_criteria>
  <done>Nine test skeletons + helpers exist, build-tag-gated for Wave 2 activation. Every downstream SEC-XX task has an automated command that targets an already-authored test file.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| test→code | Wave 0 test skeletons assert the boundary contract; no trust boundary crossed by this plan itself |

## STRIDE Threat Register

This plan ESTABLISHES the test harness that verifies boundary enforcement in subsequent plans. It does not itself mitigate a runtime threat, but it creates the mechanism by which every HIGH threat (T-87-01 through T-87-04) is proven mitigated.

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-87-01 | Elevation of Privilege | `/api/sessions` listing | mitigate | TestSecurity_UnauthenticatedClientCannotEnumerateSessions skeleton ready; impl in Plan 03 (task 87-03-XX) |
| T-87-02 | Elevation of Privilege | `/sessions/{id}/ws` cross-session | mitigate | TestSecurity_WrongSessionCapRejected skeleton ready; impl in Plan 03 |
| T-87-03 | Spoofing | Token forgery | mitigate | FuzzVerify harness seeded; capability_test.go tamper tests ready; impl in Plan 02 |
| T-87-04 | Elevation of Privilege | Read-only bypass via reconnect | mitigate | TestSecurity_ReconnectWithoutReadonlyStillBlocked skeleton ready; impl in Plan 03 |
| T-87-05 | Information Disclosure | Auto-share on create | mitigate | Covered by Plan 04 test for handleCreateSession; ⚠ task in Plan 04 adds TestHandleCreateSession_NoAutoEnable skeleton |
| T-87-06 | Availability | Key lost on restart | mitigate | TestFileKeyStore_RoundTrip + TestLoadOrGenerate_ReloadsOnSecondRun skeletons ready; impl in Plan 02 |
| T-87-07 | Elevation of Privilege | Revoked cap replay | mitigate | TestCapability_RevokedGrantReturns403 skeleton ready; impl in Plan 03 |
| T-87-08 | Information Disclosure (timing) | HMAC comparison | mitigate | TestVerify_ConstantTimeComparison skeleton ready; impl in Plan 02 (asserts hmac.Equal usage via source grep) |
</threat_model>

<verification>
Phase-level gate after this plan:
- Six new test files exist under `internal/capability/` and `internal/webserver/`
- All are build-tag-gated (phase87_wave1 or phase87_wave2) so `go build ./...` and the existing `go test ./...` continue to pass without any production code change
- Total test-skeleton count: 21 capability tests + 1 fuzz + 9 webserver tests = 31 Nyquist sampling points pre-authored
</verification>

<success_criteria>
- `go test ./... -count=1` passes (existing suite unchanged)
- `ls internal/capability/*_test.go | wc -l` returns 4
- `ls internal/webserver/capability*.go | wc -l` returns 2
- `grep -rhc "^//go:build phase87_wave" internal/capability/ internal/webserver/capability* | paste -sd+ - | bc` returns 6 (one per new file)
- No production code under `internal/capability/` exists yet (`ls internal/capability/*.go 2>/dev/null | grep -v _test.go | wc -l` returns 0)
</success_criteria>

<output>
After completion, create `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-01-SUMMARY.md` documenting which skeletons were authored, the build-tag protocol, and the exact test function names each downstream plan will un-skip.
</output>
