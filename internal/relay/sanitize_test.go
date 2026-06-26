package relay

import "testing"

func TestSanitizePTYText(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// Plain passthrough — printable text passes through unchanged.
		{
			name:  "plain passthrough",
			input: "hello world",
			want:  "hello world\n",
		},
		// Trailing spaces trimmed before the single newline.
		{
			name:  "trailing spaces trimmed",
			input: "hello  ",
			want:  "hello\n",
		},
		// LF collapses to single space.
		{
			name:  "LF collapses to space",
			input: "hello\nworld",
			want:  "hello world\n",
		},
		// CR collapses to single space.
		{
			name:  "CR collapses to space",
			input: "hello\rworld",
			want:  "hello world\n",
		},
		// CRLF collapses to exactly one space (Pitfall 1: must not yield two spaces).
		{
			name:  "CRLF collapses to single space",
			input: "hello\r\nworld",
			want:  "hello world\n",
		},
		// Null byte stripped.
		{
			name:  "null byte stripped",
			input: "hel\x00lo",
			want:  "hello\n",
		},
		// C0 BEL (0x07) stripped.
		{
			name:  "C0 BEL stripped",
			input: "hel\x07lo",
			want:  "hello\n",
		},
		// DEL (0x7F) stripped.
		{
			name:  "DEL stripped",
			input: "hel\x7flo",
			want:  "hello\n",
		},
		// C1 NEL (U+0085) stripped.
		{
			name:  "C1 NEL stripped",
			input: "hello",
			want:  "hello\n",
		},
		// C1 U+0080 stripped.
		{
			name:  "C1 U+0080 stripped",
			input: "hello",
			want:  "hello\n",
		},
		// CSI clear-screen sequence stripped.
		{
			name:  "CSI clear screen stripped",
			input: "hello\x1b[2Jworld",
			want:  "helloworld\n",
		},
		// CSI SGR color sequence stripped.
		{
			name:  "CSI color code stripped",
			input: "hel\x1b[31mlo",
			want:  "hello\n",
		},
		// OSC BEL-terminated sequence stripped.
		{
			name:  "OSC BEL-terminated stripped",
			input: "hi\x1b]0;title\x07there",
			want:  "hithere\n",
		},
		// OSC ST-terminated sequence stripped (ESC \ = String Terminator).
		{
			name:  "OSC ST-terminated stripped",
			input: "hi\x1b]0;title\x1b\\there",
			want:  "hithere\n",
		},
		// Bidi RLO (U+202E) stripped — CVE-2021-42574 class.
		{
			name:  "bidi RLO stripped",
			input: "hel‮lo",
			want:  "hello\n",
		},
		// Bidi LRM (U+200E) stripped.
		{
			name:  "bidi LRM stripped",
			input: "hel‎lo",
			want:  "hello\n",
		},
		// Empty input yields bare newline.
		{
			name:  "empty input",
			input: "",
			want:  "\n",
		},
		// Only-newlines collapse to bare newline (spaces trimmed).
		{
			name:  "only newlines",
			input: "\n\n\n",
			want:  "\n",
		},
		// Mixed attack vector: CSI + null + CRLF — only safe text survives.
		{
			name:  "mixed attack vector",
			input: "cmd\x1b[A\x00\r\n;evil",
			want:  "cmd evil\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizePTYText(tc.input)
			if got != tc.want {
				t.Errorf("SanitizePTYText(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
