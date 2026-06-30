# Phase 166: Funnel Frontend + Help Guide - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-30
**Phase:** 166-funnel-frontend-help-guide
**Areas discussed:** Risk dialog flow, Auto-expiry selector, Exposure indicator, Funnel URL + warm-up

---

## Risk dialog flow

### Presentation
| Option | Description | Selected |
|--------|-------------|----------|
| Inline panel in modal | Risk statement, expiry selector, confirm expand inline within the Share modal; no second modal stacked | ✓ |
| Nested dialog | Separate smaller risk dialog over the Share modal with backdrop | |
| Replace modal content | Share modal swaps body to risk ack, swaps back after enable | |

### Acknowledgment gesture
| Option | Description | Selected |
|--------|-------------|----------|
| Explicit confirm button | "Enable internet share" button (distinct from Cancel) is the ack | ✓ |
| Checkbox + button | "I understand…" checkbox un-gates Enable | |
| Type-to-confirm | Type a word to enable | |

**User's choice:** Inline panel + explicit confirm button.
**Notes:** Shown every enable, no "don't show again" (FUI-01). Help cross-link included in panel (FUI-06).

---

## Auto-expiry selector

### Presets + default
| Option | Description | Selected |
|--------|-------------|----------|
| 30m / 1h / 4h / 8h — default 1h | Quick pairing up to a full workday; 1h default | ✓ |
| 15m / 1h / 8h — default 1h | Coarser three presets | |
| 30m / 1h / 2h / 4h — default 30m | Shorter ceiling, 30m default | |

### Custom / no-expiry
| Option | Description | Selected |
|--------|-------------|----------|
| Presets only | No custom, no never-expire (safest) | |
| Presets + custom minutes | Numeric-minutes input added | |
| Presets + "until I disable" | No-expiry option added | ✓ |

**User's choice:** 30m / 1h / 4h / 8h (default 1h) **plus** an "until I disable" no-expiry option.
**Notes:** Claude flagged that no-expiry relaxes the auto-expiry safety rail; user accepted — the persistent indicator + one-click disable are the compensating controls. Research item raised: confirm backend `expirySeconds <= 0` sentinel for "no auto-expiry".

---

## Exposure indicator

### Label + icon
| Option | Description | Selected |
|--------|-------------|----------|
| Globe + "INTERNET" | Globe icon + uppercase "INTERNET" | ✓ (Hub card) |
| Globe + "INTERNET ACTIVE" | Fuller label from success criteria | |
| Globe + "PUBLIC" | Most compact single word | |

### Placement
| Option | Description | Selected |
|--------|-------------|----------|
| Badge on card + tab | Pill on Hub card header AND session tab | ✓ (with tab refinement below) |
| Badge + full-width banner | Badges plus a warning strip | |
| Tab badge only | Only on the session tab | |

**User's choice:** Globe + "INTERNET", on both surfaces — **refined**: Hub card shows globe + "INTERNET" (full badge); session **tab shows the globe icon only** (no visible text) to save tab space.
**Notes:** Claude added a compliance/accessibility note — the tab globe carries an `aria-label`/tooltip "Internet exposed" so FUI-03's text-label requirement and screen-reader access are still met though no visible text on the tab. Globe is shape-based → colorblind-safe (user is colorblind; verify at hex/source).

---

## Funnel URL + warm-up

### URL model
| Option | Description | Selected |
|--------|-------------|----------|
| Single public Funnel URL | One URL in its own "Internet (public)" section, read-only cap default | ✓ |
| Read + write Funnel URLs | Mirror dual links over Funnel (public write) | |
| Reuse existing link rows | Swap host to Funnel hostname on existing rows | |

### Warm-up UX
| Option | Description | Selected |
|--------|-------------|----------|
| Poll until reachable | "Starting up…" with muted URL, poll funnelActive/URL until it answers, then reveal | ✓ |
| Timed placeholder | Fixed delay then reveal regardless | |
| Show URL immediately + note | Reveal at once with a "may take a few seconds" note | |

**User's choice:** Single public Funnel URL (read-only cap default) + poll-until-reachable warm-up.
**Notes:** Separate "Internet (public)" section keeps the tailnet-vs-internet distinction clear, reinforcing the indicator.

---

## Claude's Discretion

- Pill/badge styling, inline-panel animation, poll interval/timeout for warm-up.
- Help-article prose, heading structure, specific Tailscale doc URLs to cite.

## Recorded Defaults (not separately discussed)

- Local-fallback (M-36): Funnel toggle disabled with inline note when Tailscale not running (`webServerMode !== 'tailscale'`).
- One-click disable: `SetSessionFunnel(id, false, 0)` clears indicator immediately.
- Help guide: new top-level "Sharing Outside Your Tailnet" section after Chat / before FAQ; covers Funnel + device-share/ACL alternative, copy-pasteable `autogroup:shared`→`tcp:7443` grant, `*→*` wildcard gotcha, Tailscale doc links.

## Deferred Ideas

- Public write access over Funnel (read+write dual Funnel URLs).
- Custom-minutes auto-expiry input.
- Automating device-share / ACL edits via Tailscale admin API (FUT-01 / Issue #107 — out of v4.2 scope).
