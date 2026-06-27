---
phase: 155
slug: web-share-chat-ui-cross-surface-parity-gate
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-26
validated: 2026-06-27
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
| T1 | 155-01 | 1 | EXPORT-01 | — | YAML-frontmatter serializer | Go unit | `go test ./internal/daemon/... -run TestChatStore_Export` | Yes (`internal/daemon/chat_test.go`) | COVERED |
| T2 | 155-01 | 1 | EXPORT-01 | T-155-03 | cap-gated export route (missing cap → 401) | Go | `go test ./internal/webserver/... -run TestChatExport` | Yes (`internal/webserver/chat_test.go`) | COVERED |
| T2 | 155-01 | 1 | EXPORT-01 | — | relay loopback export route | Go | `go test ./internal/daemon/... -run TestChatRoutes_Export` | Yes (`internal/daemon/chat_routes_test.go`) | COVERED |
| T2 | 155-04 | 3 | EXPORT-01 | — | export download (.md + frontmatter) both surfaces | e2e | `pnpm -C frontend exec playwright test --grep "EXPORT-01"` | Yes (`frontend/e2e/chat-parity.spec.ts`) | COVERED |
| T2 | 155-04/05 | 3 | PARITY-01 | — | two clients exchange messages (SC-1 broadcast) | e2e | `pnpm -C frontend exec playwright test --grep "PARITY-01"` | Yes (`frontend/e2e/chat-parity.spec.ts`) | COVERED |
| T2 | 155-04/06 | 3 | PARITY-01 | SEC-01 / T-155-06 | RO viewer Send disabled + server rejects (SC-3) | e2e | `pnpm -C frontend exec playwright test --grep "RO viewer"` | Yes (`frontend/e2e/chat-parity.spec.ts`) | COVERED |
| T1 | 155-03 | 2 | PARITY-01 | — | WebShareSessionView renders on web-share (smoke) | vitest | `pnpm -C frontend test run src/components/Hub/WebShareSessionView.test.tsx` | Yes | COVERED |
| T3 | 155-03 | 2 | PARITY-01 | — | WebShareSessionView builds correct wsURL (both children) | vitest | `pnpm -C frontend test run src/components/Hub/WebShareSessionView.test.tsx` | Yes | COVERED |
| — | 153 / 155-04 | 3 | PARITY-01 | SEC-02 | @session inject from web: same frame/handler, RW gate | Go unit | `go test ./internal/webserver/... -run TestInject` | Yes (`internal/webserver/inject_test.go`) | COVERED |
| T2 | 155-04 | 3 | PARITY-01 | — | @session inject indicator (.chat-msg--inject) SC-4 | e2e | `pnpm -C frontend exec playwright test --grep "inject indicator"` | Yes (`frontend/e2e/chat-parity.spec.ts`) | COVERED |

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
- [x] `cmd/playwright-fixture/main.go` — wired `SetChatHistoryProvider` + `SetChatExportProvider` (Plan 155-04, commit dff0e20a).
- [x] `frontend/e2e/chat-parity.spec.ts` — EXPORT-01 + PARITY-01 coverage (Plan 155-04; broadcast/RO fixes 155-05/06).
- [x] `frontend/src/components/Hub/WebShareSessionView.test.tsx` — PARITY-01 component render + wsURL (Plan 155-03, 11 tests).
- [x] YAML frontmatter unit test in `internal/daemon/chat_test.go` (`TestChatStore_Export`, Plan 155-01).

---

## Validation Audit 2026-06-27

State A audit of the pre-execution draft contract against the executed phase. All
9 mapped requirement→test pairs classified COVERED with green evidence re-run this
session (no auditor needed — zero gaps).

| Metric | Count |
|--------|-------|
| Requirements mapped | EXPORT-01, PARITY-01 (2) |
| Test rows audited | 10 |
| COVERED | 10 |
| PARTIAL | 0 |
| MISSING | 0 |
| Gaps found | 0 |
| Resolved | 0 (none needed) |
| Escalated | 0 |

**Evidence (re-run 2026-06-27):**
- `pnpm -C frontend exec playwright test chat-parity` → **24/24 passed** (chromium/firefox/webkit) — EXPORT-01 SC-2, PARITY-01 SC-1/SC-3/SC-4.
- `go test ./internal/daemon/... -run 'TestChatStore_Export|TestChatRoutes_Export'` → ok.
- `go test ./internal/webserver/... -run 'TestChatExport'` → ok (incl. missing-cap → 401).
- `go test ./internal/webserver/... -run TestInject` → `TestInjectRO_WebPath` PASS.
- `go test -race ./internal/relay/...` (two-phase subscribe) → ok.
- `pnpm -C frontend test run src/components/Hub/WebShareSessionView.test.tsx` → 11/11 passed.
- `bash tests/check-traceability-paths.sh` → exit 0.

No new test files generated; the phase already shipped full automated coverage.
Phase 155 is **Nyquist-compliant**.

---

*Phase: 155-web-share-chat-ui-cross-surface-parity-gate*
*Validation strategy created: 2026-06-26 (from RESEARCH.md)*
*Audited & marked compliant: 2026-06-27*
