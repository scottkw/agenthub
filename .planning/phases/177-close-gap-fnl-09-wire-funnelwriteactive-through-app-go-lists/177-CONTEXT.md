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
4. `frontend/src/wailsjs/wailsjs/go/models.ts` — the runtime `SessionInfo` class declares `funnelActive` and copies it in `createFrom` (lines ~227 / ~249) but has **no `funnelWriteActive`** in either the field list OR the constructor. ✗ Fix BOTH the declaration and the `createFrom`/constructor copy. (Audit's short fix-list understates this: even if app.go serializes the field, `createFrom` silently drops it during deserialization.)
5. Consumers already read `session.funnelWriteActive` and are correct — do not touch:
   - `frontend/src/App.tsx:1640-1642` — derives `funnelWriteActiveSessions` map from the hubSessions/ListSessions 3s poll.
   - `frontend/src/components/Hub/SessionCard.tsx:543,563` — FULL ACCESS badge.
   - `frontend/src/components/TabBar.tsx:270` — session-tab write-exposure icon.
   - `frontend/src/components/SessionShareModal.tsx` — out-of-band teardown resync.

### The fix (localized — mirror the working `funnelActive` reference in every spot)
- **D-01 — app.go SessionInfo struct:** add `FunnelWriteActive bool \`json:"funnelWriteActive"\`` beside `FunnelActive` (~line 52). NOT `omitempty` — `false` must serialize so the frontend poll detects teardown (same rule as `FunnelActive`/`BrowseEnabled`/`HomeDir`; a silent false-drop is a UAT-class bug). Carry a comment citing Phase 171 / FNL-09.
- **D-02 — app.go ListSessions conversion:** add `FunnelWriteActive: s.FunnelWriteActive,` beside the `FunnelActive: s.FunnelActive` line (~line 521).
- **D-03 — models.ts runtime binding:** add `funnelWriteActive: boolean;` to the `SessionInfo` field list AND `this.funnelWriteActive = source["funnelWriteActive"];` in the constructor, mirroring `funnelActive`. Keep the hand-authored `frontend/src/wailsjs/go/main/App.d.ts` stub (already has `funnelWriteActive` at line 27) consistent — verify it matches; do not diverge the two.
- **D-04 — binding-regeneration strategy:** hand-update `models.ts` (do NOT trigger a full `wails build` regeneration in this flow). This matches the established project pattern (the 170-03 deviation hand-authored the same stubs because `wails build -tags wailsassets` is not run in the plan/execute flow). The regression test (D-05) is what actually protects the seam long-term; the hand-edit is the immediate wiring.

### The regression guard (the audit's key ask — closes the contract-test blind spot)
- **D-05 — struct-parity / round-trip test (MANDATORY):** the existing `frontend/src/components/__tests__/funnelBinding.contract.test.tsx` asserts the `App.d.ts` stub *text*, not the real Go struct — it certified this broken bridge green (same failure mode as [[feedback_tests_encoding_same_wrong_assumption]]). Add a **genuine** guard that fails if app.go drops a daemon funnel field. Preferred: a **Go test** asserting `app.SessionInfo` mirrors `daemon.SessionInfo`'s funnel-exposure fields (`FunnelActive`, `FunnelWriteActive`) — e.g. reflection-based field/tag parity, or a `ListSessions`-level assertion that a daemon `SessionInfo{FunnelWriteActive:true}` round-trips to the app.go `SessionInfo` with `funnelWriteActive:true` in the serialized JSON. This is the durable regression backstop; it must fail against the pre-fix code and pass after.

### Claude's Discretion
- Exact test file name/location and whether the parity check is reflection-based vs. an explicit ListSessions round-trip (either satisfies D-05, as long as it exercises the REAL app.go struct/conversion, not stub text).
- Whether to also strengthen or leave the existing `funnelBinding.contract.test.tsx` (the new Go-side guard is the priority; do not regress the existing test).
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
