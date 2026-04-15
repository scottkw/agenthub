package tui

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Update processes messages and returns the updated model and any commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.BackgroundColorMsg:
		m.hasDark = msg.IsDark()
		m.styles = newStyles(m.hasDark)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case sessionsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.sessions = msg.sessions
		// Clamp selection if sessions list shrank
		if m.selected >= len(m.sessions) {
			m.selected = max(0, len(m.sessions)-1)
		}
		return m, nil

	case webStatusMsg:
		if msg.err == nil {
			m.webStatus = msg.status
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			fetchSessions(m.client),
			fetchWebStatus(m.client),
			nextTick(),
		)
	}

	return m, nil
}

// handleKey dispatches key presses. When help overlay is open,
// only ? and Esc are handled (all others are swallowed).
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// When help overlay is open, only allow closing it
	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || msg.String() == "esc" {
			m.showHelp = false
			return m, nil
		}
		// Swallow all other keys while help is open
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.selected < len(m.sessions)-1 {
			m.selected++
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.selected > 0 {
			m.selected--
		}
		return m, nil

	case key.Matches(msg, m.keys.Top):
		m.selected = 0
		return m, nil

	case key.Matches(msg, m.keys.Bottom):
		if len(m.sessions) > 0 {
			m.selected = len(m.sessions) - 1
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, tea.Batch(
			fetchSessions(m.client),
			fetchWebStatus(m.client),
		)

	case key.Matches(msg, m.keys.Attach):
		// Reserved for Phase 77 -- show toast
		m.toast = "Coming in next update"
		m.toastExp = time.Now().Add(2 * time.Second)
		return m, nil

	case key.Matches(msg, m.keys.New):
		// Reserved for Phase 77 -- show toast
		m.toast = "Coming in next update"
		m.toastExp = time.Now().Add(2 * time.Second)
		return m, nil
	}

	// Consume d and e silently (reserved for Phase 77, not shown in help)
	s := msg.String()
	if s == "d" || s == "e" {
		return m, nil
	}

	return m, nil
}
