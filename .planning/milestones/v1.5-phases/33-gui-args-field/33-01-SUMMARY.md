---
phase: 33-gui-args-field
plan: 01
subsystem: frontend
tags: [args, modal, localStorage, wails-bindings, react, css]
dependency_graph:
  requires: []
  provides: [args-field-gui, args-wails-binding, args-threading]
  affects: [NewSessionModal, App.tsx, CreateSession binding]
tech_stack:
  added: []
  patterns: [source-inspection tests, per-agent localStorage keys]
key_files:
  created: []
  modified:
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/components/NewSessionModal.tsx
    - frontend/src/style.css
    - frontend/src/App.tsx
    - frontend/src/components/__tests__/NewSessionModal.test.tsx
    - frontend/src/components/__tests__/App.test.tsx
decisions:
  - "Args parsed with argsText.trim().split(/\\s+/).filter(Boolean) so empty input yields [] not ['']"
  - "Per-agent localStorage key pattern: agenthub:args:{cliName} — consistent with agenthub:lastWorkDir"
  - "handleSelectCLI replaces bare setSelectedCLI call to load stored args on agent switch"
metrics:
  duration_minutes: 2
  completed_date: "2026-03-26"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 7
---

# Phase 33 Plan 01: GUI Args Field Summary

## One-liner

Args text field added to new-session modal with per-agent localStorage persistence, clear button, and args threaded from modal through App.tsx to the Wails CreateSession binding as string[].

## What Was Built

### Task 1: Update Wails bindings, add args field to NewSessionModal, thread through App.tsx

- **App.js**: `CreateSession` now accepts 4th `args` parameter — `[cli, name, workDir, args]`
- **App.d.ts**: TypeScript declaration updated with `args: string[]` parameter
- **NewSessionModal.tsx**: Added `ARGS_KEY` helper, `argsText` state with per-agent localStorage init, `handleSelectCLI` to load stored args on agent switch, `handleClearArgs` to remove key and clear field, updated `handleConfirm` to persist/clear and parse args array, args section JSX with input + conditional clear button
- **style.css**: Added `args-row`, `args-input`, `args-input:focus`, `args-input::placeholder`, `args-clear`, `args-clear:hover` CSS per UI-SPEC design contract
- **App.tsx**: `createTab` accepts `args: string[]`, `CreateSession` call passes 4 args, `onConfirm` callback forwards args from modal

### Task 2: Source-inspection tests (TDD)

- **NewSessionModal.test.tsx**: 9 new tests covering ARGS-02 (input class, placeholder, filter(Boolean)), ARGS-04 (localStorage key, handleSelectCLI, ARGS_KEY persistence), ARGS-05 (handleClearArgs, removeItem, aria-label)
- **App.test.tsx**: 2 new tests covering ARGS-02 threading (onConfirm passes args, CreateSession 4-arg call)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — args field is fully wired from modal UI through App.tsx to the Wails binding.

## Verification Results

- All 139 vitest tests pass (128 original + 11 new ARGS tests)
- `App.js` contains `[cli, name, workDir, args]`
- `App.d.ts` contains `args: string[]`
- `NewSessionModal.tsx` contains `agenthub:args:`, `handleClearArgs`, `handleSelectCLI`, `.filter(Boolean)`, `new-session-modal__args-input`, `Clear Args`, `aria-label="Clear arguments"`, `e.g. --model claude-opus-4-5`
- `style.css` contains `.new-session-modal__args-row`, `.new-session-modal__args-input`, `.new-session-modal__args-clear`, `border-color: #7aa2f7`
- `App.tsx` contains `createTab(cli, workDir, args)`, `CreateSession(cliName, defaultName, workDir, args)`

## Commits

| Hash | Message |
|------|---------|
| a93db10 | feat(33-01): add args field to new-session modal and thread through App |
| 5c87e12 | test(33-01): add ARGS-02/04/05 source-inspection tests |

## Self-Check: PASSED
