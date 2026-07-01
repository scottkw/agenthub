# Requirements: AgentHub v4.2 — Funnel Sharing & Polish

**Defined:** 2026-06-30
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

> Milestone closes GitHub Issues #107 (Funnel + Help guide), #110 (awaiting-input notifications), #112, #115, #116, #117, #118, #120, #121. Phase numbering continues from v4.1 (last = Phase 164) → v4.2 starts at **Phase 165**. Research: `.planning/research/SUMMARY.md`. Product decisions ratified at scoping (2026-06-30): risk acknowledgment is shown **per-enable, every time**; Funnel shares support **daemon-enforced auto-expiry**.

## v1 Requirements

Requirements for this milestone. Each maps to a roadmap phase.

### Funnel Backend (FNL)

- [x] **FNL-01**: A session owner can enable Tailscale Funnel on a shared session to expose it to the public internet; Funnel is off by default.
- [x] **FNL-02**: Enabling Funnel is configured node-locally through the embedded Tailscale LocalClient (`SetServeConfig`/`AllowFunnel`) and requires no Tailscale admin API token or ACL edit.
- [x] **FNL-03**: A recipient who is not on the host's tailnet and has no Tailscale account can open the Funnel share URL and join the session, still gated by the single-use join code + capability token.
- [x] **FNL-04**: When Funnel is active, the Origin allowlist, `BaseURL()`, and generated share URLs use the Funnel hostname, so the external join-code / capability-token exchange succeeds without a 403-before-auth.
- [x] **FNL-05**: Funnel exposure is fully torn down when the user disables it, when web-share is turned off, when the session ends, and on daemon shutdown — no session remains publicly exposed afterward.
- [x] **FNL-06**: When the tailnet has not enabled Funnel (prerequisite unmet), enabling fails with a clear, explanatory error that links to the in-app Help guide, rather than failing opaquely.
- [x] **FNL-07**: A Funnel share automatically expires after a user-chosen duration; the daemon tears the Funnel exposure down at expiry (enforced server-side, independent of any connected UI).

### Funnel UI / UX (FUI)

- [ ] **FUI-01**: Enabling Funnel requires passing a risk-acknowledgment dialog — shown every time — that states the session becomes reachable from the public internet, that the join code is the only gate for its TTL, and recommends short-lived, read-only / no-file-access shares.
- [ ] **FUI-02**: The risk dialog lets the user choose the share's auto-expiry duration before enabling.
- [ ] **FUI-03**: While a session is internet-exposed via Funnel, a persistent, colorblind-safe indicator (text label + non-color icon, never color alone) is shown on the session's surfaces.
- [ ] **FUI-04**: The user can stop Funnel / disable internet exposure with one click from the Share UI, which fully tears down the Funnel config (FNL-05).
- [ ] **FUI-05**: Once Funnel is active, the public Funnel share URL is displayed with copy-to-clipboard and a QR code.
- [ ] **FUI-06**: The risk dialog cross-links to the Help guide's contained-sharing alternative ("want tighter containment instead?").

### Help Guide (HLP) — Issue #107 Part 2

- [ ] **HLP-01**: A new in-app Help article, "Sharing a session with someone outside your tailnet," documents both the Funnel path and the contained device-share + ACL alternative (share a single device, tag it `tag:agenthub`, restrict shared users to AgentHub's port).
- [ ] **HLP-02**: The guide includes the copy-pasteable `autogroup:shared`→`tcp:7443` grant, explicitly calls out the wildcard-default (`*→*`) gotcha, and links to the relevant Tailscale docs.

### Notifications (NTF) — Issue #110

- [x] **NTF-01**: When a session transitions into the awaiting-input (`waiting`) state, the user receives a native OS notification on macOS, Windows, and Linux — including when the GUI window is hidden (tray-resident).
- [x] **NTF-02**: A notification fires once per transition into `waiting`, not repeatedly while the session remains `waiting`.
- [x] **NTF-03**: The notification identifies which session needs attention (session name + agent type).
- [x] **NTF-04**: Awaiting-input notifications can be toggled on/off in Settings → Session Behavior; default **off** (user opts in — avoids surprise notifications and first-run OS permission prompts).

### Hub / Settings Polish (UX)

- [ ] **UX-01**: A Settings → Session Behavior option lets the user prevent auto-switching to a newly Hub-created session (stay on the Hub tab). — Issue #116
- [ ] **UX-02**: The Footer "Enable Web" button is renamed "Share Session" and opens the Hub Share modal instead of directly toggling web serving, eliminating the state-drift between the button and the modal. — Issue #115

### Bug Fixes (FIX)

- [ ] **FIX-01**: Web-share guests on the `/app/` surface receive live plugin-config and SSE hot-swap updates (self-fetched via the capability-gated `/api/plugin-config` + SSE endpoints), restoring the parity lost after the Phase 159 `/sessions/{id}` → `/app/` redirect. — Issue #112
- [ ] **FIX-02**: A shared session supports multiple simultaneous remote viewers without a newly joining viewer kicking an existing one, and the Hub provides a way to disconnect a stuck viewer. — Issue #117
- [ ] **FIX-03**: Opening a remote session from the Hub opens it in an in-app tab (connecting via the remote peer's host), not an external browser window. — Issue #118
- [ ] **FIX-04**: The Hub session-card viewer count reflects only real remote/shared viewers, excluding the app's own internal WebSocket subscribers (TerminalPanel, ChatPanel, status watcher) — a never-shared local session reads 0 viewers. — Issue #121
- [ ] **FIX-05**: Tailscale connection detection reports "Connected" on non-admin macOS accounts where the `macsys` `sameuserproof` file is unreadable, via a CLI `status` fallback when the SDK read fails. — Issue #120

## v2 Requirements

Deferred to a future milestone. Tracked but not in this roadmap.

### Future

- **FUT-01**: Automate device-share / ACL edits via the Tailscale admin API (requires the user to provision an OAuth client with `policy_file` scope — explicitly out of scope for v4.2 per Issue #107).

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Automating device-share / ACL edits via the Tailscale admin API | Requires an OAuth client with `policy_file` scope — a separate, heavier feature (Issue #107 "Out of scope"). v4.2 documents the manual path in Help (HLP-01/02) instead. |
| Click-to-focus the session from a notification | `gen2brain/beeep` is fire-and-forget with no click callback; not feasible without a heavier notification stack. NTF identifies the session by name instead. |
| Funnel exposure for sessions that are not web-shared | Funnel rides on the existing web-share perimeter (join code + cap token); a session must be web-shared first. |
| The larger Warp-style terminal UX (#111), Hub fidelity depth (#93), split panes (#49), intersession orchestration (#10) | Out of v4.2 scope; remain open enhancements for future milestones. |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FNL-01 | Phase 165 | Complete |
| FNL-02 | Phase 165 | Complete |
| FNL-03 | Phase 165 | Complete |
| FNL-04 | Phase 165 | Complete |
| FNL-05 | Phase 165 | Complete |
| FNL-06 | Phase 165 | Complete |
| FNL-07 | Phase 165 | Complete |
| FUI-01 | Phase 166 | Pending |
| FUI-02 | Phase 166 | Pending |
| FUI-03 | Phase 166 | Pending |
| FUI-04 | Phase 166 | Pending |
| FUI-05 | Phase 166 | Pending |
| FUI-06 | Phase 166 | Pending |
| HLP-01 | Phase 166 | Pending |
| HLP-02 | Phase 166 | Pending |
| NTF-01 | Phase 167 | Complete |
| NTF-02 | Phase 167 | Complete |
| NTF-03 | Phase 167 | Complete |
| NTF-04 | Phase 167 | Complete |
| UX-01 | Phase 168 | Pending |
| UX-02 | Phase 168 | Pending |
| FIX-01 | Phase 168 | Pending |
| FIX-02 | Phase 168 | Pending |
| FIX-03 | Phase 168 | Pending |
| FIX-04 | Phase 168 | Pending |
| FIX-05 | Phase 169 | Pending |

**Coverage:**

- v1 requirements: 26 total
- Mapped to phases: 26 (100% coverage)
- Unmapped: 0

---
*Requirements defined: 2026-06-30*
*Last updated: 2026-07-01 — added FIX-04 (#121 phantom viewer count) to Phase 168 + FIX-05 (#120 Tailscale detection) split into new Phase 169 (orthogonal subsystem, needs non-admin macOS test env); #120 labeled bug*
