---
phase: 155
slug: web-share-chat-ui-cross-surface-parity-gate
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-26
---

# Phase 155 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: derived from `155-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (`go test ./...`) + vitest + Playwright |
| **Config file** | `frontend/playwright.config.ts`, `frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/daemon/... -run Chat -count=1 && pnpm -C frontend test run src/components/Hub/` |
| **Full suite command** | `go test ./... && pnpm -C frontend test run && pnpm -C frontend exec playwright test` |
| **Estimated runtime** | ~90 seconds (Go+vitest fast; Playwright dominates) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... -run Chat -count=1 && pnpm -C frontend test run src/components/Hub/`
- **After every plan wave:** Run `go test ./... && pnpm -C frontend test run`
- **Before `/gsd-verify-work`:** Full suite (incl. Playwright e2e) must be green
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | EXPORT-01 | — | — | Go unit | `go test ./internal/daemon/... -run TestChatStore_Export` | No (`internal/daemon/chat_test.go`) | pending |
| TBD | TBD | TBD | EXPORT-01 | — | — | Go | `go test ./internal/webserver/... -run TestChatExport` | No (`internal/webserver/chat_test.go`) | pending |
| TBD | TBD | TBD | EXPORT-01 | — | — | Go | `go test ./internal/daemon/... -run TestChatRoutes_Export` | No (`internal/daemon/chat_routes_test.go`) | pending |
| TBD | TBD | TBD | EXPORT-01 | — | — | e2e | `pnpm -C frontend exec playwright test --grep "EXPORT-01"` | No (`frontend/e2e/chat-parity.spec.ts`) | pending |
| TBD | TBD | TBD | PARITY-01 | — | two clients exchange messages | e2e | `pnpm -C frontend exec playwright test --grep "PARITY-01"` | No (`frontend/e2e/chat-parity.spec.ts`) | pending |
| TBD | TBD | TBD | PARITY-01 | SEC-01 | RO viewer Send disabled + server rejects | e2e | `pnpm -C frontend exec playwright test --grep "RO viewer"` | No (`frontend/e2e/chat-parity.spec.ts`) | pending |
| TBD | TBD | TBD | PARITY-01 | — | ChatPanel renders on web-share (smoke) | vitest | `pnpm -C frontend test run src/components/Hub/WebShareSessionView.test.tsx` | No (new file) | pending |
| TBD | TBD | TBD | PARITY-01 | — | WebShareSessionView builds correct wsURL | vitest | `pnpm -C frontend test run src/components/Hub/WebShareSessionView.test.tsx` | No (new file) | pending |
| TBD | TBD | TBD | PARITY-01 | SEC-02 | @session inject from web: same frame/handler | Go unit | `go test ./internal/webserver/... -run TestInject` | Yes (`internal/webserver/inject_test.go`) | pending |

---

## Validation Architecture (from RESEARCH.md)

### Success Criteria → Validation Method

1. **Web-share participant sees identical thread/presence/typing/unread/@mention** → Playwright e2e two-context parity test (`frontend/e2e/chat-parity.spec.ts`) + vitest component smoke (`WebShareSessionView.test.tsx`).
2. **Export downloads `.md` with YAML frontmatter (both surfaces)** → Go unit on `ChatStore.Export()` + Go route tests (webserver + relay loopback) + Playwright download-trigger assertion.
3. **RO cap cannot post/inject; RW can (server-enforced)** → Playwright RO-viewer test asserts Send disabled AND server-side rejection; RW-viewer test asserts both succeed. (Server gate already verified in 152/153 tests.)
4. **@session inject identical from both surfaces** → existing Phase 153 `internal/webserver/inject_test.go` (same frame `0x35`, same handler, RW gate) + Playwright assertion of the "→ injected into terminal" indicator broadcast.

### TESTING.md gaps to resolve in this phase (standing convention)
- Add `frontend/e2e/chat-parity.spec.ts` to Section 2 (Playwright suite group).
- Add `frontend/src/components/Hub/WebShareSessionView.test.tsx` to Section 2 (vitest group).
- Add Section 4 traceability rows for EXPORT-01 and PARITY-01 (path column = repo-relative `.go`/`.ts`/`.tsx` only).
- Add a dedicated NOTIF-02 traceability row (minor gap carried from Phase 154 VERIFICATION).
- Add Phase 155 manual UAT items to Section 5 if any behavior cannot be automated.
- Run `bash tests/check-traceability-paths.sh` before committing.

### Wave 0 Gaps (must exist before requirement tests can run)
- [ ] `cmd/playwright-fixture/main.go` — wire `SetChatHistoryProvider` + `SetChatExportProvider` (currently absent; e2e cannot exercise chat/export without them).
- [ ] `frontend/e2e/chat-parity.spec.ts` — scaffolds EXPORT-01 + PARITY-01 coverage.
- [ ] `frontend/src/components/Hub/WebShareSessionView.test.tsx` — PARITY-01 component render + wsURL.
- [ ] YAML frontmatter unit test in `internal/daemon/chat_test.go`.

---

*Phase: 155-web-share-chat-ui-cross-surface-parity-gate*
*Validation strategy created: 2026-06-26 (from RESEARCH.md)*
