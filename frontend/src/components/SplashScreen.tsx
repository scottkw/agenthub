import React, { useState, useEffect } from 'react'

interface SplashScreenProps {
  done: boolean
}

export function SplashScreen({ done }: SplashScreenProps): React.ReactElement | null {
  const [visible, setVisible] = useState(true)

  useEffect(() => {
    if (done) {
      const t = setTimeout(() => setVisible(false), 300)
      return () => clearTimeout(t)
    }
  }, [done])

  // Also hide the static HTML splash when React splash mounts
  useEffect(() => {
    const el = document.getElementById('splash-static')
    if (el) el.style.display = 'none'
  }, [])

  if (!visible) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: '#1a1b26',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9999,
        opacity: done ? 0 : 1,
        transition: done ? 'opacity 0.3s ease' : 'none',
        pointerEvents: 'none',
      }}
    >
      <img
        src="/agenthub-title-logo.png"
        alt="AgentHub"
        style={{ width: 320, maxWidth: '80%' }}
        draggable={false}
      />
    </div>
  )
}
