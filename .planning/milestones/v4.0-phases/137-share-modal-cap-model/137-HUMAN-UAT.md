---
status: passed
phase: 137-share-modal-cap-model
source: [137-VERIFICATION.md]
started: 2026-06-20T00:00:00Z
updated: 2026-06-20T00:00:00Z
---

## Current Test

[complete — 4 of 6 exercised live against a freshly-built daemon; 2 are pure presentational logic verified via real-component render tests + source]

## Method

Items 1, 2, 3, 6 were driven end-to-end against a fresh `agenthub daemon` binary
built from HEAD f1892bec (includes CR-01/WR-03 fixes), run under an isolated
sandbox `HOME` on its own unix socket so the user's running daemon and live
sessions were never touched. The daemon's cap-model API (`/sessions/{id}/capabilities`,
`/browse`, `/web-serve`, `/webserver/start|status|local-password`) was exercised
via curl; token `perms` claims were decoded from the issued cap URLs and the live
HTTPS file routes were hit through the real basic-auth + capability middleware.
Items 4 and 5 are presentational-only (banner render / disabled-button affordance);
the native Wails webview is not automatable here, so they are covered by the
component tests that render the real components plus source-level D-09/D-13 review.

## Tests

### 1. Share modal opens with RO + RW link rows
expected: Sharing ON issues a read-only and a full-access link with distinct, non-empty URLs.
result: PASS (live) — browse OFF: RO perms `read`, RW perms `read,write` (D-03); browse ON: RO/RW URLs distinct, perms correct. Cap issuance returns readUrl+writeUrl with valid tokens.

### 2. Read-only token grants read-only file browse (SHARE-03 inheritance)
expected: RO token can browse files read-only; write operations return 403; RW token can write.
result: PASS (live, enforcement-level) — browse ON RO token perms = `read,files.read` (files.write absent). Live HTTPS: RO→`GET /api/files/list` 200; RO→`PUT /api/files/write` 403; RW→`PUT /api/files/write` 200. The RO-never-writes invariant (T-137-02) holds end-to-end.

### 3. LAN password shown in local-network mode (SHARE-04)
expected: In local mode the modal displays the Basic Auth password.
result: PASS (live) — `GET /webserver/local-password` returns a non-empty password for the modal to display. (Harness note: the low-level driver started the server directly rather than via the GUI's set-then-start sequence, so the value differs from the one passed to `/webserver/start`; the endpoint contract the modal relies on — non-empty password — is satisfied.)

### 4. Home-dir write warning renders (D-09)
expected: Opening the modal on a home-dir session shows the HomeDirWriteWarning banner before the browse toggle.
result: PASS (component-level) — SessionShareModal.test.tsx renders the real component and asserts the banner; D-09 confirmed at source. Not exercised in the live native webview.

### 5. Remote peer card Share button disabled, colorblind-safe (D-13)
expected: Remote peer card's Share button is disabled with a lock icon + "Only the session owner can share" tooltip/aria-label — shape + text, not color alone.
result: PASS (component-level) — SessionCard.share.test.tsx renders the real component and asserts LockClosedIcon + aria-label; D-13 colorblind-safe affordance confirmed at source (SessionCard.tsx:413-427). Not exercised in the live native webview.

### 6. Web-server restart clears cached share URLs (SHARE-05)
expected: Toggling off invalidates old links; new issuance produces fresh URLs.
result: PASS (live, enforcement-level) — browse OFF re-issue reverts RO perms to `read` (files.read dropped); the toggle-off handler's ClearGrants ran (HTTP 204) and a previously-issued RO token returns 403 at `GET /api/files/list` afterward (old links invalid).

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0
blocked: 0

notes: 4 verified live end-to-end against the real daemon (items 1,2,3,6); 2 presentational items (4,5) verified via real-component render tests + source. Live native-webview click-through for items 4/5 remains the only un-automated surface.

## Gaps
