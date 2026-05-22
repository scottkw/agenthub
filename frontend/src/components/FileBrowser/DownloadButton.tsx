// Phase 120 UAT-1 hotfix — mode-aware download button.
//
// The Wails WKWebView ignores the HTML `<a download>` attribute — clicking
// such a link just navigates the webview to the URL, replacing the React
// app with the raw file content. To get save-to-disk UX in the desktop GUI
// we need to call a Wails Go binding (`DownloadFile`) that opens the native
// SaveFileDialog and writes bytes server-side.
//
// In real browsers (web-share viewer at /app/), `<a download>` works as
// expected — the browser respects the attribute and the Content-Disposition.
//
// This component centralizes the branch so PreviewPane + UnsupportedFile
// don't each reinvent it.

import React, { useCallback } from 'react'
import { detectMode } from '../../lib/webMode'
import { DownloadFile } from '../../wailsjs/go/main/App'

export interface DownloadButtonProps {
  url: string
  filename: string
  className?: string
  ariaLabel?: string
  title?: string
  /** Visual content (icon, text). */
  children: React.ReactNode
}

export function DownloadButton({
  url,
  filename,
  className,
  ariaLabel,
  title,
  children,
}: DownloadButtonProps): React.ReactElement {
  const isWeb = detectMode() === 'web'

  const onClick = useCallback(
    (e: React.MouseEvent) => {
      if (isWeb) return // <a download> handles it natively
      e.preventDefault()
      // Fire-and-forget — the Wails binding shows the save dialog and writes
      // the file. Failures are surfaced via console.error rather than blocking
      // the UI because the binding already returns silent success on cancel.
      DownloadFile(url, filename).catch((err: unknown) => {
        console.error('DownloadFile failed:', err)
      })
    },
    [isWeb, url, filename],
  )

  if (isWeb) {
    return (
      <a
        href={url}
        download={filename}
        className={className}
        aria-label={ariaLabel}
        title={title}
        data-testid="file-browser-download"
      >
        {children}
      </a>
    )
  }

  // Desktop mode — render a button so the webview doesn't try to navigate.
  // data-download-url is the test-only signal carrying the same URL value
  // that an <a download> href would expose in web mode; production code
  // doesn't read it.
  return (
    <button
      type="button"
      onClick={onClick}
      className={className}
      aria-label={ariaLabel}
      title={title}
      data-testid="file-browser-download"
      data-download-url={url}
    >
      {children}
    </button>
  )
}
