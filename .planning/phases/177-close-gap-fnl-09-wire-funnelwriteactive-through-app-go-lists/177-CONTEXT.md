# Phase 177: Close gap FNL-09 — wire funnelWriteActive to the native GUI — Context

**Gathered:** 2026-07-09
**Status:** Ready for planning
**Source:** v4.2 Milestone Integration Audit (`.planning/v4.2-MILESTONE-AUDIT.md` §3) — the audit IS the completed research for this gap; no discuss-phase / research pass is needed.

<domain>
## Phase Boundary

This is a **surgical, cross-phase integration gap-closure** phase, not a new feature. Phase 171 shipped public read-write ("FULL ACCESS") Funnel sharing. The write-share *flow* works end-to-end (hold-to-confirm consent gate, single-use write code, four-path teardown — all via `SetSessionFunnelWrite`/`DisableSessionFunnelWrite` RPCs). What is **silently dead in the shipped native GUI** is the persistent, colorblind-safe **FULL ACCESS exposure indicator** — because the `funnelWriteActive` field never crosses the daemon → app.go → frontend RPC seam.

**In scope:** wire the existing `SessionInfo.FunnelWriteActive` daemon field through the app.go Wails bridge and the frontend runtime bindings so the already-built consumers (badge, tab icon, modal resync) receive a real value; add a regression test that guards the seam; reconcile TESTING.md.

**Out of scope:** any change to the write-share flow itself, the consent gate, the badge/icon component styling, the daemon field (already correct), or the frontend consumers (already correct). This phase only closes the wiring between them. No new UI, no new RPC, no behavioral change to sharing.
</domain>

<decisions>
## Implementation Decisions (LOCKED — from audit §3)

### Root cause (confirmed at source)
The colorblind-safe FULL ACCESS indicator is dead because `funnelWriteActive` is dropped at the app.go Wails bridge and again at the frontend runtime binding:

1. `internal/daemon/types.go:40` — daemon `SessionInfo.FunnelWriteActive` is populated (`api.go:733`). ✅ Correct, do not touch.
2. `app.go` SessionInfo struct (~line 52) — has `FunnelActive` but **no `FunnelWriteActive`**. ✗ Fix.
3. `app.go` `ListSessions()` conversion (~line 521) — copies `FunnelActive: s.FunnelActive` but never `s.FunnelWriteActive`. ✗ Fix.
4. **CORRECTION to audit §3 point 4 (verified at source 2026-07-09):** the audit claimed the fix must also update `frontend/src/wailsjs/wailsjs/go/models.ts` `createFrom`. **That file is imported by nothing** (`grep` across `frontend/src` finds zero imports of the double-nested `wailsjs/wailsjs/go/models` tree). The app actually imports the `SessionInfo` **type** from the hand-authored stub `frontend/src/wailsjs/go/main/App.d.ts` (`App.tsx:41`), and the runtime value from `frontend/src/wailsjs/go/main/App.js`, whose `ListSessions = () => Call('main.App.ListSessions', [])` is a **raw passthrough with no `createFrom`/field-filtering**. Therefore at runtime `funnelWriteActive` is present in each session object **iff app.go serializes it** — the app.go fix (D-01/D-02) is necessary AND sufficient for the runtime. `App.d.ts` **already declares** `funnelWriteActive: boolean` (added by Phase 171-02, line 27) — which is exactly why `tsc` never flagged the dead field. Phase 166's `funnelBinding.contract.test.tsx` header already documented this pitfall ("updating only the generated wailsjs/wailsjs/go tree — not imported by the app").
5. Consumers already read `session.funnelWriteActive` and are correct — do not touch:
   - `frontend/src/App.tsx:1640-1642` — derives `funnelWriteActiveSessions` map from the hubSessions/ListSessions 3s poll.
   - `frontend/src/components/Hub/SessionCard.tsx:543,563` — FULL ACCESS badge.
   - `frontend/src/components/TabBar.tsx:270` — session-tab write-exposure icon.
   - `frontend/src/components/SessionShareModal.tsx` — out-of-band teardown resync.

### The fix (localized — mirror the working `funnelActive` reference in every spot)
- **D-01 — app.go SessionInfo struct:** add `FunnelWriteActive bool \`json:"funnelWriteActive"\`` beside `FunnelActive` (~line 52). NOT `omitempty` — `false` must serialize so the frontend poll detects teardown (same rule as `FunnelActive`/`BrowseEnabled`/`HomeDir`; a silent false-drop is a UAT-class bug). Carry a comment citing Phase 171 / FNL-09.
- **D-02 — app.go ListSessions conversion:** add `FunnelWriteActive: s.FunnelWriteActive,` beside the `FunnelActive: s.FunnelActive` line (~line 521).
- **D-03 — frontend binding (verify-only + optional hygiene):** the imported frontend path needs **no source change** for the feature to work. `frontend/src/wailsjs/go/main/App.d.ts` (the imported type) already declares `funnelWriteActive: boolean` (Phase 171-02, line 27) — **verify it is present and correctly typed; do not re-add or diverge it.** `App.js` `ListSessions` is a raw passthrough — no change. The non-imported `frontend/src/wailsjs/wailsjs/go/models.ts` `createFrom` is NOT load-bearing; **optionally** add `funnelWriteActive` there for generated-tree hygiene/future-proofing (harmless, keeps the generated copy honest if a real `wails build` ever regenerates and imports switch), but it MUST NOT be presented or tested as "the fix." NOTE: that file already has an unrelated uncommitted working-tree change (audit §5) — do not clobber it.
- **D-04 — no `wails build` regeneration in this flow:** do NOT trigger a full `wails build` regeneration (established project pattern; the 170-03 deviation hand-authored these stubs because `wails build -tags wailsassets` is not run in plan/execute). The app.go change (D-01/D-02) is the runtime fix; the D-05 test is the durable seam guard.

### The regression guard (the audit's key ask — closes the contract-test blind spot)
- **D-05 — Go struct-parity / serialization round-trip test (MANDATORY, load-bearing):** the existing `frontend/src/components/__tests__/funnelBinding.contract.test.tsx` asserts the `App.d.ts` stub *text*, not the real Go bridge — it certified this broken seam green (same failure mode as [[feedback_tests_encoding_same_wrong_assumption]]). Add a **genuine Go test** that fails if app.go drops a daemon funnel field. It must exercise the **real app.go path** (struct + `ListSessions` serialization), e.g.: (a) reflection-based parity asserting `app.SessionInfo` carries every funnel-exposure field of `daemon.SessionInfo` (`FunnelActive`, `FunnelWriteActive`) with matching json tags; and/or (b) a `ListSessions`/JSON-marshal round-trip asserting a daemon `SessionInfo{FunnelWriteActive:true}` yields `"funnelWriteActive":true` in the serialized app.go response. It MUST fail against pre-fix code and pass after (executor should prove the RED→GREEN). This is what actually protects the runtime seam — the JSON is what App.js passes through raw.
- **D-05b — optional frontend guard (cheap, on the ACTUALLY-imported stub):** extend the existing `funnelBinding.contract.test.tsx` to also assert `App.d.ts` contains `funnelWriteActive: boolean` (mirrors its existing `funnelActive: boolean` assertion). This guards the imported stub, not the dead generated tree. Do not regress the existing assertions.

### Claude's Discretion
- Exact Go test file/function name and whether the D-05 parity check is reflection-based, an explicit `ListSessions`/marshal round-trip, or both (any satisfies D-05 as long as it exercises the REAL app.go struct + serialization, not stub text or the non-imported `models.ts`).
- Whether to include the optional D-03 models.ts hygiene edit and the optional D-05b frontend assertion (both cheap; neither is load-bearing).
- Task/wave decomposition and commit granularity.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The gap diagnosis (authoritative spec)
- `.planning/v4.2-MILESTONE-AUDIT.md` — §3 "FNL-09 / FUI-03: funnelWriteActive never reaches the native GUI" is the full root-cause + fix. This CONTEXT distills it; read the audit for the complete chain and rationale.

### Source of the seam (read before editing)
- `internal/daemon/types.go` (line ~40) — daemon `SessionInfo.FunnelWriteActive` (source of truth, do not change).
- `internal/daemon/api.go` (line ~733) — where the daemon populates the field.
- `app.go` (SessionInfo struct ~31-60; `ListSessions()` conversion ~505-521) — the broken seam; mirror the `FunnelActive` handling.
- `frontend/src/wailsjs/wailsjs/go/models.ts` (`SessionInfo` ~225-252) — runtime binding to fix.
- `frontend/src/wailsjs/go/main/App.d.ts` (line ~27) — hand-authored stub, keep consistent.

### Consumers (already correct — reference only, do not change)
- `frontend/src/App.tsx:1640-1642`, `frontend/src/components/Hub/SessionCard.tsx:543,563`, `frontend/src/components/TabBar.tsx:270`, `frontend/src/components/SessionShareModal.tsx`.

### Project conventions
- `TESTING.md` (repo root) — Section 2 (Suite Manifest), Section 4 (Traceability), Section 6 (Standing Convention). The new Go regression test MUST be added to the suite manifest + a traceability row for FNL-09. Run `bash tests/check-traceability-paths.sh` before committing.
- `./CLAUDE.md` + `/Users/ken/dev/CLAUDE.md` — Go conventions (`go fmt`, context-aware), TS conventions.
</canonical_refs>

<specifics>
## Specific Ideas

- The working `funnelActive` path (Phase 165 / FNL-01) is the exact reference to mirror in all four spots (app.go struct, app.go ListSessions, models.ts declaration, models.ts constructor). Grep `FunnelActive` / `funnelActive` and add the write-sibling beside each.
- Colorblind-safety of the indicator (label + non-color icon + shape) is already built into the consumers per [[user_colorblind]]; this phase does NOT re-verify the visual treatment — it only makes the components receive a live `true/false` so they render at all. UAT confirmation is a live native-GUI check that the FULL ACCESS badge/tab icon actually appears when public write is enabled.
</specifics>

<deferred>
## Deferred Ideas

None — the gap is fully bounded by the audit. The write-share flow, consent gate, and component styling are all already shipped and out of scope.
</deferred>

---

*Phase: 177-close-gap-fnl-09-wire-funnelwriteactive-through-app-go-lists*
*Context gathered: 2026-07-09 — distilled from v4.2 Milestone Integration Audit §3 (research-complete; discuss/research skipped per [[feedback_skip_discuss_when_research_complete]])*
