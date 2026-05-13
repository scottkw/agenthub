package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/pty"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/status"
)

// SessionEngine owns all session state: registry, backend, hub manager,
// tab names, CLI path overrides, and per-session status.
// It is intentionally free of Wails imports — callers supply an onStatus
// callback if they need event emission.
type SessionEngine struct {
	hostname          string // machine hostname, captured at startup
	opencodeTUIConfig string // path to managed opencode-tui.json (set at init)
	configDir         string // cached config dir for settings persistence

	registry *pty.SessionRegistry
	backend  pty.SessionBackend
	manager  *relay.HubManager

	mu          sync.RWMutex
	tabNames    map[string]string // sessionID -> display name
	sessionCLIs map[string]string // sessionID -> raw CLI name (e.g. "opencode")
	cliPaths    map[string]string // cli name -> custom path override

	startMinimized      bool           // persisted start-minimized preference
	shellWebShareWarned bool           // Phase 101 SHELL-08: user has acknowledged the shell web-share security banner
	shellPath           string         // Phase 107 SHELL-11: user-configured shell binary path; empty = use platform default
	autoCloseSession    *bool          // nil = default (true); persisted pointer
	pluginSettings      PluginSettings // populated by loadSettingsFromDisk via defaults-merge

	// pluginSettingsListener (if non-nil) is invoked synchronously by
	// SetPluginSettings AFTER the new value is persisted, while the engine
	// mutex is still held. Phase 93 PLUG-04 — webserver registers
	// BroadcastPluginConfig here so SSE subscribers receive a frame on every
	// change. The listener MUST be non-blocking (BroadcastPluginConfig uses
	// non-blocking channel sends with drop-on-slow-consumer).
	pluginSettingsListener func()

	statusMu        sync.RWMutex
	sessionStatuses map[string]status.SessionStatus // sessionID -> current status
}

// daemonConfigDir returns ~/.config/agenthub/, creating it if needed.
// Mirrors app.go configDir() — internal packages cannot import main.
func daemonConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "agenthub")
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// ensureOpenCodeTUIConfig writes a managed tui.json that forces OpenCode's
// "system" theme. The system theme uses ANSI palette colors (0-15), making
// OpenCode respect xterm.js theme remapping. The file is overwritten on every
// call (content is a hardcoded constant, no user data). Returns the file path.
func ensureOpenCodeTUIConfig(dir string) string {
	path := filepath.Join(dir, "opencode-tui.json")
	content := []byte("{\"$schema\":\"https://opencode.ai/tui.json\",\"theme\":\"system\"}\n")
	_ = os.WriteFile(path, content, 0644)
	return path
}

// daemonSettings is the persisted settings structure.
//
// Plugins and SchemaVersion are intentionally NOT tagged `omitempty`:
// the defaults-merge load (Pitfall #14 mitigation) requires the plugins
// block + schemaVersion to always serialize so future loads observe them
// even when every plugin is at its zero value (all-false).
type daemonSettings struct {
	CLIPaths            map[string]string `json:"cliPaths,omitempty"`
	StartMinimized      bool              `json:"startMinimized,omitempty"`
	ShellWebShareWarned bool              `json:"shellWebShareWarned,omitempty"`
	ShellPath           string            `json:"shellPath,omitempty"`
	AutoCloseSession    *bool             `json:"autoCloseSession,omitempty"`
	Plugins             PluginSettings    `json:"plugins"`
	SchemaVersion       int               `json:"schemaVersion"`
}

// settingsPath returns the path to settings.json inside the config dir.
func settingsPath(dir string) string {
	return filepath.Join(dir, "settings.json")
}

// knownShells maps basenames of common shell interpreters. A stored CLI path
// whose basename is a shell and whose key is NOT that shell is almost certainly
// stale/wrong (e.g. "claude" → "/bin/sh") and should be discarded on load.
//
// Phase 100-02 (Pitfall 6 mitigation): pwsh/powershell + .exe forms added so
// stale "claude → pwsh.exe" overrides are filtered on load. On Windows
// filepath.Base("/path/to/pwsh.exe") yields "pwsh.exe" (not "pwsh"), so both
// bare and .exe basenames must be present.
var knownShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "fish": true,
	"csh": true, "tcsh": true, "dash": true, "ksh": true,
	"pwsh": true, "pwsh.exe": true,
	"powershell": true, "powershell.exe": true,
}

// isShellSession returns true if cli refers to a shell-type session (vs an
// AI CLI). Used by CreateSession to (1) bypass status.Watch (SHELL-09) and
// (2) gate the shell-specific argv / WorkDir resolution branch (SHELL-05).
func isShellSession(cli string) bool {
	switch cli {
	case "shell", "bash", "zsh", "pwsh", "powershell":
		return true
	}
	return false
}

// loadSettingsFromDisk reads settings.json and populates engine state.
// Missing file is not an error (first run).
//
// Defaults-merge: the daemonSettings literal is pre-populated with
// CurrentSchemaVersion + defaultPluginSettings() BEFORE Unmarshal so
// that v3.1 settings.json files (no plugins key, no schemaVersion key)
// round-trip with v3.2 defaults instead of zero-value (all-false) plugins.
// Pitfall #14 mitigation.
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
	data, err := os.ReadFile(settingsPath(dir))
	if err != nil {
		return // file not found or unreadable — not an error
	}
	// Pre-populate defaults BEFORE Unmarshal. Go stdlib leaves missing JSON
	// keys untouched, so v3.1 files (no plugins key) inherit defaultPluginSettings()
	// while v3.2+ files (with explicit plugins block) overwrite the defaults
	// with user choices.
	//
	// SchemaVersion intentionally NOT pre-populated: it must remain 0 for a
	// v3.1 file (no schemaVersion key) so the needsUpgradeWrite check below
	// correctly detects that an upgrade re-write is required.
	s := daemonSettings{
		Plugins: defaultPluginSettings(),
	}
	if json.Unmarshal(data, &s) != nil {
		return
	}
	e.mu.Lock()
	dirty := false
	if s.CLIPaths != nil {
		for k, v := range s.CLIPaths {
			base := filepath.Base(v)
			if knownShells[base] && base != k {
				log.Printf("daemon: dropping stale CLI path override %q=%q (shell mismatch)", k, v)
				delete(s.CLIPaths, k)
				dirty = true
				continue
			}
			e.cliPaths[k] = v
		}
	}
	e.startMinimized = s.StartMinimized
	e.shellWebShareWarned = s.ShellWebShareWarned
	e.shellPath = s.ShellPath
	e.autoCloseSession = s.AutoCloseSession
	e.pluginSettings = s.Plugins
	// Detect upgrade-path: the on-disk schemaVersion was below
	// CurrentSchemaVersion (e.g. v3.1 file with no key → 0). Re-save so
	// the next load observes the populated plugins block + schemaVersion.
	// Idempotent: on second start s.SchemaVersion == CurrentSchemaVersion
	// and this branch does not fire.
	needsUpgradeWrite := s.SchemaVersion < CurrentSchemaVersion
	if dirty || needsUpgradeWrite {
		// Rewrite settings.json without stale entries / with new schema.
		// saveSettingsToDisk runs INSIDE e.mu.Lock() per its contract
		// ("Caller holds e.mu.Lock()."). Mirrors the existing
		// SetStartMinimized pattern.
		e.saveSettingsToDisk()
	}
	e.mu.Unlock()
}

// saveSettingsToDisk writes current settings to settings.json.
// Caller holds e.mu.Lock().
func (e *SessionEngine) saveSettingsToDisk() {
	s := daemonSettings{
		CLIPaths:            e.cliPaths,
		StartMinimized:      e.startMinimized,
		ShellWebShareWarned: e.shellWebShareWarned,
		ShellPath:           e.shellPath,
		AutoCloseSession:    e.autoCloseSession,
		Plugins:             e.pluginSettings,
		SchemaVersion:       CurrentSchemaVersion,
	}
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.WriteFile(settingsPath(e.configDir), data, 0600)
}

// NewSessionEngine creates a SessionEngine with all subsystems initialised.
func NewSessionEngine() *SessionEngine {
	hostname, _ := os.Hostname()
	cfgDir := daemonConfigDir()
	tuiConfig := ensureOpenCodeTUIConfig(cfgDir)
	e := &SessionEngine{
		hostname:          hostname,
		configDir:         cfgDir,
		opencodeTUIConfig: tuiConfig,
		registry:          pty.NewSessionRegistry(),
		backend:           pty.NewNativePTYBackend(),
		manager:           relay.NewHubManager(),
		tabNames:          make(map[string]string),
		sessionCLIs:       make(map[string]string),
		cliPaths:          make(map[string]string),
		sessionStatuses:   make(map[string]status.SessionStatus),
	}
	e.loadSettingsFromDisk(cfgDir)
	return e
}

// CreateSession spawns a new CLI session and returns its ID.
// args are passed to the CLI process; pass nil if no extra arguments are needed.
// onStatus is called on each status transition; pass nil if not needed.
// onExit is called with (sessionID, exitCode) when the process exits naturally; pass nil if not needed.
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, args []string, cols, rows int, onStatus func(string, status.SessionStatus), onExit func(string, int)) (string, error) {
	cliPath := e.ResolveCLI(cli)

	// Phase 100 SHELL-05: shell-session dispatch. For cli in
	// {shell, bash, zsh, pwsh, powershell}, replace cliPath + args with the
	// resolved shell path + non-login interactive argv from Plan 01's
	// pty.DiscoverShells() / pty.KnownShellSpecs(). Empty workDir for shells
	// resolves to $HOME (Pitfall 4 mitigation) — AI CLI sessions retain
	// existing behavior (no $HOME substitution; empty WorkDir falls through
	// to backend / go-pty default).
	//
	// req.Args (caller-supplied) is INTENTIONALLY IGNORED for shells in
	// Phase 100 (RESEARCH Anti-Pattern + Assumption A6, T-100-08 mitigation):
	// we overwrite, not merge. AI CLI sessions continue to pass req.Args
	// through unchanged.
	if path, shellArgs, isShell := e.resolveShellSpawn(cli); isShell {
		cliPath = path
		args = shellArgs
		if workDir == "" {
			// WR-01: validate $HOME against known-useless values. On
			// service-mode daemons (RESEARCH.md Pitfall 4) $HOME is often
			// "/" or "."; writing shell history at those locations would
			// either fail (read-only root) or pollute the filesystem root.
			// The corresponding test (TestCreateSession_ShellEmptyWorkDirHome)
			// already skips on these values — production code must apply
			// the same guard.
			if home, err := os.UserHomeDir(); err == nil && home != "" && home != "/" && home != "." {
				workDir = home
			}
			// If UserHomeDir fails, returns empty, or returns an unreliable
			// value ("/", "."), fall through with the original empty workDir
			// (matches the AI-CLI path's behavior on missing $HOME).
		}
	}

	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	// Per-agent environment configuration.
	var env []string
	if cli == "opencode" && e.opencodeTUIConfig != "" {
		env = append(env, "OPENCODE_TUI_CONFIG="+e.opencodeTUIConfig)
	}

	sess, err := e.backend.Create(ctx, pty.CreateRequest{
		CLI:     cliPath,
		Args:    args,
		Env:     env,
		Cols:    cols,
		Rows:    rows,
		WorkDir: workDir,
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	e.registry.Add(sess)

	id := sess.ID
	resizeFn := func(cols, rows int) error {
		return e.backend.Resize(id, cols, rows)
	}
	hub := e.manager.Create(id, sess, sess, resizeFn)

	e.mu.Lock()
	e.tabNames[id] = name
	e.sessionCLIs[id] = cli // raw CLI name, NOT cliPath
	e.mu.Unlock()

	// SHELL-09: shell sessions have no AI-agent state model. Skip status.Watch
	// so sessionStatuses[id] stays empty and ListSessions falls through to its
	// conservative "running" default (see engine.go ListSessions branch). The
	// session State (running/stopped) transitions via the natural-exit
	// goroutine below — unchanged from AI-CLI behavior.
	if !isShellSession(cli) {
		go status.Watch(hub, id, cli, func(sid string, s status.SessionStatus) {
			e.statusMu.Lock()
			e.sessionStatuses[sid] = s
			e.statusMu.Unlock()
			if onStatus != nil {
				onStatus(sid, s)
			}
		})
	}

	// Watch for natural process exit (PTY EOF -> hub.Done closes).
	// Transitions session state to StateStopped and captures exit code.
	// Calls onExit callback if provided (used by API layer for web grace period per D-12).
	//
	// Does NOT call cmd.Wait() — killSession may be running concurrently and
	// go-pty's cmd.Wait() is not safe for concurrent callers. Instead, reads
	// the cached exit code from ExitCode() which checks ProcessState under mutex.
	go func() {
		<-hub.Done() // blocks until PTY read loop returns EOF (D-07)
		if sess.IsKilled() {
			return // killSession handles state transition
		}
		// Natural exit: cancel context so go-pty's waitOnContext reaps the
		// process and populates ProcessState.
		sess.CancelContext()
		// Brief wait for go-pty to populate ProcessState.
		time.Sleep(100 * time.Millisecond)
		exitCode := sess.ExitCode()
		if exitCode == -1 {
			exitCode = 0 // conservative default per D-10
		}
		sess.SetState(pty.StateStopped)
		if onExit != nil {
			onExit(id, exitCode)
		}
	}()

	return id, nil
}

// ListSessions returns a snapshot of all registered sessions.
func (e *SessionEngine) ListSessions() []SessionInfo {
	sessions := e.registry.List()
	result := make([]SessionInfo, 0, len(sessions))

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, s := range sessions {
		state := "running"
		if s.GetState() == pty.StateStopped {
			state = "stopped"
		}
		name := e.tabNames[s.ID]

		// MC-04: populate viewer count from hub subscriber count.
		// manager.Get acquires HubManager.mu; SubscriberCount acquires hub.mu.
		// Both are safe to call while holding e.mu.RLock (no lock ordering conflict).
		viewerCount := 0
		if hub, ok := e.manager.Get(s.ID); ok {
			viewerCount = hub.SubscriberCount()
		}

		// Heuristic status from detector (running/idle/waiting/errored).
		heuristicStatus := string(status.StatusRunning) // conservative default
		e.statusMu.RLock()
		if hs, ok := e.sessionStatuses[s.ID]; ok {
			heuristicStatus = string(hs)
		}
		e.statusMu.RUnlock()

		var exitCodePtr *int
		var durationPtr *int
		if state == "stopped" && !s.IsKilled() {
			// Only read exit code for natural exits. Killed sessions have
			// cmd.Wait() still running in killSession — reading ProcessState
			// would race with go-pty's internal write.
			ec := s.ExitCode()
			if ec == -1 {
				ec = 0 // SHELL-12: mirror the natural-exit goroutine's -1→0 normalization so GUI consumers never see PTY-EOF as an error code.
			}
			exitCodePtr = &ec
			dur := int(time.Since(s.CreatedAt).Seconds())
			durationPtr = &dur
		}

		result = append(result, SessionInfo{
			ID:          s.ID,
			CLI:         s.CLI,
			Name:        name,
			State:       state,
			Status:      heuristicStatus,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Hostname:    e.hostname,
			ViewerCount: viewerCount,
			ExitCode:    exitCodePtr,
			Duration:    durationPtr,
		})
	}
	return result
}

// RenameSession updates the display name of a session.
func (e *SessionEngine) RenameSession(id, name string) error {
	if _, ok := e.registry.Get(id); !ok {
		return fmt.Errorf("session %q not found", id)
	}
	e.mu.Lock()
	e.tabNames[id] = name
	e.mu.Unlock()
	return nil
}

// KillSession terminates the session and removes it from all registries.
func (e *SessionEngine) KillSession(id string) error {
	if err := e.backend.Kill(id); err != nil {
		return fmt.Errorf("kill session: %w", err)
	}
	e.manager.Remove(id)
	e.registry.Remove(id)

	e.mu.Lock()
	delete(e.tabNames, id)
	delete(e.sessionCLIs, id)
	e.mu.Unlock()

	e.statusMu.Lock()
	delete(e.sessionStatuses, id)
	e.statusMu.Unlock()

	return nil
}

// GetSessionStatus returns the current heuristic status of the session.
// Returns "running" if the session is not found (conservative default).
func (e *SessionEngine) GetSessionStatus(sessionID string) string {
	e.statusMu.RLock()
	s, ok := e.sessionStatuses[sessionID]
	e.statusMu.RUnlock()
	if !ok {
		return string(status.StatusRunning)
	}
	return string(s)
}

// ResolveCLI returns the executable path for the named CLI.
// Checks custom overrides first, then returns the name as-is.
func (e *SessionEngine) ResolveCLI(name string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if path, ok := e.cliPaths[name]; ok {
		return path
	}
	return name
}

// DiscoverShells is a thin engine-mediated wrapper around pty.DiscoverShells.
// IN-03 / PATTERNS.md line 508: route the GET /shells handler through the
// engine so HTTP tests can DI a fake without coupling to the filesystem,
// and so future engine-level caching or rate-limiting can be applied
// without restructuring callers. The wrapper is currently a passthrough.
func (e *SessionEngine) DiscoverShells() []pty.DetectedShell {
	return pty.DiscoverShells()
}

// resolveShellSpawn maps an abstract shell name (shell|bash|zsh|pwsh|powershell)
// to (absolute path, interactive non-login argv, ok=true). Returns
// ("", nil, false) when cli is not a shell name.
//
// Resolution order:
//  1. Caller's settings.json override (e.cliPaths[cli]) — basename is matched
//     against pty.KnownShellSpecs() to derive the argv. Per Plan 01 M2,
//     "powershell" is a first-class knownShellSpec alongside "pwsh", so a
//     cliPaths["powershell"] override resolves cleanly via this branch without
//     falling through to discovery.
//  2. Live discovery via pty.DiscoverShells() — returns the first entry whose
//     Name matches cli (which includes the synthetic "shell" system-default
//     entry on POSIX).
//  3. Safety net for cli="pwsh" on legacy Windows hosts: if only powershell.exe
//     is discoverable, accept the "powershell" spec entry.
//
// Locking: e.cliPaths is read under e.mu.RLock via ResolveCLI; pty.DiscoverShells
// does filesystem I/O via exec.LookPath and must run outside any mutex (it is
// already lock-free).
func (e *SessionEngine) resolveShellSpawn(cli string) (string, []string, bool) {
	if !isShellSession(cli) {
		return "", nil, false
	}

	// (0) Phase 107 SHELL-11: shellPath setting override for the bare "shell"
	// key. This fires ONLY when:
	//   - cli == "shell" (the generic system-default key; per-binary keys like
	//     "bash"/"zsh" fall through to branch (1) unchanged — cliPaths["bash"]
	//     overrides continue to take precedence over the catch-all shellPath).
	//   - cliPaths["shell"] is NOT set (branch (1) wins if both are set,
	//     preserving per-binary cliPaths[bash] overrides).
	//   - e.shellPath is non-empty (user has configured a specific binary).
	//
	// When the basename of e.shellPath matches a known shell spec, borrow that
	// spec's argv (e.g. /opt/homebrew/bin/zsh → zsh spec → ["-i"]). If the
	// basename is unrecognised, fall through to branch (1) and let discovery
	// handle it — the binary may be a custom shell that still accepts -i.
	e.mu.RLock()
	settingsShellPath := e.shellPath
	_, cliPathSet := e.cliPaths["shell"]
	e.mu.RUnlock()
	if cli == "shell" && !cliPathSet && settingsShellPath != "" {
		baseNoExt := strings.TrimSuffix(filepath.Base(settingsShellPath), ".exe")
		for _, spec := range pty.KnownShellSpecs() {
			if spec.Name == baseNoExt {
				argv := append([]string(nil), spec.Argv...)
				return settingsShellPath, argv, true
			}
		}
		// Basename not in KnownShellSpecs — fall through to discovery.
		// Do NOT error; the user may have set a custom shell binary.
	}

	// (1) Settings override path. ResolveCLI returns the override if present,
	// otherwise the bare cli name.
	override := e.ResolveCLI(cli)
	if override != cli {
		// IN-02: match on cli or the override path's basename with any
		// ".exe" suffix stripped. The prior implementation also compared
		// against the un-trimmed basename, but that branch was redundant —
		// none of knownShellSpecs (bash/zsh/pwsh/powershell) have names
		// ending in ".exe", so any spec.Name match against a ".exe"-suffixed
		// basename must come from the trimmed form.
		baseNoExt := strings.TrimSuffix(filepath.Base(override), ".exe")
		for _, spec := range pty.KnownShellSpecs() {
			if spec.Name == cli || spec.Name == baseNoExt {
				argv := append([]string(nil), spec.Argv...)
				return override, argv, true
			}
		}
		// Override basename doesn't match any known shell — fall through to
		// discovery (caller may have set an override to a path that does not
		// look like a known shell binary).
	}

	// (2) Live discovery. WR-02: cache the result so the legacy-Windows
	// safety net below reuses the same scan rather than re-running PATH
	// lookups + /etc/shells read. Beyond the I/O savings, this also
	// guarantees the two passes agree under PATH-mutating tests
	// (t.Setenv("PATH", ...) between scans could otherwise produce
	// inconsistent results).
	discovered := pty.DiscoverShells()
	for _, sh := range discovered {
		if sh.Name == cli {
			argv := append([]string(nil), sh.Argv...)
			return sh.Path, argv, true
		}
	}

	// (3) Legacy Windows safety net: cli="pwsh" but only "powershell" is
	// discoverable (Windows host with only PowerShell 5.x installed).
	if cli == "pwsh" {
		for _, sh := range discovered {
			if sh.Name == "powershell" {
				argv := append([]string(nil), sh.Argv...)
				return sh.Path, argv, true
			}
		}
	}

	return "", nil, false
}

// UpdateCLIPath stores a custom executable path for the named CLI.
// The path must exist on disk. Persists to settings.json.
func (e *SessionEngine) UpdateCLIPath(name, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("custom CLI path %q: %w", path, err)
	}
	e.mu.Lock()
	e.cliPaths[name] = path
	e.saveSettingsToDisk()
	e.mu.Unlock()
	return nil
}

// GetCLIPaths returns a copy of the current CLI path overrides map.
func (e *SessionEngine) GetCLIPaths() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]string, len(e.cliPaths))
	for k, v := range e.cliPaths {
		out[k] = v
	}
	return out
}

// GetStartMinimized returns the persisted start-minimized preference.
func (e *SessionEngine) GetStartMinimized() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.startMinimized
}

// SetStartMinimized updates and persists the start-minimized preference.
func (e *SessionEngine) SetStartMinimized(val bool) {
	e.mu.Lock()
	e.startMinimized = val
	e.saveSettingsToDisk()
	e.mu.Unlock()
}

// GetShellWebShareWarned returns the persisted "user has acknowledged the
// shell web-share security banner" flag. Phase 101 SHELL-08.
func (e *SessionEngine) GetShellWebShareWarned() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.shellWebShareWarned
}

// SetShellWebShareWarned persists the shell-web-share-warned flag.
// Returns an error path for symmetry with future-proofed callers; the
// underlying saveSettingsToDisk swallows IO errors today, so this currently
// always returns nil.
func (e *SessionEngine) SetShellWebShareWarned(val bool) error {
	e.mu.Lock()
	e.shellWebShareWarned = val
	e.saveSettingsToDisk()
	e.mu.Unlock()
	return nil
}

// resolveDefaultShellPath returns a non-empty shell path to use when no
// explicit shellPath setting has been configured. Resolution order:
//  1. $SHELL environment variable (POSIX).
//  2. pty.DiscoverShells() — first entry whose Name is "shell" (synthetic
//     system-default entry). This covers POSIX hosts where $SHELL may be unset
//     but a real shell was discovered.
//  3. Platform hard-fallback: /bin/zsh (darwin), /bin/bash (linux),
//     pwsh.exe (windows).
//
// NEVER returns an empty string. Phase 107 SHELL-11.
func resolveDefaultShellPath() string {
	// (1) $SHELL env var.
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	// (2) DiscoverShells — look for the synthetic "shell" system-default entry.
	for _, sh := range pty.DiscoverShells() {
		if sh.Name == "shell" {
			return sh.Path
		}
	}
	// (3) Platform hard-fallback.
	switch runtime.GOOS {
	case "windows":
		return "pwsh.exe"
	case "linux":
		return "/bin/bash"
	default: // darwin and others
		return "/bin/zsh"
	}
}

// GetShellPath returns the persisted shell binary path. When no path has been
// set by the user (shellPath is empty), it resolves and returns the platform
// default via resolveDefaultShellPath(). NEVER returns an empty string.
// Phase 107 SHELL-11.
func (e *SessionEngine) GetShellPath() string {
	e.mu.RLock()
	path := e.shellPath
	e.mu.RUnlock()
	if path == "" {
		return resolveDefaultShellPath()
	}
	return path
}

// SetShellPath updates and persists the shell binary path override.
// When path is empty, the override is cleared (subsequent GetShellPath calls
// will return the platform default). When path is non-empty, it must point to
// an existing executable file; otherwise an error is returned and the field is
// left unchanged. Phase 107 SHELL-11.
func (e *SessionEngine) SetShellPath(path string) error {
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("path %q does not exist or is not executable", path)
		}
		if info.IsDir() {
			return fmt.Errorf("path %q is a directory, not an executable", path)
		}
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("path %q does not exist or is not executable", path)
		}
	}
	e.mu.Lock()
	e.shellPath = path
	e.saveSettingsToDisk()
	e.mu.Unlock()
	return nil
}

// GetAutoCloseSession returns the auto-close-on-exit preference.
// Returns true when the setting is absent (nil pointer = default enabled per D-11).
func (e *SessionEngine) GetAutoCloseSession() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.autoCloseSession == nil {
		return true // default: enabled
	}
	return *e.autoCloseSession
}

// SetAutoCloseSession updates and persists the auto-close-on-exit preference.
func (e *SessionEngine) SetAutoCloseSession(val bool) {
	e.mu.Lock()
	e.autoCloseSession = &val
	e.saveSettingsToDisk()
	e.mu.Unlock()
}

// GetPluginSettings returns the current plugin enable/disable preferences.
func (e *SessionEngine) GetPluginSettings() PluginSettings {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.pluginSettings
}

// SetPluginSettings updates and persists the plugin enable/disable preferences.
// Settings are immediately written to disk while the engine mutex is held
// (saveSettingsToDisk's contract requires the caller to hold e.mu.Lock()).
//
// Phase 93 PLUG-04: after persistence, invoke pluginSettingsListener (if set)
// outside the engine mutex so a slow listener cannot deadlock with concurrent
// engine operations. The listener is the SSE BroadcastPluginConfig hook.
func (e *SessionEngine) SetPluginSettings(s PluginSettings) {
	e.mu.Lock()
	e.pluginSettings = s
	e.saveSettingsToDisk()
	listener := e.pluginSettingsListener
	e.mu.Unlock()
	if listener != nil {
		listener()
	}
}

// SetSearchConfig updates and persists ONLY the SearchConfig sub-key of
// PluginSettings, leaving the rest of PluginSettings (WebGL, Unicode11,
// Search, WebLinks, Image, Serialize, Clipboard, Progress) untouched.
//
// Phase 94-07 WR-03 (gap closure) — handleSearchOptionsChange used to call
// SetPluginSettings with a full PluginSettings constructed from the
// App-level prop, racing PluginsSection's stale local edit buffer.
// SetSearchConfig is the surgical sub-key writer that preserves
// PluginsSection's edit buffer semantics: a find-bar toggle change can no
// longer overwrite an in-flight Plugins-tab boolean edit.
//
// Concurrency / persistence contract is identical to SetPluginSettings
// (mutate under e.mu.Lock(), saveSettingsToDisk while held, capture and
// invoke listener after release). The Phase 93 PLUG-04 SSE
// pluginSettingsListener is invoked so /api/plugin-config/stream
// subscribers (web terminal) receive a frame on every search-option
// change — preserving SRC-05 web parity (94-04 behavior unchanged).
func (e *SessionEngine) SetSearchConfig(cfg SearchConfig) {
	e.mu.Lock()
	e.pluginSettings.SearchConfig = cfg
	e.saveSettingsToDisk()
	listener := e.pluginSettingsListener
	e.mu.Unlock()
	if listener != nil {
		listener()
	}
}

// SetWebLinksConfig updates and persists ONLY the WebLinksConfig sub-key
// of PluginSettings, leaving the rest of PluginSettings (WebGL, Unicode11,
// Search, SearchConfig, WebLinks boolean, Image, Serialize, Clipboard,
// Progress) untouched.
//
// Phase 95 LNK-05 / LNK-06 — mirrors Phase 94 Plan 07's SetSearchConfig
// sub-key writer verbatim. Concurrency / persistence contract is identical
// to SetPluginSettings: mutate under e.mu.Lock(), saveSettingsToDisk while
// held, capture and invoke listener after release. The Phase 93 PLUG-04
// pluginSettingsListener is invoked so /api/plugin-config/stream
// subscribers (web terminal) receive a frame on every WebLinksConfig
// change — preserving live-toggle web parity (Plan 95-06 wires the
// SSE-driven hot-swap arm in terminal.js).
//
// The sub-key writer is callable from v3.2 (this plan ships the path) but
// the in-app UI for editing the sub-fields ships in Phase 99 / PUI-03;
// until then the boolean WebLinks toggle in PluginsSection routes through
// the full SetPluginSettings path.
func (e *SessionEngine) SetWebLinksConfig(cfg WebLinksConfig) {
	e.mu.Lock()
	e.pluginSettings.WebLinksConfig = cfg
	e.saveSettingsToDisk()
	listener := e.pluginSettingsListener
	e.mu.Unlock()
	if listener != nil {
		listener()
	}
}

// SetImageConfig updates and persists ONLY the ImageConfig sub-key of
// PluginSettings, leaving the rest of PluginSettings (WebGL, Unicode11,
// Search, SearchConfig, WebLinks, WebLinksConfig, Image bool, Serialize,
// Clipboard, Progress) untouched.
//
// Phase 96 IMG-02 — mirrors Phase 95 SetWebLinksConfig and Phase 94-07
// SetSearchConfig sub-key writers verbatim. Concurrency contract:
// mutate under e.mu.Lock(); saveSettingsToDisk while held; capture
// listener; release lock; invoke listener after release (avoids
// re-entrancy deadlock if listener calls back into the engine).
//
// Note on next-session-only semantics: the listener fires (so web SSE
// consumers receive the frame and downstream UIs can update their
// internal pluginConfig prop), but the desktop TerminalPanel hot-swap
// useEffect intentionally does NOT include `imageConfig` in its dep
// array — only newly-mounted sessions pick up the new StorageLimit
// (per ROADMAP IMG-01 italic caption affordance).
func (e *SessionEngine) SetImageConfig(cfg ImageConfig) {
	e.mu.Lock()
	e.pluginSettings.ImageConfig = cfg
	e.saveSettingsToDisk()
	listener := e.pluginSettingsListener
	e.mu.Unlock()
	if listener != nil {
		listener()
	}
}

// SetPluginSettingsListener registers a callback invoked synchronously by
// SetPluginSettings AFTER the new value is persisted. Phase 93 PLUG-04 push
// channel — webserver registers BroadcastPluginConfig here so SSE subscribers
// receive a frame on every change.
//
// Single-listener slot: the two NewWebServer call sites in api.go are mutually
// exclusive at runtime (one for AutoStartWebServer / Tailscale-mode, one for
// handleWebServerStart / mode-switch), so a single slot is safe.
func (e *SessionEngine) SetPluginSettingsListener(fn func()) {
	e.mu.Lock()
	e.pluginSettingsListener = fn
	e.mu.Unlock()
}

// Manager returns the HubManager (needed by webserver and relay server).
func (e *SessionEngine) Manager() *relay.HubManager {
	return e.manager
}

// Registry returns the session registry (needed for Get lookups).
func (e *SessionEngine) Registry() *pty.SessionRegistry {
	return e.registry
}

// Backend returns the session backend (needed for resize in relay).
func (e *SessionEngine) Backend() pty.SessionBackend {
	return e.backend
}

// NotifyThemeChange signals all active OpenCode sessions to re-query the
// terminal palette. On POSIX this sends SIGUSR2; on Windows this is a no-op.
// Errors on individual sessions are logged and do not abort the broadcast.
// Safe to call when no opencode sessions exist (returns nil).
func (e *SessionEngine) NotifyThemeChange(ctx context.Context) error {
	sessions := e.registry.List()

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, sess := range sessions {
		if e.sessionCLIs[sess.ID] != "opencode" {
			continue
		}
		if sess.GetState() != pty.StateRunning {
			continue
		}
		if err := signalThemeChange(sess); err != nil {
			log.Printf("[warn] NotifyThemeChange: session %s: %v", sess.ID, err)
		}
	}
	return nil
}
