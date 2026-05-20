# Phase 112: WebGL recovery banner rendering - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning
**Mode:** Pre-authored from `.planning/milestones/v3.3.1-ROADMAP.md` + `.planning/REQUIREMENTS.md`

<domain>
## Phase Boundary

When the terminal's WebGL context is lost, the `WebGLRecoveryBanner` renders inside `.banner-stack` and auto-dismisses after 8s — restoring the Phase 93 user-visible recovery flow. The banner exists in code but doesn't render in the live path; DOM fallback continues to work, which is why the bug shipped undetected. Closes GitHub Issue #55.

</domain>

<decisions>
## Implementation Decisions

### Root cause (suspected, per v3.3.1-ROADMAP.md Phase 112)
- **Closure rot in `frontend/src/components/TerminalPanel.tsx:391-395`** — the `onContextLoss` callback captured the banner-state setter at component mount time. By the time the WebGL context is lost (potentially well after mount, possibly after re-renders), the captured setter no longer corresponds to the current React tree.
- **DOM fallback still works** because that path is independent — xterm.js falls back to the DOM renderer on context loss without needing our state.

### Fix shape
- **Use a `ref` or stable callback** for the banner-state setter so the `onContextLoss` handler always sees the current setter. Standard React pattern: `useRef` to hold the latest setter and a `useEffect` to keep the ref synced.
- **Alternative:** restructure the effect dependencies so `onContextLoss` re-binds on relevant state changes. Either pattern is acceptable; researcher should pick based on what's already in the file.
- **Verify with DevTools** — `WEBGL_lose_context.loseContext()` triggers the path. Banner must appear in `.banner-stack` and auto-dismiss after 8s.

### Cross-surface verification (release gate)
- **Desktop:** Wails dev mode (`wails dev`) — note prod app has no DevTools per project memory `project_wails_devtools_disabled_in_prod`. Alternatively web-share to a regular Chrome tab.
- **Web:** browser at the local web-share URL, open DevTools, trigger context loss, observe banner.
- **DOM fallback regression smoke:** after context loss, terminal content must remain readable (the existing fallback path is not the fix-point and must not be regressed).
- **Single fix covers both surfaces** — same React component is shared between desktop and web.

### Out of scope
- xterm.js renderer fallback logic — already works, don't touch.
- Banner styling / copy — that's Phase 93's design; just make it render.
- Other banner types in `.banner-stack` — only WebGLRecoveryBanner.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/TerminalPanel.tsx` — the React component owning xterm.js init + the `onContextLoss` callback.
- `WebGLRecoveryBanner` component — already exists, just isn't rendering. Researcher should find its file.
- `.banner-stack` container — banner-rendering surface (existing).

### Established Patterns
- React `useRef` + `useEffect` pattern for stable callbacks against re-renders. Search the codebase for prior usage.
- xterm.js addon-loss handler conventions — researcher should check what the xterm.js docs say about `loseContext` semantics.

### Integration Points
- xterm.js `addon-webgl` or equivalent — where `onContextLoss` is registered.
- React state setter for visible banners in `.banner-stack`.
- 8-second auto-dismiss timer — existing Phase 93 contract.

</code_context>

<specifics>
## Specific Ideas

- Issue #55 reproduction: open terminal session, open DevTools, run `document.querySelector('canvas').getContext('webgl2').getExtension('WEBGL_lose_context').loseContext()`. On `main`, no banner appears. After fix, banner renders in `.banner-stack`, auto-dismisses after 8s.
- macOS executor CAN do this UAT via `wails dev` OR via local web-share + Chrome DevTools.

</specifics>

<deferred>
## Deferred Ideas

None.

</deferred>
