# Phase 137: Share Modal & Cap Model — Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 9 new/modified files
**Analogs found:** 9 / 9

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/daemon/engine.go` (collapse fields) | service | CRUD | `engine.go:596-633` (`filesWriteEnabledFor`/`SetSessionFilesWrite`) | exact |
| `internal/daemon/api.go` (perm injection) | service | request-response | `api.go:1096-1116` (existing perm block) | exact |
| `internal/daemon/types.go` (add `BrowseEnabled`, `SessionBrowseRequest`) | model | — | `types.go:130-134` (`SessionFilesWriteRequest`) | exact |
| `internal/daemon/client.go` (add `SetSessionBrowse`) | service | request-response | `client.go:323-328` (`SetSessionFilesWrite`) | exact |
| `app.go` (add `SetSessionBrowse` binding) | controller | request-response | `app.go:827-835` (`SetSessionFilesWrite`) | exact |
| `frontend/src/components/Hub/SessionShareModal.tsx` (NEW) | component | request-response | `frontend/src/components/Hub/HubModal.tsx` (overlay + phase machine) | role-match |
| `frontend/src/components/Hub/SessionCard.tsx` (Share button) | component | event-driven | `SessionCard.tsx:391-402` (Open button row5) | exact |
| `frontend/src/components/SessionSharePanel.tsx` (strip CAP-05) | component | request-response | `SessionSharePanel.tsx` (self — simplification) | exact |
| `internal/daemon/api_test.go` (retire 4, add 3) | test | — | `api_test.go:1982-2115` (existing issueCapabilities tests) | exact |
| `internal/webserver/files_routes_test.go` (add 3 new) | test | — | `files_routes_test.go:93-300` (existing route tests) | exact |
| `frontend/src/components/__tests__/SessionCard.share.test.tsx` (NEW) | test | — | `SessionSharePanel.test.tsx:1-100` (mock + renderPanel pattern) | role-match |
| `frontend/src/components/__tests__/SessionShareModal.test.tsx` (NEW) | test | — | `SessionSharePanel.test.tsx:1-100` (mock + renderPanel pattern) | role-match |
| `frontend/src/components/__tests__/SessionSharePanel.test.tsx` (update) | test | — | `SessionSharePanel.test.tsx` (self — update) | exact |

---

## Pattern Assignments

### `internal/daemon/engine.go` — collapse `filesRead`/`sessionWrites` → `sessionBrowse`

**Analog:** `engine.go:596-633` — existing `filesWriteEnabledFor` / `SetSessionFilesWrite`

**Existing fields being removed** (engine.go:44-46):
```go
filesRead           *bool           // v3.4 (Phase 118 / FS-14)
filesWriteDefault   bool            // Phase 124 / CAP-08
sessionWrites       map[string]bool // Phase 124 / CAP-04: per-session write toggle
```

**Existing settings struct field being removed** (engine.go:108-109):
```go
FilesRead  *bool `json:"filesRead,omitempty"`
FilesWrite bool  `json:"filesWrite,omitempty"`
```

**Existing per-session map pattern to copy** (engine.go:608-633):
```go
// filesWriteEnabledFor reports whether file-write capability issuance is
// enabled for the given session's owner token. This is a PER-SESSION check
// (T-124-07 mitigation — never a global flag like filesReadEnabled). If the
// session has no explicit entry in sessionWrites, the persisted
// filesWriteDefault is used (false = opt-in for all, CAP-08).
func (e *SessionEngine) filesWriteEnabledFor(sessionID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.filesWriteEnabledForUnlocked(sessionID)
}

func (e *SessionEngine) filesWriteEnabledForUnlocked(sessionID string) bool {
	if v, ok := e.sessionWrites[sessionID]; ok {
		return v
	}
	return e.filesWriteDefault
}

func (e *SessionEngine) SetSessionFilesWrite(sessionID string, enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sessionWrites == nil {
		e.sessionWrites = make(map[string]bool)
	}
	e.sessionWrites[sessionID] = enabled
}
```

**New pattern to implement (D-06/D-08):** Replace the above with `sessionBrowse map[string]bool` field and corresponding `browseEnabledFor(sessionID string) bool` / `SetSessionBrowse(sessionID string, enabled bool)` methods. Key differences from the predecessor:
- No `browseDefault` field — default is always `false` (absent from map = OFF, D-06).
- No `Unlocked` split needed unless called from a write-locked context; if not, keep simple.
- Add a comment referencing D-07 (deliberate removal of `filesReadEnabled` / global kill-switch) for the secure-phase audit.

**Mutex pattern** (same as predecessor): `e.mu.RLock()`/`e.mu.RUnlock()` for reads; `e.mu.Lock()`/`e.mu.Unlock()` for write.

---

### `internal/daemon/api.go` — rewrite perm injection at lines 1096-1116

**Analog:** `api.go:1096-1116` — existing block (the exact edit site)

**Existing perm injection block** (api.go:1096-1116 — to be replaced):
```go
now := time.Now().Unix()
// Phase 118 / FS-12: owner token includes files.read unless the operator
// has explicitly disabled it via daemonSettings.FilesRead. nil filesRead
// (legacy pre-v3.4 file) is treated as enabled — the engine's
// loadSettingsFromDisk defaults-merge writes *true at load time (see
// Plan 04), so in steady-state nil only occurs in tests. The viewer
// (read) token Perms is unchanged — viewers default-off per FS-12.
ownerPerms := "read,write"
if a.engine.filesReadEnabled() {
    ownerPerms = "read,write," + capability.PermFilesRead
}
// Phase 124 / CAP-04: append files.write to the owner token ONLY when the
// per-session write toggle is ON. This is a per-session check (Inversion 2 —
// NOT a global flag). Uses capability.HasPerm semantics via
// filesWriteEnabledFor, never strings.Contains (T-124-09 + static-grep gate).
// The read-only (rClaims) token Perms is "read" — NEVER affected.
if a.engine.filesWriteEnabledFor(sessionID) {
    ownerPerms += "," + capability.PermFilesWrite
}
rClaims := capability.Claims{SID: sessionID, Perms: "read", IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
wClaims := capability.Claims{SID: sessionID, Perms: ownerPerms, IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}
```

**New pattern (D-03/D-04 matrix):** Replace the `ownerPerms` block with:
```go
now := time.Now().Unix()
// Phase 137 / SHARE-03: perm injection driven solely by the per-session browse
// toggle (D-03/D-04). No global kill-switch (D-07 deliberate removal).
// D-03: Browse OFF: RO="read", RW="read,write"
// D-04: Browse ON:  RO="read,files.read", RW="read,write,files.read,files.write"
// Security note: files.write NEVER appears in rPerms (RO token) — see Pitfall 2.
// Deliberate reversals: T-124-07 (write no longer separately gated), CAP-08/global removed.
// Audit: secure-phase to review Reversal 1 and Reversal 3 in 137-RESEARCH.md.
rPerms := "read"
wPerms := "read,write"
if a.engine.browseEnabledFor(sessionID) {
    rPerms = "read," + capability.PermFilesRead
    wPerms = "read,write," + capability.PermFilesRead + "," + capability.PermFilesWrite
}
rClaims := capability.Claims{SID: sessionID, Perms: rPerms, IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
wClaims := capability.Claims{SID: sessionID, Perms: wPerms, IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}
```

**Constants used** (capability.go:30,37 — verified):
- `capability.PermFilesRead` = `"files.read"`
- `capability.PermFilesWrite` = `"files.write"`

**Unchanged after the block** (api.go:1118-1145 — not an edit site): `capability.Sign`, `ws.AddGrant`, URL construction, `a.joinCodes.Issue` — copy as-is.

---

### `internal/daemon/types.go` — add `BrowseEnabled` to `SessionInfo`, add `SessionBrowseRequest`

**Analog:** `types.go:20-35` (`SessionInfo`) and `types.go:130-134` (`SessionFilesWriteRequest`)

**Existing `SessionInfo` struct** (types.go:20-35 — add one field):
```go
type SessionInfo struct {
	ID          string `json:"id"`
	CLI         string `json:"cli"`
	Name        string `json:"name"`
	State       string `json:"state"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	Hostname    string `json:"hostname"`
	WebEnabled  bool   `json:"webEnabled"`
	ViewerCount int    `json:"viewerCount"`
	ExitCode    *int   `json:"exitCode,omitempty"`
	Duration    *int   `json:"duration,omitempty"`
	HomeDir     bool   `json:"homeDir"`
	FilesWrite  bool   `json:"filesWrite"`         // Phase 124 analog for BrowseEnabled
	WorkDir     string `json:"workDir"`
}
```

**Pattern for new field:** Follow `FilesWrite bool` exactly — plain `bool` (not `*bool`), no `omitempty` (false must serialize, per RESEARCH.md open question 2). Tag: `json:"browseEnabled"`. Add after `FilesWrite`.

**Existing request type to copy** (types.go:130-134):
```go
// SessionFilesWriteRequest is the request body for
// POST /sessions/{id}/files-write (Phase 124 / CAP-04).
type SessionFilesWriteRequest struct {
	Enabled bool `json:"enabled"`
}
```

**New type to add** (same structure, new name):
```go
// SessionBrowseRequest is the request body for
// POST /sessions/{id}/browse (Phase 137 / SHARE-03).
type SessionBrowseRequest struct {
	Enabled bool `json:"enabled"`
}
```

---

### `internal/daemon/client.go` — add `SetSessionBrowse`

**Analog:** `client.go:323-328` (`SetSessionFilesWrite` — exact structural copy)

**Existing method to copy** (client.go:323-328):
```go
// SetSessionFilesWrite sets the per-session file-write toggle for a session.
// Phase 124 / CAP-04. Mirrors ToggleWebServing but routes to the engine's
// per-session write map (not a global write flag).
func (c *DaemonClient) SetSessionFilesWrite(sessionID string, enabled bool) error {
	return c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/files-write",
		SessionFilesWriteRequest{Enabled: enabled}, nil)
}
```

**New method pattern:** Same signature, new endpoint `/sessions/{id}/browse`, new request type `SessionBrowseRequest`. The old `SetSessionFilesWrite` method should be removed (its endpoint is retired per D-02).

---

### `app.go` — add `SetSessionBrowse` Wails binding

**Analog:** `app.go:827-835` (`SetSessionFilesWrite` Wails binding — exact structural copy)

**Existing binding to copy** (app.go:827-835):
```go
// SetSessionFilesWrite enables or disables the per-session file-write capability
// for a specific session. Phase 124 / CAP-04. Mirrors ToggleWebServing but
// targets the engine's per-session write map (not a global flag).
func (a *App) SetSessionFilesWrite(sessionID string, enabled bool) error {
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetSessionFilesWrite(sessionID, enabled)
}
```

**New binding pattern:** Same nil-client guard, same return pattern, delegate to `a.client.SetSessionBrowse`. Remove `SetSessionFilesWrite` binding (D-02 retirement).

---

### `frontend/src/components/Hub/SessionShareModal.tsx` (NEW)

**Primary analog:** `frontend/src/components/Hub/HubModal.tsx` — overlay + phase machine + focus handling

**Overlay JSX pattern** (HubModal.tsx:168-191):
```tsx
return (
  <div
    className={`hub-modal-overlay hub-modal-overlay--${phase}`}
    onClick={handleClose}
  >
    <div
      role="dialog"
      aria-modal="true"
      aria-label={ariaLabel}
      className={`hub-modal hub-modal--${phase}`}
      onClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => {
        if (e.key === 'Escape') {
          e.preventDefault()
          e.stopPropagation()
          handleClose()
        }
      }}
      onAnimationEnd={() => {
        if (phase === 'entering') setPhase('open')
        if (phase === 'exiting') onClose()
      }}
    >
      {/* header + body */}
    </div>
  </div>
)
```

**Phase machine pattern** (HubModal.tsx:96-111 — simpler version for Share modal, no grow animation required):
```tsx
const prefersReducedMotion =
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches

const [phase, setPhase] = useState<'entering' | 'open' | 'exiting'>(
  prefersReducedMotion ? 'open' : 'entering',
)

const handleClose = useCallback(() => {
  if (prefersReducedMotion) {
    onClose()
    return
  }
  setPhase('exiting')
}, [prefersReducedMotion, onClose])
```

**Focus return pattern** (HubModal.tsx:115-121):
```tsx
const cardFocusRef = useRef<HTMLElement | null>(null)
useEffect(() => {
  cardFocusRef.current = document.activeElement as HTMLElement
  return () => {
    cardFocusRef.current?.focus()
  }
}, [])
```

**Secondary analog for modal body content:** `frontend/src/components/DaemonManagerPanel.tsx` — LAN password fetch, reconcile effect, toggle-then-reissue flow.

**LAN password fetch pattern** (DaemonManagerPanel.tsx:69-77):
```tsx
const [lanPassword, setLanPassword] = useState('')
useEffect(() => {
  if (webServerMode === 'local' && webServerRunning) {
    GetLocalNetworkPassword().then(setLanPassword).catch(() => setLanPassword(''))
  } else {
    setLanPassword('')
  }
}, [webServerMode, webServerRunning])
```

**Toggle-then-reissue pattern** (DaemonManagerPanel.tsx:90-143 — the browse toggle must follow this exact sequence):
```tsx
async function handleToggleFilesWrite(sessionId: string, enabled: boolean): Promise<void> {
  setSaving((prev) => ({ ...prev, [sessionId]: true }))
  try {
    await SetSessionFilesWrite(sessionId, enabled)
    setSessionWrites((prev) => ({ ...prev, [sessionId]: enabled }))
    // Re-issue capabilities so URLs reflect the new write state.
    if (webEnabled[sessionId]) {
      try {
        const resp = await IssueCapabilities(sessionId)
        setSessionShares((prev) => ({
          ...prev,
          [sessionId]: {
            readURL: resp.readUrl,
            writeURL: resp.writeUrl,
            readCode: resp.readCode,
            writeCode: resp.writeCode,
            homeDir: resp.homeDir ?? false,
          },
        }))
      } catch (capErr) {
        // Clear stale entry so reconcile refetches on next render
        setSessionShares((prev) => {
          const next = { ...prev }
          delete next[sessionId]
          return next
        })
      }
    }
  } finally {
    setSaving((prev) => ({ ...prev, [sessionId]: false }))
  }
}
```

**Reconcile effect (server-truth seeding, SHARE-05)** (DaemonManagerPanel.tsx:176-234):
```tsx
useEffect(() => {
  let cancelled = false
  async function reconcile(): Promise<void> {
    // Drop stale shares (toggle-off or session removed)
    setSessionShares((prev) => { /* ... */ })
    // Fetch shares for newly-enabled sessions
    for (const s of sessions) {
      if (!webEnabled[s.id]) continue
      if (sessionShares[s.id]) continue
      try {
        const resp = await IssueCapabilities(s.id)
        if (cancelled) return
        setSessionShares((prev) => { /* ... */ })
      } catch (err) { /* log, don't crash */ }
    }
  }
  void reconcile()
  return () => { cancelled = true }
}, [sessions, webEnabled, webServerRunning])
```

**Web-server restart clears stale cache** (DaemonManagerPanel.tsx:171-174):
```tsx
useEffect(() => {
  if (!webServerRunning) return
  setSessionShares({})
}, [webServerRunning])
```

**Note on modal architecture:** `SessionShareModal` should be a leaner overlay than `HubModal` — no grow animation from card center (no `sourceRect`/`transformOrigin`), no inert background trap (no terminal inside). Use the same `hub-modal-overlay` CSS class pattern but a simpler `hub-share-modal` inner class. The phase machine (`entering → open → exiting`) is still correct for fade-in/out animation.

---

### `frontend/src/components/Hub/SessionCard.tsx` — add Share button

**Analog:** `SessionCard.tsx:388-402` (existing Open button in row5 — exact pattern to copy)

**Existing Open button pattern** (SessionCard.tsx:388-402):
```tsx
{/* ROW 5: actions — Open re-attaches the session's terminal tab.
    Phase 131 UAT follow-up. Only for live sessions (a stopped session has
    no PTY to attach to). Text label (not color) keeps it colorblind-safe. */}
{onOpenSession && session.state !== 'stopped' && (
  <div className="hub-card__row5">
    <button
      type="button"
      className="hub-card__open"
      onClick={(e) => { e.stopPropagation(); onOpenSession(id, name, cli) }}
      aria-label={`Open ${name}`}
    >
      Open
    </button>
  </div>
)}
```

**`isLocal` derivation** (SessionCard.tsx:156-157 — already present, use for D-13 gate):
```tsx
// Origin marker: empty or same-machine hostname → Local
const isLocal = !hostname || hostname === ''
```

**Share button pattern (new, following the Open pattern + D-13 colorblind-safe disabled state):**
```tsx
{/* Share button — D-12: dedicated Share button opens per-card Share modal.
    D-13: disabled on remote peer cards. COLORBLIND-SAFE: LockClosedIcon (shape)
    + text label + tooltip carry disabled state; color is reinforcement only. */}
<button
  type="button"
  className="hub-card__share"
  onClick={(e) => { e.stopPropagation(); onShare?.(session) }}
  disabled={!isLocal}
  aria-label={isLocal ? `Share ${name}` : 'Only the session owner can share'}
  title={isLocal ? 'Share session' : 'Only the session owner can share'}
>
  {!isLocal && <LockClosedIcon aria-hidden="true" className="hub-card__share-lock" />}
  Share
</button>
```

**Click propagation guard:** The card's `article onClick` already uses class-based guards. The `e.stopPropagation()` inline (same as Open button) is the required pattern — see Pitfall 6 in RESEARCH.md.

**Import addition needed:** `LockClosedIcon` from `@heroicons/react/24/outline`.

---

### `frontend/src/components/SessionSharePanel.tsx` — strip CAP-05 two-gate

**Analog:** Self (simplification — strip specific state and JSX blocks)

**State to remove** (SessionSharePanel.tsx:107-116):
```tsx
// Phase 124 / CAP-05: "Allow file editing" viewer opt-in state.
const [allowFileEditing, setAllowFileEditing] = useState(false)
const [showWriteConfirm, setShowWriteConfirm] = useState(false)
const surfaceWriteLink = ownerWriteEnabled && allowFileEditing
```

**Prop to remove** (SessionSharePanel.tsx:59-71):
```tsx
/**
 * Phase 124 / CAP-04: true when the owner has enabled file writes for this
 * session. Gates the "Allow file editing" viewer opt-in (Surface 2).
 */
ownerWriteEnabled?: boolean
```

**Handler functions to remove:** `handleWriteOptinToggle`, `handleWriteOptinConfirm`, `handleWriteOptinCancel` (SessionSharePanel.tsx:140-163).

**JSX blocks to remove:** The `surfaceWriteLink` conditional + the CAP-05 opt-in row + the confirmation dialog inside the write link section.

**After stripping:** The write link row should always be surfaced (when props include `writeURL`/`writeCode`) with no gate. The scope text line (SessionSharePanel.tsx:236-238) can optionally accept a `browseEnabled` prop to display "Watch only — no file access" vs "Watch + browse files".

**New props interface (simplified):**
```tsx
interface SessionSharePanelProps {
  sessionId: string
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
  browseEnabled?: boolean  // display hint for scope text only; no gating logic
}
```

---

### `internal/daemon/api_test.go` — retire 4 old tests, add 3 new browse-matrix tests

**Analog:** `api_test.go:1982-2115` — existing four `TestIssueCapabilities_*` tests (to be retired)

**Test setup helper to reuse** (api_test.go:1995-2025):
```go
func issueCapsTestSetup(t *testing.T, filesRead *bool) (*API, *webserver.WebServer, []byte) {
	t.Helper()
	api, _, _ := testDaemon(t)
	api.engine.filesRead = filesRead
	// ...
}

func extractClaimsFromURL(t *testing.T, urlStr string, key []byte) capability.Claims {
	t.Helper()
	tok := extractCapToken(urlStr)
	claims, err := capability.Verify(tok, key)
	// ...
	return claims
}
```

**New setup helper pattern (adapt from above, remove `filesRead *bool` param — use `browseEnabled bool` instead):**
```go
func issueCapsTestSetup(t *testing.T) (*API, *webserver.WebServer, []byte) {
	t.Helper()
	api, _, _ := testDaemon(t)
	ws, err := webserver.NewWebServer(webserver.Config{BindIP: "127.0.0.1", Port: 0, FQDN: "test.local"},
		api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	api.SetWebServerForTest(ws)
	key := configureCapabilityStateForTest(t, api, ws)
	return api, ws, key
}
```

**Existing single-test assertion pattern to copy** (api_test.go:2027-2052):
```go
func TestIssueCapabilities_OwnerHasFilesRead_WhenSettingNil(t *testing.T) {
	api, _, key := issueCapsTestSetup(t, nil)

	sid, err := api.engine.CreateSession(context.Background(), "cat", "owner-files-read", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = api.engine.KillSession(sid) })

	_, writeURL, _, _, err := api.issueCapabilitiesForSession(sid)
	if err != nil {
		t.Fatalf("issueCapabilitiesForSession: %v", err)
	}
	claims := extractClaimsFromURL(t, writeURL, key)
	if claims.Perms != "read,write,"+capability.PermFilesRead {
		t.Errorf("owner Perms = %q; want exact 'read,write,files.read'", claims.Perms)
	}
}
```

**New tests to write (D-03/D-04 matrix — name conventions):**
- `TestIssueCapabilities_BrowseOff_ROPermsExact` — browse OFF: `rPerms == "read"` exactly
- `TestIssueCapabilities_BrowseOff_RWPermsExact` — browse OFF: `wPerms == "read,write"` exactly
- `TestIssueCapabilities_BrowseOn_ROPermsExact` — browse ON: `rPerms == "read,files.read"` exactly (no `files.write`)
- `TestIssueCapabilities_BrowseOn_RWPermsExact` — browse ON: `wPerms == "read,write,files.read,files.write"` exactly

For each: call `api.engine.SetSessionBrowse(sid, true/false)` before `issueCapabilitiesForSession`, verify both `readURL` and `writeURL` claims via `extractClaimsFromURL`.

---

### `internal/webserver/files_routes_test.go` — add 3 new browse-aware tests

**Analog:** `files_routes_test.go:93-300` (existing owner/viewer route tests — exact pattern to copy)

**Test setup helper to reuse** (files_routes_test.go:37-58):
```go
func newFilesTestServer(t *testing.T) (ws *WebServer, client *http.Client, sid, tmp string) {
	t.Helper()
	ws, client = testServer(t)
	ws.SetSigningKey(capTestKey)
	sid = "files-sess"
	ws.EnableSession(sid)
	tmp = t.TempDir()
	// ... write hello.txt ...
	h := files.NewHandler(func(sessionID string) (*files.Sandbox, error) { ... })
	ws.SetFilesHandler(h)
	return ws, client, sid, tmp
}
```

**Token issuance pattern** (capability_test.go:38 — `issueCapFor(t, ws, sid, perms string) string`): pass exact perm string (`"read,files.read"` for RO-browse-ON, `"read,write,files.read,files.write"` for RW-browse-ON).

**New tests to write:**
```go
// TestFilesRoutes_RO_BrowseOn_FilesReadRoute200 — RO cap WITH files.read → 200 on /api/files/list
func TestFilesRoutes_RO_BrowseOn_FilesReadRoute200(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,files.read")  // browse ON, RO code
	resp, body := doRequest(t, client, http.MethodGet, fileURL(ws, "/api/files/list", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for RO-browse-on cap on /list, got %d (body=%s)", resp.StatusCode, body)
	}
}

// TestFilesRoutes_RO_BrowseOn_WriteRoute403 — RO cap WITH files.read but NO files.write → 403 on /api/files/write
func TestFilesRoutes_RO_BrowseOn_WriteRoute403(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,files.read")  // browse ON, RO code — no files.write
	resp, _ := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/write", sid, ".", token))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for RO-browse-on cap on HEAD /write, got %d", resp.StatusCode)
	}
}

// TestFilesRoutes_RW_BrowseOn_WriteRoute200 — RW cap WITH files.write → 200 on HEAD /api/files/write
func TestFilesRoutes_RW_BrowseOn_WriteRoute200(t *testing.T) {
	ws, client, sid, _ := newFilesTestServer(t)
	token := issueCapFor(t, ws, sid, "read,write,files.read,files.write")
	resp, _ := doRequest(t, client, http.MethodHead, fileURL(ws, "/api/files/write", sid, ".", token))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for RW-browse-on cap on HEAD /write, got %d", resp.StatusCode)
	}
}
```

---

### Frontend test files (NEW): `SessionCard.share.test.tsx`, `SessionShareModal.test.tsx`

**Analog:** `frontend/src/components/__tests__/SessionSharePanel.test.tsx:1-66` — mock setup + `renderPanel` helper + `describe`/`it`/`afterEach` structure

**Wails mock pattern** (SessionSharePanel.test.tsx:20-26):
```tsx
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
  // Add new bindings as needed:
  IssueCapabilities: vi.fn().mockResolvedValue({ readUrl: 'https://r', writeUrl: 'https://w', readCode: 'rc', writeCode: 'wc', homeDir: false }),
  ToggleWebServing: vi.fn().mockResolvedValue(undefined),
  SetSessionBrowse: vi.fn().mockResolvedValue(undefined),
  GetLocalNetworkPassword: vi.fn().mockResolvedValue(''),
}))
```

**Render helper pattern** (SessionSharePanel.test.tsx:33-50):
```tsx
function renderPanel(opts: RenderOpts = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(ComponentUnderTest, { ...defaultProps, ...opts }))
  })
  return { container, root }
}
```

**afterEach cleanup pattern** (SessionSharePanel.test.tsx:56-66):
```tsx
afterEach(() => {
  if (root) { flushSync(() => root!.unmount()); root = undefined }
  if (container) { container.remove(); container = undefined }
  vi.clearAllMocks()
})
```

**SessionCard test fixture pattern:** Create a minimal `SessionInfo`-shaped object with `state: 'running'`, `hostname: ''` (local) or `hostname: 'remote.host'` (remote), `id: 'sess-1'`, `name: 'Test'`, `cli: 'claude'`.

---

### `frontend/src/components/__tests__/SessionSharePanel.test.tsx` — update existing

**Analog:** Self (the existing test file — the CAP-05 tests become the pattern to remove)

**Tests to retire** (entire `describe` block in SessionSharePanel.test.tsx:52-end):
- All tests verifying `ownerWriteEnabled` prop behavior
- All tests verifying `allowFileEditing` state
- All tests verifying `showWriteConfirm` / confirmation dialog
- All tests verifying `surfaceWriteLink` gate

**Tests to add:** Verify the simplified panel always renders the write link row when `writeURL`/`writeCode` are provided, and renders appropriate scope text when `browseEnabled` prop is true vs false.

---

## Shared Patterns

### Per-session in-memory map (ephemeral toggle state)

**Source:** `engine.go:608-633` (`sessionWrites` map + mutex pattern)
**Apply to:** `SessionEngine` for the new `sessionBrowse` map field

The pattern: a `map[string]bool` field initialized lazily on first write; `false` for absent keys (no default fallback — D-06 makes default always OFF). Read with `RLock`, write with `Lock`. No persistence to disk.

### `doJSON` HTTP client method

**Source:** `client.go:323-328` (any `doJSON` call)
**Apply to:** New `SetSessionBrowse` client method

Pattern: `c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/browse", SessionBrowseRequest{Enabled: enabled}, nil)`. The fourth `nil` param means no response body expected (204 No Content).

### Wails binding nil-client guard

**Source:** `app.go:827-835`
**Apply to:** `SetSessionBrowse` Wails binding

Pattern: always check `if a.client == nil { return fmt.Errorf("daemon not connected") }` before delegating.

### `issueCapFor` token minting (test layer)

**Source:** `internal/webserver/capability_test.go:38`
**Apply to:** All new `files_routes_test.go` tests

Pass the exact perm string as the third argument. The function mints a real HMAC-signed token and registers the grant on `ws`. Use `"read,files.read"` for RO-browse-ON and `"read,write,files.read,files.write"` for RW-browse-ON scenarios.

### `HasPerm` (never `strings.Contains`)

**Source:** `internal/capability/capability.go:51-61`
**Apply to:** Any new code that checks perm strings

```go
capability.HasPerm(claims.Perms, capability.PermFilesRead)   // correct
capability.HasPerm(claims.Perms, capability.PermFilesWrite)  // correct
// strings.Contains(claims.Perms, "files.read")              // WRONG
```

### Colorblind-safe disabled state

**Source:** RESEARCH.md Pattern 5 (D-13); `SessionCard.tsx:24-51` (STATUS_CONFIG pattern)
**Apply to:** Share button disabled state on remote peer cards

Rule: shape + text + tooltip carry the state; color is reinforcement only. The `LockClosedIcon` provides the shape signal. The `title` attribute provides the tooltip. The `aria-label` provides the screen-reader label.

### `e.stopPropagation()` on card action buttons

**Source:** `SessionCard.tsx:396` (Open button `onClick`)
**Apply to:** Share button `onClick` handler

Required to prevent the card's `article onClick` from firing (which opens the Hub interactive modal). Pattern: `onClick={(e) => { e.stopPropagation(); onShare?.(session) }}`.

---

## No Analog Found

All files have close analogs in the codebase. No files require falling back to RESEARCH.md patterns exclusively.

---

## Metadata

**Analog search scope:** `internal/daemon/`, `internal/webserver/`, `frontend/src/components/`, `frontend/src/components/Hub/`, `frontend/src/components/__tests__/`, `app.go`
**Files scanned:** 13 source files read directly
**Pattern extraction date:** 2026-06-20
