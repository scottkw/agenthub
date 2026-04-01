import { describe, it, expect } from 'vitest'
import raw from '../../components/WelcomeTab.tsx?raw'

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

  it('displays a version number', () => {
    expect(raw).toContain("VERSION = '1.0.0'")
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
