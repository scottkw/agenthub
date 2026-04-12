---
phase: 66-web-server-link-ux
reviewed: 2026-04-11T23:52:20Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - app.go
  - app_test.go
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - frontend/src/components/SettingsTab.tsx
  - frontend/src/style.css
  - frontend/src/components/__tests__/SettingsTab.web-link-ux.test.tsx
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 66: Code Review Report

**Reviewed:** 2026-04-11T23:52:20Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Reviewed 7 files for the Phase 66 web server link UX feature, which adds Open/Copy/QR actions to the web server URL in SettingsTab. The Go backend (`app.go`) and its tests (`app_test.go`) are well-structured with consistent nil-client guards, proper error handling, and good test coverage. The Wails binding stubs (`App.d.ts`, `App.js`) correctly expose all new methods. The CSS additions follow existing BEM conventions and are clean.

Two warnings were found: an inconsistent clipboard API usage that could fail in certain Wails WebView contexts, and a missing error handler on an async clipboard call. Two informational items were also noted.

## Warnings

### WR-01: Inconsistent clipboard API -- `navigator.clipboard` vs `ClipboardSetText`

**File:** `frontend/src/components/SettingsTab.tsx:119`
**Issue:** `handleCopyPassword()` uses `navigator.clipboard.writeText()` (Web API), while `handleCopyURL()` on line 126 correctly uses the Wails-native `ClipboardSetText()`. The `navigator.clipboard` API requires a secure context and user gesture, and may not be available or may silently fail in all Wails WebView configurations (especially on older macOS WebKit versions or when the window is not focused). Since `ClipboardSetText` is already imported and used elsewhere in this file, the password copy should use it too for consistency and reliability.
**Fix:**
```tsx
async function handleCopyPassword() {
  if (!localPassword) return
  await ClipboardSetText(localPassword)
  setCopied(true)
  setTimeout(() => setCopied(false), 1500)
}
```

### WR-02: Missing error handling in `handleCopyPassword`

**File:** `frontend/src/components/SettingsTab.tsx:117-122`
**Issue:** `handleCopyPassword()` is an async function that `await`s the clipboard write but has no `try/catch`. If the clipboard write fails (permission denied, WebView restriction), the unhandled promise rejection will propagate silently. By contrast, `handleCopyURL()` on line 124 has the same pattern but relies on the Wails runtime which is more reliable. Regardless of which clipboard API is used (see WR-01), the async call should be guarded.
**Fix:**
```tsx
async function handleCopyPassword() {
  if (!localPassword) return
  try {
    await ClipboardSetText(localPassword)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  } catch {
    // Silently fail -- password is visible in the UI for manual copy
  }
}
```

## Info

### IN-01: Test certificate missing `NotBefore`/`NotAfter` fields

**File:** `app_test.go:300-305`
**Issue:** The `selfSignedTLSForAppTest` helper creates an `x509.Certificate` template without setting `NotBefore` or `NotAfter`. Both default to Go's zero time (year 0001), producing a technically expired certificate. This works today because Go's TLS server does not validate its own certificate's time bounds, but a future Go version or stricter TLS library could cause these tests to fail unexpectedly.
**Fix:**
```go
tmpl := &x509.Certificate{
    SerialNumber: big.NewInt(1),
    Subject:      pkix.Name{CommonName: "test"},
    IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
    DNSNames:     []string{"localhost"},
    NotBefore:    time.Now(),
    NotAfter:     time.Now().Add(time.Hour),
}
```

### IN-02: Port input allows `NaN`/`0` to reach backend

**File:** `frontend/src/components/SettingsTab.tsx:340`
**Issue:** The port `<input type="number">` uses `Number(e.target.value)` on change. When the user clears the field, `Number("")` returns `0`. The HTML `min`/`max` attributes are only enforced on form submission, not on `onChange`. While `port=0` is actually valid (OS picks a random port) and the backend handles it gracefully, this may confuse users who see `0` in the port field. This is a minor UX concern, not a bug.
**Fix:** Could clamp the value: `setSelectedPort(Math.max(1, Math.min(65535, Number(e.target.value) || 7443)))` -- or simply validate before calling `StartWebServer`.

---

_Reviewed: 2026-04-11T23:52:20Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
