---
phase: 152-relay-protocol-identity-presence
plan: "02"
subsystem: daemon/alias-store
tags: [alias, persistence, identity, json, tdd]
dependency_graph:
  requires:
    - 152-01  # relay.ValidateAlias (called by AliasStore.Set)
  provides:
    - AliasStore type and NewAliasStore/Get/Set/GetOrDefault API
    - aliases.json persistence at configDir (daemon-owned, 0600)
  affects:
    - internal/daemon/engine.go (aliasStore field wired in Plans 05/06)
    - internal/relay/server.go (Subscriber.AliasSetFn callback in Plans 05/06)
    - internal/webserver/server.go (handleWSSRelay in Plan 06)
tech_stack:
  added: []
  patterns:
    - ChatStore mu+filePath+in-memory-map idiom (RWMutex variant for read-heavy access)
    - loadSettingsFromDisk / saveSettingsToDisk engine.go pattern
    - t.TempDir() constructor-level test isolation (same as NewChatStore)
key_files:
  created:
    - internal/daemon/alias_store.go
    - internal/daemon/alias_store_test.go
  modified:
    - TESTING.md
decisions:
  - "RWMutex over Mutex: reads (every subscribe) are far more frequent than writes (user alias change)"
  - "Roll back in-memory map on persist failure: store stays consistent with disk even when WriteFile fails"
  - "Hardcoded aliases.json basename: personKey never enters the file path (T-152-06 mitigation)"
  - "relay.ValidateAlias delegated by Set: invalid aliases can never reach disk even if a future caller forgets to pre-validate"
metrics:
  duration: "~2 minutes"
  completed: "2026-06-26"
  tasks_completed: 1
  files_created: 2
  files_modified: 1
status: complete
requirements: [IDENT-02]
---

# Phase 152 Plan 02: AliasStore Persistence Summary

AliasStore implements daemon-owned JSON persistence for composite personKey→alias
mapping, using RWMutex + fixed-path aliases.json with 0600 perms and relay.ValidateAlias
gating every write.

## What Was Built

`internal/daemon/alias_store.go` — new file, package daemon.

**Type:** `AliasStore{ mu sync.RWMutex; filePath string; aliases map[string]string }`

**API:**
- `NewAliasStore(configDir string) (*AliasStore, error)` — MkdirAll 0700, fixed-path `aliases.json`, load from disk (IsNotExist is silent first-run), empty map ready to use
- `Get(personKey string) (string, bool)` — RLock; map lookup
- `GetOrDefault(personKey, def string) string` — if Get found return it, else return def
- `Set(personKey, alias string) error` — validates via `relay.ValidateAlias`; on "" returns error without mutating or persisting; writes under Lock then calls persist; rolls back in-memory on persist failure

**File path:** `filepath.Join(configDir, "aliases.json")` — basename hardcoded, personKey is never a path component (T-152-06).

**Persistence:** `json.Marshal(a.aliases)` + `os.WriteFile(filePath, data, 0600)` — serialized under write lock.

## Commits

| Hash | Description |
|------|-------------|
| ec88b0bc | test(152-02): add failing tests for AliasStore persistence (RED) |
| 81c38127 | feat(152-02): implement AliasStore with JSON persistence (GREEN) |
| d6e9eb26 | docs(152-02): update TESTING.md with AliasStore test (IDENT-02) |

## TDD Gate Compliance

- RED gate: `ec88b0bc` — `test(152-02)` commit with 9 failing tests (compilation failure: `NewAliasStore` undefined)
- GREEN gate: `81c38127` — `feat(152-02)` commit, all 9 tests pass under `-race`

## Verification

```
=== RUN   TestAliasStoreBasicGetSet          --- PASS
=== RUN   TestAliasStoreGetAbsent            --- PASS
=== RUN   TestAliasStoreGetOrDefault         --- PASS
=== RUN   TestAliasStoreReloadPersistence    --- PASS (D-02)
=== RUN   TestAliasStoreCompositeKeyIsolation --- PASS
=== RUN   TestAliasStoreRejectInvalidAlias   --- PASS (T-152-01)
    --- PASS: /40_runes_(over_limit)
    --- PASS: /control_char_U+0007
    --- PASS: /empty_after_trim
    --- PASS: /empty_string
=== RUN   TestAliasStoreFilePerms            --- PASS (0600)
=== RUN   TestAliasStoreFirstRunNoFile       --- PASS
=== RUN   TestAliasStoreFixedBasename        --- PASS (T-152-06)
PASS  ok  github.com/scottkw/agenthub/internal/daemon 1.032s
```

Acceptance criteria:
- `alias_store.go` defines `AliasStore` and `NewAliasStore`/`Get`/`Set`/`GetOrDefault`
- `grep -n 'aliases.json' internal/daemon/alias_store.go` exits 0
- `grep -n 'relay.ValidateAlias' internal/daemon/alias_store.go` exits 0
- `go test -race ./internal/daemon/ -run TestAliasStore` exits 0 including reload-persistence and invalid-alias-rejected assertions
- No import cycle (`go build ./internal/daemon/` clean)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — AliasStore is fully implemented with real persistence. All behaviors wired end-to-end.

## Threat Flags

No new threat surface beyond what the plan's threat model covers. aliases.json lives at a daemon-controlled fixed path; personKey is only ever a JSON map key, never a path component; relay.ValidateAlias gates all writes.

## Self-Check: PASSED

- `internal/daemon/alias_store.go` — FOUND
- `internal/daemon/alias_store_test.go` — FOUND
- `ec88b0bc` — FOUND (git log)
- `81c38127` — FOUND (git log)
- `d6e9eb26` — FOUND (git log)
