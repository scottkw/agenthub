package daemon

// SessionInfo is the JSON-serialisable representation of a session.
type SessionInfo struct {
	ID         string `json:"id"`
	CLI        string `json:"cli"`
	Name       string `json:"name"`
	State      string `json:"state"`
	CreatedAt  string `json:"createdAt"`
	Hostname   string `json:"hostname"`
	WebEnabled bool   `json:"webEnabled"`
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

// HealthResponse is the response body for GET /health.
type HealthResponse struct {
	Status string `json:"status"`
}

// CLIPathsResponse maps CLI name to custom path override.
type CLIPathsResponse map[string]string

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
