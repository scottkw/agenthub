import React, { useEffect, useState } from 'react'
import { GetVersion, GetLastUpdateInfo } from '../wailsjs/go/main/App'
import { EventsOn, BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'

interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}

export function WelcomeTab(): React.ReactElement {
  const [version, setVersion] = useState('dev')
  const [update, setUpdate] = useState<UpdateInfo | null>(null)

  useEffect(() => {
    GetVersion()
      .then((v) => setVersion(v))
      .catch(() => setVersion('dev'))
  }, [])

  useEffect(() => {
    // Poll bound method on mount to handle startup race
    // (event may have fired before React subscribed)
    GetLastUpdateInfo()
      .then((info) => {
        if (info) setUpdate(info)
      })
      .catch(() => {}) // silent — bound method may not exist in dev

    // Subscribe to future update events from the poller
    const offUpdate = EventsOn('update:available', (info: UpdateInfo) => {
      setUpdate(info)
    })
    return () => {
      offUpdate()
    }
  }, [])

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
        <p className="welcome-tab__version">{version}</p>

        {update && (
          <div className="update-banner" role="alert" aria-live="polite">
            <span className="update-banner__message">
              Update available:{' '}
              <span className="update-banner__version">{update.currentVersion}</span>
              {' '}
              <span className="update-banner__arrow">&rarr;</span>
              {' '}
              <span className="update-banner__version">{update.latestVersion}</span>
            </span>
            <div className="update-banner__actions">
              <button
                type="button"
                className="update-banner__btn--download"
                onClick={() => BrowserOpenURL(update.releaseURL)}
              >
                Download Update
              </button>
              <button
                type="button"
                className="update-banner__btn--dismiss"
                aria-label="Dismiss update notification"
                onClick={() => setUpdate(null)}
              >
                Dismiss
              </button>
            </div>
          </div>
        )}

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
              <code className="welcome-tab__code">brew tap scottkw/agenthub && brew install --cask agenthub</code>
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
