# Phase 137: Share Modal & Cap Model - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

A per-Hub-card **Share** modal that folds in everything the removed Sessions page (`DaemonManagerPanel`) provided — web-serve on/off, read-only (RO) + read/write (RW) links/codes, QR codes, LAN Basic Auth password — **plus** a new **"Enable remote file browsing"** toggle whose file-browse permission **inherits from the share code the visitor presents** (RO code → read-only browse; RW code → read/write browse).

This phase delivers SHARE-01..06. The cap-model change (coupling file-browse to the share code) is the **security-sensitive backend change isolated for review** — a later `/gsd:secure-phase` pass audits it against the threat model.

**Out of scope (other phases):** card resize/redesign & local/remote indicators (CARD-01..04 → Phase 138/139); mini-preview/tail VT render fix #96 (CARD-05 → Phase 139); the visual redesign direction and exact card styling (RDS → Phase 140 UI-spec gate).

</domain>

<decisions>
## Implementation Decisions

### Cap model — RW-browse write gating (security core)
- **D-01:** When "Enable remote file browsing" is ON, file permission **inherits fully from the share code presented** — no separate owner write opt-in, no viewer confirm. The single per-session browse toggle is the only owner-side gate.
- **D-02:** This **removes the CAP-05 viewer "Allow file editing" two-gate** and the separate per-session `files.write` opt-in (`SetSessionFilesWrite` is subsumed by the browse toggle). **Justification:** the RW code already grants full terminal read+write (complete session control), so granting `files.write` to that same code is not a privilege escalation beyond what the holder already has. The owner's act of (a) enabling browse and (b) handing out the RW code IS the consent model.

### Cap model — resulting perms matrix
- **D-03:** **Browse OFF (default):** RO code = `"read"`; RW code = `"read,write"`. Neither code grants any filesystem access.
- **D-04:** **Browse ON:** RO code = `"read,files.read"` (list/stat/read within sandbox, no writes); RW code = `"read,write,files.read,files.write"`. Symmetric inherit-from-code model.
- **D-05:** RO-code holders gaining read-only filesystem access is **intended and accepted** (confirmed). Today RO viewers have zero file access; enabling browse is the owner's explicit acceptance of exposing file contents to the RO link.

### Cap model — toggle scope, defaults, persistence
- **D-06:** The browse toggle is **per-session**, default **OFF** (preserves the CAP-08 opt-in spirit).
- **D-07:** **Remove the GLOBAL file-browsing setting** (`filesReadEnabled`) from Settings. The per-session browse toggle becomes the **sole** driver for injecting both `files.read` (both codes) and `files.write` (RW code only). No global master kill-switch / AND-gate.
- **D-08:** Browse-enabled state (and share-on state) is **ephemeral** — in-memory alongside the existing web-serve toggle; a daemon restart resets sharing/browsing to OFF and invalidates caps. Matches today's web-serve / `sessionWrites` lifecycle. The modal **seeds from server truth** on open (SHARE-05).
- **D-09:** The **CAP-06 home-directory write warning is retained** — when the session cwd is `$HOME`, the modal shows the home-dir warning before the owner enables browsing.

### Share modal UI & trigger
- **D-10:** Two toggles in the modal: ① **"Share the session"** (replaces "Web On"; when ON reveals RO + RW links/codes, copyable, each with a QR code, and the LAN Basic Auth password when the web server runs in local mode — SHARE-02/04); ② **"Enable remote file browsing"** (SHARE-03). The browse toggle is disabled/no-op when sharing is OFF.
- **D-11:** **Reuse + simplify the existing `SessionSharePanel`** inside the new per-card modal — strip the now-dead CAP-05 two-gate write UI, wire the single browse toggle. Preserves the proven cap/URL/QR/password lifecycle (off→on cache-clear, stale-URL cleanup, server-truth seeding) so SHARE-05 carries forward with least churn. Do **not** rebuild the share UI from scratch.
- **D-12:** A **dedicated "Share" button** on the Hub `SessionCard` opens the modal (SHARE-01 read literally). CARD-04 (Phase 138) handles the card resize to fit it.
- **D-13:** On **remote peer cards** (sessions the user does not own), the Share button is **visible but disabled with a tooltip** ("Only the session owner can share"). The disabled state must be **colorblind-safe** — not color alone (greyed + lock icon + tooltip), per the colorblind-owner release norm. Satisfies SHARE-06 (cannot re-share an unowned session).

### Claude's Discretion
- Exact mechanism for collapsing `SetSessionFilesWrite` / `filesReadEnabled` into the new per-session browse state in `SessionEngine` (rename, repurpose `sessionWrites`, or new field).
- Cap-reissue-on-toggle plumbing (the existing `issueCapabilitiesForSession` already mints both caps; how the browse flag flows into perm injection).
- Migration/cleanup of the removed global Settings control (dead-config removal noted below).
- Final modal copy/labels and exact disabled-state iconography (subject to Phase 140 UI-spec).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` §"Share Modal (SHARE)" — SHARE-01..06 (the authoritative requirement text; SHARE-05 lists the lifecycle behaviors that must not regress).
- `.planning/ROADMAP.md` — Phase 137 entry ("security-sensitive backend change isolated for review").
- `.planning/PROJECT.md` — v4.0 milestone scope, "Hub card & Share modal" section (two-toggle design, inherit-from-code).

### Capability model (backend — security-sensitive)
- `internal/capability/capability.go:15-61` — `Claims` struct, `PermFilesRead`/`PermFilesWrite` constants, whole-token `HasPerm()` (never `strings.Contains`).
- `internal/daemon/api.go:1060-1146` — `issueCapabilitiesForSession()` mints both RO + RW caps; perm injection happens here (lines 1095-1116). This is the primary edit site for D-03/D-04.
- `internal/daemon/api.go:1033-1058` — `handleWebServe()` toggle (enable/disable + `ClearGrants`).
- `internal/daemon/engine.go:588-633` — `filesReadEnabled()` (global, to be removed — D-07), `filesWriteEnabledFor()` / `SetSessionFilesWrite()` (per-session write, to be subsumed — D-02/D-07), `sessionCwdIsHome()` (CAP-06 warning source — D-09).
- `internal/webserver/capability_mw.go:37-170` — `requireCapability`/`requireFilesRead`/`requireFilesWrite` enforcement (the consumers of the injected perms; CSRF Origin check on write).
- `internal/capability/joincode.go:33-93` — single-use, 5-min-TTL join codes (one per cap).

### Prior security decisions this phase reconciles
- `.planning/milestones/v3.5-phases/127-web-share-write-security-hardening/127-CONTEXT.md` — capability-escalation audit baseline; T-124-07 (write per-session, never global), CAP-05 (viewer two-gate — being removed), CAP-08 (write opt-in default OFF), the SECURITY artifact pattern for the later secure-phase.

### Frontend
- `frontend/src/components/DaemonManagerPanel.tsx` — the page being removed; source of the web-serve/IssueCapabilities/password flow to migrate.
- `frontend/src/components/SessionSharePanel.tsx` — the component to **reuse + simplify** (D-11); contains the existing two link rows, QR, codes, password, and the CAP-05 two-gate UI to strip.
- `frontend/src/components/Hub/SessionCard.tsx` — where the dedicated Share button is added (D-12); has an overflow menu today but no Share entry.
- Wails bindings: `ToggleWebServing`, `IssueCapabilities`, `GetLocalNetworkPassword` (and the to-be-retired `SetSessionFilesWrite`) — `internal/daemon/client.go`.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`SessionSharePanel.tsx`** — already renders RO + RW link rows, join codes, QR, and LAN password; reuse-and-simplify rather than rebuild (D-11).
- **`issueCapabilitiesForSession()`** — already mints BOTH caps per toggle-on; the change is gating which perms get injected based on the new per-session browse flag, not new minting infrastructure.
- **`sessionCwdIsHome()` / CAP-06** — existing home-dir detection drives the retained warning (D-09).
- **LAN password** — already generated once per daemon lifetime (`a.localPassword`) and surfaced via `GetLocalNetworkPassword()`; just relocate into the modal.

### Established Patterns
- Whole-token `HasPerm()` semantics — never substring-match perms; inject `files.read`/`files.write` as discrete perms into the existing comma-joined `Perms` string.
- In-memory toggle lifecycle: web-serve and `sessionWrites` are not persisted; the new browse flag follows the same ephemeral pattern (D-08).
- Toggle-off clears all grants (`ClearGrants`) → invalidates outstanding caps; cap re-issue happens on next toggle-on (server-truth seeding for SHARE-05).

### Integration Points
- `SessionEngine` (engine.go): collapse `filesReadEnabled` (remove) + `filesWriteEnabledFor`/`SetSessionFilesWrite` (subsume) into one per-session browse-enabled state feeding `issueCapabilitiesForSession`.
- Settings page: remove the global file-browsing control (D-07) — coordinate the Settings UI change + dead-config cleanup.
- `SessionCard` → Share modal → reused `SessionSharePanel`; remote-card ownership check gates the disabled state (D-13).

</code_context>

<specifics>
## Specific Ideas

- The cap-model coupling is the explicit security-review target — keep the backend perm-injection change isolated and well-commented so the later secure-phase / security-auditor can audit the RO-gets-files.read and RW-gets-files.write deltas against the threat model. Note the deliberate reversals: T-124-07 (write no longer separately gated), CAP-05 (viewer confirm removed), CAP-08/global files.read (global setting removed).
- Cross-surface parity is GUI/CLI/web (TUI dropped). The Share modal is GUI-issued; the web surface consumes the caps. Verify the web file-browse surface honors RO `files.read` (read-only) vs RW `files.read,files.write` correctly under the new coupling.

</specifics>

<deferred>
## Deferred Ideas

- Persisting share/browse state across daemon restarts with cap-reissue-on-startup — deferred (chose ephemeral, D-08); revisit only if long-lived-session UX demands it (larger security-review surface).
- Card visual redesign, local/remote + connected/available indicators, and mini-preview/tail VT render fix — CARD-01..05 / RDS belong to Phases 138-140.

None outside that — discussion stayed within phase scope.

</deferred>

---

*Phase: 137-share-modal-cap-model*
*Context gathered: 2026-06-20*
