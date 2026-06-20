---
phase: 139
slug: card-rendering-tab-strip
status: ready
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-20
---

# Phase 139 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Split architecture (D-01a): local scrollback rendered Go-side (`charmbracelet/x/vt`),
> remote tail rendered JS-side (`@xterm` headless + `addon-serialize`). Two test stacks:
> `go test` for the daemon, `vitest` for the frontend.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (daemon) + vitest/pnpm (frontend) |
| **Config file** | go.mod ; frontend/vitest config (existing) |
| **Quick run command** | `go test ./internal/daemon/... -count=1` (Go) / `cd frontend && pnpm test -- --run` (FE) |
| **Full suite command** | `go test ./... -count=1 && (cd frontend && pnpm test -- --run)` |
| **Estimated runtime** | ~5s daemon pkg · ~30s full Go+FE |

---

## Sampling Rate

- **After every task commit:** Run the relevant quick command (Go task → `go test ./internal/daemon/...`; FE task → `pnpm test -- --run <files>`)
- **After every plan wave:** Run the full suite `go test ./... -count=1 && (cd frontend && pnpm test -- --run)`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 139-01-01 | 01 | 1 | CARD-05 | — | N/A (RED scaffold) | unit (Go, RED) | `go vet ./internal/daemon/ \| grep GetSessionStyledTailLines` | ❌ W0 | ⬜ pending |
| 139-01-02 | 01 | 1 | CARD-05, TAB-01..03 | — | N/A (RED scaffold) | unit (FE, RED) | `pnpm test -- --run vtColor.test MiniPreview.test TabBar.test` | ❌ W0 | ⬜ pending |
| 139-01-03 | 01 | 1 | CARD-05 | T-139-01 | A2 headless-xterm path proven before remote render | spike/verify | `node scripts/verify-xterm-headless.mjs` | ❌ W0 | ⬜ pending |
| 139-02-01 | 02 | 2 | TAB-01 | — | N/A | unit (FE) | `grep flex-shrink/container-type style.css && pnpm test -- --run TabBar.test` | ❌ W0 | ⬜ pending |
| 139-02-02 | 02 | 2 | TAB-02, TAB-03 | T-139-02 / T-139-03 | chevron keyboard a11y; floor affordances survive | unit (FE) | `grep ResizeObserver TabBar.tsx && pnpm test -- --run TabBar.test` | ❌ W0 | ⬜ pending |
| 139-03-01 | 03 | 2 | CARD-05 | T-139-04 / T-139-06 | VT grid bounded (cols×50, last-n) | unit (Go) | `go test ./internal/daemon/... -run TestGetSessionStyledTailLines -count=1` | ❌ W0 | ⬜ pending |
| 139-03-02 | 03 | 2 | CARD-05 | T-139-05 / T-139-07 | n-clamp [1..20] ×2; local-only (empty for remote ids) | unit (Go) | `go test ./internal/daemon/... -run TestHandleGetSessionStyledTailLines -count=1` | ❌ W0 | ⬜ pending |
| 139-04-01 | 04 | 3 | CARD-05 | T-139-07 | local path renders via React children (no innerHTML) | unit (FE) | `pnpm test -- --run vtColor.test MiniPreview.test` | ❌ W0 | ⬜ pending |
| 139-04-02 | 04 | 3 | CARD-05 | T-139-08 / T-139-09 | remote innerHTML = terminal-controlled agent output (risk accepted) | unit (FE) | `grep serializeAsHTML HubBriefingModal.tsx && pnpm test -- --run` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · File Exists ❌ W0 = test file authored in Wave 0 (Plan 01).*

*Checkpoint tasks (139-03 pkg-legitimacy gate; 139-04 final human-verify) are not auto-sampled — see Manual-Only Verifications.*

---

## Wave 0 Requirements

Authored RED by Plan 139-01 before any implementation:

- [ ] `internal/daemon/engine_test.go` — `TestGetSessionStyledTailLines_{ColorBold,TUI,Unknown}` stubs (CARD-05)
- [ ] `internal/daemon/api_test.go` — `TestHandleGetSessionStyledTailLines` stub (CARD-05)
- [ ] `frontend/src/lib/vtColor.test.ts` — `resolveColor` ansi:N/#rrggbb→ITheme stubs (CARD-05)
- [ ] `frontend/src/components/Hub/MiniPreview.test.tsx` — StyledSpan render + no-xterm guard (CARD-05 / CARD-07)
- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — flex-shrink/floor/chevron/title stubs (TAB-01..03)
- [ ] `frontend/scripts/verify-xterm-headless.mjs` (or `xtermHeadless.verify.test.ts`) — A2 assumption check (headless `@xterm` write + `serializeAsHTML` without `open()`)

*go test + vitest infrastructure already exists; no framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tab shrink to icon-only "favicon" floor | TAB-01 | Visual layout / proportional flex under window resize | Open enough tabs to exceed window width; confirm tabs shrink past 80px to status-dot + × only, names hidden |
| Overflow chevrons reach every tab | TAB-02 | Visual scroll affordance + position-aware show/hide | At the floor + more tabs, confirm ‹/› appear, ‹ hides at start, › hides at end, clicking scrolls to hidden tabs; keyboard-reachable |
| Floor affordances functional | TAB-03 | Interaction at minimum width | At the floor: right-click rename works, hover `title` shows full name, close × works, progress underline visible |
| VT color+bold render fidelity | CARD-05 | Colorblind-safe rule: confirm faithful agent colors (not app-status encoding) at source — themed `ITheme` mapping, no leaked escapes, columns aligned | Mini-preview card + briefing tail of a live Claude Code/TUI session render legibly; compare to interactive terminal; #96 doubled-line/leaked-escape artifacts gone |
| charmbracelet/x/vt package legitimacy | CARD-05 | Supply-chain gate ([ASSUMED] packages) | 139-03 blocking-human checkpoint: verify pkg.go.dev + proxy.golang.org for `x/vt` + `ultraviolet`, exact pin |
| Final phase human-verify | CARD-05 | End-to-end visual sign-off | 139-04 checkpoint: live render across both surfaces, both local + remote sessions |

---

## Validation Sign-Off

- [x] All auto tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive auto tasks without automated verify
- [x] Wave 0 covers all MISSING references (engine/api/vtColor/MiniPreview/TabBar tests + A2 script)
- [x] No watch-mode flags (`--run` used throughout)
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-20 (plan-phase; per-task map filled from plans 139-01..04). `wave_0_complete` flips true once Plan 139-01 executes RED.
