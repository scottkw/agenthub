package files

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/wailsapp/mimetype"
)

// extensionMIME maps a filename to a MIME type via its extension only.
//
// Source-code extensions are forced to "text/plain; charset=utf-8" so the
// magic-byte cascade never misclassifies a UTF-8 source file as binary.
//
// HTML extensions (.html, .htm, .xhtml) are also forced to text/plain per
// PITFALLS.md Pitfall 9 — HTML files in a user-supplied working directory
// are untrusted source code, never to be rendered by the browser.
//
// SVG is allowed as image/svg+xml; the frontend is responsible for never
// loading SVG via paths that allow inline-script execution.
//
// Returns "" for unknown / no-extension names so the caller may fall
// through to sniffMIME (magic-byte detection).
func extensionMIME(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	// Source code & plaintext family — forced text/plain to defeat
	// any heuristic that would mis-classify UTF-8 source as binary.
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
		".py", ".rb", ".rs", ".java", ".kt", ".scala", ".swift",
		".c", ".cc", ".cpp", ".cxx", ".h", ".hpp", ".hxx",
		".cs", ".php", ".pl", ".lua", ".sh", ".bash", ".zsh",
		".fish", ".ps1", ".bat", ".cmd",
		".md", ".markdown", ".mdx", ".rst", ".adoc", ".asciidoc",
		".txt", ".log", ".csv", ".tsv",
		".json", ".jsonc", ".json5",
		".yaml", ".yml",
		".toml", ".ini", ".cfg", ".conf", ".env",
		".xml", ".xsd", ".xsl", ".xslt", ".rss", ".atom",
		".sql", ".graphql", ".gql",
		".dockerfile", ".dockerignore", ".gitignore", ".gitattributes",
		".vue", ".svelte", ".astro":
		return "text/plain; charset=utf-8"

	// HTML family — FORCED text/plain per Pitfall 9 (never render
	// HTML from a user working directory).
	case ".html", ".htm", ".xhtml":
		return "text/plain; charset=utf-8"

	// Raster images
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".tiff", ".tif":
		return "image/tiff"

	// Vector images — frontend renders only via <img src>, never inline.
	case ".svg":
		return "image/svg+xml"

	// PDF
	case ".pdf":
		return "application/pdf"
	}
	// Unknown extension or no extension — caller falls through to sniffMIME.
	return ""
}

// sniffMIME reads the leading bytes of r and returns the detected MIME
// type via wailsapp/mimetype (magic-byte detection). On any error or
// undetectable content the returned string is "application/octet-stream".
//
// IMPORTANT: sniffMIME consumes bytes from r. The CALLER is responsible
// for calling r.Seek(0, io.SeekStart) afterwards if r is also intended to
// be streamed to a response. Documented per Plan 01 Task 3 contract.
func sniffMIME(r io.Reader) string {
	mt, err := mimetype.DetectReader(r)
	if err != nil || mt == nil {
		return "application/octet-stream"
	}
	s := mt.String()
	if s == "" {
		return "application/octet-stream"
	}
	return s
}
