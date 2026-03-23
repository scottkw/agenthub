package daemon

// SessionInfo is the JSON-serialisable representation of a session.
type SessionInfo struct {
	ID        string `json:"id"`
	CLI       string `json:"cli"`
	Name      string `json:"name"`
	State     string `json:"state"`
	CreatedAt string `json:"createdAt"`
}

// CreateRequest is the request body for POST /sessions.
type CreateRequest struct {
	CLI     string `json:"cli"`
	Name    string `json:"name"`
	WorkDir string `json:"workDir"`
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
