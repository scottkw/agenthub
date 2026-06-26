---
phase: 152
slug: relay-protocol-identity-presence
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high) severity
threats_open: 0
asvs_level: 1
created: 2026-06-26
---

# Phase 152 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Identity, presence, and typing over the relay/web-share WebSocket surfaces.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| client→server (WS frame) | Untrusted alias / typing JSON payloads arrive over the relay or web-share WebSocket and are decoded server-side | Alias text (≤32 runes), typing booleans — low sensitivity, attacker-controlled |
| owner webview→relay (loopback) | The Wails owner connects over 127.0.0.1; trusted as `local` but its alias/typing payloads still pass validation | Alias/typing payloads — trusted origin, still validated |
| web-share peer→webserver (Tailscale) | A tailnet peer (possibly RO-cap) connects over TLS; `r.RemoteAddr` feeds `lc.WhoIs`; payloads untrusted | Peer node identity (WhoIs-attested) + untrusted alias/typing |
| in-process → filesystem | personKey (Tailscale-derived) is a JSON map key; the alias store writes `aliases.json` under the daemon config dir | Composite personKey + alias map (0600 file) |
| server→client (WS frame) | Untrusted JSON bodies in 0x32/0x33 frames are parsed in the browser | Presence/typing JSON — parsed defensively |
| Hub timer goroutine→subscribers | `time.AfterFunc` typing-TTL callbacks broadcast to all subscriber channels | In-memory typing state — never persisted |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-152-01 | Tampering | Alias ingress (ValidateAlias, all surfaces) | high | mitigate | `relay.ValidateAlias` rejects C0 (<U+0020) / C1 (U+007F–U+009F) control chars and >32-rune input; called at relay ingress (`relay/server.go:355`), web ingress (`webserver/server.go:1147`), and defense-in-depth at `alias_store.go:80` | closed |
| T-152-10 | Tampering | Alias rendering (XSS at render) | high | transfer | Render-time escaping owned by chat UI Phases 154/155; this phase only parses alias into a typed value. Ingress validation already strips control chars server-side (T-152-01) | closed |
| T-152-03 | Denial of Service | Typing storm | medium | mitigate | Per-personKey 500ms rate limit (`hub.go:367`) + non-blocking `BroadcastExcept`/`CloseSlow` drop-on-slow fan-out | closed |
| T-152-08 | Information Disclosure | Typing persistence | medium | mitigate | Typing state lives only in in-memory `typingRoster` (`hub.go:66`); no code path calls `AppendMessage`/`ChatStore` — never stored | closed |
| T-152-09 | Tampering | parseServerFrame JSON.parse | medium | mitigate | All frame-body `JSON.parse` wrapped in try/catch → `{type:'unknown'}` (`relayClient.ts:105–110`); nullish field fallbacks. Hostile body cannot break the message loop | closed |
| T-152-02 | Spoofing | Alias vs TailnetID | medium | accept | Alias is a non-unique display label (D-03); authoritative identity is the WhoIs-derived TailnetID. Non-uniqueness is an accepted design property | closed |
| T-152-04 | Spoofing/Info | WhoIs failure fallback | low | mitigate | On WhoIs error, `tailnetID` stays `"unknown"` (non-`local`) so a web client is never silently merged into the owner's `local:local` entry (`webserver/server.go:1046–1058`); criterion 5 holds without a live tailnet | closed |
| T-152-06 | Tampering | aliases.json path | low | mitigate | `filepath.Join(configDir, "aliases.json")` — hardcoded basename; personKey is only ever a JSON map key, never a path component. No path traversal | closed |
| T-152-07 | Denial of Service | Post-shutdown timer callback | low | mitigate | Timer callback checks `h.closed` under mutex before broadcasting (`hub.go:377`); `Unsubscribe` stops+deletes the timer when the last connection drops — no leaked timers/panics | closed |
| T-152-05 | Elevation of Privilege | RO-cap chat vs PTY | low | accept | D-06: MsgAliasSet/MsgTyping are deliberately NOT ReadOnly-gated (benign chat); only MsgInput stays gated on `sub.ReadOnly`/`claims.Perms=="read"`. Tests assert RO clients chat but PTY input is still discarded | closed |
| T-152-11 | Spoofing | Owner `local` identity | low | accept | The loopback path is reachable only on 127.0.0.1; stamping a fixed `local` sentinel is correct and never passed through WhoIs | closed |
| T-152-SC | Tampering (supply chain) | go/npm module installs | low | accept | No packages added across all six plans — stdlib + existing `tailscale.com/client/local` (already in go.mod) only. No legitimacy gate required | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-152-01 | T-152-02 | Aliases are intentionally non-unique display labels; authoritative identity is the WhoIs TailnetID (D-03) | scottkw | 2026-06-26 |
| AR-152-02 | T-152-05 | RO-cap clients are full chat participants by design; only PTY input is privilege-gated (D-06) | scottkw | 2026-06-26 |
| AR-152-03 | T-152-11 | Loopback owner identity is a fixed `local` sentinel; reachable only on 127.0.0.1 | scottkw | 2026-06-26 |
| AR-152-04 | T-152-SC | No new third-party packages introduced this phase | scottkw | 2026-06-26 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-26 | 12 | 12 | 0 | gsd-secure-phase (L1, grep-depth) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-26
