// Phase 122-03 Task 2 — EnableWebSharingTakeover.
//
// Full-tab takeover surfaced when a remote-session file fetch returns 401 on
// the local-daemon proxy route (i.e. the upstream cap was rejected because
// web-share has been disabled remotely OR the cap rotated). Locked copy
// (122-CONTEXT D-04 verbatim):
//   "Remote session must be web-shared to browse files. Ask the owner to
//    enable sharing."
//
// Recovery affordance: a "Re-enter join code" button so the user can re-paste
// the code in case the upstream rotated. Wired by FileBrowserTab + App.tsx.
//
// Distinct from PermissionDeniedTakeover (which fires on 403 + files.read perm
// missing — a separate failure mode per Plan 122-03-PLAN.md "Pitfall 6"
// reference).

import React from 'react'

export interface EnableWebSharingTakeoverProps {
  onReenterJoinCode: () => void
}

export function EnableWebSharingTakeover({
  onReenterJoinCode,
}: EnableWebSharingTakeoverProps): React.ReactElement {
  return (
    <div
      className="file-browser__takeover file-browser__takeover--enable-web-sharing"
      data-testid="file-browser-enable-web-sharing"
      role="alert"
      aria-live="assertive"
    >
      <h2 className="file-browser__takeover-heading">Web sharing required</h2>
      <p className="file-browser__takeover-body">
        Remote session must be web-shared to browse files. Ask the owner to enable sharing.
      </p>
      <button
        type="button"
        className="file-browser__btn file-browser__btn--secondary"
        onClick={onReenterJoinCode}
      >
        Re-enter join code
      </button>
    </div>
  )
}
