# Phase 90 — Homebrew Tap Test Branch Setup (D-16)

**Prerequisite for:** Plan 06 (E2E rc verification)
**Required before:** Pushing `v3.1.0-rc1` tag

## Why

D-16 (90-CONTEXT.md): the `distribute.yml` tap step runs against a `release-90-test` branch
of `scottkw/homebrew-agenthub` when the tag matches `v*-rc*`. This prevents test artifacts
from appearing to tap users during rc verification.

## One-time setup

Run these commands locally (or from any machine with push access to
`scottkw/homebrew-agenthub`):

```bash
# Clone tap repo (skip if already cloned)
gh repo clone scottkw/homebrew-agenthub /tmp/homebrew-agenthub
cd /tmp/homebrew-agenthub

# Create branch from current main
git fetch origin main
git checkout -b release-90-test origin/main

# Push so distribute.yml can check it out during rc flow
git push -u origin release-90-test
```

## Verification

```bash
gh api /repos/scottkw/homebrew-agenthub/branches/release-90-test --jq .name
# Expected output: release-90-test
```

## Teardown (after v3.1 ships)

Once the real (non-rc) `v3.1.0` tag ships and the tap updates cleanly against `main`,
the test branch can be deleted:

```bash
git push origin --delete release-90-test
```

But NOT before the real release — it's the target of the rc distribute.yml run.
