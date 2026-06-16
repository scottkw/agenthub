package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestJoinCodePrompt_New_FocusedAndEmpty verifies the constructor returns a
// model with a focused, empty textinput (Behavior 1).
func TestJoinCodePrompt_New_FocusedAndEmpty(t *testing.T) {
	m := newJoinCodePromptModel("sid-1", "name-x", "peer-host", "https://peer.example:9443")
	if m.sessionID != "sid-1" || m.sessionName != "name-x" || m.hostname != "peer-host" {
		t.Errorf("context fields not stored: %+v", m)
	}
	if m.remoteBaseURL != "https://peer.example:9443" {
		t.Errorf("remoteBaseURL mismatch: %q", m.remoteBaseURL)
	}
	if m.input.Value() != "" {
		t.Errorf("expected empty input, got %q", m.input.Value())
	}
	if !m.input.Focused() {
		t.Error("expected input.Focused() == true after construction")
	}
}

// TestJoinCodePrompt_TypingAppendsToInput verifies typing routes through to
// the textinput (Behavior 2).
func TestJoinCodePrompt_TypingAppendsToInput(t *testing.T) {
	m := newJoinCodePromptModel("sid", "n", "h", "https://x:9443")
	for _, r := range "ABC12" {
		updated, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated
	}
	if m.input.Value() != "ABC12" {
		t.Errorf("expected input value 'ABC12', got %q", m.input.Value())
	}
}

// TestJoinCodePrompt_EnterEmpty_NoCmd verifies pressing Enter on an empty
// input is a no-op (Behavior 4).
func TestJoinCodePrompt_EnterEmpty_NoCmd(t *testing.T) {
	m := newJoinCodePromptModel("sid", "n", "h", "https://x:9443")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("expected nil cmd on Enter with empty input, got %T", cmd())
	}
}

// TestJoinCodePrompt_EnterNonEmpty_DispatchesExchange verifies pressing Enter
// on a non-empty input transitions to submitting and dispatches the exchange
// Cmd (Behavior 3).
func TestJoinCodePrompt_EnterNonEmpty_DispatchesExchange(t *testing.T) {
	m := newJoinCodePromptModel("sid", "n", "h", "https://x:9443")
	for _, r := range "ABCDE" {
		updated, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated
	}
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil cmd on Enter with non-empty input")
	}
	if updated.state != joinCodePromptSubmitting {
		t.Errorf("expected state=joinCodePromptSubmitting after Enter, got %v", updated.state)
	}
}

// TestJoinCodePrompt_Esc_EmitsCancel verifies Esc returns a cmd that, when
// invoked, yields cancelJoinCodeMsg (Behavior 5).
func TestJoinCodePrompt_Esc_EmitsCancel(t *testing.T) {
	m := newJoinCodePromptModel("sid", "n", "h", "https://x:9443")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected non-nil cmd on Esc")
	}
	msg := cmd()
	if _, ok := msg.(cancelJoinCodeMsg); !ok {
		t.Errorf("expected cancelJoinCodeMsg, got %T", msg)
	}
}

// TestExchangeJoinCodeCmd_SuccessReturnsCap verifies a 303 Location with
// ?cap=<token> is parsed into joinCodeResultMsg.cap (Behavior 6a).
func TestExchangeJoinCodeCmd_SuccessReturnsCap(t *testing.T) {
	const wantCap = "ABC.DEF.123"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/join/exchange" {
			t.Errorf("expected /join/exchange, got %q", r.URL.Path)
		}
		_ = r.ParseForm()
		if r.FormValue("code") != "ABCDE" {
			t.Errorf("expected code=ABCDE, got %q", r.FormValue("code"))
		}
		w.Header().Set("Location", "/sessions/sid-1?cap="+wantCap)
		w.WriteHeader(http.StatusSeeOther)
	}))
	defer srv.Close()

	cmd := exchangeJoinCodeCmd("sid-1", srv.URL, "ABCDE")
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	msg, ok := cmd().(joinCodeResultMsg)
	if !ok {
		t.Fatalf("expected joinCodeResultMsg, got %T", cmd())
	}
	if msg.err != nil {
		t.Errorf("unexpected error: %v", msg.err)
	}
	if msg.cap != wantCap {
		t.Errorf("expected cap=%q, got %q", wantCap, msg.cap)
	}
	if msg.sessionID != "sid-1" {
		t.Errorf("expected sessionID=sid-1, got %q", msg.sessionID)
	}
}

// TestExchangeJoinCodeCmd_ExpiredCode parses /join?error=expired (Behavior 6b).
func TestExchangeJoinCodeCmd_ExpiredCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/join?error=expired")
		w.WriteHeader(http.StatusSeeOther)
	}))
	defer srv.Close()

	msg := exchangeJoinCodeCmd("sid", srv.URL, "XXXXX")().(joinCodeResultMsg)
	if msg.err == nil || !strings.Contains(msg.err.Error(), "expired") {
		t.Errorf("expected error containing 'expired', got %v", msg.err)
	}
}

// TestExchangeJoinCodeCmd_InvalidCode parses /join?error=invalid (Behavior 6c).
func TestExchangeJoinCodeCmd_InvalidCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/join?error=invalid")
		w.WriteHeader(http.StatusSeeOther)
	}))
	defer srv.Close()
	msg := exchangeJoinCodeCmd("sid", srv.URL, "XXXXX")().(joinCodeResultMsg)
	if msg.err == nil || !strings.Contains(msg.err.Error(), "invalid") {
		t.Errorf("expected error containing 'invalid', got %v", msg.err)
	}
}

// TestExchangeJoinCodeCmd_SessionGone parses /join?error=session-gone (Behavior 6d).
func TestExchangeJoinCodeCmd_SessionGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/join?error=session-gone")
		w.WriteHeader(http.StatusSeeOther)
	}))
	defer srv.Close()
	msg := exchangeJoinCodeCmd("sid", srv.URL, "XXXXX")().(joinCodeResultMsg)
	if msg.err == nil || !strings.Contains(msg.err.Error(), "session-gone") {
		t.Errorf("expected error containing 'session-gone', got %v", msg.err)
	}
}

// TestExchangeJoinCodeCmd_Non303 verifies a non-303 status is surfaced with
// its status code in the error message.
func TestExchangeJoinCodeCmd_Non303(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal", http.StatusInternalServerError)
	}))
	defer srv.Close()
	msg := exchangeJoinCodeCmd("sid", srv.URL, "XXXXX")().(joinCodeResultMsg)
	if msg.err == nil || !strings.Contains(msg.err.Error(), "500") {
		t.Errorf("expected error containing 500, got %v", msg.err)
	}
}

// TestJoinCodePrompt_View_IncludesSessionContext verifies the View output
// includes the session name + hostname (Behavior 8).
func TestJoinCodePrompt_View_IncludesSessionContext(t *testing.T) {
	m := newJoinCodePromptModel("sid", "my-session", "peer-host", "https://x:9443")
	out := m.View(60, 12)
	if !strings.Contains(out, "my-session") {
		t.Errorf("View missing session name; got: %q", out)
	}
	if !strings.Contains(out, "peer-host") {
		t.Errorf("View missing hostname; got: %q", out)
	}
	if !strings.Contains(out, "Enter Submit") {
		t.Errorf("View missing hint line; got: %q", out)
	}
}

// TestJoinCodePrompt_View_SubmittingState renders "Joining…" while a cap
// exchange is in flight (Behavior 7).
func TestJoinCodePrompt_View_SubmittingState(t *testing.T) {
	m := newJoinCodePromptModel("sid", "x", "h", "https://x:9443")
	m.state = joinCodePromptSubmitting
	out := m.View(60, 12)
	if !strings.Contains(out, "Joining") {
		t.Errorf("expected 'Joining' in submitting-state View, got %q", out)
	}
}

// TestJoinCodePrompt_View_ErrorState renders the errMsg under the input.
func TestJoinCodePrompt_View_ErrorState(t *testing.T) {
	m := newJoinCodePromptModel("sid", "x", "h", "https://x:9443")
	m.state = joinCodePromptError
	m.errMsg = "Code expired. Ask the owner to generate a new code."
	out := m.View(60, 12)
	if !strings.Contains(out, "Code expired") {
		t.Errorf("expected errMsg in View, got %q", out)
	}
}

// TestFriendlyJoinCodeError covers the friendly-translation matrix for the
// four documented error kinds + the default fallback.
func TestFriendlyJoinCodeError(t *testing.T) {
	cases := []struct {
		raw, want string
	}{
		{"join exchange: expired", "Code expired. Ask the owner to generate a new code."},
		{"join exchange: invalid", "Code invalid. Double-check the 8-character code (XXXX-XXXX)."},
		{"join exchange: session-gone", "Remote session is no longer web-shared."},
		{"something else", "Join failed: something else"},
	}
	for _, tc := range cases {
		got := friendlyJoinCodeError(errMust(tc.raw))
		if got != tc.want {
			t.Errorf("friendlyJoinCodeError(%q): got %q want %q", tc.raw, got, tc.want)
		}
	}
}

// errMust wraps a plain string into a synthetic error for the friendly-error
// translation table test.
func errMust(s string) error { return &stringErr{s: s} }

type stringErr struct{ s string }

func (e *stringErr) Error() string { return e.s }
