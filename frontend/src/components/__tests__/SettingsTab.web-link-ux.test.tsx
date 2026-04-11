import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import raw from '../../components/SettingsTab.tsx?raw'

// Use fs.readFileSync for CSS — vitest/jsdom does not support ?raw imports for CSS.
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('WEB-01: Open in browser button', () => {
  it('imports BrowserOpenURL from Wails runtime', () => {
    expect(raw).toContain('BrowserOpenURL')
  })
  it('imports ArrowTopRightOnSquareIcon from heroicons', () => {
    expect(raw).toContain('ArrowTopRightOnSquareIcon')
  })
  it('calls BrowserOpenURL on Open button click', () => {
    expect(raw).toContain('BrowserOpenURL(serverURL)')
  })
  it('has aria-label for Open button', () => {
    expect(raw).toContain('Open dashboard in browser')
  })
})

describe('WEB-02: Copy URL to clipboard', () => {
  it('imports ClipboardSetText from Wails runtime', () => {
    expect(raw).toContain('ClipboardSetText')
  })
  it('imports ClipboardDocumentIcon from heroicons', () => {
    expect(raw).toContain('ClipboardDocumentIcon')
  })
  it('has handleCopyURL function', () => {
    expect(raw).toContain('async function handleCopyURL')
  })
  it('calls ClipboardSetText in handleCopyURL', () => {
    expect(raw).toContain('ClipboardSetText(serverURL)')
  })
  it('shows Copied! feedback', () => {
    expect(raw).toContain("Copied!")
  })
  it('has aria-label for Copy button', () => {
    expect(raw).toContain('Copy dashboard URL to clipboard')
  })
})

describe('WEB-03: QR code display', () => {
  it('imports GetWebServerQRCode from App bindings', () => {
    expect(raw).toContain('GetWebServerQRCode')
  })
  it('imports QrCodeIcon from heroicons', () => {
    expect(raw).toContain('QrCodeIcon')
  })
  it('has handleToggleDashQR function', () => {
    expect(raw).toContain('async function handleToggleDashQR')
  })
  it('renders QR image with correct alt text', () => {
    expect(raw).toContain('alt="QR code for dashboard URL"')
  })
  it('renders QR image with base64 data URI', () => {
    expect(raw).toContain('data:image/png;base64,${dashQRb64}')
  })
  it('resets QR state when server stops', () => {
    expect(raw).toContain('setDashQRb64(null)')
  })
  it('has aria-labels for QR toggle', () => {
    expect(raw).toContain('Show QR code')
    expect(raw).toContain('Hide QR code')
  })
})

describe('WEB-03: QR error handling', () => {
  it('shows error message on QR fetch failure', () => {
    expect(raw).toContain('QR unavailable')
  })
})

describe('CSS: URL action row classes', () => {
  it('has settings-web-server__url-row class', () => {
    expect(cssRaw).toContain('.settings-web-server__url-row')
  })
  it('has settings-web-server__url-text class', () => {
    expect(cssRaw).toContain('.settings-web-server__url-text')
  })
  it('has settings-web-server__action-btn class', () => {
    expect(cssRaw).toContain('.settings-web-server__action-btn')
  })
  it('has settings-web-server__action-btn--copy-done class', () => {
    expect(cssRaw).toContain('.settings-web-server__action-btn--copy-done')
  })
  it('has settings-web-server__qr class', () => {
    expect(cssRaw).toContain('.settings-web-server__qr')
  })
  it('url-row uses flex layout', () => {
    expect(cssRaw).toContain('.settings-web-server__url-row')
    const idx = cssRaw.indexOf('.settings-web-server__url-row')
    const block = cssRaw.slice(idx, cssRaw.indexOf('}', idx) + 1)
    expect(block).toContain('display: flex')
  })
  it('url-text uses text-overflow ellipsis', () => {
    const idx = cssRaw.indexOf('.settings-web-server__url-text')
    const block = cssRaw.slice(idx, cssRaw.indexOf('}', idx) + 1)
    expect(block).toContain('text-overflow: ellipsis')
  })
  it('QR image has 200px dimensions', () => {
    const idx = cssRaw.indexOf('.settings-web-server__qr')
    const block = cssRaw.slice(idx, cssRaw.indexOf('}', idx) + 1)
    expect(block).toContain('width: 200px')
    expect(block).toContain('height: 200px')
  })
})
