package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderQROverlay renders the QR code modal overlay centered on screen.
// CRITICAL: The QR code string (m.qrContent) must NOT be wrapped in any lipgloss color style.
// ToSmallString(false) uses ANSI reset codes internally; color styling corrupts half-block rendering.
func (m Model) renderQROverlay() string {
	if m.qrSession == nil {
		return m.renderFull()
	}

	// Measure QR code dimensions
	qrLines := strings.Split(strings.TrimRight(m.qrContent, "\n"), "\n")
	qrCols := 0
	for _, line := range qrLines {
		w := lipgloss.Width(line)
		if w > qrCols {
			qrCols = w
		}
	}

	urlLen := len(m.qrURL)

	// Overlay width: max(qr_cols + 6, url_len + 6, 50) clamped to min(term_cols - 4, 80)
	innerMin := qrCols + 6
	if urlLen+6 > innerMin {
		innerMin = urlLen + 6
	}
	if 50 > innerMin {
		innerMin = 50
	}
	maxWidth := m.width - 4
	if 80 < maxWidth {
		maxWidth = 80
	}
	overlayWidth := innerMin
	if overlayWidth > maxWidth {
		overlayWidth = maxWidth
	}

	// Inner width: overlay width minus border(2) + padding(4)
	innerWidth := overlayWidth - 6

	var contentParts []string

	// QR code block -- center each line WITHOUT applying color styles
	// (ToSmallString uses ANSI resets internally; lipgloss colors corrupt half-block chars)
	for _, line := range qrLines {
		lineWidth := lipgloss.Width(line)
		padLeft := 0
		if innerWidth > lineWidth {
			padLeft = (innerWidth - lineWidth) / 2
		}
		contentParts = append(contentParts, strings.Repeat(" ", padLeft)+line)
	}

	// Blank separator
	contentParts = append(contentParts, "")

	// URL line in accent color
	urlStyled := lipgloss.NewStyle().Foreground(m.styles.FgAccent).Render(m.qrURL)
	contentParts = append(contentParts, urlStyled)

	// Blank separator
	contentParts = append(contentParts, "")

	// Hint line "Esc: close" in muted color, centered
	hint := lipgloss.NewStyle().Foreground(m.styles.FgMuted).Render("Esc: close")
	hintWidth := lipgloss.Width(hint)
	hintPad := 0
	if innerWidth > hintWidth {
		hintPad = (innerWidth - hintWidth) / 2
	}
	contentParts = append(contentParts, strings.Repeat(" ", hintPad)+hint)

	content := strings.Join(contentParts, "\n")

	// Bordered overlay (same pattern as kill-confirm and new-session modals)
	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.BorderNormal).
		Width(overlayWidth-2).
		Padding(1, 2).
		Background(m.styles.BgModal).
		Render(content)

	// Title: "QR: {session-name}" in border accent color
	titleText := fmt.Sprintf(" QR: %s ", m.qrSession.Name)
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.BorderAccent).
		Render(titleText)

	bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, bordered)
}

// sessionURL returns the web URL for the selected list entry, or "" if unavailable.
// Local sessions: URL from webStatus.URL + session ID (only if web server is running).
// Remote sessions: pre-built URL from RemoteSessionEntry.URL field.
func (m Model) sessionURL(entry listEntry) string {
	switch entry.kind {
	case entryLocal:
		if entry.session == nil || !m.webStatus.Running || m.webStatus.URL == "" {
			return ""
		}
		return fmt.Sprintf("%s/sessions/%s", m.webStatus.URL, entry.session.ID)
	case entryRemote:
		if entry.remote == nil || entry.remote.URL == "" {
			return ""
		}
		return entry.remote.URL
	default:
		return ""
	}
}
