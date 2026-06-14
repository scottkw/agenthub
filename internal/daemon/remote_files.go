package daemon

// Phase 122-01 — Remote-files proxy on the daemon socket.
//
// The desktop GUI (Phase 122-03) and TUI (Phase 122-04) fetch a remote
// session's files through a same-origin local-daemon URL so the Wails
// webview never has to deal with CORS preflights (Pitfall 1 in 122-RESEARCH).
// This file implements that proxy layer:
//
//   POST /api/remote-files/caps              — deposit (sessionID, baseURL, capToken)
//   GET  /api/files/remote/{sessionID}/list  — proxy GET ${baseURL}/api/files/list?...
//   GET  /api/files/remote/{sessionID}/stat  — proxy GET ${baseURL}/api/files/stat?...
//   GET  /api/files/remote/{sessionID}/read  — proxy GET ${baseURL}/api/files/read?...
//   HEAD /api/files/remote/{sessionID}/read  — proxy HEAD for ranged byte preview
//
// The outbound HTTPS client mirrors the tailnet client pattern in
// internal/tailnet/sessions.go (TLS 1.2 minimum, modest timeout) and accepts
// the tailnet peer's self-signed cert via InsecureSkipVerify — the same shape
// that Phase 122-03's client_remote_files.go uses for the join-code exchange.

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// newRemoteFilesHTTPClient constructs the outbound HTTPS client used by the
// remote-files proxy. Mirrors internal/tailnet/sessions.go's shape:
// TLS 1.2 minimum, 10-second timeout. InsecureSkipVerify is acceptable here
// because the peer is a tailnet host with a self-signed cert (Tailscale itself
// guarantees the transport-layer authenticity).
func newRemoteFilesHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, //nolint:gosec // tailnet self-signed cert; matches 122-03 client
			},
		},
	}
}

// remoteFilesClient returns the HTTP client the proxy uses for outbound
// requests. Tests override this by setting a.remoteFilesClientForTest before
// issuing requests; production code falls back to newRemoteFilesHTTPClient().
func (a *API) remoteFilesClient() *http.Client {
	if a.remoteFilesClientForTest != nil {
		return a.remoteFilesClientForTest
	}
	return newRemoteFilesHTTPClient()
}

// handleRegisterRemoteCap accepts the (sessionID, baseURL, capToken) tuple
// from the desktop GUI / TUI after a successful join-code exchange and stores
// it in the RemoteCapStore. Subsequent /api/files/remote/{sid}/... proxy
// requests look up the cap from this store.
//
// Request body: JSON {sessionId, baseUrl, capToken}.
// Responses:
//   200 + {"ok": true}                            on success
//   400 + plain text                              on malformed body or empty fields
//
// 200 (not 204) is used so the response carries a small JSON envelope the
// frontend can assert on.
func (a *API) handleRegisterRemoteCap(w http.ResponseWriter, r *http.Request) {
	// Cap the body to avoid a malicious local process pinning memory with
	// a giant JSON blob. 8 KiB matches handleSetPluginSettings.
	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	var req struct {
		SessionID string `json:"sessionId"`
		BaseURL   string `json:"baseUrl"`
		CapToken  string `json:"capToken"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if a.remoteCaps == nil {
		http.Error(w, "remote cap store not initialised", http.StatusInternalServerError)
		return
	}
	if err := a.remoteCaps.Put(req.SessionID, req.BaseURL, req.CapToken); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleRemoteFilesList proxies GET /api/files/remote/{sessionID}/list to the
// remote peer's /api/files/list endpoint using the cap stored by
// handleRegisterRemoteCap.
func (a *API) handleRemoteFilesList(w http.ResponseWriter, r *http.Request) {
	a.proxyRemoteFiles(w, r, "list")
}

// handleRemoteFilesStat proxies /stat.
func (a *API) handleRemoteFilesStat(w http.ResponseWriter, r *http.Request) {
	a.proxyRemoteFiles(w, r, "stat")
}

// handleRemoteFilesRead proxies /read (GET and HEAD).
func (a *API) handleRemoteFilesRead(w http.ResponseWriter, r *http.Request) {
	a.proxyRemoteFiles(w, r, "read")
}

// proxyRemoteFiles is the shared body for the three /api/files/remote/...
// routes. It looks up the cap, builds the upstream URL with `?cap=<token>`
// (mirroring the webserver's requireFilesRead pattern from Phase 119), issues
// the upstream request with the same HTTP method (so HEAD → HEAD survives),
// and copies upstream status + selected headers + body back to the caller.
//
// Per-status behavior:
//   no cap registered locally    → 404 + JSON {"error": "no cap registered for session"}
//   upstream 401                 → 401 (frontend treats as "cap rejected" — re-prompt for join code)
//   upstream 403                 → 403 verbatim (frontend's PermissionDeniedTakeover already handles)
//   upstream 2xx                 → 2xx verbatim
//   upstream dial / TLS failure  → 502 + plain text (cap token redacted)
func (a *API) proxyRemoteFiles(w http.ResponseWriter, r *http.Request, op string) {
	sessionID := r.PathValue("sessionID")
	if sessionID == "" {
		http.Error(w, "missing sessionID", http.StatusBadRequest)
		return
	}
	if a.remoteCaps == nil {
		http.Error(w, "remote cap store not initialised", http.StatusInternalServerError)
		return
	}

	baseURL, capToken, ok := a.remoteCaps.Get(sessionID)
	if !ok {
		// No cap deposited for this sessionID. Frontend treats 404 as
		// "user needs to paste a join code" and pops the modal.
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no cap registered for session"})
		return
	}

	// Build the upstream URL. Carry through whichever query params the
	// caller sent (the remote /api/files/* endpoints accept `session`,
	// `path`, optional `offset`/`limit`, etc.) AND append the daemon-side
	// `cap` so the remote's requireFilesRead lets us through. Strip any
	// caller-supplied `cap` first so a malicious caller cannot smuggle a
	// different token through this proxy (defense in depth — the daemon
	// socket is already loopback-trusted but we trust nothing from the
	// query string).
	q := url.Values{}
	for k, v := range r.URL.Query() {
		if strings.EqualFold(k, "cap") {
			continue
		}
		q[k] = v
	}
	// Ensure the session param matches the path param. The remote webserver
	// uses ?session=<id> for the sandbox lookup; force-set it from the URL
	// path so a caller cannot point at one session while listing another.
	q.Set("session", sessionID)
	q.Set("cap", capToken)

	upstreamURL := strings.TrimRight(baseURL, "/") + "/api/files/" + op + "?" + q.Encode()

	// Forward the request body for write verbs (PUT, POST, PATCH). GET and HEAD
	// reads are body-less; passing nil is correct for those. Forwarding r.Body
	// opaquely (as a byte pipe) preserves multipart boundaries and
	// application/json payloads without re-parsing them (CAP-10 / Pitfall 3).
	var body io.Reader
	if r.Method == http.MethodPut || r.Method == http.MethodPost || r.Method == http.MethodPatch {
		body = r.Body
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, body)
	if err != nil {
		http.Error(w, "build upstream request: "+redactCapTokenFromError(err, capToken), http.StatusInternalServerError)
		return
	}

	// Forward the inbound Content-Type request header for write verbs so
	// multipart boundaries and application/json payloads survive transit.
	// The response-header forwarding (below) already handles response Content-Type;
	// this block covers the request side which was previously not copied.
	if body != nil {
		if ct := r.Header.Get("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}
	}

	client := a.remoteFilesClient()
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "remote unreachable: "+redactCapTokenFromError(err, capToken), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Forward selected headers from the upstream response. We deliberately
	// do not blindly copy every header (Set-Cookie, Server, etc. should
	// stay scoped to the daemon socket) — but content-type / -length /
	// -range / last-modified are required for the frontend to render
	// images, PDFs, and binary blobs correctly through HEAD on /read.
	for _, h := range []string{
		"Content-Type",
		"Content-Length",
		"Content-Range",
		"Last-Modified",
		"ETag",
		"Accept-Ranges",
	} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	// HEAD has no body to copy by definition (net/http strips it), but we
	// still call io.Copy so the standard mechanism handles it uniformly.
	_, _ = io.Copy(w, resp.Body)
}

// redactCapTokenFromError removes the cap-token substring from an error's
// rendered message so we never leak token bytes into a client-visible error
// body. The cap token is the only secret in this code path; everything else
// (URL, sessionID) is already known to both peers.
func redactCapTokenFromError(err error, capToken string) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if capToken == "" {
		return msg
	}
	return strings.ReplaceAll(msg, capToken, "<redacted>")
}

