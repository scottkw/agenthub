# Phase 114: CI test stability — webserver capability test flake - Research

**Researched:** 2026-05-18
**Domain:** Go test determinism / HMAC capability tokens / base64 padding-bit semantics
**Confidence:** HIGH (root cause empirically reproduced + mathematically explained)

## Summary

The flake is **NOT test-state pollution**. It is a self-contained bug inside the
`issueExpiredCapFor` helper in `internal/webserver/plugin_config_stream_test.go`.

The helper corrupts the last byte of a base64-encoded HMAC-SHA256 signature by
flipping `A`↔`B`. For an HMAC-SHA256 output (32 bytes = 256 bits) encoded with
`base64.RawURLEncoding`, the resulting string is **43 characters** in which the
**final character carries only 4 data bits and 2 padding bits**. Base64
characters `A`, `B`, `C`, `D` (indices 0–3) all share the same upper 4 bits and
therefore decode to the **same final signature byte**. Flipping `A`→`B` (or any
of `B`→`A`, `C`→`A`, `D`→`A`) is a **no-op at the decoded-byte level**.

When the freshly-minted signature's last base64 character falls in
`{A, B, C, D}` (4 of 64 alphabet characters = **6.25%** of the time),
`capability.Verify` still accepts the "corrupted" token, the middleware advances
to the grant-list check, finds no grant (the helper deliberately omits
`AddGrant`), and returns **403 "capability has been revoked"** instead of the
expected **401 "capability required"**.

The probability is deterministic in `(SID, Perms, IAT, GrantID, V, key)`. Within
a single `go test` process, `IAT = time.Now().Add(-365d).Unix()` only advances
when wall-clock seconds tick, so a single CI run will see either ~all-pass or
~all-fail behavior depending on which UTC second it starts in. This explains the
"feels like state pollution but reset doesn't fix it" pattern and the
macOS-vs-Linux split: macOS author runs happen to fall outside the bad-second
buckets; some Linux CI runs land inside them.

**Primary recommendation:** Replace the "flip last byte" corruption with a
guaranteed-invalidating corruption. Either (a) flip a high-bit byte in the
**middle** of the signature segment, or (b) replace the entire signature with
a constant known-bad value, or (c) use `capability.ErrInvalidSignature`-by-key:
sign with a deliberately wrong key. Option (c) is the cleanest because it
exercises the exact production path that produces 401 (HMAC mismatch under the
server's configured key) without depending on base64 internals at all.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Investigation-first (LOCKED):** Root cause MUST be identified in writing
  before any fix lands. No `t.Skip`, no retry loop, no `-shuffle=off` workaround.
- **Likely candidates per Issue #58 triage:**
  1. Test-state pollution across `internal/webserver` tests
  2. **Base64 strict-decode variance** — capability token decoding
  3. HMAC implementation under concurrent access
- **Hypothesis 1 (test-state pollution) was leading** per Issue #58 + standard
  flake patterns. Research has now eliminated Hypothesis 1 and confirmed
  Hypothesis 2 with a concrete demonstration.

### Approach (locked)
1. **Investigation phase:** reproduce the flake locally; identify root cause;
   document in writing (commit message + plan SUMMARY).
2. **Fix phase:** address the root cause.
3. **Verification:** `go test -race -shuffle=on -count=100
   ./internal/webserver/` returns 100/100 on Linux CI. macOS CI continues to
   pass.

### Out of scope
- Other flaky tests (file separately).
- Refactoring webserver test infrastructure beyond what the fix needs.

### Cross-surface verification
- N/A user surface (CI-only fix).
- Validation: Linux CI green across 100 consecutive runs at merge commit. macOS
  CI continues to pass.

### macOS executor caveat
- Test passes deterministically on macOS in author's experience. Reproduction
  may require Linux. Research-time reproduction on macOS was **negative**
  (`go test -race -shuffle=on -count=500 ./internal/webserver/` → all pass);
  research switched to code analysis + standalone simulation, which both
  confirm the root cause and the wall-clock-second dependency.

### Claude's Discretion
- Choice of fix variant (sign-with-wrong-key vs. middle-byte flip vs. constant
  garbage signature). Recommendation below.

### Deferred Ideas (OUT OF SCOPE)
None.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | `TestPluginConfigStream_ExpiredCap_Returns401` passes deterministically (100/100 runs) on Linux CI under `-race -shuffle=on`, returning 401 (not 403). Root cause documented in the fix commit. | Section 5 (root cause) + Section 6 (fix recommendation) enable a deterministic-by-construction fix; Section 7 specifies the 100/100 verification protocol. |
| TEST-02 | Underlying cause investigated and stated in writing — not a rerun-pass hack. | Sections 1–5 are the in-writing investigation; Section 6's recommended fix removes the source of nondeterminism, it does not paper over it. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Test helper `issueExpiredCapFor` | Test fixture (Go) | — | The bug lives in test code; production code is correct. |
| Capability HMAC sign/verify | `internal/capability` | — | Production crypto path; **no change**. |
| Capability auth middleware (`requireCapability`) | `internal/webserver` | — | Production policy enforcement; **no change**. 401 vs 403 mapping is already correct per T-87-08. |
| Per-test isolation | Go `*testing.T` + `t.Cleanup` | — | Already correct; `testServer` already registers `t.Cleanup(ws.Stop)`. |

## Phase Requirements → Test Map

(Same as Validation Architecture section below.)

## 1. Reproduce the flake

**Local reproduction attempt (macOS arm64, Go 1.26.3):**

```bash
go test -race -shuffle=on -count=100 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/
# Result: PASS (100/100)

go test -race -shuffle=on -count=500 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/
# Result: PASS (500/500)
```

**Could not reproduce locally.** Working hypothesis at this point was either
(a) Linux-specific scheduler timing, or (b) the test is wall-clock-second
dependent and the macOS execution happened to land in non-collision seconds.

Standalone simulation in `/tmp/proof.go` (now deleted; see Section 5)
**deterministically demonstrated** option (b): a specific IAT value
(`1700000046`) produces a signature ending in `A`, the corruption flips it to
`B`, and the corrupted token **still verifies cleanly** against the original
key, putting the request on the path to 403.

[VERIFIED: standalone reproduction on macOS proving the bytes-level no-op]

## 2. Test code walkthrough

`TestPluginConfigStream_ExpiredCap_Returns401`
(`internal/webserver/plugin_config_stream_test.go:88-103`):

```go
func TestPluginConfigStream_ExpiredCap_Returns401(t *testing.T) {
    ws, client := testServer(t)                          // fresh WebServer + TLS client; t.Cleanup(ws.Stop) registered
    ws.SetSigningKey(capTestKey)                         // installs the 32-byte deterministic test key
    ws.EnableSession("sess-pcs-expired")                 // marks session as web-enabled
    ws.SetPluginSettingsProvider(func() []byte { return []byte(`{"webgl":true}`) })
    tok := issueExpiredCapFor(t, ws, "sess-pcs-expired", "read,write")  // mints + corrupts a token

    resp, err := client.Get(ws.BaseURL() + "/api/plugin-config/stream?cap=" + tok)
    // ...
    if resp.StatusCode != http.StatusUnauthorized {       // expects 401
        t.Errorf("expected 401 for corrupted/expired cap, got %d", resp.StatusCode)
    }
}
```

`issueExpiredCapFor`
(`internal/webserver/plugin_config_stream_test.go:32-66`):

```go
func issueExpiredCapFor(t *testing.T, ws *WebServer, sessionID, perms string) string {
    claims := capability.Claims{
        SID:     sessionID,
        Perms:   perms,
        IAT:     time.Now().Add(-365 * 24 * time.Hour).Unix(),   // wall-clock dependent!
        GrantID: "grant-expired-" + sessionID,
        V:       1,
    }
    token, _ := capability.Sign(claims, capTestKey)               // signs with the SAME key the server uses
    // Note: NO ws.AddGrant call — by design

    if i := strings.LastIndex(token, "."); i >= 0 && i < len(token)-1 {
        corrupt := []byte(token)
        c := corrupt[len(corrupt)-1]
        if c == 'A' {
            corrupt[len(corrupt)-1] = 'B'         // <-- BUG: this is a no-op for base64 padding-bit chars
        } else {
            corrupt[len(corrupt)-1] = 'A'         // <-- BUG: same, for chars B/C/D
        }
        token = string(corrupt)
    }
    return token
}
```

**Setup → Exercise → Teardown:**

1. **Setup:** Fresh `WebServer` from `testServer(t)`. New `relay.HubManager`,
   new TLS config, new self-signed cert, new listener. `t.Cleanup` registers
   `ws.Stop()`. No package-level state mutated.
2. **Configure:** `SetSigningKey(capTestKey)` → `ws.signingKey = <32 bytes>`
   under `ws.mu`. `EnableSession("sess-pcs-expired")` → `ws.webEnabled` map
   entry. `SetPluginSettingsProvider(...)` — not mutex-protected but written
   once before any handler can read it.
3. **Mint token:** Build `Claims`, sign with `capTestKey`, **corrupt last
   byte** (the bug — see Section 5).
4. **Exercise:** HTTPS GET `/api/plugin-config/stream?cap=<token>`.
5. **Server processing** (see Section 3 for the full auth flow).
6. **Teardown:** `t.Cleanup(ws.Stop)` runs.

[VERIFIED: file read]

## 3. Capability auth flow — 401 vs 403

`requireCapability` (`internal/webserver/capability_mw.go:37-75`):

| Step | Condition | Response |
|------|-----------|----------|
| 1 | `?cap=` missing | **401** "capability required" |
| 2 | `ws.signingKey` is nil | **401** "capability required" |
| 3 | `capability.Verify(token, key)` returns error | **401** "capability required" (collapsed per T-87-08) |
| 4 | `pathID != "" && claims.SID != pathID` | **403** "capability does not match session" |
| 5 | `!ws.isGrantActive(claims.SID, claims.GrantID)` | **403** "capability has been revoked" |
| 6 | `!ws.IsSessionEnabled(claims.SID)` | **403** "capability has been revoked" |
| 7 | All checks pass | continues to handler |

`capability.Verify` (`internal/capability/capability.go:52-80`) error matrix:

| Failure | Sentinel |
|---------|----------|
| segment count != 2 | `ErrMalformedToken` |
| base64 decode of payload or sig fails | `ErrMalformedToken` |
| HMAC mismatch (`!hmac.Equal(sig, expected)`) | `ErrInvalidSignature` |
| payload JSON unparseable | `ErrMalformedClaims` |

**Critical observation:** `capability.Verify` does **NOT** check IAT for
expiry. The test name says "ExpiredCap" but capability tokens have no expiry
window — verification only checks structure + HMAC. Old IATs are accepted.
Therefore the "expired" rejection in this test **must** come from one of the
non-IAT paths.

[VERIFIED: file read]

### What state would convert an expected-401 into a 403?

The test's expected path to 401: `capability.Verify` fails with
`ErrInvalidSignature` (because the helper corrupted the signature byte).

The actual path to 403 (under the bug): `capability.Verify` SUCCEEDS (the
"corruption" was a no-op), then `isGrantActive` fails (the helper never called
`AddGrant`) → **403 "capability has been revoked"**.

**No external state mutation between tests can produce this transition.** Even
if a hypothetical sibling test rotated the signing key mid-flight, this test's
`SetSigningKey(capTestKey)` runs first and locks in the key; the cap is signed
with the same key the server holds at request time. Cross-test pollution is
ruled out: the per-test `*WebServer` instance is brand new from
`NewWebServer(...)` and only this test's setup writes to its `signingKey`,
`webEnabled`, and `grants` fields before the HTTPS GET fires.

[VERIFIED: code read of `testServer`, `SetSigningKey`, `NewWebServer`]

## 4. Test-state pollution audit

Audit of every shared-state mutation in `internal/webserver/` test setup.

| Mutator | Scope | Cleanup | Pollution risk |
|---------|-------|---------|----------------|
| `relay.NewHubManager()` in `testServer` | per-test `*HubManager` | freed with `ws.Stop()` via `t.Cleanup` | none — fresh instance |
| `selfSignedTLSForTest(t)` | per-test cert/CA | GC'd | none |
| `NewWebServer(cfg, manager)` | per-test `*WebServer` | `t.Cleanup(ws.Stop)` | none — fresh instance |
| `ws.SetSigningKey(capTestKey)` | per-test `ws.signingKey` | `ws.Stop` ends; instance GC'd | none — only the local `ws` is affected |
| `ws.EnableSession(id)` | per-test `ws.webEnabled` | per-instance map; GC'd | none |
| `ws.AddGrant(sid, gid)` | per-test `ws.grants` | per-instance map; GC'd | none |
| `ws.ClearGrants(sid)` | same | same | none |
| `ws.SetPluginSettingsProvider(fn)` | per-test field write | once-before-Start | none |
| `ws.SetSessionResolver(fn)` | per-test field write | once-before-Start | none |
| `capTestKey` (`capability_test.go:26`) | **package-level** | never mutated (var with IIFE init) | **none — read-only across all tests** |
| `ssExtTestKey` (`server_test.go:30`) | **package-level** | never mutated | none |

**Package-level globals are read-only after init.** Every test gets a fresh
`*WebServer` with fresh `signingKey`, `webEnabled`, `grants`, listener, port,
TLS cert, and mux. There is **no observable shared mutable state** between
tests in `internal/webserver/`.

Tests in this package do NOT use `t.Parallel()` (`grep "t.Parallel"
internal/webserver/*_test.go` returns zero hits), so even logical race-y
sharing is precluded.

**Conclusion:** Hypothesis 1 (test-state pollution) is **eliminated**. The
flake is intra-test, not inter-test.

[VERIFIED: grep + file read across `internal/webserver/*_test.go`]

## 5. Root cause — pinpointed

**File:** `internal/webserver/plugin_config_stream_test.go`
**Lines:** 54–64 (the corruption block inside `issueExpiredCapFor`)

```go
if i := strings.LastIndex(token, "."); i >= 0 && i < len(token)-1 {
    corrupt := []byte(token)
    c := corrupt[len(corrupt)-1]
    if c == 'A' {
        corrupt[len(corrupt)-1] = 'B'
    } else {
        corrupt[len(corrupt)-1] = 'A'
    }
    token = string(corrupt)
}
```

### Why this is broken

`capability.Sign` uses `base64.RawURLEncoding.EncodeToString(sig)` where `sig`
is `hmac.New(sha256.New, key).Sum(nil)` — exactly **32 bytes**.

Base64 (raw, no padding) encodes 3 bytes → 4 chars. 32 bytes / 3 = 10 full
triples (30 bytes → 40 chars) + 2 leftover bytes. Two leftover bytes encode to
**3 base64 chars** (16 data bits across 3 × 6 = 18 char-bits, with **2 trailing
padding bits**). Total length: **43 characters**.

The **last character** therefore carries **4 data bits + 2 padding bits**.
Characters whose top 4 bits match decode to the same final signature byte:

| Base64 char | Index (binary) | Top 4 bits | Padding (low 2) | Final decoded byte |
|-------------|---------------|------------|-----------------|---------------------|
| `A` | `000000` | `0000` | `00` | `0x00` |
| `B` | `000001` | `0000` | `01` | `0x00` |
| `C` | `000010` | `0000` | `10` | `0x00` |
| `D` | `000011` | `0000` | `11` | `0x00` |
| `E` | `000100` | `0001` | `00` | `0x01` |
| ... | ... | ... | ... | ... |

When the signature's final base64 character is in `{A, B, C, D}`, the
corruption `A→B` (or `B|C|D → A`) **changes only the padding bits**, which the
base64 decoder discards. `parts[1]` after decode is **byte-identical** to the
uncorrupted signature. HMAC verifies. The test falls through to the grant
check, finds no grant, returns **403**.

### Empirical confirmation

Standalone Go program (`/tmp/proof.go`, deleted after use):

```
HIT IAT=1700000046 origLastChar=A flippedTo=B → corrupted token STILL VERIFIES → would 403
```

A 1000-IAT sweep (`/tmp/b64test.go`, deleted after use):

```
Total collisions: 66 / 1000 (6.60%)
```

Empirical 6.60% matches theoretical **4 / 64 = 6.25%** (the 4 alphabet
characters whose top-4-bits group flips a same-group neighbor when toggled to
the corruption target).

### Why the wall-clock-second dependency feels like state pollution

`IAT = time.Now().Add(-365 * 24 * time.Hour).Unix()` changes by 1 each UTC
second. Within a single `go test -count=N` process, `time.Now().Unix()` is
mostly constant across iterations (a few seconds of wall clock typically
cover hundreds of iterations). So a single run is essentially "1 sample of
IAT". With ~6.25% bad seconds, ~1 of every 16 CI runs lands on a collision
second and reports 403 for **the entire run's iterations** — looking
statistically like a "this run failed across N runs" pattern that mimics
state-pollution-under-shuffle. But the IAT is independent of test ordering;
the flake is wall-clock dependent, period.

### Why macOS author runs pass deterministically

Authoring sessions exercise the test repeatedly over many seconds (multiple
edits + reruns), with enough wall-clock spread that several IAT seconds get
sampled. As long as one of the sampled seconds is non-collision (15 of 16
seconds are), the test passes on that run. The author would have to be very
unlucky to ever observe a failure during dev. Linux CI runs are short and
sample a single IAT; ~6% of those land on collision seconds → red CI.

### Why `-race -shuffle=on` is a red herring

Neither flag interacts with the bug. `-race` only changes goroutine scheduling
(no goroutine timing affects HMAC determinism). `-shuffle=on` reorders tests
but every test gets a fresh `*WebServer`. The flake reproduces with
`-shuffle=off` and without `-race` whenever the wall-clock second falls in a
bad bucket; both flags are just incidentally present in the CI invocation.

[VERIFIED: byte-level base64 decode table + IAT sweep + concrete IAT
demonstration; ASSUMED that Linux CI behavior matches the seconds-bucket model
— this is the strongest hypothesis consistent with all observed evidence
including the "Linux only" reports.]

## 6. Fix recommendation

**Replace the broken corruption with a guaranteed-401 path.** Three viable
variants, in increasing order of clarity:

### Variant A (RECOMMENDED): sign with a wrong key

```go
func issueExpiredCapFor(t *testing.T, ws *WebServer, sessionID, perms string) string {
    t.Helper()
    claims := capability.Claims{
        SID:     sessionID,
        Perms:   perms,
        IAT:     time.Now().Add(-365 * 24 * time.Hour).Unix(),
        GrantID: "grant-expired-" + sessionID,
        V:       1,
    }
    wrongKey := make([]byte, 32)
    for i := range wrongKey {
        wrongKey[i] = 0xFF // any value != capTestKey
    }
    token, err := capability.Sign(claims, wrongKey)
    if err != nil {
        t.Fatalf("capability.Sign: %v", err)
    }
    return token
}
```

**Why this is best:**
- Exercises the **production** `ErrInvalidSignature` path, which is what
  `requireCapability` already maps to 401 (`capability_mw.go:50-54`).
- Identical pattern to the already-passing
  `TestCapability_InvalidSignatureReturns401`
  (`capability_test.go:205-230`) — proven-stable approach used elsewhere in
  the file.
- Independent of base64 internals, claim layout, signature length, or alphabet.
- Independent of wall-clock time.
- Removes any temptation to misread the helper as "tests expiry" — there is no
  expiry path in `capability.Verify` to exercise; the helper's real job is
  "produce a token that fails HMAC."

**Caveat / scope creep watch:** rename `issueExpiredCapFor` to
`issueInvalidCapFor` to match what it actually does. The test name
`TestPluginConfigStream_ExpiredCap_Returns401` is locked by Issue #58 / CI
history, so leave the test name alone. Update the helper's comment block
(`plugin_config_stream_test.go:27-31, 50-53`) to drop the "expired" framing
and state the truth: "mints a cap that will fail HMAC verification."

### Variant B: flip a middle byte

```go
// Find the signature segment and flip a bit in a non-padding-affected byte.
if i := strings.LastIndex(token, "."); i >= 0 {
    sig := []byte(token[i+1:])
    if len(sig) > 0 {
        // Flip the FIRST byte of the signature, which encodes only data bits.
        if sig[0] == 'A' {
            sig[0] = 'B'
        } else {
            sig[0] = 'A'
        }
        token = token[:i+1] + string(sig)
    }
}
```

First-byte flips affect the top 6 data bits of byte 0 of the decoded sig.
Always produces a different decoded byte. **But** this still couples the test
to base64 internals; Variant A doesn't.

### Variant C: replace signature with constant garbage

```go
if i := strings.LastIndex(token, "."); i >= 0 {
    token = token[:i+1] + "AAAA" // 4 chars decode to 3 zero bytes — not 32 bytes, fails HMAC equality
}
```

Simplest. Produces `ErrInvalidSignature` because the decoded sig is 3 bytes,
not 32, so `hmac.Equal(sig, expected)` returns false (the slices are different
lengths). Slight risk: future changes to `Verify` that length-check explicitly
might shift the failure to a different sentinel — Variant A is more robust to
that.

### Why NOT add `t.Cleanup` anywhere

There is no shared state to clean up. `testServer` already registers
`t.Cleanup(ws.Stop)`. Adding ceremony for non-existent state pollution would
encode a wrong mental model in the test code.

[VERIFIED: code path analysis + audit of the parallel
`TestCapability_InvalidSignatureReturns401` pattern]

## 7. Verification strategy

### Local (macOS) — high-confidence smoke

```bash
go test -race -shuffle=on -count=1000 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/
```

Expected: 1000/1000 PASS. Variant A guarantees this by construction (the wrong
key produces `ErrInvalidSignature` deterministically; there is no probabilistic
element).

### Local — full webserver suite (regression check)

```bash
go test -race -shuffle=on -count=10 ./internal/webserver/
```

Expected: all tests pass, no other test in `internal/webserver/` is affected
by the helper change.

### Linux CI — gold standard (TEST-01 acceptance)

```bash
go test -race -shuffle=on -count=100 ./internal/webserver/
```

Expected: 100/100 PASS on Linux CI. Per CONTEXT.md, this is the locked
acceptance criterion.

### Mathematical proof of determinism (for the fix commit message)

Variant A's correctness reduces to:
> `hmac.New(sha256.New, K_wrong).Sum(payload) != hmac.New(sha256.New, K_right).Sum(payload)`
> for `K_wrong = 0xFF × 32`, `K_right = capTestKey`, any `payload`.

This is true with probability `1 - 2^-256` (HMAC-SHA256 second-preimage
resistance). The 2^-256 collision probability is below practical concerns. No
wall-clock dependence; no test-ordering dependence.

[VERIFIED: matches the existing `TestCapability_InvalidSignatureReturns401`
verification approach which has not flaked]

## Standard Stack

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go `testing` | 1.26.3 | Test runner | stdlib, no dependency |
| `crypto/hmac` + `crypto/sha256` | stdlib | HMAC-SHA256 | stdlib, constant-time `hmac.Equal` |
| `encoding/base64` (`RawURLEncoding`) | stdlib | token encoding | URL-safe alphabet, no padding chars |
| `internal/capability` | repo-local | Claims/Sign/Verify | already wired through `capability_mw.go` |
| `internal/webserver` test helpers | repo-local | `testServer`, `capTestKey` | already isolation-correct |

[VERIFIED: `go version`, code reads]

## Architecture Patterns

### Pattern: test-key wrong-signing for HMAC rejection

**What:** Sign with a deliberately-mismatched key to deterministically produce
`ErrInvalidSignature` on the verifier side.
**When to use:** any test that needs to assert "auth middleware rejects bad
HMAC" without depending on byte-level corruption of an already-signed token.
**Example** (`internal/webserver/capability_test.go:205-230`):

```go
// Source: internal/webserver/capability_test.go (TestCapability_InvalidSignatureReturns401)
wrongKey := make([]byte, 32)
for i := range wrongKey {
    wrongKey[i] = 0xFF
}
claims := capability.Claims{SID: "s-1", Perms: "read,write", IAT: time.Now().Unix(), GrantID: "g-1", V: 1}
token, _ := capability.Sign(claims, wrongKey)
// GET ... ?cap=token → expect 401
```

This test has never flaked. Apply the same pattern to `issueExpiredCapFor`.

### Anti-Patterns to Avoid

- **Byte-flipping inside an opaque base64 string** — without modeling
  padding-bit semantics, "flip the last byte" silently has a 6.25% chance of
  being a no-op for HMAC-SHA256 + `RawURLEncoding`. This is the bug.
- **Conflating "expired" with "invalid"** — `capability.Verify` has no expiry
  check. A helper that claims to mint "expired" tokens is misleading; rename
  to reflect what it actually does.
- **Adding `t.Cleanup` to chase a non-existent state leak** — every test in
  `internal/webserver/` already gets a fresh `*WebServer`; the fix must
  address the helper, not the harness.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HMAC rejection in a test | Byte-flip a base64-encoded sig | Sign with a wrong key | base64 padding bits cause silent no-op flips at 6.25% rate |
| Wall-clock-dependent test state | `time.Now()` in a deterministic check | inject a fixed `IAT` or skip the time field | non-deterministic input → non-deterministic test |

## Common Pitfalls

### Pitfall 1: Base64 padding-bit no-op flips

**What goes wrong:** Flipping a single character at the tail of a
base64-encoded fixed-length byte string can leave the decoded bytes unchanged
when the character's "free" (padding) bits absorb the flip.
**Why it happens:** `n` bytes encode to `ceil(4n/3)` base64 chars; when `n%3
!= 0`, the final 1 or 2 chars contain padding bits that the decoder discards.
**How to avoid:** flip a non-tail byte; or modify in the binary domain
pre-encode; or replace the entire field with a constant.
**Warning signs:** test passes "most of the time" but fails ~6% of CI runs
involving 32-byte HMAC values (SHA-256, AES-256 keys, Ed25519 sigs are all in
this family — 32 bytes → 43 chars → 4 data bits + 2 pad bits in the tail).

### Pitfall 2: Misreading wall-clock-dependent flake as state pollution

**What goes wrong:** A test whose result depends on `time.Now()` will look
like "fails under shuffle ordering X" when really it fails based on the UTC
second of the run.
**Why it happens:** `time.Now()` is mostly constant across iterations within a
single `go test -count=N`; CI runs sample 1–2 IAT seconds; ~6% of seconds are
bad → ~6% of CI runs fail across all iterations.
**How to avoid:** make `IAT` deterministic in tests (`const TestIAT = 0`) OR
remove the dependency on `IAT` entirely (Variant A above).
**Warning signs:** flake doesn't reproduce locally with high `-count`; cleanup
audits show no shared state; the test conceptually depends on a wall-clock
input.

### Pitfall 3: Information-disclosure-defense collapse (T-87-08)

**What goes wrong:** `requireCapability` deliberately collapses all `Verify`
error sentinels into a single 401 response. Tests that assert a specific
sentinel via the HTTP boundary will see only the 401. The path past Verify
(missing grant, wrong SID, disabled session) returns 403. Crossing the
401/403 boundary in test expectations is therefore meaningful — and exactly
what this bug does (silently shifts a test from the 401-path to the 403-path).
**How to avoid:** when a test expects 401, ensure Verify will actually fail.
Don't rely on "Verify will probably fail because I tampered with the token";
prove it by construction (Variant A signs with a key that **cannot** produce
the right HMAC).

## Code Examples

### Recommended fix (Variant A)

```go
// Source: pattern from internal/webserver/capability_test.go:205-230
// (TestCapability_InvalidSignatureReturns401), proven non-flaky.
func issueInvalidCapFor(t *testing.T, ws *WebServer, sessionID, perms string) string {
    t.Helper()
    claims := capability.Claims{
        SID:     sessionID,
        Perms:   perms,
        IAT:     time.Now().Unix(),                // IAT immaterial; Verify ignores expiry
        GrantID: "grant-invalid-" + sessionID,
        V:       1,
    }
    wrongKey := make([]byte, 32)
    for i := range wrongKey {
        wrongKey[i] = 0xFF
    }
    token, err := capability.Sign(claims, wrongKey)
    if err != nil {
        t.Fatalf("capability.Sign: %v", err)
    }
    return token
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| "Corrupt last byte of base64 token to invalidate sig" | "Sign with deliberately wrong key" | Phase 114 fix | Removes 6.25% probabilistic failure mode |

**Deprecated/outdated:**
- The "expired" framing of `issueExpiredCapFor` — `capability.Verify` has no
  expiry path; the helper's only job is to fail HMAC.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | test execution | ✓ | 1.26.3 (darwin/arm64) | — |
| Linux CI executor | TEST-01 final acceptance | (CI-only; not local) | GitHub Actions ubuntu-latest | — |

**No external services required.** Fix is pure Go test code edit.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) |
| Config file | none (`go.mod` only) |
| Quick run command | `go test -race -count=1 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/` |
| Full suite command | `go test -race -shuffle=on -count=100 ./internal/webserver/` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | `TestPluginConfigStream_ExpiredCap_Returns401` returns 401 in 100/100 runs on Linux CI under `-race -shuffle=on` | unit | `go test -race -shuffle=on -count=100 -run TestPluginConfigStream_ExpiredCap_Returns401 ./internal/webserver/` | ✅ (exists, fails ~6% of runs pre-fix) |
| TEST-02 | Root cause stated in writing, fix addresses root not symptom | review-time | n/a (verified by reading 114-RESEARCH.md + fix commit message + no `t.Skip`/retry/`-shuffle=off` in diff) | ✅ (this document + future fix commit) |

### Sampling Rate
- **Per task commit:** `go test -race -count=1 -run TestPluginConfigStream ./internal/webserver/` (< 5s)
- **Per wave merge:** `go test -race -count=10 ./internal/webserver/` (~30s)
- **Phase gate:** `go test -race -shuffle=on -count=100 ./internal/webserver/` on Linux CI, 100/100 pass

### Wave 0 Gaps
None — existing test infrastructure (`testServer`, `capTestKey`,
`internal/capability` Sign/Verify) covers all phase requirements. No new
fixtures, no framework install, no shared helpers needed.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | capability HMAC-SHA256 (existing) — not changed by this phase |
| V3 Session Management | yes | per-session web-enable + grant_id revocation (existing) — not changed |
| V4 Access Control | yes | `requireCapability` SID-bind + grant-list (existing) — not changed |
| V5 Input Validation | yes | `capability.Verify` strict base64 + sentinel error mapping (existing) — not changed |
| V6 Cryptography | yes | `hmac.Equal` constant-time compare (existing) — not changed |

**Phase 114 is test-code-only.** Production crypto, middleware, and access
control are unchanged. The fix tightens a test helper; it does not alter the
contract being tested. The contract — "any HMAC failure → 401, no
information disclosure" — remains intact and is now reliably exercised by the
test.

### Known Threat Patterns for Go HMAC capability tokens

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Forged token via wrong key | Spoofing | HMAC-SHA256 + `hmac.Equal` (constant-time) — already in `capability.Verify` |
| Timing oracle on signature comparison | Information Disclosure | `hmac.Equal` (not `bytes.Equal`) — already used |
| Error-class disclosure (401 vs 403 vs 500 leaks token structure) | Information Disclosure | T-87-08 collapse-to-401 in `requireCapability` — already in place |
| Test helper that silently passes ~6% of the time as a no-op | (test reliability, not threat) | Variant A: sign with wrong key, no byte-level tampering |

## Project Constraints (from CLAUDE.md)

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions. The
  fix is a small helper edit; no new functions added.
- **Testing:** Go `testing`; 80%+ coverage in critical components. Phase 114
  preserves existing coverage; no test is removed.
- **Make beliefs pay rent:** root cause is verified by reproduction + math,
  not asserted from training. Tag: [VERIFIED] throughout.
- **Notice confusion:** the "Linux-only flake" surprised the author. This
  research traces it to wall-clock-second sampling, not platform difference.
- **Chesterton's Fence:** `issueExpiredCapFor` exists to produce a 401-path
  token. Variant A preserves that intent while removing the bug; we are not
  removing the helper or its semantics.
- **Silent fallbacks:** the corruption block's `if c == 'A' { ... 'B' } else
  { ... 'A' }` is the canonical silent fallback — it appears to corrupt but
  silently doesn't, 6.25% of the time. Fix removes it entirely.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Linux CI failure rate is approximately the per-second collision rate (~6.25%) | Section 5 (wall-clock model) | If Linux failure rate is materially different (e.g. 50%), the model is incomplete and there may be a second cause. Risk: low — Issue #58 reports "~1 in N" which is consistent with 6.25%. Variant A still fixes it regardless. |
| A2 | The `?cap=<token>` query-string transit does not URL-encode the corrupted token in any way that disturbs the bytes-level comparison | Section 5 | `RawURLEncoding` uses URL-safe `-_` alphabet and no padding — no percent-encoding required. Risk: negligible. |
| A3 | Linux CI uses GitHub Actions `ubuntu-latest` with stock Go 1.26.x and stdlib base64 | Environment Availability | Risk: negligible — stdlib base64 behavior is platform-independent. |

## Open Questions

1. **Are there other test helpers in the codebase using the same "flip last
   byte of base64" pattern?**
   - What we know: this specific helper in `plugin_config_stream_test.go` is
     the one in Issue #58.
   - What's unclear: whether a sibling helper elsewhere (other packages) is
     latent-flaky for the same reason.
   - Recommendation: optional follow-up grep
     `grep -rn "corrupt\[len(corrupt)-1\]" .` after the fix lands. Out of
     scope for Phase 114 per CONTEXT.md "Out of scope: refactoring beyond
     what the fix needs."

2. **Should the test name be updated from `_ExpiredCap_` to `_InvalidCap_`?**
   - What we know: `capability.Verify` has no expiry path; the test name is
     historically wrong.
   - What's unclear: whether renaming breaks any external references (CI
     scripts, issue links, etc.).
   - Recommendation: leave the test name; update only the helper name and
     comments. CONTEXT.md doesn't authorize a rename; the test name is the
     ID the rest of the world knows.

## Sources

### Primary (HIGH confidence)
- `internal/webserver/plugin_config_stream_test.go` — test under investigation (lines 32-103)
- `internal/webserver/capability_mw.go` — 401/403 decision tree (lines 37-75)
- `internal/capability/capability.go` — Sign/Verify implementation (lines 30-80)
- `internal/webserver/capability_test_helpers.go` — `testServer` per-test isolation (lines 106-125)
- `internal/webserver/capability_test.go` — `capTestKey` definition + reference invalid-sig test (lines 26-32, 205-230)
- `internal/webserver/server.go` — `signingKey` / `webEnabled` / `grants` lifecycle (lines 57-231)
- `.planning/REQUIREMENTS.md` — TEST-01 / TEST-02 acceptance text (lines 56-57)
- `.planning/milestones/v3.3.1-ROADMAP.md` — Phase 114 success criteria (lines 141-159)
- `.planning/phases/114-ci-test-stability-webserver-capability-test-flake/114-CONTEXT.md` — locked decisions
- Standalone Go reproduction at `/tmp/proof.go` (since deleted) — deterministic IAT=1700000046 collision

### Secondary (MEDIUM confidence)
- Empirical IAT sweep (`/tmp/b64test.go`, deleted): 66/1000 = 6.60% collisions across 1000 IATs

### Tertiary (LOW confidence)
- None — every load-bearing claim is either code-read or executed-and-observed.

## Metadata

**Confidence breakdown:**
- Root cause identification: **HIGH** — empirically reproduced + mathematically explained from the base64 + HMAC primitives
- Test-state-pollution rule-out: **HIGH** — exhaustive grep + file audit
- Wall-clock-second model for the CI flake pattern: **HIGH** — matches theoretical 6.25% and the "Linux-only" symptom is parsimoniously explained
- Fix recommendation (Variant A): **HIGH** — identical pattern proven non-flaky elsewhere in the same file

**Research date:** 2026-05-18
**Valid until:** 2026-06-17 (30 days; stable Go stdlib, no fast-moving dependencies)
