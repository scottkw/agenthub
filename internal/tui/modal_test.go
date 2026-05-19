package tui

import (
	"testing"
)

// TestRenderNewSessionModal_ZeroDimensions verifies the Phase 117 PAPER-01
// defensive guard against zero-dimension renders. Prior to the fix, the
// underlying lipgloss.Place call panicked with "index out of range [0] with
// length 0" when m.width or m.height was zero (race between TUI mount and
// the first WindowSizeMsg).
func TestRenderNewSessionModal_ZeroDimensions(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
	}{
		{"both_zero", 0, 0},
		{"width_zero", 0, 24},
		{"height_zero", 120, 0},
		{"negative_width", -1, 24},
		{"negative_height", 120, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.width = tc.width
			m.height = tc.height

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("renderNewSessionModal panicked with width=%d height=%d: %v",
						tc.width, tc.height, r)
				}
			}()

			out := m.renderNewSessionModal()
			if out != "" {
				t.Errorf("expected empty output for width=%d height=%d, got %d bytes",
					tc.width, tc.height, len(out))
			}
		})
	}
}

// TestRenderNewSessionModal_NoDetectedCLIs verifies the Phase 109 + 117
// defense-in-depth: even with an empty detectedCLIs slice, the render
// completes without panicking (agentEntries always returns at least Shell).
func TestRenderNewSessionModal_NoDetectedCLIs(t *testing.T) {
	m := testModel()
	m.detectedCLIs = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderNewSessionModal panicked with nil detectedCLIs: %v", r)
		}
	}()

	out := m.renderNewSessionModal()
	if out == "" {
		t.Errorf("expected non-empty output with nil detectedCLIs, got empty string")
	}
}
