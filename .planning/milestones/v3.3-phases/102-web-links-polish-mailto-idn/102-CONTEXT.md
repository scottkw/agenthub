---
phase: 102-web-links-polish-mailto-idn
type: context
status: ready
mode: auto-generated
---

# Phase 102: Web-Links Polish — mailto + IDN

**Gathered:** 2026-05-12
**Mode:** Auto-generated (discuss skipped — REQUIREMENTS pre-answers grey areas)

<domain>
## Phase Boundary

Close the v3.2 Phase 95 spec gap so the Web-Links plugin matches its documented behavior:
- **POLISH-01**: `mailto:` URLs are clickable in terminal output; system mail client opens on Cmd/Ctrl-click. Closes the spec gap in the Web-Links scheme allowlist.
- **POLISH-02**: `LinkConfirmPopover` triggers when a URL contains non-ASCII (Cyrillic / IDN homograph) hostname characters, with Punycode form displayed alongside the display form.

Scope: 2 requirements (POLISH-01, POLISH-02).

Out of scope:
- New plugin architecture (Phase 95 already shipped the framework — this just plugs documented gaps)
- Replacing existing LinkConfirmPopover styling
</domain>

<decisions>
## Implementation Decisions — Claude's Discretion

All choices deferred to planner. ROADMAP + REQUIREMENTS + the existing Phase 95 plugin code in the repo provide enough context. The fix is mechanical:

- **`mailto:` allowlist**: add `mailto:` to whatever scheme allowlist regex/string the Web-Links plugin uses (likely `frontend/src/plugins/WebLinksPlugin.ts` or similar). System mail handler invoked via existing OS-handler path (Cmd-click → `window.open(href)` or Wails `BrowserOpenURL`).
- **IDN detection**: detect non-ASCII codepoints in hostname; use the URL constructor's `host` property which auto-converts to Punycode in modern browsers (otherwise `punycode` package or browser `URL`'s `hostname` getter).
- **Popover display**: show both forms — "Open `xn--80akhbyknj4f.com` (displayed as `пример.com`)?" — let user choose.
</decisions>

<code_context>
## Existing Code Insights

Phase 95 already shipped the Web-Links plugin with:
- Scheme allowlist (likely missing `mailto:`)
- LinkConfirmPopover for typosquat / IDN warnings (likely existing but not wired for IDN)
- OSC 8 hyperlink support

This phase plugs the documented gaps — both already have UI infrastructure that just needs gating logic added.

Reference files (planner will pinpoint exact paths):
- `frontend/src/plugins/` (or wherever Web-Links plugin lives)
- `frontend/src/components/LinkConfirmPopover.tsx`
- Terminal renderer hookup
</code_context>

<specifics>
## Specific Ideas

1. **POLISH-01 mailto allowlist**: One-line addition to scheme regex. RED test exercises Cmd-click on `mailto:test@example.com` → expects mail handler called.
2. **POLISH-02 IDN detection**: Hostname matches `/[^\x00-\x7F]/` (non-ASCII) → route through LinkConfirmPopover. Display both `URL.hostname` (Punycode if browser supports) and raw display form.
</specifics>

<deferred>
## Deferred

- Phishing heuristics beyond Cyrillic homograph (not in scope)
- IDN allowlist exceptions (e.g., known-safe non-ASCII domains)
</deferred>
