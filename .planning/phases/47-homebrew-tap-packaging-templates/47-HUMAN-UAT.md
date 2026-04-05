---
status: partial
phase: 47-homebrew-tap-packaging-templates
source: [47-VERIFICATION.md]
started: 2026-04-05T00:40:00Z
updated: 2026-04-05T00:40:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Install AgentHub via Homebrew tap on a clean macOS machine
expected: `brew tap scottkw/agenthub && brew install --cask agenthub` installs a working AgentHub.app without any security override prompt
result: [pending]

### 2. Trigger distribute.yml end-to-end via a real release publish
expected: After publishing a GitHub Release, the Casks/agenthub.rb in scottkw/homebrew-agenthub is updated automatically with the correct version and SHA256 within ~10 minutes
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
