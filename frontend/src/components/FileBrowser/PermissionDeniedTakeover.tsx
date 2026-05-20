// Phase 120-04 Task 1 — files.read capability missing takeover.
//
// Full-tab takeover (replaces breadcrumb + list + preview entirely) — not
// merely the preview slot. Triggered when the capability probe
// (useFilesCapability) resolves to 'denied' OR when /api/files/list returns
// 403 with body containing 'files.read'.
//
// No retry button — the user cannot recover from this state on their own;
// the session owner must re-share with files.read in the perms claim.
//
// Copy locked verbatim per UI-SPEC §"Error copy" row
// "files.read permission missing (full-tab takeover)".
//
// role="alert" + aria-live="assertive" so screen readers announce the
// takeover immediately upon mount (UI-SPEC §Accessibility "Error / takeover
// regions").

import React from 'react'

export function PermissionDeniedTakeover(): React.ReactElement {
  return (
    <div
      className="file-browser__takeover file-browser__takeover--permission-denied"
      data-testid="file-browser-permission-denied"
      role="alert"
      aria-live="assertive"
    >
      <h2 className="file-browser__takeover-heading">files.read permission required</h2>
      <p className="file-browser__takeover-body">
        Ask the session owner to re-share this session with file access enabled.
      </p>
    </div>
  )
}
