# Phase 147: In-App Help Page - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-22
**Phase:** 147-in-app-help-page
**Areas discussed:** Navigation placement, Cross-surface scope, Content source & format, Search depth

---

## Navigation placement

| Option | Description | Selected |
|--------|-------------|----------|
| 4th sidebar item | Home / Hub / Settings / Help; reopens Phase-138 3-item sidebar; most discoverable | ✓ |
| Header/chrome ? button | '?' icon opens Help tab, leaves Phase-138 sidebar untouched; less discoverable | |
| Welcome-tab link only | Reachable only from a link on Welcome/Home; lowest footprint, weakest discoverability | |

**User's choice:** 4th sidebar item
**Notes:** Help opens as its own special tab (`__help__`) mirroring Settings/Hub.

---

## Cross-surface scope

| Option | Description | Selected |
|--------|-------------|----------|
| GUI only (CLI native --help counts) | Desktop GUI page; CLI native help satisfies parity; web-share + TUI out of scope | ✓ |
| GUI + web surface | Help renders in shared React app for web dashboard viewers too | |
| GUI + web + dedicated CLI help | Adds `agenthub help` command echoing the content | |

**User's choice:** GUI only (CLI native --help counts)
**Notes:** Explicitly reconciles #69's obsolete "TUI parity is release-blocking" section — TUI was removed in v4.0; parity contract is GUI/CLI/web.

---

## Content source & format

| Option | Description | Selected |
|--------|-------------|----------|
| Bundled Markdown files | Option A: repo Markdown rendered at runtime; offline; updates ship with releases | ✓ |
| Bundled structured JSON/TS | Content objects; easier to index, less natural for prose | |
| Remote fetch (Option B) | Fetch from website/GitHub at runtime; needs offline fallback + network handling | |

**User's choice:** Bundled Markdown files
**Notes:** Matches #69 recommendation. FAQ hand-curated/maintainer-reviewed, seeded from #69's set — not auto-scraped.

---

## Search depth

| Option | Description | Selected |
|--------|-------------|----------|
| Rich full-text + highlight + snippets | Debounced live filter over doc+FAQ body, `<mark>` highlighting, context snippets, jump-to-section, empty state | ✓ |
| Simple section-label match | Reuse SettingsSearch label matching; no snippets/highlight | |

**User's choice:** Rich full-text + highlight + snippets
**Notes:** Matches #69 acceptance criteria. SettingsSearch is a structural analog for debounce + anchor-jump only; body-level snippet/highlight indexing is new.

---

## Claude's Discretion

- Markdown rendering approach (renderer lib vs. build-time HTML) + XSS-safe `<mark>` injection strategy.
- Search index granularity (section vs. paragraph vs. heading-block) for snippet extraction.
- Help sidebar icon (heroicons outline, consistent with existing set).
- Keyboard-shortcuts doc content — include only documented/implemented shortcuts; omit if none.

## Deferred Ideas

- Remote/dynamic Help content (Option B).
- `agenthub help` content CLI command.
- Web-share-viewer Help surface.
- FAQ auto-sourced from closed GitHub issues.
