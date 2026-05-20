package files

import (
	"bytes"
	"strings"
	"testing"
)

func TestExtensionMIME(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		// Source families
		{"foo.go", "text/plain; charset=utf-8"},
		{"foo.ts", "text/plain; charset=utf-8"},
		{"foo.tsx", "text/plain; charset=utf-8"},
		{"foo.py", "text/plain; charset=utf-8"},
		{"foo.md", "text/plain; charset=utf-8"},
		{"foo.markdown", "text/plain; charset=utf-8"},
		{"foo.json", "text/plain; charset=utf-8"},
		{"foo.yaml", "text/plain; charset=utf-8"},
		// HTML family forced text/plain — Pitfall 9
		{"foo.html", "text/plain; charset=utf-8"},
		{"foo.htm", "text/plain; charset=utf-8"},
		{"foo.xhtml", "text/plain; charset=utf-8"},
		// Images
		{"foo.png", "image/png"},
		{"foo.jpg", "image/jpeg"},
		{"foo.jpeg", "image/jpeg"},
		{"foo.gif", "image/gif"},
		{"foo.webp", "image/webp"},
		{"foo.svg", "image/svg+xml"},
		// Unknown
		{"foo.binarybob", ""},
		{"foo", ""},
	}
	for _, c := range cases {
		got := extensionMIME(c.name)
		if got != c.want {
			t.Errorf("extensionMIME(%q) = %q; want %q", c.name, got, c.want)
		}
	}
}

func TestSniffMIME_PNG(t *testing.T) {
	pngMagic := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	got := sniffMIME(bytes.NewReader(pngMagic))
	if !strings.HasPrefix(got, "image/png") {
		t.Errorf("sniffMIME(pngMagic) = %q; want prefix image/png", got)
	}
}

func TestSniffMIME_EmptyDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("sniffMIME panicked on empty input: %v", r)
		}
	}()
	got := sniffMIME(bytes.NewReader(nil))
	if got == "" {
		t.Errorf("sniffMIME(empty) returned empty string; want fallback MIME")
	}
}
