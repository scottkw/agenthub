# Project Research Summary

**Project:** AgentHub v1.11
**Domain:** Go/Wails desktop app — dual-mode networking, auto-serve, UI polish
**Researched:** 2026-04-08
**Confidence:** HIGH

## Executive Summary

AgentHub v1.11 is a tightly scoped milestone that adds local network fallback for users without Tailscale, auto-serve session management, a Settings-as-tab UI migration, and a native Claude Code path detection fix. All four research areas converge on the same conclusion: this is an additive milestone built on a well-understood existing architecture, and every feature can be implemented using the Go standard library and React patterns already present in the codebase. No new external dependencies are required.

The recommended approach is to implement in five sequential phases ordered by risk and dependency. The two lowest-risk, highest-isolation changes (Claude Code path detection and sidebar label rename) ship first. Settings conversion to a sidebar tab comes next because the local network password UI needs to live in the converted Settings tab. Auto-serve follows, then the local network fallback (the most complex feature) completes the milestone. This ordering ensures each phase delivers independently testable value and that the most complex work builds on stable foundations.

The primary risk is the local network fallback feature, which re-introduces infrastructure deleted in v1.2 (self-signed TLS, password auth middleware). Four specific failure modes are well-documented in PITFALLS.md: using P521 instead of P256 for the TLS curve (silent Chrome failure, not a cert warning), embedding the password in the URL (credential exposure), tangling dual-mode code paths inside a single `Start()` method (wrong URL returned in one mode), and placing the nudge banner inside the terminal flex container (terminal height regression). All four are avoidable with explicit design decisions made before implementation begins.

## Key Findings

### Recommended Stack

The existing Go stdlib is sufficient for all new v1.11 capabilities. The `crypto/x509`, `crypto/ecdsa`, `crypto/rand`, `crypto/tls`, and `net` packages are already imported in the codebase and cover self-signed cert generation, password generation, and LAN IP resolution. The existing `webserver.Config` struct has a `TLSConfig *tls.Config` override field already wired in tests — local network mode injects a generated `*tls.Config` through this same seam with no structural changes to `WebServer`.

**Core technologies — what is NEW for v1.11:**
- `crypto/x509` + `crypto/ecdsa` (P256): self-signed CA+leaf cert generation — already in codebase, no new dep; pattern confirmed from existing `app_test.go` and `server_test.go`
- `crypto/rand`: 18-byte URL-safe base64 password (~143 bits entropy) — already in codebase
- `net.InterfaceAddrs()`: LAN IP resolution for local server bind — stdlib, no new dep
- `net/http` middleware: Bearer auth for local mode — ~10 lines, no external auth library
- New files: `internal/webserver/localnet.go` (cert + IP helpers), `internal/webserver/auth.go` (password middleware)

**What not to add:**
- External self-signed cert library — Go stdlib `crypto/x509` covers it completely
- `math/rand` or `crypto/md5` for passwords — not cryptographically secure; use `crypto/rand`
- External auth library (gorilla/sessions etc.) — 10,000+ LOC for a single Bearer check
- `tsnet` for local binding — creates a second Tailscale node; wrong tool

### Expected Features

**Must have for v1.11 (table stakes — all P1):**
- Local network fallback with self-signed TLS + single random password — users without Tailscale expect LAN web access; Tailscale is optional for many
- Nudge banner when running in local mode — user must know they are in degraded security mode and how to upgrade
- Auto-start web server on daemon start (configurable, default off) — manual start each launch is friction
- Auto-enable web serving for new sessions (configurable, default off) — per-session toggle on every session is friction
- Settings as sidebar tab — modal interrupts workflow; tab matches Home/Remote/Sessions navigation pattern
- "New Session" label rename — "New Tab" is ambiguous; misleads users expecting browser behavior
- Claude Code native install path detection — `~/.local/bin/claude` missing from `AugmentServicePath`; causes "not detected" when app launched from Finder/launchd

**Should have (competitive differentiators):**
- Dual-mode web serving: mode auto-selected at server start based on Tailscale health; user sees active mode explicitly in Settings tab
- Zero-config auto-serve: server starts itself and new sessions are served with no user action (auto-start + auto-enable both on)
- Persistent non-blocking nudge: informs without blocking; user can work while Tailscale setup happens

**Defer to v1.11.x or v2+:**
- Per-session auto-serve override in new-session modal
- Runtime Tailscale-to-local fallback (graceful mode switch while browser sessions are active — race conditions with active clients)
- mDNS/Bonjour advertisement of local server for LAN device discovery
- Certificate pinning UI for browsers connecting in local mode

### Architecture Approach

The existing architecture (daemon-owned `WebServer` struct, Wails bindings in `app.go`, Unix socket IPC, React frontend) handles all five v1.11 features without structural changes. Each feature has a well-defined integration seam: `AugmentServicePath()` for path detection, two strings in `Sidebar.tsx` for the rename, `App.tsx` singleton tab pattern (mirroring `DaemonManagerPanel`) for Settings conversion, `app.go CreateSession` for auto-serve, and `webserver.Config.Mode` for dual-mode networking. The build order is driven by one hard dependency: the Settings tab must be converted before the local network password UI lands inside it.

**Major components and v1.11 changes:**
1. `internal/daemon/path.go` — add `~/.local/bin` (macOS/Linux) and `%USERPROFILE%\.local\bin` (Windows) to `AugmentServicePath()` candidates
2. `frontend/src/components/Sidebar.tsx` — two string changes: "New Tab" → "New Session"
3. `App.tsx` + `SettingsPanel.tsx` + `TabBar.tsx` — convert Settings from modal overlay to singleton tab; remove `isOpen`/`onClose`; wire via `handleOpenSettings` (find-or-add pattern); update all `setShowSettings(true)` call sites
4. `app.go` (`StartWebServer` + `CreateSession`) — auto-serve logic; new `session:web-enabled` Wails event; daemon-side decision (not frontend) to avoid stale React state race
5. `internal/webserver/localnet.go` (new) + `auth.go` (new) — self-signed P256 CA+leaf cert; password Bearer middleware; `webserver.Config.Mode` field; `startTailscale()`/`startLocal()` dispatch in `Start()`

### Critical Pitfalls

1. **P521 TLS curve rejected by Chrome** — Go's default `generate_cert.go` uses P521; Chrome returns `tls: illegal parameter` (not a cert warning). Always use `elliptic.P256()`. Test in Chrome, not curl. Warning sign: curl succeeds but Chrome fails.

2. **Dual-mode code paths tangled in `Start()`** — add explicit `Mode` field to `webserver.Config` and implement two private dispatch methods (`startTailscale`, `startLocal`). Make `BaseURL()` derived from actual listener address and active mode, not the config struct. Warning sign: `BaseURL()` returns `.ts.net` FQDN when server is in local mode.

3. **Password embedded in URL** — never put the password in a query param or session URL (browser history, logs, QR images). Use HTTP Basic Auth header only. Password belongs to daemon lifetime (generated once in `runDaemonCore()`), not per web server start/stop cycle. Expose via `GetLocalPassword()` daemon API endpoint for the GUI.

4. **Settings modal + tab conflict** — there are at least two `setShowSettings(true)` call sites in `App.tsx` (sidebar `onSettings` and `handleAddTab`'s "no CLIs found" path). Both must be migrated before the modal rendering is removed from JSX. Warning sign: Settings opens as overlay AND tab simultaneously.

5. **Nudge banner inside terminal flex container** — the terminal flex chain is fragile (double-rAF retry loop tech debt). Render nudge banner as sibling to `app__content`, never inside `terminal-wrapper`. Store dismiss state in `localStorage` (like sidebar collapsed state). Warning sign: terminal height shrinks after banner appears.

6. **Auto-serve state diverges from daemon** — `webEnabled` React state diverges from daemon authoritative state across window hide/show cycles. `ListSessions` response must include per-session `webEnabled` field; React state must be seeded from daemon on `init()`. Warning sign: StatusBar shows web toggle OFF for a session the daemon is actively serving.

## Implications for Roadmap

Based on the dependency graph from ARCHITECTURE.md and the pitfall-to-phase mapping from PITFALLS.md:

### Phase A: Claude Code Native Path Detection
**Rationale:** Highest isolation, lowest risk, no frontend changes, fully independent. Fixes a user-visible onboarding bug (Claude Code not detected when installed via native installer). Can ship and be validated on its own.
**Delivers:** `~/.local/bin/claude` detected when app launched from Finder/Dock/launchd service
**Addresses:** Claude Code native install path detection (table stakes)
**Avoids:** Pitfall 6 (native installer not in augmented PATH)
**Files:** `internal/daemon/path.go`, `internal/pty/detect.go`, tests
**Risk:** VERY LOW

### Phase B: Sidebar Label Rename
**Rationale:** Two string changes, zero behavior change, no backend changes. Trivial to validate. Completes sidebar consistency before Settings conversion.
**Delivers:** "New Session" label in sidebar (collapsed: icon only; expanded: "New Session")
**Addresses:** "New Session" label rename (table stakes)
**Avoids:** No specific pitfall; purely cosmetic
**Files:** `frontend/src/components/Sidebar.tsx`, 1 test file if label text is asserted
**Risk:** VERY LOW

### Phase C: Settings as Sidebar Tab
**Rationale:** Must precede local network fallback (Phase E) because the password display UI lives in SettingsPanel. Also unblocks auto-serve settings UI (Phase D). Follows the exact DaemonManagerPanel singleton tab pattern already in this codebase.
**Delivers:** Settings accessible as a persistent sidebar tab; modal overlay removed; `isOpen`/`onClose` props eliminated
**Addresses:** Settings as sidebar tab (table stakes)
**Avoids:** Pitfall 5 (modal + tab conflict; `isOpen` lifecycle mismatch; stale `setShowSettings` call sites)
**Files:** `App.tsx`, `SettingsPanel.tsx`, `TabBar.tsx` (type union), `Sidebar.tsx` (handler wiring)
**Risk:** LOW — pure frontend refactor following established codebase pattern; all call sites identified in PITFALLS.md

### Phase D: Auto-Serve Sessions
**Rationale:** Depends on Settings tab (Phase C) being available to show the auto-serve toggle. Logic change only — no new packages, no new middleware. Establishes the `session:web-enabled` Wails event that Phase E also emits.
**Delivers:** New sessions automatically web-enabled when server is running; `session:web-enabled` Wails event; `ListSessions` response extended with per-session `webEnabled` field
**Addresses:** Auto-enable web serving for new sessions, auto-start web server (table stakes)
**Avoids:** Pitfall 4 (auto-serve state not seeded from daemon; frontend-side auto-serve race)
**Files:** `app.go` (StartWebServer + CreateSession), `App.tsx` (event handler), daemon settings types
**Risk:** MEDIUM — changes session creation behavior; must not break no-server case; daemon must be authoritative

### Phase E: Local Network Fallback
**Rationale:** Most complex feature; re-introduces infrastructure removed in v1.2. Depends on Phase C (Settings tab for password display) and Phase D (`session:web-enabled` event established). The dual-mode `Config.Mode` architecture decision must be locked in before cert/auth/UI layers are built.
**Delivers:** Self-signed P256 CA+leaf TLS; password Bearer auth for LAN access; nudge banner; `webserver:started {url, mode, password}` Wails event; full dual-mode web serving
**Addresses:** Local network fallback, nudge banner, password protection (all table stakes and differentiators)
**Avoids:** Pitfall 1 (dual-mode Config ambiguity), Pitfall 2 (P521 curve), Pitfall 3 (password in URL / wrong lifetime), Pitfall 7 (nudge banner flex layout)
**Files:** `internal/webserver/localnet.go` (new), `internal/webserver/auth.go` (new), `server.go`, `types.go`, `api.go`, `app.go`, `SettingsPanel.tsx`, `App.tsx`
**Risk:** HIGH — new middleware, re-introduced infrastructure, most file surface area; use PITFALLS.md "Looks Done But Isn't" checklist as acceptance criteria

### Phase Ordering Rationale

- Phases A and B are fully independent; A first because it fixes a user-visible bug.
- Phase C must precede Phase E because SettingsPanel is the UI surface for password display and mode indicator.
- Phase D should precede Phase E to establish the `session:web-enabled` event pattern before local mode also emits it.
- Phase E is last — most complex, depends on C and D being stable.
- Each phase delivers independently testable, releasable value; no phase is blocked on another to be demo-able.
- Critical path: C → E. Phases A, B, D are on the fast track.

### Research Flags

Phases with well-documented patterns (no additional research needed):
- **Phase A:** One-line addition to existing `AugmentServicePath`; path confirmed by official Anthropic docs and verified on this machine at `/Users/ken/.local/bin/claude`
- **Phase B:** Two string changes; no research needed
- **Phase C:** Mirrors `DaemonManagerPanel` singleton tab pattern exactly; all integration points identified in ARCHITECTURE.md and PITFALLS.md
- **Phase D:** Logic-only change in `app.go`; all call sites identified; `session:web-enabled` event modeled on existing `session:status`

Phases that may benefit from targeted implementation-time validation:
- **Phase E:** Before generating the self-signed cert, verify in Chrome (not curl) that P256 produces the expected "not trusted" warning and not a cryptic TLS error. Also confirm LAN IP selection behavior on machines with multiple LAN interfaces (VPN + Wi-Fi). The PITFALLS.md "Looks Done But Isn't" checklist (14 items) should be used as Phase E acceptance criteria verbatim.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All new capabilities use Go stdlib already imported. Cert generation pattern confirmed from existing `app_test.go` and `server_test.go`. No new deps required. |
| Features | HIGH | Defined by direct codebase audit + official Claude Code docs + user pain points. All features are P1; no deferred scope within v1.11. |
| Architecture | HIGH | Based on direct code inspection of all affected files. Integration seams identified precisely; component dependency map in ARCHITECTURE.md covers all 5 features. |
| Pitfalls | HIGH | 7 pitfalls with concrete warning signs, recovery strategies, and phase-to-pitfall mapping. Sources: official Go issue tracker (P521), official Anthropic docs (native path), live codebase (flex chain, modal call sites). |

**Overall confidence:** HIGH

### Gaps to Address

- **LAN IP selection heuristic:** The plan is `net.InterfaceAddrs()` scanning for first non-loopback IPv4 preferring 192.168.x or 10.x range. Machines with multiple LAN interfaces (VPN + Wi-Fi) may return an unexpected IP. During Phase E implementation, define the preference order explicitly and document it in code (exclude known VPN ranges? prefer Wi-Fi interface names?).
- **Claude Code native installer path on Windows:** `%USERPROFILE%\.local\bin\claude.exe` is the expected path per official docs. The code structure allows easy addition of more paths, but the Windows native installer behavior has not been verified against an actual Windows install; confirm before shipping Phase A on Windows.
- **Settings tab state on re-focus:** After converting SettingsPanel, web server state should reload when the tab is re-activated, not just on initial mount. The `useEffect` trigger must change from `[isOpen]` to `[activeId === SETTINGS_TAB.id]`. Existing `useEffect` dependencies in `SettingsPanel.tsx` need a careful audit during Phase C — the `isOpen` guard removal is well-documented in PITFALLS.md but requires verifying every async callback path.

## Sources

### Primary (HIGH confidence)
- Official Anthropic Claude Code docs — [code.claude.com/docs/en/setup](https://code.claude.com/docs/en/setup) — native install path `~/.local/bin/claude` confirmed by uninstall instructions; verified on this machine at `/Users/ken/.local/bin/claude`
- Go `crypto/tls` P521 curve Chrome rejection — [github.com/golang/go/issues/19901](https://github.com/golang/go/issues/19901) — confirmed P521 not supported in Chrome/Firefox; causes `tls: illegal parameter`
- Go `crypto/tls/generate_cert.go` — CA+leaf vs. single self-signed cert scheme — [go.dev/src/crypto/tls/generate_cert.go](https://go.dev/src/crypto/tls/generate_cert.go)
- AgentHub codebase (direct inspection) — `internal/webserver/server.go`, `internal/webserver/tailscale.go`, `internal/daemon/api.go`, `internal/daemon/engine.go`, `internal/daemon/types.go`, `internal/daemon/path.go`, `internal/pty/detect.go`, `app.go`, `app_test.go`, `internal/webserver/server_test.go`, `frontend/src/App.tsx`, `frontend/src/components/SettingsPanel.tsx`, `frontend/src/components/Sidebar.tsx`
- `/Users/ken/dev/agenthub/.planning/PROJECT.md` — v1.2 decision log confirming self-signed cert removal; v1.1 flex chain fix; v1.6 rAF retry loop tech debt

### Secondary (MEDIUM confidence)
- [Claude Code GitHub issue #10970](https://github.com/anthropics/claude-code/issues/10970) — `~/.local/bin` PATH issue for Homebrew users; confirms native vs Homebrew path distinction
- [OpenCode GitHub issue #11997](https://github.com/anomalyco/opencode/issues/11997) — auto-start web server proposal with implementation notes (different product, same domain)
- UX research: settings-as-tab pattern validated by VS Code and Warp terminal behavior; modal-vs-tab UX analysis

---
*Research completed: 2026-04-08*
*Ready for roadmap: yes*
