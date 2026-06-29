# Plugin Architecture Assessment — AgentHub

> **Status:** Exploratory assessment (read-only investigation)
> **Date:** 2026-06-27
> **Scope:** Feasibility of adding a third-party plugin/extension architecture so external developers can extend AgentHub's capabilities.
> **Author:** Engineering assessment (Claude Code)

---

## Bottom line

A plugin architecture is **feasible and architecturally well-suited at the backend/daemon layer**, **awkward but possible at the CLI layer**, and **expensive at the frontend/UI layer**.

The single biggest gating factor is **not** the architecture — it is **security**. The daemon runs with full user privileges, owns every PTY, holds the capability signing key, and exposes sessions over the network. There is **no process isolation** today: untrusted in-process code would run as the user with no boundary. Letting third-party code in is a force-multiplier on a privileged surface, so the design must lean on out-of-process execution and the existing capability system rather than introduce in-process loading.

The good news on the security front: AgentHub already has a **working, tested authorization primitive** — the HMAC-signed capability system (`internal/capability/`), shipped in v3.1.0. This is exactly the foundation to scope plugin tokens against.

**Recommendation:** start with an **out-of-process, API-driven plugin model** (the lowest-risk, highest-leverage tier) and explicitly defer in-process and UI plugins. See [Recommended path](#recommended-path-tiered-ship-incrementally).

This is naturally a **post-v4.1 effort** — the active milestone is mid-flight.

---

## 1. A naming collision to resolve first

"Plugin" already exists in this codebase and means something narrow: **xterm.js addon feature-flags** (WebGL, search, web-links, image, clipboard, serialize, progress, unicode11). It is a toggle system, not an extension system:

- `PluginSettings` struct — `internal/daemon/plugin_settings.go`, persisted to `settings.json` with a defaults-merge.
- Config streamed to web clients over SSE — `internal/webserver/plugin_config_stream.go`, `plugin_config.go`.
- Surfaced in the UI — `frontend/src/components/PluginsSection.tsx`.
- All addons are **first-party and pre-bundled**. **There is no dynamic code loading anywhere.**

A third-party extension system is a *different concept*. To avoid colliding with the shipped feature, use a different name — **Extensions** or **Integrations**. Note that Phase 162 on the active roadmap is already renaming the existing feature to **"Terminal Plugins,"** which frees up the bare word "plugin."

**Reusable scaffolding:** the existing config plane (settings persistence with defaults-merge, the listener→SSE broadcast pattern, capability-gated config endpoints) can be reused for an extension system's *configuration* surface. It solves none of the hard problems — loading, lifecycle, isolation, discovery.

---

## 2. Architecture realities that shape the design

**The shape of the app:** a single Go binary that is simultaneously the GUI (Wails), the CLI, and the daemon.

- A long-running **daemon** owns all state and exposes an HTTP/JSON API over a Unix socket (`internal/daemon/api.go`). On Windows this is a named pipe.
- The GUI and CLI are both just **clients of that socket** (`internal/daemon/client.go`). The CLI (`cmd_cli.go`) is effectively a thin API wrapper.
- The React frontend is **statically `//go:embed`-ed at compile time** (`assets_prod.go`). One bundle, no code-splitting, no `React.lazy`, tab types hardcoded in an `App.tsx` switch.

This shape determines what is cheap and what is expensive:

| Extension layer | Fit | Why |
|---|---|---|
| **Daemon API consumers** (out-of-process) | ✅ Natural | The socket API is already the integration boundary; the CLI is itself just an API client. A plugin is "another client." |
| **Typed backend plugins** (out-of-process subprocess + RPC) | ✅ Good | Real seams exist: `SessionBackend` interface (`internal/pty`), `status.Detector` (`internal/status`), `KeyStore` (`internal/capability`), the `onStatus`/`onExit` callbacks, the plugin-settings listener. These are the join points a plugin host would drive. |
| **Agent-type definitions** (declarative) | ✅ Natural | Sessions already spawn arbitrary CLIs via configurable `cliPaths`. A manifest declaring `{command, args, icon, status-detection patterns}` maps almost 1:1 onto existing concepts — likely the single highest-value, lowest-effort extension point. |
| **In-process Go plugins** | ⚠️ Avoid | Go's native `plugin` package is platform-broken (no Windows, fragile version matching) — a non-starter for a cross-platform signed app. Would require `hashicorp/go-plugin` (gRPC-over-subprocess), which is really the typed-backend tier again. |
| **Frontend/UI plugins** | ❌ Expensive | Static embed + single bundle + hardcoded tab switch means UI extensibility requires an app-shell refactor, runtime module loading, and a contribution API. Large, and it breaks the "compiled, signed, tamper-evident binary" property. |

### Existing extension seams (already in the code)

These are not a plugin system, but they are the natural attach points one would build on:

- **Interfaces:** `SessionBackend` (PTY backend swapping), `KeyStore` (signing-key persistence), `HubLike` (status detection).
- **Callbacks at session creation:** `onStatus(sessionID, status)` and `onExit(sessionID, exitCode)` in `internal/daemon/engine.go`.
- **Listener pattern:** `SetPluginSettingsListener` → SSE broadcast (the daemon→webserver notification path).
- **Dependency injection at the webserver:** `SetSessionResolver`, `SetFilesHandler`, `SetPluginSettingsProvider`, `SetAliasProviders`, `SetStaticAppFS`, `SetSigningKey`, `SetJoinCodes`.
- **Declarative agent registration:** configurable `cliPaths` already let the app spawn arbitrary agent CLIs.

### What fights a plugin system today

- Frontend bundle is baked into the binary at compile time — no runtime asset loading.
- Tab/surface types are a hardcoded switch in `App.tsx` — new UI surfaces require source edits.
- No component registry, no `usePlugin()` hook, no dynamic `import()`.
- Daemon routes are hardcoded in `api.go` — no dynamic route registration.
- Settings persistence is direct JSON with no plugin schema/versioning story.

---

## 3. Security is the real decision, not the plumbing

Two properties of the current design make this higher-stakes than a typical app's plugin system. These are **inherent posture**, not open vulnerabilities — the network-facing authz issues from the original security review were closed in v3.1.0 (see note below).

1. **The daemon is the crown jewels.** It holds the HMAC capability signing key (`internal/capability`). Any in-process plugin — or any plugin that can read daemon memory/config — could **forge capability tokens for every session**. That is total compromise. This is the central argument for keeping plugins out-of-process.
2. **No process isolation exists today.** PTYs spawn as the daemon user with inherited environment (`internal/pty/native.go`). The only sandbox (`internal/files/sandbox.go`) is *path-based file confinement*, not process containment. There is no existing mechanism to run code with reduced privileges.

**Practical implication:** in-process untrusted code = arbitrary code execution as the user, with no boundary. Do not start there.

The only posture worth shipping is **out-of-process plugins** with:

- a restricted environment (sandboxed `HOME`, minimal `PATH`, stripped env, no shell init files);
- a **capability-scoped API** — plugins never touch the signing key, only receive narrowly-scoped tokens issued by the existing capability system;
- **signed and version-pinned** plugins with explicit per-plugin user approval;
- audit logging and rate-limiting on network/relay access.

### The existing capability system is the foundation, not a gap

A 2026-04-19 third-party security review (`.security-review/SECURITY_REVIEW.md`) originally found that web-served sessions had **no application-layer authorization** — tailnet membership was the only boundary, "read-only" was a bypassable query param, and the WebSocket accepted any origin. **All of these were remediated and shipped in v3.1.0** (GitHub issue **#35**, Phases **87–90**):

- **Phase 87** — server-issued, session-bound, HMAC-signed capability tokens now gate `/api/sessions` and `/sessions/{id}/ws`; enumeration collapsed to self-only; auto-expose removed.
- **Phase 88** — strict `requireAllowedOrigin` middleware (`internal/webserver/origin_mw.go`), with a `security_regression_test.go` guard against re-introducing `OriginPatterns: ["*"]`.
- **Phase 89** — xterm assets vendored into the binary (`web/vendor/xterm/`) under a strict CSP; no runtime CDN fetches.
- **Phase 90** — release pipeline SHA-pinned, split build/sign/publish stages with a required-reviewer signing gate, SLSA L2 attestations.

> ⚠️ The `.security-review/` documents are a **pre-remediation April 2026 snapshot** and describe the findings as open. They no longer reflect current code — verify against source before citing.

The practical upshot for plugins: read-only enforcement is now a **server-bound capability property** (`readonly := !capability.HasPerm(claims.Perms, "write")`), and permission scoping is a solved, tested problem. A plugin system extends this model (e.g. a new `plugin.execute` perm bit) rather than inventing authorization from scratch.

### Sandboxing requirements (if untrusted code is in scope)

| Concern | Current state | Required for plugins |
|---|---|---|
| Authentication/authz | Working: session-bound HMAC capability tokens + `requireCapability` middleware (v3.1.0) | New `plugin.execute` perm bit; scoped tokens; never expose signing key |
| Process isolation | None — runs as daemon user | Separate process, restricted env; cgroups/seccomp/namespaces on Linux where possible |
| Network access | No rate limits/logging on relay/webserver | Restrict or gate network; log + rate-limit plugin traffic |
| Filesystem | Path sandbox per session, denylist (`~/.ssh`, rc files, `~/.config/agenthub`) | Plugins inherit the same denylist, sandboxed per-plugin identity |
| Relay/WebSocket | Capability-gated; read/write is a server-bound cap property (not client-asserted) | Plugins route through capability-checked daemon API, not direct `hub.Subscribe()` |
| Chat identity | Sanitized, but no per-plugin identity | Tag plugin messages distinctly from human chat |

---

## 4. Recommended path (tiered, ship-incrementally)

### Tier 0 — Declarative agent definitions *(weeks, not months)*

A manifest format letting users register a new agent type: command + args, display name/icon, and `status.Detector` patterns. No code execution beyond what AgentHub already does (spawning a CLI). Delivers ~70% of the practical "extend AgentHub" value with near-zero new attack surface. **Start here.**

### Tier 1 — Out-of-process API plugins / webhooks

Formalize and version the daemon socket API as a public contract. Add an event-subscription/webhook mechanism (the `onStatus`/`onExit`/listener hooks are the foundation) so external programs react to session events and drive the API. Plugins are separate processes in any language. New capability perm bit (`plugin.execute`), scoped tokens, no access to the signing key. Moderate effort, contained risk.

### Tier 2 — Typed host-managed plugins (`hashicorp/go-plugin`)

For deeper hooks (custom `SessionBackend`, custom status detectors), run plugins as gRPC subprocesses with restricted `HOME`/`PATH`/env, signed and version-pinned in daemon config with explicit per-plugin user approval. Larger effort; do only after Tier 1 proves demand. This tier is where process-isolation work (the one remaining structural gap) must land before untrusted code is allowed in.

### Tier 3 — UI extensions

Defer. Requires a frontend app-shell refactor + runtime loading, and it breaks the signed-bundle integrity model. Only justify if Tiers 0–2 reveal real demand for custom UI surfaces.

---

## 5. Open questions to confirm before planning

- **Who are the plugin authors?** Just the maintainer / power-users writing config (→ Tier 0 is plenty), or untrusted third parties distributing code (→ the full security apparatus becomes mandatory)? This single answer changes the scope by an order of magnitude.
- **Does anything need to render UI**, or is "extend capabilities" satisfied by backend behavior + new agent types?
- **How much isolation is required?** The network-facing authz findings are already closed (v3.1.0). The remaining structural gap is **process isolation** — required before Tier 2 admits untrusted third-party code, but not blocking for Tier 0/Tier 1.

---

## 6. Suggested next step

Per project convention, release scope decisions live in GitHub issues (`scottkw/agenthub`), not chat. The natural next step is to file an issue capturing the Tier 0 / Tier 1 path as a candidate post-v4.1 milestone, with the two open questions above as the gating decisions.

---

## Appendix — Key file references

| Area | Path |
|---|---|
| Daemon API routes | `internal/daemon/api.go` |
| Daemon client (socket transport) | `internal/daemon/client.go` |
| Session engine / lifecycle / callbacks | `internal/daemon/engine.go` |
| Existing "plugin" (xterm addon) settings | `internal/daemon/plugin_settings.go` |
| Plugin config serving + SSE stream | `internal/webserver/plugin_config.go`, `plugin_config_stream.go` |
| Capability tokens (signing/verify) | `internal/capability/capability.go` |
| Join codes (one-time exchange) | `internal/capability/joincode.go` |
| Capability enforcement middleware | `internal/webserver/capability_mw.go` |
| Webserver dependency injection | `internal/webserver/server.go` |
| PTY spawning | `internal/pty/native.go`, `internal/pty/backend.go` |
| File sandbox + denylist | `internal/files/sandbox.go` |
| Status detection | `internal/status/detector.go` |
| CLI commands | `cmd_cli.go`, `main.go` |
| Frontend embed | `assets_prod.go` |
| Frontend root / tab routing | `frontend/src/App.tsx` |
| Existing "plugin" UI | `frontend/src/components/PluginsSection.tsx` |
| Addon consumption / hot-swap | `frontend/src/components/TerminalPanel.tsx` |
| Origin allowlist middleware | `internal/webserver/origin_mw.go` |
| Security review (pre-remediation snapshot, Apr 2026 — findings since fixed in v3.1.0 / issue #35 / Phases 87–90) | `.security-review/SECURITY_REVIEW.md` |
