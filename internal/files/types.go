package files

// FileEntry is the JSON-serialisable description of a single filesystem
// entry returned by the files API.
//
// Name is the BASENAME only — never a path with separators. The handler
// guarantees forward-slash normalization on every platform: even on
// Windows where filepath.Base could in principle return a backslash-laden
// fragment, the handler runs strings.ReplaceAll(name, "\\", "/") on the
// Stat path. List entries derive Name from fs.DirEntry.Name() which is
// already separator-free.
//
// IsBinary semantics depend on which handler populated the entry:
//   - List responses: extension-only heuristic — extensionMIME("")
//     means "unknown extension" → IsBinary=true. List NEVER calls
//     sniffMIME (would be N syscalls on N entries — Pitfall 6).
//   - Stat responses: extension → if "", call sniffMIME(file) on the
//     open file. IsBinary is true iff MIME does NOT begin with "text/".
//
// Size and Mtime are zero / "" in List responses (Pitfall 6 mitigation —
// listing a 10k-entry directory must not perform N stat syscalls). Only
// Stat populates Size, Mtime, full Mode, and the cascaded MIME.
//
// JSON field order is the declaration order (encoding/json emits in
// declaration order). FS-03 fixes that order explicitly.
type FileEntry struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Mtime     string `json:"mtime"`
	Mode      string `json:"mode"`
	IsDir     bool   `json:"isDir"`
	IsSymlink bool   `json:"isSymlink"`
	IsBinary  bool   `json:"isBinary"`
	MIME      string `json:"mime,omitempty"`
}

// FileListResponse is the JSON body returned by Handler.List. The list is
// never recursive — it is always exactly one directory level deep.
//
// Truncated is set to true when the directory contained 10,000 or more
// entries and the handler stopped reading at the cap. Callers should
// inform the user that further entries exist.
type FileListResponse struct {
	Entries   []FileEntry `json:"entries"`
	Truncated bool        `json:"truncated"`
}
