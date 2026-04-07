import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import raw from '../../components/WelcomeTab.tsx?raw'

const __dir = dirname(fileURLToPath(import.meta.url))
const cssRaw = readFileSync(resolve(__dir, '../../style.css'), 'utf-8')

describe('WelcomeTab (BRND-02)', () => {
  it('exports WelcomeTab function component', () => {
    expect(raw).toContain('export function WelcomeTab')
  })

  it('renders the logo image', () => {
    expect(raw).toContain('/agenthub-title-logo.png')
  })

  it('shows the tagline', () => {
    expect(raw).toContain('AI Coding Session Manager')
  })

  it('fetches version from Wails binding', () => {
    expect(raw).toContain("import { GetVersion } from '../wailsjs/go/main/App'")
    expect(raw).toContain('GetVersion()')
  })

  it('does not hardcode a version number', () => {
    expect(raw).not.toContain("VERSION = '1.0.0'")
  })

  it('includes installation instructions for macOS', () => {
    expect(raw).toContain('brew install agenthub')
  })

  it('includes installation instructions for Linux', () => {
    expect(raw).toContain('install.sh')
  })

  it('includes installation instructions for Windows', () => {
    expect(raw).toContain('winget install agenthub')
  })

  it('includes a GitHub link', () => {
    expect(raw).toContain('github.com/agenthub-dev/agenthub')
  })

  it('uses BEM-style class names', () => {
    expect(raw).toContain('welcome-tab__')
  })
})

describe('WelcomeTab CSS (UI-01)', () => {
  it('welcome logo has border-radius', () => {
    // Verify .welcome-tab__logo has border-radius
    const logoRuleMatch = cssRaw.match(/\.welcome-tab__logo\s*\{[^}]*\}/)
    expect(logoRuleMatch).not.toBeNull()
    expect(logoRuleMatch![0]).toContain('border-radius')
  })
})
