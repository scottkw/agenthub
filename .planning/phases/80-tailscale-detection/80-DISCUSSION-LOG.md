# Phase 80: Tailscale Detection - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-16
**Phase:** 80-tailscale-detection
**Areas discussed:** Detection granularity, Platform-specific paths, Failure UX, Settings override

---

## Detection Granularity

### Q1: How granular should Tailscale health states be?

| Option | Description | Selected |
|--------|-------------|----------|
| 4-state model | Not installed → Installed (daemon stopped) → Running (disconnected) → Connected. Adds `DaemonUp` field. | ✓ |
| 3-state (current + daemon) | Keep current Installed/Connected but add daemon-not-running middle state. | |
| Keep current 2-state | Installed vs not-installed is enough. Focus on detection breadth, not state granularity. | |

**User's choice:** 4-state model
**Notes:** Full cascade: Not installed → Installed (daemon stopped) → Running (disconnected from tailnet) → Connected

### Q2: Single CheckHealth call or separate functions?

| Option | Description | Selected |
|--------|-------------|----------|
| Single call | CheckHealth tries binary detection → daemon socket → status. One call, one result. | ✓ |
| Separate functions | DetectTailscale() + CheckHealth() composed by caller. More flexible but more API surface. | |

**User's choice:** Single call
**Notes:** None

---

## Platform-Specific Paths

### Q3: How aggressively should we search for Tailscale when it's not on PATH?

| Option | Description | Selected |
|--------|-------------|----------|
| Well-known paths | Hardcoded list of known install locations per platform (macOS app bundle, Homebrew, Linux system/snap/flatpak, Windows Program Files + registry) | ✓ |
| PATH + daemon socket only | Don't search for binary — just try daemon socket. Binary path only matters for CLI. | |
| Filesystem walk | Search common program directories recursively. Thorough but slow. | |

**User's choice:** Well-known paths
**Notes:** Covers macOS (app bundle, Homebrew), Linux (system, snap, flatpak), Windows (Program Files, registry)

### Q4: Should daemon socket check probe multiple paths or trust the library?

| Option | Description | Selected |
|--------|-------------|----------|
| Library default | `tailscale.com/client/local` already knows the right socket per platform. Trust it. | ✓ |
| Multi-socket probe | Try multiple socket paths in case Snap/Flatpak use non-standard locations. | |

**User's choice:** Library default
**Notes:** None

---

## Failure UX

### Q5: How should the health modal present the 4-state information?

| Option | Description | Selected |
|--------|-------------|----------|
| Stepped checklist | Vertical checklist with green/yellow/red indicators. First failing step gets platform-specific instructions. Later steps grayed out. | ✓ |
| Single status line | One message describing current state and next action. | |
| You decide | Claude picks based on codebase patterns. | |

**User's choice:** Stepped checklist
**Notes:** None

### Q6: Should 'daemon stopped' state offer an action button or just instructions?

| Option | Description | Selected |
|--------|-------------|----------|
| Instructions only | Platform-specific text: macOS "Open Tailscale", Linux "sudo systemctl start tailscaled", Windows "Open from Start menu" | ✓ |
| Action button per platform | Launch app or service programmatically. Harder cross-platform. | |
| macOS button only | Only macOS gets action button (open -a Tailscale). Others get text. | |

**User's choice:** Instructions only
**Notes:** No action buttons for starting the daemon — text instructions per platform

---

## Settings Override

### Q7: Should Settings have a manual Tailscale path override?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, with auto-detect default | Empty = auto-detect. Custom path used first, fallback to auto-detect. Follows Phase 79 pattern. | ✓ |
| No override needed | Well-known paths cover 95%+. Users can modify system PATH. | |
| Override + validation | Same as option 1 but validate the binary exists when path is set. | |

**User's choice:** Yes, with auto-detect default
**Notes:** Follows Phase 79 path persistence + browse button pattern

---

## Claude's Discretion

- Exact order of well-known path checking
- Whether to cache detected binary path
- Internal error handling and timeout strategy
- Platform-specific code structure (build tags vs runtime check)

## Deferred Ideas

None — discussion stayed within phase scope
