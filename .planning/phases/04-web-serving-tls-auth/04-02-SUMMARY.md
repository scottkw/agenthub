---
phase: 04-web-serving-tls-auth
plan: 02
subsystem: webserver/auth
tags: [go, bcrypt, sessions, tokens, crypto]
dependency_graph:
  requires: []
  provides: [webserver/auth, webserver/tokens]
  affects: [04-03-webserver-routes]
tech_stack:
  added: [golang.org/x/crypto/bcrypt]
  patterns: [bcrypt-password-hashing, crypto-rand-sessions, opaque-token-store]
key_files:
  created:
    - internal/webserver/auth.go
    - internal/webserver/auth_test.go
    - internal/webserver/tokens.go
    - internal/webserver/tokens_test.go
  modified: []
decisions:
  - "AuthManager sessions map uses string->time.Time (creation time stored for future expiry support)"
  - "TokenStore uses two maps (tokenToSession + sessionToTokens) for O(1) lookup and O(n) bulk revocation"
  - "Revoke cleans up empty sessionToTokens entries immediately (no tombstones)"
metrics:
  duration: 3min
  completed_date: "2026-03-18"
  tasks_completed: 2
  files_created: 4
requirements_satisfied: [WEB-04, WEB-05]
---

# Phase 04 Plan 02: Authentication Layer Summary

**One-liner:** bcrypt password hashing with crypto/rand session cookies plus opaque token store for per-session sharing links.

## What Was Built

### Task 1: Password Hashing and Cookie Session Auth (`auth.go`)

- `HashPassword` / `CheckPassword` using `bcrypt.GenerateFromPassword` / `bcrypt.CompareHashAndPassword`
- `AuthManager` struct with `sync.RWMutex`-protected `passwordHash []byte` and `sessions map[string]time.Time`
- `SetPassword` hashes and stores; `LoadPasswordHash` loads pre-existing hash from config
- `Login` validates password then generates a 32-byte `crypto/rand` session cookie value (base64url)
- `IsAuthenticated` validates cookie against sessions map
- `MakeSessionCookie` returns `httpOnly=true, Secure=true, SameSite=Strict, Path=/` cookie
- `ErrInvalidPassword` sentinel error for failed login

### Task 2: Opaque Token Generation and Session-Scoped Lookup (`tokens.go`)

- `GenerateToken` produces 43-char base64url string from 32-byte `crypto/rand` read
- `TokenStore` with bidirectional maps: `tokenToSession` and `sessionToTokens`
- `Create(sessionID)` generates token and atomically updates both maps under write lock
- `Lookup(token)` reads under RLock for concurrent-safe access
- `Revoke(token)` removes from both maps, prunes empty session entries
- `RevokeBySession(sessionID)` bulk removes all tokens for a session
- `TokensForSession(sessionID)` returns a copy for UI display

## Verification

```
go test ./internal/webserver/... -v -timeout 30s
```

All 22 tests pass (6 auth + 4 network + 5 TLS + 7 token).

## Deviations from Plan

None - plan executed exactly as written. The `tls.go` implementation discovered in the webserver directory was already committed (plan 04-01 was partially executed before this plan ran), so no blocking deviation occurred.

## Self-Check

### Files Created
- `/Users/ken/dev/agenthub/internal/webserver/auth.go` - EXISTS
- `/Users/ken/dev/agenthub/internal/webserver/auth_test.go` - EXISTS
- `/Users/ken/dev/agenthub/internal/webserver/tokens.go` - EXISTS
- `/Users/ken/dev/agenthub/internal/webserver/tokens_test.go` - EXISTS

### Commits
- `256c9a8` - test(04-02): add failing tests for auth (RED)
- `086438b` - feat(04-02): implement AuthManager (GREEN)
- `1fbf4a9` - test(04-02): add failing tests for TokenStore (RED)
- `5a9bc50` - feat(04-02): implement TokenStore (GREEN)

## Self-Check: PASSED
