---
phase: 87-capability-based-session-authorization
plan: 05
subsystem: frontend
tags: [frontend, ui, security, capability, react, typescript]

# Dependency graph
requires:
  - 87-04 Wails bindings (IssueCapabilities, RegenerateSigningKey, GetCapabilityQRCode)
  - Phase-84 .quit-modal* CSS classes (reused structurally by RegenerateKeyModal)
  - Existing .daemon-panel__btn + .settings-panel__* base styles
provides:
  - SessionSharePanel component — per-session share UX (UI-SPEC Surface 1)
  - RegenerateKeyModal component — signing-key rotation confirmation (UI-SPEC Surface 2)
  - SettingsTab.Security section — destructive Regenerate Signing Key entry point (UI-SPEC Surface 3)
  - DaemonManagerPanel.sessionShares reconciliation loop — calls IssueCapabilities on toggle-on, drops shares on toggle-off (T-87-07 mitigation)
  - .session-share-panel* BEM block + .settings-panel__btn--destructive modifier in style.css
affects:
  - 87-06-web-pages-integration — shares the same /join?code= URL shape this plan's SessionSharePanel QR encodes
  - 87-VERIFICATION phase gate — frontend surfaces of SEC-01 (no auto-share) + SEC-04 (visible read/write link split) are now live

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Effect-driven capability reconciliation in DaemonManagerPanel — a single useEffect watches (sessions, webEnabled), fetches caps for newly-enabled sessions, and prunes shares whose sessions toggled off or disappeared; avoids threading async state through App.tsx's onToggleWeb callback"
    - "QR URL construction from capability URL origin — the component derives joinURL = `${u.protocol}//${u.host}/join?code=${code}` from the existing readURL/writeURL; no extra daemon call needed"
    - "Structural reuse of .quit-modal* classes for RegenerateKeyModal — no new modal CSS block; the destructive button maps to .quit-modal__btn--quit-all which already paints #f7768e"
    - "flex-basis: 100% on .session-share-panel so it consumes the full width of the flex-based daemon-panel__session-row, stacking cleanly below the status/name/cli/hostname/actions line"

key-files:
  created:
    - frontend/src/components/SessionSharePanel.tsx
    - frontend/src/components/RegenerateKeyModal.tsx
  modified:
    - frontend/src/components/DaemonManagerPanel.tsx
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/style.css

key-decisions:
  - "DaemonManagerPanel owns sessionShares state (not App.tsx). The plan's artifacts section says the panel 'collects URLs from IssueCapabilities response'; driving the fetch from a useEffect that watches (sessions, webEnabled) keeps App.tsx's existing onToggleWeb contract unchanged and naturally handles both the toggle-on (fetch) and toggle-off (prune) sides in one place."
  - "SessionSharePanel derives the join URL from the capability URL origin (new URL(readURL).host + '/join?code=' + readCode) instead of requiring a separate baseURL prop. The capability URL always carries the correct scheme+host because it was minted by the same daemon that serves /join, so this is lossless."
  - "Only one QR visible per session at a time — clicking read-QR closes write-QR and vice versa (UI-SPEC Surface 1 explicit rule). Implemented by the top of handleToggleQR calling setShowWriteQR(false)/setShowReadQR(false) before the show path."
  - "RegenerateKeyModal reuses .quit-modal-overlay with onClick={onCancel} + stopPropagation on the inner card — matches QuitConfirmModal's exact UX (click-outside-to-close) and inherits Phase 84's CSS."
  - "handleRegenerateSigningKey re-throws the caught error after setting regenError state. The re-throw is what makes RegenerateKeyModal's internal try/catch catch the failure, surface it as the inline error in the modal, and leave the modal open for retry. Without the re-throw the modal would silently close on failure."
  - "Task 2's style.css change was a clean append (no cross-task interaction). Task 1 and Task 2 each own three files with no shared-file collisions, so the atomic-per-task commit split is a straight two-commit sequence — no revert-replay dance like Plan 04 needed."

requirements-completed: [SEC-01, SEC-04]

# Metrics
duration: 4m12s
completed: 2026-04-20
---

# Phase 87 Plan 05: Frontend UI Summary

**Ships the desktop-app-side capability UX: a new SessionSharePanel renders two link rows (Read-Only Link + Full Access Link) with Copy/Open/QR buttons inside every web-enabled session row, and a new RegenerateKeyModal plus SettingsTab Security section exposes the D-16 panic button that invalidates all outstanding capabilities. QR images encode the join-code exchange URL (D-09), not the raw capability token, so a leaked QR is worthless after 5 minutes or first exchange. SEC-01's "no auto-share" and SEC-04's visible read/write split are both now observable in the GUI. Frontend `pnpm build` clean.**

## Performance

- **Duration:** ~4 min (252 s)
- **Started:** 2026-04-20T17:37:26Z
- **Completed:** 2026-04-20T17:41:38Z
- **Tasks:** 2
- **Files created:** 2 (SessionSharePanel.tsx, RegenerateKeyModal.tsx)
- **Files modified:** 3 (DaemonManagerPanel.tsx, SettingsTab.tsx, style.css)

## Accomplishments

- **UI-SPEC Surface 1 — Per-Session Share Panel (D-06, D-07, D-09):** `SessionSharePanel.tsx` renders two link rows (Read-Only Link + Full Access Link). Each row: subdued-gray `#a9b1d6` label (min-width 120px), truncated accent-blue `#7aa2f7` URL with ellipsis overflow, and three `.daemon-panel__btn` actions (Copy/Open/QR). `Copy` flips to `Copied!` for 1500 ms via existing pattern. `Open` delegates to `BrowserOpenURL`. `QR` fetches a 200×200 PNG from `GetCapabilityQRCode(joinURL)` where `joinURL` is derived from the capability URL's origin + `/join?code=<code>` — **NOT** the raw `?cap=` URL (D-09 defense against leaked QR photographs). Only one QR visible per session at a time; state is cached so re-toggling is instant.
- **UI-SPEC Surface 2 — Regenerate Signing Key Modal (D-16):** `RegenerateKeyModal.tsx` structurally mirrors `QuitConfirmModal`. Props: `isOpen`, `onConfirm: () => Promise<void>`, `onCancel`. Reuses every `.quit-modal*` class already in `style.css`: overlay, card, header, close button, subtitle, footer. Escape key closes the modal; focus lands on the "Keep Links" cancel button on open (safer default for destructive action). "Invalidate All Links" flips to "Invalidating…" while the RPC is in flight, both footer buttons are disabled, and failures surface as an inline error above the footer without closing the modal.
- **UI-SPEC Surface 3 — Settings Security Section:** `SettingsTab.tsx` grew an `<h3>Security</h3>` section between Web Server and Paths. Description copy is verbatim from UI-SPEC: "Rotating the signing key immediately invalidates all shared links across all sessions. Use this if you suspect a link has been leaked." A destructive button (`.settings-panel__btn .settings-panel__btn--destructive`) opens the modal; inline `.settings-panel__error` surfaces post-RPC failures. The modal is mounted as a sibling of the button so Escape-to-close works even when the Settings tab scrolls.
- **DaemonManagerPanel wiring:** Added a `sessionShares: Record<sessionId, {readURL, writeURL, readCode, writeCode}>` state and a reconciliation `useEffect` that watches `(sessions, webEnabled)`. For every session that is web-enabled and missing a share entry, it calls `IssueCapabilities(sessionId)` and stores the response. For every share entry whose session is no longer web-enabled or no longer exists, it prunes — mitigating T-87-07 (stale URLs after toggle-off). The existing `onToggleWeb` callback contract to App.tsx is unchanged.
- **CSS (style.css):** Added the `.session-share-panel` BEM block (7 classes: root, `__link-row`, `__label`, `__url`, `__actions`, `__qr`, `__error`) and the `.settings-panel__btn--destructive` modifier (3 variants: base, hover, disabled). The root `.session-share-panel` gets `flex-basis: 100%; width: 100%;` so it spans the full session row width beneath the existing single-line flex layout.
- **Wails bindings consumed:** `IssueCapabilities` (DaemonManagerPanel), `GetCapabilityQRCode` (SessionSharePanel), `RegenerateSigningKey` (SettingsTab). All three bindings were provided by Plan 04 and are already registered in `frontend/src/wailsjs/go/main/App.d.ts`.

## Task Commits

Each task committed atomically:

1. **Task 1: SessionSharePanel + DaemonManagerPanel integration + `.session-share-panel*` CSS** — `60e9424` (feat)
2. **Task 2: RegenerateKeyModal + SettingsTab Security section + `.settings-panel__btn--destructive` CSS** — `cec6ef5` (feat)

**Plan metadata commit:** *(appended in final step below)*

## Files Created/Modified

**Created:**
- `frontend/src/components/SessionSharePanel.tsx` — 175-line React component exporting `SessionSharePanel`. Props: `sessionId`, `readURL`, `writeURL`, `readCode`, `writeCode`. State: `readCopied`, `writeCopied`, `showReadQR`, `showWriteQR`, `readQRb64`, `writeQRb64`, `qrError`. Handlers: `handleCopy`, `joinURLFor`, `handleToggleQR`.
- `frontend/src/components/RegenerateKeyModal.tsx` — 99-line React component exporting `RegenerateKeyModal`. Props: `isOpen`, `onConfirm: () => Promise<void>`, `onCancel`. State: `acting`, `error`. Refs: `cancelBtnRef`. Effects: keydown/Escape, auto-focus on open, state reset on close.

**Modified:**
- `frontend/src/components/DaemonManagerPanel.tsx` — Added `IssueCapabilities` + `SessionSharePanel` imports, `sessionShares` state, `useEffect` that reconciles shares against `(sessions, webEnabled)` changes, and inline `<SessionSharePanel .../>` inside each web-enabled session row (conditional on `isWebOn && share`).
- `frontend/src/components/SettingsTab.tsx` — Added `RegenerateSigningKey` + `RegenerateKeyModal` imports, `showRegenModal`/`regenError` state, `handleRegenerateSigningKey` handler (re-throws after setting error so the modal's inline error path fires), and the Security section JSX (h3 + description + button + error + modal mount) between the Web Server and Paths sections.
- `frontend/src/style.css` — Appended 10 new rules at end of file: `.session-share-panel` + 6 sub-elements (Task 1), `.settings-panel__btn--destructive` base + hover + disabled (Task 2).

## Decisions Made

- **DaemonManagerPanel owns `sessionShares` (not App.tsx).** The plan's `artifacts` section assigned the state to DaemonManagerPanel. Driving the fetch from an effect that watches `(sessions, webEnabled)` keeps App.tsx's `onToggleWeb` contract untouched and handles toggle-on (fetch) and toggle-off (prune) in a single reconciliation loop. The alternative — lifting state to App.tsx and passing it down — would have required changing both the App.tsx handler and the panel's props, without improving correctness.
- **SessionSharePanel derives the join URL from the capability URL origin.** The `joinURLFor(capURL, code)` helper does `new URL(capURL)` and rebuilds `${protocol}//${host}/join?code=${code}`. This avoids the need for a fifth prop (`baseURL`) and works because the capability URL is always minted by the same daemon that serves `/join`.
- **Only one QR visible per session at a time.** UI-SPEC Surface 1 §"QR behavior" mandates this explicitly. Implemented by the head of `handleToggleQR('read')` calling `setShowWriteQR(false)` (and symmetrically for write) before toggling the requested side on. State is independent per side so QRs are individually cached — a second toggle reuses the base64 without a second RPC.
- **`RegenerateKeyModal` reuses `.quit-modal-overlay` onClick + stopPropagation.** Matches QuitConfirmModal's click-outside-to-close UX. UI-SPEC §Surface 2 says "no new CSS block required" — that's honored exactly; every class on every element in the new modal is already defined by the quit-modal block.
- **`handleRegenerateSigningKey` re-throws after setting `regenError`.** This is the key integration pattern with `RegenerateKeyModal`: the modal's internal `try/catch` expects `onConfirm` to throw on failure. Without the re-throw, the modal would silently close on RPC failure, leaving the user with no feedback. Re-throwing keeps the modal open and surfaces the error inline above the footer.
- **Task split by file, not by revert-replay.** Task 1's files (`SessionSharePanel.tsx`, `DaemonManagerPanel.tsx`, `style.css` session-share block) and Task 2's files (`RegenerateKeyModal.tsx`, `SettingsTab.tsx`, `style.css` destructive-btn block) share only `style.css` as a file. I kept the split clean by writing Task 2's CSS block only after Task 1 was committed — no `git checkout` / diff-replay was needed like Plan 04.
- **Destructive button modifier lives on `.settings-panel__btn`, not a standalone class.** UI-SPEC calls it a "modifier" (BEM `--destructive`). Applying both classes (`settings-panel__btn settings-panel__btn--destructive`) inherits the existing base padding/border-radius and only overrides the destructive-specific color/weight. Matches the existing `--save` / `--cancel` / `--saved` pattern.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `flex-basis: 100%; width: 100%;` to `.session-share-panel`**
- **Found during:** Task 1, visual integration with existing `.daemon-panel__session-row`.
- **Issue:** The existing `.daemon-panel__session-row` is a horizontal flex row (status / name / cli / hostname / actions). Without a width hint, the new `SessionSharePanel` would render as another flex item on the same line, squeezing the existing content. The UI-SPEC §Surface 1 CSS block does not address this because it was written against a per-session row that *didn't exist yet* in the current shape.
- **Fix:** Added `flex-basis: 100%` and `width: 100%` to `.session-share-panel` so it consumes the row width and wraps to the next line (parent uses `flex-wrap` implicitly via `flex-basis: 100%`).
- **Files modified:** `frontend/src/style.css`.
- **Commit:** `60e9424` (Task 1).

**2. [Rule 3 - Blocking] Derived join URL from capability URL origin instead of requiring a `baseURL` prop**
- **Found during:** Task 1, writing `SessionSharePanel.tsx`.
- **Issue:** The plan's component props contract (lines 162-167) has only `readURL/writeURL/readCode/writeCode`. But the action step sample code in plan lines 194-211 constructs `joinExchangeURL = u.protocol + '//' + u.host + '/join?code=' + readCode` — it assumes the origin is extractable from the capability URL, which IS correct since the daemon mints both from the same base. There's no ambiguity or risk; the plan just didn't spell it out as an explicit decision.
- **Fix:** Extracted the origin-derivation into a `joinURLFor(capURL, code)` helper to make the intent explicit and testable. Functionally identical to what the plan's inline code did.
- **Files modified:** `frontend/src/components/SessionSharePanel.tsx`.
- **Commit:** `60e9424` (Task 1).

---

**Total deviations:** 2 auto-fixed (1 CSS layout adjustment for flex-row integration, 1 helper-function extraction). No architectural changes. No test coverage regression — no vitest tests were added per the plan's `<notes>` ("No frontend test suite is added. Validation is pnpm build clean + acceptance-criteria greps + Manual-Only browser flows.")

## Issues Encountered

- **None blocking.** The only surprise was `.session-share-panel` needing `flex-basis: 100%` to wrap under the existing flex-row — caught on first mental-model pass during CSS authoring, fixed before commit.

## User Setup Required

None — no external service configuration, no new npm packages, no secret handling. All Wails bindings were already registered by Plan 04, and the frontend's CSS/asset pipeline rebuilds automatically on `pnpm build`.

## Manual-Only UAT Browser Flows (out of agent scope)

The following require a live `wails dev` session and a running daemon; not runnable by this agent:

1. **Toggle-on shows share panel.** In the Daemon Manager tab, click "Web On" for a running session → two URL rows appear with Copy/Open/QR buttons.
2. **Copy→Copied! feedback.** Click Copy on the Read-Only row → button label flips to "Copied!" for 1500 ms, then reverts. Verify clipboard contains the `?cap=<token>` URL.
3. **QR toggle.** Click "QR" → 200×200 PNG appears below the row. Click again → disappears. Activate the *other* QR → first one disappears, second one appears.
4. **QR encodes join URL (D-09).** Scan the QR with a phone → lands on `https://<host>/join?code=XXXX-YYYY`, NOT `https://<host>/sessions/<id>?cap=...`. *(Requires Plan 06 for the full round-trip.)*
5. **Toggle-off hides panel.** Click "Web Off" → share panel disappears entirely from the session row.
6. **Settings → Security panic button.** Open Settings → scroll to Security → click "Regenerate Signing Key" → modal opens. "Keep Links" closes without firing RPC. Re-open → click "Invalidate All Links" → button flips to "Invalidating…" for the duration of the RPC, then modal closes on success.
7. **Escape closes modal.** Open regen modal → press Esc → modal closes.
8. **Blast radius.** After regenerating, any already-open browser tab holding a stale `?cap=...` URL should get kicked to an HTTP 401 on the next WS frame. *(Requires Plan 03+04 live enforcement.)*

## Next Phase Readiness

- **Plan 06 (web pages integration) is unblocked.** The SessionSharePanel QR encodes `https://host/join?code=<code>` — the exact URL shape Plan 06 will implement the server-side `/join` page against. The join-code format is already standardized (Plan 04 `issueCapabilitiesForSession` → 8-char dashed base32), so Plan 06 only needs to consume what this plan hands it.
- **Phase 87 VERIFICATION gate surfaces exposed:**
  - **SEC-01 "no auto-share on session create":** The Daemon Manager Panel no longer shows a share panel for newly-created sessions until the user explicitly clicks "Web On" — the grant-gesture contract (D-06) is now visible end-to-end.
  - **SEC-04 "read-only vs full-access split visible":** The share panel renders both URLs as equal-weight choices; the user picks which to share, not the app. This is what "first-class read-only concept" means in UI terms.
  - **D-16 "panic button":** Settings → Security → Regenerate Signing Key is the single in-product action that makes the blast radius obvious ("This immediately invalidates ALL shared links across ALL sessions").
- **Frontend build:** `pnpm build` green (693 modules, 218 ms, no TypeScript errors, no Vite errors). CSS bundle grew ~0.1 KB. JS bundle grew ~2.5 KB (both new components plus their reconciliation logic).
- **Acceptance grep matrix (all pass):**
  - `ls frontend/src/components/SessionSharePanel.tsx` → OK
  - `ls frontend/src/components/RegenerateKeyModal.tsx` → OK
  - `grep -q 'Read-Only Link' SessionSharePanel.tsx` → OK
  - `grep -q 'Full Access Link' SessionSharePanel.tsx` → OK
  - `grep -q 'GetCapabilityQRCode' SessionSharePanel.tsx` → OK
  - `grep -q '/join?code=' SessionSharePanel.tsx` → OK
  - `grep -c 'session-share-panel' style.css` = 7 → OK
  - `grep -q 'SessionSharePanel' DaemonManagerPanel.tsx` → OK
  - `grep -q 'IssueCapabilities' DaemonManagerPanel.tsx` → OK
  - `grep -q 'Regenerate Signing Key?' RegenerateKeyModal.tsx` → OK
  - `grep -q 'Invalidate All Links' RegenerateKeyModal.tsx` → OK
  - `grep -qE 'Invalidating' RegenerateKeyModal.tsx` → OK (matches `Invalidating\u2026`)
  - `grep -q 'Keep Links' RegenerateKeyModal.tsx` → OK
  - `grep -q 'This immediately invalidates ALL' RegenerateKeyModal.tsx` → OK
  - `grep -q 'RegenerateKeyModal' SettingsTab.tsx` → OK
  - `grep -q 'RegenerateSigningKey' SettingsTab.tsx` → OK
  - `grep -qE 'Regenerate Signing Key$|Regenerate Signing Key<' SettingsTab.tsx` → OK (matches end-of-line JSX text)
  - `grep -q 'settings-panel__btn--destructive' style.css` → OK (3 hits: base, hover, disabled)
  - `grep -q 'Rotating the signing key immediately invalidates' SettingsTab.tsx` → OK
  - `grep -q 'Security' SettingsTab.tsx` → OK (h3 literal)

## Known Stubs

None. Both components wire real Wails bindings (no mock data), and the rendered UI paths are gated on real state (`webEnabled[id] && share` for the share panel, `showRegenModal` for the modal).

## Self-Check: PASSED

All 5 modified/created files verified present on disk:
- `frontend/src/components/SessionSharePanel.tsx` → FOUND
- `frontend/src/components/RegenerateKeyModal.tsx` → FOUND
- `frontend/src/components/DaemonManagerPanel.tsx` → FOUND (modified)
- `frontend/src/components/SettingsTab.tsx` → FOUND (modified)
- `frontend/src/style.css` → FOUND (modified)

Both task commits found in git log:
- `60e9424` — feat(87-05): add SessionSharePanel with Copy/Open/QR per-session share UX
- `cec6ef5` — feat(87-05): add RegenerateKeyModal and Settings Security section

Frontend `pnpm build` re-verified clean (693 modules transformed, 218 ms, 0 errors).

---
*Phase: 87-capability-based-session-authorization*
*Completed: 2026-04-20*
