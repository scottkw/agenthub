---
phase: 135
slug: accessibility-hardening
status: verified
threats_open: 0
asvs_level: 1
created: 2026-06-19
register_authored_at_plan_time: true
---

# Phase 135 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Phase 135 is pure client-side accessibility hardening (CSS `:focus-visible` /
> `prefers-reduced-motion`, ARIA attributes, keyboard handlers, DOM `inert` focus
> management) over existing Hub components. **No new network surface, no new IPC/relay
> route, no new capability, no new untrusted input** is introduced.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| (none new) | All changes are client-side DOM/CSS in already-rendered local UI. No boundary is added or moved. The modal's existing data flow (TerminalPanel / RelayClient) is unchanged by this phase. | None new |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-135-01-01 | Tampering | `style.css` CSS rules | accept | Static, build-embedded CSS served from the app bundle; no user-controlled input reaches it. | closed |
| T-135-01-02 | Information Disclosure | focus ring visibility | accept | Focus rings reveal only which on-screen element is focused — already-visible UI state. | closed |
| T-135-01-SC | Tampering | npm/pip/cargo installs | n/a | Zero new dependencies (RESEARCH). No install task → legitimacy gate not applicable. | closed |
| T-135-02-01 | Tampering | filter/group keyboard handlers | accept | Handlers invoke existing callbacks (`onFilterChange`, `onGroupSelect`) with the same ids already reachable via `onClick`. No new code path/privilege. | closed |
| T-135-02-02 | Elevation of Privilege | keyboard activation of group/filter | accept | Keyboard activation reaches exactly the same client-side state changes as the existing mouse `onClick`. No backend call, no capability boundary. | closed |
| T-135-02-SC | Tampering | npm/pip/cargo installs | n/a | Zero new dependencies (RESEARCH). Legitimacy gate not applicable. | closed |
| T-135-03-01 | Denial of Service | `inert` never removed → Hub keyboard-lock | mitigate | **Mitigation verified present.** Every `hubEl.inert = true` is paired with a `useEffect` cleanup `hubEl.inert = false` on unmount; the WR-01 fix holds `inert` through `open` AND `exiting` and clears it on unmount. Guarded by the inert-lifecycle behavioral test in `HubModal.test.tsx` and confirmed by `135-VERIFICATION.md` (A11Y-04). Self-DoS usability risk, not an attacker vector. | closed |
| T-135-03-02 | Tampering | dialog-scoped Escape handler (WR-05) | accept | `stopPropagation` only halts DOM bubbling within the app's own component tree; crosses no trust boundary. Replacing `stopImmediatePropagation` *reduces* side-effect scope (correctness improvement). | closed |
| T-135-03-03 | Spoofing | `inert` / focus on background | accept | `inert` is a DOM-only focus/AT suppression on already-rendered local UI; no privilege, data, or remote surface involved. | closed |
| T-135-03-SC | Tampering | npm/pip/cargo installs | n/a | Zero new dependencies (`inert` is a native DOM property). Legitimacy gate not applicable. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-135-01 | T-135-01-01, T-135-01-02 | Static client-side CSS + focus-ring visibility expose no sensitive data and accept no untrusted input. | scottkw (plan-time disposition) | 2026-06-19 |
| AR-135-02 | T-135-02-01, T-135-02-02 | Keyboard/ARIA parity reaches the same client-side state as existing `onClick`; no new privilege or backend path. | scottkw (plan-time disposition) | 2026-06-19 |
| AR-135-03 | T-135-03-02, T-135-03-03 | Dialog-scoped Escape + DOM `inert` are local focus-management changes crossing no trust boundary. | scottkw (plan-time disposition) | 2026-06-19 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-19 | 10 | 10 | 0 | /gsd:secure-phase (short-circuit: threats_open:0, plan-time register; T-135-03-01 mitigation verified via code-review WR-01 fix + behavioral test + 135-VERIFICATION.md) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-19
