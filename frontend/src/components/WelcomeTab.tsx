import React from 'react'

const VERSION = '1.0.0'

export function WelcomeTab(): React.ReactElement {
  return (
    <div className="welcome-tab">
      <div className="welcome-tab__content">
        <img
          src="/agenthub-title-logo.png"
          alt="AgentHub"
          className="welcome-tab__logo"
          draggable={false}
        />
        <p className="welcome-tab__tagline">AI Coding Session Manager</p>
        <p className="welcome-tab__version">v{VERSION}</p>

        <div className="welcome-tab__section">
          <h3 className="welcome-tab__heading">Get Started</h3>
          <p className="welcome-tab__text">
            Click the <strong>+</strong> button in the tab bar to create a new session
            with any detected AI coding CLI.
          </p>
        </div>

        <div className="welcome-tab__section">
          <h3 className="welcome-tab__heading">Installation</h3>
          <div className="welcome-tab__install-grid">
            <div className="welcome-tab__install-item">
              <span className="welcome-tab__install-label">macOS</span>
              <code className="welcome-tab__code">brew install agenthub</code>
            </div>
            <div className="welcome-tab__install-item">
              <span className="welcome-tab__install-label">Linux</span>
              <code className="welcome-tab__code">curl -fsSL https://agenthub.dev/install.sh | sh</code>
            </div>
            <div className="welcome-tab__install-item">
              <span className="welcome-tab__install-label">Windows</span>
              <code className="welcome-tab__code">winget install agenthub</code>
            </div>
          </div>
        </div>

        <div className="welcome-tab__section">
          <h3 className="welcome-tab__heading">Links</h3>
          <div className="welcome-tab__links">
            <span className="welcome-tab__link">github.com/agenthub-dev/agenthub</span>
          </div>
        </div>
      </div>
    </div>
  )
}
