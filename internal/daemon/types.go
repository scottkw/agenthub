package daemon

import (
	"github.com/scottkw/agenthub/internal/files"
)

// FileEntry is the per-entry wire type returned by the file-browser routes
// (GET /api/files/list, /api/files/stat). Re-exported as an alias from
// internal/files so DaemonClient consumers (Phase 120 GUI, Phase 121 TUI)
// import only this package.
// Phase 118 / FS-03 / FS-04.
type FileEntry = files.FileEntry

// FileListResponse is the wire envelope for GET /api/files/list.
// Re-exported alias of files.FileListResponse — same rationale as FileEntry.
// Phase 118 / FS-03.
type FileListResponse = files.FileListResponse

// SessionInfo is the JSON-serialisable representation of a session.
type SessionInfo struct {
	ID          string `json:"id"`
	CLI         string `json:"cli"`
	Name        string `json:"name"`
	State       string `json:"state"`
	Status      string `json:"status"` // heuristic status: running/idle/waiting/errored
	CreatedAt   string `json:"createdAt"`
	Hostname    string `json:"hostname"`
	WebEnabled  bool   `json:"webEnabled"`
	ViewerCount int    `json:"viewerCount"`        // MC-04: number of active WebSocket subscribers
	ExitCode    *int   `json:"exitCode,omitempty"` // nil while running; set when State is "stopped"
	Duration    *int   `json:"duration,omitempty"` // seconds since CreatedAt; set when State is "stopped"
	HomeDir       bool   `json:"homeDir"`         // Phase 124 / CAP-06: true when the session cwd equals EvalSymlinks($HOME); drives the home-write warning on both GUI and TUI
	BrowseEnabled bool   `json:"browseEnabled"`   // Phase 137 / SHARE-05: true when per-session browse toggle is ON; NOT omitempty (false must serialize so modal can seed on open)
	WorkDir       string `json:"workDir"`         // Phase 131 / GRID-02: EvalSymlinks-resolved session working directory; populated from engine.sessionWorkDirs map; enables Hub card grouping by directory
}

// CreateRequest is the request body for POST /sessions.
type CreateRequest struct {
	CLI     string   `json:"cli"`
	Name    string   `json:"name"`
	WorkDir string   `json:"workDir"`
	Args    []string `json:"args,omitempty"`
	Cols    int      `json:"cols,omitempty"`
	Rows    int      `json:"rows,omitempty"`
}

// CreateResponse is the response body for POST /sessions.
type CreateResponse struct {
	ID string `json:"id"`
}

// RenameRequest is the request body for PATCH /sessions/{id}/name.
type RenameRequest struct {
	Name string `json:"name"`
}

// StatusResponse is the response body for GET /sessions/{id}/status.
type StatusResponse struct {
	Status string `json:"status"`
}

// TailLinesResponse is the response body for GET /sessions/{id}/tail.
// Phase 132 / CARD-07.
type TailLinesResponse struct {
	Lines []string `json:"lines"`
}

// HealthResponse is the response body for GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// CLIPathsResponse maps CLI name to custom path override.
type CLIPathsResponse map[string]string

// DetectedShell is the JSON-serialisable representation of a discovered shell.
// Mirrors internal/pty.DetectedShell; duplicated to keep daemon wire types
// decoupled from internal/pty's Go API (see SessionInfo above — wire types are
// not pty.* embeds; fields are copied with JSON tags).
type DetectedShell struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Path        string   `json:"path"`
	Argv        []string `json:"argv"`
}

// ShellsResponse is the response body for GET /shells.
type ShellsResponse struct {
	Shells []DetectedShell `json:"shells"`
}

// UpdateCLIPathRequest is the request body for PATCH /settings/cli-paths/{name}.
type UpdateCLIPathRequest struct {
	Path string `json:"path"`
}

// RelayPortResponse is the response body for GET /relay-port.
type RelayPortResponse struct {
	Port int `json:"port"`
}

// WebServerStartRequest is the request body for POST /webserver/start.
type WebServerStartRequest struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	FQDN     string `json:"fqdn"`
	Mode     string `json:"mode"`     // "tailscale" | "local"
	Password string `json:"password"` // non-empty for local mode
}

// WebServerStartResponse is the response body for POST /webserver/start.
type WebServerStartResponse struct {
	URL string `json:"url"`
}

// WebServerStatusResponse is the response body for GET /webserver/status.
type WebServerStatusResponse struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
	Addr    string `json:"addr"`
	Mode    string `json:"mode"` // "tailscale" | "local" | ""
}

// WebServeRequest is the request body for POST /sessions/{id}/web-serve.
type WebServeRequest struct {
	Enabled bool `json:"enabled"`
}

// SessionBrowseRequest is the request body for
// POST /sessions/{id}/browse (Phase 137 / SHARE-03).
type SessionBrowseRequest struct {
	Enabled bool `json:"enabled"`
}

// --- Phase 87 capability types (D-06, D-07, D-09, D-11) ------------------

// IssueCapabilitiesResponse is the response body for POST
// /sessions/{id}/capabilities. Each call produces TWO capabilities (D-07):
// one read-only link and one read-write link. Each capability is paired with
// a single-use 5-minute join code (D-09/D-11).
//
// HomeDir (Phase 124 / CAP-06): true when the session's cwd equals
// EvalSymlinks($HOME). The frontend reads this field to decide whether to show
// the home-write warning banner. Populated from engine.sessionCwdIsHome.
type IssueCapabilitiesResponse struct {
	ReadURL   string `json:"readUrl"`
	WriteURL  string `json:"writeUrl"`
	ReadCode  string `json:"readCode"`
	WriteCode string `json:"writeCode"`
	HomeDir   bool   `json:"homeDir"` // Phase 124 / CAP-06: true when session cwd == EvalSymlinks($HOME)
}

// ExchangeJoinCodeRequest is the body for POST /join/exchange.
type ExchangeJoinCodeRequest struct {
	Code string `json:"code"`
}

// ExchangeJoinCodeResponse is the response body for POST /join/exchange.
// URL carries the resolved session URL with the `?cap=<token>` query
// parameter, which the caller follows to reach the gated session.
type ExchangeJoinCodeResponse struct {
	URL string `json:"url"`
}
