# Phase 57: Quick Wins - Research

**Researched:** 2026-04-08
**Domain:** Go path augmentation (CLI detection) + React/TypeScript sidebar label
**Confidence:** HIGH

## Summary

Phase 57 has two independent, well-scoped changes:

1. **DET-01** — The Anthropic native installer places the `claude` binary at `~/.local/bin/claude` (macOS/Linux) or `%USERPROFILE%\.local\bin\claude.exe` (Windows). The existing `AugmentServicePath()` function in `internal/daemon/path.go` does NOT include `~/.local/bin` in its candidates list, so `exec.LookPath("claude")` fails when the app launches from Finder/Dock or as a daemon service (which lacks a user shell PATH). The fix is a one-line addition of `filepath.Join(home, ".local", "bin")` to the `candidates` slice. This path resolves correctly on all three platforms via `os.UserHomeDir()`.

2. **UI-01** — The `Sidebar` component in `frontend/src/components/Sidebar.tsx` has `"New Tab"` hardcoded in both the label text (`<span className="sidebar__label">New Tab</span>`) and the `aria-label` attribute (`aria-label="New Tab"`). Both must be updated to `"New Session"`. The existing `Sidebar.test.tsx` does not test the "New Tab" label text (it only tests aria-label="Sessions" and general structure), so existing tests will not break, but a new test verifying `aria-label="New Session"` and the visible label text "New Session" should be added.

**Primary recommendation:** Add `~/.local/bin` to `AugmentServicePath()` candidates; rename "New Tab" to "New Session" in `Sidebar.tsx` (both label and aria-label).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DET-01 | User can launch Claude Code sessions when Claude is installed via Anthropic native installer (~/.local/bin/claude on macOS/Linux, %USERPROFILE%\.local\bin\claude.exe on Windows) | Add `filepath.Join(home, ".local", "bin")` to `AugmentServicePath()` candidates in `internal/daemon/path.go`; also add to `runGUI()` call chain in `main.go` (already calls `AugmentServicePath`). |
| UI-01 | Sidebar displays "New Session" instead of "New Tab" | Two-character change in `Sidebar.tsx`: update `aria-label` and `<span>` text from "New Tab" to "New Session". |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `os/exec` | N/A | `exec.LookPath` for CLI binary detection | Already used in `detect.go` |
| Go stdlib `os` | N/A | `os.UserHomeDir()` for home dir resolution | Already used in `path.go` |
| Go stdlib `path/filepath` | N/A | `filepath.Join` for cross-platform path construction | Already used in `path.go` |
| React (TypeScript) | Current project | Sidebar component rendering | Already used |
| Vitest | ^4.1.0 | Frontend unit tests | Already configured |

### Supporting
None — this phase uses only existing project dependencies.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Adding path to `AugmentServicePath` | Adding path to `DetectCLIs` directly | `AugmentServicePath` is the right abstraction — it's the single place for PATH augmentation. Adding it to detect.go would create a second code path doing the same job. |

**Installation:** No new packages required.

## Architecture Patterns

### Recommended Project Structure

No structural changes. Edits are confined to:
- `internal/daemon/path.go` — add one candidate path
- `internal/daemon/path_test.go` — add test for `~/.local/bin`
- `frontend/src/components/Sidebar.tsx` — rename label + aria-label
- `frontend/src/components/__tests__/Sidebar.test.tsx` — add/update label test

### Pattern 1: AugmentServicePath Candidate List Extension

**What:** The `AugmentServicePath()` function builds a `candidates` slice of well-known bin directories and prepends any that exist to the process PATH. Adding a new candidate follows the same pattern as existing entries.

**When to use:** Any time a new tool installer places binaries in a non-standard location not covered by shell init files.

**Example:**
```go
// Source: internal/daemon/path.go (existing pattern)
candidates := []string{
    filepath.Join(home, ".local", "bin"),  // ADD: Anthropic native installer (macOS/Linux/Windows)
    filepath.Join(home, ".volta", "bin"),
    "/opt/homebrew/bin",
    "/usr/local/bin",
    "/home/linuxbrew/.linuxbrew/bin",
    nvmActiveBin(home),
}
```

**Key detail:** `os.UserHomeDir()` on Windows returns `%USERPROFILE%` (e.g., `C:\Users\ken`), so `filepath.Join(home, ".local", "bin")` correctly produces `C:\Users\ken\.local\bin` — matching the Windows native installer path documented in DET-01.

### Pattern 2: Sidebar Label Update

**What:** The "New Tab" button in `Sidebar.tsx` has both a visible label (inside `<span className="sidebar__label">`) and an `aria-label` attribute. Both must be updated for accessibility correctness.

**When to use:** Whenever a UI label changes — always update both the visible text and the aria-label together.

**Example:**
```tsx
// Before (current):
<button
  className="sidebar__item"
  onClick={onAdd}
  aria-label="New Tab"
>
  <PlusIcon className="sidebar__icon" />
  {!collapsed && <span className="sidebar__label">New Tab</span>}
</button>

// After (target):
<button
  className="sidebar__item"
  onClick={onAdd}
  aria-label="New Session"
>
  <PlusIcon className="sidebar__icon" />
  {!collapsed && <span className="sidebar__label">New Session</span>}
</button>
```

**Collapsed state:** When `collapsed === true`, the label span is hidden but `aria-label="New Session"` is still present and serves as the tooltip-equivalent for screen readers and keyboard navigation. This satisfies success criterion 3 (tooltip/collapsed state reflects "New Session").

### Anti-Patterns to Avoid

- **Do NOT add `~/.local/bin` only to `DetectCLIs`** — the path must be in `AugmentServicePath` so it is available for all CLIs (codex, gemini, opencode) and for session creation via `exec.LookPath` in the PTY backend, not just detection.
- **Do NOT update only the `<span>` label without updating `aria-label`** — this leaves the collapsed state and screen readers showing "New Tab".
- **Do NOT skip the test** — the existing Sidebar tests check structure and aria-labels (e.g., `aria-label="Sessions"`), so adding a test for `aria-label="New Session"` is consistent with the existing test style.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform home dir | Custom env var lookup | `os.UserHomeDir()` | Handles all three platforms, respects `HOME`/`USERPROFILE` correctly |
| CLI binary location | Custom PATH search | `exec.LookPath` (already used) | Standard Go approach, handles `.exe` extension on Windows automatically |

## Runtime State Inventory

> Not a rename/refactor phase. Omit — no runtime state is affected.

## Environment Availability

Step 2.6: SKIPPED — No external dependencies. Both changes are pure code edits (Go + TypeScript). The Go toolchain and Node/pnpm are already confirmed installed in the project environment.

## Common Pitfalls

### Pitfall 1: Windows Path Separator
**What goes wrong:** Using `/` in path construction instead of `filepath.Join`, producing `C:/Users/ken/.local/bin` on Windows where `\` is expected.
**Why it happens:** Copying from Unix path strings without using filepath.Join.
**How to avoid:** Always use `filepath.Join(home, ".local", "bin")` — already the pattern in `path.go`.
**Warning signs:** Test skips on Windows (`t.Skip("uses Unix PATH separator")`) suggest the existing tests only cover the Unix separator. The code itself is cross-platform via filepath.Join.

### Pitfall 2: AugmentServicePath called in daemon but not GUI
**What goes wrong:** The fix works in daemon-started sessions but not when the GUI creates sessions directly.
**Why it happens:** There are two call sites: `runGUI()` in `main.go` (line 57) and `runDaemonCore()` in `internal/daemon/process.go` (line 25). Both already call `AugmentServicePath()`.
**How to avoid:** The fix is in `AugmentServicePath()` itself — both call sites inherit the fix automatically. No action needed beyond the `path.go` edit.

### Pitfall 3: Sidebar test asserting "New Tab"
**What goes wrong:** Updating the label but forgetting to check whether existing tests assert the "New Tab" string.
**Why it happens:** Developers assume tests are testing structure, not string values.
**How to avoid:** The existing `Sidebar.test.tsx` does NOT assert "New Tab" anywhere (verified by inspection). The tests check `aria-label="Sessions"` and CSS class presence. Safe to rename.
**Warning signs:** Run `cd frontend && pnpm test` before and after the change to verify no regressions.

### Pitfall 4: Windows native installer path not verified at runtime
**What goes wrong:** The `~/.local/bin` path may not exist on a given Windows machine — `AugmentServicePath` correctly handles this (it checks `os.Stat` before adding).
**Why it happens:** Might worry the path check fails silently.
**How to avoid:** The existing `candidates` loop uses `os.Stat(dir)` to skip non-existent dirs. This is already the correct behavior — no action needed.

## Code Examples

Verified patterns from official sources (source: project codebase):

### AugmentServicePath with .local/bin added
```go
// internal/daemon/path.go
func AugmentServicePath() {
    home, err := os.UserHomeDir()
    if err != nil {
        return
    }

    candidates := []string{
        filepath.Join(home, ".local", "bin"),   // Anthropic native installer
        filepath.Join(home, ".volta", "bin"),
        "/opt/homebrew/bin",
        "/usr/local/bin",
        "/home/linuxbrew/.linuxbrew/bin",
        nvmActiveBin(home),
    }
    // ... rest unchanged
}
```

### Test for ~/.local/bin detection
```go
// internal/daemon/path_test.go (new test, follows existing pattern)
func TestAugmentServicePath_AddsLocalBin(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("uses Unix PATH separator")
    }
    home := t.TempDir()
    t.Setenv("HOME", home)
    original := "/usr/bin:/bin"
    t.Setenv("PATH", original)

    localBin := filepath.Join(home, ".local", "bin")
    if err := os.MkdirAll(localBin, 0755); err != nil {
        t.Fatalf("MkdirAll: %v", err)
    }

    AugmentServicePath()

    got := os.Getenv("PATH")
    if !strings.Contains(got, localBin) {
        t.Errorf("PATH should contain %s, got %s", localBin, got)
    }
    if !strings.HasSuffix(got, original) {
        t.Errorf("PATH should end with original %s, got %s", original, got)
    }
}
```

### Sidebar label test (new test case to add)
```tsx
// frontend/src/components/__tests__/Sidebar.test.tsx (new test in existing describe block)
it('renders "New Session" label and aria-label for the add button (UI-01)', () => {
  ;({ container, root } = renderSidebar())
  const addBtn = container.querySelector('button[aria-label="New Session"]')
  expect(addBtn).not.toBeNull()
  // Verify the visible label text
  const label = addBtn!.querySelector('.sidebar__label')
  expect(label).not.toBeNull()
  expect(label!.textContent).toBe('New Session')
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Label "New Tab" | Label "New Session" | Phase 57 | Clarifies that the action creates a terminal session, not a browser-style tab |

**No deprecated patterns in this phase.**

## Open Questions

1. **Windows ~/.local/bin installer path not runtime-verified**
   - What we know: Anthropic documents `%USERPROFILE%\.local\bin\claude.exe` as the native installer path; `os.UserHomeDir()` correctly returns `%USERPROFILE%` on Windows; `filepath.Join` uses `\` separator on Windows.
   - What's unclear: STATE.md acknowledges "Windows native installer path (%USERPROFILE%\.local\bin\claude.exe) not yet verified against actual Windows install." This cannot be resolved without a Windows machine with the native installer.
   - Recommendation: Implement the path fix as documented — the logic is provably correct from Go stdlib behavior. Flag in the plan for human UAT on Windows before final merge.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (backend) + Vitest ^4.1.0 (frontend) |
| Config file | `frontend/vitest.config.ts` (inferred from package.json), Go: none (standard) |
| Quick run command (backend) | `cd /Users/ken/dev/agenthub && go test ./internal/daemon/ -run TestAugmentServicePath -v` |
| Quick run command (frontend) | `cd /Users/ken/dev/agenthub/frontend && pnpm test -- --reporter=verbose` |
| Full suite command (backend) | `cd /Users/ken/dev/agenthub && go test ./...` |
| Full suite command (frontend) | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DET-01 | `AugmentServicePath` adds `~/.local/bin` when directory exists | unit (Go) | `go test ./internal/daemon/ -run TestAugmentServicePath_AddsLocalBin -v` | ❌ Wave 0 — add to `path_test.go` |
| UI-01 | Sidebar renders `aria-label="New Session"` and label text "New Session" | unit (React) | `cd frontend && pnpm test -- --reporter=verbose` | ❌ Wave 0 — add to `Sidebar.test.tsx` |

### Sampling Rate
- **Per task commit:** Backend: `go test ./internal/daemon/ -v` | Frontend: `cd frontend && pnpm test`
- **Per wave merge:** `go test ./...` + `cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/path_test.go` — add `TestAugmentServicePath_AddsLocalBin` covering DET-01
- [ ] `frontend/src/components/__tests__/Sidebar.test.tsx` — add test verifying `aria-label="New Session"` and `.sidebar__label` text "New Session" covering UI-01

## Sources

### Primary (HIGH confidence)
- Project source: `internal/daemon/path.go` — verified `AugmentServicePath` candidates list; `~/.local/bin` is absent
- Project source: `internal/pty/detect.go` — verified `DetectCLIs` uses `exec.LookPath` against process PATH
- Project source: `frontend/src/components/Sidebar.tsx` — verified "New Tab" appears as both `aria-label` (line 84) and `<span>` text (line 86)
- Project source: `frontend/src/components/__tests__/Sidebar.test.tsx` — verified no existing test asserts "New Tab" string
- Go stdlib docs: `os.UserHomeDir()` — returns `USERPROFILE` on Windows, `HOME` on Unix
- Go stdlib docs: `filepath.Join()` — uses OS-appropriate separator

### Secondary (MEDIUM confidence)
- STATE.md known blocker: "Phase 57 (DET-01): Windows native installer path (%USERPROFILE%\.local\bin\claude.exe) not yet verified against actual Windows install" — confirms the gap and path we are targeting

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all existing project patterns, no new dependencies
- Architecture: HIGH — both changes are minimal, well-precedented edits in established files
- Pitfalls: HIGH — verified by code inspection; Windows path gap acknowledged in STATE.md

**Research date:** 2026-04-08
**Valid until:** 2026-05-08 (stable domain — Go stdlib + React component labels)

## Project Constraints (from CLAUDE.md)

- Go: use `go fmt`, `golangci-lint`, context-aware functions
- JS/TS: `camelCase`, `PascalCase` components, ESLint + Prettier, TypeScript types
- Testing: Go `testing` + `pytest` (not applicable) + `vitest` — 80%+ coverage in critical components
- **NEVER install packages globally** (not applicable — no new packages)
- Use `pnpm` (not npm/yarn) for frontend
- Premature Abstraction rule: this phase makes targeted minimal edits — no abstraction warranted
- Chesterton's Fence: `AugmentServicePath` already augments PATH with similar user-local tool dirs; adding `~/.local/bin` follows the same pattern with clear justification (Anthropic native installer)
