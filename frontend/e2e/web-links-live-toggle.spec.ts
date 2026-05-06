// Phase 95 Plan 95-01 Task 2 — Wave 0 RED scaffold for the web parity
// live-toggle test. Plan 95-06 implements the actual walk.
//
// Walk plan (Plan 95-06 fills in):
//   1. Open web terminal page with webLinks=true (default).
//   2. Click an https URL with Cmd; assert window.open spy fired with
//      _blank + 'noopener,noreferrer'.
//   3. Toggle webLinks=false via daemon RPC.
//   4. SSE settings:plugins event arrives; next URL hover shows no
//      clickable underline.
//   5. Toggle webLinks=true again; next URL hover is clickable again.
//      NO session restart at any step.

import { test } from '@playwright/test';

test.describe('Plan 95-06 — web-links live toggle', () => {
  test.skip('Plan 95-06 implements live-toggle web parity (95-VALIDATION row 95-06-03)', async () => {
    throw new Error('RED scaffold — Plan 95-06 implements e2e per VALIDATION row 95-06-03');
  });
});
