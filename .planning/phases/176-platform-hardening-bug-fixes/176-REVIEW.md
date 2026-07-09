---
phase: 176-platform-hardening-bug-fixes
reviewed: 2026-07-09T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - main.go
  - internal/webserver/server.go
  - internal/webserver/csp_integration_test.go
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 176: Code Review Report

**Reviewed:** 2026-07-09
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Phase 176 makes three small platform-hardening changes: (1) `main.go` guards the macOS role menus (`AppMenu`/`EditMenu`/`WindowMenu`) behind `GOOS == "darwin"` and sets `WEBKIT_DISABLE_DMABUF_RENDERER` on Linux (BUG-05 / #124); (2) `server.go` wraps the `GET /app/` handler in `ws.cspHeaders(...)` to add a Content-Security-Policy header (BUG-06 / #123); (3) `csp_integration_test.go` adds `TestCSPHeaderStrict_App`.

The CSP addition is correct and closes a real gap — the guest SPA on `/app/` (public via Funnel) was previously served with no CSP. The new test correctly wires a stub `fstest.MapFS`, hits the serve-index branch, and asserts the strict D-18 suite. No security regressions or crashes were introduced.

Two defects were found. The BUG-05 menu guard is scoped to `darwin` rather than "not Linux", silently stripping the role menus from Windows builds where the GTK segfault does not apply. And wrapping the whole `/app/` handler in `cspHeaders` forces `Cache-Control: no-store` onto every `/app/*` asset — directly contradicting the WR-02 caching contract documented in the same block, which the diff even preserves verbatim. The performance aspect of that second issue is out of v1 scope, but the now-false in-code contract and the silently-defeated design decision are quality defects.

## Warnings

### WR-01: darwin-only menu guard strips role menus on Windows (scope creep beyond the Linux bug)

**File:** `main.go:113-130`
**Issue:** BUG-05 is a Linux/GTK segfault (Wails' GTK backend dereferences a nil SubMenu on the role menus). The fix guards `AppMenu()`, `EditMenu()`, and `WindowMenu()` behind `if goruntime.GOOS == "darwin"`. That condition also excludes **Windows**, which uses WebView2 (not GTK) and is not affected by the segfault. Before this change Windows received all three role menus; after it, Windows gets only the custom File + Help submenus. In particular the `EditMenu` (previously appended unconditionally) is removed from the Windows build. This is an unstated behavioral change for a platform the bug does not concern, and there is no test or verification covering the Windows menu path.
**Fix:** Scope the guard to the actually-affected platform so Windows behavior is preserved, e.g.:
```go
if goruntime.GOOS != "linux" {
    m.Append(menu.AppMenu())
}
// ...
if goruntime.GOOS != "linux" {
    m.Append(menu.EditMenu())
}
if goruntime.GOOS != "linux" {
    m.Append(menu.WindowMenu())
}
```
If Windows is intended to lose these menus, state that explicitly in the comment and confirm copy/paste + window controls still work on Windows (WebView2 handles Ctrl+C/V natively, but AppMenu/WindowMenu removal is unverified).

### WR-02: cspHeaders forces `Cache-Control: no-store` on all `/app/*` assets, contradicting the documented WR-02 caching contract

**File:** `internal/webserver/server.go:1041` (wrap site); contract comment at `server.go:1031-1040`; header set at `internal/webserver/csp_mw.go:124`
**Issue:** `cspHeaders` unconditionally calls `w.Header().Set("Cache-Control", "no-store")` for every request it wraps (csp_mw.go:124). By wrapping the **entire** `/app/` handler — not just the `serveIndex` branch — this diff now applies `Cache-Control: no-store` to the hashed JS/CSS bundle responses served via `stripped.ServeHTTP` (FileServerFS). That directly contradicts the caching contract documented in the immediately-preceding comment block (server.go:1031-1040), which the diff preserves verbatim: *"Other /app/\* assets (hashed JS/CSS bundles) inherit Go's FileServerFS default (Last-Modified from the embed.FS) with NO Cache-Control header ... Vite content-hashes every asset URL ... so stale browser caches cannot serve a mismatched bundle."* The intended browser-caching optimization for content-hashed assets is now silently defeated, and the in-code comment is now false. No test catches this: `TestAppBundle_ServesRealAsset` (app_bundle_test.go) asserts the asset body but not its `Cache-Control`, and the new `TestCSPHeaderStrict_App` only exercises the `serveIndex` branch. (Per v1 scope, the pure-performance impact of no-store is out of scope; the finding is the false in-code contract plus the silently-reversed design decision.)
**Fix:** Set the CSP header on `/app/*` without forcing `no-store` on hashed assets. Options: (a) apply CSP only, letting the handler control `Cache-Control` per-branch (serveIndex already sets `no-store`; assets keep FileServerFS default); or (b) if `no-store` on assets is now intended, delete/rewrite the WR-02 comment (server.go:1031-1040) so it stops describing behavior that no longer holds, and add a `Cache-Control` assertion to `TestAppBundle_ServesRealAsset` to lock the new contract. Example for (a) — a CSP-only wrapper that does not touch Cache-Control:
```go
// cspHeadersOnly sets the CSP header but leaves Cache-Control to the handler.
mux.HandleFunc("GET /app/", ws.cspHeadersNoCacheControl(func(w http.ResponseWriter, r *http.Request) { ... }))
```

## Info

### IN-01: `WEBKIT_DISABLE_DMABUF_RENDERER` is set on all Linux, though only Wayland needs it

**File:** `main.go:70-74`
**Issue:** The comment states the DMABUF hang is "Linux/Wayland only," but the guard sets the env var for every `GOOS == "linux"` session including X11. Disabling the DMABUF GPU renderer under X11 is harmless-but-unnecessary (falls back to a slower software/GL path). This is a conservative over-broad application, not a correctness bug — X11 detection at this point in startup is awkward, so the broad guard is a defensible trade-off. Noted for awareness.
**Fix:** Optional — leave as-is (safe), or narrow to Wayland by checking `os.Getenv("WAYLAND_DISPLAY") != ""` before setting, if the X11 perf path is a concern.

### IN-02: New test covers only the serve-index branch of `/app/`

**File:** `internal/webserver/csp_integration_test.go:137-152`
**Issue:** `TestCSPHeaderStrict_App` wires an FS containing only `index.html` and requests `/app/`, so it exercises the `serveIndex` path exclusively. It does not assert that the CSP header is present on the FileServerFS asset branch (`/app/assets/*.js`) or on the 503 no-FS branch. Because `cspHeaders` is outermost the header is present on all branches, but the asset branch (the one that also carries the WR-02 Cache-Control regression) is unverified.
**Fix:** Add an asset-path request (e.g. `/app/assets/index-abc.js`) asserting both the CSP header presence and the expected `Cache-Control` value, tying test coverage to whichever caching contract WR-02 is resolved to.

---

_Reviewed: 2026-07-09_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
