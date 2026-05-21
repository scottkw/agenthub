package tui

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// joinCodePromptState is the simple state machine driving the prompt's
// visual feedback. idle is the typing state; submitting shows "Joining…"
// while the cap-exchange Cmd is in flight; error renders the friendly
// translation of an exchange failure so the user can try again.
type joinCodePromptState int

const (
	joinCodePromptIdle joinCodePromptState = iota
	joinCodePromptSubmitting
	joinCodePromptError
)

// joinCodePromptModel is the Bubble Tea sub-model that captures a 5-char
// join code from the user and converts it (via exchangeJoinCodeCmd) into a
// session-scoped cap token. Mirrors the desktop GUI's paste-join-code modal
// (D-01).
type joinCodePromptModel struct {
	sessionID     string
	sessionName   string
	hostname      string
	remoteBaseURL string // e.g. "https://hub-a.tailnet.ts.net:9443"
	input         textinput.Model
	state         joinCodePromptState
	errMsg        string
}

// joinCodeResultMsg is the result envelope returned by exchangeJoinCodeCmd.
// sessionID echoes the original session so the Update loop can ignore a
// stale result that landed after the user dismissed the modal and tried a
// different remote session.
type joinCodeResultMsg struct {
	sessionID string
	cap       string
	err       error
}

// cancelJoinCodeMsg is emitted by the prompt's Update when the user presses
// Esc. The Update loop handles it by clearing modal state.
type cancelJoinCodeMsg struct{}

// newJoinCodePromptModel constructs an idle prompt focused on the textinput.
// CharLimit is 32 — JoinCodeManager codes are 5 characters but we accept up
// to 32 so a user pasting something extra (with surrounding whitespace) does
// not get silently truncated before they hit Enter.
func newJoinCodePromptModel(sessionID, sessionName, hostname, remoteBaseURL string) joinCodePromptModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 32
	ti.SetWidth(20)
	_ = ti.Focus()
	return joinCodePromptModel{
		sessionID:     sessionID,
		sessionName:   sessionName,
		hostname:      hostname,
		remoteBaseURL: remoteBaseURL,
		input:         ti,
		state:         joinCodePromptIdle,
	}
}

// Update handles a single Bubble Tea message inside the prompt sub-model.
// The returned tea.Cmd is non-nil only when the user just pressed Enter on
// a non-empty input — in which case the Cmd is the exchange round-trip.
func (m joinCodePromptModel) Update(msg tea.Msg) (joinCodePromptModel, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		// Non-key messages are not routed here — the parent Update loop
		// owns joinCodeResultMsg / cancelJoinCodeMsg.
		return m, nil
	}
	switch key.String() {
	case "esc":
		return m, func() tea.Msg { return cancelJoinCodeMsg{} }
	case "enter":
		code := strings.TrimSpace(m.input.Value())
		if code == "" {
			return m, nil
		}
		m.state = joinCodePromptSubmitting
		m.errMsg = ""
		return m, exchangeJoinCodeCmd(m.sessionID, m.remoteBaseURL, code)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	// Typing clears a stale error so the user can retry without a confusing
	// "previous-attempt-error" hangover after they edit the code.
	if m.state == joinCodePromptError {
		m.state = joinCodePromptIdle
		m.errMsg = ""
	}
	return m, cmd
}

// View renders the modal contents. The parent overlay-render wraps this in
// the centered modal frame consistent with the kill-confirm/new-session
// modals (modal.go).
func (m joinCodePromptModel) View(w, h int) string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	body := fmt.Sprintf("Ask the owner of %q on %q for the 5-character join code.\n"+
		"(Owner generates it via Daemon Manager.) Paste it below.",
		m.sessionName, m.hostname)

	var statusLine string
	switch m.state {
	case joinCodePromptSubmitting:
		statusLine = "Joining…"
	case joinCodePromptError:
		statusLine = m.errMsg
	}

	lines := []string{
		titleStyle.Render("Join Remote Session — Files"),
		"",
		body,
		"",
		m.input.View(),
	}
	if statusLine != "" {
		lines = append(lines, "", statusLine)
	}
	lines = append(lines, "", "Enter Submit  Esc Cancel")

	return strings.Join(lines, "\n")
}

// exchangeJoinCodeCmd POSTs the join code to the remote /join/exchange
// endpoint, parses the 303 Location header for ?cap=<token>, and returns a
// joinCodeResultMsg.
//
// The HTTP client is constructed inside the closure so the timeout and TLS
// pin live with this call only — the longer-lived RemoteFilesClient has its
// own 15s timeout for actual file ops.
//
// CheckRedirect returns http.ErrUseLastResponse so we observe the 303 and
// can extract the cap from the Location header rather than having the Go
// HTTP layer transparently follow it (which would lose the cap by chasing
// the redirect into /sessions/<sid>).
func exchangeJoinCodeCmd(sessionID, remoteBaseURL, code string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form := url.Values{"code": {code}}
		req, err := http.NewRequest(http.MethodPost,
			strings.TrimRight(remoteBaseURL, "/")+"/join/exchange",
			strings.NewReader(form.Encode()))
		if err != nil {
			return joinCodeResultMsg{sessionID: sessionID, err: fmt.Errorf("join exchange: %w", err)}
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return joinCodeResultMsg{sessionID: sessionID, err: fmt.Errorf("join exchange: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			return joinCodeResultMsg{sessionID: sessionID,
				err: fmt.Errorf("join exchange: unexpected status %d", resp.StatusCode)}
		}
		loc := resp.Header.Get("Location")

		// Error-shape Locations: /join?error=<kind> per webserver
		// handleJoinExchange (server.go:642-706).
		if strings.Contains(loc, "/join?error=") {
			kind := strings.TrimPrefix(loc, "/join?error=")
			// Strip any extra query suffix.
			if i := strings.IndexByte(kind, '&'); i >= 0 {
				kind = kind[:i]
			}
			return joinCodeResultMsg{sessionID: sessionID,
				err: fmt.Errorf("join exchange: %s", kind)}
		}

		u, err := url.Parse(loc)
		if err != nil {
			return joinCodeResultMsg{sessionID: sessionID,
				err: fmt.Errorf("join exchange: bad location header")}
		}
		capTok := u.Query().Get("cap")
		if capTok == "" {
			return joinCodeResultMsg{sessionID: sessionID,
				err: fmt.Errorf("join exchange: no cap in location header")}
		}
		return joinCodeResultMsg{sessionID: sessionID, cap: capTok}
	}
}

// friendlyJoinCodeError translates the raw error text from
// exchangeJoinCodeCmd into the user-facing copy specified by Plan 03 Task 2.
// The kind suffix is matched substring-style so we are robust to error
// wrapping introduced upstream.
func friendlyJoinCodeError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "expired"):
		return "Code expired. Ask the owner to generate a new code."
	case strings.Contains(s, "invalid"), strings.Contains(s, "not-found"):
		return "Code invalid. Double-check the 5-character code."
	case strings.Contains(s, "session-gone"):
		return "Remote session is no longer web-shared."
	default:
		return "Join failed: " + s
	}
}
